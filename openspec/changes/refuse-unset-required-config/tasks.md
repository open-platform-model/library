# Tasks: refuse-unset-required-config

Worktree `library/.claude/worktrees/refuse-unset-required-config`, branch
`fix/refuse-unset-required-config` (from `origin/main` `88afdfb`). `.cue-cache` is a copy
of the main checkout's (`cp -a`), never a symlink; `chmod -R u+w` it before the worktree is
removed. Every command runs inside the worktree with the registry env exported and an
absolute private `TMPDIR` (`mktemp -d` under the session scratchpad) for the tests:

```bash
export CUE_REGISTRY='opmodel.dev=ghcr.io/open-platform-model,testing.opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'
export OPM_REGISTRY="$CUE_REGISTRY"
```

`TestGenerate_BuildsThroughTheKernel` can fail in a full-suite run from a known
cross-process cache race. Rerun `go test ./opm/helper/platformmodule -count=1` alone before
treating it as a finding. Commit bodies never start a line with `word(` and carry no bare
at-sign. The only trailer is `Co-Authored-By: Claude <noreply@anthropic.com>`. Design
decisions are RC1 to RC5 in design.md. Code and test comments cite library#211 where a
reason is needed, never an RCn, task or section number.

## 1. Spike: pin the gap and the CUE positions (kernel tests; design RC1, RC2)

This section lands tests that pass on `origin/main` and pin the gap.

- [x] 1.1 `opm/kernel/synth_test.go`: publish a synth module (`publishSynthModule`) whose
      `#config` declares `replicas: int | *1`, `image: string`, `tag!: string`,
      `opt?: string` and `any: _`, and whose one component reads only `replicas` through a
      hidden field (`_r: #config.replicas & int`). With one source `replicas: 2`:
      `SynthesizeInstance` succeeds today (pin it with a comment naming library#211; section
      2 turns it into a refusal), while `ValidateConfigDetailed(mod.ConfigSchema(), …)`
      refuses `image`, `tag` and `any`. If the build fails or the synthesis is refused, fix
      the fixture before going on.
- [x] 1.2 Probe the paths and positions CUE reports for each unification order (a scratch
      test, not committed) and record the result in design.md, RC2 "Spike result". Done
      while applying the plan review: `values & #config` is the chosen order; the compiled
      sources add no position.
- [x] 1.3 `opm/kernel/acquire_test.go`: the acquire twin of 1.1, an instance directory of
      the same module (`writeImportedInstance`, importing the module 1.1 publishes) with its
      own `values: {replicas: 2}`: accepted today with and without a trailing source that
      sets `opt`.
- [x] 1.4 `task check` green, then commit
      `test(kernel): pin instances that leave an unread config value unset`.

## 2. Refuse an unset required config value in instance processing (kernel; design RC1 to RC4)

- [x] 2.1 `opm/kernel/process.go`: `processInstance(spec)` runs the existing
      `spec.Validate(cue.Concrete(true))` first, unchanged, then the required-config check
      of RC1 (the spec's `values` unified with `#module.#config`, both off the spec,
      validated with `cue.Concrete(true)`), framed `instance %q: not fully concrete: %w`.
      Skip when `#config` or `values` does not exist. Rewrite the `processInstance` doc
      comment to state both checks and that the second refuses a value no component reads.
- [x] 2.2 Update the `SynthesizeInstance`, `AcquireInstanceFromDir` and
      `InstanceInput.Values` godoc sentences this makes incomplete, and the values paragraph
      of `opm/kernel/doc.go` (one sentence: both verbs refuse a required `#config` value the
      values leave unset, read or not). `valuesConflict` and both post-build per-source
      checks stay without concreteness. Check `docs/getting-started.md` and `README.md`
      still hold; edit only a sentence that became false.
- [x] 2.3 Tests. Turn 1.1 and 1.3 into refusals: no instance, framing
      `Kernel.SynthesizeInstance: instance "myrel": not fully concrete: ` and
      `Kernel.AcquireInstanceFromDir: instance "<name>": not fully concrete: `, error paths
      `values.image`, `values.tag` and `values.any` and none for `opt` or `replicas`, and a
      position naming the module's `#config` declaration for `image` and `tag`. Add:
      optional-and-defaulted-only succeeds on both verbs; the parity table of RC3
      (`TestKernel_SynthesizeInstance_RequiredConfigMatchesValidateConfigDetailed`), each
      row asserting `SynthesizeInstance` refuses whenever
      `ValidateConfigDetailed(mod.ConfigSchema(), []Source{src})` does (design RC3,
      "Implementation result", records the one pre-existing row where synthesis refuses
      more). The tests live in a new file, `opm/kernel/required_config_test.go`.
