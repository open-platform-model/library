Delivery: one PR per section (proposal.md). Each section has its own gate, named as in docs-kit `docs/orchestration.md`; do not start a section before its gate holds.

## 1. Doc comments that read as reference pages

Gate: none.

- [x] 1.1 `opm/kernel/doc.go`: rewrite the "Surface" list in Go doc-comment list syntax (design.md D1); move the `ADR-005`, `ADR-002`, `ADR-007` and `ADR-009` pointers out of the package doc into ordinary comments. Verify: `go doc ./opm/kernel` shows one list item per operation and no `ADR-`.
- [x] 1.2 `opm/kernel/acquire.go`, `opm/kernel/render.go`: the `ADR-009` pointer at `acquire.go:136`; doc comments for `RenderError.Error`, `RenderError.Unwrap` and the `SkewWarn`/`SkewRefuse` group.
- [x] 1.3 `opm/errors/`: doc comments for the eight `Error` methods and `TransformError.Unwrap` (design.md D1 table); reword the "enhancement 0010:D32 as corrected by" sentence in `oversubscribed.go:12`.
- [x] 1.4 `opm/catalog/catalog.go`, `opm/platform/platform.go`, `opm/platform/doc.go`, `opm/helper/doc.go`, `opm/helper/platformmodule/generate.go`: the remaining ADR and "enhancement NNNN" pointers; the platformmodule constant group's doc.
- [x] 1.5 Verify: rerun the `go/doc` walk of design.md D1 over `opm/` (no exported symbol outside `internal/` with an empty doc) and `grep -rnE "ADR-0?[0-9]+|enhancement [0-9]{4}" opm --include=*.go` finds only non-doc comments.
- [x] 1.6 `task check` green, then commit `docs: make the exported doc comments read as reference pages`.

## 2. Adopt docs-kit

Gate G2-library: docs-kit's `add-go-api-extractor`, `add-authored-docs` and `generalize-build-assembly` are released. Use the first docs-kit release that carries all three as `vX.Y.Z` below.

- [x] 2.1 `.opm-docs-version`: `vX.Y.Z`. `.tasks/opm-docs.sh`: copy catalog_opm's byte for byte. `.gitignore`: `/out/` and `/.bin/`.
- [x] 2.2 `Taskfile.yml`: `tools:opm-docs`, `docs:bundle` (`--project library --out out`), `docs:pins:check`, `docs:bundle:check` as catalog_opm has them; `check` and `check:fast` run `docs:bundle:check` last.
- [x] 2.3 `docs-kit.cue` exactly as design.md D2. Verify: `task docs:bundle` writes `out/library/` with nine package pages and `_index.md` under `content/reference/go-api/`, and the eight authored pages.
- [x] 2.4 Read every generated page; fix in the doc comments anything that reads wrong as Markdown, and anything docs-kit's trial build recorded (`add-go-api-extractor` design.md, task 2.2). Anything only docs-kit can fix stops the section and goes to docs-kit. Verify: `task docs:bundle:check` passes.
- [x] 2.5 Backfill dry run: in a scratch worktree of `v1.0.0-beta.1`, run `.bin/opm-docs build --project library --release v1.0.0-beta.1 --source <that worktree> --out <scratch>` from this tree (`main`'s config). Verify: it builds and lints; record the page count and any warning in design.md D5.
- [x] 2.6 `.github/workflows/docs.yml`: catalog_opm's with `project: library`, tags `vX.Y.Z`, `publish.yml@vX.Y.Z`. `.github/workflows/release.yml`: `publish-docs` after `release-please` (design.md D3). Verify: `actionlint` clean; `task docs:pins:check` passes.
- [x] 2.7 `docs/site/embedding/embed-the-kernel.md`: in the Next-steps brief, replace the "Go API documentation ... (Verify: where it is published ...)" clause with `/docs/reference/go-api/kernel/` (design.md D4; the brief stays a comment).
- [x] 2.8 `AGENTS.md`: a "Docs bundles" paragraph under "Build And Dev Commands" (PR check, edge on `main`, a bundle per release; preview with `task docs:bundle` or `opm-docs serve`; recover with `gh workflow run docs.yml --ref main -f mode=release -f tag=vX.Y.Z`; fix a released page with `mode=revision`, dispatched by hand (library#164); after the site reads the bundle, an authored fix reaches it only by a release or a revision; a new exported symbol needs a doc comment because it gets a reference entry), and the tasks in "Core commands".
- [x] 2.9 `openspec validate publish-go-api-bundle --strict` passes; `task check` green, then commit `ci(docs): publish the library docs bundle with docs-kit`.

## 3. Publish the bundle the cli pins

Gate: section 2 is merged. Part of gate G2-pins (core `v2.0.0-beta.1`, library `v1.0.0-beta.1` and opm-operator `v1.0.0-beta.4` backfilled, then cli `v1.0.0-beta.6`). This section's deliverable is a publishing operation (the owner's), so its steps are the implementation.

- [ ] 3.1 Owner: dispatch `gh workflow run docs.yml --ref main -f mode=release -f tag=v1.0.0-beta.1` (owner decision 2026-10-03; design.md D5), then check `ghcr.io/open-platform-model/docs/library` is public and linked to `open-platform-model/library` (change it in the package settings if not).
- [ ] 3.2 Optional, owner: apply section 1's squash commit to the published bundle as a docs revision, `gh workflow run docs.yml --ref main -f mode=revision -f tag=v1.0.0-beta.1 -f fix=<40-hex sha>` (comments only, so C3 accepts it; manual, library#164). A later library release (release PR library#155) publishes through `publish-docs` and is not needed here; if the release cascade moves the cli's library pin before cli `v1.0.0-beta.6`, that version needs its bundle first.
- [ ] 3.3 Verify the full, release, minor and major tags of `1.0.0-beta.1` (and its revision, if made) with `cosign verify` and docs-kit C9's identity flags, or an anonymous `opm-docs pull` with a scratch `bundles.cue` naming `library` under `docs`.
- [ ] 3.4 Record in design.md D5 the run URLs, versions, digests and the verification; tell the cli that the library's part of G2-pins holds.
- [ ] 3.5 `openspec validate publish-go-api-bundle --strict` passes; `task check` green, then commit `docs(openspec): record the first library docs bundle`.

## 4. Link the Go API reference

Gate G2-switch: opmodel.dev's v1.0 reads core, cli, library and opm-operator from bundles (docs-kit `docs/orchestration.md`).

- [ ] 4.1 `docs/site/embedding/embed-the-kernel.md`: under "Next steps", a visible link `[Go API: opm/kernel](/docs/reference/go-api/kernel/)` after the brief. Verify: `task docs:bundle:check` passes (bundle-mode lint checks the link against `reference/go-api/`).
- [ ] 4.2 `openspec archive publish-go-api-bundle --yes`. Verify: `openspec validate --specs --strict` passes for `docs-bundle`.
- [ ] 4.3 `task check` green, then commit `docs(site): link the Go API reference from Embed the kernel`.
