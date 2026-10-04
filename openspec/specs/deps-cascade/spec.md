# deps-cascade Specification

## Purpose
The library's release cascade task, `task -x deps:cascade`: how it moves the library's two upstream pins (the core behind `DefaultSchemaModule` and the opm catalog) in the working tree only, reports the result by exit code, honours `.cascade-frozen` and `.cascade-hold`, keeps text-pinned test trees on the loader's core, never touches release or steering files, takes its PR title and body from the shared resolver, and is tested offline against the resolver stub and over the network against the real resolver.

## Requirements

### Requirement: The cascade task reports its result by exit code

`task -x deps:cascade` SHALL move the library's upstream pins in the working tree only. It SHALL NOT commit, branch or push. It SHALL exit 0 when the working tree changed, 3 when nothing changed, and any other code on error. It SHALL refuse a dirty working tree with exit 1 unless `CASCADE_ALLOW_DIRTY=1` is set, in which case it SHALL judge "changed" against a snapshot taken at start. A failing resolver call, `cue mod get`, `cue mod tidy` or file edit SHALL end the task with a non-zero code other than 3. The task SHALL NOT fall back to an older or guessed version, and SHALL NOT discard the failing command's output. Every resolver call that decides a target SHALL run before the first file edit, so a resolver error leaves the tree unchanged. Source: workspace `RELEASING.md`, section "The receiver".

#### Scenario: Nothing to move

- **WHEN** both pins already equal the newest published in-major versions and the render trees equal `DefaultSchemaModule`
- **THEN** the task exits 3 and `git status --porcelain` is empty

#### Scenario: Resolver error

- **WHEN** the resolver fails on the first pin it resolves
- **THEN** the task exits with a code other than 0 and 3, and no file in the tree changed

#### Scenario: Dirty tree

- **WHEN** the tree has an untracked file and `CASCADE_ALLOW_DIRTY` is unset
- **THEN** the task exits 1 and changes nothing

### Requirement: Core moves with DefaultSchemaModule and is labelled for human review

The task SHALL move the core pin `DefaultSchemaModule` in `opm/schema/loader.go` to the newest published `opmodel.dev/core@v2` release the resolver returns. It SHALL edit only that string literal. It SHALL NOT edit any other Go file. Prereleases SHALL count while the current pin is a prerelease. The pin SHALL NOT move backwards, SHALL NOT cross a major, and SHALL stop at an in-date `.cascade-hold` `max`. When `.cascade-frozen` lists `opm/schema/loader.go` for `opmodel.dev/core@v2`, the pin SHALL stay, and every later step, the catalog check included, SHALL use that unchanged core. When it moves, the task SHALL record a warning that names the move and `need-human-review`. The pin report SHALL carry the label `need-human-review` on the core row, so the PR body's `cascade-labels` marker names that label whenever core moved. Source: owner selection 9 ("Bot proposes, glue review", label `need-human-review`); workspace `RELEASING.md`, section "Labels".

#### Scenario: A newer core is published

- **WHEN** `DefaultSchemaModule` names `v2.0.0-beta.1` and the resolver returns `v2.0.0-beta.2`
- **THEN** the loader names `opmodel.dev/core@v2.0.0-beta.2`, no other Go file changed, and the body's `cascade-labels` marker lists `need-human-review`

#### Scenario: The resolver keeps core where it is

- **WHEN** the resolver answers exit 3 for `opmodel.dev/core@v2`, as it does when an in-date `.cascade-hold` caps core at the current value
- **THEN** the loader is unchanged

#### Scenario: A frozen loader holds back the catalog

- **WHEN** `.cascade-frozen` lists `opm/schema/loader.go` for `opmodel.dev/core@v2`, a newer core is published, and the newest catalog pins that newer core
- **THEN** the loader and every catalog pin are unchanged, the task exits 3, and the warnings file says the loader is frozen and that core must advance first

#### Scenario: The resolver warns of a new core major

- **WHEN** the resolver answers exit 3 for `opmodel.dev/core@v2` and emits a warning naming a new major
- **THEN** the loader stays on `opmodel.dev/core@v2` and the warning reaches the cascade warnings file under the key `opmodel.dev/core@v2`

