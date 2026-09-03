package kernel_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loaderfile "github.com/open-platform-model/library/opm/helper/loader/file"
	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/kernel"
)

// Probe group of the render-parity harness (spec render-parity: "Probe
// transformers expose definition and instance inputs"). No shipped
// transformer reads a definition field or #moduleInstance, so whether the
// kernel supplies them is invisible in the shipped group. Each probe here is
// one transformer whose output carries exactly the input in question; the
// oracle pins the value, and the kernel must agree. Both rows recorded an
// expected divergence on landing (the definition strip, the unfilled
// #moduleInstance) and both were retired by 0019's Phase A fills.
//
// Everything is served from the in-memory registry (catalog, probe module,
// and the oracle glue itself, published from testdata/parity/oracle/render.cue
// so the two groups share one oracle); core resolves from the warm workspace
// cache. Both renderers receive the same registry mapping and load the same
// on-disk instance package; Render additionally imports a #CatalogEntry-form
// platform written for the served catalog (a nested module beside the
// instance, since one module path cannot be replaced with two directories).

type parityProbe struct {
	parityCase
	// output is the transformer's #transform body: it MUST re-declare the
	// input it reads, because CUE resolves references lexically.
	transform string
	// field and want pin what the oracle must render concretely, the
	// positive half of each probe scenario.
	field, want string
}

var parityProbes = []parityProbe{
	{
		parityCase: parityCase{
			// Agrees since library-component-fill: #component is bound to
			// the evaluated component, definitions included (0019 D1/D3).
			Name:      "names-probe :: web",
			Component: "web",
			Equality:  equalityStructural,
		},
		transform: `{
			#component: _
			output: {
				kind:         "NamesProbe"
				resourceName: #component.#names.resourceName
				fqdn:         #component.#names.dns.fqdn
			}
		}`,
		field: "fqdn",
		want:  "probe-demo-web.default.svc.cluster.local",
	},
	{
		parityCase: parityCase{
			// Agrees since library-instance-fill: #moduleInstance is bound
			// to the whole evaluated instance (0019 D3, library#65).
			Name:      "instance-probe :: web",
			Component: "web",
			Equality:  equalityStructural,
		},
		transform: `{
			#moduleInstance: _
			output: {
				kind:      "InstanceProbe"
				instance:  #moduleInstance.metadata.name
				namespace: #moduleInstance.metadata.namespace
			}
		}`,
		field: "instance",
		want:  "probe-demo",
	},
}

func TestParity_Probes(t *testing.T) {
	if testing.Short() {
		t.Skip("parity probes resolve core from the workspace cache seeded from GHCR; skipping under -short")
	}
	glue, err := os.ReadFile(filepath.Join(repoLibraryRoot(t), "testdata", "parity", "oracle", "render.cue"))
	require.NoError(t, err, "reading the oracle glue")

	for _, probe := range parityProbes {
		// Subtest names feed registrytest.UniquePath, which needs a valid
		// module path, so the " :: " case name is not usable here.
		txName, _, _ := cutName(probe.Name)
		t.Run(txName, func(t *testing.T) {
			runParityProbe(t, string(glue), probe)
		})
	}
}

