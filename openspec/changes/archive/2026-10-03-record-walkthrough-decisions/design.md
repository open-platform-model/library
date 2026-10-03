## Context

See proposal.md for why. The facts this change rests on, read on library `cf79a5c`:

- ADR-008's rule 4 ends "The kernel derives no ordering of its own. (Clarified by ADR-011: this excludes module-specific ordering only; ...)". Its Status line already carries two dated amendments by `record-kubernetes-tier`. Its Consequences end with "What this does not decide" and two "Relation to" paragraphs.
- ADR-011 item 7 ("Ordering has two layers") says that module-declared order (hooks, `dependsOn`, phases) is data that comes off the CUE build, under ADR-008 rule 4. The `kubernetes-tier` main spec requirement "Kind-class order is a Kubernetes fact that lives in the tier" says the same with a SHALL.
- ADR-005 rule 2 ends "The Kernel holds no built value between calls, and a caller cannot obtain one to hold." `opm/kernel/doc.go` repeats it in the Goroutine safety section ("no built value is retained between calls, and a caller cannot obtain one to hold"). `Compiled.Value` is a live `cue.Value` into the render build (`render_decode.go`, `splitOutput`), so the second half is false. ADR-007 rule 1 already words the rule correctly for `Package`.
- ADR-004's last Consequences paragraph sends the arming condition to "the GA release checklist". No such checklist exists. The GA exit criteria are 0021:D8 (`enhancements/0021/03-decisions.md`). `README.md` § API stability → Beta promise and `migrations/README.md` § Status describe GA with no link to them. The arming condition itself is already a GA exit criterion: 0021:D8:R4.
- There are eleven ADRs. The next free number is 012. No open library PR or worktree claims it.
- core#62, "Adopt #Match from the library render glue into core", is open.

## Goals / Non-Goals

**Goals:**

- Each of the five owner answers is written down once in the library, in the ADR it belongs to, and the specs point at the ADR.
- No library text still says that module-internal ordering is coming, that a caller cannot hold a built value, or that a GA release checklist exists.

**Non-Goals:**

- Any Go code, and any doc comment in `opm/kernel/doc.go` other than the retention sentence (`restore-kernel-godoc` edits that file).
- Editing enhancements 0012 or 0021, or closing core#62.

## Decisions

### 1. Amend ADR-008 and ADR-011 in place, the record-kubernetes-tier way

