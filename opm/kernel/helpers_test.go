package kernel_test

import (
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/kernel"
)

// lookupString reads a concrete string at path out of v, failing the test
// when it is absent or not a string.
func lookupString(t *testing.T, v cue.Value, path string) string {
	t.Helper()
	s, err := v.LookupPath(cue.ParsePath(path)).String()
	require.NoError(t, err, "lookup %s", path)
	return s
}

// mustSource compiles src into a [kernel.Source] through LoadSourceFromBytes,
// with origin baked as the filename — the contract Source documents.
func mustSource(t *testing.T, k *kernel.Kernel, origin, src string) kernel.Source {
	t.Helper()
	s, err := k.LoadSourceFromBytes(origin, []byte(src))
	require.NoError(t, err)
	return s
}
