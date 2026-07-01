## Why

Enhancement `0006` (workspace-root `enhancements/0006`, decision D31, 2026-07-01) reverses the design behind slice A3: the shared `opm/inventory` package added to home entry-identity, digest, stale-set, and prune-safety logic for both the CLI and the operator. Tracing which of those functions are actually cross-actor-critical showed only the `InventoryEntry` wire shape qualifies, and that surface is already anchored by the `ModuleInstance` CRD schema and the kernel's render-digest parity (D9) — not by sharing a Go package. `ComputeStaleSet`, `ComputeDigest`, `ApplyComponentRenameSafetyCheck`, and the collision predicate are each per-actor policy that is never cross-compared; the one moment a cross-actor comparison would matter (the operator's first post-handoff stale-set read of CLI-written entries) is already gated by the handoff digest check, independent of whether the two actors share an implementation.

The package shipped as library PR #34 (commit `27acbfa`, squash of `4558ed9`) and is live in the tagged `1.0.0-alpha.4` release, but it acquired zero real consumers before D31 landed: `opm-operator`'s `go.mod` still pins `library v1.0.0-alpha.3` (never bumped to adopt it), and `cli` never imported it. The operator-adoption slice (B1) is cancelled and the CLI slice (C1) dropped its dependency on this package. Removing it now, before either downstream repo consumes it, costs nothing and stops a coordination surface from persisting that the design no longer calls for.

## What Changes

- **BREAKING**: delete the `opm/inventory` package in full — `InventoryEntry`, `NewEntryFromResource`, `IdentityEqual`, `K8sIdentityEqual`, `ComputeStaleSet`, `ComputeDigest`, `ApplyComponentRenameSafetyCheck`, and the pre-apply collision predicate all leave the public `opm/` surface. This is a public Go API removal on a package that shipped in the tagged `1.0.0-alpha.4` release, even though no known importer exists today.
- Drop the now-unused `k8s.io/apimachinery` dependency (and its transitive closure) via `go mod tidy` — it was added solely for this package and is not referenced anywhere else in the library.
- Retire the `inventory` capability spec (`openspec/specs/inventory/spec.md`) entirely.
- Record the removal in `MIGRATIONS.md` under `## Unreleased — Breaking` so the release CI gate (which blocks a release on an unmatched breaking change) has an entry to match against.
- Out of scope: `openspec/changes/archive/2026-06-30-library-inventory-pkg/` (the original slice's archived record) is left untouched — it is historical record of what shipped and why, same as the enhancement's own append-only decision log.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None.

### Removed Capabilities

- `inventory`: the runtime-neutral shared inventory package (`opm/inventory`) — entry identity, content digest, stale-set, and pure prune-safety logic intended for both the CLI and the operator to consume. Superseded by D31: only the `InventoryEntry` wire shape is cross-actor-critical, and that is already carried by the CRD schema + kernel render-parity, not by package-sharing. Each actor keeps its own local policy for the rest.

## Impact

- **Go code:** deletes `library/opm/inventory/` (10 files: `entry.go`, `entry_test.go`, `digest.go`, `digest_test.go`, `stale.go`, `stale_test.go`, `existence.go`, `existence_test.go`, `imports_test.go`, `doc.go`). No other `opm/` package imports it.
- **Dependencies:** `go.mod`/`go.sum` lose `k8s.io/apimachinery` and its transitive closure (confirmed unused elsewhere via `task tidy`).
- **OpenSpec:** `openspec/specs/inventory/spec.md` is removed via this change's `REMOVED` delta spec.
- **Downstream:** none in practice — neither `cli` nor `opm-operator` ever imported this package (verified: `opm-operator/go.mod` still pins `library v1.0.0-alpha.3`, predating the package's introduction). Recorded as a breaking change per Principle VI regardless, because the public API surface changed on a tagged release.
- **Enhancement tracking:** `enhancements/0006/planned-changes.md` (A3 row) and `config.yaml` history get a follow-up update once this change lands, marking the code revert complete (separate from this change, owned by the `enhancements` repo).
