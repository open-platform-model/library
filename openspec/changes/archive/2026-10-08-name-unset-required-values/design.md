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

**Context**: The brief of this change asks for one of two shapes: the values findings alone whenever a required value is unset, or both sets.

**Explored**: A first implementation tried a third shape: values findings first, then the built-spec findings that no unset value "explains", where a built-spec finding counted as explained when it shared a position with a values finding. The review of this change disproved it with two probes.

1. A component that is a catalog resource, the shape of a real module: the finding of a field that reads an unset value carries core's `let` line and the catalog's schema line, and not the `#config` declaration. The component findings stayed in the report. Only fields with no schema line of their own (`metadata.annotations`, the first fixture) carry the `#config` position.
2. A `#config` field and an unrelated component field typed by one shared declaration (`_S: string`): the unrelated defect shared a position and was dropped.

So the positions CUE prints do not say whether an unset value causes a finding. Nothing else on the error does.

**Decision**: The values findings alone. When a required value is unset the report holds the built-spec findings under `values` (a value the sources carry as a bare type, positioned where they carry it), then the required-value findings at every other path. Every built-spec finding outside `values` is left out.

**Rationale**:

- It depends only on finding paths, which both checks root at the instance, never on positions.
- It is what a user of the operator sees today: `ValidateConfigDetailed` runs first and reports `#config` findings only, and synthesis never runs on incomplete values. The operator can drop that pre-check with no loss.
- Both sets in full would bury the field names: one unset value read in three places gives one useful finding and three that name a catalog file, and the operator writes the findings into an event note with a 1024 character limit (review of opm-operator `drop-values-pre-validate`, "Uncertain"). `cueerrors.Details` sorts by position, so the values findings would not even print first.
- Cost, accepted: a defect of the module's own (an incomplete component field no value completes) is not reported while the values are incomplete. It is reported, with today's text, as soon as they are complete. No instance is accepted that was refused. A module author tests a module with complete values (`debugValues`), so the second round falls on few users.
- When no required value is unset the function returns the built-spec error itself, so "every other defect keeps its text" holds by construction.

A value the sources write as a bare type (`message: string`) is an unset required value by the requirement's own definition, so it takes the same shape: its `values.message` finding stays, the component findings it causes go. That row changes too (table below).

### Order of the findings

**Decision**: Built-spec `values` findings, then the required-value findings in CUE's order. Every finding is a values finding, so `Error()` (first finding and a count), `cueerrors.Errors` (the kernel's order) and `cueerrors.Details` (sorted by position) all show only fields the user sets.

### Testable by type

**Context**: library#223 added `*errors.ConfigValidationError` as the marker of `ValidateConfigDetailed`.

**Decision**: Not used here, and nothing added. Before and after, a caller has `errors.As(err, &cueerrors.Error)` and the finding paths: a path that starts with `values` under the `not fully concrete` frame is a values finding. After this change that test is reliable for the first finding whenever a required value is unset.

**Rationale**: The brief rules out an API change, and the path test is exact: when a required value is unset every finding is under `values`. A typed marker for "the values are incomplete" is an API addition for its own change, if the operator needs one.

## Design

```go
func concreteness(spec cue.Value) error {
    specErr := spec.Validate(cue.Concrete(true))
    requiredErr := requiredConfigSet(spec)
    if requiredErr == nil { return specErr }      // today's error, untouched
    if specErr == nil { return requiredErr }      // today's error, untouched
    // built-spec findings under `values`, then the required findings at any other path
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
| `message: string` written, rest set | `values.message` and 2 component findings | `values.message` |
| `message: string` written, `other` unset | `values.message` and 2 component findings; `other` missing | `values.message`, `values.other` |
| component `loose: string`, values set | `components.foo.metadata.annotations.loose` | same |
| component `loose: string`, `message` unset | `...annotations.message`, `...annotations.loose` | `values.message` |
| component `loose: string`, `other` unset | `...annotations.loose`; `other` missing | `values.other` |
| catalog resource reads `note` three times, `note` unset | 3 findings, component paths, catalog and core positions | `values.note` |

## Consumer assertions on the old text

Searched in throwaway copies of cli at origin/main b1eea50 and opm-operator at origin/main 0f75ccb (test files, for `not fully concrete`, `incomplete value` and component paths). Neither repo was edited.

- No test in either repo asserts a component path for an unset value, so none depends on the old text.
- opm-operator `internal/render/required_values_test.go` asserts the exact kernel text `Kernel.AcquireInstanceFromDir: instance "needy": not fully concrete: values.note: incomplete value string` and the prefix `not fully concrete: values.tier: incomplete value ` (`TestKernelPackageRenderer_UnsetRequiredValueIsRefused`). Both values are read by no component; the text is unchanged, and `go test ./internal/render -run UnsetRequiredValue` passes in that copy against this library tree.
- The same file asserts `#config.note: incomplete value string (...)`, the text of `ValidateConfigDetailed`, which this change does not touch.
- cli `tests/e2e/vet_output_test.go` asserts the substrings `incomplete value` and `incomplete value int` and cli `internal/publish/identity_test.go` asserts `incomplete value`; all hold for the new text. The e2e tests were not run.
- The operator's unmerged change `drop-values-pre-validate` is the consumer of the new text: its requirement that synthesis names `values.<field>` for a value a component reads becomes true with this change.

## Risks / Trade-offs

- A consumer test that asserts a component path for an unset value fails after a pin bump → listed for the consumers; the framing `not fully concrete` and the `values.` prefix stay.
- A defect of the module's own is hidden while a required value is unset → it is refused with today's text once the values are complete; nothing is accepted that was refused.
