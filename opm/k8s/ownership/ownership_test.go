package ownership_test

import (
	"go/build"
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
