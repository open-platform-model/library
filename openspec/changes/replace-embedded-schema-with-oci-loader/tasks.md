## 1. opm/schema — Add Loader, OCILoader, Cache types

- [ ] 1.1 Create `opm/schema/loader.go` with `Loader` interface (single method `Load(*cue.Context) (cue.Value, error)`) and package doc comment.
- [ ] 1.2 Add `OCILoader` struct (fields: `Module`, `Registry`, `CacheDir`) and its `Load` method: resolve empty fields via `os.Environ` (`CUE_REGISTRY`, `CUE_CACHE_DIR`); default `Module` to `"opmodel.dev/core@v0"`; build `load.Config{Env: derivedEnv}`; call `load.Instances`; call `ctx.BuildInstance`; wrap errors.
- [ ] 1.3 Add `PublicRegistry` const = `"opmodel.dev=ghcr.io/open-platform-model,registry.cue.works"` with godoc explaining it is a documented opt-in, not an auto-applied default.
- [ ] 1.4 Create `opm/schema/cache.go` with `Cache` struct (`Loader Loader`, internal `sync.Once`, `val`, `err`, `ver`), `(*Cache).Get(*cue.Context) (cue.Value, error)`, and `(*Cache).ResolvedVersion() string`.
- [ ] 1.5 Implement resolved-version capture in `Cache.Get` after successful `BuildInstance` by reading `build.Instance.Module` (or equivalent CUE SDK accessor); leave `ver` empty on error.
- [ ] 1.6 Verify `task vet` is green; the new files are additive and existing `schemavalue.go` is untouched.

## 2. testdata — Seed CUE module cache for tests

- [ ] 2.1 Add `library/testdata/cue-cache/.gitkeep` (or equivalent) so the directory exists in version control.
- [ ] 2.2 Add a `Taskfile.yml` task (e.g., `test:seed-cache`) that fetches `opmodel.dev/core@v0.3.0` (current pin) from GHCR via `cue mod tidy` or `CUE_CACHE_DIR=$(pwd)/testdata/cue-cache cue mod fetch`, writing CUE's standard module-cache layout into `testdata/cue-cache/`.
- [ ] 2.3 Run the seed task and commit the resulting `testdata/cue-cache/` contents. Verify the seeded path resolves to a recognizable CUE module-cache layout (e.g., `testdata/cue-cache/mod/extract/opmodel.dev/core@v0/v0.3.0/`).
- [ ] 2.4 Add a brief note to `library/MIGRATIONS.md` documenting "bumping CUE SDK or pinned schema requires re-running `task test:seed-cache`".

## 3. opm/schema — Tests for OCILoader and Cache

