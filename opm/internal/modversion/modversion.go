// Package modversion is the library's one home for CUE module version
// helpers: the rules for turning a version as a caller or subscription writes
// it into the form CUE's module machinery requires, the major qualifier of a
// version, SemVer validity and precedence, and the constants for the core
// module path and the CUE language floor. It is the only package under opm/
// that imports golang.org/x/mod/semver, the SemVer implementation CUE's own
// module code uses, so the library compares versions one way.
package modversion

import (
	"fmt"
	"strings"

	"golang.org/x/mod/semver"
)

const (
	// CoreModule is the module path of the OPM core schema, without a major
	// qualifier.
	CoreModule = "opmodel.dev/core"

	// CorePath is the major-qualified core path the v2 kernel builds against.
	CorePath = CoreModule + "@v2"

	// LanguageFloor is the lowest CUE language version a generated or render
	// module declares: v0.17.0 introduced cue.mod/local-module.cue.
	LanguageFloor = "v0.17.0"
)

// Canonical returns v with the "v" prefix CUE module versions require. A bare
// SemVer ("1.0.0") gains it; an already-prefixed or empty version is returned
// unchanged. It validates nothing: module.NewVersion remains the validator.
func Canonical(v string) string {
	if v == "" || strings.HasPrefix(v, "v") {
		return v
	}
	return "v" + v
}

// Bare returns v without the "v" prefix: the spelling a catalog's
// metadata.version carries and a platform registry entry stamps. It is the
// inverse of [Canonical] and validates nothing either.
func Bare(v string) string {
	return strings.TrimPrefix(v, "v")
}

// Major returns the "@vN" qualifier of a version in either spelling:
// "0.1.0" -> "v0", "v2.0.0-beta.1" -> "v2", "v2" -> "v2". Like [Canonical]
// and [Bare] it validates nothing: an invalid input yields "v" plus whatever
// precedes its first dot, and the module path built from it fails in CUE with
// CUE's own error.
func Major(v string) string {
	major, _, _ := strings.Cut(Bare(v), ".")
	return "v" + major
}

// Valid reports whether v is a SemVer version in either spelling. The
// shorthand forms "vMAJOR" and "vMAJOR.MINOR" count as valid, as they do for
// golang.org/x/mod/semver.
func Valid(v string) bool {
	return semver.IsValid(Canonical(v))
}

// Compare orders a and b by SemVer precedence, returning -1, 0 or +1 and
// accepting either spelling on each side. Prerelease builds order by SemVer 2
// precedence ("2.0.0-beta.2" < "2.0.0-beta.10" < "2.0.0") and build metadata
// is ignored. An invalid input is refused as `invalid version "<as written>"`.
func Compare(a, b string) (int, error) {
	for _, v := range []string{a, b} {
		if !Valid(v) {
			return 0, fmt.Errorf("invalid version %q", v)
		}
	}
	return semver.Compare(Canonical(a), Canonical(b)), nil
}
