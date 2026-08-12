package compat

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHighestStable(t *testing.T) {
	tests := []struct {
		name      string
		published []string
		want      string
	}{
		{
			name:      "all stable returns highest",
			published: []string{"v0.4.0", "v0.5.0", "v0.5.1"},
			want:      "v0.5.1",
		},
		{
			name:      "skips a higher pre-release",
			published: []string{"v0.5.0", "v0.5.1", "v0.6.0-dev.1"},
			want:      "v0.5.1",
		},
		{
			name:      "pre-release-only falls back to highest overall",
			published: []string{"v0.6.0-dev.1", "v0.6.0-dev.2"},
			want:      "v0.6.0-dev.2",
		},
		{
			name:      "unparseable trailing entry is skipped",
			published: []string{"v0.5.1", "not-semver"},
			want:      "v0.5.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, HighestStable(tt.published))
		})
	}
}
