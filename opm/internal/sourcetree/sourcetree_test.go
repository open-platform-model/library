package sourcetree

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/module"
)

const modFile = "module: \"x.example/m@v0\"\nlanguage: version: \"v0.17.0\"\n"

// overlaySource builds an overlay-mode source under root with an instance
// package in a subdirectory, the shape synth.Instance produces.
func overlaySource(root string) *module.Source {
	pkg := "opm-synth-instance"
	return &module.Source{
		Root: root,
		Pkg:  pkg,
		Overlay: map[string][]byte{
			filepath.Join(root, "cue.mod", "module.cue"):   []byte(modFile),
			filepath.Join(root, "module.cue"):              []byte("package web_app\n\nx: 1\n"),
			filepath.Join(root, pkg, "instance.cue"):       []byte([]byte("package instance\n\ny: 2\n")),
			filepath.Join(root, pkg, "values.cue"):         []byte("package instance\n\nz: 3\n"),
			filepath.Join(root, pkg, "notes.md"):           []byte("not cue"),
			filepath.Join(root, pkg, "nested", "deep.cue"): []byte("package other\n"),
		},
	}
}

// diskSource writes files (slash-relative path → contents) under a temp
// module root and returns it as an on-disk source with the given Pkg.
func diskSource(t *testing.T, pkg string, files map[string]string) *module.Source {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	}
	return &module.Source{Root: root, Pkg: pkg}
}

