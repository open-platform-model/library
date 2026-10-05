## Context

See proposal.md, Why. Design-local decisions are numbered OW1 to OW8 so they collide with no
other numbering. Line references are at library `origin/main` `ca7c56b` (after lib-e2e5), cli
`origin/main` `bd4d1a7c` and opm-operator `origin/main` as fetched on 2026-10-05. The evidence is
the wave-2 research entries e4 and w1-04
(`claude-stuff/kernel-plan-beta1/wave2-plan-result.json`), re-checked at those heads. Two things
changed since the research:

- `opm/k8s/labels` and `opm/k8s/object` exist (library#196), so the new package builds on them.
- The cli's operator migration has merged (cli `internal/operator/migration_plan.go`,
  archived change `2026-10-04-migrate-manifest-installed-operator`). `PreApplyExistenceCheck`
  now takes an `AdmitSet`, and `MigrationPlan.Admit()` fills it with the proven earlier-manifest
  objects, the Deployment the migration recreates, and the rendered objects that already carry
  the instance's UUID. The library's `Admit` input is shaped to take exactly that set (OW4).

The contract is enhancement 0012 as merged in enhancements#88 and amended on 2026-10-04:
0012:D8:R1 to R7, 0012:D1:R4/R7, 0012:D4:R1/R2, and the sketch in
`0012/contracts/contracts.cue` (`#SkipReason`, `#ApplyRefusalReason`, `#safetyExcluded`).

## Goals / Non-Goals

**Goals:**
- Fix the adopt annotation key.
- One pure delete verdict with the contract's four skip reasons, carrying what a DELETE
  precondition needs.
- One pure apply verdict with the contract's three refusal reasons and the adopt override.
- The operator install's admission (0012:D8:R6/R7) as an input both verdicts honour, narrowly.
- Messages worded once, in the library.

**Non-Goals:**
- Any frontend edit, any cluster read, any read-error policy.
- The deletion plan, its order and the hold (lib-f2).
- An adapter from `opm/k8s/inventory.Entry` (lib-e3 is in flight).
- Recording the key in enhancement 0012 (an enhancements change).

## Research & Decisions

### OW1: The adopt key is `opmodel.dev/adopt`, declared in `opm/k8s/labels`

**Context**: 0012:D8 fixes the key in the implementing change: "indicatively
`opmodel.dev/adopt`", and the owner's e4 answer uses the same example.
**Explored**: The OPM keys at HEAD fall into two families (`opm/k8s/labels/labels.go`).
Identity labels use a subject-prefixed domain (`module-instance.opmodel.dev/uuid`,
`component.opmodel.dev/name`), and the cli's category label uses the bare domain
(`opmodel.dev/component`). The operator's one annotation, `opm.dev/force-delete-orphan`
(opm-operator `api/v1alpha1/common_types.go`), uses a domain OPM does not own. A
`module-instance.opmodel.dev/adopt` spelling would read as an identity label, which the guard
treats as the object's owner. The adopt annotation is the opposite: an instruction a user writes
to hand the object over.
**Decision**: `opmodel.dev/adopt`, the owner's example. It is the constant
`labels.AnnotationAdopt`, an annotation key, not a label. Its value is the adopting instance's
UUID, the value of `module-instance.opmodel.dev/uuid` on that instance's rendered objects. The
`Annotation` prefix keeps it from reading as a label in a package of label keys.
**Rationale**: The code shows no better fit. The bare domain separates the user's instruction
from the identity labels the guard reads, and the operator's `opm.dev` domain is not one to copy.
Keeping it in `labels` keeps one dependency-free vocabulary package, so a frontend can print the
key without importing apimachinery.

### OW2: Plain inputs, the frontend reads the live object

**Context**: The verdicts must be pure (ADR-008, ADR-011), and the operator reads through an
impersonated, uncached client while the cli reads through a dynamic client.
**Explored**: `inventory.Entry` (lib-e3) is not merged and is written in parallel with this
change. `object.Identity` carries `APIVersion` rather than the group, and it is the duplicate
check's identity, so overloading it couples two decisions.
**Decision**:

