package loader_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/internal/cueenv"
	"github.com/open-platform-model/library/opm/internal/loader"
	"github.com/open-platform-model/library/opm/internal/schematest"
	"github.com/open-platform-model/library/opm/schema"
)

// loadDir builds the root package of dir on disk through LoadDir with no
// registry override — the shape every directory acquire verb runs.
func loadDir(dir string, spec loader.ArtifactSpec) (cue.Value, error) {
	return loader.LoadDir(cuecontext.New(), dir, ".", nil, nil, spec)
}

// writeTempPkgDir writes a single-file CUE package under a fresh temp dir as
// file and returns the dir path.
func writeTempPkgDir(t *testing.T, file, content string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, file), []byte(content), 0o644))
	return dir
}

func writeTempModuleDir(t *testing.T, content string) string {
	t.Helper()
	return writeTempPkgDir(t, "module.cue", content)
}

func writeTempInstanceDir(t *testing.T, content string) string {
	t.Helper()
	return writeTempPkgDir(t, "instance.cue", content)
}

func writeTempPlatformDir(t *testing.T, content string) string {
	t.Helper()
	return writeTempPkgDir(t, "platform.cue", content)
}

const platformFixture = `
package platform
kind: "Platform"
metadata: {
	name: "demo-platform"
}
type: "kubernetes"
`

func TestLoadDir_Module(t *testing.T) {
	dir := writeTempModuleDir(t, `
package mod
kind: "Module"
metadata: {
	name:       "demo"
	modulePath: "example.com/modules"
	version:    "0.1.0"
}
`)

	val, err := loadDir(dir, loader.ModuleSpec)
	require.NoError(t, err)
	assert.True(t, val.Exists())
}

func TestLoadDir_Instance(t *testing.T) {
	dir := writeTempInstanceDir(t, `
package instance
kind: "ModuleInstance"
metadata: {
	name: "demo"
	namespace: "ns"
}
#module: {kind: "Module"}
`)

	val, err := loadDir(dir, loader.InstanceSpec)
	require.NoError(t, err)
	assert.True(t, val.Exists())
}

func TestLoadDir_Platform(t *testing.T) {
	dir := writeTempPlatformDir(t, platformFixture)

	val, err := loadDir(dir, loader.PlatformSpec)
	require.NoError(t, err)
	assert.True(t, val.Exists())

	name, err := val.LookupPath(cue.ParsePath("metadata.name")).String()
	require.NoError(t, err)
	assert.Equal(t, "demo-platform", name)
}

