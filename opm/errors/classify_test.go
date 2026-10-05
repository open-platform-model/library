package errors_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"

	"cuelabs.dev/go/oci/ociregistry"
	"cuelang.org/go/mod/modregistry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/open-platform-model/library/opm/errors"
)

func TestClassify_Nil(t *testing.T) {
	assert.NoError(t, oerrors.Classify(nil))
}

func TestClassify_AlreadyClassifiedIsReturnedAsIs(t *testing.T) {
	// The inner cause is one Classify would recognise on its own, so only
	// the already-classified guard keeps the inner Coordinate visible.
	err := fmt.Errorf("fetching: %w", &oerrors.FetchError{Kind: oerrors.FetchNotFound, Coordinate: "m@v0.0.1", Err: modregistry.ErrNotFound})
	got := oerrors.Classify(err)
	assert.Same(t, err, got)
	var fe *oerrors.FetchError
	require.True(t, errors.As(got, &fe))
	assert.Equal(t, "m@v0.0.1", fe.Coordinate)
}

func TestClassify_CancellationIsUnchanged(t *testing.T) {
	err := fmt.Errorf("cannot do HTTP request: %w", context.Canceled)
	got := oerrors.Classify(err)
	assert.Same(t, err, got)
	assert.ErrorIs(t, got, context.Canceled)
	assert.NotErrorIs(t, got, oerrors.ErrTransient)
}

func TestClassify_UnrecognisedIsUnchanged(t *testing.T) {
	for _, err := range []error{
		errors.New("main.cue:3:1: expected operand, found '}'"),
		errors.New("a: conflicting values 1 and 2"),
		errors.New("cannot expand module graph: test.example/dep@v0.0.2: cannot parse module file from test.example/dep@v0.0.2: bogus: field not allowed"),
		errors.New("listening on port 401 Unauthorized"),
		// An import no module provides is an author defect: an import path
		// carries at most a major version.
		errors.New(`main.cue:3:8: cannot find package "a.b/c": cannot find module providing package a.b/c`),
		errors.New(`main.cue:3:8: cannot find package "a.b/c@v1": cannot find module providing package a.b/c@v1`),
		errors.New("x: 999 Bogus Status: y"),
		errors.New("x: 404 Something Else: y"),
	} {
		got := oerrors.Classify(err)
		assert.Same(t, err, got, "%q", err)
		assert.NotErrorIs(t, got, oerrors.ErrTransient)
	}
}

// TestClassify_Typed covers the typed branch on constructed causes.
func TestClassify_Typed(t *testing.T) {
	httpErr := func(status int) error {
		return fmt.Errorf("module m@v0.0.1: %w", ociregistry.NewHTTPError(errors.New("answer"), status, nil, nil))
	}
	cases := []struct {
		name      string
		err       error
		kind      oerrors.FetchKind
		status    int
		transient bool
	}{
		{"deadline", fmt.Errorf("x: %w", context.DeadlineExceeded), oerrors.FetchUnreachable, 0, true},
		{"401", httpErr(401), oerrors.FetchUnauthorized, 401, false},
		{"403", httpErr(403), oerrors.FetchUnauthorized, 403, false},
		{"404", httpErr(404), oerrors.FetchNotFound, 404, false},
		{"429", httpErr(429), oerrors.FetchOther, 429, false},
		{"500", httpErr(500), oerrors.FetchOther, 500, true},
		{"503", httpErr(503), oerrors.FetchOther, 503, true},
		{"modregistry not found", fmt.Errorf("module m@v0.0.1: %w", modregistry.ErrNotFound), oerrors.FetchNotFound, 0, false},
		{"name unknown", fmt.Errorf("x: %w", ociregistry.ErrNameUnknown), oerrors.FetchNotFound, 0, false},
		{"manifest unknown", fmt.Errorf("x: %w", ociregistry.ErrManifestUnknown), oerrors.FetchNotFound, 0, false},
		{"blob unknown", fmt.Errorf("x: %w", ociregistry.ErrBlobUnknown), oerrors.FetchNotFound, 0, false},
		{"unauthorized code", fmt.Errorf("x: %w", ociregistry.ErrUnauthorized), oerrors.FetchUnauthorized, 0, false},
		{"denied code", fmt.Errorf("x: %w", ociregistry.ErrDenied), oerrors.FetchUnauthorized, 0, false},
		{"net.OpError", fmt.Errorf("x: %w", &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")}), oerrors.FetchUnreachable, 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertClassified(t, c.err, c.kind, c.status, c.transient)
		})
	}
}

// TestClassify_Text covers each text form of the fallback on constructed
// messages; classify_cue_test.go pins the same forms against the embedded
// CUE.
func TestClassify_Text(t *testing.T) {
	cases := []struct {
		name      string
		msg       string
		kind      oerrors.FetchKind
		status    int
		transient bool
	}{
		{"unreachable", `cannot fetch m@v0.0.1: module m@v0.0.1: cannot do HTTP request: Get "http://h/v2/": dial tcp: connection refused`, oerrors.FetchUnreachable, 0, true},
		{"401", "cannot fetch m@v0.0.1: module m@v0.0.1: 401 Unauthorized: unauthorized: no", oerrors.FetchUnauthorized, 401, false},
		{"403", "x: 403 Forbidden: denied: no", oerrors.FetchUnauthorized, 403, false},
		{"404", "x: 404 Not Found: y", oerrors.FetchNotFound, 404, false},
		{"429", "x: 429 Too Many Requests: y", oerrors.FetchOther, 429, false},
		{"503", `x: 503 Service Unavailable: non-JSON error response ""; body ""`, oerrors.FetchOther, 503, true},
		{"module not found", "cannot fetch m@v0.0.1: module m@v0.0.1: module not found", oerrors.FetchNotFound, 0, false},
		{"no module provides an exact version", `cannot find module providing package a.b/c@v1.2.3`, oerrors.FetchNotFound, 0, false},
		{"tidy unreachable", `failed to resolve "a.b/c@v0": module a.b@v0: cannot do HTTP request: Get "http://h/v2/a.b/tags/list?n=1000": dial tcp: connection refused`, oerrors.FetchUnreachable, 0, true},
		{"cannot fetch other", "cannot fetch m@v0.0.1: unzip /c/m.zip: zip: not a valid zip file", oerrors.FetchOther, 0, false},
		{"graph expansion carrying a fetch form", "cannot expand module graph: m@v0.0.1: module not found", oerrors.FetchNotFound, 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertClassified(t, errors.New(c.msg), c.kind, c.status, c.transient)
		})
	}
}

func assertClassified(t *testing.T, err error, kind oerrors.FetchKind, status int, transient bool) {
	t.Helper()
	got := oerrors.Classify(err)
	var fe *oerrors.FetchError
	require.True(t, errors.As(got, &fe), "classified: %q", err)
	assert.Equal(t, kind, fe.Kind, "kind of %q", err)
	assert.Equal(t, status, fe.Status, "status of %q", err)
	assert.Equal(t, transient, errors.Is(got, oerrors.ErrTransient), "transient of %q", err)
	assert.Equal(t, err.Error(), got.Error(), "message unchanged")
	assert.ErrorIs(t, got, err, "the cause stays reachable")
	assert.Empty(t, fe.Coordinate, "Classify never invents a coordinate")
}
