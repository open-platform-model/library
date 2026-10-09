## MODIFIED Requirements

### Requirement: The ownership verdicts take no input that only lifts a refusal

`ownership.ApplyInput` SHALL have exactly the fields `Object`, `Live`, `InInventory` and `InstanceUUID`, and `ownership.DeleteInput` SHALL have exactly the fields `Object`, `Live` and `InstanceUUID`. Neither SHALL carry a field whose only effect is to lift a refusal or a skip. The caller still supplies two facts the library does not check, as the frontend's own ownership record: whether the object is in the instance's recorded inventory, and the instance UUID. Outside the inventory, the adopt annotation on the live object naming the applying instance SHALL be the only thing that lifts an ownership refusal. On delete, nothing SHALL lift the `not-opm-managed` skip. A frontend that installs the controller is judged as any other instance. Source: 0012:D8:R1/R2/R3; 0012:D8:R6 and 0012:D8:R7 are withdrawn.

#### Scenario: The verdict inputs carry no admission field

- **WHEN** a test lists the fields of `ownership.ApplyInput` and `ownership.DeleteInput`
- **THEN** they are `Object`, `Live`, `InInventory`, `InstanceUUID` and `Object`, `Live`, `InstanceUUID`

#### Scenario: An earlier install manifest's object is refused on apply

- **WHEN** the apply verdict is asked for `Namespace/opm-system`, live with managed-by `kustomize` and no UUID label, outside the inventory
- **THEN** it refuses as `foreign-object`
- **AND** the message names `opmodel.dev/adopt` with the instance UUID

#### Scenario: An earlier install manifest's Deployment is not deleted

- **WHEN** the delete verdict is asked for a live `apps` Deployment with managed-by `kustomize` and no UUID label
- **THEN** it skips as `not-opm-managed`

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
- **THEN** it states that the controller keeps `fluxcd/pkg/ssa` and the CLI keeps its own server-side apply, that the library owns the per-object apply verdict and the order and never an apply engine, and that an engine's staging may refine the library's order and never contradict it

### Requirement: Kind-class order is a Kubernetes fact that lives in the tier

The library SHALL keep one kind-class apply order (a CustomResourceDefinition before its custom resources, a Namespace before namespaced objects). It SHALL live in the Kubernetes tier, and the kernel SHALL hold none. No module-internal ordering (`dependsOn` edges, phases within one module) is planned: the library SHALL add no module-specific ordering of its own, and any module-internal ordering added later SHALL come off the CUE build as data. Lifecycle hooks remain the parked 0009 work (open under 0009:OQ7), carried as build data under ADR-008 rule 4. In the OPM model, ordering across modules belongs to a future Bundle definition, which uses the order its modules are defined in; the library SHALL add none before then. A frontend's readiness gating between its own custom resources, such as the controller's ModulePackage `spec.dependsOn`, is outside this rule; whether a future Bundle definition subsumes it is not decided. ADR-008 rule 4's "The kernel derives no ordering of its own" SHALL be read as excluding module-specific ordering only, and ADR-008 and ADR-011 SHALL say so in their own text. Source: 0012:D5.

#### Scenario: ADR-008 carries the clarification

- **WHEN** a developer reads `adr/008-kernel-plans-caller-runs.md`
- **THEN** its Status records the 2026-10-02 amendment pointing at ADR-011, and the rule ending "The kernel derives no ordering of its own" states that it excludes module-specific ordering only

#### Scenario: No module-internal ordering is planned and cross-module order belongs to a Bundle

- **WHEN** a developer reads rule 4 of ADR-008 and item 7 of ADR-011
- **THEN** both state that no module-internal ordering (`dependsOn` edges, phases) is planned and that any added later comes off the build, that lifecycle hooks remain the parked 0009 work, and that in the OPM model ordering across modules belongs to a future Bundle definition using the order its modules are defined in
- **AND** neither presents module-internal `dependsOn` edges or phases as planned work

