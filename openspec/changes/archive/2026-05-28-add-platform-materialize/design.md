## Context

Core `@v0.3.0` made `#Platform.#registry` a path-keyed map of `#Subscription` values and downgraded `#composedTransformers` / `#matchers` to optional, kernel-filled slots. The library has the OCI substrate to load a single CUE module (`opm/schema.OCILoader` + `Cache`, shipped by `replace-embedded-schema-with-oci-loader`) but nothing that resolves *subscriptions* — enumerating published versions of a catalog path, filtering by policy, pulling the survivors, and indexing their transformers. This change adds that as `Materialize`, the additive foundation `rewrite-match-materialized` consumes.

Constraints that shape the design:

- **Principle I (kernel neutrality / determinism)** — no hidden env lookups, no kernel-held process cache, I/O at edges with caller-supplied config.
- **Principle V (CUE-native resolution)** — use `cuelang.org/go/mod` + `cue/load`, not a hand-rolled OCI client.
- **Principle VII (YAGNI)** — minimal public surface; the cache is opt-in, not baked into the kernel.

## Goals / Non-Goals

**Goals:**

- A `Materialize` flow that turns a `*platform.Platform` into a `*MaterializedPlatform` whose CUE value answers the same `#composedTransformers` / `#matchers.{resources,traits}` paths the matcher already reads.
- Go-side SemVer `range`/`allow`/`deny` resolution (D10/D11).
- A `MaterializeError` carrying a `kind` discriminator (`"catalog"` | `"core-schema"`, D24).
- An opt-in `MaterializeCache` (interface + reference LRU + key derivation) that lives outside the kernel (D14).
- Full unit-testability via `modregistrytest`, with no test-only `Loader` backdoor in the public API.

**Non-Goals:**

- The match-algorithm rewrite and the `*MaterializedPlatform` signature swap (→ `rewrite-match-materialized`).
- The catalog repackage to the D19 `#Catalog` shape (independent; tests here use inline fixtures).
- Any kernel-held cache or eager materialization at `Kernel` construction.

## Decisions

### D1: `MaterializedPlatform` lives in `opm/materialize`, wrapping the source `*Platform`

```go
type MaterializedPlatform struct {
    Source   *platform.Platform        // the spec it was realized from
    Package  cue.Value                 // Source.Package with #composedTransformers + #matchers filled
    Resolved map[string]string         // subscription path → resolved SemVer (diagnostics)
}
```

**Why here, not `opm/platform`:** `Materialize` is the producer; the `platform` package should stay about the *input* artifact. `compile` (the future matcher consumer) already imports `platform`; having it import `materialize` for the materialized view introduces no cycle (`materialize` imports `platform`, never `compile`). Keeping the type next to its producer avoids `platform` taking a dependency on registry/OCI concerns.

**Alternative considered:** a bare `cue.Value` with no Go wrapper. Rejected — the resolved-version map and the `Source` back-reference are needed for diagnostics (`MaterializeError`, later `MissingFQN.alternatives`), and a named type documents the "sealed, post-realization" contract.

### D2: Build the filled value via `FillPath` onto a copy of `Source.Package`

The matcher reads `Package.LookupPath(schema.ComposedTransformers)` and `schema.MatchersResources/Traits`. To keep `rewrite-match-materialized` a minimal diff, `MaterializedPlatform.Package` MUST answer those exact paths. `Materialize` builds the composed map + reverse index as `cue.Value`s and `FillPath`s them onto `Source.Package` at `#composedTransformers` / `#matchers`.

**Open risk (spike first):** `#composedTransformers` / `#matchers` are *optional* fields on the closed `#Platform`. `FillPath` onto optional slots of a closed struct can behave unexpectedly. If it does not take, the fallback is to expose the index through accessor methods on `MaterializedPlatform` and adjust the matcher to read those instead of `LookupPath` — a larger Slice 2 diff, so confirm `FillPath` works before committing.

**Spike outcome (1.1 — RESOLVED, D2 holds):** Verified against `core@v0.3.0` loaded from the workspace cache (`opm/materialize/spike_test.go::TestSpike_FillPathOntoOptionalClosedSlots`). The spike builds a concrete `#Platform`, then fills two slots on it: a one-entry composed map at `schema.ComposedTransformers` (keyed with a `cue.MakePath(cue.Str(fqn))` string-label selector) and a `{resources,traits}` struct at `schema.Matchers`. The fill returns **no error**, and the matcher's own path constants read the content back: `LookupPath(schema.ComposedTransformers)` contains the FQN entry, and `LookupPath(schema.MatchersResources)` contains the reverse-index entry. `FillPath` is non-mutating — the source value's slot stays empty. The single-filled-value approach (D2) stands; the accessor-method fallback is not needed. Key requirement confirmed: all catalog builds MUST be built with the **same** `*cue.Context` that owns the platform value, or the filled values cross contexts.

### D3: Enumerate with `modregistry`, pull with `cue/load`

