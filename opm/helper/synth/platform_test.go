package synth_test

import (
	"errors"
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/helper/synth"
	"github.com/open-platform-model/library/opm/schema"
)

// boolPtr returns a pointer to b, for SubscriptionSpec.Enable.
func boolPtr(b bool) *bool { return &b }

// emptySchemaLoader is a schema.Loader that resolves successfully but
// builds a value lacking #Platform. It drives the ErrPlatformSchemaUnavailable
// path without a registry round-trip.
type emptySchemaLoader struct{ ctx *cue.Context }

func (l emptySchemaLoader) Load(_ *cue.Context) (cue.Value, error) {
	v := l.ctx.CompileString(`#SomethingElse: {}`)
	return v, v.Err()
}

func TestPlatform_Minimal(t *testing.T) {
	ctx := sharedCtx
	plat, err := synth.Platform(ctx, synth.PlatformInput{
		Name:        "demo",
		Type:        "kubernetes",
		SchemaCache: newCache(t),
	})
	require.NoError(t, err)
	require.True(t, plat.Exists())

	kind, err := plat.LookupPath(cue.ParsePath("kind")).String()
	require.NoError(t, err)
	assert.Equal(t, "Platform", kind)

	name, err := plat.LookupPath(cue.ParsePath("metadata.name")).String()
	require.NoError(t, err)
	assert.Equal(t, "demo", name)

	typ, err := plat.LookupPath(cue.ParsePath("type")).String()
	require.NoError(t, err)
	assert.Equal(t, "kubernetes", typ)

	// No subscriptions supplied → #registry empty.
	reg := plat.LookupPath(cue.ParsePath("#registry"))
	require.True(t, reg.Exists())
	iter, err := reg.Fields()
	require.NoError(t, err)
	assert.False(t, iter.Next(), "#registry must be empty when no subscriptions supplied")
}

func TestPlatform_RejectsEmptyName(t *testing.T) {
	_, err := synth.Platform(sharedCtx, synth.PlatformInput{
		Type:        "kubernetes",
		SchemaCache: newCache(t),
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, synth.ErrPlatformMissingName),
		"want ErrPlatformMissingName, got %v", err)
}

func TestPlatform_RejectsEmptyType(t *testing.T) {
	_, err := synth.Platform(sharedCtx, synth.PlatformInput{
		Name:        "demo",
		SchemaCache: newCache(t),
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, synth.ErrMissingType),
		"want ErrMissingType, got %v", err)
}

func TestPlatform_RejectsNilSchemaCache(t *testing.T) {
	_, err := synth.Platform(sharedCtx, synth.PlatformInput{
		Name: "demo",
		Type: "kubernetes",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, synth.ErrPlatformMissingSchemaCache),
		"want ErrPlatformMissingSchemaCache, got %v", err)
}

// Guards must fire before any schema fetch: an empty Name with a nil cache
// still returns the name sentinel, proving the order in Platform.
func TestPlatform_GuardsBeforeSchemaFetch(t *testing.T) {
	_, err := synth.Platform(sharedCtx, synth.PlatformInput{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, synth.ErrPlatformMissingName),
		"name guard must precede the schema fetch, got %v", err)
}

func TestPlatform_SchemaWithoutPlatform(t *testing.T) {
	ctx := sharedCtx
	cache := &schema.Cache{Loader: emptySchemaLoader{ctx: ctx}}
	_, err := synth.Platform(ctx, synth.PlatformInput{
		Name:        "demo",
		Type:        "kubernetes",
		SchemaCache: cache,
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, synth.ErrPlatformSchemaUnavailable),
		"want ErrPlatformSchemaUnavailable, got %v", err)
}

// Core v2 deleted #SubscriptionFilter (enhancement 0010 D14) and the
// subscription-collapse slice removed FilterSpec from the Go surface — a
// filter can no longer be expressed at the Go boundary at all (the CUE-schema
// rejection the transitional test pinned stays covered by the schema itself).
// What synthesis now enforces instead: the required version scalar. An empty
// Version is refused at synthesis, naming the subscription path — early
// validation at the write side beats a silent empty materialize at the read
// side.
func TestPlatform_RejectsEmptySubscriptionVersion(t *testing.T) {
	ctx := sharedCtx
	const path = "opmodel.dev/catalogs/opm@v1"
	_, err := synth.Platform(ctx, synth.PlatformInput{
		Name:        "demo",
		Type:        "kubernetes",
		SchemaCache: newCache(t),
		Subscriptions: map[string]synth.SubscriptionSpec{
			path: {}, // no Version
		},
	})
	require.Error(t, err, "an empty subscription Version must be refused at synthesis")
	assert.True(t, errors.Is(err, synth.ErrSubscriptionMissingVersion),
		"want ErrSubscriptionMissingVersion, got %v", err)
	assert.Contains(t, err.Error(), path, "error must name the offending subscription path")
}

