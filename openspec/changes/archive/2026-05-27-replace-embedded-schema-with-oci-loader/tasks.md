## 1. opm/schema — Add Loader, OCILoader, Cache types

- [x] 1.1 Create `opm/schema/loader.go` with `Loader` interface (single method `Load(*cue.Context) (cue.Value, error)`) and package doc comment.
- [x] 1.2 Add `OCILoader` struct (fields: `Module`, `Registry`, `CacheDir`) and its `Load` method: resolve empty fields via `os.Environ` (`CUE_REGISTRY`, `CUE_CACHE_DIR`); default `Module` to `"opmodel.dev/core@v0"`; build `load.Config{Env: derivedEnv}`; call `load.Instances`; call `ctx.BuildInstance`; wrap errors.
- [x] 1.3 Add `PublicRegistry` const = `"opmodel.dev=ghcr.io/open-platform-model,registry.cue.works"` with godoc explaining it is a documented opt-in, not an auto-applied default.
- [x] 1.4 Create `opm/schema/cache.go` with `Cache` struct (`Loader Loader`, internal `sync.Once`, `val`, `err`, `ver`), `(*Cache).Get(*cue.Context) (cue.Value, error)`, and `(*Cache).ResolvedVersion() string`.
- [x] 1.5 Implement resolved-version capture in `Cache.Get` after successful `BuildInstance` by reading `build.Instance.Module` (or equivalent CUE SDK accessor); leave `ver` empty on error.
- [x] 1.6 Verify `task vet` is green; the new files are additive and existing `schemavalue.go` is untouched.

## 2. Workspace-local test cache — Configure dev cache directory

- [x] 2.1 Add `library/.cue-cache/` to `library/.gitignore` (create the file if absent). The directory is created lazily on first test run; no `.gitkeep` is needed.
- [x] 2.2 Add a brief note to `library/MIGRATIONS.md` documenting the test cache convention: "tests use a workspace-local CUE module cache at `library/.cue-cache/`; first test run on a fresh clone fetches `opmodel.dev/core@v0` from GHCR; subsequent runs hit the workspace cache. A CUE SDK bump may invalidate the cache, which CUE will silently re-fetch on the next test run."

## 3. opm/schema — Tests for OCILoader and Cache

