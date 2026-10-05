## Context

See proposal.md, Why. Design-local decisions are numbered RC1 to RC5, so they do not collide
with any other numbering. Locations are at `origin/main` `88afdfb`:

- core `src/module_instance.cue:88`: `let unifiedModule = #module & {#config: values}`, and
  `values: _` at `:110`. Components are read from `unifiedModule`; a `#config` field no
  component reads never reaches the instance's output.
- `opm/kernel/process.go` `processInstance(spec)`: `spec.Validate(cue.Concrete(true))`,
  framed `instance %q: not fully concrete: %w`, then metadata decoding. Both verbs call it
  last. `Validate` does not descend into definitions, so `#module.#config` is never checked
  here; it does descend into `values`, so a value the instance's values carry in a
  non-concrete form is refused here already.
- `opm/kernel/synth.go` `SynthesizeInstance`: `mergeSources` returns `compiled` and
  `merged`; after a successful build, `validateCompiled(configSchema, compiled, false)` with
  `configSchema` read off the built spec (`#module.#config`); then `processInstance(spec)`.
- `opm/kernel/acquire.go` `AcquireInstanceFromDir`: `checkInstanceValues(spec, compiled)`
  runs `validateCompiled(configSchema, compiled, false)`, then
  `validateValues(configSchema, [built values], false)`; then `processInstance(spec)`.
  `compiled` is nil when the call passed no sources.
- `opm/kernel/acquire.go` `valuesConflict`: the failure-path attribution both verbs share
  (library #197), validated without concreteness.
- `opm/kernel/validate.go` `validateValues(schema, values, requireConcrete)`: unify, the
  closed-schema walk, then `schema.Unify(merged).Validate(cue.Concrete(requireConcrete))`.
  `ValidateConfigDetailed` is `validateSources(schema, sources, opts, true)`.
- opm-operator `internal/render/kernel_module_renderer.go` `synthesize`: loads `spec.values`
  (or `{}`) as one source, then `ValidateConfigDetailed(mod.ConfigSchema(), sources)` before
  `SynthesizeInstance`. This is the check the kernel takes over.

## Goals / Non-Goals

**Goals:** both instance verbs refuse an instance whose effective values leave a required
`#config` value unset, whether or not a component reads it; the refused set equals what the
operator's `ValidateConfigDetailed(#config, values)` pre-check plus `SynthesizeInstance`
refuse today for the same values; the refusal sits at the field's `#config` declaration
where CUE has a position for it, under the path `values.<field>`; every error the two verbs
return today for an instance they refuse today stays the same.

**Non-Goals:** deleting the operator's pre-validate (an operator change after the library
release); typing values errors (the kernel returns CUE's error tree, config-validation "No
Custom Validation Error Types"); changing `ValidateConfigDetailed`; changing the
failure-path attribution of library #197; attributing the existing built-spec refusal of a
non-concrete value a source wrote to that source's `Origin` (it is reported at the rendered
values file today and stays so; see Risks).

## Research & Decisions

### RC1. The check lives in the one instance processing step

**Context**: both verbs must refuse the same instances. They already share
`processInstance`, which is the kernel's concreteness step (kernel-runtime "Canonical
Implementations Live on Kernel").
**Explored**: (a) flip `requireConcrete` to true in the two post-build per-source checks;
(b) a separate helper each verb calls; (c) a second check inside `processInstance`.
(a) does not work on the acquire path: the per-source check sees only the trailing sources,
so it would flag a field the package's own `values` set, and with no sources it never runs.
It also changes the error of instances refused today: the render fixture test "a bare
constraint is not a values error" (`acquire_test.go`) expects `not fully concrete`.
**Decision**: (c). `processInstance(spec)` runs the existing
`spec.Validate(cue.Concrete(true))` first, unchanged, then validates the spec's `values`
unified with `#module.#config` (both read off `spec`) with concreteness, and frames a
failure the same way:

