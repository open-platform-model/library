package errors_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"cuelabs.dev/go/oci/ociregistry"
	"cuelabs.dev/go/oci/ociregistry/ocimem"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/mod/modregistry"
	"cuelang.org/go/mod/modregistrytest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/internal/cueenv"
	"github.com/open-platform-model/library/opm/internal/loader"
	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/internal/schematest"
	opmmodule "github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/schema"
)

// This file pins the forms in which the embedded CUE (v0.17.1) reports a
// registry fetch or a dependency resolution failure, as the library's own
// sites receive them. Classify's text fallback matches exactly these forms,
// so a CUE bump that changes one fails here first.
//
// Every case resolves against a local registry only (no catch-all mapping,
// so nothing dials a public registry) and a fresh module cache, so a warm
// cache never hides a failure.

const (
	depPath    = "test.example/dep@v0"
	depVersion = "v0.0.2"
	depModFile = "module: \"test.example/dep@v0\"\nlanguage: version: \"v0.17.0\"\n"
	octetZip   = "application/zip"
	modFileMT  = "application/vnd.cue.modulefile.v1"
	manifestMT = "application/vnd.oci.image.manifest.v1+json"
	artifactMT = "application/vnd.cue.module.v1+json"
)

// observed is what a failure's chain carries, beside its text.
type observed struct {
	contains []string // substrings the text carries
	status   int      // ociregistry.HTTPError status, 0 when none survives
	notFound bool     // modregistry.ErrNotFound in the chain
	netErr   bool     // a net.Error in the chain
	deadline bool     // context.DeadlineExceeded in the chain
	canceled bool     // context.Canceled in the chain
}

func observe(err error) observed {
	var o observed
	var he ociregistry.HTTPError
	if errors.As(err, &he) {
		o.status = he.StatusCode()
	}
	var ne net.Error
	o.notFound = errors.Is(err, modregistry.ErrNotFound)
	o.netErr = errors.As(err, &ne)
	o.deadline = errors.Is(err, context.DeadlineExceeded)
	o.canceled = errors.Is(err, context.Canceled)
	return o
}

// env returns the load environment for registry, with a fresh module cache.
func env(t *testing.T, registry string) []string {
	t.Helper()
	return cueenv.Override(registry, schematest.IsolatedCacheDir(t))
}

// servedDep serves test.example/dep at v0.0.2.
func servedDep(t *testing.T) string {
	t.Helper()
	reg, err := modregistrytest.New(fstest.MapFS{
		"test.example_dep_v0.0.2/cue.mod/module.cue": &fstest.MapFile{Data: []byte(depModFile)},
		"test.example_dep_v0.0.2/dep.cue":            &fstest.MapFile{Data: []byte("package dep\n\ny: 1\n")},
	}, "")
	require.NoError(t, err)
	t.Cleanup(reg.Close)
	return reg.Host() + "+insecure"
}

// pushRaw publishes a module artifact without the validation
// modregistrytest applies, so a test can serve a malformed module file or
// a corrupt archive.
func pushRaw(t *testing.T, r ociregistry.Interface, repo, tag string, modFile, zip []byte) {
	t.Helper()
	ctx := context.Background()
	push := func(mediaType string, b []byte) ociregistry.Descriptor {
		sum := sha256.Sum256(b)
		d := ociregistry.Descriptor{MediaType: mediaType, Size: int64(len(b)), Digest: ociregistry.Digest("sha256:" + hex.EncodeToString(sum[:]))}
		_, err := r.PushBlob(ctx, repo, d, bytes.NewReader(b))
		require.NoError(t, err)
		return d
	}
	manifest, err := json.Marshal(map[string]any{
		"schemaVersion": 2,
		"mediaType":     manifestMT,
		"config":        push(artifactMT, []byte("{}")),
		"layers":        []any{push(octetZip, zip), push(modFileMT, modFile)},
	})
	require.NoError(t, err)
	_, err = r.PushManifest(ctx, repo, tag, manifest, manifestMT)
	require.NoError(t, err)
}

func rawRegistry(t *testing.T, fill func(r ociregistry.Interface)) string {
	t.Helper()
	r := ocimem.New()
	fill(r)
	srv, err := modregistrytest.NewServer(r, nil)
	require.NoError(t, err)
	t.Cleanup(srv.Close)
	return srv.Host() + "+insecure"
}

// mainModule writes a directory module requiring test.example/dep at
// version (none when empty) whose one file imports importPath.
func mainModule(t *testing.T, version, importPath string) *opmmodule.Source {
	t.Helper()
	dir := t.TempDir()
	modFile := "module: \"spike.example/main@v0\"\nlanguage: version: \"v0.17.0\"\n"
	if version != "" {
		modFile += "deps: \"" + depPath + "\": v: \"" + version + "\"\n"
	}
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cue.mod"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cue.mod", "module.cue"), []byte(modFile), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.cue"), []byte("package main\n\nimport d \""+importPath+"\"\n\nx: d.y\n"), 0o644))
	return &opmmodule.Source{Root: dir}
}