func runParityProbe(t *testing.T, glue string, probe parityProbe) {
	t.Helper()
	const version = "0.1.0"
	catPath := registrytest.UniquePath(t, "cat")
	oraclePath := registrytest.UniquePath(t, "oracle")
	modPath := registrytest.UniquePath(t, "modules") + "/probe_app"
	txName, _, _ := cutName(probe.Name)
	txFQN := fmt.Sprintf("%s/transformers/%s@%s", catPath, txName, version)
	containerFQN := resFQN(catPath, "container")

	mapping := registrytest.NewModuleRegistry(t,
		[]registrytest.ModuleFixture{
			{Path: modPath, Version: version, File: probeModuleFile(modPath, catPath, containerFQN, version)},
			{Path: oraclePath, Version: version, File: glue},
		},
		[]registrytest.CatalogFixture{{
			Path:    catPath,
			Version: version,
			Body:    probeCatalogBody(catPath, version, txName, txFQN, containerFQN, probe.transform),
		}},
	)

	// One on-disk module holding the instance (acquired by the kernel) and
	// the oracle entry (built by cue/load), so both import the same instance;
	// the platform Render imports is a nested module beside them.
	dir := t.TempDir()
	instDir := writeImportedInstance(t, dir, "testing.opmodel.dev/library-parity-probe@v0", modPath, version, "probe-demo", "default", "{}",
		map[string]string{catPath + "@v0": version, oraclePath + "@v0": version})
	writeFile(t, filepath.Join(dir, "run", "run.cue"), fmt.Sprintf(`package run

import (
	catalog %q
	inst "testing.opmodel.dev/library-parity-probe/instance"
	oracle %q
)

oracle.#Render & {
	#instance:     inst
	#transformers: catalog.#transformers
	#runtime:      %q
}
`, catPath+"@v0", oraclePath+"@v0", parityRuntimeName))
	platDir := writeCatalogPlatform(t, dir, catPath, version)

	ctx := context.Background()
	k := kernel.New(kernel.WithRegistry(mapping))
	opts := loaderfile.LoadOptions{Registry: mapping}

	// ── kernel side ──────────────────────────────────────────────────
	inst, err := k.AcquireInstanceFromDir(ctx, instDir, opts)
	require.NoError(t, err, "acquiring the probe instance package")
	plat, err := k.AcquirePlatformFromDir(ctx, platDir, opts)
	require.NoError(t, err, "acquiring the #CatalogEntry-form probe platform")
	res, renderErr := k.Render(ctx, kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: parityRuntimeName})
	require.NoError(t, renderErr, "Render must render the probe instance")

	// ── oracle side ──────────────────────────────────────────────────
	oracle := loadOracle(t, dir, "./run", mapping)
	pairs := oraclePairs(t, oracle)
	assertPairSetsAgree(t, res.Diagnostics.Pairs, pairs)

	pair := kernel.RenderPair{Component: probe.Component, Transformer: txFQN}
	require.Contains(t, pairs, pair, "the probe must pair with the component on the oracle side")

	// Positive half: the oracle renders the probed input concretely.
	oracleOut := oracleRender(oracle, pair)
	require.NoError(t, oracleOut.Err)
	require.Len(t, oracleOut.Objects, 1)
	got, err := oracleOut.Objects[0].LookupPath(cue.ParsePath(probe.field)).String()
	require.NoError(t, err, "oracle must render %s concretely", probe.field)
	assert.Equal(t, probe.want, got)

	// Contract half: Render agrees with the oracle on the whole value.
	c := probe.parityCase
	c.Instance = "probe instance (t.TempDir)"
	c.Transformer = txFQN
	assertParity(t, c, kernelRender(res, renderErr, pair), oracleOut)
}

// probeModuleFile is a one-component module declaring the probe catalog's
// container resource, so the probe transformer pairs with `web`.
func probeModuleFile(modPath, catPath, containerFQN, version string) string {
	return fmt.Sprintf(`package probe_app

import core "opmodel.dev/core@v2"

core.#Module
metadata: {
	name:       "probe_app"
	modulePath: %q
	version:    %q
}
#config: {}
debugValues: {}
#components: {
	web: {
		metadata: name: "web"
		#resources: %q: {
			kind: "Resource"
			metadata: {name: "container", modulePath: %q, apiVersion: %q, catalogVersion: %q, fqn: %q}
			spec: container: {image: "nginx"}
		}
	}
}
`, modPath+"@v0", version, containerFQN, catPath+"/resources", registrytest.ContractAPIVersion, version, containerFQN)
}

// probeCatalogBody mirrors registrytest.BuildCatalog's v2 member shape for
// one transformer requiring `container`, but authors #transform in full so
// the probe can re-declare and reference the input it reads.
func probeCatalogBody(catPath, version, txName, txFQN, containerFQN, transform string) string {
	return fmt.Sprintf(`metadata: {
	modulePath:  %q
	version:     %q
	description: "parity probe catalog"
}
#transformers: {
	%q: {
		kind: "ComponentTransformer"
		metadata: {
			name:        %q
			description: "parity probe"
			fqn:         %q
		}
		requiredResources: {
			%q: {
				kind: "Resource"
				metadata: {name: "container", modulePath: %q, apiVersion: %q, catalogVersion: %q, fqn: %q, labels: %q: "container"}
				matchLabels: %q: "container"
				spec: container: _
			}
		}
		#transform: %s
	}
}
`, catPath+"@v0", version,
		txFQN, txName, txFQN,
		containerFQN, catPath+"/resources", registrytest.ContractAPIVersion, version, containerFQN, registrytest.PrimitiveMatchKey,
		registrytest.PrimitiveMatchKey,
		transform)
}

func cutName(caseName string) (tx, comp string, ok bool) {
	for i := 0; i+4 <= len(caseName); i++ {
		if caseName[i:i+4] == " :: " {
			return caseName[:i], caseName[i+4:], true
		}
	}
	return caseName, "", false
}