```
modconfig.NewResolver(&modconfig.Config{Env: env})         // env carries CUE_REGISTRY
  → modregistry.NewClientWithResolver(resolver)
  → client.ModuleVersions(ctx, "<subscription path>")      // sorted SemVer list
  → [Go-side filter: range ∧ allow ∧ deny]                 // D10/D11
  → per survivor: load.Instances(["<path>@<exact-ver>"],
        &load.Config{Env: env}) → ctx.BuildInstance(...)    // reuse OCILoader pattern
  → read #Catalog.metadata + #Catalog.#transformers
```

**Why split enumeration from pull:** `load.Instances` cannot answer "which versions exist?" — that is a registry tag-list query (`ModuleVersions`). Pulling each survivor as a fully-qualified `path@semver` reuses exactly the `OCILoader.loadVersioned` mechanism (`Config.Env`, no `os.Setenv`), so registry config threads through `Kernel.Registry` → `Config.Env` the same way the schema loader already does.

**Alternative considered:** `client.GetModule` + manual zip extraction + manual build. Rejected — `cue/load` already does extraction and build, and consistency with the schema loader matters (Principle V: one place for CUE plumbing).

### D4: SemVer filtering with `Masterminds/semver/v3`, normalizing the `v`-prefix

`range` is a constraint expression CUE cannot evaluate; parse it with `Masterminds/semver/v3`. **Normalization boundary:** CUE module versions carry a `v` prefix (`v0.1.0`), while the catalog FQN SemVer per D5 is bare (`0.1.0`). `ModuleVersions` returns the module form — **verified**: the spike's `ModuleVersions` returned `[v0.1.0 v0.2.0]` (see Research & Decisions). The filter MUST strip/handle the `v` prefix consistently (Masterminds tolerates a leading `v`). Resolution order is `range` restricts → `allow` force-includes → `deny` force-excludes (D10). Note the spike confirmed *enumeration* only; the filter itself is untested code.

### D5: `MaterializeError` with a `kind` discriminator

```go
type MaterializeError struct {
    Kind         string // "catalog" | "core-schema"   (D24)
    Subscription string // subscription path; empty when Kind=="core-schema"
    Version      string // resolved/attempted version
    Cause        error
}
```

`Materialize` emits `Kind: "catalog"`. The `"core-schema"` value is reserved so schema-load failures (from the OCI loader) can surface through the same shape later; this change defines the type but only produces the catalog kind.

### D6: Opt-in cache in `opm/materialize/cache`, kernel holds none

```go
type MaterializeCache interface {
    Get(key string) (*MaterializedPlatform, bool)
    Put(key string, mp *MaterializedPlatform)
}
```

Reference LRU implementation + a key derivation that hashes the canonicalized `#registry` subtree (the input that fully determines materialization). The `Kernel` does NOT hold a cache (Principle I, D14); the operator keys invalidation on CR generation, the CLI opts out and relies on CUE's on-disk module cache. Unlike `schema.Cache` (a single-value `sync.Once` memo), this is a multi-entry keyed cache — different shape, separate type.

### D7: `Kernel.Materialize` is a thin wrapper over a free function

```go
// opm/materialize
func Materialize(ctx context.Context, owner CueContextOwner, registry string, p *platform.Platform) (*MaterializedPlatform, error)

// opm/kernel
func (k *Kernel) Materialize(ctx context.Context, p *platform.Platform) (*MaterializedPlatform, error) {
    return materialize.Materialize(ctx, k, k.registry, p)
}
```

Keeps the realization logic in `materialize` (testable without a full `Kernel`) while giving callers the ergonomic `k.Materialize(...)`. `Kernel` gains a `registry` field + `WithRegistry(string)` option; absent the option it inherits the process `CUE_REGISTRY` (no auto-applied default — same stance as the schema loader).

## Research & Decisions

### modregistrytest substrate + fixture layout (verified by spike)

**Context**: D3/D4 and the whole test strategy assume an in-process OCI registry can stand in for GHCR, so `Materialize` can be built and tested before the catalog repackage exists. The `modregistrytest` *API* was confirmed (`New`, `Host`, `Close` in v0.16.1), but the fixture content layout and the end-to-end enumerate→pull→read flow were unverified.

**Explored**: A throwaway Go test (since removed) run inside the library module against the pinned `cuelang.org/go v0.16.1`, fully offline (`GOPROXY=off`, in-process `httptest` registry). It pushed two versions of a stand-in catalog module plus a consumer module that imports it, then enumerated, pulled, and read fields.

**Confirmed findings**:

