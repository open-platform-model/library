package kernel

import (
	"context"
	"fmt"

	"github.com/open-platform-model/library/pkg/helper/synth"
	"github.com/open-platform-model/library/pkg/module"
)

// SynthesizeRelease builds a *module.Release from typed in-memory inputs.
// This is the recommended entry point for callers that hold a Module and
// need a fully validated release — it mirrors how [Kernel.LoadReleasePackage]
// is the recommended entry point for the file-driven path.
//
// SynthesizeRelease chains [synth.Release] (which unifies inputs against the
// version binding's #ModuleRelease schema and lets CUE derive uuid,
// components, auto-secrets, and standard labels) into
// [Kernel.ProcessModuleRelease] (which validates supplied values against
// the module's #config, fills them into the spec, enforces concreteness,
// and decodes release metadata).
//
// The Kernel's [*cue.Context] threads through both steps so the resulting
// *module.Release.Package is reachable through cue lookups using the same
// runtime. Callers that explicitly want the helper-level primitive — for
// example, a test that wants the spec value before concreteness enforcement
// — should call [synth.Release] directly with [Kernel.CueContext] and then
// invoke [Kernel.ProcessModuleRelease] themselves.
//
// in.Values is passed through to [Kernel.ProcessModuleRelease] unchanged.
// The zero cue.Value means "no values supplied"; [Kernel.ProcessModuleRelease]
// then fails the concreteness check unless every #config field has a
// default. synth.Release never falls back to Module.debugValues — frontends
// that want a debug-values overlay layer it on the caller side.
func (k *Kernel) SynthesizeRelease(ctx context.Context, in synth.ReleaseInput) (*module.Release, error) {
	if in.Module == nil {
		return nil, fmt.Errorf("Kernel.SynthesizeRelease: %w", synth.ErrMissingModule)
	}
	spec, err := synth.Release(k.cueCtx, in)
	if err != nil {
		return nil, fmt.Errorf("Kernel.SynthesizeRelease: %w", err)
	}
	rel, err := k.ProcessModuleRelease(ctx, spec, *in.Module, in.Values)
	if err != nil {
		return nil, fmt.Errorf("Kernel.SynthesizeRelease: %w", err)
	}
	return rel, nil
}
