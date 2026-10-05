## Why

The two kernel verbs that take values sources compile each source more than once, and
`SynthesizeInstance` does not attribute a values conflict that fails its build. At
`origin/main` `a8bfc76` (after b2, library #188, and the consolidate change, library #184):

- **`AcquireInstanceFromDir` with values** compiles the sources in `mergeSources`
  (`opm/kernel/acquire.go:356-369`, over `compileSources` in `source_loader.go:127-137`).
  On a successful build, `checkInstanceValues` (`:433-446`) compiles them again through
  `validateSources` (`validate.go:56-68`). On a failed build, `attributeValuesError`
  (`:457-477`) compiles them again (`:467`).
- **`SynthesizeInstance`** compiles the sources in `mergeSources` (`synth.go:126`) and again in
  the post-build check (`:151`, `validateSources`).
- A file-backed source runs a full `load.Instances` on every compile
  (`source_loader.go:94-123`), so each extra compile is a package load. All of these compiles
  run in the call's own `cue.Context`, the one the spec is built in, so the values the merge
  compiled can be reused.
- **The g2 gap.** When the synthesized build fails (`synth.go:131-142`), `SynthesizeInstance`
  returns the build error unchanged. A values source that violates `#config` where a
  component reads the value fails the build inside the module, so its positions name the
  rendered `values.cue` in the synthesized package, not the source's `Origin`.
  `AcquireInstanceFromDir` handles the same failure through `attributeValuesError`. The
  instance-synthesis spec already says the two paths' "values attribution SHALL be
  identical", so today the synthesis path does not meet its own spec. The operator works
  around it with a `ValidateConfigDetailed` pre-validate before every synthesis
  (`opm-operator` `internal/render/kernel_module_renderer.go:167-168`).

Owner decisions from the beta-1 kernel-plan walkthrough (2026-10-02/03):

- b1: "Compile values once; land after b2 (same function), bundled with g2 in one library
  change."
- g2: "Yes, bundled with b1: SynthesizeInstance mirrors AcquireInstanceFromDir's failure-path
  values attribution (no Package read); operator deletes its pre-validate after the release."

b2 is merged, so `attributeValuesError` already builds the authored package from the overlay
read once. This change only changes how it gets the compiled values.

## What Changes

- **Compile once (b1).** `mergeSources` returns the compiled values as well as the merged
  value. The post-build check and the failure-path attribution of both verbs validate those
  compiled values (through the existing unexported `validateValues`) instead of compiling the
  sources again. Each source is compiled exactly once per call. `ValidateConfigDetailed` and
  its internal `validateSources` keep their signatures and behaviour.
- **Synthesis attributes a values conflict that fails its build (g2).** When `synth.Instance`
  fails and the call's merged values exist, `SynthesizeInstance` builds the synthesized package again
  without the rendered values file. This is the mirror of the authored package that
  `AcquireInstanceFromDir` rebuilds. It is built in the same call context, through the same
  `synth.Instance`. The values the merge compiled are validated against the `#config` of that
  build, with the same pass and the same concreteness mode as `AcquireInstanceFromDir`. On a
  values error it returns `Kernel.SynthesizeInstance: instance "<name>": <error>`, with
  positions that name each source's `Origin`. Otherwise (no values, the rebuild fails too, or
  the values are clean) it returns the build error unchanged. The schema is never read from
  `in.Module.Package` (design CV2). Both verbs share one attribution helper.
- **Docs.** The `SynthesizeInstance` godoc gains the failure-path sentence. The comments on
  `mergeSources`, `checkInstanceValues`, `attributeValuesError` and `validateSources` that
  this change makes false are rewritten. All other comment work is left to the c4 pass.

Not **BREAKING** for the API: every changed function is unexported. Behaviour change for
`SynthesizeInstance` callers: a values source that violates `#config` where a component
consumes the value now fails with the source-attributed values error, framed
`Kernel.SynthesizeInstance: instance "<name>": …`. Before, it failed with the build error
positioned in the synthesized package. The squash body states this.

SemVer class: PATCH. Release class of the PR title: `fix`. g2 corrects the attribution users
see, which outweighs b1's internal saving.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `kernel-runtime`: `Kernel.SynthesizeInstance method` states the failure-path attribution
  and gains two scenarios. Compile-once (b1) adds no spec text: no scenario can observe it,
  so design CV5 records it and the verify step checks it by reading the code.
- `instance-synthesis`: `Instance construction shares one evaluate-and-shape-gate with the
  file loader` gains a scenario showing that a values conflict that fails the build is
  attributed identically on both paths.

## Impact

- Packages: `opm/kernel` (`acquire.go`, `synth.go`, `validate.go` doc comment only, tests in
  `synth_test.go`, `acquire_test.go`). `opm/internal/synth` is called a second time on the
  failure path but does not change.
- Public surface under `opm/`: none. `ValidateConfigDetailed`, `InstanceInput`,
  `SynthesizeInstance` and `AcquireInstanceFromDir` keep their signatures.
- Cost: one extra synth build, only when a build with values fails. Success paths drop one
  compile of every source; the layered-acquire failure path drops one.
- Downstream: cli and opm-operator compile unchanged. The operator change op-i3g2 deletes the
  pre-validate at `kernel_module_renderer.go:167-168` (and `cueFindings` if it has no other
  caller) after a library release that contains this change. Before it deletes it, it
  compares its error wording for a `spec.values` violation. The pre-validate requires
  concreteness and words the error as `validating values against the module's #config: …`.
  After deletion, the same violation surfaces through `SynthesizeInstance`'s values error or
  its concreteness error.
- Sequencing: library acquire chain b2 (merged) -> this change -> d1+d3 -> g5 part A. The
  consolidate change (library #184, `validate.go` `walkDisallowed`) is merged. d1+d3 rebases
  on this change.
- No `enhancement.yaml`: the decisions come from the beta-1 kernel-plan walkthrough, not
  from an enhancement entry.
