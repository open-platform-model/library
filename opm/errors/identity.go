package errors

import (
	"fmt"
)

// IdentityError reports a mismatch between an artifact's declared identity
// and the coordinate it was fetched by (0010:D11), including the version
// clause (0010:D9): the kernel is the version label's verifier, never its source.
// It is emitted at the one library read site that holds both a fetched
// coordinate and decoded metadata: module acquire
// (the kernel's registry acquisition) returns a *IdentityError bare, so
// frontends route on it via [errors.As] with a *IdentityError target, as for
// every other typed error of this package.
//
// Transition: IdentityError is the one type of this package that still has
// a value receiver, so the value type is an error too, and a value target
// (errors.As with an IdentityError variable as target, or
// errors.AsType[IdentityError]) still matches what the library returns,
// through [IdentityError.As].
//
// The value forms are deprecated (the type itself is not, so the notice sits
// on [IdentityError.As]): build the error as &IdentityError{...} and match
// it with a *IdentityError target. A later release gives the type a pointer
// receiver and removes the As method; the value type then no longer
// implements error.
//
// A platform's catalog builds are verified structurally by core
// instead: the registry key binds to the embedded catalog's modulePath
// (0019:D5), so no catalog read site produces it.
type IdentityError struct {
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

// Error names both values, the declared identity and the fetched coordinate
// (0010:D11). The condition is a comparison, not a wrapped failure, so there
// is no Cause and no Unwrap. The receiver is a value until the consumers have
// moved to the pointer forms (see the type's Deprecated note).
func (e IdentityError) Error() string {
	return fmt.Sprintf("identity mismatch at %s: metadata declares %q but the artifact was fetched as %q (%s)",
		e.Field, e.Declared, e.Fetched, e.Coordinate)
}

// As lets both forms match during the transition: a *IdentityError in an
// error chain fills a value target, and an IdentityError value in a chain
// fills a pointer target. [errors.As] and errors.AsType call it only when the
// error's own type is not the target's type.
//
// Deprecated: it exists only for the value forms of IdentityError (a value
// built as an error, a value target of errors.As, errors.AsType[IdentityError])
// and goes with them. Use &IdentityError{...} and a *IdentityError target.
func (e IdentityError) As(target any) bool {
	switch t := target.(type) {
	case *IdentityError:
		*t = e
		return true
	case **IdentityError:
		*t = &e
		return true
	default:
		return false
	}
}
