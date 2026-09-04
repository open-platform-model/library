package platformmodule

import (
	"bytes"
	"strings"
	"testing"

	"cuelang.org/go/mod/modfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/schema"
)

const (
	opmPath    = "opmodel.dev/catalogs/opm@v4"
	k8sPath    = "opmodel.dev/catalogs/k8s@v1"
	modulePath = "opmodel.dev/platforms/cluster@v0"

	// coreVersion is the core pin the golden expectations below carry; the
	// closure fixture graph (closure_test.go) is authored against it too.
	coreVersion = "v2.0.0-alpha.7"
)

// twoCatalogInput is the operator's sample Platform (two subscriptions, one
// disabled) with the closure its module files resolve to. The golden
// expectations in TestGenerate_TwoCatalogs are the operator generator's,
// verbatim, so the operator's re-point onto this helper is a pure re-import.
func twoCatalogInput() Input {
	return Input{
		Name:       "cluster",
		Type:       "kubernetes",
		ModulePath: modulePath,
		Entries: []Entry{
			{Path: opmPath, Version: "4.0.1", Enable: true},
			{Path: k8sPath, Version: "1.0.0-alpha.2", Enable: false},
		},
		Deps: []Dep{
			{Path: CorePath, Version: coreVersion},
			{Path: opmPath, Version: "v4.0.1"},
			{Path: k8sPath, Version: "v1.0.0-alpha.2"},
			{Path: "cue.dev/x/k8s.io@v0", Version: "v0.10.0"},
		},
	}
}

// platform-module-generation spec, "Two catalogs".
func TestGenerate_TwoCatalogs(t *testing.T) {
	files, err := Generate(twoCatalogInput())
	require.NoError(t, err)
	require.Len(t, files, 2, "files: %v", keys(files))

	mf, err := modfile.Parse(files[ModuleFileName], ModuleFileName)
	require.NoError(t, err, "generated module.cue does not parse")
	assert.Equal(t, modulePath, mf.QualifiedModule())
	require.NotNil(t, mf.Language)
	assert.Equal(t, LanguageVersion, mf.Language.Version)
	want := map[string]string{
		CorePath:              coreVersion,
		opmPath:               "v4.0.1",
		k8sPath:               "v1.0.0-alpha.2",
		"cue.dev/x/k8s.io@v0": "v0.10.0",
	}
	require.Len(t, mf.Deps, len(want), "deps %v", mf.Deps)
	for path, version := range want {
		dep, ok := mf.Deps[path]
		require.True(t, ok, "module.cue does not pin %s", path)
		assert.Equal(t, version, dep.Version, "%s pinned at %s", path, dep.Version)
		assert.False(t, dep.Default, "%s carries a default-major marker; tidy writes none for a platform", path)
	}

	plat := string(files[PlatformFileName])
	for _, want := range []string{
		"package platform\n",
		"\tcore \"opmodel.dev/core@v2\"\n",
		"\tcat0 \"opmodel.dev/catalogs/k8s@v1\"\n",
		"\tcat1 \"opmodel.dev/catalogs/opm@v4\"\n",
		"core.#Platform\n",
		"metadata: name: \"cluster\"\n",
		"type: \"kubernetes\"\n",
		"\t\"opmodel.dev/catalogs/k8s@v1\": {\n\t\tenable:   false\n\t\tversion:  \"1.0.0-alpha.2\"\n\t\t#catalog: cat0\n\t}\n",
		"\t\"opmodel.dev/catalogs/opm@v4\": {\n\t\tenable:   true\n\t\tversion:  \"4.0.1\"\n\t\t#catalog: cat1\n\t}\n",
	} {
		assert.Contains(t, plat, want, "platform.cue lacks %q:\n%s", want, plat)
	}
	assert.Equal(t, 2, strings.Count(plat, "#catalog:"), "expected exactly two registry entries:\n%s", plat)
}

