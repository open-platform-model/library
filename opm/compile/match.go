// Package compile's matching logic.
//
// Match walks each consumer Module component, collects the resource and trait
// FQNs the component declares (component.#resources keys ∪ component.#traits
// keys), and looks each demanded FQN up in the materialized platform's
// #matchers.{resources, traits} reverse index. The index is filled by
// opm/materialize: matchers[FQN] yields the list of transformers that require
// that primitive FQN.
//
// The algorithm is FQN-lookup → always-unify → predicate:
//
//  1. Lookup. A demanded FQN whose #matchers bucket is empty is a hard miss —
//     recorded as a structured oerrors.MissingFQN (per (instance, component,
//     fqn)), accumulated in one pass with no fail-fast.
//  2. Always-unify (D6). For each candidate transformer in the bucket, unify
//     the component's primitive body against the transformer's required body
//     for every FQN present in BOTH component.#resources and
//     transformer.requiredResources (and the analogous traits intersection,
//     per D1) — not only the triggering FQN. A conflict records an
//     oerrors.UnifyError (verbatim CUE cause) and disqualifies the candidate.
//  3. Predicate. Surviving candidates are paired iff their requiredLabels ∧
//     requiredResources ∧ requiredTraits predicate is satisfied by the
//     component context. Multiple satisfied candidates are legitimate.
//
// Located transformer bodies live in the composed map keyed by tfFQN; both the
// composed map and the #matchers reverse index are read off the
// MaterializedPlatform's native Transformers / Matchers fields (built in the
// owner context, not filled onto the closed platform).
//
// Match is in Go (not CUE #PlatformMatch) per umbrella decision Q1: keeps the
// Go-native error/diagnostic shape, avoids one CUE evaluation per match, and
// reuses the existing #config / labels code paths unchanged.
package compile

import (
	"fmt"
	"sort"
	"strings"

	"cuelang.org/go/cue"
	"github.com/Masterminds/semver/v3"

	oerrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/materialize"
	"github.com/open-platform-model/library/opm/schema"
)

// MatchResult is the per-(component, transformer) match outcome.
type MatchResult struct {
	Matched       bool     `json:"matched"`
	MissingLabels []string `json:"missingLabels"`
}

// MatchPlan is the full result of matching components against a platform's transformers.
type MatchPlan struct {
	Matches         map[string]map[string]MatchResult
	Unmatched       []string
	UnhandledTraits map[string][]string

	// Missing holds the hard "no transformer requires this FQN" diagnostics,
	// one per (instance, component, fqn). Distinct from UnhandledTraits, which
	// flags a component trait that no matched transformer consumes (soft).
	Missing []oerrors.MissingFQN

	// Unify holds the always-unify-rung failures: a component primitive body
	// that conflicts with a candidate transformer's required body at the same
	// FQN. The conflicting candidate is not paired.
	Unify []oerrors.UnifyError
}

// MatchedPair is a single (component, transformer) pair that matched.
type MatchedPair struct {
	ComponentName  string
	TransformerFQN string
}

// NonMatchedPair is a single (component, transformer) pair that did not match,
// with the specific labels that were missing.
type NonMatchedPair struct {
	ComponentName  string
	TransformerFQN string
	MissingLabels  []string
}

