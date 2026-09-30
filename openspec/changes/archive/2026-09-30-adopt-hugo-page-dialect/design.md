## Context

See proposal.md for why. `orchestration.md` (copied verbatim into this directory) is the shared brief for the whole change set. This change is row S5 there. Its section 4 is the dialect contract, and its section 4.1 holds the lint and the page-order helper this change runs.

Current state on `origin/main` (2026-09-30, 30f08c1):

- `docs/site/` holds seven pages in two sections: six in `diagnostics/`, one in `embedding/`. Each is a leaf page; library owns no section overview.
- Every page opens with the same front matter: `title`, `description`, `type`, then a two-line Starlight block on lines 5 and 6 (`sidebar:` and `  order: N`), then `---` on line 7.
- The lint from orchestration.md 4.1 (sha256 `dae9717a...`) reports seven findings, one per page, all `<file>:5: sidebar: is Starlight front matter; write weight: N`. It reports nothing else.
- `diagnostics/` is shared across repos. Library's orders are 20 to 25; cli adds `publish-refusals.md` (26) and `unprovided-contracts.md` (27); opm-operator adds `operator-conditions.md` (40). The section overviews (`site/content/docs/diagnostics/index.md`, `site/content/docs/embedding/index.md`) are site-owned in opmodel.dev, and change A converts them.
- Library has no `docs/STYLE.md`, and its `AGENTS.md` does not mention `docs/site`.

Since planning, `origin/main` moved to 0cde6ce: library PR 147 (3a53aed) added `diagnostics/colliding-contracts.md` (how-to, `sidebar:` with `order: 19`, no other lint finding). The lint there reports eight findings, and the S recipe applied to a copy passes the lint with an unchanged page order. Decision 6 converts the page.

## Goals / Non-Goals

**Goals:**

- The lint reports no finding for library's `docs/site`.
- The order of library's pages within each section does not change, and neither does their place among other repos' pages in `diagnostics/`.

**Non-Goals:**

- Rendering the pages. Change A renders every page; this change only lints.
- Any edit outside `docs/site/**`, except task ticks and the archive in this change directory.

## Research & Decisions

### 1. Keep every order number as it is

**Context**: Starlight's `sidebar.order` becomes Hugo's `weight`. The numbers could be kept or renumbered.
**Explored**: The weights of all pages in the shared `diagnostics/` section, across repos (listed in Context). Hugo reads `weight: 0` as unset, and the contract requires an integer of 1 or more; every library number is 10 or more.
**Decision**: Each page MUST get `weight:` with the exact number its `sidebar.order` had.
**Rationale**: Renumbering (for example 1 to 7) would move library's diagnostics after cli's 26 and 27 in the shared section. The same numbers also make the order diff from orchestration.md 4.1 an exact proof: an empty diff means every order became the same weight and no page appeared or vanished.

### 2. One section, no spike

**Context**: library's `openspec/config.yaml` asks for a spike first when design.md carries an unverified assumption.
**Explored**: The lint output, a grep of the seven pages for every other construct the contract forbids, and plan-final's dry run (the S recipe applied mechanically to copies of all six trees passes the lint with an unchanged page order).
**Decision**: The change is one section with one commit. It has no spike.
**Rationale**: Every edit is mechanical and the lint checks it. The one open fact, whether Hugo renders the dialect as intended, is answered by change A's section 1 fixture build. That is a merge gate for this change (orchestration.md section 2), not a task in it.

### 3. No rule or doc file changes in this repo

