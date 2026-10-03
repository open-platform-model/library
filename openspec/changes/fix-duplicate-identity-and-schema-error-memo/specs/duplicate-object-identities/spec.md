## MODIFIED Requirements

### Requirement: Duplicate rendered identities are detected with their producers

The library SHALL provide, under `opm/helper/`, a function that scans a render's compiled objects and returns every apply identity that two or more objects share. Two objects SHALL share an apply identity when they carry the same API group (the part of `apiVersion` before the first `/`, empty for the core group and for an object with no `apiVersion`), kind, namespace and name, namespace empty when the object carries none; the version part of `apiVersion` SHALL NOT distinguish two objects, because Kubernetes addresses one object under every version of its group. Each returned row SHALL carry the identity of the first object rendered with it (its `apiVersion` verbatim, kind, namespace and name) and every producer of it, a producer being the component and the transformer the kernel recorded on the object together with that object's own `apiVersion`, in the render's pair order. Rows SHALL be returned in the order the first object of each identity was rendered, so the result is deterministic for a given render. A render with no shared identity SHALL return no rows. The exported identity type SHALL keep its fields (apiVersion, kind, namespace, name).

#### Scenario: Two components render one cluster-scoped object

- **WHEN** two components of one instance each render a `TransformerRegistration` named after the instance
- **THEN** one row is returned carrying that identity with an empty namespace and both producers, component and transformer each

#### Scenario: Three objects, two identities

- **WHEN** a render produces a Deployment `web` twice from two transformers and a Service `web` once
- **THEN** one row is returned for the Deployment with two producers, and none for the Service

#### Scenario: A clean render returns nothing

- **WHEN** every rendered object has a distinct identity
- **THEN** no rows are returned

#### Scenario: One object under two versions of its group

- **WHEN** one transformer renders an `apps/v1` Deployment `web` and another renders an `apps/v1beta2` Deployment `web` in the same namespace
- **THEN** one row is returned whose identity carries `apps/v1`, and its two producers carry `apps/v1` and `apps/v1beta2` respectively

#### Scenario: One kind and name in two groups stays distinct

- **WHEN** a render produces a kind `Widget` named `web` under `a.example.com/v1` and under `b.example.com/v1`
- **THEN** no rows are returned

### Requirement: The refusal is worded once

The helper SHALL provide an error type aggregating duplicate rows, whose message names each identity and every producer as component and transformer, one identity per line, so the CLI and the operator refuse with one wording. When the producers of a row do not all carry the same `apiVersion`, the message SHALL name each producer's `apiVersion` beside it; when they do, the line SHALL carry no per-producer version. The kernel SHALL NOT return this error: it is raised by the runtime that calls the helper, before apply.

#### Scenario: The message names both carrying components

- **WHEN** the error is built from a row with two producers
- **THEN** its message names the identity once and both components with their transformers, and a reader can tell which module components to fix without looking elsewhere

#### Scenario: The message names a version mismatch

- **WHEN** the error is built from a row whose two producers carry `apps/v1` and `apps/v1beta2`
- **THEN** its message names each producer with its own `apiVersion`, so a reader sees that the duplicate is one object under two versions