- [x] 2.4 Unchanged behaviour: `TestKernel_AcquireInstanceFromDir_NonConcreteRejected`,
      `TestKernel_AcquireInstanceFromDir_OwnValues_ValidNonConcreteAcquires`,
      `TestKernel_SynthesizeInstance_EmptyValuesNotBackfilledFromDebugValues`,
      `TestKernel_SynthesizeInstance_CleanValuesBuildErrorUnchanged` (its required-`image`
      subtest included) and
      `TestKernel_AcquireInstanceFromDir_WithSources_IncompleteValuesKeepBuildError` pass
      without edits, except the subtest "a default acquires" of
      `TestKernel_AcquireInstanceFromDir_OwnValues_ValidNonConcreteAcquires`: its value
      `replicas: int | *3` disagrees with the `#config` default and is now refused (design
      RC3, "Implementation result"); it moves to `int | *2` and a new subtest pins the
      refusal. Add one test that tells the two checks apart for the `replicas: >=1`
      own-values case: the error is at `values.replicas`, positioned in the package's
      `values.cue`, and carries no finding positioned at the module's `#config` declaration
      file. If any other existing test or fixture under `testdata/` relied on an unread
      required field staying unset, give the fixture the value or a default and say why in
      the commit body; never loosen the check.
- [x] 2.5 `go test -race ./opm/kernel -count=1`, the parity tests
      (`go test ./opm/kernel -run Parity -count=1`) and `task cue:test:flow` (it may skip
      when the registry is unreachable; say so if it does). Then `task check` green, and
      commit `fix(kernel)!: refuse an instance that leaves a required config value unset`
      with a `BREAKING CHANGE:` footer: an `AcquireInstanceFromDir` or `SynthesizeInstance`
      call accepted before, because no component read the unset value, is now refused; the
      remedy is to set the value, give the field a default, or mark it optional with `?`.

## 3. Downstream survey (no library code; design RC5)

- [ ] 3.1 `task api:diff`: nothing listed under the files this change touches.
- [ ] 3.2 Consumer build: fresh clones of cli and opm-operator `main` in the scratchpad,
      `GOTOOLCHAIN=local bash .tasks/consumer-build.sh <clone> . <work-dir>` for each.
      Verify: both green. Through the same `GOWORK`, run the cli tests
      `go test ./internal/workflow/render/... ./internal/cmdutil/... ./internal/cmd/...`
      and the operator tests `go test ./internal/render/...`, and compare any failure with
      the same run against the library at `origin/main`.
- [ ] 3.3 Instance survey: a scratch Go program (scratchpad, `go.work` with this worktree,
      never committed), run once against this worktree and once against the library at
      `origin/main`, with the workspace registry env. It runs every module's `debugValues`
      through `SynthesizeInstance` (modules, opm-modules, the cli and operator testdata
      modules, opm-operator `modules/opm_operator`) and calls `AcquireInstanceFromDir` on
      every package directory that references `#ModuleInstance` (skipping `ErrWrongKind`)
      in cli, opm-operator, modules and opm-modules, all at their `origin/main` (fresh
      clones or `git archive`, never another session's worktree). For each ModuleInstance
      CR fixture with `spec.values` in opm-operator, it runs those values through
      `SynthesizeInstance` where the module resolves from GHCR.
- [ ] 3.4 Write a "Downstream survey" section in design.md: every instance this newly
      refuses (repo, path, the unset field), or "none found", plus what was not reachable.
      List the operator pre-validate deletion and any fixture fix as follow-ups in their
      own repos. Then `openspec validate refuse-unset-required-config --strict` and
      `task check` green, and commit
      `chore(openspec): record the downstream survey for refuse-unset-required-config`.

## 4. Verify and hand over

- [ ] 4.1 Whole-tree gates on the final tree: `task check`. Verify: green.
- [ ] 4.2 `openspec validate refuse-unset-required-config --strict` passes.
- [ ] 4.3 Archive only when the PR is being opened, on this branch, so the archive rides the
      implementing PR: `openspec archive refuse-unset-required-config --yes`, then
      `openspec validate --all --strict`, then commit
      `chore(openspec): archive refuse-unset-required-config`. There is no
      `enhancement.yaml`, so no delivery log runs.
- [ ] 4.4 Open the PR titled
      `fix(kernel)!: refuse an instance that leaves a required config value unset`. Its
      body records the owner's settlement of library#211 (the kernel refuses an unset
      required `#config` value on both instance verbs; the operator then deletes its
      pre-check), says `Closes #211`, and carries the `BREAKING CHANGE:` footer of 2.5 in
      the squash body. No line of the body starts with `word(`.
