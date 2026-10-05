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

design.md carries two unverified assumptions (RC2: unifying the compiled sources again
changes no value and adds their positions; RC1: the new check fires only for instances
accepted today). This section lands tests that pass on `origin/main` and records what they
show.

- [ ] 1.1 `opm/kernel/synth_test.go`: publish a synth module (`publishSynthModule`) whose
      `#config` declares `replicas: int | *1`, `image: string`, `tag!: string`,
      `opt?: string` and `any: _`, and whose one component reads only `replicas` through a
      hidden field (`_r: #config.replicas & int`). With one source `replicas: 2`:
      `SynthesizeInstance` succeeds today (pin it with a comment naming library#211; section
      2 turns it into a refusal), while `ValidateConfigDetailed(mod.ConfigSchema(), …)`
      refuses `image`, `tag` and `any`. If the build fails or the synthesis is refused, fix
      the fixture before going on.
- [ ] 1.2 Same fixture, internal test (`process_internal_test.go` or
      `validate_internal_test.go`, package `kernel`): build the spec through the kernel,
      then validate `#module.#config` unified with the spec's `values` under
      `cue.Concrete(true)`, once alone and once with the compiled sources unified in. Record
      in design.md (RC2, a "Spike result" paragraph): the error paths CUE reports, the
      positions of each finding (the `#config` declaration, and for a source writing
      `image: string`, whether `/values/a.cue` appears), and whether the two forms refuse
      the same fields. If the double unification changes a result, RC2 falls back to the
      built `values` alone; write that down.
- [ ] 1.3 `opm/kernel/acquire_test.go`: the acquire twin of 1.1, an instance directory of
      the same module (written under `t.TempDir()`, importing the module 1.1 publishes, or
      a `renderFixtureDir` copy if one already has an unread required field) with its own
      `values: {replicas: 2}`: accepted today with and without a trailing source.
- [ ] 1.4 `task check` green, then commit
      `test(kernel): pin instances that leave an unread config value unset`.

## 2. Refuse an unset required config value in instance processing (kernel; design RC1 to RC4)

- [ ] 2.1 `opm/kernel/process.go`: `processInstance(spec, compiled []cue.Value)` runs the
      existing `spec.Validate(cue.Concrete(true))` first, unchanged, then the required-config
      check of RC1 (`#module.#config` off the spec, unified with the spec's `values` and,
      per the section 1 result, the compiled sources, validated with `cue.Concrete(true)`),
      framed `instance %q: not fully concrete: %w`. Skip when `#config` or `values` does not
      exist. Rewrite the `processInstance` doc comment to state both checks and that the
      second refuses a value no component reads.
- [ ] 2.2 `opm/kernel/synth.go` passes `compiled`; `opm/kernel/acquire.go` passes
      `compiled` (nil on the no-sources branch). `valuesConflict` and both post-build
      per-source checks stay without concreteness. Update the `SynthesizeInstance`,
      `AcquireInstanceFromDir` and `InstanceInput.Values` godoc sentences this makes
      incomplete, and the values paragraph of `opm/kernel/doc.go` (one sentence: both verbs
      refuse a required `#config` value the values leave unset, read or not). Check
      `docs/getting-started.md` line 105 and `README.md` still hold; edit only a sentence
      that became false.
- [ ] 2.3 Tests. Turn 1.1 and 1.3 into refusals: no instance, framing
      `Kernel.SynthesizeInstance: instance "myrel": not fully concrete: ` and
      `Kernel.AcquireInstanceFromDir: instance "<name>": not fully concrete: `, the messages
      name `image`, `tag` and `any` and not `opt` or `replicas`, and a position names the
      module's `#config` declaration. Add: optional-and-defaulted-only succeeds on both
      verbs; a source writing `image: string` is refused with a position naming its
      `Origin` (or, if section 1 fell back, the test asserts what RC2 records); the parity
      table of RC3 (`TestKernel_SynthesizeInstance_RequiredConfigMatchesValidateConfigDetailed`),
      each case asserting `SynthesizeInstance` refuses exactly when
      `ValidateConfigDetailed(mod.ConfigSchema(), []Source{src})` does.
- [ ] 2.4 Unchanged behaviour: `TestKernel_AcquireInstanceFromDir_NonConcreteRejected`,
      `TestKernel_AcquireInstanceFromDir_OwnValues_ValidNonConcreteAcquires`,
      `TestKernel_SynthesizeInstance_EmptyValuesNotBackfilledFromDebugValues`,
      `TestKernel_SynthesizeInstance_CleanValuesBuildErrorUnchanged` (its required-`image`
      subtest included) and
      `TestKernel_AcquireInstanceFromDir_WithSources_IncompleteValuesKeepBuildError` pass
      without edits. If any other existing test or fixture under `testdata/` relied on an
      unread required field staying unset, give the fixture the value or a default and say
      why in the commit body; never loosen the check.
- [ ] 2.5 `go test -race ./opm/kernel -count=1`, the parity tests
      (`go test ./opm/kernel -run Parity -count=1`) and `task cue:test:flow` (it may skip
      when the registry is unreachable; say so if it does). Then `task check` green, and
      commit `fix(kernel): refuse an instance that leaves a required config value unset`,
      whose body states the behaviour change: an `AcquireInstanceFromDir` or
      `SynthesizeInstance` call accepted before, because no component read the unset value,
      is now refused.

## 3. Downstream survey (no library code; design RC5)

- [ ] 3.1 `task api:diff`: nothing listed under the files this change touches.
- [ ] 3.2 Consumer build: fresh clones of cli and opm-operator `main` in the scratchpad,
      `GOTOOLCHAIN=local bash .tasks/consumer-build.sh <clone> .` for each. Verify: both
      green.
- [ ] 3.3 Instance survey: a scratch Go program (scratchpad, `go.work` with this worktree,
      never committed) that calls `AcquireInstanceFromDir` on every directory holding a
      `kind: "ModuleInstance"` package in cli, opm-operator, modules and opm-modules at
      their `origin/main` (fresh clones or `git archive`, never another session's
      worktree), with the workspace registry env. Run the same program against the library
      at `origin/main` to separate new refusals from instances refused already. For each
      ModuleInstance CR fixture with `spec.values` in opm-operator, run its values through
      `SynthesizeInstance` where the module resolves from GHCR. Where cheap, run the
      operator's registry-backed integration specs that synthesize fixture instances with
      a `replace` to this worktree in a scratch copy, under the flock and context rules in
      the workspace guide.
- [ ] 3.4 Write a "Downstream survey" section in design.md: every instance this newly
      refuses (repo, path, the unset field), or "none found", plus what was not reachable.
      List the operator pre-validate deletion and any fixture fix as follow-ups in their
      own repos. Then `openspec validate refuse-unset-required-config --strict` and
      `task check` green, and commit
      `chore(openspec): record the downstream survey for refuse-unset-required-config`.

## 4. Verify

- [ ] 4.1 Whole-tree gates on the final tree: `task check`. Verify: green.
- [ ] 4.2 `openspec validate refuse-unset-required-config --strict` passes.
- [ ] 4.3 Archive only when the PR is being opened, on this branch, so the archive rides the
      implementing PR: `openspec archive refuse-unset-required-config --yes`, then
      `openspec validate --all --strict`, then commit
      `chore(openspec): archive refuse-unset-required-config`. There is no
      `enhancement.yaml`, so no delivery log runs.
