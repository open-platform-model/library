## ADDED Requirements

### Requirement: Readiness evaluation is a pure function over fetched objects

The library SHALL provide `opm/k8s/health`, which judges the readiness of one Kubernetes object that the caller has already fetched, and aggregates such judgements. Nothing in the package SHALL read a cluster, wait or poll; the frontend fetches each live object with its own client and passes it in. A status SHALL be one of the strings `Ready`, `NotReady`, `Complete`, `Unknown`, `Missing`, `Applied` and `Bound`, byte-equal to the cli's `internal/kubernetes/health.go` at cli `bd4d1a7c`, or, for a PersistentVolumeClaim, the raw value of its `status.phase`. The evaluation rules SHALL be the cli's at that commit, unchanged:

- A Deployment is `Ready` only when `status.observedGeneration` has reached `metadata.generation`, no Progressing condition carries the reason `ProgressDeadlineExceeded`, and the updated and available replica counts equal `spec.replicas` (1 when omitted) with no more total replicas than updated ones; otherwise `NotReady`.
- A StatefulSet is `Ready` when the generation is observed and the ready replicas equal `spec.replicas`, and then: always under the `OnDelete` strategy; under a RollingUpdate partition greater than zero, when the updated replicas reach `spec.replicas` minus the partition; otherwise when the updated replicas equal `spec.replicas` and the update revision equals the current revision. Otherwise `NotReady`.
- A DaemonSet is `Ready` when the generation is observed and the updated and available counts equal `status.desiredNumberScheduled`; otherwise `NotReady`.
- A Job is `Complete` when its `Complete` condition is `True`, and `NotReady` otherwise.
- A CronJob, and every kind of the fixed passive set (ConfigMap, Secret, Service, ServiceAccount, Namespace, ClusterRole, ClusterRoleBinding, Role, RoleBinding, Ingress, NetworkPolicy, PodDisruptionBudget, ResourceQuota, LimitRange, StorageClass, PriorityClass), is `Applied`.
- A PersistentVolumeClaim reports its `status.phase` as the status (`Bound` for a bound claim), or `Ready` when it has no phase yet.
- Any other kind is `Ready` when its `Ready` condition is `True`, `NotReady` when that condition has any other status, and `Applied` when it has no `Ready` condition.

`Ready`, `Applied`, `Complete` and `Bound` SHALL be the healthy statuses, and every other string SHALL be unhealthy. The aggregate of a list of statuses and a count of tracked objects that have no status (missing or unreadable) SHALL count each of those objects toward the total and never toward ready, and SHALL be `Unknown` when the total is zero, `Ready` when every counted object is healthy, and `NotReady` otherwise, together with the ready count and the total. Source: 0012:D3; the owner's f5 decision of the beta-1 walkthrough (2026-10-02/03).

#### Scenario: The status strings match the cli's output

- **WHEN** a table test pins each status constant to its literal
- **THEN** the seven constants are `Ready`, `NotReady`, `Complete`, `Unknown`, `Missing`, `Applied` and `Bound`

#### Scenario: A Deployment mid-rollout is not ready

- **WHEN** a Deployment whose generation is observed and whose Available condition is `True` reports fewer updated replicas than `spec.replicas`
- **THEN** its status is `NotReady`

#### Scenario: A Deployment past its progress deadline is not ready

- **WHEN** a Deployment whose replica counters all equal `spec.replicas` carries a Progressing condition with reason `ProgressDeadlineExceeded`
- **THEN** its status is `NotReady`

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

- **WHEN** a developer reads the exported functions of `opm/k8s/health`
- **THEN** each takes objects or statuses as arguments, none takes a client, a REST config or a context, and none waits or polls
