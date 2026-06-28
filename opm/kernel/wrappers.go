package kernel

import (
	"context"

	"cuelang.org/go/cue"

	loaderfile "github.com/open-platform-model/library/opm/helper/loader/file"
	loaderregistry "github.com/open-platform-model/library/opm/helper/loader/registry"
	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/platform"
)

// LoadModulePackage loads a module CUE package from a directory using the
// kernel's [*cue.Context]. See [loaderfile.LoadModulePackage].
func (k *Kernel) LoadModulePackage(_ context.Context, dirPath string, opts loaderfile.LoadOptions) (cue.Value, error) {
	return loaderfile.LoadModulePackage(k.cueCtx, dirPath, opts)
}

// LoadModuleFromRegistry loads a #Module published in an OCI registry by its
// major-qualified path (e.g. "example.com/modules/hello@v0") and version (e.g.
// "v0.0.2"), using the kernel's [*cue.Context] and configured registry (set via
// [WithRegistry], inheriting CUE_REGISTRY from the process environment when
// unset). It returns the raw module [cue.Value]; callers decode it via
// [Kernel.NewModuleFromValue], mirroring [Kernel.LoadModulePackage]'s two-step
// load→decode contract. See [loaderregistry.LoadModulePackage].
func (k *Kernel) LoadModuleFromRegistry(ctx context.Context, modPath, version string) (cue.Value, error) {
	return loaderregistry.LoadModulePackage(ctx, k.cueCtx, modPath, version, loaderregistry.LoadOptions{Registry: k.registry})
}

// AcquireModuleFromRegistry loads a #Module published in an OCI registry (same
// fetch + main-module staging + shape gate as [Kernel.LoadModuleFromRegistry])
// and returns a decoded [*module.Module] whose staged source ([module.Source])
// is populated, so the module can be reused as the main module of a follow-on
// build — notably by [Kernel.SynthesizeInstance], which stages the instance
// inside the module's own root so the module's already-tidied
// cue.mod/module.cue drives transitive dependency resolution. Unlike the
// two-step [Kernel.LoadModuleFromRegistry] → [Kernel.NewModuleFromValue] path
// (which discards the staged source at the cue.Value boundary), this single
// call preserves it without a second registry fetch.
func (k *Kernel) AcquireModuleFromRegistry(ctx context.Context, modPath, version string) (*module.Module, error) {
	res, err := loaderregistry.LoadModulePackageWithSource(ctx, k.cueCtx, modPath, version, loaderregistry.LoadOptions{Registry: k.registry})
	if err != nil {
		return nil, err
	}
	mod, err := module.NewModuleFromValue(k, res.Value)
	if err != nil {
		return nil, err
	}
	mod.Source = &module.Source{Root: res.Root, Overlay: res.Overlay}
	return mod, nil
}

// LoadInstancePackage loads a #ModuleInstance CUE package from a directory
// using the kernel's [*cue.Context]. See [loaderfile.LoadInstancePackage].
//
// Was: LoadReleasePackage
func (k *Kernel) LoadInstancePackage(_ context.Context, dirPath string, opts loaderfile.LoadOptions) (cue.Value, error) {
	return loaderfile.LoadInstancePackage(k.cueCtx, dirPath, opts)
}

// LoadPlatformPackage loads a #Platform CUE package from a directory using
// the kernel's [*cue.Context]. See [loaderfile.LoadPlatformPackage].
func (k *Kernel) LoadPlatformPackage(_ context.Context, dirPath string, opts loaderfile.LoadOptions) (cue.Value, error) {
	return loaderfile.LoadPlatformPackage(k.cueCtx, dirPath, opts)
}

// NewPlatformFromValue builds a typed [*platform.Platform] from a raw
// [cue.Value]. See [platform.NewPlatformFromValue].
func (k *Kernel) NewPlatformFromValue(v cue.Value) (*platform.Platform, error) {
	return platform.NewPlatformFromValue(k, v)
}

// NewModuleFromValue builds a typed [*module.Module] from a raw [cue.Value].
// See [module.NewModuleFromValue].
func (k *Kernel) NewModuleFromValue(v cue.Value) (*module.Module, error) {
	return module.NewModuleFromValue(k, v)
}

// NewInstanceFromValue builds a typed [*module.Instance] from a raw
// [cue.Value]. See [module.NewInstanceFromValue].
//
// Was: NewReleaseFromValue
func (k *Kernel) NewInstanceFromValue(v cue.Value) (*module.Instance, error) {
	return module.NewInstanceFromValue(k, v)
}
