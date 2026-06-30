# inventory Specification

## Purpose

The `opm/inventory` package provides a runtime-neutral, pure in-memory representation of rendered resources and the canonical relations over them — identity, content digest, stale-set computation, component-rename prune safety, and pre-apply collision detection — so that every OPM frontend (operator, CLI, future runtimes) shares one deterministic, side-effect-free inventory contract.

## Requirements

### Requirement: Runtime-Neutral Inventory Entry Type

The library SHALL provide an `opm/inventory` package exposing a value type `InventoryEntry` with exactly the fields `Group`, `Kind`, `Namespace`, `Name`, `Version`, and `Component` (all strings). The package and this type MUST NOT import `sigs.k8s.io/controller-runtime`, any `github.com/fluxcd/*` package, or `k8s.io/apiextensions-apiserver`. Its only non-standard-library dependency MAY be `k8s.io/apimachinery` (for `unstructured` access and object-identity primitives).

This type is the in-memory neutral representation that downstream consumers (operator, CLI) map their own entry shapes to and from; it is distinct from any CRD serialization type.

#### Scenario: Package imports carry no controller runtime

- **WHEN** the `opm/inventory` package and its transitive imports are inspected
- **THEN** none of `sigs.k8s.io/controller-runtime`, `github.com/fluxcd/*`, or `k8s.io/apiextensions-apiserver` appears in the import graph
- **AND** the package compiles and its tests pass against `k8s.io/apimachinery` and the standard library only

#### Scenario: Entry constructed from a rendered resource

- **WHEN** a caller invokes `NewEntryFromResource(u)` with an `*unstructured.Unstructured` carrying group, kind, namespace, name, version, and an OPM component label
- **THEN** the returned `InventoryEntry` is populated from the object's GroupVersionKind, namespace, name, and component attribution
- **AND** constructing an entry performs no I/O and reads only the supplied object

### Requirement: Entry Identity Relations

The package SHALL expose two identity relations over `InventoryEntry`:

- `IdentityEqual(a, b)` — full identity, comparing Kubernetes object identity **and** `Component`.
- `K8sIdentityEqual(a, b)` — Kubernetes object identity only (`Group`, `Kind`, `Namespace`, `Name`), ignoring `Component`.

Both relations MUST be deterministic and side-effect free.

#### Scenario: Same object, different component

- **WHEN** two entries share Group, Kind, Namespace, and Name but differ in `Component`
- **THEN** `K8sIdentityEqual` returns true
- **AND** `IdentityEqual` returns false

#### Scenario: Different object

- **WHEN** two entries differ in any of Group, Kind, Namespace, or Name
- **THEN** both `K8sIdentityEqual` and `IdentityEqual` return false

### Requirement: Deterministic Content Digest

The package SHALL expose `ComputeDigest(entries)` returning a stable digest string over the set of entries. The digest MUST be order-independent (computed over entries sorted by their identity tuple) and MUST be identical for identical entry sets, so that two actors computing a digest over the same rendered inventory obtain the same value.

#### Scenario: Order independence

- **WHEN** `ComputeDigest` is called on the same set of entries supplied in two different orders
- **THEN** both calls return the identical digest string

#### Scenario: Content sensitivity

- **WHEN** two entry sets differ in any entry's identity or version
- **THEN** their digests differ

### Requirement: Canonical Stale-Set Computation

The package SHALL expose `ComputeStaleSet(previous, current)` returning the entries present in `previous` but absent from `current`, where presence is determined by **`K8sIdentityEqual`** (Kubernetes object identity, component-agnostic). This relation is the single canonical stale-set base for every OPM frontend; component moves MUST NOT be treated as stale by this base computation and are handled by the component-rename safety requirement below.

#### Scenario: Removed resource is stale

- **WHEN** a previous entry has no Kubernetes-identity match in the current set
- **THEN** `ComputeStaleSet` includes it in the returned stale set

#### Scenario: Retained resource is not stale

- **WHEN** a previous entry has a Kubernetes-identity match in the current set
- **THEN** `ComputeStaleSet` excludes it from the returned stale set, regardless of any `Component` difference

### Requirement: Component-Rename Prune Safety

The package SHALL expose `ApplyComponentRenameSafetyCheck(stale, current)` returning the stale set with any entry removed when that entry shares Kubernetes object identity (`K8sIdentityEqual`) with a current entry under a different `Component`. The function MUST be pure (no I/O, no logging). The effect is that a resource which moves between components is neither pruned nor recreated; it is left in place to be re-owned.

#### Scenario: Component move is not pruned

- **WHEN** a stale entry shares Group, Kind, Namespace, and Name with a current entry but has a different `Component`
- **THEN** `ApplyComponentRenameSafetyCheck` removes that entry from the stale set

#### Scenario: Genuinely removed resource still pruned

- **WHEN** a stale entry has no Kubernetes-identity match anywhere in the current set
- **THEN** `ApplyComponentRenameSafetyCheck` leaves that entry in the stale set

### Requirement: Pure Pre-Apply Collision Predicate

The package SHALL expose a pure predicate that decides whether applying a candidate entry would collide with an existing foreign-owned cluster object, given the already-observed state of the live object at that identity supplied by the caller. The observed-state input MUST be a neutral value (carrying at least: whether the object exists, whether it is being deleted, and its managed-by attribution) populated by the caller from its own cluster read. The predicate MUST NOT perform any cluster read, apply, or logging itself; fetching live state and reacting to the verdict are caller responsibilities.

#### Scenario: Collision with a foreign-owned object

- **WHEN** the predicate is given a candidate entry and observed state indicating a live object exists at that identity, is not being deleted, and is managed by a different owner
- **THEN** the predicate reports a collision

#### Scenario: No live object

- **WHEN** the predicate is given observed state indicating no live object exists at that identity
- **THEN** the predicate reports no collision

#### Scenario: Object owned by this release

- **WHEN** the predicate is given observed state indicating the live object is attributed to this release's own manager
- **THEN** the predicate reports no collision

#### Scenario: Predicate performs no I/O

- **WHEN** the predicate is invoked
- **THEN** it reads only its arguments and performs no cluster read, apply, delete, or logging
