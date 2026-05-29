## 1. Identity + manifest scaffolding (Phase 1)

- [x] 1.1 Update `library/modules/opm/cue.mod/module.cue`: identifier `opmodel.dev/catalogs/opm@v0`; replace the `opmodel.dev/core@v1` dep with `opmodel.dev/core@v0`; drop the stale `opmodel.dev/modules/opm@v1` self-dep if present. Run `task update-deps` (never hand-edit the version pin).
- [x] 1.2 Add `library/modules/opm/identity/identity.cue` (package `identity`) exporting `ModulePath: "opmodel.dev/catalogs/opm"` and `Version: #VersionType | *"0.0.0-dev"`. Confirm no `version_override.cue` is committed.
- [x] 1.3 Add `library/modules/opm/catalog.cue` (package `opm`) embedding bare `c.#Catalog`, with `metadata` sourced from `identity/` and a `#transformers` map keyed by each transformer's `metadata.fqn`. Use the `M=metadata` label-alias form (matches `core/src/catalog.cue`).

## 2. Primitive re-targeting sweep (Phase 1)

- [x] 2.1 Rewrite every `import c "opmodel.dev/core/v1alpha2@v1"` to `import c "opmodel.dev/core@v0"` across the tree (51 occurrences). Verify zero `core/v1alpha2` matches remain.
- [x] 2.2 Rewrite all same-module self-imports `opmodel.dev/modules/opm/...` → `opmodel.dev/catalogs/opm/...`, dropping the trailing `@vN` qualifier (resources, traits, transformers, blueprints, schemas). Verify zero `opmodel.dev/modules/opm` matches and zero same-module `@v1` qualifiers remain.
- [x] 2.3 In `resources/`, `traits/`, `blueprints/`: replace each `version: "v1"` with `id.Version` and each hardcoded `modulePath:` literal with the `id.ModulePath`-derived form (`"\(id.ModulePath)/resources"`, `/traits`, `/blueprints/workload`); add the `identity` import where needed.
- [x] 2.4 In `transformers/`: same substitution (`modulePath: "\(id.ModulePath)/transformers"`, `version: id.Version`) so each transformer's `metadata.fqn` is concrete for the `#transformers` map key.
- [x] 2.5 Fix the stray `fqn: "opmodel.dev/modules/opm/test-release:0.1.0"` in `transformers/sa_resource_transformer.cue:65` (correct or remove the test value).

## 3. Phase 1 gate

- [x] 3.1 `cd library/modules/opm && cue fmt ./... && cue vet ./...` passes (FQNs resolve to `@0.0.0-dev`).
- [x] 3.2 Verify transformer stamping is enforced: temporarily introduce a `modulePath` typo in one transformer, confirm `cue vet` fails with a "conflicting values" diagnostic, then revert.
- [x] 3.3 Commit Phase 1 (`feat(catalog): repackage onto core@v0 with #Catalog + identity`).

## 4. Publish task + guard (Phase 2)

- [x] 4.1 Add `cue:publish:catalog` to `library/Taskfile.yml`: rsync source → `.build/catalog/` excluding `cue.mod/{pkg,gen,usr}`; write `.build/catalog/identity/version_override.cue` pinning the bare `Version`; `cue vet` from build dir; `cue mod publish v<VERSION>` from build dir.
- [x] 4.2 Add the `0.0.0-dev` publish guard: the task exits non-zero (no tag pushed) when the resolved `identity.Version` is `0.0.0-dev`.
- [x] 4.3 Retarget/clean `library/cue-versions.yml`: remove the `apis/core` entry and the legacy `modules/opm` entry; add the catalog under its `catalogs/opm` identity at `v0.1.0`.

## 5. Publish workflow (Phase 2)

- [x] 5.1 Add `library/.github/workflows/publish-catalog.yml`: on push to `main`, read the catalog version from `cue-versions.yml`, HEAD the GHCR manifest for that tag, and if absent login + `task cue:publish:catalog VERSION=<that>`. Stateless (no commit-back); catalog-only; no release-please.
- [x] 5.2 Remove `library/.github/workflows/publish-cue.yml`; confirm no remaining workflow publishes `modules/opm` or `modules/opm_platform`. Confirm the Go-module `release.yml` is untouched.

## 6. Phase 2 gate

- [x] 6.1 With the local registry running (`localhost:5000`), run `task cue:publish:catalog VERSION=0.1.0`; confirm tag `v0.1.0` lands and the published `metadata.fqn` reads `opmodel.dev/catalogs/opm@0.1.0` (bare).
- [x] 6.2 Confirm the guard: attempt a publish without the stamp and verify it is rejected with no tag pushed.
- [x] 6.3 Commit Phase 2 (`ci(catalog): standalone content-diff publish for catalogs/opm`).

## 7. Restore opm_platform fixture (Phase 3)

- [x] 7.1 Rewrite `library/modules/opm_platform/_platform.cue.quarantined` to a `#Subscription`-shaped `#registry` importing the republished catalog; update `opm_platform/cue.mod/module.cue` deps (`core@v0`, `opmodel.dev/catalogs/opm@v0`).
- [x] 7.2 Rename the file back to `platform.cue`; delete `QUARANTINE.md`.
- [x] 7.3 Confirm `opm_platform` is absent from all publish config (already removed in 5.x); it remains an unpublished fixture in the `modules/` namespace.

## 8. Phase 3 gate + final validation

- [x] 8.1 `cd library/modules/opm_platform && cue fmt ./... && cue vet ./...` passes against the published catalog.
- [x] 8.2 `task cue:test:flow` (plan→match→compile integration) passes using the restored on-disk fixture.
- [x] 8.3 `task check` passes (Go side unaffected: fmt/vet/lint/test green).
- [x] 8.4 Commit Phase 3 (`test(catalog): restore opm_platform fixture on #Subscription`).

## 9. Enhancement bookkeeping

- [ ] 9.1 Add `enhancements/0001/config.yaml` history events for the landed library + catalog pieces (dated); confirm whether to flip `implementation.status` (umbrella may still await workspace `modules/*` rewire — leave per 0001 graduation).
- [ ] 9.2 Note the deviations from 0001's graduation text (publish in `library/Taskfile.yml`; CI workflow vs Taskfile-only; standalone vs RP) for 0001's README `## Deviations from Design`.
