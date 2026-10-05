## Context

`opm/k8s/object.Weight` is the kind-class order both Kubernetes frontends apply and delete by
(0012:D5). The cli submits objects in that order. The operator hands the whole set to Flux's
`ApplyAllStaged` (`github.com/fluxcd/pkg/ssa` v0.77.0, the version opm-operator pins), and Flux
re-sorts it. 0012:D5:R1 allows that engine to refine the library order and forbids it to contradict
it. Today Flux contradicts the library on 196 kind pairs by rank alone (the list `flux_order_test.go` held at `af2bf84`, the base of this change), and on more once its group-and-kind tie-break is counted, so the
same module reaches the cluster in two different orders depending on the frontend.

What Flux v0.77.0 does, read from `ssa@v0.77.0/manager_apply.go`, `sort.go` and `utils/is.go`:

1. `ApplyAllStaged` cuts the set into stages and runs one `ApplyAll` per stage:
   - the definition stage: CustomResourceDefinition (apiVersion `apiextensions.k8s.io/*`), the
     core Namespace (apiVersion exactly `v1`) and ClusterRole (apiVersion
     `rbac.authorization.k8s.io/*`); Flux waits for them to be ready;
   - the class stage: every object whose kind ends in `Class`, custom kinds included; Flux waits
     for them too;
   - the custom stage (`ApplyOptions.CustomStageKinds`; the operator sets none);
   - the rest.
2. `ApplyAll` sorts its stage with `SortableUnstructureds`. The comparison is the
   `ReconcileOrder` rank of the kind name (the `First` list ranks `-23..-1`, the `Last` list
   `1..2`, every other kind 0), then the API group, then the kind, then namespace and name. Two
   kinds of equal rank (in practice two kinds Flux does not list) are therefore ordered
   alphabetically by group and then kind. The dry-runs run
   concurrently; the applies run one by one in sorted order.

`First` is CustomResourceDefinition, Namespace, ClusterRole, ClusterClass, RuntimeClass,
PriorityClass, StorageClass, VolumeSnapshotClass, IngressClass, GatewayClass, ClusterRoleBinding,
ResourceQuota, ServiceAccount, Role, RoleBinding, ConfigMap, Secret, Service, LimitRange,
Deployment, StatefulSet, CronJob, PodDisruptionBudget. `Last` is MutatingWebhookConfiguration,
ValidatingWebhookConfiguration.

## Goals / Non-Goals

**Goals:**

- No pair of objects that Flux's staged apply orders A before B is ordered B before A by
  `object.Weight`, for any group and kind, so Flux only ever breaks the library's ties.
- A test fails on any such pair, so a later table edit or a Flux bump that reorders kinds is a
  reviewed change.
- No exported name is renamed or removed.

**Non-Goals:**

- Changing `Stages`. It already cuts one stage per weight after the definitions; with the new
  table its stages line up with Flux's.
- Modelling `CustomStageKinds`. No frontend sets it; a frontend that does owns that order.
- Changing the operator. It keeps `ApplyAllStaged`; a parity test there is its own change.
- Changing how the cli waits between stages.

## Decisions

### D1. The table is Flux's order, with ties wherever Flux falls back to the alphabet

New weights (the constant names of existing kinds are unchanged; `*` marks a new constant):

| Weight | Kinds |
| ---: | --- |
| -100 | CustomResourceDefinition (`WeightCRD`) |
| 0 | Namespace (`WeightNamespace`) |
| 5 | ClusterRole (`WeightClusterRole`) |
| 6 | PriorityClass, RuntimeClass, StorageClass, VolumeSnapshotClass, IngressClass, GatewayClass, ClusterClass, any kind ending in `Class` (`WeightClass`*, `WeightStorageClass`) |
| 7 | ClusterRoleBinding (`WeightClusterRoleBinding`) |
| 8 | ResourceQuota (`WeightResourceQuota`*) |
| 10 | ServiceAccount, Role, RoleBinding |
| 15 | Secret, ConfigMap |
| 50 | Service |
| 60 | LimitRange (`WeightLimitRange`*) |
| 100 | Deployment, StatefulSet |
| 105 | CronJob (`WeightCronJob`) |
| 108 | PodDisruptionBudget (`WeightPDB`) |
| 1000 | every kind Flux does not list: PersistentVolume, PersistentVolumeClaim, DaemonSet, ReplicaSet, Job, Ingress, NetworkPolicy, HorizontalPodAutoscaler, VerticalPodAutoscaler, custom resources (`WeightDefault`; `WeightPersistentVolume`, `WeightPVC`, `WeightDaemonSet`, `WeightJob`, `WeightIngress`, `WeightNetworkPolicy`, `WeightHPA`, `WeightVPA` take this value) |
| 2000 | MutatingWebhookConfiguration, ValidatingWebhookConfiguration (`WeightWebhook`) |

