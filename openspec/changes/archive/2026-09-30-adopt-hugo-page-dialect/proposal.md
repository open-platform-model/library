## Why

The documentation site moves from Astro + Starlight to Hugo + Hextra (opmodel.dev change `port-site-to-hugo-hextra`). The owner ruled that every source page is rewritten to the Hugo page dialect now, with no compatibility layer in the site. The new site runs a source lint before every build, and that lint rejects Starlight front matter. Library owned seven pages under `docs/site/` at planning, and each still carries a Starlight `sidebar:` block. On `origin/main` at planning (30f08c1, 2026-09-30) the lint reported exactly those seven findings for library; an eighth page has landed since (see Parallel work). Until they are gone, the Hugo site cannot build library's pages.

## What Changes

- In each of the eight pages under `docs/site/`, the front-matter block `sidebar:` / `  order: N` becomes `weight: N`, with the same number:
  - `diagnostics/unresolved-demands.md` (20), `diagnostics/unmatched-components.md` (21), `diagnostics/oversubscribed-contracts.md` (22), `diagnostics/identity-mismatch.md` (23), `diagnostics/version-skew.md` (24), `diagnostics/transform-failed.md` (25);
  - `embedding/embed-the-kernel.md` (10);
  - `diagnostics/colliding-contracts.md` (19), which landed after planning (see Parallel work).
- Nothing else in `docs/site/` changes. The lint finds no other problem: no `.mdx`, no `index.md`, no asides, no imports or component tags, no links, no images, and every code fence is already tagged `text`. No planning comment names an `opmodel.dev/site/...` path; every `Check against:` path points at source code that exists.
- No rule or doc file changes. Nothing in library outside `docs/site/` describes the Starlight or Astro page format (checked by grep, see design.md Decision 3). The site page rules live in the workspace `STYLE.md` (already on the workspace `main`).

**Not in this change:** prose edits; new pages; a `docs/site` lint in library's CI (a 0018:D13 follow-up); rendering the pages (the site change renders every page); writing `docs/site/diagnostics/colliding-contracts.md` (its content belongs to `refuse-colliding-contracts`; if it is already on `origin/main`, this change only converts its front matter, see Impact).

**Related decisions.** The pages keep `title`, `description` and `type` as 0018:D7 and 0018:D7:R1 require (seven how-to pages, one tutorial). `weight` orders a page within its section and declares no address, so 0018:D7:R3 still holds. Library keeps owning these pages under 0018:D8.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. docs/site pages only; `.openspec.yaml` sets `skip_specs: true`.

## Impact

**SemVer:** none. No `opm/` package, signature, behaviour or dependency changes. The CLI and the operator have nothing to pick up. The commit is `docs(site)`, and release-please shows `docs` commits (AGENTS.md § Commit style), so the merge opens or grows a library release PR. The owner merges that PR; this change cuts no tag.

**Packages:** none. **Files:** the eight pages above.

**Touches:** `docs/site/**`, plus this change directory (task ticks, then the archive).

**Depends on:** starts when planned on main (this change directory on library `main`). Merges when verify is green; after A section 1 is green (A is opmodel.dev `port-site-to-hugo-hextra`; its fixture build proves the dialect renders); before A section 2.

**Consequences, accepted by the owner:**
- Once all six source-repo dialect changes merge, the Astro build on opmodel.dev `main` renders degraded or fails. The site is not live.
- No library tag that exists today can feed a Hugo build, because each one carries `sidebar:`. The first usable tag is the one cut from the release PR this merge opens or grows.

**Parallel work.** `refuse-colliding-contracts` merged after this change was planned (library PR 147, 3a53aed, 2026-09-30) and added `docs/site/diagnostics/colliding-contracts.md` with a Starlight `sidebar:` block (`order: 19`). So the lint on `origin/main` now reports eight findings, and this change converts that page the same way (`weight: 19`, tasks 1.4). Any other page that reaches `origin/main` before this branch is cut or updated gets the same edit.

**Complexity (Principle VII):** none added. Eight two-line blocks become eight one-line keys.
