# duplicate-object-identities Specification

## Purpose
The opt-in check between render and apply that finds rendered objects sharing one Kubernetes apply identity, so a runtime refuses the render before the last write silently wins (enhancement 0015, "One registration per module" and "The rendered CR's name is instance-derived: dot-joined `namespace.name`").

## Requirements

### Requirement: Objects without an apply identity are skipped

A rendered value that carries no `kind` or no `metadata.name` SHALL NOT be treated as a Kubernetes object: it SHALL be skipped by the scan and SHALL NOT produce a row or an error. The check SHALL NOT validate the objects in any other way.

#### Scenario: A value without a name is ignored

- **WHEN** a transformer renders a value with a kind and no `metadata.name` beside two objects sharing an identity
- **THEN** the shared identity is returned and the nameless value contributes nothing

### Requirement: The kernel stays neutral

`opm/kernel` SHALL NOT import the check, `Render` SHALL NOT refuse a duplicate identity, and `Compiled` SHALL gain no identity fields; the check reads the identity off `Compiled.Value`. A frontend that applies to something other than Kubernetes MAY skip the check entirely.

#### Scenario: A render with duplicates still succeeds in the kernel

- **WHEN** a render produces two objects with one identity
- **THEN** `Render` returns both in `Compiled` with no error, and only a caller of the check learns of the collision

### Requirement: A function in opm/k8s/object returns every apply identity that two or more rendered objects share
The library SHALL provide, in the Kubernetes tier package `opm/k8s/object`, a function that scans a render's compiled objects and returns every apply identity that two or more objects share.
**Source**: enhancement 0015 ("One registration per module" and "The rendered CR's name is instance-derived: dot-joined `namespace.name`"); the tier placement is ADR-011 item 9.

#### Scenario: Three objects, two identities

- **WHEN** a render produces a Deployment `web` twice from two transformers and a Service `web` once
- **THEN** one row is returned for the Deployment with two producers, and none for the Service

### Requirement: Two objects share an apply identity when their API group, kind, namespace and name agree and the version part of apiVersion does not distinguish them
Two objects SHALL share an apply identity when they carry the same API group (the part of `apiVersion` before the first `/`, empty for the core group and for an object with no `apiVersion`), kind, namespace and name, namespace empty when the object carries none; the version part of `apiVersion` SHALL NOT distinguish two objects, because Kubernetes addresses one object under every version of its group.
**Source**: enhancement 0015 ("One registration per module" and "The rendered CR's name is instance-derived: dot-joined `namespace.name`"); the tier placement is ADR-011 item 9.

#### Scenario: One object under two versions of its group

- **WHEN** one transformer first renders an `apps/v1` Deployment `web`, then another renders an `apps/v1beta2` Deployment `web` in the same namespace
- **THEN** one row is returned whose identity carries `apps/v1`, and its two producers carry `apps/v1` and `apps/v1beta2` respectively

#### Scenario: An object with no apiVersion falls in the core group

- **WHEN** a render produces a ConfigMap `x` with `apiVersion` `v1` and then a ConfigMap `x` with no `apiVersion` in the same namespace
- **THEN** one row is returned whose identity carries `v1`, and its two producers carry `v1` and the empty version

#### Scenario: One kind and name in two groups stays distinct

- **WHEN** a render produces a kind `Widget` named `web` under `a.example.com/v1` and under `b.example.com/v1`
- **THEN** no rows are returned

### Requirement: Each returned row carries the identity of the first object rendered with it and every producer of it
Each returned row SHALL carry the identity (Requirement: Two objects share an apply identity when their API group, kind, namespace and name agree and the version part of apiVersion does not distinguish them) of the first object rendered with it (its `apiVersion` verbatim, kind, namespace and name) and every producer of it, a producer being the component and the transformer the kernel recorded on the object together with that object's own `apiVersion`, in the render's pair order.
**Source**: enhancement 0015 ("One registration per module" and "The rendered CR's name is instance-derived: dot-joined `namespace.name`"); the tier placement is ADR-011 item 9.

