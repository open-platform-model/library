package compat

import (
	"errors"
	"fmt"

	"cuelang.org/go/cue"
)

// Violation is one breach of 0010 D27's additive-only rule, located by the
// dotted path from the compared root. Violations are results, not errors
// (opm/errors doctrine): the walk reports every breach it finds and never
// fails. The primitive's name, apiVersion, and predecessor coordinate are
// caller-attached — the walk does not know them.
type Violation struct {
	Path string // dotted path from the compared root, "" for top-level
	Kind string // one of the Kind* constants
	Old  string // rendered prior value; "" when not applicable
	New  string // rendered new value; "" when not applicable
}

// The violation kinds. KindDomainNarrowed carries the CUE subsumption
// diagnostic verbatim in New (no reformatting, consistent with UnifyError);
// the default kinds carry the rendered defaults in Old/New.
const (
	KindFieldRemoved     = "field removed"
	KindFieldAddedStrict = "field added without optional or default"
	KindDefaultChanged   = "default changed"
	KindDefaultRemoved   = "default removed"
	KindDomainNarrowed   = "domain narrowed"
)

// Sentinel errors for [CheckAtLevel]'s error channel. A gate that cannot
// classify its input has not found an incompatibility — these are failures,
// never violations.
var (
	ErrUnparseableAPIVersion = errors.New("unparseable apiVersion")
	ErrNotStruct             = errors.New("operand is not a struct")
)

// CheckAtLevel is the level-aware entry point (0010 D34): the additive-only
// promise binds at beta and GA only, so an alpha apiVersion returns (nil, nil)
// without evaluating the operands. The apiVersion is the primitive's own — a
// catalog's release version is an independent axis and must not be passed
// here. An apiVersion outside core's #APIVersionType grammar is an error, as
// are non-struct top-level operands.
func CheckAtLevel(apiVersion string, prev, next cue.Value) ([]Violation, error) {
	_, lvl, ok := ParseLevel(apiVersion)
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnparseableAPIVersion, apiVersion)
	}
	if !lvl.Enforced() {
		return nil, nil
	}
	if k := prev.IncompleteKind(); k != cue.StructKind {
		return nil, fmt.Errorf("%w: previous operand is %v", ErrNotStruct, k)
	}
	if k := next.IncompleteKind(); k != cue.StructKind {
		return nil, fmt.Errorf("%w: next operand is %v", ErrNotStruct, k)
	}
	return Check(prev, next), nil
}

// Check reports every violation of D27's additive-only rule in next relative
// to prev: fields and options may be added and never removed; a newly added
// field must be optional or defaulted; an existing field's default is
// immutable. It is level-blind — see [CheckAtLevel] — and cannot fail given
// two valid values.
//
// The comparison is a field-wise walk, deliberately not a single Subsume call
// in either direction: adding a struct field makes a value more specific while
// adding a disjunct makes it less specific, and D27 calls both "additive", so
// the rule spans both directions of the lattice while one subsume call tests
// one (measured 10/14 and 8/14 against the D27 change classes in
// enhancements/0011/experiments/03-d27-compat-gate; the walk is 14/14).
// Structs recurse; leaves get a forward subsume, where it is correct for the
// value domain; defaults are compared explicitly at every level, because
// subsume is blind to them in both directions.
func Check(prev, next cue.Value) []Violation {
	return walk("", prev, next, nil)
}

func walk(path string, prev, next cue.Value, acc []Violation) []Violation {
	// Defaults first, at every level — no subsume direction can see them.
	acc = checkDefaults(path, prev, next, acc)

	// A disjunction of structs has StructKind but is not field-iterable
	// content (Fields silently yields nothing on it) — it is walked as a
	// leaf, so branch removal surfaces as domain narrowing and branch-
	// internal field edits are judged by subsumption alone.
	if isWalkableStruct(prev) && isWalkableStruct(next) {
		pit, perr := prev.Fields(cue.All())
		nit, nerr := next.Fields(cue.All())
		if perr == nil && nerr == nil {
			return walkStruct(path, pit, nit, next, acc)
		}
	}

	// Leaf: the new domain must accept everything the old one did — widening
	// passes, narrowing is refused. cue.Schema() and cue.Raw() are
	// load-bearing measured options (see the option-pinning tests): Schema
	// keeps optional-field edits inside disjunction branches from reporting
	// spuriously; Raw keeps the check default-sensitive so a domain narrowed
	// to its own default is not waved through.
	if err := next.Subsume(prev, cue.Schema(), cue.Raw()); err != nil {
		acc = append(acc, Violation{Path: path, Kind: KindDomainNarrowed, New: err.Error()})
	}
	return acc
}

