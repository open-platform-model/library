## ADDED Requirements

### Requirement: Inventory entries are plain values built from objects

The library SHALL provide, in `opm/k8s/inventory`, an inventory entry with exactly the fields group, kind, namespace, name, version and component, and a constructor that builds one from a `*unstructured.Unstructured`. The constructor SHALL read group, version and kind, namespace and name from the object, and the component from its `component.opmodel.dev/name` label, which gives an empty component when the label is absent. The entry type SHALL carry no struct tags, so that no serialisation the library chooses can become the frontends' wire shape. Each frontend maps entries to its own CRD or record fields. Source: 0012:D1, 0012:D7.

#### Scenario: An entry reads the object's identity and component

- **WHEN** an entry is built from a `Deployment` in group `apps`, version `v1`, namespace `team`, named `web` and labelled `component.opmodel.dev/name: web`
- **THEN** the entry's group is `apps`, kind `Deployment`, namespace `team`, name `web`, version `v1` and component `web`

#### Scenario: A core-group, cluster-scoped object without the label

- **WHEN** an entry is built from a `v1` `Namespace` named `team` with no `component.opmodel.dev/name` label
- **THEN** the entry's group, namespace and component are empty, its kind is `Namespace`, its name `team` and its version `v1`

#### Scenario: The entry type carries no tags

- **WHEN** a test reflects over the entry type's fields
- **THEN** there are exactly six string fields and none carries a struct tag

### Requirement: The stale set is component-blind

The library SHALL provide, in `opm/k8s/inventory`, one stale-set function. It SHALL return every entry of the previous inventory for which no entry of the current inventory has the same group, kind, namespace and name, whatever the component and the version of either entry. The result SHALL keep the previous inventory's order, SHALL be a non-nil empty list when nothing is stale, and SHALL leave both inputs unchanged. The same identity comparison SHALL be exported on its own. No component-aware comparison and no separate component-rename filter SHALL exist in the package. Source: 0012:D7:R1, 0012:D1:R1.

#### Scenario: A component rename is not stale

- **WHEN** the previous inventory holds `apps/Deployment team/web` with component `api` and the current inventory holds the same group, kind, namespace and name with component `web`
- **THEN** the stale set is empty

#### Scenario: An API version change is not stale

- **WHEN** the previous inventory holds an `autoscaling` `HorizontalPodAutoscaler` at version `v2beta2` and the current inventory holds the same object at version `v2`
- **THEN** the stale set is empty

#### Scenario: Removed objects are stale in previous order

- **WHEN** the previous inventory holds objects A, B, C and D in that order and the current inventory holds only B
- **THEN** the stale set is A, C, D in that order

#### Scenario: Nothing stale is an empty list, not nil

- **WHEN** the previous inventory is empty, or every previous object is in the current inventory
- **THEN** the stale set is a non-nil list of length zero

### Requirement: The inventory digest hashes a canonical field encoding

The library SHALL provide, in `opm/k8s/inventory`, one inventory digest of the form `sha256:` followed by 64 lowercase hex digits. It SHALL hash the tag line `opm-inventory-v1` with a newline, followed by every entry in order of group, kind, namespace, name, component and version (byte-wise). Each entry SHALL be written as its group, kind, namespace, name, version and component, in that order, each preceded by its byte length as an 8-byte big-endian unsigned integer. The digest SHALL depend only on the entries' field values: not on their input order and not on any JSON or other serialisation of an entry. Two inventories that differ in their entries, or in any field of an entry, SHALL produce different digests. The empty inventory SHALL hash the tag line alone. The function SHALL leave its input unchanged. A change to this encoding SHALL use a new tag line. Source: 0012:D7:R2/R3, 0012:D1:R1.

#### Scenario: The encoding is the one defined

- **WHEN** a test writes out by hand the tag line and the length-prefixed fields of three entries (a core-group cluster-scoped entry with empty group and namespace, a namespaced `apps` entry, and an entry with an empty component), sorted as defined
- **THEN** the digest of those three entries, given in another order, equals `sha256:` and the hex SHA-256 of the hand-written bytes, and equals the committed golden value

#### Scenario: Input order does not matter

- **WHEN** the same entries are digested in two different orders
- **THEN** the two digests are equal

#### Scenario: Every field counts

- **WHEN** one entry of an inventory changes in exactly one of its six fields, or an entry is added or removed
- **THEN** the digest differs from the original inventory's

#### Scenario: Empty and nil inventories agree

- **WHEN** an empty inventory and a nil inventory are digested
- **THEN** both digests equal `sha256:` and the hex SHA-256 of `opm-inventory-v1` followed by a newline

### Requirement: The render digest ignores only the managed-by label value

The library SHALL provide, in `opm/k8s/inventory`, one render digest over the objects that `opm/k8s/object`'s export returns, of the form `sha256:` followed by 64 lowercase hex digits. It SHALL read only each object's exported JSON, never a CUE value, so a caller may release its build before digesting. For each object it SHALL decode the JSON keeping every number's literal, replace the value of the label `app.kubernetes.io/managed-by` with the empty string when `metadata.labels` holds that key, and encode the result as JSON with object keys sorted and no HTML escaping, ending with a newline. It SHALL hash the tag line `opm-render-v1` with a newline, followed by the encoded objects in order of group (from `apiVersion`), kind, namespace and name, with the encoded bytes as the final tie-break. For one render, the cli (`opm-cli`) and the operator (`opm-controller`) SHALL compute the same digest. Two object sets that differ in any other label, in whether the managed-by label is present, or in any other content of any object SHALL produce different digests. The empty set SHALL hash the tag line alone. An object whose JSON does not decode to a single JSON object SHALL fail with an error naming its position. The function SHALL leave its input unchanged. A change to this encoding SHALL use a new tag line. Source: 0012:D6:R2/R3, 0012:D1:R1.

#### Scenario: The two runtimes digest one render equally

- **WHEN** the kernel renders one instance with runtime name `opm-cli` and again with `opm-controller`
- **THEN** each rendered object's managed-by label value is that render's runtime name, and the two renders are equal object by object once that one value is blanked
- **AND** two exported object sets that differ only in the managed-by label value have equal render digests

#### Scenario: Any other label counts

- **WHEN** two exported object sets differ only in the value of one label other than `app.kubernetes.io/managed-by`
- **THEN** their render digests differ

#### Scenario: Adding or removing the managed-by label counts

- **WHEN** one object set carries the managed-by label on an object and the other does not, and nothing else differs
- **THEN** their render digests differ

#### Scenario: Large integers are not rounded

- **WHEN** two exported object sets differ only in one integer field, `9007199254740993` in one and `9007199254740992` in the other
- **THEN** their render digests differ

#### Scenario: The encoding is the one defined

- **WHEN** a test writes out by hand the tag line and the sorted-key JSON of a Deployment carrying a managed-by label, a core-group Service and a cluster-scoped Namespace, sorted as defined and with the managed-by value blanked
- **THEN** the render digest of those three objects, given in another order, equals `sha256:` and the hex SHA-256 of the hand-written bytes, and equals the committed golden value

#### Scenario: The input is not changed

- **WHEN** a list of exported objects is digested
- **THEN** each object's JSON bytes and decoded object, including its managed-by label value, are unchanged afterwards

#### Scenario: A malformed object fails with its position

- **WHEN** the second object in the list carries JSON that is a list, or that is not valid JSON
- **THEN** the render digest fails with an error naming position 1
