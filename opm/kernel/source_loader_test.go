package kernel_test

import (
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/kernel"
)

func TestKernel_LoadSourceFromBytes_FilenameCarriedIntoErrors(t *testing.T) {
	k := kernel.New()
	src, err := k.LoadSourceFromBytes("user.cue", []byte(`{ replicas: 3 }`))
	require.NoError(t, err)
	assert.Equal(t, "user.cue", src.Origin)
	assert.Equal(t, []byte(`{ replicas: 3 }`), src.Data, "Data is the payload, unevaluated")

	// Validate against an incompatible schema; the error MUST cite "user.cue"
	// in its positions, demonstrating cue.Filename(Origin) was applied when
	// the source was compiled at use.
	schema := cuecontext.New().CompileString(`{ replicas: string }`)
	require.NoError(t, schema.Err())

	_, vErr := k.ValidateConfigDetailed(schema, []kernel.Source{src})
	require.Error(t, vErr)
	assert.True(t, positionsName(vErr, "user.cue"), "error positions MUST report Origin via pos.Filename()")
}

func TestKernel_LoadSourceFromBytes_NonFileOriginCarriedIntoErrors(t *testing.T) {
	k := kernel.New()
	src, err := k.LoadSourceFromBytes("config://overlay", []byte(`{ replicas: 7 }`))
	require.NoError(t, err)
	assert.Equal(t, "config://overlay", src.Origin)

	schema := cuecontext.New().CompileString(`{ replicas: string }`)
	require.NoError(t, schema.Err())

	_, vErr := k.ValidateConfigDetailed(schema, []kernel.Source{src})
	require.Error(t, vErr)
	assert.True(t, positionsName(vErr, "config://overlay"), "non-file Origins (e.g. config:// URIs) MUST flow through unchanged")
}

// config-validation spec, "Syntax errors fail at load": a payload that does
// not parse is refused by the loader, positioned at Origin, and no Source is
// returned.
func TestKernel_LoadSourceFromBytes_SyntaxErrorFailsAtLoad(t *testing.T) {
	k := kernel.New()
	src, err := k.LoadSourceFromBytes("broken.cue", []byte(`{ replicas: 3`))
	require.Error(t, err, "a payload that does not parse MUST be refused by the loader")
	assert.True(t, positionsName(err, "broken.cue"), "the syntax error is positioned at Origin: %v", err)
	assert.Equal(t, kernel.Source{}, src, "no Source is returned")
}

// kernel-runtime spec, "Syntax errors fail at load, schema errors at use": a
// payload that parses but does not evaluate loads fine and fails, positioned
// at Origin, in the operation that applies it, since the loader evaluates
// nothing.
func TestKernel_LoadSourceFromBytes_EvaluationErrorFailsAtUse(t *testing.T) {
	k := kernel.New()
	src, err := k.LoadSourceFromBytes("conflict.cue", []byte(`{ replicas: int & "string" }`))
	require.NoError(t, err, "the loader parses and evaluates nothing")

	schema := cuecontext.New().CompileString(`{ replicas: int }`)
	require.NoError(t, schema.Err())
	_, vErr := k.ValidateConfigDetailed(schema, []kernel.Source{src})
	require.Error(t, vErr, "the conflict surfaces where the source is compiled")
	assert.True(t, positionsName(vErr, "conflict.cue"), "positioned at Origin: %v", vErr)
}

func TestKernel_LoadSourceFromFile_FilenameMatchesAbsolutePath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "values.cue")
	require.NoError(t, os.WriteFile(path, []byte("replicas: 7\n"), 0o600))

	k := kernel.New()
	src, err := k.LoadSourceFromFile(path)
	require.NoError(t, err)

	absPath, _ := filepath.Abs(path)
	assert.Equal(t, absPath, src.Origin, "Origin MUST equal the absolute path")
	assert.Equal(t, []byte("replicas: 7\n"), src.Data, "Data is the file's bytes")

	// Force a validation error and confirm pos.Filename() == absolute path.
	schema := cuecontext.New().CompileString(`{ replicas: string }`)
	require.NoError(t, schema.Err())

	_, vErr := k.ValidateConfigDetailed(schema, []kernel.Source{src})
	require.Error(t, vErr)
	assert.True(t, positionsName(vErr, absPath), "error positions MUST cite the absolute file path")
}

