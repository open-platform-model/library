## Context

The library releases with release-please (`release.yml`: one job, `release-please`, outputs `releases_created` and `tag_name`); a release is the git tag. Its site pages are `docs/site/diagnostics/*.md` (seven how-to pages) and `docs/site/embedding/embed-the-kernel.md` (a tutorial still made of authoring briefs). opmodel.dev's v1.0 reads them from `main` today (`opmodel.dev/site/versions.conf`, line version), and its build fails on a broken internal link (`site/config/_default/hugo.toml` `refLinksErrorLevel = 'error'`, the link render hook check).

docs-kit contracts read: C5, C6, C9, C12, C15 (docs placement, `owns`, the owned-path link rule of bundle-mode lint) and C20 (`go-api`: `go/parser` and `go/doc` without type checking, `go/doc/comment` to Markdown, Hugo default heading anchors, page per package named by its directory relative to `./opm` with `/` as `-`). The reference adopter is catalog_opm (`docs.yml`, `release.yml` `publish-docs`, `Taskfile.yml` docs tasks, `.tasks/opm-docs.sh`).

## Goals / Non-Goals

**Goals:** doc comments that read as reference pages; the library's docs bundle with its authored pages and its Go API reference; publishing on every release; one release bundle the cli can pin.

**Non-Goals:** any change to an exported identifier, signature or behavior; `Example*` functions (C20 leaves them out); writing the "Embed the kernel" tutorial; `internal/` packages (C20 skips them).

## Decisions

### D1. Doc comments (section 1)

Found by a `go/doc` walk of `opm/` (exported symbols with an empty doc, excluding `internal/`) and a grep for citation forms, at `cf79a5c`:

| Defect | Where | Fix |
| --- | --- | --- |
| "Surface" list reflowed into one paragraph with stray `-` | `opm/kernel/doc.go:14-35` | Go doc-comment list syntax: a blank `//` line, then each item as `//   - [Kernel.AcquireModuleFromRegistry] and ...`, so `gofmt` keeps the list and `go/doc/comment` prints it as a Markdown list |
| `Error` with no doc | `opm/errors/collision.go:51` (`ContractCollisionsError`), `:68` (`NotRoutableError`), `oversubscribed.go:52`, `coretooold.go:35`, `skew.go:21`, `domain.go:17` (`TransformError`), `unmatched.go:50`, `match.go:130` (`UnresolvedDemandsError`), `opm/kernel/render.go:275` (`RenderError`) | one sentence each: what the message holds (`Error returns one line per unresolved demand, ...`) |
| `Unwrap` with no doc | `opm/errors/domain.go:22`, `opm/kernel/render.go:277` | one sentence naming what it unwraps to |
| constant group with no group doc (each constant documented) | `opm/helper/platformmodule/generate.go:16`, `opm/kernel/render.go:26` | a one-line group doc, so the group's entry has prose as well as its declaration |
| `ADR-NNN` in an exported doc comment | `opm/kernel/doc.go:20,47,72,82`, `opm/kernel/acquire.go:136`, `opm/catalog/catalog.go:6`, `opm/helper/doc.go:9,37` | say the rule in the comment's own words; the ADR pointer moves to an ordinary comment after a blank line (not part of the doc comment, so neither `go doc` nor the page shows it) |
| "enhancement NNNN" prose citation | `opm/errors/oversubscribed.go:12` (`enhancement 0010:D32 as corrected by ...`), `opm/platform/platform.go:20`, `opm/platform/doc.go:15` | as above; a bare `NNNN:DN` may stay where the sentence still reads without it, because the default `strip` policy removes it (C6 "Doc-comment rules") |

`opm/schema/paths.go:66` cites `ADR-009` inside a function body, not a doc comment; it stays. Commit types: `docs(kernel)`, `docs(errors)` and so on per package, or one `docs:` commit for the section (library hides `docs`, so no release).

### D2. `docs-kit.cue` (section 2)

```cue
bundles: library: {
	placement: {kind: "docs", root: "/docs/", owns: ["reference/go-api/"]}
	version: {from: "tag", prefix: "v"}
	sources: [{
		kind:        "go-api"
		module:      "./"
		root:        "./opm"
		packages:    ["./opm/..."]
		section:     "reference/go-api/"
		title:       "Go API"
		description: "Every exported package of the OPM library, from its doc comments."
	}, {
		// The authored pages ship in the same bundle (docs-kit DESIGN decision
		// 20). The library commits no generated pages, so nothing is excluded.
		kind: "markdown", dir: "docs/site"
	}]
}
```

This yields nine package pages today (`catalog`, `errors`, `helper`, `helper-objectset`, `helper-platformmodule`, `kernel`, `module`, `platform`, `schema`) plus `_index.md`; `opm/internal/...` is skipped (C20 D2). `citations` stays `strip`: the library's comments cite decisions for maintainers, and a link would send a Go reader to the enhancements site mid-sentence. No `weight`: the section index sorts by title among the reference sections, which opmodel.dev owns.

