package kernel

import (
	"context"
	"fmt"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/pkg/api"
	"github.com/open-platform-model/library/pkg/module"
)

// ProcessModuleRelease validates the supplied values, fills them into the
// release spec, asserts the result is fully concrete, decodes release
// metadata via the version binding, and returns a constructed
// [*module.Release].
//
// The release spec carries its source #Module inside the CUE package; the
// schema for value validation is read from spec via the binding's Module +
// Config paths. mod is supplied for fallback diagnostics (release name when
// metadata.name is not yet concrete) and is not retained on the returned
// Release — the source module remains reachable through Release.Package.
//
// values is a single, pre-unified [cue.Value] — layering happens in the
// frontend / helper before this call (see [pkg/helper/values.ValidateAndUnify]).
// The zero [cue.Value] is treated as "no values supplied": validation is
// skipped, no fill is performed, and the spec must already be concrete on
// every required field.
func (k *Kernel) ProcessModuleRelease(_ context.Context, spec cue.Value, mod module.Module, values cue.Value) (*module.Release, error) {
	name := bestEffortReleaseName(spec, mod)

	b, err := api.Lookup(mod.APIVersion)
	if err != nil {
		return nil, fmt.Errorf("release %q: resolving binding for %q: %w", name, mod.APIVersion, err)
	}
	paths := b.Paths()

	schema := spec.LookupPath(paths.Module).LookupPath(paths.Config)

	validated, cfgErr := runValidate(schema, values, "module", name, true)
	if cfgErr != nil {
		return nil, cfgErr
	}

	if validated.Exists() {
		spec = spec.FillPath(paths.Values, validated)
		if err := spec.Err(); err != nil {
			return nil, fmt.Errorf("filling values into release spec: %w", err)
		}
	}

	if err := spec.Validate(cue.Concrete(true)); err != nil {
		return nil, fmt.Errorf("release %q: not fully concrete: %w", name, err)
	}

	apiMeta, err := b.DecodeReleaseMetadata(spec)
	if err != nil {
		return nil, fmt.Errorf("release %q: %w", name, err)
	}

	return &module.Release{
		APIVersion: mod.APIVersion,
		Metadata:   module.ReleaseMetadataFromAPI(apiMeta),
		Package:    spec,
	}, nil
}

// ParseModuleRelease is the previous name for [Kernel.ProcessModuleRelease].
//
// Deprecated: use [Kernel.ProcessModuleRelease]. The verb "Process" better
// describes what the method does (validate values, fill spec, check
// concreteness, decode metadata) than "Parse". This alias will be removed
// in a future MAJOR release.
func (k *Kernel) ParseModuleRelease(ctx context.Context, spec cue.Value, mod module.Module, values cue.Value) (*module.Release, error) {
	return k.ProcessModuleRelease(ctx, spec, mod, values)
}

// bestEffortReleaseName tries to extract a release name for error messages.
// Falls back to the module name if the release name is not yet available.
func bestEffortReleaseName(spec cue.Value, mod module.Module) string {
	nameVal := spec.LookupPath(cue.ParsePath("metadata.name"))
	if nameVal.Exists() {
		if s, err := nameVal.String(); err == nil {
			return s
		}
	}
	if mod.Metadata != nil && mod.Metadata.Name != "" {
		return mod.Metadata.Name
	}
	return "<unknown>"
}
