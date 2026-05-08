## Context

The kernel today renders OPM `#ModuleRelease` artifacts assuming a single CUE schema shape — `v1alpha2`. Path strings (`metadata`, `components`, `#transformers`, `#transform`, `#component`, `#context.{moduleReleaseMetadata,componentMetadata,runtimeName}`, `#config`, `values`, `#resources`, `#traits`, `#blueprints`, `requiredLabels`, `requiredResources`, `requiredTraits`, `optionalTraits`, `output`) and Go decode struct shapes (`module.ModuleMetadata`, `module.ReleaseMetadata`, `provider.ProviderMetadata`, the unexported `moduleReleaseContextData` and `componentContextData` in `pkg/render/execute.go`) are inlined at every call site.

`apis/core/v1alpha2/types.cue` declares `#ApiVersion: #ApiVersion` — a self-reference that never resolves to a literal. The loader never reads `apiVersion`. Multi-version dispatch is therefore impossible in the current shape.

Stakeholders that consume `pkg/`:

- `cli/` — calls `loader.LoadReleaseFile`, `module.ParseModuleRelease`, `render.ProcessModuleRelease`.
- `opm-operator/` — same render-call site as the CLI, plus a Kubebuilder reconciler around it.
- Future Crossplane composition function — first integration will land directly against the new shape.

The Constitution principle VIII (small batches) constrains the scope: every step in the implementation plan MUST be independently verifiable.

## Goals / Non-Goals

**Goals:**

- The kernel SHALL detect the OPM schema version of every loaded artifact via its `apiVersion` field.
- The kernel SHALL dispatch all version-specific behaviour (CUE path constants, metadata decoding, transformer-context shape) through a single `Binding` interface.
- A new schema version (`v1alpha3`, `v1beta1`, `v1`) SHALL be addable by dropping a new `pkg/api/<version>/` package and `apis/core/<version>/` schema directory — without edits to `pkg/render`, `pkg/loader/{module,provider,release}.go`, or `pkg/module/parse.go`.
- The render pipeline MUST keep its public entry point (`render.ProcessModuleRelease`) signature stable; downstream call sites in CLI / operator MUST NOT need to change to pick up multi-version support.
- Embedded schemas (`go:embed apis/core/<version>/**/*.cue`) SHALL ship with the kernel so artifact validation is deterministic and offline-capable.

**Non-Goals:**

- Cross-version conversion (`Convert(from, to apiversion.Version, v cue.Value)`) — deferred until a real consumer requires it.
- Migration tooling for module authors — belongs in the CLI, not the kernel.
- Adding a `v1alpha1` binding now — the foundation must land first; the binding lands as a follow-up change once the legacy schema lives under `apis/core/v1alpha1/`.
- Refactoring `pkg/validate/`, `pkg/core/`, or `pkg/errors/` — these have no version-specific behaviour today.
- Hot-swap or runtime registration of bindings beyond `init()` — Constitution principle I forbids package-level singletons whose behaviour changes after init.

## Decisions

### Decision 1 — Two new packages: `pkg/apiversion` and `pkg/api`

**Choice:** split version detection (`pkg/apiversion`) from the binding contract (`pkg/api`).

**Alternatives considered:**

- Single `pkg/api` package containing both. Rejected because version detection is a pure helper any caller may want without dragging the full `Binding` interface (e.g. validation tooling that just wants to refuse unknown versions).
- Putting detection inside `pkg/loader`. Rejected because non-loader callers (a controller that already has a `cue.Value`) must also be able to detect version without re-loading.

**Rationale:** clean separation. `apiversion.Version` is a leaf type with no dependencies beyond `cuelang.org/go/cue`. `pkg/api` depends on `pkg/apiversion` but adds the registry, the `Binding` interface, and the lowest-common-denominator decoded structs. Per-version packages depend on `pkg/api`. No import cycles.

### Decision 2 — `Paths` struct, not method-per-path

**Choice:** `Binding.Paths()` returns a single `api.Paths` struct holding every CUE path the kernel reads or writes.

**Alternatives considered:**

- A method per path (`binding.TransformersPath()`, `binding.ContextPath()`, …). Rejected because it inflates the interface surface to ~17 methods and makes it harder to audit which paths a version requires. A struct keeps the contract greppable.
- A `map[string]cue.Path` on `Binding`. Rejected on principle II (type safety) — a typo at the call site (`paths["#transfomer"]`) becomes a runtime nil instead of a compile error.

**Rationale:** the path inventory is small, finite, and known up front. A struct gives compile-time field access with zero runtime cost.

### Decision 3 — `init()`-based registration

**Choice:** each `pkg/api/<version>/` package calls `api.Register(&binding{})` from `init()`.

**Alternatives considered:**

- Explicit `api.Register` call from the consumer's `main()`. Rejected because the kernel is consumed by multiple binaries (CLI, controller, function) — relying on each consumer to remember to wire bindings is a footgun, and forgetting it produces `ErrUnknownAPIVersion` at runtime instead of compile time.
- Build-tag selection of bindings. Rejected — multiple versions must coexist at runtime so a single binary can read v1alpha1 and v1alpha2 artifacts simultaneously.

