package kernel

import (
	"context"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/pkg/apiversion"
	oerrors "github.com/open-platform-model/library/pkg/errors"
	"github.com/open-platform-model/library/pkg/loader"
	"github.com/open-platform-model/library/pkg/module"
	"github.com/open-platform-model/library/pkg/provider"
	"github.com/open-platform-model/library/pkg/render"
	"github.com/open-platform-model/library/pkg/validate"
)

// LoadModulePackage loads a module CUE package from a directory using the
// kernel's [*cue.Context]. See [loader.LoadModulePackage].
func (k *Kernel) LoadModulePackage(_ context.Context, dirPath string) (cue.Value, apiversion.Version, error) {
	return loader.LoadModulePackage(k.cueCtx, dirPath) //nolint:staticcheck // SA1019: kernel methods are thin shims over the deprecated free functions during slice 01
}

// LoadReleaseFile loads a #ModuleRelease from a standalone .cue file using
// the kernel's [*cue.Context]. See [loader.LoadReleaseFile].
func (k *Kernel) LoadReleaseFile(_ context.Context, filePath string, opts loader.LoadOptions) (cue.Value, string, apiversion.Version, error) {
	return loader.LoadReleaseFile(k.cueCtx, filePath, opts) //nolint:staticcheck // SA1019: kernel methods are thin shims over the deprecated free functions during slice 01
}

// LoadValuesFile loads a standalone CUE values file using the kernel's
// [*cue.Context]. See [loader.LoadValuesFile].
func (k *Kernel) LoadValuesFile(_ context.Context, path string) (cue.Value, error) {
	return loader.LoadValuesFile(k.cueCtx, path) //nolint:staticcheck // SA1019: kernel methods are thin shims over the deprecated free functions during slice 01
}

// LoadProvider selects and wraps a provider from a pre-loaded providers map.
// See [loader.LoadProvider].
func (k *Kernel) LoadProvider(providerName string, providers map[string]cue.Value) (*provider.Provider, error) {
	return loader.LoadProvider(providerName, providers) //nolint:staticcheck // SA1019: kernel methods are thin shims over the deprecated free functions during slice 01
}

// ParseModuleRelease validates values and constructs a concrete
// [*module.Release]. See [module.ParseModuleRelease].
func (k *Kernel) ParseModuleRelease(ctx context.Context, spec cue.Value, mod module.Module, values []cue.Value) (*module.Release, error) {
	return module.ParseModuleRelease(ctx, spec, mod, values) //nolint:staticcheck // SA1019: kernel methods are thin shims over the deprecated free functions during slice 01
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

// NewRenderModule constructs a render Module for the given provider and
// runtime identity. See [render.NewModule].
func (k *Kernel) NewRenderModule(p *provider.Provider, runtimeName string) *render.Module {
	return render.NewModule(p, runtimeName) //nolint:staticcheck // SA1019: kernel methods are thin shims over the deprecated free functions during slice 01
}

// ProcessModuleRelease renders a prepared release with the given provider.
// See [render.ProcessModuleRelease].
func (k *Kernel) ProcessModuleRelease(ctx context.Context, rel *module.Release, p *provider.Provider, runtimeName string) (*render.ModuleResult, error) {
	return render.ProcessModuleRelease(ctx, rel, p, runtimeName) //nolint:staticcheck // SA1019: kernel methods are thin shims over the deprecated free functions during slice 01
}

// ValidateConfig validates supplied values against a #config schema and
// returns the merged value or a [*oerrors.ConfigError]. See [validate.Config].
func (k *Kernel) ValidateConfig(schema cue.Value, values []cue.Value, contextLabel, name string) (cue.Value, *oerrors.ConfigError) {
	return validate.Config(schema, values, contextLabel, name) //nolint:staticcheck // SA1019: kernel methods are thin shims over the deprecated free functions during slice 01
}
