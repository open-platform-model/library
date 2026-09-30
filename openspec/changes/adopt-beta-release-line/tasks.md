# Tasks: adopt-beta-release-line

Worktree `library/.claude/worktrees/beta-adopt-beta-release-line`, branch
`beta/adopt-beta-release-line` (from `origin/main`). Every command runs inside the worktree
with the registry env exported on two lines:

```bash
export CUE_REGISTRY='opmodel.dev=ghcr.io/open-platform-model,testing.opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'
export OPM_REGISTRY="$CUE_REGISTRY"
```

The branch already carries `chore(openspec): archive restructure-migration-docs` (all its tasks
were done; its `migration-docs` spec was created on archive) and this plan.

## Gates (supervisor ticks; not an apply section)

- [x] G1 core `v2.0.0-beta.1` is published on GHCR (`opmodel.dev/core@v2.0.0-beta.1` resolves).
      Recorded by the supervisor: `v2.0.0-beta.1`.
      Sections 1 to 4 start only after this box is ticked.
- [ ] PR-GATE carrier PR reviewed; supervisor merge checks of the release block passed.
- [ ] G2 library `v1.0.0-beta.1` resolvable on the Go proxy (after the retitled #151 merges).

If G1 is `v2.0.0-beta.N` with N not 1 (a burnt core tag), the supervisor records the real tag
when ticking G1 and that tag replaces `v2.0.0-beta.1` / `2.0.0-beta.1` for core in every task
below, in the constant, `registrytest`, the fixtures, the spec deltas, the PR title and the
squash body. The library's own `Release-As` stays `1.0.0-beta.1`.

Section 4 archives this change on the branch, before the PR exists, so PR-GATE, G2 and the
release block (R1 to R5) are frozen unchecked in the archived tasks.md by design. They are
tracked in the supervisor log after archive, never ticked in the archived file.

## 1. Pin core v2.0.0-beta.1 (opm/schema, opm/internal/registrytest, fixtures, tests)

- [x] 1.1 Precondition: in a scratch module under the supervisor scratchpad, `cue mod get
      opmodel.dev/core@v2.0.0-beta.1` succeeds with the GHCR mapping; fetch
      `v2.0.0-alpha.13` the same way and `diff -r` the two module extract directories in the
      CUE cache (`$(cue env CUE_CACHE_DIR)/mod/extract/opmodel.dev/core@v2.0.0-alpha.13` and
      `.../core@v2.0.0-beta.1`; the published module root is core's `src/`, so there is no
      `src/` below them). Verify: the fetch succeeds and the schema sources are identical
      (identity, docs or comment-only differences are acceptable and noted). On any schema
      difference, stop with nothing edited and report.
- [x] 1.2 `opm/schema/loader.go`: `DefaultSchemaModule = "opmodel.dev/core@v2.0.0-beta.1"`;
      rewrite the doc comment per design D3 (beta.1 is the pinned release and carries the
      alpha.13 schema unchanged; the list of what `2.0.0-alpha.13` first reported stays,
      attributed to alpha.13; in the "not the render floor" paragraph, "between the floor and
      this default" becomes "between the floor and [CollisionsSince]", matching the
      schema-dispatch delta, and the rest of it stays); move the
      `DefaultSchemaVersion` and `PinnedVersion` comment examples to `v2.0.0-beta.1`. Move the
      two assertions in `opm/schema/loader_test.go` (`TestDefaultSchemaModule_PinsVerifiedRelease`,
      `TestDefaultSchemaVersion_IsTheDefaultModulesVersion`); keep the `explicit alpha pin`
      case and add an `explicit beta pin` case (`opmodel.dev/core@v2.0.0-beta.1`).
      Verify: `go test ./opm/schema/...` passes.
- [x] 1.3 `opm/internal/registrytest/registrytest.go`: `DefaultCoreVersion = "v2.0.0-beta.1"`,
      and the `coreDep` / `Major` comment examples; `registrytest_test.go` `Major` table gains
      the beta rows (keep the alpha rows as parse cases). Doc-comment examples in
      `opm/internal/renderstage/modfile.go` (`Dep.Version`) and `opm/catalog/requires.go` move
      to `v2.0.0-beta.1`. `opm/schema/paths.go` (`ProvidedBySince`, `CollisionsSince`) is not
      touched. Verify: `grep -rn 'alpha\.13' opm --include=*.go | grep -v _test.go` lists only
      the loader.go history sentence and `CollisionsSince`.
- [x] 1.4 Re-pin the core fixtures per design D4 (core pin only; do NOT run
      `task cue:deps:update`, which moves every dependency to latest). In each of
      `modules/opm_platform`, `testdata/modules/web_app`, `testdata/parity` and
      `testdata/parity/opm_platform`: `cue mod get opmodel.dev/core@v2.0.0-beta.1 && cue mod
      tidy` with the GHCR mapping. Then hand-edit `v: "v2.0.0-alpha.13"` to
      `v: "v2.0.0-beta.1"` in `testdata/cue.mod/module.cue` and every
      `testdata/render/**/cue.mod/module.cue` (16 fixture packages plus the 14 `registry/*`
      modules), and the core mention in the header of each `testdata/render/platform*/platform.cue`
      and `modules/opm_platform/platform.cue`. `opm/kernel/parity_harness_test.go` is not
      touched. Verify: `grep -rl 'v2.0.0-alpha.13' --include=module.cue testdata modules`
      prints nothing; `grep -rh -A1 '"opmodel.dev/core@v2"' --include=module.cue testdata
      modules | grep -o 'v2[^"]*' | sort -u` prints exactly `v2.0.0-beta.1`; in the four
      modules above, `opmodel.dev/catalogs/opm@v4` still reads `v4.4.2` and `cue.dev/x/k8s.io@v0`
      (where present) still reads `v0.12.0` (`git diff` on their `module.cue` shows only the
      core line); `task cue:tidy` leaves no diff; `git diff --stat
      opm/internal/renderstage/render.cue.tmpl` is empty.
- [x] 1.5 Test literals that carry the default: `opm/kernel/render_core_floor_test.go`
      (`acquireOlderCorePlatform` builds its `require.Contains` precondition and the
      replacement source from `registrytest.DefaultCoreVersion`, target stays `v2.0.0-alpha.10`;
      design D5), `opm/kernel/render_test.go` (fixture comment and the `ModuleVersion` /
      `PlatformVersion` rows), `opm/internal/renderstage/modfile_test.go` (every alpha.13
      literal; the `alpha.6` skew rows stay), `opm/internal/renderstage/stage_test.go`,
      `opm/helper/platformmodule/generate_test.go` (lines naming the default; `coreVersion =
      alpha.7` and k8s `alpha.2` stay), `opm/catalog/requires_test.go`. Leave every deliberately
      older pin listed in design D4. Verify: `grep -rn 'alpha\.13' opm --include=*_test.go`
      shows only `CollisionsSince`-related or deliberately historical cases, each justified in
      the worker report.
- [x] 1.6 `AGENTS.md` layout line for `testdata/` (render fixtures "pinned to core
      2.0.0-beta.1"). Verify: `grep -n 'alpha\.13' AGENTS.md` prints nothing.
- [x] 1.7 Cross-cutting checks against GHCR: `task check`; `task cue:check`;
      `task cue:catalog:drift`; `OPM_FLOW_TEST_FORCE=1 task cue:test:flow`;
      `OPM_FLOW_TEST_FORCE=1 go test ./opm/kernel -run 'TestParity|TestFlow' -count=1`;
      `go test -race ./opm/kernel ./opm/internal/renderstage -count=1`. Verify: all green; a
      parity divergence stops the section and is written into design.md Risks.
- [x] 1.8 `task check` green, then commit `fix(deps): pin core v2.0.0-beta.1` (body: one
      sentence that core's first beta carries the alpha.13 schema unchanged and the render
      floor stays 2.0.0-alpha.12; no body line starting with `word(`; no bare at-sign).

## 2. Release line (release-please-config.json)

- [ ] 2.1 `release-please-config.json`: `"prerelease-type": "beta"`; `versioning`,
      `prerelease` and every other key unchanged; no `release-as` key anywhere; the manifest
      is not touched. Verify: `git diff` shows exactly one changed line and
      `grep -ci release-as release-please-config.json` prints 0.
- [ ] 2.2 `task check` green, then commit `chore(release): cut prereleases on the beta line`.

## 3. Beta promise and beta migration notes (README, ADR, migrations, AGENTS, docs)

- [ ] 3.1 `README.md` "API stability": add the beta promise per design D7 (canon wording
      adapted to the library); keep the two-track structure but rewrite the Go-module bullet's
      "A breaking change here is a major bump of the library": before GA, a breaking Go API
      change is a `feat!` that advances the prerelease counter; from GA, it is a major bump.
      Add one line: pin an explicit `v1.0.0-beta.N`, because `go get ...@latest` resolves the
      retired `v0.7.0`. Verify: the section names `feat!`, the `BREAKING CHANGE:` footer as the
      CHANGELOG migration note, the `-beta.N` counter, the stable-line rule for
      `opmodel.dev/catalogs/opm@v4` and the fleets, the owner sign-off for a core break that
      would force a catalogs/opm major, the GA carrier rule and the explicit-pin line; it does
      not say the library moves or holds core's major; `grep -n 'major bump' README.md` shows
      only the GA-qualified sentence.
- [ ] 3.2 New `adr/010-beta-migration-notes-in-changelog.md` from `adr/TEMPLATE.md`, status
      Accepted: during beta the migration note is the `BREAKING CHANGE:` footer rendered into
      `CHANGELOG.md`; `migrations/` stays dormant until GA; GA arms ADR-004; ADR-004 is not
      amended. Consequences in bold-labeled paragraphs, no bullet lists. Verify: the file
      follows the template headings and cites ADR-004.
- [ ] 3.3 `migrations/README.md` "Status": dormant through alpha and beta; name the footer rule
      and link ADR-010. `AGENTS.md` working-style bullet (pre-GA, no migration fragment): on the
      beta line a break is `feat!` with the `BREAKING CHANGE:` footer as the migration note;
      entrypoint bullet for `migrations/README.md` mentions beta. Verify: both files link
      ADR-010 and neither instructs writing a fragment before GA.
- [ ] 3.4 `docs/getting-started.md` pin example (the `OCILoader{Module: ...}` snippet and its
      resolved-version comment) moves to `opmodel.dev/core@v2.0.0-beta.1`. In
      `docs/site/embedding/embed-the-kernel.md`, the "Before you begin" brief's library version
      becomes "the newest v1.0.0-beta.N tag (Verify at writing)" and the section-1 brief's `go
      get` target becomes `github.com/open-platform-model/library@v1.0.0-beta.1` (Verify at
      writing). Verify: `grep -n 'alpha' docs/getting-started.md
      docs/site/embedding/embed-the-kernel.md` prints nothing.
- [ ] 3.5 Principle VI pre-GA clause per design D7, in `CONSTITUTION.md` (§ VI) and the
      `openspec/config.yaml` `context` copy alike, carrying the full canon promise: until GA, a
      breaking change to `opm/` is a `feat!` whose `BREAKING CHANGE:` footer is the migration
      note and which advances the prerelease counter, never a new major; stable lines
      (`opmodel.dev/catalogs/opm@v4`, the module fleets) keep the normal SemVer rule; a core
      beta break that would force a catalogs/opm major needs owner sign-off; GA drops the
      suffix via `prerelease: false` plus a visible carrier commit per package, in dependency
      order, and the MAJOR line applies from GA. Verify: both files carry the clause with the
      same meaning, including the stable-line and GA sentences; neither says the library
      moves or holds core's major; `openspec validate --all --strict` still passes, and
      neither names any `!` type other than `feat!`.
- [ ] 3.6 `task check` green, then commit `docs: state the beta promise and where beta migration
      notes live`.

## 4. Verify and archive

- [ ] 4.1 Whole-tree gates on the final tree against GHCR: `task check`, `task cue:check`,
      `task cue:catalog:drift`, `OPM_FLOW_TEST_FORCE=1 task cue:test:flow`, the parity and flow
      run of 1.7. Verify: all green.
- [ ] 4.2 `openspec validate adopt-beta-release-line --strict` passes, and `/opsx:verify`
      (pointed at this worktree) reports no CRITICAL issue beyond the unchecked boxes that are
      open by design at this point (4.2 to 4.4, the release block R1 to R5, PR-GATE, G2);
      deviations go into the worker report.
- [ ] 4.3 Archive the change: `openspec archive adopt-beta-release-line --yes`. Verify: the
      schema-dispatch, platform-artifact and migration-docs main specs carry the MODIFIED text,
      every scenario heading survived, and `openspec validate --all --strict` passes. No
      `enhancement.yaml` exists, so no delivery log runs.
- [ ] 4.4 Commit `chore(openspec): archive adopt-beta-release-line`.

## Release block (not an apply section; the tasks-rule exception for a release deliverable)

Unnumbered like the Gates block, so it carries no section-closing commit: its steps are
delivery, which `openspec/config.yaml` allows here because this change's deliverable is itself
a release. R1 and R2 are the worker's; R3 to R5 are the supervisor's and are tracked in the
supervisor log (see the Gates block).

PR (the only PR of this change, and the carrier):

- Title: `fix(deps): pin core v2.0.0-beta.1` (becomes the squash subject; conventional,
  lowercase, no at-sign).
- Squash commit type: `fix(deps)`. Carrier: YES. Footer: `Release-As: 1.0.0-beta.1`.
- Merge gate: G1 plus green CI (test, lint, cue, pr-title, mention-guard).
- Expected release PR: #151 retitled `chore(main): release 1.0.0-beta.1`. #151 is held: only
  the supervisor merges it, and only at G2 (ruling R-e); the hold comment is already posted on
  #151, so no worker step posts one.

- [ ] R1 Worker: `git merge origin/main` if main moved (never rebase), rerun 1.7's gates if
      anything merged, then `git push -u origin beta/adopt-beta-release-line`.
- [ ] R2 Worker: open the PR with the title above and a body of at most 250 words (why:
      beta cutover carrier; where to look first: `opm/schema/loader.go` and the render
      fixtures; risk: the squash must carry the footer; what the reviewer must do: nothing
      before G1). No commit list, no test-plan list, no bare at-sign (glue:
      `opmodel.dev/core@v2`). Verify: mention-guard green.
- [ ] R3 SUPERVISOR (merge time): squash-merge with `--subject "fix(deps): pin core
      v2.0.0-beta.1 (#N)"` and exactly this body (auto-filled COMMIT_MESSAGES body discarded):

      ```text
      The kernel's default schema, every served fixture and the release line move to
      core's first beta, which carries the alpha.13 schema unchanged. The render floor
      stays 2.0.0-alpha.12.

      Release-As: 1.0.0-beta.1
      Co-Authored-By: Claude <noreply@anthropic.com>
      ```

      Before merging: run the mention-guard regexes over subject and body; confirm no body line
      starts with an identifier followed by `(`; parse subject plus body with release-please's
      parser and require type `fix`, scope `deps` and RELEASE_AS `1.0.0-beta.1`.
- [ ] R4 SUPERVISOR (after merge): `git show -s --format=%B` of the merge commit shows the
      footer; within one `release.yml` run, #151 reads `chore(main): release 1.0.0-beta.1` and
      lists this change; list every `autorelease: pending` PR. No PR, or an alpha title, is a
      failure: edit the merged PR body with a BEGIN_COMMIT_OVERRIDE block (conventional header
      plus the footer) and re-run the workflow.
- [ ] R5 SUPERVISOR: merge the retitled #151 (plain squash, never `--admin`), confirm tag
      `v1.0.0-beta.1`, manifest `1.0.0-beta.1`, and `go list -m
      github.com/open-platform-model/library@v1.0.0-beta.1` resolves; tick G2.
