## 1. materialize — selection

- [x] 1.1 Add unexported `highestStable(published []string) string` to `opm/materialize/filter.go`: scan descending, return the first non-pre-release; fall back to the highest overall when no stable version exists.
- [x] 1.2 Switch the `f.isEmpty()` branch of `filterVersions` to return `highestStable(published)`.
- [x] 1.3 Update `filterVersions` / `subscriptionFilter` godoc: no-filter selects the highest **stable** version; pre-releases require `filter.allow`/`filter.range` opt-in.

## 2. fixtures

- [x] 2.1 Bump `testdata/modules/web_app/cue.mod/module.cue` catalog pin `v0.5.0 → v0.5.1` (latest stable; also clears `task cue:catalog:drift`).

## 3. tests

- [x] 3.1 Table-test `highestStable`: stable-only, mixed stable+pre-release, pre-release-only fallback, empty.
- [x] 3.2 Confirm `TestFlow_WebApp_OnOpmPlatform` + `TestFlow_WebApp_SynthPath_OnOpmPlatform` pass against the GHCR registry (`OPM_FLOW_TEST_FORCE=1`).

## 4. docs

- [x] 4.1 Add a `MIGRATIONS.md` entry describing the no-filter default change and the opt-in path for pre-releases.

## 5. validation gates

- [x] 5.1 `task fmt`
- [x] 5.2 `task vet`
- [x] 5.3 `task lint` (0 issues)
- [x] 5.4 `task test` (full suite green against the GHCR registry)
