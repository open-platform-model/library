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

func TestMissingFQN_Shape(t *testing.T) {
	e := oerrors.MissingFQN{
		Instance:     "demo",
		Component:    "web",
		FQN:          "example.com/r/container@1.0.0",
		Alternatives: []string{"example.com/r/container@1.1.0"},
	}
	msg := e.Error()
	assert.Contains(t, msg, `instance "demo"`)
	assert.Contains(t, msg, `component "web"`)
	assert.Contains(t, msg, "example.com/r/container@1.0.0")
	assert.Contains(t, msg, "example.com/r/container@1.1.0")

	// With no alternatives the message omits the alternatives clause.
	bare := oerrors.MissingFQN{Instance: "demo", Component: "web", FQN: "x@1.0.0"}
	assert.NotContains(t, bare.Error(), "alternatives")
}

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