func fetchDep(ctx context.Context, t *testing.T, registry, version string) error {
	t.Helper()
	_, _, err := loader.FetchArtifact(ctx, cuecontext.New(), depPath, version, env(t, registry), loader.ModuleSpec)
	return err
}

func loadMain(t *testing.T, registry string, src *opmmodule.Source) error {
	t.Helper()
	_, err := loader.LoadDir(cuecontext.New(), src, loader.Options{Env: env(t, registry)}, loader.ModuleSpec)
	return err
}

// cueForm is one failure as a library site receives it, and what Classify
// makes of it.
type cueForm struct {
	name  string
	run   func(t *testing.T) error
	want  observed
	class classified
}

// classified is Classify's answer for a form; the zero value means
// "returned unchanged".
type classified struct {
	ok        bool
	kind      oerrors.FetchKind
	status    int
	transient bool
}

func kindOf(kind oerrors.FetchKind, status int, transient bool) classified {
	return classified{ok: true, kind: kind, status: status, transient: transient}
}

func cueForms() []cueForm {
	statusFetch := func(status int) func(t *testing.T) error {
		return func(t *testing.T) error {
			return fetchDep(context.Background(), t, registrytest.NewStatusRegistry(t, status), depVersion)
		}
	}
	statusLoad := func(status int) func(t *testing.T) error {
		return func(t *testing.T) error {
			return loadMain(t, registrytest.NewStatusRegistry(t, status), mainModule(t, depVersion, "test.example/dep"))
		}
	}
	return []cueForm{
		// A direct fetch keeps the typed chain.
		{"fetch/absent", func(t *testing.T) error {
			return fetchDep(context.Background(), t, servedDep(t), "v0.0.9")
		}, observed{contains: []string{"module test.example/dep@v0.0.9: module not found"}, notFound: true}, kindOf(oerrors.FetchNotFound, 0, false)},
		{"fetch/unreachable", func(t *testing.T) error {
			return fetchDep(context.Background(), t, registrytest.UnreachableRegistry(t), depVersion)
		}, observed{contains: []string{"cannot do HTTP request", "connection refused"}, netErr: true}, kindOf(oerrors.FetchUnreachable, 0, true)},
		{"fetch/401", statusFetch(401), observed{contains: []string{": 401 Unauthorized: "}, status: 401}, kindOf(oerrors.FetchUnauthorized, 401, false)},
		// CUE's registry client reports a 403 on a tag lookup as not found.
		{"fetch/403", statusFetch(403), observed{contains: []string{"module not found"}, notFound: true}, kindOf(oerrors.FetchNotFound, 0, false)},
		{"fetch/404", statusFetch(404), observed{contains: []string{"module not found"}, notFound: true}, kindOf(oerrors.FetchNotFound, 0, false)},
		{"fetch/429", statusFetch(429), observed{contains: []string{": 429 Too Many Requests: "}, status: 429}, kindOf(oerrors.FetchOther, 429, false)},
		{"fetch/500", statusFetch(500), observed{contains: []string{": 500 Internal Server Error: "}, status: 500}, kindOf(oerrors.FetchOther, 500, true)},
		{"fetch/503", statusFetch(503), observed{contains: []string{": 503 Service Unavailable: "}, status: 503}, kindOf(oerrors.FetchOther, 503, true)},
		{"fetch/expired-deadline", func(t *testing.T) error {
			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			defer cancel()
			return fetchDep(ctx, t, servedDep(t), depVersion)
		}, observed{contains: []string{"cannot do HTTP request", "context deadline exceeded"}, netErr: true, deadline: true}, kindOf(oerrors.FetchUnreachable, 0, true)},
		{"fetch/cancelled", func(t *testing.T) error {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			return fetchDep(ctx, t, servedDep(t), depVersion)
		}, observed{contains: []string{"cannot do HTTP request", "context canceled"}, netErr: true, canceled: true}, classified{}},

		// cue/load flattens the cause: the %w at modpkgload's "cannot fetch"
		// does not survive into the instance error, so only text is left.
		{"load/absent", func(t *testing.T) error {
			return loadMain(t, servedDep(t), mainModule(t, "v0.0.9", "test.example/dep"))
		}, observed{contains: []string{"cannot fetch test.example/dep@v0.0.9: module test.example/dep@v0.0.9: module not found"}}, kindOf(oerrors.FetchNotFound, 0, false)},
		{"load/no-module-provides", func(t *testing.T) error {
			return loadMain(t, servedDep(t), mainModule(t, "", "test.example/other@v0"))
		}, observed{contains: []string{"cannot find module providing package test.example/other@v0"}}, kindOf(oerrors.FetchNotFound, 0, false)},
		{"load/unreachable", func(t *testing.T) error {
			return loadMain(t, registrytest.UnreachableRegistry(t), mainModule(t, depVersion, "test.example/dep"))
		}, observed{contains: []string{"cannot fetch test.example/dep@v0.0.2: ", "cannot do HTTP request", "connection refused"}}, kindOf(oerrors.FetchUnreachable, 0, true)},
		{"load/401", statusLoad(401), observed{contains: []string{": 401 Unauthorized: "}}, kindOf(oerrors.FetchUnauthorized, 401, false)},
		{"load/403", statusLoad(403), observed{contains: []string{"module not found"}}, kindOf(oerrors.FetchNotFound, 0, false)},
		{"load/404", statusLoad(404), observed{contains: []string{"module not found"}}, kindOf(oerrors.FetchNotFound, 0, false)},
		{"load/429", statusLoad(429), observed{contains: []string{": 429 Too Many Requests: "}}, kindOf(oerrors.FetchOther, 429, false)},
		{"load/500", statusLoad(500), observed{contains: []string{": 500 Internal Server Error: "}}, kindOf(oerrors.FetchOther, 500, true)},
		{"load/503", statusLoad(503), observed{contains: []string{": 503 Service Unavailable: "}}, kindOf(oerrors.FetchOther, 503, true)},
		// A published dependency whose module file does not parse: the
		// module graph cannot be expanded, and no fetch form is inside.
		{"load/malformed-dependency-module-file", func(t *testing.T) error {
			reg := rawRegistry(t, func(r ociregistry.Interface) {
				pushRaw(t, r, "test.example/dep", depVersion,
					[]byte(depModFile+"bogus: 1\ndeps: \"test.example/next@v0\": v: \"v0.0.1\"\n"), []byte("unused"))
			})
			return loadMain(t, reg, mainModule(t, depVersion, "test.example/next@v0"))
		}, observed{contains: []string{"cannot expand module graph: ", "cannot parse module file", "bogus: field not allowed"}}, classified{}},
		// A published archive that does not unzip.
		{"load/corrupt-archive", func(t *testing.T) error {
			reg := rawRegistry(t, func(r ociregistry.Interface) {
				pushRaw(t, r, "test.example/dep", depVersion, []byte(depModFile), []byte("not a zip"))
			})
			return loadMain(t, reg, mainModule(t, depVersion, "test.example/dep"))
		}, observed{contains: []string{"cannot fetch test.example/dep@v0.0.2: ", "zip: not a valid zip file"}}, kindOf(oerrors.FetchOther, 0, false)},

		// The schema loader loads a standalone package by path@version, and
		// that path keeps the typed chain.
		{"schema/unreachable", func(t *testing.T) error {
			_, err := schema.OCILoader{Module: "opmodel.dev/core@v2.0.0", Registry: registrytest.UnreachableRegistry(t), CacheDir: schematest.IsolatedCacheDir(t)}.Load(cuecontext.New())
			return err
		}, observed{contains: []string{"cannot fetch opmodel.dev/core@v2.0.0: ", "cannot do HTTP request"}, netErr: true}, kindOf(oerrors.FetchUnreachable, 0, true)},
		{"schema/503", func(t *testing.T) error {
			_, err := schema.OCILoader{Module: "opmodel.dev/core@v2.0.0", Registry: registrytest.NewStatusRegistry(t, 503), CacheDir: schematest.IsolatedCacheDir(t)}.Load(cuecontext.New())
			return err
		}, observed{contains: []string{": 503 Service Unavailable: "}, status: 503}, kindOf(oerrors.FetchOther, 503, true)},
	}
}