// platform-module-generation spec, "Same input, same bytes".
func TestGenerate_Deterministic(t *testing.T) {
	a, err := Generate(twoCatalogInput())
	require.NoError(t, err)

	// Same content, reversed input order.
	in := twoCatalogInput()
	for i, j := 0, len(in.Entries)-1; i < j; i, j = i+1, j-1 {
		in.Entries[i], in.Entries[j] = in.Entries[j], in.Entries[i]
	}
	for i, j := 0, len(in.Deps)-1; i < j; i, j = i+1, j-1 {
		in.Deps[i], in.Deps[j] = in.Deps[j], in.Deps[i]
	}
	b, err := Generate(in)
	require.NoError(t, err, "Generate (reordered)")

	for _, name := range []string{ModuleFileName, PlatformFileName} {
		assert.True(t, bytes.Equal(a[name], b[name]), "%s differs across input orderings:\n--- a\n%s\n--- b\n%s", name, a[name], b[name])
	}
}

// platform-module-generation spec, "Disabled entry still imports its catalog".
func TestGenerate_DisabledEntryIsKept(t *testing.T) {
	files, err := Generate(twoCatalogInput())
	require.NoError(t, err)
	plat := string(files[PlatformFileName])
	assert.Contains(t, plat, "\tcat0 \"opmodel.dev/catalogs/k8s@v1\"\n", "disabled catalog is not imported")
	assert.Contains(t, plat, "\t\tenable:   false\n", "disabled entry is not emitted with enable: false")
	mf, err := modfile.Parse(files[ModuleFileName], ModuleFileName)
	require.NoError(t, err)
	_, ok := mf.Deps[k8sPath]
	assert.True(t, ok, "disabled catalog is not pinned in module.cue")
}

func TestGenerate_StampsExpectedVersion(t *testing.T) {
	in := twoCatalogInput()
	in.Entries = in.Entries[:1]
	in.Entries[0].Version = "4.0.1"
	files, err := Generate(in)
	require.NoError(t, err)
	plat := string(files[PlatformFileName])
	// The stamp is the subscription's bare SemVer, never the "v"-prefixed
	// cue.mod form: it unifies with #Catalog.metadata.version, which is bare.
	assert.Contains(t, plat, "\t\tversion:  \"4.0.1\"\n", "entry does not stamp the subscription version")
	assert.NotContains(t, plat, "\"v4.0.1\"", "entry stamps the cue.mod form of the version")
}

func TestGenerate_EmptyRegistry(t *testing.T) {
	files, err := Generate(Input{
		Name:       "cluster",
		Type:       "kubernetes",
		ModulePath: modulePath,
		Deps:       []Dep{{Path: CorePath, Version: coreVersion}},
	})
	require.NoError(t, err)
	plat := string(files[PlatformFileName])
	assert.Contains(t, plat, "#registry: {}\n", "empty registry not emitted as an empty struct")
	assert.NotContains(t, plat, "cat0", "empty registry emitted an import")
}

// The module path is caller input: the generated module declares whatever
// identity the frontend owns (the operator's cluster path, the CLI's local
// path), byte-for-byte otherwise identical.
func TestGenerate_ModulePathIsInput(t *testing.T) {
	in := twoCatalogInput()
	in.ModulePath = "opmodel.dev/platforms/local@v0"
	files, err := Generate(in)
	require.NoError(t, err)
	mf, err := modfile.Parse(files[ModuleFileName], ModuleFileName)
	require.NoError(t, err)
	assert.Equal(t, "opmodel.dev/platforms/local@v0", mf.QualifiedModule())
	assert.True(t, bytes.HasPrefix(files[ModuleFileName], []byte("module: \"opmodel.dev/platforms/local@v0\"\n")), "%s", files[ModuleFileName])

	base, err := Generate(twoCatalogInput())
	require.NoError(t, err)
	assert.Equal(t, base[PlatformFileName], files[PlatformFileName], "platform.cue does not depend on the module path")
}

