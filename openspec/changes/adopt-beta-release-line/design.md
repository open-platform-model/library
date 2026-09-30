## Context

See proposal.md for the motivation. State on `main` at planning time (2026-09-30):

- `.release-please-manifest.json` reads `1.0.0-alpha.36`; release-please PR #151
  (`chore(main): release 1.0.0-alpha.37`, docs-only content) is open on
  `release-please--branches--main--components--library` and is held, never merged as alpha.
- `release-please-config.json` has `versioning: prerelease`, `prerelease: true`,
  `prerelease-type: alpha`. The repo squash-merges with the PR title as the commit subject and
  the branch commit messages as the body (`squash_merge_commit_title=PR_TITLE`,
  `squash_merge_commit_message=COMMIT_MESSAGES`).
- The default core pin, `registrytest.DefaultCoreVersion` and 35 fixture `cue.mod` files name
  `v2.0.0-alpha.13`. Four of them are `CUE_MODULE_GLOBS` modules (`modules/opm_platform`,
  `testdata/modules/web_app`, `testdata/parity`, `testdata/parity/opm_platform`), each also
  pinning `opmodel.dev/catalogs/opm@v4` at `v4.4.2` (and, except `web_app`,
  `cue.dev/x/k8s.io@v0` at `v0.12.0`); `testdata/cue.mod` and `testdata/render/**` (platforms,
  instances, scenarios and the 14 registrytest-served `registry/*` modules) are hand-pinned.
- `task cue:deps:update` runs `cue mod get <dep>` to latest for every direct dependency of the
  discovered modules (`|| true`), so it would also move `catalogs/opm@v4` (to `v4.4.3` today,
  `v4.4.4` after G3) and resolve core to whatever beta is newest when it runs.
  `task cue:catalog:drift` checks only that each catalog pin is published, so `v4.4.2` keeps
  it green.
- `TestDefaultCoreVersion_IsTheDefaultSchemaRelease` already asserts
  `DefaultCoreVersion == schema.DefaultSchemaVersion()` and that every
  `testdata/render/**/cue.mod` pins it, so a partial re-pin fails the suite.
- Floors are presence-based (`ProvidedBySince = 2.0.0-alpha.12`, `CollisionsSince =
  2.0.0-alpha.13` documented, not compared), and the only semver comparison
  (`renderstage/skew.go`, Masterminds) orders `alpha < beta`, so no Go logic changes.

## Goals / Non-Goals

**Goals:**

- The library renders against, tests against and defaults to `opmodel.dev/core@v2.0.0-beta.1`.
- The library's next release is exactly `v1.0.0-beta.1`, and later releases follow the beta
  counter.
- The beta promise and the beta migration-note rule are written where a consumer and a
  contributor look (README, AGENTS.md, migrations/README.md, ADR-010).

**Non-Goals (design-level boundaries):**

- Raising `ProvidedBySince` or retiring any older-core test: the floor stays `2.0.0-alpha.12`.
- Arming `migrations/` enforcement (ADR-004 keeps that for GA).
- Adding a Go-side guard for `testdata/cue.mod` or the `modules/`, `testdata/parity` pins beyond
  what the existing tests assert.
- Moving any non-core dependency pin. A `catalogs/opm@v4` move (to `v4.4.4` after G3) is a
  separate `test(fixtures)` PR with its parity-literal edits, per `AGENTS.md` § Commit style.

## Research & Decisions

### D1. The beta crossing is a Release-As footer on the squash commit, never config

**Context**: release-please keeps the label on a `prerelease-type` flip of an existing
`X.Y.Z-alpha.N` line (supervisor dry run on release-please 17.3.0 and 17.6.0; upstream issue
2447): with the flip alone the next PR reads `1.0.0-alpha.37`. A config `release-as` is sticky
(cli ed9774e, catalog_opm bc778ca).

**Explored**: the cutover review's release-please runs: a final-footer `Release-As` paragraph
followed by the plain co-author line parses as RELEASE AS; a GitHub multi-commit squash body
(`* type: msg` bullets) or a body line starting with `word(` loses it (library AGENTS.md,
squash-body hazard).

**Decision**: flip `prerelease-type` to `beta` in this change (so beta.2 onward follows the
beta counter) and let the supervisor cross the line with the squash commit's final footer
`Release-As: 1.0.0-beta.1`. The PR is the carrier: its squash subject is the PR title
`fix(deps): pin core v2.0.0-beta.1` plus ` (#N)`. The body is written by the supervisor at merge
time (the auto-filled COMMIT_MESSAGES body is discarded) and ends with the footer paragraph
then the plain co-author line. No inner branch commit's footer is relied on.

**Alternatives**: hand-set the manifest to `1.0.0-beta.0` (rejected: release tooling owns the
manifest); config `release-as` (rejected: sticky); trust the flip (rejected: measured to keep
alpha).

### D2. One PR, squashed as fix(deps), carrying only core pins

