package kernel

import (
	"fmt"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/schema"
)

// processInstance asserts a built instance spec is fully concrete, decodes
// its metadata and returns a constructed [*module.Instance]. It is the one
// processing step behind [Kernel.AcquireInstanceFromDir] and
// [Kernel.SynthesizeInstance]: values are already unified inside the CUE
// build each of them runs (the package's own `values`, the [WithValues]
// overlay, the synthesized values file), so nothing is validated or filled
// here and nothing on the Kernel is read. Errors are framed
// `instance "<name>": …`.
//
// The returned Instance carries no Source; the caller stamps it.
func processInstance(spec cue.Value) (*module.Instance, error) {
	name := bestEffortInstanceName(spec)

	if err := spec.Validate(cue.Concrete(true)); err != nil {
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
