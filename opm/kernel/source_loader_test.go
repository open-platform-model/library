package kernel_test

import (
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	cueerrors "cuelang.org/go/cue/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/kernel"
)

func TestKernel_LoadSourceFromBytes_FilenameCarriedIntoErrors(t *testing.T) {
	k := kernel.New()
	src, err := k.LoadSourceFromBytes("user.cue", "user values", []byte(`{ replicas: 3 }`))
	require.NoError(t, err)
	assert.Equal(t, "user.cue", src.Origin)
	assert.Equal(t, "user values", src.Name)

	// Validate against an incompatible schema; the error MUST cite "user.cue"
	// in its positions, demonstrating cue.Filename(Origin) was applied.
	schema := k.CueContext().CompileString(`{ replicas: string }`)
	require.NoError(t, schema.Err())

	_, vErr := k.ValidateConfigDetailed(schema, []kernel.Source{src})
	require.Error(t, vErr)

	gotOrigin := false
	for _, ce := range cueerrors.Errors(vErr) {
		for _, pos := range cueerrors.Positions(ce) {
			if pos.IsValid() && pos.Filename() == "user.cue" {
				gotOrigin = true
			}
		}
	}
	assert.True(t, gotOrigin, "error positions MUST report Origin via pos.Filename()")
}

func TestKernel_LoadSourceFromString_FilenameCarriedIntoErrors(t *testing.T) {
	k := kernel.New()
	src, err := k.LoadSourceFromString("config://overlay", "overlay", `{ replicas: 7 }`)
	require.NoError(t, err)
	assert.Equal(t, "config://overlay", src.Origin)

	schema := k.CueContext().CompileString(`{ replicas: string }`)
	require.NoError(t, schema.Err())

	_, vErr := k.ValidateConfigDetailed(schema, []kernel.Source{src})
	require.Error(t, vErr)

	gotOrigin := false
	for _, ce := range cueerrors.Errors(vErr) {
		for _, pos := range cueerrors.Positions(ce) {
			if pos.IsValid() && pos.Filename() == "config://overlay" {
				gotOrigin = true
			}
		}
	}
	assert.True(t, gotOrigin, "non-file Origins (e.g. config:// URIs) MUST flow through unchanged")
}

func TestKernel_LoadSourceFromBytes_CompileErrorReturned(t *testing.T) {
	k := kernel.New()
	_, err := k.LoadSourceFromBytes("broken.cue", "broken", []byte(`{ replicas: int & "string" }`))
	require.Error(t, err, "compile-time CUE errors MUST be returned by the loader")
}

func TestKernel_LoadSourceFromFile_FilenameMatchesAbsolutePath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "values.cue")
	require.NoError(t, os.WriteFile(path, []byte("replicas: 7\n"), 0o600))

	k := kernel.New()
	src, err := k.LoadSourceFromFile(path)
	require.NoError(t, err)

	absPath, _ := filepath.Abs(path)
	assert.Equal(t, absPath, src.Origin, "Origin MUST equal the absolute path baked by cue/load.Instances")
	assert.Equal(t, "values.cue", src.Name)

	// Force a validation error and confirm pos.Filename() == absolute path.
	schema := k.CueContext().CompileString(`{ replicas: string }`)
	require.NoError(t, schema.Err())

	_, vErr := k.ValidateConfigDetailed(schema, []kernel.Source{src})
	require.Error(t, vErr)

	gotPath := false
	for _, ce := range cueerrors.Errors(vErr) {
		for _, pos := range cueerrors.Positions(ce) {
			if pos.IsValid() && pos.Filename() == absPath {
				gotPath = true
			}
		}
	}
	assert.True(t, gotPath, "error positions MUST cite the absolute file path")
}

// TestKernel_LoadSourceFromFile_AutoUnwrapsValuesField pins the OPM
// convention that a file shaped as `values: { ... }` unwraps to the inner
// object. Replaces the previous TestKernel_LoadValuesFile_Parity coverage
// after the standalone helper was deleted.
func TestKernel_LoadSourceFromFile_AutoUnwrapsValuesField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "values.cue")
	require.NoError(t, os.WriteFile(path, []byte(`
package values

values: {
	replicas: 3
	name:     "demo"
}
`), 0o644))

	k := kernel.New()
	src, err := k.LoadSourceFromFile(path)
	require.NoError(t, err)

	// Auto-unwrap: the returned Source.Value is the inner object, not the
	// wrapping `values:` field.
	replicas, err := src.Value.LookupPath(cue.ParsePath("replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(3), replicas)

	name, err := src.Value.LookupPath(cue.ParsePath("name")).String()
	require.NoError(t, err)
	assert.Equal(t, "demo", name)

	// The outer "values" field MUST NOT be reachable on the returned value
	// — auto-unwrap returned the inner object.
	outer := src.Value.LookupPath(cue.ParsePath("values"))
	assert.False(t, outer.Exists(), "values field must have been unwrapped")
}

// TestKernel_LoadSourceFromFile_PassesThroughWithoutValuesField pins the
// fallback path: a file with no top-level `values:` field is returned as-is.
func TestKernel_LoadSourceFromFile_PassesThroughWithoutValuesField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "flat.cue")
	require.NoError(t, os.WriteFile(path, []byte(`replicas: 5
name: "flat"
`), 0o644))

	k := kernel.New()
	src, err := k.LoadSourceFromFile(path)
	require.NoError(t, err)

	replicas, err := src.Value.LookupPath(cue.ParsePath("replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(5), replicas)

	name, err := src.Value.LookupPath(cue.ParsePath("name")).String()
	require.NoError(t, err)
	assert.Equal(t, "flat", name)
}
