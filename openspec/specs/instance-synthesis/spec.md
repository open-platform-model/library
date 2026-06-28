# instance-synthesis Specification

## Purpose
TBD - created by archiving change rename-release-to-instance. Update Purpose after archive.
## Requirements
### Requirement: Synth Helper Package Location

The library SHALL expose a `opm/helper/synth/` subpackage that produces OPM artifact CUE values from in-memory typed inputs. The package SHALL be a peer of `opm/helper/loader/` (not nested under it) because synthesis is creation from typed inputs rather than parsing from byte streams. The package SHALL document the boundary in its `doc.go`.

#### Scenario: Synth package present at canonical path

- **WHEN** a developer reads `opm/helper/synth/doc.go`
- **THEN** the file documents that the package builds artifact CUE values from typed inputs
- **AND** documents that the package is a peer of `opm/helper/loader/`, not nested under it

### Requirement: synth.Instance function signature

The `opm/helper/synth/` package SHALL expose a function `Instance(ctx *cue.Context, in InstanceInput) (cue.Value, error)` that returns a `#ModuleInstance` artifact CUE value built by unifying the input fields against the `#ModuleInstance` schema definition resolved from the supplied `SchemaCache`.

The `InstanceInput` struct SHALL carry: `Module *module.Module` (required), `Name string` (required), `Namespace string` (required), `SchemaCache *schema.Cache` (REQUIRED), `Values cue.Value` (optional; zero value means "no values supplied"), `Labels map[string]string` (optional), `Annotations map[string]string` (optional).

`synth.Instance` MUST return a non-nil error when `SchemaCache == nil` (in addition to the existing required-field checks). The error message MUST name the missing field. The helper MUST NOT self-construct a `*schema.Cache` as a fallback; the caller is responsible for passing the cache it intends to share (typically `k.SchemaCache()` from its Kernel).

#### Scenario: Required inputs validated

- **WHEN** `synth.Instance` is called with `Module == nil`, or `Name == ""`, or `Namespace == ""`, or `SchemaCache == nil`
- **THEN** it returns the zero `cue.Value` and a non-nil error naming the missing field

#### Scenario: Returned value is schema-unified

- **WHEN** `synth.Instance` is called with valid inputs and a `SchemaCache` whose Loader resolves the OPM core schema
- **THEN** the returned `cue.Value` carries the `#ModuleInstance` shape at its root
- **AND** the value is unified with the schema's `#ModuleInstance` definition (the schema's structural constraints apply)

#### Scenario: Caller's Cache is reused, not replaced

- **WHEN** `synth.Instance` is called with a `SchemaCache` that has already been warmed by a prior `Get`
- **THEN** the helper invokes `(*Cache).Get(ctx)` to retrieve the already-cached value
- **AND** no second schema load is triggered by the helper

### Requirement: Derived fields come from schema unification

`synth.Instance` SHALL NOT compute `metadata.uuid`, `components`, or schema-stamped labels in Go. These fields SHALL flow from CUE evaluation as a consequence of unifying the inputs with `#ModuleInstance`. The Go code SHALL fill only the caller-supplied fields (name, namespace, `#module`, optional values/labels/annotations) and let CUE derive the rest.

#### Scenario: UUID is computed by CUE

- **WHEN** `synth.Instance` is called twice with the same `(Module, Name, Namespace)`
- **THEN** the returned CUE values carry identical `metadata.uuid` strings
- **AND** the UUID equals `uuid.SHA1(OPMNamespace, "<module.uuid>:<name>:<namespace>")` per the schema definition at `apis/core/v1alpha2/module_instance.cue:19`

#### Scenario: UUID diverges with namespace

- **WHEN** `synth.Instance` is called with identical `(Module, Name)` but two different `Namespace` values
- **THEN** the returned CUE values carry different `metadata.uuid` strings

#### Scenario: Components are fanned by schema comprehension

- **WHEN** `synth.Instance` is called with a Module declaring N concrete components in `#components`
- **THEN** the returned CUE value's `components` field contains exactly those N entries with the schema-applied projections
- **AND** the synth helper itself does not enumerate `#components` in Go

