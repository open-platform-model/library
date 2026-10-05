package modversion

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestBare(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"v1.0.0", "1.0.0"},
		{"1.0.0", "1.0.0"},
		{"v1.0.0-alpha.2", "1.0.0-alpha.2"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, Bare(tt.in))
			assert.Equal(t, tt.want, Bare(Canonical(tt.in)), "Bare inverts Canonical")
		})
	}
}

// TestMajor accepts both spellings a fixture writer holds: the bare version
// a served fixture is published at and the v-prefixed core release.
func TestMajor(t *testing.T) {
	for in, want := range map[string]string{
		"0.1.0":           "v0",
		"v0.1.0":          "v0",
		"2.0.0-alpha.13":  "v2",
		"v2.0.0-alpha.13": "v2",
		"2.0.0-beta.1":    "v2",
		"v2.0.0-beta.1":   "v2",
		"v2":              "v2",
		"":                "v",
	} {
		assert.Equal(t, want, Major(in), "Major(%q)", in)
	}
}

func TestValid(t *testing.T) {
	for in, want := range map[string]bool{
		"1.0.0":          true,
		"v1.0.0":         true,
		"2.0.0-beta.10":  true,
		"v2.0.0-beta.10": true,
		"not-a-version":  false,
		"":               false,
	} {
		assert.Equal(t, want, Valid(in), "Valid(%q)", in)
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"v1.0.0", "v1.0.0", 0},
		{"1.0.0", "v1.0.0", 0},
		{"v2.0.0-beta.2", "v2.0.0-beta.10", -1},
		{"v2.0.0-beta.10", "v2.0.0", -1},
		{"2.0.0", "2.0.0-beta.10", 1},
		{"v1.3.0", "1.2.0", 1},
	}
	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			got, err := Compare(tt.a, tt.b)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCompare_InvalidNamesTheInput(t *testing.T) {
	_, err := Compare("bogus", "v1.0.0")
	require.Error(t, err)
	assert.Equal(t, `invalid version "bogus"`, err.Error())

	_, err = Compare("1.0.0", "1.x")
	require.Error(t, err)
	assert.Equal(t, `invalid version "1.x"`, err.Error())
}
