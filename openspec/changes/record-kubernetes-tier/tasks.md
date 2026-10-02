# Tasks: record-kubernetes-tier

No Go code, no `go.mod` change. Every section's gate is `task fmt`, `task vet`, `task lint`, `task build` and `openspec validate record-kubernetes-tier --strict`. `task test` is optional for a change that touches no Go source except one package doc comment.

## 1. Record the decision (adr)

- [x] 1.1 Write `adr/011-kubernetes-tier-beside-the-kernel.md` in the `adr/TEMPLATE.md` shape, Status Accepted (2026-10-02). Context is value-neutral (0012:D1/D2, the two existing tiers, 0006:D31's outcome, the apply-engine asymmetry, CUE label stamping). Decision states the nine rules mapped to 0012:D3/D4/D5/D6. Rejected alternatives: inside `opm/kernel`, under `opm/helper`, a nested Go module, per-frontend copies. Consequences are bold-labelled paragraphs. Verify: no bare decision number (`grep -nE '(^|[^:0-9A-Za-z])D[0-9]+' adr/011-*.md` returns nothing).
- [x] 1.2 Amend `adr/008-kernel-plans-caller-runs.md` in place, following the ADR-007 precedent: one dated "Amended 2026-10-02 by `record-kubernetes-tier`" sentence on the Status line, and a one-line clarification at the end of rule 4 ("derives no ordering of its own" excludes module-specific ordering only; kind-class order lives in `opm/k8s/object`). Verify: the rule text is otherwise byte-identical (`git diff` shows only those two lines).
- [x] 1.3 Gates green, then commit `docs(adr): add ADR-011, the Kubernetes tier beside the kernel`.

## 2. Amend the constitution and the boundary prose (constitution, helper)

- [ ] 2.1 `CONSTITUTION.md`: Principle III lists `opm/catalog/` and the planned `opm/k8s/` tier (mandatory for a Kubernetes frontend, fenced from the kernel, ADR-011), and its summary row names the three tiers. Principle IV's runtime-concerns bullet is narrowed: `opm/k8s/` alone may import `k8s.io/apimachinery`, and no `opm/` package imports `k8s.io/client-go`, `sigs.k8s.io/controller-runtime` or Flux. Principle I gains one sentence saying that Kubernetes vocabulary in `opm/k8s/` is not a runtime assumption. The constitution has no version or amendment log, so none is added.
- [ ] 2.2 Mirror 2.1 in the `context` block of `openspec/config.yaml` (Principles I, III and IV). Verify: `openspec validate record-kubernetes-tier --strict` still loads the config.
- [ ] 2.3 `opm/helper/doc.go` (doc comment only): replace "Anything outside opm/helper/ is part of the kernel contract that every frontend ... MUST honour" with the two-tier statement (the kernel binds every frontend; `opm/k8s/` binds every frontend that targets Kubernetes), and add `opm/catalog` to the package list in the import-graph paragraph. Verify: `go doc ./opm/helper` renders, and `git diff opm/helper/doc.go` touches comment lines only.
- [ ] 2.4 `README.md` § Helper boundary and `AGENTS.md` § Repository Rules: the same correction. `AGENTS.md` § Repository Layout gains a `k8s/` line marked PLANNED (ADR-011, no package yet). Verify: `grep -rn "outside .*helper.* kernel contract\|outside \`helper/\` is kernel" AGENTS.md README.md opm` returns nothing.
- [ ] 2.5 Gates green, then commit `docs(constitution): admit the opm/k8s tier beside the kernel`.

## 3. Fence the tier in lint (.golangci.yml)

- [ ] 3.1 Add depguard rules `nothing-imports-k8s-tier` (files: `opm/kernel`, `opm/module`, `opm/platform`, `opm/catalog`, `opm/schema`, `opm/errors`, `opm/internal`, `opm/helper`, test files included; deny `.../opm/k8s`) and `k8s-tier-imports-no-runtime` (files: `opm/k8s`; deny `k8s.io/client-go`, `sigs.k8s.io/controller-runtime`, `github.com/fluxcd`, `.../opm/internal`, `.../opm/helper`), commented in the style of `kernel-never-imports-helper`. Add `opm/catalog` to `kernel-never-imports-helper`'s file list, which the MODIFIED `helper-packages` requirement now states it covers.
- [ ] 3.2 Prove both rules bite in a throwaway copy of the tree under the scratchpad, never committed: (a) a probe package `opm/k8s/probe` and an import of it from a file in `opm/kernel` make `golangci-lint run` fail on `nothing-imports-k8s-tier`; (b) a file in `opm/k8s/probe` importing a package under `opm/internal/`, then one under `opm/helper/`, makes it fail on `k8s-tier-imports-no-runtime`; (c) the same for a `k8s.io/client-go` import, with the dependency added to the scratch copy's `go.mod` only. Verify: each probe fails with the rule name, and the worktree has no `opm/k8s` directory afterwards.
- [ ] 3.3 Gates green, then commit `chore(lint): fence the opm/k8s tier with depguard`.
