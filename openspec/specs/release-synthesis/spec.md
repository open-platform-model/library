# release-synthesis Specification

## Purpose

The `opm/helper/synth/` package produces OPM artifact CUE values from in-memory typed inputs. It is the in-memory counterpart to the file-based loader: where `opm/helper/loader/` parses artifacts from byte streams, `synth` synthesizes them by unifying typed Go inputs against the OPM core schema resolved through a caller-supplied `*schema.Cache`. The package exposes `synth.Release` as its first artifact builder, returning a `#ModuleRelease` `cue.Value` ready for downstream validation, concreteness enforcement, and metadata decoding via `Kernel.ProcessModuleRelease`.

## Requirements

### Requirement: Synth Helper Package Location

The library SHALL expose a `opm/helper/synth/` subpackage that produces OPM artifact CUE values from in-memory typed inputs. The package SHALL be a peer of `opm/helper/loader/` (not nested under it) because synthesis is creation from typed inputs rather than parsing from byte streams. The package SHALL document the boundary in its `doc.go`.

#### Scenario: Synth package present at canonical path

- **WHEN** a developer reads `opm/helper/synth/doc.go`
- **THEN** the file documents that the package builds artifact CUE values from typed inputs
- **AND** documents that the package is a peer of `opm/helper/loader/`, not nested under it

### Requirement: synth.Release function signature

The `opm/helper/synth/` package SHALL expose a function `Release(ctx *cue.Context, in ReleaseInput) (cue.Value, error)` that returns a `#ModuleRelease` artifact CUE value built by unifying the input fields against the `#ModuleRelease` schema definition resolved from the supplied `SchemaCache`.

The `ReleaseInput` struct SHALL carry: `Module *module.Module` (required), `Name string` (required), `Namespace string` (required), `SchemaCache *schema.Cache` (REQUIRED), `Values cue.Value` (optional; zero value means "no values supplied"), `Labels map[string]string` (optional), `Annotations map[string]string` (optional).

`synth.Release` MUST return a non-nil error when `SchemaCache == nil` (in addition to the existing required-field checks). The error message MUST name the missing field. The helper MUST NOT self-construct a `*schema.Cache` as a fallback; the caller is responsible for passing the cache it intends to share (typically `k.SchemaCache()` from its Kernel).

#### Scenario: Required inputs validated

- **WHEN** `synth.Release` is called with `Module == nil`, or `Name == ""`, or `Namespace == ""`, or `SchemaCache == nil`
- **THEN** it returns the zero `cue.Value` and a non-nil error naming the missing field

#### Scenario: Returned value is schema-unified

- **WHEN** `synth.Release` is called with valid inputs and a `SchemaCache` whose Loader resolves the OPM core schema
- **THEN** the returned `cue.Value` carries the `#ModuleRelease` shape at its root
- **AND** the value is unified with the schema's `#ModuleRelease` definition (the schema's structural constraints apply)

#### Scenario: Caller's Cache is reused, not replaced

- **WHEN** `synth.Release` is called with a `SchemaCache` that has already been warmed by a prior `Get`
- **THEN** the helper invokes `(*Cache).Get(ctx)` to retrieve the already-cached value
- **AND** no second schema load is triggered by the helper

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

`synth.Release` SHALL NOT consult `Module.debugValues` or any other implicit source when `ReleaseInput.Values` is the zero `cue.Value`. When `Values.Exists()` is true, the helper SHALL render it into the synthesized package's values source (via `format.Node` on the value's syntax, never string-interpolating raw input) so it participates in the single build. When `Values.Exists()` is false, the helper SHALL omit the values source and return the unified value as-is; concreteness enforcement is deferred to `Kernel.ProcessModuleRelease`.

#### Scenario: Caller-supplied values participate in the build

- **WHEN** `synth.Release` is called with `Values` set to a concrete CUE value satisfying the module's `#config`
- **THEN** the returned value carries those values at the schema's values path
- **AND** the values entered the build as a rendered source file, not a post-build cross-build unification

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

### Requirement: Schema obtained through caller-supplied Cache

`synth.Release` SHALL obtain the `#ModuleRelease` definition by calling `in.SchemaCache.Get(ctx)` on the caller-supplied `*schema.Cache`, then `LookupPath("#ModuleRelease")` on the returned value. The helper MUST NOT call `load.Instances` directly, MUST NOT consult `os.Getenv("CUE_REGISTRY")`, MUST NOT read from the filesystem, and MUST NOT construct its own `*schema.Cache` or `Loader`.

#### Scenario: Helper delegates schema loading to the Cache

- **WHEN** `synth.Release` is called with a `SchemaCache` configured against a pre-seeded test cache
- **THEN** the call succeeds without any direct call to `load.Instances`, `os.Getenv`, or filesystem reads originating in `opm/helper/synth/`

#### Scenario: Schema load failure surfaces as a wrapped error

- **WHEN** `(*Cache).Get(ctx)` returns a non-nil error during a `synth.Release` invocation
- **THEN** `synth.Release` returns the zero `cue.Value` and an error wrapping the Cache's error

#### Scenario: No registry round-trip on warm cache

- **WHEN** `synth.Release` is called with a `SchemaCache` whose underlying CUE module cache is already warm
- **THEN** the call completes without contacting any external registry, regardless of `CUE_REGISTRY` value

