## 1. k8s: remove the install admission from the ownership verdicts

- [x] 1.1 `opm/k8s/ownership/apply.go`: remove `ApplyInput.Admit` and `admittedForApply`; `CanApply` refuses `foreign-object` on `!opmManaged(in.Live)` alone; its godoc drops "or admitted" and "unless admitted".
- [x] 1.2 `opm/k8s/ownership/delete.go`: remove `DeleteInput.Admit`, `admittedForDelete` and `installDeletable`; `CanDelete` skips `not-opm-managed` on `!opmManaged(in.Live)` alone; its godoc drops "unless admitted".
- [x] 1.3 `opm/k8s/ownership/ownership.go`: remove `carriesNoOtherIdentity`, which only admission used.
- [x] 1.4 `opm/k8s/ownership/apply_test.go` and `delete_test.go`: delete every case that sets `Admit`, and the fixtures only those cases used (`operatorNS`, `clusterRole`, `roleBind`, `clusterRB`, `configMap`, `widget`).
- [x] 1.5 `opm/k8s/labels/labels.go`, `opm/k8s/labels/nowriter_test.go` and `AGENTS.md` (package map): cite 0012:D8 instead of 0012:D8:R6; drop the install admission from the `ownership/` line.
- [x] 1.6 `task check` green, then commit `refactor(k8s)!: remove the install admission from the ownership verdicts` with a `BREAKING CHANGE:` footer naming the removed fields.
