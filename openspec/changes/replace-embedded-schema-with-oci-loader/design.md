## Context

The OPM core CUE schema is currently shipped as an `embed.FS` (`library/apis/core/embed.go`) co-located with a vendored CUE module (`apis/core/cue.mod/module.cue`). `opm/schema.SchemaValue(*cue.Context)` loads that embed via `cue/load.Instances` with an in-memory overlay and caches the resulting `cue.Value` in a package-level `sync.Once`. Production has exactly one caller of `SchemaValue` (`opm/helper/synth/release.go:87`); two test files reach onto disk for `apis/core/synthtest/fixture.cue` for synth fixtures. Everything else in the kernel uses the Go-side `schema.*` path constants and never touches the embed.

The OPM core schema is now published as an OCI CUE module at `opmodel.dev/core@v0` (currently `v0.3.0` on GHCR under `ghcr.io/open-platform-model/core`). CUE's `cue/load.Instances` natively resolves module IDs against `CUE_REGISTRY` and caches resolved modules on disk under `$CUE_CACHE_DIR` (default `~/.cache/cuelang/mod/`). Cache hits skip the network entirely; misses fetch from the registry; offline + miss returns an error. The OCI artifact format (`application/vnd.cue.module.v1+json` + `application/vnd.cue.modulefile.v1`) is what `cue mod publish` writes, so no oras-go dependency is needed.

This proposal swaps the embed for an OCI-backed Loader behind the same `sync.Once` boundary. The in-process cache layer is unchanged; the on-disk cache layer becomes CUE's responsibility.

## Goals / Non-Goals

**Goals:**

- Delete `library/apis/` entirely.
- Replace the embed-backed `SchemaValue` with an OCI-backed loader that uses CUE's module system.
- Keep schema resolution behind a single in-process `sync.Once` per Kernel instance (no per-call re-fetch).
- Make the Loader configurable (module pin, registry, cache dir) without forcing every caller to configure it.
- Preserve Principle I (kernel neutrality) by routing all I/O config through caller-supplied options or environment.
- Apply Principle V (CUE-native module resolution) literally — the library is the single place CUE plumbing lives.
- Provide a hermetic, deterministic test path that does not require a network or a running local registry.
- Surface the resolved schema version for diagnostics.

**Non-Goals:**

- A pluggable `Loader` ecosystem with multiple public implementations (no `FSLoader`, no `MemoryLoader`). One public Loader: `OCILoader`.
- An embedded build-time mirror of the schema (no `helper/schema/embedded/`). Air-gap consumers are expected to pre-seed `$CUE_CACHE_DIR` or mirror to an internal OCI registry.
- A custom OCI client. CUE's `load.Instances` + `mod/modconfig` is the only fetch path.
- Backwards-compatible deprecation shims. `schema.SchemaValue` and `schema.EmbeddedSchema` are removed outright (consistent with the recent `remove-api-binding-dispatch` posture: library has no published external consumers).
- Per-file or per-definition lazy loading. CUE loads packages atomically; once built, lookups are O(path depth) and effectively free.
- Schema version negotiation or compat-range checking. The Go code in `opm/schema` paths assumes a shape that lives within the `v0` major; if a future `v0` minor changes that shape, that is a separate change.

## Decisions

### D1. Use `cue/load.Instances` against a module ID — not a custom OCI client

The library calls `load.Instances([]string{moduleID}, &load.Config{Env: derivedEnv})`. CUE handles registry resolution, OCI fetching, on-disk caching, and module unzipping.

**Alternatives considered:**

- *Direct oras-go OCI client.* Requires a new dependency, custom cache layout, custom artifact parsing. Reinvents what CUE already does correctly.
- *Lower-level `mod/modregistry.Client`.* Public but marked experimental; tightly coupled to CUE's internal architecture. Adds surface area for marginal control.