### Requirement: Labels are stamped in CUE and the shared digest ignores the runtime name

Label stamping SHALL stay in CUE at render, where core's `#runtimeName` fills the managed-by label. The tier's shared render digest SHALL leave the managed-by label's value out of its input, so that the CLI and the controller, rendering the same instance, compute the same digest bytes. Source: 0012:D6, 0012:D1:R1.

#### Scenario: The digest rule is recorded

- **WHEN** a developer reads ADR-011
- **THEN** it states that labels stay stamped in CUE at render and that the shared digest excludes the managed-by label value

### Requirement: The label vocabulary names labels and never stamps them

The library SHALL provide `opm/k8s/labels`, which imports nothing outside the Go standard library. It SHALL declare the label keys and values both Kubernetes frontends read or write, with values byte-equal to the frontends' copies: the managed-by key `app.kubernetes.io/managed-by`, its runtime values `opm-cli` and `opm-controller`, its legacy value `open-platform-model`, `opmodel.dev/component`, `component.opmodel.dev/name`, `module-instance.opmodel.dev/name`, `module-instance.opmodel.dev/namespace` and `module-instance.opmodel.dev/uuid`. The constant for `opmodel.dev/component` SHALL be documented as the OPM object category, not a component name. It SHALL provide a predicate that reports whether a managed-by value identifies an OPM runtime, which accepts exactly the two runtime values and the legacy value. No function in the package SHALL set a label on an object. Labels on rendered objects are stamped by the CUE build at render. The repository lint gate SHALL refuse any non-test import into the package from outside the standard library. Source: 0012:D6, 0012:D3.

#### Scenario: Every OPM managed-by value is recognised

- **WHEN** the predicate is called with `opm-cli`, `opm-controller` or `open-platform-model`
- **THEN** it reports true for each
- **AND** it reports false for `helm`, for the empty string and for `OPM-CLI`

#### Scenario: The vocabulary matches the frontends' copies

- **WHEN** a table test pins each declared key and value to its literal
- **THEN** each equals the literal, and the literals are the ones a scripted comparison found byte-equal to the cli's `pkg/core/labels.go` at cli `1338e700` and the controller's at opm-controller `9b83611`

#### Scenario: Lint keeps the label package dependency-free

- **WHEN** a change adds an import of `k8s.io/apimachinery`, `cuelang.org/go/cue` or a library package to a non-test file under `opm/k8s/labels/`
- **THEN** `task lint` fails naming the import

### Requirement: One weight table orders apply and delete

The library SHALL keep one kind-class weight table in `opm/k8s/object`, and it SHALL agree with the staged apply order of Flux's `ssa` package at the version the controller pins (v0.77.0), so that an engine applying through Flux only refines it. A CustomResourceDefinition weighs -100. A Namespace weighs 0. ClusterRoles weigh 5. PriorityClasses, RuntimeClasses, StorageClasses, VolumeSnapshotClasses, IngressClasses, GatewayClasses, ClusterClasses and every other kind whose name ends in `Class` weigh 6. ClusterRoleBindings weigh 7. ResourceQuotas weigh 8. ServiceAccounts, Roles and RoleBindings weigh 10. Secrets and ConfigMaps weigh 15. Services weigh 50. LimitRanges weigh 60. Deployments and StatefulSets weigh 100. CronJobs weigh 105. PodDisruptionBudgets weigh 108. Validating and mutating webhook configurations weigh 2000. Every other kind, among them PersistentVolumes, PersistentVolumeClaims, DaemonSets, ReplicaSets, Jobs, Ingresses, NetworkPolicies, the pod autoscalers and every custom resource, weighs 1000. A weight SHALL be looked up by group, version and kind first; then a CustomResourceDefinition of `apiextensions.k8s.io`, a Namespace of the core group and a ClusterRole of `rbac.authorization.k8s.io` by group and kind in any version; then by kind alone, where a kind named CustomResourceDefinition, Namespace or ClusterRole in another group weighs 6; then any kind whose name ends in `Class` weighs 6; else 1000. The library SHALL provide a stable sort by weight, ascending for apply and descending for delete. Source: 0012:D5:R1.

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

