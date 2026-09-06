package kernel_test

import (
	"context"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loaderfile "github.com/open-platform-model/library/opm/helper/loader/file"
	"github.com/open-platform-model/library/opm/helper/synth"
	"github.com/open-platform-model/library/opm/module"
)

// These tests pin what the single build derives, refuses and keeps for an
// instance synthesized from a published core-v2 module. They carry the
// scenarios the core-v1 synth integration tests covered before the v0/v1
// module-address convention was retired, on the same served fixtures the
// kernel's other synth tests use.

const synthTwoComponentBody = `#components: {
	foo: {metadata: name: "foo"}
	bar: {metadata: name: "bar"}
}
#config: {}
debugValues: {}
`

// expectedInstanceUUID computes the schema's instance UUID through CUE in the
// kernel's own context: a UUID v5 of the instance fqn
// ("<module registryPath>:<name>:<namespace>", core v2 0010 D41). Failing the
// assertion built on it is the drift sentinel for module_instance.cue.
func expectedInstanceUUID(t *testing.T, ctx *cue.Context, fqn string) string {
	t.Helper()
	v := ctx.CompileString(`
import cue_uuid "uuid"
OPMNamespace: "11bc6112-a6e8-4021-bec9-b3ad246f9466"
out: cue_uuid.SHA1(OPMNamespace, ` + literalString(fqn) + `)
`)
	require.NoError(t, v.Err())
	s, err := v.LookupPath(cue.ParsePath("out")).String()
	require.NoError(t, err)
	return s
}

