# duplicate-object-identities Specification

## Purpose
The opt-in check between render and apply that finds rendered objects sharing one Kubernetes apply identity, so a runtime refuses the render before the last write silently wins (enhancement 0015 D15 and D12).
## Requirements

### Requirement: Duplicate rendered identities are detected with their producers

The library SHALL provide, under `opm/helper/`, a function that scans a render's compiled objects and returns every apply identity (apiVersion, kind, namespace and name, namespace empty when the object carries none) that two or more objects share. Each returned row SHALL carry the identity and every producer of it, a producer being the component and the transformer the kernel recorded on the object, in the render's pair order. Rows SHALL be returned in the order the first object of each identity was rendered, so the result is deterministic for a given render. A render with no shared identity SHALL return no rows.

#### Scenario: Two components render one cluster-scoped object

- **WHEN** two components of one instance each render a `TransformerRegistration` named after the instance
- **THEN** one row is returned carrying that identity with an empty namespace and both producers, component and transformer each

#### Scenario: Three objects, two identities

- **WHEN** a render produces a Deployment `web` twice from two transformers and a Service `web` once
- **THEN** one row is returned for the Deployment with two producers, and none for the Service

#### Scenario: A clean render returns nothing

- **WHEN** every rendered object has a distinct identity
- **THEN** no rows are returned

### Requirement: Objects without an apply identity are skipped

A rendered value that carries no `kind` or no `metadata.name` SHALL NOT be treated as a Kubernetes object: it SHALL be skipped by the scan and SHALL NOT produce a row or an error. The helper SHALL NOT validate the objects in any other way.

#### Scenario: A value without a name is ignored

- **WHEN** a transformer renders a value with a kind and no `metadata.name` beside two objects sharing an identity
- **THEN** the shared identity is returned and the nameless value contributes nothing

### Requirement: The refusal is worded once

The helper SHALL provide an error type aggregating duplicate rows, whose message names each identity and every producer as component and transformer, one identity per line, so the CLI and the operator refuse with one wording. The kernel SHALL NOT return this error: it is raised by the runtime that calls the helper, before apply.

#### Scenario: The message names both carrying components

- **WHEN** the error is built from a row with two producers
- **THEN** its message names the identity once and both components with their transformers, and a reader can tell which module components to fix without looking elsewhere

### Requirement: The kernel stays neutral

`opm/kernel` SHALL NOT import the helper, `Render` SHALL NOT refuse a duplicate identity, and `Compiled` SHALL gain no identity fields; the helper reads the identity off `Compiled.Value`. A frontend that applies to something other than Kubernetes MAY skip the helper entirely.

#### Scenario: A render with duplicates still succeeds in the kernel

- **WHEN** a render produces two objects with one identity
- **THEN** `Render` returns both in `Compiled` with no error, and only a caller of the helper learns of the collision
