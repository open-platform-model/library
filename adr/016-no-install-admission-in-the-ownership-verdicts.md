# ADR-016: No install admission in the ownership verdicts

## Status

Accepted (2026-10-09). Records, as a library rule, the owner's decision of 2026-10-09 to remove all legacy-migration code, which the enhancement record holds as the withdrawal of 0012:D8:R6 and 0012:D8:R7 (enhancements#109). Reversibility: two-way until `v1.0.0` is tagged. After it, bringing an input back is an additive change to the Go API, but a change of who may act on an object, so it needs a new enhancement decision first.

## Context

`opm/k8s/ownership` gives both frontends one verdict for each object they apply or delete. The apply verdict refuses a live object outside the instance's inventory that OPM does not manage, and the delete verdict skips one. The only override was meant to be the annotation `opmodel.dev/adopt`, which a user sets on the live object (0012:D8:R2).

`opm operator install` then needed to take over an operator that an earlier release installed from a manifest. Those objects carry no OPM label, so the verdicts refused them. 0012:D8:R6 and 0012:D8:R7 added an exception, and the library carried it as a boolean input, `Admit`, on `ApplyInput` and `DeleteInput`. Set by the caller, it lifted `foreign-object` on apply, and `not-opm-managed` on delete for an `apps` Deployment, a RoleBinding and a ClusterRoleBinding. The library could not check the caller's proof: the field was the caller's word.

The owner decided to remove the migration of a manifest-installed operator. The cli, the only caller that set the input, removes its migration in cli#357; opm-operator never set it. The choices were to keep the field unused, to deprecate it and remove it after `v1.0.0`, or to remove it now.

## Decision

The library removes `ApplyInput.Admit` and `DeleteInput.Admit`, with the helpers and the kind list that existed only for them, in one breaking beta before `v1.0.0`. The verdict inputs carry no field whose only effect is to lift a refusal or a skip. Outside the instance's inventory, the adopt annotation naming the applying instance is the only override of an apply refusal (0012:D8:R3), and nothing lifts the `not-opm-managed` skip on delete. A frontend that installs the operator is judged as any other instance. A test in `opm/k8s/ownership` pins the fields of both inputs by name, so a new input fails there first.

The adopt rule is not changed: 0012:D8:R8 and every `adopted-elsewhere` verdict stand as they were. No verdict changes for an input that did not set `Admit`, because each of the two conditions that read it reduces to the condition that remains.

Keeping the field unused was rejected: from `v1.0.0` it is contract (CONSTITUTION VI), and it is an exception to the guard that rests on the caller's word alone. Deprecating first was rejected: no consumer needs the time, and a deprecation that outlives `v1.0.0` can only end in a new major.

Security: the change removes a bypass of the ownership guard and adds no trust boundary, input, secret or outbound call. Two caller-supplied inputs remain, and the library trusts both as the frontend's ownership record: `InInventory` and `InstanceUUID`. For the same live object the apply verdict refuses outside the inventory and applies inside it, so a frontend must derive `InInventory` only from the inventory it recorded itself, and whoever can write that record can widen what the instance applies over. The other residual risk is the one 0012 already records: a principal who may patch an object can set its adopt annotation.

## Consequences

**Positive:** Outside the inventory the ownership guard has one override, and it is visible on the object. Two exported fields and four unexported functions leave the package. The fields of both inputs are pinned by a test, so an input cannot be added without a spec change.

**Negative:** `opm operator install` no longer takes over an operator that was installed from a manifest: each such object is refused as `foreign-object` with the annotation to set, and the earlier Deployment and role bindings are not deleted for the user. The cli owns the guidance for that case.

**Operational:** The removal is a `feat!` commit whose `BREAKING CHANGE:` footer is the migration note (ADR-010). cli `main` sets `Admit` until cli#357 merges, so this change merges after it; `Consumer build (cli)` is red on the pull request until then.

**Trade-off:** A future migration that must take over unlabelled objects has to mark them on the cluster (the annotation) or go through a new enhancement decision. That cost is accepted to keep the guard free of an input that only lifts a refusal.
