package synth_test

import (
	"context"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/internal/cueenv"
	"github.com/open-platform-model/library/opm/internal/loader"
	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/internal/synth"
	"github.com/open-platform-model/library/opm/module"
)

// servedModule publishes a minimal #Module to an in-process registry and
// acquires it back WITH its staged source, the way
// Kernel.AcquireModuleFromRegistry does: Instance builds the synthesized
// package inside the module's own staged tree, so a full synthesis (unlike
// the guard tests in instance_test.go) needs a source-carrying module.
// Returns the module and the environment slice the build resolves through.
func servedModule(t *testing.T) (*module.Module, []string) {
	t.Helper()
	modPath := registrytest.UniquePath(t, "modules") + "/demo"
	fixture := registrytest.ModuleFixture{
		Path: modPath, Version: "0.1.0",
		File: "package demo\n\nimport core \"opmodel.dev/core@v2\"\n\ncore.#Module\n" +
			"metadata: {\n\tname:       \"demo\"\n\tmodulePath: \"" + modPath + "@v0\"\n\tversion:    \"0.1.0\"\n}\n" +
			"#components: {}\n#config: {}\ndebugValues: {}\n",
	}
	reg := registrytest.NewModuleRegistry(t, []registrytest.ModuleFixture{fixture}, nil)
	env := cueenv.Override(reg, "")

	val, src, err := loader.FetchModule(context.Background(), cuecontext.New(), modPath+"@v0", "v0.1.0", env)
	require.NoError(t, err)
	mod, err := module.NewModuleFromValue(val)
	require.NoError(t, err)
	mod.Source = src
	require.True(t, mod.HasSource(), "the served module carries its staged source")
	return mod, env
}

// instance-synthesis spec, "Label and annotation order is stable": the staged
// instance.cue is byte-identical for identical inputs whatever order the
// caller filled its maps in, and lists every entry in ascending key order.
// One map is filled ascending and the other descending so that, were the
// writer to range the map, Go's iteration order could not satisfy both the
// identity and the ordering assertion.
func TestInstance_LabelsAndAnnotationsSorted(t *testing.T) {
	mod, env := servedModule(t)

	type entry struct{ key, value string }
	labels := []entry{{"app", "demo"}, {"env", "prod"}, {"tier", "web"}, {"zone", "eu"}}
	annotations := []entry{
		{"opmodel.dev/alpha", "a"}, {"opmodel.dev/note", "n"},
		{"opmodel.dev/owner", "team-x"}, {"opmodel.dev/ticket", "t"},
	}
	fill := func(entries []entry, descending bool) map[string]string {
		m := make(map[string]string, len(entries))
		for i := range entries {
			e := entries[i]
			if descending {
				e = entries[len(entries)-1-i]
			}
			m[e.key] = e.value
		}
		return m
	}

	stage := func(descending bool) string {
		t.Helper()
		_, src, err := synth.Instance(cuecontext.New(), coreVersion, synth.Input{
			Module:      mod,
			Name:        "myrel",
			Namespace:   "ns",
			Labels:      fill(labels, descending),
			Annotations: fill(annotations, descending),
			Env:         env,
		})
		require.NoError(t, err)
		require.NotNil(t, src)
		data, ok := src.Overlay[filepath.Join(src.Root, src.Pkg, "instance.cue")]
		require.True(t, ok, "the synthesized instance.cue is in the staged overlay")
		return string(data)
	}

	ascendingFill := stage(false)
	descendingFill := stage(true)
	assert.Equal(t, ascendingFill, descendingFill, "identical inputs must stage identical bytes")

	assert.Contains(t, ascendingFill,
		"\tlabels: {\n\t\t\"app\": \"demo\"\n\t\t\"env\": \"prod\"\n\t\t\"tier\": \"web\"\n\t\t\"zone\": \"eu\"\n\t}\n",
		"labels are listed in ascending key order")
	assert.Contains(t, ascendingFill,
		"\tannotations: {\n\t\t\"opmodel.dev/alpha\": \"a\"\n\t\t\"opmodel.dev/note\": \"n\"\n\t\t\"opmodel.dev/owner\": \"team-x\"\n\t\t\"opmodel.dev/ticket\": \"t\"\n\t}\n",
		"annotations are listed in ascending key order")
}
