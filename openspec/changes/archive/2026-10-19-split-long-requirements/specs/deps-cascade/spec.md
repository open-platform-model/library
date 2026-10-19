## MODIFIED Requirements

### Requirement: The docs examples follow the loader's core
In a run that moves `DefaultSchemaModule`, the task SHALL rewrite the version in every `Module: "opmodel.dev/core@<v>"` literal in `docs/getting-started.md` and `AGENTS.md` whose `<v>` is a full release of the loader's major (`vMAJOR.MINOR.PATCH` with an optional prerelease, and nothing between the version and the closing quote) to the new loader value. Source: workspace `RELEASING.md`, sections "What each repo's task moves" and "Title from diff class".

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

## REMOVED Requirements

### Requirement: The cascade task reports its result by exit code
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The cascade task changes the working tree only and reports its result by exit code", "The cascade task refuses a dirty working tree unless told otherwise" and "A failing step ends the cascade task with a non-zero code other than 3".

### Requirement: Core moves with DefaultSchemaModule and is labelled for human review
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The core pin moves to the newest published release the resolver returns", "The core pin never moves backwards or across a major, stops at a hold and counts prereleases while it is a prerelease", "A frozen loader keeps the core pin and every later step uses that core", "A core move records a warning that names it and need-human-review" and "The pin report labels the core row need-human-review".

### Requirement: Text-pinned test trees follow the loader
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The text-pinned test trees take the core version DefaultSchemaModule names" and "The text-pinned test trees keep every other dependency and run no cue command".

### Requirement: The opm catalog moves only as far as the loader's core allows
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The opm catalog moves to the newest published v4 release the resolver returns", "The catalog stays when the core its release pins is newer than the loader's", "A module is in scope when its core differs from the loader value or its catalog is below the target", "One cue mod get and one cue mod tidy move a module in scope and name the catalog only when it is below the target", "The get names no third-party dependency and a dependency that tidy raises is a warning" and "A frozen key is left out of the get and the task exits 1 if tidy changed it".

### Requirement: The cascade task never touches release or steering files
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The cascade task modifies no release or steering file", "The cascade task publishes to no registry", "The cascade task warns of a newer language version and edits nothing" and "The cascade task warns when the docs name a core release other than the loader's".

### Requirement: Title and body come from the shared resolver
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Title and body come from the shared resolver with the repo's class map" and "pins.sh reports the core and catalog pins and reads them without running cue".

### Requirement: The cascade task is tested offline and over the network
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The cascade test task runs against the canonical resolver stub and passes only when every selected scenario passes", "The offline set needs no network and runs in the required CI job, and a non-required workflow runs the full set", "The lagging-docs scenario expects exit 3, an unchanged tree and a warning naming the file", "The full set runs the older-pins scenario", "The full set runs the frozen-module scenario" and "The full set runs the title and body scenarios when a real resolver is given".

## ADDED Requirements

### Requirement: The cascade task changes the working tree only and reports its result by exit code
`task -x deps:cascade` SHALL move the library's upstream pins in the working tree only. It SHALL NOT commit, branch or push. It SHALL exit 0 when the working tree changed, 3 when nothing changed, and any other code on error. Source: workspace `RELEASING.md`, section "The receiver".

#### Scenario: Nothing to move

- **WHEN** both pins already equal the newest published in-major versions and the render trees equal `DefaultSchemaModule`
- **THEN** the task exits 3 and `git status --porcelain` is empty

### Requirement: The cascade task refuses a dirty working tree unless told otherwise
`task -x deps:cascade` SHALL refuse a dirty working tree with exit 1 unless `CASCADE_ALLOW_DIRTY=1` is set, in which case it SHALL judge "changed" against a snapshot taken at start. Source: workspace `RELEASING.md`, section "The receiver".

#### Scenario: Dirty tree

- **WHEN** the tree has an untracked file and `CASCADE_ALLOW_DIRTY` is unset
- **THEN** the task exits 1 and changes nothing