**Rationale:** Principle V — CUE-native resolution. Zero new dependencies. The cache layer is CUE's, which means upstream improvements (auth, parallelism, retries) accrue automatically.

### D2. Single `Loader` interface, `OCILoader` is the only public implementation

```go
type Loader interface {
    Load(*cue.Context) (cue.Value, error)
}

type OCILoader struct {
    Module   string // default "opmodel.dev/core@v0"
    Registry string // optional override; if empty, reads CUE_REGISTRY
    CacheDir string // optional override; if empty, reads CUE_CACHE_DIR
}
```

`OCILoader` zero-value works: empty `Module` → `"opmodel.dev/core@v0"`; empty `Registry` → caller's `CUE_REGISTRY` env via `os.Environ`; empty `CacheDir` → caller's `CUE_CACHE_DIR` env or CUE's default.

**Alternatives considered:**

- *Add `FSLoader` for embed/test use.* User explicitly excluded this: "skip FSLoader and only rely on OCILoader". Tests use pre-seeded cache (see D7).
- *`Loader` as a function type (`type Loader func(*cue.Context) (cue.Value, error)`).* Lighter, but forecloses adding methods (e.g., a future `Module() string` for diagnostics). Interface is cheap; keep the option open.
- *Public bypass constructor (`NewCacheFromValue(cue.Value)`).* The name does not enforce intent; first downstream user finds it and uses it in prod to "skip slow OCI fetches", and we end up with two schema-loading code paths. Rejected.

**Rationale:** Principle VII (YAGNI). One Loader keeps the public surface minimal and the SemVer obligation contained. Tests get hermetic behavior through pre-seeded cache, not through a parallel Loader (D7).

### D3. `Cache` is separate from `Loader`

```go
type Cache struct {
    Loader Loader
    once   sync.Once
    val    cue.Value
    err    error
    ver    string // resolved version, populated during Load
}

func (c *Cache) Get(ctx *cue.Context) (cue.Value, error)
func (c *Cache) ResolvedVersion() string // empty until first Get
```

The `Loader` is the *strategy* (how to fetch + build). The `Cache` is the *memoization* (`sync.Once` per instance + resolved-version capture).

**Alternatives considered:**

- *Caching inside `OCILoader`.* Mixes responsibilities; makes "use a fresh loader per request" awkward. Also makes the lifetime contract implicit (every Loader caches; what if you wanted to test the un-cached path?).
- *Package-level cache via `schema.SchemaValue(ctx, opts...)`.* Hidden state; violates Principle I (no package-level singletons hiding behavior).

**Rationale:** Explicit lifetime. One `*schema.Cache` per `Kernel`; tests can construct their own with a different cache dir without sharing process state.

### D4. Default module pin: `opmodel.dev/core@v0` (floating major)

CUE resolves `@v0` to the latest `v0.x.y` available at first-load time. The resolution sticks for the cache's lifetime.

**Alternatives considered:**

- *Source-pinned exact version (`const DefaultSchemaVersion = "v0.3.0"`).* Reproducible across builds; surfaces compatibility issues at library-release time. But requires a coordinated bump every time core publishes.
- *No default — require caller pin.* Most paranoid. Friction on every caller for marginal safety.

**Rationale:** User preference. The library's Go code commits to *shape* compatibility within the v0 major (`#Platform.#registry`, `#FQNType`, `matchers` paths). Within v0, additive schema changes are absorbed transparently; a shape-breaking change in v0 is itself a library-breaking event and should be addressed by the library code, not by a default pin.

**Mitigation for non-determinism:** `Cache.ResolvedVersion()` exposes the actual resolved version after first Load (D8) so operators and CI can record what they got.

### D5. Lazy load on first `Cache.Get`, not eager at `Kernel.New`

`Cache.Get` is sync.Once-guarded. `Kernel.New` does no I/O; the first `Validate`/`Compile`/`Match` that needs schema triggers the fetch.

**Alternatives considered:**

