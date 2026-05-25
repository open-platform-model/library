## Context

The library was built when multi-version OPM schemas were anticipated: `apis/core/v1alpha2/`, `apis/core/<vN>/`, and a per-version `api.Binding` interface dispatched through a process-wide registry keyed by `apiversion.Version`. Every kernel consumer (module, platform, kernel, compile, helper/*) went through `api.Lookup(x.APIVersion).Paths()...` to access CUE paths and metadata decoders.

The core schemas have since been moved out of `library/` into a dedicated `core` repo. With the move, `apiVersion: #ApiVersion` was removed from every artifact root and the `#ApiVersion` constant was dropped. The library no longer hosts the authoritative CUE source; the new `core` repo does.

With a single, unversioned schema:
- `api.Lookup(rel.APIVersion)` resolves to exactly one Binding, every time.
- `apiversion.Detect(v)` parses a field that does not exist any more.
- The init-time registry, the duplicate-registration panic, and the blank-import contract in `main`/`cmd/flow-inspect` exist to coordinate something that has only one valid value.

Per the user, the library has no external consumers yet, so this is an opportune moment for a clean break.

## Goals / Non-Goals

**Goals:**
- Delete `opm/api` and `opm/apiversion` entirely. No deprecation shims.
- Move the real logic the binding hid (Paths, metadata decoders, context builder, embedded-schema loader) into a single flat `opm/schema` package consumed by free functions and package-level vars.
- Re-sync `library/apis/core/` with the new `core` repo so the embedded schema matches reality (flat layout, package `core`, no `apiVersion` field).
- Drop the `APIVersion` field from `Module`, `Release`, `Platform`. Stop returning `apiversion.Version` from loader helpers.
- Drop the `api.Binding` parameter from `compile.Match` and `(*compile.Module).Execute`.
- Keep `task check` green at the end.

**Non-Goals:**
- Re-architect the kernel beyond removing this indirection. Compile pipeline, match algorithm, validation tiers, error-wrap conventions stay as-is.
- Rewrite the consumer CUE fixtures (`library/modules/opm_platform/`, `library/testdata/modules/web_app/`) against the post-0001 schema. They are pre-0001 in *both* import path (`opmodel.dev/core/v1alpha2@v1`) and registry / FQN shape (Module-valued `#registry`, MAJOR-only `@v1` FQNs). The shape rewrite is design implementation that belongs to enhancement 0001's library slice, not Part B's mechanical cleanup. Part B quarantines them (see D10 + tasks 12a.1–12a.4). Mechanical import-path rewires alone (e.g. `core/v1alpha2@v1` → `core@v0` without re-shaping `#registry`) would not produce a unifiable fixture — they would just fail at a different evaluation point — so Part B does not perform partial rewires either.
- Repackage `library/modules/opm` into the D19 catalog shape or republish it at `opmodel.dev/catalogs/opm@0.1.0`. That is squarely enhancement 0001 D23's scope.
- Provide a backwards-compatibility shim. Every API rename is a hard break.
- Defer deletion behind a feature flag. The package goes; importers either compile against the new shape or fail.

## Decisions

### D1: New target package is `opm/schema`, not folded into `opm/core`
Free functions + package-level path vars live in a new `opm/schema` package.

**Alternatives considered:**
- Fold into existing `opm/core`. Pro: one less package. Con: `opm/core` is about output types (`Compiled`, `Resource`). Adding schema dispatch (Paths, metadata decoders, embed loader) overloads its purpose and forces every existing consumer of `opm/core` (e.g. `opm/compile`) to drag the schema surface along.
- Inline Paths + decoders directly into each consumer (`opm/module`, `opm/platform`). Pro: no shared package. Con: `Paths` is used heavily by `opm/compile/{match,execute}.go` and `opm/helper/platform/compose.go`; duplicating the path constants forks the schema's truth.

A focused `opm/schema` package is the smallest shared surface.

### D2: Paths are package-level `var`s, not a struct
The old `api.Paths` struct grouped 30+ `cue.Path` fields and was returned by `binding.Paths()` on every call. With one schema, the struct adds nothing — every consumer reaches for one or two named fields. Exposing each path as a top-level `var Metadata = cue.ParsePath("metadata")` is more idiomatic Go and removes the indirection through a return value.

**Trade-off:** loses a small amount of "structured collection" feel, but no consumer iterated the struct.

### D3: `loadSchemaValue` cache is package-level `sync.Once`, not per-binding
The old per-binding cache lived on `*v1alpha2.binding` and was justified by "one binding instance per version." With no binding type, the cache becomes a package-level `sync.Once`. The documented invariant ("one Kernel per process, one `*cue.Context` per process") is unchanged; the cache keys implicitly on the first context passed in, same as before.

**Trade-off:** two test runs in the same `go test` invocation share the cache. Mitigation: any test that wants a fresh schema build must use a separate process, which `go test` does anyway for separate `_test.go` packages.

### D4: `apis/core/` is re-synced from the new `core` repo, flat layout
`library/apis/core/v1alpha2/*.cue` is deleted. The contents of `core/src/*.cue` (from the new repo — sources live under `src/`, not at the repo root) land at `library/apis/core/*.cue` directly (package `core`, no `v1alpha2/` subdir). `cue.mod/module.cue` is copied verbatim — the new module identifier is `opmodel.dev/core@v0` per enhancement 0001 D12. `embed.go`'s `go:embed` pattern changes from `v1alpha2/*.cue` to `*.cue`.

The file set on the day of re-sync includes `catalog.cue` and `module_context.cue` — both introduced by enhancement 0001's core slice (D19/D25 and D1–D4) ahead of Part B landing. Part B does not consume either file (Go-side reads paths via `opm/schema`, and neither `#Catalog` nor `#ReleaseIdentity` is referenced by the kernel until 0001's library slice lands). Copy the directory contents wholesale rather than enumerating files — the enumeration drifts every time 0001 adds a file.

**Why a vendored copy at all?**
The library uses the embed for offline schema validation in Go-side code paths (`schema.SchemaValue`, kernel tests, `synth.Release`). Tests run without a CUE registry; the embed is the only source. The alternative — drop the embed and require registry access for every schema lookup — is a bigger architectural call and out of scope here.

**Sync drift risk:** the in-library copy can fall behind the new `core` repo. Mitigation is operational, not architectural: a `task` target (out of scope here) to re-vendor on demand.

### D5: `ReleaseView` interface is kept
`BuildTransformerContext` currently takes a `ReleaseView` (`ReleaseName()`/`Namespace()`/`ModuleFQN()`/...). It exists so the context builder doesn't depend on `opm/module`. With one schema there's no version-dispatch reason to keep it, but the decoupling is still useful: tests in `opm/schema` can fake a release without pulling in `opm/module`.

**Alternative considered:** inline to `*module.Release`. Drops the interface. But forces `opm/schema` to import `opm/module`, which currently imports `opm/api` — and after this change, `opm/module` imports `opm/schema`. Keeping `ReleaseView` keeps the dependency arrow `module → schema`, not the reverse.

### D6: `cmd/flow-inspect` keeps working
The CLI tool tied to `_ "opm/api/v1alpha2"` (init-time registration) and hardcoded `apiVersion: "opmodel.dev/v1alpha2"` in its release-building string. Both go away: drop the blank import, drop the `apiVersion: ...` line from the compiled-in release skeleton. The tool still reads paths but now from `opm/schema` directly.

### D7: No new `Detect`-style helper survives
`apiversion.Detect` read the apiVersion field. With no field, no detection. The kernel previously dispatched on this; now every artifact type is known statically at the call site (Module/Release/Platform). Caller knows what artifact it is loading.

This means `(*Kernel).DetectAPIVersion` is deleted outright. Two tests (`TestKernel_DetectAPIVersion`, `TestKernel_DetectAPIVersion_Unknown`) get deleted with it.

### D8: Loader helpers drop the `apiversion.Version` return
`LoadModulePackage`/`LoadReleasePackage`/`LoadPlatformPackage` currently return `(cue.Value, apiversion.Version, error)`. With version dispatch gone, the second return is dead weight. All three signatures collapse to `(cue.Value, error)`.

Same applies to the kernel wrappers in `opm/kernel/wrappers.go`.

### D9: Release/platform apiVersion-mismatch checks are gone
`kernel/phases.go::Match` and `kernel/compile.go::compileModuleRelease` previously cross-checked `rel.APIVersion == plat.APIVersion`. With both fields gone, the check is impossible *and* unnecessary — both artifacts unify against the same schema definitions. Code deleted.

### D10: Match algorithm structural cleanup
`compile/match.go` has four functions that take `paths api.Paths` only to thread paths through:
- `lookupCandidates(matchersIndex, fqn, paths, ...)` 
- `candidateSatisfied(cand, paths, ...)`
- `pairTransformer(plan, compName, tfFQN, composed, paths, ...)`

Plus `extractComponentSummaries(schemaComponents, b api.Binding)` in `compile/module.go`. All become parameter-less in their paths argument; they reference `opm/schema` package vars directly.

`Match(components cue.Value, plat *platform.Platform, b api.Binding)` becomes `Match(components cue.Value, plat *platform.Platform)`. Caller chain in `kernel/{phases,compile}.go` drops the lookup-then-pass-binding. The `*platform.Platform` argument is unchanged — `MaterializedPlatform` is enhancement 0001's library-slice surface, not Part B's.

### D10: Consumer fixtures are quarantined, not rewritten
The re-sync (D4) lands the post-enhancement-0001 `core` schema: `#Platform.#registry` is path-keyed `[Path]: #Subscription`, `#FQNType` uses SemVer, `#Module` carries `#ctx` instead of `#defines`. The library's existing consumer fixtures predate all of that:

- `library/modules/opm_platform/platform.cue` uses Module-valued `#registry: {opm: {#module: …, enabled: true}}` and imports `opmodel.dev/core/v1alpha2@v1`.
- `library/testdata/modules/web_app/{module,components}.cue` imports `opmodel.dev/core/v1alpha2@v1` and references the `opmodel.dev/modules/opm` catalog at MAJOR-only `@v1` FQNs.
- `opm/kernel/flow_integration_test.go` + `flow_synth_integration_test.go` consume both.

A partial rewrite — change the import path but keep the registry shape — does not produce a unifiable value; it just shifts the failure point. A full rewrite requires authoring against `#Subscription`, `#ctx`, and SemVer FQNs, which is enhancement 0001's library-slice design implementation. Per enhancement 0001 D22, folding design implementation into Part B "defeats reviewability."

**Decision:** Part B quarantines the affected fixtures (delete or guard with an unset build tag) and `t.Skip`s the two integration tests with an explicit pointer to this D10 + 0001's library slice. The test suite stays green; the broken-for-now state is announced in code rather than implicit.

**Alternative considered:** sync `apis/core/` from an earlier core/ tag (pre-0001). Rejected — no such tag exists once 0001's core slice ships, and maintaining a parallel snapshot lineage in the standalone core repo just to keep one library refactor green is more work than the quarantine.

## Risks / Trade-offs

- **[Risk] CUE module identity drift in `apis/core/cue.mod/module.cue`** → Mitigation: copy `cue.mod/module.cue` verbatim from the new `core` repo. The library Go code only uses the embed via the overlay-based `load.Instances` (no module-path resolution required), so the in-library copy's module name only needs to be *consistent with itself*, not with consumer modules.
- **[Risk] Test fixtures contain `apiVersion: "opmodel.dev/v1alpha2"` inline strings.** Removing the field from `#Module` etc. makes the field undefined in the schema; unifying a fixture that *adds* `apiVersion: "..."` will either silently leave it as a free field (likely fine) or fail closed-struct admission (depends on whether `#Module` is closed). → Mitigation: rewrite all test fixtures to drop the inline `apiVersion: "..."` lines as part of the test sweep.
- **[Risk] Sync drift between `library/apis/core/*.cue` and the new `core` repo over time.** → Mitigation: out-of-scope here, but the next change can add a `task vendor:core` target. Note: enhancement 0001's library slice (per D24) deletes `library/apis/core/` entirely and switches the kernel to lazy OCI pull of the core schema via `Kernel.Registry` + `Kernel.CoreVersion`, which retires this drift risk wholesale shortly after Part B ships.
- **[Risk] Quarantined fixtures + skipped integration tests leave a coverage gap until enhancement 0001's library slice lands.** → Mitigation: D10 documents the boundary; `t.Skip` strings point at the unblocking change so the gap is visible at every `task test` run. The Go unit tests in `opm/{module,platform,kernel,compile,helper/*}` exercise every code path Part B touches; the lost coverage is end-to-end flow against the OPM catalog, not the schema-dispatch refactor itself.
- **[Trade-off] One-shot deletion vs. incremental refactor.** Library principle VIII demands tiny batches. This change touches ~13 Go packages plus CUE schema. → Justification: the binding API is a coherent unit. Splitting the deletion across multiple changes leaves the codebase in a half-deleted state with broken cross-references for days. The blast radius is wide but mechanical; every edit is a 1-to-1 rewrite. Compile-time errors guide the work to completion.
- **[Risk] cmd/flow-inspect manual smoke check.** No Go unit tests for it. → Mitigation: build it and run with `-stages module` after the refactor.

## Migration Plan

No external consumers. The migration is purely internal:

1. Land `opm/schema` (additive; nothing else changes).
2. Sync `apis/core/` (deletes `v1alpha2/`, adds flat schema, updates embed). Rebuild — `opm/api/v1alpha2/binding.go` will fail to compile because of the embed pattern change.
3. Rewrite consumers in dependency-leaf order: `opm/module`, `opm/platform`, then `opm/compile`, then `opm/helper/*`, then `opm/kernel`, then `cmd/flow-inspect`.
4. Delete `opm/api` + `opm/apiversion`.
5. Rewrite affected tests; delete tests that asserted the deleted contract.
6. `task fmt && task vet && task lint && task test`.

**Rollback:** `git revert` the change. No on-disk state, no data migration.

## Open Questions

None at this point. User has explicitly authorised the full deletion and confirmed library has no external consumers.
