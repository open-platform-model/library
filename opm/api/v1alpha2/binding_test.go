package v1alpha2_test

import (
	"reflect"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/api"
	"github.com/open-platform-model/library/opm/api/v1alpha2"
	"github.com/open-platform-model/library/opm/apiversion"
)

func lookup(t *testing.T) api.Binding {
	t.Helper()
	b, err := api.Lookup(apiversion.V1alpha2)
	require.NoError(t, err)
	require.NotNil(t, b)
	return b
}

func TestPathsCoverExpectedFields(t *testing.T) {
	b := lookup(t)
	p := b.Paths()

	cases := []struct {
		name string
		got  cue.Path
		want string
	}{
		{"APIVersion", p.APIVersion, "apiVersion"},
		{"Metadata", p.Metadata, "metadata"},
		{"Components", p.Components, "components"},
		{"Values", p.Values, "values"},
		{"DebugValues", p.DebugValues, "debugValues"},
		{"Transformers", p.Transformers, "#transformers"},
		{"Transform", p.Transform, "#transform"},
		{"Component", p.Component, "#component"},
		{"Output", p.Output, "output"},
		{"TransformerRequiredLabels", p.TransformerRequiredLabels, "requiredLabels"},
		{"TransformerRequiredResources", p.TransformerRequiredResources, "requiredResources"},
		{"TransformerRequiredTraits", p.TransformerRequiredTraits, "requiredTraits"},
		{"TransformerOptionalTraits", p.TransformerOptionalTraits, "optionalTraits"},
		{"MetadataLabels", p.MetadataLabels, "metadata.labels"},
		{"MetadataAnnotations", p.MetadataAnnotations, "metadata.annotations"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, c.got.String(), "path %s", c.name)
		})
	}
}

func TestVersion(t *testing.T) {
	b := lookup(t)
	assert.Equal(t, apiversion.V1alpha2, b.Version())
}

func TestDecodeReleaseMetadataRoundTrip(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString(`
metadata: {
	name: "demo"
	namespace: "ns"
	uuid: "11111111-1111-1111-1111-111111111111"
	labels: app: "x"
	annotations: note: "y"
}`)
	require.NoError(t, v.Err())

	b := lookup(t)
	got, err := b.DecodeReleaseMetadata(v)
	require.NoError(t, err)
	assert.Equal(t, "demo", got.Name)
	assert.Equal(t, "ns", got.Namespace)
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", got.UUID)
	assert.Equal(t, map[string]string{"app": "x"}, got.Labels)
	assert.Equal(t, map[string]string{"note": "y"}, got.Annotations)
}

func TestDecodeProviderMetadataFallbackName(t *testing.T) {
	ctx := cuecontext.New()
	// Provider with no metadata — fallback name kicks in.
	v := ctx.CompileString(`other: 1`)
	require.NoError(t, v.Err())

	b := lookup(t)
	got, err := b.DecodeProviderMetadata(v, "kubernetes")
	require.NoError(t, err)
	assert.Equal(t, "kubernetes", got.Name)
}

// fakeRelease is a minimal api.ReleaseView for context tests.
type fakeRelease struct {
	name, ns, uuid, fqn, ver string
	labels, anns             map[string]string
}

func (f fakeRelease) ReleaseName() string            { return f.name }
func (f fakeRelease) Namespace() string              { return f.ns }
func (f fakeRelease) ReleaseUUID() string            { return f.uuid }
func (f fakeRelease) ModuleFQN() string              { return f.fqn }
func (f fakeRelease) ModuleVersion() string          { return f.ver }
func (f fakeRelease) Labels() map[string]string      { return f.labels }
func (f fakeRelease) Annotations() map[string]string { return f.anns }

