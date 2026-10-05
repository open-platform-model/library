## Why

`Kernel.SynthesizeInstance` and `Kernel.AcquireInstanceFromDir` accept an instance whose
values leave a required `#config` value unset, as long as no component reads that value
(library#211). Core's `#ModuleInstance` unifies `values` into a `let` binding
(`let unifiedModule = #module & {#config: values}`) that only the components read, so an
unread field never reaches the output. The instance processing step asserts concreteness on
the built spec (`processInstance`, `opm/kernel/process.go`), and an unread `#config` field
that the values never mention is not part of that spec. The per-source checks both verbs run
after the build (`checkInstanceValues` in `acquire.go`, the post-build check in `synth.go`)
validate with `requireConcrete` false, so they do not flag a missing field either.

The operator refuses such an instance today, through its own pre-validate before every
synthesis: `ValidateConfigDetailed(mod.ConfigSchema(), sources)` in opm-operator
`internal/render/kernel_module_renderer.go`, which asserts concreteness on `#config`
unified with the values. The operator's planned deletion of that pre-validate is on hold,
because deleting it would loosen what the operator accepts. The cli has no such check on
its instance-directory paths (`internal/workflow/render/render.go` and
`internal/cmdutil/instance_arg.go`, over `AcquireInstanceFromDir`), nor on its module
render path when no `-f` values are given (`internal/workflow/render/module.go` leaves
`debugValues` to the build). Neither has the operator's package renderer
(`internal/render/kernel_package_renderer.go`). The same instance is accepted or refused
depending on the frontend.

The owner settled library#211 on 2026-10-05: the kernel refuses it. Both
`SynthesizeInstance` and `AcquireInstanceFromDir` refuse an unset required `#config` value
at its position, and the operator then deletes its pre-check. This may tighten
`AcquireInstanceFromDir` for the cli. library#211 itself only asks the question; the pull
request that closes it records the settlement in its body.

## What Changes

- **Both instance verbs refuse an unset required `#config` value.** The kernel's one
  instance processing step, which both verbs already share, gains a second concreteness
  check after the existing one: the built spec's `values` unified with the module's
  `#config` (read off the built spec, never off `Module.Package`), validated with
  `cue.Concrete(true)`. A required field the values leave unset (a `foo!:` field, or a
  field with a type and no default such as `image: string` or `_`) is refused even when no
  component reads it. It is the same concreteness rule the operator's pre-validate runs, so
  the set of refused instances is the same.
- **Position and path.** The refusal is CUE's own error tree, unchanged by the kernel. Each
  finding is at path `values.<field>`. An unset field is reported at its `#config`
  declaration in the module source where CUE has a position for it (a `foo!:` field or a
  bare type); a field declared `_` has no position in CUE and is identified by its path
  only. No values source wrote an unset field, so there is no source `Origin` to name. A
  value a source wrote in a non-concrete form (`image: string`) is not new: the existing
  built-spec check already refuses it, unchanged. The error is framed like every other
  concreteness refusal of the two verbs: `Kernel.<Verb>: instance "<name>": not fully
  concrete: …`.
- **Unchanged.** The failure-path attribution of library #197 (`valuesConflict`, validated
  without concreteness) stays as it is: a build that fails for another reason still returns
  its build error, not a missing-field error. The existing concreteness check on the built
  spec runs first and keeps its errors, so every instance refused today is refused with the
  same error. `ValidateConfigDetailed` does not change.
- **Docs.** The godoc of `SynthesizeInstance`, `AcquireInstanceFromDir` and
  `processInstance`, and the values paragraph of the `opm/kernel` package doc, state the
  new refusal. `docs/getting-started.md` keeps its one orientation sentence if it stays true.

**Breaking behaviour change, no API change.** No exported signature, type or sentinel
changes. An `AcquireInstanceFromDir` or `SynthesizeInstance` call that leaves a required
`#config` value unset where no component reads it was accepted before and is now refused.
Per CONSTITUTION VI (pre-GA clause) and ADR-010 the change lands breaking-marked, with a
`BREAKING CHANGE:` footer that is the migration note the CHANGELOG shows: which instances
are now refused, and the remedy (set the value, give the field a default, or mark it
optional with `?`).

Release effect: advances `-beta.N` (release-please `versioning: prerelease`). CONSTITUTION
VI lands a pre-GA behaviour break only as `feat!`, so the squash title is
`feat(kernel)!: refuse an instance that leaves a required config value unset`. The PR
closes library#211.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `kernel-runtime`: adds the requirement "Instance verbs refuse an unset required config
  value", with scenarios for both verbs, the unread field, the `_` field, the parity with
  `ValidateConfigDetailed` and the unchanged error of instances refused before. The
  failure-path attribution scenario stays in `instance-synthesis`, its one home.
- `instance-synthesis`: modifies "Values field is caller-supplied with no implicit
  fallback"; the THEN of "Zero Values is not replaced by debugValues" now ends "... and the
  call fails on concreteness".

## Impact

- Packages: `opm/kernel` (`process.go`, `synth.go`, `acquire.go`, `doc.go`; tests in the
  new `required_config_test.go`, including a parity test against `ValidateConfigDetailed`,
  and one edited subtest in `acquire_test.go`).
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
    (`internal/workflow/render/render.go`) newly refuse an instance that leaves a required
    `#config` value unset. So does path targeting (`internal/cmdutil/instance_arg.go`,
    `opm instance delete/status ./dir`): an instance deployed before that leaves an unread
    required value unset can no longer be targeted by its directory; a name or UUID
    argument still works. `opm module render/apply <dir>` with no `-f` synthesizes from
    `debugValues` without pre-validating them, so a module whose `debugValues` leave an
    unread required field unset is newly refused there too; with `-f`, the cli already
    pre-validates with `ValidateConfigDetailed` and nothing changes.
  - Fixtures and the module fleets: section 3 runs every module's `debugValues` through
    `SynthesizeInstance` and every on-disk instance package through
    `AcquireInstanceFromDir` in cli, opm-operator, modules and opm-modules, on this kernel
    and on `origin/main`, plus the cli and operator unit tests against this tree, and lists
    any instance this newly refuses in design.md.
- No `enhancement.yaml`: the decision is the owner's settlement of library#211, not an
  enhancement decision.
