# kubernetes-tier Specification

## Purpose
The Kubernetes tier `opm/k8s/` beside the kernel (ADR-011, enhancement 0012:D3 to 0012:D6): where the Kubernetes decisions OPM makes live, how lint fences the tier from the kernel and from cluster runtimes, what it owns for apply and delete, and that every frontend targeting Kubernetes must use it.

## Requirements

### Requirement: The Kubernetes tier lives at opm/k8s beside the kernel

The library SHALL place the Kubernetes decisions OPM makes under `opm/k8s/`, a tier outside the kernel and outside the opt-in helper tier. The tier holds inventory entry construction, stale-set computation, digests, the prune and ownership guards at apply and delete time, deletion ordering, the deletion hold protocol, Compiled-to-object conversion, the label vocabulary, kind-class apply order and readiness evaluation. No package under `opm/kernel/` or `opm/helper/` SHALL implement any of these decisions once the `opm/k8s/` package that owns it exists. The tier SHALL be part of the library's Go module, `github.com/open-platform-model/library`, and SHALL NOT carry a `go.mod` of its own. ADR-011 records the placement. Source: 0012:D3.

#### Scenario: The placement is recorded before any package exists

- **WHEN** a developer reads `adr/011-kubernetes-tier-beside-the-kernel.md`
- **THEN** it names `opm/k8s/*` as the Kubernetes tier, states that the tier is outside `opm/kernel` and outside `opm/helper`, and names placement inside the kernel, under the helper tier, in a nested Go module and in per-frontend copies as rejected alternatives with reasons

#### Scenario: The tier is not a nested module

- **WHEN** a developer lists every `go.mod` file tracked in the repository
- **THEN** exactly one is found, at the repository root, and none is under `opm/k8s/`

### Requirement: No other library package imports the Kubernetes tier

No package outside `opm/k8s/` SHALL import a package under `opm/k8s/`. This covers `opm/kernel`, `opm/module`, `opm/platform`, `opm/catalog`, `opm/schema`, `opm/errors`, every package under `opm/internal/` and every package under `opm/helper/`, test files included. The rule SHALL be enforced by the repository lint gate before any package under `opm/k8s/` exists. Source: 0012:D3.

#### Scenario: Lint refuses a kernel import of the tier

- **WHEN** a change adds an import of a package under `opm/k8s/` to a file under `opm/kernel/`, `opm/internal/` or any other `opm/` package outside `opm/k8s/`
- **THEN** `task lint` fails naming the forbidden import

#### Scenario: The kernel's dependency list is free of the tier

- **WHEN** the dependency list of `opm/kernel` and of every package under `opm/internal/` is computed
- **THEN** no import path under `opm/k8s/` and no path under `k8s.io/` appears in it

### Requirement: The kernel imports no Kubernetes package and no library package imports a cluster runtime

No file under `opm/kernel`, `opm/module`, `opm/platform`, `opm/catalog`, `opm/schema`, `opm/errors` or `opm/internal/` SHALL import a package under `k8s.io/` or `sigs.k8s.io/`, test files included. No file under `opm/` SHALL import `k8s.io/client-go`, `sigs.k8s.io/controller-runtime` or a package under `github.com/fluxcd/`. Both rules SHALL be enforced by the repository lint gate before any package under `opm/k8s/` exists. Source: 0012:D2, 0012:D3.

#### Scenario: Lint refuses a Kubernetes import in the kernel

- **WHEN** a change adds an import of a `k8s.io/` or `sigs.k8s.io/` package to a file under a kernel package
- **THEN** `task lint` fails naming the forbidden import

#### Scenario: Lint refuses a cluster runtime anywhere under opm

- **WHEN** a change adds an import of `k8s.io/client-go`, `sigs.k8s.io/controller-runtime` or a `github.com/fluxcd/` package to any file under `opm/`, the helper tier included
- **THEN** `task lint` fails naming the forbidden import

### Requirement: The Kubernetes tier imports no cluster client or controller framework

Beyond the standard library and the CUE SDK that the kernel's output types carry, a file under `opm/k8s/` MAY import only the kernel's exported packages and `k8s.io/apimachinery`. No file under `opm/k8s/` SHALL import `k8s.io/client-go`, `sigs.k8s.io/controller-runtime`, any package under `github.com/fluxcd/`, any cluster client, or a package under `opm/internal/` or `opm/helper/`, and of the Kubernetes modules it SHALL import only `k8s.io/apimachinery`. The repository lint gate SHALL enforce these denials, including the refusal of every other `k8s.io` and `sigs.k8s.io` module. For non-test files the gate SHALL also enforce the allowed set as a strict allow list: the Go standard library, `cuelang.org/go/cue` and its subpackages (not sibling modules such as `cuelang.org/go/cuego`), `k8s.io/apimachinery` and the library's own `opm/` packages, so any other third-party import is refused without a reviewer having to notice it. The one exception is the test file `opm/k8s/object/objectset_parity_test.go`, which imports the deprecated `opm/helper/objectset` to check the duplicate-object-identities parity requirement; the change that removes `opm/helper/objectset` deletes it. Source: 0012:D3.

#### Scenario: Lint refuses a cluster client in the tier

- **WHEN** a change adds an import of `k8s.io/client-go`, `sigs.k8s.io/controller-runtime` or a `github.com/fluxcd/` package to a file under `opm/k8s/`
- **THEN** `task lint` fails naming the forbidden import

#### Scenario: Lint refuses a Kubernetes module other than apimachinery in the tier

- **WHEN** a change adds an import of `k8s.io/api`, `k8s.io/utils`, `k8s.io/klog` or `sigs.k8s.io/yaml` to a file under `opm/k8s/`
- **THEN** `task lint` fails naming the forbidden import
- **AND** an import of `k8s.io/apimachinery` together with `opm/kernel` in the same file passes

#### Scenario: Lint refuses a Kubernetes module in the helper tier

- **WHEN** a change adds an import of `k8s.io/apimachinery` or any other `k8s.io` or `sigs.k8s.io` package to a file under `opm/helper/`
- **THEN** `task lint` fails naming the forbidden import

#### Scenario: Lint refuses a helper or internal import in the tier

- **WHEN** a change adds an import of a package under `opm/helper/` or `opm/internal/` to a file under `opm/k8s/`
- **THEN** `task lint` fails naming the forbidden import

#### Scenario: Lint refuses an unlisted third-party import in the tier

- **WHEN** a change adds an import of `github.com/google/uuid` to a non-test file under `opm/k8s/object/`
- **THEN** `task lint` fails naming the import

### Requirement: Nothing in the library performs a cluster action

Nothing in the library SHALL perform a cluster action, and the Kubernetes tier SHALL perform no cluster I/O. No executor (a loop that drives a plan to completion) and no executor backend that performs a planned action against a cluster SHALL ship anywhere under `opm/`: not in the kernel, not under `opm/k8s/` and not under `opm/helper/`. ADR-008 rule 3's allowance for opt-in executor backends under `opm/helper/` is narrowed to executor backends that perform no planned action against a cluster (0009's wasm, HTTP, `cue.eval` and local container hosts; 0012:D3 amends 0009:D4 so its cluster-acting Ops are performed by the frontend). A deletion plan SHALL advance one action per call from a serialisable state the caller owns, and the frontend SHALL perform each action with its own client, in a loop it writes itself. The tier SHALL own the whole deletion sequence (plan, transition, hold verdict). For apply, it SHALL own the per-object verdict (CanApply) and the order, never an apply engine. Each frontend SHALL submit objects in the library's order. An engine's own staging MAY refine that order, for example by sorting within a stage, and SHALL NOT contradict it. ADR-008 rules 1 to 3 apply to the tier. Source: 0012:D3, 0012:D4, 0012:D5.

#### Scenario: No executor loop or cluster backend ships anywhere in the library

- **WHEN** a developer searches every package under `opm/` (the kernel, `opm/k8s/` and `opm/helper/`) for a function that takes a cluster client, a REST config or a controller-runtime client, that performs a planned action against a cluster, or that drives a plan to completion
- **THEN** none exists

#### Scenario: ADR-008 allows only non-cluster executor backends

- **WHEN** a developer reads rule 3 of ADR-008
- **THEN** it states that no executor backend that performs a planned action against a cluster ships in the library, `opm/helper/` included, that each frontend performs such an action with its own client, and that opt-in executor backends that perform no planned action against a cluster may ship under `opm/helper/`

#### Scenario: Apply engines stay with the frontends

- **WHEN** a developer reads ADR-011
- **THEN** it states that the operator keeps `fluxcd/pkg/ssa` and the CLI keeps its own server-side apply, that the library owns the per-object apply verdict and the order and never an apply engine, and that an engine's staging may refine the library's order and never contradict it

### Requirement: The Kubernetes tier binds a Kubernetes frontend

The Kubernetes tier SHALL NOT be described as opt-in. A frontend that targets Kubernetes uses it, and when it adopts a package it deletes its own copy in the same release with no alias. The library's constitution and package documentation SHALL say so, and SHALL NOT state that everything outside `opm/helper/` is kernel contract. Source: 0012:D3.

#### Scenario: The constitution names three tiers

