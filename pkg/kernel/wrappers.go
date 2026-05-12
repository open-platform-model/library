package kernel

import (
	"context"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/pkg/apiversion"
	loaderfile "github.com/open-platform-model/library/pkg/helper/loader/file"
	helperplatform "github.com/open-platform-model/library/pkg/helper/platform"
	"github.com/open-platform-model/library/pkg/module"
	"github.com/open-platform-model/library/pkg/platform"
)

// LoadModulePackage loads a module CUE package from a directory using the
// kernel's [*cue.Context]. See [loaderfile.LoadModulePackage].
func (k *Kernel) LoadModulePackage(_ context.Context, dirPath string, opts loaderfile.LoadOptions) (cue.Value, apiversion.Version, error) {
	return loaderfile.LoadModulePackage(k.cueCtx, dirPath, opts)
}

// LoadReleasePackage loads a #ModuleRelease CUE package from a directory
// using the kernel's [*cue.Context]. See [loaderfile.LoadReleasePackage].
func (k *Kernel) LoadReleasePackage(_ context.Context, dirPath string, opts loaderfile.LoadOptions) (cue.Value, apiversion.Version, error) {
	return loaderfile.LoadReleasePackage(k.cueCtx, dirPath, opts)
}

// LoadPlatformFile loads a #Platform from a standalone .cue file (or from a
// directory containing platform.cue) using the kernel's [*cue.Context].
// See [loaderfile.LoadPlatformFile].
func (k *Kernel) LoadPlatformFile(_ context.Context, path string, opts loaderfile.LoadOptions) (cue.Value, string, error) {
	return loaderfile.LoadPlatformFile(k.cueCtx, path, opts)
}

// NewPlatformFromValue builds a typed [*platform.Platform] from a raw
// [cue.Value] using the version-aware binding registry.
// See [platform.NewPlatformFromValue].
func (k *Kernel) NewPlatformFromValue(v cue.Value) (*platform.Platform, error) {
	return platform.NewPlatformFromValue(k, v)
}

// ComposePlatform returns a new [*platform.Platform] with the given Modules
// FillPath-injected into shell.Package at the binding's Registry path.
// See [helperplatform.Compose] for the full contract, including the
// multi-fulfiller error surface.
func (k *Kernel) ComposePlatform(shell *platform.Platform, modules []*module.Module) (*platform.Platform, error) {
	return helperplatform.Compose(k, shell, modules)
}

// NewModuleFromValue builds a typed [*module.Module] from a raw [cue.Value]
// using the version-aware binding registry. See [module.NewModuleFromValue].
func (k *Kernel) NewModuleFromValue(v cue.Value) (*module.Module, error) {
	return module.NewModuleFromValue(k, v)
}

// NewReleaseFromValue builds a typed [*module.Release] from a raw [cue.Value]
// using the version-aware binding registry. See [module.NewReleaseFromValue].
func (k *Kernel) NewReleaseFromValue(v cue.Value) (*module.Release, error) {
	return module.NewReleaseFromValue(k, v)
}
