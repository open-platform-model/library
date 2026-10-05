## Context

See proposal.md, Why. Design-local decisions are numbered CV1 to CV5, so they do not collide
with any other numbering. Locations are at `origin/main` `a8bfc76`:

- `opm/kernel/acquire.go:356-369` `mergeSources(cueCtx, sources, env) (cue.Value, error)`:
  `compileSources`, then `unifyValues`, then a merged-value error check.
- `acquire.go:377-416` `loadInstanceWithValues`: `dirSource` (the authored overlay, read
  once), `mergeSources`, then the rendered `opm-values.cue` added to a clone. On a `LoadDir`
  failure it calls `attributeValuesError(cueCtx, authored, sources)` (`:409`).
- `acquire.go:433-446` `checkInstanceValues(spec, sources)`: `validateSources(configSchema,
  sources, env, false)` (a compile), then `validateValues` over the built `values`.
- `acquire.go:457-477` `attributeValuesError(cueCtx, authoredSrc, sources)`: builds the
  authored overlay, then `compileSources` again (`:467`), then
  `validateValues(configSchema, own+compiled, true)`.
- `opm/kernel/synth.go:97-169` `SynthesizeInstance`: `mergeSources` (`:126`), `synth.Instance`
  (`:131`) with the build error returned as is (`:140-142`), then
  `validateSources(configSchema, in.Values, env, false)` (`:151`, a compile), then
  `processInstance`.
- `opm/kernel/validate.go:56-68` `validateSources`: short-circuits on no sources or a missing
  schema, then `compileSources(schema.Context(), …)` and `validateValues`.
  `validate.go:79-105` `validateValues(schema, values, requireConcrete)` short-circuits on a
  merged value that does not exist.
- `opm/internal/synth/instance.go:155-196` `Instance(cueCtx, coreVersion, in)`: a zero
  `in.Values` writes no `values.cue` (`buildOverlay`, `:218-224`), so the package builds
  with `values` left open.
- core `src/module_instance.cue:88`: `let unifiedModule = #module & {#config: values}`.
  Components are read from `unifiedModule`, so a value that violates `#config` fails the
  build only where a component consumes it. Otherwise the build passes and the post-build
  check reports it.
- `module.Module.ConfigSchema()` (`opm/module/module.go:30-35`) is
  `m.Package.LookupPath(#config)`. It reads the module's `Package`, built in the context of
  the acquiring call.

## Goals / Non-Goals

**Goals:** every values source compiled once per `AcquireInstanceFromDir` or
`SynthesizeInstance` call; `SynthesizeInstance` attributes a values conflict that fails its
build exactly as `AcquireInstanceFromDir` does; no change to any exported signature,
sentinel, `Kernel.<Verb>:` prefix or success-path error text.

**Non-Goals:** the operator's deletion of its pre-validate (op-i3g2); context cancellation
between stages and typed fetch errors (d1+d3, next on this chain); a per-Kernel registry
client (g5 part A); changing `AcquireInstanceFromDir`'s attribution mode (CV3); a runtime
counter that proves the compile count (CV5); the general comment pass (c4).

## Research & Decisions

### CV1. `mergeSources` returns the compiled values, and a `validateCompiled` keeps the short-circuit

**Context**: b1 asks for one compile per source. Every consumer of a second compile runs in
the same `cue.Context` as the merge: `checkInstanceValues` and the synth post-build check
take `#config` from the spec built in `cueCtx`, and `attributeValuesError` builds the authored
package in `cueCtx`. So the values the merge compiled can unify with those schemas directly.
**Explored**: (a) a struct that carries the compiled slice beside the merged value; (b) a
second return value; (c) caching inside `Source` (refused: `Source` carries no `cue.Value`,
config-validation "Source Type and Layered Input").
**Decision**: (b).

```go
// mergeSources compiles a values stack in cueCtx once and returns both the
// compiled values (for the checks that follow the build) and their merge.
func mergeSources(cueCtx *cue.Context, sources []Source, env []string) (compiled []cue.Value, merged cue.Value, err error)

// validateCompiled is validateSources without the compile: values already
// compiled in schema's context.
func validateCompiled(schema cue.Value, values []cue.Value, requireConcrete bool) (cue.Value, error) {
	if len(values) == 0 || !schema.Exists() {
		return cue.Value{}, nil
	}
	return validateValues(schema, values, requireConcrete)
}

func validateSources(schema cue.Value, sources []Source, env []string, requireConcrete bool) (cue.Value, error) {
	if len(sources) == 0 || !schema.Exists() {
		return cue.Value{}, nil
	}
	values, err := compileSources(schema.Context(), sources, env)
	if err != nil {
		return cue.Value{}, err
	}
	return validateCompiled(schema, values, requireConcrete)
}
```

