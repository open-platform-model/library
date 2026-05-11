// Package compile's matching logic.
//
// Match walks each consumer Module component, collects the union of resource
// and trait FQNs the component declares (component.#resources keys ∪
// component.#traits keys), and looks each demanded FQN up in
// Platform.#matchers.{resources, traits}. The reverse index is built by the
// schema at apis/core/v1alpha2/platform.cue: matchers[FQN] yields the list
// of transformers that require that primitive FQN.
//
// Multiple candidates per FQN are normal. The matcher evaluates each
// candidate's predicate (requiredLabels ∧ requiredResources ∧ requiredTraits)
// against the component context and pairs every survivor. Same-FQN
// transformer collisions are caught upstream by CUE map unification on
// #composedTransformers.
//
// Located transformer bodies live in Platform.#composedTransformers[tfFQN];
// requiredLabels matching against component metadata.labels still applies
// and is retained verbatim from the previous implementation. All path access
// goes through binding.Paths() so v1alpha1/v1alpha2/etc. share one walk.
//
// Match is in Go (not CUE #PlatformMatch) per umbrella decision Q1: keeps
// the Go-native error/diagnostic shape, avoids one CUE evaluation per match,
// and reuses the existing #config / labels code paths unchanged.
package compile

import (
	"fmt"
	"sort"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/pkg/api"
	"github.com/open-platform-model/library/pkg/platform"
)

// MatchResult is the per-(component, transformer) match outcome.
type MatchResult struct {
	Matched          bool     `json:"matched"`
	MissingLabels    []string `json:"missingLabels"`
	MissingResources []string `json:"missingResources"`
	MissingTraits    []string `json:"missingTraits"`
}

// MatchPlan is the full result of matching components against a platform's transformers.
type MatchPlan struct {
	Matches         map[string]map[string]MatchResult
	Unmatched       []string
	UnhandledTraits map[string][]string
}

// MatchedPair is a single (component, transformer) pair that matched.
type MatchedPair struct {
	ComponentName  string
	TransformerFQN string
}

// NonMatchedPair is a single (component, transformer) pair that did not match,
// with the specific labels/resources/traits that were missing.
type NonMatchedPair struct {
	ComponentName    string
	TransformerFQN   string
	MissingLabels    []string
	MissingResources []string
	MissingTraits    []string
}

