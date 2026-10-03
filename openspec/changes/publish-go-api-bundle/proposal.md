## Why

The library's public Go API has no reference a reader of opmodel.dev can open: today there is only `go doc`. docs-kit phase 2 publishes every product repository's pages as a signed docs bundle, and its `go-api` extractor (docs-kit contract C20) turns the exported packages under `opm/` into reference pages at `/docs/reference/go-api/`. The owner decided one cutover per repository (docs-kit DESIGN decision 20): the bundle carries the library's authored `docs/site/` (the diagnostics pages and "Embed the kernel") together with the generated reference from adoption on. The library commits no generated pages, so nothing needs excluding and nothing needs deleting later.

The pages print the doc comments as they are, so they must read well first: `opm/kernel/doc.go`'s "Surface" list is garbled (its items are not indented, so gofmt reflowed them into one paragraph with stray hyphens, lines 14-35), eleven exported methods and two constant groups have no doc comment, and twelve exported doc comments cite `ADR-NNN` or "enhancement NNNN", which neither of docs-kit's citation policies removes cleanly.

Sequence and contracts: docs-kit `docs/orchestration.md` (phase 2, "library: `publish-go-api-bundle`") and `https://github.com/open-platform-model/docs-kit/blob/main/docs/contracts.md` (C5, C6, C9, C12, C15, C20). Until a docs-kit change lands, its `openspec/changes/<change>/design.md` on docs-kit `main` shows the contract.

Delivery: one PR per section (cli needs the docs/library 1.0.0-beta.1 bundle after section 3)

## What Changes

- **Section 1, doc comments that read as pages (no gate).** Indent the kernel's "Surface" list; document the eleven `Error`/`Unwrap` methods and the two constant groups; move `ADR-NNN` and "enhancement NNNN" pointers out of exported doc comments into ordinary comments.
- **Section 2, adopt (gate G2-library).** `docs-kit.cue` declares the project `library`: docs placement owning `reference/go-api/`, a `go-api` source over `./opm/...` and a `markdown` source over `docs/site` (no `exclude`). `.opm-docs-version`, `.tasks/opm-docs.sh`, the tasks `tools:opm-docs`, `docs:bundle`, `docs:pins:check` and `docs:bundle:check` (in `task check`), `.github/workflows/docs.yml`, and `publish-docs` in `release.yml` after `release-please`. Any doc-comment problem docs-kit's trial build or the first bundle shows is fixed here. "Embed the kernel"'s Next-steps brief names `/docs/reference/go-api/kernel/`. A local dry run builds `v1.0.0-beta.1` with this config. `AGENTS.md` gains a "Docs bundles" paragraph.
- **Section 3, the bundle the cli pins (owner, part of gate G2-pins).** A dispatched backfill of `v1.0.0-beta.1`, the version the cli's `main` pins (owner decision 2026-10-03), plus `v1.0.0-beta.2` if that release is cut before section 2 merges; record the runs. No docs revision of beta.1: the comment fixes do not apply cleanly to the tag (design.md D5), so they reach the site with the next library release, which publishes through `publish-docs` on its own.
- **Section 4, the visible link (gate G2-switch).** "Embed the kernel" links the Go API reference as a real link once the site reads the library from bundles; archive.

## Capabilities

### New Capabilities

- `docs-bundle`: how the library's docs bundle is configured, checked and published, what the Go API reference covers, and how the docs-kit release is pinned.

### Modified Capabilities

None.

## Impact

**SemVer: no bump, no release of its own.** No exported identifier, signature or behavior changes; sections 1 and 4 are `docs:` commits and sections 2 and 3 `ci:`/`docs:`, all hidden from release-please. The public surface in `opm/` is unchanged; only its doc comments move. Principle VII: the change adds no Go code; the tooling is docs-kit's released binary.

**Affected packages (doc comments only):** `opm/kernel`, `opm/errors`, `opm/helper`, `opm/helper/platformmodule`, `opm/platform`, `opm/catalog`, `opm/schema`.

**Downstream consumers:**

| Consumer | What it has to do |
| --- | --- |
| cli | Nothing: its `main` already pins `v1.0.0-beta.1`, which section 3 backfills (gate G2-pins); no `go.mod` bump. |
| opmodel.dev | Nothing until `pull-reference-bundles`; then v1.0 gains `/docs/reference/go-api/` from the pinned library's bundle. |
| opm-operator | Nothing. |

**Owner items:** merging docs-kit's release PRs (gate G2-library), the `v1.0.0-beta.1` backfill dispatch (and `v1.0.0-beta.2`'s, if it is released first), checking `ghcr.io/open-platform-model/docs/library` is public on first push.