#### Scenario: Two components render one cluster-scoped object

- **WHEN** two components of one instance each render a `TransformerRegistration` named after the instance
- **THEN** one row is returned carrying that identity with an empty namespace and both producers, component and transformer each

### Requirement: Rows are returned in the order the first object of each identity was rendered
Rows (Requirement: Each returned row carries the identity of the first object rendered with it and every producer of it) SHALL be returned in the order the first object of each identity was rendered, so the result is deterministic for a given render.
**Source**: enhancement 0015 ("One registration per module" and "The rendered CR's name is instance-derived: dot-joined `namespace.name`"); the tier placement is ADR-011 item 9.

#### Scenario: Rows follow the order of the first render of each identity

- **WHEN** a render shares identities
- **THEN** the rows are returned in the order the first object of each identity was rendered, so the result is deterministic for a given render

### Requirement: A render with no shared identity returns no rows
A render with no shared identity SHALL return no rows (Requirement: A function in opm/k8s/object returns every apply identity that two or more rendered objects share).
**Source**: enhancement 0015 ("One registration per module" and "The rendered CR's name is instance-derived: dot-joined `namespace.name`"); the tier placement is ADR-011 item 9.

#### Scenario: A clean render returns nothing

- **WHEN** every rendered object has a distinct identity
- **THEN** no rows are returned

### Requirement: The exported identity type keeps its fields
The exported identity type (Requirement: Each returned row carries the identity of the first object rendered with it and every producer of it) SHALL keep its fields (apiVersion, kind, namespace, name).
**Source**: enhancement 0015 ("One registration per module" and "The rendered CR's name is instance-derived: dot-joined `namespace.name`"); the tier placement is ADR-011 item 9.

#### Scenario: The identity type keeps its four fields

- **WHEN** the exported identity type is read
- **THEN** it keeps its fields: apiVersion, kind, namespace and name

### Requirement: opm/k8s/object provides an error type that aggregates duplicate rows in one wording
`opm/k8s/object` SHALL provide an error type aggregating duplicate rows, whose message names each identity and every producer as component and transformer, one identity per line, so the CLI and the controller refuse with one wording.

#### Scenario: The message names both carrying components

- **WHEN** the error is built from a row with two producers
- **THEN** its message names the identity once and both components with their transformers, and a reader can tell which module components to fix without looking elsewhere

### Requirement: The aggregated error names each producer's apiVersion only when the producers of a row differ
When the producers of a row do not all carry the same `apiVersion`, the message (Requirement: opm/k8s/object provides an error type that aggregates duplicate rows in one wording) SHALL name each producer's `apiVersion` beside it, worded as `<no apiVersion>` for a producer whose object carries none; when they do, the line SHALL carry no per-producer version.

#### Scenario: The message names a version mismatch

- **WHEN** the error is built from a row whose two producers carry `apps/v1` and `apps/v1beta2`
- **THEN** its message names each producer with its own `apiVersion`, so a reader sees that the duplicate is one object under two versions

#### Scenario: A same-version row carries no versions

- **WHEN** the error is built from a row whose producers all carry one `apiVersion`
- **THEN** its message line names the identity and the producers with no per-producer version

#### Scenario: A missing apiVersion is worded

- **WHEN** the error is built from a row whose producers carry `v1` and no `apiVersion`
- **THEN** its message names the second producer `as <no apiVersion>`

### Requirement: The kernel does not return the aggregated duplicate-identity error
The kernel SHALL NOT return this error (Requirement: opm/k8s/object provides an error type that aggregates duplicate rows in one wording): it is raised by the runtime that calls the check, before apply.

#### Scenario: The kernel leaves the aggregated error to the runtime

- **WHEN** the kernel renders
- **THEN** the kernel does not return this error, which is raised by the runtime that calls the check, before apply
