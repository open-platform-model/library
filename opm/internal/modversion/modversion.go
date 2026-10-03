// Package modversion holds the library's CUE module version helpers: the
// rules for turning a version as a caller or subscription writes it into the
// form CUE's module machinery requires.
package modversion

import "strings"

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
