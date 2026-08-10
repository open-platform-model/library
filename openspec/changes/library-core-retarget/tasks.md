# Tasks — library-core-retarget

> Staged so every commit before and after the atomic flip is independently green. The flip itself (3.x) is one commit — `DefaultSchemaModule`, `defaultCoreVersion`, and the mainline fixtures move together or the suite fails. Task-count note: the crossing is atomic by nature; 0010's plan sized it as one slice, and the proposal records why it doesn't split further.

## 1. Major-aware fixture writers (green pre-step)

- [x] 1.1 `registrytest.addCatalogs`: derive the core dep line's major and the emitted `import c "opmodel.dev/core@vN"` from `coreVersionOr(f.CoreVersion)` — emitted import and declared dep MUST agree. Existing v1-pinned callers unchanged.
- [x] 1.2 `registrytest.addModules`: same derivation for the deps line; module-path major suffix derived from the fixture's version instead of hardcoded `@v0`.
- [x] 1.3 `registrytest.BuildModuleFile` / `BuildCatalog`: accept the core major (or full version) as input; keep v1 defaults for now so the suite stays green.
- [x] 1.4 Run `task test` — all green with defaults still on v1.

## 2. Fixture catalog + publish plumbing (green pre-step)

- [ ] 2.1 Author the on-disk fixture catalog under the `testing.opmodel.dev` prefix beside `modules/opm_platform`: core v2, one transformer + the resources/traits `web_app` demands, a flat `blueprints/` package (D42), `apiVersion!`/`catalogVersion!`/authored transformer `fqn!`, explicit `metadata.labels` mirroring `matchLabels` (transitional invariant 2, see design).
- [ ] 2.2 `registrytest`: add a disk-tree publish helper (read a fixture module directory into the `modregistrytest` MapFS) so tests serve the on-disk catalog in-process; route the `testing.opmodel.dev` prefix to the in-process host in the test `CUE_REGISTRY` mapping.
- [ ] 2.3 `cue vet` the fixture catalog standalone (v2 core resolves from GHCR); wire it into `task cue:*` fixture checks; exactly one version is ever published per test registry (transitional invariant 1).

## 3. The atomic flip (one commit)

- [ ] 3.1 `schema/loader.go`: `DefaultSchemaModule = "opmodel.dev/core@v2"`; `registrytest.defaultCoreVersion = "v2.0.0-alpha.4"`; writer defaults from 1.3 flip to v2.
- [ ] 3.2 `testdata/cue.mod/module.cue` + `testdata/synth/fixture.cue`: core v2 dep, v2 metadata shape (`modulePath` with `@vN`, snake leaf name).
- [ ] 3.3 `testdata/modules/web_app/`: core v2; catalog dep re-pointed to the fixture catalog; `components.cue` blueprint import flattened (D42 site #1); metadata to v2 shape.
- [ ] 3.4 `modules/opm_platform/platform.cue` + `cue.mod`: core v2; subscription re-keyed to the fixture catalog path; `filter: range:` → required scalar `version:`.
- [ ] 3.5 Kernel flow/synth tests (`flow_integration_test`, `flow_synth_*`, `synth_test`, `synth_platform_test`): `CoreVersion` pins and inline deps text (`flow_synth_imported_test.go:136`) to `v2.0.0-alpha.4`; fixture bodies to v2 shape; catalog references from `opmodel.dev/catalogs/opm` to the fixture catalog.
- [ ] 3.6 Run `task test` — mainline suite green on v2.

## 4. Kept tests + docs (green post-steps)

- [ ] 4.1 Verify untouched-and-green: `composed_open_test` (v0.5.2 canary), `instance_integration_test` v1-pinned family incl. the library#31 regression (deep `blueprints/workload` import at `:154` stays — recorded D42 deviation), `docs/design/repro-hidden-field/`.
- [ ] 4.2 Doc-comment sweep: `kernel.go`, `schema/doc.go`, `schema/cache.go`, `materialize/doc.go`, `schematest`/`registrytest` package docs — `core@v1` citations → `core@v2`.
- [ ] 4.3 `CLAUDE.md` (§ CUE/registry notes citing `core@v1`) and `Taskfile.yml` fixture/publish comments updated.
- [ ] 4.4 `MIGRATIONS.md` `## Unreleased — Breaking` entry `### Changed — \`library-core-retarget\``: default schema module is now core v2; pin `OCILoader{Module: "opmodel.dev/core@v1.0.0-alpha.1"}` to stay on v1; rollback = reverse rewrite to `v1.1.0-alpha.1`. PR carries `Migration: library-core-retarget`.

## 5. Verify & record

- [ ] 5.1 `task check` (fmt, vet, lint, test) clean.
- [ ] 5.2 Confirm no behaviour change outside the default flip: `git diff` over `opm/` touches only `schema/loader.go`'s constant, doc comments, and `internal/` test plumbing.
- [ ] 5.3 Record back in `enhancements/0010/`: slice `library-core-retarget` → `done` with `openspec_ref`, a `history` event, and the D42 one-site deviation noted (plan concern says two sites; one moved).