- *Eager at construction (Kernel.New blocks on Load).* Fails fast on bad config but pays first-call cost upfront. For tools that construct a Kernel and never call schema-touching methods (e.g., a `--help` flow), this is wasted work.

**Rationale:** Eager-at-construction does not persist across processes either — the cache is per-instance. The implementer-side contract is "keep the Kernel alive across operations" (documented in `library/CLAUDE.md` per the proposal). Long-running consumers (opm-operator) pay the first fetch once at controller startup; short-lived consumers (cli) pay it on cold disk cache, hit the disk cache thereafter.

### D6. `synth.ReleaseInput.SchemaCache` is REQUIRED, not optional

```go
type ReleaseInput struct {
    // ... existing fields ...
    SchemaCache *schema.Cache // REQUIRED. Typically kernel.SchemaCache().
}
```

`synth.Release` returns an error if `SchemaCache` is nil.

**Alternatives considered:**

- *Optional with self-construct fallback (default `&schema.Cache{Loader: schema.OCILoader{}}`).* Convenient but creates a second cache shadowing the kernel's, leading to duplicate fetches and divergent resolved versions if the caller's Kernel was configured with non-default options.
- *`schema.Cache` as a free function variable / package singleton fallback.* Hidden state. Rejected by Principle I.

**Rationale:** User preference; one cache per process. Forces every caller to be explicit about cache ownership.

### D7. Tests use a pre-seeded CUE module cache under `testdata/cue-cache/`

Test setup sets `CUE_CACHE_DIR=<repo>/library/testdata/cue-cache` (via `t.Setenv` or `OCILoader.CacheDir`), then uses the real `OCILoader`. No test-only Loader exists.

A new `task test:seed-cache` (name TBD) fetches `opmodel.dev/core@v0.<pinned>` from GHCR once and writes the result into `testdata/cue-cache/` in CUE's cache layout. The seeded directory is committed.

**Alternatives considered:**

- *`internal/testschema/` package providing an unexported test Loader.* Effectively re-introduces `FSLoader` under a different name; tests no longer exercise the real `OCILoader` code path.
- *Localhost:5000 registry dependency (matching the existing flow-test pattern).* Adds setup friction; bare `go test` would skip; CI requires registry running just for unit tests.
- *Pull from GHCR on each test run.* Slow, network-dependent, breaks in restricted CI environments.

**Rationale:** Tests exercise the real Loader code path against a hermetic fixture. The fixture *is* the CUE module cache CUE would have produced — there is no parallel test loader to drift from production. CI loader-path coverage can hit GHCR directly because core publishes before the library consumes (no temporal coupling).

### D8. `Cache.ResolvedVersion() string` for diagnostics

After the first `Get`, the Cache captures the resolved module version from `build.Instance.Module` (or equivalent CUE SDK accessor). Before the first `Get`, returns the empty string.

**Alternatives considered:**

- *No introspection.* Pure but unhelpful when "schema v0.3.1 broke X" arrives as a bug report.
- *On the `Loader` interface itself.* Implies every Loader knows what it resolved; `OCILoader.ResolvedVersion()` returning `""` until used is awkward.

**Rationale:** Diagnostic value at near-zero cost. Cache is the natural owner since it owns the post-Load state.

### D9. `schema.PublicRegistry` is a public const, not a default

```go
const PublicRegistry = "opmodel.dev=ghcr.io/open-platform-model,registry.cue.works"
```

The library does NOT auto-set `CUE_REGISTRY` to this. Callers opt in:

```go
os.Setenv("CUE_REGISTRY", schema.PublicRegistry) // cli/main.go startup
```

Or operators set it via pod env, K8s deployment, etc.

**Alternatives considered:**

- *Bake `ghcr.io/open-platform-model` as the `OCILoader` zero-value default.* Library makes a policy decision about where to fetch from. If GHCR moves or becomes unavailable, the library becomes a single point of failure. Also surprising — Principle I says no hidden lookups.