### Requirement: Text-pinned test trees follow the loader

The task SHALL set the core `v:` of `testdata/cue.mod/module.cue` and of every `testdata/render/**/cue.mod/module.cue` that a walk of `testdata/render` finds to the version `DefaultSchemaModule` names after the core step. It SHALL rewrite only the `v:` inside the `"opmodel.dev/core@v2"` block. It SHALL NOT change any other dependency in those files, including the synthetic `testing.opmodel.dev/library-render/*` pins, and SHALL NOT run `cue mod get` or `cue mod tidy` in them. A file that `.cascade-frozen` lists for `opmodel.dev/core@v2` SHALL be left byte-unchanged.

#### Scenario: Every render tree follows

- **WHEN** core moves and a render tree is added under `testdata/render` that no list names
- **THEN** that tree's core pin equals the new loader value after the run, and `TestDefaultCoreVersion_IsTheDefaultSchemaRelease` passes

#### Scenario: A lagging tree catches up

- **WHEN** core does not move but one render tree pins an older core than the loader
- **THEN** that tree is re-pinned to the loader value and the task exits 0

### Requirement: The opm catalog moves only as far as the loader's core allows

The task SHALL move `opmodel.dev/catalogs/opm@v4`, read from `testdata/parity/cue.mod/module.cue`, to the newest published v4 release the resolver returns. Prereleases SHALL NOT count while the current pin is a release. When the core that catalog release pins is newer than `DefaultSchemaModule`, the catalog SHALL stay and the task SHALL warn that core must advance first. Each module of the `CUE_MODULE_GLOBS` set SHALL be in scope when its core differs from the loader value, or when it pins the catalog below the target. For each module in scope, the task SHALL run one `cue mod get` that names core at the loader value and the catalog at the exact target, followed by one `cue mod tidy`. A module where nothing moved SHALL be left untouched. A catalog pin above the target SHALL never be lowered. The catalog SHALL be named in a module's get only when that module's catalog is below the target. Third-party dependencies SHALL NOT be named. When tidy raises one, that SHALL be a warning, not a revert. A key that `.cascade-frozen` lists for a module's `cue.mod/module.cue` SHALL be left out of the get, and the task SHALL exit 1 if tidy changed it anyway.

#### Scenario: Catalog-only move

- **WHEN** core is current and a newer catalog is published whose own core pin is not newer than the loader
- **THEN** the four catalog-pinning modules move to that catalog, the loader is unchanged, and the title the resolver computes from the diff is `test(fixtures): ...`

#### Scenario: Catalog needs a newer core

- **WHEN** the newest catalog pins a core newer than `DefaultSchemaModule`
- **THEN** no catalog pin moves, and the warnings say to advance core first

#### Scenario: Frozen module

- **WHEN** `.cascade-frozen` lists `testdata/modules/web_app/cue.mod/module.cue` for both OPM keys and both pins move
- **THEN** that file is byte-unchanged and every other module moves

#### Scenario: Tidy raises a frozen key

- **WHEN** `.cascade-frozen` lists a module's `cue.mod/module.cue` for `opmodel.dev/core@v2` only, and moving its catalog makes `tidy` raise that core
- **THEN** the task exits with a code other than 0 and 3, naming the file and the frozen key

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

### Requirement: Title and body come from the shared resolver

`task -x deps:cascade:title` and `task -x deps:cascade:body` SHALL call the shared resolver's `title` and `body` with the repo's `.tasks/cascade/classes` and `.tasks/cascade/pins.sh`. The class map SHALL classify `testdata/`, `modules/` and `*_test.go` as `test`, and every other path as `shipped`. `pins.sh` SHALL report two pins:

- `opmodel.dev/core@v2`, read from the loader, class `shipped`, label `need-human-review`;
- `opmodel.dev/catalogs/opm@v4`, read from the parity module file, class `test`.

`pins.sh` SHALL read them from the working tree or from a git ref without running `cue`.

#### Scenario: Core and catalog moved

- **WHEN** the run moved both pins
- **THEN** the title is `fix(deps): bump core to <core> and opm catalog to <catalog>`, and the body lists two moved pins and the label `need-human-review`

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
