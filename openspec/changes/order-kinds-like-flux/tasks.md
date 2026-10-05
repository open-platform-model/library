## 1. object: order kinds as Flux's staged apply does

Library tests run with an absolute private `TMPDIR`
(`export TMPDIR=$(mktemp -d -p <session scratchpad>)`), and the worktree's `.cue-cache` is a copy
of the main checkout's, never a symlink. Every commit task stages the files it names with
`git add <file>`.

- [ ] 1.1 `opm/k8s/object/flux_order_test.go`: replace the rank-only comparison with Flux
  v0.77.0's comparison (design D3). Keep `fluxFirst` and `fluxLast` as literals; add the stage
  predicates (definition stage by group and kind for CustomResourceDefinition of
  `apiextensions.k8s.io`, core Namespace, ClusterRole of `rbac.authorization.k8s.io`; class stage
  by the case-sensitive `Class` suffix; the rest), `fluxLess(a, b schema.GroupKind)` over stage,
  rank, group, kind, and `contradictions(weight func(schema.GroupVersionKind) int) []string` over
  the universe of design D3, each pair written `<group>/<kind> before <group>/<kind>`. Delete the
  committed list and `TestFluxOrderContradictions`. Add `TestWeightNeverContradictsFlux`
  (`contradictions(Weight)` is empty) and `TestFluxComparisonCatchesAContradiction` (Deployment
  moved below Service reports `apps/Deployment` vs `/Service`; an `autoscaling`
  HorizontalPodAutoscaler weighed above a `batch` Job reports that pair). Rewrite
  `TestFluxOrderDefinitionStage` over `apiextensions.k8s.io/v1 CustomResourceDefinition` and the
  core `v1` Namespace with `fluxLess` and `Weight`, and delete the group-less `libraryWeight`
  helper. Run it once against the old table to confirm it fails, and
  that its failure lists the webhook, class, quota and tie-break pairs.
- [ ] 1.2 `opm/k8s/object/weights.go`: set the values of design D1; add `WeightClass`,
  `WeightResourceQuota` and `WeightLimitRange`; set `WeightStorageClass = WeightClass` and the
  eight constants of kinds Flux does not list to `WeightDefault`. Add table rows for PriorityClass
  (`scheduling.k8s.io/v1`), RuntimeClass (`node.k8s.io/v1`), IngressClass
  (`networking.k8s.io/v1`), ResourceQuota and LimitRange (core `v1`) in the GVK table, and for
  every Flux-listed class kind, ResourceQuota and LimitRange in the kind table. Add the
  group-and-kind step for the three cluster definitions and map their names in the kind table to
  `WeightClass`; add the `Class` suffix step before the default (design D2). Rewrite the file
  header and the `Weight` doc: the table agrees with Flux's staged apply order (0012:D5:R1),
  every kind Flux does not list weighs the default so Flux's tie-break only refines, and the
  lookup order. No exported name is renamed or removed.
- [ ] 1.3 `opm/k8s/object/weights_test.go`: rewrite `TestWeightTableGuard` to pin the new
  constants and both tables; change `TestWeightUnknownVersionFallsBackToKind` to
  `apps/v1beta2 Deployment` (100). Add tests for the new spec scenarios: webhook configurations
  after custom resources (ascending and descending), class kinds after ClusterRoles including a
  custom `EC2NodeClass`, quotas and limits before the workloads, the kinds Flux does not list at
  1000, and a definition kind name in another group (`example.com` Namespace is 6;
  `rbac.authorization.k8s.io/v1beta1 ClusterRole` is 5).
- [ ] 1.4 `opm/k8s/object/sort_test.go`: `TestSortApplyOrder` uses a ConfigMap in place of the
  PersistentVolumeClaim and expects CustomResourceDefinition, Namespace, ConfigMap, Deployment,
  custom resource. Fix any other test in `opm/k8s/object` or `opm/k8s/lifecycle` whose expected
  order moved, and only by the new table (the lifecycle scenario "Entries are ordered for
  deletion" keeps its expected order).
- [ ] 1.5 `opm/k8s/object/stages.go`, `opm/k8s/object/stages_test.go` and `opm/k8s/object/doc.go`: the comments that say the table
  was ported unchanged, or that a frontend submits one stage per Flux call, now say the table
  agrees with Flux's staged apply, so a frontend can hand Flux the whole set or any stage in one
  call. Comment text only; the spec's "Apply stages follow the weight table" carries the same
  rationale through its MODIFIED delta.
- [ ] 1.6 `go test -race -count=1 ./opm/k8s/...` green, then `task check` green. Commit
  `fix(k8s): order kinds as flux's staged apply does`.

## 2. adr, docs: record the new order

- [ ] 2.1 `adr/011-kubernetes-tier-beside-the-kernel.md`: item 1 no longer says the table is
  ported unchanged from the cli; it says the table was ported from the cli's
  `pkg/resourceorder` and then aligned with Flux's staged apply order wherever Flux orders two
  kinds, with every kind Flux leaves to its alphabetical tie-break at one weight, so the
  operator's engine only refines the order the cli submits (0012:D5:R1). Append to Status an
  "Amended <date> by `order-kinds-like-flux`: …" sentence in the form of the existing ones.
- [ ] 2.2 `AGENTS.md` layout line for `object/`: "the kind-class Weight table ported from cli
  pkg/resourceorder" becomes "the kind-class Weight table, aligned with Flux's staged apply
  order". `README.md` needs no edit (it names the table without its origin); confirm.
- [ ] 2.3 Cross-cutting checks: `task api:diff` lists only the "value changed" entries of the
  thirteen moved constants and no removed or renamed name (warn mode); the consumer build
  (`GOTOOLCHAIN=local bash .tasks/consumer-build.sh <consumer-checkout> . <work-dir>`) passes
  against cli and opm-operator `main`. Informational, not a gate: run the cli's
  `go test ./internal/kubernetes/ -run 'Order|Parity'` through the same `go.work` and record in
  the PR body which literals of `order_parity_test.go` the cli's bump must edit.
- [ ] 2.4 `task check` green. Commit `docs(k8s): record that the weight table follows flux`.

## 3. openspec: deltas

- [ ] 3.1 Confirm the `kubernetes-tier` delta: the MODIFIED "One weight table orders apply and
  delete" keeps all three main-spec scenarios by name, the MODIFIED "Apply stages follow the
  weight table" keeps its three, the REMOVED requirement carries Reason and
  Migration, and every new scenario has a test from section 1. Run
  `openspec validate order-kinds-like-flux --strict` green.
- [ ] 3.2 Tick every task in this file and commit
  `chore(openspec): mark order-kinds-like-flux implemented`.