```go
func processInstance(spec cue.Value) (*module.Instance, error) {
	name := bestEffortInstanceName(spec)
	if err := spec.Validate(cue.Concrete(true)); err != nil {
		return nil, fmt.Errorf("instance %q: not fully concrete: %w", name, err)
	}
	if err := requiredConfigSet(spec); err != nil {
		return nil, fmt.Errorf("instance %q: not fully concrete: %w", name, err)
	}
	// metadata decoding unchanged
}

// requiredConfigSet: values & #config, under cue.Concrete(true).
func requiredConfigSet(spec cue.Value) error {
	configSchema := spec.LookupPath(schema.Module).LookupPath(schema.Config)
	built := spec.LookupPath(schema.Values)
	if !configSchema.Exists() || !built.Exists() {
		return nil
	}
	return built.Unify(configSchema).Validate(cue.Concrete(true))
}
```

**Rationale**: one place, both verbs, identical by construction (instance-synthesis
"Instance construction shares one evaluate-and-shape-gate with the file loader"). Running
second keeps every error of an instance refused today: a non-concrete value a component
reads, or a non-concrete `values` field, still fails the spec check first with the same
text. The new check fires only for an instance the kernel accepts today.

### RC2. Effective values, in the order values then `#config`

**Context**: on the acquire path the instance's values are the package's own `values` plus
the trailing sources; on the synthesis path they are the merged sources only. The plan
first had the compiled sources unified in again, to add their `Origin` positions.
**Decision**: validate the built spec's `values` (which already carries every source
through the rendered values file) unified with `#config`, in that order, and nothing else.
The compiled sources are not passed in.
**Rationale**: the built `values` is what the instance deploys, on both verbs. A CUE error
carries the position of one conjunct, and which one depends on the unification order. The
probe below shows the compiled sources add nothing the check can use: a field this check
refuses is one the values do not carry at all, so no source wrote it and there is no
`Origin` to name; a value a source wrote in a non-concrete form is refused by the built-spec
check before this one runs. Putting `values` first gives the path `values.<field>`, the
same path the kernel's disallowed-field errors use, instead of `#module.#config.<field>`.

**Spike result** (cue v0.17.1, through the kernel, module `#config: {replicas: int | *1,
image: string, tag!: string, opt?: string, any: _}` with one component reading only
`replicas`, synthesized with the source `replicas: 2` at `/values/a.cue`; recorded before
the check existed, when the synthesis still succeeded):

| Unification | `image` | `tag` | `any` |
| --- | --- | --- | --- |
| `#config & values` | `#module.#config.image`, declaration | `#module.#config.tag`, declaration | `#module.#config.any`, no position |
| `#config & (values & sources)` | same as above | same | same |
| `sources & values & #config` | `image`, declaration | `tag`, declaration | `any`, no position |
| `values & #config` (chosen) | `values.image`, declaration | `values.tag`, declaration | `values.any`, no position |
| `ValidateConfigDetailed` | `#config.image`, declaration | `#config.tag`, declaration | `#config.any`, no position |

The messages are `incomplete value string`, `field is required but not present` and
`incomplete value _`. `opt` and `replicas` pass in every form. All forms refuse the same
three fields. With the source `replicas: 2` plus `image: string`, the synthesis was refused
by the existing built-spec check at `values.image`, positioned in the synthesized package's
rendered `values.cue`, never at `/values/a.cue`, so no new-check finding can carry a source
`Origin`.

### RC3. Same refused set as the operator's pre-check plus synthesis

**Context**: the operator deletes its pre-validate only if the kernel refuses at least what
it refuses.
**Decision**: the concreteness rule is the one `validateValues(…, true)` applies
(`schema.Unify(merged).Validate(cue.Concrete(true))`). The disallowed-field walk and type
checks are not repeated here: the post-build per-source checks have already run them on
both verbs, a build-failing violation is attributed by `valuesConflict`, and the spec check
covers what the build reads. A parity test runs the operator's shape,
`ValidateConfigDetailed(mod.ConfigSchema(), []Source{src})`, and `SynthesizeInstance` with
the same source over a table of values: all set, an unread `string` unset, an unread `foo!`
unset, an unread `_` unset, an optional field unset, a defaulted field unset, a read field
unset, a disallowed key, a type mismatch on an unread field, a constraint violation on a
read field, and the operator's `{}` no-values source. Each row asserts that
`SynthesizeInstance` refuses whenever `ValidateConfigDetailed` refuses.

