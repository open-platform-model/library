package kernel

import (
	"fmt"
	"strings"

	"cuelang.org/go/cue"
	cueerrors "cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/token"

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
// Concreteness is checked twice and reported once, see [concreteness]: the
// whole built spec, and the spec's `values` unified with the module's
// #config. A failure is framed `instance "<name>": not fully concrete: …`;
// other errors `instance "<name>": …`.
//
// The returned Instance carries no Source; the caller stamps it.
func processInstance(spec cue.Value) (*module.Instance, error) {
	name := bestEffortInstanceName(spec)

	if err := concreteness(spec); err != nil {
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

// concreteness checks the built spec twice and returns one report.
//
// The whole spec first, so a value the instance carries in a non-concrete
// form is refused at its place in the build. Then the spec's `values` unified
// with the module's #config ([requiredConfigSet]): #ModuleInstance only
// unifies `values` into a let binding the components read, so a required
// #config value the values leave unset is named by the first check only at
// the component fields that read it, and not at all when none does
// (library#211).
//
// When the second check passes, the first check's error is returned as it
// is, and likewise the other way round. When both fail, the report holds:
//
//   - the second check's findings, in its order, each at `values.<field>`.
//     Where the first check has a finding at the same path (the values carry
//     the field as a bare type), that finding takes its place: it is
//     positioned where the values carry the field.
//   - then the first check's other findings, except those an unset value
//     explains. CUE positions a component field left incomplete by an unset
//     #config value at that value's #config declaration, the position the
//     second check's finding for it carries, so a finding of the first check
//     that shares a position with one of the second is left out. A finding
//     that shares none (a defect of the module's own, or a value read from a
//     field declared `_`, which has no position) stays.
//
// The values findings come first so that the error's one-line text, which
// CUE takes from the first finding, names a field the user sets.
func concreteness(spec cue.Value) error {
	specErr := spec.Validate(cue.Concrete(true))
	requiredErr := requiredConfigSet(spec)
	if requiredErr == nil {
		return specErr
	}
	if specErr == nil {
		return requiredErr
	}

	built := cueerrors.Errors(specErr)
	builtAt := make(map[string]cueerrors.Error, len(built))
	for _, e := range built {
		builtAt[strings.Join(e.Path(), ".")] = e
	}

	var report cueerrors.Error
	reported := make(map[string]bool)
	declared := make(map[token.Position]bool)
	for _, e := range cueerrors.Errors(requiredErr) {
		path := strings.Join(e.Path(), ".")
		reported[path] = true
		for _, pos := range cueerrors.Positions(e) {
			declared[pos.Position()] = true
		}
		if b, ok := builtAt[path]; ok {
			e = b
		}
		report = cueerrors.Append(report, e)
	}
	for _, e := range built {
		if reported[strings.Join(e.Path(), ".")] || positionedAt(e, declared) {
			continue
		}
		report = cueerrors.Append(report, e)
	}
	return report
}

// positionedAt reports whether any position of e is in positions.
func positionedAt(e cueerrors.Error, positions map[token.Position]bool) bool {
	for _, pos := range cueerrors.Positions(e) {
		if positions[pos.Position()] {
			return true
		}
	}
	return false
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
