## ADDED Requirements

### Requirement: The docs examples follow the loader's core

In a run that moves `DefaultSchemaModule`, the task SHALL rewrite the version in every `Module: "opmodel.dev/core@<v>"` literal in `docs/getting-started.md` and `AGENTS.md` whose `<v>` is a full release of the loader's major (`vMAJOR.MINOR.PATCH` with an optional prerelease, and nothing between the version and the closing quote) to the new loader value. It SHALL change nothing else in those files: a core release named anywhere else, in prose included, is never rewritten. It SHALL NOT edit either file in a run that does not move `DefaultSchemaModule`, a run with a frozen loader included, so a docs lag on its own never produces a diff. It SHALL NOT rewrite a release of another major. A file that `.cascade-frozen` lists for `opmodel.dev/core@v2` SHALL be left byte-unchanged. Source: workspace `RELEASING.md`, sections "What each repo's task moves" and "Title from diff class".

#### Scenario: Core moves and the examples follow

- **WHEN** `DefaultSchemaModule` and both examples name `v2.0.0-beta.1`, and the resolver returns `v2.0.0-beta.2`
- **THEN** both examples name `opmodel.dev/core@v2.0.0-beta.2`, no other line in either file changed, and no warning names either file

#### Scenario: A lagging example catches up on the next core move

- **WHEN** `DefaultSchemaModule` names `v2.0.0-beta.1`, the example in `docs/getting-started.md` names `v2.0.0-alpha.12`, and the resolver returns `v2.0.0-beta.2`
- **THEN** the example in `docs/getting-started.md` names `opmodel.dev/core@v2.0.0-beta.2`

#### Scenario: No core move, no docs edit

- **WHEN** the resolver keeps core where it is and the example in `docs/getting-started.md` names an older `v2` release
- **THEN** the task exits 3, the file is unchanged, and the warnings name it

#### Scenario: Another major is left alone

- **WHEN** core moves within `v2` and `AGENTS.md` names `opmodel.dev/core@v1.0.0`
- **THEN** that literal is unchanged and the warnings name `AGENTS.md`

#### Scenario: A release named in prose is left alone

- **WHEN** core moves within `v2` and a prose line in `AGENTS.md`, outside any `Module: "…"` literal, names an older `v2` release
- **THEN** that line is unchanged and the warnings name `AGENTS.md`

## MODIFIED Requirements

### Requirement: The cascade task never touches release or steering files

The task SHALL NOT modify these:

- `.cascade-frozen` and `.cascade-hold`;
- `.release-please-manifest.json`, `release-please-config.json` and `CHANGELOG.md`;
- anything under `.github/`;
- `.opm-docs-version` and `docs-kit.cue`;
- `cue-versions.yml`;
- any `language.version`.

It SHALL NOT publish to any registry. It SHALL warn, without editing, when an upstream it moved declares a `language.version` newer than the CUE version that `.github/workflows/cue.yml` installs. It SHALL warn when `docs/getting-started.md` or `AGENTS.md` still names a core release other than the loader's, build metadata included, after the docs examples step ("The docs examples follow the loader's core"), which edits only the `Module: "…"` examples in those files and only in a run that moves core.

#### Scenario: Stale getting-started prose

- **WHEN** core does not move and `docs/getting-started.md` names a core release other than the loader's
- **THEN** the file is unchanged and the warnings name it

### Requirement: The cascade task is tested offline and over the network

`task -x deps:cascade:test` SHALL run the task in throwaway copies of the tree against the canonical resolver stub. It SHALL assert the stub's checksum. It SHALL exit 0 only when every selected scenario passes. With `CASCADE_TEST_SET=offline`, it SHALL run the no-op, resolver-error, dirty-tree, lagging-tree, catalog-needs-core, frozen-loader, frozen-lag-needs-no-cue and lagging-docs scenarios, which need no network, and the required `Go tests` CI job SHALL run that set. The lagging-docs scenario SHALL give the example in `docs/getting-started.md` an older same-major core while core does not move, and SHALL expect exit 3, an unchanged tree and a warning naming the file. With the full set, it SHALL also run these scenarios:

- **older pins:** every location set to an older published version, the two docs examples included, with the example in `docs/getting-started.md` set older still; the run's result equal to the current tree, no warning naming either docs file, and a second run that exits 3;
- **frozen module:** a module listed in `.cascade-frozen` stays byte-unchanged; in the same run, a `v1` release and an older `v2` release named in `AGENTS.md` prose stay unchanged and are warned about;
- **title and body**, when a real resolver is given, for a core-and-catalog move and for a catalog-only move.

A non-required workflow SHALL run the full set.

#### Scenario: Idempotent catch-up

- **WHEN** the older-pins scenario runs the task twice without resetting the base
- **THEN** the first run exits 0 with a tree equal to the original, and the second exits 3 with no further change

#### Scenario: Lagging docs without a core move

- **WHEN** the lagging-docs scenario runs offline
- **THEN** the task exits 3, `git status --porcelain` is empty, and a warning names `docs/getting-started.md`