**Implementation result**: the two agree on every row but one, and that one runs the other
way and predates this change. A source that gives a defaulted field a bare type
(`port: int` where `#config` declares `port: int | *80`) passes `ValidateConfigDetailed`,
because the `#config` default completes it, and is refused by `SynthesizeInstance`'s
built-spec check, because the instance's `values` carry the bare `int`. The operator
refuses that instance today too (in synthesis), so deleting its pre-check loosens
nothing. The table pins this row, and the requirement says "every value
`ValidateConfigDetailed` refuses", not "exactly". A separate test pins the `{}` source
over a fully defaulted `#config`: both accept.

A second finding, in an existing test: an instance whose own `values` give an unread
field a default that disagrees with the `#config` default (`replicas: int | *3` against
`replicas: int | *2`) is now refused. Two disagreeing defaults leave the unified field with
no default (`int | 2 | 3`), which `ValidateConfigDetailed` refuses too. The render fixture
test "a default acquires" used that value; it now uses the agreeing default `int | *2`, and
a new subtest pins the refusal of the disagreeing one.
**Rationale**: the parity is the condition for the operator's deletion, so it is pinned by
a test, not by argument.

The no-values synthesis path is unchanged: with an empty `Values` the values path stays
`_`, the spec check refuses it as today, and the operator's `{}` source reaches the new
check as `{}`, so `#config` defaults apply and only fields with no default are refused.

### RC4. Error shape and framing

