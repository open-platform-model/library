## Context

`processInstance` (`opm/kernel/process.go`) is the one processing step behind `SynthesizeInstance` and `AcquireInstanceFromDir`. It checks concreteness twice: `spec.Validate(cue.Concrete(true))` on the whole built spec, then `requiredConfigSet`, which validates the spec's `values` unified with the module's `#config`. It returns at the first failure. `#ModuleInstance` feeds `values` to the components through a `let`, so an unset value a component reads makes the component field incomplete, the first check fails, and the second, the only one that names `values.<field>`, never runs.

Measured on origin/main d21adc9 with a module whose `#config` is `message: string, other: int, port: int & >0 | *80, db: {host: string, size: int | *1}` and whose component reads `message` (twice, once in an interpolation), `db.host` and `port` into `metadata.annotations`, values `{}`:

```text
Kernel.SynthesizeInstance: instance "myrel": not fully concrete: components.foo.metadata.annotations.message: incomplete value string (and 2 more errors)
  components.foo.metadata.annotations.host: incomplete value string      (core module_instance.cue:104:12, module.cue:15:13)
  components.foo.metadata.annotations.message: incomplete value string   (core module_instance.cue:104:12, module.cue:12:11)
  unifiedModule.#components.foo.metadata.annotations.again: invalid interpolation: non-concrete value string (type string)   (module.cue:18:100, module.cue:12:11)
```

`other` is not named. No `values` path appears.

## Goals / Non-Goals

**Goals:**

- Every unset required value is named at `values.<field>`, first, read or not, on both verbs.
- Every refusal with no unset required value keeps its text.
- No change to which instances are refused, and none to the exported API.

**Non-Goals:**

- `ValidateConfigDetailed` and `opm/errors` classification stay as they are.
- No error type for a required-value refusal (see "Testable by type").
- The operator's and the cli's assertions on the old text; they are listed below for their follow-up.

## Research & Decisions

### Which findings the report holds

**Context**: The brief of this change asks for one of two shapes: the required-value findings alone whenever any exist, or both sets.

**Explored**: A prototype of each against fourteen value documents (the table under "Before and after"). Three facts came out of it.

1. Every built-spec finding that an unset value causes carries the position of that value's `#config` declaration (`module.cue:12:11` above), next to the position of the place that reads it. The required-value finding for the same field carries the same position.
2. A component defect that has nothing to do with the values (`loose: string` in a component) carries no `#config` position.
3. For a value the values carry in a non-concrete form (`message: string` written in a source), both checks report `values.message`; the built-spec finding is positioned in the values file, which is where the user must edit.

**Decision**: Neither shape as stated; a third that the facts allow. The report holds the required-value findings first, then every built-spec finding that no unset required value explains. A built-spec finding is explained, and left out, when it shares a position with a required-value finding. Where both checks report the same path, the built-spec finding is the one kept.

**Rationale**:

- The required-value findings alone would hide a module defect (fact 2) until the user has fixed the values: a second round for a different audience (the module author), and a refusal whose text says less than the kernel knows.
- Both sets in full would bury the field name. One unset value read in three places gives one useful finding and three that name a catalog file; the operator writes the findings into an event note with a 1024 character limit (review of opm-operator `drop-values-pre-validate`, "Uncertain").
- Position sharing is evidence CUE itself records, not a guess from the path. When it is absent (a field declared `_` has no position), the finding is kept: the rule errs toward saying more.
- When no required value is unset the function returns the built-spec error itself, so "every other defect keeps its text" holds by construction and not by re-assembly.

### Order of the findings

**Decision**: `values` findings first. `Error()` of a CUE error list prints its first element and a count, so the one-line text names a field. `cueerrors.Details` and `cueerrors.Print` sort by position before printing; the kernel does not control that order, and in the mixed case (an unexplained component finding next to a values finding) the sorted print can show the component finding first. `cueerrors.Errors(err)` returns the kernel's order.

### Testable by type

**Context**: library#223 added `*errors.ConfigValidationError` as the marker of `ValidateConfigDetailed`.