**Context**: The S changes also update each repo's rule and doc files that describe the Starlight or Astro format.
**Explored**: `git grep -n -i` for `starlight`, `astro`, `.mdx`, `sidebar`, `opmodel.dev/site`, `:::` and `docs/site` outside `docs/site/` and archived changes. The only hits are Mermaid `:::` class markers in `.tasks/enhancements.yaml`, which are unrelated. No library file outside `docs/site/` describes the Starlight or Astro page format.
**Decision**: No file outside `docs/site/` changes.
**Rationale**: Nothing in library describes the old format, so nothing is stale. The page rules live in the workspace `STYLE.md`. Adding a new pointer file (as S1 does in opm's existing `docs/STYLE.md`) would be scope growth: library has no such file.

### 4. The gates are orchestration.md's library row, not `task check`

**Context**: library's `openspec/config.yaml` names `task check` as the section gate.
**Explored**: orchestration.md section 10, library row, and section 11 trap 31.
**Decision**: The section MUST end green under: the lint; the order diff; `task -d <wt> vet`; `cd <wt> && openspec validate adopt-hugo-page-dialect --strict --no-interactive`; and `git -C <wt> diff --check`. The worker MUST NOT run `task test` or `task check`.
**Rationale**: The change edits no Go or CUE file, so `fmt` and `lint` have nothing to check. In a fresh worktree `task test` fetches `.cue-cache` cold into a read-only extract tree that makes `git worktree remove` fail, and `TestGenerate_BuildsThroughTheKernel` flakes in full-suite runs. `git diff --check` is added because a front-matter edit is where stray whitespace would land.

### 5. Take the order baseline before the first edit

**Context**: The order diff compares `origin/main`'s tree with the worktree. If the worker fetches after creating the worktree and `origin/main` gained a page, the baseline would list a page the worktree lacks.
**Decision**: The worker MUST write the baseline (`order-before.txt`) before editing any page, without fetching in between. If the supervisor later asks for a branch update (`git merge origin/main`), the worker MUST rerun the whole recipe from orchestration.md 4.1 against the fresh `origin/main`.
**Rationale**: The baseline must describe the tree the branch started from, or the diff proves nothing.

### 6. A page that lands on `origin/main` after planning gets the same edit

**Context**: `refuse-colliding-contracts` (its task 4.1) planned `docs/site/diagnostics/colliding-contracts.md` "in the shape of `oversubscribed-contracts.md`". It merged after planning (library PR 147, 3a53aed) with a `sidebar:` block (`order: 19`), so the page is on `origin/main` and section 1 converts it (`weight: 19`).
**Decision**: If that page, or any other page, is in the worktree when section 1 runs, it MUST get the same edit (same number) inside section 1. Any finding on it that is not a `sidebar:` block MUST be fixed only if orchestration.md section 4 gives the exact form (for example an aside to a GitHub alert, or an untagged fence to `text`). Anything else is a stop-and-report. If the page arrives through a branch update after the archive, the worker MUST make the same edit in its own commit, `docs(site): adopt the hugo page dialect for <page>`, and MUST NOT edit the archived `tasks.md` (tasks.md hand-off, step 7).
**Rationale**: The lint gate covers the whole tree, so it cannot pass with that page unconverted. The extra commit keeps section 1's commit as it was verified, and the squash merge folds both into the PR title. If this change merges first, the supervisor tells that change's owner to write `weight:` (orchestration.md section 9).

## Interface

This change adds nothing to orchestration.md section 6. It relies on these names from it:

- `task lint:sources` and check 2 (the source lint): A commits the section 4.1 lint byte for byte as `opmodel.dev/site/scripts/lint-sources.sh` and runs it before every build.
- Check 3 (front-matter validation inside Hugo): `title`, `description` and `type` stay as they are.
- `_partials/sidebar.html` and `_partials/opm/section-children.html`: order by `weight`, then title. This is what the kept numbers feed.
- `_partials/opm/source.html`: maps each page to `{repo, path, editURL, viewURL}` from `/src/library/docs/site/`.
- `OPM_SRC_LIBRARY` and `OPM_SRC_WORKTREE`: after the S merges, A reads library from `WS/library/.claude/worktrees/site-src`, mounted read-only at `/src/library`.
- Checks 5 (A1, two sources publishing one URL), 6 (Q2, expected and unexpected pages) and 8 (reserved prefixes): library's pages sit only under `diagnostics/` and `embedding/`, so none is under `docs/reference/`. On `origin/main` (2026-09-30) the only other repos with pages in those sections are cli (`diagnostics/publish-refusals.md`, `diagnostics/unprovided-contracts.md`) and opm-operator (`diagnostics/operator-conditions.md`), so no library path collides with another repo's.
- `site/.check/<version>/nav-order.txt`: where A shows the resulting sidebar order.

## Risks / Trade-offs

- [A new page in the old dialect lands on `origin/main` after this merges and breaks the site build (orchestration.md trap 27).] → Decision 6 covers a page that lands first. For a page that lands later, the supervisor tells in-flight changes to write `weight:`, and A's lint names file and line. Fix the page, never the lint.
- [`index.md` is a leaf bundle in Hugo and swallows its siblings (trap 1).] → Library has none, and this change adds none. Section overviews stay site-owned.
- [An invalid `type:` silently picks another layout (trap 2).] → The types (six `how-to`, one `tutorial`) are valid and unchanged. The lint and A's `errorf` check them.
- [Hugo expands shortcodes inside code fences (trap 13).] → No page contains `{{<` or `{{%`. The planning comment in `embed-the-kernel.md` holds a Go composite literal (`[]platformmodule.Entry{{Path: ...}}`), which is not a shortcode; the lint passes it and it needs no edit.
- [`.claude/worktrees/` is not gitignored in library (trap 24), and `plan-hugo` and `refuse-colliding-contracts` worktrees exist.] → Stage explicit paths only.
- [`task test` leaves a read-only `.cue-cache` that blocks `git worktree remove`, and one test flakes (trap 31).] → Decision 4: do not run it.
- [The history-rewrite hook refuses rebase and lease-push (trap 34).] → Update the branch only by merging `origin/main`, on the supervisor's ask.
- [No library tag that exists today can be built (trap 36).] → Accepted. The `docs(site)` merge opens or grows a release PR, and the owner merges it. B's manifest takes this change's merge SHA as library's dialect floor, and B's resolver refuses an older ref.
- [Between the S merges and A's merge, the Astro site on opmodel.dev `main` degrades.] → Accepted by the owner; the site is not live.

## Migration Plan

None. The pages are data for the site build. Rollback is a revert of the squash commit, but A's lint would then fail on library's pages again, so a revert only makes sense together with a revert of A.

## Open Questions

None.
