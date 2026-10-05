package kernel_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/kernel"
)

// unreadRequiredBody is a module whose #config declares required values no
// component reads (image, tag, any) next to an optional and a defaulted one;
// its only component reads replicas.
const unreadRequiredBody = "#config: {\n" +
	"\treplicas: int | *1\n" +
	"\timage:    string\n" +
	"\ttag!:     string\n" +
	"\topt?:     string\n" +
	"\tany:      _\n" +
	"}\n" +
	"debugValues: {}\n" +
	"#components: foo: {metadata: name: \"foo\", _r: #config.replicas & int}\n"

// library#211: an instance that leaves a required #config value unset is
// accepted when no component reads that value, while ValidateConfigDetailed
// refuses the same values.
func TestKernel_SynthesizeInstance_UnreadRequiredConfigUnset(t *testing.T) {
	k, mod := publishSynthModule(t, "demo", "0.1.0", unreadRequiredBody)
	src := mustSource(t, k, "/values/a.cue", `replicas: 2`)

	_, vErr := k.ValidateConfigDetailed(mod.ConfigSchema(), []kernel.Source{src})
	require.Error(t, vErr)
	for _, field := range []string{"image", "tag", "any"} {
		assert.True(t, hasErrorPath(vErr, "#config."+field), "ValidateConfigDetailed does not name %s: %v", field, vErr)
	}

	inst, err := k.SynthesizeInstance(context.Background(), kernel.InstanceInput{
		Module: mod, Name: "myrel", Namespace: "default", Values: []kernel.Source{src},
	})
	require.NoError(t, err)
	require.NotNil(t, inst)
}

// library#211, the directory twin: the package's own values leave image, tag
// and any unset, with and without a trailing source, and the instance is
// accepted.
func TestKernel_AcquireInstanceFromDir_UnreadRequiredConfigUnset(t *testing.T) {
	k, _, modPath := publishSynthModuleAt(t, "demo", "0.1.0", unreadRequiredBody)
	dir := writeImportedInstance(t, t.TempDir(), "authored.opmodel.dev/instance@v0", modPath, "0.1.0",
		"myrel", "default", "{replicas: 2}", nil)

	for name, sources := range map[string][]kernel.Source{
		"no sources":          nil,
		"an unrelated source": {mustSource(t, k, "/values/opt.cue", `opt: "x"`)},
	} {
		t.Run(name, func(t *testing.T) {
			inst, err := k.AcquireInstanceFromDir(context.Background(), dir, sources...)
			require.NoError(t, err)
			require.NotNil(t, inst)
		})
	}
}
