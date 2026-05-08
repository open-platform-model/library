package module

import (
	"context"
	"fmt"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/pkg/api"
	"github.com/open-platform-model/library/pkg/validate"
)

// ParseModuleRelease validates values, fills them into the release spec,
// ensures the result is concrete, decodes metadata, and constructs Release.
//
// The release spec carries its source #Module inside the CUE package; the
// schema for value validation is read from spec via the binding's Module +
// Config paths. mod is supplied for fallback diagnostics (release name when
// metadata.name is not yet concrete) and is not retained on the returned
// Release — the source module remains reachable through Release.Package.
//
// values is a single, pre-unified cue.Value — layering happens in the
// frontend / helper before this call. The zero cue.Value{} is treated as
// "no values supplied".
//
// Deprecated: use Kernel.ParseModuleRelease. The Kernel is the public anchor
// type for all OPM runtime operations.
func ParseModuleRelease(_ context.Context, spec cue.Value, mod Module, values cue.Value) (*Release, error) {
	// Best-effort name for error messages — metadata.name may already be
	// concrete before values filling (it comes from the module definition).
	name := bestEffortReleaseName(spec, mod)

	b, err := api.Lookup(mod.APIVersion)
	if err != nil {
		return nil, fmt.Errorf("release %q: resolving binding for %q: %w", name, mod.APIVersion, err)
	}
	paths := b.Paths()

	// The #config schema lives on the source module: read it from the release
	// spec's #module reference so Package stays the source of truth.
	schema := spec.LookupPath(paths.Module).LookupPath(paths.Config)

	// Tier-2 schema validation against the module's #config.
	validated, cfgErr := validate.Config(schema, values, "module", name) //nolint:staticcheck // SA1019: parse.go is itself the deprecated path called by Kernel.ParseModuleRelease
	if cfgErr != nil {
		return nil, cfgErr
	}

	// Fill validated values into the release spec.
	if validated.Exists() {
		spec = spec.FillPath(paths.Values, validated)
		if err := spec.Err(); err != nil {
			return nil, fmt.Errorf("filling values into release spec: %w", err)
		}
	}

	// Validate the filled spec is fully concrete.
	if err := spec.Validate(cue.Concrete(true)); err != nil {
		return nil, fmt.Errorf("release %q: not fully concrete: %w", name, err)
	}

	// Decode release metadata from the concrete spec via the binding so the
	// release shape stays consistent across versions.
	apiMeta, err := b.DecodeReleaseMetadata(spec)
	if err != nil {
		return nil, fmt.Errorf("release %q: %w", name, err)
	}

	return &Release{
		APIVersion: mod.APIVersion,
		Metadata:   releaseMetadataFromAPI(apiMeta),
		Package:    spec,
	}, nil
}

// bestEffortReleaseName tries to extract a release name for error messages.
// Falls back to the module name if the release name is not yet available.
func bestEffortReleaseName(spec cue.Value, mod Module) string {
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