**Rationale:** `init()` registration is the standard Go pattern for plug-in style version dispatch (`database/sql`, `image`). Constitution principle I forbids "package-level singletons that hide behavior" — the registry is initialised exactly once at process start and is read-only after that, which is acceptable. We document that no side-by-side duplicate registration is permitted (`Register` panics on duplicates) so the behaviour stays deterministic.

### Decision 4 — `BuildTransformerContext` returns a `cue.Value`, not fills it

**Choice:** the binding builds the `#context` value and returns it; the renderer fills it into the unified transformer value via `unified.FillPath(paths.Context, ctxVal)`.

**Alternatives considered:**

- Binding owns the entire fill operation. Rejected because the renderer needs the resulting `unified` value to look up `output` afterwards — splitting the fill across binding and renderer creates two cue.Value lifetimes for the same logical operation.

**Rationale:** the binding owns *shape*, the renderer owns *plumbing*. The split keeps the renderer trivially mockable for future testing.

### Decision 5 — Lowest-common-denominator metadata structs in `pkg/api`

**Choice:** `pkg/api` declares `ModuleMetadata`, `ReleaseMetadata`, `ProviderMetadata` as the canonical decoded shape every binding returns. Per-version richer data, if any, stays internal to that version's package.

**Alternatives considered:**

- Each binding returns its own typed struct via `any`. Rejected because every consumer would have to type-switch — that pushes version awareness back into downstream code, defeating the abstraction.
- Make the metadata structs themselves an interface. Rejected as premature; today's two prospective versions have the same metadata shape, and YAGNI applies.

**Rationale:** v1alpha1 and v1alpha2 metadata are functionally identical (name, namespace, fqn, version, uuid, labels, annotations). If a future version diverges substantially (e.g. introduces multi-tenant identity), promote the per-version richer struct then — not before.

### Decision 6 — `apiVersion` lives at artifact root, not inside metadata

**Choice:** `apiversion.Detect(v)` looks up `apiVersion` directly on the passed `cue.Value`. Each artifact root (`#Module`, `#ModuleRelease`, `#Provider`, `#Component`) carries its own `apiVersion` field — already the case in `apis/core/v1alpha2/`.

**Alternatives considered:**

- Detect via the CUE module path (e.g. `opmodel.dev/core@v1alpha2`). Rejected because release files import a module — the apiVersion of the *release* matters more than the importing-module's path. Module path detection would also break for releases authored against multiple module versions.

**Rationale:** the artifact-root field is what users authoritatively assert. The CUE schema already enforces `apiVersion: #ApiVersion`; pinning the literal makes the assertion concrete without changing user authoring.

### Decision 7 — `render.Match` takes the binding; `render.ProcessModuleRelease` resolves it internally

**Choice:** `render.Match(components, provider, binding)` is breaking. `render.ProcessModuleRelease(ctx, rel, p, runtimeName)` keeps its signature; it pulls the binding from `rel.APIVersion` (which the loader populated) via `api.Lookup`.

**Alternatives considered:**

- Pass binding to `ProcessModuleRelease` too. Rejected because every downstream call site (CLI, operator) would need to plumb the binding through; the goal of the change is to make multi-version support invisible to consumers of the public render entry point.
- Resolve binding inside `Match`. Rejected because `Match` does not have access to the release — components are passed in directly. Forcing `Match` to round-trip back to the release for the binding is awkward.

**Rationale:** `Match` is a lower-level primitive used by `ProcessModuleRelease`. Today no downstream code calls `Match` directly. The break is paid once at refactor time and is invisible to actual consumers.

### Decision 8 — Embed schemas via `go:embed`, not vendor

**Choice:** `apis/core/<version>/embed.go` declares `//go:embed *.cue cue.mod/module.cue` exposing an `embed.FS` the loader can use for offline schema validation.

**Alternatives considered:**

- Re-fetch schemas from the OCI registry at first load. Rejected — adds runtime dependency on network and breaks determinism (principle I).
- Vendor schemas as Go string literals via codegen. Rejected — `go:embed` is the canonical Go-native solution since 1.16.

**Rationale:** schemas are part of the kernel's contract for that version. Shipping them with the binary makes the kernel self-contained and lets `task check` run offline.

### Decision 9 — `APIVersion` casing on Go fields

**Choice:** the Go field on every artifact type is spelled `APIVersion` (initialism in caps), not `ApiVersion`.

**Alternatives considered:**

- `ApiVersion`. Rejected — Go style guide treats acronyms uniformly (`URL`, `ID`, `HTTP`, `API`); mixed casing is non-idiomatic and `gofmt`/`golint` flag it eventually. The lowercase `apiversion` *package* and the `apiversion.Version` *type* are intentional (Go package names are short and lowercase); only the struct field is renamed.
- Defer to a future MAJOR. Rejected — this MINOR already ships breaking signatures (`render.Match`, loader return types). Bundling one more field-name break now costs zero extra coordination for `cli` and `opm-operator`; deferring would force a separate breaking release whose only purpose is cosmetic.