- **WHEN** a developer reads Principle III of `CONSTITUTION.md` and of the constitution embedded in `openspec/config.yaml`
- **THEN** both name the kernel, the opt-in `opm/helper/` tier and the `opm/k8s/` tier, and state that `opm/k8s/` is mandatory for a Kubernetes frontend and fenced from the kernel

#### Scenario: The runtime-concerns clause names what is admitted

- **WHEN** a developer reads Principle IV of `CONSTITUTION.md` and of the constitution embedded in `openspec/config.yaml`
- **THEN** both state that `opm/k8s/` alone may import `k8s.io/apimachinery` and that no `opm/` package imports `k8s.io/client-go`, `sigs.k8s.io/controller-runtime` or Flux

### Requirement: Kind-class order is a Kubernetes fact that lives in the tier

The library SHALL keep one kind-class apply order (a CustomResourceDefinition before its custom resources, a Namespace before namespaced objects). It SHALL live in the Kubernetes tier, and the kernel SHALL hold none. No module-internal ordering (`dependsOn` edges, phases within one module) is planned: the library SHALL add no module-specific ordering of its own, and any module-internal ordering added later SHALL come off the CUE build as data. Lifecycle hooks remain the parked 0009 work (open under 0009:OQ7), carried as build data under ADR-008 rule 4. In the OPM model, ordering across modules belongs to a future Bundle definition, which uses the order its modules are defined in; the library SHALL add none before then. A frontend's readiness gating between its own custom resources, such as the operator's ModulePackage `spec.dependsOn`, is outside this rule; whether a future Bundle definition subsumes it is not decided. ADR-008 rule 4's "The kernel derives no ordering of its own" SHALL be read as excluding module-specific ordering only, and ADR-008 and ADR-011 SHALL say so in their own text. Source: 0012:D5.

#### Scenario: ADR-008 carries the clarification

- **WHEN** a developer reads `adr/008-kernel-plans-caller-runs.md`
- **THEN** its Status records the 2026-10-02 amendment pointing at ADR-011, and the rule ending "The kernel derives no ordering of its own" states that it excludes module-specific ordering only

#### Scenario: No module-internal ordering is planned and cross-module order belongs to a Bundle

- **WHEN** a developer reads rule 4 of ADR-008 and item 7 of ADR-011
- **THEN** both state that no module-internal ordering (`dependsOn` edges, phases) is planned and that any added later comes off the build, that lifecycle hooks remain the parked 0009 work, and that in the OPM model ordering across modules belongs to a future Bundle definition using the order its modules are defined in
- **AND** neither presents module-internal `dependsOn` edges or phases as planned work

### Requirement: Labels are stamped in CUE and the shared digest ignores the runtime name

Label stamping SHALL stay in CUE at render, where core's `#runtimeName` fills the managed-by label. The tier's shared render digest SHALL leave the managed-by label's value out of its input, so that the CLI and the operator, rendering the same instance, compute the same digest bytes. Source: 0012:D6, 0012:D1:R1.

#### Scenario: The digest rule is recorded

- **WHEN** a developer reads ADR-011
- **THEN** it states that labels stay stamped in CUE at render and that the shared digest excludes the managed-by label value

### Requirement: Deletion plans come from the inventory and the live objects

An instance's deletion plan (uninstall) SHALL be built from the persisted inventory plus the live objects in the cluster. Building it SHALL NOT need a render and SHALL NOT read a plan stored at apply time. A prune's stale set SHALL be computed by the inventory package and handed to the same plan. ADR-008 SHALL record this, and SHALL record that planning from a render and lifecycle transition detection (install, upgrade, reconfigure, no-op, uninstall) are deferred to the lifecycle hook work of enhancement 0009. This deferral does not touch the deletion step transition, which the tier owns as part of the deletion sequence. Source: ADR-008 (owner decision 2026-10-03); the deletion protocol is 0012:D4.

#### Scenario: ADR-008 records the deletion-plan source

- **WHEN** a developer reads `adr/008-kernel-plans-caller-runs.md`
- **THEN** it states that an instance's deletion plan (uninstall) is built from the persisted inventory plus the live objects, with no render and no stored plan, and that a prune's stale set is computed by the inventory package and handed to the same plan
- **AND** it lists planning from a render and lifecycle transition detection among the questions it does not decide, deferred to the 0009 hook work, distinct from the deletion step transition the tier owns

### Requirement: The label vocabulary names labels and never stamps them

The library SHALL provide `opm/k8s/labels`, which imports nothing outside the Go standard library. It SHALL declare the label keys and values both Kubernetes frontends read or write, with values byte-equal to the frontends' copies: the managed-by key `app.kubernetes.io/managed-by`, its runtime values `opm-cli` and `opm-controller`, its legacy value `open-platform-model`, `opmodel.dev/component`, `component.opmodel.dev/name`, `module-instance.opmodel.dev/name`, `module-instance.opmodel.dev/namespace` and `module-instance.opmodel.dev/uuid`. The constant for `opmodel.dev/component` SHALL be documented as the OPM object category, not a component name. It SHALL provide a predicate that reports whether a managed-by value identifies an OPM runtime, which accepts exactly the two runtime values and the legacy value. No function in the package SHALL set a label on an object. Labels on rendered objects are stamped by the CUE build at render. The repository lint gate SHALL refuse any non-test import into the package from outside the standard library. Source: 0012:D6, 0012:D3.

#### Scenario: Every OPM managed-by value is recognised

- **WHEN** the predicate is called with `opm-cli`, `opm-controller` or `open-platform-model`
- **THEN** it reports true for each
- **AND** it reports false for `helm`, for the empty string and for `OPM-CLI`

#### Scenario: The vocabulary matches the frontends' copies

- **WHEN** a table test pins each declared key and value to its literal
- **THEN** each equals the literal, and the literals are the ones a scripted comparison found byte-equal to the cli's `pkg/core/labels.go` at cli `1338e700` and the operator's at opm-operator `9b83611`

#### Scenario: Lint keeps the label package dependency-free

- **WHEN** a change adds an import of `k8s.io/apimachinery`, `cuelang.org/go/cue` or a library package to a non-test file under `opm/k8s/labels/`
- **THEN** `task lint` fails naming the import

### Requirement: Compiled objects convert to Kubernetes objects through one export

The library SHALL provide, in `opm/k8s/object`, a `Resource` that carries a rendered object's CUE value and its instance, component and transformer provenance, together with constructors from the kernel's `*kernel.Compiled`. A nil `Compiled` SHALL yield no Resource. `Resource` SHALL expose the object's kind, name, namespace, apiVersion, group-version-kind, labels and annotations, each empty or nil when absent rather than an error. It SHALL also expose its JSON form and its `*unstructured.Unstructured` form. The library SHALL provide an export of a list of Resources that exports each Resource's value from CUE once and returns, in input order, that JSON, the object decoded from those same bytes, and the provenance. The export SHALL NOT change or release the input Resources. A failure SHALL name the failing Resource, its position, and whether the CUE export or the JSON decode failed. A value that exports to JSON that is not an object, `null` included, SHALL fail as a decode failure, so every exported object carries a non-nil object map. A Resource keeps its whole CUE build alive; releasing it after export is the caller's responsibility. Source: 0012:D1, 0012:D3.

#### Scenario: One export feeds every consumer of the object

- **WHEN** a list of Resources built from compiled objects is exported
- **THEN** each result's JSON equals that Resource's own JSON form, and each result's object equals what that JSON decodes to
- **AND** each result carries its Resource's instance, component and transformer, in input order

#### Scenario: A failing export names the resource and the step

- **WHEN** one Resource in the list carries a CUE value that cannot be exported to JSON
- **THEN** the export fails with an error naming that Resource and its position and reporting that the CUE export failed

#### Scenario: A value that is not an object fails at the decode

- **WHEN** one Resource in the list carries a concrete value that exports as a JSON list, a JSON string or `null`
- **THEN** the export fails with an error naming that Resource and its position and reporting that the JSON decode failed

#### Scenario: The input survives the export

- **WHEN** a list of Resources is exported
- **THEN** the list and each Resource's value, instance, component and transformer are unchanged afterwards

#### Scenario: Accessors are best-effort

- **WHEN** a Resource's value has no `metadata.namespace` and no `metadata.annotations`
- **THEN** its namespace is the empty string and its annotations are nil, with no error

### Requirement: One weight table orders apply and delete

The library SHALL keep one kind-class weight table in `opm/k8s/object`, and it SHALL agree with the staged apply order of Flux's `ssa` package at the version the operator pins (v0.77.0), so that an engine applying through Flux only refines it. A CustomResourceDefinition weighs -100. A Namespace weighs 0. ClusterRoles weigh 5. PriorityClasses, RuntimeClasses, StorageClasses, VolumeSnapshotClasses, IngressClasses, GatewayClasses, ClusterClasses and every other kind whose name ends in `Class` weigh 6. ClusterRoleBindings weigh 7. ResourceQuotas weigh 8. ServiceAccounts, Roles and RoleBindings weigh 10. Secrets and ConfigMaps weigh 15. Services weigh 50. LimitRanges weigh 60. Deployments and StatefulSets weigh 100. CronJobs weigh 105. PodDisruptionBudgets weigh 108. Validating and mutating webhook configurations weigh 2000. Every other kind, among them PersistentVolumes, PersistentVolumeClaims, DaemonSets, ReplicaSets, Jobs, Ingresses, NetworkPolicies, the pod autoscalers and every custom resource, weighs 1000. A weight SHALL be looked up by group, version and kind first; then a CustomResourceDefinition of `apiextensions.k8s.io`, a Namespace of the core group and a ClusterRole of `rbac.authorization.k8s.io` by group and kind in any version; then by kind alone, where a kind named CustomResourceDefinition, Namespace or ClusterRole in another group weighs 6; then any kind whose name ends in `Class` weighs 6; else 1000. The library SHALL provide a stable sort by weight, ascending for apply and descending for delete. Source: 0012:D5:R1.