Where Flux ranks two kinds apart, the library weighs them the same way round or equal: a tie is
not a contradiction, so ConfigMap and Secret, Role and RoleBinding, the class kinds among
themselves, and the mutating and validating webhook configurations share a weight although Flux
ranks them apart. Only kinds Flux does not list are ordered by its group-and-kind fallback; the
library weighs them all equal, so that fallback refines and cannot contradict.

### D2. Lookup mirrors Flux's stage predicates

```go
func Weight(gvk schema.GroupVersionKind) int {
	if w, ok := gvkWeights[gvk]; ok { // exact group, version, kind
		return w
	}
	if w, ok := definitionWeights[gvk.GroupKind()]; ok { // CRD, core Namespace, rbac ClusterRole, any version
		return w
	}
	if w, ok := kindWeights[gvk.Kind]; ok { // kind alone, any group
		return w
	}
	if strings.HasSuffix(gvk.Kind, "Class") { // Flux's class stage
		return WeightClass
	}
	return WeightDefault
}
```

`kindWeights` maps the names CustomResourceDefinition, Namespace and ClusterRole to
`WeightClass`. A kind with one of those names in another group is not in Flux's definition stage:
Flux applies it in the last stage, first by rank, so after every class kind and before
ClusterRoleBinding. Weight 6 ties it with the classes and keeps it before 7, which is the only
place that contradicts nothing. Its weight today is the definition's own (-100, 0 or 5), which
puts it ahead of things Flux applies first.

The core Namespace is matched on group and kind in any version, as `Stages` already matches it
for its definition stage. Flux's `IsNamespace` also requires apiVersion exactly `v1`, so a core
Namespace of another version would be in Flux's last stage while the library weighs it 0. That is
a deliberate gap: no such version exists, and keying the weight on `v1` alone would make `Weight`
and the definition stage of `Stages` disagree with each other. The guard compares group and kind,
so it does not see the gap.

The `Class` suffix rule is case-sensitive, as Flux's `strings.HasSuffix(kind, "Class")` is. It
sends a custom class kind (Karpenter's `EC2NodeClass`, for example) to weight 6, where Flux applies
it, instead of 1000 after the workloads.

### D3. The guard models Flux's comparison, not just the rank

`flux_order_test.go` keeps `ReconcileOrder` as literals (depguard keeps Flux out of the library) and
adds the stage predicates. `fluxLess(a, b schema.GroupKind)` compares stage, then rank, then group,
then kind. `contradictions(weight func(schema.GroupVersionKind) int)` returns every pair with
`fluxLess(a, b)` and `weight(a) > weight(b)`. Its universe:

- every GVK in the library's GVK table, and every kind in its kind table in the kind's canonical
  group;
- every `ReconcileOrder` kind in its canonical group;
- a same-named copy of each of those kinds in group `a.example` (sorts before `apps`) and
  `zz.example` (sorts after every built-in group);
- custom kinds `Widget` in `a.example` and `zz.example`, and `WidgetClass` in `a.example`.

`TestWeightNeverContradictsFlux` asserts `contradictions(Weight)` is empty.
`TestFluxComparisonCatchesAContradiction` passes a weight function that moves Deployment below
Service and asserts the pair is reported, so the guard cannot pass vacuously; a second case weighs
an `autoscaling` HorizontalPodAutoscaler above a `batch` Job and asserts that pair is reported, so
the tie-break is proven to count.
`TestFluxOrderDefinitionStage` is rewritten over `apiextensions.k8s.io/v1 CustomResourceDefinition`
and the core `v1` Namespace with `fluxLess` and `Weight`; the group-less `libraryWeight` helper
goes, since a group-less CustomResourceDefinition is no longer a definition.

### D4. Delete order follows without code

`Sort(…, Descending)` and `lifecycle.NewDeletionPlan` read the same table. Webhook configurations
are now deleted first, which on this point matches Helm 4's uninstall order (Helm agrees with
Flux only in part; see "Helm as partial corroboration") and stops a webhook from blocking
the deletion of the objects it guards. Class kinds are deleted after everything that names them.

## Research & Decisions

