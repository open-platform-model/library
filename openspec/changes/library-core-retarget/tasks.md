# Tasks — library-core-retarget

> Second landing. Staged so every commit before and after the atomic flip is independently green; the flip itself (3.x) is one commit — `DefaultSchemaModule`, `defaultCoreVersion`, and the mainline fixtures move together or the suite fails. Re-land the carried-forward pieces from library#51 (`git show 9199bdf`) rather than rediscovering them; the fixture catalog, its disk-tree publish helper, the `testing.opmodel.dev` routing, and the cue:vet skip guard from that commit are NOT re-landed.

## 1. Major-aware fixture writers (green pre-step, re-land)

- [x] 1.1 `registrytest.addCatalogs`: derive the core dep line's major and the emitted `import c "opmodel.dev/core@vN"` from `coreVersionOr(f.CoreVersion)` — emitted import and declared dep MUST agree. Existing v1-pinned callers unchanged.
- [x] 1.2 `registrytest.addModules`: same derivation for the deps line; module-path major suffix derived from the fixture's version instead of hardcoded `@v0`.
- [x] 1.3 `registrytest.BuildModuleFile` / `BuildCatalog`: accept the core major (or full version) as input; keep v1 defaults for now so the suite stays green.
- [x] 1.4 Run `task test` — all green with defaults still on v1.

## 2. Address-seam adaptations (green pre-step; three re-land, one new)

- [x] 2.1 `materialize` pull (re-land): split a major-suffixed `#registry` key before composing the load ID — `…@v2@v2.0.0-alpha.2` is rejected as an invalid ID; major-free keys pass through unchanged.
- [x] 2.2 `materialize` enumeration (new): scope the published-version list to the key's major when the subscription key carries `@vN`, before `filterVersions` runs; a major-free key enumerates the whole repo as today. Unit-test against a mixed-major published list modelled on the real repo (stable v0 tags + v1/v2 prereleases): a `@v2` key with no filter MUST select the highest v2 alpha, never `v0.6.0`; a major-free key MUST keep selecting `v0.6.0`.
- [x] 2.3 `helper/synth` (re-land): a v2 `modulePath` is the full major-suffixed import path — pass it through verbatim instead of composing parent path + snake leaf + version major; v1 inputs byte-identical.
- [x] 2.4 `materialize/cache.Key` (re-land): the scalar subscription `version` joins the normalized projection so two v2 platforms differing only in it get distinct keys; v1 inputs byte-identical.
- [x] 2.5 Run `task test` — all green, defaults still on v1.

## 3. The atomic flip (one commit)

- [ ] 3.1 `schema/loader.go`: `DefaultSchemaModule = "opmodel.dev/core@v2"`; `registrytest.defaultCoreVersion = "v2.0.0-alpha.4"` (raise to alpha.5 if cut by then); writer defaults from 1.3 flip to v2.
- [ ] 3.2 `testdata/cue.mod/module.cue` + `testdata/synth/fixture.cue`: core v2 dep, v2 metadata shape (`modulePath` with `@vN`, snake leaf name).
- [ ] 3.3 `testdata/modules/web_app/`: core v2; catalog dep re-pointed to the real `opmodel.dev/catalogs/opm@v2` at the newest consolidated tag (≥ `v2.0.0-alpha.2`); imports move to the D49 versioned packages (`…/blueprints/v1beta1`, `…/traits/v1beta1`, `…/resources/v1beta1`); metadata to v2 shape.
- [ ] 3.4 `modules/opm_platform/platform.cue` + `cue.mod`: core v2; subscription re-keyed to the major-suffixed `opmodel.dev/catalogs/opm@v2`; `filter: range:` → required scalar `version:` pinned to the same newest tag as 3.3 (transitional invariant 1 — the pin MUST track the newest tag on the v2 line until library-acquire-and-subscription makes it load-bearing).
- [ ] 3.5 Kernel flow/synth tests (`flow_integration_test`, `flow_synth_*`, `synth_test`, `synth_platform_test`, `integration_*`): `CoreVersion` pins and inline deps text to the 3.1 version; fixture bodies to v2 shape; in-process `registrytest` catalog fixtures stay hermetic; platform-driven filter tests removed (v2 has no filter) while Go-side filter semantics stay pinned in `filter_test.go` until library-acquire-and-subscription deletes them.
- [ ] 3.6 Run `task test` — mainline suite green on v2; run the registry-touching flow test against GHCR (`OPM_FLOW_TEST_FORCE=1`) so the real-catalog materialization path is exercised, not skipped.

## 4. Kept tests + docs (green post-steps)

- [ ] 4.1 Verify untouched-and-green: `composed_open_test` (v0.5.2 canary), `instance_integration_test` v1-pinned family incl. the library#31 regression (deep `blueprints/workload` import at `:154` stays — recorded D42 deviation, not reopened by D49), `docs/design/repro-hidden-field/`.
- [ ] 4.2 Doc-comment sweep: `kernel.go`, `schema/doc.go`, `schema/cache.go`, `materialize/doc.go`, `enumerate.go` (the "regardless of major" comment now describes only major-free keys), `schematest`/`registrytest` package docs — `core@v1` citations → `core@v2`.
- [ ] 4.3 `CLAUDE.md` (§ CUE/registry notes citing `core@v1` and `catalogs/opm@v0`) and `Taskfile.yml` fixture comments updated; no vet-skip guard or `testing.opmodel.dev` routing appears anywhere.
- [ ] 4.4 `MIGRATIONS.md` `## Unreleased — Breaking` entry `### Changed — \`library-core-retarget\``: default schema module is now core v2; pin `OCILoader{Module: "opmodel.dev/core@v1.0.0-alpha.1"}` to stay on v1; rollback = reverse rewrite to `v1.1.0-alpha.1`. PR carries `Migration: library-core-retarget`.

## 5. Verify & record

- [ ] 5.1 `task check` (fmt, vet, lint, test) clean; `task cue:check` clean against GHCR with no local registry running.
- [ ] 5.2 Confirm no behaviour change for v1 inputs: `git diff` over `opm/` touches only `schema/loader.go`'s constant, doc comments, the four seam sites (each guarded on v2-only input shapes), and `internal/` test plumbing.
- [ ] 5.3 Record back in `enhancements/0010/`: slice `library-core-retarget` → `done` with a `history` event; note any deviations from this redo's own plan.
