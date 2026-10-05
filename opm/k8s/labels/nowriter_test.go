package labels_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/k8s/labels"
)

// TestNoLibraryCodeSetsTheAdoptAnnotation keeps the library from writing the
// adopt annotation. Outside this package, which declares the key, and
// opm/k8s/ownership, which only reads it, no non-test file under opm/ spells
// its literal or refers to [labels.AnnotationAdopt]. Only a user writes the
// annotation (0012:D8:R6).
func TestNoLibraryCodeSetsTheAdoptAnnotation(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	self, err := filepath.Abs(".")
	require.NoError(t, err)
	reader := filepath.Join(root, "k8s", "ownership")

	var offences []string
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		dir := filepath.Dir(path)
		if dir == self {
			return nil
		}
		fset := token.NewFileSet()
		f, perr := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if perr != nil {
			return perr
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.BasicLit:
				if n.Kind != token.STRING {
					return true
				}
				if v, uerr := strconv.Unquote(n.Value); uerr == nil && strings.Contains(v, labels.AnnotationAdopt) {
					offences = append(offences, fset.Position(n.Pos()).String()+": spells the adopt annotation key")
				}
			case *ast.SelectorExpr:
				if dir != reader && n.Sel.Name == "AnnotationAdopt" {
					offences = append(offences, fset.Position(n.Pos()).String()+": refers to labels.AnnotationAdopt")
				}
			}
			return true
		})
		return nil
	})
	require.NoError(t, err)
	require.Empty(t, offences)
}
