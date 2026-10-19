## MODIFIED Requirements

### Requirement: The Kubernetes tier imports no cluster client or controller framework
Beyond the standard library and the CUE SDK that the kernel's output types carry, a file under `opm/k8s/` MAY import only the kernel's exported packages and `k8s.io/apimachinery`. No file under `opm/k8s/` SHALL import `k8s.io/client-go`, `sigs.k8s.io/controller-runtime`, any package under `github.com/fluxcd/` or any cluster client. Source: enhancement 0012 (the fenced Kubernetes tier).

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

### Requirement: Deletion plans come from the inventory and the live objects
An instance's deletion plan (uninstall) SHALL be built from the persisted inventory plus the live objects in the cluster. Building it SHALL NOT need a render and SHALL NOT read a plan stored at apply time. ADR-008 SHALL record this. Source: ADR-008; ADR-013, decision j5; the deletion protocol is enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply").

#### Scenario: ADR-008 records the deletion-plan source

- **WHEN** a developer reads `adr/008-kernel-plans-caller-runs.md`
- **THEN** it states that an instance's deletion plan (uninstall) is built from the persisted inventory plus the live objects, with no render and no stored plan, and that a prune's stale set is computed by the inventory package and handed to the same plan
- **AND** it lists planning from a render and lifecycle transition detection among the questions it does not decide, deferred to the 0009 hook work, distinct from the deletion step transition the tier owns

### Requirement: One weight table orders apply and delete
The library SHALL keep one kind-class weight table in `opm/k8s/object`. The weights SHALL be those that scenario "The weights of the table" lists. Source: enhancement 0012 ("Ordering is kind-class order in the Kubernetes tier; no module-internal ordering is planned, and cross-module order belongs to a future Bundle").

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

#### Scenario: The weights of the table

- **WHEN** the weight of each kind is looked up
- **THEN** a CustomResourceDefinition weighs -100
- **AND** a Namespace weighs 0
- **AND** ClusterRoles weigh 5
- **AND** PriorityClasses, RuntimeClasses, StorageClasses, VolumeSnapshotClasses, IngressClasses, GatewayClasses, ClusterClasses and every other kind whose name ends in `Class` weigh 6
- **AND** ClusterRoleBindings weigh 7
- **AND** ResourceQuotas weigh 8
- **AND** ServiceAccounts, Roles and RoleBindings weigh 10
- **AND** Secrets and ConfigMaps weigh 15
- **AND** Services weigh 50
- **AND** LimitRanges weigh 60
- **AND** Deployments and StatefulSets weigh 100
- **AND** CronJobs weigh 105
- **AND** PodDisruptionBudgets weigh 108
- **AND** Validating and mutating webhook configurations weigh 2000
- **AND** every other kind, among them PersistentVolumes, PersistentVolumeClaims, DaemonSets, ReplicaSets, Jobs, Ingresses, NetworkPolicies, the pod autoscalers and every custom resource, weighs 1000

### Requirement: Apply stages follow the weight table
The library SHALL provide a function that returns a sorted copy of an apply set cut into stages. The first stage holds the cluster definitions, which are every CustomResourceDefinition of `apiextensions.k8s.io` and every Namespace of the core group, and is marked as such. It is omitted when there are none. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply").

#### Scenario: Stages for a typical module

- **WHEN** a CustomResourceDefinition, a Namespace, a ConfigMap, a Secret, a Deployment and a Service are staged
- **THEN** the stages are the cluster definitions (the CustomResourceDefinition and the Namespace), then the ConfigMap and the Secret, then the Service, then the Deployment

#### Scenario: A Namespace kind in another group is not a cluster definition

- **WHEN** an object of kind `Namespace` in group `example.com` is staged beside a core Namespace
- **THEN** only the core Namespace is in the cluster-definition stage

#### Scenario: No cluster definitions, no definition stage

- **WHEN** a set holding only a Deployment and a Service is staged
- **THEN** two stages are returned, neither marked as cluster definitions

### Requirement: The weight table never contradicts Flux's staged apply order
The library SHALL keep a test that models the staged apply of Flux's `ssa` package at the version the operator pins, as literals without importing Flux: its stages as scenario "The model of Flux's stages" lists, its `ReconcileOrder` ranks (every kind it does not list at rank 0), and its tie-break by API group, then kind. Source: enhancement 0012 ("Ordering is kind-class order in the Kubernetes tier; no module-internal ordering is planned, and cross-module order belongs to a future Bundle").

#### Scenario: The shipped table contradicts Flux nowhere

- **WHEN** the comparison runs over the universe with the library's `Weight`
- **THEN** it reports no pair

#### Scenario: A weight edit that creates a contradiction is caught

- **WHEN** the comparison runs with a weight function that moves Deployment below Service
- **THEN** it reports the Service and Deployment pair

#### Scenario: Flux's tie-break between unlisted kinds is counted

- **WHEN** the comparison runs with a weight function that weighs an `autoscaling` HorizontalPodAutoscaler above a `batch` Job
- **THEN** it reports the HorizontalPodAutoscaler and Job pair, because Flux orders them by group

#### Scenario: The model of Flux's stages

- **WHEN** the model of Flux's staged apply is read
- **THEN** its first stage holds the CustomResourceDefinition of `apiextensions.k8s.io`, the core Namespace and the ClusterRole of `rbac.authorization.k8s.io`
- **AND** its second stage holds every kind whose name ends in `Class`
- **AND** its third stage holds the rest

### Requirement: The ownership verdicts take no input that only lifts a refusal
`ownership.ApplyInput` and `ownership.DeleteInput` SHALL have exactly the fields that scenario "The verdict inputs carry no admission field" lists. Neither SHALL carry a field whose only effect is to lift a refusal or a skip. Source: enhancement 0012 ("The apply guard runs on every apply, refusing foreign objects outside the inventory and any object adopted by another instance; a per-object adopt annotation is its only override") are withdrawn.

#### Scenario: The verdict inputs carry no admission field

- **WHEN** a test lists the fields of `ownership.ApplyInput` and `ownership.DeleteInput`
- **THEN** they are `Object`, `Live`, `InInventory`, `InstanceUUID` and `Object`, `Live`, `InstanceUUID`

#### Scenario: An earlier install manifest's object is refused on apply

- **WHEN** the apply verdict is asked for `Namespace/opm-operator-system`, live with managed-by `kustomize` and no UUID label, outside the inventory
- **THEN** it refuses as `foreign-object`
- **AND** the message names `opmodel.dev/adopt` with the instance UUID

#### Scenario: An earlier install manifest's Deployment is not deleted

- **WHEN** the delete verdict is asked for a live `apps` Deployment with managed-by `kustomize` and no UUID label
- **THEN** it skips as `not-opm-managed`

## REMOVED Requirements

### Requirement: The Kubernetes tier lives at opm/k8s beside the kernel
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The Kubernetes tier lives under opm/k8s outside the kernel and the helper tier", "No kernel or helper package implements a decision of the Kubernetes tier", "The Kubernetes tier is part of the library's Go module".

### Requirement: Nothing in the library performs a cluster action
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The library performs no cluster action and ships no cluster executor", "ADR-008 allows only executor backends that perform no cluster action", "The frontend performs each deletion action and the tier owns the deletion sequence", "The library owns the apply verdict and the order and never an apply engine", "Each frontend submits objects in the library order".

### Requirement: Kind-class order is a Kubernetes fact that lives in the tier
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The kind-class apply order lives in the Kubernetes tier", "No module-internal ordering is planned", "Ordering across modules belongs to a future Bundle", "A frontend's readiness gating is outside the cross-module ordering rule" and "ADR-008 rule 4 excludes module-specific ordering only".

### Requirement: The label vocabulary names labels and never stamps them
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The labels package imports only the standard library", "The labels package declares the label keys and values both frontends use", "A predicate recognises the OPM managed-by values", "The labels package never stamps labels".

### Requirement: Compiled objects convert to Kubernetes objects through one export
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "A Resource carries a rendered object and its provenance", "A Resource exposes the object's fields and its JSON and unstructured forms", "One export returns each Resource's JSON, object and provenance", "The export leaves the input Resources alone", "A failing export names the Resource, its position and the step".

### Requirement: Readiness evaluation is a pure function over fetched objects
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Readiness is judged by a pure function over fetched objects", "Lint holds the health package to the standard library and apimachinery", "Readiness statuses and evaluation rules are the cli's", "A Deployment is ready only when its rollout is complete", "A StatefulSet is ready by its update strategy", "A StatefulSet is ready when its updated replicas and revisions agree", "A DaemonSet and a Job are judged by their counts and conditions", "A CronJob and the passive kinds are applied", "A claim reports its phase and any other kind reports its Ready condition", "Four statuses are healthy and every other string is unhealthy", "The aggregate counts missing and unreadable objects against ready" and "ProgressDeadlineExceeded tells a stalled rollout from one in progress".

### Requirement: The ownership verdicts are pure
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The ownership verdicts are pure functions of explicit inputs", "The ownership package performs no I/O and never modifies the live object", "Ownership inputs name the object and carry the live object".

### Requirement: The delete verdict skips with a reason or proceeds with the judged object's identity
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The delete verdict decides in a fixed order", "Empty UUIDs and blank annotations pass the delete verdict", "An annotation naming this instance and a deleting object pass the delete verdict", "A prune or an instance deletion leaves an object handed to another instance", "A proceed verdict carries the judged object's identity" and "A skip verdict yields no precondition and carries a message".

### Requirement: The adopt annotation is the only override and the refusal names it
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "An adopt annotation naming the instance lifts an ownership refusal", "An adopt annotation with any other value lifts nothing", "A refusal names the adopt annotation key and the UUID to set", "The other-instance refusal asks to remove the object from its owner first", "No refusal message names another way past the refusal".

### Requirement: Inventory entries are plain values built from objects
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "An inventory entry has exactly six fields and a constructor from an object", "The entry constructor reads identity from the object and component from its label", "The entry type carries no struct tags".

### Requirement: The stale set is component-blind
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The stale set compares group, kind, namespace and name only", "The stale set keeps the previous order and is never nil", "The package exports one identity comparison and no component-aware one".

### Requirement: The inventory digest hashes a canonical field encoding
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The inventory digest is sha256 over a tagged, length-prefixed encoding", "Each entry of the inventory digest is written with length-prefixed fields", "The inventory digest depends only on the entries' field values" and "The empty inventory hashes the tag line and a new encoding uses a new tag line".

### Requirement: The render digest ignores only the managed-by label value
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The render digest reads only exported JSON", "The render digest blanks the managed-by value and sorts object keys", "The render digest encodes strings as Go's JSON encoder does with HTML escaping off", "The render digest orders the encoded objects and hashes them after a tag line", "Sets that differ only in the managed-by value have the same render digest", "The cli and the operator compute the same render digest for one render", "Any other difference changes the render digest", "The render digest leaves its input unchanged and fails on a malformed object with its position".

### Requirement: The deletion plan orders inventory entries by kind-class delete order
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The library provides a deletion plan built from inventory entries, a policy and a UUID", "The deletion plan orders its entries by delete weight", "Protected kinds are marked safety-excluded in the plan before any live read", "A deletion plan needs no render and no stored plan", "The plan constructor is the only way to build a plan with steps".

### Requirement: The deletion transition names one action per call
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The deletion transition returns the next state and one action", "The deletion transition performs no action and names only steps of the plan", "A safety-excluded step is skipped without a read", "Every other step is read, judged, and then skipped or deleted", "A plan that does not prune names done at once", "A prune leaves an object that another instance adopted".

### Requirement: Action outcomes are classified by the library
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The library classifies the raw error of each read or delete", "A NotFound read or delete is already absent", "A successful delete is recorded as deleted and any other error as failed", "A mismatched live object is never judged or deleted", "A failure does not stop the plan", "Each call appends the outcomes of the steps it finished".

### Requirement: The deletion state is a serialisable value the caller owns
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The deletion state is a plain value the caller owns", "A state written to JSON advances identically", "A state that cannot belong to the plan is refused", "A state that awaits something it cannot await is refused", "The JSON encoding of the deletion state is fixed field by field".