- [ ] 3.1 Add an internal test helper (`opm/schema/internal/testhelper/` or `opm/schema/cache_test.go`'s local helpers) that resolves the `testdata/cue-cache/` absolute path via `runtime.Caller` and constructs a `*schema.Cache` whose `OCILoader.CacheDir` points there.
- [ ] 3.2 Write `loader_test.go`: assert zero-value `OCILoader` resolves the default module ID, threads `Env` correctly, returns a non-zero `cue.Value` whose `LookupPath("#ModuleRelease")` exists. Assert explicit overrides take precedence over env.
- [ ] 3.3 Write `cache_test.go`: assert repeated `Get` returns identical `cue.Value`; assert exactly one `Loader.Load` invocation under concurrent first-call (counter-Loader spy); assert error caching; assert `ResolvedVersion()` is empty before `Get` and `"v0.3.0"` after success.
- [ ] 3.4 Confirm tests pass with no network access (`unshare -n go test ./opm/schema/...` or equivalent verification) — proves cache hit on the seeded directory.

## 4. opm/kernel — WithSchemaLoader option and SchemaCache accessor

- [ ] 4.1 Add `kernel.WithSchemaLoader(schema.Loader) Option` in `opm/kernel/options.go` (or wherever existing options live). Document that omitting it defaults to `schema.OCILoader{}`.
- [ ] 4.2 Modify the Kernel constructor to build `k.schemaCache = &schema.Cache{Loader: chosenLoader}` from the supplied option, defaulting to `OCILoader{}` when no option is provided.
- [ ] 4.3 Add `(*Kernel).SchemaCache() *schema.Cache` accessor returning the kernel-owned cache; document that the accessor does not trigger a load.
- [ ] 4.4 Replace any kernel-internal direct calls to `schema.SchemaValue(ctx)` with `k.schemaCache.Get(ctx)`. (Investigation showed kernel internals do not currently call `SchemaValue` — verify and only add `k.schemaCache.Get` where future schema access is needed.)
- [ ] 4.5 Add a transitional shim: `opm/schema.SchemaValue(ctx)` now constructs (lazily, via `sync.Once`) a default `*Cache{Loader: OCILoader{}}` and calls `Get` on it. The shim exists only to keep `synth.Release` callsites compiling until Task 5 lands; it will be removed in Task 8.
- [ ] 4.6 Run `task vet` and `task test` — kernel + schema packages green.

## 5. opm/helper/synth — Require SchemaCache on ReleaseInput

- [ ] 5.1 Add `SchemaCache *schema.Cache` field to `synth.ReleaseInput` in `opm/helper/synth/release.go`. Update godoc to mark it REQUIRED.
- [ ] 5.2 In `synth.Release`, add the nil check at the top alongside the existing required-field validation; return a clear error naming the field. Replace the `schema.SchemaValue(ctx)` call with `in.SchemaCache.Get(ctx)`.
- [ ] 5.3 Update `opm/helper/synth/release_test.go` to construct a `*schema.Cache` via the test helper from Task 3.1 and pass it in `ReleaseInput.SchemaCache`. Remove direct path resolution into `apis/core/`.
- [ ] 5.4 Update `opm/kernel/synth_test.go` the same way — pass `kernel.SchemaCache()` (now that `WithSchemaLoader` is wired and the kernel owns a cache) where the kernel-driven tests construct a release input.
- [ ] 5.5 Run `task test ./opm/helper/synth/... ./opm/kernel/...` — green.

## 6. testdata — Relocate synthtest fixture

- [ ] 6.1 Create `library/testdata/synth/` directory.
- [ ] 6.2 Move `apis/core/synthtest/fixture.cue` → `library/testdata/synth/fixture.cue` (content unchanged).
- [ ] 6.3 Update `apisCoreDir()` in `opm/helper/synth/release_test.go` and `kernelSynthApisCoreDir()` in `opm/kernel/synth_test.go` to resolve the new path (rename helpers if they no longer reference `apis/core/`).
- [ ] 6.4 Run `task test ./opm/helper/synth/... ./opm/kernel/...` — green.

## 7. cmd/flow-inspect — Migrate to new API

- [ ] 7.1 Update `cmd/flow-inspect/main.go` (and any helper files) to construct a `Kernel` via the standard constructor — the zero-value `OCILoader` default is sufficient unless flow-inspect grows registry/cache flags (deferred per design OQ2).
- [ ] 7.2 Verify `task build ./cmd/flow-inspect/...` succeeds and the diagnostic CLI runs end-to-end against the seeded testdata or a populated `$CUE_CACHE_DIR`.

## 8. library/apis — Delete the directory and shim

- [ ] 8.1 Delete `library/apis/` in full: `apis/core/*.cue`, `apis/core/embed.go`, `apis/core/cue.mod/`, `apis/core/INDEX.md`, `apis/core/synthtest/` (now empty after Task 6), `apis/Taskfile.yaml`, `apis/.tasks/`.
- [ ] 8.2 Remove the transitional `schema.SchemaValue(ctx)` shim added in Task 4.5 along with `schema.EmbeddedSchema()`. The only public schema-load surface is `(*Cache).Get`.
- [ ] 8.3 Remove the import of `github.com/open-platform-model/library/apis/core` from `opm/schema/schemavalue.go`; the file is either deleted outright (its types now live in `loader.go`/`cache.go`) or trimmed to a doc-only placeholder.
- [ ] 8.4 Remove the `apis/` entry from any module-discovery glob in the root `Taskfile.yml` (`CUE_MODULE_GLOBS` or equivalent).
- [ ] 8.5 Run `task check:fast` — fmt + vet + tests green across the whole library tree.

## 9. Docs — Lifetime contract, migration recipe, registry setup

- [ ] 9.1 Update `library/CLAUDE.md`: add a one-paragraph lifetime contract note ("schema cache is per-Kernel-instance; long-running consumers MUST keep the Kernel alive across operations; short-lived consumers pay one fetch per cold disk cache, hit the warm CUE cache thereafter").
- [ ] 9.2 Update `library/MIGRATIONS.md`: add a section for this change with the `schema.SchemaValue(ctx)` → `kernel.SchemaCache().Get(ctx)` recipe; the `synth.ReleaseInput.SchemaCache` field addition; the `CUE_REGISTRY` env requirement; the seed-task rerun checklist (cross-link from Task 2.4).
- [ ] 9.3 Update `library/docs/getting-started.md`: replace any embedded-schema language with the new flow — set `CUE_REGISTRY=schema.PublicRegistry` (or whatever the consumer prefers), construct a Kernel, optionally pin a specific schema version via `WithSchemaLoader(schema.OCILoader{Module: "opmodel.dev/core@v0.X.Y"})`.

## 10. Validation gates

- [ ] 10.1 Run `task fmt` — formatting clean.
- [ ] 10.2 Run `task vet` — no vet findings.
- [ ] 10.3 Run `task lint` — golangci-lint passes.
- [ ] 10.4 Run `task test` — all tests pass (with `$CUE_CACHE_DIR` defaulting to `testdata/cue-cache/` for the schema package's tests via the test helper).
- [ ] 10.5 (Optional CI verification) Run a one-off OCILoader test against GHCR directly to confirm the production code path works end-to-end (separate from the seeded-cache tests).
- [ ] 10.6 Run `openspec validate replace-embedded-schema-with-oci-loader --strict` — change artifacts pass strict validation.