func TestBuildTransformerContext(t *testing.T) {
	ctx := cuecontext.New()
	schemaComp := ctx.CompileString(`
metadata: {
	name: "web"
	labels: { app: "web" }
	annotations: { owner: "team-a" }
}`)
	require.NoError(t, schemaComp.Err())

	rel := fakeRelease{
		name: "demo", ns: "ns", uuid: "u", fqn: "f", ver: "1.0.0",
		labels: map[string]string{"k": "v"},
		anns:   map[string]string{"a": "b"},
	}

	b := lookup(t)
	got, warnings, err := b.BuildTransformerContext(ctx, rel, "web", schemaComp, "opm-cli")
	require.NoError(t, err)
	assert.Empty(t, warnings)

	mrm := got.LookupPath(cue.ParsePath("#moduleReleaseMetadata"))
	require.True(t, mrm.Exists(), "context.#moduleReleaseMetadata missing")
	name, err := mrm.LookupPath(cue.ParsePath("name")).String()
	require.NoError(t, err)
	assert.Equal(t, "demo", name)
	fqn, err := mrm.LookupPath(cue.ParsePath("fqn")).String()
	require.NoError(t, err)
	assert.Equal(t, "f", fqn)

	cm := got.LookupPath(cue.ParsePath("#componentMetadata"))
	require.True(t, cm.Exists(), "context.#componentMetadata missing")
	compName, err := cm.LookupPath(cue.ParsePath("name")).String()
	require.NoError(t, err)
	assert.Equal(t, "web", compName)
	cmLabels := cm.LookupPath(cue.ParsePath("labels.app"))
	got1, err := cmLabels.String()
	require.NoError(t, err)
	assert.Equal(t, "web", got1)

	rt := got.LookupPath(cue.ParsePath("#runtimeName"))
	require.True(t, rt.Exists(), "context.#runtimeName missing")
	rtName, err := rt.String()
	require.NoError(t, err)
	assert.Equal(t, "opm-cli", rtName)
}

func TestBuildTransformerContextRejectsEmptyRuntime(t *testing.T) {
	ctx := cuecontext.New()
	schemaComp := ctx.CompileString(`metadata: name: "web"`)
	require.NoError(t, schemaComp.Err())

	rel := fakeRelease{name: "demo", ns: "ns"}

	b := lookup(t)
	_, _, err := b.BuildTransformerContext(ctx, rel, "web", schemaComp, "")
	require.Error(t, err)
}

// Importing _ "v1alpha2" via the package's own import already triggered init();
// this asserts the binding actually landed in the registry.
func TestBindingRegistered(t *testing.T) {
	b, err := api.Lookup(apiversion.V1alpha2)
	require.NoError(t, err)
	require.NotNil(t, b)
	assert.IsType(t, v1alpha2.ModuleReleaseContextData{}, v1alpha2.ModuleReleaseContextData{})
}

// TestBindingHasNoDebugArtifactDecoder enforces that the binding never grows
// a top-level debug artifact decoder (e.g. DecodeModuleDebugMetadata). Debug
// values are a Module field accessed via Paths().DebugValues, not a kernel
// artifact. See enhancement 001-kernel-redesign-around-platform D6 and the
// retire-module-debug change.
func TestBindingHasNoDebugArtifactDecoder(t *testing.T) {
	b := lookup(t)
	rt := reflect.TypeOf(b)
	for i := 0; i < rt.NumMethod(); i++ {
		name := rt.Method(i).Name
		if strings.HasPrefix(name, "Decode") && strings.Contains(strings.ToLower(name), "debug") {
			t.Fatalf("binding exposes forbidden debug-artifact decoder %q; debug overlays are a frontend concern, accessed via Paths().DebugValues", name)
		}
	}
}

// TestDebugValuesPathIsModuleInternal asserts DebugValues resolves to the
// Module-internal "debugValues" field, not a definition (which would imply a
// separate artifact). The path must be readable as Module.Package.LookupPath.
func TestDebugValuesPathIsModuleInternal(t *testing.T) {
	p := lookup(t).Paths()
	assert.Equal(t, "debugValues", p.DebugValues.String(),
		"DebugValues must be a regular field path within Module.Package, not a definition")
}
