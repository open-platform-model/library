package kernel_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/require"
)

// Parity comparator for the render-parity harness (enhancement 0019 D1/D4/D14;
// openspec change library-parity-harness). The oracle is plain CUE
// unification of a transformer's #transform with its declared inputs in one
// build; the kernel is compile.Execute. Where they differ, the kernel is the
// defective side, and the fix is removing kernel behaviour, never loosening
// this comparison.
//
// The comparison is ORDER-SENSITIVE by contract (D14): CUE's natural,
// unfinalized field order is the render output contract, so the encoder used
// here must preserve evaluation order rather than sort. cue.Value.MarshalJSON
// does (TestParityEncoder_ReportsReordering proves it); Syntax(cue.Final())
// is deliberately not used because finalization is the very pass D14 names as
// the source of today's reordering.

// parityEquality mirrors #ParityCase.equality in
// enhancements/0019/contracts/contracts.cue.
type parityEquality string

const (
	// equalityStructural compares the whole rendered value. Reachable only
	// once 0019 D12 lands and both sides derive #context identically.
	equalityStructural parityEquality = "structural"
	// equalityOutputFieldsOnly compares the transformer's `output` value and
	// excludes nothing else. The interim mode while #context is built in Go
	// on the kernel side and projected in CUE on the oracle side.
	equalityOutputFieldsOnly parityEquality = "output-fields-only"
)

// parityCase is one row of the harness table, field for field the
// #ParityCase contract. OrderSensitive is not a field: the contract fixes it
// true, so it is a property of the comparator.
type parityCase struct {
	Name        string
	Instance    string
	Component   string
	Transformer string
	Equality    parityEquality

	// ExpectedDivergence names the kernel behaviour that makes this case
	// diverge today. Empty means the two sides MUST agree. Non-empty means
	// the kernel side MUST fail or differ; when it unexpectedly agrees the
	// case fails, telling the author to delete the entry (0019 D4: every
	// entry is emptied by the time the enhancement is implemented).
	ExpectedDivergence string
}

// parityRender is what one renderer produced for one case's pair: either a
// list of rendered objects (one per output element, as compile.Execute
// flattens a list output) or an error.
type parityRender struct {
	Objects []cue.Value
	Err     error
}

