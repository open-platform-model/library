## ADDED Requirements

### Requirement: The library publishes one docs bundle

The repository SHALL declare one docs-kit project, `library`, in `docs-kit.cue`: placed in a site version's `/docs/` tree, owning `reference/go-api/`, versioned from tags with the prefix `v`, built from a `go-api` source over the packages matching `./opm/...` and a `markdown` source over `docs/site`. The bundle SHALL carry the authored pages under `docs/site/` and the Go API reference together (docs-kit DESIGN decision 20). `task docs:bundle` SHALL build it into `out/library/` and `task docs:bundle:check` SHALL build and lint it without publishing.

#### Scenario: The bundle holds both kinds of page

- **WHEN** `task docs:bundle` runs on a clean checkout
- **THEN** `out/library/content/` holds `diagnostics/version-skew.md` (authored), `reference/go-api/_index.md` and `reference/go-api/kernel.md` (generated)

#### Scenario: Internal packages stay out

- **WHEN** the bundle is built
- **THEN** no page documents a package under `opm/internal/`

### Requirement: Every exported symbol of the Go API reference has prose

Every exported package, type, function, method and constant or variable group under `opm/` outside `internal/` SHALL carry a doc comment, and no exported doc comment SHALL cite an ADR (`ADR-NNN`) or an enhancement by the prose form "enhancement NNNN"; a rationale pointer of that kind SHALL live in an ordinary comment that is not a doc comment. Lists in a doc comment SHALL use Go doc-comment list syntax, so the generated page shows them as lists.

#### Scenario: The kernel's surface is a list

- **WHEN** `go doc ./opm/kernel` prints the package doc
- **THEN** the "Surface" section shows one item per operation, each starting with `-`

#### Scenario: An error type's message method is documented

- **WHEN** a reader opens the `errors` page of the Go API reference
- **THEN** `SkewError.Error` has a sentence of documentation, not only its declaration

#### Scenario: A maintainer pointer does not reach the page

- **WHEN** the bundle is built
- **THEN** no page under `reference/go-api/` contains the text `ADR-`

### Requirement: Every library release publishes its docs bundle

`release.yml` SHALL run a `publish-docs` job that calls docs-kit's `publish.yml` with `project: library`, `mode: release` and the release's tag, in the workflow run of the push that merged the release PR, only when release-please created a release.

#### Scenario: A release publishes its bundle

- **WHEN** the release PR for `v1.0.0-beta.2` merges
- **THEN** `publish-docs` publishes `ghcr.io/open-platform-model/docs/library` for `1.0.0-beta.2`, signed by the library's workflow

#### Scenario: A push without a release publishes no release bundle

- **WHEN** a commit lands on `main` and release-please creates no release
- **THEN** `publish-docs` is skipped and only `docs.yml`'s `edge` job publishes

### Requirement: Pull requests check the bundle and main publishes edge

`.github/workflows/docs.yml` SHALL run `publish.yml` in `check` mode on every pull request (permissions `contents: read`, `packages: read`), in `edge` mode on every push to `main`, and on `workflow_dispatch` in the `release` or `revision` mode with a `tag` and, for `revision`, a `fix` commit; the publishing jobs SHALL hold only `contents: read`, `packages: write` and `id-token: write`, and the workflow SHALL declare `permissions: {}` at the top.

#### Scenario: A link into the reference that names no page fails the check

- **WHEN** a pull request adds a link to `/docs/reference/go-api/kernels/` to a page under `docs/site/`
- **THEN** the `Docs / check` job's bundle-mode lint fails, naming the page and the link, because `reference/go-api/` is owned by this bundle and holds no such page

#### Scenario: A release without a bundle is recovered

- **WHEN** the owner dispatches `docs.yml` with `mode: release` and `tag: v1.0.0-beta.1`
- **THEN** the job builds the tag's `opm/` and `docs/site/` with `main`'s `docs-kit.cue` and publishes `docs/library` for `1.0.0-beta.1`

### Requirement: The docs-kit release is pinned once per form and the forms agree

`.opm-docs-version` SHALL hold one docs-kit release tag, and every `open-platform-model/docs-kit/.github/workflows/publish.yml@<ref>` under `.github/workflows/` SHALL name that same tag, never a SHA (docs-kit C5, C9). `task docs:pins:check` SHALL refuse a disagreement offline, and `task docs:bundle:check` SHALL run it first.

#### Scenario: A half-moved pin is refused

- **WHEN** `.opm-docs-version` names `v0.4.0` and `release.yml` still names `publish.yml@v0.3.0`
- **THEN** `task docs:pins:check` fails, listing the stale ref

#### Scenario: Agreeing pins pass

- **WHEN** both forms name `v0.4.0`
- **THEN** `task docs:pins:check` passes
