package kernel

import (
	"context"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/pkg/apiversion"
	"github.com/open-platform-model/library/pkg/compile"
	oerrors "github.com/open-platform-model/library/pkg/errors"
	loaderfile "github.com/open-platform-model/library/pkg/helper/loader/file"
	"github.com/open-platform-model/library/pkg/module"
	"github.com/open-platform-model/library/pkg/provider"
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

// LoadProvider selects and wraps a provider from a pre-loaded providers map.
// See [loaderfile.LoadProvider].
func (k *Kernel) LoadProvider(providerName string, providers map[string]cue.Value) (*provider.Provider, error) {
	return loaderfile.LoadProvider(providerName, providers)
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

// NewRenderModule constructs a compile Module for the given provider and
// runtime identity. See [compile.NewModule].
func (k *Kernel) NewRenderModule(p *provider.Provider, runtimeName string) *compile.Module {
	return compile.NewModule(p, runtimeName) //nolint:staticcheck // SA1019: kernel method wraps the deprecated free function
}

// ProcessModuleRelease renders a prepared release with the given provider.
//
// Deprecated: use [Kernel.Compile].
func (k *Kernel) ProcessModuleRelease(ctx context.Context, rel *module.Release, p *provider.Provider, runtimeName string) (*compile.CompileResult, error) {
	return compile.CompileModuleRelease(ctx, rel, p, runtimeName) //nolint:staticcheck // SA1019: kernel methods are thin shims over the deprecated free functions
}

// ValidateConfig validates supplied values against a #config schema and
// returns the merged value or a [*oerrors.ConfigError]. See [validate.Config].
func (k *Kernel) ValidateConfig(schema cue.Value, values []cue.Value, contextLabel, name string) (cue.Value, *oerrors.ConfigError) {
	return validate.Config(schema, values, contextLabel, name) //nolint:staticcheck // SA1019: kernel method wraps the deprecated free function
}
