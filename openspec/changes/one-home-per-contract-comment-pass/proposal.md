## Why

The owner decided this in the kernel-plan walkthrough (task c4): "Runtime contracts in godoc, rationale in ADRs, SHALL requirements in specs; other copies become links; fix the two false claims (source.go:35, :56). One comment-only pass after c1, c2 and b2." c1, c2 (#183) and b2 (#188) have merged, and so have the round-2 library changes that rewrote the same comments. At `origin/main` ca7c56b the copies have drifted, and some say false things:

- **Which artifacts carry a Source.** `opm/module/source.go:34-37` says Source is carried by "Module (registry path only), Instance (synthesis and directory acquire) and Platform (directory acquire)". `AcquireModuleFromDir` stamps an overlay Source (`opm/kernel/acquire.go:106`), and a `Catalog` carries one from both of its acquire verbs (`opm/catalog/catalog.go:54-67`, `acquire.go:197`, `:238`). The mode list in the same comment (`source.go:15-25`) already names the module-from-directory case, so the file contradicts itself.
- **"The one place" the kernel hands cue/load an overlay.** `opm/module/source.go:55-56` and the comment at `opm/internal/loader/load.go:93-95` both claim one site. There are three: `loader.LoadDir` (`load.go:100`), the render build (`opm/internal/renderstage/stage.go:208`) and file-backed values compilation (`opm/kernel/source_loader.go:102`).
- **"never os.Setenv" restated.** The rule's home is `opm/internal/cueenv` (`cueenv.go:5-11`) and, for every kernel verb, the `opm/kernel` package doc ("never written back to the environment", `doc.go:44-47`). The same words are repeated at `opm/kernel/acquire.go:32`, `:91`, `:223`, `:258` and `opm/internal/loader/load.go:21`.
- **The render contract restated in prose.** `AGENTS.md` § Render contract (`:244-257`) says "Edit the godoc; do not restate it here", and then § Kernel API surface (`:329-335`) and § Render pipeline (`:337-357`) restate it. `README.md` § Render (`:66-86`) restates it again, gate order and all, and `docs/getting-started.md:220-228` lists the gate order a fourth time.
- **ADR pointers in doc comments.** `AGENTS.md:290` keeps `ADR-NNN` out of doc comments, because the docs bundle publishes them. `opm/k8s/object/doc.go:3`, `:13`, `opm/k8s/labels/doc.go:11` and the `Deprecated:` paragraph of `opm/helper/objectset/doc.go:31` carry them.
- **Spec text that the code no longer matches.**
  - `openspec/specs/schema-dispatch/spec.md:138` says a kernel "whose loader names a bare major" resolves through the cache. The code (`opm/kernel/synth.go:207-234`) does that for any loader that pins no exact release: a bare-major `OCILoader` or any `Loader` that is not an `OCILoader`.
  - Two scenarios of the first schema-dispatch requirement still give the default as the bare major `"opmodel.dev/core@v2"` (`:18-22`, `:29-33`). The default pins an exact release (`DefaultSchemaModule`). lib-c2 left both to this change.
  - schema-dispatch "Path inventory exposed as package-level vars" (`:265-284`) says the inventory is exactly six paths, and one of its scenarios says `Transformers` does not exist. `opm/schema/paths.go:20-84` exports fifteen. Its readers now include the platform contract inventory, the render core floor, the catalog provider set, and the instance and module accessors that library#194 added: `Instance.Values`, `Instance.ModuleMetadata` and `Module.DebugValues`.
  - `openspec/specs/kernel-runtime/spec.md:65` says each operation "creates and releases its own `cue.Context`". What the kernel does, under ADR-007's holder-bounded rule, is create the context and drop its own references to it on return. A value the caller holds keeps the context alive.
- **Only-bare-major claims.** `opm/kernel/kernel.go` (`New`, `SchemaCache`), the `SynthesizeInstance` doc, `ErrSchemaUnavailable`, `AGENTS.md` § Schema cache lifetime contract, `docs/getting-started.md:41` and `README.md:114` say only a bare-major loader makes synthesis load the schema; any loader that pins no exact release does. `AGENTS.md:210-211` and `README.md:108` also name `opmodel.dev/core@v2` as what the kernel resolves, while the default is the exact pin.
- **The Render godoc is ragged.** `opm/kernel/render.go:306-316` has lines from 40 to 85 columns, left over from earlier edits.
- **Supervisor citations in committed files.** `AGENTS.md:317`, `openspec/specs/api-diff-check/spec.md:10` and `.tasks/api-diff.sh:29` cite a supervisor decision by number. `AGENTS.md:311`, `api-diff-check/spec.md:68` and `openspec/specs/cascade-wiring/spec.md:115` cite "the supervisor" as a source. None of these resolves from the repo. Each should state the rule, with a source a reader can open.

## What Changes

- **Fix the false Go comments.** `module.Source` names each artifact that carries a Source and the mode it is carried in. The overlay sentence in `source.go` and the `LoadDir` comment stop claiming a single cue/load overlay site.
- **One home per runtime contract.** The env-override rule lives in the `cueenv` package doc (mechanism) and the `opm/kernel` package doc (every verb). The acquire verbs and `loader.Options` link to those homes instead of repeating "never os.Setenv". `OCILoader`'s own "does not mutate process state" stays, because it is that type's contract.
- **Prose copies become links.** In `AGENTS.md`, § Schema cache lifetime contract, § Kernel API surface and § Render pipeline become short pointers to the godoc that owns each fact. They keep only the maintainer-only facts that no godoc carries. `README.md` § Render becomes an orientation paragraph that links to the `opm/kernel` package doc. `docs/getting-started.md` points at the `RenderError` doc for the gate order instead of listing it. Before a copy goes, any claim in it that no godoc or spec carries is moved into the godoc that owns it.
- **ADR pointers leave doc comments.** In `opm/k8s/object`, `opm/k8s/labels` and `opm/helper/objectset`, the ADR pointers move to a non-doc comment after the package clause, as `opm/kernel/doc.go:290-293` already does.
- **Render godoc reflow.** `render.go:306-316` is reflowed to about 75 columns. The words stay the same.
- **AGENTS.md states the rule.** A short paragraph says where each kind of statement lives (runtime contract in godoc, rationale in ADRs, SHALL in specs), and that other copies link.
- **Supervisor citations go.** Each one is replaced with the rule plus a source a reader can open: the owner decision by its walkthrough id, the PR number, or the archived change.
- **Spec deltas.** These are spec text only:
  - schema-dispatch: the first requirement is re-added under a new name, with corrected default scenarios; the `:138` wording; the path inventory, MODIFIED under its own name with every path's readers named.
  - kernel-runtime: the `:65` wording, plus a new requirement stating the one-home rule.
  - api-diff-check and cascade-wiring: the `Source:` lines.

No behaviour changes. No exported identifier, signature, error text or test changes.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `schema-dispatch`: corrects the default-resolution scenarios (the default is an exact pin, the bare major opt-in), corrects which loaders resolve the core release through the cache, and names the full path inventory and its readers, including `Instance.Values`, `Instance.ModuleMetadata` and `Module.DebugValues`.
- `kernel-runtime`: states the holder-bounded context lifetime accurately in the goroutine-safety contract, and adds the rule that each runtime contract has one home and that other copies link to it.
- `api-diff-check`: the `Source:` lines cite the owner decision and the PR, not a supervisor decision.
- `cascade-wiring`: the pin requirement's `Source:` line cites the change and contract version that extended owner decision 24, not "the supervisor".

## Impact

**SemVer: no release.** All commits are `docs`, which release-please hides (`AGENTS.md` § Commit style). The doc-comment fixes reach the published Library reference at the next library release. They can reach it sooner through a docs revision (`AGENTS.md` § Docs bundles).

**Affected files:**

- Go comments: `opm/module/source.go`, `opm/internal/loader/load.go`, `opm/kernel/acquire.go`, `opm/kernel/render.go`, `opm/k8s/object/doc.go`, `opm/k8s/labels/doc.go`, `opm/helper/objectset/doc.go`, and any package doc that gains a claim moved out of the prose.
- Other files: `README.md`, `AGENTS.md`, `docs/getting-started.md` and `.tasks/api-diff.sh` (a comment line).
- Specs: four main specs, changed through archive.

**Downstream consumers:**

| Consumer | What it has to do |
| --- | --- |
| cli, opm-operator | Nothing. No Go identifier changes, and `api:diff` and the consumer build see no change. |
| opmodel.dev | Nothing. The Library reference picks up the comments with the next bundle it pulls. |

**Order and conflicts:** This change merges last among the round-3 library changes (lib-d1d3, lib-e3, lib-e4, lib-f5, lib-core-beta4). Those changes edit code next to the comments touched here and add new `opm/k8s` packages. Before the PR, this branch merges `origin/main` and re-runs the section 5 sweep over whatever they landed. Line numbers in these artifacts are at `origin/main` ca7c56b. A cascade repin change planned before this merges carries its own copy of the cascade-wiring pin requirement, and so must take the new `Source:` line when it absorbs `main`.
