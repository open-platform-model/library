## Context

See proposal.md, Why. Design-local decisions are numbered AD1 to AD7 so they collide with no other
numbering. Code references are at library `origin/main` `88afdfb` (2026-10-05):
`opm/k8s/ownership/apply.go` (`CanApply`), `opm/k8s/ownership/delete.go` (`CanDelete`),
`opm/k8s/ownership/doc.go`, and `opm/k8s/lifecycle/advance.go` (`answerRead`, the only library
caller of `CanDelete` with a live object).

The contract is enhancement 0012:D8:R1 to R7 as merged, plus the amendment the owner chose for
enhancements#103: "the apply guard also refuses an in-inventory object whose live adopt
annotation names another instance; A drops it from its next inventory. Extends 0012:D8." The
owner first selected "adopt annotation or UUID label" and narrowed it to the annotation only, so
that a module-path move, which changes the instance UUID, keeps working.

## Goals / Non-Goals

**Goals:**
- `CanApply` refuses an object in the inventory that another instance adopted, with a reason a
  frontend can act on (drop it from its next inventory).
- The hand-over ends with B holding the object: A neither takes it back on its next render nor
  deletes it on its next prune.
- No outcome changes for an input that has nothing to do with another instance's adoption.

**Non-Goals:**
- Any frontend edit, any cluster read.
- Changing what the adopt annotation lifts. It still lifts only `foreign-object` and
  `other-instance`, and only when it names this instance.
- A helper that edits an inventory. Dropping the object is the frontend's write; the library
  states which verdict calls for it.

## Research & Decisions

### AD1: One new reason, `adopted-elsewhere`, on both verdicts

**Context**: The owner's selection names the case "an in-inventory object whose live adopt
annotation names another instance". A frontend has to tell it apart from the
other refusals, because only this one asks it to drop the object from its inventory.
**Explored**: Reusing `other-instance`. It already means "outside the inventory, belongs to
another instance", and both frontends will treat it as "leave the inventory as it is". Making it
mean "drop it" for inventoried objects only would put the action in the frontend's knowledge of
`InInventory`, not in the reason.
**Decision**: `RefuseAdoptedElsewhere ApplyRefusal = "adopted-elsewhere"` and
`SkipAdoptedElsewhere SkipReason = "adopted-elsewhere"`. The two literals are equal on purpose:
they name the same fact about the live object, seen from apply and from delete.
**Rationale**: A reason is the contract's literal (0012:D4), and one literal per fact keeps the
frontends' reports aligned.

### AD2: The apply order

**Context**: Where the new checks sit decides which reason wins when several apply, and must
leave every other input's outcome as it is today.
**Decision**: `CanApply` checks, in order, and stops at the first match:

```go
if in.Live == nil                          { apply }
if deletion timestamp set                  { refuse terminating }        // unchanged, wins everywhere
annotation := adoptAnnotation(in.Live)     // trimmed, "" when blank
if annotation != "" && in.InstanceUUID != "" && annotation == in.InstanceUUID {
    apply                                  // the override, inside or outside the inventory
}
if in.InInventory {
    if in.InstanceUUID == "" { apply }     // AD3
    if annotation != ""                    { refuse adopted-elsewhere }   // names another instance
    apply                                  // a UUID label alone never refuses here
}
if !opmManaged(live) && !admitted          { refuse foreign-object }     // unchanged
if annotation != "" && annotation == liveUUID(live) { refuse adopted-elsewhere } // completed hand-over
if u := liveUUID(live); u != "" && u != in.InstanceUUID { refuse other-instance } // unchanged
if annotation != ""                        { refuse adopted-elsewhere }   // AD4
apply
```

Moving the "annotation names this instance" check above the inventory check changes no outcome
today: an inventoried object applied anyway. Inside the inventory, an annotation naming this
instance wins over another instance's UUID label, the same precedence the override has outside
the inventory: the annotation is the user's latest word on who holds the object. So a user who
annotates the object back for A makes A apply it, and B, which now holds it in its inventory,
refuses it as `adopted-elsewhere` and drops it. Taking an object back is the same act as handing
it over.

Inside the inventory a UUID label naming another instance does not refuse. The instance UUID is
SHA1 of registry path, name and namespace, so pointing an instance at a moved module path keeps
its inventory but changes its UUID; refusing on the old label would drop every one of its own
objects. B can relabel an object only after an annotation naming B, so the annotation already
covers every real hand-over.

Outside the inventory, once B has applied the object its annotation and UUID label both name B.
That is the completed hand-over, and it refuses as `adopted-elsewhere` ahead of
`other-instance`, so A's frontend, which treats `adopted-elsewhere` as non-fatal, does not fail
A's apply for as long as A's module still renders the object. An annotation that differs from
the label keeps `other-instance`.
**Rationale**: The existing reasons keep precedence outside the inventory (AD4), and
`terminating` stays first, so no input the main spec covers changes outcome except the one
scenario this change reverses.

### AD3: An empty instance UUID applies inside the inventory

**Context**: The main spec makes an empty instance UUID fail closed outside the inventory: it
never matches an annotation. Applying that inside the inventory would refuse every inventoried
object that carries any adopt annotation, including one the instance itself was adopted by.
**Explored**: Failing closed inside the inventory too. A refusal there asks the frontend to drop
the object from its inventory, so a missing identity would make the instance let go of its own
annotated objects.
**Decision**: With an empty instance UUID an inventoried object that is not terminating applies,
as it does today.
**Rationale**: "Names another instance" needs an instance to compare against. Where refusing
drops tracking, the safe outcome is the one that keeps the object and its tracking.

