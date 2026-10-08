package schema_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// exportedDecls parses the non-test Go files of the package in dir and
// returns its exported package-level type specs by name, and the names of
// its exported variables and constants.
func exportedDecls(t *testing.T, dir string) (types map[string]*ast.TypeSpec, vars, consts []string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	types = map[string]*ast.TypeSpec{}
	fset := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, 0)
		require.NoError(t, err)
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range gen.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if s.Name.IsExported() {
						types[s.Name.Name] = s
					}
				case *ast.ValueSpec:
					for _, n := range s.Names {
						switch {
						case !n.IsExported():
						case gen.Tok == token.VAR:
							vars = append(vars, n.Name)
						default:
							consts = append(consts, n.Name)
						}
					}
				}
			}
		}
	}
	require.NotEmpty(t, types, "found no exported type in %s; the scan is broken", dir)
	return types, vars, consts
}

// Each metadata type and the staged-source type has exactly one name: it is
// declared in the package of its artifact, and no package re-exports it
// through an alias.
func TestSurface_OneNamePerType(t *testing.T) {
	homes := map[string]string{
		"ModuleMetadata":   "module",
		"InstanceMetadata": "module",
		"PlatformMetadata": "platform",
		"CatalogMetadata":  "catalog",
		"Source":           "module",
	}
	// catalog.Source is a deprecated alias, kept while opm-operator at main
	// still uses the name. The change that removes it empties this list.
	deprecatedAliases := map[string]bool{"catalog.Source": true}
	sawDeprecated := map[string]bool{}
	for _, pkg := range []string{"schema", "module", "platform", "catalog"} {
		types, _, _ := exportedDecls(t, filepath.Join("..", pkg))
		for name, home := range homes {
			spec, declared := types[name]
			if declared && deprecatedAliases[pkg+"."+name] {
				assert.True(t, spec.Assign.IsValid(), "%s.%s is kept only as an alias of %s.%s", pkg, name, home, name)
				sawDeprecated[pkg+"."+name] = true
				continue
			}
			if pkg != home {
				assert.False(t, declared, "opm/%s declares %s; its one name is %s.%s", pkg, name, home, name)
				continue
			}
			require.True(t, declared, "opm/%s must declare %s", pkg, name)
			assert.False(t, spec.Assign.IsValid(), "%s.%s must be a declared type, not an alias", pkg, name)
		}
		for name, spec := range types {
			if deprecatedAliases[pkg+"."+name] {
				continue
			}
			assert.False(t, spec.Assign.IsValid(), "opm/%s exports the alias %s; an exported type has one name", pkg, name)
		}
	}
	for alias := range deprecatedAliases {
		assert.True(t, sawDeprecated[alias], "%s is gone: remove it from deprecatedAliases", alias)
	}
}

// opm/schema exports only the paths and core-release constants a consumer
// reads. Every other path the kernel's Go code reads lives in
// opm/internal/corepath, where no importer can reach or reassign it.
func TestSurface_SchemaExportsOnlyConsumerPaths(t *testing.T) {
	_, vars, consts := exportedDecls(t, ".")

	// Every exported variable of the package is pinned by name, whatever its
	// type or initializer: the three paths are the only ones.
	assert.ElementsMatch(t, []string{"Metadata", "Module", "CatalogProvides"}, vars,
		"a new exported variable is a v1 contract: a path goes in opm/internal/corepath unless a consumer must read it")

	var since []string
	for _, name := range consts {
		if strings.HasSuffix(name, "Since") {
			since = append(since, name)
		}
	}
	assert.ElementsMatch(t, []string{"ProvidedBySince", "ProvidesSince"}, since,
		"a core-release constant is exported only when code outside the library's tests reads it")
}
