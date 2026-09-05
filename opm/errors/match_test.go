package errors_test

import (
	"errors"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	cueerrors "cuelang.org/go/cue/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/open-platform-model/library/opm/errors"
)

func TestUnifyError_VerbatimCueCauseReachable(t *testing.T) {
	ctx := cuecontext.New()
	a := ctx.CompileString(`x: "foo"`)
	require.NoError(t, a.Err())
	b := ctx.CompileString(`x: "bar"`)
	require.NoError(t, b.Err())

	cueErr := a.Unify(b).Validate()
	require.Error(t, cueErr, "conflicting string values must produce a CUE error")

	ue := oerrors.UnifyError{
		Component: "web",
		FQN:       "example.com/r/container@v0",
		Cause:     cueErr,
	}

	// The verbatim CUE message is preserved on the Cause.
	assert.Equal(t, cueErr.Error(), ue.Cause.Error())
	assert.Contains(t, ue.Error(), cueErr.Error(), "wrapper surfaces the cause verbatim")

	// Unwrap reaches the cause, and errors.As reaches the CUE error tree.
	assert.Equal(t, cueErr, errors.Unwrap(ue))
	var asCue cueerrors.Error
	require.True(t, errors.As(ue, &asCue), "UnifyError must be walkable to cuelang.org/go/cue/errors.Error")
}

func TestUnresolvedDemand_Shape(t *testing.T) {
	// No alternatives: the contract is unimplemented on this platform.
	bare := oerrors.UnresolvedDemand{Component: "web", FQN: "example.com/r/volume@v1", Kind: "resource"}
	assert.Contains(t, bare.Error(), `component "web"`)
	assert.Contains(t, bare.Error(), "unresolved resource demand")
	assert.Contains(t, bare.Error(), "nothing on this platform implements this contract")

	// Alternatives present: the D4 different-apiVersion diagnostic.
	alt := oerrors.UnresolvedDemand{
		Component:    "web",
		FQN:          "example.com/r/volume@v1",
		Kind:         "resource",
		Alternatives: []string{"example.com/r/volume@v2"},
	}
	assert.Contains(t, alt.Error(), "implemented at a different apiVersion")
	assert.Contains(t, alt.Error(), "example.com/r/volume@v2")

	// Disqualified candidates are counted.
	disq := oerrors.UnresolvedDemand{
		Component:    "web",
		FQN:          "example.com/t/backup@v1",
		Kind:         "trait",
		Disqualified: []oerrors.UnifyError{{Component: "web", FQN: "example.com/t/backup@v1"}},
	}
	assert.Contains(t, disq.Error(), "1 candidate(s) disqualified")
}

func TestUnresolvedDemandsError_UnwrapWalkable(t *testing.T) {
	agg := &oerrors.UnresolvedDemandsError{Demands: []oerrors.UnresolvedDemand{
		{Component: "web", FQN: "example.com/r/a@v1", Kind: "resource"},
		{Component: "db", FQN: "example.com/t/b@v1", Kind: "trait"},
	}}
	assert.Contains(t, agg.Error(), "2 unresolved demand(s)")
	assert.Contains(t, agg.Error(), `component "web"`)
	assert.Contains(t, agg.Error(), `component "db"`)

	// errors.As reaches the value-typed demand through the aggregate.
	var d oerrors.UnresolvedDemand
	require.True(t, errors.As(agg, &d))
	assert.Equal(t, "web", d.Component, "errors.As finds the first demand")
}
