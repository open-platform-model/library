## Why

`Catalog.Provides()` (`opm/catalog/provides.go`) derives the provider-fulfilled contracts a catalog implements by walking its `#transformers` in Go. The same rule was written twice more in core: once in `#Platform.#contracts._providerSet` and once again by the operator's 0015:D11 claim check, which compares a `TransformerRegistration`'s claimed `provides` against `Provides()`. ADR-012 (`adr/012-matching-stays-in-the-library-glue.md`) lets one derived rule at a time move into core, with a parity test, and names the per-catalog provider set as the next one.

The owner decided h2 in the beta-1 kernel-plan walkthrough (2026-10-02/03, `claude-stuff/kernel-plan-beta1/walkthrough-decisions.md`):

> h2: Core gains a per-catalog provider set (additive core release); Provides() decodes it and falls back to the deprecated Go fold for older catalogs; the fallback is removed before GA, after catalog_opm is republished.

Core did its half in core#120, released as core `v2.0.0-beta.3` (published to GHCR on 2026-10-05). `#Catalog` now carries a regular field `provides`: the sorted, deduplicated FQNs of every `requiredResources` or `requiredTraits` key of its `#transformers` whose requirement reads `fulfilment: "provider"`. `#Platform` folds it into `providedBy`. This change is the library half: the library pins that core and reads the field.

An old catalog can still reach `Provides()`. The library acquires a catalog standalone, at the core its own `cue.mod` pins. Every published `opmodel.dev/catalogs/opm` build pins core `v2.0.0-beta.2` or older until catalog_opm is republished, and so has no `provides`. Inside a platform build MVS lifts every embedded catalog to the platform's core, so the platform inventory is not affected; the standalone acquisition the operator's claim check uses is.

## What Changes

- **The library pins core `v2.0.0-beta.3`.** This covers `schema.DefaultSchemaModule` (`opm/schema/loader.go:43`), and through it `registrytest.DefaultCoreVersion`, which derives from it. It also covers the core pin of `testdata/cue.mod`, every `testdata/render` tree, the `CUE_MODULE_GLOBS` modules (the parity modules, `testdata/modules/web_app`, `modules/opm_platform`), and the two docs examples (`docs/getting-started.md`, `AGENTS.md`). The bump runs through the repo's own `task -x deps:cascade`, the task the release cascade's bot PR runs. The cascade is in dry-run (`CASCADE_DRY_RUN=true`), so no bot PR carries the bump, and on 2026-10-05 no open library PR does. The same task moves the opm catalog pin to the newest published build the new core allows. If the catalog pin moves, the parity harness literals that name the shipped build move in the same commit, and so does any golden that `TestParity_ShippedCatalog` and `TestParity_ShippedCatalogDiscriminated` compare. The `api-diff` check reports the `DefaultSchemaModule` value change on its fixed allow line (SD17), so it is not charged as a break.
- **`Provides()` decodes core's provider set when the catalog carries it.** The field is read at `schema.CatalogProvides` (new, `cue.ParsePath("provides")`). The result keeps today's contract: sorted, deduplicated, a non-nil empty slice when the catalog provides nothing, and no partial set on error. A `provides` that is present but does not decode as a concrete list of strings is reported as an error naming the field. The fold reports an unreadable `fulfilment` the same way, so a catalog whose provider set cannot be read errors on both paths and is never silently empty.
- **The Go fold stays as a deprecated fallback.** It becomes the unexported `providesFold`, marked Deprecated in its doc comment, and runs only when the catalog has no `provides` field, which means a catalog built against a core older than `schema.ProvidesSince`. Its doc comment states the removal condition: before GA, after catalog_opm is republished against core `v2.0.0-beta.3` or later. Removing it is its own later change.
- **`schema.ProvidesSince = "2.0.0-beta.3"`** is new: the first core release deriving `#Catalog.provides`. Like `CollisionsSince`, it documents the field and tests use it. It is never a floor, because an older catalog is answered by the fallback rather than refused.
- **Tests.**
  - The existing unit cases keep pinning the fold over catalogs that carry no `provides`.
  - New unit cases pin the decode: present wins, order and duplicates normalised, empty list, a non-concrete or wrongly typed field errors.
  - A parity test (ADR-012) builds provider catalogs against the pinned core and asserts the decoded field equals the fold, element for element, for each of them. The catalogs are the unit-case shapes rebuilt as real `#Catalog` artifacts, plus every committed `testdata/render/registry` catalog.
  - An old-catalog test acquires a catalog pinned to core `v2.0.0-beta.2`, asserts it has no `provides`, and asserts `Provides()` still returns the fold's result.

Not **BREAKING**. `Provides()` keeps its signature and its result for every catalog: core's field and the fold state the same rule, and the parity test pins that. Error text for an undecodable `provides` is new; no consumer matches `Provides()` error text (grep of cli and opm-operator at `origin/main`, 2026-10-05). The public surface grows by two `opm/schema` symbols (`CatalogProvides`, `ProvidesSince`); no symbol is removed or renamed (SD1).

SemVer class: MINOR. Release class of the PR: `feat`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `catalog-acquisition`: the provider-fulfilled contract set is read from core's per-catalog `provides` field when the catalog carries it. A catalog built against an older core still derives it through the deprecated fold. Both paths return the same set, and an unreadable set is an error on both.

## Impact

- Packages:
  - `opm/catalog` (`provides.go`, its tests, an `export_test.go` exposing the fallback to the parity test);
  - `opm/schema` (`paths.go`: `CatalogProvides`, `ProvidesSince`, and the doc of the paths the fold reads; `loader.go`: `DefaultSchemaModule`);
  - the core pins under `testdata/` and `modules/opm_platform`;
  - `docs/getting-started.md` and the `AGENTS.md` core example (both moved by the cascade task);
  - the `README.md` and `AGENTS.md` layout lines for `catalog/`, which say where `Provides()` reads from;
  - `opm/kernel/parity_harness_test.go`, only if the catalog pin moves.
- Downstream:
  - opm-operator's `transformerregistration_controller.go` calls `Provides()` for the 0015:D11 claim check. It gets the same answer for every catalog, published or new. Its old-catalog test is the operator's half of h2, not this change.
  - The cli does not call `Provides()`.
  - Both frontends pick up the core pin through the deps cascade. The library pin bump changes no Go API they use. Both frontends are built against this head by the `consumer-build` job.
- Ordering (wave-2 serialization): gated on `release:core@v2.0.0-beta.3`, which is met. Not in the k8s-tier release gate (SD3). It is independent of the round-2 library siblings (lib-b1g2, lib-i3d2, lib-e2e5): it touches neither the acquire/load chain nor `go.mod`. If a sibling's merge also moves a core pin, the second to merge brings its branch up to date with `origin/main` (SD16).
- Follow-ups outside this change:
  - catalog_opm republished against core `v2.0.0-beta.3` (cat-h2-republish);
  - the operator's old-catalog test;
  - the later change that deletes `providesFold` before GA.
- No `enhancement.yaml`. The decision comes from the beta-1 kernel-plan walkthrough, not from an enhancement. The 0015:D11 claim check is unchanged, and this change logs no delivery against it.