### Requirement: A failing step ends the cascade task with a non-zero code other than 3
A failing resolver call, `cue mod get`, `cue mod tidy` or file edit SHALL end the task with a non-zero code other than 3. The task SHALL NOT fall back to an older or guessed version, and SHALL NOT discard the failing command's output. Every resolver call that decides a target SHALL run before the first file edit, so a resolver error leaves the tree unchanged. Source: workspace `RELEASING.md`, section "The receiver".

#### Scenario: Resolver error

- **WHEN** the resolver fails on the first pin it resolves
- **THEN** the task exits with a code other than 0 and 3, and no file in the tree changed

### Requirement: The core pin moves to the newest published release the resolver returns
The task SHALL move the core pin `DefaultSchemaModule` in `opm/schema/loader.go` to the newest published `opmodel.dev/core@v2` release the resolver returns. It SHALL edit only that string literal. It SHALL NOT edit any other Go file. Source: owner selection 9 ("Bot proposes, glue review", label `need-human-review`); workspace `RELEASING.md`, section "Labels".

#### Scenario: A newer core is published

- **WHEN** `DefaultSchemaModule` names `v2.0.0-beta.1` and the resolver returns `v2.0.0-beta.2`
- **THEN** the loader names `opmodel.dev/core@v2.0.0-beta.2`, no other Go file changed, and the body's `cascade-labels` marker lists `need-human-review`

### Requirement: The core pin never moves backwards or across a major, stops at a hold and counts prereleases while it is a prerelease
The task SHALL move the core pin `DefaultSchemaModule` in `opm/schema/loader.go` to the newest published `opmodel.dev/core@v2` release the resolver returns. Prereleases SHALL count while the current pin is a prerelease. The pin SHALL NOT move backwards, SHALL NOT cross a major, and SHALL stop at an in-date `.cascade-hold` `max`. Source: owner selection 9 ("Bot proposes, glue review", label `need-human-review`); workspace `RELEASING.md`, section "Labels".

#### Scenario: The resolver keeps core where it is

- **WHEN** the resolver answers exit 3 for `opmodel.dev/core@v2`, as it does when an in-date `.cascade-hold` caps core at the current value
- **THEN** the loader is unchanged

#### Scenario: The resolver warns of a new core major

- **WHEN** the resolver answers exit 3 for `opmodel.dev/core@v2` and emits a warning naming a new major
- **THEN** the loader stays on `opmodel.dev/core@v2` and the warning reaches the cascade warnings file under the key `opmodel.dev/core@v2`

### Requirement: A frozen loader keeps the core pin and every later step uses that core
The task SHALL move the core pin `DefaultSchemaModule` in `opm/schema/loader.go` to the newest published `opmodel.dev/core@v2` release the resolver returns. When `.cascade-frozen` lists `opm/schema/loader.go` for `opmodel.dev/core@v2`, the pin SHALL stay, and every later step, the catalog check included, SHALL use that unchanged core. Source: owner selection 9 ("Bot proposes, glue review", label `need-human-review`); workspace `RELEASING.md`, section "Labels".

#### Scenario: A frozen loader holds back the catalog

- **WHEN** `.cascade-frozen` lists `opm/schema/loader.go` for `opmodel.dev/core@v2`, a newer core is published, and the newest catalog pins that newer core
- **THEN** the loader and every catalog pin are unchanged, the task exits 3, and the warnings file says the loader is frozen and that core must advance first

### Requirement: A core move records a warning that names it and need-human-review
The task SHALL move the core pin `DefaultSchemaModule` in `opm/schema/loader.go` to the newest published `opmodel.dev/core@v2` release the resolver returns. When it moves, the task SHALL record a warning that names the move and `need-human-review`. Source: owner selection 9 ("Bot proposes, glue review", label `need-human-review`); workspace `RELEASING.md`, section "Labels".

#### Scenario: A core move is recorded in a warning

- **WHEN** the core pin moves
- **THEN** the task records a warning that names the move and `need-human-review`