// Match walks a consumer Module's components against a MaterializedPlatform's
// #matchers index and returns a MatchPlan describing matched pairs, unmatched
// components, structured missing-FQN diagnostics, and unify failures.
// instanceName populates MissingFQN.Instance; a blank value is tolerated when
// Match is called outside the kernel. The composed map comes from
// mp.Transformers and the reverse index from mp.Matchers.{resources,traits}.
//
//nolint:gocyclo // matching is naturally branchy but kept in one place
func Match(components cue.Value, mp *materialize.MaterializedPlatform, instanceName string) (*MatchPlan, error) {
	if mp == nil {
		return nil, fmt.Errorf("materialized platform is required")
	}
	plan := &MatchPlan{Matches: map[string]map[string]MatchResult{}, UnhandledTraits: map[string][]string{}}

	composed := mp.Transformers
	matchersResources := mp.Matchers.LookupPath(cue.ParsePath("resources"))
	matchersTraits := mp.Matchers.LookupPath(cue.ParsePath("traits"))

	compIter, err := components.Fields()
	if err != nil {
		return nil, fmt.Errorf("iterating components: %w", err)
	}

	for compIter.Next() {
		compName := compIter.Selector().Unquoted()
		compVal := compIter.Value()
		labels := labelPairs(compVal.LookupPath(schema.MetadataLabels))
		resources := fieldKeys(compVal.LookupPath(schema.ComponentResources))
		traits := fieldKeys(compVal.LookupPath(schema.ComponentTraits))
		resourceSet := stringSet(resources)
		traitSet := stringSet(traits)

		plan.Matches[compName] = map[string]MatchResult{}
		matched := map[string]struct{}{}
		traitHandled := map[string]struct{}{}
		// unify is evaluated once per (component, candidate); cache the verdict
		// so a transformer demanded via several FQNs is not re-unified (and does
		// not record duplicate UnifyErrors).
		unifyChecked := map[string]bool{}
		unifyOK := map[string]bool{}

		// walk processes one demanded FQN: lookup → unify → predicate.
		walk := func(matchersIndex cue.Value, fqn string, isTrait bool) {
			candidates, exists := bucketTransformers(matchersIndex, fqn)
			if !exists {
				plan.Missing = append(plan.Missing, oerrors.MissingFQN{
					Instance:     instanceName,
					Component:    compName,
					FQN:          fqn,
					Alternatives: alternativesFor(matchersIndex, fqn),
				})
				return
			}
			for _, cand := range candidates {
				tfFQN, ferr := cand.LookupPath(schema.MetadataFQN).String()
				if ferr != nil {
					continue
				}
				if !unifyChecked[tfFQN] {
					unifyChecked[tfFQN] = true
					unifyOK[tfFQN] = runUnify(plan, compName, compVal, cand)
				}
				if !unifyOK[tfFQN] {
					continue
				}
				if !candidateSatisfied(cand, labels, resourceSet, traitSet) {
					continue
				}
				pairTransformer(plan, compName, tfFQN, composed, labels, matched)
				if isTrait {
					traitHandled[fqn] = struct{}{}
				}
			}
		}

		// Resource demand walk, then trait demand walk. matched is keyed by
		// transformer FQN so a transformer demanded via several FQNs pairs once.
		for _, fqn := range resources {
			walk(matchersResources, fqn, false)
		}
		for _, fqn := range traits {
			walk(matchersTraits, fqn, true)
		}

		if len(matched) == 0 {
			plan.Unmatched = append(plan.Unmatched, compName)
		}
		// Carry forward optionalTraits handled by any matched transformer so
		// they are not flagged as unhandled.
		for tfFQN := range matched {
			tfVal := composed.LookupPath(cue.MakePath(cue.Str(tfFQN)))
			for _, fqn := range fieldKeys(tfVal.LookupPath(schema.TransformerOptionalTraits)) {
				traitHandled[fqn] = struct{}{}
			}
		}
		for _, fqn := range traits {
			if _, ok := traitHandled[fqn]; !ok {
				plan.UnhandledTraits[compName] = append(plan.UnhandledTraits[compName], fqn)
			}
		}
		sort.Strings(plan.UnhandledTraits[compName])
	}

	sort.Strings(plan.Unmatched)
	return plan, nil
}

