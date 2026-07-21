# Tasks: closedness-canary-and-filter-tests

## 1. Closedness canary

- [ ] 1.1 Create `opm/internal/cueregression/closedness_test.go` with the trigger-form and hoisted-form fixtures as embedded string constants, transcribed verbatim from `docs/design/cue-closedness-regression-alpha2.md`
- [ ] 1.2 Trigger-form test: `cuecontext.New().CompileString` + `Validate`; assert error present, matching `field not allowed` at `x.out.a.b`; on clean validation FAIL with the re-evaluation message including the running `cuelang.org/go` version (`debug.ReadBuildInfo`)
- [ ] 1.3 Hoisted-form test: assert clean validation; comment marks it a CUE-bump release gate
- [ ] 1.4 Confirm both run offline under `go test -short ./opm/internal/cueregression`

## 2. filterVersions pre-release range rows

- [ ] 2.1 Add table rows to `opm/materialize/filter_test.go`: `Range: ">=0.6.0-dev.0 <0.7.0"` over `[v0.5.0, v0.6.0-dev.1, v0.6.0]` → selects dev.1 + 0.6.0, excludes 0.5.0
- [ ] 2.2 Add the OQ18-shape row: `Range: ">=1.0.0-alpha"` over `[v1.0.0-alpha, v1.0.0-alpha.1, v1.0.0-dev.1784212239.g0c11c12]` → all selected
- [ ] 2.3 Add a Materialize-level assertion that the resolved version from that survivor set is the `-dev.*` tag (dev out-sorts alpha); reference 0006 OQ18 in the test comment

## 3. Fixture hygiene

- [ ] 3.1 Re-verify zero consumers (`grep -rn "testdata/platform\|testdata/out.cue" --include="*.go" .`), then delete `testdata/platform/` and `testdata/out.cue`; stop and record instead if a consumer surfaces

## 4. Verification

- [ ] 4.1 `task check` green; full suite unaffected
- [ ] 4.2 Sync/archive per openspec flow
