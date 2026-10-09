## REMOVED Requirements

### Requirement: The operator install admission lifts only a proven object's ownership refusal

**Reason**: The enhancement record withdrew the install admission and the install deletions (0012:D8:R6 and 0012:D8:R7, removed on 2026-10-09 in enhancements#109). The cli was the only caller that set the admission input, and it stops in cli#357. The fields `ApplyInput.Admit` and `DeleteInput.Admit` are removed with every rule that existed only for them.

**Migration**: Delete `Admit` from any `ownership.ApplyInput` or `ownership.DeleteInput` literal. The verdicts then decide exactly as they did for `Admit: false`. To let an instance take over an existing object, a user sets the annotation `opmodel.dev/adopt` on it to the instance UUID (0012:D8:R2); to remove an object OPM does not manage, a user deletes it.

## MODIFIED Requirements

### Requirement: The apply verdict refuses terminating, foreign, other-instance and adopted-elsewhere objects

The apply verdict SHALL decide in this order and stop at the first match. When there is no live object, it SHALL apply. When the live object has a deletion timestamp, it SHALL refuse as `terminating`, whether or not the object is in the instance's recorded inventory, and nothing SHALL lift that refusal. When the live object's adopt annotation equals the non-empty instance UUID, it SHALL apply. When the object is in the instance's recorded inventory: with an empty instance UUID it SHALL apply; when its adopt annotation is non-blank, it SHALL refuse as `adopted-elsewhere`; otherwise it SHALL apply, whatever its live UUID label, so an instance whose UUID changed keeps applying its own objects, except an object whose adopt annotation still names the old UUID, which applies again once re-annotated with the new UUID. Outside the inventory: when the live managed-by label is not an OPM runtime's value, it SHALL refuse as `foreign-object`, and the adopt annotation naming this instance is the only input that lifts that refusal; when the adopt annotation is non-blank and equals the live UUID label, it SHALL refuse as `adopted-elsewhere`, since the instance it names completed the hand-over; when the live UUID label is non-empty and differs from the instance UUID, it SHALL refuse as `other-instance`; when the adopt annotation is non-blank, it SHALL refuse as `adopted-elsewhere`; otherwise it SHALL apply. Outside the inventory an empty instance UUID SHALL never match an adopt annotation, and any non-empty live UUID or non-blank adopt annotation SHALL then count as another instance's. Source: 0012:D8:R1/R2/R5, 0012:D8:R8 (enhancements#103), 0012:D1:R7, 0012:D4:R2.

The `ApplyInput.InInventory` doc and the package doc SHALL state that a frontend drops an object refused as `adopted-elsewhere` while it is in the instance's recorded inventory from the inventory it records next, keeps applying the instance's other objects, and never deletes the object for that refusal. The `adopted-elsewhere` message SHALL name the object and the instance its adopt annotation names, and SHALL name the annotation key `opmodel.dev/adopt` with the instance UUID to set it to, unless the instance UUID is empty, as the only way for this instance to take the object back. For an inventoried object it SHALL say that this instance no longer applies the object and drops it from its inventory. Source: 0012:D8:R3, 0012:D8:R8 (enhancements#103).

#### Scenario: A new object is applied

- **WHEN** the apply verdict is asked for an object with no live object
- **THEN** it applies

#### Scenario: A terminating object is refused everywhere

- **WHEN** the live object has a deletion timestamp
- **THEN** the verdict refuses as `terminating` when the object is in the inventory, when it is outside it, when it carries the adopt annotation naming this instance, and when it carries the adopt annotation naming another instance

#### Scenario: An inventoried foreign object is applied

- **WHEN** the object is in the instance's recorded inventory, its live managed-by label is `helm`, and it carries no UUID label and no adopt annotation
- **THEN** the verdict applies

#### Scenario: An inventoried object of this instance is applied

- **WHEN** the object is in the instance's recorded inventory and its live UUID label equals the instance UUID, with no adopt annotation
- **THEN** the verdict applies

#### Scenario: An inventoried object annotated for another instance is refused

- **WHEN** the object is in the instance's recorded inventory, carries this instance's UUID label `u-9`, and its adopt annotation is `u-1`, for instance UUID `u-9`
- **THEN** the verdict refuses as `adopted-elsewhere`
- **AND** the message is `Deployment/web/api was adopted by module instance u-1; this instance no longer applies it and drops it from its inventory; to take it back, annotate it opmodel.dev/adopt=u-9`

#### Scenario: An inventoried object labelled for another instance without an annotation is applied

- **WHEN** the object is in the instance's recorded inventory and its live UUID label is `u-1`, with no adopt annotation, for instance UUID `u-9`
- **THEN** the verdict applies

#### Scenario: An inventoried object the adopter has taken is refused

- **WHEN** the object is in the instance's recorded inventory, its live UUID label is `u-1`, and its adopt annotation is `u-1`, for instance UUID `u-9`
- **THEN** the verdict refuses as `adopted-elsewhere`

#### Scenario: An annotation naming this instance takes an inventoried object back

- **WHEN** the object is in the instance's recorded inventory, its live UUID label is `u-1`, and its adopt annotation is `u-9` with surrounding whitespace, for instance UUID `u-9`
- **THEN** the verdict applies

#### Scenario: An empty instance UUID applies an inventoried object

- **WHEN** the instance UUID is empty and the object in the inventory carries a UUID label and an adopt annotation
- **THEN** the verdict applies

#### Scenario: A foreign object outside the inventory is refused

- **WHEN** the object is outside the inventory and its live managed-by label is missing or is `kustomize`
- **THEN** the verdict refuses as `foreign-object`

#### Scenario: Another instance's object outside the inventory is refused

- **WHEN** the object is outside the inventory, OPM-managed, and its UUID label differs from the instance UUID
- **THEN** the verdict refuses as `other-instance`

#### Scenario: A handed-over object stays adopted-elsewhere after the adopter applies

- **WHEN** the object is outside the inventory, OPM-managed, its live UUID label is `u-1`, and its adopt annotation is `u-1`, for instance UUID `u-9`
- **THEN** the verdict refuses as `adopted-elsewhere`, not `other-instance`
- **AND** the message is `Deployment/web/api was adopted by module instance u-1; this instance does not apply it; to let this instance take it back, annotate it opmodel.dev/adopt=u-9`

#### Scenario: An annotation naming another instance than the label keeps other-instance

- **WHEN** the object is outside the inventory, OPM-managed, its live UUID label is `u-1`, and its adopt annotation is `u-7`, for instance UUID `u-9`
- **THEN** the verdict refuses as `other-instance`

#### Scenario: A dropped object is not taken back on the next apply

- **WHEN** the object is outside the inventory, OPM-managed, carries this instance's UUID label, and its adopt annotation names a different instance
- **THEN** the verdict refuses as `adopted-elsewhere`
- **AND** the message does not say the instance drops it from its inventory

#### Scenario: An OPM object without a UUID label is applied

- **WHEN** the object is outside the inventory, OPM-managed, and carries no UUID label and no adopt annotation
- **THEN** the verdict applies

#### Scenario: An OPM object of this instance outside the inventory is applied

- **WHEN** the object is outside the inventory, OPM-managed, its UUID label equals the instance UUID, and it carries no adopt annotation
- **THEN** the verdict applies

#### Scenario: An empty instance UUID fails closed outside the inventory

- **WHEN** the instance UUID is empty and the object outside the inventory is OPM-managed with a non-empty UUID label
- **THEN** the verdict refuses as `other-instance`

#### Scenario: An empty instance UUID refuses an annotated object outside the inventory

- **WHEN** the instance UUID is empty and the object outside the inventory is OPM-managed, carries no UUID label, and its adopt annotation is `u-1`
- **THEN** the verdict refuses as `adopted-elsewhere`
- **AND** the message is `Deployment/web/api is being adopted by module instance u-1; this instance does not apply it`, with no remedy clause

#### Scenario: The docs state the frontend's part of the hand-over

- **WHEN** the `ApplyInput.InInventory` doc comment and the package doc are read
- **THEN** each says a frontend drops an object refused as `adopted-elsewhere` from the inventory it records next and never deletes it for that refusal
### Requirement: The adopt annotation key is fixed in the label vocabulary

`opm/k8s/labels` SHALL declare the adopt annotation key `opmodel.dev/adopt`. It is an annotation a user sets on an existing live object, and its value names the adopting instance by the value of that instance's `module-instance.opmodel.dev/uuid` label. No function in the library SHALL set it on an object, and the package documentation SHALL say that no OPM runtime writes it. Source: 0012:D8 (the implementing change fixes the key), 0012:D8:R3.

#### Scenario: The key is the fixed literal

- **WHEN** a test reads the adopt annotation constant
- **THEN** it equals `opmodel.dev/adopt`

#### Scenario: No library code sets the adopt annotation

- **WHEN** every non-test Go file under `opm/` is parsed
- **THEN** no file outside `opm/k8s/labels` spells the literal `opmodel.dev/adopt`
- **AND** no file outside `opm/k8s/labels` and `opm/k8s/ownership`, which only reads the key, refers to the adopt annotation constant


## ADDED Requirements

### Requirement: The ownership verdicts take no caller-asserted override

`ownership.ApplyInput` SHALL have exactly the fields `Object`, `Live`, `InInventory` and `InstanceUUID`, and `ownership.DeleteInput` SHALL have exactly the fields `Object`, `Live` and `InstanceUUID`. Neither SHALL carry a field by which a caller asserts, without evidence on the live object, that a refusal or a skip does not apply. On apply, the adopt annotation on the live object naming the applying instance SHALL be the only input that lifts an ownership refusal. On delete, nothing SHALL lift the `not-opm-managed` skip. A frontend that installs the operator is judged as any other instance. Source: 0012:D8:R1/R2/R3; 0012:D8:R6 and 0012:D8:R7 are withdrawn.

#### Scenario: The verdict inputs carry no admission field

- **WHEN** a test lists the fields of `ownership.ApplyInput` and `ownership.DeleteInput`
- **THEN** they are `Object`, `Live`, `InInventory`, `InstanceUUID` and `Object`, `Live`, `InstanceUUID`

#### Scenario: An earlier install manifest's object is refused on apply

- **WHEN** the apply verdict is asked for `Namespace/opm-operator-system`, live with managed-by `kustomize` and no UUID label, outside the inventory
- **THEN** it refuses as `foreign-object`
- **AND** the message names `opmodel.dev/adopt` with the instance UUID

#### Scenario: An earlier install manifest's Deployment is not deleted

- **WHEN** the delete verdict is asked for a live `apps` Deployment with managed-by `kustomize` and no UUID label
- **THEN** it skips as `not-opm-managed`