ADR-008 gets one dated sentence on its Status line per answer ("Amended 2026-10-03 by `record-walkthrough-decisions` ..."), following the ADR-007 and ADR-008 precedent. Rule 4's parenthetical grows by one sentence: no module-internal ordering (`dependsOn` edges, phases) is planned, and if one is ever needed it comes off the build. Lifecycle hooks are not on that list: they remain the parked 0009 work, carried as build data under rule 4, which the j5 answer in this same change also defers to. In the OPM model, ordering across modules belongs to a future Bundle definition, which uses the order its modules are defined in; the library adds none before then. A frontend's readiness gating between its own custom resources (the operator's ModulePackage `spec.dependsOn`) is outside this rule. A future Bundle definition may subsume that gating; the owner has not decided that, and the ADRs say it is not decided. The new Status sentence names the 2026-10-02 sentence "Module-declared order (hooks, `dependsOn`, phases) is still data off the build" as superseded, so the dated amendment stays as written and the Status line does not contradict itself. The rule's first sentence ("Ordering edges, hook declarations and wait conditions are emitted by the CUE build") stays, because it says where such facts would come from, not that they are planned.

The deletion answer goes into a new Consequences paragraph, "Deletion plans", placed before "What this does not decide": an instance's deletion plan (uninstall) is built from the persisted inventory plus the live objects, with no render and no stored plan, and a prune's stale set is computed by the inventory package and handed to the same plan. The requirement's Source is ADR-008 (owner decision 2026-10-03); the deletion protocol is 0012:D4. "What this does not decide" gains the two deferred questions (planning from a render, and lifecycle transition detection: install, upgrade, reconfigure, no-op, uninstall), deferred to the parked 0009 hook work. The text says this is distinct from the deletion step transition ADR-011 item 6 assigns to the tier, which stays the tier's.

ADR-011 item 7 gets the same ordering sentence, with one dated Status sentence. 0012:D5 is the decision ADR-011 records, so ADR-011 has to agree with the amended 0012:D5 and ADR-008.

**Reading chosen (ambiguity):** the owner named "0012:D5 and ADR-008 rule 4". ADR-011 is the library's record of 0012:D5, so leaving its item 7 unchanged would contradict both. This change amends it, with the smallest edit. ADR-008's closing "Relation to enhancement 0012" paragraph is reworded so it names kind-class order, a Kubernetes fact the tier delivers, and the deletion sequence as the first lifecycle facts, and says module-declared ordering is not planned (supervisor triage 2026-10-03).

### 2. ADR-012 is a new ADR, not an amendment

Where matching lives is a separate question from what ADR-005 to ADR-008 settle, and core#62 needs one place to point at. ADR-012 follows `adr/TEMPLATE.md`:

- **Context:** 0019:D10 moved matching from Go into the render glue (`opm/internal/renderstage/render.cue.tmpl`), and 0019:D17 removed `#Platform.#matchers` from core because the glue owns the reverse index. core#62 proposes moving `#Match` to core so that a second runtime can read the matching contract without embedding the kernel, and so that the matcher version is pinned by the platform's core rather than by the kernel binary. `read-provider-count-from-core` moved one derived rule, the provider count, into core: the glue reads `#contracts.providedBy`, and `opm/kernel/render_inventory_parity_test.go` checks the render's verdicts against the platform inventory.
- **Decision:** the matching algorithm stays in the library glue, and 0019:D10/D17 stand. A single derived rule may move into core, one at a time, when core can compute it from core shapes alone. Each move is an additive core release and ships with a parity test that proves the core field and the verdict it replaces agree over the served fixtures. The per-catalog provider set (behind `Catalog.Provides()`) is the next candidate.
- **Rejected:** moving `#Match` whole (core#62, not now), and reversing 0019:D17 with a core reverse index. Either one makes every matching fix a core release and every matching break a `feat!` on core (a new major after GA), and needs an enhancement decision amending 0019.
- **Consequences:** core#62 is to be closed as not now, with a pointer to ADR-012. This change does not close it.

**Reading chosen (additions):** the conditions "an additive core release" and "when core can compute it from core shapes alone" are the planner's reading, not the owner's d5 text. They follow from the owner's h2 answer (each move an additive core release) and from the `providedBy` precedent. The supervisor accepted the "core shapes alone" condition on 2026-10-03.

### 3. ADR-005 gets ADR-007's holder-bounded wording, not a rewrite

Rule 2's last sentence becomes: "The Kernel holds no built value between calls. Each returned `*Compiled` carries a `cue.Value` into the render's build, so a caller that holds one keeps that build alive until it releases it, and retention is bounded by what the caller holds (ADR-007 rule 1)." The rest of rule 2 is made consistent with it: the heading becomes "The kernel does not hold the context past the render", "drops the context" becomes "drops its own references to the context", and the stale type name `[]*core.Compiled` becomes `[]*kernel.Compiled`. The Status line gets one dated sentence: the rule is reworded to ADR-007's holder-bounded rule, Render output keeps `Compiled.Value` for now, and it moves to bytes before GA at the latest, earlier if render output is to be cached. `doc.go` gets the whole retention sentence rewritten (its "dropped when Render returns" half is false in the same way) and nothing else. The retention figures in ADR-005's Consequences stay. They measured a worker that dropped its results, which is still the bounded case.

### 4. Link to 0021:D8, and name the library's own criteria

ADR-004's "so the GA release checklist picks it up" becomes a pointer to the GA exit criteria in 0021:D8 (`enhancements/0021/03-decisions.md` at the workspace root). It cites 0021:D8 with its arming criterion, 0021:D8:R4. The arming condition is recorded here and in `migrations/README.md`. `README.md` § API stability → Beta promise gains one sentence after "GA drops the suffix ...": GA is cut only when 0021:D8's exit criteria hold, among them the library's own (`0021:D8:R11` to `0021:D8:R14`: no `cue.Value` in Render output, typed fetch and resolution errors, docs and specs that match the code, and three consecutive library betas with no breaking change). `migrations/README.md` § Status gets the same link. ADR-010 already defers to ADR-004 and is not touched.

**Reading chosen:** the owner text says "R11 to R14 in decision 0021:D8". The repo's reference rule writes one requirement of a decision as `0021:D8:R11`, so that form is used.

### 5. Spec deltas

- `kubernetes-tier`: MODIFIED "Kind-class order is a Kubernetes fact that lives in the tier". Its existing scenario is kept unchanged (OpenSpec 1.12 refuses a MODIFIED that drops one), and a scenario for the Bundle and no-module-internal-ordering text is added. ADDED "Deletion plans come from the inventory and the live objects".
- `single-build-render`: ADDED "Matching stays in the glue; derived rules move into core one at a time".
- `kernel-runtime`: ADDED "A held Render output keeps its build alive", and MODIFIED "Goroutine Safety Contract" (all five scenarios kept), whose "a fresh `cue.Context` that is released when `Render` returns" becomes "whose references the kernel releases when `Render` returns".
- `single-build-render`: MODIFIED "Each render is its own build in its own context" (both scenarios kept), same rewording of "release it when `Render` returns".
- `migration-docs`: MODIFIED "Enforcement design is recorded and armed at GA", with both scenarios kept and one added for the 0021:D8 link.

## Risks / Trade-offs

- [This change cites 0012:D5 wording and `0021:D8:R11` to `0021:D8:R14` that the enhancements change `record-walkthrough-decisions` writes] → this change merges after that one. Until then, the R11 to R14 citations point at text that is not on enhancements `main`.
- [`restore-kernel-godoc` edits `opm/kernel/doc.go` in parallel] → its task 1.1 rewraps every prose paragraph of doc.go, so whichever change merges second gets a conflict over the whole Goroutine-safety paragraph. The second merger re-applies the retention sentence by hand, wraps it to the restored width, and re-runs restore-kernel-godoc's word-diff check and doc_test.
- [ADR-005's "a caller cannot obtain one to hold" may be quoted elsewhere] → a grep at section 3 covers `adr/`, `opm/`, `README.md`, `AGENTS.md` and `openspec/specs/`.
