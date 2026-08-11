# Design — library-core-retarget

## Overview

The runtime retarget is one constant. Everything else is making the test fleet coherent with it, under one rule discovered during the first landing's exploration: **an artifact evaluates against the core version its own `cue.mod` declares, not against the library's loaded schema.** Unification with the loaded (v2) schema happens only where the library fills evaluated values into schema-typed slots — `#ModuleInstance` construction, transformer-context assembly, match/execute. Those seams are where v1-authored mainline fixtures break, and where coherently-pinned historical fixtures don't.

```
                     loaded schema (DefaultSchemaModule)
                     v1 ──────────────▶ v2
                                        │ unifies at seams with
                                        ▼
   mainline fixtures ──── MUST move to v2 (this change), consuming the
                          REAL consolidated catalogs/opm v2 line
   pinned fixtures   ──── pin schema cache AND core dep to v1 → untouched
   pulled catalogs   ──── evaluate in their own graph → untouched where
                          they never meet a v2-typed value (canary test)
```

This is the second landing. library#51 proved the crossing but was reverted (library#52) over its stand-in fixture catalog; 0010 then inverted the ordering so the real catalog moved first. What carries forward from #51 unchanged: the seam rule above, the major-aware fixture writers, three of the four seam adaptations, and the kept-pins analysis. What changes: the catalog the mainline fixtures consume, the import paths (D49 amended D42's flat filing before it could ship here), and a fourth seam adaptation the real catalog's mixed-major tag history forces.

## Research & Decisions

### Float vs pin for the default schema module

**Context**: `DefaultSchemaModule` is a bare major; `OCILoader.loadVersioned` expands it for `load.Instances`' standalone-package loader, resolving the latest version within the major. 0010's history warns alpha.1–alpha.3 are partial-but-resolvable.
**Explored**: GHCR tag list for `opmodel.dev/core`. The v2 line carries `v2.0.0-alpha.1`–`alpha.4` plus `v2.0.0-0.dev.*` snapshots; `v2.0.0-alpha.5` (the D49 gate amendment, core#45) trails core#46's release-metadata fix. SemVer prerelease ordering: `0.dev.*` sorts below `alpha.*` (numeric identifier `0` < alphanumeric `alpha`), so dev snapshots never outrank an alpha.
**Decision**: Float. `DefaultSchemaModule = "opmodel.dev/core@v2"`, mechanism unchanged.
**Rationale**: Identical to today's v1 behaviour; resolves `v2.0.0-alpha.4` now and future alphas later — alpha.5 is safe to absorb, since core#45 touches only the publish-side filing gate (`#CatalogMemberFQNGate` equality, must-fail pins) and no shape the library unifies against. The partial-alpha hazard is a *retarget-target* hazard, not a *float* hazard: floating always resolves the newest tag, and every tag from alpha.4 on is complete. Exact pins remain available per-caller via `OCILoader.Module` (used by the kept boundary tests).

### The real catalog replaces the fixture catalog

**Context**: The first landing needed a v2-built catalog and none could exist — 0010's ordering put the real catalogs' v2 authoring behind the library slices — so #51 authored `modules/opm_catalog`, a minimal stand-in under the `testing.opmodel.dev` prefix, with a registrytest disk-tree publish helper and a cue:vet skip guard. The revert (library#52) judged the stand-in a regression: duplicated shape knowledge with no named retirement owner. 0010 inverted the ordering; `catalogs-identity-authoring` and `catalogs-consolidation` are now **done**, and `catalog_opm` is the single first-party catalog (D47) with D49 versioned filing, released on its v2 line via release-please (`2.0.0-alpha.1` authoring; `2.0.0-alpha.2` consolidated — the first tag with the `<kind>/<apiVersion>/` layout and both families).
**Explored**: `catalog_opm` main: `src/blueprints/v1beta1/`, `src/traits/v1beta1/`, `src/resources/{v1,v1alpha1,v1beta1,v2}/`, transformers flat; package clause equals the version segment, so imports name the level (`bp "opmodel.dev/catalogs/opm/blueprints/v1beta1"`). The abstraction family (all the members `web_app` demands) sits at `v1beta1`. `metadata.labels` is authored alongside `matchLabels` (recorded 0010 deviation) — the duplicate the matcher still reads.
**Decision**: Mainline fixtures consume the real `opmodel.dev/catalogs/opm@v2` from GHCR — `web_app`'s `cue.mod` dep and `opm_platform`'s subscription both pin the newest consolidated tag. In-process `registrytest` inline `#Catalog` fixtures remain the hermetic path for materialize/kernel tests. Nothing from the fixture-catalog apparatus is re-landed.
**Rationale**: The revert's reason dissolves — there is no stand-in to retire. CI stays GHCR-only with no special-casing (the guard existed only because `testing.opmodel.dev` needed a local registry). And the fixtures become representative of the production layout: the imports this change writes are the imports the fleet will write, so `catalogs-republish` later moves no library fixture.

### Major-scoped version enumeration (the fourth seam adaptation)

**Context**: `materialize/enumerate.go` lists **every** published version of a path — its own doc comment says "a bare path enumerates every published version regardless of major" — and the `catalogs/opm` OCI repo carries the v0 line's stable `v0.5.2`/`v0.6.0` beside the v1 and v2 prereleases. The v1 fixture avoids selecting across lines with an explicit `filter: range:`, and its comment states why: "An unfiltered subscription resolves the highest *stable* version (v0.6.0, a core@v0 catalog)". Core v2 deleted `#SubscriptionFilter` (D14), so a v2 platform has no filter to say this with — the unscoped no-filter default would pull a core-v0 catalog into a v2 platform.
**Explored**: `filter.go`'s `highestStable`: skips prereleases, falls back to the highest overall only when *no* stable exists — so scoping is what makes the fallback reach the v2 alphas. The v2 `#registry` key carries the mandatory `@vN` (D1's `#ModulePathType`); #51's pull adaptation already splits it off the load ID but discarded it.
**Decision**: The major suffix scopes enumeration. A subscription key `path@vN` narrows the published list to major N before `filterVersions` runs; a major-free key (core v1) enumerates the whole repo exactly as today.
**Rationale**: This is address decomposition, not selection semantics. Under D1 a v2 `modulePath` *is* the coordinate, majors included; reading the whole address is the same move as the other three seam adaptations, and it is strictly input-extending — a major-suffixed key was not expressible in v1, so no existing input changes behaviour. The alternative of reading the scalar `version:` field instead is `library-acquire-and-subscription`'s D14 work (delete `filter.go`, make the pin load-bearing) and stays there; pulling half of it forward would leave the slice boundary meaningless. Leaving enumeration unscoped is not an option: it selects `v0.6.0` deterministically, and there is no filter left to override it.

