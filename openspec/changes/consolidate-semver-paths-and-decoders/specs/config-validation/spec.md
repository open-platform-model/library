## MODIFIED Requirements

### Requirement: Internal Closed-Schema Workaround

The library SHALL retain `walkDisallowed` and `fieldNotAllowedError` as private internals of `opm/kernel/validate.go`. The error type SHALL implement `cuelang.org/go/cue/errors.Error` so that disallowed-field diagnostics flow alongside CUE-native errors transparently. The error's `Path()` SHALL return `values` followed by one segment per field label on the way to the disallowed field, each in CUE's selector form (a definition as `#name`, a plain label bare, a label that needs quoting quoted), with a leading `#module` `#config` pair or a leading `#config` omitted. A label that contains dots SHALL be one segment, so joining the segments with `.` gives the field's dotted path and splitting is never needed to recover a label.

#### Scenario: Disallowed field in closed schema produces positioned error

- **WHEN** validation runs against a closed schema and encounters a field the schema does not declare
- **THEN** the resulting error includes a `cueerrors.Error` with `Position()` pointing to the offending field in the user's source (not the schema's closure declaration)
- **AND** the error's `Path()` returns one segment per field label of the disallowed field, which joined with `.` give its dotted path

#### Scenario: A label containing dots is one path segment

- **WHEN** validation runs against a closed schema whose `labels` struct declares no `"app.kubernetes.io/name"` field and the values set `labels: "app.kubernetes.io/name": "web"`
- **THEN** the disallowed-field error's `Path()` is `["values", "labels", "\"app.kubernetes.io/name\""]`, three segments with the dotted label as one

#### Scenario: Internal types not exported

- **WHEN** a developer searches `opm/kernel/` for `WalkDisallowed` or `FieldNotAllowedError`
- **THEN** no exported symbol with that name exists
- **AND** the unexported helpers are documented in the package's internal godoc only