### Requirement: The hold verdict is decided from the policy and the plan's outcome
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The hold verdict is decided from the policy, the plan's outcome and the identity", "Each hold verdict carries its reason and a message".

### Requirement: The apply verdict refuses terminating, foreign, other-instance and adopted-elsewhere objects
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The apply verdict decides in a fixed order", "Nothing lifts a terminating refusal", "An instance whose UUID changed keeps applying its own inventoried objects", "Outside the inventory an empty instance UUID fails closed", "The docs state the frontend's part of the hand-over", "The adopted-elsewhere message names the object, the adopter and the way back" and "The adopted-elsewhere message for an inventoried object says it is dropped".

## ADDED Requirements

### Requirement: The Kubernetes tier lives under opm/k8s outside the kernel and the helper tier
The library SHALL place the Kubernetes decisions OPM makes under `opm/k8s/`, a tier outside the kernel and outside the opt-in helper tier. The tier holds inventory entry construction, stale-set computation, digests, the prune and ownership guards at apply and delete time, deletion ordering, the deletion hold protocol, Compiled-to-object conversion, the label vocabulary, kind-class apply order and readiness evaluation. Source: enhancement 0012 (the fenced Kubernetes tier).

#### Scenario: The tier sits beside the kernel and the helper tier

- **WHEN** a developer lists the packages that decide inventory entry construction, stale-set computation, digests, the prune and ownership guards, deletion ordering, the deletion hold protocol, Compiled-to-object conversion, the label vocabulary, kind-class apply order or readiness evaluation
- **THEN** each is under `opm/k8s/`, outside `opm/kernel/` and outside `opm/helper/`

### Requirement: No kernel or helper package implements a decision of the Kubernetes tier
No package under `opm/kernel/` or `opm/helper/` SHALL implement any decision that the Kubernetes tier holds (inventory entry construction, stale-set computation, digests, the prune and ownership guards at apply and delete time, deletion ordering, the deletion hold protocol, Compiled-to-object conversion, the label vocabulary, kind-class apply order, readiness evaluation) once the `opm/k8s/` package that owns it exists. Source: enhancement 0012 (the fenced Kubernetes tier).

#### Scenario: A kernel or helper package does not decide for the tier

- **WHEN** the `opm/k8s/` package that owns one of these decisions exists
- **THEN** no package under `opm/kernel/` or `opm/helper/` implements that decision

### Requirement: The Kubernetes tier is part of the library's Go module
The tier SHALL be part of the library's Go module, `github.com/open-platform-model/library`, and SHALL NOT carry a `go.mod` of its own. ADR-011 records the placement. Source: enhancement 0012 (the fenced Kubernetes tier).

#### Scenario: The placement is recorded before any package exists

- **WHEN** a developer reads `adr/011-kubernetes-tier-beside-the-kernel.md`
- **THEN** it names `opm/k8s/*` as the Kubernetes tier, states that the tier is outside `opm/kernel` and outside `opm/helper`, and names placement inside the kernel, under the helper tier, in a nested Go module and in per-frontend copies as rejected alternatives with reasons

#### Scenario: The tier is not a nested module

- **WHEN** a developer lists every `go.mod` file tracked in the repository
- **THEN** exactly one is found, at the repository root, and none is under `opm/k8s/`

### Requirement: The Kubernetes tier imports no internal or helper package and one Kubernetes module
No file under `opm/k8s/` SHALL import a package under `opm/internal/` or `opm/helper/`, and of the Kubernetes modules it SHALL import only `k8s.io/apimachinery`. Source: enhancement 0012 (the fenced Kubernetes tier).

#### Scenario: The tier imports only apimachinery of the Kubernetes modules

- **WHEN** a file under `opm/k8s/` imports `k8s.io/apimachinery` together with `opm/kernel`
- **THEN** `task lint` passes it, and it fails an import of a package under `opm/internal/` or `opm/helper/`

### Requirement: The lint gate enforces the import denials of the Kubernetes tier
The repository lint gate SHALL enforce the import denials of `opm/k8s/` (no `k8s.io/client-go`, `sigs.k8s.io/controller-runtime`, package under `github.com/fluxcd/`, cluster client, or package under `opm/internal/` or `opm/helper/`, and of the Kubernetes modules only `k8s.io/apimachinery`), including the refusal of every other `k8s.io` and `sigs.k8s.io` module. Source: enhancement 0012 (the fenced Kubernetes tier).

#### Scenario: Lint refuses every other Kubernetes module

- **WHEN** a file under `opm/k8s/` imports a `k8s.io` or `sigs.k8s.io` module other than `k8s.io/apimachinery`
- **THEN** `task lint` fails naming the forbidden import

### Requirement: Non-test files of the Kubernetes tier import from a strict allow list
For non-test files under `opm/k8s/` the repository lint gate SHALL also enforce the allowed set as a strict allow list: the Go standard library, `cuelang.org/go/cue` and its subpackages (not sibling modules such as `cuelang.org/go/cuego`), `k8s.io/apimachinery` and the library's own `opm/` packages, so any other third-party import is refused without a reviewer having to notice it. Source: enhancement 0012 (the fenced Kubernetes tier).

#### Scenario: Lint refuses a sibling CUE module in the tier

- **WHEN** a change adds an import of `cuelang.org/go/cuego` to a non-test file under `opm/k8s/`
- **THEN** `task lint` fails naming the import

### Requirement: One test file of the Kubernetes tier imports the deprecated objectset
The one exception to the `opm/helper/` import denial of `opm/k8s/` SHALL be the test file `opm/k8s/object/objectset_parity_test.go`, which imports the deprecated `opm/helper/objectset` to check the duplicate-object-identities parity requirement; the change that removes `opm/helper/objectset` deletes it. Source: enhancement 0012 (the fenced Kubernetes tier).

#### Scenario: The parity test file is the one exception

- **WHEN** a file under `opm/k8s/` imports the deprecated `opm/helper/objectset`
- **THEN** the file is `opm/k8s/object/objectset_parity_test.go`, which checks the duplicate-object-identities parity requirement

### Requirement: The library performs no cluster action and ships no cluster executor
Nothing in the library SHALL perform a cluster action, and the Kubernetes tier SHALL perform no cluster I/O. No executor (a loop that drives a plan to completion) and no executor backend that performs a planned action against a cluster SHALL ship anywhere under `opm/`: not in the kernel, not under `opm/k8s/` and not under `opm/helper/`. Source: enhancement 0012 ("The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module").

#### Scenario: No executor loop or cluster backend ships anywhere in the library

- **WHEN** a developer searches every package under `opm/` (the kernel, `opm/k8s/` and `opm/helper/`) for a function that takes a cluster client, a REST config or a controller-runtime client, that performs a planned action against a cluster, or that drives a plan to completion
- **THEN** none exists

### Requirement: ADR-008 allows only executor backends that perform no cluster action
ADR-008 rule 3's allowance for opt-in executor backends under `opm/helper/` SHALL be narrowed to executor backends that perform no planned action against a cluster (0009's wasm, HTTP, `cue.eval` and local container hosts). Source: enhancement 0012 ("The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module").
**Source**: Enhancement 0012 ("The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module") amends enhancement 0009 ("Executor backends ship in the library's opt-in layer; frontends compose them à la carte") so its cluster-acting Ops are performed by the frontend.

#### Scenario: ADR-008 allows only non-cluster executor backends

- **WHEN** a developer reads rule 3 of ADR-008
- **THEN** it states that no executor backend that performs a planned action against a cluster ships in the library, `opm/helper/` included, that each frontend performs such an action with its own client, and that opt-in executor backends that perform no planned action against a cluster may ship under `opm/helper/`

### Requirement: The frontend performs each deletion action and the tier owns the deletion sequence
A deletion plan SHALL advance one action per call from a serialisable state the caller owns, and the frontend SHALL perform each action with its own client, in a loop it writes itself. The tier SHALL own the whole deletion sequence (plan, transition, hold verdict). ADR-008 rules 1 to 3 apply to the tier. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply").

#### Scenario: The caller performs each deletion action

- **WHEN** a deletion plan is advanced
- **THEN** each call names one action from a serialisable state the caller owns, and the frontend performs that action with its own client in a loop it writes itself

### Requirement: The library owns the apply verdict and the order and never an apply engine
For apply, the tier SHALL own the per-object verdict (CanApply) and the order, never an apply engine. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply").

#### Scenario: Apply engines stay with the frontends

- **WHEN** a developer reads ADR-011
- **THEN** it states that the operator keeps `fluxcd/pkg/ssa` and the CLI keeps its own server-side apply, that the library owns the per-object apply verdict and the order and never an apply engine, and that an engine's staging may refine the library's order and never contradict it

### Requirement: Each frontend submits objects in the library order
Each frontend SHALL submit objects in the library's order. An engine's own staging MAY refine that order, for example by sorting within a stage, and SHALL NOT contradict it. Source: enhancement 0012 ("Ordering is kind-class order in the Kubernetes tier; no module-internal ordering is planned, and cross-module order belongs to a future Bundle").

#### Scenario: An engine sorts within a stage

- **WHEN** a frontend's apply engine sorts objects within a stage
- **THEN** the order refines the library's order and does not contradict it

### Requirement: The kind-class apply order lives in the Kubernetes tier
The library SHALL keep one kind-class apply order (a CustomResourceDefinition before its custom resources, a Namespace before namespaced objects). It SHALL live in the Kubernetes tier, and the kernel SHALL hold none. Source: enhancement 0012 ("Ordering is kind-class order in the Kubernetes tier; no module-internal ordering is planned, and cross-module order belongs to a future Bundle").

#### Scenario: The kernel holds no apply order

- **WHEN** the kind-class apply order is looked for
- **THEN** it is in the Kubernetes tier and the kernel holds none

### Requirement: No module-internal ordering is planned
No module-internal ordering (`dependsOn` edges, phases within one module) is planned: the library SHALL add no module-specific ordering of its own, and any module-internal ordering added later SHALL come off the CUE build as data. Source: enhancement 0012 ("Ordering is kind-class order in the Kubernetes tier; no module-internal ordering is planned, and cross-module order belongs to a future Bundle").

#### Scenario: No module-internal ordering is planned and cross-module order belongs to a Bundle

- **WHEN** a developer reads rule 4 of ADR-008 and item 7 of ADR-011
- **THEN** both state that no module-internal ordering (`dependsOn` edges, phases) is planned and that any added later comes off the build, that lifecycle hooks remain the parked 0009 work, and that in the OPM model ordering across modules belongs to a future Bundle definition using the order its modules are defined in
- **AND** neither presents module-internal `dependsOn` edges or phases as planned work

### Requirement: Ordering across modules belongs to a future Bundle
In the OPM model, ordering across modules belongs to a future Bundle definition, which uses the order its modules are defined in; the library SHALL add none before then. Source: enhancement 0012 ("Ordering is kind-class order in the Kubernetes tier; no module-internal ordering is planned, and cross-module order belongs to a future Bundle").

#### Scenario: The library adds no cross-module ordering

- **WHEN** the library orders the objects of several modules
- **THEN** it adds no ordering across modules before a Bundle definition exists

### Requirement: A frontend's readiness gating is outside the cross-module ordering rule
The rule is the one of requirement "Ordering across modules belongs to a future Bundle". A frontend's readiness gating between its own custom resources, such as the operator's ModulePackage `spec.dependsOn`, SHALL be outside this rule; whether a future Bundle definition subsumes it is not decided. Source: enhancement 0012 ("Ordering is kind-class order in the Kubernetes tier; no module-internal ordering is planned, and cross-module order belongs to a future Bundle").

#### Scenario: Readiness gating between custom resources is outside the rule

- **WHEN** a frontend gates readiness between its own custom resources, such as the operator's ModulePackage `spec.dependsOn`
- **THEN** the gating is outside the rule, and whether a future Bundle definition subsumes it is not decided

### Requirement: ADR-008 rule 4 excludes module-specific ordering only
ADR-008 rule 4's "The kernel derives no ordering of its own" SHALL be read as excluding module-specific ordering only, and ADR-008 and ADR-011 SHALL say so in their own text. Lifecycle hooks remain the parked 0009 work (an open question of enhancement 0009), carried as build data under ADR-008 rule 4. Source: enhancement 0012 ("Ordering is kind-class order in the Kubernetes tier; no module-internal ordering is planned, and cross-module order belongs to a future Bundle").

