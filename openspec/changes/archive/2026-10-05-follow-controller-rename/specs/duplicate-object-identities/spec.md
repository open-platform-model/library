## MODIFIED Requirements

### Requirement: The refusal is worded once

`opm/k8s/object` SHALL provide an error type aggregating duplicate rows, whose message names each identity and every producer as component and transformer, one identity per line, so the CLI and the controller refuse with one wording. When the producers of a row do not all carry the same `apiVersion`, the message SHALL name each producer's `apiVersion` beside it, worded as `<no apiVersion>` for a producer whose object carries none; when they do, the line SHALL carry no per-producer version. The kernel SHALL NOT return this error: it is raised by the runtime that calls the check, before apply.

#### Scenario: The message names both carrying components

- **WHEN** the error is built from a row with two producers
- **THEN** its message names the identity once and both components with their transformers, and a reader can tell which module components to fix without looking elsewhere

#### Scenario: The message names a version mismatch

- **WHEN** the error is built from a row whose two producers carry `apps/v1` and `apps/v1beta2`
- **THEN** its message names each producer with its own `apiVersion`, so a reader sees that the duplicate is one object under two versions

#### Scenario: A same-version row carries no versions

- **WHEN** the error is built from a row whose producers all carry one `apiVersion`
- **THEN** its message line names the identity and the producers with no per-producer version

#### Scenario: A missing apiVersion is worded

- **WHEN** the error is built from a row whose producers carry `v1` and no `apiVersion`
- **THEN** its message names the second producer `as <no apiVersion>`
