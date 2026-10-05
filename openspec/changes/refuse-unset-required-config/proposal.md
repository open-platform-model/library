## Why

`Kernel.SynthesizeInstance` and `Kernel.AcquireInstanceFromDir` accept an instance whose
values leave a required `#config` value unset, as long as no component reads that value
(library#211). Core's `#ModuleInstance` unifies `values` into a `let` binding
(`let unifiedModule = #module & {#config: values}`) that only the components read, so an
unread field never reaches the output. The instance processing step asserts concreteness on
the built spec (`processInstance`, `opm/kernel/process.go`), and an unread `#config` field is
not part of that spec. The per-source checks both verbs run after the build
(`checkInstanceValues` in `acquire.go`, the post-build check in `synth.go`) validate with
`requireConcrete` false, so they do not flag a missing field either.

The operator refuses such an instance today, through its own pre-validate before every
synthesis: `ValidateConfigDetailed(mod.ConfigSchema(), sources)` in opm-operator
`internal/render/kernel_module_renderer.go`, which asserts concreteness on `#config`
unified with the values. The operator's planned deletion of that pre-validate is on hold,
because deleting it would loosen what the operator accepts. The cli has no such check on
its instance-directory render path (`internal/workflow/render/render.go`, over
`AcquireInstanceFromDir`), and neither has the operator's package renderer
(`internal/render/kernel_package_renderer.go`). The same instance is accepted or refused
depending on the frontend.

The owner settled library#211 on 2026-10-05: the kernel refuses it. Both
`SynthesizeInstance` and `AcquireInstanceFromDir` refuse an unset required `#config` value
at its position, and the operator then deletes its pre-check. This may tighten
`AcquireInstanceFromDir` for the cli.

## What Changes

- **Both instance verbs refuse an unset required `#config` value.** The kernel's one
  instance processing step, which both verbs already share, gains a second concreteness
  check after the existing one: the module's `#config` (read off the built spec, never off
  `Module.Package`) unified with the instance's effective values, validated with
  `cue.Concrete(true)`. A required field the values leave unset (a `foo!:` field, or a
  field with a type and no default such as `image: string` or `_`) is refused even when no
  component reads it. It is the same check the operator's pre-validate runs, so the set of
  refused instances is the same.
- **Position and attribution.** The refusal is CUE's own error tree, unchanged by the
  kernel. A missing field is reported at its `#config` declaration in the module source,
  under the `#config` path. A field a source wrote in a non-concrete form also carries the
  source's `Origin` position, because the source's compiled values join the unification.
  The error is framed like every other concreteness refusal of the two verbs:
  `Kernel.<Verb>: instance "<name>": not fully concrete: …`.
- **Unchanged.** The failure-path attribution of library #197 (`valuesConflict`, validated
  without concreteness) stays as it is: a build that fails for another reason still returns
  its build error, not a missing-field error. The existing concreteness check on the built
  spec runs first and keeps its errors, so every instance refused today is refused with the
  same error. `ValidateConfigDetailed` does not change.
- **Docs.** The godoc of `SynthesizeInstance`, `AcquireInstanceFromDir` and
  `processInstance`, and the values paragraph of the `opm/kernel` package doc, state the
  new refusal. `docs/getting-started.md` keeps its one orientation sentence if it stays true.

**Behaviour change, not an API change.** No exported signature, type or sentinel changes.
An `AcquireInstanceFromDir` or `SynthesizeInstance` call that leaves a required `#config`
value unset where no component reads it was accepted before and is now refused. The release
note (the squash body) says so. Callers that pre-validate with `ValidateConfigDetailed`
(the operator's module renderer, the cli's module render path) see no difference.

SemVer class: PATCH (a correction to what the kernel accepts; the owner chose the `fix`
class). Release class of the PR title: `fix`. The PR closes library#211.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `kernel-runtime`: adds the requirement "Instance verbs refuse an unset required config
  value", with scenarios for both verbs, the unread field, the source position, the parity
  with `ValidateConfigDetailed` and the unchanged failure-path attribution.

## Impact

- Packages: `opm/kernel` (`process.go`, `synth.go`, `acquire.go`, `doc.go`; tests in
  `synth_test.go`, `acquire_test.go`, and a parity test against `ValidateConfigDetailed`).
  No change under `opm/internal/synth`, `opm/module` or `opm/errors`.
- Public surface under `opm/`: none. `task api:diff` lists nothing new.
- Cost: one extra unify-and-validate of `#config` with the values per successful instance
  acquisition or synthesis; no extra build.
- Downstream:
  - opm-operator: after a library release that contains this change, the module renderer's
    pre-validate (`kernel_module_renderer.go`, the `ValidateConfigDetailed` call and, if it
    has no other caller, `cueFindings`) can be deleted. The operator's error wording for
    such an instance then comes from `SynthesizeInstance`. The package renderer
    (`kernel_package_renderer.go`, `AcquireInstanceFromDir`) newly refuses such packages.
  - cli: `opm` commands that render an instance directory through `AcquireInstanceFromDir`
    newly refuse an instance that leaves a required `#config` value unset. The module render
    path already pre-validates with `ValidateConfigDetailed`, so it is unchanged.
  - Fixtures and the module fleets: section 3 runs every on-disk `ModuleInstance` package in
    cli, opm-operator, modules and opm-modules through the changed kernel, plus the
    consumer-build script and the operator's registry-backed integration specs where cheap,
    and lists any instance this newly refuses in design.md.
- No `enhancement.yaml`: the decision is the owner's settlement of library#211, not an
  enhancement decision.
