## Context

Inventory logic — "which Kubernetes objects does this release own, and which of them are now stale" — exists twice in OPM today:

- `opm-operator/internal/inventory/` (`entry.go`, `digest.go`, `stale.go`): operates directly on the CRD type `api/v1alpha1.InventoryEntry`. Imports only stdlib, `k8s.io/apimachinery/.../unstructured`, the operator's `api/v1alpha1`, and `pkg/core` — **no controller-runtime, no Flux**.
- `cli/pkg/inventory/` and `cli/internal/inventory/`: a near-identical copy over the CLI's own `InventoryEntry` struct, plus CLI-only prune-safety logic (`ApplyComponentRenameSafetyCheck`, `PreApplyExistenceCheck`) and CLI-only I/O (`PruneStaleResources`, Secret CRUD) that depend on `cli/internal/kubernetes` and `cli/internal/output`.

The two copies have **drifted**: the operator's `ComputeStaleSet` compares entries with `K8sIdentityEqual` (group/kind/namespace/name only — component-agnostic), the CLI's with `IdentityEqual` (also compares `Component`). For a release where a resource moves between components, the CLI marks the old entry stale (delete-and-recreate) while the operator does not. Enhancement [0006](../../../../enhancements/0006/) unifies the CLI and operator onto one render path and one CR-backed inventory store; its handoff (D7.4) computes the operator's first post-takeover prune set as `previous status.inventory − current render`, and that is only a guaranteed no-op if both actors compute entry identity, digests, and the stale set with identical code (0006 D13). A single shared implementation in a neutral location is therefore a correctness requirement, not just deduplication.

This change creates that implementation as a new kernel-library package. It is slice **A3** of 0006; the operator (B1) and CLI (C1) adopt it in their own repos.

## Goals / Non-Goals

**Goals:**

- A new `opm/inventory` package that is the single source of inventory entry identity, content digest, and stale-set computation for every OPM frontend.
- A library-native `InventoryEntry` value type that carries no Kubernetes-framework baggage, so importing it never drags `controller-runtime` or Flux into a consumer.
- One canonical, documented stale-set semantics that ends the CLI/operator drift.
- The parity-critical prune-safety logic (component-rename safety; pre-apply collision decision) homed here as **pure** functions, with all I/O and logging left to the caller per the kernel-neutrality constitution.
- Zero new module dependencies; `apimachinery` + stdlib only.

**Non-Goals:**

- Migrating the operator or CLI onto this package — those are 0006 slices B1 and C1, in their own repos. This change ships the package and its tests; nothing in `library` imports it yet.
- Any cluster I/O: fetching live objects, applying, pruning, or deleting. Those stay at the CLI/operator edges.
- The CR serialization shape. `api/v1alpha1.InventoryEntry` remains the operator's CRD wire type; this package's type is the in-memory neutral type the two map to/from.
- Logging, output formatting, resource ordering (`resourceorder`), or Secret/CR persistence — all caller concerns.

## Decisions

### D1: A new library-native `InventoryEntry`, not the operator's CRD type

The package defines its own `InventoryEntry struct { Group, Kind, Namespace, Name, Version, Component string }` (field set taken verbatim from both existing copies, which already agree on it). It does **not** reuse `opm-operator/api/v1alpha1.InventoryEntry`.

*Why:* the operator's type lives in a Kubebuilder API package whose `groupversion_info.go` scheme builder transitively imports `controller-runtime/pkg/scheme`, `fluxcd/pkg/apis/meta`, and `apiextensions-apiserver`. Go compiles whole packages, so any importer of that type inherits the whole stack — exactly what 0006 D13 forbids for the one-shot CLI. A plain struct in `opm/inventory` has none of that. Consumers map at their boundary: the operator maps `api/v1alpha1.InventoryEntry ↔ inventory.InventoryEntry` (B1), the CLI maps its own struct (C1). *Alternative — put the shared type in the operator's `api/v1alpha1` and have both import it:* rejected, that is the dependency this design exists to avoid.

### D2: Canonical stale-set semantics = Kubernetes identity base + explicit component-move handling

