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
// returns its exported package-level type specs and value names (vars and
// consts) by name.
func exportedDecls(t *testing.T, dir string) (types map[string]*ast.TypeSpec, values map[string]*ast.ValueSpec) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	types, values = map[string]*ast.TypeSpec{}, map[string]*ast.ValueSpec{}
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
						if n.IsExported() {
							values[n.Name] = s
						}
					}
				}
			}
		}
	}
	require.NotEmpty(t, types, "found no exported type in %s; the scan is broken", dir)
	return types, values
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
	for _, pkg := range []string{"schema", "module", "platform", "catalog"} {
		types, _ := exportedDecls(t, filepath.Join("..", pkg))
		for name, home := range homes {
			spec, declared := types[name]
			if pkg != home {
				assert.False(t, declared, "opm/%s declares %s; its one name is %s.%s", pkg, name, home, name)
				continue
			}
			require.True(t, declared, "opm/%s must declare %s", pkg, name)
			assert.False(t, spec.Assign.IsValid(), "%s.%s must be a declared type, not an alias", pkg, name)
		}
		for name, spec := range types {
			assert.False(t, spec.Assign.IsValid(), "opm/%s exports the alias %s; an exported type has one name", pkg, name)
		}
	}
}

// opm/schema exports only the paths and core-release constants a consumer
// reads. Every other path the kernel's Go code reads lives in
// opm/internal/corepath, where no importer can reach or reassign it.
func TestSurface_SchemaExportsOnlyConsumerPaths(t *testing.T) {
	_, values := exportedDecls(t, ".")

	var paths []string
	for name, spec := range values {
		for _, v := range spec.Values {
			call, ok := v.(*ast.CallExpr)
			if !ok {
				continue
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if ok && (sel.Sel.Name == "ParsePath" || sel.Sel.Name == "MakePath") {
				paths = append(paths, name)
				break
			}
		}
	}
	assert.ElementsMatch(t, []string{"Metadata", "Module", "CatalogProvides"}, paths,
		"a new exported path is a v1 contract: put it in opm/internal/corepath unless a consumer must read it")

	var since []string
	for name := range values {
		if strings.HasSuffix(name, "Since") {
			since = append(since, name)
		}
	}
	assert.ElementsMatch(t, []string{"ProvidedBySince", "ProvidesSince"}, since,
		"a core-release constant is exported only when code outside the library's tests reads it")
}