**Rationale:** Library exports a documented value; callers compose policy from that value. No magic, no copy-paste.

### D10. `Kernel.SchemaCache() *schema.Cache` accessor

```go
func (k *Kernel) SchemaCache() *schema.Cache
```

Returns the kernel's owned Cache. Consumers pass it to `synth.ReleaseInput.SchemaCache` and any future helper that needs schema.

**Alternatives considered:**

- *No accessor; synth must construct its own.* Forces duplicate caches if the kernel was configured non-default (e.g., a test that pinned a specific schema version). Synth would silently use the default.
- *`Kernel.SchemaValue(ctx) (cue.Value, error)` instead.* Hides the Cache, prevents `ResolvedVersion()` access from outside the kernel.

**Rationale:** Expose the cache, not its result — callers can do `Get`, query `ResolvedVersion()`, or hold the pointer for repeated use. Aligns with "accept interfaces, return concrete structs" (Principle IV).

### D11. `WithSchemaLoader(Loader)` rather than `WithSchemaCache(*Cache)`

```go
func WithSchemaLoader(l schema.Loader) Option
```

The kernel wraps the supplied Loader in a fresh `*schema.Cache`. Callers cannot inject a pre-built Cache.

**Alternatives considered:**

- *`WithSchemaCache(*schema.Cache)`.* Two callers could pass the same cache into two Kernels and share it — sometimes desired, sometimes a source of bugs. Easy to layer on later if needed; harder to remove.

**Rationale:** YAGNI. One Kernel = one Cache is the simple model. If multi-Kernel cache sharing becomes a real need, `WithSchemaCache` can be added as a non-breaking addition.

## Risks / Trade-offs

**[R1] CUE's module cache layout is not a public contract** → Pin CUE version in `go.mod` (already done at `v0.16.1`). The `task test:seed-cache` re-bakes `testdata/cue-cache/` from GHCR using the same CUE version the tests use, so seed and consumer always agree. Add a probe in test helpers that verifies the expected module path exists under `testdata/cue-cache/` and fails loud (not silently as "module not found") if CUE bumped layout. Record this in `MIGRATIONS.md` as a "bumping CUE? rerun seed task" checklist item.

**[R2] Floating-major default is non-deterministic across processes** → `Cache.ResolvedVersion()` exposes the resolved version. Document in `library/CLAUDE.md` that operators wanting reproducibility set `OCILoader.Module = "opmodel.dev/core@v0.X.Y"`. Surface the resolved version in any diagnostic output the consuming frontend produces (operator status conditions, CLI `--verbose`).

**[R3] Network unavailable on cold cache** → CUE returns a clean load error; we do not silently fall back. Document the warm-cache prerequisite for restricted environments. The operator-shipping pattern is "image build copies a known-good `$CUE_CACHE_DIR` into the runtime image", same as Go module caches in multi-stage builds.

**[R4] Caller forgets to set `CUE_REGISTRY`** → CUE falls back to `registry.cue.works`, which does not host `opmodel.dev/core`; load returns a "module not found" error. Document the `schema.PublicRegistry` opt-in prominently in `library/docs/getting-started.md` and `library/MIGRATIONS.md`. Consider a sanity check on first `Cache.Get` that returns a clearer error if the load fails and `CUE_REGISTRY` is empty — but this risks duplicating CUE's error messaging; resolve in implementation.

**[R5] `apis/core/synthtest/fixture.cue` relocation breaks two test files** → Mechanical: move to `testdata/synth/fixture.cue`, update path resolution. Independent of the loader change, can land in its own commit.

**[R6] `cmd/flow-inspect` startup behavior changes** → Today it constructs a kernel internally (or hits `schema` paths directly). With the loader change, its kernel must be constructed against `CUE_REGISTRY`. Add explicit `--registry` and `--cache-dir` flags, or honor env-driven defaults. Decide in implementation.

