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
- **THEN** they are `safety-excluded`, `already-absent`, `not-opm-managed`, `owner-mismatch`, `terminating`, `foreign-object` and `other-instance`

### Requirement: Safety-excluded kinds match on group and kind

The library SHALL report a core-group `Namespace` and an `apiextensions.k8s.io` `CustomResourceDefinition` as safety-excluded, and no other group and kind pair. The test SHALL NOT need a live object. Source: 0012:D1:R3, the 0012 contract's `#safetyExcluded`.

#### Scenario: The two protected kinds are excluded

- **WHEN** the core group with kind `Namespace`, and `apiextensions.k8s.io` with kind `CustomResourceDefinition`, are tested
- **THEN** both are safety-excluded

#### Scenario: A same-named kind in another group is not excluded

- **WHEN** `example.com` with kind `Namespace`, and `example.com` with kind `CustomResourceDefinition`, are tested
- **THEN** neither is safety-excluded

### Requirement: The delete verdict skips with a reason or proceeds with the judged object's identity

The delete verdict SHALL decide in this order and stop at the first match: skip as `safety-excluded` when the kind is safety-excluded, whatever the live object; skip as `already-absent` when there is no live object; skip as `not-opm-managed` when the live managed-by label is not an OPM runtime's value; skip as `owner-mismatch` when the live UUID label and the instance's UUID are both non-empty and differ; otherwise proceed. An empty UUID on either side SHALL pass the owner comparison. An object being deleted SHALL NOT be skipped for that reason. A proceed verdict SHALL carry the UID and the resourceVersion of the live object it judged. Its DELETE precondition SHALL name that UID and SHALL NOT name a resourceVersion. A caller MAY add the carried resourceVersion itself. A skip verdict, and a proceed verdict whose judged UID is empty, SHALL yield no precondition at all, never a precondition on an empty UID. A skip verdict SHALL carry a message naming the object and the reason. Source: 0012:D1:R3/R4, 0012:D4:R1.

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

- **WHEN** the live Deployment is OPM-managed and either its UUID label or the instance UUID is empty
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

### Requirement: The apply verdict refuses terminating, foreign and other-instance objects

The apply verdict SHALL decide in this order and stop at the first match. When there is no live object, it SHALL apply. When the live object has a deletion timestamp, it SHALL refuse as `terminating`, whether or not the object is in the instance's recorded inventory, and nothing SHALL lift that refusal. When the object is in the instance's recorded inventory, it SHALL apply. When the live object's adopt annotation equals the non-empty instance UUID, it SHALL apply. When the live managed-by label is not an OPM runtime's value, it SHALL refuse as `foreign-object`, unless the operator install admission lifts it. When the live UUID label is non-empty and differs from the instance UUID, it SHALL refuse as `other-instance`. Otherwise it SHALL apply. An empty instance UUID SHALL never match an adopt annotation, and any non-empty live UUID SHALL then count as another instance's. Source: 0012:D8:R1/R2/R5, 0012:D1:R7, 0012:D4:R2.

#### Scenario: A new object is applied

- **WHEN** the apply verdict is asked for an object with no live object
- **THEN** it applies

#### Scenario: A terminating object is refused everywhere

- **WHEN** the live object has a deletion timestamp
- **THEN** the verdict refuses as `terminating` when the object is in the inventory, when it is outside it, when it carries the adopt annotation naming this instance, and when it is admitted

#### Scenario: An inventoried object is not judged for ownership

- **WHEN** the object is in the instance's recorded inventory and its live managed-by label is `helm`
- **THEN** the verdict applies

#### Scenario: An inventoried object annotated for another instance still applies

- **WHEN** the object is in the instance's recorded inventory, carries this instance's UUID label, and its adopt annotation names a different instance
- **THEN** the verdict applies

#### Scenario: A foreign object outside the inventory is refused

- **WHEN** the object is outside the inventory and its live managed-by label is missing or is `kustomize`
- **THEN** the verdict refuses as `foreign-object`

#### Scenario: Another instance's object outside the inventory is refused

- **WHEN** the object is outside the inventory, OPM-managed, and its UUID label differs from the instance UUID
- **THEN** the verdict refuses as `other-instance`

#### Scenario: An OPM object without a UUID label is applied

- **WHEN** the object is outside the inventory, OPM-managed, and carries no UUID label
- **THEN** the verdict applies

#### Scenario: An OPM object of this instance outside the inventory is applied

- **WHEN** the object is outside the inventory, OPM-managed, and its UUID label equals the instance UUID
- **THEN** the verdict applies

#### Scenario: An empty instance UUID fails closed

- **WHEN** the instance UUID is empty and the object outside the inventory is OPM-managed with a non-empty UUID label
- **THEN** the verdict refuses as `other-instance`

### Requirement: The adopt annotation is the only override and the refusal names it

An adopt annotation whose value equals the instance UUID SHALL lift the `foreign-object` and `other-instance` refusals. The annotation value SHALL be compared with surrounding whitespace trimmed, and a value that is empty or only whitespace SHALL count as no annotation. An adopt annotation with any other value SHALL lift nothing, and the refusal message SHALL say that the annotation names another instance. The refusal message for `foreign-object` and `other-instance` SHALL name the object, the annotation key `opmodel.dev/adopt` and the instance UUID to set it to, unless the instance UUID is empty. The `other-instance` message SHALL ask the user to remove the object from the instance that owns it before annotating it, since that instance applies the object again for as long as it holds it in its inventory. No message SHALL name any other way past a refusal, such as a command-line flag. No message SHALL carry an enhancement reference. Source: 0012:D8:R2/R3.

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

Both verdicts SHALL take an admission input that a caller sets only for an object it has proven came from an earlier operator release's install manifest. On apply, admission SHALL lift `foreign-object` only, and only when the live object carries no UUID label or carries the instance UUID. On delete, admission SHALL lift `not-opm-managed` only, only when the live object carries no UUID label at all, and only for an `apps` `Deployment`, a `rbac.authorization.k8s.io` `RoleBinding` or a `rbac.authorization.k8s.io` `ClusterRoleBinding`, the kinds install may delete outside the inventory. The UUID label is the only identity admission compares. Admission SHALL NOT lift `terminating`, `other-instance`, `owner-mismatch`, `safety-excluded` or `already-absent`. The library does not check the proof. Source: 0012:D8:R6/R7, 0012:D4:R1.

#### Scenario: A proven earlier-manifest object is admitted on apply

- **WHEN** the operator install asks for the earlier `Namespace/opm-operator-system`, live with managed-by `kustomize` and no UUID label, admitted and outside the inventory
- **THEN** the verdict applies

#### Scenario: An admitted object carrying another identity is refused

- **WHEN** an admitted object outside the inventory is not OPM-managed but carries a UUID label that differs from the instance UUID
- **THEN** the apply verdict refuses as `foreign-object`

#### Scenario: Admission never lifts the terminating refusal

- **WHEN** an admitted proven object has a deletion timestamp
- **THEN** the apply verdict refuses as `terminating`

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