- `modregistrytest.New(fsys fs.FS, prefix string) (*Registry, error)` builds a live OCI registry from a filesystem; `(*Registry).Host()` returns the host, `(*Registry).Close()` tears it down. An `fstest.MapFS` is a sufficient `fsys`.
- **Fixture layout**: one top-level directory per `(module, version)`, named `<modulePath with "/"→"_">_<vX.Y.Z>` (e.g. `test.example_cat_v0.1.0/`). Each holds `cue.mod/module.cue` (must carry both `module:` and `language: version:`) plus the package files. A module that imports another MUST declare the dep in its own `cue.mod/module.cue` (`deps: "<path>@v0": v: "v0.1.0"`) for transitive resolution to succeed.
- **Enumeration**: `modregistry.NewClientWithResolver(modconfig.NewResolver(&modconfig.Config{})).ModuleVersions(ctx, "test.example/cat@v0")` returned `[v0.1.0 v0.2.0]` — `v`-prefixed, SemVer-sorted. This is the primitive a warm on-disk cache cannot supply (it answers a registry tag-list query, not a cache read).
- **Pull + read**: with `CUE_REGISTRY=<Host()>+insecure`, `load.Instances(["test.example/cat@v0.1.0"], &load.Config{})` → `ctx.BuildInstance(...)` resolved the exact version; reading a top-level field and iterating the `transformers` map's FQN keys both worked. Distinct versions resolved to distinct content (`@v0.1.0` vs `@v0.2.0`).
- **Transitive import** across two modules resolved through the same registry — the "catalog imports `opmodel.dev/core@v0`" shape works in principle.
- **Cache-cleanup gotcha**: CUE writes module-cache extracts read-only (0444); pointing `CUE_CACHE_DIR` at a fresh `t.TempDir()` per test fails the automatic cleanup with `permission denied`. Tests MUST chmod-walk the cache dir writable before `RemoveAll`. Ship one shared test helper for this (the spike used a `cacheDir(t)` that registered such a cleanup).

**Rationale**: Confirms the enumerate-with-`modregistry` / pull-with-`cue/load` split (D3) on real infrastructure, the `v`-prefix normalization boundary (D4), and that no test-only `Loader` backdoor is needed — the production resolver→client→loader path runs unchanged; only `CUE_REGISTRY` differs.

**Still unverified — carry into implementation, do not assume resolved**:

- The spike used a *simplified* stand-in catalog (`transformers: {<fqn>: {...}}`), **not** the real `c.#Catalog` shape (`M=metadata`, pattern-stamped transformer subpaths). Indexing the real `#Catalog.#transformers`, and importing `opmodel.dev/core@v0` into the fixture, is not yet exercised.
- The Masterminds `range`/`allow`/`deny` *filter* itself was not run — only the enumeration that feeds it. The filter is Go code still to write and test.
- `FillPath` onto the optional, closed `#composedTransformers` / `#matchers` slots (D2 / Q1) was **not** part of this spike. It remains the open item to spike before committing to D2's single-filled-value approach.

## Risks / Trade-offs

- **`FillPath` onto optional closed-struct slots may not recompute** → spike with a 10-line test before building the rest; fall back to accessor-method exposure if needed (D2).
- **`v`-prefix mismatch between CUE module versions and catalog FQN SemVer** → normalize at the filter boundary; cover with a test pushing `v0.1.0`/`v0.2.0` and asserting bare-SemVer FQNs index correctly (D4). Enumeration confirmed to return the `v`-prefixed form (Research & Decisions).
- **`modregistrytest` fixture ergonomics** → ✓ **resolved by spike** (Research & Decisions): `fstest.MapFS`, one dir per `(module, version)` named `<path with "/"→"_">_<vX.Y.Z>`, each with a `cue.mod/module.cue` carrying `module:` + `language: version:`; importing modules need explicit `deps:`. Remaining test-infra item: a shared chmod-walk cache-cleanup helper (CUE writes extracts 0444).
- **Combinatorial pulls under a wide `range`** → bounded by the subscription filter; the opt-in cache amortizes repeated materializations; CUE's on-disk cache amortizes the OCI fetch even without it.
- **New external dependency (`Masterminds/semver/v3`)** → justified: CUE has no range evaluation; Masterminds is the de-facto Go constraint library. One dep, one concern (D4).

## Migration Plan

Additive — no caller migration. Existing `Match` / `Plan` / `Compile` keep taking `*platform.Platform`. New surface (`Materialize`, `WithRegistry`, `MaterializedPlatform`, `MaterializeError`, the cache package) is opt-in. `cli/` and `opm-operator/` adopt it when `rewrite-match-materialized` flips the phase signatures.

## Open Questions

- **Q1:** ~~`MaterializedPlatform.Package` as a single filled value (D2) vs. accessor methods.~~ **RESOLVED** by the 1.1 spike (see D2 § Spike outcome): single filled value works; `MaterializedPlatform.Package` is the filled `cue.Value`.
- **Q2:** Cache key over `#registry` only, or the whole platform spec? Lean `#registry` (it alone determines the materialized output); revisit if a diagnostic needs `metadata.name` in the key.
- **Q3:** ~~Should `Materialize` fail-fast on the first bad subscription, or accumulate?~~ **RESOLVED for this slice: fail-fast.** `Materialize` returns the first `MaterializeError` it hits (unresolvable path, filter conflict, pull failure, divergent FQN). Accumulation across subscriptions is deferred — it pairs naturally with the `Match`-side `MissingFQN` accumulation in `rewrite-match-materialized`, and adding it later is a non-breaking change to the error surface (single `*MaterializeError` → a joined error still satisfies `errors.As`).
