## Why

`ownership.CanApply` judges ownership only for objects outside the applying instance's recorded
inventory (0012:D8:R1). An object inside it applies whatever its live labels and annotations say.
That leaves one hand-over unguarded (enhancements#103): instance B adopts an object, by the adopt
annotation `opmodel.dev/adopt=<B's UUID>`, while instance A still records the object in its
inventory. A never judges the object, so it applies it again on every reconcile and relabels it
as its own. B then applies it back. The two instances take turns relabelling the object, and a
later prune or delete by either side races the other.

The owner settled enhancements#103: A refuses it too. A's apply guard also refuses an object in
A's inventory whose live adopt annotation or UUID label names another instance, and A drops the
object from its next inventory. This extends 0012:D8 with one requirement (the enhancements
amendment is planned in parallel, see design.md AD7).

Two gaps follow from "A drops it", and the change closes both so that the hand-over ends with B
holding the object and A leaving it alone:

- **A's next render.** A's module still renders the object, so on A's next apply it sits outside
  A's inventory. If B has not yet applied it, its UUID label is still A's, and today's verdict
  applies it and A records it again. The annotation naming B must refuse there too, or A takes
  the object back one apply after dropping it.
- **A's next prune.** Dropping the object from A's inventory puts it in A's stale set, and A's
  prune runs it through `ownership.CanDelete`. If B has not yet applied it, its UUID label is
  still A's and the delete verdict proceeds: A deletes the object B is adopting. The delete
  verdict must leave in place an object whose adopt annotation names another instance.

## What Changes

- **A new refusal reason, `adopted-elsewhere`.** `ownership.RefuseAdoptedElsewhere`. `CanApply`
  refuses with it:
  - an object in the instance's inventory whose live adopt annotation names another instance,
    or whose live UUID label names another instance while its adopt annotation does not name
    this one;
  - an OPM-managed object outside the inventory whose adopt annotation names another instance
    and that the existing reasons do not already refuse.
  The message names the object and the instance it moved to, says this instance no longer applies
  it and drops it from its inventory, and names the adopt annotation with this instance's UUID as
  the way to take it back. A refused in-inventory object is the frontend's to drop from the
  inventory it records next; it is never pruned for that (below).
- **A new skip reason, `adopted-elsewhere`.** `ownership.SkipAdoptedElsewhere`. `CanDelete` leaves
  an object in place when its adopt annotation and the instance UUID are both set and differ. So a
  prune or an instance deletion never deletes an object another instance is adopting, before or
  after that instance has applied it. `opm/k8s/lifecycle` reports the skip through its existing
  skipped outcome, and the hold verdict counts it as done, as it counts every skip.
- **An empty instance UUID keeps today's behaviour inside the inventory.** With no instance UUID
  the verdict cannot tell this instance's label from another's, and with no UUID `CanDelete`
  compares nothing, so a refusal would drop the object from the inventory and the next prune
  would delete it. An inventoried object therefore applies when the instance UUID is empty
  (design.md AD3). Outside the inventory an empty instance UUID still fails closed.
- **Docs.** The `ownership` package doc, the `ApplyInput.InInventory` doc, `AGENTS.md`'s package
  map line, `README.md`'s layout line and an ADR-011 amendment line follow the new verdicts. The
  package doc's sentence that an object still in another instance's inventory moves only after
  that instance stops rendering it is replaced.

Not in this change:

- Any frontend edit. Neither the cli nor the operator calls `CanApply` yet; both adopt it in their
  ownership changes after the next library release, and they take this behaviour at once,
  including dropping a refused in-inventory object from their next inventory.
- The enhancements amendment of 0012:D8 and the `#ApplyRefusalReason` and `#SkipReason`
  literals in `0012/contracts/contracts.cue` (planned in parallel in the enhancements repo).
- Any change to the `foreign-object`, `other-instance`, `terminating`, `not-opm-managed`,
  `owner-mismatch`, `safety-excluded` or `already-absent` outcomes for an input that produces one
  today, except the main-spec scenario this change reverses on purpose: an inventoried object,
  carrying this instance's UUID label, whose adopt annotation names a different instance, now
  refuses as `adopted-elsewhere` instead of applying.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `kubernetes-tier`: the apply verdict gains the `adopted-elsewhere` refusal (the requirement is
  replaced, since one of its scenarios is reversed), the delete verdict gains the
  `adopted-elsewhere` skip, the reason literals gain `adopted-elsewhere`, and the adopt-annotation
  requirement's rationale for the `other-instance` remedy is reworded.

## Impact

- **Packages:** `opm/k8s/ownership` (two exported constants, `CanApply` and `CanDelete` logic,
  messages, docs). `opm/k8s/lifecycle` gains a test only. No other `opm/` package changes.
- **Public surface:** additive. `RefuseAdoptedElsewhere` and `SkipAdoptedElsewhere` are new
  exported constants; no signature changes. `task api:diff` reports no incompatible change.
- **Behaviour:** `CanApply` refuses inputs it applied before (in-inventory adoption by another
  instance, and an out-of-inventory object annotated for another instance), and `CanDelete`
  skips inputs it proceeded on before (an object annotated for another instance). No frontend
  calls `CanApply` at cli or opm-operator `main`. `CanDelete` is called only through
  `opm/k8s/lifecycle`, which no frontend calls at `main` either, so no consumer changes
  behaviour until it adopts the tier.
- **SemVer:** MINOR on the beta line, released as `feat`. The verdicts are new in this beta line
  and unconsumed, so the behaviour change carries no migration cost.
- **Downstream:** the cli and opm-operator ownership adoption changes must drop an object refused
  as `adopted-elsewhere` from the inventory they record, keep applying the rest, and report the
  refusal like the other ownership refusals.