// encodeOrdered serialises a value preserving CUE's evaluation order.
func encodeOrdered(v cue.Value) (string, error) {
	b, err := v.MarshalJSON()
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// firstDiffPath walks a and b in parallel and returns the first path at which
// they differ (in kind, in field label order, in list length, or in scalar
// value), or "" when they encode identically. Field order is part of the
// comparison: {a:1,b:2} and {b:2,a:1} differ at ".b" (the first label position
// where the two structs disagree).
func firstDiffPath(a, b cue.Value) string {
	return diffAt("", a, b)
}

func diffAt(path string, a, b cue.Value) string {
	if a.Kind() != b.Kind() {
		return orRoot(path)
	}
	switch a.Kind() {
	case cue.StructKind:
		ai, aErr := a.Fields()
		bi, bErr := b.Fields()
		if aErr != nil || bErr != nil {
			return orRoot(path)
		}
		for {
			an, bn := ai.Next(), bi.Next()
			if !an || !bn {
				if an != bn {
					// One side has more fields than the other.
					if an {
						return path + "." + ai.Selector().String()
					}
					return path + "." + bi.Selector().String()
				}
				return ""
			}
			if ai.Selector().String() != bi.Selector().String() {
				return path + "." + ai.Selector().String()
			}
			if d := diffAt(path+"."+ai.Selector().String(), ai.Value(), bi.Value()); d != "" {
				return d
			}
		}
	case cue.ListKind:
		ai, aErr := a.List()
		bi, bErr := b.List()
		if aErr != nil || bErr != nil {
			return orRoot(path)
		}
		for i := 0; ; i++ {
			an, bn := ai.Next(), bi.Next()
			if !an || !bn {
				if an != bn {
					return fmt.Sprintf("%s[%d]", path, i)
				}
				return ""
			}
			if d := diffAt(fmt.Sprintf("%s[%d]", path, i), ai.Value(), bi.Value()); d != "" {
				return d
			}
		}
	default:
		ea, aErr := encodeOrdered(a)
		eb, bErr := encodeOrdered(b)
		if aErr != nil || bErr != nil || ea != eb {
			return orRoot(path)
		}
		return ""
	}
}

func orRoot(path string) string {
	if path == "" {
		return "."
	}
	return path
}

// compareRendered compares the kernel's objects for a pair against the
// oracle's, order-sensitively, and returns a description of the first
// divergence or "" when they agree. Objects are compared positionally, as
// the list output of a transformer is positional.
func compareRendered(kernel, oracle []cue.Value) string {
	if len(kernel) != len(oracle) {
		return fmt.Sprintf("object count: kernel rendered %d, oracle rendered %d", len(kernel), len(oracle))
	}
	for i := range kernel {
		ek, err := encodeOrdered(kernel[i])
		if err != nil {
			return fmt.Sprintf("object[%d]: kernel value does not encode: %v", i, err)
		}
		eo, err := encodeOrdered(oracle[i])
		if err != nil {
			return fmt.Sprintf("object[%d]: oracle value does not encode: %v", i, err)
		}
		if ek != eo {
			class := "values differ beyond field order"
			if equalModuloOrder(ek, eo) {
				class = "ordering-only divergence: same fields and values, different field order (0019 D14)"
			}
			return fmt.Sprintf("object[%d] differs at %s (%s)\n  kernel: %s\n  oracle: %s",
				i, firstDiffPath(kernel[i], oracle[i]), class, ek, eo)
		}
	}
	return ""
}

// checkParity applies a case's contract to the two renders and returns nil
// when the case passes. The oracle is the reference: it MUST render (an
// oracle error is a broken fixture, not a divergence). With
// ExpectedDivergence empty the kernel MUST agree; with it set the kernel MUST
// fail or differ, and agreement is itself a failure naming the entry to
// delete.
func checkParity(c parityCase, kernel, oracle parityRender) error {
	switch c.Equality {
	case equalityOutputFieldsOnly:
	case equalityStructural:
		// resolved-by-D12: until core derives #context from the other two
		// inputs, the kernel builds it in Go and the oracle projects it in
		// CUE, so a structural comparison would compare two constructions
		// rather than two renderers. Refused rather than approximated.
		return fmt.Errorf("case %q: structural equality is not implementable until 0019 D12 lands; declare %q", c.Name, equalityOutputFieldsOnly)
	default:
		return fmt.Errorf("case %q: unknown equality %q", c.Name, c.Equality)
	}
	if oracle.Err != nil {
		return fmt.Errorf("case %q (%s :: %s): the pure-CUE oracle must render; a failing oracle is a broken fixture, not a kernel divergence: %w",
			c.Name, c.Component, c.Transformer, oracle.Err)
	}

	var divergence string
	switch {
	case kernel.Err != nil:
		divergence = "kernel failed to render: " + kernel.Err.Error()
	default:
		divergence = compareRendered(kernel.Objects, oracle.Objects)
	}

	if c.ExpectedDivergence == "" {
		if divergence != "" {
			return fmt.Errorf("case %q (%s :: %s): kernel diverges from pure-CUE unification (0019 D1: the kernel is the defective side; close this by removing kernel behaviour, not by loosening the comparison)\n%s",
				c.Name, c.Component, c.Transformer, divergence)
		}
		return nil
	}
	if divergence == "" {
		return fmt.Errorf("case %q (%s :: %s): expected divergence %q no longer reproduces; the kernel now agrees with the oracle. Delete this case's ExpectedDivergence entry (0019 D4)",
			c.Name, c.Component, c.Transformer, c.ExpectedDivergence)
	}
	return nil
}

// equalModuloOrder reports whether two ordered encodings carry the same
// fields and values once struct field order is disregarded. List order is
// still significant. Used only to CLASSIFY a divergence in the failure
// message; the comparison itself never ignores order.
func equalModuloOrder(a, b string) bool {
	var va, vb any
	if json.Unmarshal([]byte(a), &va) != nil || json.Unmarshal([]byte(b), &vb) != nil {
		return false
	}
	// encoding/json sorts map keys on output, so re-marshalling canonicalises
	// struct order while preserving list order.
	ca, errA := json.Marshal(va)
	cb, errB := json.Marshal(vb)
	return errA == nil && errB == nil && string(ca) == string(cb)
}

// assertParity is checkParity as a test assertion, logging a reproduced
// expected divergence so the evidence stays visible in -v output.
func assertParity(t *testing.T, c parityCase, kernel, oracle parityRender) {
	t.Helper()
	require.NoError(t, checkParity(c, kernel, oracle))
	if c.ExpectedDivergence != "" {
		why := "kernel differs"
		if kernel.Err != nil {
			why = firstLine(kernel.Err.Error())
		}
		t.Logf("case %q: expected divergence reproduces (%s): %s", c.Name, c.ExpectedDivergence, why)
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