**Context**: the canon forbids mixing a shipped bump with a fixture bump in one PR (its list:
`hack/*`, `examples/*`, `config/samples/*`, `tests/fixtures`, `test/fixtures`), and the library's
`AGENTS.md` § Commit style routes "the parity platform and other test fixture pins" to
`test(fixtures)`. Against that stands the schema-dispatch requirement "DefaultSchemaModule
constant": the default names the release the render glue, the fixtures and the parity oracle
were verified against, and every served fixture declares that same release. Moving the
default without the fixtures' core pin makes the default an unverified claim; the test
`TestDefaultCoreVersion_IsTheDefaultSchemaRelease` enforces this mechanically only for
`testdata/render/**`, while `testdata/cue.mod`, `testdata/parity*`, `testdata/modules/web_app`
and `modules/opm_platform` are bound by the requirement alone (the parity run and the flow test
are what verify the default against them).

**Decision**: one PR, squash type `fix(deps)` (the default pin ships in the Go module),
following the precedent of the alpha.13 re-pin (`fix(deps): pin core v2.0.0-alpha.13`, archived
change `refuse-colliding-contracts`, task 1.5). It carries the core pin of every fixture,
including the parity platform: this is a named exception to `AGENTS.md` § Commit style, taken
because the requirement couples the core pins to the default, and it needs supervisor sign-off
(open question in the worker report). No other dependency pin moves (D4): a catalog move is
not coupled to the default and goes to its own `test(fixtures)` PR. Inner branch commits keep
their own types (`fix(deps)`, `chore(release)`, `docs`, `chore(openspec)`) for review; only the
squash reaches main.

### D3. Keep the default's history true in the doc comment

**Decision**: the `DefaultSchemaModule` comment names `2.0.0-beta.1` as the pinned release and
says it carries the `2.0.0-alpha.13` schema unchanged; the paragraph that lists what
`2.0.0-alpha.13` first reported stays, attributed to `alpha.13`. The "not the render floor"
paragraph's "a platform pinning a release between the floor and this default" becomes "between
the floor and [CollisionsSince]": with the default at beta.1 the old range would include
alpha.13, which does carry the report, and the schema-dispatch delta already says
`2.0.0-alpha.13`. `CollisionsSince` and `ProvidedBySince` do not move. Examples in doc comments that only illustrate a canonical
version string (`DefaultSchemaVersion`'s example, `PinnedVersion`, `renderstage.Dep.Version`,
`catalog.Requires`, `registrytest.coreDep` and `Major`) move to `v2.0.0-beta.1`, so a reader
never meets the retired default as if it were current.

### D4. Re-pin mechanics follow the library's documented practice

**Explored**: `grep -rl 'v2.0.0-alpha.13' --include=module.cue testdata modules` (35 files) against
`CUE_MODULE_GLOBS` in `Taskfile.yml` (4 modules discovered), and the alpha.13 re-pin in the
archived `refuse-colliding-contracts` change.