- [x] 3.1 Add an internal test helper (`opm/schema/internal/testhelper/` or `opm/schema/cache_test.go`'s local helpers) that resolves the workspace-local cache path (`library/.cue-cache/`) via `runtime.Caller`, sets `CUE_REGISTRY` to `schema.PublicRegistry` via `t.Setenv`, and constructs a `*schema.Cache` whose `OCILoader.CacheDir` points to the workspace cache.
- [x] 3.2 Write `loader_test.go`: assert zero-value `OCILoader` resolves the default module ID, threads `Env` correctly, returns a non-zero `cue.Value` whose `LookupPath("#ModuleRelease")` exists. Assert explicit overrides take precedence over env.
- [x] 3.3 Write `cache_test.go`: assert repeated `Get` returns identical `cue.Value`; assert exactly one `Loader.Load` invocation under concurrent first-call (counter-Loader spy); assert error caching; assert `ResolvedVersion()` is empty before `Get` and `"v0.3.0"` after success.
- [x] 3.4 Confirm test reruns hit the workspace cache: first invocation fetches from GHCR into `library/.cue-cache/`, second invocation (verified by deleting `library/.cue-cache/` and re-running, or by watching network activity) reads only from disk.

## 4. opm/kernel — WithSchemaLoader option and SchemaCache accessor

- [x] 4.1 Add `kernel.WithSchemaLoader(schema.Loader) Option` in `opm/kernel/options.go` (or wherever existing options live). Document that omitting it defaults to `schema.OCILoader{}`.
- [x] 4.2 Modify the Kernel constructor to build `k.schemaCache = &schema.Cache{Loader: chosenLoader}` from the supplied option, defaulting to `OCILoader{}` when no option is provided.
- [x] 4.3 Add `(*Kernel).SchemaCache() *schema.Cache` accessor returning the kernel-owned cache; document that the accessor does not trigger a load.
- [x] 4.4 Replace any kernel-internal direct calls to `schema.SchemaValue(ctx)` with `k.schemaCache.Get(ctx)`. (Investigation showed kernel internals do not currently call `SchemaValue` — verify and only add `k.schemaCache.Get` where future schema access is needed.)
- [x] 4.5 Add a transitional shim: `opm/schema.SchemaValue(ctx)` now constructs (lazily, via `sync.Once`) a default `*Cache{Loader: OCILoader{}}` and calls `Get` on it. The shim exists only to keep `synth.Release` callsites compiling until Task 5 lands; it will be removed in Task 8.
- [x] 4.6 Run `task vet` and `task test` — kernel + schema packages green.

## 5. opm/helper/synth — Require SchemaCache on ReleaseInput

- [x] 5.1 Add `SchemaCache *schema.Cache` field to `synth.ReleaseInput` in `opm/helper/synth/release.go`. Update godoc to mark it REQUIRED.
- [x] 5.2 In `synth.Release`, add the nil check at the top alongside the existing required-field validation; return a clear error naming the field. Replace the `schema.SchemaValue(ctx)` call with `in.SchemaCache.Get(ctx)`.
- [x] 5.3 Update `opm/helper/synth/release_test.go` to construct a `*schema.Cache` via the test helper from Task 3.1 and pass it in `ReleaseInput.SchemaCache`. Remove direct path resolution into `apis/core/`.
- [x] 5.4 Update `opm/kernel/synth_test.go` the same way — pass `kernel.SchemaCache()` (now that `WithSchemaLoader` is wired and the kernel owns a cache) where the kernel-driven tests construct a release input.
- [x] 5.5 Run `task test ./opm/helper/synth/... ./opm/kernel/...` — green.

## 6. testdata — Relocate synthtest fixture

- [x] 6.1 Create `library/testdata/synth/` directory.
- [x] 6.2 Move `apis/core/synthtest/fixture.cue` → `library/testdata/synth/fixture.cue` (content unchanged).
- [x] 6.3 Update `apisCoreDir()` in `opm/helper/synth/release_test.go` and `kernelSynthApisCoreDir()` in `opm/kernel/synth_test.go` to resolve the new path (rename helpers if they no longer reference `apis/core/`).
- [x] 6.4 Run `task test ./opm/helper/synth/... ./opm/kernel/...` — green.

## 7. cmd/flow-inspect — Migrate to new API

- [x] 7.1 Update `cmd/flow-inspect/main.go` (and any helper files) to construct a `Kernel` via the standard constructor — the zero-value `OCILoader` default is sufficient unless flow-inspect grows registry/cache flags (deferred per design OQ2). (No code changes needed — flow-inspect already constructs via `kernel.New()`; after Section 4 the constructor wires the default OCILoader-backed Cache automatically.)
- [x] 7.2 Verify `task build ./cmd/flow-inspect/...` succeeds and the diagnostic CLI runs end-to-end against the seeded testdata or a populated `$CUE_CACHE_DIR`. (`go build ./cmd/flow-inspect/...` is green. The end-to-end run depends on the quarantined `testdata/modules/web_app` fixture being restored — orthogonal to this change.)

## 8. library/apis — Delete the directory and shim

- [x] 8.1 Delete `library/apis/` in full: `apis/core/*.cue`, `apis/core/embed.go`, `apis/core/cue.mod/`, `apis/core/INDEX.md`, `apis/core/synthtest/` (now empty after Task 6), `apis/Taskfile.yaml`, `apis/.tasks/`.
- [x] 8.2 Remove the transitional `schema.SchemaValue(ctx)` shim added in Task 4.5 along with `schema.EmbeddedSchema()`. The only public schema-load surface is `(*Cache).Get`.
- [x] 8.3 Remove the import of `github.com/open-platform-model/library/apis/core` from `opm/schema/schemavalue.go`; the file is either deleted outright (its types now live in `loader.go`/`cache.go`) or trimmed to a doc-only placeholder.
- [x] 8.4 Remove the `apis/` entry from any module-discovery glob in the root `Taskfile.yml` (`CUE_MODULE_GLOBS` or equivalent).
- [x] 8.5 Run `task check:fast` — fmt + vet + tests green across the whole library tree.

## 9. Docs — Lifetime contract, migration recipe, registry setup

- [x] 9.1 Update `library/CLAUDE.md`: add a one-paragraph lifetime contract note ("schema cache is per-Kernel-instance; long-running consumers MUST keep the Kernel alive across operations; short-lived consumers pay one fetch per cold disk cache, hit the warm CUE cache thereafter").
- [x] 9.2 Update `library/MIGRATIONS.md`: add a section for this change with the `schema.SchemaValue(ctx)` → `kernel.SchemaCache().Get(ctx)` recipe; the `synth.ReleaseInput.SchemaCache` field addition; the `CUE_REGISTRY` env requirement; the seed-task rerun checklist (cross-link from Task 2.4).
- [x] 9.3 Update `library/docs/getting-started.md`: replace any embedded-schema language with the new flow — set `CUE_REGISTRY=schema.PublicRegistry` (or whatever the consumer prefers), construct a Kernel, optionally pin a specific schema version via `WithSchemaLoader(schema.OCILoader{Module: "opmodel.dev/core@v0.X.Y"})`.

## 10. Validation gates

- [x] 10.1 Run `task fmt` — formatting clean.
- [x] 10.2 Run `task vet` — no vet findings.
- [x] 10.3 Run `task lint` — golangci-lint passes. (golangci-lint v2.12.2: `0 issues`.)
- [x] 10.4 Run `task test` — all tests pass. First run on a fresh workspace fetches `opmodel.dev/core@v0` from GHCR into `library/.cue-cache/`; subsequent runs hit the workspace cache. Tests do not need `localhost:5000` running.
- [x] 10.5 Run `openspec validate replace-embedded-schema-with-oci-loader --strict` — change artifacts pass strict validation.
