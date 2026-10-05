## Context

0012:D4 gives the library the whole deletion sequence: the plan, the transition that names the next
action, and the hold verdict. Both frontends' delete paths must use it. ADR-008 fixes the shape.
The library plans and the caller runs (rule 1). A plan advances one step per call, from a
serialisable state the caller owns (rule 2). The library names an action and never performs one
(rule 3). ADR-008's Deletion plans paragraph fixes the input: the persisted inventory plus the
live objects, with no render and no stored plan. A prune's stale set comes from the inventory
package and goes into the same plan.

Three tier packages already exist on `origin/main` (b2d51d7, library v1.0.0-beta.6):

- `opm/k8s/inventory`: `Entry{Group, Kind, Namespace, Name, Version, Component}`, with no
  struct tags, and `StaleSet(previous, current)`, which returns the previous entries in previous
  order.
- `opm/k8s/ownership`: `SafetyExcluded(group, kind)` and `CanDelete(DeleteInput) DeleteVerdict`.
  A skip carries a `SkipReason` and a message. A proceed carries the UID and resourceVersion, and
  `Preconditions()` turns them into a precondition that holds the UID only.
- `opm/k8s/object`: `Weight(gvk)` and `Sort(items, gvkOf, Descending)`, a stable sort.

The two frontends today (cli `bd4d1a7c`, opm-operator `bcfa722`):

| | cli `kubernetes.Delete` | operator `apply.Prune` and `handleDeletion` |
| --- | --- | --- |
| Order | descending weight | stored inventory order |
| Live re-read | yes, per object | yes, per object |
| Ownership skip | managed-by and UUID | managed-by and UUID, CRD and Namespace by kind only |
| Propagation | Foreground | client default (Background) |
| NotFound on read or delete | already gone, not an error | already gone, not an error |
| Other error | per-resource error; continue; instance kept | joined error; continue; finalizer kept |
| Hold | none (0012:OQ6) | finalizer `opmodel.dev/cleanup`; release branches below |

The operator's `handleDeletion` branches, in this order:

1. `!spec.prune`: remove the finalizer.
2. The inventory is empty: remove the finalizer.
3. The impersonation ServiceAccount is missing and `opm.dev/force-delete-orphan: "true"` is set:
   clear the inventory and remove the finalizer.
4. The ServiceAccount is missing, without the annotation: stall (`DeletionSAMissing`) and hold.
5. Any other impersonation error: stall (`ImpersonationFailed`) and hold.
6. The prune returned an error that is Forbidden: stall and hold.
7. The prune returned any other error: return the error and hold. The controller requeues.
8. Otherwise: remove the finalizer.

## Goals / Non-Goals

**Goals:**

- One package, `opm/k8s/lifecycle`, holding `DeletionPlan`, `State`, `Advance` and
  `MayReleaseHold`. Given the same inventory, policy, owner UUID, live objects and outcomes, every
  frontend gets the same order, actions, skip reasons and hold verdict (0012:D1:R1/R5/R6).
- A frontend makes progress only by asking for the next action, so it has no way to skip a guard
  (ADR-011 item 3).
- The state can be written out and read back, and advances identically (0012:D4:R5).
- No hook semantics (0012:D4:R6).

**Non-Goals:**

- Frontend loops, status conditions, events, dry-run reporting and exit codes. These stay with the
  frontends.
- Answering 0012:OQ5 or 0012:OQ6.
- A stored plan, a stored state or a new CRD field.
- The operator install's admitted deletions outside the inventory (0012:D8:R7).

## Research & Decisions

### LC1: The plan is ordered entries with safety exclusions marked up front

**Context**: 0012:D1:R6 asks for one defined deletion order. The 0012 contract describes a plan as
"Ordered. Reverse apply-weight". `object.Sort` already exists and is stable.

**Explored**: three options. (a) Keep inventory order. That is the operator's order today, and
0012:D1:R6 rules it out. (b) Sort inside `Advance`. That is redundant on every call, and the plan
would no longer be something a frontend can list up front. (c) Sort once in the constructor.

**Decision**: (c).