### Requirement: The pin report labels the core row need-human-review
The pin report (Requirement: pins.sh reports the core and catalog pins and reads them without running cue) SHALL carry the label `need-human-review` on the core row, so the PR body's `cascade-labels` marker names that label whenever core moved. Source: owner selection 9 ("Bot proposes, glue review", label `need-human-review`); workspace `RELEASING.md`, section "Labels".

#### Scenario: The core row carries the label

- **WHEN** core moved
- **THEN** the pin report carries the label `need-human-review` on the core row, so the PR body's `cascade-labels` marker names that label

### Requirement: The text-pinned test trees take the core version DefaultSchemaModule names
The task SHALL set the core `v:` of `testdata/cue.mod/module.cue` and of every `testdata/render/**/cue.mod/module.cue` that a walk of `testdata/render` finds to the version `DefaultSchemaModule` names after the core step. It SHALL rewrite only the `v:` inside the `"opmodel.dev/core@v2"` block. A file that `.cascade-frozen` lists for `opmodel.dev/core@v2` SHALL be left byte-unchanged.

#### Scenario: Every render tree follows

- **WHEN** core moves and a render tree is added under `testdata/render` that no list names
- **THEN** that tree's core pin equals the new loader value after the run, and `TestDefaultCoreVersion_IsTheDefaultSchemaRelease` passes

#### Scenario: A lagging tree catches up

- **WHEN** core does not move but one render tree pins an older core than the loader
- **THEN** that tree is re-pinned to the loader value and the task exits 0

### Requirement: The text-pinned test trees keep every other dependency and run no cue command
The task SHALL set the core `v:` of `testdata/cue.mod/module.cue` and of every `testdata/render/**/cue.mod/module.cue` that a walk of `testdata/render` finds to the version `DefaultSchemaModule` names after the core step. It SHALL NOT change any other dependency in those files, including the synthetic `testing.opmodel.dev/library-render/*` pins, and SHALL NOT run `cue mod get` or `cue mod tidy` in them.

#### Scenario: A render tree keeps its other dependencies

- **WHEN** the task sets the core `v:` of a render tree
- **THEN** it does not change any other dependency, including the synthetic `testing.opmodel.dev/library-render/*` pins, and does not run `cue mod get` or `cue mod tidy` in it

### Requirement: The opm catalog moves to the newest published v4 release the resolver returns
The task SHALL move `opmodel.dev/catalogs/opm@v4`, read from `testdata/parity/cue.mod/module.cue`, to the newest published v4 release the resolver returns. Prereleases SHALL NOT count while the current pin is a release. A catalog pin above the target SHALL never be lowered.

#### Scenario: Catalog-only move

- **WHEN** core is current and a newer catalog is published whose own core pin is not newer than the loader
- **THEN** the four catalog-pinning modules move to that catalog, the loader is unchanged, and the title the resolver computes from the diff is `test(fixtures): ...`

### Requirement: The catalog stays when the core its release pins is newer than the loader's
The task SHALL move `opmodel.dev/catalogs/opm@v4`, read from `testdata/parity/cue.mod/module.cue`, to the newest published v4 release the resolver returns. When the core that catalog release pins is newer than `DefaultSchemaModule`, the catalog SHALL stay and the task SHALL warn that core must advance first.

#### Scenario: Catalog needs a newer core

- **WHEN** the newest catalog pins a core newer than `DefaultSchemaModule`
- **THEN** no catalog pin moves, and the warnings say to advance core first

### Requirement: A module is in scope when its core differs from the loader value or its catalog is below the target
The task SHALL move `opmodel.dev/catalogs/opm@v4`, read from `testdata/parity/cue.mod/module.cue`, to the newest published v4 release the resolver returns. Each module of the `CUE_MODULE_GLOBS` set SHALL be in scope when its core differs from the loader value, or when it pins the catalog below the target. A module where nothing moved SHALL be left untouched.

#### Scenario: A module whose core differs is in scope