// bucketTransformers reads matchersIndex[FQN] and returns the candidate
// transformer values in the bucket. exists is false when the FQN key is absent
// or yields no candidates — the caller treats that as a hard MissingFQN. The
// materialized #matchers index stores each bucket as a list of transformer
// struct values (see opm/materialize/index.go); a lone struct is tolerated
// defensively.
func bucketTransformers(matchersIndex cue.Value, fqn string) ([]cue.Value, bool) {
	if !matchersIndex.Exists() {
		return nil, false
	}
	entry := matchersIndex.LookupPath(cue.MakePath(cue.Str(fqn)))
	if !entry.Exists() {
		return nil, false
	}
	switch entry.Kind() {
	case cue.ListKind:
		iter, err := entry.List()
		if err != nil {
			return nil, false
		}
		var out []cue.Value
		for iter.Next() {
			out = append(out, iter.Value())
		}
		return out, len(out) > 0
	case cue.StructKind:
		return []cue.Value{entry}, true
	default:
		return nil, false
	}
}

// runUnify is the always-unify rung. It unifies the component's primitive
// bodies against the candidate transformer's required primitive bodies for
// every FQN present in BOTH sides — the full intersection, per D1, not only the
// FQN that triggered the bucket lookup. A conflict appends an oerrors.UnifyError
// carrying the verbatim CUE cause and returns false, so the caller skips
// predicate evaluation for the candidate.
func runUnify(plan *MatchPlan, compName string, compVal, cand cue.Value) bool {
	// Evaluate both intersections (no short-circuit) so every conflicting FQN
	// is recorded, then combine the verdicts.
	resourcesOK := unifyIntersection(plan, compName,
		compVal.LookupPath(schema.ComponentResources),
		cand.LookupPath(schema.TransformerRequiredResources))
	traitsOK := unifyIntersection(plan, compName,
		compVal.LookupPath(schema.ComponentTraits),
		cand.LookupPath(schema.TransformerRequiredTraits))
	return resourcesOK && traitsOK
}

// unifyIntersection unifies have[FQN] against required[FQN] for each FQN in
// required that is also present in have. Validation uses cue.Concrete(false):
// matching happens pre-render on schema bodies, so structural agreement (not
// fully-concrete values) is the correct bar (Q3). Returns false if any FQN
// conflicts, recording one UnifyError per conflict.
func unifyIntersection(plan *MatchPlan, compName string, have, required cue.Value) bool {
	if !required.Exists() {
		return true
	}
	iter, err := required.Fields(cue.Optional(true))
	if err != nil {
		return true
	}
	ok := true
	for iter.Next() {
		fqn := iter.Selector().Unquoted()
		cv := have.LookupPath(cue.MakePath(cue.Str(fqn)))
		if !cv.Exists() {
			// Not in the intersection — the predicate's subset check handles a
			// required primitive the component lacks.
			continue
		}
		if vErr := cv.Unify(iter.Value()).Validate(cue.Concrete(false)); vErr != nil {
			plan.Unify = append(plan.Unify, oerrors.UnifyError{
				Component: compName,
				FQN:       fqn,
				Cause:     vErr,
			})
			ok = false
		}
	}
	return ok
}

// alternativesFor walks the matchers index keys (the primitive-FQN universe the
// platform's transformers require) and returns every FQN other than missingFQN
// that shares the same modulePath/name — the FQN substring before the final
// "@" — sorted by SemVer. Per D2 this surfaces "a transformer exists for a
// different version of this primitive". The full same-name set is returned;
// trimming to truly-adjacent versions is a frontend presentation nuance.
func alternativesFor(matchersIndex cue.Value, missingFQN string) []string {
	if !matchersIndex.Exists() {
		return nil
	}
	base := fqnBase(missingFQN)
	iter, err := matchersIndex.Fields()
	if err != nil {
		return nil
	}
	var alts []string
	for iter.Next() {
		key := iter.Selector().Unquoted()
		if key == missingFQN {
			continue
		}
		if fqnBase(key) == base {
			alts = append(alts, key)
		}
	}
	sortFQNsBySemVer(alts)
	return alts
}

