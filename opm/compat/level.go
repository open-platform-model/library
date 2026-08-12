// Package compat implements publish-side catalog compatibility logic:
// enhancement 0010 D27's additive-only comparison walk ([Check],
// [CheckAtLevel]), the D34 contract-level ladder ([Level], [ParseLevel],
// [CompareAPIVersions]), D9 predecessor selection ([HighestStable]), and the
// D30 provenance strip ([StripProvenance]).
//
// The package is pure logic: no I/O, no registry access, no schema-cache
// dependency, no state. Callers compose it with member enumeration, predecessor
// pulling, and gate policy. Its consumers are the 0011 publish gate
// (`opm catalog publish`), `opm catalog registry check --compat` (0011 D7), and
// library-matching's D34/D30 reads (contract-key ordering, unify-rung strip).
package compat

import (
	"cmp"
	"regexp"
	"strconv"
	"strings"
)

// Level is a contract apiVersion's position on the Kubernetes ladder
// (enhancement 0010 D34): vNalphaM → vNbetaM → vN. The level decides whether
// D27's additive-only promise binds — see [Level.Enforced].
type Level int

const (
	LevelAlpha Level = iota
	LevelBeta
	LevelGA
)

func (l Level) String() string {
	switch l {
	case LevelAlpha:
		return "alpha"
	case LevelBeta:
		return "beta"
	}
	return "ga"
}

// apiVersionPattern is the exact grammar of core's #APIVersionType
// (core/src/types.cue:83): a digit is required after alpha/beta, so "v1alpha"
// is rejected. level_test.go pins this string against a literal copy of the
// core source; if core moves, that test fails rather than silently drifting.
const apiVersionPattern = `^v[0-9]+((alpha|beta)[0-9]+)?$`

// parseRE recognizes the same language as apiVersionPattern with capture
// groups for the components; the equivalence is asserted by test.
var (
	patternRE = regexp.MustCompile(apiVersionPattern)
	parseRE   = regexp.MustCompile(`^v([0-9]+)(?:(alpha|beta)([0-9]+))?$`)
)

// ParseLevel classifies a contract's apiVersion on the D34 ladder and reports
// its major. ok is false when the string is outside #APIVersionType's grammar
// ("v1alpha", "V1", "1.2.0"); D34 keys enforcement to the primitive's own
// apiVersion, never to the catalog's release version, which is an independent
// axis.
func ParseLevel(apiVersion string) (major int, l Level, ok bool) {
	if !patternRE.MatchString(apiVersion) {
		return 0, LevelGA, false
	}
	m := parseRE.FindStringSubmatch(apiVersion)
	if m == nil {
		return 0, LevelGA, false
	}
	major, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, LevelGA, false
	}
	switch m[2] {
	case "alpha":
		return major, LevelAlpha, true
	case "beta":
		return major, LevelBeta, true
	}
	return major, LevelGA, true
}

// Enforced reports whether D27's additive-only promise binds at this level
// (D34): beta and GA yes, alpha no — alpha's definition is that it promises
// nothing, so the publish gate is off there.
func (l Level) Enforced() bool { return l != LevelAlpha }

// CompareAPIVersions is a total, transitive ordering over apiVersion strings,
// following the Kubernetes kube-aware ordering: level first (alpha < beta <
// GA), then major, then the alpha/beta number. It returns a negative value if
// a sorts before b, zero if they tie, positive otherwise.
//
// This exists because SemVer cannot order the ladder — measured against
// Masterminds v3, a per-pair rule switch makes v1alpha1 < v2, v2 < v10 and
// v10 < v1alpha1 all true at once (0010 D34). Strings outside the grammar
// sort before every valid one, lexically among themselves; that branch keeps
// the ordering total and is not part of the contract.
func CompareAPIVersions(a, b string) int {
	amaj, alvl, aok := ParseLevel(a)
	bmaj, blvl, bok := ParseLevel(b)
	switch {
	case !aok && !bok:
		return strings.Compare(a, b)
	case !aok:
		return -1
	case !bok:
		return 1
	}
	if alvl != blvl {
		return cmp.Compare(alvl, blvl)
	}
	if amaj != bmaj {
		return cmp.Compare(amaj, bmaj)
	}
	return cmp.Compare(minorOf(a), minorOf(b))
}

// minorOf returns the alpha/beta number of a valid apiVersion, 0 for GA.
func minorOf(apiVersion string) int {
	m := parseRE.FindStringSubmatch(apiVersion)
	if m == nil || m[3] == "" {
		return 0
	}
	n, _ := strconv.Atoi(m[3])
	return n
}