// literalString renders s as a CUE string literal for the probes above; the
// values in play are module paths and DNS names, so escaping is defensive.
func literalString(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

// A #NameType-violating instance name fails at schema unification inside the
// build (a real admission error), not through a module-resolution accident.
func TestKernel_SynthesizeInstance_BadNameFailsUnification(t *testing.T) {
	k, mod := publishSynthModule(t, "demo", "0.1.0", "#components: {}\n#config: {}\ndebugValues: {}\n")

	inst, err := k.SynthesizeInstance(context.Background(), synth.InstanceInput{
		Module:      mod,
		Name:        "BAD-UPPER", // #NameType forbids uppercase
		Namespace:   "default",
		Values:      k.CueContext().CompileString("{}"),
		SchemaCache: k.SchemaCache(),
	})
	require.Error(t, err, "a name violating #NameType must surface as a unification error")
	assert.Nil(t, inst)
	assert.NotContains(t, err.Error(), "cannot find package",
		"failure must be name unification, not module resolution")
	assert.Contains(t, err.Error(), "BAD-UPPER", "the refusal names the offending value")
}

// With no caller Values, the schema's values path stays unfilled and is NEVER
// backfilled from Module.debugValues: the helper-tier build leaves it
// non-concrete, and the kernel refuses the instance rather than shipping the
// debug value.
func TestKernel_SynthesizeInstance_EmptyValuesNotBackfilledFromDebugValues(t *testing.T) {
	// #config requires a sentinel with no default; debugValues supplies one.
	k, mod := publishSynthModule(t, "demo", "0.1.0",
		"#components: {}\n#config: {sentinel: string}\ndebugValues: {sentinel: \"from-debug\"}\n")
	ctx := context.Background()

	// Helper tier: the synthesized value carries no debug value under values.
	tree, _, err := synth.Instance(k.CueContext(), synth.InstanceInput{
		Module:      mod,
		Name:        "myrel",
		Namespace:   "default",
		SchemaCache: k.SchemaCache(),
		// Values omitted; MUST NOT fall back to debugValues.
	})
	require.NoError(t, err)
	values := tree.LookupPath(cue.ParsePath("values"))
	if values.Exists() {
		assert.Error(t, values.Validate(cue.Concrete(true)),
			"values must be non-concrete when no Values were supplied, not backfilled from debugValues")
		sentinel, _ := values.LookupPath(cue.ParsePath("sentinel")).String()
		assert.NotEqual(t, "from-debug", sentinel)
	}

	// Kernel tier: the same synthesis is refused by the concreteness check
	// and the debug value never appears.
	inst, err := k.SynthesizeInstance(ctx, synth.InstanceInput{
		Module:      mod,
		Name:        "myrel",
		Namespace:   "default",
		SchemaCache: k.SchemaCache(),
	})
	require.Error(t, err, "an unfilled required config field must refuse the instance")
	assert.Nil(t, inst)
	assert.NotContains(t, err.Error(), "from-debug", "debugValues never reach the instance")
	assert.NotContains(t, err.Error(), "cannot find package")
}

// Every field the schema derives is produced by CUE unification of the
// synthesized package, not by Go-side fill: metadata.uuid is the canonical
// UUID v5 of the instance fqn (stable, namespace-divergent), components is
// fanned from the module's #components, the standard
// module-instance.opmodel.dev/{name,uuid} labels coexist with caller labels,
// and annotations pass through.
func TestKernel_SynthesizeInstance_DerivedFieldsFromSchema(t *testing.T) {
	k, mod := publishSynthModule(t, "demo", "0.1.0", synthTwoComponentBody)
	ctx := context.Background()

	synthesize := func(name, namespace string, labels, annotations map[string]string) *module.Instance {
		t.Helper()
		inst, err := k.SynthesizeInstance(ctx, synth.InstanceInput{
			Module:      mod,
			Name:        name,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotations,
			Values:      k.CueContext().CompileString("{}"),
			SchemaCache: k.SchemaCache(),
		})
		require.NoError(t, err)
		require.NotNil(t, inst)
		return inst
	}

	inst := synthesize("myrel", "default",
		map[string]string{"env": "prod"},
		map[string]string{"opmodel.dev/owner": "team-x"})

	// metadata.uuid is the canonical UUID v5 of the instance fqn.
	registryPath, err := mod.Package.LookupPath(cue.ParsePath("metadata.registryPath")).String()
	require.NoError(t, err, "a core-v2 module exposes metadata.registryPath")
	assert.Equal(t, expectedInstanceUUID(t, k.CueContext(), registryPath+":myrel:default"), inst.Metadata.UUID,
		"schema-derived UUID must equal uuid.SHA1(OPMNamespace, <registryPath>:<name>:<namespace>)")

	// components fanned from #components.
	components := inst.Package.LookupPath(cue.ParsePath("components"))
	require.True(t, components.Exists())
	assert.True(t, components.LookupPath(cue.ParsePath("foo")).Exists(), "components.foo fanned from #components.foo")
	assert.True(t, components.LookupPath(cue.ParsePath("bar")).Exists(), "components.bar fanned from #components.bar")

	// Caller labels coexist with the schema-stamped identity labels.
	labels := map[string]string{}
	require.NoError(t, inst.Package.LookupPath(cue.ParsePath("metadata.labels")).Decode(&labels))
	assert.Equal(t, "prod", labels["env"], "caller-supplied label must be present")
	assert.Equal(t, "myrel", labels["module-instance.opmodel.dev/name"], "schema-stamped name label must coexist")
	assert.Equal(t, inst.Metadata.UUID, labels["module-instance.opmodel.dev/uuid"], "schema-stamped uuid label carries the derived uuid")

	// Annotations pass through unchanged.
	annotations := map[string]string{}
	require.NoError(t, inst.Package.LookupPath(cue.ParsePath("metadata.annotations")).Decode(&annotations))
	assert.Equal(t, "team-x", annotations["opmodel.dev/owner"], "caller-supplied annotation must survive")

	// The UUID diverges across namespaces and is deterministic for identical
	// inputs.
	other := synthesize("myrel", "ns-b", nil, nil)
	again := synthesize("myrel", "default", nil, nil)
	assert.NotEqual(t, inst.Metadata.UUID, other.Metadata.UUID, "different namespaces must produce different UUIDs")
	assert.Equal(t, inst.Metadata.UUID, again.Metadata.UUID, "identical inputs must produce identical UUIDs")
}

// A synthesized instance and an authored instance package importing the
// SAME published module construct the same instance value: the same
// schema-derived uuid, the same fanned components and the same identity
// labels, with the imported #module's identity concrete on both paths. This
// is the convergence synthesis exists to guarantee at the value level; the
// render-level parity lives in TestFlow_ImportedModule_SynthToRender.
func TestKernel_SynthesizeInstance_ParityWithAuthoredPackage(t *testing.T) {
	const version = "0.1.0"
	k, mod := publishSynthModule(t, "demo", version, synthTwoComponentBody)
	ctx := context.Background()

	synthInst, err := k.SynthesizeInstance(ctx, synth.InstanceInput{
		Module:      mod,
		Name:        "myrel",
		Namespace:   "default",
		Values:      k.CueContext().CompileString("{}"),
		SchemaCache: k.SchemaCache(),
	})
	require.NoError(t, err)

	// Authored path: an on-disk instance package importing the published
	// module, resolved through the same registry mapping the kernel carries.
	modPath := strings.TrimSuffix(mod.Metadata.ModulePath, "@"+majorOf(version))
	instDir := writeImportedInstance(t, t.TempDir(), "authored.opmodel.dev/instance@v0", modPath, version, "myrel", "default", "{}", nil)
	authored, err := k.AcquireInstanceFromDir(ctx, instDir, loaderfile.LoadOptions{})
	require.NoError(t, err, "authored instance.cue importing the published module must acquire")

	// The imported #module's identity is concrete end-to-end on the authored
	// path (the path that rotted invisibly before the core self-cycle fix).
	for _, p := range []string{"#module.metadata.modulePath", "#module.metadata.version", "#module.metadata.fqn"} {
		s, err := authored.Package.LookupPath(cue.ParsePath(p)).String()
		require.NoErrorf(t, err, "authored %s must be concrete", p)
		require.NotEmpty(t, s, "authored %s must be non-empty", p)
	}

	assert.Equal(t, authored.Metadata.UUID, synthInst.Metadata.UUID, "synth and authored paths must derive the same instance UUID")
	for _, comp := range []string{"foo", "bar"} {
		assert.True(t, synthInst.Package.LookupPath(cue.ParsePath("components."+comp)).Exists(), "synth components."+comp)
		assert.True(t, authored.Package.LookupPath(cue.ParsePath("components."+comp)).Exists(), "authored components."+comp)
	}
	synthLabels, authoredLabels := map[string]string{}, map[string]string{}
	require.NoError(t, synthInst.Package.LookupPath(cue.ParsePath("metadata.labels")).Decode(&synthLabels))
	require.NoError(t, authored.Package.LookupPath(cue.ParsePath("metadata.labels")).Decode(&authoredLabels))
	assert.Equal(t, authoredLabels, synthLabels, "both paths stamp the same identity labels")
}
