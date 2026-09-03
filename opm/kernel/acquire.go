package kernel

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"

	loaderfile "github.com/open-platform-model/library/opm/helper/loader/file"
	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/platform"
)

// AcquirePlatformFromDir loads a #Platform CUE package from a directory and
// returns it as a typed, source-carrying [*platform.Platform]. It composes
// [Kernel.LoadPlatformPackage] (evaluation and the platform shape gate,
// identical to a direct call) with [platform.NewPlatformFromValue], then
// stamps [platform.Platform.Source] in on-disk mode: Root is the enclosing
// module root (the nearest ancestor holding cue.mod/module.cue, the directory
// itself when it is the root), Pkg the package directory relative to it, and
// Overlay nil.
//
// It is the directory peer of [Kernel.AcquireModuleFromRegistry] ("Acquire"
// returns a typed artifact that knows where its source lives) and the
// recommended entry point when the platform will be imported as a package by
// a follow-on build. A caller that wants only the raw value keeps using
// [Kernel.LoadPlatformPackage]. opts.Registry, when non-empty, is applied via
// the load configuration's environment, never os.Setenv, exactly as the
// loader applies it.
//
// Loader failures propagate unchanged (missing directory, no package, or a
// shape-gate sentinel such as [loaderfile.ErrWrongKind]); no partial platform
// is returned.
func (k *Kernel) AcquirePlatformFromDir(ctx context.Context, dirPath string, opts loaderfile.LoadOptions) (*platform.Platform, error) {
	absDir, err := filepath.Abs(dirPath)
	if err != nil {
		return nil, fmt.Errorf("Kernel.AcquirePlatformFromDir: resolving platform directory: %w", err)
	}
	val, err := k.LoadPlatformPackage(ctx, absDir, opts)
	if err != nil {
		return nil, err
	}
	plat, err := platform.NewPlatformFromValue(k, val)
	if err != nil {
		return nil, fmt.Errorf("Kernel.AcquirePlatformFromDir: %w", err)
	}
	plat.Source = sourceForDir(absDir)
	return plat, nil
}

// AcquireInstanceFromDir loads a #ModuleInstance CUE package from a directory
// and returns it as a validated, source-carrying [*module.Instance]. It
// composes [Kernel.LoadInstancePackage] (evaluation and the instance shape
// gate) with [Kernel.ProcessModuleInstance] — the validated entry point, called
// with no extra values, so the package must already be fully concrete, as an
// authored instance package is — then stamps [module.Instance.Source] in
// on-disk mode: Overlay is nil, Root is the enclosing module root (the
// nearest ancestor holding cue.mod/module.cue, the directory itself when it
// is the root) and Pkg the package directory relative to it, so a
// package in a subdirectory of its module imports correctly from a follow-on
// build.
//
// This is the same bar [Kernel.SynthesizeInstance] output meets, and the
// recommended entry point when the instance will be imported as a package by a
// follow-on build. Draft flows that need an unvalidated value keep using
// [Kernel.LoadInstancePackage]. opts.Registry is applied as the loader applies
// it (load configuration environment, never os.Setenv).
//
// Loader failures propagate unchanged (missing directory, no package, or a
// shape-gate sentinel); a non-concrete package surfaces the validation error
// [Kernel.ProcessModuleInstance] produces. No partial instance is returned.
func (k *Kernel) AcquireInstanceFromDir(ctx context.Context, dirPath string, opts loaderfile.LoadOptions) (*module.Instance, error) {
	absDir, err := filepath.Abs(dirPath)
	if err != nil {
		return nil, fmt.Errorf("Kernel.AcquireInstanceFromDir: resolving instance directory: %w", err)
	}
	spec, err := k.LoadInstancePackage(ctx, absDir, opts)
	if err != nil {
		return nil, err
	}
	inst, err := k.ProcessModuleInstance(ctx, spec, module.Module{}, cue.Value{})
	if err != nil {
		return nil, fmt.Errorf("Kernel.AcquireInstanceFromDir: %w", err)
	}
	inst.Source = sourceForDir(absDir)
	return inst, nil
}

// sourceForDir describes an on-disk package directory as a Source: Root is
// the nearest ancestor (the directory itself included) holding
// cue.mod/module.cue, Pkg the slash-separated path of dir relative to it.
// A directory with no enclosing module is its own root with an empty Pkg,
// which is what a module-less package loads as today.
func sourceForDir(absDir string) *module.Source {
	dir := absDir
	for {
		if _, err := os.Stat(filepath.Join(dir, "cue.mod", "module.cue")); err == nil {
			rel, err := filepath.Rel(dir, absDir)
			if err != nil {
				break
			}
			if rel == "." {
				rel = ""
			}
			return &module.Source{Root: dir, Pkg: filepath.ToSlash(rel)}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return &module.Source{Root: absDir}
}
