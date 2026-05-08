package values

import (
	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/pkg/validate"
)

// KernelOwner is the minimal contract [ValidateAndUnify] needs from the
// kernel: access to the [*cue.Context] that owns the [Layer] values. The
// [pkg/kernel.Kernel] type satisfies this interface, so call sites pass
// `*kernel.Kernel` directly. The interface lives here so pkg/helper/values
// does not import pkg/kernel and we avoid a cycle with the kernel wrapper
// that delegates back to this package.
type KernelOwner interface {
	CueContext() *cue.Context
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
// [Layer] in the [Stack], then unifies in order on success. Any layer that
// fails validation contributes a [LayerError] to the returned
// [*MultiSourceError]; if any layer fails, unification is skipped and the
// returned [cue.Value] is the zero value.
//
// On full success the returned value is `layers[0].Value ∪ layers[1].Value ∪
// … ∪ layers[N-1].Value` and the error is nil. An empty [Stack] returns
// `cue.Value{}` and nil — the kernel treats the zero [cue.Value] as
// "no values supplied".
//
// The owner parameter is reserved for future use (helpers may build
// [cue.Value]s in the kernel's [*cue.Context]); today the implementation
// does not consult it. It is required so the helper signature stays stable
// as the helper grows and so call sites read uniformly across the
// pkg/helper/* packages.
//
// ValidateAndUnify does NOT call the kernel's Tier-2 validation. The
// caller passes the unified value to the kernel (e.g. via
// [Kernel.ParseModuleRelease] or [Kernel.ValidateConfig]) so Tier-2 runs
// against the same merged result.
func ValidateAndUnify(_ KernelOwner, schema cue.Value, layers Stack) (cue.Value, *MultiSourceError) {
	if len(layers) == 0 {
		return cue.Value{}, nil
	}

	var perLayer []LayerError
	for _, l := range layers {
		_, cfgErr := validate.ConfigPartial(schema, l.Value, "values", l.Name)
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
