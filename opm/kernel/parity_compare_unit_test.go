package kernel_test

import (
	"errors"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The negative test the harness rests on (design D4): the encoder MUST report
// a reordering. If a future cuelang.org/go release starts sorting JSON output,
// this fails first, before any parity case can be silently weakened.
func TestParityEncoder_ReportsReordering(t *testing.T) {
	ctx := cuecontext.New()
	a := ctx.CompileString(`{a: 1, b: {y: 1, x: 2}, l: [2, 1]}`)
	b := ctx.CompileString(`{b: {x: 2, y: 1}, a: 1, l: [2, 1]}`)
	require.NoError(t, a.Err())
	require.NoError(t, b.Err())

	ea, err := encodeOrdered(a)
	require.NoError(t, err)
	eb, err := encodeOrdered(b)
	require.NoError(t, err)
	assert.NotEqual(t, ea, eb, "encoder must preserve field order, not sort")
	assert.Equal(t, `{"a":1,"b":{"y":1,"x":2},"l":[2,1]}`, ea)

	// Same fields, same values, reordered at the top level: first diff is the
	// first label position where the two structs disagree.
	assert.Equal(t, ".a", firstDiffPath(a, b))

	// Nested reordering only.
	c := ctx.CompileString(`{a: 1, b: {x: 2, y: 1}, l: [2, 1]}`)
	assert.Equal(t, ".b.y", firstDiffPath(a, c))

	// List element order.
	d := ctx.CompileString(`{a: 1, b: {y: 1, x: 2}, l: [1, 2]}`)
	assert.Equal(t, ".l[0]", firstDiffPath(a, d))

	// Identical values: no diff.
	assert.Equal(t, "", firstDiffPath(a, ctx.CompileString(`{a: 1, b: {y: 1, x: 2}, l: [2, 1]}`)))
}

func TestParityCompareRendered(t *testing.T) {
	ctx := cuecontext.New()
	obj := func(s string) cue.Value { return ctx.CompileString(s) }

	assert.Equal(t, "", compareRendered(
		[]cue.Value{obj(`{kind: "A", n: 1}`), obj(`{kind: "B"}`)},
		[]cue.Value{obj(`{kind: "A", n: 1}`), obj(`{kind: "B"}`)},
	))
	assert.Contains(t, compareRendered(
		[]cue.Value{obj(`{kind: "A"}`)},
		[]cue.Value{obj(`{kind: "A"}`), obj(`{kind: "B"}`)},
	), "object count")
	reordered := compareRendered(
		[]cue.Value{obj(`{kind: "A", n: 1}`)},
		[]cue.Value{obj(`{n: 1, kind: "A"}`)},
	)
	assert.Contains(t, reordered, "object[0] differs at .kind")
	assert.Contains(t, reordered, "ordering-only divergence")

	changed := compareRendered(
		[]cue.Value{obj(`{kind: "A", n: 1}`)},
		[]cue.Value{obj(`{kind: "A", n: 2}`)},
	)
	assert.Contains(t, changed, "object[0] differs at .n")
	assert.Contains(t, changed, "beyond field order")

	// List element order is never disregarded, even by the classifier.
	assert.False(t, equalModuloOrder(`{"l":[1,2]}`, `{"l":[2,1]}`))
	assert.True(t, equalModuloOrder(`{"a":1,"b":{"y":1,"x":2}}`, `{"b":{"x":2,"y":1},"a":1}`))
}

func TestParityCheck_Contract(t *testing.T) {
	ctx := cuecontext.New()
	same := []cue.Value{ctx.CompileString(`{kind: "A"}`)}
	other := []cue.Value{ctx.CompileString(`{kind: "B"}`)}
	base := parityCase{Name: "c", Component: "web", Transformer: "t", Equality: equalityOutputFieldsOnly}
	expecting := base
	expecting.ExpectedDivergence = "FinalizeValue strips #names from #component"

	t.Run("agreement passes when no divergence is expected", func(t *testing.T) {
		assert.NoError(t, checkParity(base, parityRender{Objects: same}, parityRender{Objects: same}))
	})
	t.Run("expected divergence passes when the kernel fails", func(t *testing.T) {
		assert.NoError(t, checkParity(expecting, parityRender{Err: errors.New("incomplete")}, parityRender{Objects: same}))
	})
	t.Run("expected divergence passes when the kernel differs", func(t *testing.T) {
		assert.NoError(t, checkParity(expecting, parityRender{Objects: other}, parityRender{Objects: same}))
	})
	t.Run("unexpected divergence fails naming the path", func(t *testing.T) {
		err := checkParity(base, parityRender{Objects: other}, parityRender{Objects: same})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "object[0] differs at .kind")
		assert.Contains(t, err.Error(), "0019 D1")
	})
	t.Run("unexpected kernel failure fails", func(t *testing.T) {
		err := checkParity(base, parityRender{Err: errors.New("boom")}, parityRender{Objects: same})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "kernel failed to render: boom")
	})
	t.Run("retired divergence fails and names the entry", func(t *testing.T) {
		err := checkParity(expecting, parityRender{Objects: same}, parityRender{Objects: same})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no longer reproduces")
		assert.Contains(t, err.Error(), expecting.ExpectedDivergence)
	})
	t.Run("structural equality is refused until D12", func(t *testing.T) {
		c := base
		c.Equality = equalityStructural
		err := checkParity(c, parityRender{Objects: same}, parityRender{Objects: same})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "0019 D12")
	})
	t.Run("oracle error is a fixture failure, never a divergence", func(t *testing.T) {
		err := checkParity(expecting, parityRender{Err: errors.New("x")}, parityRender{Err: errors.New("oracle broken")})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "broken fixture")
	})
}
