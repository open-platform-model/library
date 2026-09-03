package kernel

import (
	"errors"
	"fmt"
	"sort"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/compile"
	"github.com/open-platform-model/library/opm/core"
	oerrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/internal/renderstage"
)

// glueDiagnostics mirrors the `diagnostics` struct the embedded glue emits
// (opm/internal/renderstage/render.cue.tmpl).
type glueDiagnostics struct {
	Pairs               []gluePair       `json:"pairs"`
	UnmatchedComponents []string         `json:"unmatchedComponents"`
	Missing             []glueDemand     `json:"missing"`
	Unresolved          []glueUnresolved `json:"unresolved"`
	Warnings            []glueDemand     `json:"warnings"`
	UnifyFailures       []glueUnify      `json:"unifyFailures"`
	BucketKeys          struct {
		Resources []string `json:"resources"`
		Traits    []string `json:"traits"`
	} `json:"bucketKeys"`
	Resolved       bool                 `json:"resolved"`
	OverSubscribed []glueOverSubscribed `json:"overSubscribed"`
	FailedPairs    []gluePair           `json:"failedPairs"`
}

type glueOverSubscribed struct {
	Key      string   `json:"key"`
	Catalogs []string `json:"catalogs"`
}

type gluePair struct {
	Component   string `json:"component"`
	Transformer string `json:"transformer"`
}

type glueDemand struct {
	Component string `json:"component"`
	Kind      string `json:"kind"`
	FQN       string `json:"fqn"`
}

type glueUnresolved struct {
	Component    string   `json:"component"`
	Kind         string   `json:"kind"`
	FQN          string   `json:"fqn"`
	Disqualified []string `json:"disqualified"`
}

type glueUnify struct {
	Component   string   `json:"component"`
	Transformer string   `json:"transformer"`
	Conflicts   []string `json:"conflicts"`
}

var (
	pathDiagnostics   = cue.ParsePath("diagnostics")
	pathTraitPostures = cue.ParsePath("traitPostures")
	pathRendered      = cue.ParsePath("rendered")
	pathOutput        = cue.ParsePath("output")
)

// decodeRenderDiagnostics reads `diagnostics` off the built value. It is read
// through LookupPath so it stays decodable beside a failing gate; a value that
// cannot decode (the unstated-posture case, an incomplete bool at the trait's
// own `optional`) is a build error surfaced verbatim.
func decodeRenderDiagnostics(built cue.Value, rows []ResolvedVersion) (RenderDiagnostics, error) {
	dv := built.LookupPath(pathDiagnostics)
	if !dv.Exists() {
		return RenderDiagnostics{}, fmt.Errorf("render module carries no diagnostics field: %w", built.Err())
	}
	var g glueDiagnostics
	if err := dv.Decode(&g); err != nil {
		if perr := unstatedPosture(dv); perr != nil {
			return RenderDiagnostics{}, perr
		}
		return RenderDiagnostics{}, fmt.Errorf("decoding render diagnostics (a matching verdict did not evaluate): %w", err)
	}

	diag := RenderDiagnostics{
		Pairs:            pairsOf(g.Pairs),
		Unmatched:        append([]string(nil), g.UnmatchedComponents...),
		UnhandledTraits:  map[string][]string{},
		FailedPairs:      pairsOf(g.FailedPairs),
		ResolvedVersions: rows,
	}
	sort.Strings(diag.Unmatched)

	// Always-unify disqualifications, one UnifyError per conflicting FQN,
	// indexed by (component, transformer) for demand attribution.
	byCandidate := map[gluePair][]oerrors.UnifyError{}
	for _, u := range g.UnifyFailures {
		for _, fqn := range u.Conflicts {
			ue := oerrors.UnifyError{
				Component: u.Component,
				FQN:       fqn,
				Cause:     fmt.Errorf("component %q primitive %q conflicts with the required body of transformer %q", u.Component, fqn, u.Transformer),
			}
			diag.Unify = append(diag.Unify, ue)
			key := gluePair{Component: u.Component, Transformer: u.Transformer}
			byCandidate[key] = append(byCandidate[key], ue)
		}
	}

	for _, u := range g.Unresolved {
		universe := g.BucketKeys.Resources
		if u.Kind == "trait" {
			universe = g.BucketKeys.Traits
		}
		d := oerrors.UnresolvedDemand{
			Component:    u.Component,
			FQN:          u.FQN,
			Kind:         u.Kind,
			Alternatives: renderstage.Alternatives(universe, u.FQN),
		}
		for _, tfqn := range u.Disqualified {
			d.Disqualified = append(d.Disqualified, byCandidate[gluePair{Component: u.Component, Transformer: tfqn}]...)
		}
		diag.Unresolved = append(diag.Unresolved, d)
	}

	for _, w := range g.Warnings {
		diag.UnhandledTraits[w.Component] = append(diag.UnhandledTraits[w.Component], w.FQN)
	}
	for c := range diag.UnhandledTraits {
		sort.Strings(diag.UnhandledTraits[c])
	}

	// Single-provider guard rows (0010 D32/D37), key-sorted with sorted
	// registry keys so the refusal is deterministic.
	for _, o := range g.OverSubscribed {
		catalogs := append([]string(nil), o.Catalogs...)
		sort.Strings(catalogs)
		diag.OverSubscribed = append(diag.OverSubscribed, oerrors.OverSubscribedContractError{Key: o.Key, Catalogs: catalogs})
	}
	sort.Slice(diag.OverSubscribed, func(i, j int) bool { return diag.OverSubscribed[i].Key < diag.OverSubscribed[j].Key })
	return diag, nil
}

