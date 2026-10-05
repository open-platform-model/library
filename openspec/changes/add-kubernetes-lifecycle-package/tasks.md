# Tasks: add-kubernetes-lifecycle-package

Worktree: `library/.claude/worktrees/add-kubernetes-lifecycle-package`. Branch:
`feat/add-kubernetes-lifecycle-package`, cut from `origin/main` at `b2d51d7` (library
v1.0.0-beta.6).

Setup:

- Seed `.cue-cache` by copying the main checkout's copy with `cp -a`. Never symlink it.
- Run every command inside the worktree, with an absolute private `TMPDIR` made by `mktemp -d`
  under the session scratchpad.
- Set the registry environment on two lines:

```bash
export CUE_REGISTRY='opmodel.dev=ghcr.io/open-platform-model,testing.opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'
export OPM_REGISTRY="$CUE_REGISTRY"
```

`TestGenerate_BuildsThroughTheKernel` can fail in a full-suite run because of a known
cross-process cache race. Before treating a failure as a finding, rerun
`go test ./opm/helper/platformmodule -count=1` on its own.

Commits:

- No body line starts with `word(`, and no commit carries a bare at-sign.
- The only trailer is `Co-Authored-By: Claude <noreply@anthropic.com>`.
- Comments cite enhancement decisions as `0012:D4:R5`, never as a bare `D4`.
- Messages the package returns carry no enhancement reference.

## 1. The deletion plan and its transition (lifecycle; design LC1, LC2, LC3, LC4, LC6, LC7)