// fqnBase returns the FQN with its "@<version>" suffix removed (the
// modulePath/name portion). An FQN without "@" is returned unchanged.
func fqnBase(fqn string) string {
	if i := strings.LastIndex(fqn, "@"); i >= 0 {
		return fqn[:i]
	}
	return fqn
}

// fqnVersion returns the bare version following the final "@", or "" if absent.
func fqnVersion(fqn string) string {
	if i := strings.LastIndex(fqn, "@"); i >= 0 {
		return fqn[i+1:]
	}
	return ""
}

// sortFQNsBySemVer sorts FQNs ascending by their "@<version>" suffix parsed as
// SemVer (Masterminds tolerates a leading "v"). FQNs whose version does not
// parse fall back to lexical order.
func sortFQNsBySemVer(fqns []string) {
	sort.Slice(fqns, func(i, j int) bool {
		vi, ei := semver.NewVersion(fqnVersion(fqns[i]))
		vj, ej := semver.NewVersion(fqnVersion(fqns[j]))
		if ei == nil && ej == nil {
			if vi.Equal(vj) {
				return fqns[i] < fqns[j]
			}
			return vi.LessThan(vj)
		}
		return fqns[i] < fqns[j]
	})
}

// candidateSatisfied reports whether the supplied component context satisfies
// every required* clause of cand. requiredLabels: every k=v must be in
// compLabels. requiredResources / requiredTraits: every FQN must be present in
// the corresponding set. Missing required* fields are trivially satisfied
// (transformer doesn't constrain that dimension).
func candidateSatisfied(
	cand cue.Value,
	compLabels map[string]struct{},
	compResources map[string]struct{},
	compTraits map[string]struct{},
) bool {
	if missing := missingMapLabels(cand.LookupPath(schema.TransformerRequiredLabels), compLabels); len(missing) > 0 {
		return false
	}
	if !fqnSubset(cand.LookupPath(schema.TransformerRequiredResources), compResources) {
		return false
	}
	if !fqnSubset(cand.LookupPath(schema.TransformerRequiredTraits), compTraits) {
		return false
	}
	return true
}

// fqnSubset reports whether every field key in required is present in have.
// An absent or non-existent required value is trivially a subset.
func fqnSubset(required cue.Value, have map[string]struct{}) bool {
	if !required.Exists() {
		return true
	}
	iter, err := required.Fields(cue.Optional(true))
	if err != nil {
		return true
	}
	for iter.Next() {
		fqn := iter.Selector().Unquoted()
		if _, ok := have[fqn]; !ok {
			return false
		}
	}
	return true
}

// stringSet returns a set view over a slice of strings.
func stringSet(s []string) map[string]struct{} {
	out := make(map[string]struct{}, len(s))
	for _, k := range s {
		out[k] = struct{}{}
	}
	return out
}

// pairTransformer records a (component, transformer) outcome in the plan.
// The transformer's requiredLabels are evaluated against the component's
// labels; mismatches surface as MissingLabels and the pair is not marked
// matched.
func pairTransformer(
	plan *MatchPlan,
	compName, tfFQN string,
	composed cue.Value,
	labels map[string]struct{},
	matched map[string]struct{},
) {
	if _, already := plan.Matches[compName][tfFQN]; already {
		// Multiple demanded FQNs may map to the same transformer; record once.
		if _, ok := matched[tfFQN]; ok {
			return
		}
	}
	tfVal := composed.LookupPath(cue.MakePath(cue.Str(tfFQN)))
	missingLabels := missingMapLabels(tfVal.LookupPath(schema.TransformerRequiredLabels), labels)
	result := MatchResult{
		Matched:       len(missingLabels) == 0,
		MissingLabels: missingLabels,
	}
	plan.Matches[compName][tfFQN] = result
	if result.Matched {
		matched[tfFQN] = struct{}{}
	}
}

