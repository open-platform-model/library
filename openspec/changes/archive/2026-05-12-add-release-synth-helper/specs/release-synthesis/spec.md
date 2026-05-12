## ADDED Requirements

### Requirement: Synth Helper Package Location

The library SHALL expose a `pkg/helper/synth/` subpackage that produces OPM artifact CUE values from in-memory typed inputs. The package SHALL be a peer of `pkg/helper/loader/` (not nested under it) because synthesis is creation from typed inputs rather than parsing from byte streams. The package SHALL document the boundary in its `doc.go`.

#### Scenario: Synth package present at canonical path

- **WHEN** a developer reads `pkg/helper/synth/doc.go`
- **THEN** the file documents that the package builds artifact CUE values from typed inputs
- **AND** documents that the package is a peer of `pkg/helper/loader/`, not nested under it

### Requirement: synth.Release function signature

The `pkg/helper/synth/` package SHALL expose a function `Release(ctx *cue.Context, in ReleaseInput) (cue.Value, error)` that returns a `#ModuleRelease` artifact CUE value built by unifying the input fields against the embedded `#ModuleRelease` schema definition for the input module's API version.

The `ReleaseInput` struct SHALL carry: `Module *module.Module` (required), `Name string` (required), `Namespace string` (required), `Values cue.Value` (optional; zero value means "no values supplied"), `Labels map[string]string` (optional), `Annotations map[string]string` (optional).

#### Scenario: Required inputs validated

- **WHEN** `synth.Release` is called with `Module == nil`, or `Name == ""`, or `Namespace == ""`
- **THEN** it returns the zero `cue.Value` and a non-nil error naming the missing field

#### Scenario: Returned value is schema-unified

- **WHEN** `synth.Release` is called with valid inputs against the v1alpha2 binding
- **THEN** the returned `cue.Value` carries `apiVersion == "opmodel.dev/v1alpha2"` and `kind == "ModuleRelease"` at its root
- **AND** the value is unified with the schema's `#ModuleRelease` definition (the schema's structural constraints apply)

### Requirement: Derived fields come from schema unification

`synth.Release` SHALL NOT compute `metadata.uuid`, `components`, or schema-stamped labels in Go. These fields SHALL flow from CUE evaluation as a consequence of unifying the inputs with `#ModuleRelease`. The Go code SHALL fill only the caller-supplied fields (name, namespace, `#module`, optional values/labels/annotations) and let CUE derive the rest.

#### Scenario: UUID is computed by CUE

- **WHEN** `synth.Release` is called twice with the same `(Module, Name, Namespace)`
- **THEN** the returned CUE values carry identical `metadata.uuid` strings
- **AND** the UUID equals `uuid.SHA1(OPMNamespace, "<module.uuid>:<name>:<namespace>")` per the schema definition at `apis/core/v1alpha2/module_release.cue:19`

#### Scenario: UUID diverges with namespace

- **WHEN** `synth.Release` is called with identical `(Module, Name)` but two different `Namespace` values
- **THEN** the returned CUE values carry different `metadata.uuid` strings

#### Scenario: Components are fanned by schema comprehension

- **WHEN** `synth.Release` is called with a Module declaring N concrete components in `#components`
- **THEN** the returned CUE value's `components` field contains exactly those N entries with the schema-applied projections
- **AND** the synth helper itself does not enumerate `#components` in Go

#### Scenario: Auto-secrets component included when module has #Secret instances

- **WHEN** `synth.Release` is called with a Module whose `#config` (after `Values` is filled) contains at least one `#Secret` instance
- **THEN** the returned CUE value's `components` field contains an `opm-secrets` entry
- **AND** when the module contains no `#Secret` instances, no `opm-secrets` entry appears

#### Scenario: Standard release labels are stamped by schema

- **WHEN** `synth.Release` is called with valid inputs
- **THEN** the returned CUE value's `metadata.labels` contains keys `module-release.opmodel.dev/name` and `module-release.opmodel.dev/uuid`
- **AND** their values equal the release name and the derived UUID respectively