- [x] 1.1 Write `opm/k8s/lifecycle/doc.go`.
      - The package doc covers five points:
        - the plan is built from inventory entries (the persisted inventory for an uninstall, the
          inventory package's stale set for a prune) and is ordered for deletion;
        - the caller owns the state, and it is a JSON value;
        - each call to `Advance` names one action, which the frontend performs with its own
          client before it hands back what it saw;
        - every delete carries Foreground propagation and a UID precondition, and a frontend
          reports a failed precondition as left behind or to retry, never as deleted;
        - the package has no hook semantics.
      - The doc publishes into the Library reference, so it carries no ADR or enhancement
        reference.
      - After the package clause, add a non-doc comment with the maintainer pointers: ADR-008
        rules 1 to 3 and its Deletion plans consequence, ADR-011 item 6, and 0012:D4 (cited once).
      Verify: `go build ./opm/k8s/...` is clean.
- [x] 1.2 Write `opm/k8s/lifecycle/plan.go`.
      - Declare `Policy`, `Step`, `DeletionPlan` (unexported fields) and `NewDeletionPlan`, plus
        the `Steps` (a copy), `Policy`, `OwnerUUID` and `Len` accessors, as in LC1.
      - `NewDeletionPlan` copies the entries and sorts them with `object.Sort(..., object.Descending)`.
        The `gvkOf` function reads the entry's group, version and kind.
      - It marks a step `ownership.SkipSafetyExcluded` where `ownership.SafetyExcluded` holds.
      Verify: `go vet ./opm/k8s/...` is clean.
- [x] 1.3 Write `opm/k8s/lifecycle/advance.go`.
      - Declare `ActionKind` with its four constants, `Action`, `Event`, `Awaiting`, `Result`,
        `FailureClass`, `Outcome` and `State` (with the LC4 JSON tags), and `Advance`.
      - `Advance` follows the LC2 transition. It builds the `ownership.Object` from the step's
        entry, and the `CanDelete` input carries `plan.OwnerUUID()`.
      - It classifies errors with `apierrors.IsNotFound`, `IsForbidden` and `IsConflict`.
      - A `delete` names `metav1.DeletePropagationForeground` and `verdict.Preconditions()`.
      - A live object that answers the wrong step (an identity field it sets differs) records a
        `failed` outcome of class `error` naming both objects, and the next step's action follows.
      - It returns the three LC2 state errors (next out of range, awaiting at the end, unknown
        awaiting). Each one returns the input state unchanged and the zero `Action`.
      - Each call appends only the outcomes of the steps it finished, in step order.
      - It never appends to the input state's `Outcomes` backing array: copy before appending, so
        a caller's earlier state value stays valid.
      Verify: `go vet ./opm/k8s/...` is clean.
- [x] 1.4 Write the tests.
      - `plan_test.go` covers every scenario of "The deletion plan orders inventory entries by
        kind-class delete order". The stale-set scenario calls `inventory.StaleSet` and hands the
        result to `NewDeletionPlan`.
      - `advance_test.go` holds one table-driven transition test with these cases:
        - every scenario of "The deletion transition names one action per call";
        - every scenario of "Action outcomes are classified by the library", using
          `apierrors.NewNotFound`, `NewForbidden`, `NewConflict` and `NewInternalError`, one of
          them wrapped with `%w`;
        - every scenario of "The deletion state is a serialisable value the caller owns";
        - a dry-run run that answers every `delete` with `Event{}`;
        - one call that records both a failure and a safety-excluded skip, asserting the two
          appended outcomes in step order.
      - `TestStateJSONGolden` marshals a state with every field set and compares the bytes with
        the literal in the spec scenario "The JSON encoding is fixed".
      - The JSON round-trip test drives the same plan to done with an in-memory state and with a
        state re-decoded from `json.Marshal` before every call, and asserts equal action sequences
        and final states. The idempotence test calls `Advance` twice on equal inputs at every
        step.
      - A "drive to done" helper asserts the four action kinds, that every non-done action's
        `Step` and `Entry` match `plan.Steps()`, and that `done` names step -1. It also asserts that
        `Advance` did not change the live object (deep-equal against a copy).
      - `purity_test.go` covers both purity scenarios: the direct-imports check with `go list`
        or `go/build`, and a `go/parser` walk for `*ast.GoStmt`.
      Verify: `go test ./opm/k8s/lifecycle -count=1` and
      `go test -race ./opm/k8s/... -count=1` are green.
- [x] 1.5 Run `task check` until green, then commit
      `feat(k8s): add opm/k8s/lifecycle with the deletion plan and its transition`. The body says
      four things: the plan is ordered descending by kind weight, with CRD and Namespace steps
      marked up front; every delete is Foreground with a UID precondition; the library classifies
      the caller's errors; and the state is a caller-owned JSON value.

## 2. The hold verdict (lifecycle; design LC5)

- [x] 2.1 Write `opm/k8s/lifecycle/hold.go`.
      - Declare `Identity` with its three constants, `HoldInput`, `HoldReason` with its seven
        constants (the contract's literals), `HoldVerdict` and `MayReleaseHold`, in the LC5 order.
      - Messages name the reason in plain words and give a count where one exists (for example,
        objects left to retry). They contain no enhancement id, no annotation key and no frontend
        name. The force-orphan and identity messages speak of "the deleting identity", so the cli
        and the operator can each add their own remedy.
      Verify: `go vet ./opm/k8s/...` is clean.
- [x] 2.2 Write `hold_test.go`.
      - One table covers every scenario of "The hold verdict is decided from the policy and the
        plan's outcome".
      - One row per operator `handleDeletion` branch (design Context, branches 1 to 8). Each row
        is named after the branch, so a reader can map it.
      - A reason-literal test.
      - A check that no message contains `0012:`.
      - An end-to-end case drives a plan with `Advance` (one deleted step, one skipped
        `not-opm-managed`, one safety-excluded) and asserts `cleanup-complete`.
      Verify: `go test ./opm/k8s/lifecycle -count=1` is green.
- [x] 2.3 Run `task check` until green, then commit
      `feat(k8s): add the deletion hold verdict to opm/k8s/lifecycle`. The body names the seven
      reasons and says that force-orphan releases only when the deleting identity is missing.

## 3. Docs and whole-tree checks (README, AGENTS, CONSTITUTION, ADR-011)

- [ ] 3.1 Find every edit by its text, not by line number.
      - `README.md` § Helper boundary: the sentence listing the tier's packages gains
        `opm/k8s/lifecycle`, the deletion plan, its transition and the hold verdict. The Layout
        tree gains a `k8s/lifecycle/` row.
      - `AGENTS.md`: the rules line that lists `opm/k8s/` packages (`labels`, `object`, `health`,
        `ownership`, `inventory`) gains `lifecycle`. The layout block gains a `k8s/lifecycle/` row
        listing `NewDeletionPlan` (descending weight, safety exclusions marked up front), `Advance`
        (read, delete with Foreground and a UID precondition, skip, done), the caller-owned JSON
        `State` and `MayReleaseHold` with its seven reasons.
      - `CONSTITUTION.md` Principle III: the sentence "Its packages today are ..." gains
        `opm/k8s/lifecycle` (the deletion plan, its transition and the hold verdict), and the
        trailing clause "the rest arrive in later changes" is dropped, since every package
        ADR-011 item 1 lists now exists.
      - `adr/011-kubernetes-tier-beside-the-kernel.md` Status: append "Amended 2026-10-05 by
        `add-kubernetes-lifecycle-package`: `opm/k8s/lifecycle` holds the deletion plan, built
        from inventory entries in descending kind weight, the one-action transition, with
        Foreground deletes and the UID precondition, the caller-owned JSON state and the hold
        verdict (0012:D4)."
      Verify: `grep -n "k8s/lifecycle" README.md AGENTS.md CONSTITUTION.md adr/011-*.md` shows
      each edit; the layout block is still one code fence; `task docs:bundle:check` is green.
- [ ] 3.2 Run the whole-tree checks on the final code.
      - `task check`.
      - `task api:diff`. Expected: it charges no incompatible entry to this change (base tag
        v1.0.0-beta.6; it lists incompatible changes only).
      - `openspec validate add-kubernetes-lifecycle-package --strict`.
      Then run the consumer check, which is not committed. Clone cli and opm-operator fresh at
      their `origin/main` into the scratch dir, and run `.tasks/consumer-build.sh` against this
      tree for each. Both must build and vet. Record the two heads for the report.
      Verify: every check is green.
- [ ] 3.3 Run `task check` until green, then commit
      `docs(k8s): list opm/k8s/lifecycle in the tier docs`.

## 4. Archive (at PR time)

- [ ] 4.1 First merge `origin/main` into the branch, because lib-h4 may touch the same doc lines.
      Then run `openspec archive add-kubernetes-lifecycle-package --yes` on this branch, so the
      archive rides the implementing PR. If `origin/main` moves after the archive, merge it again
      and re-run the archive step.
      Skip the delivery log; the claim belongs to op-f2 and cli-f2.
      Verify: the `kubernetes-tier` main spec carries the six new requirements, and
      `openspec validate --all --strict` passes.
- [ ] 4.2 Commit `chore(openspec): archive add-kubernetes-lifecycle-package`.
