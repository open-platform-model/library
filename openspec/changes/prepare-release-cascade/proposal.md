## Why

The OPM repos are moving to a release cascade (workspace RELEASING.md, sections "The cascade"
and "Gates"): an upstream release notifies its downstreams, each downstream keeps one rolling
`deps/cascade` PR, and every release PR passes a release-pin gate. The library is in tier 1 of
"Release order" beside catalog_opm and feeds opm-operator and cli, so it needs three things
before it can join: a gate that keeps a dev or replaced pin out of a release, release-please
outputs a later notify job can read, and a changelog that stops releasing (and so cascading)
on documentation-only commits. None exists today: `.github/workflows/release.yml:31-36` has no
step id and no job outputs, no workflow inspects pins on a release PR, and
`release-please-config.json:21` lists `docs` as visible, so a docs-only merge cuts a library
release that would then ripple through opm-operator and cli.

## What Changes

- **G1 release-pin gate.** A new `task deps:release-check` fails when the tree carries a Go
  `replace` directive, a `github.com/open-platform-model/*` Go requirement at a
  pseudo-version, a `-0.dev.` CUE pin in any tracked `cue.mod/module.cue` or in a quoted
  version literal of a non-test Go file under `opm/` (where `DefaultSchemaModule` lives,
  `opm/schema/loader.go:44`), or a tracked `cue.mod/local-module.cue`. It runs as a step inside
  the existing `Go tests` job (`.github/workflows/test.yml:15-39`) only when
  `github.head_ref || github.ref_name` starts with `release-please--`. The library has no OPM
  Go dependency today (`go.mod:5-9`), so the pseudo-version check guards a future one and is
  kept to one line.
- **Release outputs.** The release-please step in `.github/workflows/release.yml` gets
  `id: release`, and the `release-please` job exposes `releases_created` and `tag_name` as job
  outputs, the same shape opm-operator already uses (`opm-operator/.github/workflows/release.yml:33-35`).
  Nothing consumes them yet; the notify job arrives with `join-release-cascade`.
- **Docs-only commits stop releasing**, per owner decision 2026-10-01 (RELEASING.md, "Pin
  classes"). `release-please-config.json` sets the `docs` changelog section to `"hidden": true`. `refactor` stays visible and keeps
  releasing, so library rewrites still integrate downstream early. `AGENTS.md:347`, which lists
  `docs` among the releasing types, is corrected in the same section, and AGENTS.md § Build And
  Dev Commands documents `task deps:release-check` with the gate.
  Today opmodel.dev builds library docs at the library version the newest cli tag pins
  (`opmodel.dev/site/versions.conf`, line mode). Per owner decision 2026-10-02 (RELEASING.md,
  "Rollout and changes") opmodel.dev moves library docs to the branch head first, so this
  section's commit waits for that change (see "Depends on / gates"; design, "Risks / Trade-offs").

Out of scope: deriving fixture versions from their cue.mods (`derive-fixture-versions`, its own
change), the notify job and the cascade receiver (`join-release-cascade`), the cascade bump
task (`add-deps-cascade-task`), the freshness and settled statuses (G2, G3), and the rulesets
that make G1 required (an owner setting, RELEASING.md "Owner settings").

SemVer: no `opm/` surface changes; CI and release configuration only. Every section lands as a
hidden `ci` commit, so the change itself cuts no release.

## Depends on / gates

- **Depends on (soft):** workspace branch `docs/release-cascade` (RELEASING.md). This change
  cites it; it should merge first or alongside so the citations resolve.
- **No code dependency** on `derive-fixture-versions`. Both edit `Taskfile.yml`, in different
  tasks (`cue:deps:update` there, a new `deps:release-check` here); whichever merges second
  rebases a trivial hunk.
- **Gates `join-release-cascade` (library):** its notify job reads the `releases_created` and
  `tag_name` outputs added in section 2.
- **Gates `add-deps-cascade-task` (library):** the cascade PR's dev pins must never reach a
  release, which section 1 enforces on the release PR.
- **Enforcement depends on an owner setting:** until the library ruleset requires the
  `Go tests` check (RELEASING.md "Owner settings"), G1 is advisory: it fails red
  but does not block the merge.
- **Depends on: opmodel.dev change `build-docs-from-branch-head` merged before this PR merges.**
  Section 3 (docs hiding) needs it, and the change merges as one PR, so the gate holds G1 and
  the release outputs too. If that change lags while they are needed, drop section 3's commit
  from this branch and land it as its own follow-up PR; never merge it before the opmodel.dev
  change. That change builds library, opm-operator and cli docs from the branch head, as core and catalog_opm already are. Until it merges, a docs-only fix in this
  repo reaches opmodel.dev only with the next library release (and the cli release that pins
  it), because a hidden `docs` commit cuts no release.
- **Peers:** the same `docs`-hiding edit lands in opm-operator and cli `prepare-release-cascade`.
  The G1 step and its `head_ref || ref_name` condition match catalog_opm, opm-operator and cli
  `prepare-release-cascade`, but the CUE dev-pin scope is deliberately wider here: every tracked
  `cue.mod/module.cue`, where the peers check only shipped or published ones (catalog_opm its
  `MODULES`, opm-operator published fixtures, cli templates). See design D-c; the wide scan was
  settled on 2026-10-02.

## Capabilities

### New Capabilities

- `release-pipeline`: what the library's release automation guarantees: release PRs refuse
  dev, pseudo-version and replaced pins; the release job publishes its outcome as outputs;
  documentation-only commits cut no release.

### Modified Capabilities

None.

## Impact

- Files: `Taskfile.yml` (new `deps:release-check` task), `.github/workflows/test.yml` (one
  step), `.github/workflows/release.yml` (step id, job outputs),
  `release-please-config.json` (one flag), `AGENTS.md` (the commit-style sentence and one
  command line).
- `opm/` packages: none. Downstream consumers (cli, opm-operator): no code impact; they see
  fewer library releases (no docs-only ones).
- CI: release PRs run one extra step in the `Go tests` job, offline, under a second.
