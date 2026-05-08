package values_test

import (
	"errors"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/open-platform-model/library/pkg/errors"
	"github.com/open-platform-model/library/pkg/helper/values"
	"github.com/open-platform-model/library/pkg/kernel"
)

func makeKernel(t *testing.T) *kernel.Kernel {
	t.Helper()
	return kernel.New()
}

func compile(t *testing.T, k *kernel.Kernel, src string) cue.Value {
	t.Helper()
	v := k.CueContext().CompileString(src)
	require.NoError(t, v.Err())
	return v
}

func TestValidateAndUnify_EmptyStack(t *testing.T) {
	k := makeKernel(t)
	schema := compile(t, k, `{ replicas: int }`)

	got, err := values.ValidateAndUnify(k, schema, nil)
	require.Nil(t, err, "empty stack MUST return a nil MultiSourceError")
	assert.False(t, got.Exists(), "empty stack MUST return cue.Value{}")
}

func TestValidateAndUnify_SingleValidLayer(t *testing.T) {
	k := makeKernel(t)
	schema := compile(t, k, `{ replicas: int & >0 }`)
	v := compile(t, k, `{ replicas: 3 }`)

	got, err := values.ValidateAndUnify(k, schema, values.Stack{
		{Name: "user", Source: "values.cue", Value: v},
	})
	require.Nil(t, err)
	require.True(t, got.Exists())

	r, lookupErr := got.LookupPath(cue.ParsePath("replicas")).Int64()
	require.NoError(t, lookupErr)
	assert.Equal(t, int64(3), r)
}

func TestValidateAndUnify_MultipleValidLayers(t *testing.T) {
	k := makeKernel(t)
	schema := compile(t, k, `{ replicas: int & >0, image: string, env: string }`)

	// CUE unification is intersection: layers must use *defaults or leave
	// fields open for later layers to set them. This mirrors the typical
	// OPM defaults-cue pattern.
	a := compile(t, k, `{ replicas: int | *1, image: string | *"nginx", env: string | *"dev" }`)
	b := compile(t, k, `{ replicas: 3 }`)
	c := compile(t, k, `{ env: "prod" }`)

	got, err := values.ValidateAndUnify(k, schema, values.Stack{
		{Name: "defaults", Source: "embedded", Value: a},
		{Name: "user", Source: "values.cue", Value: b},
		{Name: "overlay", Source: "overlay.cue", Value: c},
	})
	require.Nil(t, err)
	require.True(t, got.Exists())

	r, lookupErr := got.LookupPath(cue.ParsePath("replicas")).Int64()
	require.NoError(t, lookupErr)
	assert.Equal(t, int64(3), r)

	img, lookupErr := got.LookupPath(cue.ParsePath("image")).String()
	require.NoError(t, lookupErr)
	assert.Equal(t, "nginx", img)

	env, lookupErr := got.LookupPath(cue.ParsePath("env")).String()
	require.NoError(t, lookupErr)
	assert.Equal(t, "prod", env)
}

func TestValidateAndUnify_OneLayerSchemaViolation(t *testing.T) {
	k := makeKernel(t)
	schema := compile(t, k, `{ replicas: int & >0 }`)

	good := compile(t, k, `{ replicas: 3 }`)
	bad := compile(t, k, `{ replicas: -1 }`)

	got, err := values.ValidateAndUnify(k, schema, values.Stack{
		{Name: "defaults", Source: "embedded", Value: good},
		{Name: "user", Source: "values.cue", Value: bad},
	})
	require.NotNil(t, err)
	assert.False(t, got.Exists(), "any failing layer MUST yield zero cue.Value")

	entries := err.Errors()
	require.Len(t, entries, 1)
	assert.Equal(t, "user", entries[0].LayerName)
	assert.Equal(t, "values.cue", entries[0].Source)
	require.NotNil(t, entries[0].Err)
}

func TestValidateAndUnify_MultipleLayerViolations(t *testing.T) {
	k := makeKernel(t)
	schema := compile(t, k, `{ replicas: int & >0, image: string }`)

	v1 := compile(t, k, `{ replicas: -1, image: "nginx" }`)
	v2 := compile(t, k, `{ replicas: 0, image: 42 }`)

	got, err := values.ValidateAndUnify(k, schema, values.Stack{
		{Name: "first", Source: "a.cue", Value: v1},
		{Name: "second", Source: "b.cue", Value: v2},
	})
	require.NotNil(t, err)
	assert.False(t, got.Exists())

	entries := err.Errors()
	require.Len(t, entries, 2, "every failing layer contributes one entry")
	assert.Equal(t, "first", entries[0].LayerName)
	assert.Equal(t, "second", entries[1].LayerName)

	// Error() summary mentions both layers.
	msg := err.Error()
	assert.Contains(t, msg, `layer "first"`)
	assert.Contains(t, msg, `layer "second"`)

	// Unwrap exposes per-layer ConfigError for stdlib errors.As walks.
	unwrapped := err.Unwrap()
	require.Len(t, unwrapped, 2)
	var cfgErr *oerrors.ConfigError
	require.True(t, errors.As(unwrapped[0], &cfgErr))
	require.NotNil(t, cfgErr)
}

