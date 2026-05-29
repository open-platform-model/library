## Context

The OPM core catalog lives at `library/modules/opm/` and is currently broken: it imports `opmodel.dev/core/v1alpha2@v1` and pins `opmodel.dev/core@v1.0.6` (unresolvable), so `cue vet ./...` fails. Its `cue.mod` was already partially renamed to `opmodel.dev/catalogs/opm@v1` while every internal import and metadata literal still uses the old `opmodel.dev/modules/opm` path and `version: "v1"`. The `opm_platform` consumer fixture is quarantined (`_platform.cue.quarantined`) because it predates the new shape.

Enhancement 0001's core slice shipped `opmodel.dev/core@v0.3.0` with: a SemVer-only `#FQNType` (`@\d+\.\d+\.\d+…`), a top-level `#Catalog`/`#CatalogFQNType` (D19, D25), and a `#Subscription`-shaped `#Platform.#registry`. This change is the `library`/`modules` slice — it brings the catalog and its consumer onto that shape and republishes. Per D22, Part B (binding-dispatch removal + OCI-loader) has shipped, so this can land now. The materialize/match kernel slice has also landed and consumes `#composedTransformers`/`#matchers` off the materialized platform — it is agnostic to the catalog's identity, so no Go code changes here.

Constraints: pure-CUE module work + Taskfile + CI; no `opm/` Go API surface touched (Principle VI: PATCH-equivalent for the Go library). Library Principle VIII (small batches) makes the three phases distinct commit groups.

## Goals / Non-Goals

**Goals:**

- Catalog vets and stamps correctly against `core@v0` under the `#Catalog` contract, with `identity/` as the single path/version source.
- A repeatable, guarded publish flow produces `opmodel.dev/catalogs/opm@v0.1.0`.
- The `opm_platform` fixture is restored against `#Subscription` and the published catalog, and un-quarantined.

**Non-Goals:**

- Conventional-commit/release-please-driven versioning or per-catalog changelogs (explicitly deferred — see Decisions).
- Rewiring workspace `modules/*` (jellyfin, garage, …) onto the new catalog tag (separate non-blocking wave per D23).
- Any `opm/` Go kernel change; any change to the Go-module release-please flow.
- Per-primitive SemVer or introspection sibling maps on `#Catalog` (rejected/deferred in 0001 D18/D19).

## Decisions

### D-A: `identity/` is the single source; all four primitive kinds source from it

Resources, traits, blueprints, **and** transformers derive `metadata.modulePath`/`version` from the `identity/` subpackage. Transformers must source it too (not rely solely on the `#Catalog` stamp) so their `metadata.fqn` is concrete *in the transformers package* — `catalog.cue` keys `#transformers` by `t.X.metadata.fqn`, which must resolve before the stamp applies. The stamp then re-asserts the identical value and unifies cleanly.

- *Why:* keeps FQNs concrete for map keys without a circular import; `identity/` (constants only) sits at the bottom of the import graph.
- *Alternative:* rely on stamping alone → map keys non-concrete at author time → fails. Rejected.

### D-B: `#Catalog` uses the `M=metadata` label-alias (inherited from 0001 D25)

`catalog.cue` embeds bare `c.#Catalog` (modules pattern, no `Catalog:` wrapper). The stamping bridge is the field-label alias the published core schema already uses.

- *Why:* matches the landed `core/src/catalog.cue` exactly; experiment 09 validated it; value-alias form fails `cue vet`.

### D-C: Standalone registry-presence publish, not release-please

A dedicated `cue:publish:catalog` task does the D9/D19 stamping (rsync → `.build/`, write `identity/version_override.cue`, vet, `cue mod publish vX.Y.Z`, reject `0.0.0-dev`). A catalog-only `publish-catalog.yml` runs on `main` with a **registry-presence (version-gated)** trigger: it reads the catalog version from `cue-versions.yml`, HEADs the GHCR manifest for that tag, and invokes the task only if the tag is absent. The trigger is **stateless** — nothing is written back to the repo. Releases are cut by **manually bumping** the `cue-versions.yml` version in a PR; on merge, the new (absent) tag publishes.

- *Why standalone, not RP:* the library repo already runs release-please for the **Go module** with **bare `vX.Y.Z`** tags, which `go get` requires. Adding the catalog as an RP component with `include-component-in-tag: true` would prefix *every* tag (`library-v…`), breaking the Go module's tag contract. Keeping the Go flow untouched and giving the catalog a standalone publisher avoids that collision entirely.
- *Why registry-presence over checksum content-diff:* the existing `cue:publish:smart` checksum approach must write the new checksum back to `cue-versions.yml`, which in an unattended `push: main` workflow means a bot commit-back that can re-trigger the run; it also auto-patch-bumps, mislabelling breaking primitive changes as patches. Registry-presence is stateless and keeps version intent (major/minor/patch) in the author's hands. It is the proven `publish-cue.yml` pattern.
- *Alternatives considered:* (1) **checksum content-diff + auto-bump** (`cue:publish:smart` style) — needs CI commit-back, dishonest semver; rejected. (2) **multi-component RP** with mixed per-package tag formats — uncertain RP supports bare-for-one + prefixed-for-another, higher risk to `go get`; deferred. (3) extract the catalog to its own repo (clean core-style RP) — contradicts 0001 D23 + graduation ("at `library/modules/opm/`"), biggest restructure; deferred. Revisitable if conventional-commit versioning becomes a real need.

