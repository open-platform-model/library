package kernel_test

import (
	"context"
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/helper/synth"
	"github.com/open-platform-model/library/opm/internal/schematest"
	"github.com/open-platform-model/library/opm/kernel"
)

// TestIntegration_Validate exercises the Tier-2 config-validation surface end to
// end. These cases are fully hermetic and registry-free: ValidateConfig only
// unifies the #config schema with values — no core schema or catalog needed.
func TestIntegration_Validate(t *testing.T) {
	k := kernel.New()

	t.Run("module values ok", func(t *testing.T) {
		mod := buildModule(t, k, `{ replicas: int | *1, image: string }`)
		out, err := k.ValidateModuleValues(mod, cueVal(t, k, `{ replicas: 3, image: "nginx" }`, "values.cue"))
		require.NoError(t, err)
		assert.True(t, out.Exists())
	})

	t.Run("type error rejected", func(t *testing.T) {
		mod := buildModule(t, k, `{ replicas: int, image: string }`)
		_, err := k.ValidateModuleValues(mod, cueVal(t, k, `{ replicas: "three", image: "n" }`, "values.cue"))
		require.Error(t, err)
	})

	t.Run("partial tolerates missing required field", func(t *testing.T) {
		mod := buildModule(t, k, `{ replicas: int, image: string }`)
		values := cueVal(t, k, `{ replicas: 2 }`, "values.cue") // image omitted
		_, errFull := k.ValidateModuleValues(mod, values)
		require.Error(t, errFull, "full validation requires every field concrete")
		_, errPartial := k.ValidateModuleValuesPartial(mod, values)
		require.NoError(t, errPartial, "partial validation tolerates missing fields")
	})

	t.Run("disallowed field under closed config", func(t *testing.T) {
		mod := buildModule(t, k, `close({ replicas: int | *1 })`)
		_, err := k.ValidateModuleValues(mod, cueVal(t, k, `{ replicas: 1, extra: true }`, "values.cue"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "field not allowed")
	})

	t.Run("no values is a noop", func(t *testing.T) {
		mod := buildModule(t, k, `{ replicas: int | *1 }`)
		out, err := k.ValidateModuleValues(mod, cue.Value{})
		require.NoError(t, err)
		assert.False(t, out.Exists(), "zero values → zero result, no error")
	})

	t.Run("detailed layered sources", func(t *testing.T) {
		mod := buildModule(t, k, `{ replicas: int, image: string }`)
		base := kernel.Source{Value: cueVal(t, k, `{ replicas: 2 }`, "base.cue"), Name: "base", Origin: "base.cue"}
		over := kernel.Source{Value: cueVal(t, k, `{ image: "nginx" }`, "override.cue"), Name: "override", Origin: "override.cue"}

		merged, err := k.ValidateConfigDetailed(mod.ConfigSchema(), []kernel.Source{base, over})
		require.NoError(t, err, "layered sources together satisfy the schema")
		require.True(t, merged.Exists())

		_, errPartial := k.ValidateConfigDetailed(mod.ConfigSchema(), []kernel.Source{base}, kernel.Partial())
		require.NoError(t, errPartial, "partial mode tolerates the incomplete single source")

		_, errFull := k.ValidateConfigDetailed(mod.ConfigSchema(), []kernel.Source{base})
		require.Error(t, errFull, "full mode rejects the incomplete single source")
	})
}

// TestIntegration_SynthesizeRelease covers the synth release-construction path:
// a module plus typed inputs is unified against the core #ModuleRelease,
// producing a release whose identity fields are stamped by the schema. Needs
// the core schema (warm workspace cache via schematest.SetEnv); no catalog.
func TestIntegration_SynthesizeRelease(t *testing.T) {
	schematest.SetEnv(t)
	k := kernel.New()

	mod := buildModule(t, k, `{ replicas: int | *1, image: string }`)
	values := cueVal(t, k, `{ image: "nginx" }`, "values.cue")

	rel, err := k.SynthesizeRelease(context.Background(), synth.ReleaseInput{
		Module:      mod,
		Name:        "web",
		Namespace:   "default",
		Values:      values,
		SchemaCache: k.SchemaCache(),
	})
	require.NoError(t, err)
	require.NotNil(t, rel)
	assert.Equal(t, "web", rel.Metadata.Name)
	assert.Equal(t, "default", rel.Metadata.Namespace)
	assert.NotEmpty(t, rel.Metadata.UUID, "release UUID is stamped by the schema (SHA1 over identity)")
}
