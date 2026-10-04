Depends on: nothing unmerged (`add-deps-cascade-task` is on `main`). Independent of `.github` `add-release-cascade-workflows` and library `join-release-cascade` (Phase 3 wiring contract §1), but certain to conflict with the latter in `AGENTS.md:301` (proposal.md, Impact). Merge in the same review round as the workspace `RELEASING.md` amendment (proposal.md, Impact), and before library's receiver goes live in Phase 4. Section 1 is one commit (design.md D4).

## 1. Docs examples follow the loader's core

- [x] 1.1 `docs/getting-started.md:51`: `Module: "opmodel.dev/core@v2.0.0-beta.2"`. `docs/getting-started.md:56`: `// → the version after the "@" in Module` (design.md D5). `AGENTS.md:350`: `schema.OCILoader{Module: "opmodel.dev/core@v2.0.0-beta.2"}`. Verify: `grep -nE 'core@v[0-9]+\.' docs/getting-started.md AGENTS.md` lists only the two `v2.0.0-beta.2` examples and the `AGENTS.md:345` placeholder.
- [x] 1.2 `.tasks/cascade/cascade.sh` C4 (`:288-296`): check `D` against `SEMVER_RE`. When `D != D0` and the file is not frozen for `opmodel.dev/core@v2`, rewrite with `sed -E` the version in every `Module: "opmodel.dev/core@<same-major release>"` literal of `docs/getting-started.md` and `AGENTS.md` to `D`, the closing quote part of the match (design.md D3). Die when an anchored same-major example other than `D` remains (an internal assert, like C1's). Then warn for each release named anywhere in the file that differs from `D`, with the file-wide matcher that also catches build metadata (design.md D2, D3, D6). Update the phase-C header comment (`:14`) and the C4 comment. Verify: `shellcheck .tasks/cascade/*.sh` clean.
- [x] 1.3 `.tasks/cascade/test.sh`:
  - `set_older core|both` also rewrites the examples' current core in the two docs to the older core.
  - S2: drop the docs-warning assertion (`:325`); set the example in `docs/getting-started.md` to `v2.0.0-alpha.12`, older than `older.tsv`'s core, so `same_as_base` proves the catch-up; assert that no warning names either docs file.
  - S4: before `set_older`, add a prose line to `AGENTS.md` naming `opmodel.dev/core@v1.0.0` and `opmodel.dev/core@v2.0.0-alpha.12`; also append an anchored `Module: "opmodel.dev/core@v1.0.0"` line, and freeze `docs/getting-started.md` for `opmodel.dev/core@v2`; assert `AGENTS.md` ends equal to that setup copy except the example (back at the current core), that `docs/getting-started.md` is byte-unchanged after `set_older`, that both prose releases, the v1 example and the frozen file's older example are warned about, and leave both docs out of the "other files" diff. `set_example` rewrites only examples of the target version's major.
  - New offline `s11_lagging_docs`: an older same-major core in the `docs/getting-started.md` example, core current, expect exit 3, an empty status and a warning naming the file. Add it to the offline list in the header comment and to the run list.
  - Verify: `CASCADE_TEST_SET=offline task -x deps:cascade:test` passes, including S11. The full `task -x deps:cascade:test` passes (network), with S2 and S4 green.
- [x] 1.4 `.github/workflows/cascade-task.yml`: add `docs/getting-started.md` and `AGENTS.md` to the `pull_request` paths, since S2 and S4 now read them. Verify: actionlint clean.
- [x] 1.5 Spot checks:
  - On this branch, `CASCADE_RESOLVER=<workspace>/.github/.github/scripts/cascade/cascade-resolve.sh task -x deps:cascade` exits 0 or 3 (0 while the published opm catalog v4.5.2 is newer than the pinned v4.5.1), and its warnings file names neither docs file.
  - In a scratch copy with only the loader and both examples set to `v2.0.0-beta.1`, and a `.cascade-hold` entry `{pin: opmodel.dev/catalogs/opm@v4, max: v4.5.1}` so a newer published catalog cannot move, the same run exits 0, and `git diff` shows exactly the loader line and the two example lines at `v2.0.0-beta.2`.
  - Record both results in the PR body.
- [x] 1.6 `AGENTS.md`, "Release cascade task" (`:301`): one clause saying that a core move also rewrites the `Module: "…"` core examples in `docs/getting-started.md` and `AGENTS.md`, and that any other release named in either file is warned about.
- [x] 1.7 `openspec validate refresh-docs-example-pins --strict` passes. `task check` is green. Then commit `ci(cascade): keep the docs example core pins on the loader`.

## 2. Verify and archive

- [ ] 2.1 `openspec verify` (the repo's verify skill) reports no CRITICAL finding.
- [ ] 2.2 `openspec archive refresh-docs-example-pins -y`.
- [ ] 2.3 `openspec validate --all --strict` passes.
- [ ] 2.4 Commit `chore(openspec): archive refresh-docs-example-pins`.
