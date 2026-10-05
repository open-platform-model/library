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
- A stalled-rollout status value (HP4 adds a predicate instead).
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

### HP4: A `ProgressDeadlineExceeded` predicate tells a stall apart

**Context**: op-f5 must stop fast requeues when a rollout has stalled: its plan entry requires
that `ProgressDeadlineExceeded` leads to Healthy=False with a reason and that fast requeues
stop. `Evaluate` folds that case into `NotReady`, so the status alone cannot tell "still
rolling out" from "stalled". op-f5 gates on the library release that follows this change.
**Explored**: (a) a new status value, which changes the cli's output; (b) an additive exported
predicate in `opm/k8s/health`; (c) op-f5 reads the Progressing condition itself.
**Decision**: (b), in this change:

```go
// ProgressDeadlineExceeded reports whether obj is a Deployment whose controller has
// observed its current generation and reports the rollout stalled: a Progressing
// condition with the reason ProgressDeadlineExceeded.
func ProgressDeadlineExceeded(obj *unstructured.Unstructured) bool
```

It reads through the moved `getConditions` and `generationObserved`, and `Evaluate` is
unchanged (it still returns `NotReady` for such a Deployment). The generation check keeps a
condition left over from the previous rollout from reporting a fresh apply as stalled before the
controller has seen it. Any other kind returns false.
**Rationale**: (a) breaks the verbatim move. (c) puts a readiness rule back into a frontend,
against ADR-011 items 1 and 3 and the owner's f5 decision ("Evaluator moves to
opm/k8s/health"). (b) proposed later by op-f5 would add a second library release before op-f5
could start. The predicate is one function and one test; the requeue policy that reads it stays
op-f5's.

### HP5: Tests are the cli's, moved; the package is internal-tested

**Context**: The cli's tests are in package `kubernetes` and use unexported helpers
(`makeResource`, `makeWorkload`).
**Decision**: Move them into `opm/k8s/health/health_test.go`, package `health`, renamed to
the new identifiers, with every case and expectation unchanged. The cli's task-number header
comment is dropped, and its issue reference is spelled `cli#228` so it does not read as a
library issue. Add `TestStatusStrings` (the
seven literals), `TestAggregate` (all healthy, one unhealthy plus missing, empty, `Applied`
counted healthy, unhealthy only), the HP2 equivalence test and `TestProgressDeadlineExceeded` (HP4). testify is already a library
test dependency, and the tier's allow list covers non-test files only.
**Rationale**: The cli's tests are the evidence that the behaviour did not change in the move.

## Risks / Trade-offs

- [The cli and the library copy drift before cli-f5 lands] → cli-f5 deletes the cli copy with
  no alias in the release that adopts the package. Until then, a fix to either copy is ported
  to the other by hand. The cli source commit is recorded in a maintainer comment in `doc.go`,
  separate from the package doc (which opm-docs publishes and which keeps no maintainer
  pointers), so the drift can be checked.
- [Pure evaluation of a stale read] → `Evaluate` judges the object it is given. A frontend
  that reads through a cache right after an apply can get the object as it was before the
  apply, whose `metadata.generation` the controller has already observed, and `Evaluate` then
  reports the old rollout as `Ready`. The library cannot detect that. Reading the object after
  the apply, uncached, is the frontend's job (op-f5 plans to), and the package doc says so.

## Migration Plan

None. The package is new. cli-f5 and op-f5 adopt it.