```go
package ownership

// Object is the identity of the object being judged: what a frontend
// reads from an inventory entry or a rendered object.
type Object struct{ Group, Kind, Namespace, Name string }

type DeleteInput struct {
    Object       Object
    Live         *unstructured.Unstructured // nil: the read found nothing
    InstanceUUID string                     // the deleting instance; "" disables the comparison
    Admit        bool                       // proven earlier-manifest object (OW4)
}

type ApplyInput struct {
    Object       Object
    Live         *unstructured.Unstructured // nil: the object does not exist
    InInventory  bool                       // in the instance's recorded inventory
    InstanceUUID string                     // the applying instance, from the render
    Admit        bool                       // proven earlier-manifest object (OW4)
}

func SafetyExcluded(group, kind string) bool
func CanDelete(in DeleteInput) DeleteVerdict
func CanApply(in ApplyInput) ApplyVerdict
```

A nil `Live` means the frontend read the object and got NotFound. A read error is not an input;
what a frontend does with one is its own policy, decided in its adoption change. Neither
function changes `Live`.
**Rationale**: Four plain fields map from either frontend's entry type and from lib-e3's later
`Entry` with no import between sibling packages in flight. `SafetyExcluded` is exported so a
frontend can skip the live read for a protected kind, as the cli's `recordUnreadable` does today.

### OW3: The apply verdict, in order

**Context**: 0012:D8:R1/R2/R5 and 0012:D1:R7. An object that does not exist is never refused.
**Decision**: `CanApply` decides in this order and returns at the first match:

1. `Live == nil`: apply.
2. `Live` has a `deletionTimestamp`: refuse `terminating`. Nothing below lifts it, in the
   inventory or not, adopted or admitted.
3. `InInventory`: apply. The ownership refusals cover only objects outside the inventory.
4. The adopt annotation's value equals `InstanceUUID` and `InstanceUUID` is not empty: apply.
5. The managed-by value is not an OPM runtime's (`labels.IsOPMManagedBy`): refuse
   `foreign-object`, unless `Admit` holds and the object carries no other instance's UUID (OW4).
6. The live UUID label is not empty and differs from `InstanceUUID`: refuse `other-instance`.
7. Otherwise apply.

```go
type ApplyRefusal string // "terminating" | "foreign-object" | "other-instance"

type ApplyVerdict struct {
    Refuse  ApplyRefusal // "" = apply
    Message string       // set on refuse
}

func (v ApplyVerdict) Allowed() bool
```

Two identity edge cases, both decided toward the existing tolerance or toward refusal:

- An OPM-managed object outside the inventory with no UUID label is applied. It predates UUID
  stamping and carries no other instance's identity, which is what 0012:D8:R1 refuses. This is
  the tolerance both frontends' delete paths apply today.
- An empty `InstanceUUID` cannot be adopted into (step 4 never matches), and any non-empty live
  UUID counts as another instance's (step 6). Every rendered object carries the instance UUID
  (core stamps `module-instance.opmodel.dev/uuid`), so a frontend always has one to pass. An
  empty value is a caller defect, and refusing is the safe direction.

**Rationale**: Terminating before inventory makes 0012:D8:R5 hold for every object. Adoption
before the two ownership tests means one annotation lifts both, as 0012:D8:R2 says, and the
annotation is read only when the guard would otherwise refuse.

### OW4: The install admission is narrower than adoption

**Context**: 0012:D8:R6 admits, "as if adopted", exactly the objects the operator install proves
an earlier operator release's install manifest created. A proven object "carries no OPM instance
identity". 0012:D8:R7 lets install delete the proven earlier Deployment and superseded role
bindings outside the inventory, and never a CustomResourceDefinition, a Namespace or a custom
resource. The cli's `MigrationPlan.Admit()` admits proven objects, the Deployment it recreates,
and rendered objects that already carry the instance's own UUID.
**Explored**: A first draft let `Admit` lift both ownership refusals, like the annotation.
0012:D8:R6 is narrower: a proven object carries no instance identity, so it could never meet
`other-instance`.
**Decision**: `Admit` lifts only `foreign-object` on apply and only `not-opm-managed` on delete,
and only when the live object carries no UUID label or carries `InstanceUUID`. It never lifts
`terminating`, `other-instance`, `owner-mismatch`, `safety-excluded` or `already-absent`. The
caller sets `Admit` only for an object it has proven. The library cannot check the proof, which
needs the earlier manifests. Whether an object is a custom resource is also the caller's to
exclude, since the verdict sees only group and kind.
**Rationale**: This is the reading that best matches the owner's 2026-10-04 decision. An
admitted object whose UUID label names another instance is not a proven object, and the library
refuses it even if a caller's proof has a hole. Every object in the cli's set passes as before:
proven objects carry no identity, and `Ours` objects carry the instance's own.

### OW5: The delete verdict carries the judged UID and resourceVersion; the precondition is UID only

