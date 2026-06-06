## 1. Helper: synth.PlatformInput + sentinels

- [x] 1.1 Add `PlatformInput`, `SubscriptionSpec`, `FilterSpec` types in new `opm/helper/synth/platform.go` (fields per design D3; `Enable *bool`)
- [x] 1.2 Add platform sentinel errors with `synth.Platform:` wording: `ErrMissingType` plus platform-scoped missing-name / missing-schema-cache / schema-unavailable (per design D4)
- [x] 1.3 Document the `Enable *bool` "omitted defers to schema default" semantics in the field doc comment

## 2. Helper: synth.Platform

- [x] 2.1 Implement `Platform(ctx *cue.Context, in PlatformInput) (cue.Value, error)` with required-input guards returning sentinels before any schema fetch
- [x] 2.2 Resolve schema via `in.SchemaCache.Get(ctx)`; return `ErrSchemaUnavailable` when `#Platform` is absent
- [x] 2.3 Render CUE source (`platform: { #Platform, metadata, type, #registry }`) and `CompileString` with the schema package as `cue.Scope` (no `userModule` overlay — per design D2)
- [x] 2.4 Render subscriptions: emit `enable` only when `Enable != nil`; emit `filter.{range,allow,deny}` only when non-empty (mirror `writeStringMap` style)
- [x] 2.5 Return the `release`-equivalent `platform` lookup value; leave `#composedTransformers` / `#matchers` unset

## 3. Kernel: SynthesizePlatform

- [x] 3.1 Add `Kernel.SynthesizePlatform(ctx, synth.PlatformInput) (*platform.Platform, error)` in `opm/kernel/synth.go`
- [x] 3.2 Default `in.SchemaCache` to `k.schemaCache` when nil (mirror `SynthesizeRelease`)
- [x] 3.3 Chain `synth.Platform(k.cueCtx, in)` → `NewPlatformFromValue`; do NOT call `Materialize`
- [x] 3.4 Wrap errors with `Kernel.SynthesizePlatform:` prefix

## 4. Docs

- [x] 4.1 Update `opm/helper/synth/doc.go` to cover Platform alongside Release (remove "ModuleRelease only" framing; name `SynthesizePlatform` as the recommended entry point)
- [x] 4.2 Add an additive-surface entry to `library/MIGRATIONS.md` (new public API, no break)
- [x] 4.3 Update `library/CLAUDE.md` synth one-liner in the Repository Layout block to mention Platform

## 5. Tests

- [x] 5.1 Unit tests for `synth.Platform`: minimal valid platform; missing-name/type/cache sentinels via `errors.Is`; schema-without-#Platform sentinel
- [x] 5.2 Subscription/filter tests: filter range present; `Enable` nil → default true; `Enable` false → false; invalid catalog path → unification error
- [x] 5.3 Assert `#composedTransformers` / `#matchers` unset on synthesized value
- [x] 5.4 Kernel test for `SynthesizePlatform`: decoded `Metadata.{Name,Type}`; nil-cache defaulting; no registry I/O
- [x] 5.5 Integration test: `SynthesizePlatform` output feeds `Kernel.Materialize` identically to a file-loaded platform (mirror `flow_synth_integration_test.go`)

## 6. Verify

- [x] 6.1 `task fmt && task vet`
- [x] 6.2 `task test` (and `task check` before merge)
- [x] 6.3 Confirm `cli/` and `opm-operator/` still build (additive surface, no break)
