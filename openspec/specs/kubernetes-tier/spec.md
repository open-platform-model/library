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
