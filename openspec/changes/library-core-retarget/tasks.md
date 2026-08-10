# Tasks — library-core-retarget

> Staged so every commit before and after the atomic flip is independently green. The flip itself (3.x) is one commit — `DefaultSchemaModule`, `defaultCoreVersion`, and the mainline fixtures move together or the suite fails. Task-count note: the crossing is atomic by nature; 0010's plan sized it as one slice, and the proposal records why it doesn't split further.

## 1. Major-aware fixture writers (green pre-step)

- [x] 1.1 `registrytest.addCatalogs`: derive the core dep line's major and the emitted `import c "opmodel.dev/core@vN"` from `coreVersionOr(f.CoreVersion)` — emitted import and declared dep MUST agree. Existing v1-pinned callers unchanged.
- [x] 1.2 `registrytest.addModules`: same derivation for the deps line; module-path major suffix derived from the fixture's version instead of hardcoded `@v0`.
- [x] 1.3 `registrytest.BuildModuleFile` / `BuildCatalog`: accept the core major (or full version) as input; keep v1 defaults for now so the suite stays green.
- [x] 1.4 Run `task test` — all green with defaults still on v1.

## 2. Fixture catalog + publish plumbing (green pre-step)

- [x] 2.1 Author the on-disk fixture catalog under the `testing.opmodel.dev` prefix beside `modules/opm_platform`: core v2, one transformer + the resources/traits `web_app` demands, a flat `blueprints/` package (D42), `apiVersion!`/`catalogVersion!`/authored transformer `fqn!`, explicit `metadata.labels` mirroring `matchLabels` (transitional invariant 2, see design). (Landed as `modules/opm_catalog` with the four transformers the flow tests pair against: deployment, service, http-route, configmap.)
- [x] 2.2 `registrytest`: add a disk-tree publish helper (read a fixture module directory into the `modregistrytest` MapFS) so tests serve the on-disk catalog in-process; route the `testing.opmodel.dev` prefix to the in-process host in the test `CUE_REGISTRY` mapping.
- [x] 2.3 `cue vet` the fixture catalog standalone (v2 core resolves from GHCR); wire it into `task cue:*` fixture checks (auto-discovered via `modules/*` glob); exactly one version is ever published per test registry (transitional invariant 1).
- [x] 2.4 `materialize/pull.go`: split a subscription key's `@vN` major off before composing the load ID (`ast.SplitPackageVersion`) — v2 keys carry the major and the composed `…@vN@vX.Y.Z` form is unloadable; major-free v1 keys pass through unchanged. Discovered during apply; see design "Subscription-key major handling at pull". `task test` stays green pre-flip.

## 3. The atomic flip (one commit)

- [x] 3.0 v2 identity-shape seams (discovered during apply, see design): `synth.moduleImportPath` passes a major-suffixed `modulePath` through verbatim (D1) instead of composing parent+leaf; `cache.Key` normalizes the v2 scalar `version` alongside the v1 filter fields. Both strictly input-extending — v1 inputs keep byte-identical behaviour.
- [x] 3.1 `schema/loader.go`: `DefaultSchemaModule = "opmodel.dev/core@v2"`; `registrytest.defaultCoreVersion = "v2.0.0-alpha.4"`; writer defaults from 1.3 flip to v2 (v2 catalog bodies key contracts by `ContractAPIVersion`, author `apiVersion`/`catalogVersion`/`fqn`).
- [x] 3.2 `testdata/cue.mod/module.cue` + `testdata/synth/fixture.cue`: core v2 dep, v2 metadata shape (`modulePath` with `@vN`, snake leaf name).
- [x] 3.3 `testdata/modules/web_app/`: core v2; catalog dep re-pointed to the fixture catalog; `components.cue` blueprint import flattened (D42 site #1); metadata to v2 shape (`web_app` snake name, full `modulePath`, version `1.0.0`).
- [x] 3.4 `modules/opm_platform/platform.cue` + `cue.mod`: core v2; subscription re-keyed to the fixture catalog path (major-suffixed, per v2 `#ModulePathType`); `filter: range:` → required scalar `version:`.
- [x] 3.5 Kernel flow/synth tests (`flow_integration_test`, `flow_synth_*`, `synth_test`, `synth_platform_test`) plus the materialize/loader/cache harnesses: v2 pins and fixture bodies; subscription bodies re-keyed with `version:`; contract FQNs apiVersion-keyed; flow tests serve the fixture catalog from the in-process registry (`NewDiskRegistry`). The two platform-driven filter tests (range/allow/deny, prerelease-range) were removed — v2 cannot author a `filter`; Go-side semantics stay pinned in `filter_test.go`. `TestPlatform_SubscriptionWithFilterRange` became `TestPlatform_SubscriptionFilterRejectedByV2Schema`. `Taskfile.yml` `cue:vet` skips modules with `testing.opmodel.dev` deps when the local registry is down (CI stays GHCR-only; the Go flow tests cover them in-process).
- [x] 3.6 Run `task test` — mainline suite green on v2 (verified incl. flow tests, `task cue:check`/`cue:tidy` with and without the local registry, and `cue:test:flow:inspect` against a locally-published fixture catalog).

## 4. Kept tests + docs (green post-steps)

- [x] 4.1 Verify untouched-and-green: `composed_open_test` (v0.5.2 canary), `instance_integration_test` v1-pinned family incl. the library#31 regression (deep `blueprints/workload` import at `:154` stays — recorded D42 deviation), `docs/design/repro-hidden-field/`. (`git diff main` over those paths is empty; all pass.)
- [x] 4.2 Doc-comment sweep: `kernel.go`, `schema/doc.go`, `schema/cache.go`, `materialize/doc.go`, `schematest`/`registrytest` package docs — `core@v1` citations → `core@v2`. (Plus `README.md` and `docs/getting-started.md`, which also cite the default.)
- [x] 4.3 `CLAUDE.md` (§ CUE/registry notes citing `core@v1`) and `Taskfile.yml` fixture/publish comments updated (drift fixture list emptied — no fixture consumes the real catalog until catalogs-republish).
- [x] 4.4 `MIGRATIONS.md` `## Unreleased — Breaking` entry `### Changed — \`library-core-retarget\``: default schema module is now core v2; pin `OCILoader{Module: "opmodel.dev/core@v1.0.0-alpha.1"}` to stay on v1; rollback = reverse rewrite to `v1.1.0-alpha.1`. PR carries `Migration: library-core-retarget`.

## 5. Verify & record

- [x] 5.1 `task check` (fmt, vet, lint, test) clean — golangci-lint reports 0 issues, full suite green.
- [x] 5.2 Confirm no behaviour change outside the default flip: `git diff` over non-test `opm/` touches only `schema/loader.go`'s constant, the three strictly input-extending v2-seam adaptations recorded in design (`materialize/pull.go`, `helper/synth/render.go`, `materialize/cache/key.go`), doc comments, and `internal/` test plumbing.
- [x] 5.3 Record back in `enhancements/0010/`: slice `library-core-retarget` → `done` with `openspec_ref: "library/library-core-retarget"`, a `history` event covering the landing, the D42 one-site resolution, the seam adaptations, the fixture catalog, and the two transitional invariants (branch `0010-library-core-retarget-done` in the enhancements repo).
