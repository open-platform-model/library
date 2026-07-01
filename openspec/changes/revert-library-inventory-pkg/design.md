## Context

`opm/inventory` shipped in library PR #34 (`27acbfa`, released as `1.0.0-alpha.4`) as the intended single implementation of entry identity, digest, stale-set, and prune-safety logic for both the CLI and the operator (enhancement `0006` slice A3, decision D13). Enhancement `0006` decision D31 (2026-07-01) reverses the premise: an explore-mode trace of which functions are actually compared across actors found only the `InventoryEntry` wire shape qualifies, and that shape is already carried by the `ModuleInstance` CRD schema plus the kernel's render-digest parity (D9) — not by the two actors sharing a Go package. `ComputeStaleSet`, `ComputeDigest`, `ApplyComponentRenameSafetyCheck`, and the collision predicate are each per-actor policy, never cross-compared, and (per D31's own finding) the CLI's and operator's stale-set logic had already diverged (`IdentityEqual` vs `K8sIdentityEqual`) despite sharing no code yet — evidence the "shared implementation" framing wasn't earning its coordination cost.

No downstream repo consumes the package: `opm-operator/go.mod` is still pinned to `library v1.0.0-alpha.3` (predates the package), and `cli` never added the `library` edge for it. The operator-adoption slice (B1) is cancelled and the CLI slice (C1) dropped the dependency. This design covers the mechanics of removing the package cleanly from a repo where it is, in practice, dead code with no live importer — but which is nonetheless a public API surface that shipped in a tagged release.

## Goals / Non-Goals

**Goals:**
- Remove `opm/inventory` and its sole dependency (`k8s.io/apimachinery`) from the library's public surface and module graph, leaving no dangling references in code, specs, or `go.mod`/`go.sum`.
- Preserve the historical record: the original slice's archived OpenSpec change and PR stay untouched, and this removal is itself recorded (in `MIGRATIONS.md` and via a new archived OpenSpec change) rather than silently erased from history.
- Keep the change small and single-purpose per Principle VIII — this is a deletion plus dependency cleanup plus documentation, nothing else.

**Non-Goals:**
- Re-deciding whether shared inventory logic is the right design — that decision (D31) was already made in `enhancements/0006`; this change only executes it.
- Touching `opm-operator/internal/inventory` or `cli/pkg/inventory` — neither ever depended on this package, so there is nothing to migrate on either side.
- Rewriting `openspec/changes/archive/2026-06-30-library-inventory-pkg/` — archives are immutable historical record, not live documentation.

## Decisions

**Delete the package outright rather than deprecate-then-remove.** With zero known importers and the package only one release old, there is no compatibility window worth preserving (mirrors the precedent in `openspec/changes/archive/2026-05-31-remove-library-catalog/`, which deleted rather than deprecated a similarly young, low-adoption surface). Alternative considered: mark the package `Deprecated:` for a release cycle first — rejected, since deprecating something with no consumers only delays a decision already made in D31.

**Record it as a breaking change in `MIGRATIONS.md` despite no known importer.** Principle VI defines MAJOR as "any breaking change to `opm/` types, signatures, or behavior" — it does not carve out an exception for unused symbols, and the release CI gate checks for a matching `MIGRATIONS.md` entry on any breaking commit in range. Treating this as breaking is also just honest: a hypothetical external importer (there is provably none in this workspace, but the library has no way to enforce that globally) would see a compile break.

**Let `task tidy` compute the `go.mod`/`go.sum` diff rather than hand-editing.** The original PR #34 diff added `k8s.io/apimachinery` plus ~15 transitive lines; hand-reverting risks leaving a stale indirect entry `go mod tidy` would otherwise catch (e.g. a `google.golang.org/protobuf` version bump that may or may not still be required by something else). Verified beforehand: no other file in the repo imports `k8s.io/apimachinery`, so `task tidy` after the source deletion is expected to reproduce the pre-PR-#34 dependency set exactly.

## Risks / Trade-offs

- **[Risk]** A future consumer could have started depending on `1.0.0-alpha.4`'s `opm/inventory` between its release and this revert landing, outside this workspace's visibility. → **Mitigation**: none possible from the library side beyond the standard channel (breaking-change entry in `MIGRATIONS.md` + CHANGELOG `Reverts` section from the `revert:` commit type); this is the accepted cost of shipping alpha releases, and D31 already made the call that the coordination cost of keeping the package exceeds this residual risk.
- **[Trade-off]** The `inventory` capability spec's requirement text (entry identity, digest, stale-set, prune-safety, collision predicate) is retired rather than reused, even though the *behavior* it describes still exists — just duplicated per-actor in `cli/pkg/inventory` and `opm-operator/internal/inventory`. → Accepted: per D31, per-actor specs for that duplicated behavior belong in `cli/` and `opm-operator/` respectively (already the status quo), not in a library-level spec for a package that no longer exists here.
