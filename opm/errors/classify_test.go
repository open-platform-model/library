package errors_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
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

// TestClassify_UnrecognisedIsUnchanged holds the errors that are neither a
// fetch failure nor an author-defect resolution failure: they come back as
// they are. An import no module provides is a resolution failure
// (TestClassify_Resolution).
func TestClassify_UnrecognisedIsUnchanged(t *testing.T) {
	for _, err := range []error{
		errors.New("main.cue:3:1: expected operand, found '}'"),
		errors.New("a: conflicting values 1 and 2"),
		errors.New("listening on port 401 Unauthorized"),
		// An import failure with a file position after the version is not
		// the direct-path module-file form.
		errors.New(`import failed: /c/mod/extract/test.example/dep@v0.0.2/dep.cue:3:8: expected operand, found '}'`),
		// The coordinate has to name an exact version, as cue/load writes
		// it; a major-only path is not the direct-path module-file form.
		errors.New("import failed: a.b/c@v0: x"),
		errors.New("x: 999 Bogus Status: y"),
		errors.New("x: 404 Something Else: y"),
	} {
		got := oerrors.Classify(err)
		assert.Same(t, err, got, "%q", err)
		assert.NotErrorIs(t, got, oerrors.ErrTransient)
		var re *oerrors.ResolutionError
		assert.False(t, errors.As(got, &re), "%q", err)
	}
}

// TestClassify_Resolution covers the author-defect half of the text
// fallback on constructed messages, and the rule that a fetch form anywhere
// in the text wins over it.
func TestClassify_Resolution(t *testing.T) {
	const unprovided = "cannot find module providing package example.com/x"
	for _, c := range []struct {
		name string
		msg  string
		kind oerrors.ResolutionKind
	}{
		// An import path carries at most a major version.
		{"unprovided import", `main.cue:3:8: cannot find package "a.b/c": cannot find module providing package a.b/c`, oerrors.ResolutionImportUnprovided},
		{"unprovided import at a major version", `main.cue:3:8: cannot find package "a.b/c@v1": cannot find module providing package a.b/c@v1`, oerrors.ResolutionImportUnprovided},
		{"the cli's unprovided form", unprovided, oerrors.ResolutionImportUnprovided},
		{"ambiguous import", "main.cue:3:8: cannot find package \"a.b/c\": ambiguous import: found package a.b/c in multiple locations:\n\ta.b@v0 (c)\n\ta.b/c@v0 v0.0.1 (.)", oerrors.ResolutionImportAmbiguous},
		{"module file in graph expansion", "cannot expand module graph: test.example/dep@v0.0.2: cannot parse module file from test.example/dep@v0.0.2: bogus: field not allowed", oerrors.ResolutionModuleFileInvalid},
		{"module file on the direct import path", "loading module package from /d (.): import failed: test.example/dep@v0.0.2: bogus: field not allowed", oerrors.ResolutionModuleFileInvalid},
		{"module file at a prerelease", "import failed: test.example/dep@v1.0.0-beta.1: bogus: field not allowed", oerrors.ResolutionModuleFileInvalid},
		// The first author-defect form in the order unprovided, ambiguous,
		// module file decides.
		{"several author-defect forms", "ambiguous import: x\ncannot parse module file from m@v0.0.1: y\n" + unprovided, oerrors.ResolutionImportUnprovided},
		{"ambiguous before module file", "cannot parse module file from m@v0.0.1: y\nambiguous import: x", oerrors.ResolutionImportAmbiguous},
	} {
		t.Run(c.name, func(t *testing.T) {
			assertResolution(t, errors.New(c.msg), c.kind)
		})
	}
	// A registry failure beside the unprovided form is a fetch failure: the
	// cli's TestUnprovidedImport_RegistryFailureIsNotAbsent pins the same
	// texts.
	for _, c := range []struct {
		name      string
		msg       string
		kind      oerrors.FetchKind
		status    int
		transient bool
	}{
		{"unprovided beside unreachable", unprovided + ": cannot do HTTP request: dial tcp: connection refused", oerrors.FetchUnreachable, 0, true},
		{"unprovided beside 503", unprovided + ": GET /v2/x: 503 Service Unavailable: busy", oerrors.FetchOther, 503, true},
		{"graph expansion carrying unreachable", "cannot expand module graph: m@v0.0.1: cannot do HTTP request: dial tcp: connection refused", oerrors.FetchUnreachable, 0, true},
		// The generic fetch form runs before the author-defect forms too.
		{"module file behind cannot fetch", "cannot fetch m@v0.0.1: cannot parse module file from m@v0.0.1: bogus: field not allowed", oerrors.FetchOther, 0, false},
		{"archive that does not unzip beside unprovided", "cannot fetch m@v0.0.1: unzip /c/m.zip: zip: not a valid zip file\n" + `main.cue:3:8: cannot find package "a.b/c": cannot find module providing package a.b/c`, oerrors.FetchOther, 0, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := errors.New(c.msg)
			assertClassified(t, err, c.kind, c.status, c.transient)
			var re *oerrors.ResolutionError
			assert.False(t, errors.As(oerrors.Classify(err), &re), "no *ResolutionError")
		})
	}
	t.Run("classifying twice", func(t *testing.T) {
		once := oerrors.Classify(errors.New(unprovided))
		assert.Same(t, once, oerrors.Classify(once))
		wrapped := fmt.Errorf("loading: %w", once)
		assert.Same(t, wrapped, oerrors.Classify(wrapped))
	})
}

