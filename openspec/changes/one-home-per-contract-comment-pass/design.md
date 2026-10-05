## Context

This change edits comments, prose and spec text only. Every fact below was checked at library `origin/main` ca7c56b, after rounds 1 and 2 of wave 2 merged (#183 front-door docs, #188 one-read directory acquire, #194 render demand and instance reads, #196 `opm/k8s`). Line numbers are at that commit. Round-3 changes are open in parallel (core beta.4 pin, typed fetch errors, three new `opm/k8s` packages), so the implementation re-reads every site by symbol, not by line number, and section 5 sweeps whatever they land.

## Goals / Non-Goals

**Goals:**

- Each runtime contract is stated once, in the godoc of the package that owns it. Rationale lives in ADRs and SHALL requirements in specs. Every other copy is a link, or a short orientation sentence that names its home.
- No comment, prose line or spec sentence touched here says something the code does not do.
- No committed file cites a supervisor decision or "the supervisor" as a source.

**Non-Goals:**

- Any change to behaviour, an exported identifier, a signature, an error string or a test. A finding that would need one is reported, not fixed.
- Spec requirements that restate a rule correctly for their own verb, for example the "no `os.Setenv`" clauses in artifact-types, platform-artifact, registry-module-loading and schema-dispatch. A spec is the SHALL home for its requirement, so these stay.
- The `DefaultSchemaModule` requirement's "At this change that is `…beta.1`" history. The core-beta4 change owns the default move.
- `CONSTITUTION.md` and `openspec/config.yaml`, and the frozen `enhancements/`.

## Research & Decisions

### D1. The homes

**Context**: The owner rule names three homes. This table assigns them for the contracts this pass touches.

**Decision**:

| Contract | Home | Copies that become links or go |
| --- | --- | --- |
| Source modes, and which artifact carries which | `opm/module.Source` type doc | `source.go:34-37` (rewritten in place, see D2) |
| How cue/load receives an overlay | `opm/internal/loader.LoadDir` doc for artifacts; `renderstage.Build` doc for the render build | `source.go:55-56`, `load.go:93-95` |
| Env override, never the process environment | `opm/internal/cueenv` package doc (mechanism); `opm/kernel` package doc § Surface (every verb) | `acquire.go:32`, `:91`, `:223`, `:258`; `load.go:21` |
| Render contract, gate and cause order | `opm/kernel` package doc § Rendering; `RenderError` type doc | `README.md:66-86`, `AGENTS.md:329-357`, `docs/getting-started.md:220-228` |
| Context lifetime (holder-bounded) | `opm/kernel` package doc § Every operation shares nothing | spec kernel-runtime `:65` (wording fixed, see D6) |
| Why the tiers, catalog and context rules exist | ADR-005, ADR-007, ADR-009, ADR-011 | ADR pointers in exported doc comments (D5) |

`OCILoader`'s "does not mutate process state (no os.Setenv)" (`opm/schema/loader.go:85-86`) stays. It is a public type's own contract, and the internal `cueenv` package cannot be linked from the published reference.

### D2. Which artifacts carry a Source

**Context**: `source.go:34-37` is false twice (the module-from-directory case and Catalog are missing).

**Explored**: Each artifact stamps a Source as follows:

- `AcquireModuleFromRegistry`: overlay (`loader.FetchModule`).
- `AcquireModuleFromDir`: overlay (`acquire.go:96-106`).
- `AcquireCatalogFromRegistry` and `AcquireCatalogFromDir`: overlay (`acquire.go:168-197`, `:205-238`).
- `AcquirePlatformFromDir`: on disk (`acquire.go:244-273`).
- `AcquireInstanceFromDir`: on disk with no values sources, overlay with them (`acquire.go:280-345`).
- `SynthesizeInstance`: overlay.

`platform.Source` and `catalog.Source` are both aliases of `module.Source`.

**Decision**: The paragraph names the four artifacts and the mode each is carried in, as listed above. It names the two aliases, and keeps the "nil for an artifact constructed from a bare value" sentence. The mode list above it (`:15-25`) is cut to the two modes and their meaning. Which verb stamps which mode is said once, in the new paragraph.

**Rationale**: The mode list and the carrier list stated the same facts twice and disagreed. A single list cannot disagree with itself.

### D3. Overlay sites

**Decision**: `source.go:55-56` becomes "the kernel wraps them with load.FromBytes wherever it hands an overlay to cue/load" (no count). The `load.go:93-95` comment says what that block does: the staged tree travels as bytes on `module.Source` and is wrapped here for this build, so no caller deals in `load.Source`. It drops "the one place".

The same pass corrects the sites that say only a bare-major loader makes synthesis load the schema. The code (`resolveCoreVersion` in `opm/kernel/synth.go`) loads for any loader that pins no exact release. The sites (`kernel.go` `New` and `SchemaCache` docs, the `SynthesizeInstance` doc, `ErrSchemaUnavailable`, `AGENTS.md`, `docs/getting-started.md`, `README.md`) say "a loader that pins no exact release (a bare-major `OCILoader`, or any other `Loader`)", as the `resolveCoreVersion` doc already does. `README.md` § Schema resolution also stops naming `opmodel.dev/core@v2` as what the kernel resolves.

**Rationale**: A count is a claim that rots when someone adds a site. A truthful count today is three, and nothing depends on it.

### D4. Prose copies become links

**Context**: The owner rule is "other copies become links". `AGENTS.md` § Render contract already follows it; the two sections after it do not.

**Decision**:

- **`AGENTS.md` § Schema cache lifetime contract.** It restates the `schema.Cache`, `kernel.New` and `Kernel.SchemaCache` contract in prose, and says OCILoader resolves `opmodel.dev/core@v2`, while the default is the exact pin `DefaultSchemaModule`. It becomes pointers to those three docs. It keeps the maintainer-only facts: tests use the `opm/internal/schematest` cache, and which consumers call `Get` (the cli publish gate, the operator's startup smoke check).
- **`AGENTS.md` § Kernel API surface and § Render pipeline.** These become one section of pointers: which godoc to edit for verbs, values, the context rule and the render steps (`opm/kernel` package doc, `Kernel.Render`, `RenderError`, the `opm/internal/renderstage` package doc). Some maintainer-only facts have no godoc home: `opm/kernel/kernel_test.go` pins the absence of the retired verbs, and the old verbs must not come back. These stay as one bullet each. `*kernel.Compiled` as terminal output already has its home in the `Compiled` doc, so the AGENTS line becomes a pointer to it.
- **`README.md` § Render.** It keeps two or three sentences of orientation: `Render` is the single verb, one CUE build per render over an instance and a platform, verdicts as data. It drops the pipeline block and links to `go doc ./opm/kernel`. The concurrency paragraph (`:84`) keeps one sentence and its link.
- **`docs/getting-started.md`.** The tutorial stays a tutorial. The gate-order enumeration in the `RenderError` code comment (`:220-228`) becomes "the typed causes, in the order the `RenderError` doc gives". The `WithRegistry` paragraph (`:39`) keeps its first sentence and links to the package doc for the list of operations.
- **Before deleting a copy.** The implementer checks each claim in it against its home. A claim the home lacks is added to that godoc in the same commit. Nothing is lost by moving it.

**Rationale**: The README and the tutorial are read by people who have not opened godoc yet, so they keep an orientation. AGENTS.md is read by maintainers who edit the godoc, so a pointer is what they need.

### D5. ADR pointers out of doc comments

**Decision**: In `opm/k8s/object/doc.go`, `opm/k8s/labels/doc.go` and `opm/helper/objectset/doc.go`, each "(ADR-NNN)" in the package doc is removed from the doc text. The doc says the fact in its own words: "the Kubernetes tier beside the kernel", "holder-bounded: it lives as long as the caller holds it". One non-doc comment after the package clause names the ADRs, in the shape `opm/kernel/doc.go:290-293` uses. The `Deprecated:` paragraph in `objectset` keeps its first sentence and its meaning; only the pointer moves. Packages that round 3 adds under `opm/k8s/` get the same treatment in section 5 if they carry pointers.

### D6. Spec deltas, and where REMOVED + ADDED is needed

**Context**: OpenSpec 1.12 refuses a MODIFIED requirement that drops a main-spec scenario. A scenario whose name turns false is therefore replaced through REMOVED plus ADDED under a new requirement name.

**Decision**:

- **schema-dispatch "Single OPM schema, externally resolved, with no apiVersion field".** REMOVED, and ADDED as "Single OPM schema, externally resolved and pinned by default". "Default resolves within the v2 major" is false by name: the default resolves exactly `DefaultSchemaModule`. The new requirement keeps "No in-tree schema source", "Evaluated module has no apiVersion field" and "Caller-pinned earlier major still loads" verbatim. It renames "Schema resolved via module identifier" to "Schema resolved via the default module identifier" (default `schema.DefaultSchemaModule`). It replaces the false scenario with "Default resolves the pinned release" and "Bare major resolves within the v2 major (opt-in)", which carries the prerelease-ordering clause.
- **schema-dispatch "A pinned schema release is known without a load".** MODIFIED. The last sentence reads "a kernel whose loader pins no exact release (a bare-major `OCILoader`, or any `Loader` that is not an `OCILoader`) SHALL resolve the release through its schema cache". All four scenarios are kept. No scenario is added for a custom loader: no test backs one today, and adding a test is outside a comment-only change.
- **schema-dispatch "Path inventory exposed as package-level vars".** MODIFIED, keeping the requirement name and its three scenario names, so the citations of the name in `openspec/specs/artifact-types/spec.md` and `opm/kernel/render_demand_test.go` stay valid. The scenario "Matcher and transformer paths are gone" drops `Transformers` from its list; the name still reads true, because it refers to the matcher's `Transformer*` paths. The body lists all fifteen exported paths with their readers, as checked by grep at ca7c56b:
  - `ContractsProvidedBy` has one Go reader by variable, the `Kernel.Render` core-floor presence check. `Platform.Contracts()` decodes the same field relative to `schema.Contracts`, by name.
  - `ContractsCollisions` and `ContractsCollidingEntries` are read in Go only by tests. They name fields `Platform.Contracts()` reads relative to `schema.Contracts` and the glue reads in CUE, and they document the collision report (the archived refuse-colliding-contracts change records them as documentation and test paths). The removal sentence is qualified to match: a path no kernel code reads, by variable or relative to an inventory path, is removed.
  - `Transformers` has two readers: `Catalog.Provides()` on both of its paths, as an evaluation guard, and the deprecated fold.
  - The readers include `Instance.Values`, `Instance.ModuleMetadata` and `Module.DebugValues` (library#194). A new scenario, "Instance and module reads go through the inventory", covers them. It asserts only that each accessor reads through its path: other code spells `#module` as a string (`opm/internal/loader/shape.go`, `opm/kernel/validate.go`), so a "no other string" clause would be false.
  - No "every exported path has a reader" scenario: the two collision paths would make it false.
- **kernel-runtime "Goroutine Safety Contract".** MODIFIED. "creates and releases its own `cue.Context`" becomes "creates its own `cue.Context` and drops its references to it on return; a value the caller holds keeps that context alive (holder-bounded)". "whose references the kernel releases" becomes "drops", and "no shared platform value" becomes "no shared materialized platform value". "SHALL NOT present any shared built value" becomes "SHALL NOT present a value built into a shared context, or a shared materialized platform". The package doc lets one acquired platform, whose `Package` is a built value, be shared by concurrent renders, so the retraction names the Materialize-era shape, not every shared value. All five scenarios are kept.
- **kernel-runtime, ADDED "Each runtime contract has one home".** This is the owner rule, with scenarios a verifier can check by grep. The session-local citation rule is scoped to files outside `openspec/changes/`, because archived changes cite their own planning sessions, and its scenario is the `git grep` of task 4.3. Which artifacts carry a Source is not a scenario here: it is an artifact fact, not part of the one-home rule, and task 1.4 checks it.
- **api-diff-check, two requirements MODIFIED.** In "Pull requests show breaking changes to the public API", the rule (the one fixed allow line for the `DefaultSchemaModule` value change) is already in the body; the `Source:` line drops its supervisor decision number. In "The check runs read-only with a committed tool checksum", the line "Source: supervisor note on library#181" becomes "Source: library#181 (the workflow hardening pass); workflow-hardening". All scenarios are kept.
- **cascade-wiring "The cascade references carry one .github SHA, checked on every PR".** MODIFIED, `Source:` line only. "owner decision 24 and the supervisor's extension" becomes "owner decision 24, extended to the two reusable workflows and the resolver checkout by the change join-release-cascade (`openspec/changes/archive/2026-10-04-join-release-cascade`)". "with the supervisor's addendum" becomes "as amended in contract version 3.1.2". All scenarios are kept.

### D7. Citation replacements in files that are not specs

**Decision**:

- `AGENTS.md:317`: "(a literal prefix for that value change only, …)" drops the supervisor decision number and becomes "(a literal prefix that matches that value change and nothing else)".
- `AGENTS.md:311`: "(owner decision 24, extended by the supervisor to the workflows and the resolver)" becomes "(owner decision 24 for the actions; the join-release-cascade change extended it to the two reusable workflows and the resolver checkout)".
- `.tasks/api-diff.sh:29`: the supervisor decision number is dropped. The comment keeps why the line exists: the cascade rewrites the constant on every core move. `task api:diff:test` must stay green. Only a comment line changes.

### D8. Recording the rule

**Decision**: `AGENTS.md` gains a short subsection, "Where a statement lives", next to "Enhancement references in comments". It says:

- A runtime contract goes in the godoc of the package that owns it.
- Rationale goes in an ADR.
- A SHALL requirement goes in an `openspec/specs` capability.
- `README.md`, `AGENTS.md` and `docs/` link to those homes. They may keep a short orientation sentence, but never a second statement of the contract.

The kernel-runtime ADDED requirement is the SHALL form of the same rule.

## Risks / Trade-offs

- [Round-3 changes rewrite comments this change also edits] → This change merges last. It rebases onto `main` before the PR and re-runs section 5. The hunks are comments, so conflicts are textual.
- [A claim dropped with a prose copy had no other home] → D4 requires checking each claim against its home before a copy goes, and adding it to the godoc when missing.
- [A doc-comment fix waits for the next release to reach the published reference] → Accepted. A docs revision can ship it earlier (`AGENTS.md` § Docs bundles).
- [A repin change written before this merges re-introduces the old cascade-wiring `Source:` line at its archive] → Named in the proposal. The repin author takes the new line when rebasing.