**Context**: The cli's `checkDeletable` and the operator's `Prune` re-read and then DELETE with no
precondition (cli `internal/kubernetes/delete.go` `deleteResource`, opm-operator
`internal/apply/prune.go` `c.Delete(ctx, live)`).
**Decision**: `CanDelete` decides in this order: `SafetyExcluded` (no live object needed) →
`already-absent` when `Live` is nil → `not-opm-managed` (unless admitted, OW4) →
`owner-mismatch` when both UUIDs are non-empty and differ → proceed. An object being deleted
proceeds, since deleting it again changes nothing. A proceed verdict carries `UID` and
`ResourceVersion` from `Live`.

```go
type SkipReason string // "safety-excluded" | "already-absent" | "not-opm-managed" | "owner-mismatch"

type DeleteVerdict struct {
    Skip            SkipReason // "" = proceed
    Message         string     // set on skip
    UID             types.UID  // set on proceed
    ResourceVersion string     // set on proceed
}

func (v DeleteVerdict) Proceed() bool
func (v DeleteVerdict) Preconditions() metav1.Preconditions // UID only
```

`Preconditions` returns `{UID: &v.UID}` and no resourceVersion. A frontend that wants the strict
form sets `ResourceVersion` itself from the verdict.
**Rationale**: The UID precondition closes the real hazard, an object deleted and recreated
under the same name between the read and the DELETE. A resourceVersion precondition also fails
whenever any writer touches the object in that window. A controller's status update is one, and
foreground propagation adds finalizer writes, so a strict default would turn ordinary churn into
spurious conflicts. The frontend decides how a failed precondition is reported. It must never
count one as a deletion.

### OW6: The library words every message

**Context**: 0012:D1:R7 says the verdict is the same on both frontends, so its wording should be
too. A user can adopt only with the key and the instance UUID in hand, and a ModuleInstance's
UUID is known only after render. 0012:D8:R3 says no message names another override.
**Decision**: Every refusal and every skip carries `Message`. The object is written
`Kind/namespace/name`, or `Kind/name` when cluster-scoped, as the cli's install lines write it.
The ownership refusals end with the remedy, which names the key and this instance's UUID, for
example:

```text
Deployment/web/api exists and is not managed by OPM; to let this instance take it over, annotate it opmodel.dev/adopt=<uuid>
Deployment/web/api belongs to module instance <other-uuid>; to move it to this instance, annotate it opmodel.dev/adopt=<uuid>
Deployment/web/api is being deleted; wait for the deletion to finish, then apply again
```

When the live object carries the annotation with another value, the message says so. With an
empty `InstanceUUID` it gives no remedy. Messages carry no enhancement reference, since they
reach CLI output, and name no flag.
**Rationale**: One wording keeps the two frontends from drifting. Printing the UUID answers the
research risk that a user cannot adopt without knowing it.

### OW7: `SafetyExcluded` is its own predicate

**Context**: `opm/k8s/object` has an unexported `isClusterDefinition` with the same set (core
Namespace, `apiextensions.k8s.io` CustomResourceDefinition), which it uses for apply stages.
**Decision**: `ownership.SafetyExcluded` is a separate exported function. `object` keeps its own.
**Rationale**: Two decisions coincide today. "Applied in the first stage" and "never deleted by
OPM" could part (a future cluster-scoped kind may need staging without protection), and exporting
one from `object` for the other would bind them. The set is two lines, and each package's tests
pin it.

### OW8: Reasons are the contract's literals

**Decision**: `SkipReason` and `ApplyRefusal` are string types whose constants equal the
contract's literals (`safety-excluded`, `already-absent`, `not-opm-managed`, `owner-mismatch`;
`terminating`, `foreign-object`, `other-instance`). The empty value means proceed or apply, so
the zero verdict is the permissive one only when the function returned it.
**Rationale**: A frontend can put a reason in a status condition or a log key as it stands, and
both frontends then report the same token.

## Risks / Trade-offs

- [The operator's guard needs a live read per object outside the inventory, through the
  impersonated client] → Out of scope here. The verdict takes the read's result, and op-e4 owns
  the read and the error policy.
- [Fail-closed on an empty `InstanceUUID` could refuse an apply a frontend expected to pass] →
  Every render stamps the UUID, so this fires only on a caller defect. The tests pin it.
- [lib-e3 and lib-f5 edit the same doc lines] → Merge `origin/main` into the branch at PR time.
  The edits are list items.