#### Scenario: Apply order puts definitions before their users

- **WHEN** a Deployment, a Namespace, a CustomResourceDefinition, a custom resource and a ConfigMap are sorted ascending
- **THEN** the order is the CustomResourceDefinition, the Namespace, the ConfigMap, the Deployment, the custom resource

#### Scenario: Delete order is the reverse and stable

- **WHEN** two Services and a Deployment are sorted descending
- **THEN** the Deployment comes first and the two Services keep their relative input order

#### Scenario: An unknown version of a known kind falls back to the kind

- **WHEN** the weight of `apps/v1beta2 Deployment` is looked up
- **THEN** it is 100

#### Scenario: Webhook configurations come after custom resources

- **WHEN** a ValidatingWebhookConfiguration, a MutatingWebhookConfiguration, a custom resource and a Service are sorted ascending
- **THEN** the order is the Service, the custom resource, then the two webhook configurations
- **AND** sorted descending, the two webhook configurations come first

#### Scenario: Class kinds come right after ClusterRoles

- **WHEN** a ClusterRoleBinding, a PriorityClass, a custom kind `EC2NodeClass`, a ClusterRole and a ServiceAccount are sorted ascending
- **THEN** the order is the ClusterRole, then the PriorityClass and the `EC2NodeClass` in input order, then the ClusterRoleBinding, then the ServiceAccount

#### Scenario: Quotas and limits come before the workloads they constrain

- **WHEN** a Deployment, a LimitRange, a Service, a ResourceQuota and a ServiceAccount are sorted ascending
- **THEN** the order is the ResourceQuota, the ServiceAccount, the Service, the LimitRange, the Deployment

#### Scenario: Kinds Flux does not list weigh the default

- **WHEN** the weights of a PersistentVolumeClaim, a DaemonSet, a Job, an Ingress, a HorizontalPodAutoscaler and a custom resource are looked up
- **THEN** each is 1000

#### Scenario: A definition kind name in another group is not a definition

- **WHEN** the weights of kind `Namespace` in group `example.com` and of `rbac.authorization.k8s.io/v1beta1 ClusterRole` are looked up
- **THEN** the first is 6 and the second is 5

### Requirement: Apply stages follow the weight table

The library SHALL provide a function that returns a sorted copy of an apply set cut into stages. The first stage holds the cluster definitions, which are every CustomResourceDefinition of `apiextensions.k8s.io` and every Namespace of the core group, and is marked as such. It is omitted when there are none. After it comes one stage per distinct weight of the remaining objects, in ascending weight. Order within a stage SHALL be the stable sort order. The input SHALL NOT be reordered, and no stage SHALL be empty. Because the weight table never contradicts Flux's staged apply order, a frontend whose apply engine is Flux's staged apply MAY submit the whole set or any one stage in one call: the engine then only refines the library's order. Source: 0012:D4, 0012:D5.

#### Scenario: Stages for a typical module

- **WHEN** a CustomResourceDefinition, a Namespace, a ConfigMap, a Secret, a Deployment and a Service are staged
- **THEN** the stages are the cluster definitions (the CustomResourceDefinition and the Namespace), then the ConfigMap and the Secret, then the Service, then the Deployment

#### Scenario: A Namespace kind in another group is not a cluster definition

- **WHEN** an object of kind `Namespace` in group `example.com` is staged beside a core Namespace
- **THEN** only the core Namespace is in the cluster-definition stage

#### Scenario: No cluster definitions, no definition stage

- **WHEN** a set holding only a Deployment and a Service is staged
- **THEN** two stages are returned, neither marked as cluster definitions

### Requirement: Readiness evaluation is a pure function over fetched objects

The library SHALL provide `opm/k8s/health`, which judges the readiness of one Kubernetes object that the caller has already fetched, and aggregates such judgements. Nothing in the package SHALL read a cluster, wait or poll; the frontend fetches each live object with its own client and passes it in. A status SHALL be one of the strings `Ready`, `NotReady`, `Complete`, `Unknown`, `Missing`, `Applied` and `Bound`, byte-equal to the cli's `internal/kubernetes/health.go` at cli `bd4d1a7c`, or, for a PersistentVolumeClaim, the raw value of its `status.phase`. The evaluation rules SHALL be the cli's at that commit, unchanged:

- A Deployment is `Ready` only when `status.observedGeneration` has reached `metadata.generation`, no Progressing condition carries the reason `ProgressDeadlineExceeded`, and the updated and available replica counts equal `spec.replicas` (1 when omitted) with no more total replicas than updated ones; otherwise `NotReady`.
- A StatefulSet is `Ready` when the generation is observed and the ready replicas equal `spec.replicas`, and then: always under the `OnDelete` strategy; under a RollingUpdate partition greater than zero, when the updated replicas reach `spec.replicas` minus the partition; otherwise when the updated replicas equal `spec.replicas` and the update revision equals the current revision. Otherwise `NotReady`.
- A DaemonSet is `Ready` when the generation is observed and the updated and available counts equal `status.desiredNumberScheduled`; otherwise `NotReady`.
- A Job is `Complete` when, in condition order, a `Complete` condition with status `True` appears before any `Failed` condition with status `True`; otherwise `NotReady`.
- A CronJob, and every kind of the fixed passive set (ConfigMap, Secret, Service, ServiceAccount, Namespace, ClusterRole, ClusterRoleBinding, Role, RoleBinding, Ingress, NetworkPolicy, PodDisruptionBudget, ResourceQuota, LimitRange, StorageClass, PriorityClass), is `Applied`.
- A PersistentVolumeClaim reports its `status.phase` as the status (`Bound` for a bound claim), or `Ready` when it has no phase yet.
- Any other kind is `Ready` when its `Ready` condition is `True`, `NotReady` when that condition has any other status, and `Applied` when it has no `Ready` condition.

`Ready`, `Applied`, `Complete` and `Bound` SHALL be the healthy statuses, and every other string SHALL be unhealthy. The aggregate of a list of statuses and a count of tracked objects that have no status (missing or unreadable) SHALL count each of those objects toward the total and never toward ready, and SHALL be `Unknown` when the total is zero, `Ready` when every counted object is healthy, and `NotReady` otherwise, together with the ready count and the total. The package SHALL also report, through `ProgressDeadlineExceeded`, whether a Deployment whose controller has observed its current generation carries a Progressing condition with the reason `ProgressDeadlineExceeded`, false for any other object, so a frontend can tell a stalled rollout from one in progress without reading conditions itself. A strict lint rule SHALL hold the package's non-test files to the Go standard library and `k8s.io/apimachinery`. Source: 0012:D3; owner decision f5 (beta.1 walkthrough).

#### Scenario: The status strings match the cli's output

- **WHEN** a table test pins each status constant to its literal
- **THEN** the seven constants are `Ready`, `NotReady`, `Complete`, `Unknown`, `Missing`, `Applied` and `Bound`

#### Scenario: A Deployment mid-rollout is not ready

- **WHEN** a Deployment whose generation is observed and whose Available condition is `True` reports fewer updated replicas than `spec.replicas`
- **THEN** its status is `NotReady`

#### Scenario: A Deployment past its progress deadline is not ready

- **WHEN** a Deployment whose replica counters all equal `spec.replicas` carries a Progressing condition with reason `ProgressDeadlineExceeded`
- **THEN** its status is `NotReady`

#### Scenario: A stalled Deployment is told apart from one rolling out

- **WHEN** `ProgressDeadlineExceeded` is asked about a Deployment whose generation is observed and whose Progressing condition has reason `ProgressDeadlineExceeded`, and about one whose updated replicas still lag with Progressing reason `ReplicaSetUpdated`
- **THEN** it reports true for the first and false for the second, and `Evaluate` reports `NotReady` for both

#### Scenario: A partitioned StatefulSet is ready at its partition

- **WHEN** a StatefulSet with three replicas, its generation observed and a RollingUpdate partition of 2, reports three ready replicas and one updated replica
- **THEN** its status is `Ready`

#### Scenario: A pending claim reports its phase

- **WHEN** a PersistentVolumeClaim with `status.phase` `Pending` is evaluated
- **THEN** its status is the string `Pending`, and that status is not healthy

#### Scenario: A custom resource without a Ready condition counts as applied

- **WHEN** an object of an unlisted kind with no `Ready` condition is evaluated
- **THEN** its status is `Applied`, and that status is healthy

#### Scenario: Missing and unreadable objects count against the aggregate

