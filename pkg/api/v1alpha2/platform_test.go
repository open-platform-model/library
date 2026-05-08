package v1alpha2_test

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const platformFixture = `
apiVersion: "opmodel.dev/v1alpha2"
kind: "Platform"
metadata: {
	name: "demo-platform"
	description: "binding-path test fixture"
	labels: env: "dev"
	annotations: owner: "team"
}
type: "kubernetes"
#registry: {}
#knownResources: {}
#knownTraits: {}
#composedTransformers: {}
#matchers: {
	resources: {}
	traits: {}
}
`

func TestPlatformPathsResolve(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString(platformFixture)
	require.NoError(t, v.Err())

	b := lookup(t)
	p := b.Paths()

	cases := []struct {
		name string
		path cue.Path
		want string
	}{
		{"Registry", p.Registry, "#registry"},
		{"KnownResources", p.KnownResources, "#knownResources"},
		{"KnownTraits", p.KnownTraits, "#knownTraits"},
		{"ComposedTransformers", p.ComposedTransformers, "#composedTransformers"},
		{"Matchers", p.Matchers, "#matchers"},
		{"MatchersResources", p.MatchersResources, "#matchers.resources"},
		{"MatchersTraits", p.MatchersTraits, "#matchers.traits"},
	}
	for _, c := range cases {
		t.Run(c.name+"/Literal", func(t *testing.T) {
			assert.Equal(t, c.want, c.path.String())
		})
		t.Run(c.name+"/Resolves", func(t *testing.T) {
			got := v.LookupPath(c.path)
			require.True(t, got.Exists(), "path %s did not resolve on fixture", c.name)
		})
	}

	// Matchers has typed sub-paths exercised by the matcher in slice 09.
	matchers := v.LookupPath(p.Matchers)
	require.True(t, matchers.LookupPath(cue.ParsePath("resources")).Exists())
	require.True(t, matchers.LookupPath(cue.ParsePath("traits")).Exists())
}

func TestDecodePlatformMetadata(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString(platformFixture)
	require.NoError(t, v.Err())

	b := lookup(t)
	got, err := b.DecodePlatformMetadata(v)
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, "demo-platform", got.Name)
	assert.Equal(t, "kubernetes", got.Type, "type field hoisted from #Platform.type into PlatformMetadata.Type")
	assert.Equal(t, "binding-path test fixture", got.Description)
	assert.Equal(t, map[string]string{"env": "dev"}, got.Labels)
	assert.Equal(t, map[string]string{"owner": "team"}, got.Annotations)
}

func TestDecodePlatformMetadata_RequiresMetadata(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString(`
apiVersion: "opmodel.dev/v1alpha2"
kind: "Platform"
type: "kubernetes"
`)
	require.NoError(t, v.Err())

	b := lookup(t)
	got, err := b.DecodePlatformMetadata(v)
	require.Error(t, err)
	assert.Nil(t, got)
}