### Requirement: synth.Release does not validate or enforce concreteness

`synth.Release` SHALL return the unified CUE value without invoking `cue.Concrete` validation. Validation of values against `#config` and concreteness enforcement on the final spec are downstream responsibilities (the kernel wrapper handles both via `Kernel.ProcessModuleRelease`). Errors from CUE during unification (e.g. type mismatch between caller-supplied labels and the schema's label-map type) SHALL be returned as the result of `cue.Value.Err()` on the returned value, surfaced to the caller through the returned `error`.

#### Scenario: Unification error returned

- **WHEN** `synth.Release` is called with inputs that conflict with the schema (e.g. `Name` containing characters disallowed by `#NameType`)
- **THEN** the returned error is non-nil and the returned `cue.Value` is the zero value or carries the unification error

#### Scenario: No concreteness check at synth time

- **WHEN** `synth.Release` is called with `Values == cue.Value{}` against a `#config` that has no defaults
- **THEN** the call succeeds (returns a non-zero `cue.Value` and a nil error)
- **AND** the returned value's values path is unfilled rather than concrete

### Requirement: Release construction shares one evaluate-and-shape-gate with the file loader

The library SHALL provide a single build-and-validate routine that both `synth.Release` (in-memory overlay source) and `LoadReleasePackage` (on-disk source) invoke to evaluate a release package and run the shape gate. The two entry points SHALL differ only in how the package source is supplied (overlay vs. filesystem directory); the evaluation, shape-gating, and error wrapping SHALL be identical.

#### Scenario: Overlay and on-disk releases evaluate identically

- **WHEN** the same release content is supplied once as an in-memory overlay (via `synth.Release`) and once as an on-disk package (via `LoadReleasePackage`)
- **THEN** both produce a value of the same shape passing the same release shape gate
- **AND** a malformed release fails the shape gate identically in both paths

### Requirement: Imported-module render coverage exists

The library SHALL include a test that renders a release whose module is referenced by import (not inlined) end-to-end through construction and `Kernel.Compile`, producing concrete resources. This coverage SHALL exist for both the synth path and an authored-package path so that a regression in either surfaces.

#### Scenario: Real imported module renders to resources

- **WHEN** a release referencing a published module by import is rendered through the kernel
- **THEN** the compile output contains the module's expected resources
- **AND** the test fails if import-based construction regresses to a `field not allowed` admission error

<!-- Synced from simplify-render-single-build as MODIFIED, but it had no matching requirement name in this spec; appended as new. Review whether it should supersede parts of "synth.Release function signature" / "Derived fields come from schema unification". -->
### Requirement: synth.Release constructs the release by single-build CUE evaluation

`synth.Release` SHALL construct the `#ModuleRelease` value by synthesizing an in-memory CUE package and evaluating it in a **single build** (per ADR-003), through the same loader build path used by `LoadReleasePackage`. The synthesized package SHALL consist of a fabricated `cue.mod/module.cue` declaring the dependencies (the resolved `opmodel.dev/core` version and the module's `path@version`), a release source that **imports** the module and writes `#module: <import>` plus caller-supplied `metadata`, and a values source rendered from `ReleaseInput.Values`.

`synth.Release` SHALL NOT inject the module via `cue.Scope` / a `userModule` field, and SHALL NOT pre-merge `Values` into the module's `#config` in Go. The values merge SHALL be performed by the schema in CUE (`#ModuleRelease`'s `unifiedModule = #module & {#config: values}`). The Go code SHALL fill only caller-supplied inputs and let CUE derive `metadata.uuid`, `components`, `opm-secrets`, and stamped labels.

The public Go signature `Release(ctx *cue.Context, in ReleaseInput) (cue.Value, error)` and the `ReleaseInput` field set SHALL be unchanged. For every input that succeeds today, the returned value SHALL be observably equivalent (same `metadata.uuid`, same `components`, same labels).

#### Scenario: No scope or Go pre-merge in the construction path

- **WHEN** `synth.Release` is called with valid inputs
- **THEN** the construction performs no `cue.Scope`-based compile and no `FillPath` of `Values` into the module's `#config`
- **AND** the returned value carries the `#ModuleRelease` shape with `components` fanned by the schema comprehension

#### Scenario: Values merged in-build by the schema

- **WHEN** `synth.Release` is called with `Values` satisfying the module's `#config`
- **THEN** the returned value's `components` reflect the unified `#config = values` configuration
- **AND** the merge is produced by CUE evaluation of the synthesized package, not by a Go-side `#config` fill

#### Scenario: Imported-module construction succeeds

- **WHEN** `synth.Release` is called with a module whose identity is author-supplied (`metadata.modulePath` / `metadata.version` concrete) against a resolved `core` schema exposing author-supplied `#Module` identity
- **THEN** the release constructs without a `field not allowed` admission error
- **AND** the value unifies cleanly against `#ModuleRelease`

#### Scenario: Single-build parity with an authored package

- **WHEN** `synth.Release` builds a release for module M with values V, and an authored `release.cue` package imports the same M and sets the same V
- **THEN** both, passed through `Kernel.Compile`, produce the same set of compiled resources