// TestCUEFailureForms pins each form as the embedded CUE produces it.
func TestCUEFailureForms(t *testing.T) {
	for _, f := range cueForms() {
		t.Run(f.name, func(t *testing.T) {
			err := f.run(t)
			require.Error(t, err)
			got := observe(err)
			for _, s := range f.want.contains {
				assert.Contains(t, err.Error(), s)
			}
			got.contains = f.want.contains
			assert.Equal(t, f.want, got, "typed chain of %q", err.Error())
		})
	}
}

// TestClassify_CUEFailureForms pins Classify's answer for each form the
// embedded CUE produces: its kind, its status and whether it is transient,
// or that it comes back unchanged.
func TestClassify_CUEFailureForms(t *testing.T) {
	for _, f := range cueForms() {
		t.Run(f.name, func(t *testing.T) {
			err := f.run(t)
			require.Error(t, err)
			got := oerrors.Classify(err)
			assert.Equal(t, err.Error(), got.Error(), "the message is unchanged")
			var fe *oerrors.FetchError
			if !f.class.ok {
				assert.Same(t, err, got, "returned unchanged")
				assert.False(t, errors.As(got, &fe))
				assert.NotErrorIs(t, got, oerrors.ErrTransient)
				return
			}
			require.True(t, errors.As(got, &fe), "classified: %q", err.Error())
			assert.Equal(t, f.class.kind, fe.Kind)
			assert.Equal(t, f.class.status, fe.Status)
			assert.Equal(t, f.class.transient, errors.Is(got, oerrors.ErrTransient))
		})
	}
}
