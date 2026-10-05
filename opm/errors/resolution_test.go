package errors_test

import (
	"errors"
	"fmt"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	cueerrors "cuelang.org/go/cue/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/open-platform-model/library/opm/errors"
)

func TestResolutionError_MessageIsTheCause(t *testing.T) {
	cause := errors.New("cannot find module providing package a.b/c")
	re := &oerrors.ResolutionError{Kind: oerrors.ResolutionImportUnprovided, Err: cause}
	assert.Equal(t, cause.Error(), re.Error())
	assert.ErrorIs(t, re, cause)
}

func TestResolutionError_NilCauseDoesNotPanic(t *testing.T) {
	assert.Equal(t, "dependency resolution failed: import unprovided", (&oerrors.ResolutionError{Kind: oerrors.ResolutionImportUnprovided}).Error())
	assert.Equal(t, "dependency resolution failed: other", (&oerrors.ResolutionError{}).Error())
}

func TestResolutionError_NeverTransientNorFetch(t *testing.T) {
	err := fmt.Errorf("loading: %w", &oerrors.ResolutionError{Kind: oerrors.ResolutionImportUnprovided, Err: errors.New("x")})
	assert.NotErrorIs(t, err, oerrors.ErrTransient)
	var fe *oerrors.FetchError
	assert.False(t, errors.As(err, &fe))
}

// TestResolutionError_CUEListStaysReachable holds that cueerrors.Errors
// through a *ResolutionError returns the list's errors and positions.
func TestResolutionError_CUEListStaysReachable(t *testing.T) {
	v := cuecontext.New().CompileString("a: 1\na: 2\nb: 3\nb: 4\n")
	cause := v.Validate()
	require.Error(t, cause)
	re := &oerrors.ResolutionError{Kind: oerrors.ResolutionImportUnprovided, Err: cause}
	want := cueerrors.Errors(cause)
	got := cueerrors.Errors(re)
	require.Len(t, got, len(want))
	for i := range want {
		assert.Equal(t, want[i].Error(), got[i].Error())
		assert.Equal(t, cueerrors.Positions(want[i]), cueerrors.Positions(got[i]))
	}
}

func TestResolutionKind_String(t *testing.T) {
	assert.Equal(t, "other", oerrors.ResolutionOther.String())
	assert.Equal(t, "import unprovided", oerrors.ResolutionImportUnprovided.String())
	assert.Equal(t, "import ambiguous", oerrors.ResolutionImportAmbiguous.String())
	assert.Equal(t, "module file invalid", oerrors.ResolutionModuleFileInvalid.String())
}