### Which tests keep their v1 pins

**Context**: "Keep historical tests if possible" (user decision, first landing). Three families pull or pin v0/v1-era artifacts, all still published on GHCR (verified: `catalogs/opm` v0.5.2, v0.6.0, v1.0.0-alpha.x — published tags name fixed bytes permanently).
**Explored**: What each test actually unifies. The v0.5.2 closedness canary (`composed_open_test.go`) pulls the catalog into its own context and reads a `#transform` off the composed map — the loaded core schema is never involved. The synth boundary tests and the library#31 regression pin `pinnedCache("v1.0.0-alpha.1")` *and* author fixtures with `CoreVersion: "v1.0.0-alpha.1"` — both sides of every unification stay v1.
**Decision**: Keep all three families unchanged, including the deep `blueprints/workload` import at `instance_integration_test.go:154` (pinned to catalog v0.6.0, where that path is correct — the recorded D42 deviation, which D49's versioned filing does not reopen: the site is pinned below both). `docs/design/repro-hidden-field/` also stays pinned (v1-era bug repro). Mainline v2 coverage of synth comes from the kernel-level synth/flow tests, which move to the default (v2) path.
**Rationale**: The pins are coherent and self-contained; moving them changes what the tests guard. The retarget's correctness is proven by the mainline suite on v2, not by rewriting regression pins.

### Transitional invariants (why "no behaviour change" holds)

**Context**: Core v2 deleted `#SubscriptionFilter` (D14) and moved the matching vocabulary to `matchLabels` (D36), but the Go code implementing both moves belongs to later slices (`library-acquire-and-subscription`, `library-matching`). The retargeted fixtures must satisfy the v2 schema *and* the v1-era Go reads.
**Explored**: `materialize/filter.go` (reads an optional `filter` field; absent → `highestStable(published)`, falling back to the highest prerelease on a prerelease-only list), `compile/match.go:111` and friends (read `metadata.labels`). The catalog's v2 line publishes only prereleases and will until 0010's `catalogs-republish`.
**Decision**: Two invariants, stated here rather than discovered later:
1. **The platform fixture's scalar `version:` pins the newest published tag on the catalog's v2 line.** The Go side ignores the pin and resolves `highestStable`, which on the major-scoped, prerelease-only v2 list falls back to the highest alpha — the same tag. **This invariant is time-fragile by construction**: the next catalog release diverges the two until the fixture pin is bumped. Accepted, because v2 contract keys are apiVersion-keyed (D4/D25) — a newer materialized build still supplies every demanded key, so drift degrades determinism, not correctness — and because `library-acquire-and-subscription` deletes `filter.go` and makes the pin load-bearing, removing the fragility at its root.
2. **The catalog authors `metadata.labels` explicitly alongside `matchLabels`** — a recorded 0010 catalog-authoring deviation kept precisely for this window. The matcher keeps reading `metadata.labels`; `library-matching` later flips the read and the catalog drops the duplicate. Unlike the first landing, this invariant is inherited from the real catalog rather than enforced by fixture construction.

## Technical Notes

### The atomic step

`DefaultSchemaModule`, `registrytest.defaultCoreVersion`, and the mainline fixture bodies MUST flip in one commit — any partial state fails the suite. Everything before it (major-aware writers, the four seam adaptations) and after it (doc sweeps, MIGRATIONS entry) lands as independently green commits. The seam adaptations were staged exactly this way in library#51 and were green in isolation; re-land them from that commit (`git show 9199bdf`) rather than rediscovering them.

### registrytest major-awareness (re-land)

`addCatalogs`, `addModules`, and `BuildModuleFile` hardcode `core@v1` (deps line, import line) and `@v0` (module-path major). Each MUST derive the core major from the fixture's effective core version (`coreVersionOr`), and the module-path major from the fixture's declared version, so `CoreVersion: "v1.0.0-alpha.1"` fixtures keep emitting v1 imports. The emitted import and the declared dep MUST always agree on the major. The disk-tree publish helper from #51 is NOT re-landed.

### v2 fixture shape (what reauthoring means)

Per core v2 (`v2.0.0-alpha.4`) and catalog_opm's D49 layout:
- `#Module.metadata`: `modulePath!` carries the `@vN` major suffix; `name` is snake_case and MUST equal the path leaf; `version!` required. `fqn` is computed (= modulePath) — never authored.
- Catalog members: `apiVersion!` and `catalogVersion!` required; transformer `fqn!` authored at the definition site (`#ImplFQNType`, full build SemVer); resource/trait `fqn` keyed by `apiVersion` (`#ContractFQNType`).
- `#Platform` subscriptions: key is the major-suffixed path (`opmodel.dev/catalogs/opm@v2`); required scalar `version:`; `filter:` does not exist.
- Imports name the level (D49): `…/blueprints/v1beta1`, `…/traits/v1beta1`, `…/resources/v1beta1`, package clause = version segment. Transformers are never imported by fixtures (they arrive via materialize).
- `web_app` uses abstraction-family members only, so the `k8s-` raw family's spec-key consequence (`spec: k8sDeployment:`) does not reach it.

### Public-surface delta

`schema.DefaultSchemaModule` (value only) and the doc comments that cite it (`kernel.go`, `schema/doc.go`, `materialize/doc.go`, `schema/cache.go` example). No signature or type changes anywhere in `opm/`. The major-scoped enumeration and the other three seam adaptations are internal behaviour under the `platform-materialization` and `instance-synthesis` capabilities, visible only to inputs that were inexpressible before core v2.

### Apply-time findings (2026-08-11)

Two deviations from this design's own plan surfaced during implementation; both are also recorded in 0010's history.

1. **The fourth seam adaptation required no new resolution logic.** `cuelang.org/go`'s `modregistry.ModuleVersions` already scopes a major-suffixed path to its major (it splits the suffix and filters tags on `semver.Major`), so materialize's existing pass-through of the raw subscription key into `enumerateVersions` *is* the scoping. What landed is the locking unit test (`TestEnumerate_MajorSuffixScopesSelection`, mixed-major published list modelled on the real repo) and the `enumerate.go` doc rewrite. The `platform-materialization` spec delta's scenarios hold verbatim; only the imagined diff was wrong.
2. **`web_app` authors `updateStrategy.rollingUpdate` explicitly** (`maxUnavailable`/`maxSurge` 1) although `#UpdateStrategySchema` marks the field optional: `catalog_opm`'s deployment-transformer dereferences `#component.spec.updateStrategy.rollingUpdate` unguarded whenever `type` is `"RollingUpdate"`, so a schema-legal omission fails transform execution with an empty-disjunction error. Latent `catalog_opm` bug to fix catalog-side (same idiom the transformer's own `_updateStrategy` warning comment documents); the fixture accommodation is annotated at the authoring site and reverts when the guard lands.

Verification additionally repaired the `cue:catalog:drift` Taskfile check — previously vacuous (jq key hardcoded to the v0 dep form; stable-only tag filter) — so it now enforces transitional invariant 1: both `web_app`'s dep pin and `opm_platform`'s `#registry` scalar `version:` are compared against the newest tag on their own major line, stable-first with a prerelease fallback mirroring `filter.go`'s `highestStable`.
