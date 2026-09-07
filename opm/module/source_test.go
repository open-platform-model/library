package module_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/module"
)

const modFile = "module: \"x.example/m@v0\"\nlanguage: version: \"v0.17.0\"\n"

// overlaySource builds an overlay-mode source under root with an instance
// package in a subdirectory, the shape instance synthesis produces.
func overlaySource(root string) *module.Source {
	pkg := "opm-synth-instance"
	return &module.Source{
		Root: root,
		Pkg:  pkg,
		Overlay: map[string][]byte{
			filepath.Join(root, "cue.mod", "module.cue"):   []byte(modFile),
			filepath.Join(root, "module.cue"):              []byte("package web_app\n\nx: 1\n"),
			filepath.Join(root, pkg, "instance.cue"):       []byte("package instance\n\ny: 2\n"),
			filepath.Join(root, pkg, "values.cue"):         []byte("package instance\n\nz: 3\n"),
			filepath.Join(root, pkg, "notes.md"):           []byte("not cue"),
			filepath.Join(root, pkg, "nested", "deep.cue"): []byte("package other\n"),
		},
	}
}

// artifact-types spec, "An overlay source can be written to a directory":
// every entry lands at its Root-relative path with identical bytes, the
// returned list names them relative to dir in sorted order, and the whole
// overlay is validated before anything is written.
func TestSourceWriteTo_RoundTrip(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "opm-registry-module", "web_app")
	src := overlaySource(root)
	dir := t.TempDir()

	written, err := src.WriteTo(dir)
	require.NoError(t, err)

	want := make([]string, 0, len(src.Overlay))
	for key, entry := range src.Overlay {
		rel, err := filepath.Rel(root, key)
		require.NoError(t, err)
		want = append(want, rel)

		got, err := os.ReadFile(filepath.Join(dir, rel))
		require.NoError(t, err, rel)
		assert.Equal(t, entry, got, "%s round-trips byte for byte", rel)
	}
	sort.Strings(want)
	assert.Equal(t, want, written, "the returned paths are dir-relative and sorted")
	assert.True(t, sort.StringsAreSorted(written))
}

// artifact-types spec, "Entry outside the root refused" and "On-disk source
// refused": validation runs over the whole overlay first, so a refusal leaves
// the target directory untouched.
func TestSourceWriteTo_Refusals(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "opm-registry-module", "web_app")

	t.Run("entry outside the root", func(t *testing.T) {
		escaped := overlaySource(root)
		escaped.Overlay[filepath.Join(string(filepath.Separator), "elsewhere", "x.cue")] = []byte("package x\n")
		dir := t.TempDir()

		written, err := escaped.WriteTo(dir)
		require.ErrorContains(t, err, "outside the source root")
		assert.Nil(t, written)

		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		assert.Empty(t, entries, "nothing is written, including the entries that were valid")
	})

	t.Run("on-disk source", func(t *testing.T) {
		written, err := (&module.Source{Root: root}).WriteTo(t.TempDir())
		require.Error(t, err)
		assert.Nil(t, written)
	})

	t.Run("nil receiver", func(t *testing.T) {
		var src *module.Source
		written, err := src.WriteTo(t.TempDir())
		require.Error(t, err)
		assert.Nil(t, written)
	})
}

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
			mod:  &module.Module{Source: &module.Source{Overlay: map[string][]byte{"/x/a.cue": []byte("")}}},
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
				Overlay: map[string][]byte{"/opm-registry-module/x/cue.mod/module.cue": []byte("module: \"x@v0\"\n")},
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
