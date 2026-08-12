package compat

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		apiVersion string
		major      int
		level      Level
		ok         bool
	}{
		{"v1alpha1", 1, LevelAlpha, true},
		{"v1alpha2", 1, LevelAlpha, true},
		{"v1beta1", 1, LevelBeta, true},
		{"v1", 1, LevelGA, true},
		{"v2", 2, LevelGA, true},
		{"v10", 10, LevelGA, true},
		{"v2beta3", 2, LevelBeta, true},
		// Outside #APIVersionType's grammar.
		{"v1alpha", 0, LevelGA, false}, // digit required after alpha/beta
		{"v1beta", 0, LevelGA, false},
		{"V1", 0, LevelGA, false},
		{"1.2.0", 0, LevelGA, false},
		{"", 0, LevelGA, false},
		{"v1alpha1extra", 0, LevelGA, false},
	}
	for _, tt := range tests {
		t.Run(tt.apiVersion, func(t *testing.T) {
			major, level, ok := ParseLevel(tt.apiVersion)
			require.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.Equal(t, tt.major, major)
				assert.Equal(t, tt.level, level)
			}
		})
	}
}

func TestLevelEnforced(t *testing.T) {
	assert.False(t, LevelAlpha.Enforced(), "alpha promises nothing (D34)")
	assert.True(t, LevelBeta.Enforced())
	assert.True(t, LevelGA.Enforced())
}

// TestCompareAPIVersionsTransitive pins the fix for the measured
// non-transitivity in sortFQNsBySemVer (0010 D34): under a per-pair rule
// switch, v1alpha1 < v2, v2 < v10 and v10 < v1alpha1 are all true at once, so
// the same three strings sort differently depending on input order. The
// kube-aware ordering must produce one order from every permutation.
func TestCompareAPIVersionsTransitive(t *testing.T) {
	want := []string{"v1alpha1", "v2", "v10"}
	perms := [][]string{
		{"v1alpha1", "v2", "v10"},
		{"v1alpha1", "v10", "v2"},
		{"v2", "v1alpha1", "v10"},
		{"v2", "v10", "v1alpha1"},
		{"v10", "v1alpha1", "v2"},
		{"v10", "v2", "v1alpha1"},
	}
	for _, p := range perms {
		in := append([]string(nil), p...)
		sort.Slice(in, func(i, j int) bool { return CompareAPIVersions(in[i], in[j]) < 0 })
		assert.Equal(t, want, in, "input order %v", p)
	}
}

func TestCompareAPIVersionsLadder(t *testing.T) {
	// Level dominates, then major, then the alpha/beta number.
	ordered := []string{"v1alpha1", "v1alpha2", "v2alpha1", "v1beta1", "v1beta2", "v2beta1", "v1", "v2", "v10"}
	for i := range ordered {
		for j := range ordered {
			got := CompareAPIVersions(ordered[i], ordered[j])
			switch {
			case i < j:
				assert.Negative(t, got, "%s vs %s", ordered[i], ordered[j])
			case i > j:
				assert.Positive(t, got, "%s vs %s", ordered[i], ordered[j])
			default:
				assert.Zero(t, got, "%s vs itself", ordered[i])
			}
		}
	}
}

// TestAPIVersionPatternCoreParity pins apiVersionPattern to core's
// #APIVersionType. The literal below is copied verbatim from
// core/src/types.cue:83 — if core changes the grammar, update both core and
// this package deliberately; this failing test is the drift alarm.
func TestAPIVersionPatternCoreParity(t *testing.T) {
	const coreAPIVersionType = `^v[0-9]+((alpha|beta)[0-9]+)?$` // core/src/types.cue:83
	require.Equal(t, coreAPIVersionType, apiVersionPattern)
}

// TestParseRegexAgreesWithPattern asserts the capture-group parse regex
// recognizes exactly the language of core's pattern across a corpus spanning
// every branch of both.
func TestParseRegexAgreesWithPattern(t *testing.T) {
	corpus := []string{
		"v1", "v2", "v10", "v0",
		"v1alpha1", "v1alpha10", "v1beta1", "v2beta3", "v10alpha2",
		"v1alpha", "v1beta", "v1gamma1", "V1", "1.2.0", "", "v", "va1",
		"v1alpha1beta1", "v1 ", " v1", "v-1", "v1.1",
	}
	for _, s := range corpus {
		assert.Equal(t, patternRE.MatchString(s), parseRE.MatchString(s), "disagreement on %q", s)
	}
}
