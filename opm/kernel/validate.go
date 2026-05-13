package kernel

import (
	"strings"

	"cuelang.org/go/cue"
	cueerrors "cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/token"
)

const fieldNotAllowed = "field not allowed"

// ValidateConfig is the kernel's primitive Tier-2 validation entry point.
// It unifies values with schema, runs the closed-schema disallowed-field
// walk, and asserts concreteness via [cue.Concrete](true).
//
// Returns the unified [cue.Value] on success and the zero value on failure.
// The returned error is the raw CUE error tree; walk it via
// [cuelang.org/go/cue/errors.Errors] / [cuelang.org/go/cue/errors.Positions],
// or print it via [cuelang.org/go/cue/errors.Print]. Presentation is
// outside the kernel's contract — frontends own their own formatting.
//
// values is a single, pre-merged [cue.Value]. Layered inputs flow through
// [Kernel.ValidateConfigDetailed]; partial-mode validation flows through
// [Kernel.ValidateConfigPartial]. Module-name framing is the caller's
// responsibility — wrap with [fmt.Errorf] if a context prefix is required.
//
// The zero [cue.Value] is treated as "no values": ValidateConfig returns
// (zero, nil) without running any schema check.
func (k *Kernel) ValidateConfig(schema cue.Value, values cue.Value) (cue.Value, error) {
	return runValidate(schema, values, true)
}

// ValidateConfigPartial validates partial values against schema without
// requiring every schema field to be concrete. It catches type errors,
// disallowed fields, and pattern/regex violations on fields that ARE set,
// but does NOT flag fields that are missing entirely.
//
// Used for CLI lint subcommands, IDE/LSP live feedback, admission webhooks,
// and any callsite that intentionally validates a draft slice of the full
// configuration.
//
// The zero [cue.Value] (no values) is treated as success.
func (k *Kernel) ValidateConfigPartial(schema cue.Value, values cue.Value) (cue.Value, error) {
	return runValidate(schema, values, false)
}

// runValidate is the shared internal that backs [Kernel.ValidateConfig],
// [Kernel.ValidateConfigPartial], and [Kernel.ValidateConfigDetailed]. It
// returns the unified value on success and the raw CUE error tree on
// failure (callers wrap with [fmt.Errorf] if they want context framing).
func runValidate(schema, values cue.Value, requireConcrete bool) (cue.Value, error) {
	if !schema.Exists() || !values.Exists() {
		return cue.Value{}, nil
	}

	var combined cueerrors.Error
	combined, _ = appendSchemaErrors(schema, values, combined, requireConcrete)

	if combined != nil {
		return cue.Value{}, combined
	}

	return schema.Unify(values), nil
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

// ValidateConfigDetailed unifies an ordered slice of [Source] values, then
// validates the merged value against schema. Per-source attribution flows
// through [token.Pos.Filename], populated from [cue.Filename](Origin) at
// the time each Source.Value was compiled — see [Kernel.LoadSourceFromFile],
// [Kernel.LoadSourceFromBytes], and [Kernel.LoadSourceFromString] for
// constructors that bake the filename for you.
//
// Without options the merged value must be concrete (every required field
// set). With [Partial] in opts, concreteness is not enforced; the merged
// value is checked only for type errors, constraint violations, and
// disallowed fields under closed schemas.
//
// Returns the merged [cue.Value] on success and the zero value on failure.
// The returned error is the raw CUE error tree; walk it via
// [cuelang.org/go/cue/errors.Errors] and
// [cuelang.org/go/cue/errors.Positions], or print it via
// [cuelang.org/go/cue/errors.Print]. Presentation is outside the kernel's
// contract — frontends own their own formatting.
//
// Empty sources, a zero schema, or a merged value that does not exist all
// short-circuit to (zero, nil) — the "no values supplied" path documented
// across the kernel's validation surface.
func (k *Kernel) ValidateConfigDetailed(schema cue.Value, sources []Source, opts ...ValidateOption) (cue.Value, error) {
	if len(sources) == 0 {
		return cue.Value{}, nil
	}

	cfg := validateConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}
	requireConcrete := !cfg.partial

	merged := sources[0].Value
	for i := 1; i < len(sources); i++ {
		merged = merged.Unify(sources[i].Value)
	}

	return runValidate(schema, merged, requireConcrete)
}