`ComputeStaleSet(previous, current)` computes "present in previous, absent from current" using **`K8sIdentityEqual`** (the operator's component-agnostic relation) as the base comparison. Component moves are then handled **explicitly** by `ApplyComponentRenameSafetyCheck`, which removes from the stale set any entry whose Kubernetes object identity matches a current entry under a different component. The net effect: a resource that moves between components is *not* deleted and recreated — it is left in place and re-owned.

*Why:* this reconciles the drift in the safe direction. The CLI's component-aware base (`IdentityEqual`) treats a component move as stale → the resource is pruned then re-applied, a needless delete/recreate of a live object (and a brief outage). The operator's component-agnostic base avoids that but only because it never compared component at all. Making `K8sIdentityEqual` the base **and** adding the explicit rename-safety pass gives one relation both actors run, with the safe no-delete behavior stated as a requirement rather than emerging implicitly. Both `IdentityEqual` and `K8sIdentityEqual` are kept as exported relations (callers and tests need both), but the stale-set contract is pinned to this composition. *Alternative — pick one of the two existing behaviors wholesale:* rejected; component-aware-base reintroduces the outage, and component-agnostic-base-with-no-explicit-check leaves the rename behavior undocumented and untested.

### D3: Prune-safety is pure; I/O and logging stay at the caller edge

`ApplyComponentRenameSafetyCheck(stale, current []InventoryEntry) []InventoryEntry` is already pure and moves over unchanged in spirit. The pre-apply existence check is **split**: this package exposes a pure decision function that, given a candidate entry and the *already-fetched* state of the live object at that identity (its existence, deletion timestamp, and managed-by attribution, passed in as a small neutral input struct), returns whether applying would collide with a foreign-owned resource. The *fetching* of that live state (a cluster GET) and any operator/CLI logging remain in the consumer.

*Why:* the kernel-neutrality constitution is explicit — "I/O (filesystem, registry, network) MUST live at edges and accept caller-supplied configuration" and "Logging MUST come from the caller." The CLI's current `PreApplyExistenceCheck` does a live `client.Get` and writes to `internal/output`; lifting that verbatim would import a Kubernetes client and a logger into the kernel, violating the constitution and defeating D1's whole point. Splitting predicate from I/O keeps the parity-critical *decision* shared (both actors decide collisions identically) while the *mechanism* stays where the runtime lives. *Alternative — define a `K8sClient` interface in library and inject it:* rejected for this slice (Principle VII/YAGNI) — a pure predicate over caller-fetched state is simpler, has no interface to version, and is just as parity-safe; an injected-client abstraction can come later if a real consumer needs the library to orchestrate the fetch.

### D4: Deterministic digest, ported unchanged

`ComputeDigest(entries)` sorts entries by their identity tuple and hashes a canonical JSON encoding (SHA-256), matching the operator's existing implementation. The CLI's copy differs only in a best-effort error-fallback branch; the operator's form is the reference. Determinism is a constitution requirement (Principle I) and the basis of every digest-equality check in 0006.

## Risks / Trade-offs

- **Operator behavior change at adoption** → The operator currently has no explicit component-rename check; once B1 adopts this package the operator gains the no-delete-on-move behavior. That is the intended fix, but it changes operator prune output for the rename case. *Mitigation:* B1 carries operator-side tests for the rename scenario; the behavior is specified here (D2) so B1 is verifying against a written contract, not a guess.
- **Type-mapping boilerplate at two boundaries** → operator and CLI each write a small `to/from` mapping against the neutral type. *Mitigation:* the field set is identical across all three structs, so the mapping is mechanical and total; it is the price of severing the controller-runtime/Flux dependency, which D1 judges worth it.
- **Pre-apply predicate input shape is new** → splitting the existence check means defining a neutral "observed live object state" input the caller populates from its own GET. If that shape misses a field the decision needs, consumers diverge. *Mitigation:* derive the input fields directly from what the CLI's current `PreApplyExistenceCheck` actually inspects (existence, deletion timestamp, managed-by label/attribution) — no speculative fields — and keep it in one place so both consumers populate the same struct.
- **CUE/dependency surface** → none added; the package uses `apimachinery` already present in the kernel's `go.mod`. Low risk.

## Migration Plan

No runtime migration — additive new package, nothing imports it yet. Rollback is deleting the package; no consumer breaks because none has adopted it within this slice. Adoption and any data migration (Secret→CR, operator inventory rewire) live in 0006 slices B1/C1, sequenced after this lands.

## Open Questions

None blocking. The enhancement's open questions are all resolved (0006 D11–D30); the pre-apply predicate input shape (D3 risk) is settled by mirroring the CLI's current field usage rather than inventing new inputs.
