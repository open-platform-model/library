# Tasks — concurrent-render-recontract

Scope: thread the caller-Kernel `*cue.Context` through the compile pipeline (v0.16.1, behavior-preserving). v0.17 pin + concurrent `-race` test are a **deferred follow-up** (see `design.md` § Deferred) and are intentionally NOT listed here.

## 0. Pre-flight

- [x] 0.1 Confirm no other non-test site derives the build context from the platform: `grep -rn 'Package.Context()\|platform.*\.Context()' opm/ | grep -v _test`. Expect exactly `kernel/compile.go` and `compile/module.go`. Note any extras for `design.md`.

## 1. compile package

- [x] 1.1 `compile/module.go`: add `cueCtx *cue.Context` as the first field of `Module`; change `NewModule` to `NewModule(cueCtx *cue.Context, mp *materialize.MaterializedPlatform, runtimeName string)`.
- [x] 1.2 `compile/module.go:~112`: in `Execute`, replace `cueCtx := r.platform.Package.Context()` with `cueCtx := r.cueCtx`. (`executeTransforms` already takes `cueCtx` as a param — no body change there.)
- [x] 1.3 Update `compile` package tests that call `NewModule` to pass a `*cue.Context` (use the test's existing context, mirroring single-Kernel behavior).

## 2. kernel package

- [x] 2.1 `kernel/compile.go`: make `compileModuleRelease` a method on `*Kernel` (or thread `k.cueCtx` explicitly). Replace `compile.FinalizeValue(mp.Package.Context(), …)` with `compile.FinalizeValue(k.cueCtx, …)`.
- [x] 2.2 `kernel/compile.go`: update the `compile.NewModule(mp, runtimeName)` call to `compile.NewModule(k.cueCtx, mp, runtimeName)`.
- [x] 2.3 `kernel/phases.go`: update the `Kernel.Compile` call site to the method form (`k.compileModuleRelease(ctx, …)`). Confirm `Kernel.Finalize` (already `k.cueCtx`) is untouched and now consistent with the pipeline.

## 3. Behavior assertion (v0.16-safe)

- [x] 3.1 Extend an existing kernel/compile test: after `k.Compile`, assert a rendered value's context is `k.CueContext()` (the `kernel-runtime` delta's observable contract). No new cross-Kernel concurrency test here — that is deferred.
- [x] 3.2 `grep` proves the two former `Value.Context()` sites are gone from the non-test compile path.

## 4. Docs

- [x] 4.1 `MIGRATIONS.md`: add the `compile.NewModule` signature-change entry (leading `*cue.Context`), noting `Kernel` callers are unaffected.
- [x] 4.2 `adr/002-concurrent-render-shared-materialized-platform.md`: status `Proposed → Accepted`; add a one-line note that this change implements the v0.16-landable half and link this OpenSpec change.

## 5. Validation gates

- [x] 5.1 `task fmt`
- [x] 5.2 `task vet`
- [x] 5.3 `task lint`
- [x] 5.4 `task test` — the existing compile + kernel suites MUST pass unchanged (proves behavior preservation on the v0.16.1 pin).
