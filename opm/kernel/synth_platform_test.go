package kernel_test

import (
	"context"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loader "github.com/open-platform-model/library/opm/helper/loader/file"
	"github.com/open-platform-model/library/opm/helper/synth"
	"github.com/open-platform-model/library/opm/kernel"
)

// TestKernel_SynthesizePlatform_DecodesMetadata asserts the happy path:
// SynthesizePlatform returns a *platform.Platform whose decoded metadata
// matches the inputs and whose Package is the synthesized value. The core
// schema is resolved from the warm workspace cache (no catalog pull), so the
// test does not gate on GHCR — mirroring the SynthesizeInstance kernel tests.
func TestKernel_SynthesizePlatform_DecodesMetadata(t *testing.T) {
	k := newSynthKernel(t)

	plat, err := k.SynthesizePlatform(context.Background(), synth.PlatformInput{
		Name:        "demo",
		Type:        "kubernetes",
		SchemaCache: k.SchemaCache(),
	})
	require.NoError(t, err)
	require.NotNil(t, plat)
	require.NotNil(t, plat.Metadata)

	assert.Equal(t, "demo", plat.Metadata.Name)
	assert.Equal(t, "kubernetes", plat.Metadata.Type)
	require.True(t, plat.Package.Exists(), "Package must carry the synthesized value")
}

// TestKernel_SynthesizePlatform_DefaultSchemaCache asserts that omitting
// SchemaCache on PlatformInput falls back to the kernel-owned cache.
func TestKernel_SynthesizePlatform_DefaultSchemaCache(t *testing.T) {
	k := newSynthKernel(t)

	plat, err := k.SynthesizePlatform(context.Background(), synth.PlatformInput{
		Name: "demo",
		Type: "kubernetes",
		// SchemaCache intentionally omitted; SynthesizePlatform fills it from
		// k.SchemaCache().
	})
	require.NoError(t, err)
	require.NotNil(t, plat)
	assert.Equal(t, "demo", plat.Metadata.Name)
}

// TestKernel_SynthesizePlatform_NoRegistryIO asserts that synthesis stops
// before Materialize: the returned Package carries #registry as authored with
// the kernel-filled materialization slots unset. No catalog round-trip occurs.
func TestKernel_SynthesizePlatform_NoRegistryIO(t *testing.T) {
	k := newSynthKernel(t)
	const path = "opmodel.dev/catalogs/opm"

	plat, err := k.SynthesizePlatform(context.Background(), synth.PlatformInput{
		Name:        "demo",
		Type:        "kubernetes",
		SchemaCache: k.SchemaCache(),
		Subscriptions: map[string]synth.SubscriptionSpec{
			path: {},
		},
	})
	require.NoError(t, err)

	// #registry as authored.
	enable, err := plat.Package.LookupPath(cue.MakePath(
		cue.Def("registry"), cue.Str(path), cue.Str("enable"),
	)).Bool()
	require.NoError(t, err)
	assert.True(t, enable, "subscription enable must resolve to the schema default true")

	// Materialization slots unset — Materialize was never called.
	assert.False(t, plat.Package.LookupPath(cue.ParsePath("#composedTransformers")).Exists(),
		"#composedTransformers must be unset before Materialize")
	assert.False(t, plat.Package.LookupPath(cue.ParsePath("#matchers")).Exists(),
		"#matchers must be unset before Materialize")
}

// TestFlow_SynthesizedPlatform_MaterializesLikeFileLoaded asserts that the
// *platform.Platform produced by SynthesizePlatform feeds Kernel.Materialize
// exactly as a file-loaded platform of the same content does. It synthesizes a
// platform subscribing to the published opm catalog and materializes both the
// synthesized and the on-disk fixture (modules/opm_platform), asserting they
// resolve to the same catalog version and the same composed-transformer set.
//
// Skips under -short or when GHCR is unreachable, matching the gating in the
// file-driven flow tests.
func TestFlow_SynthesizedPlatform_MaterializesLikeFileLoaded(t *testing.T) {
	if testing.Short() {
		t.Skip("flow integration test pulls the catalog + core schema from GHCR; skipping under -short")
	}
	skipUnlessRegistry(t)

	registry := flowRegistry()
	t.Setenv("CUE_REGISTRY", registry)

	k := kernel.New()
	ctx := context.Background()
	const path = "opmodel.dev/catalogs/opm"

	// ── Synthesize the platform from typed inputs ────────────────────
	synthPlat, err := k.SynthesizePlatform(ctx, synth.PlatformInput{
		Name:        "k8s-default",
		Description: "Default Kubernetes Platform — subscribes to the opm core catalog",
		Type:        "kubernetes",
		Subscriptions: map[string]synth.SubscriptionSpec{
			path: {},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "kubernetes", synthPlat.Metadata.Type)

	synthMP, err := k.Materialize(ctx, synthPlat)
	require.NoError(t, err, "materializing the synthesized platform against the published catalog")
	require.NotNil(t, synthMP)

	// ── Load + materialize the on-disk fixture for comparison ────────
	libraryRoot := repoLibraryRoot(t)
	platformDir := filepath.Join(libraryRoot, "modules", "opm_platform")
	fileVal, err := k.LoadPlatformPackage(ctx, platformDir, loader.LoadOptions{Registry: registry})
	require.NoErrorf(t, err, "loading platform fixture from %s", platformDir)
	filePlat, err := k.NewPlatformFromValue(fileVal)
	require.NoError(t, err)
	fileMP, err := k.Materialize(ctx, filePlat)
	require.NoError(t, err, "materializing the file-loaded platform")

	// Same subscription resolves to the same catalog version.
	assert.Equal(t, fileMP.Resolved[path], synthMP.Resolved[path],
		"synthesized and file-loaded platforms must resolve the same catalog version for %q", path)

	// The composed-transformer set must be identical between the two paths.
	synthTransformers := transformerKeys(t, synthMP.Transformers)
	fileTransformers := transformerKeys(t, fileMP.Transformers)
	assert.Equal(t, fileTransformers, synthTransformers,
		"synthesized platform must compose the same transformers as the file-loaded one")
	assert.NotEmpty(t, synthTransformers, "materialized platform must compose at least one transformer")
}

// transformerKeys returns the sorted set of top-level FQN keys of a
// materialized platform's native Transformers composed map.
func transformerKeys(t *testing.T, composed cue.Value) []string {
	t.Helper()
	require.True(t, composed.Exists(), "Transformers must be populated after Materialize")
	iter, err := composed.Fields()
	require.NoError(t, err)
	var keys []string
	for iter.Next() {
		keys = append(keys, iter.Selector().String())
	}
	return keys
}
