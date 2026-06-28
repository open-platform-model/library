package file

import (
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"

	"github.com/open-platform-model/library/opm/helper/loader/internal/shape"
)

// buildAndShapeGate is the single evaluate-and-shape-gate step shared by the
// directory loaders (LoadInstancePackage et al., filesystem source) and
// synth.Instance (in-memory overlay source). It builds exactly one CUE package
// rooted at root, builds it in ctx, and runs the artifact shape gate described
// by spec. The two source modes are selected by overlay:
//
//   - overlay == nil  → on-disk package: load.Config.Dir is root, files are
//     read from the filesystem.
//   - overlay != nil  → in-memory package: the overlay supplies every file
//     under root and root doubles as the module root, so the synthesized
//     cue.mod/module.cue drives transitive dependency resolution (mirrors the
//     registry loader's overlay strategy — see opm/helper/loader/registry).
//
// env, when non-nil, is the environment slice load.Config consults (used to
// override CUE_REGISTRY without mutating process state via registryEnv); nil
// falls back to the process environment unchanged.
//
// Keeping this routine single-sourced guarantees an overlay-built artifact and
// an on-disk artifact are evaluated, shape-gated, and error-wrapped
// identically: the only difference between the two entry points is where the
// package files come from.
func buildAndShapeGate(ctx *cue.Context, root, pkg string, overlay map[string]load.Source, env []string, spec shape.ArtifactSpec) (cue.Value, error) {
	if pkg == "" {
		pkg = "."
	}
	cfg := &load.Config{
		Dir: root,
		Env: env,
	}
	if overlay != nil {
		cfg.Overlay = overlay
		cfg.ModuleRoot = root
	}

	instances := load.Instances([]string{pkg}, cfg)
	if len(instances) != 1 {
		return cue.Value{}, fmt.Errorf("expected exactly one CUE package in %s (%s), found %d: %w", root, pkg, len(instances), ErrInvalidPackage)
	}
	if instances[0].Err != nil {
		return cue.Value{}, fmt.Errorf("loading package from %s (%s): %w", root, pkg, instances[0].Err)
	}

	val := ctx.BuildInstance(instances[0])
	if err := val.Err(); err != nil {
		return cue.Value{}, fmt.Errorf("building package from %s: %w", root, err)
	}

	if err := shapeGate(val, spec); err != nil {
		return cue.Value{}, fmt.Errorf("validating package in %s: %w", root, err)
	}

	return val, nil
}

// BuildInstanceOverlayAt evaluates an in-memory #ModuleInstance package that
// lives in a SUBDIRECTORY (pkg, e.g. "./opm-synth-instance") of a module rooted
// at moduleRoot, supplied as a load.Source overlay. moduleRoot is the
// load.Config.ModuleRoot, so the overlay's cue.mod/module.cue at that root —
// typically a registry-staged module's own already-tidied module file — drives
// transitive dependency resolution, while the instance package itself is loaded
// from the subdirectory. Pass pkg "." to load the package at the module root.
//
// This is the entry point synth.Instance uses to construct an instance INSIDE
// the acquired module's own staged source tree (so the module import resolves
// locally and the module's tidied closure is reused), rather than fabricating a
// standalone module. It runs the same build-and-shape-gate step (shape.InstanceSpec)
// as the on-disk LoadInstancePackage. opts.Registry, when non-empty, overrides
// CUE_REGISTRY via load.Config.Env (never os.Setenv), so it is safe under
// concurrency.
//
// Was: BuildInstanceOverlay / BuildReleaseOverlay
func BuildInstanceOverlayAt(ctx *cue.Context, moduleRoot, pkg string, overlay map[string]load.Source, opts LoadOptions) (cue.Value, error) {
	return buildAndShapeGate(ctx, moduleRoot, pkg, overlay, registryEnv(opts.Registry), instanceSpec)
}
