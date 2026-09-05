package kernel_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/kernel"
)

// mustSource compiles src into a [kernel.Source] through LoadSourceFromBytes,
// with origin baked as the filename — the contract Source documents.
func mustSource(t *testing.T, k *kernel.Kernel, origin, src string) kernel.Source {
	t.Helper()
	s, err := k.LoadSourceFromBytes(origin, []byte(src))
	require.NoError(t, err)
	return s
}