- **WHEN** a module of the `CUE_MODULE_GLOBS` set has a core that differs from the loader value
- **THEN** the module is in scope

#### Scenario: A module that pins the catalog below the target is in scope

- **WHEN** a module of the `CUE_MODULE_GLOBS` set pins the catalog below the target
- **THEN** the module is in scope

#### Scenario: A module where nothing moved is left untouched

- **WHEN** nothing moved in a module
- **THEN** the module is left untouched

### Requirement: One cue mod get and one cue mod tidy move a module in scope and name the catalog only when it is below the target
For each module in scope, the task SHALL run one `cue mod get` that names core at the loader value and the catalog at the exact target, followed by one `cue mod tidy`. The catalog SHALL be named in a module's get only when that module's catalog is below the target. The target and the modules in scope are those of requirement "A module is in scope when its core differs from the loader value or its catalog is below the target".

#### Scenario: One get and one tidy per module in scope

- **WHEN** a module is in scope
- **THEN** the task runs one `cue mod get` that names core at the loader value and the catalog at the exact target, followed by one `cue mod tidy`
- **AND** the catalog is named in the module's get only when that module's catalog is below the target

### Requirement: The get names no third-party dependency and a dependency that tidy raises is a warning
For each module in scope, the task SHALL run one `cue mod get` that names core at the loader value and the catalog at the exact target, followed by one `cue mod tidy`. Third-party dependencies SHALL NOT be named. When tidy raises one, that SHALL be a warning, not a revert. The target and the modules in scope are those of requirement "A module is in scope when its core differs from the loader value or its catalog is below the target".

#### Scenario: A third-party dependency is not named

- **WHEN** a module is in scope
- **THEN** the `cue mod get` of the task does not name a third-party dependency
- **AND** when tidy raises a third-party dependency, that is a warning, not a revert

### Requirement: A frozen key is left out of the get and the task exits 1 if tidy changed it
For each module in scope, the task SHALL run one `cue mod get` that names core at the loader value and the catalog at the exact target, followed by one `cue mod tidy`. A key that `.cascade-frozen` lists for a module's `cue.mod/module.cue` SHALL be left out of the get, and the task SHALL exit 1 if tidy changed it anyway. The target and the modules in scope are those of requirement "A module is in scope when its core differs from the loader value or its catalog is below the target".

#### Scenario: Frozen module

- **WHEN** `.cascade-frozen` lists `testdata/modules/web_app/cue.mod/module.cue` for both OPM keys and both pins move
- **THEN** that file is byte-unchanged and every other module moves

#### Scenario: Tidy raises a frozen key

- **WHEN** `.cascade-frozen` lists a module's `cue.mod/module.cue` for `opmodel.dev/core@v2` only, and moving its catalog makes `tidy` raise that core
- **THEN** the task exits with a code other than 0 and 3, naming the file and the frozen key

### Requirement: The cascade task modifies no release or steering file
The task SHALL NOT modify these:

- `.cascade-frozen` and `.cascade-hold`;
- `.release-please-manifest.json`, `release-please-config.json` and `CHANGELOG.md`;
- anything under `.github/`;
- `.opm-docs-version` and `docs-kit.cue`;
- `cue-versions.yml`;
- any `language.version`.

#### Scenario: No release or steering file changes

- **WHEN** the task runs
- **THEN** it modifies none of `.cascade-frozen` and `.cascade-hold`, `.release-please-manifest.json`, `release-please-config.json` and `CHANGELOG.md`, anything under `.github/`, `.opm-docs-version` and `docs-kit.cue`, `cue-versions.yml`, or any `language.version`

### Requirement: The cascade task publishes to no registry
The task SHALL NOT publish to any registry.

#### Scenario: No publish to a registry

- **WHEN** the task runs
- **THEN** it does not publish to any registry

### Requirement: The cascade task warns of a newer language version and edits nothing
The task SHALL warn, without editing, when an upstream it moved declares a `language.version` newer than the CUE version that `.github/workflows/cue.yml` installs.

