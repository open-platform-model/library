package kernel

import (
	"context"
	"fmt"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/helper/synth"
	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/platform"
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
	// The Kernel owns the cache; callers MUST NOT need to thread it
	// through SynthesizeRelease explicitly. If they did set SchemaCache,
	// honor it (a test may pin a different one), otherwise fall back to
	// the kernel-owned cache.
	if in.SchemaCache == nil {
		in.SchemaCache = k.schemaCache
	}
	spec, err := synth.Release(k.cueCtx, in)
	if err != nil {
		return nil, fmt.Errorf("Kernel.SynthesizeRelease: %w", err)
	}
	// synth.Release bakes in.Values into the single build (as values.cue), so
	// the spec already carries them — exactly like an authored release.cue
	// package. Re-filling here would write values a second time into the now-set
	// `values` path and conflict. Pass the zero value: ProcessModuleRelease then
	// validates concreteness and decodes metadata without re-filling, the same
	// way it processes a file-loaded release whose values live in the package.
	rel, err := k.ProcessModuleRelease(ctx, spec, *in.Module, cue.Value{})
	if err != nil {
		return nil, fmt.Errorf("Kernel.SynthesizeRelease: %w", err)
	}
	return rel, nil
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
// none). It is part of the signature for parity with [Kernel.SynthesizeRelease]
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
