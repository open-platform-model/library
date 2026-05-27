## Why

The library currently embeds the OPM core CUE schema via `go:embed` in `library/apis/core/`. That mirror has to be re-synced by hand from `core/src/` (the source-of-truth `core` repo) whenever the schema changes, ties every library release to a schema re-sync, and bakes schema bytes into the kernel binary that the kernel can fetch directly from the published `opmodel.dev/core@v0` OCI artifact on GHCR. Removing the embed lets the kernel consume the schema the same way every other CUE consumer does — through the CUE module system, against `CUE_REGISTRY`, with CUE's on-disk module cache providing offline-first behavior. This is also the literal application of Principle V (CUE-Native Module Resolution).

## What Changes

- **BREAKING (`opm/schema`)** — Delete `opm/schema/schemavalue.go`. Replace with:
  - `loader.go` — exports `Loader` interface (`Load(*cue.Context) (cue.Value, error)`), `OCILoader{Module, Registry, CacheDir}` as the **only** public `Loader` implementation, and `PublicRegistry` const = `"opmodel.dev=ghcr.io/open-platform-model,registry.cue.works"`.
  - `cache.go` — exports `Cache{Loader Loader}`, `(*Cache).Get(*cue.Context) (cue.Value, error)` (sync.Once-guarded), `(*Cache).ResolvedVersion() string` for diagnostics.
  - `EmbeddedSchema()` removed. `SchemaValue(ctx)` removed; callers use `(*Cache).Get(ctx)`.
- **BREAKING (`opm/kernel`)** — `Kernel` gains a `WithSchemaLoader(schema.Loader)` option. Zero-value default = `schema.OCILoader{}` resolving to module `"opmodel.dev/core@v0"` (floating major), env-driven registry + cache. The `Kernel` holds one `*schema.Cache` per instance and exposes it via a `SchemaCache()` accessor.
- **BREAKING (`opm/helper/synth`)** — `ReleaseInput` grows a REQUIRED `SchemaCache *schema.Cache` field. `Release` no longer constructs its own loader; callers MUST pass a `Cache` (typically `kernel.SchemaCache()`).
- **REMOVED (`apis/`)** — Delete `library/apis/` entirely: `apis/core/*.cue`, `apis/core/embed.go`, `apis/core/cue.mod/`, `apis/core/INDEX.md`, `apis/core/synthtest/`, `apis/Taskfile.yaml`, `apis/.tasks/`.
- **Tests** — `library/testdata/cue-cache/` ships a pre-seeded CUE module cache populated with `opmodel.dev/core@v0.<pinned>`. Test helpers set `CUE_CACHE_DIR` to that path and use the real `OCILoader` — no test-only `Loader` backdoor in the public API. The `synthtest/fixture.cue` fixture relocates to `library/testdata/synth/fixture.cue`.
- **CI** — Loader path coverage targets GHCR directly (no `localhost:5000` dependency for schema loading). The `core` repo publishes before the library consumes, so registry availability is not a temporal coupling risk.
- **Docs** — Add a lifetime-contract note ("long-running consumers keep the Kernel alive for schema cache reuse") to `library/CLAUDE.md` and/or a kernel doc page. Add a `task test:seed-cache` (or similar) that re-fetches the pinned schema version into `testdata/cue-cache/` from GHCR; rerun when CUE version or pinned schema bumps.
- **(`cmd/flow-inspect`)** — Diagnostic CLI migrates to the new `Cache` API; constructs an `OCILoader`-backed `Cache` (or accepts one from a `Kernel` it builds).
- **MIGRATIONS.md** — Add a section documenting the `schema.SchemaValue(ctx)` → `kernel.SchemaCache().Get(ctx)` recipe and the `synth.ReleaseInput.SchemaCache` requirement.

## Capabilities

### New Capabilities

None. This change restructures how `schema-dispatch` resolves the schema; it does not introduce a new capability.

### Modified Capabilities

- `schema-dispatch`: Schema source changes from an embedded `fs.FS` (rooted at `apis/core/`) to an OCI-fetched CUE module resolved via `cue/load.Instances` against `CUE_REGISTRY`. Requirements about `EmbeddedSchema()`, the embed.FS-backed `SchemaValue(ctx)` cache, and the `apis/core` embed location are replaced with requirements about the `Loader` interface, `OCILoader` defaults (`opmodel.dev/core@v0`, env-driven registry/cache), the per-Kernel `Cache` lifetime contract, `ResolvedVersion()` for diagnostics, and the `PublicRegistry` const.
- `schema-testing`: Test schema acquisition changes from on-disk path resolution into `apis/core/` (`apisCoreDir()`, `kernelSynthApisCoreDir()`) to a pre-seeded CUE module cache under `testdata/cue-cache/`. Requirements about overlay loading from `apis/core/synthtest/fixture.cue` are replaced with requirements about `CUE_CACHE_DIR` seeding through a Task and fixture relocation to `testdata/synth/`.
- `release-synthesis`: `ReleaseInput` adds a required `SchemaCache *schema.Cache` field; `synth.Release` no longer self-constructs schema loading.

## Impact

- **Affected packages**: `opm/schema` (rewritten), `opm/kernel` (new option + accessor), `opm/helper/synth` (input shape change), `cmd/flow-inspect` (callsite migration). `opm/compile`, `opm/module`, `opm/platform`, `opm/helper/loader/file`, `opm/helper/platform` are unaffected (they don't touch `SchemaValue`; they use the Go-side `schema.*` path constants which stay).
- **Removed**: `library/apis/` directory in full (Go package + CUE module + Taskfile + INDEX).
- **Affected schemas**: None in `core/`. The OPM schema itself does not change — only how the library acquires it.
- **Downstream consumers**: Library has no published external consumers yet (per the recent `remove-api-binding-dispatch` archive). When `cli/` and `opm-operator/` adopt:
  - MUST set `CUE_REGISTRY` (e.g., to `schema.PublicRegistry`) so the kernel can resolve `opmodel.dev/core@v0`.
  - MUST keep the `Kernel` object alive across operations to reuse the in-process schema cache. Operator: one `Kernel` per controller-manager lifetime. CLI: one `Kernel` per invocation, warm `$CUE_CACHE_DIR` makes that a disk read.
  - MUST pass `kernel.SchemaCache()` into `synth.ReleaseInput.SchemaCache` when calling release synthesis.
- **SemVer**: MAJOR. Breaks `opm/schema` (functions removed), `opm/helper/synth.ReleaseInput` (new required field), and the entire `github.com/open-platform-model/library/apis/core` import path. `opm/kernel` adds a non-required option; default callers compile unchanged.
- **Constitutional alignment**:
  - Principle I (Kernel Neutrality): I/O strategy is caller-supplied via `WithSchemaLoader`. Default still routes through env vars — that is the caller's responsibility to configure, and no hidden lookup is added beyond what `cue/load.Instances` already does for any CUE consumer.
  - Principle V (CUE-Native Module Resolution): direct application — schema acquisition flows through CUE's module system + cache rather than a custom OCI client.
  - Principle VII (YAGNI): one public `Loader` (`OCILoader`). No `FSLoader` or test-only `Loader` exported; tests use pre-seeded cache + the real `OCILoader`.
  - Principle VIII (Small Batch): scope is constrained to `opm/schema` + one kernel option + one synth-input field + the `apis/` deletion + test relocation. Each can be a separate commit during apply.
