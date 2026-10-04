## Why

Two maintainer and reader examples show how to pin the core schema explicitly, and both name a core release the library no longer builds against:

- `docs/getting-started.md:51` writes `Module: "opmodel.dev/core@v2.0.0-beta.1"`, and `:56` shows the resolved version as `// → "v2.0.0-beta.1"`.
- `AGENTS.md:350` writes `schema.OCILoader{Module: "opmodel.dev/core@v2.0.0-beta.1"}`.

`DefaultSchemaModule` is `opmodel.dev/core@v2.0.0-beta.2` (`opm/schema/loader.go:43`).

The release cascade's task notices this and warns, without editing. Step C4 in `.tasks/cascade/cascade.sh:288-296` greps both files for `opmodel.dev/core@v…` and adds one warning per release that differs from the loader's. The main spec `deps-cascade`, requirement "The cascade task never touches release or steering files" (`openspec/specs/deps-cascade/spec.md:89-105`), requires exactly that. The warning lands in the cascade PR body (workspace `RELEASING.md`, "The cascade" › "Title from diff class": the body holds the moved pins, the triggering releases and the warnings). Today two warnings name these two files on every cascade run, including runs that move nothing a reader cares about.

A one-off bump clears the warnings only until the next core release. The examples are literals, so each core move brings the warnings back and needs a human commit on `deps/cascade` to clear them. The library's core bumps already carry `need-human-review` (`RELEASING.md`, "Runbook" › "Handling a cascade PR"), and that review is meant for the glue, not for prose.

This change makes the two examples follow the loader. The cascade rewrites them in the same run that moves `DefaultSchemaModule`, so a core cascade PR carries the docs edit and no warning. It also brings the examples to beta.2 once, now.

Timing: the library's receiver goes live in Phase 4 (`RELEASING.md`, "Rollout and changes" › "Phases"; Phase 3 wiring contract §1 and §9.1). It should not carry a standing warning into the first live run, and it should not teach the owner to ignore the warnings block.

## What Changes

- **Examples current now.**
  - `docs/getting-started.md:51` and `AGENTS.md:350` name `opmodel.dev/core@v2.0.0-beta.2`.
  - `docs/getting-started.md:56` no longer prints a literal version. It says the resolved version is the one after the `@` in `Module`, so the example has one literal, which the task keeps current.
- **The cascade task moves the examples with the loader.**
  - Step C4 of `.tasks/cascade/cascade.sh` becomes an edit, and only in a run where C1 moved `DefaultSchemaModule`. In `docs/getting-started.md` and `AGENTS.md` it rewrites the version in every `Module: "opmodel.dev/core@vN.x.y[-pre]"` example with the loader's major N to the new loader value, unless `.cascade-frozen` lists the file for `opmodel.dev/core@v2`. A release named anywhere else in either file, prose included, is never rewritten.
  - A run that does not move core never edits either file. So a docs-only lag can never produce a `fix(deps)` PR by itself, and never trips G2.
  - Whatever still differs from the loader after the step stays a warning: a different major, a release named in prose, a frozen file, or a run without a core move.
- **Tests.**
  - The network scenario S2 also sets the examples to an older core (the `docs/getting-started.md` one older still) and expects them back at the original bytes. That makes the existing "result equals the original tree" check cover the docs and the catch-up.
  - S2 no longer expects a docs warning.
  - The network scenario S4 adds a `v1` and an older `v2` release in `AGENTS.md` prose and expects both unchanged and warned about.
  - The non-required `Cascade task (network)` workflow also runs when either docs file changes.
  - A new offline scenario, S11, shows that a lagging example is warned about and left untouched when core does not move.
- **`AGENTS.md`, "Release cascade task"** (`:301`) says the two examples follow the loader.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `deps-cascade`:
  - The docs examples follow the loader's core in a run that moves core. This adds a requirement.
  - "The cascade task never touches release or steering files" warns only for what the docs step leaves behind.
  - The offline test set gains S11, and its list names the frozen-loader and frozen-lag scenarios that already run.

## Impact

**SemVer: no release.** Nothing under `opm/` changes. The one commit is a `ci(cascade)` commit, which release-please hides (`AGENTS.md`, "Commit style"). The docs edits ride in it because the S2 test asserts both halves together (design.md D4). No public surface changes. Principle VII: the step is about 15 lines of shell. It reuses the task's existing frozen check and warning helper, and it adds no new file and no new tool.

**Affected files:** `.tasks/cascade/cascade.sh` (C4), `.tasks/cascade/test.sh` (S2, S4, `set_older`, new S11), `.github/workflows/cascade-task.yml` (path filter only; a human edit, not a bot one), `docs/getting-started.md`, `AGENTS.md`. No `opm/` package.

**Downstream consumers:**

| Consumer | What it has to do |
| --- | --- |
| cli, opm-operator | Nothing. Neither file ships in the Go module's API, and `docs/getting-started.md` is outside the docs bundle (`docs-kit.cue:24` bundles `docs/site` only). |
| Cascade PR readers | A library core cascade PR now also changes `docs/getting-started.md` and `AGENTS.md`. Both are `shipped` class (`.tasks/cascade/classes` lists only test paths), but the loader is already shipped in the same diff, so the title stays `fix(deps): bump core to …` (`RELEASING.md`, "Title from diff class"). |
| G2 `cascade/freshness` (Phase 3 wiring contract §8.2) | Nothing. G2 runs `task -x deps:cascade` at the release head. A docs lag without a core move yields exit 3, so it never reads as a stale shipped pin. |

**Depends on and gates:**

- Depends on nothing unmerged: `add-deps-cascade-task` (library PR 176) is on `main`.
- Does not depend on `.github` `add-release-cascade-workflows` or library `join-release-cascade` (Phase 3 wiring contract §1, rows A and B3). It changes only the task the receiver calls, and the task's interface (exit codes, env, title and body tasks; Phase 2 contract §5.1, §5.4) is unchanged.
- It should merge before library's receiver goes live (Phase 4), so the first live core cascade carries no docs warning.
- Certain conflict with B3 (library `join-release-cascade`): the "Release cascade task" paragraph is one physical line (`AGENTS.md:301`), and B3 task 4.1 extends the same line. Whichever PR merges second runs `git merge origin/main` and keeps both clauses.
- A hand-made core bump (like PR 173) that leaves the examples behind fails no required check: the no-op scenario does not assert that `main` is warning-free (design.md D6). The next cascade run warns, and the next core move brings the examples current.
- Gate to merge: `task check` green, the full `task -x deps:cascade:test` green (the non-required `Cascade task (network)` workflow runs it on this PR because `.tasks/cascade/**` changes), and on this branch `task -x deps:cascade` exits 3 with no docs warning.

**Amendments outside this repo** (for the supervisor; not made here):

- **Merge gate:** this PR merges only together with a workspace PR, in the same review round, that rewrites `RELEASING.md:321` ("What each repo's task moves", library row, Shipped column), so the doc never describes behaviour that differs from what is merged. The cell becomes: "`DefaultSchemaModule`, labelled `need-human-review`, and the two core examples in `docs/getting-started.md` and `AGENTS.md`, which follow it in the same run; `DefaultCoreVersion` and the other test literals derive from it after `derive-fixture-versions`" (the word "only" removed).
- This amends the Phase 2 contract's library step 5 (`.github` `openspec/changes/archive/2026-10-04-add-cascade-resolver/contract.md`, §6.2, "Warnings"), which said warn only. The archived contract stays as written. The library main spec becomes the record.
