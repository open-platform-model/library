## Why

On 2026-10-02 and 2026-10-03 the owner went through the beta.1 kernel plan task by task and answered each task. Every answer has a short id (`a1` to `j5`). Those answers shaped most of the library, cli and opm-operator changes merged since then, and committed text cites them by id: "owner decision j4 (beta.1 walkthrough)", "owner decision c4 of the beta.1 walkthrough", "decision j4, kernel plan walkthrough 2026-10-03". The ids are written down nowhere a repo reader can open. The only record is a log outside every repository, so each such citation names a source no reader can find.

On 2026-10-05 the owner chose how to fix that: record the walkthrough decisions in one library ADR, and cite that ADR instead. This change writes the ADR and sweeps the library. The cli and opm-operator sweeps are separate changes in those repos, and they cite this ADR.

The library's own rule already asks for this. The `kernel-runtime` requirement "Each runtime contract has one home" says a committed file cites a source a reader can open, and today it lists "an owner decision by its walkthrough id" as one. That item is the gap: a walkthrough id cannot be opened. After this change, a walkthrough decision is cited through ADR-013.

## What Changes

- **ADR-013** (`adr/013-kernel-plan-walkthrough-decisions.md`, new). It records the walkthrough in the `adr/TEMPLATE.md` shape. Its Decision section holds one table row per decided task, `a1` to `j5`: the id, the decision in one or two plain sentences (the final answer where the owner replaced an earlier one), and where it landed, as PR numbers across library, cli, opm-operator, core, catalog_opm, enhancements and workspace. A decision that is decided but not yet implemented, in whole or in part, says so in its row. Tasks the walkthrough did not decide are left out: `a2` and `c3` (already done before the walkthrough) and the research items N1 to N8 (not walked). Process notes about how the changes were run are left out. The citation form is `ADR-013, decision j4`.
- **Library citation sweep.** Every committed file outside `openspec/changes/` that cites a walkthrough id cites ADR-013 instead. At `origin/main` 88afdfb these are `AGENTS.md` (consumer build, API diff, "Where a statement lives"), `.github/workflows/consumer-build.yml`, `.tasks/consumer-build.sh`, `adr/012-matching-stays-in-the-library-glue.md` (Status), `opm/catalog/provides_parity_test.go` (a test comment), and the main specs `api-diff-check` (Purpose and two `Source:` lines), `consumer-build` (four `Source:` lines), `kernel-runtime` (the one-home requirement) and `kubernetes-tier` (two `Source:` lines).
- **Archive citation sweep.** The archived changes under `openspec/changes/archive/` cite walkthrough ids on 79 lines (43 in the explicit form, 36 bare). Each citation cites ADR-013 instead; only the citation words change (design.md, decision 2).
- **The one-home rule names ADR-013.** In `AGENTS.md` § Where a statement lives and the `kernel-runtime` requirement "Each runtime contract has one home", "an owner decision by its walkthrough id" becomes a decision ADR-013 records, cited as `ADR-013, decision <id>`. The requirement gains two scenarios: a walkthrough id resolves through ADR-013, and no committed file outside `openspec/changes/` cites a walkthrough id without ADR-013.

## Not in this change

- The cli and opm-operator sweeps. Each is its own change in its repo, and cites `library` ADR-013.
- Commit messages and PR bodies already merged. They keep their bare ids, and ADR-013's table resolves them. (`CHANGELOG.md` cites no walkthrough id.)
- Wiring the walkthrough-id check into `task check`. It joins the ungated one-home checks that library#198 tracks, and that issue should also cover it.
- Numbered owner selections from the release-cascade and security-pass planning ("owner decision 24", "owner decision 29", "owner selection 9", "owner selection 28" to "owner selection 30") in `AGENTS.md`, `.github/workflows/release.yml`, `.tasks/cascade/wiring-check.sh` and the `cascade-wiring`, `deps-cascade`, `release-pipeline` and `workflow-hardening` specs. They are not walkthrough ids, ADR-013 does not record them, and the owner's answer named only the walkthrough. They are listed as a follow-up.
- Owner answers given after the walkthrough (2026-10-05: required `#config` values, typed author-defect resolution errors, the adopt fight on an in-inventory object). Each lands with its own change and cites its issue or enhancement decision (library#211, `0021:D8:R12`, `0012:D8`). ADR-013 names them only in the row of the walkthrough decision each one refines.
- Any Go code, exported identifier, error text or test behaviour.

## Classification

**No release.** One new ADR, Markdown (archived changes included), comment lines in a workflow, a shell script and one Go test file, and spec text. No exported symbol, signature or behaviour changes. The commits are `docs(adr)`, `docs` and `chore(openspec)`, which release-please hides. The later PR title is `docs(adr): record the kernel-plan walkthrough decisions in ADR-013`. Principle VII: one new ADR, which adds a written record and no code.

## Order

This change merges last among the round-8 library changes (`refuse-unset-required-config`, `type-author-resolution-errors`, `refuse-an-object-another-instance-adopted`). Before its PR merges it merges `origin/main`, re-runs the sweep over whatever those changes landed, and fills ADR-013's "Landed" column with the PR numbers merged by then.

## Downstream consumers

- **cli** and **opm-operator**: no code change. Their own sweeps, separate changes, replace their walkthrough-id citations with `library` ADR-013 (`adr/013-kernel-plan-walkthrough-decisions.md` in the library repo).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `kernel-runtime`: the one-home requirement names ADR-013 as the source for a walkthrough decision, and adds a scenario that resolves an id through ADR-013 and one that finds no bare walkthrough id.
- `api-diff-check`: the two `Source:` lines cite `ADR-013, decision j4`. The Purpose text is edited in place, because a delta cannot carry it (design.md, decision 4).
- `consumer-build`: the four `Source:` lines cite `ADR-013, decision j4`.
- `kubernetes-tier`: the deletion-plan requirement cites `ADR-013, decision j5` and the readiness requirement cites `ADR-013, decision f5`.

## Impact

- `adr/013-kernel-plan-walkthrough-decisions.md` (new), `adr/012-matching-stays-in-the-library-glue.md` (Status line).
- `AGENTS.md`, `.github/workflows/consumer-build.yml`, `.tasks/consumer-build.sh` (comment line), `opm/catalog/provides_parity_test.go` (comment lines).
- `openspec/specs/api-diff-check/spec.md` (Purpose, in place); four main specs through archive.
- Citation words in archived changes under `openspec/changes/archive/` (about 28 files).
