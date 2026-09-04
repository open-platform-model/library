package valuesfile

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A missing value yields no file at all: the caller omits it so the
// package's values path stays open.
func TestRender_NoValuesNoFile(t *testing.T) {
	out, err := Render("instance", cue.Value{})
	require.NoError(t, err)
	assert.Nil(t, out)
}

// The file declares the caller's package and a single `values` field whose
// body is the value rendered back to canonical CUE, not interpolated input.
func TestRender_DeclaresPackageAndValues(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString(`{replicas: 3, image: "nginx:1.27", nested: {on: true}}`)
	require.NoError(t, v.Err())

	out, err := Render("instance", v)
	require.NoError(t, err)
	got := string(out)
	assert.True(t, len(got) > 0 && got[:len("package instance\n\nvalues: ")] == "package instance\n\nvalues: ", "%q", got)

	// Round-trips: compiling the file yields the same values.
	back := ctx.CompileString(got)
	require.NoError(t, back.Err())
	assert.True(t, back.LookupPath(cue.ParsePath("values")).Equals(v), "rendered values differ:\n%s", got)
}

// A string that looks like CUE stays a string literal: the renderer emits
// the value's syntax, so an attacker-influenced value cannot inject source.
func TestRender_DoesNotInterpolate(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString(`{name: "x\"\nvalues: injected: true\n"}`)
	require.NoError(t, v.Err())

	out, err := Render("p", v)
	require.NoError(t, err)
	back := ctx.CompileString(string(out))
	require.NoError(t, back.Err())
	assert.False(t, back.LookupPath(cue.ParsePath("values.injected")).Exists())
	name, err := back.LookupPath(cue.ParsePath("values.name")).String()
	require.NoError(t, err)
	assert.Equal(t, "x\"\nvalues: injected: true\n", name)
}
