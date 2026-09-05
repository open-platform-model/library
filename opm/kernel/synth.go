package kernel

import (
	"context"
	"fmt"

	"github.com/open-platform-model/library/opm/helper/synth"
	"github.com/open-platform-model/library/opm/module"
)

// SynthesizeInstance builds a *module.Instance from typed in-memory inputs.
// This is the recommended entry point for callers that hold a Module and
// need a fully validated instance — it mirrors how [Kernel.AcquireInstanceFromDir]
// (the validated entry over [Kernel.LoadInstancePackage]) is the recommended
// entry point for the file-driven path.
//
// SynthesizeInstance runs [synth.Instance] (which stages a package importing
// the module, renders in.Values into it, and builds it once so CUE derives
// uuid, components, auto-secrets and standard labels and performs the values
// merge against the module's #config) and then the kernel's internal instance
// processing (concreteness on the whole built spec, metadata decoding).
//
// The Kernel's [*cue.Context] threads through both steps so the resulting
// *module.Instance.Package is reachable through cue lookups using the same
// runtime. A caller without a Kernel can call [synth.Instance] directly with
// its own context; that yields the built value and staged source only, with
// no instance processing.
//
// The returned instance carries [module.Instance.Source]: the staged tree
// [synth.Instance] built, in overlay mode, with Pkg naming the reserved
// instance subdirectory inside the module's staged root.
//
// in.Values is the only values input. The zero cue.Value means "no values
// supplied"; the concreteness check then fails unless every #config field
// has a default. synth.Instance never falls back to Module.debugValues —
// frontends that want a debug-values overlay layer it on the caller side.
//
// Was: SynthesizeRelease
func (k *Kernel) SynthesizeInstance(_ context.Context, in synth.InstanceInput) (*module.Instance, error) {
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
	// package. processInstance checks concreteness and decodes metadata, the
	// same way it processes a file-loaded instance whose values live in the
	// package.
	inst, err := processInstance(spec)
	if err != nil {
		return nil, fmt.Errorf("Kernel.SynthesizeInstance: %w", err)
	}
	// Stamp the staged tree synth built (the module's cloned overlay plus the
	// synthesized instance package under its reserved subdirectory) so a
	// follow-on build can import the instance as a package.
	inst.Source = src
	return inst, nil
}