// TestLoadDir_MultiFile pins the multi-file package behavior that motivates a
// single package loader: instance.cue and values.cue share a package
// declaration; LoadDir MUST unify them into one cue.Value.
func TestLoadDir_MultiFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "instance.cue"), []byte(`
package instance

kind:       "ModuleInstance"
metadata: {
	name:      "demo"
	namespace: "ns"
}
#module: {kind: "Module"}
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "values.cue"), []byte(`
package instance

values: { replicas: 3 }
`), 0o644))

	val, err := loadDir(dir, loader.InstanceSpec)
	require.NoError(t, err)
	assert.True(t, val.Exists())

	name, err := val.LookupPath(cue.ParsePath("metadata.name")).String()
	require.NoError(t, err)
	assert.Equal(t, "demo", name)

	replicas, err := val.LookupPath(cue.ParsePath("values.replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(3), replicas)
}

// A non-empty registry mapping is plumbed through to load.Config.Env without
// aborting the load of a package that imports nothing registry-backed.
func TestLoadDir_RegistryOverrideAccepted(t *testing.T) {
	env := cueenv.Override("testing.opmodel.dev=localhost:5000+insecure", "")
	require.NotNil(t, env, "a non-empty mapping produces an environment slice")

	for name, tc := range map[string]struct {
		dir  string
		spec loader.ArtifactSpec
	}{
		"instance": {writeTempInstanceDir(t, `
package instance
kind: "ModuleInstance"
metadata: { name: "demo", namespace: "ns" }
#module: {kind: "Module"}
`), loader.InstanceSpec},
		"platform": {writeTempPlatformDir(t, platformFixture), loader.PlatformSpec},
	} {
		t.Run(name, func(t *testing.T) {
			val, err := loader.LoadDir(cuecontext.New(), tc.dir, ".", nil, env, tc.spec)
			require.NoError(t, err, "registry override must be accepted even when no imports use it")
			assert.True(t, val.Exists())
		})
	}
}

func TestLoadDir_NotADirectory(t *testing.T) {
	dir := writeTempPlatformDir(t, platformFixture)

	_, err := loadDir(filepath.Join(dir, "platform.cue"), loader.PlatformSpec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not a directory")
}

func TestLoadDir_MissingPath(t *testing.T) {
	_, err := loadDir(filepath.Join(t.TempDir(), "no", "such", "path"), loader.PlatformSpec)
	require.Error(t, err)
	assert.True(t, errors.Is(err, os.ErrNotExist), "got %v", err)
}

// TestShapeGate_RejectsMalformedPackages drives every artifact spec through a
// fixture that violates one shape-gate rule and asserts the returned error
// wraps the expected sentinel.
func TestShapeGate_RejectsMalformedPackages(t *testing.T) {
	tests := []struct {
		name     string
		spec     loader.ArtifactSpec
		content  string
		sentinel error
		contains []string // substrings the message must carry, if any
	}{
		{
			name: "module loaded from a platform package",
			spec: loader.ModuleSpec,
			content: `
package mod
kind:       "Platform"
metadata: {name: "demo", modulePath: "example.com/modules", version: "0.1.0"}
`,
			sentinel: oerrors.ErrWrongKind,
		},
		{
			name: "module missing metadata.name",
			spec: loader.ModuleSpec,
			content: `
package mod
kind:       "Module"
metadata: {modulePath: "example.com/modules", version: "0.1.0"}
`,
			sentinel: oerrors.ErrMissingRequiredField,
		},
		{
			name: "module with a defaulted-disjunction metadata.version",
			spec: loader.ModuleSpec,
			content: `
package mod
#T: string & =~"^\\d+\\.\\d+\\.\\d+"
Version: #T | *"1.0.1"
kind:    "Module"
metadata: {name: "demo", modulePath: "example.com/modules", version: Version}
`,
			sentinel: oerrors.ErrMissingRequiredField,
			contains: []string{`"metadata.version"`, `"1.0.1"`, "defaulted disjunction", "concrete literals"},
		},
		{
			name: "module with an open (non-defaulted) metadata.version",
			spec: loader.ModuleSpec,
			content: `
package mod
kind:       "Module"
metadata: {name: "demo", modulePath: "example.com/modules", version: string}
`,
			sentinel: oerrors.ErrMissingRequiredField,
			contains: []string{`"metadata.version" is not concrete`},
		},
		{
			name: "instance with a non-module #module",
			spec: loader.InstanceSpec,
			content: `
package instance
kind:       "ModuleInstance"
metadata: {name: "demo", namespace: "ns"}
#module: {kind: "Platform"}
`,
			sentinel: oerrors.ErrWrongKind,
		},
		{
			name: "instance missing #module",
			spec: loader.InstanceSpec,
			content: `
package instance
kind:       "ModuleInstance"
metadata: {name: "demo", namespace: "ns"}
`,
			sentinel: oerrors.ErrMissingRequiredField,
		},
		{
			name: "platform missing type",
			spec: loader.PlatformSpec,
			content: `
package platform
kind:       "Platform"
metadata: {name: "demo"}
`,
			sentinel: oerrors.ErrMissingRequiredField,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := writeTempPkgDir(t, "artifact.cue", tc.content)

			_, err := loadDir(dir, tc.spec)
			require.Error(t, err)
			assert.True(t, errors.Is(err, tc.sentinel), "want %v, got %v", tc.sentinel, err)
			for _, want := range tc.contains {
				assert.Contains(t, err.Error(), want)
			}
		})
	}
}

// TestShapeGate_RejectsConflictingPackageClauses pins the CUE-loader behavior:
// two files in one directory declaring different package names is a hard
// error, surfaced rather than silently resolved to one instance.
func TestShapeGate_RejectsConflictingPackageClauses(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.cue"), []byte("package one\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.cue"), []byte("package two\n"), 0o644))

	_, err := loadDir(dir, loader.ModuleSpec)
	require.Error(t, err)
}

// TestShapeGate_WellFormedArtifactsPass is the regression guard: a well-formed
// module, instance, and platform each still load successfully.
func TestShapeGate_WellFormedArtifactsPass(t *testing.T) {
	t.Run("module with a plain-literal identity value", func(t *testing.T) {
		// The identity-package form: Version is a concrete literal and
		// metadata.version references it. The counterpart of the
		// defaulted-disjunction rejection above.
		dir := writeTempModuleDir(t, `
package mod
Version: "1.0.1"
kind:    "Module"
metadata: {name: "demo", modulePath: "example.com/modules", version: Version}
`)
		val, err := loadDir(dir, loader.ModuleSpec)
		require.NoError(t, err)
		assert.True(t, val.Exists())
	})

	t.Run("platform with an empty registry", func(t *testing.T) {
		dir := writeTempPlatformDir(t, `
package platform
kind:       "Platform"
metadata: {name: "demo"}
type: "kubernetes"
#registry: {}
`)
		val, err := loadDir(dir, loader.PlatformSpec)
		require.NoError(t, err)
		assert.True(t, val.Exists())
	})
}

// writePlatformModule writes a platform module importing core (resolved from
// the warm workspace cache at the release schema.DefaultSchemaModule pins)
// whose #registry is registryBody, plus the extra package-level source in
// extra, and returns the module directory.
func writePlatformModule(t *testing.T, registryBody, extra string) string {
	t.Helper()
	dir := t.TempDir()
	coreVersion := strings.TrimPrefix(schema.DefaultSchemaModule, "opmodel.dev/core@")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cue.mod"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cue.mod", "module.cue"), []byte(fmt.Sprintf(`module: "testing.opmodel.dev/loader-platform-test@v0"
language: version: "v0.17.0"
deps: "opmodel.dev/core@v2": v: %q
`, coreVersion)), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "platform.cue"), []byte(fmt.Sprintf(`package platform

import c "opmodel.dev/core@v2"

c.#Platform
metadata: name: "gated"
type: "kubernetes"
#registry: %s
%s`, registryBody, extra)), 0o644))
	return dir
}

// artifact-types spec, "Registry entry with no embedded catalog rejected":
// the retired subscription shape (a version scalar, no #catalog) is
// incomplete exactly where the embedded catalog would have completed it
// (core derives `version` from it, 0019 D5), and the gate names the entry.
func TestLoadDir_SubscriptionShapedRegistryRejected(t *testing.T) {
	schematest.SetEnv(t)
	dir := writePlatformModule(t, `"example.test/cat@v0": {enable: true, version: "0.1.0"}`, "")

	_, err := loadDir(dir, loader.PlatformSpec)
	require.Error(t, err)
	assert.True(t, errors.Is(err, oerrors.ErrMissingRequiredField), "got %v", err)
	assert.Contains(t, err.Error(), "example.test/cat@v0", "the refusal names the entry")
	assert.Contains(t, err.Error(), "version", "the refusal names the field the embedded catalog would supply")
	assert.Contains(t, err.Error(), "#registry")
}

// artifact-types spec, "Registry entry with no embedded catalog rejected",
// second half: the same entry with a catalog embedded passes, its `version`
// derived from the catalog's stamped identity. An empty #registry passes too.
func TestLoadDir_EmbeddedCatalogRegistryPasses(t *testing.T) {
	schematest.SetEnv(t)
	dir := writePlatformModule(t, `"example.test/cat@v0": {enable: true, #catalog: _cat}`, `
_cat: c.#Catalog & {
	metadata: {
		modulePath:  "example.test/cat@v0"
		version:     "0.1.0"
		description: "inline catalog"
	}
	#transformers: {}
}
`)

	val, err := loadDir(dir, loader.PlatformSpec)
	require.NoError(t, err)
	version, err := val.LookupPath(cue.ParsePath(`#registry."example.test/cat@v0".version`)).String()
	require.NoError(t, err)
	assert.Equal(t, "0.1.0", version, "core derives the entry version from the embedded catalog")

	empty := writePlatformModule(t, `{}`, "")
	_, err = loadDir(empty, loader.PlatformSpec)
	require.NoError(t, err, "a platform carrying no catalogs is complete")
}
