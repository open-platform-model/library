package registrytest

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// NewStatusRegistry stands up an HTTP server that answers every request with
// status, and returns a CUE_REGISTRY value routing every module path to it
// (with +insecure, and no other mapping, so nothing dials a public
// registry). The body is an OCI distribution error for the codes that have
// one (UNAUTHORIZED for 401, DENIED for 403, NAME_UNKNOWN for 404,
// TOOMANYREQUESTS for 429), and empty otherwise, as a 5xx from a proxy
// usually is. It exists to pin how the embedded CUE reports a registry that
// answered with an error; the server is closed at test end. It sets no
// environment: the caller sets CUE_REGISTRY and CUE_CACHE_DIR.
func NewStatusRegistry(t *testing.T, status int) string {
	t.Helper()
	code := map[int]string{
		http.StatusUnauthorized:    "UNAUTHORIZED",
		http.StatusForbidden:       "DENIED",
		http.StatusNotFound:        "NAME_UNKNOWN",
		http.StatusTooManyRequests: "TOOMANYREQUESTS",
	}[status]
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if code == "" {
			w.WriteHeader(status)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = fmt.Fprintf(w, `{"errors":[{"code":%q,"message":"registrytest status %d"}]}`, code, status)
	}))
	t.Cleanup(srv.Close)
	return strings.TrimPrefix(srv.URL, "http://") + "+insecure"
}

// UnreachableRegistry returns a CUE_REGISTRY value routing every module path
// to a loopback address nothing listens on: a listener is opened to reserve a
// free port and closed again, so a request is refused without leaving the
// host. Like [NewStatusRegistry] it sets no environment.
func UnreachableRegistry(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "reserve a loopback port")
	addr := l.Addr().String()
	require.NoError(t, l.Close(), "release the loopback port")
	return addr + "+insecure"
}
