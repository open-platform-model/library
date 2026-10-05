## Why

0012:D5:R1 says the cli and the operator order the same objects identically, and that an apply
engine's own staging may refine that order within a stage but never contradict it. The operator
applies through Flux's `ApplyAllStaged` (`github.com/fluxcd/pkg/ssa` v0.77.0) over the whole set,
so Flux's order is the order the operator's objects reach the cluster in. The library's weight
table in `opm/k8s/object` was ported unchanged from the cli's `pkg/resourceorder`, and it
disagrees with Flux on many pairs. `opm/k8s/object/flux_order_test.go` records them:

- Mutating and validating webhook configurations weigh 500, so the library applies them before
  custom resources (1000). Flux applies them last. Applied first, a webhook with
  `failurePolicy: Fail` whose backend is not running yet refuses every later object it matches,
  which breaks first installs of modules like cert-manager.
- The cluster-scoped class kinds (PriorityClass, RuntimeClass, IngressClass, GatewayClass,
  ClusterClass, VolumeSnapshotClass), ResourceQuota and LimitRange fall to the default 1000, after
  the workloads that name them. Flux applies them early: the class kinds in a stage of their own
  that it waits on, ResourceQuota before ServiceAccounts and LimitRange before Deployments.
- StorageClass (20) comes after RBAC, ServiceAccounts, Secrets and ConfigMaps; Flux applies it
  with the other classes, right after ClusterRoles.
- PodDisruptionBudget (200) and CronJob (110) come after workloads, Jobs, Ingresses and volumes;
  Flux puts CronJob right after StatefulSet and PodDisruptionBudget right after CronJob.
- Every kind Flux does not list (PersistentVolume, PersistentVolumeClaim, DaemonSet, ReplicaSet,
  Job, Ingress, NetworkPolicy, the autoscalers, every custom resource) is one group to Flux, after
  all the kinds it lists and before the webhooks. Among themselves Flux orders them by API group
  and then by kind name, an alphabetical fallback. The library spreads those kinds over weights 20
  to 1000, so
  Flux's alphabetical order among them contradicts the library wherever the groups sort the other
  way (an `autoscaling` HorizontalPodAutoscaler before a `batch` Job, a custom resource of group
  `acme.io` before an `apps` DaemonSet).

The cli applies by the library table, so today the two frontends apply one module in different
orders. This change makes the table agree with Flux's staged apply order on every pair Flux orders,
so Flux only ever refines the library order.

## What Changes

- **The weight table follows Flux's staged apply.** New values in `opm/k8s/object`:
  CustomResourceDefinition -100, Namespace 0, ClusterRole 5, every class kind 6 (new
  `WeightClass`; `WeightStorageClass` becomes 6), ClusterRoleBinding 7, ResourceQuota 8 (new
  `WeightResourceQuota`), ServiceAccount, Role and RoleBinding 10, Secret and ConfigMap 15,
  Service 50, LimitRange 60 (new `WeightLimitRange`), Deployment and StatefulSet 100, CronJob 105,
  PodDisruptionBudget 108, every kind Flux does not list 1000 (`WeightDefault`), mutating and
  validating webhook configurations 2000. The constants of the kinds Flux does not list
  (`WeightPersistentVolume`, `WeightPVC`, `WeightDaemonSet`, `WeightJob`, `WeightIngress`,
  `WeightNetworkPolicy`, `WeightHPA`, `WeightVPA`) stay exported and take the value of
  `WeightDefault`.
- **Lookup follows Flux's stages.** `Weight` looks a GVK up by exact group, version and kind, then
  the three cluster definitions (CustomResourceDefinition of `apiextensions.k8s.io`, the core
  Namespace, ClusterRole of `rbac.authorization.k8s.io`) by group and kind in any version, then by
  kind alone, then any kind whose name ends in `Class` weighs `WeightClass` (Flux stages every such
  kind, custom ones included), else `WeightDefault`. A kind named CustomResourceDefinition,
  Namespace or ClusterRole in another group weighs `WeightClass`: Flux applies it in its last
  stage, ahead of ClusterRoleBinding.
- **`flux_order_test.go` becomes a guard.** It models Flux v0.77.0's staged apply as Flux's own
  comparison does (stage, then `ReconcileOrder` rank, then group, then kind) over the library's
  table, Flux's listed kinds, a same-named kind in another group for each, custom kinds in groups
  that sort before and after the built-in ones, and a custom class kind. It fails on any pair the
  library orders strictly opposite to Flux. A unit test proves the comparison catches an injected
  contradiction.