**Decision**: not `task cue:deps:update` (it resolves every direct dependency to latest, so the
carrier's content would depend on when it runs; see Context). Instead, in each of the four
discovered modules, `cue mod get opmodel.dev/core@v2.0.0-beta.1 && cue mod tidy` with the GHCR
mapping, which moves the core pin to exactly that tag and nothing else; then hand-edit
`v: "v2.0.0-alpha.13"` to `v: "v2.0.0-beta.1"` in `testdata/cue.mod` and every
`testdata/render/**/cue.mod/module.cue`, plus the core mention in each platform fixture header,
exactly as the alpha.13 re-pin did (archived `refuse-colliding-contracts` task 1.3). Verified
by two checks: every `opmodel.dev/core@v2` pin under `testdata` and `modules` is exactly
`v2.0.0-beta.1`, and `catalogs/opm@v4` stays `v4.4.2` and `cue.dev/x/k8s.io@v0` stays `v0.12.0`
in the four discovered modules. The parity-harness literals do not move.

Deliberately older pins stay: `modfile_test.go` skew rows (`alpha.6`), `generate_test.go`
`coreVersion = alpha.7` and the synthetic `closure_test.go` graph (`alpha.6/7/8`, k8s
`alpha.2`), `contracts_test.go` and `render_collision_test.go` floors (`alpha.9` to `alpha.12`),
`coretooold` strings, the `loader_test.go` explicit `alpha.4` pin case, `CHANGELOG.md`,
archives and `docs/site/diagnostics/*` since-release text.

### D5. render_core_floor_test reads the harness version

**Decision**: `acquireOlderCorePlatform` builds its precondition and replacement from
`registrytest.DefaultCoreVersion` (as `render_collision_test.go` already does), so the next
default move needs no edit there. Captured as a scenario in the schema-dispatch delta.

### D6. Beta migration notes live in the CHANGELOG footer (ADR-010)

**Context**: ADR-004 made `migrations/` dormant until GA; the beta promise allows breaking
changes only as `feat!` whose `BREAKING CHANGE:` footer is the migration note.

**Decision**: add ADR-010 recording that during beta the migration note is the
`BREAKING CHANGE:` footer as release-please renders it into `CHANGELOG.md`, that `migrations/`
stays dormant until GA, and that GA (not beta) arms ADR-004. `migrations/README.md` "Status"
and `AGENTS.md` working-style bullet point to it. ADR-004 is not edited (accepted ADRs are
records); ADR-010 references it.

**Alternatives**: arm `migrations/` at beta (rejected: owner decision says GA; both consumers
still migrate in the same PR wave); amend ADR-004 in place (rejected: rewrites a record).

### D7. Beta promise wording

The README "API stability" section carries the canon wording, adapted to the library: from its
first beta the library is on the path to GA; a breaking change during beta is only a `feat!`
commit whose `BREAKING CHANGE:` footer is the migration note in the CHANGELOG, advances the
`-beta.N` counter and never moves the Go module path or the core major; stable lines
(`opmodel.dev/catalogs/opm@v4`, the module fleets) keep the normal SemVer rule; GA drops the
suffix via `prerelease: false` plus a visible carrier commit, in dependency order. Note that
the Go module path has no `/vN` suffix today, so "never a new major" means no `v2` of the
library during beta.

The two-track paragraph's "a breaking change here is a major bump of the library" contradicts
this and is rewritten: before GA a breaking Go API change is a `feat!` that advances the
prerelease counter; from GA it is a major bump. The section also tells a consumer to pin an
explicit `v1.0.0-beta.N`, because `go get ...@latest` resolves the retired `v0.7.0` (retiring
it stays a GA item).

Principle VI states the same rule as law ("MAJOR: any breaking change to `opm/` types,
signatures, or behavior"), and every proposal classifies itself against it. It gains a pre-GA
clause in both `CONSTITUTION.md` and the `openspec/config.yaml` context: until GA, a breaking
change is a `feat!` whose `BREAKING CHANGE:` footer is the migration note and which advances
the prerelease counter, never a new major. Only `feat!` is named, matching the canon; a wider
`!` rule would need owner sign-off and a canon change first.

## Risks / Trade-offs

- [G1 not met, or core beta.1 differs from alpha.13] -> task 1.1 fetches both releases from
  GHCR and diffs the two module extract directories in the CUE cache; a schema difference stops
  the section and is reported (the doc comment's "unchanged" claim and the glue verification
  would both be wrong).
- [G1 lands on a later core beta] -> a failed core publish after tagging burns the version, so
  G1 may be `v2.0.0-beta.N` with N greater than 1. The supervisor records the real tag when
  ticking G1, and every core literal of this change (constant, registrytest, fixtures, spec
  deltas, PR title, squash body) uses it; the library's own `Release-As` stays `1.0.0-beta.1`.
- [Footer lost at merge: squash body auto-filled, footer not final, or a body line starting
  with `word(`] -> release-please proposes `1.0.0-alpha.37` or drops the commit. Supervisor
  parses the exact message with release-please's own parser before merging, and after merge
  reads the exact merge commit (`gh pr view <N> --json mergeCommit`, fetch, then
  `git show -s --format=%B <oid>`; never `git log -1`, which can read another commit) and
  checks that #151 reads
  `chore(main): release 1.0.0-beta.1` within one release run; fallback is a
  BEGIN_COMMIT_OVERRIDE edit of the merged PR body and a re-run.
- [#151 merged as alpha before the carrier] -> burns `1.0.0-alpha.37`; harmless but the carrier
  must then still land. Held by the supervisor.
- [Partial or over-reaching re-pin] -> `TestDefaultCoreVersion_IsTheDefaultSchemaRelease`
  fails for `testdata/render`; task 1.4 also asserts that every core pin under `testdata` and
  `modules` is exactly `v2.0.0-beta.1` and that no catalog or k8s pin moved.
- [Carrier must be cancelled after merge, before #151 merges] -> a revert does not cancel it:
  release-please takes the `Release-As` note of the first commit carrying one in its
  newest-first window, a GitHub revert message carries none, and the original carrier stays in the window, so #151 would still read
  `1.0.0-beta.1` and tag an immutable `v1.0.0-beta.1` over an alpha default and alpha config.
  Keep #151 on hold and fix forward (see Migration Plan).
- [Older-core tests need old alphas on GHCR] -> they already fetch alpha.9 to alpha.12; old tags
  are never deleted.
- [Downstream generated platforms pin beta.1 the moment cli/operator adopt the release] ->
  intended; C3 and C4 carry the matching fixture and test moves.
- [Go `@latest` resolves `v0.7.0`] -> unchanged by this change; consumers pin explicitly. GA
  exit item, not beta.

## Migration Plan

Merge after G1; the release PR (#151, retitled) is merged by the supervisor to produce G2
(`v1.0.0-beta.1` on the Go proxy).

Rollback before the release PR merges: keep #151 on hold and fix forward with a follow-up PR on
the carrier's content. A bare revert is never a rollback here (see Risks), and #151 is never
merged after one. If the crossing itself must be cancelled, the cancelling commit on main is a
conventional commit whose own final footer carries an overriding `Release-As:` (the version
the supervisor decides to release instead), because the newest note in the window wins; the
supervisor re-reads #151's title immediately before any merge of it. After the tag exists the
version is burnt and the next attempt is `1.0.0-beta.2`.