func TestValidateAndUnify_LayerOverrideSemantics(t *testing.T) {
	k := makeKernel(t)
	schema := compile(t, k, `{ name: string, env: string }`)

	// In CUE, "later overrides earlier" composes via *defaults in the
	// earlier layer. The later layer's concrete value narrows the
	// disjunction.
	defaults := compile(t, k, `{ name: string | *"first", env: string | *"dev" }`)
	user := compile(t, k, `{ name: "second" }`)

	got, err := values.ValidateAndUnify(k, schema, values.Stack{
		{Name: "defaults", Source: "embedded", Value: defaults},
		{Name: "user", Source: "values.cue", Value: user},
	})
	require.Nil(t, err)
	require.True(t, got.Exists())

	name, lookupErr := got.LookupPath(cue.ParsePath("name")).String()
	require.NoError(t, lookupErr)
	assert.Equal(t, "second", name, "later concrete value MUST win over earlier *default")

	env, lookupErr := got.LookupPath(cue.ParsePath("env")).String()
	require.NoError(t, lookupErr)
	assert.Equal(t, "dev", env, "earlier *default MUST survive when no later layer overrides it")
}

// TestValidateAndUnify_RoundTripIntoKernelTier2 confirms a Tier-1 success
// produces a value the kernel's Tier-2 ValidateConfig accepts. This is the
// integration path frontends will use: helper Tier-1, then kernel Tier-2.
func TestValidateAndUnify_RoundTripIntoKernelTier2(t *testing.T) {
	k := makeKernel(t)
	schema := compile(t, k, `{ replicas: int & >0, image: string, env: string }`)

	// Defaults set every required field via *N defaults; later layers
	// supply concrete overrides where the user wants to deviate. The
	// merged result is concrete on every schema field, so Tier-2's
	// concrete-required check passes.
	defaults := compile(t, k, `{ replicas: int | *1, image: string | *"nginx", env: string | *"dev" }`)
	user := compile(t, k, `{ replicas: 3 }`)
	overlay := compile(t, k, `{ env: "prod" }`)

	merged, err := values.ValidateAndUnify(k, schema, values.Stack{
		{Name: "defaults", Source: "embedded", Value: defaults},
		{Name: "user", Source: "values.cue", Value: user},
		{Name: "overlay", Source: "overlay.cue", Value: overlay},
	})
	require.Nil(t, err)
	require.True(t, merged.Exists())

	got, cfgErr := k.ValidateConfig(schema, merged, "module", "demo")
	require.Nil(t, cfgErr, "Tier-2 MUST pass when every Tier-1 layer passed")
	require.True(t, got.Exists())
}

func TestKernel_ValidateAndUnify_DelegatesToHelper(t *testing.T) {
	k := makeKernel(t)
	schema := compile(t, k, `{ replicas: int & >0 }`)
	v := compile(t, k, `{ replicas: 5 }`)

	stack := values.Stack{{Name: "user", Source: "values.cue", Value: v}}

	mergedHelper, errHelper := values.ValidateAndUnify(k, schema, stack)
	mergedMethod, errMethod := k.ValidateAndUnify(schema, stack)

	require.Nil(t, errHelper)
	require.Nil(t, errMethod)
	require.True(t, mergedHelper.Exists())
	require.True(t, mergedMethod.Exists())

	helperReplicas, _ := mergedHelper.LookupPath(cue.ParsePath("replicas")).Int64()
	methodReplicas, _ := mergedMethod.LookupPath(cue.ParsePath("replicas")).Int64()
	assert.Equal(t, helperReplicas, methodReplicas)
}

func TestMultiSourceError_NilSafety(t *testing.T) {
	var e *values.MultiSourceError
	assert.Equal(t, "<nil>", e.Error())
	assert.Nil(t, e.Errors())
	assert.Nil(t, e.Unwrap())
}

// TestMultiSourceError_ErrorMentionsConfigErrorBlock confirms the aggregate
// embeds the per-layer ConfigError summary so users see file:line positions.
func TestMultiSourceError_ErrorMentionsConfigErrorBlock(t *testing.T) {
	k := makeKernel(t)
	schema := compile(t, k, `{ replicas: int & >0 }`)
	bad := compile(t, k, `{ replicas: -1 }`)

	_, err := values.ValidateAndUnify(k, schema, values.Stack{
		{Name: "user", Source: "values.cue", Value: bad},
	})
	require.NotNil(t, err)

	msg := err.Error()
	assert.Contains(t, msg, `layer "user"`)
	assert.True(t, strings.Contains(msg, "values do not satisfy #config") ||
		strings.Contains(msg, "replicas"),
		"summary should embed the underlying ConfigError block; got: %s", msg)
}
