// Package apiversion provides the OPM schema version type and detection.
//
// A Version is the literal string carried in the apiVersion field of every OPM
// artifact root (#Module, #ModuleRelease, #Provider, #Component). Detect reads
// that field from any cue.Value root and returns the matching Version constant.
//
// The package is dependency-light by design: it imports cuelang.org/go/cue and
// nothing else from this module. Higher-level dispatch and the per-version
// Binding contract live in opm/api.
package apiversion

import (
	"errors"
	"fmt"

	"cuelang.org/go/cue"
)

// Version is the canonical apiVersion literal of an OPM schema.
type Version string

// Known versions. Add a new constant here when introducing a new schema package
// under apis/core/<vN>/ and a corresponding opm/api/<vN>/ binding.
const (
	V1alpha2 Version = "opmodel.dev/v1alpha2"
)

// ErrUnknownAPIVersion is returned when an artifact's apiVersion is missing,
// not a string, or not a literal the kernel recognises.
var ErrUnknownAPIVersion = errors.New("unknown OPM apiVersion")

// known is the set of versions Detect will accept. Kept as a map so future
// versions can be added with a single entry.
var known = map[Version]struct{}{
	V1alpha2: {},
}

// Detect reads the apiVersion field from the root of the given cue.Value and
// returns the matching Version. It does not mutate v and is safe to call
// concurrently. Failures (missing field, wrong kind, unrecognised literal) all
// return errors that wrap ErrUnknownAPIVersion.
func Detect(v cue.Value) (Version, error) {
	field := v.LookupPath(cue.ParsePath("apiVersion"))
	if !field.Exists() {
		return "", fmt.Errorf("apiVersion field missing: %w", ErrUnknownAPIVersion)
	}
	if field.Kind() != cue.StringKind {
		return "", fmt.Errorf("apiVersion is not a string (kind=%s): %w", field.Kind(), ErrUnknownAPIVersion)
	}
	s, err := field.String()
	if err != nil {
		return "", fmt.Errorf("reading apiVersion: %w", errors.Join(err, ErrUnknownAPIVersion))
	}
	ver := Version(s)
	if _, ok := known[ver]; !ok {
		return "", fmt.Errorf("apiVersion %q not recognised: %w", s, ErrUnknownAPIVersion)
	}
	return ver, nil
}
