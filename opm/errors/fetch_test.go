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

func TestFetchError_MessageIsTheCauses(t *testing.T) {
	cause := errors.New("module test.example/dep@v0.0.2: module not found")
	fe := &oerrors.FetchError{Kind: oerrors.FetchNotFound, Err: cause}
	assert.Equal(t, cause.Error(), fe.Error())
	assert.Same(t, cause, errors.Unwrap(fe))
	assert.ErrorIs(t, fmt.Errorf("fetching: %w", fe), cause)
}

func TestFetchError_NilCauseDoesNotPanic(t *testing.T) {
	fe := &oerrors.FetchError{Kind: oerrors.FetchUnreachable}
	assert.NotPanics(t, func() { _ = fe.Error() })
	assert.Contains(t, fe.Error(), "unreachable")
	assert.NoError(t, fe.Unwrap())
}

func TestFetchKind_String(t *testing.T) {
	assert.Equal(t, "other", oerrors.FetchOther.String())
	assert.Equal(t, "not found", oerrors.FetchNotFound.String())
	assert.Equal(t, "unauthorized", oerrors.FetchUnauthorized.String())
	assert.Equal(t, "unreachable", oerrors.FetchUnreachable.String())
}

// ErrTransient holds exactly for an unreachable registry and a 5xx answer.
func TestFetchError_Transient(t *testing.T) {
	cases := []struct {
		name      string
		fe        oerrors.FetchError
		transient bool
	}{
		{"unreachable", oerrors.FetchError{Kind: oerrors.FetchUnreachable}, true},
		{"not found", oerrors.FetchError{Kind: oerrors.FetchNotFound, Status: 404}, false},
		{"unauthorized", oerrors.FetchError{Kind: oerrors.FetchUnauthorized, Status: 401}, false},
		{"other without status", oerrors.FetchError{Kind: oerrors.FetchOther}, false},
		{"429", oerrors.FetchError{Kind: oerrors.FetchOther, Status: 429}, false},
		{"500", oerrors.FetchError{Kind: oerrors.FetchOther, Status: 500}, true},
		{"503", oerrors.FetchError{Kind: oerrors.FetchOther, Status: 503}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fe := c.fe
			fe.Err = errors.New("cause")
			wrapped := fmt.Errorf("fetching: %w", &fe)
			assert.Equal(t, c.transient, errors.Is(wrapped, oerrors.ErrTransient))
			assert.Equal(t, c.transient, fe.Transient())
		})
	}
	assert.NotErrorIs(t, errors.New("plain"), oerrors.ErrTransient, "no FetchError in the chain")
}

// A FetchError wrapping a CUE error list keeps every error and position
// reachable through cueerrors.Errors.
func TestFetchError_KeepsCUEErrorPositions(t *testing.T) {
	v := cuecontext.New().CompileString("a: 1\na: 2\nb: string\nb: 3\n")
	cause := v.Validate()
	require.Error(t, cause)
	want := cueerrors.Errors(cause)
	require.NotEmpty(t, want)

	got := cueerrors.Errors(fmt.Errorf("loading: %w", &oerrors.FetchError{Kind: oerrors.FetchOther, Err: cause}))
	require.Len(t, got, len(want))
	for i := range want {
		assert.Equal(t, want[i].Error(), got[i].Error())
		assert.Equal(t, cueerrors.Positions(want[i]), cueerrors.Positions(got[i]))
	}
	var list cueerrors.Error
	assert.True(t, errors.As(&oerrors.FetchError{Err: cause}, &list))
}
