# Tasks: closedness-canary-and-filter-tests

## 1. Closedness canary

- [x] 1.1 Create `opm/internal/cueregression/closedness_test.go` with the trigger-form and hoisted-form fixtures as embedded string constants, transcribed verbatim from `docs/design/cue-closedness-regression-alpha2.md`
- [x] 1.2 Trigger-form test: `cuecontext.New().CompileString` + `Validate`; assert error present, matching `field not allowed` at `x.out.a.b`; on clean validation FAIL with the re-evaluation message including the running `cuelang.org/go` version (`debug.ReadBuildInfo`)
- [x] 1.3 Hoisted-form test: assert clean validation; comment marks it a CUE-bump release gate
- [x] 1.4 Confirm both run offline under `go test -short ./opm/internal/cueregression`
- [x] 1.5 (added during apply, user-approved) Delete the superseded `opm/kernel/cue_closedness_regression_test.go` (PR #40) — the new package strictly replaces it — and repoint its references (`CLAUDE.md`, `docs/design/cue-closedness-regression-alpha2.md`, `docs/design/repro-cue-closedness/repro.cue`)

## 2. filterVersions pre-release range rows

- [x] 2.1 Add table rows to `opm/materialize/filter_test.go`: `Range: ">=0.6.0-dev.0 <0.7.0"` over `[v0.5.0, v0.6.0-dev.1, v0.6.0]` → selects dev.1 + 0.6.0, excludes 0.5.0
- [x] 2.2 Add the OQ18-shape row: `Range: ">=1.0.0-alpha"` over `[v1.0.0-alpha, v1.0.0-alpha.1, v1.0.0-dev.1784212239.g0c11c12]` → all selected
- [x] 2.3 Add a Materialize-level assertion that the resolved version from that survivor set is the `-dev.*` tag (dev out-sorts alpha); reference 0006 OQ18 in the test comment (`TestMaterialize_PrereleaseRangeResolvesDevOverAlpha`; required deriving the fixture module major from the version in `registrytest.addCatalogs`, previously hardcoded `@v0`)

## 3. Fixture hygiene

- [x] 3.1 Re-verify zero consumers (`grep -rn "testdata/platform\|testdata/out.cue" --include="*.go" .`), then delete `testdata/platform/` and `testdata/out.cue`; stop and record instead if a consumer surfaces (Go grep clean; only other mention is historical `MIGRATIONS.md` changelog prose, left as-is; `testdata/out.cue` was untracked)

## 4. Verification

- [x] 4.1 `task check` green; full suite unaffected
- [x] 4.2 Sync/archive per openspec flow (delta synced into `openspec/specs/platform-materialization/spec.md`; new `openspec/specs/cue-regression-canary/spec.md` created; archived 2026-07-23)
