# Design — library-core-retarget

## Overview

The runtime retarget is one constant. Everything else is making the test fleet coherent with it, under one rule discovered during exploration: **an artifact evaluates against the core version its own `cue.mod` declares, not against the library's loaded schema.** Unification with the loaded (v2) schema happens only where the library fills evaluated values into schema-typed slots — `#ModuleInstance` construction, transformer-context assembly, match/execute. Those seams are where v1-authored mainline fixtures break, and where coherently-pinned historical fixtures don't.

```
                     loaded schema (DefaultSchemaModule)
                     v1 ──────────────▶ v2
                                        │ unifies at seams with
                                        ▼
   mainline fixtures ──── MUST move to v2 (this change)
   pinned fixtures   ──── pin schema cache AND core dep to v1 → untouched
   pulled catalogs   ──── evaluate in their own graph → untouched where
                          they never meet a v2-typed value (canary test)
```

## Research & Decisions

### Float vs pin for the default schema module

**Context**: `DefaultSchemaModule` is a bare major; `OCILoader.loadVersioned` expands it for `load.Instances`' standalone-package loader, resolving the latest version within the major. 0010's plan warns alpha.1–alpha.3 are partial-but-resolvable.
**Explored**: GHCR tag list for `opmodel.dev/core`. The v2 line carries `v2.0.0-alpha.1`–`alpha.4` plus `v2.0.0-0.dev.*` snapshots. SemVer prerelease ordering: `0.dev.*` sorts below `alpha.*` (numeric identifier `0` < alphanumeric `alpha`), so dev snapshots never outrank an alpha.
**Decision**: Float. `DefaultSchemaModule = "opmodel.dev/core@v2"`, mechanism unchanged.
**Rationale**: Identical to today's v1 behaviour; resolves `v2.0.0-alpha.4` now and future alphas (all of which contain the identity package) later. The partial-alpha hazard is a *retarget-target* hazard, not a *float* hazard: floating always resolves the newest tag, and alpha.4 is the newest. Exact pins remain available per-caller via `OCILoader.Module` (used by the kept boundary tests). No new "latest" surface is introduced anywhere.

### Where the fixture catalog lives (option A, refined)

**Context**: Mainline kernel/flow tests materialize the real `opmodel.dev/catalogs/opm` (v1-built). `testdata/modules/web_app` declares it as a `cue.mod` dep. No v2-built catalog exists anywhere, and 0010's ordering puts the real catalogs' v2 authoring *behind* three library slices that depend on this one. CI must stay GHCR-only (workspace Registry Policy); `cmd/flow-inspect` and `task cue:publish` are local flows.
**Explored**: (a) in-memory `registrytest` fixtures only; (b) publishing an early v2-authored `catalogs/opm` alpha to GHCR ahead of the plan. (b) rejected in exploration: it front-runs three planned slices, uses the copy-and-stamp task 0011 is retiring, and publishes permanent bytes for an artifact whose v2 authoring is undesigned. Within (a), a pure in-memory body breaks `flow-inspect` and `cue:publish`, whose on-disk fixtures need a resolvable dep.
**Decision**: A library-owned, on-disk fixture catalog authored against core v2, under the **`testing.opmodel.dev` prefix** (e.g. `testing.opmodel.dev/catalogs/opm@v1`), living beside `modules/opm_platform`. Tests publish it into the in-process `modregistrytest` registry (a disk-tree publish helper is added to `registrytest`); local flows publish it to `localhost:5000` via the existing `task cue:publish` path.
**Rationale**: The `testing.opmodel.dev` prefix is exactly what the Registry Policy routes to the local registry, and tests can route any prefix to the in-process host — so one fixture serves both consumers without a local registry in CI and without touching the real `opmodel.dev` namespace. `web_app` already lives under this prefix.

### Which tests keep their v1 pins

**Context**: "Keep historical tests if possible" (user decision). Three families pull or pin v0/v1-era artifacts, all still published on GHCR (verified: `catalogs/opm` v0.5.2, v0.6.0, v1.0.0-alpha.x).
**Explored**: What each test actually unifies. The v0.5.2 closedness canary (`composed_open_test.go`) pulls the catalog into its own context and reads a `#transform` off the composed map — the loaded core schema is never involved. The synth boundary tests and the library#31 regression pin `pinnedCache("v1.0.0-alpha.1")` *and* author fixtures with `CoreVersion: "v1.0.0-alpha.1"` — both sides of every unification stay v1.
**Decision**: Keep all three families unchanged, including the deep `blueprints/workload` import at `instance_integration_test.go:154` (pinned to catalog v0.6.0, where that path is correct — the recorded D42 deviation). `docs/design/repro-hidden-field/` also stays pinned (v1-era bug repro). Mainline v2 coverage of synth comes from the kernel-level synth/flow tests, which move to the default (v2) path.
**Rationale**: The pins are coherent and self-contained; moving them changes what the tests guard. The retarget's correctness is proven by the mainline suite on v2, not by rewriting regression pins.