func sortedKeys(overlay map[string][]byte) []string {
	keys := make([]string, 0, len(overlay))
	for k := range overlay {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func TestPackageName_Overlay(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "opm-registry-module", "web_app")

	// One clause shared by the package's files; the non-CUE file and the
	// nested directory are not part of the package.
	name, err := PackageName(overlaySource(root))
	require.NoError(t, err)
	assert.Equal(t, "instance", name)

	// The root package.
	name, err = PackageName(&module.Source{Root: root, Overlay: overlaySource(root).Overlay})
	require.NoError(t, err)
	assert.Equal(t, "web_app", name)

	// Two clauses in one directory.
	mixed := overlaySource(root)
	mixed.Overlay[filepath.Join(root, "opm-synth-instance", "other.cue")] = []byte("package other\n")
	_, err = PackageName(mixed)
	require.ErrorContains(t, err, "more than one package: [instance other]")

	// No clause at all: a clause-less file is skipped, so the directory
	// declares nothing.
	bare := &module.Source{Root: root, Pkg: "data", Overlay: map[string][]byte{
		filepath.Join(root, "data", "data.cue"): []byte("a: 1\n"),
	}}
	_, err = PackageName(bare)
	require.ErrorContains(t, err, "no package clause")

	_, err = PackageName(nil)
	require.Error(t, err)
	_, err = PackageName(&module.Source{})
	require.Error(t, err)
}

func TestPackageName_Disk(t *testing.T) {
	one := diskSource(t, "inst", map[string]string{
		"cue.mod/module.cue": modFile,
		"inst/instance.cue":  "package instance\n\ny: 2\n",
		"inst/values.cue":    "package instance\n\nz: 3\n",
		"inst/notes.md":      "package notmarkdown\n",
		"inst/nested/d.cue":  "package other\n",
	})
	name, err := PackageName(one)
	require.NoError(t, err)
	assert.Equal(t, "instance", name)

	two := diskSource(t, "", map[string]string{
		"a.cue": "package a\n",
		"b.cue": "package b\n",
	})
	_, err = PackageName(two)
	require.ErrorContains(t, err, "more than one package: [a b]")

	zero := diskSource(t, "", map[string]string{"data.cue": "a: 1\n"})
	_, err = PackageName(zero)
	require.ErrorContains(t, err, "no package clause")

	missing := &module.Source{Root: filepath.Join(t.TempDir(), "absent")}
	_, err = PackageName(missing)
	require.Error(t, err)
}

func TestOverlayFromDir_CueFilesOnly(t *testing.T) {
	src := diskSource(t, "", map[string]string{
		"cue.mod/module.cue":             modFile,
		"module.cue":                     "package m\n",
		"README.md":                      "docs",
		"cue.mod/pkg/x.example/y/y.cue":  "package y\n",
		"cue.mod/pkg/x.example/y/y.json": "{}",
		".git/config":                    "[core]",
		"sub/notes.txt":                  "text",
	})
	overlay, err := OverlayFromDir(src.Root)
	require.NoError(t, err)
	want := []string{
		filepath.Join(src.Root, "cue.mod", "module.cue"),
		filepath.Join(src.Root, "cue.mod", "pkg", "x.example", "y", "y.cue"),
		filepath.Join(src.Root, "module.cue"),
	}
	assert.Equal(t, want, sortedKeys(overlay), "only .cue files, nested cue.mod/pkg included")

	assert.Equal(t, "package m\n", string(overlay[filepath.Join(src.Root, "module.cue")]),
		"entries are the file's bytes")

	_, err = OverlayFromDir(filepath.Join(src.Root, "absent"))
	require.Error(t, err)
}

func TestOverlayFromFS_RekeysUnderSyntheticRoot(t *testing.T) {
	fsys := fstest.MapFS{
		"mod/cue.mod/module.cue":  {Data: []byte(modFile)},
		"mod/module.cue":          {Data: []byte("package m\n")},
		"mod/LICENSE":             {Data: []byte("MIT")},
		"mod/cue.mod/pkg/a/b.cue": {Data: []byte("package b\n")},
		"other/x.cue":             {Data: []byte("package x\n")},
	}
	keyRoot := SyntheticRoot("x.example/m@v0", "v0.1.0")

	overlay, err := OverlayFromFS(fsys, "mod", keyRoot)
	require.NoError(t, err)
	want := []string{
		filepath.Join(keyRoot, "cue.mod", "module.cue"),
		filepath.Join(keyRoot, "cue.mod", "pkg", "a", "b.cue"),
		filepath.Join(keyRoot, "module.cue"),
	}
	assert.Equal(t, want, sortedKeys(overlay), "the license and the sibling tree are excluded; keys sit under the synthetic root")

	assert.Equal(t, "package m\n", string(overlay[filepath.Join(keyRoot, "module.cue")]),
		"entries are the file's bytes")

	// dir "" and "." mean the tree's root.
	for _, dir := range []string{"", "."} {
		overlay, err := OverlayFromFS(fsys, dir, keyRoot)
		require.NoError(t, err)
		assert.Contains(t, overlay, filepath.Join(keyRoot, "mod", "module.cue"), dir)
		assert.Contains(t, overlay, filepath.Join(keyRoot, "other", "x.cue"), dir)
		assert.NotContains(t, overlay, filepath.Join(keyRoot, "mod", "LICENSE"), dir)
	}

	// A tree with no .cue file yields an empty overlay, not an error; the
	// caller decides what an empty module means.
	overlay, err = OverlayFromFS(fstest.MapFS{"mod/LICENSE": {Data: []byte("MIT")}}, "mod", keyRoot)
	require.NoError(t, err)
	assert.Empty(t, overlay)

	_, err = OverlayFromFS(fsys, "absent", keyRoot)
	require.Error(t, err)
}

func TestSyntheticRoot(t *testing.T) {
	got := SyntheticRoot("x.example/modules/hello@v0", "v0.0.2")
	assert.Equal(t, filepath.Join(string(filepath.Separator), "opm-registry-module", "x.example_modules_hello_v0_v0.0.2"), got)
	assert.True(t, filepath.IsAbs(got))
	assert.Equal(t, got, SyntheticRoot("x.example/modules/hello@v0", "v0.0.2"), "deterministic")
	assert.NotEqual(t, got, SyntheticRoot("x.example/modules/hello@v0", "v0.0.3"))
}

func TestReadFile_OverlayAndDisk(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "opm-registry-module", "web_app")
	src := overlaySource(root)
	data, err := ReadFile(src, filepath.Join(root, "cue.mod", "module.cue"))
	require.NoError(t, err)
	assert.Equal(t, modFile, string(data))
	_, err = ReadFile(src, filepath.Join(root, "absent.cue"))
	require.ErrorContains(t, err, "not present in the staged overlay")

	disk := diskSource(t, "", map[string]string{"cue.mod/module.cue": modFile})
	data, err = ReadFile(disk, filepath.Join(disk.Root, "cue.mod", "module.cue"))
	require.NoError(t, err)
	assert.Equal(t, modFile, string(data))
	_, err = ReadFile(disk, filepath.Join(disk.Root, "absent.cue"))
	require.Error(t, err)

	_, err = ReadFile(nil, "x")
	require.Error(t, err)
}
