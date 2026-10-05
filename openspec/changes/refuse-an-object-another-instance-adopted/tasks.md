Gates per section: `task check` with an absolute `TMPDIR`, in a worktree whose `.cue-cache` is
copied (never symlinked) from the main checkout. Every design.md assumption is checked against
`origin/main` `88afdfb`, so no spike section.

## 1. ownership: refuse an inventoried object another instance adopted

- [x] 1.1 `opm/k8s/ownership/apply.go`: add `RefuseAdoptedElsewhere ApplyRefusal = "adopted-elsewhere"`
  with a doc line (design AD1). Reorder `CanApply` as design AD2 shows: the "adopt annotation
  names this instance" check moves above the inventory check; inside the inventory, an empty
  instance UUID applies (AD3), a non-blank annotation refuses as `adopted-elsewhere` and a UUID
  label alone never does; outside the inventory, an annotation equal to the live UUID label (the
  completed hand-over) refuses as `adopted-elsewhere` ahead of `other-instance`, and after
  `other-instance` a non-blank annotation refuses as `adopted-elsewhere` (AD2, AD4). Add the
  messages of design AD6, naming the annotation's value as the other instance. Rewrite the
  `CanApply` doc's order to match, and cite `0012:D8:R8` once at the symbol.
- [x] 1.2 `opm/k8s/ownership/apply.go`: the `ApplyInput.InInventory` doc says an inventoried object
  is judged only for another instance's adoption, and that a frontend drops an object refused as
  `adopted-elsewhere` from the inventory it records next and never deletes it for that refusal.
- [x] 1.3 `opm/k8s/ownership/doc.go`: replace the sentence "An object still in another instance's
  inventory moves only after that instance stops rendering it, since that instance applies it
  again for as long as it holds it." with how the hand-over now ends: the instance that held the
  object refuses it as `adopted-elsewhere`, drops it from its inventory and leaves it in place;
  annotating it back for that instance reverses the hand-over.
- [x] 1.4 `opm/k8s/ownership/apply_test.go`: in `TestCanApply`, replace the case "an inventoried
  object annotated for another instance still applies" with the refusing case, and add one case
  per new scenario of the ADDED apply requirement (another instance's UUID label in the inventory
  without an annotation applies, the adopter's label and annotation in the inventory refuse, the
  completed hand-over outside the inventory refuses as `adopted-elsewhere`, an annotation that
  differs from the label keeps `other-instance`, the annotation naming this instance over another's UUID label with surrounding whitespace, an
  empty instance UUID inside the inventory, the dropped object outside the inventory with this
  instance's UUID label, a terminating object annotated for another instance, an inventoried
  object of this instance with no annotation, an admitted foreign object outside the inventory
  annotated for another instance, an empty instance UUID refusing an annotated object outside
  the inventory). Rename
  "an inventoried object is not judged for ownership" to "an inventoried foreign object is
  applied" and keep its input. Extend `TestApplyRefusalLiterals` with `adopted-elsewhere`.
  Extend `TestRefusalMessageWording` with the three `adopted-elsewhere` messages, verbatim (other
  instance `u-1`), and the outside-inventory message with an empty instance UUID (OPM-managed,
  no UUID label, annotation `u-1`): no `annotate it` remedy. Add a test that the
  `ApplyInput.InInventory` doc and the package doc state the frontend's part (drop from the
  next inventory, never delete for that refusal).
- [x] 1.5 `go test -race -count=1 ./opm/k8s/...` green, then `task check` green. Commit
  `feat(k8s): refuse an inventoried object another instance adopted`.

## 2. ownership: leave an object another instance is adopting in place on delete

- [x] 2.1 `opm/k8s/ownership/delete.go`: add `SkipAdoptedElsewhere SkipReason = "adopted-elsewhere"`.
  In `CanDelete`, after the `owner-mismatch` check, skip as `adopted-elsewhere` when the trimmed
  adopt annotation is non-blank and the instance UUID is empty or differs from it, with the
  message of design AD6 (design AD5). Update the `CanDelete` doc's order and the
  `DeleteInput.InstanceUUID` and `DeleteInput.Admit` docs (an empty UUID counts any annotation
  as another instance's; admission does not lift the adoption skip).
- [x] 2.2 `opm/k8s/ownership/delete_test.go`: in `TestCanDelete`, add the new delete scenarios
  (annotated for another instance, annotation naming this instance, empty instance UUID skipping
  an annotated object, an admitted Deployment annotated for another instance), and give "an
  empty UUID on either side passes" no annotation as before. Extend
  `TestSkipReasonLiterals` with `adopted-elsewhere`, and pin the skip message verbatim.
- [x] 2.3 `opm/k8s/lifecycle/advance_test.go`: add a case to `TestAdvance` (or a named test beside
  it) for the scenario "A prune of a dropped object leaves it for the adopting instance": a plan
  from a stale set, a live read carrying the owner's UUID label and an adopt annotation naming
  another instance; assert a skipped outcome with `adopted-elsewhere`, no delete action, and that
  `MayReleaseHold` releases once the plan is done. `TestStateJSONGolden` stays unchanged.
- [x] 2.4 `go test -race -count=1 ./opm/k8s/...` green, then `task check` green. Commit
  `feat(k8s): leave an object another instance is adopting in place on delete`.

## 3. docs: package map, layout and ADR-011

- [x] 3.1 Merge gate: this change merges only after the enhancements PR that adds 0012:D8:R8 and
  the `adopted-elsewhere` literal to `#ApplyRefusalReason` and `#SkipReason` in
  `0012/contracts/contracts.cue` has merged. Before merging, check the merged requirement number;
  if it is not `R8`, correct every `0012:D8:R8` in this change's code, docs and spec delta, then
  run the gates again. The PR body states the gate.
- [x] 3.2 `AGENTS.md` § Repository Layout, the `ownership/` line: `CanDelete` also skips
  adopted-elsewhere; `CanApply` also refuses adopted-elsewhere, inside the inventory (the frontend
  drops it from its next inventory) and outside it. `README.md` lists no reasons; leave it.
- [x] 3.3 `adr/011-kubernetes-tier-beside-the-kernel.md`: append to the Status paragraph
  "Amended 2026-10-05 by `refuse-an-object-another-instance-adopted` (0012:D8:R8): `CanApply`
  refuses as `adopted-elsewhere` an object whose adopt annotation names another instance, inside the inventory too, and
  the frontend drops it from its next inventory; `CanDelete` leaves an object annotated for
  another instance in place." (with the number from 3.1).
- [x] 3.4 `task api:diff` reports no incompatible change (two added constants only), and the
  consumer build (`GOTOOLCHAIN=local bash .tasks/consumer-build.sh <consumer-checkout> . <work-dir>`)
  passes against fresh clones of cli and opm-operator `main`. `openspec validate
  refuse-an-object-another-instance-adopted --strict` green. `task check` green. Commit
  `docs(k8s): record the adopted-elsewhere verdicts in the package map and ADR-011`.

## 4. openspec: verify and archive

- [x] 4.1 `openspec verify` (the opsx:verify skill) reports no CRITICAL finding.
- [x] 4.2 At PR time, after the 3.1 gate: `openspec archive refuse-an-object-another-instance-adopted --yes`,
  then `task openspec:check` and `openspec validate --all --strict` green.
- [x] 4.3 Commit `chore(openspec): archive refuse-an-object-another-instance-adopted`.
