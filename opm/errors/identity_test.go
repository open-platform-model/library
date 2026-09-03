package errors

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestIdentityErrorMessageNamesBothValues(t *testing.T) {
	err := IdentityError{
		Artifact:   "module",
		Field:      "version",
		Declared:   "1.0.0",
		Fetched:    "1.2.0",
		Coordinate: "example.com/modules/hello@v0 v1.2.0",
	}
	msg := err.Error()
	for _, want := range []string{"module", "version", `"1.0.0"`, `"1.2.0"`, "example.com/modules/hello@v0 v1.2.0"} {
		if !strings.Contains(msg, want) {
			t.Errorf("Error() = %q, missing %q", msg, want)
		}
	}
}

// TestIdentityErrorAsThroughFmtWrap pins the acquire-site path: the loader
// returns the error wrapped with %w context.
func TestIdentityErrorAsThroughFmtWrap(t *testing.T) {
	inner := IdentityError{Artifact: "module", Field: "path", Declared: "a@v0", Fetched: "b@v0", Coordinate: "b@v0 v0.1.0"}
	wrapped := fmt.Errorf("validating module package: %w", inner)

	var got IdentityError
	if !errors.As(wrapped, &got) {
		t.Fatalf("errors.As(fmt-wrapped, *IdentityError) = false, want true")
	}
	if got != inner {
		t.Errorf("errors.As extracted %+v, want %+v", got, inner)
	}
}
