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
