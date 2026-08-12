package compat

import (
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
)

// provenanceDenylist is 0010 D30's exact set: the metadata fields that change
// per catalog release by construction (D25 — provenance only) and must
// therefore be removed before compatibility comparison or cross-build
// unification. Nothing else is stripped — identity fields and labels stay.
var provenanceDenylist = map[string]bool{
	"catalogVersion": true,
	"description":    true,
}

// StripProvenance removes metadata.catalogVersion and metadata.description
// from v — the D30 strip both the compat gate and library-matching's unify
// rung apply before comparing values from different catalog builds. Without
// it every comparison reports a violation, because catalogVersion differs
// between any two releases by construction.
//
// The mechanism is a syntax round-trip: Syntax(cue.All(),
// cue.InlineImports(true)) → delete the denylisted fields from every metadata
// block → rebuild in v's own context. The strip reaches the definition as
// well as the instance — removing only the instance's field would leave a
// required `catalogVersion!` nothing satisfies, a shape
// Validate(cue.Concrete(false)) cannot flag if missed. InlineImports makes
// the emitted syntax self-contained, so values whose definitions come from
// imported packages rebuild without their loader. Known cost, accepted by
// D30: the round-trip discards document positions, so downstream CUE
// messages on the stripped value lose file/line — the comparator's own
// violations are path-located by the walk, not by CUE positions.
func StripProvenance(v cue.Value) (cue.Value, error) {
	if err := v.Err(); err != nil {
		return cue.Value{}, fmt.Errorf("stripping provenance: input value: %w", err)
	}
	syn := v.Syntax(cue.All(), cue.InlineImports(true))
	stripNode(syn)

	// The rebuild happens in v's own context, deprecation notwithstanding:
	// library-matching applies the strip at the unify rung, where the
	// stripped value is filled together with platform values, and the
	// repo-wide contract (see the materialize registry contract) keeps all
	// values that unify in one context. A one-off context would hand the
	// caller a value that violates that.
	//nolint:staticcheck // SA1019 — same-context rebuild is load-bearing, see above.
	buildCtx := v.Context()
	var out cue.Value
	switch n := syn.(type) {
	case *ast.File:
		out = buildCtx.BuildFile(n)
	case ast.Expr:
		out = buildCtx.BuildExpr(n)
	default:
		return cue.Value{}, fmt.Errorf("stripping provenance: unexpected syntax node %T", syn)
	}
	if err := out.Err(); err != nil {
		return cue.Value{}, fmt.Errorf("stripping provenance: rebuilding value: %w", err)
	}
	return out, nil
}

// stripNode walks the emitted syntax and scrubs the value of every field
// named metadata, wherever it appears — the round-trip of a unified value
// carries one metadata block per operand (definition and instance), and all
// of them must lose the denylisted fields.
func stripNode(n ast.Node) {
	switch x := n.(type) {
	case *ast.File:
		for _, d := range x.Decls {
			stripNode(d)
		}
	case *ast.StructLit:
		for _, e := range x.Elts {
			stripNode(e)
		}
	case *ast.Field:
		if name, _, _ := ast.LabelName(x.Label); name == "metadata" {
			scrubMetadata(x.Value)
		}
		stripNode(x.Value)
	case *ast.EmbedDecl:
		stripNode(x.Expr)
	case *ast.BinaryExpr:
		stripNode(x.X)
		stripNode(x.Y)
	case *ast.ParenExpr:
		stripNode(x.X)
	case *ast.ListLit:
		for _, e := range x.Elts {
			stripNode(e)
		}
	}
}

// scrubMetadata deletes the denylisted direct fields from every struct
// literal contributing to a metadata value. Only direct children go — the
// denylist names metadata.catalogVersion and metadata.description, not any
// deeper occurrence of the same field names.
func scrubMetadata(e ast.Expr) {
	switch x := e.(type) {
	case *ast.StructLit:
		kept := x.Elts[:0]
		for _, el := range x.Elts {
			if f, ok := el.(*ast.Field); ok {
				if name, _, _ := ast.LabelName(f.Label); provenanceDenylist[name] {
					continue
				}
			}
			kept = append(kept, el)
		}
		x.Elts = kept
	case *ast.BinaryExpr:
		scrubMetadata(x.X)
		scrubMetadata(x.Y)
	case *ast.ParenExpr:
		scrubMetadata(x.X)
	}
}