### Transitional invariants (why "no behaviour change" holds)

**Context**: Core v2 deleted `#SubscriptionFilter` (D14) and moved the matching vocabulary to `matchLabels` (D36), but the Go code implementing both moves belongs to later slices (`library-subscription-collapse`, `library-match-labels`). The retargeted fixtures must satisfy the v2 schema *and* the v1-era Go reads.
**Explored**: `materialize/filter.go` (reads an optional `filter` field; absent → `highestStable(published)`), `compile/match.go:111` and friends (read `metadata.labels`). Core v1's "upward label union" was normative prose with no implementing code, so v1 catalogs already author `metadata.labels` explicitly.
**Decision**: Two invariants, stated here and enforced by fixture construction:
1. **Every fixture catalog publishes exactly one version per subscription.** The v2 `#Subscription` carries a required scalar `version:`; the Go side ignores it and resolves `highestStable` until the collapse slice — with one published version the two answers coincide.
2. **The fixture catalog authors `metadata.labels` explicitly alongside `matchLabels`.** The matcher keeps reading `metadata.labels`; `library-match-labels` later flips the read and drops the duplication.
**Rationale**: Both invariants are deletions-in-waiting owned by named downstream slices; encoding them in fixtures (not in `opm/` code) keeps this change behaviour-free.

### Subscription-key major handling at pull (discovered during apply)

**Context**: v2's `#ModulePathType` requires the `@vN` suffix, and `#Platform.#registry` keys take that type, so every v2 platform subscription key carries its catalog's major. `materialize.pullCatalog` composes `<key>@<version>`; with a suffixed key that yields `…opm@v1@v1.0.0`, which `load.Instances` rejects ("does not specify a valid semantic version" — measured against cue v0.17.1). `enumerateVersions` needs no change: `modregistry.ModuleVersions` accepts the suffixed form natively and filters tags to that major.
**Explored**: (a) keying the platform major-free — inexpressible: `#registry` is pattern-constrained inside a closed definition, so a major-free key is `field not allowed` (measured); (b) deferring to `library-subscription-collapse` — impossible: the mainline flow/materialize tests must materialize a v2 platform in THIS change.
**Decision**: `pullCatalog` splits the key's `@vN` suffix off (via `ast.SplitPackageVersion`) before composing the load ID. A few lines in `opm/materialize/pull.go`; no signature changes.
**Rationale**: Strictly input-extending. Under core v1 the subscription-key type admitted no `@vN` suffix, so every previously-expressible key is major-free and passes through byte-identical — the proposal's "no behaviour change" claim survives for all previously-valid inputs; the change gives semantics only to keys that were previously unloadable. The `library-subscription-collapse` slice needs the same decomposition and inherits it. This amends the proposal's "subscription-resolution code paths are untouched" and widens task 5.2's expected `opm/` diff by this one file.

## Technical Notes

### The atomic step

`DefaultSchemaModule`, `registrytest.defaultCoreVersion`, and the mainline fixture bodies MUST flip in one commit — any partial state fails the suite. Everything before it (major-aware writers, fixture catalog authoring, disk-publish helper) and after it (doc sweeps, MIGRATIONS entry) lands as independently green commits.

### registrytest major-awareness

`addCatalogs`, `addModules`, and `BuildModuleFile` hardcode `core@v1` (deps line, import line) and `@v0` (module-path major). Each MUST derive the core major from the fixture's effective core version (`coreVersionOr`), and the module-path major from the fixture's declared version, so `CoreVersion: "v1.0.0-alpha.1"` fixtures keep emitting v1 imports. The emitted import and the declared dep MUST always agree on the major.

### v2 fixture shape (what reauthoring means)

Per core v2 (`v2.0.0-alpha.4`):
- `#Module.metadata`: `modulePath!` carries the `@vN` major suffix; `name` is snake_case and MUST equal the path leaf; `version!` required. `fqn` is computed (= modulePath) — never authored.
- Catalog members: `apiVersion!` and `catalogVersion!` required; transformer `fqn!` authored at the definition site (`#ImplFQNType`, full build SemVer); resource/trait `fqn` keyed by `apiVersion` (`#ContractFQNType`).
- `#Platform` subscriptions: required scalar `version:`; `filter:` does not exist.
- Blueprints sit flat under `…/blueprints` (D42) — the fixture catalog is authored flat, which is what moves the `web_app` import.

### Public-surface delta

`schema.DefaultSchemaModule` (value only) and the doc comments that cite it (`kernel.go`, `schema/doc.go`, `materialize/doc.go`, `schema/cache.go` example). No signature or type changes anywhere in `opm/`.
