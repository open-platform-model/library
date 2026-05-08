## Why

The kernel currently hardcodes the `v1alpha2` OPM schema contract throughout `pkg/render`, `pkg/loader`, `pkg/module`, and `pkg/provider` — CUE path strings and Go decode struct shapes are inlined at every call site. There is no mechanism to detect or dispatch on `apiVersion`, and `#ApiVersion` in the embedded schema is currently a self-reference (`#ApiVersion: #ApiVersion`) that never resolves to a literal.

The kernel must be able to load and render multiple OPM schema versions side by side so that downstream implementations (CLI, controller, future Crossplane composition function) inherit multi-version support without per-implementation effort. Doing this now — while only `v1alpha2` exists — is dramatically cheaper than retrofitting once the catalog has shipped a `v1alpha1` consumer in the wild that the kernel cannot read.

## What Changes

- Pin a concrete `#ApiVersion` literal in each schema directory under `apis/core/<version>/types.cue` (currently a self-reference that never resolves).
- Add a new `pkg/apiversion` package owning the `Version` type, the registered constants (`V1alpha2`, future `V1alpha1`), and a `Detect(cue.Value)` helper that reads `apiVersion` from any OPM artifact root.
- Add a new `pkg/api` package defining a `Binding` interface and a registry. Each binding owns its version's CUE path constants (`Paths`), per-version metadata decoders, and the version-specific `BuildTransformerContext` shape. Versions register themselves via `init()`.
- Add `pkg/api/v1alpha2/` as the first concrete binding. Move the unexported transformer-context structs (`moduleReleaseContextData`, `componentContextData`) out of `pkg/render/execute.go` and into the binding.
- Add an `APIVersion apiversion.Version` field on `module.Module`, `module.Release`, and `provider.Provider`, populated by the loader. **BREAKING** for `pkg/loader`, `pkg/module`, `pkg/provider` constructors that downstream code calls directly — additive otherwise.
- Replace every hardcoded `cue.ParsePath("…")` / `cue.MakePath(cue.Def("…"))` in `pkg/render/match.go` and `pkg/render/execute.go` with paths from the binding's `Paths()`. Replace `injectContext` with a one-line delegation to `binding.BuildTransformerContext`. **BREAKING** for `render.Match` (adds binding parameter); `render.ProcessModuleRelease` keeps its signature and looks the binding up internally from the release.
- `go:embed` each `apis/core/<version>/**/*.cue` so the kernel ships its schemas with the Go binary and can validate artifacts deterministically without a network registry round-trip.
- **Out of scope (deferred):** cross-version conversion (`Convert(from, to, v cue.Value)`); migration tooling — the latter belongs in the CLI, not the kernel; introduction of any schema version other than the existing `v1alpha2`.

### Follow-up cleanups (post `/opsx:verify`)

A verification pass against the live v1alpha2 schema and the kernel-redesign enhancement surfaced four binding-coherence cleanups, folded into this change as task groups 10–14:

- Rename `#Transformer` → `#ComponentTransformer` (kind literal + type name) so the schema file matches `apis/core/v1alpha2/docs/adapters.md` and the catalog 014 design.
- Drop `ModuleMetadata.DefaultNamespace` (Go field never populated by `Decode()`); expose `AnnotationDefaultNamespace` constant instead. Implements ADR-001.
- Drop `Paths.ComponentBlueprints` — dead code. Blueprints unify into Component `spec` at CUE-evaluation time per `component.cue:_allFields`; the renderer never walks them.
- Rename `ApiVersion` field → `APIVersion` (idiomatic Go casing) on every public artifact type. Bundled with the existing breaking changes for the next MINOR. Propagated to draft slices that already reference the field.

## Capabilities

### New Capabilities

- `api-version-dispatch`: loading, identifying, and dispatching on the OPM schema version of incoming artifacts (`#Module`, `#ModuleRelease`, `#Provider`). Covers the public `apiversion` and `api` packages, the `Binding` contract, registry semantics, embedded schema validation, and how the render pipeline consumes a binding.

### Modified Capabilities

None — `openspec/specs/` is empty; this is the first capability.

## Impact

**Affected packages:**

- New: `pkg/apiversion`, `pkg/api`, `pkg/api/v1alpha2`.
- Modified: `pkg/loader/{module,provider,release}.go` (detect + surface version); `pkg/module/{module,release,parse}.go` (carry `ApiVersion`, delegate metadata decode to binding); `pkg/provider/provider.go` (carry `ApiVersion`, delegate metadata decode); `pkg/render/{match,execute,process_module}.go` (path constants come from binding, context injection via binding).
- Unchanged: `pkg/core/`, `pkg/errors/`, `pkg/validate/`, `pkg/render/finalize.go`.

**Affected schemas:**

- `apis/core/v1alpha2/types.cue` — pin `#ApiVersion` literal.
- `apis/core/v1alpha2/**/*.cue` — embedded via `go:embed` (no source change).

**Downstream consumers:**

- `cli/`: depends on `pkg/loader`, `pkg/module`, `pkg/render`. Will need to consume the new binding-aware constructors. Migration is mechanical — pass through the version returned by the loader; the public render entry point keeps its signature.
- `opm-operator/`: same pattern as CLI. Affected at the controller's render-call site only.
- Crossplane function (planned): no migration cost — first integration will land against the new shape.

**SemVer classification:** MINOR with one localised BREAKING change.

The new `pkg/apiversion` and `pkg/api` packages are purely additive. Adding `ApiVersion` fields to existing public structs is additive. The signature change to `render.Match` (binding parameter) is the single breaking change in the public surface; downstream call sites today do not call `Match` directly — they call `ProcessModuleRelease` — so the practical migration cost is zero. The change is therefore released as the next MINOR with the breaking signature called out in CHANGELOG.

**Dependencies:** no new Go module dependencies. CUE SDK `cuelang.org/go` already covers everything required.

**Risk surface:** test coverage today exists for `pkg/core/identity` and `pkg/errors`. The renderer is currently uncovered — the refactor must add regression tests for `Match`, `executeTransforms`, and `ProcessModuleRelease` against the new binding shape before the breaking change ships.