#### Scenario: Auto-secrets component included when module has #Secret instances

- **WHEN** `synth.Instance` is called with a Module whose `#config` (after `Values` is filled) contains at least one `#Secret` instance
- **THEN** the returned CUE value's `components` field contains an `opm-secrets` entry
- **AND** when the module contains no `#Secret` instances, no `opm-secrets` entry appears

#### Scenario: Standard instance labels are stamped by schema

- **WHEN** `synth.Instance` is called with valid inputs
- **THEN** the returned CUE value's `metadata.labels` contains keys `module-instance.opmodel.dev/name` and `module-instance.opmodel.dev/uuid`
- **AND** their values equal the instance name and the derived UUID respectively

### Requirement: Values field is caller-supplied with no implicit fallback

`synth.Instance` SHALL NOT consult `Module.debugValues` or any other implicit source when `InstanceInput.Values` is the zero `cue.Value`. When `Values.Exists()` is true, the helper SHALL render it into the synthesized package's values source (via `format.Node` on the value's syntax, never string-interpolating raw input) so it participates in the single build. When `Values.Exists()` is false, the helper SHALL omit the values source and return the unified value as-is; concreteness enforcement is deferred to `Kernel.ProcessModuleInstance`.

#### Scenario: Caller-supplied values participate in the build

- **WHEN** `synth.Instance` is called with `Values` set to a concrete CUE value satisfying the module's `#config`
- **THEN** the returned value carries those values at the schema's values path
- **AND** the values entered the build as a rendered source file, not a post-build cross-build unification

#### Scenario: Zero Values is not replaced by debugValues

- **WHEN** `synth.Instance` is called with `Values == cue.Value{}` against a Module that defines `debugValues`
- **THEN** the returned CUE value's values path is unfilled (does not equal `debugValues`)

### Requirement: Optional labels and annotations are filled into instance metadata

When `InstanceInput.Labels` is non-empty, `synth.Instance` SHALL fill `metadata.labels` with those entries. When `InstanceInput.Annotations` is non-empty, `synth.Instance` SHALL fill `metadata.annotations` with those entries. The schema's label stamping (Requirement: Derived fields come from schema unification) unifies with caller-supplied labels — caller labels MUST NOT be allowed to remove schema-stamped labels.

#### Scenario: Caller labels merged with schema-stamped labels

- **WHEN** `synth.Instance` is called with `Labels == {"env": "prod"}`
- **THEN** the returned CUE value's `metadata.labels` contains the `env: prod` entry
- **AND** still contains `module-instance.opmodel.dev/name` and `module-instance.opmodel.dev/uuid`

#### Scenario: Annotations are passed through unchanged

- **WHEN** `synth.Instance` is called with `Annotations == {"opmodel.dev/owner": "team-x"}`
- **THEN** the returned CUE value's `metadata.annotations` contains that entry

### Requirement: Schema obtained through caller-supplied Cache

`synth.Instance` SHALL obtain the `#ModuleInstance` definition by calling `in.SchemaCache.Get(ctx)` on the caller-supplied `*schema.Cache`, then `LookupPath("#ModuleInstance")` on the returned value. The helper MUST NOT call `load.Instances` directly, MUST NOT consult `os.Getenv("CUE_REGISTRY")`, MUST NOT read from the filesystem, and MUST NOT construct its own `*schema.Cache` or `Loader`.

#### Scenario: Helper delegates schema loading to the Cache

- **WHEN** `synth.Instance` is called with a `SchemaCache` configured against a pre-seeded test cache
- **THEN** the call succeeds without any direct call to `load.Instances`, `os.Getenv`, or filesystem reads originating in `opm/helper/synth/`

#### Scenario: Schema load failure surfaces as a wrapped error

- **WHEN** `(*Cache).Get(ctx)` returns a non-nil error during a `synth.Instance` invocation
- **THEN** `synth.Instance` returns the zero `cue.Value` and an error wrapping the Cache's error

#### Scenario: No registry round-trip on warm cache

- **WHEN** `synth.Instance` is called with a `SchemaCache` whose underlying CUE module cache is already warm
- **THEN** the call completes without contacting any external registry, regardless of `CUE_REGISTRY` value

