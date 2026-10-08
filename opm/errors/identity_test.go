package errors

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

func TestIdentityErrorMessageNamesBothValues(t *testing.T) {
	err := &IdentityError{
		Field:      "version",
		Declared:   "1.0.0",
		Fetched:    "1.2.0",
		Coordinate: "example.com/modules/hello@v0 v1.2.0",
	}
	msg := err.Error()
	for _, want := range []string{"identity mismatch", "version", `"1.0.0"`, `"1.2.0"`, "example.com/modules/hello@v0 v1.2.0"} {
		if !strings.Contains(msg, want) {
			t.Errorf("Error() = %q, missing %q", msg, want)
		}
	}
}

// TestIdentityErrorAsThroughFmtWrap pins the acquire-site path: the loader
// returns the error wrapped with %w context.
func TestIdentityErrorAsThroughFmtWrap(t *testing.T) {
	inner := &IdentityError{Field: "path", Declared: "a@v0", Fetched: "b@v0", Coordinate: "b@v0 v0.1.0"}
	wrapped := fmt.Errorf("validating module package: %w", inner)

	var got *IdentityError
	if !errors.As(wrapped, &got) {
		t.Fatalf("errors.As(fmt-wrapped, *IdentityError) = false, want true")
	}
	if got != inner {
		t.Errorf("errors.As extracted %+v, want %+v", got, inner)
	}
}

// TestTypedErrorsUsePointerReceivers pins the one receiver shape of the
// package: every Error method is declared on a pointer receiver, so a caller
// matches every typed error with errors.As and a pointer target, and the
// value type is not an error at all.
func TestTypedErrorsUsePointerReceivers(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("listing the package: %v", err)
	}
	fset := token.NewFileSet()
	seen := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", name, err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Name.Name != "Error" {
				continue
			}
			seen++
			if _, isPtr := fn.Recv.List[0].Type.(*ast.StarExpr); !isPtr {
				t.Errorf("%s: Error is declared on a value receiver; every typed error uses a pointer receiver",
					fset.Position(fn.Pos()))
			}
		}
	}
	if seen == 0 {
		t.Fatal("found no Error method; the scan is broken")
	}
}

// TestConfigValidationErrorWrapsWithoutRewording pins the marker: the text
// is the wrapped error's own, and the wrapped error stays reachable.
func TestConfigValidationErrorWrapsWithoutRewording(t *testing.T) {
	cause := errors.New("replicas: invalid value -1")
	err := fmt.Errorf("module %q: %w", "demo", &ConfigValidationError{Err: cause})

	var got *ConfigValidationError
	if !errors.As(err, &got) {
		t.Fatalf("errors.As(wrapped, **ConfigValidationError) = false, want true")
	}
	if got.Error() != cause.Error() {
		t.Errorf("Error() = %q, want the wrapped text %q", got.Error(), cause.Error())
	}
	if !errors.Is(err, cause) {
		t.Errorf("errors.Is(wrapped, cause) = false, want true")
	}
}
