package kernel

import (
	"context"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/pkg/apiversion"
	loaderfile "github.com/open-platform-model/library/pkg/helper/loader/file"
	helperplatform "github.com/open-platform-model/library/pkg/helper/platform"
	helpervalues "github.com/open-platform-model/library/pkg/helper/values"
	"github.com/open-platform-model/library/pkg/module"
	"github.com/open-platform-model/library/pkg/platform"
)

// LoadModulePackage loads a module CUE package from a directory using the
// kernel's [*cue.Context]. See [loaderfile.LoadModulePackage].
func (k *Kernel) LoadModulePackage(_ context.Context, dirPath string) (cue.Value, apiversion.Version, error) {
	return loaderfile.LoadModulePackage(k.cueCtx, dirPath)
}

// LoadReleaseFile loads a #ModuleRelease from a standalone .cue file using
// the kernel's [*cue.Context]. See [loaderfile.LoadReleaseFile].
func (k *Kernel) LoadReleaseFile(_ context.Context, filePath string, opts loaderfile.LoadOptions) (cue.Value, string, apiversion.Version, error) {
	return loaderfile.LoadReleaseFile(k.cueCtx, filePath, opts)
}

// LoadValuesFile loads a standalone CUE values file using the kernel's
// [*cue.Context]. See [loaderfile.LoadValuesFile].
func (k *Kernel) LoadValuesFile(_ context.Context, path string) (cue.Value, error) {
	return loaderfile.LoadValuesFile(k.cueCtx, path)
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

// ValidateAndUnify performs Tier-1 source-positioned validation on each
// [helpervalues.Layer] in the [helpervalues.Stack], then unifies in order
// on success. Per-layer failures aggregate into a
// [*helpervalues.MultiSourceError] so frontends can render diagnostics
// with per-source attribution. The kernel's Tier-2 validation (see
// [Kernel.ValidateConfig]) remains the safety net; pass the unified value
// returned here to the kernel for Tier-2.
//
// Canonical implementation lives in [helpervalues.ValidateAndUnify]; this
// method is an ergonomic shortcut so readers anchored on [Kernel] discover
// the helper through the same surface.
func (k *Kernel) ValidateAndUnify(schema cue.Value, layers helpervalues.Stack) (cue.Value, *helpervalues.MultiSourceError) {
	return helpervalues.ValidateAndUnify(k, schema, layers)
}
