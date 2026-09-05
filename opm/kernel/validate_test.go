package kernel_test

import (
	"testing"

	"cuelang.org/go/cue"
	cueerrors "cuelang.org/go/cue/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/module"
)

func TestKernel_ValidateConfigDetailed_ZeroValueSourceIsNoOp(t *testing.T) {
	k := kernel.New()
	schema := k.CueContext().CompileString(`{ replicas: int & >0 }`)
	require.NoError(t, schema.Err())

	got, err := k.ValidateConfigDetailed(schema, []kernel.Source{{Value: cue.Value{}, Origin: "none"}})
	require.NoError(t, err, "a source whose merged value does not exist MUST be treated as 'no values' and succeed")
	assert.False(t, got.Exists(), "no values supplied → returned cue.Value is the zero value")
}

func TestKernel_ValidateConfigDetailed_SchemaErrorReturnsCueError(t *testing.T) {
	k := kernel.New()
	schema := k.CueContext().CompileString(`{ replicas: int & >0 }`)
	require.NoError(t, schema.Err())
	bad := mustSource(t, k, "bad.cue", `{ replicas: -1 }`)

	_, vErr := k.ValidateConfigDetailed(schema, []kernel.Source{bad})
	require.Error(t, vErr)
	// Error MUST be walkable as a CUE error tree.
	require.NotEmpty(t, cueerrors.Errors(vErr), "validation error MUST yield at least one cueerrors.Error")
}

func TestKernel_ValidateConfigDetailed_FieldNotAllowed(t *testing.T) {
	k := kernel.New()
	schema := k.CueContext().CompileString(`close({ replicas: int })`)
	require.NoError(t, schema.Err())
	stray := mustSource(t, k, "stray.cue", `{ replicas: 1, stray: "x" }`)

	_, vErr := k.ValidateConfigDetailed(schema, []kernel.Source{stray})
	require.Error(t, vErr, "field outside the closed schema MUST surface")
}

func TestKernel_ValidateConfigDetailed_MissingRequiredFieldFails(t *testing.T) {
	k := kernel.New()
	// Schema requires both `replicas` and `name` to be concrete.
	schema := k.CueContext().CompileString(`{ replicas: int & >0, name: string }`)
	require.NoError(t, schema.Err())
	// Only `replicas` is set; `name` is missing.
	partial := mustSource(t, k, "partial.cue", `{ replicas: 3 }`)

	_, err := k.ValidateConfigDetailed(schema, []kernel.Source{partial})
	require.Error(t, err, "the one public primitive is concrete-only: a missing required field MUST fail")
}

func TestKernel_ValidateConfigDetailed_EmptySourcesReturnsZeroNil(t *testing.T) {
	k := kernel.New()
	schema := k.CueContext().CompileString(`{ replicas: int & >0 }`)
	require.NoError(t, schema.Err())

	got, err := k.ValidateConfigDetailed(schema, nil)
	require.NoError(t, err)
	assert.False(t, got.Exists(), "empty sources MUST return zero cue.Value, nil error")
}

func TestKernel_ValidateConfigDetailed_SingleSourceSuccess(t *testing.T) {
	k := kernel.New()
	schema := k.CueContext().CompileString(`{ replicas: int & >0 }`)
	require.NoError(t, schema.Err())
	v := mustSource(t, k, "user.cue", `replicas: 3`)

	merged, vErr := k.ValidateConfigDetailed(schema, []kernel.Source{v})
	require.NoError(t, vErr)
	assert.True(t, merged.Exists())
}

func TestKernel_ValidateConfigDetailed_TwoSourceUnifySuccess(t *testing.T) {
	k := kernel.New()
	schema := k.CueContext().CompileString(`{ replicas: int & >0, image: string }`)
	require.NoError(t, schema.Err())

	a := mustSource(t, k, "defaults.cue", `replicas: 1`)
	b := mustSource(t, k, "user.cue", `image: "nginx"`)

	merged, vErr := k.ValidateConfigDetailed(schema, []kernel.Source{a, b})
	require.NoError(t, vErr)
	require.True(t, merged.Exists())

	// Sanity: both fields present in the merged value.
	rep, _ := merged.LookupPath(cue.ParsePath("replicas")).Int64()
	assert.Equal(t, int64(1), rep)
}

func TestKernel_ValidateConfigDetailed_ConflictSurfacesBothPositions(t *testing.T) {
	k := kernel.New()
	schema := k.CueContext().CompileString(`{ replicas: int & >0 }`)
	require.NoError(t, schema.Err())

	a := mustSource(t, k, "a.cue", `replicas: 3`)
	b := mustSource(t, k, "b.cue", `replicas: 5`)

	_, vErr := k.ValidateConfigDetailed(schema, []kernel.Source{a, b})
	require.Error(t, vErr, "conflicting concrete values MUST fail")

	filenames := map[string]bool{}
	for _, ce := range cueerrors.Errors(vErr) {
		for _, pos := range cueerrors.Positions(ce) {
			if pos.IsValid() {
				filenames[pos.Filename()] = true
			}
		}
	}
	assert.True(t, filenames["a.cue"], "diagnostics MUST cite the originating Source.Origin (a.cue)")
	assert.True(t, filenames["b.cue"], "diagnostics MUST cite the originating Source.Origin (b.cue)")
}