### Which pairs count as "Flux defines an order"

**Context**: Flux's comparison is total: two kinds it does not list are still ordered, by API group
and then kind. The library could treat that tie-break as an order to agree with, or ignore it.

**Explored**: `ssa@v0.77.0/sort.go` (`IsLessThan`, which falls back to group and then kind when two
ranks are equal, and `computeKind2index`, whose comment on the `ReconcileOrder` list says that in
some cases the order is specified just to provide determinism); the old `flux_order_test.go`, which counted rank
differences only; 0012:D5:R1 ("An engine's own staging may refine it within a stage and never
contradicts it"). The operator applies the whole set in one `ApplyAllStaged`, so every unlisted
kind meets every other one inside one Flux sort.

**Decision**: The guard compares Flux's full comparison. Every kind Flux does not list weighs
`WeightDefault`.

**Rationale**: With the unlisted kinds spread over several weights, Flux's alphabetical order
reverses some of them for any group name (an `autoscaling` HorizontalPodAutoscaler before a
`batch` Job; a custom resource of group `acme.io` before an `apps` DaemonSet). That reorders
objects across two library stages, which 0012:D5:R1 forbids. Only equal weights leave the
tie-break as a refinement. The library loses its own grading among those kinds (volumes before
workloads, Ingresses after workloads, autoscalers last); Kubernetes' eventual consistency covers
each of them (a Pod waits for its claim, an autoscaler retries its target), as 0012:D5 already
relies on. The narrower reading, comparing ranks only, would keep that grading but leave the
operator applying those kinds in an order the library forbids.

### Helm as partial corroboration

**Context**: Helm's install order is the other widely used kind order.

**Explored**: `InstallOrder` in Helm v3.19.0 (`pkg/releaseutil/kind_sorter.go`) and Helm v4.0.0
(`pkg/release/v1/util/kind_sorter.go`).

**Decision**: Cite Helm as partial corroboration only, where it agrees with Flux; do not follow
it where it does not.

**Rationale**: Helm agrees with Flux, and so with the new table, that PriorityClass, ResourceQuota
and LimitRange come before the workloads, that PodDisruptionBudget comes before volumes,
DaemonSets, ReplicaSets and Jobs, that CronJob comes after Deployment and StatefulSet, and (from
v4.0.0, which lists them last) that webhook configurations come after every other kind Helm lists.
Helm disagrees with Flux elsewhere: it applies StorageClass after Secrets and ConfigMaps,
IngressClass just before Ingress, PodDisruptionBudget, volumes and DaemonSets before Deployments,
and any kind it does not list, custom resources included, after the webhook configurations. The operator's engine is Flux, so Flux decides; Helm only shows the moves
are not Flux idiosyncrasies.

### Numbers

**Context**: Values could be renumbered from scratch or changed only where needed.

**Decision**: Keep every value that already agrees (-100, 0, 5, 10, 15, 50, 100, 1000); fit the new
ranks into the gaps.

**Rationale**: Fewer moved constants means fewer `api:diff` entries and a cli parity diff that shows
only real moves.

## Risks / Trade-offs

- [Thirteen constants change value; `task api:diff` lists them as incompatible] → warn mode on the
  beta base; no consumer's non-test code reads the numbers; the release class is `feat!` with a
  `BREAKING CHANGE:` footer that names each old and new value (ADR-010).
- [PersistentVolumeClaims now follow Deployments in the cli] → a Pod stays Pending until its claim
  exists and then starts; the operator already applies in this order.
- [A NetworkPolicy still lands after the Pods it selects] → unchanged from today (it weighed 150,
  after the workloads) and the same as Flux; a default-deny policy that must exist first is
  module-internal ordering, which 0012:D5 leaves to a future Bundle.
- [A Flux bump changes `ReconcileOrder` or the stages] → the literals in `flux_order_test.go` name
  the version, and nothing in the library follows the operator's pin. The operator's next change
  adds a parity test that compares Flux's own staged order with `object.Weight`; it fails on such a
  bump, and the literals are then updated here.
- [The cli's `order_parity_test.go` fails on the bump] → by design; the bump PR edits it.
- [Delete order moves too: PersistentVolumeClaims, PersistentVolumes, Jobs, DaemonSets, Ingresses
  and autoscalers are now deleted before Deployments, StatefulSets and PodDisruptionBudgets] → a
  claim then sits Terminating under `pvc-protection` until its Pods go. The cli's delete does not
  wait between stages, so this only shows in its output.
