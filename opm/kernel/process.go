package kernel

import (
	"fmt"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/internal/corepath"
	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/schema"
)

// processInstance asserts a built instance spec is fully concrete, decodes
// its metadata and returns a constructed [*module.Instance]. It is the one
// processing step behind [Kernel.AcquireInstanceFromDir] and
// [Kernel.SynthesizeInstance]: values are already unified inside the CUE
// build each of them runs (the package's own `values`, the acquire-time values
// overlay, the synthesized values file), so nothing is filled here and nothing
// on the Kernel is read.
//
// Concreteness is checked twice. The whole built spec first, so a value the
// instance carries in a non-concrete form is refused at its place in the build.
// Then the spec's `values` unified with the module's #config (both read off
// the spec): #ModuleInstance only unifies `values` into a let binding the
// components read, so a required #config value the values leave unset, and
// no component reads, never reaches the spec; this check refuses it at the
// path `values.<field>` (library#211). Both failures are framed
// `instance "<name>": not fully concrete: …`; other errors
// `instance "<name>": …`.
//
// The returned Instance carries no Source; the caller stamps it.
func processInstance(spec cue.Value) (*module.Instance, error) {
	name := bestEffortInstanceName(spec)

	if err := spec.Validate(cue.Concrete(true)); err != nil {
		return nil, fmt.Errorf("instance %q: not fully concrete: %w", name, err)
	}
	if err := requiredConfigSet(spec); err != nil {
		return nil, fmt.Errorf("instance %q: not fully concrete: %w", name, err)
	}

	meta, err := decodeInstanceMetadata(spec)
	if err != nil {
		return nil, fmt.Errorf("instance %q: %w", name, err)
	}

	return &module.Instance{
		Metadata: meta,
		Package:  spec,
	}, nil
}

// requiredConfigSet validates the built spec's `values` unified with the
// module's #config under concreteness, so a required #config value the
// values leave unset is refused whether or not a component reads it. The
// values come first in the unification, so CUE reports each finding at the
// path `values.<field>`, positioned at the field's #config declaration where
// CUE records one. It is the concreteness rule [Kernel.ValidateConfigDetailed]
// applies; the disallowed-field walk and type checks are not repeated, the
// per-source checks both verbs run before processInstance have done them. A
// spec without #config or `values` has nothing to check.
func requiredConfigSet(spec cue.Value) error {
	configSchema := spec.LookupPath(schema.Module).LookupPath(corepath.Config)
	built := spec.LookupPath(corepath.Values)
	if !configSchema.Exists() || !built.Exists() {
		return nil
	}
	return built.Unify(configSchema).Validate(cue.Concrete(true))
}

// decodeInstanceMetadata extracts the instance metadata from a
// #ModuleInstance artifact root. A missing metadata field is fatal.
func decodeInstanceMetadata(v cue.Value) (*module.InstanceMetadata, error) {
	metaVal := v.LookupPath(schema.Metadata)
	if !metaVal.Exists() {
		return nil, fmt.Errorf("instance metadata field is required")
	}
	meta := &module.InstanceMetadata{}
	if err := metaVal.Decode(meta); err != nil {
		return nil, fmt.Errorf("decoding instance metadata: %w", err)
	}
	return meta, nil
}

// bestEffortInstanceName reads the instance name for error framing. The
// framing is a diagnostic and must not itself fail, so a spec whose name is
// not a concrete string yields a placeholder.
func bestEffortInstanceName(spec cue.Value) string {
	nameVal := spec.LookupPath(schema.Metadata).LookupPath(cue.ParsePath("name"))
	if nameVal.Exists() {
		if s, err := nameVal.String(); err == nil {
			return s
		}
	}
	return "<unknown>"
}
