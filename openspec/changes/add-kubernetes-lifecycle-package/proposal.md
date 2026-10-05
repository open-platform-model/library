## Why

The cli and the operator each delete an instance's objects with a loop of their own, and the two
loops decide differently.

- **Order.** The cli sorts by descending kind weight before it deletes (cli
  `internal/kubernetes/delete.go`). The operator deletes in stored inventory order
  (opm-operator `internal/apply/prune.go`, called from `handleDeletion` in
  `internal/reconcile/moduleinstance.go` and its ModulePackage twin).
- **Propagation.** The cli deletes with Foreground propagation. The operator uses the client
  default, which is Background.
- **Hold release.** Only the operator holds a deletion finalizer, and its `handleDeletion`
  branches on `spec.prune`, an empty inventory, an impersonation failure, the
  `opm.dev/force-delete-orphan` annotation and a Forbidden prune. No other code states those
  branches, so a second frontend that wants the same answer has to copy them.

Enhancement 0012 gives the whole deletion sequence to the library: the plan, the transition that
names the next action and the verdict on releasing the hold (0012:D4, 0012:D1:R1/R2/R5/R6). The
owner decided walkthrough task f2 on 2026-10-03, and 0012:D4 quotes it: "Deletion protocol only, in
opm/k8s/lifecycle, owned by 0012 (DeletionPlan, serialisable State, Advance, MayReleaseHold), used
by both frontends' delete paths; closes the ownership half of 0012:OQ10. No hook semantics." ADR-008
(amended 2026-10-03) fixes where a deletion plan comes from: the persisted inventory plus the live
objects, with no render and no stored plan, and a prune's stale set is computed by the inventory
package and handed to the same plan.

The pieces the plan stands on have shipped in library v1.0.0-beta.6: `inventory.Entry` and
`inventory.StaleSet` (`opm/k8s/inventory`), `ownership.CanDelete` and `ownership.SafetyExcluded`
(`opm/k8s/ownership`), and `object.Sort` with `object.Descending` (`opm/k8s/object`). This change
is the library half of f2. The frontends adopt it in their own changes.

## What Changes

