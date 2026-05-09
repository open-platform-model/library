package values

import (
	"cuelang.org/go/cue"

	oerrors "github.com/open-platform-model/library/pkg/errors"
)

// KernelOwner is the minimal contract [ValidateAndUnify] needs from the
// kernel: access to the [*cue.Context] that owns the [Layer] values, and
// the partial-validation entry point used per layer. The [pkg/kernel.Kernel]
// type satisfies this interface, so call sites pass `*kernel.Kernel`
// directly.
//
// The interface lives here (rather than importing pkg/kernel) so this
// package does not import the kernel — that import would close the cycle
// helper/values → kernel → helper/values, and the kernel's own
// `ValidateAndUnify` ergonomic shortcut would no longer compile. By
// expressing the dependency as an interface, the helper stays kernel-free
// and any non-kernel implementation can plug in (notably tests).
type KernelOwner interface {
	// CueContext returns the [*cue.Context] the helper uses for any value
	// construction it needs to perform.
	CueContext() *cue.Context

	// ValidateConfigPartial performs the kernel's Tier-1 partial validation
	// on a single layer. The helper invokes this once per [Layer] in a
	// [Stack] so each source's diagnostics are positioned independently.
	ValidateConfigPartial(schema cue.Value, values cue.Value, contextLabel, name string) (cue.Value, *oerrors.ConfigError)
}

// Layer is a single labeled values source. Frontends construct one Layer
// per origin (a CLI -f flag, a K8s ConfigMap, a CR overlay, a composition
// input) so Tier-1 errors carry per-source attribution.
type Layer struct {
	// Name is a human-friendly identifier shown in error messages
	// ("user-values.cue", "ConfigMap/foo", "CRD spec.values").
	Name string

	// Source is a stable identifier for machine-readable correlation
	// (file path, K8s object reference, composition input key).
	Source string

	// Value is the raw values payload for this layer.
	Value cue.Value
}

// Stack is an ordered sequence of [Layer]s. Later layers override earlier
// layers on conflicting fields; `Stack{a, b, c}` unifies as `a → a∪b → a∪b∪c`.
type Stack []Layer

// ValidateAndUnify performs Tier-1 source-positioned validation on each
// [Layer] in the [Stack] by calling owner.ValidateConfigPartial, then
// unifies in order on success. Any layer that fails validation contributes
// a [LayerError] to the returned [*MultiSourceError]; if any layer fails,
// unification is skipped and the returned [cue.Value] is the zero value.
//
// On full success the returned value is `layers[0].Value ∪ layers[1].Value ∪
// … ∪ layers[N-1].Value` and the error is nil. An empty [Stack] returns
// `cue.Value{}` and nil — the kernel treats the zero [cue.Value] as
// "no values supplied".
//
// ValidateAndUnify does NOT call the kernel's Tier-2 validation. The
// caller passes the unified value to the kernel (e.g. via
// [Kernel.ProcessModuleRelease] or [Kernel.ValidateConfig]) so Tier-2 runs
// against the same merged result.
func ValidateAndUnify(owner KernelOwner, schema cue.Value, layers Stack) (cue.Value, *MultiSourceError) {
	if len(layers) == 0 {
		return cue.Value{}, nil
	}

	var perLayer []LayerError
	for _, l := range layers {
		_, cfgErr := owner.ValidateConfigPartial(schema, l.Value, "values", l.Name)
		if cfgErr != nil {
			perLayer = append(perLayer, LayerError{
				LayerName: l.Name,
				Source:    l.Source,
				Err:       cfgErr,
			})
		}
	}
	if len(perLayer) > 0 {
		return cue.Value{}, &MultiSourceError{errors: perLayer}
	}

	merged := layers[0].Value
	for i := 1; i < len(layers); i++ {
		merged = merged.Unify(layers[i].Value)
	}
	return merged, nil
}