### Requirement: synth.Instance does not validate or enforce concreteness

`synth.Instance` SHALL return the unified CUE value without invoking `cue.Concrete` validation. Validation of values against `#config` and concreteness enforcement on the final spec are downstream responsibilities (the kernel wrapper handles both via `Kernel.ProcessModuleInstance`). Errors from CUE during unification (e.g. type mismatch between caller-supplied labels and the schema's label-map type) SHALL be returned as the result of `cue.Value.Err()` on the returned value, surfaced to the caller through the returned `error`.

#### Scenario: Unification error returned

- **WHEN** `synth.Instance` is called with inputs that conflict with the schema (e.g. `Name` containing characters disallowed by `#NameType`)
- **THEN** the returned error is non-nil and the returned `cue.Value` is the zero value or carries the unification error

#### Scenario: No concreteness check at synth time

- **WHEN** `synth.Instance` is called with `Values == cue.Value{}` against a `#config` that has no defaults
- **THEN** the call succeeds (returns a non-zero `cue.Value` and a nil error)
- **AND** the returned value's values path is unfilled rather than concrete

### Requirement: Instance construction shares one evaluate-and-shape-gate with the file loader

The library SHALL provide a single build-and-validate routine that both `synth.Instance` (in-memory overlay source) and `LoadInstancePackage` (on-disk source) invoke to evaluate an instance package and run the shape gate. The two entry points SHALL differ only in how the package source is supplied (overlay vs. filesystem directory); the evaluation, shape-gating, and error wrapping SHALL be identical.

#### Scenario: Overlay and on-disk instances evaluate identically

- **WHEN** the same instance content is supplied once as an in-memory overlay (via `synth.Instance`) and once as an on-disk package (via `LoadInstancePackage`)
- **THEN** both produce a value of the same shape passing the same instance shape gate
- **AND** a malformed instance fails the shape gate identically in both paths

### Requirement: Imported-module render coverage exists

The library SHALL include a test that renders an instance whose module is referenced by import (not inlined) end-to-end through construction and `Kernel.Compile`, producing concrete resources. This coverage SHALL exist for both the synth path and an authored-package path so that a regression in either surfaces. The synth-path coverage SHALL include a module that imports a **catalog subpackage** (e.g. a workload blueprint under `opmodel.dev/catalogs/opm/...`), so that a regression to a dependency-incomplete synthesis surfaces as a failing test.

#### Scenario: Real imported module renders to resources

- **WHEN** an instance referencing a published module by import is rendered through the kernel
- **THEN** the compile output contains the module's expected resources
- **AND** the test fails if import-based construction regresses to a `field not allowed` admission error

#### Scenario: Module importing a catalog subpackage synthesizes and renders

- **WHEN** `synth.Instance` is called with a published module whose source imports a transitive catalog subpackage (the library#31 shape), against a registry serving that catalog
- **THEN** synthesis succeeds without `cannot find module providing package opmodel.dev/catalogs/opm/...`
- **AND** the instance renders to the module's expected resources through `Kernel.Compile`

### Requirement: synth.Instance constructs the instance by single-build CUE evaluation

`synth.Instance` SHALL construct the `#ModuleInstance` value by synthesizing an in-memory CUE package and evaluating it in a **single build** (per ADR-003), through the same loader build path used by `LoadInstancePackage`. The instance source SHALL be overlaid **into the acquired module's own staged source tree** (under a reserved synthetic subdirectory of the module's staged root), so the module's published, already-tidied `cue.mod/module.cue` is the build's main-module module file and drives all transitive dependency resolution. `synth.Instance` SHALL NOT fabricate a `cue.mod/module.cue` and SHALL NOT declare a dependency set; it reuses the module's tidied deps verbatim.

The instance source SHALL **import** the module's own package by its module path and write `#module: <import>` plus caller-supplied `metadata`, and SHALL import `core` (resolved from the module's own deps) to embed `#ModuleInstance`. Because the instance is built inside the module's main module, the module import resolves **locally** (no fabricated dependency, no registry round-trip for the module itself). A values source rendered from `InstanceInput.Values` SHALL be overlaid alongside the instance source when `Values.Exists()`.

