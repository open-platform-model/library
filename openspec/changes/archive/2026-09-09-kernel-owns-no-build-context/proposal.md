## Why

Three facts on `main` today:

1. **The Kernel's `cue.Context` only grows.** `kernel.New` creates one context (`opm/kernel/kernel.go:59`) and every verb except `Render` builds into it: the loader (`internal/loader.LoadDir(k.cueCtx, ...)`), synthesis (`synth.go:120`), the source helpers (`source_loader.go:21,60`) and the schema cache (`synth.go:166`). cue v0.17.1's runtime records every built package in its `imports` and `importsByBuild` maps (`internal/core/runtime/imports.go:85-118`, filled by `AddInst`) and nothing removes an entry. Measured once on 2026-09-05 against the warm workspace cache: `AcquirePlatformFromDir` with the result dropped adds about 37 MB per call while the Kernel lives, a module acquire plus `SynthesizeInstance` about 14 MB, and the heap returns to 1 MB only when the Kernel is dropped. The operator keeps one Kernel for the process (`opm-operator/cmd/main.go:253`), so every platform generation and every render reconcile adds to a heap that never shrinks. ADR-005's retention figure measured the render context, not this one.
2. **The shared context serialises every verb.** A CUE runtime is not safe for concurrent builds, so the operator wraps every acquire and synth in `Store.kernelMu` (`internal/platform/store.go:67-71`) and the library documents one Kernel per goroutine (`opm/kernel/doc.go:37-39`, `CLAUDE.md` § Render contract). `Render` needs neither, because it already creates and drops its own context (ADR-005).
3. **The context leaks into the API.** Values from different contexts must not be unified, so anything that will meet a kernel-built value has to be compiled in the Kernel's context: `kernel.Source` carries a `cue.Value` that "MUST have been compiled with `cue.Filename(Origin)`" in that context, `Kernel.CueContext()` exists for it (used at `cli/internal/cmd/module/vet.go:169`, `cli/internal/cmdutil/publish.go:53,71`, `opm-operator/cmd/main.go:358`, `opm-operator/internal/render/kernel_module_renderer.go:112`), and `schema.Cache.Get` takes the caller's context.

This is slice 6a of the simplification plan reviewed on 2026-09-05: ADR-005's shares-nothing rule extended from `Render` to every verb. It is breaking, so it rides the re-pin wave the cli and the operator already owe for slices 2, 4 and 5.

**Scope statement (Principle VIII).** One idea, three packages: `opm/kernel` (the context, the source type, the verbs), `opm/schema` (the cache's context), `opm/internal/*` (signatures unchanged; they already take a context). No verb changes what it computes.

## What Changes

**`opm/kernel` (BREAKING, `refactor(kernel)!:`):**

- **BREAKING** `Kernel` holds no `cue.Context` and `Kernel.CueContext()` is removed. Every verb creates its own context with `cuecontext.New()`, builds in it, and lets it go when it returns; an artifact's `Package` value keeps that verb's runtime alive for as long as the caller holds the artifact, and nothing else does.
- **BREAKING** `kernel.Source` becomes `{Origin string; Data []byte}`. `LoadSourceFromFile` reads the file and `LoadSourceFromBytes` wraps the bytes; both parse for syntax and evaluate nothing. Each verb compiles the sources it receives in the context of the schema they meet, with `cue.Filename(Origin)`; a file-backed source is loaded through cue/load at the file's directory so imports resolve as they do today, and the `values:` auto-unwrap happens at that point. No consumer reads `Source.Value` today.
- `ValidateConfigDetailed(schema, sources)` compiles the sources in `schema.Context()`; `AcquireInstanceFromDir` and `SynthesizeInstance` compile theirs in the verb's own context before rendering the values file.
- The goroutine contract inverts: one Kernel is safe for concurrent use across its method calls; the documentation recommends one Kernel per process.

**`opm/schema` (BREAKING):**

- **BREAKING** `(*Cache).Get()` takes no context. The cache creates a private context on first use, hands it to `Loader.Load(ctx)` (whose signature is unchanged) and never exposes it; a caller that must compile against the schema uses the returned value's `Context()`.

**Docs:** `opm/kernel/doc.go` (the goroutine section rewritten, the `CueContext` section deleted), `CLAUDE.md` (Render contract concurrency sentence, Kernel API surface, schema cache lifetime), `README.md`, `docs/getting-started.md`; `adr/007-shares-nothing-verbs.md` records the rule for every verb and amends ADR-005's scope.

**Not in this change:** what acquisition evaluates (`core-version-without-schema-build`), how the render stage serves overlay sources (`overlay-served-in-memory`), the cli's own copies of kernel code (slice 7).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `kernel-runtime`: construction creates no context; the encapsulation requirement is replaced by "the Kernel owns no build context"; the goroutine contract becomes "safe for concurrent use"; `LoadSourceFromFile` reads bytes and the unwrap moves to compile time; the synthesis requirement's context scenario is restated.
- `config-validation`: `Source` carries bytes; the loader helpers parse and do not evaluate.
- `schema-dispatch`: the cache memoizes into a private context and `Get()` takes none.
- `instance-synthesis`: the input requirement's "the kernel owns both" sentence is restated.

## Impact

**SemVer:** MAJOR on the alpha line (Principle VI): a public method and a struct field are removed, a public field changes type, a public method loses its parameter. Pre-GA, so no migration fragment (ADR-004); both consumers migrate in the same PR wave as slices 2, 4 and 5.

**Downstream migration cost, `cli` (3 sites plus tests):**

- `internal/cmd/module/vet.go:169`: values compiled through `k.LoadSourceFromFile` and `ValidateConfigDetailed`; no context needed.
- `internal/cmdutil/publish.go:53,71`: `k.SchemaCache().Get()` and `Context: schemaVal.Context()`.
- Sources are already built with `LoadSourceFromFile` (`internal/workflow/render/values.go:22`); no production `.Value` read exists. Two tests do read it and must go through `ValidateConfigDetailed` instead: `internal/workflow/render/module_test.go` (debugValues fallback) and `internal/workflow/render/render_test.go` (values-file unwrap); `internal/publish/realtree_test.go` and `internal/workflow/render/module_test.go` also call `CueContext()`.
- Target tree: the edits land in the `migrate-kernel-api-and-verdicts` worktree (already on alpha.27 with slices 2, 4 and 5), not the `main` checkout, which pre-dates the wave.

**Downstream migration cost, `opm-operator` (3 files):**

- `cmd/main.go:358`: `k.SchemaCache().Get()`.
- `internal/platform/store.go`: `kernelMu` and `AcquireKernel` are deleted with their three call sites (`platform_controller.go:193`, `kernel_package_renderer.go:99`, `kernel_module_renderer.go:99`); acquire and synth run concurrently like `Render`.
- `internal/render/kernel_module_renderer.go:112`: already leaving with the slice 2 migration (`LoadSourceFromBytes`).
- Target tree: the `migrate-kernel-api-and-verdicts` worktree, as for the cli.

**Library:** `opm/kernel` (`kernel.go`, `source.go`, `source_loader.go`, `validate.go`, `acquire.go`, `synth.go`, `doc.go`), `opm/schema/cache.go`, `opm/internal/schematest` (the test cache helper), docs, one ADR. Tests: 41 `CueContext()` uses in 11 test files compile their values with `cuecontext.New()` or the schema value's context.

**Complexity justification (Principle VII):** net deletion: the accessor, the "compiled in the Kernel's context" contract text, the operator's mutex and gate. Added: one `compileSource` helper of about 30 lines and a three-line private context in `schema.Cache`.