**Rationale:** the rename lands in lockstep with the binding work so downstream consumers update against a stable shape. Draft slices that already reference the field (`unify-artifact-shape`, `add-platform-construct`, `add-platform-composition-helper`, `add-phase-methods-and-rename-compile`) and the umbrella enhancement docs are propagated in the same change so future implementers do not contradict the implemented API.

## Risks / Trade-offs

- [`render.Match` signature break] → No downstream code currently calls `Match` directly; CHANGELOG flags it; CLI and operator only depend on `ProcessModuleRelease` which keeps its signature.
- [`init()`-based registration is "magic"] → Documented in `pkg/api/doc.go`; the registry panics on duplicate registration so misuse fails loud at process start, not silently at runtime.
- [Embedding schemas pins them to library version] → Acceptable: the library's SemVer is already the authoritative pin for what schema versions a kernel build supports. Catalog publishing remains the canonical distribution; embed is a determinism convenience.
- [Test coverage gap in renderer is exposed by the refactor] → The change adds regression tests for `Match`, `executeTransforms`, and `ProcessModuleRelease` against the new binding shape before the breaking signature ships.
- [Two prospective versions today have identical metadata shape — abstraction may be over-engineered] → Constitution VII (YAGNI) tension. Decision 5 is the YAGNI-aligned compromise: build the dispatch infrastructure now (because we have the request), but don't introduce per-version Go shapes until a version actually diverges.
- [`BuildTransformerContext` allocates a fresh `cue.Value` per pair] → Acceptable: the renderer already does similar per-pair allocations via `FillPath`; CUE context isolation prevents cross-pair contamination.

## Migration Plan

The change is implemented across a sequence of independently mergeable PRs, each within Constitution VIII's small-batch envelope:

1. **PR 1 — schema literal pin.** Replace `#ApiVersion: #ApiVersion` with `#ApiVersion: "opmodel.dev/v1alpha2"` in `apis/core/v1alpha2/types.cue`. Re-run `task check`. Catalog dependents (`modules/`, `releases/`) re-publish on next routine update; old artifacts continue to validate because the CUE constraint is still `apiVersion: #ApiVersion`.

2. **PR 2 — `pkg/apiversion`.** New package only; no edits to existing code. Tests cover: present + recognised, present + unknown, absent.

3. **PR 3 — `pkg/api` interface + registry.** New package only; no edits to existing code. Tests cover: register, lookup, duplicate-register panic, unknown-version error.

4. **PR 4 — `pkg/api/v1alpha2` binding.** Implements `Binding` with paths and decoders mirroring today's hardcoded values. Self-registers in `init()`. The transformer-context structs (`moduleReleaseContextData`, `componentContextData`) move out of `pkg/render/execute.go` into this package; `execute.go` keeps a temporary local definition in this PR so it still compiles.

5. **PR 5 — wire loader.** `pkg/loader/{module,release,provider}.go` calls `apiversion.Detect` after `BuildInstance`. Loader output additively gains a `Version` field; existing callers ignore it. New regression tests assert detected version on a v1alpha2 fixture.

6. **PR 6 — wire render path lookups.** `pkg/render/match.go` accepts `binding api.Binding` (BREAKING). `pkg/render/process_module.go` resolves the binding via `api.Lookup(rel.APIVersion)` and passes it through. Hardcoded `cue.ParsePath` strings become `binding.Paths().X`. Existing render tests adjusted to construct a binding-aware test fixture.

7. **PR 7 — wire context injection.** `injectContext` in `pkg/render/execute.go` becomes a one-line delegation to `binding.BuildTransformerContext`. The temporary local context structs from PR 4 are deleted. Snapshot tests assert rendered output is byte-identical before vs after the refactor.

8. **PR 8 — embed schemas.** `apis/core/v1alpha2/embed.go` exposes the embed.FS. Loader exposes a `LoadEmbeddedSchema(version)` helper. Used initially only by self-tests; downstream wiring follows in subsequent changes.

**Rollback strategy:** each PR is independently reversible via `git revert`. PRs 1, 2, 3, 8 are purely additive. PR 4 introduces a registered binding but does not yet change render behaviour. PRs 5–7 carry the behavioural changes; if any regression surfaces in downstream `cli/` or `opm-operator/` integration tests, revert just that PR — earlier PRs stay landed because they are no-ops at runtime.

**Downstream coordination:** `cli/` and `opm-operator/` get a single update PR after PRs 6+7 land that picks up the new module version; no signature changes required at their call sites.

## Open Questions

- Do we want a `v1alpha1` binding to land in the same change set, or is that a follow-up after the foundation is proven? Default: follow-up, to keep this change scope honest.
- Should `Register` panic on duplicate, or return an error? Panic is the standard library pattern (`sql.Register`) and detects misconfiguration earliest; sticking with panic unless a downstream needs graceful handling.
- Where does the `v1alpha1` schema source live — copied from `catalog/` into `apis/core/v1alpha1/`, or symlinked? Out of scope for this change; revisit when the v1alpha1 binding lands.