- **WHEN** three `Ready` statuses are aggregated with two tracked objects that have no status
- **THEN** the aggregate is `NotReady` with 3 ready out of 5

#### Scenario: Nothing to aggregate is unknown

- **WHEN** no statuses and no objects without a status are aggregated
- **THEN** the aggregate is `Unknown` with 0 ready out of 0

#### Scenario: The package performs no cluster I/O

- **WHEN** a non-test file of `opm/k8s/health` imports any package other than the Go standard library and `k8s.io/apimachinery`, such as `k8s.io/client-go` or a library `opm/` package
- **THEN** `task lint` fails on the depguard rule `k8s-health-imports-only-apimachinery`, so the package can hold no cluster client and takes every object as an argument

### Requirement: The adopt annotation key is fixed in the label vocabulary

`opm/k8s/labels` SHALL declare the adopt annotation key `opmodel.dev/adopt`. It is an annotation a user sets on an existing live object, and its value names the adopting instance by the value of that instance's `module-instance.opmodel.dev/uuid` label. No function in the library SHALL set it on an object, and the package documentation SHALL say that no OPM runtime writes it. Source: 0012:D8 (the implementing change fixes the key), 0012:D8:R6.

#### Scenario: The key is the fixed literal

- **WHEN** a test reads the adopt annotation constant
- **THEN** it equals `opmodel.dev/adopt`

#### Scenario: No library code sets the adopt annotation

- **WHEN** every non-test Go file under `opm/` is parsed
- **THEN** no file outside `opm/k8s/labels` spells the literal `opmodel.dev/adopt`
- **AND** no file outside `opm/k8s/labels` and `opm/k8s/ownership`, which only reads the key, refers to the adopt annotation constant

### Requirement: The ownership verdicts are pure

The library SHALL provide `opm/k8s/ownership` with a per-object delete verdict and a per-object apply verdict, as pure functions of explicit inputs. Each input SHALL name the object by group, kind, namespace and name and SHALL carry the live object as the caller read it, absent when the read found nothing, together with the judging instance's UUID. The package SHALL perform no cluster read or write, read no clock or environment, and log nothing. It SHALL NOT modify the live object it is given. Each reason SHALL be a string equal to the contract literal for it. A frontend reads the live object with its own client. Source: 0012:D4, ADR-008, ADR-011.

#### Scenario: The live object is not modified

- **WHEN** either verdict is computed for a live object
- **THEN** the object is deeply equal to a copy taken before the call

#### Scenario: The package imports no clock, environment or logger

- **WHEN** the direct imports of `opm/k8s/ownership` are listed
- **THEN** none of them is `os`, `time`, `log` or `log/slog`

#### Scenario: Reasons are the contract's literals

- **WHEN** a test reads the skip and refusal reason constants
- **THEN** the skip reasons are `safety-excluded`, `already-absent`, `not-opm-managed`, `owner-mismatch` and `adopted-elsewhere`
- **AND** the refusal reasons are `terminating`, `foreign-object`, `other-instance` and `adopted-elsewhere`

### Requirement: Safety-excluded kinds match on group and kind

The library SHALL report a core-group `Namespace` and an `apiextensions.k8s.io` `CustomResourceDefinition` as safety-excluded, and no other group and kind pair. The test SHALL NOT need a live object. Source: 0012:D1:R3, the 0012 contract's `#safetyExcluded`.

#### Scenario: The two protected kinds are excluded

- **WHEN** the core group with kind `Namespace`, and `apiextensions.k8s.io` with kind `CustomResourceDefinition`, are tested
- **THEN** both are safety-excluded

#### Scenario: A same-named kind in another group is not excluded

- **WHEN** `example.com` with kind `Namespace`, and `example.com` with kind `CustomResourceDefinition`, are tested
- **THEN** neither is safety-excluded

### Requirement: The delete verdict skips with a reason or proceeds with the judged object's identity

