package kernel

import (
	"slices"

	"cuelang.org/go/cue"
	cueerrors "cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/token"

	"github.com/open-platform-model/library/opm/internal/loader"
)

const fieldNotAllowed = "field not allowed"

// ValidateConfigDetailed is the kernel's one validation entry: it compiles an
// ordered slice of [Source] values in the schema's own context, unifies them
// in stack order, runs the closed-schema disallowed-field walk, and asserts
// concreteness on the merged value via [cue.Concrete](true). A single value
// is a one-element slice.
//
// Per-source attribution flows through [token.Pos.Filename]: each source is
// compiled with [cue.Filename](Origin) here, where it meets the schema, so
// every position in the returned error names the source's Origin — see
// [Kernel.LoadSourceFromFile] and [Kernel.LoadSourceFromBytes] for the
// constructors, and [Source] for what a file-backed origin means.
//
// Returns the merged [cue.Value] on success and the zero value on failure.
// The returned error is the raw CUE error tree (a source that fails to
// compile included); walk it via [cuelang.org/go/cue/errors.Errors] and
// [cuelang.org/go/cue/errors.Positions], or print it via
// [cuelang.org/go/cue/errors.Print]. Presentation is outside the kernel's
// contract — frontends own their own formatting. Module-name framing is the
// caller's responsibility — wrap with [fmt.Errorf] if a context prefix is
// required.
//
// Empty sources, a zero schema, or a merged value that does not exist all
// short-circuit to (zero, nil) — the "no values supplied" path documented
// across the kernel's validation surface.
func (k *Kernel) ValidateConfigDetailed(schema cue.Value, sources []Source) (cue.Value, error) {
	return validateSources(schema, sources, k.loadOptions(), true)
}

// validateSources compiles sources in the schema's own context, in stack
// order, and hands the values to [validateCompiled]. It backs
// [Kernel.ValidateConfigDetailed]; the acquire and synthesis paths compile
// their sources once for the merge and validate those values through
// [validateCompiled] instead. requireConcrete false still surfaces type
// errors, constraint violations and disallowed fields on the fields that are
// set, but not missing required fields. opts are the load settings a
// file-backed source's load uses ([Kernel.loadOptions]); the zero value
// reads the process environment and lets cue/load build its own registry.
//
// Returns the unified value on success and the zero value plus the raw CUE
// error tree on failure; callers wrap with [fmt.Errorf] if they want context
// framing. Empty sources, a zero schema or a merged value that does not
// exist short-circuit to (zero, nil).
func validateSources(schema cue.Value, sources []Source, opts loader.Options, requireConcrete bool) (cue.Value, error) {
	if len(sources) == 0 || !schema.Exists() {
		return cue.Value{}, nil
	}
	// The sources are compiled in the context that built the schema so the
	// two can unify; Value.Context is deprecated for combining values from
	// different contexts, which is exactly what compiling here avoids.
	values, err := compileSources(schema.Context(), sources, opts) //nolint:staticcheck // the schema's own context is the one the sources must be built in
	if err != nil {
		return cue.Value{}, err
	}
	return validateCompiled(schema, values, requireConcrete)
}

// validateCompiled is [validateSources] for values already compiled in the
// schema's context: no values or a zero schema short-circuit to (zero, nil),
// anything else goes to [validateValues].
func validateCompiled(schema cue.Value, values []cue.Value, requireConcrete bool) (cue.Value, error) {
	if len(values) == 0 || !schema.Exists() {
		return cue.Value{}, nil
	}
	return validateValues(schema, values, requireConcrete)
}

// validateValues unifies values (already compiled in schema's context) in
// stack order and validates the merged value against schema: the
// closed-schema disallowed-field walk first, so a stray field is reported at
// the source's own position, then CUE's own Validate on the unified value,
// with concreteness enforced when requireConcrete is set.
//
// Returns the unified value on success and the zero value plus the raw CUE
// error tree on failure. A merged value that does not exist short-circuits
// to (zero, nil).
func validateValues(schema cue.Value, values []cue.Value, requireConcrete bool) (cue.Value, error) {
	merged := unifyValues(values)
	if !merged.Exists() {
		return cue.Value{}, nil
	}

	acc := walkDisallowed(schema, merged, nil, nil)

	unified := schema.Unify(merged)
	var opts []cue.Option
	if requireConcrete {
		opts = append(opts, cue.Concrete(true))
	}
	if err := unified.Validate(opts...); err != nil {
		for _, ce := range cueerrors.Errors(err) {
			// walkDisallowed already reported every disallowed field at the
			// source's position; CUE's own copy would duplicate it.
			if msg, _ := ce.Msg(); msg == fieldNotAllowed {
				continue
			}
			acc = cueerrors.Append(acc, ce)
		}
	}
	if acc != nil {
		return cue.Value{}, acc
	}
	return unified, nil
}

// walkDisallowed reports every field of val that schema does not allow, at
// the field's own position. It records the path as selectors, so a label
// that contains dots ("app.kubernetes.io/name") stays one step.
func walkDisallowed(schema, val cue.Value, pathPrefix []cue.Selector, acc cueerrors.Error) cueerrors.Error {
	iter, err := val.Fields(cue.Optional(true))
	if err != nil {
		return acc
	}
	for iter.Next() {
		sel := iter.Selector()
		child := iter.Value()
		fieldPath := append(slices.Clone(pathPrefix), sel)

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
	path []cue.Selector
}

func (e *fieldNotAllowedError) Position() token.Pos         { return e.pos }
func (e *fieldNotAllowedError) InputPositions() []token.Pos { return nil }
func (e *fieldNotAllowedError) Error() string               { return fieldNotAllowed }

// Path returns "values" followed by one segment per selector, each in CUE's
// selector form (a definition as #name, a plain label bare, a label that
// needs quoting quoted), with the schema's leading #module #config steps
// omitted. A dotted label is one segment, so joining with "." gives the
// field's dotted path.
func (e *fieldNotAllowedError) Path() []string {
	out := []string{"values"}
	for _, sel := range trimSchemaPrefix(e.path) {
		out = append(out, sel.String())
	}
	return out
}
func (e *fieldNotAllowedError) Msg() (msg string, args []any) {
	return fieldNotAllowed, nil
}

// trimSchemaPrefix drops a leading #module #config pair, or a leading
// #config, comparing whole selectors. It is defensive: walkDisallowed iterates
// regular and optional fields only, which yields no definitions, so no path it
// builds starts with either selector today. The config-validation spec still
// requires the omission, so keep it until that clause goes.
func trimSchemaPrefix(path []cue.Selector) []cue.Selector {
	if len(path) >= 2 && path[0].String() == "#module" && path[1].String() == "#config" {
		return path[2:]
	}
	if len(path) >= 1 && path[0].String() == "#config" {
		return path[1:]
	}
	return path
}