// platform-module-generation spec, "Invalid input is refused": every refusal
// names the offending field or path and emits no files.
func TestGenerate_Refusals(t *testing.T) {
	cases := map[string]struct {
		mutate func(*Input)
		want   string
	}{
		"missing name":          {func(in *Input) { in.Name = "" }, "name is required"},
		"missing type":          {func(in *Input) { in.Type = "" }, "type is required"},
		"missing module path":   {func(in *Input) { in.ModulePath = "" }, "module path is required"},
		"duplicate entry":       {func(in *Input) { in.Entries = append(in.Entries, in.Entries[0]) }, `duplicate registry entry "opmodel.dev/catalogs/opm@v4"`},
		"entry not pinned":      {func(in *Input) { in.Deps = in.Deps[:1] }, "does not pin registry entry opmodel.dev/catalogs/k8s@v1"},
		"core not pinned":       {func(in *Input) { in.Deps = in.Deps[1:] }, "does not pin opmodel.dev/core@v2"},
		"dep pinned twice":      {func(in *Input) { in.Deps = append(in.Deps, Dep{Path: opmPath, Version: "v4.0.2"}) }, `"opmodel.dev/catalogs/opm@v4" pinned twice (v4.0.1 and v4.0.2)`},
		"empty entry version":   {func(in *Input) { in.Entries[0].Version = "" }, "empty path or version"},
		"empty entry path":      {func(in *Input) { in.Entries[0].Path = "" }, "empty path or version"},
		"empty dep version":     {func(in *Input) { in.Deps[3].Version = "" }, `dependency "cue.dev/x/k8s.io@v0" has an empty path or version`},
		"empty dep path":        {func(in *Input) { in.Deps[3].Path = "" }, "empty path or version"},
		"dep pinned twice same": {func(in *Input) { in.Deps = append(in.Deps, Dep{Path: opmPath, Version: "v4.0.1"}) }, ""},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			in := twoCatalogInput()
			tc.mutate(&in)
			files, err := Generate(in)
			if tc.want == "" {
				// An identical duplicate pin is not a conflict.
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
			assert.Nil(t, files, "a refusal emits no files")
		})
	}
}

// platform-module-generation spec, "Default core pin": roots pin core at
// the kernel's verified release plus one root per entry, sorted by path.
func TestRoots_DefaultCorePin(t *testing.T) {
	roots := Roots([]Entry{
		{Path: opmPath, Version: "4.0.1", Enable: true},
		{Path: k8sPath, Version: "1.0.0-alpha.2", Enable: false},
	})
	assert.Equal(t, []Dep{
		{Path: k8sPath, Version: "v1.0.0-alpha.2"},
		{Path: opmPath, Version: "v4.0.1"},
		{Path: CorePath, Version: schema.DefaultSchemaVersion()},
	}, roots)
	assert.Equal(t, "opmodel.dev/core@"+schema.DefaultSchemaVersion(), schema.DefaultSchemaModule,
		"the default core pin is the version of the kernel's default schema module")
}

// platform-module-generation spec, "Explicit core pin": WithCoreVersion
// replaces the default, bare SemVer canonicalised, and the generated module
// file reflects it.
func TestRoots_ExplicitCorePin(t *testing.T) {
	entries := []Entry{{Path: opmPath, Version: "4.0.1", Enable: true}}
	for _, v := range []string{"2.0.0-alpha.9", "v2.0.0-alpha.9"} {
		roots := Roots(entries, WithCoreVersion(v))
		assert.Equal(t, []Dep{
			{Path: opmPath, Version: "v4.0.1"},
			{Path: CorePath, Version: "v2.0.0-alpha.9"},
		}, roots, "core version %q", v)

		files, err := Generate(Input{Name: "cluster", Type: "kubernetes", ModulePath: modulePath, Entries: entries, Deps: roots})
		require.NoError(t, err)
		mf, err := modfile.Parse(files[ModuleFileName], ModuleFileName)
		require.NoError(t, err)
		assert.Equal(t, "v2.0.0-alpha.9", mf.Deps[CorePath].Version)
	}
}

func keys(files Files) []string {
	out := make([]string, 0, len(files))
	for k := range files {
		out = append(out, k)
	}
	return out
}