### D-D: Version layering — `v`-prefixed OCI tag, bare FQN

The OCI module tag is `v0.1.0` (CUE *requires* the `v`: `cue mod publish 0.1.0` is rejected as invalid semver; a bare `@0` module path is rejected too — verified empirically). The schema-side `metadata.version`/FQN is bare `0.1.0` (core's `#FQNType` regex forbids the `v`). `identity.Version` holds the bare form; `cue-versions.yml` and `cue mod publish` use the `v`-prefixed form. The kernel bridges with a single `strings.TrimPrefix(ver, "v")` already present in `materialize.go`.

- *Why:* these are two different systems (CUE module versions vs OPM FQN), each with its own mandated convention. The mismatch is irreducible and the bridge is one line.
- *Alternative:* unify by adding `v` to `#FQNType` → a core schema break + release to make FQNs *less* SemVer-2.0-faithful and uglier, to delete one line. Rejected.

### D-E: Same-module imports drop the `@vN` qualifier

Self-imports within the catalog become `opmodel.dev/catalogs/opm/...` with **no** trailing `@vN`; cross-module imports keep theirs. Verified: under an `@v0` module, an unqualified self-import resolves, a matching `@v0` resolves, and a stale `@v1` fails (`cannot find package …@v1`).

- *Why:* dropping the qualifier survives future major bumps without re-sweeping every import; leaving the stale `@v1` would break the build outright.

### D-F: `opm_platform` becomes an unpublished fixture

It is a `#Platform` (consumer), used only by `flow_synth_integration_test.go` on-disk. It is rewritten to `#Subscription` + the new catalog dep, un-quarantined, and dropped from all publish config. `publish-cue.yml` (which published `apis/core`, `modules/opm`, `modules/opm_platform`) is replaced by the catalog-only `publish-catalog.yml`; the dead `apis/core` entry and the legacy `modules/opm` entry leave `cue-versions.yml`.

- *Why:* `apis/core` publishing died with D24; the catalog moves to the new identity; nothing external consumes the `opm-platform` tag (confirmed in discussion). Stays in `modules/` namespace — it is not a catalog.

## Risks / Trade-offs

- **Resource/trait/blueprint metadata typos ship silently** (only transformers are schema-stamped; D19/experiment 10) → mitigated downstream: a wrong subpath surfaces as a match-time `MissingFQN`. Out of scope to close here; tracked as an additive follow-up.
- **`cue vet` alone can't fully validate the `#Catalog` stamping pattern** (experiment 09: a bad pattern can pass plain `cue vet`) → the in-repo `cue vet ./...` gate plus the existing flow integration test (which materializes against the catalog) provide the concrete-eval coverage.
- **Phase ordering coupling:** phase 3 (fixture) cannot vet until phase 2 pushes `v0.1.0` to the registry → sequence enforced in tasks; local registry (`localhost:5000`) used for the in-repo gate.
- **Two publish paths to maintain** (Go RP + standalone catalog) → accepted as the cost of not breaking `go get`; both are thin.
- **Deviations from 0001 graduation text** (publish lives in `library/Taskfile.yml` not `modules/Taskfile.yml`; a CI workflow rather than only a Taskfile target; standalone not RP) → recorded here and to be noted in 0001's README `## Deviations from Design` when the umbrella is marked implemented.

## Migration Plan

Three independently verifiable commit groups; phase 3 gated on phase 2's tag:

1. **Repackage** (no registry): add `identity/` + `catalog.cue`; retarget `core@v0` (+ `task update-deps`); sweep `modules→catalogs` self-imports dropping `@vN`; replace `version:"v1"` → `id.Version` and `modulePath:` literals → `id.ModulePath`-derived; fix the stray `test-release` FQN. Gate: `cue vet ./...` clean (FQNs read `@0.0.0-dev`).
2. **Publish**: add `cue:publish:catalog` + `0.0.0-dev` guard; replace `publish-cue.yml` with `publish-catalog.yml`; retarget/clean `cue-versions.yml`. Gate: publish `opmodel.dev/catalogs/opm@v0.1.0` to the local registry.
3. **Restore fixture**: rewrite `opm_platform` to `#Subscription` + catalog dep; un-quarantine; confirm unpublished. Gate: `cue vet` clean + flow integration test passes.

Rollback: each phase is its own commit; the catalog is currently non-building, so there is no working baseline to regress — phase 1 is strictly an improvement. A bad `v0.1.0` publish is corrected by a `v0.1.1` (tags are immutable; no un-publish).

## Open Questions

None — all four open questions from exploration are resolved (standalone publish; `opm_platform` unpublished; `v`-tag/bare-FQN layering; dropped self-import qualifiers).