- **`opm/k8s/lifecycle`** (new, on `k8s.io/apimachinery`, `opm/k8s/inventory`,
  `opm/k8s/ownership` and `opm/k8s/object`). It has pure functions and plain values. It does no
  I/O, reads no clock, starts no goroutine and logs nothing (ADR-008 rules 1 to 3).
  - `Policy{Prune, ForceOrphan}`: the 0012 contract's `#DeletionPolicy`.
  - `NewDeletionPlan(entries, policy, ownerUUID) DeletionPlan`: copies the entries, orders them
    by `object.Sort` Descending (stable), and marks each CustomResourceDefinition and Namespace
    step safety-excluded up front (`ownership.SafetyExcluded`). An instance's uninstall builds it
    from the persisted inventory. A prune builds it from `inventory.StaleSet(previous, current)`.
    Nothing renders, and no plan is stored. The plan's fields are unexported, so this is its only
    constructor and no frontend can hand `Advance` a plan in another order. Each caller states its
    policy: the operator passes `spec.prune` and the force-orphan annotation, and the cli's
    instance delete and its stale-set prune pass `Prune: true`.
  - `State`: a JSON-serialisable value the caller owns. It records the next step, what the state
    awaits and one outcome per finished step. Because it is serialisable, a controller can carry
    it across reconciles (0012:D4:R5). op-f2 plans to rebuild it each pass from
    `status.inventory` plus the live objects, with no new CRD field (ADR-008, Deletion plans). The
    cli holds it in memory for one command. A golden test pins its JSON encoding.
  - `Advance(plan, state, event) (State, Action, error)`: one action per call. An action is one
    of these four:
    - `read` (the caller GETs the step's object and feeds back the object, its absence or the
      error);
    - `delete` (Foreground propagation, plus the UID precondition from `ownership.CanDelete`);
    - `skip` (a reason from the contract's `#SkipReason` and a message the library words);
    - `done`.

    The per-object decision is `ownership.CanDelete`. The library classifies the caller's raw
    error with `k8s.io/apimachinery/pkg/api/errors`: NotFound means already absent, and Forbidden,
    Conflict (a failed precondition) and other errors are recorded failures. A failure does not
    stop the plan: the next step still runs, as both frontends behave today. A live object that
    answers the wrong step is a recorded failure, never deleted and never a wedge. Each call
    appends the outcomes of the steps it finished, in step order, and a frontend reports the
    outcomes appended since the state it passed in. When `Policy.Prune` is false, the first call
    names `done` and no object is read or deleted.
  - `MayReleaseHold(plan, state, HoldInput) HoldVerdict`: release or hold, with the contract's
    `#HoldVerdict` reason and a library-worded message. It maps every branch of the operator's
    `handleDeletion` to a verdict. The operator stalls on a Forbidden prune only while it
    impersonates and otherwise requeues; both are a hold, and how a hold is surfaced stays
    frontend policy (design LC5):
    - release when pruning is disabled (`prune-disabled`);
    - release when the inventory is empty (`inventory-empty`);
    - release when the deleting identity is missing and force-orphan is set (`force-orphan`);
    - hold when the identity is missing without force-orphan, or failed (`identity-unavailable`);
    - hold when the plan is not finished, or a step failed (`cleanup-incomplete`);
    - hold when a step was refused as Forbidden (`cleanup-forbidden`);
    - otherwise release (`cleanup-complete`).

    Its inputs do not depend on who bears the hold, so the open questions 0012:OQ5 (the
    `spec.prune` default) and 0012:OQ6 (whether a CLI-owned instance carries a hold) stay open.
  - The protocol has no hook semantics. Its actions are reads, deletions and skips of the plan's
    own inventory entries, plus the hold verdict (0012:D4:R6).
- **Docs**: the `opm/k8s/` package lists in `README.md`, `AGENTS.md` and `CONSTITUTION.md` gain
  `lifecycle`. ADR-011's Status gains the sentence "Amended 2026-10-05 by
  `add-kubernetes-lifecycle-package`".
- **Specs**: `kubernetes-tier` gains the requirements for the deletion plan, the transition,
  outcome classification, the serialisable state, the hold verdict and the package's purity.

Not **BREAKING**. Every symbol is new, and nothing is removed or renamed, so the cli and the
operator compile unchanged. SemVer class: MINOR. Release class of the PR: `feat`.

## Not in this change

- **Frontend adoption.** In the operator change (op-f2), both reconcilers' deletion paths and
  their prune-on-reconcile paths run a short loop around `Advance` with the impersonated client,
  and `MayReleaseHold` decides when the finalizer comes off. That moves the operator's delete order
  to descending weight and its propagation from Background to Foreground. Both are behaviour
  changes, and op-f2 carries the release note. `MayReleaseHold` returns `cleanup-forbidden` for a
  Forbidden failure whatever the identity source, while today's operator stalls only when it
  impersonates and requeues otherwise. op-f2 keeps that split (stall when impersonating, requeue
  when not), so the verdict alone changes no operator behaviour there; if op-f2 chooses
  otherwise, it carries that change in the same release note. The cli change (cli-f2) replaces the body of
  `kubernetes.Delete` and its stale-set prune with the same loop. Each frontend keeps its own
  status messages, events, dry-run reporting and exit codes.
- **Hook semantics.** No step a module declares runs as part of deletion (0012:D4:R6). The rest of
  0012:OQ10 stays with 0009:OQ7.
- **A stored plan or a new CRD field.** The plan is rebuilt from the inventory on each pass
  (ADR-008, Deletion plans).
- **Answers to 0012:OQ5 and 0012:OQ6.** `MayReleaseHold` takes the policy as an input. It does not
  choose the `spec.prune` default, and it does not make the cli claim a hold.
- **A resourceVersion precondition.** The delete action carries the UID precondition only, as
  `ownership.DeleteVerdict.Preconditions` does.
- **The conformance test.** The 0012 contract's `#Conformance` property (executed ⊆ authorised:
  no delete the plan did not name) is asserted by a shared test in library, and 0012 graduation
  needs it. It needs a delete loop to observe, so it lands with the frontend loops in op-f2 and
  cli-f2, or in a later library test-helper change they call. This change ships no loop and does
  not deliver it.
- **The operator install's admitted deletions** (0012:D8:R7). They act on objects outside the
  inventory, so they are not inventory steps. The install calls `ownership.CanDelete` with `Admit`
  directly, as it does today.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `kubernetes-tier`: the deletion plan and its order, the one-action transition, outcome
  classification, the serialisable caller-owned state, the hold verdict and the lifecycle
  package's purity.

## Impact

- Packages: the new `opm/k8s/lifecycle`. No existing package changes. The tier's depguard allow
  list already admits everything the new package imports: the standard library,
  `k8s.io/apimachinery` and the library's own `opm/` packages.
- Downstream: the cli and opm-operator compile unchanged against this tree, and the consumer-build
  job stays green. Their adoption changes (op-f2, cli-f2) gate on the first library release that
  contains this change. cli-f2 checks against cli#307 (install-operator-from-module) if that PR is
  still open.
- Ordering: this change follows lib-e3 (#203) and lib-e4 (#202), both merged. lib-h4 runs beside
  it and may edit the same doc lines. Whichever merges second merges `origin/main` into its branch.
- `enhancement.yaml` declares 0012 and claims no decision. 0012:D4 and the deletion halves of
  0012:D1:R1/R2/R5/R6 are delivered only when both frontends' delete paths run this protocol, so
  that claim belongs to op-f2 and cli-f2. Under-claiming is the safe direction.