// The guard fires before any schema fetch — a versionless subscription with a
// nil-loader cache still returns the version sentinel.
func TestPlatform_VersionGuardBeforeSchemaFetch(t *testing.T) {
	_, err := synth.Platform(sharedCtx, synth.PlatformInput{
		Name:        "demo",
		Type:        "kubernetes",
		SchemaCache: &schema.Cache{Loader: emptySchemaLoader{ctx: sharedCtx}},
		Subscriptions: map[string]synth.SubscriptionSpec{
			"opmodel.dev/catalogs/opm@v1": {},
		},
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, synth.ErrSubscriptionMissingVersion),
		"version guard must precede the schema fetch, got %v", err)
}

func TestPlatform_EnableOmittedDefaultsTrue(t *testing.T) {
	ctx := sharedCtx
	const path = "opmodel.dev/catalogs/opm@v1"
	plat, err := synth.Platform(ctx, synth.PlatformInput{
		Name:        "demo",
		Type:        "kubernetes",
		SchemaCache: newCache(t),
		Subscriptions: map[string]synth.SubscriptionSpec{
			path: {Version: "1.0.0"}, // Enable nil → schema default *true
		},
	})
	require.NoError(t, err)

	enable, err := plat.LookupPath(cue.MakePath(
		cue.Def("registry"), cue.Str(path), cue.Str("enable"),
	)).Bool()
	require.NoError(t, err)
	assert.True(t, enable, "omitted Enable must resolve to the schema default true")

	version, err := plat.LookupPath(cue.MakePath(
		cue.Def("registry"), cue.Str(path), cue.Str("version"),
	)).String()
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", version, "the authored version scalar must render onto the subscription")
}

func TestPlatform_EnableExplicitFalse(t *testing.T) {
	ctx := sharedCtx
	const path = "opmodel.dev/catalogs/opm@v1"
	plat, err := synth.Platform(ctx, synth.PlatformInput{
		Name:        "demo",
		Type:        "kubernetes",
		SchemaCache: newCache(t),
		Subscriptions: map[string]synth.SubscriptionSpec{
			path: {Enable: boolPtr(false), Version: "1.0.0"},
		},
	})
	require.NoError(t, err)

	enable, err := plat.LookupPath(cue.MakePath(
		cue.Def("registry"), cue.Str(path), cue.Str("enable"),
	)).Bool()
	require.NoError(t, err)
	assert.False(t, enable, "explicit Enable=false must render false")
}

func TestPlatform_InvalidCatalogPath(t *testing.T) {
	_, err := synth.Platform(sharedCtx, synth.PlatformInput{
		Name:        "demo",
		Type:        "kubernetes",
		SchemaCache: newCache(t),
		Subscriptions: map[string]synth.SubscriptionSpec{
			"NOT A VALID MODULE PATH": {Enable: boolPtr(true), Version: "1.0.0"},
		},
	})
	require.Error(t, err, "a key violating #ModulePathType must surface as a unification error")
}

// Synthesis never fills the materialization slots — those are Materialize's
// job. They must be absent on the synthesized value.
func TestPlatform_MaterializationSlotsUnset(t *testing.T) {
	ctx := sharedCtx
	plat, err := synth.Platform(ctx, synth.PlatformInput{
		Name:        "demo",
		Type:        "kubernetes",
		SchemaCache: newCache(t),
		Subscriptions: map[string]synth.SubscriptionSpec{
			"opmodel.dev/catalogs/opm@v1": {Enable: boolPtr(true), Version: "1.0.0"},
		},
	})
	require.NoError(t, err)

	assert.False(t, plat.LookupPath(cue.ParsePath("#composedTransformers")).Exists(),
		"#composedTransformers must be unset on a synthesized platform")
	assert.False(t, plat.LookupPath(cue.ParsePath("#matchers")).Exists(),
		"#matchers must be unset on a synthesized platform")
}
