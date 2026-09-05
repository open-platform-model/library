# Design: cut-dead-surface

## Context

See `proposal.md` § Why. The symbols leaving are listed there; this document records how each removal is made without changing behaviour a consumer observes, and how the tests that reached the removed symbols are re-homed. Constraints: every removal must leave `cli` and `opm-operator` compiling unchanged against the new alpha; the internal per-source partial pass under `WithValues` (`opm/kernel/acquire.go`) and the `attributeValuesError` re-validation must keep their attribution behaviour; the kernel wrappers `Kernel.NewModuleFromValue` / `Kernel.NewPlatformFromValue` / `Kernel.LoadInstancePackage` keep their signatures.

Test packages are external (`kernel_test`, `module_test`, `synth_test`), so an unexported symbol disappears from them; each removal therefore has a test consequence recorded below.

## Goals / Non-Goals

**Goals:**

- Remove every symbol in the proposal with no change to any render, acquire or synthesize result.
- Keep the internal partial-validation pass and its per-source attribution intact.
- Land as one PR whose only downstream effect is a version bump.

**Non-Goals:**

- No new verb, option or type (that is slice 2: `AcquireModuleFromDir`, loader fold, one registry knob).
- No change to the `schema.Loader` interface or the `versionedLoader` side channel.
- No change to the render glue or `glueDiagnostics` beyond leaving it alone (slice 4).
- No deletion of duplicated tests or helpers beyond what the removals force (slice 8).

## Decisions

### Partial mode becomes an unexported boolean, not a public option

**Context**: `Partial()` has one caller, `acquire.go:219`, which validates the extra-values sources against `#config` without concreteness so a type or constraint violation is reported at the source's own positions before the whole-instance concreteness check.
**Options**: (1) keep `ValidateOption`/`Partial()` public for that one internal use; (2) an unexported `validateSources(schema cue.Value, sources []Source, requireConcrete bool) (cue.Value, error)` that `ValidateConfigDetailed` calls with `true` and the acquire path with `false`; (3) drop the per-source pass and rely on the build error.
**Decision**: option 2. The option type existed to let a public method carry a knob nobody outside used; a boolean on an unexported function is the same behaviour with no surface. Option 3 loses attribution, which the extra-values tests pin. `runValidate` and `appendSchemaErrors` folded into `validateSources` after verification showed each had exactly one caller and the latter's bool return was never read; `walkDisallowed` stays separate because it recurses.

### `processInstance(spec)` keeps the error framing, drops the inputs that were always empty

**Context**: `ProcessModuleInstance(ctx, spec, mod, values)` is called twice, both with `cue.Value{}` values and (once) with `module.Module{}`. Its live work is `spec.Validate(cue.Concrete(true))`, metadata decoding, and the `instance %q:` framing.
**Decision**: `func processInstance(spec cue.Value) (*module.Instance, error)`, a free function: it reads neither the Kernel nor a context (verification showed both unused, so the receiver and `ctx` went too). The name for framing is read from `spec` through `schema.Metadata`; the `"<unknown>"` fallback stays for a spec whose name is somehow not a string, since the framing is a diagnostic and must not itself fail. `bestEffortInstanceName` loses its `mod` parameter. The `ValidateConfig` call and the `FillPath(schema.Values, …)` branch are deleted, which makes `ValidateConfig` caller-free and lets it go in the same commit.

### Constructors take the value only; kernel wrappers absorb the change

