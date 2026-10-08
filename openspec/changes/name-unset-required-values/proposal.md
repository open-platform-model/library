## Why

Since library#217 the kernel refuses an instance whose values leave a required `#config` value unset, and names it at `values.<field>` only when no component reads it. In the normal case a component reads the value: the concreteness check on the built spec fails first and is the only one reported, so the refusal names component paths (`components.<...>: incomplete value string`, positioned in a catalog file and in core), never the field, and a second unset value is not reported at all. The operator keeps its own pre-check (`ValidateConfigDetailed`) only because of this (opm-operator#258 and the review of its change `drop-values-pre-validate`).

## What Changes

- The instance processing step of `SynthesizeInstance` and `AcquireInstanceFromDir` runs both concreteness checks and returns one report. When a required `#config` value is unset, the report names every such value at `values.<field>`, whether or not a component reads it, and holds nothing else: the findings of the built spec outside `values` are left out until the values are complete.
- When no required value is unset, the refusal is the one returned today, to the character. A wrong type, a failed constraint, a field `#config` does not declare and an incomplete component value with complete values keep their text.
- The set of refused instances does not change. The Go API does not change. No error type is added.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `kernel-runtime`: the requirement "Instance verbs refuse an unset required config value" no longer lets the built-spec check hide the unset values; its scenario "A value the built spec refuses is reported once" gains the two values it used to leave out.

## Impact

- Code: `opm/kernel/process.go` (`processInstance` and one new unexported helper), godoc of `SynthesizeInstance` and `AcquireInstanceFromDir`, `opm/kernel/required_config_test.go`.
- SemVer: PATCH (`fix`). No exported symbol changes; `task api:diff` reports nothing. The text of one class of refusal changes, which a consumer test can assert on: the operator and the cli follow up in their own changes (design.md lists the assertions found).
- Consumers: opm-operator can drop `ValidateConfigDetailed` before synthesis once it pins a release with this change. The cli needs no change to build.
- Complexity (Principle VII): about twenty lines that merge two CUE error lists by path. A module's own incomplete field is reported only once the values are complete; design.md says why the report cannot tell it from a field that reads an unset value.
