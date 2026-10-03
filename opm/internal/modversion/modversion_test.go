package modversion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanonical(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"1.0.0", "v1.0.0"},
		{"v1.0.0", "v1.0.0"},
		{"1.0.0-alpha.2", "v1.0.0-alpha.2"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, Canonical(tt.in))
		})
	}
}
