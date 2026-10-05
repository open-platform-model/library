# Tasks: add-kubernetes-ownership-package

Worktree `library/.claude/worktrees/add-kubernetes-ownership-package`, branch
`feat/add-kubernetes-ownership-package` (from `origin/main` `ca7c56b`). Seed `.cue-cache` by
copying the main checkout's (`cp -a`), never by symlinking. Every command runs inside the
worktree with an absolute private `TMPDIR` from `mktemp -d` under the session scratchpad. The
registry env goes on two lines:

```bash
export CUE_REGISTRY='opmodel.dev=ghcr.io/open-platform-model,testing.opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'
export OPM_REGISTRY="$CUE_REGISTRY"
```

`TestGenerate_BuildsThroughTheKernel` can fail in a full-suite run because of a known
cross-process cache race. Rerun `go test ./opm/helper/platformmodule -count=1` on its own before
treating it as a finding. Commit bodies never start a line with `word(` and carry no bare
at-sign. The only trailer is `Co-Authored-By: Claude <noreply@anthropic.com>`. Comments cite
enhancement decisions as `0012:D8:R5`, never a bare `D8`. Messages a verdict returns carry no
enhancement reference.

## 1. The adopt annotation key (labels; design OW1)

- [x] 1.1 `opm/k8s/labels/labels.go`: add `AnnotationAdopt = "opmodel.dev/adopt"` in its own
      `const` block after the labels. Its doc says three things: it is an annotation a user sets
      on an existing live object; its value is the adopting instance's
      `module-instance.opmodel.dev/uuid`; and no OPM runtime writes it (0012:D8:R6, cited once).
      `doc.go`: the package names the labels and this one annotation, and still stamps nothing.
      Verify: `go build ./opm/k8s/...` clean.
- [x] 1.2 `labels_test.go`: a new `TestAdoptAnnotationKey` pins `AnnotationAdopt` to its
      literal (scenario "The key is the fixed literal"). `TestVocabularyLiterals` stays as it is:
      its table is the frontend-parity table, and neither frontend has an adopt key. A
      `TestNoLibraryCodeSetsTheAdoptAnnotation` parses every non-test Go file under `opm/` and
      fails if a file outside `opm/k8s/labels` spells `opmodel.dev/adopt` or any file calls a
      `SetAnnotations` method (scenario "No library code sets the adopt annotation"). Verify:
      `go test ./opm/k8s/labels -count=1` green.
- [x] 1.3 `task check` green, then commit
      `feat(k8s): add the adopt annotation key to opm/k8s/labels`.

## 2. opm/k8s/ownership: SafetyExcluded and the delete verdict (design OW2, OW5, OW6, OW7, OW8)

- [x] 2.1 `opm/k8s/ownership/doc.go`. The package doc covers four things:
      - pure verdicts over inputs the caller reads with its own client, with no cluster I/O,
        clock or logging, stated in its own words with no ADR or other maintainer pointer (the
        doc publishes into the Library reference);
      - the frontend obligation, which is to consult a verdict on every apply, prune and delete
        path (0012:D4:R1/R2);
      - that a refusal or skip message is the library's wording for both frontends;
      - that a frontend reports a failed DELETE precondition as left behind or to retry, never
        as deleted.
- [x] 2.2 `opm/k8s/ownership/ownership.go`: `Object` and its `String`
      (`Kind/namespace/name`, or `Kind/name` when cluster-scoped), and `SafetyExcluded(group,
      kind)` with group and kind constants for the two protected kinds. `delete.go`:
      `SkipReason` and its four constants, `DeleteInput`, `DeleteVerdict` (`Proceed`,
      `Preconditions` returning `*metav1.Preconditions` with the UID only, nil unless the verdict
      proceeds with a non-empty UID), and `CanDelete` in the OW5 order with the OW4 admission
      (lifts `not-opm-managed` only, only when the live UUID label is empty or equals
      `InstanceUUID`, and only for `apps` Deployment and `rbac.authorization.k8s.io`
      RoleBinding and ClusterRoleBinding). It reads the managed-by and UUID label keys and `IsOPMManagedBy`
      from `opm/k8s/labels`, never a literal. Verify: `go vet ./opm/k8s/...` clean.
- [x] 2.3 Tests:
      - `ownership_test.go`: the `SafetyExcluded` table, both scenarios of "Safety-excluded
        kinds match on group and kind", plus `Kind` and `Kind/namespace/name` strings for
        `Object`.
      - `delete_test.go`: one table over every scenario of "The delete verdict skips with a
        reason or proceeds with the judged object's identity". Also the three delete scenarios of
        "The operator install admission lifts only a proven object's ownership refusal", and an
        admitted object whose UUID label names another instance (scenario "An admitted object
        carrying another identity is not deleted"). Then the reason literals (half of scenario "Reasons are the
        contract's literals"), the `Preconditions` result (UID set, `ResourceVersion` nil; nil
        on a skip and on an empty live UID), the admitted ConfigMap and custom resource that
        still skip, and a deep-equal check that `Live` is unchanged after each case (scenario
        "The live object is not modified"). A `TestImportsNoClockEnvOrLogger` lists the
        package's direct imports with `go/build` (or `go list`) and fails on `os`, `time`,
        `log` or `log/slog` (scenario "The package imports no clock, environment or logger").

      Verify: `go test ./opm/k8s/ownership -count=1` green.
