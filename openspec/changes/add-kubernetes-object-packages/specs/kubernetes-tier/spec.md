## ADDED Requirements

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

The library SHALL keep one kind-class weight table in `opm/k8s/object`. Its values SHALL be those of the cli's `pkg/resourceorder` at the time of this change (cli `origin/main` `0b37e3f2`, the order cli#289 applies by). A CustomResourceDefinition weighs -100. A Namespace weighs 0. ClusterRoles and ClusterRoleBindings weigh 5. ServiceAccounts, Roles and RoleBindings weigh 10. Secrets and ConfigMaps weigh 15. StorageClasses, PersistentVolumes and PersistentVolumeClaims weigh 20. Services weigh 50. Deployments, StatefulSets, DaemonSets and ReplicaSets weigh 100. Jobs and CronJobs weigh 110. Ingresses and NetworkPolicies weigh 150. Horizontal and vertical pod autoscalers and PodDisruptionBudgets weigh 200. Validating and mutating webhook configurations weigh 500. Every other kind weighs 1000. A weight SHALL be looked up by group, version and kind first, then by kind alone. The library SHALL provide a stable sort by weight, ascending for apply and descending for delete. Source: 0012:D5.

#### Scenario: Apply order puts definitions before their users

- **WHEN** a Deployment, a Namespace, a CustomResourceDefinition, a custom resource and a PersistentVolumeClaim are sorted ascending
- **THEN** the order is the CustomResourceDefinition, the Namespace, the PersistentVolumeClaim, the Deployment, the custom resource

#### Scenario: Delete order is the reverse and stable

- **WHEN** two Services and a Deployment are sorted descending
- **THEN** the Deployment comes first and the two Services keep their relative input order

#### Scenario: An unknown version of a known kind falls back to the kind

- **WHEN** the weight of `autoscaling/v2beta2 HorizontalPodAutoscaler` is looked up
- **THEN** it is 200

### Requirement: Apply stages follow the weight table

The library SHALL provide a function that returns a sorted copy of an apply set cut into stages. The first stage holds the cluster definitions, which are every CustomResourceDefinition of `apiextensions.k8s.io` and every Namespace of the core group, and is marked as such. It is omitted when there are none. That stage spans two weights, -100 and 0; it is safe to submit in one call to an engine that re-sorts by Flux's order, because Flux orders a CustomResourceDefinition before a Namespace too. After it comes one stage per distinct weight of the remaining objects, in ascending weight. Order within a stage SHALL be the stable sort order. The input SHALL NOT be reordered, and no stage SHALL be empty. A frontend whose apply engine re-sorts within a call SHALL be able to submit one stage per call, so the engine can refine the library's order and never contradict it. Source: 0012:D4, 0012:D5.

#### Scenario: Stages for a typical module

- **WHEN** a CustomResourceDefinition, a Namespace, a ConfigMap, a Secret, a Deployment and a Service are staged
- **THEN** the stages are the cluster definitions (the CustomResourceDefinition and the Namespace), then the ConfigMap and the Secret, then the Service, then the Deployment

#### Scenario: A Namespace kind in another group is not a cluster definition

- **WHEN** an object of kind `Namespace` in group `example.com` is staged beside a core Namespace
- **THEN** only the core Namespace is in the cluster-definition stage

#### Scenario: No cluster definitions, no definition stage

- **WHEN** a set holding only a Deployment and a Service is staged
- **THEN** two stages are returned, neither marked as cluster definitions

### Requirement: Differences from Flux's apply order are recorded

The library SHALL keep a test that lists the kinds of Flux's `ReconcileOrder` for the Flux version the operator pins, as literals without importing Flux. Its universe SHALL be Flux's listed kinds, the kinds of the library's kind table and one representative custom kind, with every kind Flux does not list at Flux's rank 0. A pair SHALL count only when the Flux ranks differ and the library weights are strictly opposite. The test SHALL assert that the set of such pairs equals a committed list, so that adding or removing such a difference fails the test until the list is edited, and SHALL assert that a CustomResourceDefinition before a Namespace is not such a pair. Source: 0012:D4.

#### Scenario: A weight edit that creates a contradiction is caught

- **WHEN** a change moves the Deployment weight below the Service weight
- **THEN** the Flux comparison test fails naming the new Service and Deployment pair

#### Scenario: Kinds Flux does not list are compared at rank 0

- **WHEN** the comparison runs over a PersistentVolumeClaim, a custom resource and a validating webhook configuration
- **THEN** the PersistentVolumeClaim after Deployment and the webhook configuration after the custom resource are both in the committed list

## MODIFIED Requirements

### Requirement: The Kubernetes tier imports no cluster client or controller framework

Beyond the standard library and the CUE SDK that the kernel's output types carry, a file under `opm/k8s/` MAY import only the kernel's exported packages and `k8s.io/apimachinery`. No file under `opm/k8s/` SHALL import `k8s.io/client-go`, `sigs.k8s.io/controller-runtime`, any package under `github.com/fluxcd/`, any cluster client, or a package under `opm/internal/` or `opm/helper/`, and of the Kubernetes modules it SHALL import only `k8s.io/apimachinery`. The repository lint gate SHALL enforce these denials, including the refusal of every other `k8s.io` and `sigs.k8s.io` module. For non-test files the gate SHALL also enforce the allowed set as a strict allow list: the Go standard library, `cuelang.org/go/cue`, `k8s.io/apimachinery` and the library's own `opm/` packages, so any other third-party import is refused without a reviewer having to notice it. Source: 0012:D3.

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