func pairsOf(rows []gluePair) []RenderPair {
	out := make([]RenderPair, 0, len(rows))
	for _, r := range rows {
		out = append(out, RenderPair(r))
	}
	return out
}

// gateErrors is the fail-closed gate (0010 D28, D37) as the kernel enforces
// it from the decoded verdicts: unresolved demands, unmatched components and
// over-subscribed provider-fulfilled contracts all refuse, through one exit
// path, each reachable via errors.As.
func gateErrors(diag RenderDiagnostics) error {
	var gate []error
	if len(diag.Unresolved) > 0 {
		gate = append(gate, &oerrors.UnresolvedDemandsError{Demands: diag.Unresolved})
	}
	for _, o := range diag.OverSubscribed {
		gate = append(gate, o)
	}
	if len(diag.Unmatched) > 0 {
		matches := map[string]map[string]compile.MatchResult{}
		for _, c := range diag.Unmatched {
			matches[c] = map[string]compile.MatchResult{}
		}
		gate = append(gate, &compile.UnmatchedComponentsError{Components: diag.Unmatched, Matches: matches})
	}
	if len(gate) == 0 {
		return nil
	}
	return errors.Join(gate...)
}

// decodeRendered reads each matched pair's output off `rendered`, in pair
// order. A pair the glue reported as failed carries its CUE cause; a pair
// whose output is not concrete (invisible to the glue's `== _|_` guards) is
// refused here at a path naming the pair. Output kind dispatch is the same
// as the old path: a struct is one object, a list is one object per item.
func decodeRendered(built cue.Value, diag RenderDiagnostics, instanceName string) ([]*core.Compiled, error) {
	rendered := built.LookupPath(pathRendered)
	if !rendered.Exists() {
		return nil, fmt.Errorf("render module carries no rendered field: %w", built.Err())
	}
	failed := map[RenderPair]bool{}
	for _, p := range diag.FailedPairs {
		failed[p] = true
	}

	compiled := make([]*core.Compiled, 0, len(diag.Pairs))
	var errs []error
	for _, p := range diag.Pairs {
		key := fmt.Sprintf("%s :: %s", p.Component, p.Transformer)
		out := rendered.LookupPath(cue.MakePath(cue.Str(key))).LookupPath(pathOutput)
		if !out.Exists() {
			errs = append(errs, &oerrors.TransformError{ComponentName: p.Component, TransformerFQN: p.Transformer,
				Cause: fmt.Errorf("rendered output missing at %q", key)})
			continue
		}
		if err := out.Err(); err != nil || failed[p] {
			if err == nil {
				err = errors.New("transformer output is an error")
			}
			errs = append(errs, &oerrors.TransformError{ComponentName: p.Component, TransformerFQN: p.Transformer, Cause: err})
			continue
		}
		if err := out.Validate(cue.Concrete(true)); err != nil {
			errs = append(errs, &oerrors.TransformError{ComponentName: p.Component, TransformerFQN: p.Transformer,
				Cause: fmt.Errorf("output is not concrete: %w", err)})
			continue
		}
		items, err := splitOutput(out, p, instanceName)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		compiled = append(compiled, items...)
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("executing transforms: %w", errors.Join(errs...))
	}
	return compiled, nil
}