#### Scenario: ADR-008 carries the clarification

- **WHEN** a developer reads `adr/008-kernel-plans-caller-runs.md`
- **THEN** its Status records the 2026-10-02 amendment pointing at ADR-011, and the rule ending "The kernel derives no ordering of its own" states that it excludes module-specific ordering only

### Requirement: A prune hands its stale set to the deletion plan
A prune's stale set SHALL be computed by the inventory package and handed to the same plan. Source: ADR-008; ADR-013, decision j5; the deletion protocol is enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply").

#### Scenario: A prune builds the plan from the stale set

- **WHEN** a prune builds its deletion plan
- **THEN** the stale set was computed by the inventory package and is handed to the same plan

### Requirement: ADR-008 records the deferral of render planning and transition detection
ADR-008 SHALL also record that planning from a render and lifecycle transition detection (install, upgrade, reconfigure, no-op, uninstall) are deferred to the lifecycle hook work of enhancement 0009. This deferral does not touch the deletion step transition, which the tier owns as part of the deletion sequence. Source: ADR-008; ADR-013, decision j5; the deletion protocol is enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply").

#### Scenario: ADR-008 records the deferral

- **WHEN** a developer reads `adr/008-kernel-plans-caller-runs.md`
- **THEN** it lists planning from a render and lifecycle transition detection among the questions it does not decide, deferred to the 0009 hook work, distinct from the deletion step transition the tier owns

### Requirement: The labels package imports only the standard library
The library SHALL provide `opm/k8s/labels`, which imports nothing outside the Go standard library. The repository lint gate SHALL refuse any non-test import into the package from outside the standard library. Source: enhancement 0012 ("Labels are stamped by the CUE render; the shared render digest ignores the runtime-name label value" and "The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module").

#### Scenario: Lint keeps the label package dependency-free

- **WHEN** a change adds an import of `k8s.io/apimachinery`, `cuelang.org/go/cue` or a library package to a non-test file under `opm/k8s/labels/`
- **THEN** `task lint` fails naming the import

### Requirement: The labels package declares the label keys and values both frontends use
`opm/k8s/labels` SHALL declare the label keys and values both Kubernetes frontends read or write, with values byte-equal to the frontends' copies, as scenario "The package declares these keys and values" lists. The constant for `opmodel.dev/component` SHALL be documented as the OPM object category, not a component name. Source: enhancement 0012 ("Labels are stamped by the CUE render; the shared render digest ignores the runtime-name label value").

#### Scenario: The vocabulary matches the frontends' copies

- **WHEN** a table test pins each declared key and value to its literal
- **THEN** each equals the literal, and the literals are the ones a scripted comparison found byte-equal to the cli's `pkg/core/labels.go` at cli `1338e700` and the operator's at opm-operator `9b83611`

#### Scenario: The package declares these keys and values

- **WHEN** a developer lists the keys and values that `opm/k8s/labels` declares
- **THEN** they are the managed-by key `app.kubernetes.io/managed-by`, its runtime values `opm-cli` and `opm-controller`, its legacy value `open-platform-model`, `opmodel.dev/component`, `component.opmodel.dev/name`, `module-instance.opmodel.dev/name`, `module-instance.opmodel.dev/namespace` and `module-instance.opmodel.dev/uuid`
- **AND** the constant for `opmodel.dev/component` is documented as the OPM object category, not a component name

### Requirement: A predicate recognises the OPM managed-by values
`opm/k8s/labels` SHALL provide a predicate that reports whether a managed-by value identifies an OPM runtime, which accepts exactly the two runtime values (`opm-cli` and `opm-controller`) and the legacy value (`open-platform-model`). Source: enhancement 0012 ("Labels are stamped by the CUE render; the shared render digest ignores the runtime-name label value" and "The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module").

#### Scenario: Every OPM managed-by value is recognised

- **WHEN** the predicate is called with `opm-cli`, `opm-controller` or `open-platform-model`
- **THEN** it reports true for each
- **AND** it reports false for `helm`, for the empty string and for `OPM-CLI`

### Requirement: The labels package never stamps labels
No function in `opm/k8s/labels` SHALL set a label on an object. Labels on rendered objects are stamped by the CUE build at render. Source: enhancement 0012 ("Labels are stamped by the CUE render; the shared render digest ignores the runtime-name label value" and "The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module").

#### Scenario: No function in the labels package sets a label

- **WHEN** the functions of `opm/k8s/labels` are listed
- **THEN** none sets a label on an object, and labels on rendered objects come from the CUE build at render

### Requirement: A Resource carries a rendered object and its provenance
The library SHALL provide, in `opm/k8s/object`, a `Resource` that carries a rendered object's CUE value and its instance, component and transformer provenance, together with constructors from the kernel's `*kernel.Compiled`. A nil `Compiled` SHALL yield no Resource. Source: enhancement 0012 ("The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module").

#### Scenario: A nil Compiled yields no Resource

- **WHEN** a Resource is constructed from a nil `Compiled`
- **THEN** no Resource is yielded

### Requirement: A Resource exposes the object's fields and its JSON and unstructured forms
`Resource` SHALL expose the object's kind, name, namespace, apiVersion, group-version-kind, labels and annotations, each empty or nil when absent rather than an error. It SHALL also expose its JSON form and its `*unstructured.Unstructured` form. Source: enhancement 0012 ("The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module").

#### Scenario: Accessors are best-effort

- **WHEN** a Resource's value has no `metadata.namespace` and no `metadata.annotations`
- **THEN** its namespace is the empty string and its annotations are nil, with no error

### Requirement: One export returns each Resource's JSON, object and provenance
The library SHALL provide an export of a list of Resources that exports each Resource's value from CUE once and returns, in input order, that JSON, the object decoded from those same bytes, and the provenance. Source: enhancement 0012 ("The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module").

#### Scenario: One export feeds every consumer of the object

- **WHEN** a list of Resources built from compiled objects is exported
- **THEN** each result's JSON equals that Resource's own JSON form, and each result's object equals what that JSON decodes to
- **AND** each result carries its Resource's instance, component and transformer, in input order

