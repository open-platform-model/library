package compat

import (
	"bytes"
	"errors"
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/format"
)

// provenanceDenylist is 0010 D30's exact set: the metadata fields that change
// per catalog release by construction (D25 — provenance only) and are
// therefore never contract surface. The comparison walk skips them directly
// under any metadata field at every depth; nothing else is excluded —
// identity fields and labels stay.
var provenanceDenylist = map[string]bool{
	"catalogVersion": true,
	"description":    true,
}

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
	KindFieldRemoved      = "field removed"
	KindFieldAddedStrict  = "field added without optional or default"
	KindDefaultChanged    = "default changed"
	KindDefaultRemoved    = "default removed"
	KindDomainNarrowed    = "domain narrowed"
	KindFieldMadeRequired = "field made required"
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
//
// Three rules keep the walk from reporting non-changes (measured on
// catalog_opm PR 51, cli issue 165):
//
//   - 0010 D30's provenance fields are skipped at every depth: catalogVersion
//     and description directly under any field named metadata, so a member
//     reference embedded in another member (appliesTo, composedResources)
//     does not report the referenced member's per-release provenance.
//   - Closed lists of equal length are walked element-wise (paths name[i]),
//     so those embedded references reach the rule above; any other list pair
//     is a leaf.
//   - A leaf whose emitted syntax is byte-identical on both sides reports
//     nothing: it cannot have narrowed, and the forward subsume false-positives
//     on unchanged leaves carrying matchN or a pending comprehension.
func Check(prev, next cue.Value) []Violation {
	return walk("", prev, next, false, nil)
}

// walk compares one position. underMetadata is true when path names a direct
// child of a field called metadata, which is where D30's denylist applies.
func walk(path string, prev, next cue.Value, underMetadata bool, acc []Violation) []Violation {
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
			return walkStruct(path, pit, nit, next, underMetadata, acc)
		}
	}

	if out, ok := walkList(path, prev, next, acc); ok {
		return out
	}

	// An unchanged leaf cannot have narrowed. Syntax(cue.All()) expands
	// references (a changed definition behind a reference renders
	// differently), so byte equality is a sound "unchanged" signal; rendering
	// failure falls through to the subsume.
	if leafIdentical(prev, next) {
		return acc
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

func walkStruct(path string, pit, nit *cue.Iterator, next cue.Value, underMetadata bool, acc []Violation) []Violation {
	// The new side, by name, so the prior-side loop can compare selectors
	// (constraint markers live on the selector, not the value) and the
	// addition loop below needs no second iteration. Keyed by name on
	// purpose: `y`, `y?` and `y!` are three constraint flavours of one label
	// and must find each other, which a LookupPath by selector does not do
	// (a plain selector misses optional and required fields).
	type field struct {
		sel cue.Selector
		val cue.Value
	}
	added := map[string]field{}
	var order []string
	for nit.Next() {
		sel := nit.Selector()
		if sel.LabelType() == cue.HiddenLabel {
			continue
		}
		name := fieldName(sel)
		added[name] = field{sel: sel, val: nit.Value()}
		order = append(order, name)
	}

	for pit.Next() {
		sel := pit.Selector()
		if sel.LabelType() == cue.HiddenLabel {
			continue // hidden fields are not contract surface
		}
		name := fieldName(sel)
		if underMetadata && provenanceDenylist[name] {
			continue // D30: per-release provenance, never contract surface
		}
		nf, present := added[name]
		delete(added, name)
		if !present {
			acc = append(acc, Violation{Path: join(path, name), Kind: KindFieldRemoved})
			continue
		}
		// An optional field that becomes required breaks every consumer
		// that omitted it (0010 D27). Judged on the selectors: the leaf
		// subsume below compares value domains and cannot see a constraint
		// marker. A defaulted field that loses its default is the default
		// rule's finding, not this one's.
		if sel.ConstraintType()&cue.OptionalConstraint != 0 && required(nf.sel, nf.val) {
			acc = append(acc, Violation{Path: join(path, name), Kind: KindFieldMadeRequired})
		}
		acc = walk(join(path, name), pit.Value(), nf.val, name == "metadata", acc)
	}

	for _, name := range order {
		nf, ok := added[name]
		if !ok || (underMetadata && provenanceDenylist[name]) {
			continue
		}
		// An added field must be optional or carry a default — a new
		// required field breaks every existing consumer.
		if required(nf.sel, nf.val) {
			acc = append(acc, Violation{Path: join(path, name), Kind: KindFieldAddedStrict})
		}
	}
	return acc
}

// required reports 0010 D27's posture of a field: a `!` field, or a regular
// field with no default, is required — a consumer must supply it. A `?` field
// or a defaulted one is optional.
func required(sel cue.Selector, v cue.Value) bool {
	ct := sel.ConstraintType()
	if ct&cue.OptionalConstraint != 0 {
		return false
	}
	if ct&cue.RequiredConstraint != 0 {
		return true
	}
	_, hasDefault := v.Default()
	return !hasDefault
}

// walkList walks two closed lists of equal length element-wise, paths
// name[i], and reports ok=false for any other pair (open lists, differing
// lengths, non-lists), which the caller judges as a leaf. Element-wise is the
// member-reference case (appliesTo, composedResources); D27 states no list
// semantics, so addition and removal keep their subsume verdict.
func walkList(path string, prev, next cue.Value, acc []Violation) ([]Violation, bool) {
	if prev.IncompleteKind() != cue.ListKind || next.IncompleteKind() != cue.ListKind {
		return acc, false
	}
	if prev.Allows(cue.AnyIndex) || next.Allows(cue.AnyIndex) {
		return acc, false // open list: no fixed elements to pair
	}
	pn, perr := prev.Len().Int64()
	nn, nerr := next.Len().Int64()
	if perr != nil || nerr != nil || pn != nn {
		return acc, false
	}
	pit, perr := prev.List()
	nit, nerr := next.List()
	if perr != nil || nerr != nil {
		return acc, false
	}
	for i := 0; pit.Next() && nit.Next(); i++ {
		acc = walk(fmt.Sprintf("%s[%d]", path, i), pit.Value(), nit.Value(), false, acc)
	}
	return acc, true
}

// leafIdentical reports whether two leaves emit byte-identical syntax under
// cue.All(). Both operands come from the same conventions, so an unchanged
// leaf formats identically; a rendering failure is treated as "different".
func leafIdentical(prev, next cue.Value) bool {
	pb, err := format.Node(prev.Syntax(cue.All()))
	if err != nil {
		return false
	}
	nb, err := format.Node(next.Syntax(cue.All()))
	if err != nil {
		return false
	}
	return bytes.Equal(pb, nb)
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

func render(v cue.Value) string { return fmt.Sprintf("%v", v) }

func join(p, n string) string {
	if p == "" {
		return n
	}
	return p + "." + n
}