#### Scenario: A newer language version only warns

- **WHEN** an upstream the task moved declares a `language.version` newer than the CUE version that `.github/workflows/cue.yml` installs
- **THEN** the task warns and edits nothing

### Requirement: The cascade task warns when the docs name a core release other than the loader's
The task SHALL warn when `docs/getting-started.md` or `AGENTS.md` still names a core release other than the loader's, build metadata included, after the docs examples step ("The docs examples follow the loader's core"), which edits only the `Module: "…"` examples in those files and only in a run that moves core.

#### Scenario: Stale getting-started prose

- **WHEN** core does not move and `docs/getting-started.md` names a core release other than the loader's
- **THEN** the file is unchanged and the warnings name it

### Requirement: Title and body come from the shared resolver with the repo's class map
`task -x deps:cascade:title` and `task -x deps:cascade:body` SHALL call the shared resolver's `title` and `body` with the repo's `.tasks/cascade/classes` and `.tasks/cascade/pins.sh`. The class map SHALL classify `testdata/`, `modules/` and `*_test.go` as `test`, and every other path as `shipped`.

#### Scenario: Core and catalog moved

- **WHEN** the run moved both pins
- **THEN** the title is `fix(deps): bump core to <core> and opm catalog to <catalog>`, and the body lists two moved pins and the label `need-human-review`

### Requirement: pins.sh reports the core and catalog pins and reads them without running cue
`pins.sh` SHALL report two pins:

- `opmodel.dev/core@v2`, read from the loader, class `shipped`, label `need-human-review`;
- `opmodel.dev/catalogs/opm@v4`, read from the parity module file, class `test`.

`pins.sh` SHALL read them from the working tree or from a git ref without running `cue`.

#### Scenario: pins.sh reports two pins

- **WHEN** `pins.sh` runs
- **THEN** it reports `opmodel.dev/core@v2`, read from the loader, class `shipped`, label `need-human-review`, and `opmodel.dev/catalogs/opm@v4`, read from the parity module file, class `test`
- **AND** it reads them from the working tree or from a git ref without running `cue`

### Requirement: The cascade test task runs against the canonical resolver stub and passes only when every selected scenario passes
`task -x deps:cascade:test` SHALL run the task in throwaway copies of the tree against the canonical resolver stub. It SHALL assert the stub's checksum. It SHALL exit 0 only when every selected scenario passes.

#### Scenario: The stub checksum is asserted

- **WHEN** `task -x deps:cascade:test` runs
- **THEN** it runs the task in throwaway copies of the tree against the canonical resolver stub and asserts the stub's checksum
- **AND** it exits 0 only when every selected scenario passes

### Requirement: The offline set needs no network and runs in the required CI job, and a non-required workflow runs the full set
`task -x deps:cascade:test` SHALL run the task in throwaway copies of the tree against the canonical resolver stub. With `CASCADE_TEST_SET=offline`, it SHALL run the no-op, resolver-error, dirty-tree, lagging-tree, catalog-needs-core, frozen-loader, frozen-lag-needs-no-cue and lagging-docs scenarios, which need no network, and the required `Go tests` CI job SHALL run that set. A non-required workflow SHALL run the full set.

#### Scenario: The offline set needs no network

- **WHEN** `CASCADE_TEST_SET=offline` is set
- **THEN** the test task runs the no-op, resolver-error, dirty-tree, lagging-tree, catalog-needs-core, frozen-loader, frozen-lag-needs-no-cue and lagging-docs scenarios, which need no network
- **AND** the required `Go tests` CI job runs that set

### Requirement: The lagging-docs scenario expects exit 3, an unchanged tree and a warning naming the file
`task -x deps:cascade:test` SHALL run the task in throwaway copies of the tree against the canonical resolver stub. The lagging-docs scenario SHALL give the example in `docs/getting-started.md` an older same-major core while core does not move, and SHALL expect exit 3, an unchanged tree and a warning naming the file.

#### Scenario: Lagging docs without a core move