```go
type Policy struct {
    Prune       bool // remove the instance's objects; false orphans them
    ForceOrphan bool // the operator's escape hatch (opm.dev/force-delete-orphan)
}

type Step struct {
    Entry inventory.Entry
    Skip  ownership.SkipReason // SkipSafetyExcluded when marked up front, else ""
}

type DeletionPlan struct {
    OwnerUUID string
    Policy    Policy
    Steps     []Step
}

func NewDeletionPlan(entries []inventory.Entry, policy Policy, ownerUUID string) DeletionPlan
```

`NewDeletionPlan` copies `entries` and never changes the caller's slice. It sorts the copy with
`object.Sort(steps, gvkOf, object.Descending)`, where `gvkOf` reads the entry's group, version and
kind. It sets `Skip = ownership.SkipSafetyExcluded` on each step where
`ownership.SafetyExcluded(group, kind)` holds, so every frontend lists those steps as left behind
without a live read. Duplicate entries are kept: the second read of an object finds it absent and
skips it as `already-absent`. An empty or nil `entries` gives a plan with no steps.

Uninstall passes the persisted inventory. A prune passes `inventory.StaleSet(previous, current)`.
There is one constructor and no separate prune plan.

**Rationale**: the order is a value the frontend can show before it acts, it is computed once, and
it comes from the one weight table (0012:D5).

### LC2: The transition names one action per call

**Context**: ADR-008 rule 2: "Advancing is a pure function of a plan and a state value that
returns the next state and the action the caller is to perform." The caller must hand back what it
saw, either the live object or the outcome of a delete.

**Explored**: two options. (a) A sum type through interfaces (`NeedLive`, `Delete`, `Skipped`,
`Done`). (b) One `Action` struct with a `Kind` and an `Event` struct in. (a) is idiomatic for
pattern matching, but an interface value does not survive a JSON round trip without a type switch
the library would have to own. (b) is a plain value, prints well in a test table, and matches the
contract's `action: "delete" | "skip"` literal style.

**Decision**: (b).

```go
type ActionKind string

const (
    ActionRead   ActionKind = "read"   // GET plan.Steps[Step].Entry; feed back Event{Live} or Event{Err}
    ActionDelete ActionKind = "delete" // DELETE it with Propagation and Preconditions; feed back Event{Err}
    ActionSkip   ActionKind = "skip"   // report Skip and Message; the state has already moved on
    ActionDone   ActionKind = "done"
)

type Action struct {
    Kind          ActionKind
    Step          int             // index into plan.Steps; -1 for done
    Entry         inventory.Entry // zero for done
    Propagation   metav1.DeletionPropagation // delete only: Foreground
    Preconditions *metav1.Preconditions      // delete only: UID, from ownership.CanDelete
    Skip          ownership.SkipReason       // skip only
    Message       string                     // skip only, the library's wording
}

type Event struct {
    Live *unstructured.Unstructured // answer to read: the object read; nil when not found
    Err  error                      // the action's error; nil on success
}

func Advance(plan DeletionPlan, state State, ev Event) (State, Action, error)
```

The transition works like this:

- **The state awaits nothing.** `ev` is ignored.
  - If `!plan.Policy.Prune`, or every step is finished, the action is `done`.
  - Otherwise look at step `state.Next`. A step marked safety-excluded records a `skipped`
    outcome, moves `Next` on, and names `skip` with the `CanDelete` message for that object. Any
    other step sets the state to await a read and names `read`.
- **The state awaits a read.**
  - If `ev.Err` is NotFound, or `ev.Err` is nil and `ev.Live` is nil, the live object is absent.
  - If `ev.Err` is any other error, the step records a `failed` outcome and moves on, and
    `Advance` names the next step's action.
  - Otherwise `ownership.CanDelete` judges the step with `Object` from the entry, `Live`, and
    `InstanceUUID: plan.OwnerUUID`. A skip records a `skipped` outcome, moves on and names `skip`.
    A proceed sets the state to await a delete and names `delete`, with
    `Propagation: metav1.DeletePropagationForeground` and `Preconditions: verdict.Preconditions()`.
- **The state awaits a delete.**
  - If `ev.Err` is nil, the step records `deleted`.
  - If `ev.Err` is NotFound, the step records `skipped` with `already-absent` and names `skip`.
    The object went between the read and the DELETE.
  - If `ev.Err` is any other error, the step records `failed`. Then the next step's action is
    named.