**Context**: `module.NewModuleFromValue(_ CueContextOwner, v)` and `platform.NewPlatformFromValue(_ CueContextOwner, v)` ignore their first argument; `module.NewInstanceFromValue` has no non-test caller once `Kernel.NewInstanceFromValue` goes.
**Decision**: `module.NewModuleFromValue(v cue.Value)` and `platform.NewPlatformFromValue(v cue.Value)`; both `CueContextOwner` interfaces deleted; `module.NewInstanceFromValue` deleted (the kernel's `processInstance` builds `&module.Instance{}` directly, as it already does). `Kernel.NewModuleFromValue(v)` and `Kernel.NewPlatformFromValue(v)` keep their signatures and forward, so the cli's two call sites compile. The `stubOwner` in `opm/helper/synth/instance_test.go` and the `k` argument in `opm/module` and `opm/platform` tests go.

### Metadata decoders move to their callers, unexported

**Context**: the proposal unexports the three `opm/schema` decoders, but their only callers (`module.NewModuleFromValue`, `platform.NewPlatformFromValue`, `kernel.processInstance`) live in other packages, and Go cannot call an unexported function across a package boundary.
**Options**: (1) keep them exported; (2) a new `opm/internal/...` package holding the three; (3) move each decoder, unexported, into the package of its single caller and delete `schema/decode.go`.
**Decision**: option 3. It is the pure removal this change is scoped to: no new package, one ten-line function beside each constructor, identical error messages. Option 2 adds a package for three functions that share nothing (Principle VII). `opm/schema` keeps the metadata types and the `Metadata` path the decoders read.

### Registry loader returns the artifact `Source` type

**Context**: `loaderregistry.StagedSource{Value, Root, Overlay}` is unpacked into `module.Source{Root, Overlay}` by its only caller (`wrappers.go:40`).
**Decision**: `LoadModulePackageWithSource(...) (cue.Value, *module.Source, error)`. `opm/helper/loader/registry` gains an import of `opm/module` (no cycle: `opm/module` imports only `opm/schema`). `loaderregistry.LoadOptions` stays; slice 2 folds the loaders and settles the option type.

### `Source` loses `Name`; `LoadSourceFromBytes` loses its `name` parameter

**Context**: `Source.Name` is written by every constructor and read by nothing outside two test assertions. `LoadSourceFromBytes(origin, name string, b []byte)` exists to fill it.
**Decision**: `Source{Value, Origin}`; `LoadSourceFromBytes(origin string, b []byte)`; `LoadSourceFromFile` unchanged in signature. `LoadSourceFromString` is deleted rather than forwarded (user decision: purity over saving test edits). Tests that compiled strings through it use a test-local `mustSource(t, k, origin, src)` over `LoadSourceFromBytes`.

### `IdentityError` without `Artifact`

**Context**: the field is always `"module"`; the doc on the type says no catalog site produces it; the operator only type-checks the error.
**Decision**: delete the field and the `%s ` prefix in `Error()`: `identity mismatch at %s: metadata declares %q but the artifact was fetched as %q (%s)`. The registry loader's construction sites drop the field. Consumer tests are grepped for the old message before landing (none expected: the operator asserts on the type, the cli does not construct the error).

### Removed names are pinned, not just gone

**Context**: `TestKernel_PrunedSurface` reflects over `*kernel.Kernel` for the names earlier prunes removed. Consumers' compile errors pin absence just as well, and slice 8 may delete the reflection tests, but until then the list is the repo's own record.
**Decision**: add `ValidateConfig`, `ValidateConfigPartial`, `ProcessModuleInstance`, `NewInstanceFromValue`, `LoadSourceFromString` to the list; fix its comment ("values enter through `WithValues` and `SynthesizeInstance`"). `TestNoLoadModuleFromRegistryMethod` is untouched.

### Test re-homing, per removal

| Removal | Tests today | After |
| --- | --- | --- |
| `ValidateConfig` / `ValidateConfigPartial` / `Partial` | `validate_test.go` (11 refs), `integration_validate_test.go`, `kernel_test.go`, `integration_live_test.go` | single-value cases call `ValidateConfigDetailed` with one `Source`; partial-mode cases move to an internal `validate_internal_test.go` (package `kernel`) calling `validateSources(…, false)`, so `walkDisallowed`-under-partial stays covered |
| `ProcessModuleInstance` | `synth_test.go`, `kernel_test.go` | cases run through `SynthesizeInstance` / `AcquireInstanceFromDir`; the "zero values means no fill" case is deleted (the branch no longer exists) |
| `LoadSourceFromString` | 17 refs in `acquire_test.go`, `validate_test.go`, `source_loader_test.go` | `mustSource` helper over `LoadSourceFromBytes` in one `helpers_test.go` |
| `Source.Name` | `source_loader_test.go:21,82` | asserts deleted |
| `CueContextOwner` | `module_test.go`, `instance_test.go`, `platform_test.go`, `synth/instance_test.go` | first argument dropped; `stubOwner` deleted |
| `NewInstanceFromValue` | `module/module_test.go`, `module/instance_test.go` | the metadata-decoding cases move onto `NewModuleFromValue` (same decoder shape) or are deleted where they duplicate kernel acquire tests |
| `UnstatedPosture` | `errors/match_test.go:64` | field removed from the fixture literal |
| `WithCoreVersion` | `platformmodule/closure_test.go`, `generate_test.go` | tests build the `[]Dep` roots directly (`Roots` is ten lines) |
| `ValuesFileName` | `acquire_test.go` (1 ref) | literal `"opm-values.cue"` |
| `IdentityError.Artifact` | `loader/registry` tests asserting the message | expectation updated |

## Risks / Trade-offs

- [A consumer test asserts the old `IdentityError` message text] → grep `cli` and `opm-operator` for `identity mismatch` before landing; the operator type-checks only.
- [The cli's `module vet` needs per-source partial validation and the kernel no longer offers it publicly] → accepted by decision; `vet` keeps its own copy until slice 7, which designs a kernel spelling (likely a per-source variant of `ValidateConfigDetailed` returning attribution per source, not a resurrected `Partial()`).
- [Dropping `LoadSourceFromString` costs 17 test edits for a 3-line method] → accepted by decision; the helper is one function.
- [Unexporting `ProcessModuleInstance` removes the "helper-level" composition `synth.Instance` then `ProcessModuleInstance` the docs advertised] → no consumer used it; `SynthesizeInstance` is the entry, and `synth.Instance` alone still returns the built value and staged source for a caller without a Kernel.
- [`CLAUDE.md`, `README.md`, `docs/getting-started.md` and `opm/kernel/doc.go` mention removed names] → grep-driven doc pass is a task; the docs are part of the PR.

## Migration Plan

MAJOR on the alpha line, released by release-please from `refactor(kernel)!:`. `cli` and `opm-operator` re-pin in their next `fix(deps)` wave with no source change. Rollback is a revert of the release; nothing downstream depends on the new shapes until they re-pin.

## Open Questions

None. The four calls that shaped this change (cut `Partial` now, delete `LoadSourceFromString`, keep the `Instance` accessors, cut `IdentityError.Artifact`) were made before the specs were written.
