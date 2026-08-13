# Proposal — library-acquire-and-subscription

## Why

Two defects, one seam retirement.

**A subscription's authored version is dead text today.** Core v2's `#Subscription` is `{enable, version!}` (D14 deleted `#SubscriptionFilter`), but the kernel still runs the v1-era resolution: `decodeFilter` looks up `filter.range`/`allow`/`deny`, always misses on a v2 platform, and falls through to `highestStable` over the enumerated list. The authored `version:` is never read. The retarget recorded this as transitional invariant 1 and named it time-fragile by construction: the fixture pin and the float agree only until the next catalog release, propped up by a shell reimplementation of `highestStable` in `Taskfile.yml`'s drift check. D14's rule — "catalog selection is a pure function of committed source … the platform file *is* the resolution" — makes the pin load-bearing and removes the fragility at its root: no range solving, no allow/deny arbitration, no highest-stable default, no maturity inference. One rule survives: the named build must sit in the subscription key's major.

**Nothing verifies that an artifact lives where its metadata says it lives.** D11: a mismatch between declared identity and fetched coordinate is a typed error naming both values, checked where artifacts are read. D9 adds the version clause: an acquired artifact's `metadata.version` is compared against the tag it was fetched by (the kernel is the version label's *verifier*, never its source — measured defect: three published `jellyfin` artifacts carried one label value). Today the equivalent condition surfaces downstream as `module not found` or not at all.

This change is 0010's `library-acquire-and-subscription` slice (see `enhancements/0010/plan.yaml`). Its decisions: D9, D11, D14, D41. It depends on `library-compat-comparator` having moved `highestStable` out of `filter.go` (the declared move-before-delete seam).

## What Changes

- **`opm/materialize` reads the scalar.** For each enabled subscription: read `version!`, check major agreement against the key's `@vN` suffix, pull exactly that build. `filter.go` and `filter_test.go` are deleted entire (`filterVersions`, `subscriptionFilter`, `matchingPublished`, `decodeFilter` in `materialize.go`); `enumerate_pull_test.go`'s filter-dependent assertions are rewritten. Enumeration leaves the selection path and survives only as a failure diagnostic: when the pull of a named build fails, `enumerateVersions` runs to enrich the error with what *is* published (D14 keeps `published` as a diagnostic surface; the happy path makes no enumeration round-trip).
- **Identity verified at both library read sites** (new typed error, see design):
  - *Module acquire* — `opm/helper/loader/registry/module.go`, immediately after the shape gate (both compared fields are shape-guaranteed concrete there). One insertion covers `LoadModulePackage` and `LoadModulePackageWithSource`, so `Kernel.AcquireModuleFromRegistry` and the CLI and operator inherit one implementation, as D11 requires. Declared `metadata.modulePath` vs the fetched path; declared `metadata.version` vs the fetched tag with `v` stripped (the `#FetchedArtifact` shape).
  - *Catalog materialize* — after `pullCatalog`: `metadata.modulePath` vs the subscription key (direct string comparison, both `@vN`-suffixed, no recomposition) and `metadata.version` vs the pulled tag.
  - *Recorded deviation:* D11's third read point ("when a catalog is added to a platform's registry") has no library home — nothing in the platform loader resolves subscriptions, by documented design. In-library it collapses into the materialize check; the fires-earliest, platform-author-facing property is delivered by frontends calling materialize at subscription time. Recorded in 0010 history on landing.
- **`materialize/cache` drops the v1 filter fields** from `normSub` (`Range`/`Allow`/`Deny` and the two sorts). Emitted keys for v2 platforms are byte-identical before and after (a v2 subscription can never populate those fields); v1 platforms are no longer loadable since the retarget, so no live key changes.
- **`opm/helper/synth` stops writing the unwritable.** `SubscriptionSpec` gains required `Version` and loses `Filter`; `FilterSpec` and `writeFilter` are deleted; `Platform()` refuses an empty `Version` at synthesis (early validation). Without this, every synthesized platform materializes nothing the moment the kernel reads the scalar — the coupling is unavoidable, and the transitional test pinning the old behavior names this slice as its remover.
- **`MaterializedPlatform.Resolved` survives** with its doc rewritten: it records what the platform *said* (now verified), not what the kernel *chose* — those are the same value by construction. The "highest survivor" clause is unreachable.
- **D41 residue:** `schema/context.go`'s `#moduleInstanceMetadata.fqn` flips from the module's FQN to the instance's own `metadata.fqn` (core v2 defines it; measured free — no shipped transformer reads `.fqn` from that block). `InstanceMetadata` gains the field; `Version` stays, fed by the declared (now acquire-verified) version.
- **Transitional scaffolding retired:** `Taskfile.yml`'s `cue:catalog:drift` shrinks from a shell `highestStable` mirror to a named-tag-exists check; `modules/opm_platform/platform.cue`'s pin comment is rewritten (the pin is load-bearing now); `materialize_test.go`'s NOTE tombstone and `helper/synth`'s transitional filter-rejection test are resolved.

## Capabilities

### New Capabilities

<!-- none -->

### Modified Capabilities

- `platform-materialization`: subscription resolution becomes the scalar rule (named build + major agreement + identity verification); the enumeration-and-filtering requirement is removed.
- `registry-module-loading`: acquire gains the declared-vs-fetched identity verification requirement.
- `platform-synthesis`: a synthesized subscription carries a required `version` and can no longer express a filter.

## Impact

- **`opm/` public surface:** `synth.SubscriptionSpec` shape change (field removed, field added-required); `synth.FilterSpec` deleted; new error type in `opm/errors`; `InstanceMetadata` gains a field. `materialize` internals shrink; no materialize signature changes.
- **SemVer: breaking** (Principle VI). Ships as `feat!:` with a `MIGRATIONS.md` `## Unreleased — Breaking` entry (and a `Migration:` trailer if `add-migration-guard` has landed). Recipes: platforms must author `version:` per subscription (they already must to pass the v2 schema); synth callers move `Filter` → `Version`.
- **Behavior changes for existing inputs:** a v2 platform whose authored version is absent from the registry now fails materialize (previously the float silently selected something else — that silent divergence is the defect); synthesized platforms without a version are refused at synth instead of materializing nothing.
- **Packages touched:** `opm/materialize` (+`cache`), `opm/helper/loader/registry`, `opm/helper/synth`, `opm/errors`, `opm/schema` (context + metadata), `opm/kernel` (synth-platform test fixture), `Taskfile.yml`, `modules/opm_platform`.
- **Downstream (coordination, out of scope here):** `cli/internal/platform`'s `wireFilter` → `synth.FilterSpec` mapping breaks and moves to `Version` in `cli-coordinate-adoption`; `opm-operator`'s CRD carries `Subscription.Filter` — a versioned API field whose retirement is an operator-side concern. The latter contradicts 0010 02-design's "operator needs no feature code" claim; record as a deviation in 0010 history so `operator-library-retarget` inherits it.
- **Ordering:** requires `library-compat-comparator` (the `highestStable` move); should land before `library-matching` only by plan convention (no code dependency — the two touch disjoint files in `materialize`).
