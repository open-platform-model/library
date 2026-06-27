package materialize

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/schema"
)

// TestTransformers_RenderConcreteWhereClosedPlatformDoesNot is the regression
// guard for the output-local hidden field bug
// (docs/design/transformer-output-hidden-field-scope-bug.md, ADR-003).
//
// Root cause: FillPath-ing the composed transformer map into a closed,
// independently-built c.#Platform value corrupts the lazy in-expression
// resolution of output-local hidden fields in the transformers. Federation
// (this change) never builds that closed twin: MaterializedPlatform.Transformers
// is the native composed map produced by indexCatalogs in the owner context,
// and is the only surface the executor reads.
//
// This test pins the invariant with the real buggy catalog (v0.5.2, whose
// deployment-transformer declares `_convertedSidecars` INSIDE output):
//   - a #transform read from the native composed map (what mp.Transformers
//     exposes) renders a concrete, marshallable Deployment (the federated
//     surface);
//   - the SAME #transform read out of a closed platform value does NOT
//     (the bug — proving the federation is load-bearing, not decorative).
//
// If someone reintroduces a closed-fill seam, or reads transforms off a closed
// platform value again, the first assertion breaks.
func TestTransformers_RenderConcreteWhereClosedPlatformDoesNot(t *testing.T) {
	const reg = "opmodel.dev=localhost:5000+insecure,testing.opmodel.dev=localhost:5000+insecure,registry.cue.works"
	if v := os.Getenv("CUE_REGISTRY"); v != "" {
		// honor an externally configured registry (CI), else use the local default
	} else {
		t.Setenv("CUE_REGISTRY", reg)
	}
	env := resolverEnv(os.Getenv("CUE_REGISTRY"))

	octx := cuecontext.New()

	// Buggy catalog: v0.5.2 declares _convertedSidecars inside output.
	cv, err := pullCatalog(octx, env, "opmodel.dev/catalogs/opm", "v0.5.2")
	if err != nil {
		t.Skipf("catalog pull failed (registry unreachable?): %v", err)
	}
	const fqn = "opmodel.dev/catalogs/opm/transformers/deployment-transformer@0.5.2"

	composed, _, err := indexCatalogs(octx, []catalogBuild{{
		Subscription: "opmodel.dev/catalogs/opm", Version: "0.5.2", Value: cv,
	}})
	require.NoError(t, err)

	// Load the real, closed c.#Platform fixture and fill the composed map onto
	// it — reproducing the rejected closed-fill seam locally to prove it still
	// corrupts (Materialize itself no longer does this).
	platDir := platformFixtureDir(t)
	insts := load.Instances([]string{"."}, &load.Config{Dir: platDir, Env: env})
	require.NotEmpty(t, insts)
	require.NoError(t, insts[0].Err)
	loadedPlatform := octx.BuildInstance(insts[0])
	require.NoError(t, loadedPlatform.Err())
	require.True(t, loadedPlatform.IsClosed(), "platform fixture must be a closed c.#Platform for this guard to be meaningful")
	pkg := loadedPlatform.FillPath(schema.ComposedTransformers, composed)

	// A concrete component + context, as the executor fills them.
	component := octx.CompileString(`{
		kind: "Component"
		metadata: { name: "web", resourceName: "web", labels: {
			"component.opmodel.dev/name": "web", "core.opmodel.dev/workload-type": "stateless" } }
		spec: {
			initContainers: []
			restartPolicy: "Always"
			scaling: count: 1
			sidecarContainers: []
			container: { name: "web", image: {
				repository: "nginx", tag: "1.27", digest: "", pullPolicy: "IfNotPresent", reference: "nginx:1.27" } }
		}
	}`)
	// NOTE: this fixture pins the real pre-rename catalog v0.5.2, whose
	// deployment-transformer@0.5.2 reads #moduleReleaseMetadata (the old
	// vocabulary). It is intentionally NOT renamed to #moduleInstanceMetadata —
	// the context must match the pinned transformer's contract, not core@v1's.
	ctxv := octx.CompileString(`{
		#moduleReleaseMetadata: { name: "web-app", namespace: "default", uuid: "11111111-2222-5333-8444-555555555555" }
		#componentMetadata: { name: "web" }
		#runtimeName: "opm-test"
	}`)

	render := func(transform cue.Value) error {
		out := transform.
			FillPath(schema.Component, component).
			FillPath(schema.Context, ctxv).
			LookupPath(schema.Output)
		_, err := out.MarshalJSON()
		return err
	}

	txFromComposed := composed.
		LookupPath(cue.MakePath(cue.Str(fqn))).
		LookupPath(schema.Transform)
	require.True(t, txFromComposed.Exists())

	txFromPackage := pkg.
		LookupPath(schema.ComposedTransformers).
		LookupPath(cue.MakePath(cue.Str(fqn))).
		LookupPath(schema.Transform)
	require.True(t, txFromPackage.Exists())

	// The federated surface: the native composed map (what mp.Transformers
	// exposes) renders concrete.
	require.NoError(t, render(txFromComposed),
		"transform read from the native composed map must render a concrete, marshallable resource")

	// The bug: a closed platform value corrupts it. This asserts the federation
	// is load-bearing; if CUE fixes the underlying Go-API bug, this will start
	// passing and the test should be revisited (federation is then belt-and-braces).
	require.Error(t, render(txFromPackage),
		"sanity: reading the transform from a closed platform value is still corrupt (the reason we federate, never fill)")
}

func platformFixtureDir(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	// opm/materialize/<this file> → repo root is three dirs up.
	root := filepath.Dir(filepath.Dir(filepath.Dir(file)))
	return filepath.Join(root, "modules", "opm_platform")
}
