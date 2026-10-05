## Why

`ownership.CanApply` judges ownership only for objects outside the applying instance's recorded
inventory (0012:D8:R1). An object inside it applies whatever its live labels and annotations say.
That leaves one hand-over unguarded (enhancements#103): instance B adopts an object, by the adopt
annotation `opmodel.dev/adopt=<B's UUID>`, while instance A still records the object in its
inventory. A never judges the object, so it applies it again on every reconcile and relabels it
as its own. B then applies it back. The two instances take turns relabelling the object, and a
later prune or delete by either side races the other.

The owner settled enhancements#103: A refuses it too. A's apply guard also refuses an object in
A's inventory whose live adopt annotation names another instance, and A drops the object from its
next inventory. The annotation alone triggers it: a UUID label naming another instance does not,
so an instance whose UUID changes (a module moved to a new path) keeps applying its own objects, except an object whose adopt annotation still names the old UUID; re-annotate it with the new UUID.
This extends 0012:D8 with one requirement, 0012:D8:R8, added by the enhancements amendment that
merges first (design.md AD7).

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
  - an object in the instance's inventory whose live adopt annotation names another instance
    (a UUID label naming another instance alone does not refuse it);
  - an object outside the inventory whose adopt annotation and UUID label name the same other
    instance: the hand-over is complete, and reporting it as `other-instance` would fail A's
    apply after B has taken the object;
  - an object outside the inventory whose adopt annotation names another instance and that
    `foreign-object` and `other-instance` do not refuse (an OPM-managed object, or one the
    operator install admission lifted past `foreign-object`).
  The message names the object and the instance it moved to, says this instance no longer applies
  it and drops it from its inventory, and names the adopt annotation with this instance's UUID as
  the way to take it back. A refused in-inventory object is the frontend's to drop from the
  inventory it records next; it is never pruned for that (below).
- **A new skip reason, `adopted-elsewhere`.** `ownership.SkipAdoptedElsewhere`. `CanDelete` leaves
  an object in place when its adopt annotation is set and does not name this instance (with an
  empty instance UUID, any set annotation). So a
  prune or an instance deletion never deletes an object another instance is adopting, before or
  after that instance has applied it. `opm/k8s/lifecycle` reports the skip through its existing
  skipped outcome, and the hold verdict counts it as done, as it counts every skip.
- **An empty instance UUID keeps today's behaviour inside the inventory.** With no instance UUID
  the verdict cannot tell this instance's label from another's, and with no UUID `CanDelete`
  compares nothing, so a refusal would drop the object from the inventory and the next prune
  would delete it. An inventoried object therefore applies when the instance UUID is empty
  (design.md AD3). Outside the inventory an empty instance UUID still fails closed.
- **Admission lifts neither.** The operator install admission lifts neither the new refusal nor
  the new skip.
- **Docs.** The `ownership` package doc, the `ApplyInput.InInventory` doc, `AGENTS.md`'s package
  map line and an ADR-011 amendment line follow the new verdicts. The two docs state the
  frontend's part: drop a refused inventoried object from the next inventory, never delete it. The
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
  refuses as `adopted-elsewhere` instead of applying. An object outside the inventory whose
  adopt annotation equals its UUID label, naming another instance, now refuses as
  `adopted-elsewhere` instead of `other-instance`. Three more main-spec scenarios are narrowed
  to inputs without an adopt annotation, because their annotated inputs now refuse or skip:
  "An OPM object without a UUID label is applied", "An OPM object of this instance outside the
  inventory is applied" and, on delete, "An empty UUID on either side passes".

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
  instance, and an out-of-inventory object annotated for another instance) and reports a
  completed hand-over as `adopted-elsewhere` instead of `other-instance`; `CanDelete` skips
  inputs it proceeded on before (an object annotated for another instance). One reversed
  main-spec scenario and three narrowed ones (see Not in this change). No frontend
  calls `CanApply` at cli or opm-operator `main`. `CanDelete` is called only through
  `opm/k8s/lifecycle`, which no frontend calls at `main` either, so no consumer changes
  behaviour until it adopts the tier.
- **SemVer:** released as `feat(k8s)!:` with a `BREAKING CHANGE:` footer. Constitution VI asks
  for `feat!` for a breaking change to `opm/` behaviour before GA, `opm/k8s/ownership` shipped in
  v1.0.0-beta.6, and this change reverses a shipped scenario. The footer names both verdict
  changes (`CanApply` refuses an inventoried object annotated for another instance and reports a
  completed hand-over as `adopted-elsewhere`; `CanDelete` skips an object annotated for another
  instance) and the frontends' duty to drop an `adopted-elsewhere` object from the inventory they
  record next. No type or signature changes, and `task api:diff` is clean.
- **Downstream:** the cli and opm-operator ownership adoption changes must drop an object refused
  as `adopted-elsewhere` from the inventory they record, keep applying the rest, never delete it
  for that refusal, and report the refusal like the other ownership refusals. The 0012:D8
  amendment carries these frontend obligations.
- **Merge order:** this PR merges only after the enhancements PR that adds 0012:D8:R8 and the
  `adopted-elsewhere` literal to `#ApplyRefusalReason` and `#SkipReason` has merged.
