## MODIFIED Requirements

### Requirement: Optional labels and annotations are filled into instance metadata

When `InstanceInput.Labels` is non-empty, `synth.Instance` SHALL fill `metadata.labels` with those entries. When `InstanceInput.Annotations` is non-empty, `synth.Instance` SHALL fill `metadata.annotations` with those entries. The schema's label stamping (Requirement: Derived fields come from schema unification) unifies with caller-supplied labels — caller labels MUST NOT be allowed to remove schema-stamped labels. The staged instance source SHALL be deterministic: for identical inputs, the bytes synthesis emits for `metadata.labels` and `metadata.annotations` SHALL be identical across calls and across processes, independent of the order the caller's maps iterate in. Entries SHALL be emitted in ascending key order.

#### Scenario: Caller labels merged with schema-stamped labels

- **WHEN** `synth.Instance` is called with `Labels == {"env": "prod"}`
- **THEN** the returned CUE value's `metadata.labels` contains the `env: prod` entry
- **AND** still contains `module-instance.opmodel.dev/name` and `module-instance.opmodel.dev/uuid`

#### Scenario: Annotations are passed through unchanged

- **WHEN** `synth.Instance` is called with `Annotations == {"opmodel.dev/owner": "team-x"}`
- **THEN** the returned CUE value's `metadata.annotations` contains that entry

#### Scenario: Label and annotation order is stable

- **WHEN** `synth.Instance` is called twice with `Labels` and `Annotations` each holding the same four or more entries, inserted into the maps in different orders
- **THEN** the `instance.cue` bytes on the two returned staged trees are byte-identical
- **AND** the entries appear in ascending key order
