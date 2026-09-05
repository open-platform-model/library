package kernel

import (
	"testing"

	"cuelang.org/go/cue"
	cueerrors "cuelang.org/go/cue/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Partial mode (requireConcrete=false) is kernel-internal: it is the
// attribution pass AcquireInstanceFromDir runs over WithValues sources. These
// tests pin its behaviour against validateSources directly, since no public
// method exposes it.

func internalSource(t *testing.T, k *Kernel, origin, src string) Source {
	t.Helper()
	s, err := k.LoadSourceFromBytes(origin, []byte(src))
	require.NoError(t, err)
	return s
}

func TestValidateSources_PartialAllowsMissingRequiredFields(t *testing.T) {
	k := New()
	// Schema requires both `replicas` and `name` to be concrete.
	schema := k.CueContext().CompileString(`{ replicas: int & >0, name: string }`)
	require.NoError(t, schema.Err())
	// Partial value sets only `replicas`; `name` is missing.
	partial := internalSource(t, k, "partial.cue", `{ replicas: 3 }`)

	_, partialErr := validateSources(schema, []Source{partial}, false)
	require.NoError(t, partialErr, "partial validation MUST allow missing required fields")

	_, fullErr := validateSources(schema, []Source{partial}, true)
	require.Error(t, fullErr, "concrete validation MUST flag the missing required field")
}

func TestValidateSources_PartialTypeErrorStillSurfaces(t *testing.T) {
	k := New()
	schema := k.CueContext().CompileString(`{ replicas: int & >0, name: string }`)
	require.NoError(t, schema.Err())
	// `replicas` set but with the wrong type — partial validation MUST flag it.
	wrongType := internalSource(t, k, "wrong.cue", `{ replicas: "three" }`)

	_, vErr := validateSources(schema, []Source{wrongType}, false)
	require.Error(t, vErr, "partial validation still flags type errors on fields that ARE set")
	require.NotEmpty(t, cueerrors.Errors(vErr))
}

func TestValidateSources_PartialZeroValueIsNoOp(t *testing.T) {
	k := New()
	schema := k.CueContext().CompileString(`{ replicas: int & >0 }`)
	require.NoError(t, schema.Err())

	got, err := validateSources(schema, []Source{{Value: cue.Value{}, Origin: "none"}}, false)
	require.NoError(t, err)
	assert.False(t, got.Exists())

	got, err = validateSources(schema, nil, false)
	require.NoError(t, err)
	assert.False(t, got.Exists())
}

func TestValidateSources_PartialSkipsConcreteButRunsWalkDisallowed(t *testing.T) {
	k := New()
	// Closed schema requiring two fields. Stray field is disallowed.
	schema := k.CueContext().CompileString(`close({ replicas: int & >0, name: string })`)
	require.NoError(t, schema.Err())

	// Single source with only `replicas` set + a stray field.
	src := internalSource(t, k, "draft.cue", `{replicas: 1, stray: "x"}`)

	// Concrete: missing `name` AND stray field both fail.
	_, fullErr := validateSources(schema, []Source{src}, true)
	require.Error(t, fullErr, "concrete check fails on missing required field")

	// Partial: missing `name` ignored; stray STILL surfaces (walkDisallowed).
	_, partErr := validateSources(schema, []Source{src}, false)
	require.Error(t, partErr, "partial mode does NOT silence walkDisallowed disallowed-field errors")

	// Verify the partial error is specifically about the stray field, not missing-name.
	gotStrayMessage := false
	for _, ce := range cueerrors.Errors(partErr) {
		f, _ := ce.Msg()
		if f == "field not allowed" {
			gotStrayMessage = true
		}
	}
	assert.True(t, gotStrayMessage, "partial mode error MUST include 'field not allowed' for stray field")
}

func TestValidateSources_PartialLayeredSources(t *testing.T) {
	k := New()
	schema := k.CueContext().CompileString(`{ replicas: int & >0, name: string, image: string }`)
	require.NoError(t, schema.Err())

	a := internalSource(t, k, "a.cue", `replicas: 2`)
	b := internalSource(t, k, "b.cue", `name: "inst"`)
	// image still missing: partial tolerates it, concrete does not.
	layered, vErr := validateSources(schema, []Source{a, b}, false)
	require.NoError(t, vErr)
	assert.True(t, layered.Exists())

	_, fullErr := validateSources(schema, []Source{a, b}, true)
	require.Error(t, fullErr)
}