- **Delete order is the reverse and follows.** `Sort(…, Descending)` and `lifecycle.NewDeletionPlan`
  now delete webhook configurations first and the class kinds just before ClusterRoles, Namespaces
  and CustomResourceDefinitions.
- **Records.** ADR-011 item 1 and its Status record that the table no longer matches the cli's
  ported values and why. The `opm/k8s/object` docs and the `AGENTS.md` layout line say the table
  agrees with Flux's staged apply. The kubernetes-tier spec replaces the "differences are recorded"
  requirement with "the table never contradicts Flux's staged apply order", and its "Apply stages
  follow the weight table" now says a frontend may hand Flux the whole set or any one stage, because
  the table never contradicts Flux.

Release note (for this PR body and for the changelog entry of the cli PR that takes this release;
release-please writes only a fix commit's title into the library CHANGELOG): the kind-class apply
order now agrees with Flux's
staged apply. Moved kinds: MutatingWebhookConfiguration and ValidatingWebhookConfiguration (now
after custom resources); PriorityClass, RuntimeClass, IngressClass, GatewayClass, ClusterClass,
VolumeSnapshotClass and StorageClass (now right after ClusterRoles, as is any kind whose name ends
in `Class`); ClusterRoleBinding (now after the class kinds); ResourceQuota (now before
ServiceAccounts); LimitRange (now after Services, before Deployments); CronJob and
PodDisruptionBudget (now right after StatefulSets); PersistentVolume, PersistentVolumeClaim,
DaemonSet, ReplicaSet, Job, Ingress, NetworkPolicy and the pod autoscalers (now with the custom
resources, after PodDisruptionBudgets); a kind named CustomResourceDefinition, Namespace or
ClusterRole outside its canonical group (now with the class kinds). Delete order is the reverse. A frontend that applies,
prunes or deletes through `opm/k8s/object` or `opm/k8s/lifecycle` (the cli) changes its order when
it takes this release.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `kubernetes-tier`: the weight table's values and lookup change so they agree with Flux's staged
  apply order; the recorded-differences requirement becomes a no-contradiction guard.

## Impact

- Packages: `opm/k8s/object` (`weights.go`, `doc.go`, `stages.go` comment) and its tests
  (`weights_test.go`, `flux_order_test.go`, `sort_test.go`). `opm/k8s/lifecycle` orders by
  `object.Sort` and needs no code change; its tests are re-run.
- Public API: additive. New constants `WeightClass`, `WeightResourceQuota` and `WeightLimitRange`;
  no exported name is renamed or removed. The values of thirteen existing constants change, which
  `task api:diff` lists as "value changed" entries (warn mode on the beta base). No consumer's
  non-test code reads a weight's number: the cli calls `object.Weight` and `object.Sort`, the
  operator calls neither.
- api:diff advice: the job summary says each of those thirteen "value changed" warnings needs a
  `feat!` commit with a `BREAKING CHANGE:` footer (ADR-010). This change deliberately keeps the
  release class `fix` against that advice: the constants must track the table, keeping their old
  values would make them state an order the library no longer applies, and their only contract is
  the relative order. For the same reason these warnings are not counted as a breaking change for
  the three quiet betas of 0021:D8:R14; the owner confirms that reading on the PR.
- SemVer: PATCH, release class `fix`. The values are ranks whose only contract is the relative
  order, and the order is corrected to the one the operator's engine already applies by and
  0012:D5:R1 requires of both frontends.
- cli: its apply, prune, delete, `tree` and manifest output order change on the bump.
  `internal/kubernetes/order_parity_test.go` pins the retired literals and fails on the bump PR by
  design ("an order change is a reviewed edit in both repositories"); that PR edits it and carries
  the release-note line above in its changelog.
- opm-operator: nothing changes at its `main` today. It applies through Flux, which orders the set,
  and it calls neither `object.Sort` nor `opm/k8s/lifecycle`. A parity test there can assert that
  Flux's comparison never contradicts `object.Weight`.
- `enhancement.yaml` links this change to enhancement 0012 like the earlier library slices of it,
  with no decision claimed: 0012:D5 is delivered only once both frontends adopt the Kubernetes tier
  (ADR-011 item 3), and this change corrects the table they will adopt.
