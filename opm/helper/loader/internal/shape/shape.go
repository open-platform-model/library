// Package shape is the single source of the OPM artifact shape gate shared by
// the package loaders (opm/helper/loader/file and opm/helper/loader/registry).
//
// The gate is the loader boundary's fast-fail structural check: it confirms an
// artifact carries the right concrete kind and the identity fields the schema
// never defaults, but deliberately stops short of full schema validation, which
// is the Kernel/Binding layer's contract. "Concrete" is judged before default
// finalization: an identity field authored as a defaulted disjunction is
// refused, with the default named in the error. Single-sourcing it here
// guarantees a directory-loaded artifact and a registry-loaded artifact are
// validated identically and fail with the same sentinel values.
//
// It lives under opm/helper/loader/internal/ so it stays out of the library's
// public SemVer surface (kernel neutrality) while remaining importable by both
// loader subpackages. The sentinels are re-exported from loader/file with
// unchanged identity so existing errors.Is callers are unaffected.
package shape

import (
	"errors"
	"fmt"

	"cuelang.org/go/cue"
)

// Sentinel errors returned by the shape gate. Each Load*Package wraps the
// relevant sentinel via %w so frontends (CLI, controller, Crossplane function)
// can branch on the failure class with errors.Is rather than string matching.
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

// ArtifactSpec describes the shape gate for one artifact type. ExpectedKind is
// the concrete kind literal the package must carry; RequiredConcreteFields are
// dotted paths to scalar identity fields that must be present and concrete;
// ModuleRefs point at embedded #Module values whose kind must in turn be
// "Module".
type ArtifactSpec struct {
	ExpectedKind           string
	RequiredConcreteFields []string
	ModuleRefs             []ModuleRef

	// CompleteEntryMaps lists paths to maps whose every entry must be
	// complete: each regular field of each entry validates under
	// cue.Concrete(true). An absent map passes. Used for #Platform.#registry,
	// where core derives an entry's `version` from the catalog the entry
	// embeds (enhancement 0019 D5), so an entry with no embedded catalog is
	// incomplete exactly where the catalog would have completed it.
	CompleteEntryMaps []string
}

// ModuleRef locates an embedded #Module value within an artifact: Path points
// directly at a #Module value whose kind must be "Module" (the
// #ModuleInstance.#module shape).
type ModuleRef struct {
	Path string
}

// ModuleSpec, InstanceSpec, and PlatformSpec are the shape-gate definitions for
// the three package loaders. The required field lists carry only the identity
// fields the schema never defaults — fields the schema fills in (or leaves as
// open `_`) are out of scope here and validated by the Kernel/Binding layer.
var (
	ModuleSpec = ArtifactSpec{
		ExpectedKind:           "Module",
		RequiredConcreteFields: []string{"metadata.name", "metadata.modulePath", "metadata.version"},
	}

	InstanceSpec = ArtifactSpec{
		ExpectedKind:           "ModuleInstance",
		RequiredConcreteFields: []string{"metadata.name", "metadata.namespace"},
		ModuleRefs:             []ModuleRef{{Path: "#module"}},
	}

	// #Platform.#registry carries path-keyed #CatalogEntry values, each
	// embedding its catalog by import (enhancement 0019 D5). Core derives the
	// entry's `version` from the embedded catalog's stamped metadata, so an
	// entry that names no catalog (the retired subscription shape: a
	// `version` scalar and nothing else) is refused here as a missing
	// required field naming the entry. #registry is a definition, so no
	// root-level validation reaches it; the gate walks it explicitly.
	PlatformSpec = ArtifactSpec{
		ExpectedKind:           "Platform",
		RequiredConcreteFields: []string{"metadata.name", "type"},
		CompleteEntryMaps:      []string{"#registry"},
	}
)

// Gate runs the structural validation described by spec against a freshly built
// artifact value. It is the loader boundary's fast-fail check: it confirms the
// artifact is the right kind and carries concrete identity, but deliberately
// stops short of full schema validation, which is the Kernel/Binding layer's
// contract.
func Gate(val cue.Value, spec ArtifactSpec) error {
	if val.IncompleteKind() != cue.StructKind {
		return fmt.Errorf("package root is %s, not a struct: %w", val.IncompleteKind(), ErrInvalidPackage)
	}

	if err := checkKind(val, spec.ExpectedKind); err != nil {
		return err
	}

	for _, path := range spec.RequiredConcreteFields {
		if err := requireConcrete(val, path); err != nil {
			return err
		}
	}

	for _, ref := range spec.ModuleRefs {
		if err := checkModuleRef(val, ref); err != nil {
			return err
		}
	}

	for _, path := range spec.CompleteEntryMaps {
		if err := requireCompleteEntries(val, path); err != nil {
			return err
		}
	}

	return nil
}

// requireCompleteEntries asserts that every entry of the map at path (a
// struct keyed by string) has every regular field complete under
// cue.Concrete(true). Definitions and hidden fields inside an entry are not
// visited: for a #Platform.#registry entry that is the embedded #catalog,
// whose transformer bodies are open by design. An absent map passes; a map
// whose entries cannot be iterated is a missing-field failure.
func requireCompleteEntries(val cue.Value, path string) error {
	m := val.LookupPath(cue.ParsePath(path))
	if !m.Exists() {
		return nil
	}
	entries, err := m.Fields()
	if err != nil {
		return fmt.Errorf("required field %q: %v: %w", path, err, ErrMissingRequiredField)
	}
	for entries.Next() {
		key := entries.Selector().Unquoted()
		fields, err := entries.Value().Fields()
		if err != nil {
			return fmt.Errorf("required field %q entry %q: %v: %w", path, key, err, ErrMissingRequiredField)
		}
		for fields.Next() {
			if err := fields.Value().Validate(cue.Concrete(true)); err != nil {
				return fmt.Errorf("required field %q entry %q is incomplete at %q (an embedded catalog supplies it): %v: %w",
					path, key, fields.Selector().Unquoted(), err, ErrMissingRequiredField)
			}
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
//
// Concreteness is judged on the value as authored, before default
// finalization: a disjunction with a default (`#VersionType | *"1.0.1"`) is
// NOT concrete here even though cue.Value.String and `cue eval` would resolve
// it. A default arm is a suggestion a consumer may unify away, not the value a
// release moved, and the identity fields this gate guards are exactly those
// values. That case gets its own message naming the default, because the
// generic "not concrete" points the author at the reference in module.cue
// rather than at the declaration in identity/identity.cue.
func requireConcrete(val cue.Value, path string) error {
	f := val.LookupPath(cue.ParsePath(path))
	if !f.Exists() {
		return fmt.Errorf("required field %q is absent: %w", path, ErrMissingRequiredField)
	}
	if !f.IsConcrete() {
		if d, ok := f.Default(); ok {
			return fmt.Errorf("required field %q is a defaulted disjunction (default %v), not a concrete value: identity fields must be concrete literals: %w", path, d, ErrMissingRequiredField)
		}
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
func checkModuleRef(val cue.Value, ref ModuleRef) error {
	target := val.LookupPath(cue.ParsePath(ref.Path))
	if !target.Exists() {
		return fmt.Errorf("required field %q is absent: %w", ref.Path, ErrMissingRequiredField)
	}
	if err := checkKind(target, "Module"); err != nil {
		return fmt.Errorf("%s: %w", ref.Path, err)
	}
	return nil
}
