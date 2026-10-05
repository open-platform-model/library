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
// adopt annotation: no non-test file under opm/ outside this package spells
// its literal, and no non-test file calls a SetAnnotations method. Only a
// user writes the annotation (0012:D8:R6).
func TestNoLibraryCodeSetsTheAdoptAnnotation(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	self, err := filepath.Abs(".")
	require.NoError(t, err)

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
		fset := token.NewFileSet()
		f, perr := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if perr != nil {
			return perr
		}
		inSelf := filepath.Dir(path) == self
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.BasicLit:
				if inSelf || n.Kind != token.STRING {
					return true
				}
				if v, uerr := strconv.Unquote(n.Value); uerr == nil && strings.Contains(v, labels.AnnotationAdopt) {
					offences = append(offences, fset.Position(n.Pos()).String()+": spells the adopt annotation key")
				}
			case *ast.CallExpr:
				if sel, ok := n.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "SetAnnotations" {
					offences = append(offences, fset.Position(n.Pos()).String()+": calls SetAnnotations")
				}
			}
			return true
		})
		return nil
	})
	require.NoError(t, err)
	require.Empty(t, offences)
}
