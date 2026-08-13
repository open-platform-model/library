package schema_test

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/schema"
)

// fakeInstance is a minimal schema.InstanceView for context-building tests.
type fakeInstance struct{}

func (fakeInstance) InstanceName() string           { return "demo" }
func (fakeInstance) Namespace() string              { return "ns" }
func (fakeInstance) InstanceUUID() string           { return "u-inst" }
func (fakeInstance) InstanceFQN() string            { return "reg.example/modules:demo:ns" }
func (fakeInstance) ModuleFQN() string              { return "example.com/m/demo:1.0.0" }
func (fakeInstance) ModuleVersion() string          { return "1.0.0" }
func (fakeInstance) Labels() map[string]string      { return nil }
func (fakeInstance) Annotations() map[string]string { return nil }

// TestBuildTransformerContext_CarriesDescriptiveLabelsNotMatchLabels pins the
// D36 site split: the transformer render context's #componentMetadata reads
// the component's descriptive metadata.labels — NOT matchLabels, which D36
// explicitly keeps out of #TransformerContext. Render reads (e.g. the
// hpa_transformer's workload-type lookup) depend on metadata.labels surviving
// here, so this site must never flip with the matcher.
func TestBuildTransformerContext_CarriesDescriptiveLabelsNotMatchLabels(t *testing.T) {
	cueCtx := cuecontext.New()
	schemaComp := cueCtx.CompileString(`
metadata: {
	name: "web"
	labels: { "core.opmodel.dev/workload-type": "stateless" }
}
matchLabels: { "match.example/only": "yes" }
`)
	require.NoError(t, schemaComp.Err())

	ctxVal, warnings, err := schema.BuildTransformerContext(cueCtx, fakeInstance{}, "web", schemaComp, "rt")
	require.NoError(t, err)
	assert.Empty(t, warnings)

	labels := ctxVal.LookupPath(cue.ParsePath(`#componentMetadata.labels`))
	require.True(t, labels.Exists(), "component metadata.labels must reach the context")
	var got map[string]string
	require.NoError(t, labels.Decode(&got))
	assert.Equal(t, map[string]string{"core.opmodel.dev/workload-type": "stateless"}, got,
		"context labels are the descriptive metadata.labels, verbatim")
	assert.NotContains(t, got, "match.example/only",
		"matchLabels keys must not leak into the render context")

	matchLabels := ctxVal.LookupPath(cue.ParsePath(`#componentMetadata.matchLabels`))
	assert.False(t, matchLabels.Exists(),
		"D36: matchLabels does not reach #TransformerContext")
}
