## Why

OPM has two independent copies of "what resources does this release own" logic — one in the operator (`opm-operator/internal/inventory`) and one in the CLI (`cli/pkg/inventory`) — and they have already drifted: the operator computes its stale set with component-agnostic identity (`K8sIdentityEqual`) while the CLI uses component-aware identity (`IdentityEqual`), so the two compute *different* prune sets for the same release today. Enhancement [0006](../../../../enhancements/0006/) makes the CLI and operator share one render path (the kernel) and one inventory store (the `ModuleInstance` CR), and its zero-downtime handoff (D7.4) is only safe if both actors also compute entry identity, digests, and the stale/prune set with the *same code*. That shared implementation has to live somewhere neutral both can import without inheriting the other's runtime. This change creates that home in the kernel library. It is slice **A3** of 0006 and the foundation both the operator (B1) and CLI (C1) consume.

## What Changes

- **New `opm/inventory` package** — the runtime-neutral, pure inventory logic both the CLI and operator will consume, over a **new library-native `InventoryEntry` type** (`Group`, `Kind`, `Namespace`, `Name`, `Version`, `Component`). This type is deliberately *not* the operator's `api/v1alpha1.InventoryEntry`: that type lives in a Kubebuilder API package that transitively drags `controller-runtime`, Flux, and `apiextensions-apiserver` into any importer. Consumers map their own entry shapes to/from this neutral type at their boundary (enhancement 0006 D13).
- **Entry construction + identity** — `NewEntryFromResource` (build an entry from an `unstructured.Unstructured`), `IdentityEqual` (full identity, component-aware), and `K8sIdentityEqual` (Kubernetes object identity, component-agnostic), ported from the operator's implementation.
- **Digest + stale set** — `ComputeDigest` (deterministic content digest over a sorted entry set) and `ComputeStaleSet` (entries present in the previous inventory but absent from the current render). **This change reconciles the existing CLI/operator drift** by defining one canonical stale-set semantics: the base stale set is computed on Kubernetes object identity (`K8sIdentityEqual`), with component moves handled explicitly by the rename-safety function below rather than by silently treating a moved resource as stale-and-recreate.
- **Prune-safety logic (0006 D26)** — `ApplyComponentRenameSafetyCheck` (pure: drop from the stale set any entry that is the same Kubernetes object as a current entry under a different component, so a component rename does not delete-and-recreate live resources) and a pre-apply existence **predicate** that, given the already-fetched state of a candidate object, decides whether applying it would collide with a foreign-owned resource. Per the kernel-neutrality constitution (I/O at the edges, no direct logging), the predicate is pure: the *fetching* of live cluster state and any logging stay at the caller's edge (CLI/operator), only the *decision* is shared.
- **No controller-runtime, no Flux.** The package depends only on the standard library and `k8s.io/apimachinery` (`unstructured`, identity primitives) — verified against the operator's current imports, which already touch none of those frameworks.

This is **MINOR** (SemVer): a new additive package under `opm/`, no change to any existing `opm/` type, signature, or behavior. Nothing in the library imports it yet.

## Capabilities

### New Capabilities

- `inventory`: the kernel's shared inventory contract — the neutral `InventoryEntry` type, entry construction from rendered resources, the two identity relations, the content digest, the stale-set computation with its canonical component-move semantics, and the pure prune-safety functions (rename safety + pre-apply collision predicate). Defines the behavior both the operator and CLI must compute identically for handoff prune-set parity to hold.

### Modified Capabilities

<!-- None. This change is purely additive; no existing library capability's requirements change. The operator (0006 B1) and CLI (0006 C1) adopt this package in their own repos' slices. -->

## Impact

- **New package:** `library/opm/inventory/` (Go), plus its unit tests. New public SemVer surface (MINOR).
- **Dependencies:** adds no new module dependencies — uses `k8s.io/apimachinery` (already in `go.mod` via the kernel's existing usage) and the standard library only. No `controller-runtime`, no `fluxcd/*`.
- **Downstream (out of scope here, tracked by 0006):** the operator's `internal/inventory` migrates to consume this package (0006 slice B1), mapping `api/v1alpha1.InventoryEntry` ↔ the neutral type; the CLI deletes `cli/pkg/inventory` and consumes this package (0006 slice C1). Both gain the reconciled stale-set semantics — for the operator that means it newly runs the explicit component-rename safety check, which B1 covers with operator-side tests.
- **Parity contract:** this package is the single source of entry identity, digest, and stale-set computation. The handoff render-digest parity experiment (0006 D30) and the operator's prune-set parity both depend on this being the only implementation.
