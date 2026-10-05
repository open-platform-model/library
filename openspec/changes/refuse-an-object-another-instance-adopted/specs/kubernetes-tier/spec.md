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

The delete verdict SHALL decide in this order and stop at the first match: skip as `safety-excluded` when the kind is safety-excluded, whatever the live object; skip as `already-absent` when there is no live object; skip as `not-opm-managed` when the live managed-by label is not an OPM runtime's value; skip as `owner-mismatch` when the live UUID label and the instance's UUID are both non-empty and differ; skip as `adopted-elsewhere` when the live adopt annotation, compared with surrounding whitespace trimmed, is non-blank and differs from the instance's UUID; otherwise proceed. An empty UUID on either side SHALL pass the owner comparison. A blank adopt annotation SHALL pass the adoption comparison, and with an empty instance UUID any non-blank adopt annotation SHALL count as another instance's, since an empty UUID can never be the one an annotation names. An adopt annotation naming this instance SHALL change nothing. An object being deleted SHALL NOT be skipped for that reason. A proceed verdict SHALL carry the UID and the resourceVersion of the live object it judged. Its DELETE precondition SHALL name that UID and SHALL NOT name a resourceVersion. A caller MAY add the carried resourceVersion itself. A skip verdict, and a proceed verdict whose judged UID is empty, SHALL yield no precondition at all, never a precondition on an empty UID. A skip verdict SHALL carry a message naming the object and the reason. So a prune or an instance deletion never deletes an object the user is handing to another instance, before or after that instance has applied it. Source: 0012:D1:R3/R4, 0012:D4:R1, 0012:D8:R8 (enhancements#103).

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

- **WHEN** the live Deployment is OPM-managed, carries this instance's UUID label `u-9`, and its adopt annotation is `u-1`, for instance UUID `u-9`
- **THEN** the verdict skips as `adopted-elsewhere`
- **AND** its message is `Deployment/web/api is being adopted by module instance u-1, not this one; left in place`
- **AND** it yields no DELETE precondition

#### Scenario: An annotation naming this instance changes nothing on delete

- **WHEN** the live Deployment is OPM-managed, carries UUID label `u-9`, and its adopt annotation is `u-9`, for instance UUID `u-9`
- **THEN** the verdict proceeds

#### Scenario: An empty instance UUID skips an annotated object

- **WHEN** the live Deployment is OPM-managed and its adopt annotation is `u-1`, for an empty instance UUID
- **THEN** the verdict skips as `adopted-elsewhere`

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

### Requirement: The operator install admission lifts only a proven object's ownership refusal

Both verdicts SHALL take an admission input that a caller sets only for an object it has proven came from an earlier operator release's install manifest. On apply, admission SHALL lift `foreign-object` only, and only when the live object carries no UUID label or carries the instance UUID. On delete, admission SHALL lift `not-opm-managed` only, only when the live object carries no UUID label at all, and only for an `apps` `Deployment`, a `rbac.authorization.k8s.io` `RoleBinding` or a `rbac.authorization.k8s.io` `ClusterRoleBinding`, the kinds install may delete outside the inventory. The UUID label is the only identity admission compares. Admission SHALL NOT lift `terminating`, `other-instance`, `adopted-elsewhere`, `owner-mismatch`, `safety-excluded` or `already-absent`, on either side. The library does not check the proof. Source: 0012:D8:R6/R7, 0012:D8:R8 (enhancements#103), 0012:D4:R1.

#### Scenario: A proven earlier-manifest object is admitted on apply

- **WHEN** the operator install asks for the earlier `Namespace/opm-operator-system`, live with managed-by `kustomize` and no UUID label, admitted and outside the inventory
- **THEN** the verdict applies

#### Scenario: An admitted object carrying another identity is refused

- **WHEN** an admitted object outside the inventory is not OPM-managed but carries a UUID label that differs from the instance UUID
- **THEN** the apply verdict refuses as `foreign-object`

#### Scenario: Admission never lifts the terminating refusal

- **WHEN** an admitted proven object has a deletion timestamp
- **THEN** the apply verdict refuses as `terminating`

#### Scenario: Admission never lifts the adopted-elsewhere refusal

- **WHEN** an admitted object outside the inventory, live with managed-by `kustomize` and no UUID label, carries an adopt annotation naming another instance
- **THEN** the apply verdict refuses as `adopted-elsewhere`

#### Scenario: A proven earlier Deployment may be deleted

- **WHEN** the delete verdict is asked for an admitted live Deployment with managed-by `kustomize` and no UUID label
- **THEN** it proceeds and carries the Deployment's UID

#### Scenario: An admitted object carrying another identity is not deleted

- **WHEN** the delete verdict is asked for an admitted live Deployment with managed-by `kustomize` and a UUID label that differs from the non-empty instance UUID
- **THEN** it skips as `not-opm-managed`

#### Scenario: An admitted object carrying this instance's UUID is not deleted

- **WHEN** the delete verdict is asked for an admitted live ClusterRoleBinding with no OPM managed-by label and a UUID label equal to the instance UUID
- **THEN** it skips as `not-opm-managed`

#### Scenario: Admission deletes only the install's deletable kinds

- **WHEN** the delete verdict is asked for an admitted live ConfigMap, or an admitted live custom resource, with managed-by `kustomize` and no UUID label
- **THEN** it skips as `not-opm-managed`

#### Scenario: Admission never deletes a protected kind

- **WHEN** the delete verdict is asked for an admitted CustomResourceDefinition of `apiextensions.k8s.io`
- **THEN** it skips as `safety-excluded`

#### Scenario: Admission never deletes an object another instance is adopting

- **WHEN** the delete verdict is asked for an admitted live Deployment with managed-by `kustomize`, no UUID label, and an adopt annotation naming another instance
- **THEN** it skips as `adopted-elsewhere`

### Requirement: The deletion transition names one action per call

The deletion transition SHALL take the plan, the caller's state and the outcome of the action it last named, and SHALL return the next state and exactly one action: read one step's live object, delete one step's object, report one step as skipped, or done. It SHALL perform no action itself. A safety-excluded step SHALL be reported as skipped with the reason `safety-excluded` without a read. Every other step SHALL be read before it is judged, and the library's delete verdict SHALL judge it against the plan's instance UUID. A skip verdict SHALL be reported as skipped with the verdict's reason and message. A proceed verdict SHALL name a delete with Foreground propagation and the verdict's UID precondition. When the policy does not prune, the first call SHALL name done and no step SHALL be read or deleted. Every read, delete and skip action SHALL name a step of the plan, and the transition SHALL name no other kind of action. So a prune of an object the instance dropped because another instance adopted it leaves the object in place. Source: 0012:D4:R1/R6, 0012:D1:R2/R4, 0012:D8:R8 (enhancements#103), ADR-008 rules 1 to 3.

#### Scenario: A protected kind is skipped without a read

- **WHEN** the next step is a Namespace
- **THEN** the action is a skip with reason `safety-excluded`
- **AND** no read is named for that step

#### Scenario: An owned object is read, then deleted with Foreground propagation

- **WHEN** a Deployment step is read and the live object is OPM-managed, carries the plan's instance UUID and has UID `u1`
- **THEN** the next action deletes that Deployment with Foreground propagation and a precondition on UID `u1`

#### Scenario: A foreign object is skipped with the verdict's reason

- **WHEN** a step's live object is not OPM-managed
- **THEN** the next action is a skip with reason `not-opm-managed` and the delete verdict's message

#### Scenario: Another instance's object is skipped

- **WHEN** a step's live object carries a different, non-empty instance UUID than the plan's non-empty owner UUID
- **THEN** the next action is a skip with reason `owner-mismatch`

#### Scenario: A plan that does not prune names done at once

- **WHEN** the policy's `Prune` is false and the transition is called with the zero state
- **THEN** the action is done
- **AND** no read or delete is named for any step

#### Scenario: A prune built with the zero policy deletes nothing

- **WHEN** a plan is built from a non-empty stale set with a policy whose `Prune` is false
- **THEN** the first action is done
- **AND** the state records no outcome

#### Scenario: The transition names only reads, deletions, skips and done

- **WHEN** a plan over a mixed inventory is advanced to done through every outcome kind
- **THEN** every action is a read, a delete, a skip or done
- **AND** every read, delete and skip names a step of the plan

#### Scenario: A prune of a dropped object leaves it for the adopting instance

- **WHEN** a deletion plan built from a stale set holds a Deployment whose live object carries the plan owner's UUID label and an adopt annotation naming another instance
- **THEN** advancing the plan past its read records the step as skipped with `adopted-elsewhere` and names no delete action for it
- **AND** the hold verdict releases once the plan is done

## REMOVED Requirements

### Requirement: The apply verdict refuses terminating, foreign and other-instance objects

**Reason**: Its order applies every object in the instance's recorded inventory without judging ownership, and its scenario "An inventoried object annotated for another instance still applies" is the hand-over fight of enhancements#103. The owner decided that an instance refuses an inventoried object another instance adopted, which reverses that scenario.

**Migration**: Replaced by "The apply verdict refuses terminating, foreign, other-instance and adopted-elsewhere objects", which keeps every other scenario's outcome. No frontend calls the apply verdict at the time of this change.

## ADDED Requirements

### Requirement: The apply verdict refuses terminating, foreign, other-instance and adopted-elsewhere objects

The apply verdict SHALL decide in this order and stop at the first match. When there is no live object, it SHALL apply. When the live object has a deletion timestamp, it SHALL refuse as `terminating`, whether or not the object is in the instance's recorded inventory, and nothing SHALL lift that refusal. When the live object's adopt annotation equals the non-empty instance UUID, it SHALL apply. When the object is in the instance's recorded inventory: with an empty instance UUID it SHALL apply; when its adopt annotation is non-blank, it SHALL refuse as `adopted-elsewhere`; otherwise it SHALL apply, whatever its live UUID label, so an instance whose UUID changed keeps applying its own objects. Outside the inventory: when the live managed-by label is not an OPM runtime's value, it SHALL refuse as `foreign-object`, unless the operator install admission lifts it; when the adopt annotation is non-blank and equals the live UUID label, it SHALL refuse as `adopted-elsewhere`, since the instance it names completed the hand-over; when the live UUID label is non-empty and differs from the instance UUID, it SHALL refuse as `other-instance`; when the adopt annotation is non-blank, it SHALL refuse as `adopted-elsewhere`; otherwise it SHALL apply. Outside the inventory an empty instance UUID SHALL never match an adopt annotation, and any non-empty live UUID or non-blank adopt annotation SHALL then count as another instance's. Source: 0012:D8:R1/R2/R5, 0012:D8:R8 (enhancements#103), 0012:D1:R7, 0012:D4:R2.

The `ApplyInput.InInventory` doc and the package doc SHALL state that a frontend drops an object refused as `adopted-elsewhere` while it is in the instance's recorded inventory from the inventory it records next, keeps applying the instance's other objects, and never deletes the object for that refusal. The `adopted-elsewhere` message SHALL name the object and the instance its adopt annotation names, and SHALL name the annotation key `opmodel.dev/adopt` with the instance UUID to set it to, unless the instance UUID is empty, as the only way for this instance to take the object back. For an inventoried object it SHALL say that this instance no longer applies the object and drops it from its inventory. Source: 0012:D8:R3, 0012:D8:R8 (enhancements#103).

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