func assertResolution(t *testing.T, err error, kind oerrors.ResolutionKind) {
	t.Helper()
	got := oerrors.Classify(err)
	var re *oerrors.ResolutionError
	require.True(t, errors.As(got, &re), "a resolution failure: %q", err)
	assert.Equal(t, kind, re.Kind, "kind of %q", err)
	var fe *oerrors.FetchError
	assert.False(t, errors.As(got, &fe), "no *FetchError in %q", err)
	assert.NotErrorIs(t, got, oerrors.ErrTransient)
	assert.Equal(t, err.Error(), got.Error(), "message unchanged")
	assert.ErrorIs(t, got, err, "the cause stays reachable")
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
		// A failed round trip is a net.Error. It is no response, unless its
		// text is the token endpoint's answer to a token refresh.
		{"round trip with no response", roundTrip(&net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")}), oerrors.FetchUnreachable, 0, true},
		{"round trip holding a token refresh 401", roundTrip(errors.New("cannot acquire access token: 401 Unauthorized")), oerrors.FetchUnauthorized, 401, false},
		{"round trip holding a token refresh 403", roundTrip(errors.New("cannot acquire access token: 403 Forbidden")), oerrors.FetchUnauthorized, 403, false},
		{"round trip holding a token refresh 503", roundTrip(errors.New("cannot acquire access token: 503 Service Unavailable")), oerrors.FetchOther, 503, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertClassified(t, c.err, c.kind, c.status, c.transient)
		})
	}
}

// roundTrip is the chain the OCI client returns when an HTTP round trip
// fails with cause.
func roundTrip(cause error) error {
	return fmt.Errorf("cannot do HTTP request: %w", &url.Error{Op: "Get", URL: "http://h/v2/", Err: cause})
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
		// A token endpoint that answered: the no-response prefix, then the
		// status at the end of the line.
		{"token answer 401 on a push", `cannot make scratch config: cannot do HTTP request: Post "http://h/v2/m/blobs/uploads/": 401 Unauthorized`, oerrors.FetchUnauthorized, 401, false},
		{"token answer 403 on a push", `cannot make scratch config: cannot do HTTP request: Post "http://h/v2/m/blobs/uploads/": 403 Forbidden`, oerrors.FetchUnauthorized, 403, false},
		{"token answer 401 on a load", `import failed: /d/main.cue:3:8: cannot find package "a.b/c": cannot fetch a.b@v0.0.1: module a.b@v0.0.1: cannot do HTTP request: Get "http://h/v2/a.b/manifests/v0.0.1": 401 Unauthorized`, oerrors.FetchUnauthorized, 401, false},
		{"token answer 503", `cannot do HTTP request: Post "http://h/v2/m/blobs/uploads/": 503 Service Unavailable`, oerrors.FetchOther, 503, true},
		{"token answer on the first of two lines", "cannot do HTTP request: Get \"http://h/v2/\": 403 Forbidden\nand more", oerrors.FetchUnauthorized, 403, false},
		{"token answer on a refresh", `cannot do HTTP request: Get "http://h/v2/": cannot acquire access token: 401 Unauthorized`, oerrors.FetchUnauthorized, 401, false},
		// Not a token answer: each stays a request that got no response.
		{"a status text that is not the code's", `cannot do HTTP request: Get "http://h/v2/": 401 Forbidden`, oerrors.FetchUnreachable, 0, true},
		{"a code below 400", `cannot do HTTP request: Get "http://h/v2/": 200 OK`, oerrors.FetchUnreachable, 0, true},
		{"a status that is not at the end of the line", `cannot do HTTP request: Get "http://h/v2/": 403 Forbidden, then dial tcp: connection refused`, oerrors.FetchUnreachable, 0, true},
		{"a status in the quoted URL", `cannot do HTTP request: Get "http://h/v2/x:%20403%20Forbidden": dial tcp: connection refused`, oerrors.FetchUnreachable, 0, true},
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
