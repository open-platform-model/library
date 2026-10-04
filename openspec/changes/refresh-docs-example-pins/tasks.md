Depends on: nothing unmerged (`add-deps-cascade-task` is on `main`). Independent of `.github` `add-release-cascade-workflows` and library `join-release-cascade` (Phase 3 wiring contract §1). Merge before library's receiver goes live in Phase 4. One section, one commit (design.md D4).

## 1. Docs examples follow the loader's core

- [ ] 1.1 `docs/getting-started.md:51`: `Module: "opmodel.dev/core@v2.0.0-beta.2"`. `docs/getting-started.md:56`: `// → the version after the "@" in Module` (design.md D5). `AGENTS.md:350`: `schema.OCILoader{Module: "opmodel.dev/core@v2.0.0-beta.2"}`. Verify: `grep -n 'core@v' docs/getting-started.md AGENTS.md` names only `v2.0.0-beta.2` and the `AGENTS.md:341` placeholder.
- [ ] 1.2 `.tasks/cascade/cascade.sh` C4 (`:288-296`): when `D != D0` and the file is not frozen for `opmodel.dev/core@v2`, rewrite every same-major `opmodel.dev/core@<v>` in `docs/getting-started.md` and `AGENTS.md` to `D`, with the tightened matcher. Die when a same-major literal other than `D` remains. Then warn, as today, for each name that differs from `D`, using the same version grammar (design.md D2, D3, D6). Update the phase-C header comment (`:14`) and the C4 comment. Verify: `shellcheck .tasks/cascade/*.sh` clean.
- [ ] 1.3 `.tasks/cascade/test.sh`:
  - `set_older core|both` also rewrites the loader's current core in the two docs to the older core.
  - S2: drop the docs-warning assertion (`:325`) and assert that no warning names either docs file.
  - S1: assert that no warning names either docs file.
  - New offline `s11_lagging_docs`: an older same-major core in `docs/getting-started.md`, core current, expect exit 3, an empty status and a warning naming the file. Add it to the offline list in the header comment and to the run list.
  - Verify: `CASCADE_TEST_SET=offline task -x deps:cascade:test` passes, including S11. The full `task -x deps:cascade:test` passes (network), with S2 green.
- [ ] 1.4 Spot checks:
  - On this branch, `CASCADE_RESOLVER=<workspace>/.github/.github/scripts/cascade/cascade-resolve.sh task -x deps:cascade` exits 3, and its warnings file names neither docs file.
  - In a scratch copy with the loader and both docs set to `v2.0.0-beta.1`, the same run exits 0, and `git diff` shows the loader and the two example lines at `v2.0.0-beta.2` beside the test-tree edits.
  - Record both results in the PR body.
- [ ] 1.5 `AGENTS.md`, "Release cascade task" (`:301`): one clause saying that a core move also rewrites the core examples in `docs/getting-started.md` and `AGENTS.md`, and that anything left over is warned about.
- [ ] 1.6 `openspec validate refresh-docs-example-pins --strict` passes. `task check` is green. Then commit `ci(cascade): keep the docs example core pins on the loader`.
