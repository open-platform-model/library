package kernel_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/helper/synth"
)

// TestIntegration_SynthesizeInstance covers the synth instance-construction path:
// a module plus typed inputs is unified against the core #ModuleInstance,
// producing an instance whose identity fields are stamped by the schema. Needs
// the core schema (warm workspace cache); no catalog.
func TestIntegration_SynthesizeInstance(t *testing.T) {
	// synth.Instance imports the module by its canonical registry path, so the
	// module must be published (a locally-built value no longer resolves).
	k, mod := publishSynthModule(t, "demo", "0.1.0",
		"#components: {}\n#config: {replicas: int | *1, image: string}\ndebugValues: {}\n")
	values := cueVal(t, k, `{ image: "nginx" }`, "values.cue")

	inst, err := k.SynthesizeInstance(context.Background(), synth.InstanceInput{
		Module:      mod,
		Name:        "web",
		Namespace:   "default",
		Values:      values,
		SchemaCache: k.SchemaCache(),
	})
	require.NoError(t, err)
	require.NotNil(t, inst)
	assert.Equal(t, "web", inst.Metadata.Name)
	assert.Equal(t, "default", inst.Metadata.Namespace)
	assert.NotEmpty(t, inst.Metadata.UUID, "instance UUID is stamped by the schema (SHA1 over identity)")
}