**[R7] CUE bumps `mod/modconfig` semantics in a minor** → Locked behavior at `cuelang.org/go v0.16.1`; bumping CUE is itself a coordinated change with its own review. Out of scope for this proposal.

## Migration Plan

The library has no published external consumers (per the recent `remove-api-binding-dispatch` archive), so the migration is in-repo only. Staged for Principle VIII (small batches), each step is a separate commit:

1. **Add new types** — Create `opm/schema/loader.go` (`Loader`, `OCILoader`, `PublicRegistry`) and `opm/schema/cache.go` (`Cache`, `Get`, `ResolvedVersion`). Leave the existing `schemavalue.go` in place. Compiles green.
2. **Wire kernel** — Add `WithSchemaLoader` option, `SchemaCache()` accessor. Kernel's internal schema-using code (if any) starts going through `k.schemaCache`. The old `schema.SchemaValue` package function is reimplemented as a thin shim that constructs a default `OCILoader`-backed cache (transitional only). Compiles green.
3. **Migrate synth** — Update `synth.ReleaseInput` with required `SchemaCache`. Update the two synth callsites (production + tests) to pass it. Tests temporarily use a `schema.Cache{Loader: schema.OCILoader{CacheDir: testdataPath}}` constructed inline.
4. **Seed test cache** — Add `task test:seed-cache` (or equivalent). Run it to populate `testdata/cue-cache/`. Commit the seeded directory. Update test helpers to share a common helper that constructs the test Cache.
5. **Relocate fixture** — Move `apis/core/synthtest/fixture.cue` → `testdata/synth/fixture.cue`. Update path constants in `release_test.go` and `synth_test.go`.
6. **Migrate flow-inspect** — Update `cmd/flow-inspect/` to construct a kernel via the new Loader. Add registry/cache flags if D6 needs them.
7. **Delete `library/apis/`** — Remove the directory in full. Remove the transitional `schema.SchemaValue` shim. Remove the import in `opm/schema/schemavalue.go` (which becomes either deleted or trivially empty).
8. **Docs** — Update `library/CLAUDE.md` (lifetime contract, env setup), `library/MIGRATIONS.md` (SchemaValue → Cache.Get recipe + synth.ReleaseInput.SchemaCache requirement), `library/docs/getting-started.md` (CUE_REGISTRY setup, `schema.PublicRegistry`).
9. **CI** — Add a `task` (or extend existing) that runs OCILoader against GHCR for at least one test path; verify happy path of the new code is exercised against the real registry, not just the seeded cache.

**Rollback:** Each step is a single commit. Up through step 6, the change is reversible by reverting commits. After step 7 (`apis/` deletion), rollback requires re-syncing from `core/src/` — but at that point downstream impact is zero (no external consumers).

## Open Questions

- **OQ1** — Final pinned schema version for `testdata/cue-cache/`. Currently `v0.3.0` is the latest published; will use that unless something newer lands during implementation.
- **OQ2** — Whether to add a `--registry` / `--cache-dir` flag to `cmd/flow-inspect`, or rely on `CUE_REGISTRY` / `CUE_CACHE_DIR` env. Suggested: env-only, consistent with workspace conventions; revisit if usability suffers.
- **OQ3** — Whether `Loader.Load` should accept `context.Context` for cancellation/timeout of the OCI fetch. CUE's `load.Instances` does not take a context. Adding one ourselves would mean wrapping `load.Instances` in a goroutine. Defer until a real cancellation use case appears.
- **OQ4** — `Cache.ResolvedVersion()` semantics before first `Get` — return empty string (proposed), panic, or block waiting for first Get? Empty string is simplest and matches "diagnostic, not authoritative" framing.
- **OQ5** — Whether to add a clearer error wrap when `CUE_REGISTRY` is empty and the default registry returns "module not found". Risks duplicating CUE's error messaging.