### D3. Publishing (section 2)

`docs.yml` is catalog_opm's with `project: library` and tags `vX.Y.Z`. `release.yml` gains:

```yaml
  publish-docs:
    name: Publish the library docs bundle
    needs: release-please
    if: needs.release-please.outputs.releases_created == 'true'
    permissions:
      contents: read
      packages: write
      id-token: write
    # Pinned by docs-kit release tag, not a SHA: the signing certificate names
    # publish.yml at this ref, and the site trusts only docs-kit's v* tags
    # (docs-kit C5, C9). Moves with .opm-docs-version in one PR.
    uses: open-platform-model/docs-kit/.github/workflows/publish.yml@vX.Y.Z
    with:
      project: library
      mode: release
      tag: ${{ needs.release-please.outputs.tag_name }}
```

A library release is the tag itself, so `publish-docs` follows `release-please` directly. The workflow's top-level `permissions` (`contents: write`, `pull-requests: write`) stay; the job sets its own. Local tasks, `.tasks/opm-docs.sh`, `.gitignore` (`/out/`, `/.bin/`) as catalog_opm; `task check` and `task check:fast` both run `docs:bundle:check` last.

### D4. The link from "Embed the kernel"

The page's Next-steps section is an authoring brief that says "Reference: the Go API documentation ... (Verify: where it is published, for example pkg.go.dev, before linking)". Section 2 replaces that clause with "Reference: [Go API: opm/kernel](/docs/reference/go-api/kernel/)", inside the brief, so nothing visible changes while the site still reads the library from git. A visible link waits for section 4: until G2-switch the site builds this page from `main`, where `/docs/reference/go-api/` does not exist, and a broken internal link fails the site build. After G2-switch the page reaches the site only inside a library bundle, whose bundle-mode lint checks the link against the bundle's own `reference/go-api/` (C15), so it cannot ship broken.

### D5. Which library bundles the cli can pin

The cli's pins name a library version, and that library's `DefaultSchemaModule` names the core version (`opm/schema/loader.go:43`, `opmodel.dev/core@v2.0.0-beta.1` on `main` and at `v1.0.0-beta.1`). Gate G2-pins needs a bundle for all three. Section 3 therefore publishes two bundles: a dispatched backfill of `v1.0.0-beta.1` (its tree has `opm/` and `docs/site/`; `main`'s config builds it, C5), which serves only if core `v2.0.0-beta.1` also gets a bundle; and the next library release, which the release cascade cuts pinning core's first bundled release (core `publish-definitions-bundle` section 2). The backfill carries beta.1's garbled kernel doc; the fresh release carries section 1's fixes.

## Research & Decisions

### Citation forms docs-kit does not handle

**Context**: docs-kit's `strip` policy removes decision citations (`0010:D32`), OQ numbers, `SPEC.md § N` and experiment references; `link` turns decision citations into `/enhancements/NNNN/decisions/` links. Neither names `ADR-NNN` or the prose "enhancement NNNN"; the library's public doc comments hold nine and three of them.
**Explored**: docs-kit `generalize-build-assembly` D5; `git grep -i adr` over docs-kit `main` `internal/doctext` (no match); `grep -rnE "ADR-0?[0-9]+|enhancement [0-9]{4}" opm` (list in D1).
**Decision**: fix them in the library (D1), not by a docs-kit rule.
**Rationale**: the library's ADRs live in `library/adr/`, which no site version publishes, so a kept `ADR-009` names nothing a reader can open, and a stripped "enhancement" leaves broken prose. The pointer is maintainer context; an ordinary comment keeps it for maintainers.

### Section 1 before the gate

**Decision**: the doc-comment fixes are their own section with no gate.
**Rationale**: they help `go doc` readers now, they make docs-kit's trial build (`add-go-api-extractor` task 2.2) report only what is left, and they keep section 2 to adoption.

### A fourth section for one link

**Context**: orchestration gives the library no section 3, because it has no generator to delete.
**Decision**: a short section 4 after G2-switch for the visible link (D4) and the archive.
**Rationale**: the owner asked for the link; the site build forbids it earlier.

## Risks / Trade-offs

- docs-kit's trial build may find Markdown problems a `go doc` reader never sees (headings, code blocks). Section 2 fixes them before the first bundle; anything only docs-kit can fix stops the section.
- After G2-switch, an authored fix on `main` reaches the site only through a library release or a docs revision (`docs.yml` `mode: revision`). The diagnostics pages, still being written, are affected most; `AGENTS.md`'s new paragraph says so.
- The undocumented-symbol list is a snapshot; a symbol added before section 2 without a doc comment shows an empty entry. golangci-lint does not enforce exported doc comments here today.