`loadInstanceWithValues` returns the compiled slice as a fourth result (nil on the no-values
branch). `checkInstanceValues(spec, compiled)` and the synth post-build check call
`validateCompiled(configSchema, compiled, false)`. The attribution helper (CV2) takes the
slice. After the change `compileSources` has two callers, `mergeSources` and
`validateSources`, and `validateSources` has one non-test caller,
`ValidateConfigDetailed`.
**Rationale**: the smallest change that removes every second compile. `validateCompiled`
keeps `validateSources`'s missing-schema short-circuit, which `validateValues` does not have
(`walkDisallowed` over a missing schema is not a case anyone has pinned). The public verb and
the `validate_internal_test.go` pins on `validateSources` do not change.

### CV2. Synthesis takes `#config` from a values-free rebuild, never from `in.Module.Package`

**Context**: g2 says synthesis "mirrors AcquireInstanceFromDir's failure-path values
attribution (no Package read)". The plan entry proposed `in.Module.ConfigSchema()`. The
supervisor read that as consistent with "no Package read", because no spec exists on the
failure path, and asked review to confirm it.
**Explored**:
1. `in.Module.ConfigSchema()`. It is `Package.LookupPath(#config)`
   (`opm/module/module.go:30-35`), so it is a Package read, which g2 rules out in so many
   words. It would also break two promises the code already makes: the `SynthesizeInstance`
   godoc says the method reads the module's Metadata and Source, "never its Package"
   (`synth.go:84-86`), and the kernel-runtime spec rules out unifying values across calls
   (the build "SHALL run in a context the method creates and releases"). CUE itself would
   allow the unification: since cuelang.org/go v0.17, values from different contexts may be
   combined (`cue/context.go:61-63`). The reason to refuse is the owner's rule and the two
   promises, not CUE.
2. Rebuild the synthesized package without the values file in `cueCtx`
   (`synth.Instance(cueCtx, coreVersion, input with Values zero)`). Read
   `#module.#config` off that build and validate the compiled values against it. This is
   exactly what `AcquireInstanceFromDir` does: it rebuilds the package as authored, without
   the rendered values file, and reads `#config` from it.
**Decision**: option 2. One helper serves both verbs:

```go
// valuesConflict validates compiled (the call's values, compiled once in the
// context authored was built in) together with authored's own values against
// the #config of authored, an instance package built without the rendered
// values file. It returns the raw CUE error, or nil when the values are clean
// or there is nothing to check.
func valuesConflict(authored cue.Value, compiled []cue.Value) error
```

`attributeValuesError` becomes: build the authored overlay (unchanged from b2); on a load
error return nil; otherwise `valuesConflict(authored, compiled)` framed
`Kernel.AcquireInstanceFromDir: instance %q: %w` with `bestEffortInstanceName(authored)`.
In `SynthesizeInstance`, when `synth.Instance` fails and the merged values exist, rebuild
with `Values` zero. A call with no sources, or only empty ones, has nothing to attribute.
Without this guard, the rebuilt package's open `values: _` alone would fail the concreteness
check on any required `#config` field. If the rebuild fails, return the original build error. Otherwise
run `valuesConflict` and frame a hit as `Kernel.SynthesizeInstance: instance %q: %w` with
`in.Name`, which is the name the spec would carry. Every other outcome returns the original
`Kernel.SynthesizeInstance: <build error>` unchanged.
**Rationale**: it follows the owner's two words, "mirrors" and "no Package read", literally,
and keeps the godoc's "never its Package" and the per-call context rule true. It costs one
extra build, and only on a failure path that already ended in an error. The plan entry's
reading (ConfigSchema from the Module) is departed from on purpose; plan review confirmed
this.

### CV3. Same concreteness mode as the acquire path