A `skip` action is only a report: the returned state has already moved past the step. Every
`skip` from `Advance` comes from `CanDelete`, so its message is the library's wording. A frontend
running a dry run feeds `Event{}` back to a `delete` without performing it. That records
`deleted`, and the frontend words its own dry-run line.

`Advance` returns an error only when the inputs cannot belong together:

- `state.Next` is outside `0..len(plan.Steps)`;
- the state awaits something while `Next == len(plan.Steps)`;
- an `ev.Live` names a different group, kind, namespace or name than the step it answers.

On an error, `Advance` returns the input state unchanged and no action. Accepting a mismatched
answer would let a frontend delete an object the verdict never judged.

**Rationale**: every branch the frontend loop would otherwise hold (order, guard, propagation,
precondition, NotFound handling) is decided once. The frontend's loop is a switch over four kinds,
so it has nothing to decide.

### LC3: The library classifies the caller's error with apimachinery

**Context**: both frontends treat NotFound as already gone, and the operator holds the finalizer
differently on Forbidden. If each frontend classified its own errors, the classification could
drift.

**Explored**: two options. (a) The caller passes a pre-classified enum. (b) The caller passes the
raw error and the library classifies it with `k8s.io/apimachinery/pkg/api/errors`, which the tier
may import. `apierrors.IsNotFound`, `IsForbidden` and `IsConflict` look through `errors.As`
(apimachinery v0.36.4 `ReasonForError`), so a wrapped client error classifies correctly.

**Decision**: (b). An outcome records a `FailureClass`:

- `forbidden` when `apierrors.IsForbidden`;
- `conflict` when `apierrors.IsConflict`, which is how a failed UID precondition comes back;
- `error` for anything else.

It also records the error text as `Message`. NotFound is never a failure.

**Rationale**: one classification, in one place, over the error both frontends already hold.

### LC4: The state is a JSON value the caller owns, and the operator rebuilds it each pass

**Context**: 0012:D4:R5 requires a written-out state to advance identically to one held in memory.
The operator's reconcile is level-triggered. Persisting the state would need a new
ModuleInstance and ModulePackage status field, and nothing needs one: after a pass, a deleted
object reads as absent, so a fresh plan over the same inventory converges on the next pass.

**Decision**:

```go
type Awaiting string // "" | "read" | "delete"

type State struct {
    Next     int       `json:"next"`
    Awaiting Awaiting  `json:"awaiting,omitempty"`
    Outcomes []Outcome `json:"outcomes,omitempty"`
}

type Outcome struct {
    Step    int                  `json:"step"`
    Result  Result               `json:"result"`            // deleted | skipped | failed
    Skip    ownership.SkipReason `json:"skip,omitempty"`
    Failure FailureClass         `json:"failure,omitempty"` // forbidden | conflict | error
    Message string               `json:"message,omitempty"`
}
```

The zero `State` is the start. The live object is never part of the state: `Advance` judges it in
the same call it arrives in. The state carries struct tags, while `inventory.Entry` deliberately
does not. The difference is that the state is the library's own value, and R5 needs one encoding.
The JSON encoding is SemVer surface. The operator does not store it (op-f2 rebuilds it per
reconcile from `status.inventory` plus the live objects), and the cli holds it in memory.

**Rationale**: the state contains only data, so it survives a round trip (ADR-008 trade-off). The
round-trip test advances an in-memory state and a JSON-copied state through the same events after
every step, and requires equal actions and equal states.

### LC5: The hold verdict reproduces handleDeletion with inputs that do not depend on who holds the hold

**Context**: 0012:D1:R5 says whether an instance's hold may be released is decided from its policy
and the plan's outcome, identically for whichever frontend asks, with the reason named. The 0012
contract's `#HoldVerdict` lists seven reasons. The operator's branches are listed in Context. The
cli bears no hold today (0012:OQ6), and the verdict must not presume an answer to that question.

**Decision**:

```go
type Identity string

const (
    IdentityAvailable Identity = ""        // the caller can act as the deleting identity
    IdentityMissing   Identity = "missing" // that identity does not exist (the operator's SA NotFound)
    IdentityFailed    Identity = "failed"  // obtaining it failed for another reason
)

type HoldInput struct{ Identity Identity }

type HoldReason string // the contract's literals

type HoldVerdict struct {
    Release bool
    Because HoldReason
    Message string
}

func MayReleaseHold(plan DeletionPlan, state State, in HoldInput) HoldVerdict
```

