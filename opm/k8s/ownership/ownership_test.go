package ownership_test

import (
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/k8s/ownership"
)

func TestSafetyExcluded(t *testing.T) {
	tests := []struct {
		group, kind string
		want        bool
	}{
		{"", "Namespace", true},
		{"apiextensions.k8s.io", "CustomResourceDefinition", true},
		{"example.com", "Namespace", false},
		{"example.com", "CustomResourceDefinition", false},
		{"", "CustomResourceDefinition", false},
		{"apiextensions.k8s.io", "Namespace", false},
		{"apps", "Deployment", false},
		{"", "ConfigMap", false},
	}
	for _, tt := range tests {
		t.Run(tt.group+"/"+tt.kind, func(t *testing.T) {
			assert.Equal(t, tt.want, ownership.SafetyExcluded(tt.group, tt.kind))
		})
	}
}

func TestObjectString(t *testing.T) {
	assert.Equal(t, "Namespace/team-a", ownership.Object{Kind: "Namespace", Name: "team-a"}.String())
	assert.Equal(t, "Deployment/web/api",
		ownership.Object{Group: "apps", Kind: "Deployment", Namespace: "web", Name: "api"}.String())
}

// TestImportsNoClockEnvOrLogger backs the purity rule with the package's
// direct imports: a verdict that read a clock, the environment or a logger
// would no longer be a function of its inputs.
func TestImportsNoClockEnvOrLogger(t *testing.T) {
	pkg, err := build.ImportDir(".", 0)
	require.NoError(t, err)
	for _, forbidden := range []string{"os", "time", "log", "log/slog"} {
		assert.NotContains(t, pkg.Imports, forbidden)
	}
}

// TestDocsStateTheFrontendsPartOfTheHandOver checks that the two docs a
// frontend author reads say what to do with an inventoried object refused as
// adopted-elsewhere: drop it from the next inventory and never delete it for
// that refusal. The library cannot check the frontend, so it checks the docs.
func TestDocsStateTheFrontendsPartOfTheHandOver(t *testing.T) {
	fset := token.NewFileSet()
	docFile, err := parser.ParseFile(fset, "doc.go", nil, parser.ParseComments)
	require.NoError(t, err)
	applyFile, err := parser.ParseFile(fset, "apply.go", nil, parser.ParseComments)
	require.NoError(t, err)

	packageDoc := docFile.Doc.Text()
	var inInventoryDoc string
	ast.Inspect(applyFile, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok || ts.Name.Name != "ApplyInput" {
			return true
		}
		for _, field := range ts.Type.(*ast.StructType).Fields.List {
			if len(field.Names) == 1 && field.Names[0].Name == "InInventory" {
				inInventoryDoc = field.Doc.Text()
			}
		}
		return false
	})
	for name, doc := range map[string]string{"package doc": packageDoc, "ApplyInput.InInventory doc": inInventoryDoc} {
		flat := strings.Join(strings.Fields(doc), " ")
		assert.Contains(t, flat, "drops an object refused as adopted-elsewhere from the inventory it records next", name)
		assert.Contains(t, flat, "never deletes the object for that refusal", name)
	}
}
