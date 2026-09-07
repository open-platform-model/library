## Why

Three facts on `main` once `cue-owned-verdicts` lands:

1. **`opm/compat` has no runtime consumer.** Its callers are two cli sites: the publish gate (`cli/internal/publish/compat.go:133,287`, `ParseLevel` and `CheckAtLevel`) and template version selection (`cli/internal/scaffold/scaffold.go:109`, `HighestStable`). The operator never imports it, and `catalog_opm`'s release workflow runs `opm catalog registry check --compat` through the pinned cli binary. The one in-library use, the apiVersion ladder sort in `opm/internal/renderstage/alternatives.go`, moves into the CUE glue with `cue-owned-verdicts`, after which the package is a leaf nothing under `opm/` imports.
2. **Every comparator change costs two repos.** Because the comparator ships in the library, a fix to the walk or a new publish-gate rule is a library release followed by a cli re-pin. Enhancement 0020 (draft, contract promotion and retirement) plans four more cross-build rules for the same package and orders their delivery "library before cli" for exactly that reason. With the comparator beside its only caller, each becomes one cli PR.
3. **The placement is recorded design intent, so this is a deliberate reversal.** Enhancement 0011 D9's rationale put the comparator in the library "so `opm catalog publish`, `opm catalog registry check --compat` and any CI action share one implementation", and D23 notes that `HighestStable` "stays in `library/opm/compat`". Every consumer named there is the cli binary. The owner chose the move on 2026-09-07 over two alternatives (keep the package and leave placement to 0020's revision; move it to the cli's importable `pkg/` tree). The draft entry 0020 is amended to the new landing point and 0011's delivery log records the relocation; no accepted decision text changes.

This is slice 5 of the eight-slice simplification plan reviewed on 2026-09-05, the half that waited for slice 4. `core.Compiled` and `Source.WriteTo` landed with slices 4 and 2. Pre-GA, so the cli re-pins once for the whole wave.

**Scope statement (Principle VIII).** One package leaves the library, one lint list and three documents lose a line each, one spec capability is removed and one boundary requirement is restated. No behaviour changes anywhere: the cli PR carries the files in verbatim.

## What Changes

**`opm/compat` (BREAKING, `refactor(compat)!:`):**

- **BREAKING** `opm/compat` is deleted: `Check`, `CheckAtLevel`, `Violation` and its `Kind*` constants, `Level` (`LevelAlpha`, `LevelBeta`, `LevelGA`, `Enforced`, `String`), `ParseLevel`, `CompareAPIVersions`, `HighestStable`, and the three test files. The cli receives the comparator and the ladder as `cli/internal/compat` (same package name, same files, same tests) and `HighestStable` as an unexported selector in `cli/internal/scaffold`, its only caller.
- The library keeps `github.com/Masterminds/semver/v3`: `opm/internal/renderstage` (`skew.go`, `promote.go`) uses it. The cli promotes the same module from an indirect to a direct requirement.

**Lint and docs:**

- `.golangci.yml`: the `kernel-never-imports-helper` file list drops `**/opm/compat/**` (`**/opm/core/**` leaves with `cue-owned-verdicts`).
- `CONSTITUTION.md` III drops the `opm/compat/` line; `CLAUDE.md` drops the `compat/` layout line and the `compat` commit scope; `README.md` drops the `compat/` layout line and names the boundary list without `opm/compat`.

**Enhancements (a separate commit in `enhancements/`, tasked here so it is not forgotten):**

- 0020 (draft): "Cross-Repo Coordination" item 3 and the four `schemas/target.cue` comments naming `library/opm/compat` retarget to `cli/internal/compat`; the "library before cli" ordering for the comparator collapses to cli-only.
- 0011 (accepted): `enhancement.yaml` in this change declares D9 so the archive step logs the relocation in `0011/delivery.yaml` with a summary naming the new home.

**Not in this change:** any edit to the comparator, the ladder or the selector; the cli's `publish` package beyond one import path; the cli's OpenSpec record for the moved capability (the cli PR adds it); the slice 4 shapes in `opm/errors`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `catalog-compatibility`: every requirement is removed from the library; the capability moves to the cli with the code.
- `helper-packages`: the boundary requirement's package list no longer names `opm/compat` (nor `opm/core`, deleted by `cue-owned-verdicts`).

## Impact

**SemVer:** MAJOR on the alpha line (Principle VI): an exported package is deleted. Pre-GA, so no migration fragment (ADR-004); the cli migrates in the same PR wave as slices 2 and 4.

**Downstream migration cost, `cli` (3 files edited, 4 files added):**

- `internal/compat/{compat,level}.go` and `{compat,level}_test.go`: copied verbatim from the library; the package doc's consumer list drops "library-matching" (gone since `cue-owned-verdicts`).
- `internal/scaffold/scaffold.go:17,109`: the `compat` import goes; `highestStable(versions)` is declared in the package with `predecessor.go`'s doc comment, and its four test cases join the scaffold tests.
- `internal/publish/compat.go:18`: the import path becomes `github.com/open-platform-model/cli/internal/compat`.
- `go.mod`: `github.com/Masterminds/semver/v3` becomes a direct requirement.
- The cli's OpenSpec gains the `catalog-compatibility` capability (the three requirements this change removes, restated against the cli package) in the same PR.

**Downstream migration cost, `opm-operator`:** none; it has no import.

**`catalog_opm`, `modules`, `core`:** none; `catalog_opm` CI calls the cli binary.

**Library:** `opm/compat/**` (deleted), `.golangci.yml`, `CONSTITUTION.md`, `CLAUDE.md`, `README.md`, `openspec/specs/{catalog-compatibility,helper-packages}`. Tests: the three compat test files move with the package; no other library test references it once slice 4 has removed `TestAlternatives`.

**Complexity justification (Principle VII):** net deletion of 485 non-test and 508 test lines from the library and one package from its SemVer surface; the cli gains the same lines under `internal/` and nothing new.
