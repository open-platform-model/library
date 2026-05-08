package kernel

import (
	"context"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/pkg/apiversion"
	oerrors "github.com/open-platform-model/library/pkg/errors"
	loaderfile "github.com/open-platform-model/library/pkg/helper/loader/file"
	"github.com/open-platform-model/library/pkg/module"
	"github.com/open-platform-model/library/pkg/platform"
	"github.com/open-platform-model/library/pkg/validate"
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

// ParseModuleRelease validates values and constructs a concrete
// [*module.Release]. See [module.ParseModuleRelease].
func (k *Kernel) ParseModuleRelease(ctx context.Context, spec cue.Value, mod module.Module, values []cue.Value) (*module.Release, error) {
	return module.ParseModuleRelease(ctx, spec, mod, values) //nolint:staticcheck // SA1019: kernel method wraps the deprecated free function
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

// ValidateConfig validates supplied values against a #config schema and
// returns the merged value or a [*oerrors.ConfigError]. See [validate.Config].
func (k *Kernel) ValidateConfig(schema cue.Value, values []cue.Value, contextLabel, name string) (cue.Value, *oerrors.ConfigError) {
	return validate.Config(schema, values, contextLabel, name) //nolint:staticcheck // SA1019: kernel method wraps the deprecated free function
}
