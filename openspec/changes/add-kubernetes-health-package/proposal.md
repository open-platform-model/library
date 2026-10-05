## Why

ADR-011 and 0012:D3 put readiness evaluation in the Kubernetes tier, and ADR-011 item 1 names
`health` among the indicative packages. Today the only evaluator OPM has is the cli's
`internal/kubernetes/health.go` (cli `origin/main` `bd4d1a7c`). It decides when a Deployment,
StatefulSet or DaemonSet has rolled out, when a Job is complete, what a PersistentVolumeClaim's
phase means, and when a custom resource is ready. The operator has none: it sets Ready=True
once the apply succeeds and never looks at rollout state.

The owner decided task f5 in the beta-1 kernel-plan walkthrough (2026-10-02/03): "Evaluator
moves to opm/k8s/health (pure, frontend fetches objects); CLI switches. Operator adds a separate
Healthy condition (requeue until rolled out); Ready keeps meaning applied for now; whether Ready
requires Healthy decided later." This change is the library half. The operator's Healthy
condition (op-f5) and the cli switch (cli-f5) are later changes that adopt the first library
release containing this one.

The evaluator moves verbatim, status strings included. The cli serialises the status into
`opm instance status -o json|yaml` (`internal/kubernetes/status.go`, `resourceHealth.Status`
and `StatusResult.AggregateStatus`), so the strings are a machine-readable contract. A
PersistentVolumeClaim with a phase reports that raw phase (`Pending`, `Lost`), and that
passthrough is kept too. If the library normalised any of it, the cli's output would change
when it adopts the package.

## What Changes

- **`opm/k8s/health`** (new, on `k8s.io/apimachinery` only), ported from cli
  `internal/kubernetes/health.go` at `bd4d1a7c`:
  - `Status`, a string type, with the seven values `Ready`, `NotReady`, `Complete`, `Unknown`,
    `Missing`, `Applied` and `Bound`, byte-equal to the cli's. The constant names drop the
    cli's `Health` prefix, which would stutter as `health.HealthReady` (design HP1).
  - `Evaluate(*unstructured.Unstructured) Status`, the cli's `EvaluateHealth` with its logic
    unchanged: rollout state for Deployments (observed generation, the
    `ProgressDeadlineExceeded` Progressing reason, updated, available and total replicas),
    StatefulSets (counters, RollingUpdate partition, OnDelete) and DaemonSets; Job `Complete`
    and `Failed`; CronJob and the fixed passive kinds as `Applied`; a PersistentVolumeClaim's
    raw `status.phase`, or `Ready` when it has none; any other kind by its `Ready` condition,
    or `Applied` when it reports none.
  - `IsHealthy(Status) bool`, unchanged: `Ready`, `Applied`, `Complete` and `Bound` are healthy.
  - `Aggregate(statuses []Status, unhealthy int) (status Status, ready, total int)`, the cli's
    `QuickInstanceHealth` with the per-object evaluation lifted out (design HP2). It takes
    statuses, not objects, because every caller already holds them: the cli's status and tree
    views evaluate each object for display, and the operator will evaluate each inventory
    object once. `QuickInstanceHealth(resources, n)` equals `Aggregate` over `Evaluate` of each
    resource with the same `n`.
  - The package is pure: the caller fetches the live objects with its own client and passes
    them in. Nothing in it reads a cluster, waits or polls (ADR-008 rule 1, ADR-011 item 4).
  - The cli's table tests move with it as the library's tests, plus a test for `Aggregate`'s
    three outcomes and one that pins the seven status strings.
- **Docs**: the `opm/k8s/` package lists in `README.md`, `AGENTS.md` and `CONSTITUTION.md`
  gain `health`. The ADR-011 Status records that the readiness evaluator arrived.
- **Spec**: `kubernetes-tier` gains the readiness requirement.

Not **BREAKING**. Every addition is a new package; nothing is removed or renamed, and neither
frontend imports anything that changes. The tier's existing allow list already admits the
package's imports.

SemVer class: MINOR. Release class of the PR: `feat`.

## Not in this change

- **The cli switch** (cli-f5): deleting `internal/kubernetes/health.go` and moving its call
  sites (status, tree, wait, pods, operator wait, query status and list) to `opm/k8s/health`
  with no aliases, keeping JSON and YAML status output byte-stable.
- **The operator's Healthy condition** (op-f5): reading each inventory object through the
  impersonated client after a successful apply, the RolledOut and NotRolledOut reasons, the
  requeue floor and ceiling, the printcolumn. Ready keeps meaning "applied". Whether Ready
  requires Healthy is left for the owner to decide later.
- **New evaluation rules.** No kind is added or re-judged. A ReplicaSet or
  ReplicationController rendered by a module still falls to the condition rule and reports
  `Applied`, as in the cli today (design HP3).
- **A stalled-rollout verdict.** `Evaluate` reports a Deployment past its progress deadline
  as `NotReady`, the same string as one still rolling out. op-f5 needs to tell the two apart
  to stop fast requeues. This change does not decide where that check lives (design HP4).
- **The CRD `Established` wait** the cli added in cli#289. It is a separate predicate in the
  cli's `wait.go`, not part of this evaluator.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `kubernetes-tier`: readiness evaluation lives in `opm/k8s/health`, a pure evaluator over
  objects the frontend fetches, with the cli's status strings.

## Impact

- Packages: new `opm/k8s/health`. No existing package changes.
- Repo files: `README.md`, `AGENTS.md`, `CONSTITUTION.md`,
  `adr/011-kubernetes-tier-beside-the-kernel.md`. `go.mod` and `go.sum` do not change:
  `k8s.io/apimachinery` is already required.
- Downstream: cli and opm-operator compile unchanged against this tree; the consumer-build job
  must stay green. cli-f5 and op-f5 gate on the first library release that contains this
  change.
- Ordering (wave-2 serialization): written in parallel with lib-e3 (`opm/k8s/inventory`) and
  lib-e4 (`opm/k8s/ownership`). All three are new packages, but each edits the same `opm/k8s/`
  package lines in `README.md`, `AGENTS.md`, `CONSTITUTION.md` and the ADR-011 Status, so
  whichever merges second merges `origin/main` and keeps every package in those lines.
- `enhancement.yaml` declares 0012 with no decision claimed. 0012:D3 lists readiness evaluation
  among the tier's decisions, but it is delivered only once both frontends use the tier
  (ADR-011 item 3), so the claim belongs to the adoption changes.