**Context**: the refusal must be readable at the field's position and walkable by
frontends.
**Decision**: return CUE's error tree unchanged, wrapped as
`Kernel.<Verb>: instance "<name>": not fully concrete: <cue error>`. The path is
`values.<field>` (RC2); the kernel does not rewrite it. A field declared `_` carries no
position, because CUE has none for it; it is identified by its path.
**Rationale**: config-validation "No Custom Validation Error Types": the kernel returns
CUE-native errors and frontends own presentation. The `not fully concrete` frame keeps one
wording for every concreteness refusal of the two verbs (artifact-types "Validation
failures propagate"), so a frontend that matches on the instance frame needs no new case.
The operator's pre-validate today words the same finding under `#config.<field>` with
`validating values against the module's #config:`; the operator change that deletes it
compares the wording and decides whether its status message needs a prefix.

### RC5. Downstream survey decides nothing in the library

**Context**: the settlement of library#211 asks for a survey of the instances in the cli
and operator fixtures and the module fleets that this newly refuses.
**Decision**: section 3 builds a scratch program (in the session scratchpad, never
committed) in a `go.work` with this worktree, and the same program in a `go.work` with the
library at `origin/main`, so new refusals separate from old ones. The program runs every
module's `debugValues` through `SynthesizeInstance` (modules, opm-modules, the cli and
operator testdata modules, opm-operator `modules/opm_operator`), and calls
`AcquireInstanceFromDir` on every package directory that references `#ModuleInstance`
(skipping `ErrWrongKind`) in cli, opm-operator, modules and opm-modules, all at their
`origin/main` (fresh clones or `git archive`, never another session's worktree). It also
runs `.tasks/consumer-build.sh` against fresh clones of cli and opm-operator, and the cli
tests `go test ./internal/workflow/render/... ./internal/cmdutil/... ./internal/cmd/...`
and the operator tests `go test ./internal/render/...` through the same `GOWORK` that
script writes. Findings go into a "Downstream survey" section of this file.
**Rationale**: a newly refused fixture is a fix in its own repo (set the value, or give it
a default), not a reason to loosen the kernel. The owner chose the refusal knowing it
may tighten the cli.

## Risks / Trade-offs

- [A cli user's instance directory that rendered yesterday is refused] → it is the owner's
  chosen correction; the release note (the `BREAKING CHANGE:` footer) names it and the
  error names the field and, where CUE has one, its `#config` position.
- [A `#config` field typed `_` or a bare type with no default that the author meant as
  optional] → the author marks it optional (`foo?:`) or gives it a default; the operator
  already refuses such instances.
- [A `_` field is reported with no position] → its path names it; CUE records no position
  for `_`, and the kernel does not invent one.
- [A non-concrete value a source wrote is reported at the rendered values file, not the
  source] → unchanged by this change; attributing it to the source is a separate
  improvement of the built-spec check, listed as a follow-up.
- [The new check costs one unify-and-validate per instance] → no extra build; small next to
  the build itself.

## Downstream survey

Run 2026-10-05 against fresh clones of `main`: cli `cd10f2d`, opm-operator `c303a46`,
modules `eb4cb97`, opm-modules `f16a187`. Every target ran once on this change's kernel and
once on the library at `origin/main` `88afdfb` (`git archive`), through a scratch program in
a `go.work`. Registries: `opmodel.dev` and `testing.opmodel.dev` from GHCR,
`jacero.se` from `ghcr.io/emil-jacero`, and a second pass with `testing.opmodel.dev` from
the workspace's local registry for the cli and operator fixtures.

**Newly refused: none found.** Every target gave the same result on both kernels.

- Module `debugValues` through `SynthesizeInstance`, accepted on both: cli 10 (the
  templates, `tests/fixtures`, the render and `instinit` testdata, the integration and e2e
  testdata modules), modules 8, opm-modules 12, opm-operator 7 (the six
  `test/fixtures/modules` and `modules/opm_operator`, the last synthesized under the name
  and namespace its own guard requires, `opm-operator` in `opm-operator-system`).
- Instance packages through `AcquireInstanceFromDir`, accepted on both: cli 3
  (`examples/instances/podinfo`, `internal/workflow/render/testdata/skip-unprovided/instance`,
  `tests/e2e/testdata/operator-owned`), opm-operator 4 (`test/fixtures/modulepackages/*`).
  modules and opm-modules hold no instance package; the one directory each that mentions
  `#ModuleInstance` is a module (`ErrWrongKind`, skipped).
- opm-operator ModuleInstance CRs with `spec.values` through `SynthesizeInstance`, the
  module acquired from its registry: 7 accepted on both (`config/samples` hello and the six
  `test/fixtures/modules/*/moduleinstance.yaml`).
- Consumer build: cli and opm-operator build and vet against this tree
  (`.tasks/consumer-build.sh`, `GOTOOLCHAIN=local`).
- Consumer tests through the same `GOWORK`: cli `go test ./internal/... ./pkg/... ./hack/...`
  and opm-operator `go test ./internal/...` (the controller suite on envtest 1.36, 240 of
  240 specs) fail identically on both kernels. The only failures, in cli `hack/docskit-dump`
  (the pin reads `(devel)` under a `go.work`) and `internal/kubernetes` (the weight table
  that library #214 changed), predate this change.

Not reached, on both kernels alike: cli `tests/e2e/testdata/vet-errors/*` (refused by
design, a vet fixture), cli `tests/e2e/testdata/operator-owned` as a module (it is an
instance), opm-operator `hack/operator-module` (build constraints exclude every file), and
the opm-operator sample `config/samples/opmodel.dev_v1alpha1_moduleinstance_jellyfin.yaml`
(`opmodel.dev/modules/jellyfin@v1` no longer resolves; jellyfin moved to `jacero.se`). The
kind cluster suites of cli and opm-operator were not run.

Follow-ups in their own repos:

- opm-operator: after a library release with this change, delete the module renderer's
  `ValidateConfigDetailed` pre-check (`internal/render/kernel_module_renderer.go`) and, if
  nothing else calls it, `cueFindings`; compare the status wording.
- opm-operator: the jellyfin sample names a module path that no longer resolves.
- library: attribute the built-spec refusal of a non-concrete value a source wrote to the
  source's `Origin` instead of the rendered values file (Risks).
