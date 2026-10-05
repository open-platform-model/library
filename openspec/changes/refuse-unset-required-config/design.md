## Context

See proposal.md, Why. Design-local decisions are numbered RC1 to RC5, so they do not collide
with any other numbering. Locations are at `origin/main` `88afdfb`:

- core `src/module_instance.cue:88`: `let unifiedModule = #module & {#config: values}`, and
  `values: _` at `:110`. Components are read from `unifiedModule`; a `#config` field no
  component reads never reaches the instance's output.
- `opm/kernel/process.go` `processInstance(spec)`: `spec.Validate(cue.Concrete(true))`,
  framed `instance %q: not fully concrete: %w`, then metadata decoding. Both verbs call it
  last. `Validate` does not descend into definitions, so `#module.#config` is never checked
  here.
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

A probe with the cue v0.17.1 CLI over `#config: {replicas: int | *1, image: string, tag!:
string, opt?: string, any: _}` unified with `{replicas: 2}` reports, under concreteness:
`any: incomplete value _`, `image: incomplete value string` (at the `image` declaration)
and `tag: field is required but not present` (at the `tag` declaration). `opt` and
`replicas` pass. Section 1 confirms the same through the kernel and records the exact paths
and positions.

## Goals / Non-Goals

**Goals:** both instance verbs refuse an instance whose effective values leave a required
`#config` value unset, whether or not a component reads it; the refused set equals what
`ValidateConfigDetailed(#config, values)` refuses for the same values; the refusal sits at
the field's `#config` position, and at the source's `Origin` where a source wrote the
non-concrete value; every error the two verbs return today for an instance they refuse
today stays the same.

**Non-Goals:** deleting the operator's pre-validate (an operator change after the library
release); typing values errors (the kernel returns CUE's error tree, config-validation "No
Custom Validation Error Types"); changing `ValidateConfigDetailed`; changing the
failure-path attribution of library #197; rewording the path CUE reports (`#module.#config`
prefix, RC4).

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
**Decision**: (c). `processInstance(spec, compiled)` runs the existing
`spec.Validate(cue.Concrete(true))` first, unchanged, then validates
`#module.#config` (read off `spec`) unified with the spec's `values` and the call's
compiled sources, with concreteness, and frames a failure the same way:

```go
func processInstance(spec cue.Value, compiled []cue.Value) (*module.Instance, error) {
	name := bestEffortInstanceName(spec)
	if err := spec.Validate(cue.Concrete(true)); err != nil {
		return nil, fmt.Errorf("instance %q: not fully concrete: %w", name, err)
	}
	if err := requiredConfigSet(spec, compiled); err != nil {
		return nil, fmt.Errorf("instance %q: not fully concrete: %w", name, err)
	}
	// metadata decoding unchanged
}

// requiredConfigSet: #config & values & compiled..., under cue.Concrete(true).
func requiredConfigSet(spec cue.Value, compiled []cue.Value) error {
	configSchema := spec.LookupPath(schema.Module).LookupPath(schema.Config)
	built := spec.LookupPath(schema.Values)
	if !configSchema.Exists() || !built.Exists() {
		return nil
	}
	return configSchema.Unify(unifyValues(append([]cue.Value{built}, compiled...))).
		Validate(cue.Concrete(true))
}
```

**Rationale**: one place, both verbs, identical by construction (instance-synthesis
"Instance construction shares one evaluate-and-shape-gate with the file loader"). Running
second keeps every error of an instance refused today: a non-concrete value a component
reads, or a non-concrete `values` field, still fails the spec check first with the same
text. The new check fires only for an instance the kernel accepts today.

### RC2. Effective values, not sources alone

**Context**: on the acquire path the instance's values are the package's own `values` plus
the trailing sources; on the synthesis path they are the merged sources only.
**Decision**: validate against the built spec's `values` (which already carries every
source through the rendered values file), with the compiled sources unified in as well.
**Rationale**: the built `values` is what the instance deploys, on both verbs. Unifying the
compiled sources again changes no value (they are already conjuncts of `values`) but adds
their `Origin` positions to an error about a value a source wrote, the attribution the
owner asked for "where possible". A field nobody wrote has no source position; its
position is its `#config` declaration, as CUE reports it. Section 1 verifies both claims;
if the double unification changes a result or a position, the spike records it and the
check falls back to the built `values` alone.

### RC3. Same refused set as `ValidateConfigDetailed`

**Context**: the operator deletes its pre-validate only if the kernel refuses at least what
it refuses.
**Decision**: the concreteness rule is the one `validateValues(…, true)` applies
(`schema.Unify(merged).Validate(cue.Concrete(true))`). The disallowed-field walk and type
checks are not repeated here: the post-build per-source checks have already run them on
both verbs, and the spec check covers what the build reads. A parity test runs the
operator's shape, `ValidateConfigDetailed(mod.ConfigSchema(), []Source{src})`, and
`SynthesizeInstance` with the same source over a table of values (all set, an unread
`string` unset, an unread `foo!` unset, an unread `_` unset, an optional field unset, a
defaulted field unset, a read field unset) and asserts that each refuses exactly when the
other does.
**Rationale**: the parity is the condition for the operator's deletion, so it is pinned by
a test, not by argument.

The no-values synthesis path is unchanged: with an empty `Values` the values path stays
`_`, the spec check refuses it as today, and the operator's `{}` source reaches the new
check as one compiled value.

### RC4. Error shape and framing

**Context**: the refusal must be readable at the field's position and walkable by
frontends.
**Decision**: return CUE's error tree unchanged, wrapped as
`Kernel.<Verb>: instance "<name>": not fully concrete: <cue error>`. CUE names the path as
reached from the spec (`#module.#config.<field>`); the kernel does not rewrite it.
**Rationale**: config-validation "No Custom Validation Error Types": the kernel returns
CUE-native errors and frontends own presentation. The `not fully concrete` frame keeps one
wording for every concreteness refusal of the two verbs (artifact-types "Validation
failures propagate"), so a frontend that matches on the instance frame needs no new case.
The operator's pre-validate today words the same finding under `#config.<field>` with
`validating values against the module's #config:`; the operator change that deletes it
compares the wording and decides whether its status message needs a prefix.

### RC5. Downstream survey decides nothing in the library

**Context**: the supervisor asked for a list of instances in the cli and operator fixtures
and the module fleets that this newly refuses.
**Decision**: section 3 builds a scratch program (in the session scratchpad, never
committed) in a `go.work` with this worktree, which runs every on-disk `ModuleInstance`
package found in cli, opm-operator, modules and opm-modules (at their `origin/main`, from
fresh clones or `git archive`, never another session's worktree) through
`AcquireInstanceFromDir` and reports each refusal. It also runs
`.tasks/consumer-build.sh` against fresh clones of cli and opm-operator, and the
operator's registry-backed integration specs that synthesize fixture instances, where that
is cheap. Findings go into a "Downstream survey" section of this file.
**Rationale**: a newly refused fixture is a fix in its own repo (set the value, or give it
a default), not a reason to loosen the kernel. The owner chose the refusal knowing it
may tighten the cli.

## Risks / Trade-offs

- [A cli user's instance directory that rendered yesterday is refused] → it is the owner's
  chosen correction; the release note names it and the error names the field and its
  `#config` position.
- [A `#config` field typed `_` or a bare type with no default that the author meant as
  optional] → the author marks it optional (`foo?:`) or gives it a default; the operator
  already refuses such instances.
- [The new check costs one unify-and-validate per instance] → no extra build; small next to
  the build itself.
