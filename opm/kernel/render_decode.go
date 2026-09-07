package kernel

import (
	"errors"
	"fmt"
	"sort"

	"cuelang.org/go/cue"

	oerrors "github.com/open-platform-model/library/opm/errors"
)

// glueDiagnostics mirrors the `diagnostics` struct the embedded glue emits
// (opm/internal/renderstage/render.cue.tmpl) exactly: every field the build
// exports is decoded and read, and nothing is derived, joined, grouped or
// re-sorted on this side.
type glueDiagnostics struct {
	Pairs          []gluePair                       `json:"pairs"`
	Unmatched      []oerrors.UnmatchedComponent     `json:"unmatched"`
	Unresolved     []oerrors.UnresolvedDemand       `json:"unresolved"`
	Warnings       []glueDemand                     `json:"warnings"`
	UnifyFailures  []oerrors.UnifyRefusal           `json:"unifyFailures"`
	OverSubscribed []oerrors.OverSubscribedContract `json:"overSubscribed"`
	FailedPairs    []gluePair                       `json:"failedPairs"`
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

var (
	pathDiagnostics   = cue.ParsePath("diagnostics")
	pathTraitPostures = cue.ParsePath("traitPostures")
	pathRendered      = cue.ParsePath("rendered")
	pathOutput        = cue.ParsePath("output")
)

// decodeRenderDiagnostics reads `diagnostics` off the built value. It is read
// through LookupPath so it stays decodable beside a failing gate; a value that
// cannot decode (the unstated-posture case, an incomplete bool at the trait's
// own `optional`) is a build error surfaced verbatim. The glue's rows land on
// the diagnostics as they were emitted: alternatives, disqualification
// conflicts, the candidate matrix and the ladder order are all decided inside
// the build.
func decodeRenderDiagnostics(built cue.Value, rows []ResolvedVersion) (RenderDiagnostics, error) {
	dv := built.LookupPath(pathDiagnostics)
	if !dv.Exists() {
		return RenderDiagnostics{}, fmt.Errorf("render module carries no diagnostics field: %w", built.Err())
	}
	// Every verdict must be concrete before it is read. A comprehension whose
	// guard did not evaluate decodes as an empty list, which would read as
	// "no verdict" rather than "the verdict is unknown", so concreteness is
	// asserted first and the fail-closed refusal is raised here.
	if err := dv.Validate(cue.Concrete(true)); err != nil {
		if perr := unstatedPosture(dv); perr != nil {
			return RenderDiagnostics{}, perr
		}
		return RenderDiagnostics{}, fmt.Errorf("decoding render diagnostics (a matching verdict did not evaluate): %w", err)
	}
	var g glueDiagnostics
	if err := dv.Decode(&g); err != nil {
		return RenderDiagnostics{}, fmt.Errorf("decoding render diagnostics (a matching verdict did not evaluate): %w", err)
	}

	diag := RenderDiagnostics{
		Pairs:            pairsOf(g.Pairs),
		Unmatched:        g.Unmatched,
		Unresolved:       g.Unresolved,
		Unify:            g.UnifyFailures,
		OverSubscribed:   g.OverSubscribed,
		UnhandledTraits:  map[string][]string{},
		FailedPairs:      pairsOf(g.FailedPairs),
		ResolvedVersions: rows,
	}

	for _, w := range g.Warnings {
		diag.UnhandledTraits[w.Component] = append(diag.UnhandledTraits[w.Component], w.FQN)
	}
	for c := range diag.UnhandledTraits {
		sort.Strings(diag.UnhandledTraits[c])
	}
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
// path, each reachable via errors.As. Each cause carries the diagnostics'
// rows unchanged and in the same order.
func gateErrors(diag RenderDiagnostics) error {
	var gate []error
	if len(diag.Unresolved) > 0 {
		gate = append(gate, &oerrors.UnresolvedDemandsError{Demands: diag.Unresolved})
	}
	if len(diag.OverSubscribed) > 0 {
		gate = append(gate, &oerrors.OverSubscribedContractsError{Contracts: diag.OverSubscribed})
	}
	if len(diag.Unmatched) > 0 {
		gate = append(gate, &oerrors.UnmatchedComponentsError{Components: diag.Unmatched})
	}
	if len(gate) == 0 {
		return nil
	}
	return errors.Join(gate...)
}

// decodeRendered reads each matched pair's output off `rendered`, in pair
// order. A pair the glue reported as failed carries its CUE cause; a pair
// whose output is not concrete (invisible to the glue's `== _|_` guards) is
// refused here at a path naming the pair. Output kind dispatch: a struct is
// one object, a list is one object per item.
func decodeRendered(built cue.Value, diag RenderDiagnostics, instanceName string) ([]*Compiled, error) {
	rendered := built.LookupPath(pathRendered)
	if !rendered.Exists() {
		return nil, fmt.Errorf("render module carries no rendered field: %w", built.Err())
	}
	failed := map[RenderPair]bool{}
	for _, p := range diag.FailedPairs {
		failed[p] = true
	}

	compiled := make([]*Compiled, 0, len(diag.Pairs))
	var errs []error
	for _, p := range diag.Pairs {
		key := fmt.Sprintf("%s :: %s", p.Component, p.Transformer)
		out := rendered.LookupPath(cue.MakePath(cue.Str(key))).LookupPath(pathOutput)
		if !out.Exists() {
			errs = append(errs, &oerrors.TransformError{Component: p.Component, Transformer: p.Transformer,
				Cause: fmt.Errorf("rendered output missing at %q", key)})
			continue
		}
		if err := out.Err(); err != nil || failed[p] {
			if err == nil {
				err = errors.New("transformer output is an error")
			}
			errs = append(errs, &oerrors.TransformError{Component: p.Component, Transformer: p.Transformer, Cause: err})
			continue
		}
		if err := out.Validate(cue.Concrete(true)); err != nil {
			errs = append(errs, &oerrors.TransformError{Component: p.Component, Transformer: p.Transformer,
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

func splitOutput(out cue.Value, p RenderPair, instanceName string) ([]*Compiled, error) {
	switch out.Kind() {
	case cue.StructKind:
		return []*Compiled{{Value: out, Instance: instanceName, Component: p.Component, Transformer: p.Transformer}}, nil
	case cue.ListKind:
		iter, err := out.List()
		if err != nil {
			return nil, &oerrors.TransformError{Component: p.Component, Transformer: p.Transformer, Cause: fmt.Errorf("iterating output list: %w", err)}
		}
		var items []*Compiled
		for iter.Next() {
			items = append(items, &Compiled{Value: iter.Value(), Instance: instanceName, Component: p.Component, Transformer: p.Transformer})
		}
		return items, nil
	default:
		return nil, &oerrors.TransformError{Component: p.Component, Transformer: p.Transformer,
			Cause: fmt.Errorf("unexpected output kind %s (must be struct for a single resource or list for multiple)", out.Kind())}
	}
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
