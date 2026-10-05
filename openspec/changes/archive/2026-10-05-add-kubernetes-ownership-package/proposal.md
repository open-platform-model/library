## Why

The cli and the operator each judge, in their own code, whether they may apply over an existing
object and whether they may delete one. The two judgements differ, and each has a gap.

- **Apply.** The cli's guard (`PreApplyExistenceCheck`, cli `internal/inventory/stale.go`) runs
  only on an instance's first apply and never on a dry run. It skips an object it cannot read,
  checks only `deletionTimestamp` and the managed-by label, and never compares the instance
  identity. The operator has no apply guard at all: `internal/apply/apply.go` hands the rendered
  set to Flux's server-side apply, which always forces ownership.
- **Delete.** The cli's `checkDeletable` (`internal/kubernetes/delete.go`) re-reads each object
  and skips it unless it is OPM-managed and its instance UUID matches. The cli's prune
  (`PruneStaleResources`) deletes with no re-read. The operator's prune
  (`internal/apply/prune.go`) re-reads, but it protects CustomResourceDefinitions and Namespaces
  by kind alone. Neither frontend sends a DELETE precondition, so an object replaced between the
  re-read and the DELETE (same name, new UID) is deleted on the strength of a verdict about its
  predecessor.

Enhancement 0012 makes this one verdict that both frontends consult (0012:D1:R4/R7, 0012:D4:R1/R2,
0012:D8, merged in enhancements#88). The owner decided task e4 in the beta-1 kernel-plan
walkthrough (2026-10-03): "opm/k8s/ownership CanApply/CanDelete (pure verdicts with reasons)
used on every apply/prune/delete path in both frontends. Apply guard runs on every apply for
objects not already in the instance inventory. Override = per-object adopt annotation on the
existing object (e.g. `opmodel.dev/adopt: <instance-uuid>`) [...]". 0012:D8 leaves the
annotation key to the implementing change, and this is that change. It is the library half;
the frontends adopt it in their own changes.

The delete verdict also closes the precondition gap. It carries the UID and resourceVersion of
the live object it judged, so a frontend can make its DELETE conditional on the object being the
one that was judged.

## What Changes

- **`opm/k8s/labels`**: `AnnotationAdopt = "opmodel.dev/adopt"`, the adopt annotation key
  0012:D8 asks this change to fix. It is a user-written annotation on a live object, and its
  value is the adopting instance's UUID. The package doc says it now names one annotation besides
  the labels, and that no OPM runtime writes it (0012:D8:R6).
- **`opm/k8s/ownership`** (new, on `k8s.io/apimachinery` and `opm/k8s/labels`): pure functions
  over explicit inputs. It does no I/O, reads no clock, and logs nothing.
  - `SafetyExcluded(group, kind)`: a core Namespace or an `apiextensions.k8s.io`
    CustomResourceDefinition. The match is on group and kind, as the cli's `IsProtectedKind`
    matches since cli#285, so a same-named kind in another group is not excluded.
  - `CanDelete(DeleteInput) DeleteVerdict`: proceed, or skip with one of the reasons
    `safety-excluded`, `already-absent`, `not-opm-managed` and `owner-mismatch` (0012:D1:R4,
    the 0012 contract's `#SkipReason`). The instance UUID tolerance is today's in both frontends:
    an empty UUID on either side passes. A proceed verdict carries the live object's UID and
    resourceVersion. `DeleteVerdict.Preconditions()` returns the DELETE precondition with the UID
    only, and nil on a skip or when the live object has no UID (design OW5).
  - `CanApply(ApplyInput) ApplyVerdict`: apply, or refuse with one of `terminating`,
    `foreign-object` and `other-instance` (0012:D8, the contract's `#ApplyRefusalReason`). An
    object that does not exist is always applied. An object being deleted is refused whether or
    not it is in the inventory, and nothing lifts that refusal (0012:D8:R5). An existing object
    outside the instance's inventory is refused when OPM does not manage it or when it carries
    another instance's UUID (0012:D8:R1). The adopt annotation naming this instance lifts both
    ownership refusals (0012:D8:R2). An annotation naming another instance lifts nothing.
  - **The install admission** (0012:D8:R6/R7): both inputs take `Admit`, which the caller sets
    only for an object it has proven came from an earlier operator release's install manifest.
    On apply it lifts `foreign-object` for an object that carries no other instance's UUID; on
    delete it lifts `not-opm-managed` under the same condition, and only for the kinds
    0012:D8:R7 lets install delete (`apps` Deployment, `rbac.authorization.k8s.io` RoleBinding
    and ClusterRoleBinding). It never lifts `terminating`,
    `other-instance`, `owner-mismatch` or `safety-excluded` (design OW4).
  - Every verdict carries a message the library words, so both frontends report a refusal or a
    skip in the same words. An ownership refusal names the annotation key and this instance's
    UUID; the `other-instance` refusal first asks the user to remove the object from the
    instance that owns it. No message names another override (0012:D8:R3).
- **Docs**: the `opm/k8s/` package lists in `README.md`, `AGENTS.md` and `CONSTITUTION.md` gain
  `ownership`.
- **Specs**: `kubernetes-tier` gains the adopt-key, delete-verdict, apply-verdict and admission
  requirements.

Not **BREAKING**. Every symbol is new and nothing is removed or renamed, so the cli and the
operator compile unchanged. SemVer class: MINOR. Release class of the PR: `feat`.

## Not in this change

- **Frontend adoption.** The operator change (op-e4) runs the apply guard before both
  reconcilers apply, through the impersonated client, uses `CanDelete` in prune and deletion,
  adds the additive ModulePackage `status.instanceUUID` (0012:D8:R4), and documents
  `spec.rollout.forceConflicts` as no ownership override. The cli change (cli-e4) replaces
  `PreApplyExistenceCheck` with a per-object `CanApply` on every apply, puts a live read and
  `CanDelete` in front of prune, and turns `checkDeletable` into `CanDelete`. Both send the
  DELETE preconditions. Those changes decide their own read-error policy and how they report a
  precondition failure.
- **Cluster reads.** The library never reads the live object. The frontend reads it with its own
  client and hands it in (ADR-008, ADR-011).
- **The deletion plan and hold** (lib-f2, `opm/k8s/lifecycle`). It will call `CanDelete` for each
  entry. This change ships only the per-object verdict.
- **Inventory entries** (lib-e3, `opm/k8s/inventory`, in flight beside this change). The verdicts
  take their own four-field `Object` identity. A frontend fills it from an inventory entry, and no
  adapter is added here.
- **A resourceVersion precondition by default.** The verdict carries the resourceVersion, and a
  caller that wants the strict precondition adds it (design OW5).
- **Recording the fixed key in enhancement 0012.** 0012:D8 calls the key indicative; the
  enhancements repo records `opmodel.dev/adopt` in its own change.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `kubernetes-tier`: the adopt annotation key, the delete verdict with its skip reasons and
  preconditions, the apply verdict with its refusal reasons and the adopt override, and the
  operator install admission.

## Impact

- Packages: `opm/k8s/labels` (one constant, package doc) and the new `opm/k8s/ownership`. No
  other package changes. The tier's depguard allow list already admits what the new package
  imports (`k8s.io/apimachinery`, the library's own `opm/` packages).
- Downstream: cli and opm-operator compile unchanged against this tree, and the consumer-build
  job stays green. Their adoption changes (op-e4, then cli-e4) gate on the first library release
  that contains this change. The operator also needs its CRD release first, for
  ModulePackage `status.instanceUUID`.
- Ordering (wave-2 serialization): this change follows lib-e2e5 (merged, `ca7c56b`). lib-e3 and
  lib-f5 add sibling packages in parallel and edit the same package-list lines in `README.md`,
  `AGENTS.md` and `CONSTITUTION.md`, so whichever merges second merges `origin/main` into its
  branch. lib-f2 builds on `CanDelete`.
- `enhancement.yaml` declares 0012 with no decision claimed. 0012:D8 (and the apply halves of
  0012:D1:R7 and 0012:D4:R2) are delivered only when both frontends consult the verdict; that
  claim belongs to their adoption changes. Under-claiming is the safe direction.
