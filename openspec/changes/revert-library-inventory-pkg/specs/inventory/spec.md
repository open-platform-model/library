# inventory (REMOVED)

The `opm/inventory` package is removed from the library. It was introduced to home entry-identity, digest, stale-set, and prune-safety logic shared by the CLI and the operator (enhancement `0006` slice A3); decision D31 (`0006/03-decisions.md`) found only the `InventoryEntry` wire shape is actually cross-actor-critical, and that shape is already anchored by the `ModuleInstance` CRD schema plus the kernel's render-digest parity — not by sharing this package. Every requirement in this capability is removed. The behavior each requirement described continues to exist, duplicated per-actor, in `cli/pkg/inventory` and `opm-operator/internal/inventory` — those are not governed by a library-level spec.

## REMOVED Requirements

### Requirement: Runtime-Neutral Inventory Entry Type

**Reason**: No cross-actor consumer ever adopted this package (`opm-operator/go.mod` remained pinned to `v1.0.0-alpha.3`, predating it; `cli` never added the dependency). D31 found the `InventoryEntry` wire shape is the only genuinely shared surface, and it is already carried by the `ModuleInstance` CRD schema — a package-level shared Go type adds a coordination cost without adding safety beyond what the CRD schema already provides.
**Migration**: None. There is no known importer. Any future need for this shape is met by the CRD's `status.inventory` field, not a shared Go package.

### Requirement: Entry Identity Relations

**Reason**: `IdentityEqual` / `K8sIdentityEqual` are per-actor policy — D31 found the CLI's and the operator's stale-set logic already diverge on exactly this relation (component-aware vs component-agnostic identity) despite never having shared code, evidence that a shared implementation was not the thing keeping them aligned.
**Migration**: None. Each actor keeps its own identity-equality logic (`cli/pkg/inventory`, `opm-operator/internal/inventory`).

### Requirement: Deterministic Content Digest

**Reason**: `ComputeDigest` is local per-actor policy, never cross-compared between the CLI and the operator; the moment digest parity actually matters (handoff) is already gated by the kernel render-digest check (D9/D7.4), independent of whether the two actors share this function.
**Migration**: None. Each actor computes its own digest.

### Requirement: Canonical Stale-Set Computation

**Reason**: `ComputeStaleSet` is per-actor policy. D31 recorded that the CLI's and operator's stale-set implementations already diverge (component-aware `IdentityEqual` vs component-agnostic `K8sIdentityEqual`, open as OQ15 in `enhancements/0006`) — sharing this package was not preventing that divergence.
**Migration**: None. Each actor keeps its own stale-set computation.

### Requirement: Component-Rename Prune Safety

**Reason**: `ApplyComponentRenameSafetyCheck` is per-actor prune policy, superseded in full by D31 alongside D26 (the decision that originally placed it here).
**Migration**: None. If a component-rename safety check is needed by an actor, it lives in that actor's own inventory package.

### Requirement: Pure Pre-Apply Collision Predicate

**Reason**: The collision predicate was never adopted outside this package, and D31's investigation found the operator's apply path has no equivalent guard at all (open as OQ16 in `enhancements/0006`) — whether and how to build one is an operator-local design question, not a shared-package one.
**Migration**: None. The CLI keeps its existing `PreApplyExistenceCheck`; whether the operator gains an equivalent is tracked by OQ16, independently of this package.
