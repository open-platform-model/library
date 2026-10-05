## MODIFIED Requirements

### Requirement: The ownership verdicts are pure

The library SHALL provide `opm/k8s/ownership` with a per-object delete verdict and a per-object apply verdict, as pure functions of explicit inputs. Each input SHALL name the object by group, kind, namespace and name and SHALL carry the live object as the caller read it, absent when the read found nothing, together with the judging instance's UUID. The package SHALL perform no cluster read or write, read no clock or environment, and log nothing. It SHALL NOT modify the live object it is given. Each reason SHALL be a string equal to the contract literal for it. A frontend reads the live object with its own client. Source: 0012:D4, ADR-008, ADR-011.

#### Scenario: The live object is not modified

- **WHEN** either verdict is computed for a live object
- **THEN** the object is deeply equal to a copy taken before the call

#### Scenario: The package imports no clock, environment or logger

- **WHEN** the direct imports of `opm/k8s/ownership` are listed
- **THEN** none of them is `os`, `time`, `log` or `log/slog`

#### Scenario: Reasons are the contract's literals

- **WHEN** a test reads the skip and refusal reason constants
- **THEN** the skip reasons are `safety-excluded`, `already-absent`, `not-opm-managed`, `owner-mismatch` and `adopted-elsewhere`
- **AND** the refusal reasons are `terminating`, `foreign-object`, `other-instance` and `adopted-elsewhere`

### Requirement: The delete verdict skips with a reason or proceeds with the judged object's identity

The delete verdict SHALL decide in this order and stop at the first match: skip as `safety-excluded` when the kind is safety-excluded, whatever the live object; skip as `already-absent` when there is no live object; skip as `not-opm-managed` when the live managed-by label is not an OPM runtime's value; skip as `owner-mismatch` when the live UUID label and the instance's UUID are both non-empty and differ; skip as `adopted-elsewhere` when the live adopt annotation, compared with surrounding whitespace trimmed, and the instance's UUID are both non-empty and differ; otherwise proceed. An empty UUID on either side SHALL pass the owner comparison, and an empty instance UUID or a blank adopt annotation SHALL pass the adoption comparison. An adopt annotation naming this instance SHALL change nothing. An object being deleted SHALL NOT be skipped for that reason. A proceed verdict SHALL carry the UID and the resourceVersion of the live object it judged. Its DELETE precondition SHALL name that UID and SHALL NOT name a resourceVersion. A caller MAY add the carried resourceVersion itself. A skip verdict, and a proceed verdict whose judged UID is empty, SHALL yield no precondition at all, never a precondition on an empty UID. A skip verdict SHALL carry a message naming the object and the reason. So a prune or an instance deletion never deletes an object the user is handing to another instance, before or after that instance has applied it. Source: 0012:D1:R3/R4, 0012:D4:R1, 0012:D8:R8 (enhancements#103).

#### Scenario: A protected kind is skipped without a live read

- **WHEN** the delete verdict is asked for a core Namespace with no live object
- **THEN** it skips as `safety-excluded`

#### Scenario: A gone object is skipped as already absent

- **WHEN** the delete verdict is asked for a Deployment with no live object
- **THEN** it skips as `already-absent`

#### Scenario: A foreign object is skipped

- **WHEN** the live Deployment's managed-by label is `helm`
- **THEN** the verdict skips as `not-opm-managed` and its message names the Deployment

#### Scenario: Another instance's object is skipped

- **WHEN** the live Deployment is OPM-managed and its UUID label differs from the non-empty instance UUID
- **THEN** the verdict skips as `owner-mismatch`

#### Scenario: An empty UUID on either side passes

- **WHEN** the live Deployment is OPM-managed, carries no adopt annotation, and either its UUID label or the instance UUID is empty
- **THEN** the verdict proceeds

#### Scenario: Proceed carries the precondition for the judged object

- **WHEN** the verdict proceeds for a live object with UID `u-1` and resourceVersion `42`
- **THEN** it carries UID `u-1` and resourceVersion `42`
- **AND** its DELETE precondition names UID `u-1` and no resourceVersion

#### Scenario: A skip verdict yields no precondition

- **WHEN** the delete verdict skips as `already-absent`
- **THEN** it yields no DELETE precondition

#### Scenario: A live object without a UID yields no precondition