// The composition tests below cover what the retired typed wrappers
// (ValidateModuleValues* / ValidateInstanceValues*) used to: resolving
// #config via the ConfigSchema() accessors and handing it to the primitive.

// configFixture is a minimal Module + Instance pair sharing one #config
// schema (replicas: int & >0, name: string), built as bare values in the
// kernel's context: the ConfigSchema() accessors read Package only, so no
// source, registry or platform is involved.
type configFixture struct {
	mod  *module.Module
	inst *module.Instance
}

func newConfigFixture(t *testing.T, k *kernel.Kernel) configFixture {
	t.Helper()
	ctx := k.CueContext()

	modPkg := ctx.CompileString(`
kind: "Module"
metadata: {
	name: "demo-mod"
	modulePath: "example.com/m"
	version: "1.0.0"
	fqn: "example.com/m/demo-mod:1.0.0"
	uuid: "11111111-1111-1111-1111-111111111111"
}
#config: {
	replicas: int & >0
	name: string
}
`)
	require.NoError(t, modPkg.Err())

	instPkg := ctx.CompileString(`
kind: "ModuleInstance"
metadata: { name: "demo", namespace: "ns", uuid: "u-inst" }
#module: {
	kind: "Module"
	metadata: {
		name: "demo-mod"
		modulePath: "example.com/m"
		version: "1.0.0"
		fqn: "example.com/m/demo-mod:1.0.0"
		uuid: "11111111-1111-1111-1111-111111111111"
	}
	#config: {
		replicas: int & >0
		name: string
	}
}
components: {}
`)
	require.NoError(t, instPkg.Err())

	return configFixture{
		mod: &module.Module{
			Metadata: &module.ModuleMetadata{
				Name:       "demo-mod",
				ModulePath: "example.com/m",
				Version:    "1.0.0",
				FQN:        "example.com/m/demo-mod:1.0.0",
				UUID:       "11111111-1111-1111-1111-111111111111",
			},
			Package: modPkg,
		},
		inst: &module.Instance{
			Metadata: &module.InstanceMetadata{Name: "demo", Namespace: "ns", UUID: "u-inst"},
			Package:  instPkg,
		},
	}
}

func TestKernel_ValidateConfigDetailed_ComposedWithModuleConfigSchema(t *testing.T) {
	k := kernel.New()
	f := newConfigFixture(t, k)

	values := mustSource(t, k, "values.cue", `{ replicas: 3, name: "demo" }`)
	merged, err := k.ValidateConfigDetailed(f.mod.ConfigSchema(), []kernel.Source{values})
	require.NoError(t, err)
	assert.True(t, merged.Exists())

	bad := mustSource(t, k, "bad.cue", `{ replicas: -1, name: "demo" }`)
	_, vErr := k.ValidateConfigDetailed(f.mod.ConfigSchema(), []kernel.Source{bad})
	require.Error(t, vErr)

	// Only replicas set; name missing — the concrete check flags it.
	partial := mustSource(t, k, "partial.cue", `{ replicas: 3 }`)
	_, fullErr := k.ValidateConfigDetailed(f.mod.ConfigSchema(), []kernel.Source{partial})
	require.Error(t, fullErr, "concrete check MUST flag missing required field")

	// Layered: two sources together satisfy the schema.
	a := mustSource(t, k, "defaults.cue", `replicas: 1`)
	b := mustSource(t, k, "user.cue", `name: "prod"`)
	layered, vErr := k.ValidateConfigDetailed(f.mod.ConfigSchema(), []kernel.Source{a, b})
	require.NoError(t, vErr)
	assert.True(t, layered.Exists())
}

func TestKernel_ValidateConfigDetailed_ComposedWithInstanceConfigSchema(t *testing.T) {
	k := kernel.New()
	f := newConfigFixture(t, k)

	values := mustSource(t, k, "values.cue", `{ replicas: 3, name: "demo" }`)
	merged, err := k.ValidateConfigDetailed(f.inst.ConfigSchema(), []kernel.Source{values})
	require.NoError(t, err)
	assert.True(t, merged.Exists())

	a := mustSource(t, k, "a.cue", `replicas: 2`)
	b := mustSource(t, k, "b.cue", `name: "inst"`)
	layered, vErr := k.ValidateConfigDetailed(f.inst.ConfigSchema(), []kernel.Source{a, b})
	require.NoError(t, vErr)
	assert.True(t, layered.Exists())
}