### Requirement: Values field is caller-supplied with no implicit fallback

`synth.Release` SHALL NOT consult `Module.debugValues` or any other implicit source when `ReleaseInput.Values` is the zero `cue.Value`. When `Values.Exists()` is true, the helper SHALL fill the schema's values path with it. When `Values.Exists()` is false, the helper SHALL leave the values path unfilled and return the unified value as-is; concreteness enforcement is deferred to downstream processing.

#### Scenario: Caller-supplied values are filled

- **WHEN** `synth.Release` is called with `Values` set to a concrete CUE value satisfying the module's `#config`
- **THEN** the returned CUE value carries those values at the schema's values path
- **AND** the `components` comprehension sees the unified `#config := values` configuration

#### Scenario: Zero Values is not replaced by debugValues

- **WHEN** `synth.Release` is called with `Values == cue.Value{}` against a Module that defines `debugValues`
- **THEN** the returned CUE value's values path is unfilled (does not equal `debugValues`)

### Requirement: Optional labels and annotations are filled into release metadata

When `ReleaseInput.Labels` is non-empty, `synth.Release` SHALL fill `metadata.labels` with those entries. When `ReleaseInput.Annotations` is non-empty, `synth.Release` SHALL fill `metadata.annotations` with those entries. The schema's label stamping (Requirement: Derived fields come from schema unification) unifies with caller-supplied labels — caller labels MUST NOT be allowed to remove schema-stamped labels.

#### Scenario: Caller labels merged with schema-stamped labels

- **WHEN** `synth.Release` is called with `Labels == {"env": "prod"}`
- **THEN** the returned CUE value's `metadata.labels` contains the `env: prod` entry
- **AND** still contains `module-release.opmodel.dev/name` and `module-release.opmodel.dev/uuid`

#### Scenario: Annotations are passed through unchanged

- **WHEN** `synth.Release` is called with `Annotations == {"opmodel.dev/owner": "team-x"}`
- **THEN** the returned CUE value's `metadata.annotations` contains that entry

### Requirement: Schema obtained through binding

`synth.Release` SHALL obtain the `#ModuleRelease` definition by calling `binding.SchemaValue(ctx)` on the binding looked up for `Module.APIVersion`, then `LookupPath("#ModuleRelease")` on the returned value. The helper MUST NOT call `load.Instances` directly, MUST NOT consult `os.Getenv("CUE_REGISTRY")`, and MUST NOT read from the filesystem.

#### Scenario: No registry access during synthesis

- **WHEN** `synth.Release` is called in an environment with `CUE_REGISTRY` unset and no network access
- **THEN** the call succeeds (provided the binding for the module's API version is registered)

#### Scenario: Schema load failure surfaces as a wrapped error

- **WHEN** `binding.SchemaValue(ctx)` returns a non-nil error
- **THEN** `synth.Release` returns the zero `cue.Value` and an error wrapping the binding's error

### Requirement: synth.Release does not validate or enforce concreteness

`synth.Release` SHALL return the unified CUE value without invoking `cue.Concrete` validation. Validation of values against `#config` and concreteness enforcement on the final spec are downstream responsibilities (the kernel wrapper handles both via `Kernel.ProcessModuleRelease`). Errors from CUE during unification (e.g. type mismatch between caller-supplied labels and the schema's label-map type) SHALL be returned as the result of `cue.Value.Err()` on the returned value, surfaced to the caller through the returned `error`.

#### Scenario: Unification error returned

- **WHEN** `synth.Release` is called with inputs that conflict with the schema (e.g. `Name` containing characters disallowed by `#NameType`)
- **THEN** the returned error is non-nil and the returned `cue.Value` is the zero value or carries the unification error

#### Scenario: No concreteness check at synth time

- **WHEN** `synth.Release` is called with `Values == cue.Value{}` against a `#config` that has no defaults
- **THEN** the call succeeds (returns a non-zero `cue.Value` and a nil error)
- **AND** the returned value's values path is unfilled rather than concrete
