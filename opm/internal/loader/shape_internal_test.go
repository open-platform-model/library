package loader

import (
	"errors"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/open-platform-model/library/opm/errors"
)

// TestGate_NonStructRoot covers the structural guard that LoadDir cannot
// exercise through a file package: a CUE package root is always a struct, so
// a scalar root can only be reached by calling Gate directly. The guard exists
// so a future non-file source (bytes loader, embedded SDK) cannot slip a
// scalar past the gate.
func TestGate_NonStructRoot(t *testing.T) {
	ctx := cuecontext.New()

	for _, src := range []string{`42`, `"just a string"`, `[1, 2, 3]`} {
		val := ctx.CompileString(src)
		require.NoError(t, val.Err())

		err := Gate(val, ModuleSpec)
		require.Error(t, err)
		assert.True(t, errors.Is(err, oerrors.ErrInvalidPackage), "src %q: want ErrInvalidPackage, got %v", src, err)
	}
}
