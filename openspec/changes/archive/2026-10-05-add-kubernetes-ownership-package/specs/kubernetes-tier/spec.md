## ADDED Requirements

### Requirement: The adopt annotation key is fixed in the label vocabulary

`opm/k8s/labels` SHALL declare the adopt annotation key `opmodel.dev/adopt`. It is an annotation a user sets on an existing live object, and its value names the adopting instance by the value of that instance's `module-instance.opmodel.dev/uuid` label. No function in the library SHALL set it on an object, and the package documentation SHALL say that no OPM runtime writes it. Source: 0012:D8 (the implementing change fixes the key), 0012:D8:R6.

#### Scenario: The key is the fixed literal

- **WHEN** a test reads the adopt annotation constant
- **THEN** it equals `opmodel.dev/adopt`

#### Scenario: No library code sets the adopt annotation

- **WHEN** every non-test Go file under `opm/` is parsed
- **THEN** no file outside `opm/k8s/labels` spells the literal `opmodel.dev/adopt`
- **AND** no file outside `opm/k8s/labels` and `opm/k8s/ownership`, which only reads the key, refers to the adopt annotation constant

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
- **THEN** they are `safety-excluded`, `already-absent`, `not-opm-managed`, `owner-mismatch`, `terminating`, `foreign-object` and `other-instance`

### Requirement: Safety-excluded kinds match on group and kind

The library SHALL report a core-group `Namespace` and an `apiextensions.k8s.io` `CustomResourceDefinition` as safety-excluded, and no other group and kind pair. The test SHALL NOT need a live object. Source: 0012:D1:R3, the 0012 contract's `#safetyExcluded`.

#### Scenario: The two protected kinds are excluded

- **WHEN** the core group with kind `Namespace`, and `apiextensions.k8s.io` with kind `CustomResourceDefinition`, are tested
- **THEN** both are safety-excluded

#### Scenario: A same-named kind in another group is not excluded

- **WHEN** `example.com` with kind `Namespace`, and `example.com` with kind `CustomResourceDefinition`, are tested
- **THEN** neither is safety-excluded

### Requirement: The delete verdict skips with a reason or proceeds with the judged object's identity

The delete verdict SHALL decide in this order and stop at the first match: skip as `safety-excluded` when the kind is safety-excluded, whatever the live object; skip as `already-absent` when there is no live object; skip as `not-opm-managed` when the live managed-by label is not an OPM runtime's value; skip as `owner-mismatch` when the live UUID label and the instance's UUID are both non-empty and differ; otherwise proceed. An empty UUID on either side SHALL pass the owner comparison. An object being deleted SHALL NOT be skipped for that reason. A proceed verdict SHALL carry the UID and the resourceVersion of the live object it judged. Its DELETE precondition SHALL name that UID and SHALL NOT name a resourceVersion. A caller MAY add the carried resourceVersion itself. A skip verdict, and a proceed verdict whose judged UID is empty, SHALL yield no precondition at all, never a precondition on an empty UID. A skip verdict SHALL carry a message naming the object and the reason. Source: 0012:D1:R3/R4, 0012:D4:R1.

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

- **WHEN** the live Deployment is OPM-managed and either its UUID label or the instance UUID is empty
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

### Requirement: The apply verdict refuses terminating, foreign and other-instance objects

The apply verdict SHALL decide in this order and stop at the first match. When there is no live object, it SHALL apply. When the live object has a deletion timestamp, it SHALL refuse as `terminating`, whether or not the object is in the instance's recorded inventory, and nothing SHALL lift that refusal. When the object is in the instance's recorded inventory, it SHALL apply. When the live object's adopt annotation equals the non-empty instance UUID, it SHALL apply. When the live managed-by label is not an OPM runtime's value, it SHALL refuse as `foreign-object`, unless the operator install admission lifts it. When the live UUID label is non-empty and differs from the instance UUID, it SHALL refuse as `other-instance`. Otherwise it SHALL apply. An empty instance UUID SHALL never match an adopt annotation, and any non-empty live UUID SHALL then count as another instance's. Source: 0012:D8:R1/R2/R5, 0012:D1:R7, 0012:D4:R2.

#### Scenario: A new object is applied

- **WHEN** the apply verdict is asked for an object with no live object
- **THEN** it applies

#### Scenario: A terminating object is refused everywhere

- **WHEN** the live object has a deletion timestamp
- **THEN** the verdict refuses as `terminating` when the object is in the inventory, when it is outside it, when it carries the adopt annotation naming this instance, and when it is admitted

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

### Requirement: The adopt annotation is the only override and the refusal names it

An adopt annotation whose value equals the instance UUID SHALL lift the `foreign-object` and `other-instance` refusals. The annotation value SHALL be compared with surrounding whitespace trimmed, and a value that is empty or only whitespace SHALL count as no annotation. An adopt annotation with any other value SHALL lift nothing, and the refusal message SHALL say that the annotation names another instance. The refusal message for `foreign-object` and `other-instance` SHALL name the object, the annotation key `opmodel.dev/adopt` and the instance UUID to set it to, unless the instance UUID is empty. The `other-instance` message SHALL ask the user to remove the object from the instance that owns it before annotating it, since that instance applies the object again for as long as it holds it in its inventory. No message SHALL name any other way past a refusal, such as a command-line flag. No message SHALL carry an enhancement reference. Source: 0012:D8:R2/R3.

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

Both verdicts SHALL take an admission input that a caller sets only for an object it has proven came from an earlier operator release's install manifest. On apply, admission SHALL lift `foreign-object` only, and only when the live object carries no UUID label or carries the instance UUID. On delete, admission SHALL lift `not-opm-managed` only, only when the live object carries no UUID label at all, and only for an `apps` `Deployment`, a `rbac.authorization.k8s.io` `RoleBinding` or a `rbac.authorization.k8s.io` `ClusterRoleBinding`, the kinds install may delete outside the inventory. The UUID label is the only identity admission compares. Admission SHALL NOT lift `terminating`, `other-instance`, `owner-mismatch`, `safety-excluded` or `already-absent`. The library does not check the proof. Source: 0012:D8:R6/R7, 0012:D4:R1.

#### Scenario: A proven earlier-manifest object is admitted on apply

- **WHEN** the operator install asks for the earlier `Namespace/opm-operator-system`, live with managed-by `kustomize` and no UUID label, admitted and outside the inventory
- **THEN** the verdict applies

#### Scenario: An admitted object carrying another identity is refused

- **WHEN** an admitted object outside the inventory is not OPM-managed but carries a UUID label that differs from the instance UUID
- **THEN** the apply verdict refuses as `foreign-object`

#### Scenario: Admission never lifts the terminating refusal

- **WHEN** an admitted proven object has a deletion timestamp
- **THEN** the apply verdict refuses as `terminating`

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
