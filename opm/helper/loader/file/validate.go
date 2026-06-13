package file

import (
	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/helper/loader/internal/shape"
)

// The module shape gate is single-sourced in opm/helper/loader/internal/shape
// so that loader/file and loader/registry validate artifacts identically. The
// sentinels are re-exported here with unchanged identity: existing
// errors.Is(err, file.ErrWrongKind) callers keep working because these are the
// same error values the shared gate wraps. See [shape] for the gate itself.
var (
	// ErrInvalidPackage marks a structurally invalid package: the built root
	// is not a struct, or load.Instances did not resolve exactly one instance.
	ErrInvalidPackage = shape.ErrInvalidPackage

	// ErrWrongKind marks a package whose concrete kind does not match the
	// artifact the loader was asked for.
	ErrWrongKind = shape.ErrWrongKind

	// ErrMissingRequiredField marks a package missing a required identity
	// field, or carrying it in non-concrete form.
	ErrMissingRequiredField = shape.ErrMissingRequiredField
)

// shape-gate specs for this package's three loaders, aliased to the shared
// definitions so the gate stays single-sourced.
var (
	moduleSpec   = shape.ModuleSpec
	releaseSpec  = shape.ReleaseSpec
	platformSpec = shape.PlatformSpec
)

// shapeGate runs the shared structural validation against a freshly built
// artifact value. It is a thin alias over [shape.Gate] kept at the package
// boundary so the file loaders (and their tests) reference a local symbol.
func shapeGate(val cue.Value, spec shape.ArtifactSpec) error {
	return shape.Gate(val, spec)
}
