package kernel_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/core"
	loaderfile "github.com/open-platform-model/library/opm/helper/loader/file"
	"github.com/open-platform-model/library/opm/helper/synth"
	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/kernel"
)

// TestFlow_ImportedModule_SynthToCompile is the end-to-end imported-module
// render coverage (task 4.4 / spec ADDED requirement "Imported-module render
// coverage exists"): a real published module — referenced by IMPORT, with a
// hyphenated name so the nameSnakeCase address derivation is exercised — is
// synthesized into a #ModuleRelease and run through Match + Compile to concrete
// resources. An authored release.cue importing the SAME module is compiled too,
// and both MUST yield the same Compiled set (single-build parity through
// Kernel.Compile, not merely at the release-value level).
//
// Hermetic: the module + catalog are served from an in-memory registry;
// opmodel.dev/core@v0.6.0 resolves from the warm workspace cache, matching the
// other registrytest-backed integration tests in this package.
func TestFlow_ImportedModule_SynthToCompile(t *testing.T) {
	const version = "0.1.0"
	catPath := registrytest.UniquePath(t, "cat")
	metaPath := registrytest.UniquePath(t, "modules")
	const modName = "web-app" // hyphenated → published at the snake_case leaf
	const snake = "web_app"
	modPath := metaPath + "/" + snake
	containerFQN := fmt.Sprintf("%s/resources/container@%s", catPath, version)

	// Catalog: one transformer requiring the `container` resource, emitting a
	// single Deployment. Pinned to core@v0.6.0 so the transformer's required
	// #Resource and the module's inline #Resource share one core closedness.
	cat := standardCatalog(catPath, version)
	cat.CoreVersion = "v0.6.0"

	// Module: a single component `web` declaring an inline #Resource keyed by the
	// catalog's container FQN (the matcher pairs on that key). No catalog import
	// is needed — the FQN is a plain key, exactly as a release authors it.
	modBody := fmt.Sprintf(`#config: {}
debugValues: {}
#components: {
	web: {
		metadata: name: "web"
		#resources: %q: {
			kind: "Resource"
			metadata: {name: "container", modulePath: %q, version: %q}
			spec: container: {image: "nginx"}
		}
	}
}
`, containerFQN, catPath+"/resources", version)

	var modFile strings.Builder
	fmt.Fprintf(&modFile, "package %s\n\n", snake)
	modFile.WriteString("import core \"opmodel.dev/core@v0\"\n\n")
	modFile.WriteString("core.#Module\n")
	fmt.Fprintf(&modFile, "metadata: {\n\tname:       %q\n\tmodulePath: %q\n\tversion:    %q\n}\n", modName, metaPath, version)
	modFile.WriteString(modBody)

	registryMapping := registrytest.NewModuleRegistry(t,
		[]registrytest.ModuleFixture{{Path: modPath, Version: version, File: modFile.String(), CoreVersion: "v0.6.0"}},
		[]registrytest.CatalogFixture{cat},
	)

	k := kernel.New(kernel.WithRegistry(registryMapping))
	ctx := context.Background()

	modVal, err := k.LoadModuleFromRegistry(ctx, modPath+"@v0", "v"+version)
	require.NoErrorf(t, err, "loading published module %s", modPath)
	mod, err := k.NewModuleFromValue(modVal)
	require.NoError(t, err)
	require.Equal(t, "web-app", mod.Metadata.Name)

	mp, err := materializePlatform(t, k, catPath)
	require.NoError(t, err, "materializing platform subscribed to the catalog")

	// ── synth path ───────────────────────────────────────────────────────
	rel, err := k.SynthesizeRelease(ctx, synth.ReleaseInput{
		Module:      mod,
		Name:        "web-rel",
		Namespace:   "default",
		Values:      k.CueContext().CompileString("{}"), // #config is empty; supply concrete (empty) values
		SchemaCache: k.SchemaCache(),
	})
	require.NoError(t, err, "synthesizing release from the imported module")

	plan, err := k.Match(ctx, kernel.MatchInput{ModuleRelease: rel, Platform: mp})
	require.NoError(t, err)
	pairs := matchPairsToMap(plan.MatchedPairs())
	require.Contains(t, pairs, "web", "web component must pair with a transformer")
	assertContainsFQNSub(t, pairs["web"], "transformers/deployment@", "web must match the deployment transformer")

	out, err := k.Compile(ctx, kernel.CompileInput{ModuleRelease: rel, Platform: mp, RuntimeName: "rt"})
	require.NoError(t, err)
	require.NotEmpty(t, out.Compiled, "synth release must compile to at least one resource")
	synthKinds := compiledKinds(t, out.Compiled)
	assert.Contains(t, synthKinds, "Deployment", "the container resource must render a Deployment")

	// ── authored path (single-build parity through Kernel.Compile) ─────────
	dir := t.TempDir()
	importPath := modPath + "@v0"
	releaseSrc := fmt.Sprintf(`package release

import (
	core "opmodel.dev/core@v0"
	opmModule %q
)

core.#ModuleRelease

metadata: {
	name:      "web-rel"
	namespace: "default"
}

#module: opmModule
values: {}
`, importPath)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "release.cue"), []byte(releaseSrc), 0o644))

	moduleSrc := fmt.Sprintf(`module: "authored.opmodel.dev/release@v0"
language: version: "v0.17.0"
deps: {
	"opmodel.dev/core@v0": v: "v0.6.0"
	%q: v: %q
}
`, importPath, "v"+version)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cue.mod"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cue.mod", "module.cue"), []byte(moduleSrc), 0o644))

	authoredVal, err := k.LoadReleasePackage(ctx, dir, loaderfile.LoadOptions{Registry: registryMapping})
	require.NoError(t, err, "authored release.cue importing the published module must load")
	authoredRel, err := k.ProcessModuleRelease(ctx, authoredVal, *mod, cue.Value{})
	require.NoError(t, err, "processing the authored release")

	authoredOut, err := k.Compile(ctx, kernel.CompileInput{ModuleRelease: authoredRel, Platform: mp, RuntimeName: "rt"})
	require.NoError(t, err)
	assert.Equal(t, synthKinds, compiledKinds(t, authoredOut.Compiled),
		"synth and authored imported-module releases must compile to the same resources")
}

// compiledKinds returns the sorted `kind` strings of every compiled resource,
// a stable fingerprint of a Compile result's output for parity comparison.
func compiledKinds(t *testing.T, compiled []*core.Compiled) []string {
	t.Helper()
	kinds := make([]string, 0, len(compiled))
	for _, c := range compiled {
		k, err := c.Value.LookupPath(cue.ParsePath("kind")).String()
		require.NoError(t, err, "compiled resource must carry a concrete kind")
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	return kinds
}
