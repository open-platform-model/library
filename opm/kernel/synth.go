package kernel

import (
	"context"
	"fmt"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/helper/synth"
	"github.com/open-platform-model/library/opm/module"
)

// SynthesizeInstance builds a *module.Instance from typed in-memory inputs.
// This is the recommended entry point for callers that hold a Module and
// need a fully validated instance — it mirrors how [Kernel.LoadInstancePackage]
// is the recommended entry point for the file-driven path.
//
// SynthesizeInstance chains [synth.Instance] (which unifies inputs against the
// version binding's #ModuleInstance schema and lets CUE derive uuid,
// components, auto-secrets, and standard labels) into
// [Kernel.ProcessModuleInstance] (which validates supplied values against
// the module's #config, fills them into the spec, enforces concreteness,
// and decodes instance metadata).
//
// The Kernel's [*cue.Context] threads through both steps so the resulting
// *module.Instance.Package is reachable through cue lookups using the same
// runtime. Callers that explicitly want the helper-level primitive — for
// example, a test that wants the spec value before concreteness enforcement
// — should call [synth.Instance] directly with [Kernel.CueContext] and then
// invoke [Kernel.ProcessModuleInstance] themselves.
//
// The returned instance carries [module.Instance.Source]: the staged tree
// [synth.Instance] built, in overlay mode, with Pkg naming the reserved
// instance subdirectory inside the module's staged root.
//
// in.Values is passed through to [Kernel.ProcessModuleInstance] unchanged.
// The zero cue.Value means "no values supplied"; [Kernel.ProcessModuleInstance]
// then fails the concreteness check unless every #config field has a
// default. synth.Instance never falls back to Module.debugValues — frontends
// that want a debug-values overlay layer it on the caller side.
//
// Was: SynthesizeRelease
func (k *Kernel) SynthesizeInstance(ctx context.Context, in synth.InstanceInput) (*module.Instance, error) {
	if in.Module == nil {
		return nil, fmt.Errorf("Kernel.SynthesizeInstance: %w", synth.ErrMissingModule)
	}
	// The Kernel owns the cache; callers MUST NOT need to thread it
	// through SynthesizeInstance explicitly. If they did set SchemaCache,
	// honor it (a test may pin a different one), otherwise fall back to
	// the kernel-owned cache.
	if in.SchemaCache == nil {
		in.SchemaCache = k.schemaCache
	}
	spec, src, err := synth.Instance(k.cueCtx, in)
	if err != nil {
		return nil, fmt.Errorf("Kernel.SynthesizeInstance: %w", err)
	}
	// synth.Instance bakes in.Values into the single build (as values.cue), so
	// the spec already carries them — exactly like an authored instance.cue
	// package. Re-filling here would write values a second time into the now-set
	// `values` path and conflict. Pass the zero value: ProcessModuleInstance then
	// validates concreteness and decodes metadata without re-filling, the same
	// way it processes a file-loaded instance whose values live in the package.
	inst, err := k.ProcessModuleInstance(ctx, spec, *in.Module, cue.Value{})
	if err != nil {
		return nil, fmt.Errorf("Kernel.SynthesizeInstance: %w", err)
	}
	// Stamp the staged tree synth built (the module's cloned overlay plus the
	// synthesized instance package under its reserved subdirectory) so a
	// follow-on build can import the instance as a package.
	inst.Source = src
	return inst, nil
}