// MatchedPairs returns all matched component-transformer pairs,
// sorted by component name and then transformer FQN.
func (p *MatchPlan) MatchedPairs() []MatchedPair {
	pairs := make([]MatchedPair, 0)
	for compName, tfResults := range p.Matches {
		for tfFQN, result := range tfResults {
			if result.Matched {
				pairs = append(pairs, MatchedPair{ComponentName: compName, TransformerFQN: tfFQN})
			}
		}
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].ComponentName != pairs[j].ComponentName {
			return pairs[i].ComponentName < pairs[j].ComponentName
		}
		return pairs[i].TransformerFQN < pairs[j].TransformerFQN
	})
	return pairs
}

// NonMatchedPairs returns all non-matched component-transformer pairs
// with missing labels. Sorted by component name then transformer FQN.
func (p *MatchPlan) NonMatchedPairs() []NonMatchedPair {
	pairs := make([]NonMatchedPair, 0)
	for compName, tfResults := range p.Matches {
		for tfFQN, result := range tfResults {
			if !result.Matched {
				pairs = append(pairs, NonMatchedPair{
					ComponentName:  compName,
					TransformerFQN: tfFQN,
					MissingLabels:  result.MissingLabels,
				})
			}
		}
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].ComponentName != pairs[j].ComponentName {
			return pairs[i].ComponentName < pairs[j].ComponentName
		}
		return pairs[i].TransformerFQN < pairs[j].TransformerFQN
	})
	return pairs
}

// Warnings returns warnings for traits not handled by any matched
// transformer. Those trait values will be ignored in rendering.
//
// UnhandledTraits is intentionally distinct from MatchPlan.Missing: a trait is
// "unhandled" when no matched transformer consumes it (the trait FQN may still
// have a transformer on the platform), whereas a MissingFQN means no transformer
// on the platform requires the FQN at all.
func (p *MatchPlan) Warnings() []string {
	if len(p.UnhandledTraits) == 0 {
		return nil
	}
	compNames := make([]string, 0, len(p.UnhandledTraits))
	for compName := range p.UnhandledTraits {
		compNames = append(compNames, compName)
	}
	sort.Strings(compNames)
	var warnings []string
	for _, compName := range compNames {
		traits := append([]string(nil), p.UnhandledTraits[compName]...)
		sort.Strings(traits)
		for _, fqn := range traits {
			warnings = append(warnings, fmt.Sprintf(
				"component %q: trait %q is not handled by any matched transformer (values will be ignored)",
				compName, fqn,
			))
		}
	}
	return warnings
}

// labelPairs converts a cue struct of string fields into a set of
// "key=value" pairs for matching against required labels.
func labelPairs(v cue.Value) map[string]struct{} {
	pairs := map[string]struct{}{}
	iter, err := v.Fields(cue.Optional(true))
	if err != nil {
		return pairs
	}
	for iter.Next() {
		str, err := iter.Value().String()
		if err != nil {
			continue
		}
		pairs[fmt.Sprintf("%s=%s", iter.Selector().Unquoted(), str)] = struct{}{}
	}
	return pairs
}

// fieldKeys returns the sorted list of field keys in the given cue struct value.
// No options are passed so that definition fields (#resources, #traits) are returned correctly.
func fieldKeys(v cue.Value) []string {
	iter, err := v.Fields()
	if err != nil {
		return nil
	}
	var out []string
	for iter.Next() {
		out = append(out, iter.Selector().Unquoted())
	}
	sort.Strings(out)
	return out
}

// missingMapLabels compares required labels in a transformer against
// the "key=value" pairs present in a component's metadata.labels.
func missingMapLabels(required cue.Value, have map[string]struct{}) []string {
	iter, err := required.Fields(cue.Optional(true))
	if err != nil {
		return nil
	}
	var missing []string
	for iter.Next() {
		str, err := iter.Value().String()
		if err != nil {
			continue
		}
		pair := fmt.Sprintf("%s=%s", iter.Selector().Unquoted(), str)
		if _, ok := have[pair]; !ok {
			missing = append(missing, pair)
		}
	}
	sort.Strings(missing)
	return missing
}
