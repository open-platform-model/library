package kernel

import (
	"context"

	"cuelang.org/go/cue"

	loaderfile "github.com/open-platform-model/library/opm/helper/loader/file"
	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/platform"
)

// LoadModulePackage loads a module CUE package from a directory using the
// kernel's [*cue.Context]. See [loaderfile.LoadModulePackage].
func (k *Kernel) LoadModulePackage(_ context.Context, dirPath string, opts loaderfile.LoadOptions) (cue.Value, error) {
	return loaderfile.LoadModulePackage(k.cueCtx, dirPath, opts)
}

// LoadReleasePackage loads a #ModuleRelease CUE package from a directory
// using the kernel's [*cue.Context]. See [loaderfile.LoadReleasePackage].
func (k *Kernel) LoadReleasePackage(_ context.Context, dirPath string, opts loaderfile.LoadOptions) (cue.Value, error) {
	return loaderfile.LoadReleasePackage(k.cueCtx, dirPath, opts)
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

// NewReleaseFromValue builds a typed [*module.Release] from a raw
// [cue.Value]. See [module.NewReleaseFromValue].
func (k *Kernel) NewReleaseFromValue(v cue.Value) (*module.Release, error) {
	return module.NewReleaseFromValue(k, v)
}
