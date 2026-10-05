## MODIFIED Requirements

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

## REMOVED Requirements

### Requirement: Differences from Flux's apply order are recorded

**Reason**: The table now agrees with Flux's staged apply order, so there are no differences to record. The committed list of contradicting pairs is replaced by a guard that requires the list to be empty and that also counts Flux's stages and its group-and-kind tie-break.

**Migration**: None for consumers. The guard is the new requirement "The weight table never contradicts Flux's staged apply order".

## ADDED Requirements

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