// kernel-runtime spec, "Values file is auto-unwrapped": a file shaped as
// `values: { ... }` is unwrapped to the inner object when the source is
// compiled at use, so the value validated is the inner object.
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
	absPath, _ := filepath.Abs(path)
	assert.Equal(t, absPath, src.Origin)

	schema := cuecontext.New().CompileString(`{ replicas: int, name: string }`)
	require.NoError(t, schema.Err())
	merged, err := k.ValidateConfigDetailed(schema, []kernel.Source{src})
	require.NoError(t, err)

	// Auto-unwrap: the value validated is the inner object, not the wrapping
	// `values:` field.
	replicas, err := merged.LookupPath(cue.ParsePath("replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(3), replicas)

	name, err := merged.LookupPath(cue.ParsePath("name")).String()
	require.NoError(t, err)
	assert.Equal(t, "demo", name)

	// The outer "values" field MUST NOT be reachable on the validated value
	// — auto-unwrap compiled the inner object.
	outer := merged.LookupPath(cue.ParsePath("values"))
	assert.False(t, outer.Exists(), "values field must have been unwrapped")
}

// kernel-runtime spec, "File without values field passes through": a file
// with no top-level `values:` field is validated whole.
func TestKernel_LoadSourceFromFile_PassesThroughWithoutValuesField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "flat.cue")
	require.NoError(t, os.WriteFile(path, []byte(`replicas: 5
name: "flat"
`), 0o644))

	k := kernel.New()
	src, err := k.LoadSourceFromFile(path)
	require.NoError(t, err)

	schema := cuecontext.New().CompileString(`{ replicas: int, name: string }`)
	require.NoError(t, schema.Err())
	merged, err := k.ValidateConfigDetailed(schema, []kernel.Source{src})
	require.NoError(t, err)

	replicas, err := merged.LookupPath(cue.ParsePath("replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(5), replicas)

	name, err := merged.LookupPath(cue.ParsePath("name")).String()
	require.NoError(t, err)
	assert.Equal(t, "flat", name)
}

// kernel-runtime spec, "Syntax errors fail at load, schema errors at use":
// a file that does not parse is refused by the loader, positioned at the
// file's path; a file that parses but violates the schema loads fine and
// fails, positioned at the file's path, in the operation that applies it.
func TestKernel_LoadSourceFromFile_SyntaxAtLoadSchemaAtUse(t *testing.T) {
	dir := t.TempDir()
	k := kernel.New()

	broken := filepath.Join(dir, "broken.cue")
	require.NoError(t, os.WriteFile(broken, []byte("replicas: {\n"), 0o644))
	src, err := k.LoadSourceFromFile(broken)
	require.Error(t, err, "a file that does not parse MUST be refused by the loader")
	assert.True(t, positionsName(err, broken), "positioned at the file's path: %v", err)
	assert.Equal(t, kernel.Source{}, src)

	violating := filepath.Join(dir, "violating.cue")
	require.NoError(t, os.WriteFile(violating, []byte("replicas: \"three\"\n"), 0o644))
	src, err = k.LoadSourceFromFile(violating)
	require.NoError(t, err, "a file that parses loads fine")

	schema := cuecontext.New().CompileString(`{ replicas: int }`)
	require.NoError(t, schema.Err())
	_, vErr := k.ValidateConfigDetailed(schema, []kernel.Source{src})
	require.Error(t, vErr, "the violation surfaces where the source is applied")
	assert.True(t, positionsName(vErr, violating), "positioned at the file's path: %v", vErr)
}

// A file-backed source is compiled from the bytes the Source carries, not
// re-read from disk: editing the file after LoadSourceFromFile does not
// change what the kernel validates.
func TestKernel_LoadSourceFromFile_CompilesCarriedBytes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "values.cue")
	require.NoError(t, os.WriteFile(path, []byte("replicas: 3\n"), 0o644))

	k := kernel.New()
	src, err := k.LoadSourceFromFile(path)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, []byte("replicas: \"changed\"\n"), 0o644))

	schema := cuecontext.New().CompileString(`{ replicas: int }`)
	require.NoError(t, schema.Err())
	merged, err := k.ValidateConfigDetailed(schema, []kernel.Source{src})
	require.NoError(t, err, "the bytes loaded are the bytes compiled")
	replicas, err := merged.LookupPath(cue.ParsePath("replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(3), replicas)
}