**Context**: `attributeValuesError` validates with `requireConcrete` true. A build that fails
for a reason unrelated to values, while the values are also incomplete, is then reported as
the missing values rather than the build error.
**Explored**: `requireConcrete` false for synthesis only. That would make the two paths
differ, against g2's "mirrors" and the instance-synthesis requirement that values
attribution is identical.
**Decision**: one helper, one mode (true) for both verbs, as today on the acquire path. The
masking case is recorded as an open point for the supervisor, not changed here.
**Rationale**: the owner decided "mirror". Changing the acquire path's mode is a behaviour
change no decision covers.

### CV4. Tests: a component that consumes the value

**Context**: the existing `TestKernel_SynthesizeInstance_ViolationAttributedToSource` builds
successfully, because no component reads `replicas`. It exercises the post-build check, not
the failure path. g2 needs a fixture whose build fails.
**Decision**: a synth module fixture with a component that reads `#config.replicas` through
a hidden field (`_r: #config.replicas & int`; a component's `spec` is closed over its
resources, so a hidden field is the smallest shape core accepts), fed `replicas: "three"`
from a `Source` with Origin `/values/bad.cue`. Section 1 adds the test with the assertions
that hold today: an error, no instance, and the path `replicas` named. Before this change
the build error is framed `Kernel.SynthesizeInstance: instance synthesis: building instance
package from …/opm-synth-instance: unifiedModule.#components.foo._r: conflicting values …`,
and no position names the source (recorded when section 1 ran; see task 1.1). Section 3
tightens the test to assert that a position names `/values/bad.cue` and that the message
reads `Kernel.SynthesizeInstance: instance "myrel": `.

The build-error-unchanged tests need failures that do not come from the values:

- A component that fails for every instance: the module declares `#ctx: _` and a component
  field `_n: #ctx.instance.namespace & "elsewhere"`. It acquires cleanly and fails every
  instance build in namespace `default`, the values-free rebuild included. A component field
  set to two conflicting literals does not work: the module's root fails, so acquisition
  refuses it. A failure that depends on values does not work either for this case: without
  values, `#config.replicas & >5` is only incomplete, so the rebuild succeeds.
- That second shape is the other branch: `_r: #config.replicas & >5` fed a clean
  `replicas: 2` fails the build, the rebuild succeeds, the values are clean, and the original
  error comes back.

The first module is also the no-values case. The instance-synthesis scenario gets a test that
feeds the violating fixture through both verbs and asserts that both name the source's
Origin: the module is published to an in-memory registry and authored as a directory
instance importing it (`writeImportedInstance`), as the parity probe fixtures do.
**Rationale**: the failure path is otherwise uncovered. Pinning first keeps section 1 green
on `origin/main`.

### CV5. No compile counter

**Context**: the plan entry allows a compile-count assertion "only if that is cheap".
**Explored**: a package-level hook (refused: Principle I, no package-level mutable state); a
counting registry behind a file-backed source's import (heavy, and it counts module fetches,
not compiles); deleting the origin file between compiles (not observable: a missing file
falls back to compiling `Data` as bytes).
**Decision**: no runtime counter. The compile count is checked by reading the diff: after the
change, `compileSources` is reached from `mergeSources` (once per verb call) and from
`validateSources` (only `ValidateConfigDetailed`). The verify step checks it with `grep`.
The spec gains no normative sentence for it: a SHALL that no scenario exercises would stay
green while a refactor breaks it, so compile-once stays a design property of this change.
**Rationale**: b1's value is the simpler code path. Its run-time gain is small (the research
says so), so a test seam would cost more than it proves.

## Risks / Trade-offs

- [The values-free synth rebuild fails for a reason the values build did not hit] → the
  original build error is returned, so the result is the same as today.
- [Reusing compiled values across two unifications] → `cue.Value` is immutable and `Unify`
  returns a new value, so the merge and the later check do not share state. The existing
  layered-sources tests (`LayeredSourcesUnifyInOrder`, `WithSources_ConflictAttributed`) are
  the guard.
- [Error text drift] → success-path and non-values failure text is unchanged. Only a values
  conflict that fails the synth build changes, which is the point of g2. The squash body says
  so for the operator's op-i3g2 comparison.
- [CV3 masking] → open point, unchanged from today's acquire behaviour.

## Open Points

- CV2 departs from the plan entry's `in.Module.ConfigSchema()` reading, for the context and
  Package-read reasons above. Review should confirm.
- CV3: both attribution paths require concreteness, so incomplete values can mask an
  unrelated build error. Raise this with the owner only if a frontend reports it.