- **WHEN** the lagging-docs scenario runs offline
- **THEN** the task exits 3, `git status --porcelain` is empty, and a warning names `docs/getting-started.md`

### Requirement: The full set runs the older-pins scenario
`task -x deps:cascade:test` SHALL run the task in throwaway copies of the tree against the canonical resolver stub. With the full set, it SHALL also run these scenarios:

- **older pins:** every location set to an older published version, the two docs examples included, with the example in `docs/getting-started.md` set older still; the run's result equal to the current tree, no warning naming either docs file, and a second run that exits 3.

#### Scenario: Idempotent catch-up

- **WHEN** the older-pins scenario runs the task twice without resetting the base
- **THEN** the first run exits 0 with a tree equal to the original, and the second exits 3 with no further change

### Requirement: The full set runs the frozen-module scenario
`task -x deps:cascade:test` SHALL run the task in throwaway copies of the tree against the canonical resolver stub. With the full set, it SHALL also run these scenarios:

- **frozen module:** a module listed in `.cascade-frozen` stays byte-unchanged; in the same run, a `v1` release and an older `v2` release named in `AGENTS.md` prose stay unchanged and are warned about.

#### Scenario: A frozen module stays byte-unchanged

- **WHEN** the frozen-module scenario runs
- **THEN** a module listed in `.cascade-frozen` stays byte-unchanged; in the same run, a `v1` release and an older `v2` release named in `AGENTS.md` prose stay unchanged and are warned about

### Requirement: The full set runs the title and body scenarios when a real resolver is given
`task -x deps:cascade:test` SHALL run the task in throwaway copies of the tree against the canonical resolver stub. With the full set, it SHALL also run these scenarios:

- **title and body**, when a real resolver is given, for a core-and-catalog move and for a catalog-only move.

#### Scenario: Title and body are checked with a real resolver

- **WHEN** a real resolver is given
- **THEN** the test task runs the title and body scenarios for a core-and-catalog move and for a catalog-only move

### Requirement: The docs examples change nothing else in the docs files
The task SHALL change nothing else in those files: a core release named anywhere else, in prose included, is never rewritten. The files are those of requirement "The docs examples follow the loader's core". Source: workspace `RELEASING.md`, sections "What each repo's task moves" and "Title from diff class".

#### Scenario: Nothing else in the docs files is rewritten

- **WHEN** core moves and a core release is named anywhere else in the docs files, in prose included
- **THEN** that release is never rewritten

### Requirement: The docs examples are not edited in a run that does not move DefaultSchemaModule
The task SHALL NOT edit either file in a run that does not move `DefaultSchemaModule`, a run with a frozen loader included, so a docs lag on its own never produces a diff. The files are those of requirement "The docs examples follow the loader's core". Source: workspace `RELEASING.md`, sections "What each repo's task moves" and "Title from diff class".

#### Scenario: A run without a core move produces no docs diff

- **WHEN** a run does not move `DefaultSchemaModule`, a run with a frozen loader included
- **THEN** neither file is edited, so a docs lag on its own never produces a diff

### Requirement: The docs examples leave a release of another major alone
The task SHALL NOT rewrite a release of another major. The literals are those of requirement "The docs examples follow the loader's core". Source: workspace `RELEASING.md`, sections "What each repo's task moves" and "Title from diff class".

#### Scenario: A release of another major is not rewritten

- **WHEN** core moves within `v2`
- **THEN** the task does not rewrite a release of another major

### Requirement: A frozen docs file stays byte-unchanged
A file that `.cascade-frozen` lists for `opmodel.dev/core@v2` SHALL be left byte-unchanged. The files are those of requirement "The docs examples follow the loader's core". Source: workspace `RELEASING.md`, sections "What each repo's task moves" and "Title from diff class".

#### Scenario: A frozen docs file is left byte-unchanged

- **WHEN** `.cascade-frozen` lists a file for `opmodel.dev/core@v2`
- **THEN** that file is left byte-unchanged