### Requirement: The render digest ignores only the managed-by label value

The library SHALL provide, in `opm/k8s/inventory`, one render digest over the objects that `opm/k8s/object`'s export returns, of the form `sha256:` followed by 64 lowercase hex digits. It SHALL read only each object's exported JSON, never a CUE value, so a caller may release its build before digesting. For each object it SHALL decode the JSON keeping every number's literal, replace the value of the label `app.kubernetes.io/managed-by` with the empty string when `metadata.labels` holds that key, and encode the result as JSON with object keys sorted and no HTML escaping, ending with a newline. Strings SHALL be encoded as Go's `encoding/json` Encoder writes them with HTML escaping off: `"` and `\` escaped with a backslash; U+0008, U+000C, U+000A, U+000D and U+0009 as the short escapes `\b`, `\f`, `\n`, `\r` and `\t`; every other character from U+0000 to U+001F, and U+2028 and U+2029, as a `\u` escape with four lowercase hex digits; every other character, including U+007F and `<`, `>` and `&`, written as is; and invalid UTF-8 replaced by U+FFFD. It SHALL hash the tag line `opm-render-v1` with a newline, followed by the encoded objects in order of group (from `apiVersion`), kind, namespace and name, with the encoded bytes as the final tie-break. Two exported object sets that differ only in the value of the managed-by label SHALL produce the same digest. The cli (`opm-cli`) and the controller (`opm-controller`) therefore compute the same digest for one render as long as the catalogs pass `#runtimeName` only through `#context.labels`, which is the case the kernel test checks on the library's render fixture and on the shipped catalog. Two object sets that differ in any other label, in whether the managed-by label is present, or in any other content of any object SHALL produce different digests. The empty set SHALL hash the tag line alone. An object whose JSON does not decode to a single JSON object SHALL fail with an error naming its position. The function SHALL leave its input unchanged. A change to this encoding SHALL use a new tag line. Source: 0012:D6:R2/R3, 0012:D1:R1.

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

### Requirement: The weight table never contradicts Flux's staged apply order

The library SHALL keep a test that models the staged apply of Flux's `ssa` package at the version the controller pins, as literals without importing Flux: its stages (the CustomResourceDefinition of `apiextensions.k8s.io`, the core Namespace and the ClusterRole of `rbac.authorization.k8s.io`; then every kind whose name ends in `Class`; then the rest), its `ReconcileOrder` ranks (every kind it does not list at rank 0), and its tie-break by API group, then kind. The test's universe SHALL hold the library's table, Flux's listed kinds, a copy of each of those kinds in an API group that sorts before the built-in groups and in one that sorts after them, custom kinds in both such groups and a custom kind whose name ends in `Class`. The test SHALL fail when the library weighs any pair of the universe strictly opposite to the order Flux applies them in, naming the pair. A second test SHALL show that the comparison reports a contradiction injected through a modified weight function, and the test SHALL keep asserting that both orders put a CustomResourceDefinition before a Namespace. Source: 0012:D5:R1.

#### Scenario: The shipped table contradicts Flux nowhere

- **WHEN** the comparison runs over the universe with the library's `Weight`
- **THEN** it reports no pair

#### Scenario: A weight edit that creates a contradiction is caught

- **WHEN** the comparison runs with a weight function that moves Deployment below Service
- **THEN** it reports the Service and Deployment pair

#### Scenario: Flux's tie-break between unlisted kinds is counted

- **WHEN** the comparison runs with a weight function that weighs an `autoscaling` HorizontalPodAutoscaler above a `batch` Job
- **THEN** it reports the HorizontalPodAutoscaler and Job pair, because Flux orders them by group
