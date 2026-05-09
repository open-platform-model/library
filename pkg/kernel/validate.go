package kernel

import (
	"strings"

	"cuelang.org/go/cue"
	cueerrors "cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/token"

	oerrors "github.com/open-platform-model/library/pkg/errors"
)

const fieldNotAllowed = "field not allowed"

// ValidateConfig is the kernel's Tier-2 schema-validation entry point. It
// validates the pre-unified values against the supplied #config schema and
// returns the validated value or a [*oerrors.ConfigError] carrying grouped
// CUE diagnostics. contextLabel is a display label ("module"); name is the
// release name for display.
//
// values is a single [cue.Value] the caller has already merged. Layering
// policy (CLI -f stack, operator ConfigMap → Secret → CR overlay, XR fn
// composition input) lives outside the kernel — see
// [pkg/helper/values.ValidateAndUnify] for the source-positioned Tier-1
// helper. The zero [cue.Value] is treated as "no values": ValidateConfig
// returns success without running schema checks.
func (k *Kernel) ValidateConfig(schema cue.Value, values cue.Value, contextLabel, name string) (cue.Value, *oerrors.ConfigError) {
	return runValidate(schema, values, contextLabel, name, true)
}

// ValidateConfigPartial is the kernel's Tier-1 building block: it validates
// partial values against the schema without requiring every schema field to
// be concrete. It catches type errors, disallowed fields, and pattern/regex
// violations on fields that ARE set, but does NOT flag fields that are
// missing entirely.
//
// [pkg/helper/values.ValidateAndUnify] calls this through the
// [pkg/helper/values.KernelOwner] interface so each layer in a [Stack]
// validates cleanly even when the full schema requires fields the layer
// does not set. The merged result is then re-validated by [ValidateConfig]
// (Tier-2) with full concreteness.
//
// The zero [cue.Value] (no values) is treated as success.
func (k *Kernel) ValidateConfigPartial(schema cue.Value, values cue.Value, contextLabel, name string) (cue.Value, *oerrors.ConfigError) {
	return runValidate(schema, values, contextLabel, name, false)
}

func runValidate(schema, values cue.Value, contextLabel, name string, requireConcrete bool) (cue.Value, *oerrors.ConfigError) {
	if !schema.Exists() || !values.Exists() {
		return cue.Value{}, nil
	}

	var combined cueerrors.Error
	combined, _ = appendSchemaErrors(schema, values, combined, requireConcrete)

	if combined != nil {
		return cue.Value{}, &oerrors.ConfigError{Context: contextLabel, Name: name, RawError: combined}
	}

	return values, nil
}

func appendSchemaErrors(schema, value cue.Value, acc cueerrors.Error, requireConcrete bool) (cueerrors.Error, bool) {
	beforeCount := len(cueerrors.Errors(acc))
	changed := false
	acc = walkDisallowed(schema, value, nil, acc)

	unified := schema.Unify(value)
	validateOpts := []cue.Option{}
	if requireConcrete {
		validateOpts = append(validateOpts, cue.Concrete(true))
	}
	if err := unified.Validate(validateOpts...); err != nil {
		for _, ce := range cueerrors.Errors(err) {
			f, _ := ce.Msg()
			if f == fieldNotAllowed {
				continue
			}
			acc = cueerrors.Append(acc, ce)
			changed = true
		}
	}

	if len(cueerrors.Errors(acc)) > beforeCount {
		changed = true
	}

	return acc, changed
}

func walkDisallowed(schema, val cue.Value, pathPrefix []string, acc cueerrors.Error) cueerrors.Error {
	iter, err := val.Fields(cue.Optional(true))
	if err != nil {
		return acc
	}
	for iter.Next() {
		sel := iter.Selector()
		child := iter.Value()
		fieldPath := append(append([]string{}, pathPrefix...), sel.String())

		if !schema.Allows(sel) {
			acc = cueerrors.Append(acc, &fieldNotAllowedError{pos: child.Pos(), path: fieldPath})
			continue
		}

		if child.IncompleteKind() == cue.StructKind {
			childSchema := schema.LookupPath(cue.MakePath(sel))
			if !childSchema.Exists() {
				continue
			}
			acc = walkDisallowed(childSchema, child, fieldPath, acc)
		}
	}
	return acc
}

type fieldNotAllowedError struct {
	pos  token.Pos
	path []string
}

func (e *fieldNotAllowedError) Position() token.Pos         { return e.pos }
func (e *fieldNotAllowedError) InputPositions() []token.Pos { return nil }
func (e *fieldNotAllowedError) Error() string               { return fieldNotAllowed }
func (e *fieldNotAllowedError) Path() []string {
	return append([]string{"values"}, normalizeFieldPath(e.path)...)
}
func (e *fieldNotAllowedError) Msg() (msg string, args []any) {
	return fieldNotAllowed, nil
}

func normalizeFieldPath(path []string) []string {
	if len(path) == 0 {
		return nil
	}
	joined := strings.Join(path, ".")
	joined = strings.TrimPrefix(joined, "#module.#config.")
	joined = strings.TrimPrefix(joined, "#module.#config")
	joined = strings.TrimPrefix(joined, "#config.")
	joined = strings.TrimPrefix(joined, "#config")
	joined = strings.TrimPrefix(joined, ".")
	if joined == "" {
		return nil
	}
	return strings.Split(joined, ".")
}
