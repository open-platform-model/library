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

### Requirement: The helper copy is deprecated and kept until both frontends migrate

`opm/helper/objectset`, the check's earlier home, SHALL stay in the library with its exported names and behaviour unchanged. Its rows and error message SHALL be identical to those of `opm/k8s/object` for the same render. Its package documentation and each exported symbol SHALL carry a `Deprecated:` notice that names the replacement in `opm/k8s/object`. It SHALL be removed only by a later change, merged after both the cli and the controller have stopped importing it. Source: the library rule deprecate, then remove: a library change never removes API that a frontend imports at `main` (AGENTS.md, Consumer build paragraph).

#### Scenario: Both homes agree on a render

- **WHEN** the same compiled objects are scanned by `opm/helper/objectset` and by `opm/k8s/object`
- **THEN** both return rows with the same identities and producers in the same order, and both errors carry the same message

#### Scenario: The deprecated home says where to go

- **WHEN** a developer reads the documentation of `opm/helper/objectset` or of its `Duplicates`
- **THEN** it is marked Deprecated and names `opm/k8s/object` as the replacement