// isWalkableStruct reports whether v is a plain struct the walk may iterate —
// StructKind and not a disjunction.
func isWalkableStruct(v cue.Value) bool {
	if v.IncompleteKind() != cue.StructKind {
		return false
	}
	op, _ := v.Expr()
	return op != cue.OrOp
}

func walkStruct(path string, pit, nit *cue.Iterator, next cue.Value, acc []Violation) []Violation {
	seen := map[string]bool{}

	for pit.Next() {
		sel := pit.Selector()
		if sel.LabelType() == cue.HiddenLabel {
			continue // hidden fields are not contract surface
		}
		name := fieldName(sel)
		seen[name] = true
		nv := next.LookupPath(cue.MakePath(lookupSelector(sel)))
		if !nv.Exists() {
			acc = append(acc, Violation{Path: join(path, name), Kind: KindFieldRemoved})
			continue
		}
		acc = walk(join(path, name), pit.Value(), nv, acc)
	}

	for nit.Next() {
		sel := nit.Selector()
		if sel.LabelType() == cue.HiddenLabel {
			continue
		}
		name := fieldName(sel)
		if seen[name] {
			continue
		}
		// An added field must be optional or carry a default — a new
		// required field breaks every existing consumer.
		optional := sel.ConstraintType()&cue.OptionalConstraint != 0
		_, hasDefault := nit.Value().Default()
		if !optional && !hasDefault {
			acc = append(acc, Violation{Path: join(path, name), Kind: KindFieldAddedStrict})
		}
	}
	return acc
}

// checkDefaults enforces default immutability (D27). Defaults are compared
// only when the prior build has one — adding a default where none existed is
// additive. When either side's default is non-concrete, equality is judged by
// mutual subsumption so a merely-reordered disjunction does not report.
func checkDefaults(path string, prev, next cue.Value, acc []Violation) []Violation {
	pd, phas := prev.Default()
	if !phas {
		return acc
	}
	nd, nhas := next.Default()
	if !nhas {
		return append(acc, Violation{Path: path, Kind: KindDefaultRemoved, Old: render(pd)})
	}
	if pd.IsConcrete() && nd.IsConcrete() {
		if !pd.Equals(nd) {
			return append(acc, Violation{Path: path, Kind: KindDefaultChanged, Old: render(pd), New: render(nd)})
		}
		return acc
	}
	if pd.Subsume(nd, cue.Schema(), cue.Raw()) != nil || nd.Subsume(pd, cue.Schema(), cue.Raw()) != nil {
		return append(acc, Violation{Path: path, Kind: KindDefaultChanged, Old: render(pd), New: render(nd)})
	}
	return acc
}

// fieldName is the clean field name for paths and dedup — no ?/! markers, no
// quotes on ident-safe labels.
func fieldName(sel cue.Selector) string {
	if sel.LabelType() == cue.StringLabel {
		return sel.Unquoted()
	}
	return sel.String()
}

// lookupSelector normalizes a selector for lookup in the other operand: a
// plain selector does not find optional or required fields, so regular labels
// look up in optional form, which finds all three constraint flavors.
func lookupSelector(sel cue.Selector) cue.Selector {
	if sel.LabelType() == cue.StringLabel {
		return sel.Optional()
	}
	return sel
}

func render(v cue.Value) string { return fmt.Sprintf("%v", v) }

func join(p, n string) string {
	if p == "" {
		return n
	}
	return p + "." + n
}
