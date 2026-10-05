## Context

See proposal.md, Why. Design-local decisions are numbered HP1 to HP5 so they collide with no
other numbering. Line references are at library `origin/main` `ca7c56b` (after
add-kubernetes-object-packages, library#196), cli `origin/main` `bd4d1a7c`, fetched
2026-10-05. The evidence is the wave-2 research entry f5
(`claude-stuff/kernel-plan-beta1/wave2-plan-result.json`), re-checked at those heads:
`internal/kubernetes/health.go` (339 lines) and `health_test.go` (507 lines) last changed in
`bd4d1a7c` (cli#314, which renamed `QuickInstanceHealth`'s second parameter to
`unhealthyCount` so unreadable resources count against readiness).

The source file uses five kind constants (`kindDeployment`, `kindStatefulSet`,
`kindDaemonSet`, `kindJob`, `kindPersistentVolumeClaim`) declared in the cli's
`internal/kubernetes/tree.go:111-116`. The library package declares its own, unexported.

## Goals / Non-Goals

**Goals:**
- One readiness evaluator in the tier, with the cli's behaviour and strings unchanged.
- An aggregate both frontends can call with statuses they already hold.
- The cli's tests as the library's tests.

**Non-Goals:**
- Any frontend edit.
- New or changed evaluation rules.
- A stalled-rollout verdict (HP4).
- Fetching, waiting or polling.

## Research & Decisions

### HP1: Names: `health.Status`, constants without the `Health` prefix

**Context**: The cli exports `HealthStatus`, `HealthReady` and so on from a package named
`kubernetes`. In a package named `health` those names stutter (`health.HealthReady`).
add-kubernetes-object-packages dropped the same stutter from the label names (its design KO2).
**Explored**: Keep the cli names verbatim, so cli-f5 is a pure import swap; or drop the prefix.
cli-f5 has to touch every call site anyway, because the qualifier changes from `kubernetes.` to
`health.` and ADR-011 item 3 forbids an alias.
**Decision**: `type Status string`; constants `Ready`, `NotReady`, `Complete`, `Unknown`,
`Missing`, `Applied`, `Bound`; functions `Evaluate`, `IsHealthy`, `Aggregate`. The string
values are byte-equal to the cli's, and a test pins each one to its literal.
**Rationale**: The renames cost cli-f5 nothing extra, and the strings, the part the cli's
output depends on, do not move. `IsHealthy` keeps its name because the plan and both adoption
changes name it, and `health.Healthy` would read as a status value.

### HP2: `Aggregate` takes statuses, not objects

**Context**: The cli's `QuickInstanceHealth(resources, unhealthyCount)` evaluates each object
itself and returns the aggregate, the ready count and the total. The cli's status view
(`status.go:205-226`) and tree view already evaluate each object for display and then
aggregate by hand with the same rule. op-f5 will evaluate each inventory object once and needs
both the per-object statuses (to name the unhealthy kinds in the condition message) and the
aggregate.
**Explored**: (a) Port `QuickInstanceHealth` as it is, taking objects. Callers that already
hold statuses would evaluate twice or keep the hand-written aggregate. (b) Take statuses.
**Decision**: (b).

```go
// Aggregate counts the healthy statuses among statuses. unhealthy is the number of
// tracked objects that have no status to give: missing from the cluster, or not readable.
// Each counts toward total and never toward ready.
func Aggregate(statuses []Status, unhealthy int) (status Status, ready, total int) {
	total = len(statuses) + unhealthy
	if total == 0 {
		return Unknown, 0, 0
	}
	for _, s := range statuses {
		if IsHealthy(s) {
			ready++
		}
	}
	if ready == total {
		return Ready, ready, total
	}
	return NotReady, ready, total
}
```

For any input, `QuickInstanceHealth(rs, n)` equals `Aggregate` over `Evaluate(r)` for each `r`
in `rs`, with the same `n`. A test checks that equivalence over the cli's own fixtures, so
cli-f5 can swap `QuickInstanceHealth` for a one-line loop. A negative `unhealthy` is the
caller's bug; `Aggregate` does not check it, as `QuickInstanceHealth` does not.
**Rationale**: One evaluation per object per pass, and one aggregation rule in the tier instead
of a library one plus the cli's hand-written copy in `status.go`.

### HP3: No new kinds

**Context**: The research noted that ReplicaSets and ReplicationControllers are not evaluated:
they fall to the condition rule and report `Applied` because they carry no `Ready` condition.
**Decision**: Unchanged. The move is verbatim (owner f5), and the cli's output for such an
object must stay the same when it adopts the package.
**Rationale**: A new rule is a behaviour change for the cli's `instance status`. If op-f5 or a
module author needs one, it is its own change with its own test.

### HP4: The progress-deadline distinction is not decided here

**Context**: op-f5 must stop fast requeues when a rollout has stalled, and the plan names
`ProgressDeadlineExceeded` as the signal. `Evaluate` folds that case into `NotReady`, so the
status alone cannot tell "still rolling out" from "stalled".
**Explored**: (a) a new status value, which changes the cli's output; (b) an additive exported
predicate in `opm/k8s/health`, such as one that reports whether a Deployment's Progressing
condition carries `ProgressDeadlineExceeded`; (c) op-f5 reads the condition itself.
**Decision**: None of them in this change. (a) breaks the verbatim move. (b) and (c) are op-f5's
design choice, and (b) is a small additive change to this package that op-f5 can propose with
its own test. The unexported condition reader moves as it is, so (b) needs no rework.
**Rationale**: The owner decided a verbatim move here and left the operator's requeue policy to
op-f5. Adding API now would guess at a caller that has not been designed.

### HP5: Tests are the cli's, moved; the package is internal-tested

**Context**: The cli's tests are in package `kubernetes` and use unexported helpers
(`makeResource`, `makeWorkload`).
**Decision**: Move them into `opm/k8s/health/health_test.go`, package `health`, renamed to
the new identifiers, with every case and expectation unchanged. Add `TestStatusStrings` (the
seven literals), `TestAggregate` (all healthy, one unhealthy plus missing, empty, `Applied`
counted healthy, unhealthy only) and the HP2 equivalence test. testify is already a library
test dependency, and the tier's allow list covers non-test files only.
**Rationale**: The cli's tests are the evidence that the behaviour did not change in the move.

## Risks / Trade-offs

- [The cli and the library copy drift before cli-f5 lands] → cli-f5 deletes the cli copy with
  no alias in the release that adopts the package. Until then, a fix to either copy is ported
  to the other by hand. Record the cli source commit in the package doc so the drift can be
  checked.
- [Pure evaluation of a stale read] → `Evaluate` judges the object it is given. A frontend
  that reads through a cache right after an apply can get the object as it was before the
  apply, whose `metadata.generation` the controller has already observed, and `Evaluate` then
  reports the old rollout as `Ready`. The library cannot detect that. Reading the object after
  the apply, uncached, is the frontend's job (op-f5 plans to), and the package doc says so.

## Migration Plan

None. The package is new. cli-f5 and op-f5 adopt it.
