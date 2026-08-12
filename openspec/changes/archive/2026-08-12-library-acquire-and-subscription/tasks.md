# Tasks — library-acquire-and-subscription

> Depends on `library-compat-comparator` (the `highestStable` move — `filter.go` must no longer own it). Groups 1–2 are additive and green in isolation; group 3 is the semantic flip (scalar read + deletions land together or the suite breaks); groups 4–6 are independently green follow-ons. Breaking change — MIGRATIONS entry mandatory, `Migration:` trailer if `add-migration-guard` has landed by then.

## 1. The typed error

- [x] 1.1 `opm/errors/identity.go`: `IdentityError{Artifact, Field, Declared, Fetched, Coordinate}`, value-receiver `Error()` naming both values; doc cites D9/D11.
- [x] 1.2 `opm/errors/identity_test.go`: message shape, `errors.As` through a `*MaterializeError` wrap (the catalog-site path).

## 2. Module-acquire verification (green pre-step)

- [x] 2.1 `opm/helper/loader/registry/module.go`: after the shape gate, compare declared `metadata.modulePath` vs requested path and declared `metadata.version` vs tag (v-stripped); return `oerrors.IdentityError` on mismatch. One insertion point covering both entrypoints. *(Amended: major-free v0/v1-shaped declarations verify the enhancements/0003 parent-path convention — see design.md amendment.)*
- [x] 2.2 Tests via `registrytest`: publish a module whose metadata declares a different path; a different version; assert the typed error carries both values. Happy path unchanged.
- [x] 2.3 `task check:fast` green (materialize still on the old resolution — untouched so far).

## 3. The scalar flip (one commit)

- [x] 3.1 `materialize.go`: resolution block per design — `subscriptionVersion` read, major-agreement check (target-schema `_majorAgrees` semantics; suffixless key is an error), single `pullCatalog`, `verifyCatalogIdentity` (modulePath vs key, version vs tag; `IdentityError` wrapped in `MaterializeError`), `resolved[sub] = ver`. Delete `decodeFilter`.
- [x] 3.2 Delete `filter.go` and `filter_test.go`. Verify the semver import leaves `materialize` entirely.
- [x] 3.3 `enumerate.go`: doc rewrite to diagnostic-only; wire the lazy enumeration into the pull-failure error path ("named build not published; published in this major: …").
- [x] 3.4 Test rewrites: `enumerate_pull_test.go` (major-scoping test now asserts the diagnostic list, not selection); `materialize_test.go` (authored `version:` is the asserted selection; remove the NOTE tombstone; add: named-version-absent → error carrying the published list; major-disagreement → error; identity-mismatch fixture → typed error).
- [x] 3.5 `types.go`: `Resolved` doc comment rewritten (records what the platform said; verified).
- [x] 3.6 Full `task test` green; run the GHCR flow test (`OPM_FLOW_TEST_FORCE=1`) — `modules/opm_platform`'s pin resolves the real catalog by its authored version.

## 4. Cache key trim

- [x] 4.1 `materialize/cache/key.go`: drop `Range`/`Allow`/`Deny` from `normSub`, the filter lookup block, both sorts, the `sort` import; doc comment updated.
- [x] 4.2 `cache_test.go`: assert a v2 platform's key is byte-identical before/after the trim (golden from the pre-change code path); drop v1-shape cases.

## 5. Synth write side

- [x] 5.1 `helper/synth/platform.go`: `SubscriptionSpec{Enable *bool; Version string}`; delete `FilterSpec` + `writeFilter`; `writeRegistry` emits `version:`; `Platform()` errors on empty `Version` naming the subscription path.
- [x] 5.2 `helper/synth/platform_test.go`: replace the transitional filter-rejection test; add empty-version refusal; update all `SubscriptionSpec` literals.
- [x] 5.3 `kernel/synth_platform_test.go`: fixtures gain `Version` (pin to the same tag as `modules/opm_platform`); assert the synthesized platform materializes the named build.

## 6. D41 residue

- [x] 6.1 `schema/metadata.go`: `InstanceMetadata` gains `FQN` (decoded via `schema.MetadataFQN` from the instance value).
- [x] 6.2 `schema/context.go`: `#moduleInstanceMetadata.fqn` fills from the instance's own FQN; `Version` unchanged (doc comment notes it is acquire-verified). Test asserting the context block carries `registryPath:name:namespace`.

## 7. Transitional scaffolding + docs

- [x] 7.1 `Taskfile.yml` `cue:catalog:drift`: replace both jq `highestStable` mirrors with a named-tag-exists check; rewrite the header comment.
- [x] 7.2 `modules/opm_platform/platform.cue`: pin comment rewritten (load-bearing pin, ordinary updates).
- [x] 7.3 `materialize/doc.go` step 2 + `CLAUDE.md` § Materialize lifetime ("version enumeration + OCI pulls" → per-subscription pull; enumeration on failure only).
- [x] 7.4 Spec deltas: `platform-materialization` (MODIFIED/REMOVED/ADDED per design), `registry-module-loading` (ADDED identity verification), `platform-synthesis` (MODIFIED subscription shape).

## 8. Verify & record

- [x] 8.1 `MIGRATIONS.md` `## Unreleased — Breaking` — `### Changed — \`library-acquire-and-subscription\``: scalar subscription (author `version:`; absent-from-registry now fails), synth `Filter` → `Version`, new `IdentityError` on acquire/materialize. PR carries `Migration:` trailer if the guard workflow is live.
- [x] 8.2 `task check` clean; `task cue:check` clean against GHCR.
- [x] 8.3 Record back in `enhancements/0010/`: slice → `done` + history event + two recorded deviations (third read point collapsed into materialize; operator CRD `Subscription.Filter` contradicts 02-design's "no operator feature code" — flagged for `operator-library-retarget`).
- [x] 8.4 Downstream heads-up notes in the record: `cli/internal/platform` `wireFilter` breakage lands in `cli-coordinate-adoption`; template shape in `cli/internal/config/templates.go` moves `filter: range:` → `version:` there too.
