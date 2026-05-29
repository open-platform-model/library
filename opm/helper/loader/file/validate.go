package file

import (
	"errors"
	"fmt"

	"cuelang.org/go/cue"
)

// Sentinel errors returned by the package loaders' shape gate. Each
// Load*Package wraps the relevant sentinel via %w so frontends (CLI,
// controller, Crossplane function) can branch on the failure class with
// errors.Is rather than string matching.
var (
	// ErrInvalidPackage marks a structurally invalid package: the built root
	// is not a struct, or load.Instances did not resolve exactly one instance.
	ErrInvalidPackage = errors.New("invalid OPM package")

	// ErrWrongKind marks a package whose concrete kind does not match the
	// artifact the loader was asked for.
	ErrWrongKind = errors.New("wrong OPM artifact kind")

	// ErrMissingRequiredField marks a package missing a required identity
	// field, or carrying it in non-concrete form.
	ErrMissingRequiredField = errors.New("missing required field")
)

// artifactSpec describes the shape gate for one artifact type. expectedKind is
// the concrete kind literal the package must carry; requiredConcreteFields are
// dotted paths to scalar identity fields that must be present and concrete;
// moduleRefs point at embedded #Module values whose kind must in turn be
// "Module".
type artifactSpec struct {
	expectedKind           string
	requiredConcreteFields []string
	moduleRefs             []moduleRef
}

// moduleRef locates an embedded #Module value within an artifact: path points
// directly at a #Module value whose kind must be "Module" (the
// #ModuleRelease.#module shape).
type moduleRef struct {
	path string
}

// moduleSpec, releaseSpec, and platformSpec are the shape-gate definitions for
// the three package loaders. The required field lists carry only the identity
// fields the schema never defaults — fields the schema fills in (or leaves as
// open `_`) are out of scope here and validated by the Kernel/Binding layer.
var (
	moduleSpec = artifactSpec{
		expectedKind:           "Module",
		requiredConcreteFields: []string{"metadata.name", "metadata.modulePath", "metadata.version"},
	}

	releaseSpec = artifactSpec{
		expectedKind:           "ModuleRelease",
		requiredConcreteFields: []string{"metadata.name", "metadata.namespace"},
		moduleRefs:             []moduleRef{{path: "#module"}},
	}

	// #Platform.#registry carries path-keyed #Subscription values (enhancement
	// 0001), not embedded #Module registrations — so there is no per-entry
	// #module to gate here. Subscription resolution is the materialize layer's
	// contract; the loader only checks the platform's own identity.
	platformSpec = artifactSpec{
		expectedKind:           "Platform",
		requiredConcreteFields: []string{"metadata.name", "type"},
	}
)

// shapeGate runs the structural validation described by spec against a freshly
// built artifact value. It is the loader boundary's fast-fail check: it
// confirms the artifact is the right kind and carries concrete identity, but
// deliberately stops short of full schema validation, which is the
// Kernel/Binding layer's contract.
func shapeGate(val cue.Value, spec artifactSpec) error {
	if val.IncompleteKind() != cue.StructKind {
		return fmt.Errorf("package root is %s, not a struct: %w", val.IncompleteKind(), ErrInvalidPackage)
	}

	if err := checkKind(val, spec.expectedKind); err != nil {
		return err
	}

	for _, path := range spec.requiredConcreteFields {
		if err := requireConcrete(val, path); err != nil {
			return err
		}
	}

	for _, ref := range spec.moduleRefs {
		if err := checkModuleRef(val, ref); err != nil {
			return err
		}
	}

	return nil
}

// checkKind asserts val carries a concrete string kind field equal to want.
// A missing or non-string kind is treated as a wrong-kind failure: the package
// is not the artifact the loader was asked for.
func checkKind(val cue.Value, want string) error {
	got := val.LookupPath(cue.ParsePath("kind"))
	if !got.Exists() {
		return fmt.Errorf("expected kind %q, found no kind field: %w", want, ErrWrongKind)
	}
	s, err := got.String()
	if err != nil {
		return fmt.Errorf("expected kind %q, kind is not a concrete string: %w", want, ErrWrongKind)
	}
	if s != want {
		return fmt.Errorf("expected kind %q, got %q: %w", want, s, ErrWrongKind)
	}
	return nil
}

// requireConcrete asserts the field at path exists and is concrete. String
// fields must additionally be non-empty — an empty identity string is as
// useless to downstream code as an absent one.
func requireConcrete(val cue.Value, path string) error {
	f := val.LookupPath(cue.ParsePath(path))
	if !f.Exists() {
		return fmt.Errorf("required field %q is absent: %w", path, ErrMissingRequiredField)
	}
	if !f.IsConcrete() {
		return fmt.Errorf("required field %q is not concrete: %w", path, ErrMissingRequiredField)
	}
	if f.Kind() == cue.StringKind {
		s, err := f.String()
		if err != nil {
			return fmt.Errorf("required field %q: %w", path, ErrMissingRequiredField)
		}
		if s == "" {
			return fmt.Errorf("required field %q is empty: %w", path, ErrMissingRequiredField)
		}
	}
	return nil
}

// checkModuleRef asserts that the #Module value located by ref carries kind
// "Module". An absent ref is a missing-field failure.
func checkModuleRef(val cue.Value, ref moduleRef) error {
	target := val.LookupPath(cue.ParsePath(ref.path))
	if !target.Exists() {
		return fmt.Errorf("required field %q is absent: %w", ref.path, ErrMissingRequiredField)
	}
	if err := checkKind(target, "Module"); err != nil {
		return fmt.Errorf("%s: %w", ref.path, err)
	}
	return nil
}