func splitOutput(out cue.Value, p RenderPair, instanceName string) ([]*core.Compiled, error) {
	switch out.Kind() {
	case cue.StructKind:
		return []*core.Compiled{{Value: out, Instance: instanceName, Component: p.Component, Transformer: p.Transformer}}, nil
	case cue.ListKind:
		iter, err := out.List()
		if err != nil {
			return nil, &oerrors.TransformError{ComponentName: p.Component, TransformerFQN: p.Transformer, Cause: fmt.Errorf("iterating output list: %w", err)}
		}
		var items []*core.Compiled
		for iter.Next() {
			items = append(items, &core.Compiled{Value: iter.Value(), Instance: instanceName, Component: p.Component, Transformer: p.Transformer})
		}
		return items, nil
	default:
		return nil, &oerrors.TransformError{ComponentName: p.Component, TransformerFQN: p.Transformer,
			Cause: fmt.Errorf("unexpected output kind %s (must be struct for a single resource or list for multiple)", out.Kind())}
	}
}

// unhandledTraitWarnings renders the unhandled-trait map as the same
// advisory lines the old path's MatchPlan.Warnings emits.
func unhandledTraitWarnings(unhandled map[string][]string) []string {
	if len(unhandled) == 0 {
		return nil
	}
	comps := make([]string, 0, len(unhandled))
	for c := range unhandled {
		comps = append(comps, c)
	}
	sort.Strings(comps)
	var out []string
	for _, c := range comps {
		for _, fqn := range unhandled[c] {
			out = append(out, fmt.Sprintf(
				"component %q: trait %q is not handled by any matched transformer (values will be ignored)", c, fqn))
		}
	}
	return out
}

// unstatedPosture finds, in the diagnostics' traitPostures table, an attached
// trait whose effective `optional` is neither concrete nor defaulted: the
// declaring catalog stated no posture, so the verdicts that depend on it
// cannot evaluate (0010 D28/D46; measured boundary, 0019 D10). The refusal
// is fail-closed and names the component, the trait and the `optional` field.
// Returns nil when every posture is stated.
func unstatedPosture(diagnostics cue.Value) error {
	comps, err := diagnostics.LookupPath(pathTraitPostures).Fields()
	if err != nil {
		return nil
	}
	for comps.Next() {
		traits, err := comps.Value().Fields()
		if err != nil {
			continue
		}
		for traits.Next() {
			opt := traits.Value().LookupPath(cue.ParsePath("optional"))
			if _, defaulted := opt.Default(); defaulted || opt.IsConcrete() {
				continue
			}
			return fmt.Errorf("component %q: trait %q states no optional posture (its `optional` field is an incomplete bool); an unhandled trait with an unstated posture refuses the render (fail-closed): %w",
				comps.Selector().Unquoted(), traits.Selector().Unquoted(), opt.Validate(cue.Concrete(true)))
		}
	}
	return nil
}