### AD4: Outside the inventory, an annotation naming another instance refuses an OPM object

**Context**: Once A drops the object, A's module still renders it, and A's next apply judges it
outside the inventory. If B has applied it, its UUID label is B's and `other-instance` refuses
it. If B has not, its UUID label is still A's (or absent), the object is OPM-managed, and today
the verdict applies it: A records it again and the fight resumes one apply later.
**Decision**: After the `foreign-object` and `other-instance` checks, an object whose adopt
annotation is non-blank and does not name this instance refuses as `adopted-elsewhere`. That
includes an object the operator install admission lifted past `foreign-object`: admission lifts
`foreign-object` only, so it lifts neither the new refusal nor (AD5) the new skip. With an
empty instance UUID any non-blank annotation counts as another instance's, as the live UUID does
(outside the inventory a refusal drops nothing, so failing closed is safe there).
**Rationale**: The annotation is a hand-over instruction for exactly this object; an instance
it does not name has no business applying over it. `foreign-object` and `other-instance` keep
their place, so their messages, which already say the annotation names another instance, stay as
they are.

### AD5: The delete verdict leaves an object annotated for another instance in place

**Context**: The object A drops lands in A's stale set (`inventory.StaleSet`), and A's prune
plans its deletion through `opm/k8s/lifecycle`, which judges each read with `CanDelete`. If B
has applied the object, its UUID label is B's and `owner-mismatch` already skips it. If B has
not, the label is A's and the verdict proceeds: A deletes the object the user is handing to B.
The same holds when A is deleted outright, or when A's module stops rendering the object.
**Decision**: `CanDelete` checks, after `owner-mismatch` and before proceeding: when the live
adopt annotation (trimmed) is non-blank and the instance UUID is empty or differs from it, skip
as `adopted-elsewhere`. An annotation naming this instance changes nothing.
**Explored**: Letting an empty instance UUID disable the comparison, as it disables the owner
comparison. That deletes the object being handed over: a ModulePackage without a UUID holds X,
the user annotates X for B and removes it from the module, and the prune deletes X. AD3's
argument does not carry over: skipping on delete only leaves an object in place, an empty-UUID
instance can never be the one an annotation names, and AD4 already treats any non-blank
annotation as another instance's when the UUID is empty.
**Rationale**: The owner's "A drops it from its next inventory" means A lets go of the object,
not that A deletes it. Placing the check after `owner-mismatch` keeps that reason for every
object that already carries another instance's label. The lifecycle needs no code change: a skip
is a skipped outcome, and the hold verdict releases over skipped steps.

### AD6: Wording

**Decision**: the `adopted-elsewhere` apply message is

```
<obj> was adopted by module instance <other>; this instance no longer applies it and drops it from its inventory; to take it back, annotate it opmodel.dev/adopt=<this>
```

where `<other>` is the annotation's value. The remedy clause is left out when the instance UUID is empty, as `adoptRemedy` does
today. For the outside-the-inventory case (AD4) the frontend has nothing to drop, so the message
reads `<obj> is being adopted by module instance <other>; this instance does not apply it`, and
its remedy is the one the `foreign-object` message uses: `; to let this instance take it over,
annotate it opmodel.dev/adopt=<this>`. For the completed hand-over outside the inventory the
message reads `<obj> was adopted by module instance <other>; this instance does not apply it;
to let this instance take it back, annotate it opmodel.dev/adopt=<this>`. The delete skip message is
`<obj> is being adopted by module instance <other>, not this one; left in place`. No message
names a flag or an enhancement reference. A test pins each wording, as `TestRefusalMessageWording`
does today.

### AD7: Citing the amended decision

**Context**: The 0012:D8 amendment is written in parallel in the enhancements repo and not merged
when this change is planned, so its requirement number is not yet fixed.
**Decision**: The spec deltas and comments cite `0012:D8:R8`, the next free requirement number of
0012:D8, together with enhancements#103. Task 3.1 is a hard merge gate: the library PR
merges only after the amendment has merged, and every citation is corrected if its number
differs.
**Rationale**: A committed citation must resolve; the issue resolves now and the requirement
resolves once the amendment merges.

## Risks / Trade-offs

- **Frontends that stop the whole apply on a refusal.** If a frontend fails the instance on any
  refusal, A stays failed for as long as its module renders the adopted object. That is the same
  exposure `other-instance` has today, and the remedy is the same: remove the object from A's
  module. The frontends' adoption changes decide whether a refusal fails the instance or the
  object.
- **A stale annotation.** An adopt annotation is never removed by OPM. An object adopted into A
  keeps `opmodel.dev/adopt=<A>`; B can take it only by changing the annotation, which is the
  deliberate act the guard asks for.
- **An annotation naming a nonexistent or mistyped UUID.** A refuses the object and drops it,
  every apply refuses it for as long as A renders it, and prune and uninstall skip it (the hold
  still releases). The object is left orphaned with A's label. Remedy: re-annotate it with A's
  UUID, or remove the annotation and adopt it again.
- **Two instances that both render the object, annotation naming B.** A refuses and leaves it,
  B applies it; stable. Annotation changed back to A: B refuses and drops, A applies; stable.
  There is no input under which both apply.