### Requirement: The export leaves the input Resources alone
The export (Requirement: One export returns each Resource's JSON, object and provenance) SHALL NOT change or release the input Resources. A Resource keeps its whole CUE build alive; releasing it after export is the caller's responsibility.
**Source**: enhancement 0012 ("The Kubernetes runtime surface homes in the library's Kubernetes tier (supersedes only the placement conclusion of the CLI inventory and handoff enhancement)" and "The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module").

#### Scenario: The input survives the export

- **WHEN** a list of Resources is exported
- **THEN** the list and each Resource's value, instance, component and transformer are unchanged afterwards

### Requirement: A failing export names the Resource, its position and the step
A failure (Requirement: One export returns each Resource's JSON, object and provenance) SHALL name the failing Resource, its position, and whether the CUE export or the JSON decode failed. A value that exports to JSON that is not an object, `null` included, SHALL fail as a decode failure, so every exported object carries a non-nil object map. Source: enhancement 0012 ("The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module").

#### Scenario: A failing export names the resource and the step

- **WHEN** one Resource in the list carries a CUE value that cannot be exported to JSON
- **THEN** the export fails with an error naming that Resource and its position and reporting that the CUE export failed

#### Scenario: A value that is not an object fails at the decode

- **WHEN** one Resource in the list carries a concrete value that exports as a JSON list, a JSON string or `null`
- **THEN** the export fails with an error naming that Resource and its position and reporting that the JSON decode failed

### Requirement: The weight table agrees with the staged apply order of Flux
The weight table SHALL agree with the staged apply order of Flux's `ssa` package at the version the operator pins (v0.77.0), so that an engine applying through Flux only refines it. Source: enhancement 0012 ("Ordering is kind-class order in the Kubernetes tier; no module-internal ordering is planned, and cross-module order belongs to a future Bundle").

#### Scenario: An engine applying through Flux only refines the table

- **WHEN** an engine applies through Flux's `ssa` package at the version the operator pins (v0.77.0)
- **THEN** it only refines the order of the weight table

### Requirement: The library sorts by weight with a stable sort
The library SHALL provide a stable sort by weight, ascending for apply and descending for delete. Source: enhancement 0012 ("Ordering is kind-class order in the Kubernetes tier; no module-internal ordering is planned, and cross-module order belongs to a future Bundle").

#### Scenario: Equal weights keep their relative order

- **WHEN** objects of equal weight are sorted ascending or descending
- **THEN** the sort is stable, ascending for apply and descending for delete

### Requirement: A weight is looked up by group, version and kind first
A weight SHALL be looked up by group, version and kind first; then a CustomResourceDefinition of `apiextensions.k8s.io`, a Namespace of the core group and a ClusterRole of `rbac.authorization.k8s.io` by group and kind in any version. Source: enhancement 0012 ("Ordering is kind-class order in the Kubernetes tier; no module-internal ordering is planned, and cross-module order belongs to a future Bundle").

#### Scenario: A kind is looked up by group and kind in any version

- **WHEN** the weight of a CustomResourceDefinition of `apiextensions.k8s.io`, a Namespace of the core group or a ClusterRole of `rbac.authorization.k8s.io` is looked up
- **THEN** it is looked up by group and kind in any version

### Requirement: A weight falls back to the kind alone, then to the Class weight, then to 1000
The lookups of requirement "A weight is looked up by group, version and kind first" come first. A weight SHALL be looked up then by kind alone, where a kind named CustomResourceDefinition, Namespace or ClusterRole in another group weighs 6; then any kind whose name ends in `Class` weighs 6; else 1000. Source: enhancement 0012 ("Ordering is kind-class order in the Kubernetes tier; no module-internal ordering is planned, and cross-module order belongs to a future Bundle").

#### Scenario: A custom kind ending in Class falls back to the Class weight

- **WHEN** the weight of the custom kind `EC2NodeClass` is looked up
- **THEN** it is 6, because no entry matches by group, version and kind and its name ends in `Class`

### Requirement: Apply stages keep the stable sort order and are never empty
Order within a stage SHALL be the stable sort order. The input SHALL NOT be reordered, and no stage SHALL be empty. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply" and "Ordering is kind-class order in the Kubernetes tier; no module-internal ordering is planned, and cross-module order belongs to a future Bundle").

#### Scenario: Stages keep input order and are not empty

- **WHEN** two Services are staged in a given input order
- **THEN** they keep that order within their stage, the input is not reordered and no stage is empty

### Requirement: Stages after the first follow ascending weight
The first stage is the one that requirement "Apply stages follow the weight table" holds. After it SHALL come one stage per distinct weight of the remaining objects, in ascending weight. Source: enhancement 0012 ("Ordering is kind-class order in the Kubernetes tier; no module-internal ordering is planned, and cross-module order belongs to a future Bundle").

#### Scenario: Stages after the first follow ascending weight

- **WHEN** the remaining objects of an apply set are staged after the first stage
- **THEN** one stage per distinct weight of the remaining objects follows it, in ascending weight

### Requirement: A Flux frontend may submit a whole set or one stage
The weight table SHALL NOT contradict Flux's staged apply order, so a frontend whose apply engine is Flux's staged apply MAY submit the whole set or any one stage in one call: the engine then only refines the library's order. Source: enhancement 0012 ("Ordering is kind-class order in the Kubernetes tier; no module-internal ordering is planned, and cross-module order belongs to a future Bundle").

#### Scenario: A frontend submits one stage in one call

- **WHEN** a frontend whose apply engine is Flux's staged apply submits the whole set or any one stage in one call
- **THEN** the engine only refines the library's order

### Requirement: Readiness is judged by a pure function over fetched objects
The library SHALL provide `opm/k8s/health`, which judges the readiness of one Kubernetes object that the caller has already fetched, and aggregates such judgements. Nothing in the package SHALL read a cluster, wait or poll; the frontend fetches each live object with its own client and passes it in. Source: enhancement 0012 ("The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module"); ADR-013, decision f5.

#### Scenario: The package reads no cluster

- **WHEN** `opm/k8s/health` judges an object
- **THEN** the caller has already fetched the object with its own client, and the package reads no cluster, waits and polls nothing

### Requirement: Lint holds the health package to the standard library and apimachinery
A strict lint rule SHALL hold the non-test files of `opm/k8s/health` to the Go standard library and `k8s.io/apimachinery`. Source: enhancement 0012 ("The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module"); ADR-013, decision f5.

#### Scenario: The package performs no cluster I/O

- **WHEN** a non-test file of `opm/k8s/health` imports any package other than the Go standard library and `k8s.io/apimachinery`, such as `k8s.io/client-go` or a library `opm/` package
- **THEN** `task lint` fails on the depguard rule `k8s-health-imports-only-apimachinery`, so the package can hold no cluster client and takes every object as an argument

### Requirement: Readiness statuses and evaluation rules are the cli's
A status SHALL be one of the strings `Ready`, `NotReady`, `Complete`, `Unknown`, `Missing`, `Applied` and `Bound`, byte-equal to the cli's `internal/kubernetes/health.go` at cli `bd4d1a7c`, or, for a PersistentVolumeClaim, the raw value of its `status.phase`. The evaluation rules SHALL be the cli's at that commit, unchanged. Source: enhancement 0012 ("The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module"); ADR-013, decision f5.

#### Scenario: The status strings match the cli's output

- **WHEN** a table test pins each status constant to its literal
- **THEN** the seven constants are `Ready`, `NotReady`, `Complete`, `Unknown`, `Missing`, `Applied` and `Bound`

### Requirement: A Deployment is ready only when its rollout is complete
A Deployment SHALL be `Ready` only when `status.observedGeneration` has reached `metadata.generation`, no Progressing condition carries the reason `ProgressDeadlineExceeded`, and the updated and available replica counts equal `spec.replicas` (1 when omitted) with no more total replicas than updated ones; otherwise `NotReady`. Source: enhancement 0012 ("The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module"); ADR-013, decision f5.

#### Scenario: A Deployment mid-rollout is not ready

- **WHEN** a Deployment whose generation is observed and whose Available condition is `True` reports fewer updated replicas than `spec.replicas`
- **THEN** its status is `NotReady`

#### Scenario: A Deployment past its progress deadline is not ready

- **WHEN** a Deployment whose replica counters all equal `spec.replicas` carries a Progressing condition with reason `ProgressDeadlineExceeded`
- **THEN** its status is `NotReady`

### Requirement: A StatefulSet is ready by its update strategy
A StatefulSet SHALL be `Ready` when the generation is observed and the ready replicas equal `spec.replicas`, and then: always under the `OnDelete` strategy; under a RollingUpdate partition greater than zero, when the updated replicas reach `spec.replicas` minus the partition. Source: enhancement 0012 ("The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module").

#### Scenario: A partitioned StatefulSet is ready at its partition

- **WHEN** a StatefulSet with three replicas, its generation observed and a RollingUpdate partition of 2, reports three ready replicas and one updated replica
- **THEN** its status is `Ready`

### Requirement: A StatefulSet is ready when its updated replicas and revisions agree
The cases of requirement "A StatefulSet is ready by its update strategy" come first. A StatefulSet SHALL be `Ready` when the generation is observed and the ready replicas equal `spec.replicas`, and then: otherwise when the updated replicas equal `spec.replicas` and the update revision equals the current revision. Otherwise `NotReady`. Source: enhancement 0012 ("The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module").

#### Scenario: A StatefulSet with matching replicas and revisions is ready

- **WHEN** a StatefulSet's generation is observed, its ready replicas equal `spec.replicas`, its updated replicas equal `spec.replicas` and its update revision equals its current revision
- **THEN** its status is `Ready`

### Requirement: A DaemonSet and a Job are judged by their counts and conditions
A DaemonSet SHALL be `Ready` when the generation is observed and the updated and available counts equal `status.desiredNumberScheduled`; otherwise `NotReady`. A Job SHALL be `Complete` when, in condition order, a `Complete` condition with status `True` appears before any `Failed` condition with status `True`; otherwise `NotReady`. Source: enhancement 0012 ("The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module"); ADR-013, decision f5.

#### Scenario: A DaemonSet with all its pods available is ready

- **WHEN** a DaemonSet whose generation is observed reports updated and available counts equal to `status.desiredNumberScheduled`
- **THEN** its status is `Ready`

#### Scenario: A Job completes before it fails

- **WHEN** a Job carries a `Complete` condition with status `True` before any `Failed` condition with status `True`
- **THEN** its status is `Complete`

### Requirement: A CronJob and the passive kinds are applied
A CronJob, and every kind of the fixed passive set (ConfigMap, Secret, Service, ServiceAccount, Namespace, ClusterRole, ClusterRoleBinding, Role, RoleBinding, Ingress, NetworkPolicy, PodDisruptionBudget, ResourceQuota, LimitRange, StorageClass, PriorityClass), SHALL be `Applied`. Source: enhancement 0012 ("The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module"); ADR-013, decision f5.

#### Scenario: A passive kind is applied

- **WHEN** a ConfigMap or a CronJob is evaluated
- **THEN** its status is `Applied`

### Requirement: A claim reports its phase and any other kind reports its Ready condition
A PersistentVolumeClaim SHALL report its `status.phase` as the status (`Bound` for a bound claim), or `Ready` when it has no phase yet. Any other kind SHALL be `Ready` when its `Ready` condition is `True`, `NotReady` when that condition has any other status, and `Applied` when it has no `Ready` condition. Source: enhancement 0012 ("The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module"); ADR-013, decision f5.

#### Scenario: A pending claim reports its phase

- **WHEN** a PersistentVolumeClaim with `status.phase` `Pending` is evaluated
- **THEN** its status is the string `Pending`, and that status is not healthy

#### Scenario: A custom resource without a Ready condition counts as applied

- **WHEN** an object of an unlisted kind with no `Ready` condition is evaluated
- **THEN** its status is `Applied`, and that status is healthy

### Requirement: Four statuses are healthy and every other string is unhealthy
`Ready`, `Applied`, `Complete` and `Bound` SHALL be the healthy statuses, and every other string SHALL be unhealthy. Source: enhancement 0012 ("The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module"); ADR-013, decision f5.

#### Scenario: Only four statuses are healthy

- **WHEN** the statuses `Ready`, `Applied`, `Complete`, `Bound` and any other string are tested
- **THEN** the four named are healthy and every other string is unhealthy

### Requirement: The aggregate counts missing and unreadable objects against ready
The aggregate of a list of statuses and a count of tracked objects that have no status (missing or unreadable) SHALL count each of those objects toward the total and never toward ready, and SHALL be `Unknown` when the total is zero, `Ready` when every counted object is healthy, and `NotReady` otherwise, together with the ready count and the total. Source: enhancement 0012 ("The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module").

#### Scenario: Missing and unreadable objects count against the aggregate

- **WHEN** three `Ready` statuses are aggregated with two tracked objects that have no status
- **THEN** the aggregate is `NotReady` with 3 ready out of 5

#### Scenario: Nothing to aggregate is unknown

- **WHEN** no statuses and no objects without a status are aggregated
- **THEN** the aggregate is `Unknown` with 0 ready out of 0

### Requirement: ProgressDeadlineExceeded tells a stalled rollout from one in progress
`opm/k8s/health` SHALL also report, through `ProgressDeadlineExceeded`, whether a Deployment whose controller has observed its current generation carries a Progressing condition with the reason `ProgressDeadlineExceeded`, false for any other object, so a frontend can tell a stalled rollout from one in progress without reading conditions itself. Source: enhancement 0012 ("The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module").

#### Scenario: A stalled Deployment is told apart from one rolling out

- **WHEN** `ProgressDeadlineExceeded` is asked about a Deployment whose generation is observed and whose Progressing condition has reason `ProgressDeadlineExceeded`, and about one whose updated replicas still lag with Progressing reason `ReplicaSetUpdated`
- **THEN** it reports true for the first and false for the second, and `Evaluate` reports `NotReady` for both

### Requirement: The ownership verdicts are pure functions of explicit inputs
The library SHALL provide `opm/k8s/ownership` with a per-object delete verdict and a per-object apply verdict, as pure functions of explicit inputs. Each reason SHALL be a string equal to the contract literal for it. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply"), ADR-008, ADR-011.

#### Scenario: Reasons are the contract's literals

- **WHEN** a test reads the skip and refusal reason constants
- **THEN** the skip reasons are `safety-excluded`, `already-absent`, `not-opm-managed`, `owner-mismatch` and `adopted-elsewhere`
- **AND** the refusal reasons are `terminating`, `foreign-object`, `other-instance` and `adopted-elsewhere`

### Requirement: The ownership package performs no I/O and never modifies the live object
The package `opm/k8s/ownership` SHALL perform no cluster read or write, read no clock or environment, and log nothing. It SHALL NOT modify the live object it is given. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply"), ADR-008, ADR-011.

#### Scenario: The live object is not modified

- **WHEN** either verdict is computed for a live object
- **THEN** the object is deeply equal to a copy taken before the call

#### Scenario: The package imports no clock, environment or logger

- **WHEN** the direct imports of `opm/k8s/ownership` are listed
- **THEN** none of them is `os`, `time`, `log` or `log/slog`

### Requirement: Ownership inputs name the object and carry the live object
Each input (Requirement: The ownership verdicts are pure functions of explicit inputs) SHALL name the object by group, kind, namespace and name and SHALL carry the live object as the caller read it, absent when the read found nothing, together with the judging instance's UUID. A frontend reads the live object with its own client. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply"), ADR-008, ADR-011.

#### Scenario: An input names the object and carries the live object

- **WHEN** a verdict is asked for an object
- **THEN** the input names it by group, kind, namespace and name and carries the live object as the frontend read it with its own client, absent when the read found nothing, together with the judging instance's UUID

### Requirement: The delete verdict decides in a fixed order
The delete verdict SHALL decide in the order that scenario "The delete verdict order" lists and stop at the first match: skip as `safety-excluded`, `already-absent`, `not-opm-managed`, `owner-mismatch` or `adopted-elsewhere`, otherwise proceed. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply").

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

#### Scenario: The delete verdict order

- **WHEN** the delete verdict is asked for an object
- **THEN** it skips as `safety-excluded` when the kind is safety-excluded, whatever the live object
- **AND** it skips as `already-absent` when there is no live object
- **AND** it skips as `not-opm-managed` when the live managed-by label is not an OPM runtime's value
- **AND** it skips as `owner-mismatch` when the live UUID label and the instance's UUID are both non-empty and differ
- **AND** it skips as `adopted-elsewhere` when the live adopt annotation, compared with surrounding whitespace trimmed, is non-blank and differs from the instance's UUID
- **AND** otherwise it proceeds

### Requirement: Empty UUIDs and blank annotations pass the delete verdict
An empty UUID on either side SHALL pass the owner comparison. A blank adopt annotation SHALL pass the adoption comparison, and with an empty instance UUID any non-blank adopt annotation SHALL count as another instance's, since an empty UUID can never be the one an annotation names. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply").

#### Scenario: An empty UUID on either side passes

- **WHEN** the live Deployment is OPM-managed, carries no adopt annotation, and either its UUID label or the instance UUID is empty
- **THEN** the verdict proceeds

#### Scenario: An empty instance UUID skips an annotated object

- **WHEN** the live Deployment is OPM-managed and its adopt annotation is `u-1`, for an empty instance UUID
- **THEN** the verdict skips as `adopted-elsewhere`

### Requirement: An annotation naming this instance and a deleting object pass the delete verdict
The owner comparison and the adoption comparison are those of requirement "Empty UUIDs and blank annotations pass the delete verdict". An adopt annotation naming this instance SHALL change nothing. An object being deleted SHALL NOT be skipped for that reason. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply").

#### Scenario: An object being deleted proceeds

- **WHEN** the live OPM-managed object of this instance has a deletion timestamp
- **THEN** the verdict proceeds

#### Scenario: An annotation naming this instance changes nothing on delete

- **WHEN** the live Deployment is OPM-managed, carries UUID label `u-9`, and its adopt annotation is `u-9`, for instance UUID `u-9`
- **THEN** the verdict proceeds

### Requirement: A prune or an instance deletion leaves an object handed to another instance
A prune or an instance deletion SHALL NOT delete an object the user is handing to another instance, before or after that instance has applied it. Source: enhancement 0012 ("The apply guard runs on every apply, refusing foreign objects outside the inventory and any object adopted by another instance; a per-object adopt annotation is its only override") (enhancements#103).

#### Scenario: An object annotated for another instance is left in place

- **WHEN** the live Deployment is OPM-managed, carries this instance's UUID label `u-9`, and its adopt annotation is `u-1`, for instance UUID `u-9`
- **THEN** the verdict skips as `adopted-elsewhere`
- **AND** its message is `Deployment/web/api is being adopted by module instance u-1, not this one; left in place`
- **AND** it yields no DELETE precondition

### Requirement: A proceed verdict carries the judged object's identity
A proceed verdict SHALL carry the UID and the resourceVersion of the live object it judged. Its DELETE precondition SHALL name that UID and SHALL NOT name a resourceVersion. A caller MAY add the carried resourceVersion itself. Source: enhancement 0012 ("The Kubernetes runtime surface homes in the library's Kubernetes tier (supersedes only the placement conclusion of the CLI inventory and handoff enhancement)").

#### Scenario: Proceed carries the precondition for the judged object

- **WHEN** the verdict proceeds for a live object with UID `u-1` and resourceVersion `42`
- **THEN** it carries UID `u-1` and resourceVersion `42`
- **AND** its DELETE precondition names UID `u-1` and no resourceVersion

### Requirement: A skip verdict yields no precondition and carries a message
A skip verdict, and a proceed verdict whose judged UID is empty, SHALL yield no precondition at all, never a precondition on an empty UID. A skip verdict SHALL carry a message naming the object and the reason. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply").

#### Scenario: A skip verdict yields no precondition

- **WHEN** the delete verdict skips as `already-absent`
- **THEN** it yields no DELETE precondition

#### Scenario: A live object without a UID yields no precondition

- **WHEN** the verdict proceeds for a live object that carries no UID
- **THEN** it yields no DELETE precondition

### Requirement: An adopt annotation naming the instance lifts an ownership refusal
An adopt annotation whose value equals the instance UUID SHALL lift the `foreign-object`, `other-instance` and `adopted-elsewhere` refusals. The annotation value SHALL be compared with surrounding whitespace trimmed, and a value that is empty or only whitespace SHALL count as no annotation. Source: enhancement 0012 ("The apply guard runs on every apply, refusing foreign objects outside the inventory and any object adopted by another instance; a per-object adopt annotation is its only override").

#### Scenario: Adoption lifts a foreign refusal

- **WHEN** the object outside the inventory is not OPM-managed and its adopt annotation equals the instance UUID
- **THEN** the verdict applies

#### Scenario: Adoption lifts an other-instance refusal

- **WHEN** the object outside the inventory carries another instance's UUID label and its adopt annotation equals the instance UUID
- **THEN** the verdict applies

#### Scenario: Surrounding whitespace in the annotation value is ignored

- **WHEN** the object outside the inventory is not OPM-managed and its adopt annotation is the instance UUID with surrounding whitespace
- **THEN** the verdict applies

### Requirement: An adopt annotation with any other value lifts nothing
The annotation value (Requirement: An adopt annotation naming the instance lifts an ownership refusal) SHALL be compared with surrounding whitespace trimmed, and a value that is empty or only whitespace SHALL count as no annotation. An adopt annotation with any other value (Requirement: An adopt annotation naming the instance lifts an ownership refusal) SHALL lift nothing, and the refusal message SHALL say that the annotation names another instance.
**Source**: enhancement 0012 ("The apply guard runs on every apply, refusing foreign objects outside the inventory and any object adopted by another instance; a per-object adopt annotation is its only override").

#### Scenario: An annotation naming another instance lifts nothing

- **WHEN** the object outside the inventory is not OPM-managed and its adopt annotation names a different UUID
- **THEN** the verdict refuses as `foreign-object`
- **AND** the message says the annotation names another instance

#### Scenario: A blank annotation counts as none

- **WHEN** the object outside the inventory is not OPM-managed and its adopt annotation is empty or only whitespace
- **THEN** the verdict refuses as `foreign-object`
- **AND** the message does not say the annotation names another instance

### Requirement: A refusal names the adopt annotation key and the UUID to set
The refusal message for `foreign-object` and `other-instance` SHALL name the object, the annotation key `opmodel.dev/adopt` and the instance UUID to set it to, unless the instance UUID is empty. Source: enhancement 0012 ("The apply guard runs on every apply, refusing foreign objects outside the inventory and any object adopted by another instance; a per-object adopt annotation is its only override").

#### Scenario: The refusal tells the user how to adopt

- **WHEN** the verdict refuses `Deployment/web/api` as `foreign-object` for instance UUID `u-9`
- **THEN** the message contains `Deployment/web/api`, `opmodel.dev/adopt` and `u-9`
- **AND** it contains no `--` flag and no `force`

### Requirement: The other-instance refusal asks to remove the object from its owner first
The `other-instance` message SHALL ask the user to remove the object from the instance that owns it before annotating it, so that instance stops rendering it and is not refused on every apply after the hand-over. Source: enhancement 0012 ("The apply guard runs on every apply, refusing foreign objects outside the inventory and any object adopted by another instance; a per-object adopt annotation is its only override").

#### Scenario: The other-instance remedy moves the object only after its owner lets go

- **WHEN** the verdict refuses `Deployment/web/api` as `other-instance` for instance UUID `u-9`, the live UUID label being `u-1`
- **THEN** the message asks to remove it from module instance `u-1` before annotating it `opmodel.dev/adopt=u-9`

### Requirement: No refusal message names another way past the refusal
No message (Requirement: A refusal names the adopt annotation key and the UUID to set) SHALL name any other way past a refusal, such as a command-line flag. No message SHALL carry an enhancement reference. Source: enhancement 0012 ("The apply guard runs on every apply, refusing foreign objects outside the inventory and any object adopted by another instance; a per-object adopt annotation is its only override").

#### Scenario: A refusal message names no other remedy and no enhancement

- **WHEN** an ownership refusal message is produced
- **THEN** it names no other way past the refusal, such as a command-line flag, and carries no enhancement reference

### Requirement: An inventory entry has exactly six fields and a constructor from an object
The library SHALL provide, in `opm/k8s/inventory`, an inventory entry with exactly the fields group, kind, namespace, name, version and component, and a constructor that builds one from a `*unstructured.Unstructured`. Source: enhancement 0012 ("The stale set is component-blind, and the inventory digest hashes a canonical field encoding").

#### Scenario: The entry has six fields and an object constructor

- **WHEN** a developer lists the fields of the inventory entry and its constructors
- **THEN** the fields are group, kind, namespace, name, version and component, and one constructor builds an entry from a `*unstructured.Unstructured`

### Requirement: The entry constructor reads identity from the object and component from its label
The constructor (Requirement: An inventory entry has exactly six fields and a constructor from an object) SHALL read group, version and kind, namespace and name from the object, and the component from its `component.opmodel.dev/name` label, which gives an empty component when the label is absent.
**Source**: enhancement 0012 ("The Kubernetes runtime surface homes in the library's Kubernetes tier (supersedes only the placement conclusion of the CLI inventory and handoff enhancement)" and "The stale set is component-blind, and the inventory digest hashes a canonical field encoding").

#### Scenario: An entry reads the object's identity and component

- **WHEN** an entry is built from a `Deployment` in group `apps`, version `v1`, namespace `team`, named `web` and labelled `component.opmodel.dev/name: web`
- **THEN** the entry's group is `apps`, kind `Deployment`, namespace `team`, name `web`, version `v1` and component `web`

#### Scenario: A core-group, cluster-scoped object without the label

- **WHEN** an entry is built from a `v1` `Namespace` named `team` with no `component.opmodel.dev/name` label
- **THEN** the entry's group, namespace and component are empty, its kind is `Namespace`, its name `team` and its version `v1`

### Requirement: The entry type carries no struct tags
The entry type (Requirement: An inventory entry has exactly six fields and a constructor from an object) SHALL carry no struct tags, so that no serialisation the library chooses can become the frontends' wire shape. Each frontend maps entries to its own CRD or record fields.
**Source**: enhancement 0012 ("The Kubernetes runtime surface homes in the library's Kubernetes tier (supersedes only the placement conclusion of the CLI inventory and handoff enhancement)" and "The stale set is component-blind, and the inventory digest hashes a canonical field encoding").

#### Scenario: The entry type carries no tags

- **WHEN** a test reflects over the entry type's fields
- **THEN** there are exactly six string fields and none carries a struct tag

### Requirement: The stale set compares group, kind, namespace and name only
The library SHALL provide, in `opm/k8s/inventory`, one stale-set function. It SHALL return every entry of the previous inventory for which no entry of the current inventory has the same group, kind, namespace and name, whatever the component and the version of either entry. Source: enhancement 0012 ("The stale set is component-blind, and the inventory digest hashes a canonical field encoding").

#### Scenario: A component rename is not stale

- **WHEN** the previous inventory holds `apps/Deployment team/web` with component `api` and the current inventory holds the same group, kind, namespace and name with component `web`
- **THEN** the stale set is empty

#### Scenario: An API version change is not stale

- **WHEN** the previous inventory holds an `autoscaling` `HorizontalPodAutoscaler` at version `v2beta2` and the current inventory holds the same object at version `v2`
- **THEN** the stale set is empty

### Requirement: The stale set keeps the previous order and is never nil
The result (Requirement: The stale set compares group, kind, namespace and name only) SHALL keep the previous inventory's order, SHALL be a non-nil empty list when nothing is stale, and SHALL leave both inputs unchanged.
**Source**: enhancement 0012 ("The stale set is component-blind, and the inventory digest hashes a canonical field encoding" and "The Kubernetes runtime surface homes in the library's Kubernetes tier (supersedes only the placement conclusion of the CLI inventory and handoff enhancement)").

#### Scenario: Removed objects are stale in previous order

- **WHEN** the previous inventory holds objects A, B, C and D in that order and the current inventory holds only B
- **THEN** the stale set is A, C, D in that order

#### Scenario: Nothing stale is an empty list, not nil

- **WHEN** the previous inventory is empty, or every previous object is in the current inventory
- **THEN** the stale set is a non-nil list of length zero

### Requirement: The package exports one identity comparison and no component-aware one
The identity comparison of the stale set SHALL be exported on its own from `opm/k8s/inventory`. No component-aware comparison and no separate component-rename filter SHALL exist in the package. Source: enhancement 0012 ("The stale set is component-blind, and the inventory digest hashes a canonical field encoding" and "The Kubernetes runtime surface homes in the library's Kubernetes tier (supersedes only the placement conclusion of the CLI inventory and handoff enhancement)").

#### Scenario: The package exports no other comparison

- **WHEN** a test lists the exported identifiers of `opm/k8s/inventory`
- **THEN** they are exactly `Entry`, `NewEntry`, `SameObject`, `StaleSet`, `Digest` and `RenderDigest`

### Requirement: The inventory digest is sha256 over a tagged, length-prefixed encoding
The library SHALL provide, in `opm/k8s/inventory`, one inventory digest of the form `sha256:` followed by 64 lowercase hex digits. It SHALL hash the tag line `opm-inventory-v1` with a newline, followed by every entry in order of group, kind, namespace, name, component and version (byte-wise). Source: enhancement 0012 ("The stale set is component-blind, and the inventory digest hashes a canonical field encoding").

#### Scenario: The encoding is the one defined

- **WHEN** a test writes out by hand the tag line and the length-prefixed fields of the fixture entries (a core-group cluster-scoped entry with empty group and namespace, a namespaced `apps` entry, an entry with an empty component, two `apps` entries whose namespace order and name order disagree, two entries of one kind and namespace whose name order and component order disagree, two entries with one identity whose component order and version order disagree, and two entries that differ only in version), sorted as defined
- **THEN** the digest of those entries, given in another order, equals `sha256:` and the hex SHA-256 of the hand-written bytes, and equals the committed golden value

### Requirement: Each entry of the inventory digest is written with length-prefixed fields
The library SHALL provide, in `opm/k8s/inventory`, one inventory digest of the form `sha256:` followed by 64 lowercase hex digits. Each entry SHALL be written as its group, kind, namespace, name, version and component, in that order, each preceded by its byte length as an 8-byte big-endian unsigned integer. Source: enhancement 0012 ("The stale set is component-blind, and the inventory digest hashes a canonical field encoding").

#### Scenario: Each entry is written with length-prefixed fields

- **WHEN** an entry is written for the inventory digest
- **THEN** it is written as its group, kind, namespace, name, version and component, in that order, each preceded by its byte length as an 8-byte big-endian unsigned integer

### Requirement: The inventory digest depends only on the entries' field values
The digest (Requirement: The inventory digest is sha256 over a tagged, length-prefixed encoding) SHALL depend only on the entries' field values: not on their input order and not on any JSON or other serialisation of an entry. Two inventories that differ in their entries, or in any field of an entry, SHALL produce different digests.
**Source**: enhancement 0012 ("The Kubernetes runtime surface homes in the library's Kubernetes tier (supersedes only the placement conclusion of the CLI inventory and handoff enhancement)").

#### Scenario: Input order does not matter

- **WHEN** the same entries are digested in two different orders
- **THEN** the two digests are equal

#### Scenario: Every field counts

- **WHEN** one entry of an inventory changes in exactly one of its six fields, or an entry is added, removed or repeated, or a byte moves from one field to the next (group `ab` and kind `` against group `a` and kind `b`)
- **THEN** the digest differs from the original inventory's

### Requirement: The empty inventory hashes the tag line and a new encoding uses a new tag line
The inventory digest of the empty inventory SHALL hash the tag line `opm-inventory-v1` alone. The inventory digest function SHALL leave its input unchanged. A change to the encoding of the inventory digest SHALL use a new tag line. Source: enhancement 0012 ("The stale set is component-blind, and the inventory digest hashes a canonical field encoding").

#### Scenario: Empty and nil inventories agree

- **WHEN** an empty inventory and a nil inventory are digested
- **THEN** both digests equal `sha256:` and the hex SHA-256 of `opm-inventory-v1` followed by a newline

### Requirement: The render digest reads only exported JSON
The library SHALL provide, in `opm/k8s/inventory`, one render digest over the objects that `opm/k8s/object`'s export returns, of the form `sha256:` followed by 64 lowercase hex digits. It SHALL read only each object's exported JSON, never a CUE value, so a caller may release its build before digesting. Source: enhancement 0012 (labels are stamped by the CUE render and the shared digest ignores the runtime name).

#### Scenario: The render digest reads no CUE value

- **WHEN** the render digest is computed over the objects that the export returns
- **THEN** it reads only each object's exported JSON and never a CUE value, so a caller may release its build before digesting

### Requirement: The render digest blanks the managed-by value and sorts object keys
For each object the render digest SHALL decode the JSON keeping every number's literal, replace the value of the label `app.kubernetes.io/managed-by` with the empty string when `metadata.labels` holds that key, and encode the result as JSON with object keys sorted and no HTML escaping, ending with a newline. Source: enhancement 0012 (labels are stamped by the CUE render and the shared digest ignores the runtime name).

#### Scenario: Large integers are not rounded

- **WHEN** two exported object sets differ only in one integer field, `9007199254740993` in one and `9007199254740992` in the other
- **THEN** their render digests differ

### Requirement: The render digest encodes strings as Go's JSON encoder does with HTML escaping off
Strings SHALL be encoded as Go's `encoding/json` Encoder writes them with HTML escaping off, with the escapes that scenario "Strings are escaped as defined" lists. Source: enhancement 0012 (labels are stamped by the CUE render and the shared digest ignores the runtime name).

#### Scenario: Strings are escaped as defined

- **WHEN** a string is encoded for the render digest
- **THEN** `"` and `\` are escaped with a backslash
- **AND** U+0008, U+000C, U+000A, U+000D and U+0009 are written as the short escapes `\b`, `\f`, `\n`, `\r` and `\t`
- **AND** every other character from U+0000 to U+001F, and U+2028 and U+2029, is written as a `\u` escape with four lowercase hex digits
- **AND** every other character, including U+007F and `<`, `>` and `&`, is written as is
- **AND** invalid UTF-8 is replaced by U+FFFD

### Requirement: The render digest orders the encoded objects and hashes them after a tag line
The render digest SHALL hash the tag line `opm-render-v1` with a newline, followed by the encoded objects in order of group (from `apiVersion`), kind, namespace and name, with the encoded bytes as the final tie-break. The empty set SHALL hash the tag line alone. A change to this encoding SHALL use a new tag line. Source: enhancement 0012 (labels are stamped by the CUE render and the shared digest ignores the runtime name).

#### Scenario: The encoding is the one defined

- **WHEN** a test writes out by hand the tag line and the sorted-key JSON of a Deployment carrying a managed-by label and an annotation with U+2028, a newline, U+0001 and U+007F, a core-group Service, a cluster-scoped Namespace, a core-group ConfigMap in a namespace that sorts after the Services' namespaces, two Services whose namespace order and name order disagree, and two Services in one namespace whose name order and encoded-bytes order disagree, sorted as defined and with the managed-by value blanked
- **THEN** the render digest of those objects, given in another order, equals `sha256:` and the hex SHA-256 of the hand-written bytes, and equals the committed golden value

### Requirement: Sets that differ only in the managed-by value have the same render digest
Two exported object sets that differ only in the value of the managed-by label SHALL produce the same digest. Source: enhancement 0012 (labels are stamped by the CUE render and the shared digest ignores the runtime name).

#### Scenario: Sets that differ only in the managed-by value digest equally

- **WHEN** two exported object sets differ only in the value of the managed-by label
- **THEN** their render digests are equal

### Requirement: The cli and the operator compute the same render digest for one render
The cli (`opm-cli`) and the operator (`opm-controller`) SHALL compute the same digest for one render as long as the catalogs pass `#runtimeName` only through `#context.labels`, which is the case the kernel test checks on the library's render fixture and on the shipped catalog. Source: enhancement 0012 (labels are stamped by the CUE render and the shared digest ignores the runtime name).

#### Scenario: The two runtimes digest one render equally

- **WHEN** the kernel renders one instance with runtime name `opm-cli` and again with `opm-controller`, for the library's render fixture and for the shipped-catalog parity instance
- **THEN** each rendered object's managed-by label value is that render's runtime name, and the two renders are equal object by object once that one value is blanked
- **AND** two exported object sets that differ only in the managed-by label value have equal render digests

### Requirement: Any other difference changes the render digest
Two object sets that differ in any other label, in whether the managed-by label is present, or in any other content of any object SHALL produce different digests. Source: enhancement 0012 (labels are stamped by the CUE render and the shared digest ignores the runtime name).

#### Scenario: Any other label counts

- **WHEN** two exported object sets differ only in the value of one label other than `app.kubernetes.io/managed-by`
- **THEN** their render digests differ

#### Scenario: Adding or removing the managed-by label counts

- **WHEN** one object set carries the managed-by label on an object and the other does not, and nothing else differs
- **THEN** their render digests differ

### Requirement: The render digest leaves its input unchanged and fails on a malformed object with its position
An object whose JSON does not decode to a single JSON object SHALL fail with an error naming its position. The function (Requirement: The render digest reads only exported JSON) SHALL leave its input unchanged. Source: enhancement 0012 (labels are stamped by the CUE render and the shared digest ignores the runtime name).

#### Scenario: The input is not changed

- **WHEN** a list of exported objects is digested
- **THEN** each object's JSON bytes and decoded object, including its managed-by label value, are unchanged afterwards

#### Scenario: A malformed object fails with its position

- **WHEN** the second object in the list carries JSON that is a list, that is `null`, or that is not valid JSON
- **THEN** the render digest fails with an error naming position 1

### Requirement: The library provides a deletion plan built from inventory entries, a policy and a UUID
The library SHALL provide `opm/k8s/lifecycle` with a deletion plan built from a list of inventory entries, a deletion policy (`Prune`, `ForceOrphan`) and the deleting instance's UUID. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply"), ADR-008 (Deletion plans).

#### Scenario: A deletion plan is built from entries, a policy and a UUID

- **WHEN** a caller uses `opm/k8s/lifecycle`
- **THEN** it can build a deletion plan from a list of inventory entries, a deletion policy (`Prune`, `ForceOrphan`) and the deleting instance's UUID

### Requirement: The deletion plan orders its entries by delete weight
The plan of requirement "The library provides a deletion plan built from inventory entries, a policy and a UUID" SHALL order the entries by the tier's weight table in descending order, keeping the relative order of entries of equal weight, and SHALL NOT change the caller's list. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply"), ADR-008 (Deletion plans).

#### Scenario: Entries are ordered for deletion

- **WHEN** a plan is built from a ConfigMap, a Deployment, a Namespace and a ValidatingWebhookConfiguration, in that order
- **THEN** its steps are the ValidatingWebhookConfiguration, the Deployment, the ConfigMap and the Namespace, in that order
- **AND** the caller's list is unchanged

#### Scenario: Equal weights keep their order

- **WHEN** a plan is built from two Deployments `b` then `a`
- **THEN** its steps name `b` before `a`

### Requirement: Protected kinds are marked safety-excluded in the plan before any live read
Every CustomResourceDefinition and Namespace entry, as the tier's safety exclusion matches them, SHALL be marked safety-excluded in the plan before any live object is read. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply" and "The Kubernetes runtime surface homes in the library's Kubernetes tier (supersedes only the placement conclusion of the CLI inventory and handoff enhancement)"), ADR-008 (Deletion plans).

#### Scenario: Protected kinds are marked up front

- **WHEN** a plan is built from an inventory holding a CustomResourceDefinition of `apiextensions.k8s.io` and a core Namespace
- **THEN** both steps are marked safety-excluded in the plan as built

### Requirement: A deletion plan needs no render and no stored plan
Building the plan SHALL NOT need a render or a stored plan: an uninstall builds it from the persisted inventory, and a prune builds it from the stale set the inventory package computes, through the same constructor. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply"), ADR-008 (Deletion plans).

#### Scenario: A prune hands its stale set to the same plan

- **WHEN** the stale set of a previous and a current inventory is passed to the plan constructor
- **THEN** the plan's steps are exactly the stale entries, in kind-class delete order

### Requirement: The plan constructor is the only way to build a plan with steps
The constructor (Requirement: The library provides a deletion plan built from inventory entries, a policy and a UUID) SHALL be the only way to build a plan with steps, so no frontend can hand the transition a plan in another order, and the plan's steps SHALL be readable as a copy.
**Source**: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply" and "The Kubernetes runtime surface homes in the library's Kubernetes tier (supersedes only the placement conclusion of the CLI inventory and handoff enhancement)"), ADR-008 (Deletion plans).

#### Scenario: Changing the listed steps does not change the plan

- **WHEN** a caller reorders the steps it read from a plan
- **THEN** the plan's own steps keep their deletion order

### Requirement: The deletion transition returns the next state and one action
The deletion transition SHALL take the plan, the caller's state and the outcome of the action it last named, and SHALL return the next state and exactly one action: read one step's live object, delete one step's object, report one step as skipped, or done. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply") (enhancements#103), ADR-008 rules 1 to 3.

#### Scenario: The transition names only reads, deletions, skips and done

- **WHEN** a plan over a mixed inventory is advanced to done through every outcome kind
- **THEN** every action is a read, a delete, a skip or done
- **AND** every read, delete and skip names a step of the plan

#### Scenario: A call returns the next state and one action

- **WHEN** the deletion transition is called with the plan, the caller's state and the outcome of the action it last named
- **THEN** it returns the next state and exactly one action: read one step's live object, delete one step's object, report one step as skipped, or done

### Requirement: The deletion transition performs no action and names only steps of the plan
The deletion transition SHALL perform no action itself. Every read, delete and skip action SHALL name a step of the plan, and the transition SHALL name no other kind of action. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply") (enhancements#103), ADR-008 rules 1 to 3.

#### Scenario: The transition performs no action and names no other kind of action

- **WHEN** the deletion transition is called
- **THEN** it performs no action itself
- **AND** every read, delete and skip action names a step of the plan, and the transition names no other kind of action

### Requirement: A safety-excluded step is skipped without a read
A safety-excluded step SHALL be reported as skipped with the reason `safety-excluded` without a read. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply") (enhancements#103), ADR-008 rules 1 to 3.

#### Scenario: A protected kind is skipped without a read

- **WHEN** the next step is a Namespace
- **THEN** the action is a skip with reason `safety-excluded`
- **AND** no read is named for that step

### Requirement: Every other step is read, judged, and then skipped or deleted
Every step that is not safety-excluded SHALL be read before it is judged, and the library's delete verdict SHALL judge it against the plan's instance UUID. A skip verdict SHALL be reported as skipped with the verdict's reason and message. A proceed verdict SHALL name a delete with Foreground propagation and the verdict's UID precondition. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply").

#### Scenario: An owned object is read, then deleted with Foreground propagation

- **WHEN** a Deployment step is read and the live object is OPM-managed, carries the plan's instance UUID and has UID `u1`
- **THEN** the next action deletes that Deployment with Foreground propagation and a precondition on UID `u1`

#### Scenario: A foreign object is skipped with the verdict's reason

- **WHEN** a step's live object is not OPM-managed
- **THEN** the next action is a skip with reason `not-opm-managed` and the delete verdict's message

#### Scenario: Another instance's object is skipped

- **WHEN** a step's live object carries a different, non-empty instance UUID than the plan's non-empty owner UUID
- **THEN** the next action is a skip with reason `owner-mismatch`

### Requirement: A plan that does not prune names done at once
When the policy does not prune, the first call (Requirement: The deletion transition returns the next state and one action) SHALL name done and no step SHALL be read or deleted. Source: enhancement 0012 ("The Kubernetes runtime surface homes in the library's Kubernetes tier (supersedes only the placement conclusion of the CLI inventory and handoff enhancement)").

#### Scenario: A plan that does not prune names done at once

- **WHEN** the policy's `Prune` is false and the transition is called with the zero state
- **THEN** the action is done
- **AND** no read or delete is named for any step

#### Scenario: A prune built with the zero policy deletes nothing

- **WHEN** a plan is built from a non-empty stale set with a policy whose `Prune` is false
- **THEN** the first action is done
- **AND** the state records no outcome

### Requirement: A prune leaves an object that another instance adopted
A prune of an object the instance dropped because another instance adopted it SHALL leave the object in place. Source: enhancement 0012 ("The apply guard runs on every apply, refusing foreign objects outside the inventory and any object adopted by another instance; a per-object adopt annotation is its only override") (enhancements#103).

#### Scenario: A prune of a dropped object leaves it for the adopting instance

- **WHEN** a deletion plan built from a stale set holds a Deployment whose live object carries the plan owner's UUID label and an adopt annotation naming another instance
- **THEN** advancing the plan past its read records the step as skipped with `adopted-elsewhere` and names no delete action for it
- **AND** the hold verdict releases once the plan is done

### Requirement: The library classifies the raw error of each read or delete
The caller SHALL hand back the raw error of each read or delete, and the library SHALL classify it with `k8s.io/apimachinery`'s API error helpers, which see through wrapping. Source: enhancement 0012 ("The Kubernetes runtime surface homes in the library's Kubernetes tier (supersedes only the placement conclusion of the CLI inventory and handoff enhancement)" and "The library owns the whole deletion sequence but only the per-object verdict and the order for apply").

#### Scenario: A wrapped Forbidden error is classified

- **WHEN** a delete returns a Forbidden API error wrapped with `fmt.Errorf("deleting: %w", err)`
- **THEN** the step's outcome is failed with failure class `forbidden`

### Requirement: A NotFound read or delete is already absent
A NotFound read, or a read that returns no object, SHALL count as already absent, and SHALL be skipped as `already-absent`. A NotFound delete SHALL be skipped as `already-absent`. Source: enhancement 0012 ("The Kubernetes runtime surface homes in the library's Kubernetes tier (supersedes only the placement conclusion of the CLI inventory and handoff enhancement)" and "The library owns the whole deletion sequence but only the per-object verdict and the order for apply").

#### Scenario: A gone object is already absent

- **WHEN** a read returns a NotFound error
- **THEN** the next action is a skip with reason `already-absent`

#### Scenario: An object gone before its delete is already absent

- **WHEN** a delete returns a NotFound error
- **THEN** the next action is a skip with reason `already-absent`
- **AND** the step is not recorded as deleted

### Requirement: A successful delete is recorded as deleted and any other error as failed
A successful delete SHALL record the step as deleted. Any error that is not a NotFound SHALL record the step as failed, with the failure class `forbidden`, `conflict` or `error` and the error's text. Source: enhancement 0012 ("The Kubernetes runtime surface homes in the library's Kubernetes tier (supersedes only the placement conclusion of the CLI inventory and handoff enhancement)" and "The library owns the whole deletion sequence but only the per-object verdict and the order for apply").

#### Scenario: A failed precondition is a conflict

- **WHEN** a delete returns a Conflict API error
- **THEN** the step's outcome is failed with failure class `conflict`
- **AND** it is not recorded as deleted

### Requirement: A mismatched live object is never judged or deleted
A live object handed back for a read that names a different object than the step, in any identity field the live object sets, SHALL NOT be judged or deleted: the step SHALL be recorded as failed with class `error` and a message naming both objects. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply").

#### Scenario: A live object for another step is not deleted

- **WHEN** the state awaits the read of `Deployment/app/web` and the live object handed back is `Deployment/app/api`
- **THEN** no delete is named for either object
- **AND** the step is recorded as failed with failure class `error`
- **AND** the next action names the next step

#### Scenario: An empty live namespace or kind is not a mismatch

- **WHEN** the step is a ClusterRole recorded with a namespace, and the live object handed back has no namespace and no `apiVersion` or `kind`
- **THEN** the live object is judged for the step

### Requirement: A failure does not stop the plan
A failure SHALL NOT stop the plan: the next step's action SHALL follow. Source: enhancement 0012 ("The Kubernetes runtime surface homes in the library's Kubernetes tier (supersedes only the placement conclusion of the CLI inventory and handoff enhancement)" and "The library owns the whole deletion sequence but only the per-object verdict and the order for apply").

#### Scenario: A failure does not stop the plan

- **WHEN** the read of the first of two steps fails with an Internal error
- **THEN** the next action reads the second step

### Requirement: Each call appends the outcomes of the steps it finished
Each call (Requirement: The deletion transition returns the next state and one action) SHALL append, in step order, the outcomes of the steps it finished and only those, and SHALL NOT change an earlier outcome, so a frontend reports the outcomes appended since the state it passed in. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply").

#### Scenario: One call can finish two steps

- **WHEN** the read of a Deployment step fails with an Internal error and the next step is a Namespace
- **THEN** that call appends a failed outcome for the Deployment and then a skipped outcome for the Namespace, in that order
- **AND** the action is a skip naming the Namespace step

### Requirement: The deletion state is a plain value the caller owns
The deletion state SHALL be a plain value with a defined JSON encoding, holding the next step, what the state awaits and one outcome per finished step. The zero state SHALL be the start of a plan. The library SHALL keep nothing between calls. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply"), ADR-008 rule 2.

#### Scenario: The zero state starts a plan

- **WHEN** the transition is called with the zero state
- **THEN** it starts the plan at its first step, and the library keeps nothing between calls

### Requirement: A state written to JSON advances identically
A state written to JSON and read back SHALL advance identically to the state held in memory, and advancing the same plan, state and outcome twice SHALL name the same action and return equal states. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply"), ADR-008 rule 2.

#### Scenario: A round-tripped state advances identically

- **WHEN** a plan is advanced to done once with an in-memory state and once with a state written to JSON and read back before every call, through the same outcomes
- **THEN** both runs name the same actions in the same order
- **AND** their final states are equal

#### Scenario: Advancing twice names the same action

- **WHEN** the same plan, state and outcome are advanced twice
- **THEN** both calls return equal states and equal actions

### Requirement: A state that cannot belong to the plan is refused
The transition (Requirement: The deletion transition returns the next state and one action) SHALL return an error, the input state unchanged and no action when the state cannot belong to the plan: its next step lies outside the plan, it records a number of outcomes other than its next step, an outcome is out of step order or carries a result other than deleted, skipped or failed.
**Source**: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply"), ADR-008 rule 2.

#### Scenario: A state past the plan is refused

- **WHEN** the state's next step is beyond the plan's last step
- **THEN** the transition returns an error

#### Scenario: A state whose outcomes do not match its next step is refused

- **WHEN** the state names step 1 of a plan and records no outcome
- **THEN** the transition returns an error and the input state unchanged

#### Scenario: A state with an out-of-order or unknown outcome is refused

- **WHEN** the state names step 2 of a plan and its second outcome records step 9, or records a result other than deleted, skipped or failed
- **THEN** the transition returns an error and the input state unchanged

### Requirement: A state that awaits something it cannot await is refused
The transition (Requirement: The deletion transition returns the next state and one action) SHALL return an error, the input state unchanged and no action when the state cannot belong to the plan: it awaits something after the last step, or what it awaits is not a defined value. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply"), ADR-008 rule 2.

#### Scenario: A state awaiting something after the last step is refused

- **WHEN** the state's next step equals the number of steps and the state awaits a read
- **THEN** the transition returns an error and the input state unchanged

### Requirement: The JSON encoding of the deletion state is fixed field by field
The JSON encoding (Requirement: The deletion state is a plain value the caller owns) SHALL be fixed field by field, so that renaming a field fails a check. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply"), ADR-008 rule 2.

#### Scenario: The JSON encoding is fixed

- **WHEN** a state with a next step, an awaited read and a failed outcome carrying every field is written to JSON
- **THEN** the bytes equal `{"next":2,"awaiting":"read","outcomes":[{"step":0,"result":"failed","failure":"forbidden","message":"denied"},{"step":1,"result":"skipped","skip":"already-absent","message":"gone"}]}`

### Requirement: The hold verdict is decided from the policy, the plan's outcome and the identity
The library SHALL decide whether an instance's deletion hold may be released from the plan's policy, its steps, the state's outcomes and whether the caller can act as the deleting identity, and from nothing that names the hold's bearer or the frontend. It SHALL check in the order that scenario "The hold verdict order" lists and stop at the first match. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply").

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

#### Scenario: The hold verdict order

- **WHEN** the hold verdict is decided
- **THEN** when the policy does not prune, it releases as `prune-disabled`
- **AND** when the plan has no steps, it releases as `inventory-empty`
- **AND** when the deleting identity is missing and the policy sets force-orphan, it releases as `force-orphan`
- **AND** when the identity is missing or could not be obtained, it holds as `identity-unavailable`
- **AND** when the state cannot belong to the plan, for any reason the transition refuses it, it holds as `cleanup-incomplete` and never releases
- **AND** when the plan is not finished, it holds as `cleanup-incomplete`
- **AND** when any step failed as Forbidden, it holds as `cleanup-forbidden`
- **AND** when any other step failed, it holds as `cleanup-incomplete`
- **AND** otherwise it releases as `cleanup-complete`, with skipped steps counting as complete

### Requirement: Each hold verdict carries its reason and a message
Each verdict (Requirement: The hold verdict is decided from the policy, the plan's outcome and the identity) SHALL carry the reason as the contract's literal and a message the library words.
**Source**: enhancement 0012 ("The Kubernetes runtime surface homes in the library's Kubernetes tier (supersedes only the placement conclusion of the CLI inventory and handoff enhancement)" and "The library owns the whole deletion sequence but only the per-object verdict and the order for apply"), the 0012 contract's `#HoldVerdict`.

#### Scenario: Reasons are the contract's literals

- **WHEN** a test reads the hold reason constants
- **THEN** they are `prune-disabled`, `inventory-empty`, `cleanup-complete`, `force-orphan`, `cleanup-incomplete`, `cleanup-forbidden` and `identity-unavailable`

### Requirement: The test universe of the Flux model covers the table, Flux's kinds and both group orders
The test's universe (Requirement: The weight table never contradicts Flux's staged apply order) SHALL hold the library's table, Flux's listed kinds, a copy of each of those kinds in an API group that sorts before the built-in groups and in one that sorts after them, custom kinds in both such groups and a custom kind whose name ends in `Class`.
**Source**: enhancement 0012 ("Ordering is kind-class order in the Kubernetes tier; no module-internal ordering is planned, and cross-module order belongs to a future Bundle").

#### Scenario: The universe holds copies in groups before and after the built-in groups

- **WHEN** the universe of the Flux model is listed
- **THEN** it holds a copy of each of Flux's listed kinds in an API group that sorts before the built-in groups and in one that sorts after them, custom kinds in both such groups and a custom kind whose name ends in `Class`

### Requirement: The Flux model test fails on any contradicting pair and names it
The test (Requirement: The weight table never contradicts Flux's staged apply order) SHALL fail when the library weighs any pair of the universe strictly opposite to the order Flux applies them in, naming the pair. Source: enhancement 0012 ("Ordering is kind-class order in the Kubernetes tier; no module-internal ordering is planned, and cross-module order belongs to a future Bundle").

#### Scenario: A contradicting pair fails the test and is named

- **WHEN** the library weighs a pair of the universe strictly opposite to the order Flux applies them in
- **THEN** the test fails and names the pair

### Requirement: A second test shows that the comparison reports an injected contradiction
A second test (Requirement: The weight table never contradicts Flux's staged apply order) SHALL show that the comparison reports a contradiction injected through a modified weight function, and the test SHALL keep asserting that both orders put a CustomResourceDefinition before a Namespace. Source: enhancement 0012 ("Ordering is kind-class order in the Kubernetes tier; no module-internal ordering is planned, and cross-module order belongs to a future Bundle").

#### Scenario: Both orders put a CustomResourceDefinition before a Namespace

- **WHEN** the library's order and Flux's order are compared for a CustomResourceDefinition and a Namespace
- **THEN** both orders put the CustomResourceDefinition before the Namespace

### Requirement: The apply verdict decides in a fixed order
The apply verdict SHALL decide in the order that scenario "The apply verdict order" lists and stop at the first match: apply, or refuse as `terminating`, `foreign-object`, `other-instance` or `adopted-elsewhere`. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply").

#### Scenario: A new object is applied

- **WHEN** the apply verdict is asked for an object with no live object
- **THEN** it applies

#### Scenario: An inventoried foreign object is applied

- **WHEN** the object is in the instance's recorded inventory, its live managed-by label is `helm`, and it carries no UUID label and no adopt annotation
- **THEN** the verdict applies

#### Scenario: An inventoried object of this instance is applied

- **WHEN** the object is in the instance's recorded inventory and its live UUID label equals the instance UUID, with no adopt annotation
- **THEN** the verdict applies

#### Scenario: An inventoried object the adopter has taken is refused

- **WHEN** the object is in the instance's recorded inventory, its live UUID label is `u-1`, and its adopt annotation is `u-1`, for instance UUID `u-9`
- **THEN** the verdict refuses as `adopted-elsewhere`

#### Scenario: An empty instance UUID applies an inventoried object

- **WHEN** the instance UUID is empty and the object in the inventory carries a UUID label and an adopt annotation
- **THEN** the verdict applies

#### Scenario: A foreign object outside the inventory is refused

- **WHEN** the object is outside the inventory and its live managed-by label is missing or is `kustomize`
- **THEN** the verdict refuses as `foreign-object`

#### Scenario: Another instance's object outside the inventory is refused

- **WHEN** the object is outside the inventory, OPM-managed, and its UUID label differs from the instance UUID
- **THEN** the verdict refuses as `other-instance`

#### Scenario: An annotation naming another instance than the label keeps other-instance

- **WHEN** the object is outside the inventory, OPM-managed, its live UUID label is `u-1`, and its adopt annotation is `u-7`, for instance UUID `u-9`
- **THEN** the verdict refuses as `other-instance`

#### Scenario: An OPM object without a UUID label is applied

- **WHEN** the object is outside the inventory, OPM-managed, and carries no UUID label and no adopt annotation
- **THEN** the verdict applies

#### Scenario: An OPM object of this instance outside the inventory is applied

- **WHEN** the object is outside the inventory, OPM-managed, its UUID label equals the instance UUID, and it carries no adopt annotation
- **THEN** the verdict applies

#### Scenario: The apply verdict order

- **WHEN** the apply verdict is asked for an object
- **THEN** it applies when there is no live object
- **AND** it refuses as `terminating` when the live object has a deletion timestamp, whether or not the object is in the instance's recorded inventory
- **AND** it applies when the live object's adopt annotation equals the non-empty instance UUID
- **AND** for an object in the instance's recorded inventory, it applies with an empty instance UUID, refuses as `adopted-elsewhere` when its adopt annotation is non-blank, and otherwise applies, whatever its live UUID label
- **AND** for an object outside the inventory, it refuses as `foreign-object` when the live managed-by label is not an OPM runtime's value
- **AND** it refuses as `adopted-elsewhere` when the adopt annotation is non-blank and equals the live UUID label, since the instance it names completed the hand-over
- **AND** it refuses as `other-instance` when the live UUID label is non-empty and differs from the instance UUID
- **AND** it refuses as `adopted-elsewhere` when the adopt annotation is non-blank
- **AND** otherwise it applies

### Requirement: Nothing lifts a terminating refusal
When the live object has a deletion timestamp, the apply verdict SHALL refuse as `terminating`, whether or not the object is in the instance's recorded inventory, and nothing SHALL lift that refusal. Source: enhancement 0012 ("The Kubernetes runtime surface homes in the library's Kubernetes tier (supersedes only the placement conclusion of the CLI inventory and handoff enhancement)").

#### Scenario: A terminating object is refused everywhere

- **WHEN** the live object has a deletion timestamp
- **THEN** the verdict refuses as `terminating` when the object is in the inventory, when it is outside it, when it carries the adopt annotation naming this instance, and when it carries the adopt annotation naming another instance

### Requirement: An instance whose UUID changed keeps applying its own inventoried objects
An object in the instance's recorded inventory with no deletion timestamp and no non-blank adopt annotation SHALL apply, whatever its live UUID label, so an instance whose UUID changed keeps applying its own objects, except an object whose adopt annotation still names the old UUID, which applies again once re-annotated with the new UUID. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply").

#### Scenario: An inventoried object labelled for another instance without an annotation is applied

- **WHEN** the object is in the instance's recorded inventory and its live UUID label is `u-1`, with no adopt annotation, for instance UUID `u-9`
- **THEN** the verdict applies

#### Scenario: An annotation naming this instance takes an inventoried object back

- **WHEN** the object is in the instance's recorded inventory, its live UUID label is `u-1`, and its adopt annotation is `u-9` with surrounding whitespace, for instance UUID `u-9`
- **THEN** the verdict applies

### Requirement: Outside the inventory an empty instance UUID fails closed
Outside the inventory an empty instance UUID SHALL never match an adopt annotation, and any non-empty live UUID or non-blank adopt annotation SHALL then count as another instance's. Source: enhancement 0012 ("The apply guard runs on every apply, refusing foreign objects outside the inventory and any object adopted by another instance; a per-object adopt annotation is its only override") (enhancements#103).

#### Scenario: An empty instance UUID fails closed outside the inventory

- **WHEN** the instance UUID is empty and the object outside the inventory is OPM-managed with a non-empty UUID label
- **THEN** the verdict refuses as `other-instance`

#### Scenario: An empty instance UUID refuses an annotated object outside the inventory

- **WHEN** the instance UUID is empty and the object outside the inventory is OPM-managed, carries no UUID label, and its adopt annotation is `u-1`
- **THEN** the verdict refuses as `adopted-elsewhere`
- **AND** the message is `Deployment/web/api is being adopted by module instance u-1; this instance does not apply it`, with no remedy clause

### Requirement: The docs state the frontend's part of the hand-over
The `ApplyInput.InInventory` doc and the package doc SHALL state that a frontend drops an object refused as `adopted-elsewhere` while it is in the instance's recorded inventory from the inventory it records next, keeps applying the instance's other objects, and never deletes the object for that refusal. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply") (enhancements#103).

#### Scenario: The docs state the frontend's part of the hand-over

- **WHEN** the `ApplyInput.InInventory` doc comment and the package doc are read
- **THEN** each says a frontend drops an object refused as `adopted-elsewhere` from the inventory it records next and never deletes it for that refusal

### Requirement: The adopted-elsewhere message names the object, the adopter and the way back
The `adopted-elsewhere` message SHALL name the object and the instance its adopt annotation names, and SHALL name the annotation key `opmodel.dev/adopt` with the instance UUID to set it to, unless the instance UUID is empty, as the only way for this instance to take the object back. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply") (enhancements#103).

#### Scenario: An inventoried object annotated for another instance is refused

- **WHEN** the object is in the instance's recorded inventory, carries this instance's UUID label `u-9`, and its adopt annotation is `u-1`, for instance UUID `u-9`
- **THEN** the verdict refuses as `adopted-elsewhere`
- **AND** the message is `Deployment/web/api was adopted by module instance u-1; this instance no longer applies it and drops it from its inventory; to take it back, annotate it opmodel.dev/adopt=u-9`

#### Scenario: A handed-over object stays adopted-elsewhere after the adopter applies

- **WHEN** the object is outside the inventory, OPM-managed, its live UUID label is `u-1`, and its adopt annotation is `u-1`, for instance UUID `u-9`
- **THEN** the verdict refuses as `adopted-elsewhere`, not `other-instance`
- **AND** the message is `Deployment/web/api was adopted by module instance u-1; this instance does not apply it; to let this instance take it back, annotate it opmodel.dev/adopt=u-9`

#### Scenario: A dropped object is not taken back on the next apply

- **WHEN** the object is outside the inventory, OPM-managed, carries this instance's UUID label, and its adopt annotation names a different instance
- **THEN** the verdict refuses as `adopted-elsewhere`
- **AND** the message does not say the instance drops it from its inventory

### Requirement: The adopted-elsewhere message for an inventoried object says it is dropped
For an inventoried object the `adopted-elsewhere` message SHALL say that this instance no longer applies the object and drops it from its inventory. Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object verdict and the order for apply") (enhancements#103).

#### Scenario: The message for an inventoried object says it is dropped

- **WHEN** the `adopted-elsewhere` message is built for an inventoried object
- **THEN** it says that this instance no longer applies the object and drops it from its inventory

### Requirement: The caller supplies the inventory fact and the instance UUID
The caller (Requirement: Ownership inputs name the object and carry the live object) still supplies two facts the library does not check, as the frontend's own ownership record: whether the object is in the instance's recorded inventory, and the instance UUID. A frontend that installs the operator SHALL be judged as any other instance.
**Source**: enhancement 0012 ("The apply guard runs on every apply, refusing foreign objects outside the inventory and any object adopted by another instance; a per-object adopt annotation is its only override") are withdrawn.

#### Scenario: The caller supplies two facts and the operator installer is judged as any instance

- **WHEN** a frontend that installs the operator asks for a verdict
- **THEN** it supplies whether the object is in the instance's recorded inventory and the instance UUID, and is judged as any other instance

### Requirement: Only an adopt annotation outside the inventory lifts an ownership refusal
Outside the inventory, the adopt annotation on the live object naming the applying instance SHALL be the only thing that lifts an ownership refusal. On delete, nothing SHALL lift the `not-opm-managed` skip. Source: enhancement 0012 ("The apply guard runs on every apply, refusing foreign objects outside the inventory and any object adopted by another instance; a per-object adopt annotation is its only override") are withdrawn.

#### Scenario: Nothing else lifts an ownership refusal

- **WHEN** an object outside the inventory is refused on apply, or skipped as `not-opm-managed` on delete
- **THEN** only an adopt annotation naming the applying instance lifts the apply refusal, and nothing lifts the `not-opm-managed` skip
