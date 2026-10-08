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

// TestIdentityErrorMatchesBothForms pins the transition contract: the
// library returns a *IdentityError, and both the pointer target (the form
// every typed error uses) and the deprecated value target match it, through
// a %w wrap, with errors.As and with errors.AsType. A value in the chain,
// which a consumer's test may still build, matches both forms too.
func TestIdentityErrorMatchesBothForms(t *testing.T) {
	want := IdentityError{Field: "path", Declared: "a@v0", Fetched: "b@v0", Coordinate: "b@v0 v0.1.0"}
	ptr := want
	for name, inChain := range map[string]error{"pointer in the chain": &ptr, "value in the chain": want} {
		t.Run(name, func(t *testing.T) {
			wrapped := fmt.Errorf("acquiring module: %w", inChain)

			var asPtr *IdentityError
			if !errors.As(wrapped, &asPtr) || *asPtr != want {
				t.Errorf("errors.As with a *IdentityError target: got %+v, want %+v", asPtr, want)
			}
			var asVal IdentityError
			if !errors.As(wrapped, &asVal) || asVal != want {
				t.Errorf("errors.As with a deprecated IdentityError target: got %+v, want %+v", asVal, want)
			}
			if got, ok := errors.AsType[*IdentityError](wrapped); !ok || *got != want {
				t.Errorf("errors.AsType[*IdentityError]: got %+v, %v", got, ok)
			}
			if got, ok := errors.AsType[IdentityError](wrapped); !ok || got != want {
				t.Errorf("errors.AsType[IdentityError] (deprecated): got %+v, %v", got, ok)
			}
			var other *FetchError
			if errors.As(wrapped, &other) {
				t.Errorf("an identity error matched a *FetchError target")
			}
		})
	}
}

// TestTypedErrorsUsePointerReceivers pins the one receiver shape of the
// package: every Error method is declared on a pointer receiver, so a caller
// matches every typed error with errors.As and a pointer target, and the
// value type is not an error at all. valueReceiverUntilConsumersMigrate
// lists the one exception and why it stands.
func TestTypedErrorsUsePointerReceivers(t *testing.T) {
	// IdentityError keeps its value receiver while opm-operator at main still
	// builds it as a value and matches it with a value target. The later
	// change that gives it a pointer receiver empties this list.
	valueReceiverUntilConsumersMigrate := map[string]bool{"IdentityError": true}

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("listing the package: %v", err)
	}
	fset := token.NewFileSet()
	seen := 0
	sawException := map[string]bool{}
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
			if ident, isValue := fn.Recv.List[0].Type.(*ast.Ident); isValue && valueReceiverUntilConsumersMigrate[ident.Name] {
				sawException[ident.Name] = true
				continue
			}
			if _, isPtr := fn.Recv.List[0].Type.(*ast.StarExpr); !isPtr {
				t.Errorf("%s: Error is declared on a value receiver; every typed error uses a pointer receiver",
					fset.Position(fn.Pos()))
			}
		}
	}
	if seen == 0 {
		t.Fatal("found no Error method; the scan is broken")
	}
	for name := range valueReceiverUntilConsumersMigrate {
		if !sawException[name] {
			t.Errorf("%s no longer has a value receiver: remove it from valueReceiverUntilConsumersMigrate", name)
		}
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
