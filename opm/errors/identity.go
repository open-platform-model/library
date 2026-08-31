package errors

import (
	"fmt"
)

// IdentityError reports a mismatch between an artifact's declared identity
// and the coordinate it was fetched by (0010 D11), including the version
// clause (D9): the kernel is the version label's verifier, never its source.
// It is emitted at the two library read sites that hold both a fetched
// coordinate and decoded metadata — module acquire
// (opm/helper/loader/registry) returns it bare; catalog materialize
// (opm/materialize) wraps it in a *MaterializeError (Kind "catalog") as
// Cause, so frontends route on it via [errors.As] either way.
type IdentityError struct {
	// Artifact discriminates the read site: "module" | "catalog".
	Artifact string

	// Field names the mismatched identity field: "path" | "version".
	Field string

	// Declared is the value the artifact's metadata claims.
	Declared string

	// Fetched is the coordinate/tag the artifact was actually fetched by.
	Fetched string

	// Coordinate is the full fetched coordinate for context,
	// e.g. "opmodel.dev/catalogs/opm@v4 v4.0.1".
	Coordinate string
}

// Error names both values (D11: "a typed error naming both"). Value receiver:
// the condition is a comparison, not a wrapped failure, so there is no Cause
// and no Unwrap.
func (e IdentityError) Error() string {
	return fmt.Sprintf("%s identity mismatch at %s: metadata declares %q but the artifact was fetched as %q (%s)",
		e.Artifact, e.Field, e.Declared, e.Fetched, e.Coordinate)
}