`MayReleaseHold` checks these in order and stops at the first match:

1. `!plan.Policy.Prune` releases with `prune-disabled`.
2. A plan with no steps releases with `inventory-empty`.
3. `IdentityMissing` with `plan.Policy.ForceOrphan` releases with `force-orphan`.
4. `IdentityMissing` or `IdentityFailed` holds with `identity-unavailable`.
5. A plan that is not finished (`Next < len(Steps)` or the state awaits something) holds with
   `cleanup-incomplete`.
6. Any `forbidden` outcome holds with `cleanup-forbidden`.
7. Any other `failed` outcome holds with `cleanup-incomplete`.
8. Otherwise it releases with `cleanup-complete`. Skipped steps count as complete.

Force-orphan releases only when the identity is missing, as the operator does today. It does not
lift a Forbidden or incomplete cleanup. Widening it would change operator behaviour that no
decision covers.

A frontend without impersonation passes `IdentityAvailable`. A prune never calls `MayReleaseHold`.

**Rationale**: each operator branch is one case, in the operator's own precedence. Forbidden wins
over other failures, as the operator's `isForbidden` check on the joined error does. The inputs are
the policy, the outcome and whether the caller could act as the deleting identity. None of them
names a finalizer or a frontend.

### LC6: Foreground propagation in the plan

**Context**: the contract's `#PlannedAction` names no propagation policy, and the frontends differ.

**Decision**: the `delete` action carries `metav1.DeletePropagationForeground`. That is the cli's
behaviour today, and the operator adopts it in op-f2 with a release note. This is the per-frontend
divergence 0012:D1:R1 exists to remove. Making propagation a policy field would let it diverge
again.

**Rationale**: with Foreground, a dependent the object owns (a ReplicaSet under a Deployment) is
gone before the owner's DELETE completes. A frontend that waits for absence therefore sees the
whole subtree go.

### LC7: The package is pure and has no hook vocabulary

**Decision**: `opm/k8s/lifecycle` imports only:

- the standard library, without `os`, `time`, `log`, `log/slog` or `sync`;
- `k8s.io/apimachinery` (`api/errors`, `apis/meta/v1`, `apis/meta/v1/unstructured`,
  `runtime/schema`);
- `opm/k8s/inventory`, `opm/k8s/ownership` and `opm/k8s/object`.

A test reads the package's direct imports and fails on `os`, `time`, `log`, `log/slog` and `sync`.
Another test parses the package's non-test files and fails on any `go` statement, which pins "no
goroutines". The action kinds are exactly the four above, and every `read`, `delete` and `skip`
names a plan step (0012:D4:R6). No new depguard rule is added. The tier's allow list already holds
the package to the standard library, apimachinery and `opm/`, and the import test covers the
rest.

The package doc publishes into the Library reference, so it carries no ADR or enhancement pointer.
The maintainer pointers (ADR-008 rules 1 to 3, ADR-011 item 6, 0012:D4) go in a non-doc comment
after the package clause, as in `opm/k8s/ownership/doc.go`.

## Risks / Trade-offs

- **The operator's behaviour changes on adoption.** The delete order becomes descending weight, and
  propagation becomes Foreground. A Foreground delete of an object with many dependents returns
  before they are gone, and the object lingers with a `foregroundDeletion` finalizer. On its next
  pass the operator reads it as present, OPM-managed and of this instance, and `CanDelete` proceeds
  ("An object being deleted proceeds"), so the plan names `delete` again. Deleting it again is
  harmless. op-f2 owns the release note.
- **Rebuilding the plan on every pass gives up cross-pass memory.** A step that failed on one pass
  is read again on the next. This is the operator's behaviour today, and it is what level
  triggering wants.
- **The JSON state is a public encoding.** A later change to it is a SemVer event. No frontend
  stores it today, which keeps the cost of that low.
- **`Advance` returns an error.** A pure step function with an error return is slightly more API
  than ADR-008's sketch. The alternative is to silently accept an answer about a different object,
  which could delete an unjudged object.
