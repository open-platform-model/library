package module_test

import (
	"testing"

	"cuelang.org/go/cue/load"
	"github.com/stretchr/testify/assert"

	"github.com/open-platform-model/library/opm/module"
)

func TestModule_HasSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		mod  *module.Module
		want bool
	}{
		{
			name: "nil module",
			mod:  nil,
			want: false,
		},
		{
			name: "nil Source",
			mod:  &module.Module{},
			want: false,
		},
		{
			name: "empty root",
			mod:  &module.Module{Source: &module.Source{Overlay: map[string]load.Source{"/x/a.cue": load.FromString("")}}},
			want: false,
		},
		{
			name: "empty overlay",
			mod:  &module.Module{Source: &module.Source{Root: "/x"}},
			want: false,
		},
		{
			name: "populated source",
			mod: &module.Module{Source: &module.Source{
				Root:    "/opm-registry-module/x",
				Overlay: map[string]load.Source{"/opm-registry-module/x/cue.mod/module.cue": load.FromString("module: \"x@v0\"\n")},
			}},
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, tc.mod.HasSource())
		})
	}
}