- **WHEN** the verdict proceeds for a live object that carries no UID
- **THEN** it yields no DELETE precondition

#### Scenario: An object being deleted proceeds

- **WHEN** the live OPM-managed object of this instance has a deletion timestamp
- **THEN** the verdict proceeds

#### Scenario: An object annotated for another instance is left in place

- **WHEN** the live Deployment is OPM-managed, carries this instance's UUID label `u-9`, and its adopt annotation is `u-2`, for instance UUID `u-9`
- **THEN** the verdict skips as `adopted-elsewhere`
- **AND** its message names the Deployment and module instance `u-2`
- **AND** it yields no DELETE precondition

#### Scenario: An annotation naming this instance changes nothing on delete

- **WHEN** the live Deployment is OPM-managed, carries UUID label `u-9`, and its adopt annotation is `u-9`, for instance UUID `u-9`
- **THEN** the verdict proceeds

#### Scenario: An empty instance UUID passes the adoption comparison

- **WHEN** the live Deployment is OPM-managed and its adopt annotation is `u-2`, for an empty instance UUID
- **THEN** the verdict proceeds

#### Scenario: A prune of a dropped object leaves it for the adopting instance

- **WHEN** a deletion plan built from a stale set holds a Deployment whose live object carries the plan owner's UUID label and an adopt annotation naming another instance
- **THEN** advancing the plan past its read records the step as skipped with `adopted-elsewhere` and names no delete action for it

### Requirement: The adopt annotation is the only override and the refusal names it

An adopt annotation whose value equals the instance UUID SHALL lift the `foreign-object`, `other-instance` and `adopted-elsewhere` refusals. The annotation value SHALL be compared with surrounding whitespace trimmed, and a value that is empty or only whitespace SHALL count as no annotation. An adopt annotation with any other value SHALL lift nothing, and the refusal message SHALL say that the annotation names another instance. The refusal message for `foreign-object` and `other-instance` SHALL name the object, the annotation key `opmodel.dev/adopt` and the instance UUID to set it to, unless the instance UUID is empty. The `other-instance` message SHALL ask the user to remove the object from the instance that owns it before annotating it, so that instance stops rendering it and is not refused on every apply after the hand-over. No message SHALL name any other way past a refusal, such as a command-line flag. No message SHALL carry an enhancement reference. Source: 0012:D8:R2/R3.

#### Scenario: Adoption lifts a foreign refusal

- **WHEN** the object outside the inventory is not OPM-managed and its adopt annotation equals the instance UUID
- **THEN** the verdict applies

#### Scenario: Adoption lifts an other-instance refusal

- **WHEN** the object outside the inventory carries another instance's UUID label and its adopt annotation equals the instance UUID
- **THEN** the verdict applies

#### Scenario: An annotation naming another instance lifts nothing

- **WHEN** the object outside the inventory is not OPM-managed and its adopt annotation names a different UUID
- **THEN** the verdict refuses as `foreign-object`
- **AND** the message says the annotation names another instance

#### Scenario: Surrounding whitespace in the annotation value is ignored

- **WHEN** the object outside the inventory is not OPM-managed and its adopt annotation is the instance UUID with surrounding whitespace
- **THEN** the verdict applies

#### Scenario: A blank annotation counts as none

- **WHEN** the object outside the inventory is not OPM-managed and its adopt annotation is empty or only whitespace
- **THEN** the verdict refuses as `foreign-object`
- **AND** the message does not say the annotation names another instance

#### Scenario: The other-instance remedy moves the object only after its owner lets go

- **WHEN** the verdict refuses `Deployment/web/api` as `other-instance` for instance UUID `u-9`, the live UUID label being `u-1`
- **THEN** the message asks to remove it from module instance `u-1` before annotating it `opmodel.dev/adopt=u-9`

#### Scenario: The refusal tells the user how to adopt

- **WHEN** the verdict refuses `Deployment/web/api` as `foreign-object` for instance UUID `u-9`
- **THEN** the message contains `Deployment/web/api`, `opmodel.dev/adopt` and `u-9`
- **AND** it contains no `--` flag and no `force`

## REMOVED Requirements

### Requirement: The apply verdict refuses terminating, foreign and other-instance objects

