## 1. Package scaffold and neutral type

- [x] 1.1 Create `opm/inventory/` with a `doc.go` stating the package is the runtime-neutral, shared inventory contract (kernel core, not under `opm/helper/`) consumed by every OPM frontend; note the no-controller-runtime/no-Flux constraint (design D1).
- [x] 1.2 Define `InventoryEntry struct { Group, Kind, Namespace, Name, Version, Component string }` in `entry.go`, with field doc comments; no methods yet.
- [x] 1.3 Add a compile/import guard test (e.g. a small test asserting via `go list`-style check or a build-tag-free import audit) that the package's import graph excludes `controller-runtime`, `fluxcd/*`, and `apiextensions-apiserver` (spec: Runtime-Neutral Inventory Entry Type).

## 2. Entry construction and identity relations

- [x] 2.1 Implement `NewEntryFromResource(u *unstructured.Unstructured) InventoryEntry`, ported from the operator's `internal/inventory/entry.go`, reading GVK/namespace/name and the OPM component label; pure, no I/O.
- [x] 2.2 Implement `IdentityEqual(a, b InventoryEntry) bool` (component-aware) and `K8sIdentityEqual(a, b InventoryEntry) bool` (component-agnostic) in `entry.go`.
- [x] 2.3 Table tests for 2.1–2.2: construction from a representative unstructured object; same-object-different-component (K8s equal, full not equal); different-object (both not equal). (spec: Entry Identity Relations.)

## 3. Digest and canonical stale set

- [x] 3.1 Implement `ComputeDigest(entries []InventoryEntry) string` in `digest.go`: sort by identity tuple, canonical JSON encode, SHA-256; ported from the operator's reference form (design D4).
- [x] 3.2 Implement `ComputeStaleSet(previous, current []InventoryEntry) []InventoryEntry` in `stale.go` using `K8sIdentityEqual` as the base relation (design D2); document that component moves are NOT stale here and are handled by the rename-safety check.
- [x] 3.3 Tests: digest order-independence and content-sensitivity; stale-set removed-vs-retained including the retained-but-component-changed case staying non-stale. (specs: Deterministic Content Digest, Canonical Stale-Set Computation.)

## 4. Pure prune-safety

- [x] 4.1 Implement `ApplyComponentRenameSafetyCheck(stale, current []InventoryEntry) []InventoryEntry` in `stale.go`: drop stale entries that `K8sIdentityEqual` a current entry under a different `Component`; pure. (spec: Component-Rename Prune Safety.)
- [x] 4.2 Define the neutral observed-live-state input type (fields: exists, being-deleted, managed-by attribution — derived from what the CLI's current `PreApplyExistenceCheck` inspects, no speculative fields) and implement the pure collision predicate in `existence.go`: given a candidate entry + observed state, return whether applying collides with a foreign-owned object; no cluster read, no logging (design D3).
- [x] 4.3 Tests: rename-move removed from stale vs genuinely-removed still stale; collision predicate over foreign-owned / absent / self-owned observed states; assert the predicate touches only its arguments. (specs: Component-Rename Prune Safety, Pure Pre-Apply Collision Predicate.)

## 5. Validation gates

- [x] 5.1 `task fmt` — gofmt + goimports clean; imports ordered per the constitution (stdlib, external, local).
- [x] 5.2 `task vet` and `task lint` — pass.
- [x] 5.3 `task test` — all `opm/inventory` tests pass; confirm no new module dependency entered `go.mod`/`go.sum` (apimachinery + stdlib only).
