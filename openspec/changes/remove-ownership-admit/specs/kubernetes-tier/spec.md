## MODIFIED Requirements

### Requirement: The adopt annotation key is fixed in the label vocabulary

`opm/k8s/labels` SHALL declare the adopt annotation key `opmodel.dev/adopt`. It is an annotation a user sets on an existing live object, and its value names the adopting instance by the value of that instance's `module-instance.opmodel.dev/uuid` label. No function in the library SHALL set it on an object, and the package documentation SHALL say that no OPM runtime writes it. Source: 0012:D8 (the implementing change fixes the key).

#### Scenario: The key is the fixed literal

- **WHEN** a test reads the adopt annotation constant
- **THEN** it equals `opmodel.dev/adopt`

#### Scenario: No library code sets the adopt annotation

- **WHEN** every non-test Go file under `opm/` is parsed
- **THEN** no file outside `opm/k8s/labels` spells the literal `opmodel.dev/adopt`
- **AND** no file outside `opm/k8s/labels` and `opm/k8s/ownership`, which only reads the key, refers to the adopt annotation constant

### Requirement: The apply verdict refuses terminating, foreign and other-instance objects

The apply verdict SHALL decide in this order and stop at the first match. When there is no live object, it SHALL apply. When the live object has a deletion timestamp, it SHALL refuse as `terminating`, whether or not the object is in the instance's recorded inventory, and nothing SHALL lift that refusal. When the object is in the instance's recorded inventory, it SHALL apply. When the live object's adopt annotation equals the non-empty instance UUID, it SHALL apply. When the live managed-by label is not an OPM runtime's value, it SHALL refuse as `foreign-object`. When the live UUID label is non-empty and differs from the instance UUID, it SHALL refuse as `other-instance`. Otherwise it SHALL apply. An empty instance UUID SHALL never match an adopt annotation, and any non-empty live UUID SHALL then count as another instance's. Source: 0012:D8:R1/R2/R5, 0012:D1:R7, 0012:D4:R2.

#### Scenario: A new object is applied

- **WHEN** the apply verdict is asked for an object with no live object
- **THEN** it applies

#### Scenario: A terminating object is refused everywhere

- **WHEN** the live object has a deletion timestamp
- **THEN** the verdict refuses as `terminating` when the object is in the inventory, when it is outside it, and when it carries the adopt annotation naming this instance

#### Scenario: An inventoried object is not judged for ownership

- **WHEN** the object is in the instance's recorded inventory and its live managed-by label is `helm`
- **THEN** the verdict applies

#### Scenario: An inventoried object annotated for another instance still applies

- **WHEN** the object is in the instance's recorded inventory, carries this instance's UUID label, and its adopt annotation names a different instance
- **THEN** the verdict applies

#### Scenario: A foreign object outside the inventory is refused

- **WHEN** the object is outside the inventory and its live managed-by label is missing or is `kustomize`
- **THEN** the verdict refuses as `foreign-object`

#### Scenario: Another instance's object outside the inventory is refused

- **WHEN** the object is outside the inventory, OPM-managed, and its UUID label differs from the instance UUID
- **THEN** the verdict refuses as `other-instance`

#### Scenario: An OPM object without a UUID label is applied

- **WHEN** the object is outside the inventory, OPM-managed, and carries no UUID label
- **THEN** the verdict applies

#### Scenario: An OPM object of this instance outside the inventory is applied

- **WHEN** the object is outside the inventory, OPM-managed, and its UUID label equals the instance UUID
- **THEN** the verdict applies

#### Scenario: An empty instance UUID fails closed

- **WHEN** the instance UUID is empty and the object outside the inventory is OPM-managed with a non-empty UUID label
- **THEN** the verdict refuses as `other-instance`

## REMOVED Requirements

### Requirement: The operator install admission lifts only a proven object's ownership refusal

**Reason**: The install admission served only the migration of a manifest-installed controller to the module-installed one. The owner removed all legacy-migration code on 2026-10-05 (nobody runs OPM yet), and the enhancement withdraws 0012:D8:R6/R7. `ApplyInput.Admit`, `DeleteInput.Admit` and `installDeletable` are removed.

**Migration**: A caller deletes the `Admit` field from its `ApplyInput` and `DeleteInput` literals. The verdicts then decide as they did for `Admit: false`; the adopt annotation is the only override.