**Reason**: Its order applies every object in the instance's recorded inventory without judging ownership, and its scenario "An inventoried object annotated for another instance still applies" is the hand-over fight of enhancements#103. The owner decided that an instance refuses an inventoried object another instance adopted, which reverses that scenario.

**Migration**: Replaced by "The apply verdict refuses terminating, foreign, other-instance and adopted-elsewhere objects", which keeps every other scenario's outcome. No frontend calls the apply verdict at the time of this change.

## ADDED Requirements

### Requirement: The apply verdict refuses terminating, foreign, other-instance and adopted-elsewhere objects

The apply verdict SHALL decide in this order and stop at the first match. When there is no live object, it SHALL apply. When the live object has a deletion timestamp, it SHALL refuse as `terminating`, whether or not the object is in the instance's recorded inventory, and nothing SHALL lift that refusal. When the live object's adopt annotation equals the non-empty instance UUID, it SHALL apply. When the object is in the instance's recorded inventory: with an empty instance UUID it SHALL apply; when its adopt annotation is non-blank, or its live UUID label is non-empty and differs from the instance UUID, it SHALL refuse as `adopted-elsewhere`; otherwise it SHALL apply. Outside the inventory: when the live managed-by label is not an OPM runtime's value, it SHALL refuse as `foreign-object`, unless the operator install admission lifts it; when the live UUID label is non-empty and differs from the instance UUID, it SHALL refuse as `other-instance`; when the adopt annotation is non-blank, it SHALL refuse as `adopted-elsewhere`; otherwise it SHALL apply. Outside the inventory an empty instance UUID SHALL never match an adopt annotation, and any non-empty live UUID or non-blank adopt annotation SHALL then count as another instance's. Source: 0012:D8:R1/R2/R5, 0012:D8:R8 (enhancements#103), 0012:D1:R7, 0012:D4:R2.

A frontend SHALL drop an object the verdict refuses as `adopted-elsewhere` while it is in the instance's recorded inventory from the inventory it records next, SHALL keep applying the instance's other objects, and SHALL NOT delete the object for that refusal. The `adopted-elsewhere` message SHALL name the object and the instance its adopt annotation names, or its UUID label names when the annotation is blank, and SHALL name the annotation key `opmodel.dev/adopt` with the instance UUID to set it to, unless the instance UUID is empty, as the only way for this instance to take the object back. For an inventoried object it SHALL say that this instance no longer applies the object and drops it from its inventory. Source: 0012:D8:R3, 0012:D8:R8 (enhancements#103).

#### Scenario: A new object is applied

- **WHEN** the apply verdict is asked for an object with no live object
- **THEN** it applies

#### Scenario: A terminating object is refused everywhere

- **WHEN** the live object has a deletion timestamp
- **THEN** the verdict refuses as `terminating` when the object is in the inventory, when it is outside it, when it carries the adopt annotation naming this instance, when it carries the adopt annotation naming another instance, and when it is admitted

#### Scenario: An inventoried foreign object is applied

- **WHEN** the object is in the instance's recorded inventory, its live managed-by label is `helm`, and it carries no UUID label and no adopt annotation
- **THEN** the verdict applies

#### Scenario: An inventoried object of this instance is applied

- **WHEN** the object is in the instance's recorded inventory and its live UUID label equals the instance UUID, with no adopt annotation
- **THEN** the verdict applies

#### Scenario: An inventoried object annotated for another instance is refused

- **WHEN** the object is in the instance's recorded inventory, carries this instance's UUID label `u-9`, and its adopt annotation is `u-2`, for instance UUID `u-9`
- **THEN** the verdict refuses as `adopted-elsewhere`
- **AND** the message is `Deployment/web/api was adopted by module instance u-2; this instance no longer applies it and drops it from its inventory; to take it back, annotate it opmodel.dev/adopt=u-9`

#### Scenario: An inventoried object another instance has taken is refused

- **WHEN** the object is in the instance's recorded inventory and its live UUID label is `u-2`, with no adopt annotation, for instance UUID `u-9`
- **THEN** the verdict refuses as `adopted-elsewhere` and the message names module instance `u-2`

#### Scenario: An annotation naming this instance takes an inventoried object back

- **WHEN** the object is in the instance's recorded inventory, its live UUID label is `u-2`, and its adopt annotation is `u-9` with surrounding whitespace, for instance UUID `u-9`
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