// Match walks a consumer Module's components against a Platform's
// #matchers index and returns a MatchPlan describing matched pairs,
// unmatched FQNs, and ambiguous FQNs. The binding argument supplies the
// per-schema-version CUE path inventory; every lookup goes through b.Paths().
//
//nolint:gocyclo // matching is naturally branchy but kept in one place
func Match(components cue.Value, plat *platform.Platform, b api.Binding) (*MatchPlan, error) {
	if plat == nil {
		return nil, fmt.Errorf("platform is required")
	}
	if b == nil {
		return nil, fmt.Errorf("binding is required")
	}
	paths := b.Paths()
	plan := &MatchPlan{Matches: map[string]map[string]MatchResult{}, UnhandledTraits: map[string][]string{}}

	composed := plat.Package.LookupPath(paths.ComposedTransformers)
	matchersResources := plat.Package.LookupPath(paths.MatchersResources)
	matchersTraits := plat.Package.LookupPath(paths.MatchersTraits)

	compIter, err := components.Fields()
	if err != nil {
		return nil, fmt.Errorf("iterating components: %w", err)
	}

	for compIter.Next() {
		compName := compIter.Selector().Unquoted()
		compVal := compIter.Value()
		labels := labelPairs(compVal.LookupPath(paths.MetadataLabels))
		resources := fieldKeys(compVal.LookupPath(paths.ComponentResources))
		traits := fieldKeys(compVal.LookupPath(paths.ComponentTraits))
		resourceSet := stringSet(resources)
		traitSet := stringSet(traits)

		plan.Matches[compName] = map[string]MatchResult{}
		matched := map[string]struct{}{}
		traitHandled := map[string]struct{}{}

		// Resource demand walk. Every transformer whose predicate is satisfied
		// by the component pairs; matched is keyed by transformer FQN so the
		// trait walk below idempotently dedupes.
		for _, fqn := range resources {
			survivors := lookupCandidates(matchersResources, fqn, paths, labels, resourceSet, traitSet)
			if len(survivors) == 0 {
				plan.Matches[compName][fqn] = MatchResult{MissingResources: []string{fqn}}
				continue
			}
			for _, tfFQN := range survivors {
				pairTransformer(plan, compName, tfFQN, composed, paths, labels, matched)
			}
		}

		// Trait demand walk.
		for _, fqn := range traits {
			survivors := lookupCandidates(matchersTraits, fqn, paths, labels, resourceSet, traitSet)
			if len(survivors) == 0 {
				plan.Matches[compName][fqn] = MatchResult{MissingTraits: []string{fqn}}
				continue
			}
			for _, tfFQN := range survivors {
				pairTransformer(plan, compName, tfFQN, composed, paths, labels, matched)
			}
			traitHandled[fqn] = struct{}{}
		}

		if len(matched) == 0 {
			plan.Unmatched = append(plan.Unmatched, compName)
		}
		// Carry forward optionalTraits handled by any matched transformer so
		// they are not flagged as unhandled.
		for tfFQN := range matched {
			tfVal := composed.LookupPath(cue.MakePath(cue.Str(tfFQN)))
			for _, fqn := range fieldKeys(tfVal.LookupPath(paths.TransformerOptionalTraits)) {
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

// lookupCandidates reads matchersIndex[FQN] and returns the FQNs of every
// transformer in the bucket whose predicate (requiredLabels ∧
// requiredResources ∧ requiredTraits) is satisfied by the supplied component
// context. Multiple satisfied candidates are legitimate — they describe
// independent transformers that all fire for the component.
//
// Empty result means no candidate's predicate fits; the caller treats that
// as a missing-resource / missing-trait diagnostic.
func lookupCandidates(
	matchersIndex cue.Value,
	fqn string,
	paths api.Paths,
	compLabels map[string]struct{},
	compResources map[string]struct{},
	compTraits map[string]struct{},
) []string {
	if !matchersIndex.Exists() {
		return nil
	}
	entry := matchersIndex.LookupPath(cue.MakePath(cue.Str(fqn)))
	if !entry.Exists() {
		return nil
	}
	switch entry.Kind() {
	case cue.ListKind:
		iter, err := entry.List()
		if err != nil {
			return nil
		}
		var survivors []string
		for iter.Next() {
			cand := iter.Value()
			tfFQN, err := cand.LookupPath(paths.MetadataFQN).String()
			if err != nil {
				continue
			}
			if !candidateSatisfied(cand, paths, compLabels, compResources, compTraits) {
				continue
			}
			survivors = append(survivors, tfFQN)
		}
		return survivors
	case cue.StringKind:
		s, err := entry.String()
		if err != nil {
			return nil
		}
		return []string{s}
	default:
		return nil
	}
}

// candidateSatisfied reports whether the supplied component context satisfies
// every required* clause of cand. requiredLabels: every k=v must be in
// compLabels. requiredResources / requiredTraits: every FQN must be present in
// the corresponding set. Missing required* fields are trivially satisfied
// (transformer doesn't constrain that dimension).
func candidateSatisfied(
	cand cue.Value,
	paths api.Paths,
	compLabels map[string]struct{},
	compResources map[string]struct{},
	compTraits map[string]struct{},
) bool {
	if missing := missingMapLabels(cand.LookupPath(paths.TransformerRequiredLabels), compLabels); len(missing) > 0 {
		return false
	}
	if !fqnSubset(cand.LookupPath(paths.TransformerRequiredResources), compResources) {
		return false
	}
	if !fqnSubset(cand.LookupPath(paths.TransformerRequiredTraits), compTraits) {
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
	paths api.Paths,
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
	missingLabels := missingMapLabels(tfVal.LookupPath(paths.TransformerRequiredLabels), labels)
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
// with missing labels, resources, and traits. Sorted by component
// name then transformer FQN.
func (p *MatchPlan) NonMatchedPairs() []NonMatchedPair {
	pairs := make([]NonMatchedPair, 0)
	for compName, tfResults := range p.Matches {
		for tfFQN, result := range tfResults {
			if !result.Matched {
				pairs = append(pairs, NonMatchedPair{
					ComponentName:    compName,
					TransformerFQN:   tfFQN,
					MissingLabels:    result.MissingLabels,
					MissingResources: result.MissingResources,
					MissingTraits:    result.MissingTraits,
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
