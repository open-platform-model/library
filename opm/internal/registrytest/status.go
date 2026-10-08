package registrytest

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync/atomic"
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

// RefuseFirst returns mapping with prefix routed through a proxy in front of
// the host mapping names for it: the proxy answers the first refusals
// requests with 503 Service Unavailable and forwards every later one. It
// exists to pin that a transient registry failure is not remembered past the
// operation that saw it. mapping is one the constructors here return
// ("<prefix>=<host>+insecure,..."); the proxy is closed at test end. It sets
// no environment.
func RefuseFirst(t *testing.T, mapping, prefix string, refusals int) string {
	t.Helper()
	first, rest, _ := strings.Cut(mapping, ",")
	host, ok := strings.CutPrefix(first, prefix+"=")
	require.True(t, ok, "mapping %q routes %s first", mapping, prefix)
	host = strings.TrimSuffix(host, "+insecure")
	target, err := url.Parse("http://" + host)
	require.NoError(t, err)
	proxy := httputil.NewSingleHostReverseProxy(target)
	var seen atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if seen.Add(1) <= int64(refusals) {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		proxy.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)
	routed := prefix + "=" + strings.TrimPrefix(srv.URL, "http://") + "+insecure"
	if rest != "" {
		routed += "," + rest
	}
	return routed
}

// NewTokenRegistry returns a CUE_REGISTRY value routing every module path to
// a local server that uses token authentication, as GHCR and Docker Hub do:
// every registry request is answered 401 with a Bearer challenge naming the
// server's own /token endpoint, and that token endpoint answers every
// request with tokenStatus (the OCI distribution error body for 401 and 403,
// an empty body otherwise). It exists to pin how the embedded CUE reports a
// token endpoint that refuses; the server is closed at test end. Like
// [NewStatusRegistry] it sets no environment.
func NewTokenRegistry(t *testing.T, tokenStatus int) string {
	t.Helper()
	code := map[int]string{
		http.StatusUnauthorized: "UNAUTHORIZED",
		http.StatusForbidden:    "DENIED",
	}[tokenStatus]
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/token" {
			w.Header().Set("Www-Authenticate", fmt.Sprintf(`Bearer realm=%q,service="registrytest"`, srv.URL+"/token"))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = fmt.Fprint(w, `{"errors":[{"code":"UNAUTHORIZED","message":"registrytest token required"}]}`)
			return
		}
		if code == "" {
			w.WriteHeader(tokenStatus)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(tokenStatus)
		_, _ = fmt.Fprintf(w, `{"errors":[{"code":%q,"message":"registrytest token status %d"}]}`, code, tokenStatus)
	}))
	t.Cleanup(srv.Close)
	return strings.TrimPrefix(srv.URL, "http://") + "+insecure"
}