- [x] 2.4 `task check` green, then commit
      `feat(k8s): add opm/k8s/ownership with the delete verdict`. The body says that a proceed
      verdict carries the judged UID and resourceVersion, and that the precondition is UID
      only.

## 3. The apply verdict and the adopt override (ownership; design OW3, OW4, OW6)

- [x] 3.1 `opm/k8s/ownership/apply.go`: `ApplyRefusal` and its three constants, `ApplyInput`,
      `ApplyVerdict` (`Allowed`) and `CanApply` in the OW3 order. Admission lifts
      `foreign-object` only, under the OW4 UUID condition. Messages are worded as in OW6. The
      ownership refusals end with the remedy `annotate it opmodel.dev/adopt=<instance uuid>`,
      built from `labels.AnnotationAdopt`; the `other-instance` remedy first says to remove the
      object from module instance `<other uuid>`. A blank (empty or whitespace) annotation
      counts as none. A wrong-valued annotation adds that it names another instance. An empty
      `InstanceUUID` gets no remedy. Verify: `go vet ./opm/k8s/...` clean.
- [x] 3.2 `apply_test.go`: one table covering every scenario of "The apply verdict refuses
      terminating, foreign and other-instance objects" and "The adopt annotation is the only
      override and the refusal names it". It also covers the three apply scenarios of "The
      operator install admission lifts only a proven object's ownership refusal". The proven
      earlier-manifest case is the cli migration's shape: `Namespace/opm-operator-system` with
      `app.kubernetes.io/managed-by: kustomize`, no identity label, admitted, outside the
      inventory, which applies. Its admitted ClusterRole twin with no labels at all also
      applies. An inventoried object whose adopt annotation names another instance applies,
      and a blank annotation row refuses `foreign-object` without the "names another
      instance" clause. Each refusal message is asserted to contain the object path, and for the two
      ownership refusals the key and the UUID. No message contains `--`, `force` or an
      enhancement id (`0012:`). Add the three refusal literals to the reason test, and the
      deep-equal `Live` check. Verify: `go test ./opm/k8s/ownership -count=1` green;
      `go test -race ./opm/k8s/... -count=1` green.
- [x] 3.3 `task check` green, then commit
      `feat(k8s): add the apply verdict with the adopt override`. The body names the three
      refusals, the adopt key, and that admission lifts only `foreign-object`.

## 4. Docs and whole-tree checks (README, AGENTS, CONSTITUTION)

- [x] 4.1 Locate every edit by text, not by line number:
      - `README.md` § Helper boundary: the sentence listing the tier's packages gains
        `opm/k8s/ownership`, the apply and delete verdicts. The Layout tree gains a
        `k8s/ownership/` row.
      - `AGENTS.md`: the rules line `opm/k8s/` (`labels`, `object`) gains `ownership`. The
        layout block gains a `k8s/ownership/` row (SafetyExcluded, CanDelete with UID/RV and a
        UID precondition, CanApply with terminating, foreign-object and other-instance, the
        adopt override, and the install admission). The `labels/` row mentions
        `AnnotationAdopt`.
      - `CONSTITUTION.md` Principle III: "Its packages today are ..." gains `opm/k8s/ownership`
        (the apply and delete ownership verdicts).

      Verify: `grep -n "opm/k8s/ownership\|k8s/ownership" README.md AGENTS.md CONSTITUTION.md`
      shows each edit; the layout block is still one code fence; `task docs:bundle:check`
      green.
- [x] 4.2 Whole-tree checks on the final code:
      - `task check`;
      - `task api:diff` against `origin/main`: it reports only additions, in `opm/k8s/labels`
        and `opm/k8s/ownership`;
      - `openspec validate add-kubernetes-ownership-package --strict`.

      Consumer check, not committed: clone cli and opm-operator fresh at their `origin/main`
      into the scratch dir, and run `.tasks/consumer-build.sh` against this tree for each. Both
      must build and vet. Verify: both green; record the two heads for the report.
- [x] 4.3 `task check` green, then commit
      `docs(k8s): list opm/k8s/ownership in the tier docs`.

## 5. Archive (at PR time)

- [ ] 5.1 Merge `origin/main` into the branch first (lib-e3 and lib-f5 append to the same
      `kubernetes-tier` main spec and doc lines), then
      `openspec archive add-kubernetes-ownership-package --yes` on this branch, so the archive
      rides the implementing PR. If `origin/main` moves after the archive, merge it and re-run
      the archive step. Verify: the `kubernetes-tier` main spec carries the new requirements,
      and `openspec validate --all --strict` passes. Skip the delivery log; the claim belongs
      to op-e4 and cli-e4.
- [ ] 5.2 Commit `chore(openspec): archive add-kubernetes-ownership-package`.
