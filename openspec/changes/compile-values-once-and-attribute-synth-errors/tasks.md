# Tasks: compile-values-once-and-attribute-synth-errors

Worktree `library/.claude/worktrees/compile-values-once-and-attribute-synth-errors`, branch
`fix/compile-values-once-and-attribute-synth-errors` (from `origin/main` `a8bfc76`). Seed
`.cue-cache` by copying the main checkout's (`cp -a`), never symlinking. Every command runs
inside the worktree with the registry env exported on two lines and an absolute private
`TMPDIR` (`mktemp -d` under the session scratchpad) for the tests:

```bash
export CUE_REGISTRY='opmodel.dev=ghcr.io/open-platform-model,testing.opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'
export OPM_REGISTRY="$CUE_REGISTRY"
```

`TestGenerate_BuildsThroughTheKernel` can fail in a full-suite run from a known cross-process
cache race. Rerun `go test ./opm/helper/platformmodule -count=1` alone before treating it as
a finding. Commit bodies never start a line with `word(` and carry no bare at-sign. The only
trailer is `Co-Authored-By: Claude <noreply@anthropic.com>`. Design decisions are CV1 to CV5
in design.md. Code and test comments cite no CVn, task or section number.

## 1. Pin a values conflict that fails the instance build (kernel tests; design CV4)

Tests only. Every one passes on `origin/main` before any code moves.

- [ ] 1.1 `opm/kernel/synth_test.go`: `TestKernel_SynthesizeInstance_BuildFailingViolation`
      publishes a synth module (`publishSynthModule`) whose `#config` declares
      `replicas: int | *1` and whose one component has a field that reads `#config.replicas`.
      Use the smallest component shape core accepts, copied from an existing synth or render
      fixture. Synthesize with `Values: []kernel.Source{mustSource(t, k, "/values/bad.cue",
      `replicas: "three"`)}`. Assert an error, a nil instance and `replicas` in the message.
      First confirm the failure comes from `synth.Instance`, not from the post-build check:
      the message must not start `Kernel.SynthesizeInstance: instance "`. Write in design.md
      CV4 which file today's positions name (expected: the synthesized `values.cue`). If the
      build does not fail, change the component until it does. Do not continue with a
      fixture whose build passes.
- [ ] 1.2 Same file: `TestKernel_SynthesizeInstance_CleanValuesBuildErrorUnchanged`, a module
      whose component fails on its own (for example a field set to two conflicting
      literals), fed a clean `replicas: 2`. Assert the error is framed
      `Kernel.SynthesizeInstance: ` and does not contain `instance "myrel": `. It must hold
      before and after.
- [ ] 1.3 `opm/kernel/acquire_test.go`: the acquire twin of 1.1, an instance directory whose
      module component consumes `#config.replicas`, acquired with the same bad source.
      Assert it fails with positions naming `/values/bad.cue` and the message framed
      `Kernel.AcquireInstanceFromDir: instance "`. This already holds; it is the target
      shape for section 3. Reuse a render fixture with a component that reads a `#config`
      value if one exists (`renderFixtureDir`). Otherwise write one under `t.TempDir()`
      that imports the module 1.1 publishes.
- [ ] 1.4 `task check` green, then commit
      `test(kernel): pin values conflicts that fail the instance build`.

## 2. Compile each values source once per call (kernel; design CV1, CV5)

- [ ] 2.1 `opm/kernel/validate.go`: add `validateCompiled(schema, values, requireConcrete)`
      with the no-values and missing-schema short-circuit. `validateSources` becomes
      `compileSources` plus `validateCompiled`, with its signature and the
      `validate_internal_test.go` pins unchanged. Rewrite only the lines of the
      `validateSources` doc comment that name its internal callers, which are now
      `ValidateConfigDetailed` only.
- [ ] 2.2 `opm/kernel/acquire.go`: `mergeSources` returns `(compiled []cue.Value, merged
      cue.Value, err error)`. `loadInstanceWithValues` returns the compiled slice as a
      fourth result, and `AcquireInstanceFromDir` holds it (nil on the no-values branch).
      `checkInstanceValues(spec, compiled)` calls `validateCompiled(configSchema, compiled,
      false)`. `attributeValuesError(cueCtx, authored, compiled)` drops its
      `compileSources` call and its `compiling values sources` error branch. Rewrite the
      doc-comment lines these signatures make false.