`synth.Instance` SHALL NOT inject the module via `cue.Scope` / a `userModule` field, and SHALL NOT pre-merge `Values` into the module's `#config` in Go. The values merge SHALL be performed by the schema in CUE (`#ModuleInstance`'s `unifiedModule = #module & {#config: values}`). The Go code SHALL fill only caller-supplied inputs and let CUE derive `metadata.uuid`, `components`, `opm-secrets`, and stamped labels.

The public Go signature `Instance(ctx *cue.Context, in InstanceInput) (cue.Value, error)` and the `InstanceInput` field set SHALL be unchanged. For every input that succeeds today, the returned value SHALL be observably equivalent (same `metadata.uuid`, same `components`, same labels).

#### Scenario: No scope or Go pre-merge in the construction path

- **WHEN** `synth.Instance` is called with valid inputs
- **THEN** the construction performs no `cue.Scope`-based compile and no `FillPath` of `Values` into the module's `#config`
- **AND** the returned value carries the `#ModuleInstance` shape with `components` fanned by the schema comprehension

#### Scenario: No fabricated module file

- **WHEN** `synth.Instance` constructs an instance
- **THEN** the build's main-module `cue.mod/module.cue` is the module's own published module file (carrying its full tidied dependency closure)
- **AND** `synth.Instance` does not generate a `cue.mod/module.cue` or a `deps:` block of its own

#### Scenario: Values merged in-build by the schema

- **WHEN** `synth.Instance` is called with `Values` satisfying the module's `#config`
- **THEN** the returned value's `components` reflect the unified `#config = values` configuration
- **AND** the merge is produced by CUE evaluation of the synthesized package, not by a Go-side `#config` fill

#### Scenario: Imported-module construction succeeds

- **WHEN** `synth.Instance` is called with a module whose identity is author-supplied (`metadata.modulePath` / `metadata.version` concrete)
- **THEN** the instance constructs without a `field not allowed` admission error
- **AND** the value unifies cleanly against `#ModuleInstance`

#### Scenario: Single-build parity with an authored package

- **WHEN** `synth.Instance` builds an instance for module M with values V, and an authored `instance.cue` package imports the same M and sets the same V
- **THEN** both, passed through `Kernel.Compile`, produce the same set of compiled resources

### Requirement: synth.Instance requires the module's staged source

`synth.Instance` SHALL construct the instance from the acquired module's staged source (the overlay + synthetic root produced when the module is fetched from the registry). When the supplied `InstanceInput.Module` does not carry staged source (it was constructed from a bare value rather than acquired with source), `synth.Instance` SHALL return the zero `cue.Value` and a non-nil error explaining that the module must be acquired with its source (e.g. via the source-carrying registry acquire entrypoint). `synth.Instance` SHALL NOT itself perform a registry fetch to obtain the module source.

#### Scenario: Module without staged source is rejected

- **WHEN** `synth.Instance` is called with a `Module` that carries no staged source
- **THEN** it returns the zero `cue.Value` and a non-nil error naming the missing source precondition
- **AND** `synth.Instance` performs no registry fetch of its own

#### Scenario: Module acquired with source synthesizes

- **WHEN** `synth.Instance` is called with a `Module` acquired through the source-carrying registry path
- **THEN** the instance is staged inside the module's source tree and synthesis proceeds

### Requirement: Transitive module dependencies resolve via the module's own module file

Because the instance is built inside the module's staged main module, `synth.Instance` SHALL resolve the module's transitive dependencies (core, catalog packages, and any further indirect deps) through the module's own `cue.mod/module.cue`, identically to how `registry-module-loading` resolves them when loading the module standalone. The caller SHALL NOT be required to declare, derive, or tidy the module's transitive closure for synthesis to succeed.

#### Scenario: Indirect catalog dependency resolves without caller declaration

- **WHEN** a module's source imports a package whose own dependency is an indirect catalog module not directly imported by the module
- **THEN** `synth.Instance` resolves that indirect dependency via the module's tidied `cue.mod/module.cue`
- **AND** synthesis succeeds without the caller supplying the transitive closure

