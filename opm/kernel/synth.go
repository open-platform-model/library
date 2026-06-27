package kernel

import (
	"context"
	"fmt"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/helper/synth"
	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/platform"
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
	spec, err := synth.Instance(k.cueCtx, in)
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
	return inst, nil
}

// SynthesizePlatform builds a *platform.Platform from typed in-memory inputs.
// This is the recommended entry point for callers that hold platform
// configuration as typed data — an operator reconciling a Platform CRD spec,
// a CLI assembling subscriptions from flags — and is the synthesis peer of
// [Kernel.LoadPlatformPackage] on the file-driven path.
//
// SynthesizePlatform chains [synth.Platform] (which unifies inputs against the
// resolved #Platform schema) into [platform.NewPlatformFromValue] (which
// decodes the platform metadata and stores the value as the platform's
// Package). It returns the pre-materialize *platform.Platform twin.
//
// It does NOT call [Kernel.Materialize]: resolving the platform's #registry
// subscriptions into a *MaterializedPlatform performs registry I/O and stays
// an explicit, separate, caller-driven step (Principle I / design D14). The
// returned Package carries #registry as authored with #composedTransformers /
// #matchers unset.
//
// The Kernel owns the schema cache; callers MUST NOT need to thread it
// through explicitly. If in.SchemaCache is set it is honored (a test may pin a
// different one), otherwise it defaults to the kernel-owned cache.
//
// ctx is unused today — synthesis touches no I/O the caller could cancel
// (synth.Platform uses the Kernel's *cue.Context; NewPlatformFromValue takes
// none). It is part of the signature for parity with [Kernel.SynthesizeInstance]
// and so a future materialize-aware variant can honor cancellation without an
// API break. Keep it.
func (k *Kernel) SynthesizePlatform(_ context.Context, in synth.PlatformInput) (*platform.Platform, error) {
	if in.SchemaCache == nil {
		in.SchemaCache = k.schemaCache
	}
	spec, err := synth.Platform(k.cueCtx, in)
	if err != nil {
		return nil, fmt.Errorf("Kernel.SynthesizePlatform: %w", err)
	}
	plat, err := platform.NewPlatformFromValue(k, spec)
	if err != nil {
		return nil, fmt.Errorf("Kernel.SynthesizePlatform: %w", err)
	}
	return plat, nil
}