- [ ] 2.3 `opm/kernel/synth.go`: take `compiled` from `mergeSources` and run the post-build
      check as `validateCompiled(configSchema, compiled, false)`. The error framing is
      unchanged.
- [ ] 2.4 Verify (CV5): `grep -n 'compileSources(' opm/kernel/*.go | grep -v _test` shows
      the definition plus exactly two calls, in `mergeSources` and `validateSources`.
      `grep -n 'validateSources(' opm/kernel/*.go | grep -v _test` shows only the
      definition and `ValidateConfigDetailed`. `go test ./opm/kernel -count=1` green,
      including the section 1 tests unchanged, `LayeredSourcesUnifyInOrder`,
      `ViolationAttributedToSource` and the `WithSources_ConflictAttributed` family.
- [ ] 2.5 `task check` green, then commit
      `perf(kernel): compile each values source once per call`.

## 3. Attribute a synthesized instance's values conflict to its source (kernel; design CV2, CV3)

- [ ] 3.1 `opm/kernel/acquire.go`: extract `valuesConflict(authored cue.Value, compiled
      []cue.Value) error` from `attributeValuesError`. It reads `#module.#config` off
      `authored`, prepends `authored`'s own `values` when it exists, and calls
      `validateCompiled(configSchema, all, true)`. It returns the raw CUE error or nil.
      `attributeValuesError` keeps its framing and calls it.
- [ ] 3.2 `opm/kernel/synth.go`: when `synth.Instance` fails and `merged.Exists()`, call
      `synth.Instance` again in the same `cueCtx` with the same input and `Values` zero. If
      that fails, return the original build error. Otherwise `valuesConflict(rebuilt,
      compiled)`. A non-nil result returns `Kernel.SynthesizeInstance: instance %q: %w` with
      `in.Name`; a nil result returns the original build error unchanged. Never call
      `in.Module.ConfigSchema()` or read `in.Module.Package` (CV2).
- [ ] 3.3 Update the `SynthesizeInstance` godoc with one paragraph: a build failure is
      attributed to the values sources the way `AcquireInstanceFromDir` does it, through a
      values-free rebuild in the call's context. Keep the "never its Package" sentence true.
- [ ] 3.4 Tests. Tighten 1.1 to assert the `Kernel.SynthesizeInstance: instance "myrel": `
      prefix and that a position names `/values/bad.cue` (`positionsName`). 1.2 must stay
      green unchanged. Add `TestKernel_SynthesizeInstance_NoValuesBuildErrorUnchanged`: the
      1.2 module (its component fails on its own) synthesized with no values, and again with
      one empty-`Data` source. Both return the build error framed `Kernel.SynthesizeInstance: `
      without `instance "myrel": `, because there are no merged values to attribute. Add the instance-synthesis scenario test: the 1.1 and 1.3
      fixtures together, asserting both verbs name `/values/bad.cue` under their own
      `instance "<name>": ` framing. Extend `TestKernel_SynthesizeInstance_BuildsInItsOwnContext`
      or a sibling only if it already covers a failure path cheaply; do not invent a
      cross-context probe.
- [ ] 3.5 Cross-cutting checks: `go test -race ./opm/kernel -count=1`, the parity tests
      (`go test ./opm/kernel -run Parity -count=1`) and `task cue:test:flow`. Verify: all
      green (the flow test may skip when the registry is unreachable; say so if it does).
- [ ] 3.6 `task check` green (it includes api-diff, consumer-build and the cascade wiring
      check; api-diff must report no exported change), then commit
      `fix(kernel): attribute a synthesized instance's values conflict to its source`.

## 4. Verify

- [ ] 4.1 Whole-tree gates on the final tree: `task check`. Verify: green.
- [ ] 4.2 `openspec validate compile-values-once-and-attribute-synth-errors --strict` passes.
- [ ] 4.3 Archive only when the supervisor says the PR is being opened, on this branch, so
      the archive rides the implementing PR:
      `openspec archive compile-values-once-and-attribute-synth-errors --yes`, then
      `openspec validate --all --strict`, then commit
      `chore(openspec): archive compile-values-once-and-attribute-synth-errors`. There is no
      `enhancement.yaml`, so no delivery log runs.