The delete verdict SHALL decide in this order and stop at the first match: skip as `safety-excluded` when the kind is safety-excluded, whatever the live object; skip as `already-absent` when there is no live object; skip as `not-opm-managed` when the live managed-by label is not an OPM runtime's value; skip as `owner-mismatch` when the live UUID label and the instance's UUID are both non-empty and differ; skip as `adopted-elsewhere` when the live adopt annotation, compared with surrounding whitespace trimmed, is non-blank and differs from the instance's UUID; otherwise proceed. An empty UUID on either side SHALL pass the owner comparison. A blank adopt annotation SHALL pass the adoption comparison, and with an empty instance UUID any non-blank adopt annotation SHALL count as another instance's, since an empty UUID can never be the one an annotation names. An adopt annotation naming this instance SHALL change nothing. An object being deleted SHALL NOT be skipped for that reason. A proceed verdict SHALL carry the UID and the resourceVersion of the live object it judged. Its DELETE precondition SHALL name that UID and SHALL NOT name a resourceVersion. A caller MAY add the carried resourceVersion itself. A skip verdict, and a proceed verdict whose judged UID is empty, SHALL yield no precondition at all, never a precondition on an empty UID. A skip verdict SHALL carry a message naming the object and the reason. So a prune or an instance deletion never deletes an object the user is handing to another instance, before or after that instance has applied it. Source: 0012:D1:R3/R4, 0012:D4:R1, 0012:D8:R8 (enhancements#103).

#### Scenario: A protected kind is skipped without a live read

- **WHEN** the delete verdict is asked for a core Namespace with no live object
- **THEN** it skips as `safety-excluded`

#### Scenario: A gone object is skipped as already absent

- **WHEN** the delete verdict is asked for a Deployment with no live object
- **THEN** it skips as `already-absent`

#### Scenario: A foreign object is skipped

- **WHEN** the live Deployment's managed-by label is `helm`
- **THEN** the verdict skips as `not-opm-managed` and its message names the Deployment

#### Scenario: Another instance's object is skipped

- **WHEN** the live Deployment is OPM-managed and its UUID label differs from the non-empty instance UUID
- **THEN** the verdict skips as `owner-mismatch`

#### Scenario: An empty UUID on either side passes

- **WHEN** the live Deployment is OPM-managed, carries no adopt annotation, and either its UUID label or the instance UUID is empty
- **THEN** the verdict proceeds

#### Scenario: Proceed carries the precondition for the judged object

- **WHEN** the verdict proceeds for a live object with UID `u-1` and resourceVersion `42`
- **THEN** it carries UID `u-1` and resourceVersion `42`
- **AND** its DELETE precondition names UID `u-1` and no resourceVersion

#### Scenario: A skip verdict yields no precondition

- **WHEN** the delete verdict skips as `already-absent`
- **THEN** it yields no DELETE precondition

#### Scenario: A live object without a UID yields no precondition

- **WHEN** the verdict proceeds for a live object that carries no UID
- **THEN** it yields no DELETE precondition

#### Scenario: An object being deleted proceeds

- **WHEN** the live OPM-managed object of this instance has a deletion timestamp
- **THEN** the verdict proceeds

#### Scenario: An object annotated for another instance is left in place

- **WHEN** the live Deployment is OPM-managed, carries this instance's UUID label `u-9`, and its adopt annotation is `u-1`, for instance UUID `u-9`
- **THEN** the verdict skips as `adopted-elsewhere`
- **AND** its message is `Deployment/web/api is being adopted by module instance u-1, not this one; left in place`
- **AND** it yields no DELETE precondition

#### Scenario: An annotation naming this instance changes nothing on delete

- **WHEN** the live Deployment is OPM-managed, carries UUID label `u-9`, and its adopt annotation is `u-9`, for instance UUID `u-9`
- **THEN** the verdict proceeds

#### Scenario: An empty instance UUID skips an annotated object

- **WHEN** the live Deployment is OPM-managed and its adopt annotation is `u-1`, for an empty instance UUID
- **THEN** the verdict skips as `adopted-elsewhere`

### Requirement: The adopt annotation is the only override and the refusal names it

An adopt annotation whose value equals the instance UUID SHALL lift the `foreign-object`, `other-instance` and `adopted-elsewhere` refusals. The annotation value SHALL be compared with surrounding whitespace trimmed, and a value that is empty or only whitespace SHALL count as no annotation. An adopt annotation with any other value SHALL lift nothing, and the refusal message SHALL say that the annotation names another instance. The refusal message for `foreign-object` and `other-instance` SHALL name the object, the annotation key `opmodel.dev/adopt` and the instance UUID to set it to, unless the instance UUID is empty. The `other-instance` message SHALL ask the user to remove the object from the instance that owns it before annotating it, so that instance stops rendering it and is not refused on every apply after the hand-over. No message SHALL name any other way past a refusal, such as a command-line flag. No message SHALL carry an enhancement reference. Source: 0012:D8:R2/R3.

#### Scenario: Adoption lifts a foreign refusal

- **WHEN** the object outside the inventory is not OPM-managed and its adopt annotation equals the instance UUID
- **THEN** the verdict applies

#### Scenario: Adoption lifts an other-instance refusal

- **WHEN** the object outside the inventory carries another instance's UUID label and its adopt annotation equals the instance UUID
- **THEN** the verdict applies

#### Scenario: An annotation naming another instance lifts nothing

- **WHEN** the object outside the inventory is not OPM-managed and its adopt annotation names a different UUID
- **THEN** the verdict refuses as `foreign-object`
- **AND** the message says the annotation names another instance

#### Scenario: Surrounding whitespace in the annotation value is ignored

- **WHEN** the object outside the inventory is not OPM-managed and its adopt annotation is the instance UUID with surrounding whitespace
- **THEN** the verdict applies

#### Scenario: A blank annotation counts as none

- **WHEN** the object outside the inventory is not OPM-managed and its adopt annotation is empty or only whitespace
- **THEN** the verdict refuses as `foreign-object`
- **AND** the message does not say the annotation names another instance

#### Scenario: The other-instance remedy moves the object only after its owner lets go

- **WHEN** the verdict refuses `Deployment/web/api` as `other-instance` for instance UUID `u-9`, the live UUID label being `u-1`
- **THEN** the message asks to remove it from module instance `u-1` before annotating it `opmodel.dev/adopt=u-9`

#### Scenario: The refusal tells the user how to adopt

- **WHEN** the verdict refuses `Deployment/web/api` as `foreign-object` for instance UUID `u-9`
- **THEN** the message contains `Deployment/web/api`, `opmodel.dev/adopt` and `u-9`
- **AND** it contains no `--` flag and no `force`

### Requirement: The operator install admission lifts only a proven object's ownership refusal

Both verdicts SHALL take an admission input that a caller sets only for an object it has proven came from an earlier operator release's install manifest. On apply, admission SHALL lift `foreign-object` only, and only when the live object carries no UUID label or carries the instance UUID. On delete, admission SHALL lift `not-opm-managed` only, only when the live object carries no UUID label at all, and only for an `apps` `Deployment`, a `rbac.authorization.k8s.io` `RoleBinding` or a `rbac.authorization.k8s.io` `ClusterRoleBinding`, the kinds install may delete outside the inventory. The UUID label is the only identity admission compares. Admission SHALL NOT lift `terminating`, `other-instance`, `adopted-elsewhere`, `owner-mismatch`, `safety-excluded` or `already-absent`, on either side. The library does not check the proof. Source: 0012:D8:R6/R7, 0012:D8:R8 (enhancements#103), 0012:D4:R1.

#### Scenario: A proven earlier-manifest object is admitted on apply

- **WHEN** the operator install asks for the earlier `Namespace/opm-operator-system`, live with managed-by `kustomize` and no UUID label, admitted and outside the inventory
- **THEN** the verdict applies

#### Scenario: An admitted object carrying another identity is refused

- **WHEN** an admitted object outside the inventory is not OPM-managed but carries a UUID label that differs from the instance UUID
- **THEN** the apply verdict refuses as `foreign-object`

#### Scenario: Admission never lifts the terminating refusal

- **WHEN** an admitted proven object has a deletion timestamp
- **THEN** the apply verdict refuses as `terminating`

#### Scenario: Admission never lifts the adopted-elsewhere refusal

- **WHEN** an admitted object outside the inventory, live with managed-by `kustomize` and no UUID label, carries an adopt annotation naming another instance
- **THEN** the apply verdict refuses as `adopted-elsewhere`

#### Scenario: A proven earlier Deployment may be deleted

- **WHEN** the delete verdict is asked for an admitted live Deployment with managed-by `kustomize` and no UUID label
- **THEN** it proceeds and carries the Deployment's UID

#### Scenario: An admitted object carrying another identity is not deleted

- **WHEN** the delete verdict is asked for an admitted live Deployment with managed-by `kustomize` and a UUID label that differs from the non-empty instance UUID
- **THEN** it skips as `not-opm-managed`

#### Scenario: An admitted object carrying this instance's UUID is not deleted

- **WHEN** the delete verdict is asked for an admitted live ClusterRoleBinding with no OPM managed-by label and a UUID label equal to the instance UUID
- **THEN** it skips as `not-opm-managed`

#### Scenario: Admission deletes only the install's deletable kinds

- **WHEN** the delete verdict is asked for an admitted live ConfigMap, or an admitted live custom resource, with managed-by `kustomize` and no UUID label
- **THEN** it skips as `not-opm-managed`

#### Scenario: Admission never deletes a protected kind

- **WHEN** the delete verdict is asked for an admitted CustomResourceDefinition of `apiextensions.k8s.io`
- **THEN** it skips as `safety-excluded`

#### Scenario: Admission never deletes an object another instance is adopting

- **WHEN** the delete verdict is asked for an admitted live Deployment with managed-by `kustomize`, no UUID label, and an adopt annotation naming another instance
- **THEN** it skips as `adopted-elsewhere`

### Requirement: Inventory entries are plain values built from objects

The library SHALL provide, in `opm/k8s/inventory`, an inventory entry with exactly the fields group, kind, namespace, name, version and component, and a constructor that builds one from a `*unstructured.Unstructured`. The constructor SHALL read group, version and kind, namespace and name from the object, and the component from its `component.opmodel.dev/name` label, which gives an empty component when the label is absent. The entry type SHALL carry no struct tags, so that no serialisation the library chooses can become the frontends' wire shape. Each frontend maps entries to its own CRD or record fields. Source: 0012:D1, 0012:D7.

#### Scenario: An entry reads the object's identity and component

- **WHEN** an entry is built from a `Deployment` in group `apps`, version `v1`, namespace `team`, named `web` and labelled `component.opmodel.dev/name: web`
- **THEN** the entry's group is `apps`, kind `Deployment`, namespace `team`, name `web`, version `v1` and component `web`

#### Scenario: A core-group, cluster-scoped object without the label

- **WHEN** an entry is built from a `v1` `Namespace` named `team` with no `component.opmodel.dev/name` label
- **THEN** the entry's group, namespace and component are empty, its kind is `Namespace`, its name `team` and its version `v1`

#### Scenario: The entry type carries no tags

- **WHEN** a test reflects over the entry type's fields
- **THEN** there are exactly six string fields and none carries a struct tag

### Requirement: The stale set is component-blind

The library SHALL provide, in `opm/k8s/inventory`, one stale-set function. It SHALL return every entry of the previous inventory for which no entry of the current inventory has the same group, kind, namespace and name, whatever the component and the version of either entry. The result SHALL keep the previous inventory's order, SHALL be a non-nil empty list when nothing is stale, and SHALL leave both inputs unchanged. The same identity comparison SHALL be exported on its own. No component-aware comparison and no separate component-rename filter SHALL exist in the package. Source: 0012:D7:R1, 0012:D1:R1.

#### Scenario: A component rename is not stale

- **WHEN** the previous inventory holds `apps/Deployment team/web` with component `api` and the current inventory holds the same group, kind, namespace and name with component `web`
- **THEN** the stale set is empty

#### Scenario: An API version change is not stale

- **WHEN** the previous inventory holds an `autoscaling` `HorizontalPodAutoscaler` at version `v2beta2` and the current inventory holds the same object at version `v2`
- **THEN** the stale set is empty

#### Scenario: Removed objects are stale in previous order

- **WHEN** the previous inventory holds objects A, B, C and D in that order and the current inventory holds only B
- **THEN** the stale set is A, C, D in that order

#### Scenario: The package exports no other comparison

- **WHEN** a test lists the exported identifiers of `opm/k8s/inventory`
- **THEN** they are exactly `Entry`, `NewEntry`, `SameObject`, `StaleSet`, `Digest` and `RenderDigest`

#### Scenario: Nothing stale is an empty list, not nil

- **WHEN** the previous inventory is empty, or every previous object is in the current inventory
- **THEN** the stale set is a non-nil list of length zero

### Requirement: The inventory digest hashes a canonical field encoding

The library SHALL provide, in `opm/k8s/inventory`, one inventory digest of the form `sha256:` followed by 64 lowercase hex digits. It SHALL hash the tag line `opm-inventory-v1` with a newline, followed by every entry in order of group, kind, namespace, name, component and version (byte-wise). Each entry SHALL be written as its group, kind, namespace, name, version and component, in that order, each preceded by its byte length as an 8-byte big-endian unsigned integer. The digest SHALL depend only on the entries' field values: not on their input order and not on any JSON or other serialisation of an entry. Two inventories that differ in their entries, or in any field of an entry, SHALL produce different digests. The empty inventory SHALL hash the tag line alone. The function SHALL leave its input unchanged. A change to this encoding SHALL use a new tag line. Source: 0012:D7:R2/R3, 0012:D1:R1.

#### Scenario: The encoding is the one defined

- **WHEN** a test writes out by hand the tag line and the length-prefixed fields of the fixture entries (a core-group cluster-scoped entry with empty group and namespace, a namespaced `apps` entry, an entry with an empty component, two `apps` entries whose namespace order and name order disagree, two entries of one kind and namespace whose name order and component order disagree, two entries with one identity whose component order and version order disagree, and two entries that differ only in version), sorted as defined
- **THEN** the digest of those entries, given in another order, equals `sha256:` and the hex SHA-256 of the hand-written bytes, and equals the committed golden value

#### Scenario: Input order does not matter

- **WHEN** the same entries are digested in two different orders
- **THEN** the two digests are equal

#### Scenario: Every field counts

- **WHEN** one entry of an inventory changes in exactly one of its six fields, or an entry is added, removed or repeated, or a byte moves from one field to the next (group `ab` and kind `` against group `a` and kind `b`)
- **THEN** the digest differs from the original inventory's

#### Scenario: Empty and nil inventories agree

- **WHEN** an empty inventory and a nil inventory are digested
- **THEN** both digests equal `sha256:` and the hex SHA-256 of `opm-inventory-v1` followed by a newline

### Requirement: The render digest ignores only the managed-by label value

The library SHALL provide, in `opm/k8s/inventory`, one render digest over the objects that `opm/k8s/object`'s export returns, of the form `sha256:` followed by 64 lowercase hex digits. It SHALL read only each object's exported JSON, never a CUE value, so a caller may release its build before digesting. For each object it SHALL decode the JSON keeping every number's literal, replace the value of the label `app.kubernetes.io/managed-by` with the empty string when `metadata.labels` holds that key, and encode the result as JSON with object keys sorted and no HTML escaping, ending with a newline. Strings SHALL be encoded as Go's `encoding/json` Encoder writes them with HTML escaping off: `"` and `\` escaped with a backslash; U+0008, U+000C, U+000A, U+000D and U+0009 as the short escapes `\b`, `\f`, `\n`, `\r` and `\t`; every other character from U+0000 to U+001F, and U+2028 and U+2029, as a `\u` escape with four lowercase hex digits; every other character, including U+007F and `<`, `>` and `&`, written as is; and invalid UTF-8 replaced by U+FFFD. It SHALL hash the tag line `opm-render-v1` with a newline, followed by the encoded objects in order of group (from `apiVersion`), kind, namespace and name, with the encoded bytes as the final tie-break. Two exported object sets that differ only in the value of the managed-by label SHALL produce the same digest. The cli (`opm-cli`) and the operator (`opm-controller`) therefore compute the same digest for one render as long as the catalogs pass `#runtimeName` only through `#context.labels`, which is the case the kernel test checks on the library's render fixture and on the shipped catalog. Two object sets that differ in any other label, in whether the managed-by label is present, or in any other content of any object SHALL produce different digests. The empty set SHALL hash the tag line alone. An object whose JSON does not decode to a single JSON object SHALL fail with an error naming its position. The function SHALL leave its input unchanged. A change to this encoding SHALL use a new tag line. Source: 0012:D6:R2/R3, 0012:D1:R1.

#### Scenario: The two runtimes digest one render equally

- **WHEN** the kernel renders one instance with runtime name `opm-cli` and again with `opm-controller`, for the library's render fixture and for the shipped-catalog parity instance
- **THEN** each rendered object's managed-by label value is that render's runtime name, and the two renders are equal object by object once that one value is blanked
- **AND** two exported object sets that differ only in the managed-by label value have equal render digests

#### Scenario: Any other label counts

- **WHEN** two exported object sets differ only in the value of one label other than `app.kubernetes.io/managed-by`
- **THEN** their render digests differ

#### Scenario: Adding or removing the managed-by label counts

- **WHEN** one object set carries the managed-by label on an object and the other does not, and nothing else differs
- **THEN** their render digests differ

#### Scenario: Large integers are not rounded

- **WHEN** two exported object sets differ only in one integer field, `9007199254740993` in one and `9007199254740992` in the other
- **THEN** their render digests differ

#### Scenario: The encoding is the one defined

- **WHEN** a test writes out by hand the tag line and the sorted-key JSON of a Deployment carrying a managed-by label and an annotation with U+2028, a newline, U+0001 and U+007F, a core-group Service, a cluster-scoped Namespace, a core-group ConfigMap in a namespace that sorts after the Services' namespaces, two Services whose namespace order and name order disagree, and two Services in one namespace whose name order and encoded-bytes order disagree, sorted as defined and with the managed-by value blanked
- **THEN** the render digest of those objects, given in another order, equals `sha256:` and the hex SHA-256 of the hand-written bytes, and equals the committed golden value

#### Scenario: The input is not changed

- **WHEN** a list of exported objects is digested
- **THEN** each object's JSON bytes and decoded object, including its managed-by label value, are unchanged afterwards

#### Scenario: A malformed object fails with its position

- **WHEN** the second object in the list carries JSON that is a list, that is `null`, or that is not valid JSON
- **THEN** the render digest fails with an error naming position 1

### Requirement: The deletion plan orders inventory entries by kind-class delete order

The library SHALL provide `opm/k8s/lifecycle` with a deletion plan built from a list of inventory entries, a deletion policy (`Prune`, `ForceOrphan`) and the deleting instance's UUID. The plan SHALL order the entries by the tier's weight table in descending order, keeping the relative order of entries of equal weight, and SHALL NOT change the caller's list. Every CustomResourceDefinition and Namespace entry, as the tier's safety exclusion matches them, SHALL be marked safety-excluded in the plan before any live object is read. Building the plan SHALL NOT need a render or a stored plan: an uninstall builds it from the persisted inventory, and a prune builds it from the stale set the inventory package computes, through the same constructor. The constructor SHALL be the only way to build a plan with steps, so no frontend can hand the transition a plan in another order, and the plan's steps SHALL be readable as a copy. Source: 0012:D4, 0012:D1:R3/R6, ADR-008 (Deletion plans).

#### Scenario: Entries are ordered for deletion

- **WHEN** a plan is built from a ConfigMap, a Deployment, a Namespace and a ValidatingWebhookConfiguration, in that order
- **THEN** its steps are the ValidatingWebhookConfiguration, the Deployment, the ConfigMap and the Namespace, in that order
- **AND** the caller's list is unchanged

#### Scenario: Equal weights keep their order

- **WHEN** a plan is built from two Deployments `b` then `a`
- **THEN** its steps name `b` before `a`

#### Scenario: Protected kinds are marked up front

- **WHEN** a plan is built from an inventory holding a CustomResourceDefinition of `apiextensions.k8s.io` and a core Namespace
- **THEN** both steps are marked safety-excluded in the plan as built

#### Scenario: A prune hands its stale set to the same plan

- **WHEN** the stale set of a previous and a current inventory is passed to the plan constructor
- **THEN** the plan's steps are exactly the stale entries, in kind-class delete order

#### Scenario: Changing the listed steps does not change the plan

- **WHEN** a caller reorders the steps it read from a plan
- **THEN** the plan's own steps keep their deletion order

### Requirement: The deletion transition names one action per call

The deletion transition SHALL take the plan, the caller's state and the outcome of the action it last named, and SHALL return the next state and exactly one action: read one step's live object, delete one step's object, report one step as skipped, or done. It SHALL perform no action itself. A safety-excluded step SHALL be reported as skipped with the reason `safety-excluded` without a read. Every other step SHALL be read before it is judged, and the library's delete verdict SHALL judge it against the plan's instance UUID. A skip verdict SHALL be reported as skipped with the verdict's reason and message. A proceed verdict SHALL name a delete with Foreground propagation and the verdict's UID precondition. When the policy does not prune, the first call SHALL name done and no step SHALL be read or deleted. Every read, delete and skip action SHALL name a step of the plan, and the transition SHALL name no other kind of action. So a prune of an object the instance dropped because another instance adopted it leaves the object in place. Source: 0012:D4:R1/R6, 0012:D1:R2/R4, 0012:D8:R8 (enhancements#103), ADR-008 rules 1 to 3.

#### Scenario: A protected kind is skipped without a read

- **WHEN** the next step is a Namespace
- **THEN** the action is a skip with reason `safety-excluded`
- **AND** no read is named for that step

#### Scenario: An owned object is read, then deleted with Foreground propagation

- **WHEN** a Deployment step is read and the live object is OPM-managed, carries the plan's instance UUID and has UID `u1`
- **THEN** the next action deletes that Deployment with Foreground propagation and a precondition on UID `u1`

#### Scenario: A foreign object is skipped with the verdict's reason

- **WHEN** a step's live object is not OPM-managed
- **THEN** the next action is a skip with reason `not-opm-managed` and the delete verdict's message

#### Scenario: Another instance's object is skipped

- **WHEN** a step's live object carries a different, non-empty instance UUID than the plan's non-empty owner UUID
- **THEN** the next action is a skip with reason `owner-mismatch`

#### Scenario: A plan that does not prune names done at once

- **WHEN** the policy's `Prune` is false and the transition is called with the zero state
- **THEN** the action is done
- **AND** no read or delete is named for any step

#### Scenario: A prune built with the zero policy deletes nothing

- **WHEN** a plan is built from a non-empty stale set with a policy whose `Prune` is false
- **THEN** the first action is done
- **AND** the state records no outcome

#### Scenario: The transition names only reads, deletions, skips and done

- **WHEN** a plan over a mixed inventory is advanced to done through every outcome kind
- **THEN** every action is a read, a delete, a skip or done
- **AND** every read, delete and skip names a step of the plan

#### Scenario: A prune of a dropped object leaves it for the adopting instance

- **WHEN** a deletion plan built from a stale set holds a Deployment whose live object carries the plan owner's UUID label and an adopt annotation naming another instance
- **THEN** advancing the plan past its read records the step as skipped with `adopted-elsewhere` and names no delete action for it
- **AND** the hold verdict releases once the plan is done

### Requirement: Action outcomes are classified by the library

The caller SHALL hand back the raw error of each read or delete, and the library SHALL classify it with `k8s.io/apimachinery`'s API error helpers, which see through wrapping. A NotFound read, or a read that returns no object, SHALL count as already absent, and SHALL be skipped as `already-absent`. A NotFound delete SHALL be skipped as `already-absent`. A successful delete SHALL record the step as deleted. Any other error SHALL record the step as failed, with the failure class `forbidden`, `conflict` or `error` and the error's text. A live object handed back for a read that names a different object than the step, in any identity field the live object sets, SHALL NOT be judged or deleted: the step SHALL be recorded as failed with class `error` and a message naming both objects. A failure SHALL NOT stop the plan: the next step's action SHALL follow. Each call SHALL append, in step order, the outcomes of the steps it finished and only those, and SHALL NOT change an earlier outcome, so a frontend reports the outcomes appended since the state it passed in. Source: 0012:D1:R1, 0012:D4:R1.

#### Scenario: A gone object is already absent

- **WHEN** a read returns a NotFound error
- **THEN** the next action is a skip with reason `already-absent`

#### Scenario: An object gone before its delete is already absent

- **WHEN** a delete returns a NotFound error
- **THEN** the next action is a skip with reason `already-absent`
- **AND** the step is not recorded as deleted

#### Scenario: A wrapped Forbidden error is classified

- **WHEN** a delete returns a Forbidden API error wrapped with `fmt.Errorf("deleting: %w", err)`
- **THEN** the step's outcome is failed with failure class `forbidden`

#### Scenario: A failed precondition is a conflict

- **WHEN** a delete returns a Conflict API error
- **THEN** the step's outcome is failed with failure class `conflict`
- **AND** it is not recorded as deleted

#### Scenario: A failure does not stop the plan

- **WHEN** the read of the first of two steps fails with an Internal error
- **THEN** the next action reads the second step

#### Scenario: A live object for another step is not deleted

- **WHEN** the state awaits the read of `Deployment/app/web` and the live object handed back is `Deployment/app/api`
- **THEN** no delete is named for either object
- **AND** the step is recorded as failed with failure class `error`
- **AND** the next action names the next step

#### Scenario: An empty live namespace or kind is not a mismatch

- **WHEN** the step is a ClusterRole recorded with a namespace, and the live object handed back has no namespace and no `apiVersion` or `kind`
- **THEN** the live object is judged for the step

#### Scenario: One call can finish two steps

- **WHEN** the read of a Deployment step fails with an Internal error and the next step is a Namespace
- **THEN** that call appends a failed outcome for the Deployment and then a skipped outcome for the Namespace, in that order
- **AND** the action is a skip naming the Namespace step

### Requirement: The deletion state is a serialisable value the caller owns

The deletion state SHALL be a plain value with a defined JSON encoding, holding the next step, what the state awaits and one outcome per finished step. The zero state SHALL be the start of a plan. The library SHALL keep nothing between calls. A state written to JSON and read back SHALL advance identically to the state held in memory, and advancing the same plan, state and outcome twice SHALL name the same action and return equal states. The transition SHALL return an error, the input state unchanged and no action when the state cannot belong to the plan: its next step lies outside the plan, it records a number of outcomes other than its next step, an outcome is out of step order or carries a result other than deleted, skipped or failed, it awaits something after the last step, or what it awaits is not a defined value. The JSON encoding SHALL be fixed field by field, so that renaming a field fails a check. Source: 0012:D4:R5, ADR-008 rule 2.

#### Scenario: A round-tripped state advances identically

- **WHEN** a plan is advanced to done once with an in-memory state and once with a state written to JSON and read back before every call, through the same outcomes
- **THEN** both runs name the same actions in the same order
- **AND** their final states are equal

#### Scenario: Advancing twice names the same action

- **WHEN** the same plan, state and outcome are advanced twice
- **THEN** both calls return equal states and equal actions

#### Scenario: A state past the plan is refused

- **WHEN** the state's next step is beyond the plan's last step
- **THEN** the transition returns an error

#### Scenario: A state awaiting something after the last step is refused

- **WHEN** the state's next step equals the number of steps and the state awaits a read
- **THEN** the transition returns an error and the input state unchanged

#### Scenario: A state whose outcomes do not match its next step is refused

- **WHEN** the state names step 1 of a plan and records no outcome
- **THEN** the transition returns an error and the input state unchanged

#### Scenario: A state with an out-of-order or unknown outcome is refused

- **WHEN** the state names step 2 of a plan and its second outcome records step 9, or records a result other than deleted, skipped or failed
- **THEN** the transition returns an error and the input state unchanged

#### Scenario: The JSON encoding is fixed

- **WHEN** a state with a next step, an awaited read and a failed outcome carrying every field is written to JSON
- **THEN** the bytes equal `{"next":2,"awaiting":"read","outcomes":[{"step":0,"result":"failed","failure":"forbidden","message":"denied"},{"step":1,"result":"skipped","skip":"already-absent","message":"gone"}]}`

### Requirement: The hold verdict is decided from the policy and the plan's outcome

The library SHALL decide whether an instance's deletion hold may be released from the plan's policy, its steps, the state's outcomes and whether the caller can act as the deleting identity, and from nothing that names the hold's bearer or the frontend. It SHALL check in this order and stop at the first match. When the policy does not prune, it SHALL release as `prune-disabled`. When the plan has no steps, it SHALL release as `inventory-empty`. When the deleting identity is missing and the policy sets force-orphan, it SHALL release as `force-orphan`. When the identity is missing or could not be obtained, it SHALL hold as `identity-unavailable`. When the state cannot belong to the plan, for any reason the transition refuses it, it SHALL hold as `cleanup-incomplete` and never release. When the plan is not finished, it SHALL hold as `cleanup-incomplete`. When any step failed as Forbidden, it SHALL hold as `cleanup-forbidden`. When any other step failed, it SHALL hold as `cleanup-incomplete`. Otherwise it SHALL release as `cleanup-complete`, with skipped steps counting as complete. Each verdict SHALL carry the reason as the contract's literal and a message the library words. Source: 0012:D1:R5, 0012:D4:R1, the 0012 contract's `#HoldVerdict`.

#### Scenario: Pruning disabled releases

- **WHEN** the policy does not prune
- **THEN** the verdict is release with reason `prune-disabled`, whatever the identity and the state

#### Scenario: An empty inventory releases

- **WHEN** the policy prunes and the plan has no steps
- **THEN** the verdict is release with reason `inventory-empty`

#### Scenario: Force-orphan releases only for a missing identity

- **WHEN** the policy prunes and sets force-orphan, the plan has steps and the identity is missing
- **THEN** the verdict is release with reason `force-orphan`
- **AND** with the identity failed instead of missing, the verdict is hold with reason `identity-unavailable`

#### Scenario: A missing identity without force-orphan holds

- **WHEN** the policy prunes without force-orphan, the plan has steps and the identity is missing
- **THEN** the verdict is hold with reason `identity-unavailable`

#### Scenario: An unfinished plan holds

- **WHEN** the identity is available and the state has not reached done
- **THEN** the verdict is hold with reason `cleanup-incomplete`

#### Scenario: A state that cannot belong to the plan holds

- **WHEN** the policy prunes, the identity is available, and the state of a two-step plan names step 2 with no outcomes, or names step 5, or awaits a read at step 2
- **THEN** the verdict is hold with reason `cleanup-incomplete`

#### Scenario: A Forbidden failure wins over other failures

- **WHEN** the plan is finished and one step failed as `forbidden` and another as `error`
- **THEN** the verdict is hold with reason `cleanup-forbidden`

#### Scenario: Force-orphan does not lift a failed cleanup

- **WHEN** the policy sets force-orphan, the identity is available, and the finished plan has a step failed with class `error`
- **THEN** the verdict is hold with reason `cleanup-incomplete`

#### Scenario: A finished plan with skips releases

- **WHEN** the plan is finished, every step is deleted or skipped, and the identity is available
- **THEN** the verdict is release with reason `cleanup-complete`

#### Scenario: Reasons are the contract's literals

- **WHEN** a test reads the hold reason constants
- **THEN** they are `prune-disabled`, `inventory-empty`, `cleanup-complete`, `force-orphan`, `cleanup-incomplete`, `cleanup-forbidden` and `identity-unavailable`

### Requirement: The lifecycle package is pure and carries no hook semantics

`opm/k8s/lifecycle` SHALL perform no cluster read or write, read no clock or environment, start no goroutine and log nothing. It SHALL NOT change the entries, the live objects or the errors it is given. It SHALL name no step a module declares around a deletion: its plan holds only the inventory entries it was given. Its package documentation SHALL carry no ADR or enhancement reference, since it publishes into the Library reference. Source: 0012:D4:R6, ADR-008 rule 3, ADR-011.

#### Scenario: The package imports no clock, environment, logger or sync

- **WHEN** the direct imports of `opm/k8s/lifecycle` are listed
- **THEN** none of them is `os`, `time`, `log`, `log/slog` or `sync`

#### Scenario: The package starts no goroutine

- **WHEN** the package's non-test Go files are parsed
- **THEN** none contains a `go` statement

#### Scenario: The live object is not modified

- **WHEN** a live object is handed to the transition
- **THEN** it is deeply equal to a copy taken before the call

### Requirement: The weight table never contradicts Flux's staged apply order

The library SHALL keep a test that models the staged apply of Flux's `ssa` package at the version the operator pins, as literals without importing Flux: its stages (the CustomResourceDefinition of `apiextensions.k8s.io`, the core Namespace and the ClusterRole of `rbac.authorization.k8s.io`; then every kind whose name ends in `Class`; then the rest), its `ReconcileOrder` ranks (every kind it does not list at rank 0), and its tie-break by API group, then kind. The test's universe SHALL hold the library's table, Flux's listed kinds, a copy of each of those kinds in an API group that sorts before the built-in groups and in one that sorts after them, custom kinds in both such groups and a custom kind whose name ends in `Class`. The test SHALL fail when the library weighs any pair of the universe strictly opposite to the order Flux applies them in, naming the pair. A second test SHALL show that the comparison reports a contradiction injected through a modified weight function, and the test SHALL keep asserting that both orders put a CustomResourceDefinition before a Namespace. Source: 0012:D5:R1.

#### Scenario: The shipped table contradicts Flux nowhere

- **WHEN** the comparison runs over the universe with the library's `Weight`
- **THEN** it reports no pair

#### Scenario: A weight edit that creates a contradiction is caught

- **WHEN** the comparison runs with a weight function that moves Deployment below Service
- **THEN** it reports the Service and Deployment pair

#### Scenario: Flux's tie-break between unlisted kinds is counted

- **WHEN** the comparison runs with a weight function that weighs an `autoscaling` HorizontalPodAutoscaler above a `batch` Job
- **THEN** it reports the HorizontalPodAutoscaler and Job pair, because Flux orders them by group

### Requirement: The apply verdict refuses terminating, foreign, other-instance and adopted-elsewhere objects

The apply verdict SHALL decide in this order and stop at the first match. When there is no live object, it SHALL apply. When the live object has a deletion timestamp, it SHALL refuse as `terminating`, whether or not the object is in the instance's recorded inventory, and nothing SHALL lift that refusal. When the live object's adopt annotation equals the non-empty instance UUID, it SHALL apply. When the object is in the instance's recorded inventory: with an empty instance UUID it SHALL apply; when its adopt annotation is non-blank, it SHALL refuse as `adopted-elsewhere`; otherwise it SHALL apply, whatever its live UUID label, so an instance whose UUID changed keeps applying its own objects, except an object whose adopt annotation still names the old UUID, which applies again once re-annotated with the new UUID. Outside the inventory: when the live managed-by label is not an OPM runtime's value, it SHALL refuse as `foreign-object`, unless the operator install admission lifts it; when the adopt annotation is non-blank and equals the live UUID label, it SHALL refuse as `adopted-elsewhere`, since the instance it names completed the hand-over; when the live UUID label is non-empty and differs from the instance UUID, it SHALL refuse as `other-instance`; when the adopt annotation is non-blank, it SHALL refuse as `adopted-elsewhere`; otherwise it SHALL apply. Outside the inventory an empty instance UUID SHALL never match an adopt annotation, and any non-empty live UUID or non-blank adopt annotation SHALL then count as another instance's. Source: 0012:D8:R1/R2/R5, 0012:D8:R8 (enhancements#103), 0012:D1:R7, 0012:D4:R2.

The `ApplyInput.InInventory` doc and the package doc SHALL state that a frontend drops an object refused as `adopted-elsewhere` while it is in the instance's recorded inventory from the inventory it records next, keeps applying the instance's other objects, and never deletes the object for that refusal. The `adopted-elsewhere` message SHALL name the object and the instance its adopt annotation names, and SHALL name the annotation key `opmodel.dev/adopt` with the instance UUID to set it to, unless the instance UUID is empty, as the only way for this instance to take the object back. For an inventoried object it SHALL say that this instance no longer applies the object and drops it from its inventory. Source: 0012:D8:R3, 0012:D8:R8 (enhancements#103).

#### Scenario: A new object is applied

- **WHEN** the apply verdict is asked for an object with no live object
- **THEN** it applies

#### Scenario: A terminating object is refused everywhere

- **WHEN** the live object has a deletion timestamp
- **THEN** the verdict refuses as `terminating` when the object is in the inventory, when it is outside it, when it carries the adopt annotation naming this instance, when it carries the adopt annotation naming another instance, and when it is admitted

#### Scenario: An inventoried foreign object is applied

- **WHEN** the object is in the instance's recorded inventory, its live managed-by label is `helm`, and it carries no UUID label and no adopt annotation
- **THEN** the verdict applies

#### Scenario: An inventoried object of this instance is applied

- **WHEN** the object is in the instance's recorded inventory and its live UUID label equals the instance UUID, with no adopt annotation
- **THEN** the verdict applies

#### Scenario: An inventoried object annotated for another instance is refused

- **WHEN** the object is in the instance's recorded inventory, carries this instance's UUID label `u-9`, and its adopt annotation is `u-1`, for instance UUID `u-9`
- **THEN** the verdict refuses as `adopted-elsewhere`
- **AND** the message is `Deployment/web/api was adopted by module instance u-1; this instance no longer applies it and drops it from its inventory; to take it back, annotate it opmodel.dev/adopt=u-9`

#### Scenario: An inventoried object labelled for another instance without an annotation is applied

- **WHEN** the object is in the instance's recorded inventory and its live UUID label is `u-1`, with no adopt annotation, for instance UUID `u-9`
- **THEN** the verdict applies

#### Scenario: An inventoried object the adopter has taken is refused

- **WHEN** the object is in the instance's recorded inventory, its live UUID label is `u-1`, and its adopt annotation is `u-1`, for instance UUID `u-9`
- **THEN** the verdict refuses as `adopted-elsewhere`

#### Scenario: An annotation naming this instance takes an inventoried object back

- **WHEN** the object is in the instance's recorded inventory, its live UUID label is `u-1`, and its adopt annotation is `u-9` with surrounding whitespace, for instance UUID `u-9`
- **THEN** the verdict applies

#### Scenario: An empty instance UUID applies an inventoried object

- **WHEN** the instance UUID is empty and the object in the inventory carries a UUID label and an adopt annotation
- **THEN** the verdict applies

#### Scenario: A foreign object outside the inventory is refused

- **WHEN** the object is outside the inventory and its live managed-by label is missing or is `kustomize`
- **THEN** the verdict refuses as `foreign-object`

#### Scenario: Another instance's object outside the inventory is refused

- **WHEN** the object is outside the inventory, OPM-managed, and its UUID label differs from the instance UUID
- **THEN** the verdict refuses as `other-instance`

#### Scenario: A handed-over object stays adopted-elsewhere after the adopter applies

- **WHEN** the object is outside the inventory, OPM-managed, its live UUID label is `u-1`, and its adopt annotation is `u-1`, for instance UUID `u-9`
- **THEN** the verdict refuses as `adopted-elsewhere`, not `other-instance`
- **AND** the message is `Deployment/web/api was adopted by module instance u-1; this instance does not apply it; to let this instance take it back, annotate it opmodel.dev/adopt=u-9`

#### Scenario: An annotation naming another instance than the label keeps other-instance

- **WHEN** the object is outside the inventory, OPM-managed, its live UUID label is `u-1`, and its adopt annotation is `u-7`, for instance UUID `u-9`
- **THEN** the verdict refuses as `other-instance`

#### Scenario: A dropped object is not taken back on the next apply

- **WHEN** the object is outside the inventory, OPM-managed, carries this instance's UUID label, and its adopt annotation names a different instance
- **THEN** the verdict refuses as `adopted-elsewhere`
- **AND** the message does not say the instance drops it from its inventory

#### Scenario: An OPM object without a UUID label is applied

- **WHEN** the object is outside the inventory, OPM-managed, and carries no UUID label and no adopt annotation
- **THEN** the verdict applies

#### Scenario: An OPM object of this instance outside the inventory is applied

- **WHEN** the object is outside the inventory, OPM-managed, its UUID label equals the instance UUID, and it carries no adopt annotation
- **THEN** the verdict applies

#### Scenario: An empty instance UUID fails closed outside the inventory

- **WHEN** the instance UUID is empty and the object outside the inventory is OPM-managed with a non-empty UUID label
- **THEN** the verdict refuses as `other-instance`

#### Scenario: An empty instance UUID refuses an annotated object outside the inventory

- **WHEN** the instance UUID is empty and the object outside the inventory is OPM-managed, carries no UUID label, and its adopt annotation is `u-1`
- **THEN** the verdict refuses as `adopted-elsewhere`
- **AND** the message is `Deployment/web/api is being adopted by module instance u-1; this instance does not apply it`, with no remedy clause

#### Scenario: The docs state the frontend's part of the hand-over

- **WHEN** the `ApplyInput.InInventory` doc comment and the package doc are read
- **THEN** each says a frontend drops an object refused as `adopted-elsewhere` from the inventory it records next and never deletes it for that refusal
