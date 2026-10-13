# CUE toolchain pin

Covers the versions that a CUE bump moves: the SDK in `go.mod`, the apimachinery floor, the declared
CUE language floor, the CUE CLI in CI, and the unfixed closedness regression.

Two independent knobs — do not conflate them:

- **SDK** — `cuelang.org/go` in `go.mod`, currently **`v0.17.1`**. Because Go uses MVS, every embedder (`cli`, `opm-operator`) resolves *at least* this version; the library effectively sets their CUE floor.
- **`k8s.io/apimachinery`** — in `go.mod` for the Kubernetes tier, currently **`v0.36.4`**. By the same MVS rule it is every embedder's apimachinery floor, as `cuelang.org/go` is their CUE floor. No bump policy is set here.
- **Declared `language.version`** — what the CUE modules here (`modules/opm_platform`, `testdata/**`, and the one Go constant `modversion.LanguageFloor` in `opm/internal/modversion`, which sets the render floor, the generated platform module and the `opm/internal/registrytest` fixtures) declare, currently **`v0.17.0`**. This is a *consumer* floor: a module declaring `vX` is rejected by every `cue` older than `vX`. Declare `v0.17.0` — the minimum enabling `cue.mod/local-module.cue` — not `v0.17.1`, which would lock out v0.17.0 tools for no gain.

**`v0.17.x` carries an unfixed evaluator closedness regression** (`docs/design/cue-closedness-regression-alpha2.md`). The pin is safe only because the catalog encodes the hoisted-guard workaround; `opm/internal/cueregression/closedness_test.go` is the canary pair that fails when upstream fixes it (trigger form) or when the workaround shape breaks on a CUE bump (hoisted form). Do not treat a passing suite as evidence the bug is gone.

**CUE CLI in CI** is a third thing a CUE bump checks: `v0.17.1` in three places, `.github/workflows/cue.yml` (`setup-cue` `version`), `.github/workflows/cascade-task.yml` (the same), and the `cue-version` default of the reusable `cascade-receive.yml` at the pinned `.github` SHA, which `deps-cascade.yml` does not override (wiring contract §5.2). Move the first two in the bump PR; if the reusable default lags, raise it in `.github` and move the cascade pin (`RELEASING.md` "Moving the cascade pin").
