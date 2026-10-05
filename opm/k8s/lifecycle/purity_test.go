package lifecycle_test

import (
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPackageImportsNoClockEnvironmentLoggerOrSync pins the package's purity:
// none of its direct imports reads a clock or the environment, logs, or
// synchronises goroutines.
func TestPackageImportsNoClockEnvironmentLoggerOrSync(t *testing.T) {
	pkg, err := build.ImportDir(".", 0)
	require.NoError(t, err)
	require.NotEmpty(t, pkg.Imports)
	for _, banned := range []string{"os", "time", "log", "log/slog", "sync"} {
		assert.NotContains(t, pkg.Imports, banned)
	}
}

// TestPackageStartsNoGoroutine parses the package's non-test files and fails
// on any go statement.
func TestPackageStartsNoGoroutine(t *testing.T) {
	pkg, err := build.ImportDir(".", 0)
	require.NoError(t, err)
	fset := token.NewFileSet()
	for _, name := range pkg.GoFiles {
		f, err := parser.ParseFile(fset, filepath.Join(pkg.Dir, name), nil, 0)
		require.NoError(t, err)
		ast.Inspect(f, func(n ast.Node) bool {
			if g, ok := n.(*ast.GoStmt); ok {
				t.Errorf("%s: go statement", fset.Position(g.Pos()))
			}
			return true
		})
	}
}
