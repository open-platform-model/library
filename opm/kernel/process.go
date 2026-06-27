package kernel

import (
	"context"
	"fmt"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/schema"
)

// ProcessModuleInstance validates the supplied values, fills them into the
// instance spec, asserts the result is fully concrete, decodes instance
// metadata via opm/schema, and returns a constructed [*module.Instance].
//
// The instance spec carries its source #Module inside the CUE package; the
// schema for value validation is read from spec via schema.Module +
// schema.Config. mod is supplied for fallback diagnostics (instance name
// when metadata.name is not yet concrete) and is not retained on the
// returned Instance — the source module remains reachable through
// Instance.Package.
//
// values is a single, pre-unified [cue.Value] — layering is performed by
// callers via [Kernel.ValidateConfigDetailed] before this call. The zero
// [cue.Value] is treated as "no values supplied": validation is skipped,
// no fill is performed, and the spec must already be concrete on every
// required field.
//
// Was: ProcessModuleRelease
func (k *Kernel) ProcessModuleInstance(_ context.Context, spec cue.Value, mod module.Module, values cue.Value) (*module.Instance, error) {
	name := bestEffortInstanceName(spec, mod)

	configSchema := spec.LookupPath(schema.Module).LookupPath(schema.Config)

	validated, vErr := k.ValidateConfig(configSchema, values)
	if vErr != nil {
		return nil, fmt.Errorf("instance %q: %w", name, vErr)
	}

	if validated.Exists() {
		spec = spec.FillPath(schema.Values, validated)
		if err := spec.Err(); err != nil {
			return nil, fmt.Errorf("filling values into instance spec: %w", err)
		}
	}

	if err := spec.Validate(cue.Concrete(true)); err != nil {
		return nil, fmt.Errorf("instance %q: not fully concrete: %w", name, err)
	}

	meta, err := schema.DecodeInstanceMetadata(spec)
	if err != nil {
		return nil, fmt.Errorf("instance %q: %w", name, err)
	}

	return &module.Instance{
		Metadata: meta,
		Package:  spec,
	}, nil
}

// bestEffortInstanceName tries to extract an instance name for error messages.
// Falls back to the module name if the instance name is not yet available.
func bestEffortInstanceName(spec cue.Value, mod module.Module) string {
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