**Decision**: Not used here, and nothing added. Before and after, a caller has `errors.As(err, &cueerrors.Error)` and the finding paths: a path that starts with `values` under the `not fully concrete` frame is a values finding. After this change that test is reliable for the first finding whenever a required value is unset.

**Rationale**: The refusal can now hold component findings next to values findings, so wrapping the whole tree in `ConfigValidationError` would mislabel them, and the brief rules out an API change. A typed marker for "the values are incomplete" is an API addition for its own change, if the operator needs one.

## Design

```go
func concreteness(spec cue.Value) error {
    specErr := spec.Validate(cue.Concrete(true))
    requiredErr := requiredConfigSet(spec)
    if requiredErr == nil { return specErr }      // today's error, untouched
    if specErr == nil { return requiredErr }      // today's error, untouched
    // required findings in order, the built-spec finding in place of one at the same path;
    // then built-spec findings that share no position with a required finding
}
```

`processInstance` wraps the result once with `instance %q: not fully concrete: %w`, as today. `requiredConfigSet` is unchanged. Both checks always run; the second is one more `Validate` over `values & #config`, which the passing path already paid for.

## Before and after

`SynthesizeInstance`, the module of "Context". Paths only; the tests pin the full text.

| Values | Before | After |
| --- | --- | --- |
| `{}` | 3 findings, component paths; `other` missing | `values.message`, `values.other`, `values.db.host` |
| `message` unset | 2 findings, component paths | `values.message` |
| `db.host` unset | `components.foo.metadata.annotations.host` | `values.db.host` |
| `other` unset (unread) | `values.other` | same |
| `message: 7` | `#module.#config.message: conflicting values string and 7 (mismatched types string and int)` | same |
| `port: -1` | `#module.#config.port: 2 errors in empty disjunction: (and 2 more errors)` | same |
| `extra: 1` | `field not allowed` (finding at `values.extra`) | same |
| `port: int` | `values.port: incomplete value int` | same |
| `message: string` written, rest set | `values.message` and 2 component findings | same |
| `message: string` written, `other` unset | `values.message` and 2 component findings; `other` missing | the same three and `values.other` |
| component `loose: string`, values set | `components.foo.metadata.annotations.loose` | same |
| component `loose: string`, `message` unset | `...annotations.message`, `...annotations.loose` | `values.message`, `...annotations.loose` |
| component `loose: string`, `other` unset | `...annotations.loose`; `other` missing | `values.other`, `...annotations.loose` |

## Consumer assertions on the old text

Searched in the cli checkout at ef54c046 and the opm-operator checkout at 59537b7 (test files, for `not fully concrete`, `incomplete value`, `values.<field>:` and component paths). Nothing was edited.

- No test in either repo asserts a component path for an unset value, so none fails on the new text.
- cli `tests/e2e/vet_output_test.go` (`TestE2E_ModuleVet_OpenDebugValuesRefusedAtSynthesis`) asserts the substring `incomplete value` on a synthesis refusal; it holds before and after.
- cli `tests/e2e/vet_output_test.go` (the `open-debug-values` case: `values do not satisfy #config`, `values.replicas`) and opm-operator `internal/render/required_values_test.go` assert the text of `ValidateConfigDetailed`, which this change does not touch.
- opm-operator `test/integration/reconcile/kernel_module_renderer_test.go` asserts the substring `spec.values` of the operator's own pre-check message.
- The operator's unmerged change `drop-values-pre-validate` is the consumer of the new text: its requirement that synthesis names `values.<field>` for a value a component reads becomes true with this change.

## Risks / Trade-offs

- A consumer test that asserts a component path for an unset value fails after a pin bump → listed for the consumers; the framing `not fully concrete` and the `values.` prefix stay.
- A built-spec finding that shares a `#config` position with an unset value and is also wrong for another reason is left out until the value is set → it then appears; nothing is accepted that was refused.
- The rule depends on CUE recording the declaration position on both findings (cuelang.org/go v0.17.1) → the tests assert the exact finding count for the read case, so a CUE bump that drops the position fails them by showing more findings, never fewer.
