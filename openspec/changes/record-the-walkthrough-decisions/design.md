## Context

See proposal.md for why. Facts read at library `origin/main` 88afdfb:

- `adr/` holds ADR-001 to ADR-012 and `TEMPLATE.md`. The next free number is 013. The three other round-8 library worktrees (`refuse-unset-required-config`, `type-author-resolution-errors`, `refuse-an-object-another-instance-adopted`) are at 88afdfb with no `adr/` change, and the only open library PR is the release PR (#210).
- ADRs are not published: `docs-kit.cue` bundles the `opm/` Go API and `docs/site/` only, so a new ADR needs no site registration.
- Walkthrough-id citations outside `openspec/changes/` (`CHANGELOG.md` has none):
  - `.github/workflows/consumer-build.yml:3` ("Owner decision j4 (kernel plan walkthrough, 2026-10-03)").
  - `.tasks/consumer-build.sh:10-11` ("owner decision j4, kernel plan walkthrough 2026-10-03").
  - `AGENTS.md:297` ("owner decision j4"), `:299` ("owner decision j4 of the beta.1 walkthrough"), `:370-371` ("owner decision c4 of the beta.1 walkthrough"), `:379` ("an owner decision by its walkthrough id").
  - `adr/012-matching-stays-in-the-library-glue.md:5` ("the owner's answer ... given in the beta.1 kernel plan walkthrough").
  - `opm/catalog/provides_parity_test.go:64-65` ("owner decision j3, beta.1 walkthrough"), a comment in a test file.
  - `openspec/specs/api-diff-check/spec.md:4` (Purpose), `:10`, `:44`.
  - `openspec/specs/consumer-build/spec.md:13`, `:32`, `:41`, `:55` ("owner walkthrough decision j4 (2026-10-03)").
  - `openspec/specs/kernel-runtime/spec.md:457` (the one-home requirement: "an owner decision by its walkthrough id", and "Source: owner decision c4 (beta.1 walkthrough)").
  - `openspec/specs/kubernetes-tier/spec.md:139` ("Source: ADR-008 (owner decision 2026-10-03)", the j5 answer) and `:273` ("owner decision f5 (beta.1 walkthrough)").
- Archived changes under `openspec/changes/archive/` (2026-10-02 to 2026-10-05) cite walkthrough ids in the explicit form ("owner decision j4", "walkthrough task f5", "the beta-1 kernel-plan walkthrough", "owner decision 2026-10-03") on 43 lines in 28 files, and in the bare form ("decision d2", "task e3") on 36 more lines. The one-home requirement's grep already exempts `openspec/changes/`.
- ADR-004, ADR-005, ADR-008 and ADR-011 name the archived change `record-walkthrough-decisions` in their Status lines. That is a resolvable source and is not a walkthrough id.

## Goals / Non-Goals

**Goals:**

- One durable, resolvable record of every decided walkthrough task, with where it landed.
- No committed library file cites a walkthrough id except through ADR-013, archived OpenSpec changes included. Only this change's own artifacts quote the old forms, as the record of what it replaced.
- The one-home rule says how to cite a walkthrough decision, and a scenario checks it.

**Non-Goals:**

- Sweeping cli or opm-operator (separate changes).
- Recording the release-cascade and security-pass owner selections (numbered 9 to 30), which are not walkthrough decisions.
- Any Go code beyond the comment lines of one test file.

## Decisions

### 1. One ADR whose Decision section is the table

ADR-013 follows `adr/TEMPLATE.md`. **Status:** Accepted (2026-10-05), recording decisions the owner gave on 2026-10-02 and 2026-10-03, and naming this change. **Context:** what the walkthrough was (the beta.1 kernel plan, 49 tasks, the owner's answer to each), that commits, comments and specs across library, cli and opm-operator cite the answers by id, and that no repository held the ids until now. **Decision:** this ADR is the record. A walkthrough decision is cited as `ADR-013, decision <id>` (several: `ADR-013, decisions e2 and e5`). Then the table. Where an answer's rationale already has its own ADR (ADR-005, ADR-008, ADR-011, ADR-012) or enhancement decision, the row names it and does not restate it. **Consequences:** bold-labelled paragraphs: a reader can resolve every id; the Landed column records the PRs merged when the ADR was accepted and is not updated by later PRs (a later PR cites the ADR, the ADR does not cite it); committed files, archived changes included, cite it; older commit messages and PR bodies keep their bare ids, which the table resolves.

A row holds the final answer only. Where the owner replaced an answer (e1's executor scope, g1, e2's pairing with g1), the row states the replacement and adds "(replaces an earlier answer)", without the earlier text.

Rows: `a1`, `a3`, `a4`, `a5`, `b1` to `b5`, `c1`, `c2`, `c4`, `c5`, `d1` to `d5`, `e1` to `e5`, `f1` to `f5`, `g1` to `g5`, `h1` to `h4`, `i1` to `i5`, `j1` to `j5`. Left out: `a2` and `c3` (done before the walkthrough, no answer recorded) and N1 to N8 (not walked). `f3` and `d4`, `h1` are decided (deferred, or to land inside other work) and keep a row that says so.

**Planned table** (implementation refreshes the Landed column against the PRs merged on the day; "frontends pending" means a decided frontend half has not merged):

| Id | Decision | Landed |
| --- | --- | --- |
| a1 | The cli applies in stages like the operator: sort by weight, CRDs and Namespaces first, wait for each CRD to be Established, then the rest. A dry run skips a custom resource whose CRD is new in the same apply, with a warning. Hand-rolled staging, no Flux in the cli. | cli#289, cli#313 |
| a3 | The operator retries acquisition failures as transient through one sentinel (`ErrAcquire`) and `errors.Is`, with no message-text matching; typed library errors (d1) refine it later. | opm-operator#190 |
| a4 | Duplicate objects are keyed on group, kind, namespace and name; the exported `Identity` stays and `Producer.APIVersion` is added. Non-breaking. | library#168 |
| a5 | The `opm/kernel` package doc gets its lists and code examples back, and a go/doc guard test keeps them. | library#166 |
| b1 | Instance values are compiled once. | library#197 |
| b2 | Directory acquisition reads the tree once through one source-taking helper, builds from the overlay it read, and serves files through `os.DirFS`. | library#188 |
| b3 | One canonical version helper (after i1); `golang.org/x/mod/semver` replaces Masterminds; one `major()`, one core-path constant, one language-floor constant. | library#184 |
| b4 | Decoder defects only: no doubled error wrap, and a dotted key stays one path segment. The per-package decoders stay. | library#184 |
| b5 | `IsLocal` and the catalog dependency read are consolidated with b3 and b4; `OverlayFromDir` rides b2. | library#184, library#188 |
| c1 | The stale library specs are synced with the code before the code changes that follow. | library#169 |
| c2 | The front-door docs (README, AGENTS, CONSTITUTION) match the kernel as built; the default core pin is exact, a bare major is opt-in. | library#183 |
| c4 | A runtime contract lives in godoc, rationale in ADRs, SHALL requirements in specs; every other copy links. | library#201 |
| c5 | 0021:D8 gains the library's GA criteria (no `cue.Value` in Render output, typed fetch and resolution errors, docs and specs that match the code, three consecutive betas without a break); the library docs link to it. | enhancements#88, library#167 |
| d1 | Fetch failures are typed: `ErrTransient` and `*FetchError`, read with `errors.Is` and `errors.As`, and `Classify` for raw CUE, modconfig and module errors, so the frontends stop matching message text. Refined 2026-10-05: author-defect resolution errors get a typed kind too, and the cli drops its last text match. | library#205, opm-operator#249, cli#325; refinement pending |
| d2 | Additive accessors (instance module metadata, merged values, module debug values), and the cli reads through them. | library#194, cli#311, cli#327 |
| d3 | Every kernel verb checks cancellation at entry and between stages; 0009:D9 is revised to allow it ahead of the 0009 work. | enhancements#88, library#205 |
| d4 | Coverage and skew checks cover every catalog a platform lists, delivered inside the 0026 work with h1. | not landed |
| d5 | The matching algorithm stays in the library glue; single derived rules move into core one at a time, each with a parity test. core#62 (move `#Match` into core) is closed as not now. Rationale: ADR-012. | library#167 |
| e1 | A Kubernetes tier `opm/k8s/` sits beside the kernel in the same Go module. Nothing in the library performs a planned action against a cluster; non-cluster executor backends may ship opt-in under `opm/helper/` (replaces an earlier answer). Each frontend submits in the library's order, which an engine's staging may refine but never contradict. Rationale: ADR-011, ADR-008. | enhancements#82, library#159 |
| e2 | The frontends' `pkg/core` moves into `opm/k8s/labels` and `opm/k8s/object`, and each frontend deletes its copy. It ships on its own (replaces the pairing with g1). | library#196, opm-operator#253, cli#328 |
| e3 | `opm/k8s/inventory` holds the entry, the stale-set function and both digests; the inventory digest hashes a canonical field encoding, so each frontend's stored digest changes once. | library#203, opm-operator#255, cli#329 |
| e4 | `opm/k8s/ownership` gives pure apply and delete verdicts, used on every apply, prune and delete path. The apply guard judges objects not already in the instance inventory; the override is a per-object adopt annotation; ModulePackage gets a persisted UUID. Refined 2026-10-05 (0012:D8): the apply guard also refuses an in-inventory object whose live adopt annotation or UUID label names another instance, and that instance drops it from its next inventory. | library#202; refinement and frontends pending |
| e5 | The weight table moves into `opm/k8s/object` and agrees with Flux's staged apply order (library#214); the operator keeps Flux's staged apply, and a parity test proves it never contradicts the library order. | library#196, library#214, cli#328; operator parity test pending |
| f1 | No ordering within a module is planned; ordering across modules belongs to a future Bundle definition, in the order its modules are defined (the Bundle idea is filed as enhancements#87). Kind-class staging stays. Rationale: ADR-008, ADR-011, 0012:D5. | library#167, enhancements#88 |
| f2 | The deletion protocol lives in `opm/k8s/lifecycle` (plan, serialisable state, one transition per call, hold verdict), used by both frontends' delete paths. No hook semantics. | library#209; frontends pending |
| f3 | Deferred until hook objects exist, with the 0009 hook work. | not landed (deferred) |
| f4 | 0009 is parked, with a note that its in-module step ordering conflicts with f1; 0025:OQ13 stays open. | enhancements#88 |
| f5 | The readiness evaluator moves into `opm/k8s/health` as a pure function over fetched objects; the cli switches, and the operator adds a separate Healthy condition. | library#199; frontends pending |
| g1 | Render output keeps `Compiled.Value` for now and moves to bytes before GA at the latest, earlier if render output is cached (replaces an earlier answer). The operator drops held render results after conversion, shares one render limit and sets GOMEMLIMIT to about 80% of the pod limit. Rationale: ADR-005. | library#167, opm-operator#192 |
| g2 | `SynthesizeInstance` attributes values failures the way directory acquisition does; the operator deletes its own pre-check after that. Refined 2026-10-05: both verbs refuse an unset required `#config` value at its position (library#211). | library#197; refinement and operator half pending |
| g3 | The operator skips a render whose inputs are unchanged, and re-renders on a throttle to catch drift. | opm-operator#246 |
| g4 | Failed pairs leave the render glue and are filled in Go, and the matching steps key on candidates, measured first. | library#186 |
| g5 | Part A: the render module is staged in memory and each kernel shares one registry client. Part B (platform overlay mode, dropping `Platform.Package`) follows h4 and the 0026 reshape. | library#212; Part B pending |
| h1 | A kernel verb inside the 0026 work; catalog dependencies are recognised by what the module is, not by a path prefix. | not landed |
| h2 | Core derives a per-catalog provider set; `Provides()` reads it and keeps a deprecated fallback for older catalogs until before GA. | core#120, library#195, catalog_opm#156, catalog_opm#157, opm-operator#250 |
| h3 | A cold concurrent-render test. | library#168 |
| h4 | The core floor and the contracts inventory are decoded once at acquire into plain fields on `Platform`. | library#208 |
| i1 | The kernel accepts `1.0.0` and `v1.0.0` alike through one helper; an operator e2e test drives a real claim through acceptance and platform build. | library#170, opm-operator#238 |
| i2 | cli prune, preview and delete never delete CRDs or Namespaces and list them as left behind; delete re-reads each live object and skips it unless its ownership still matches. | cli#285 |
| i3 | The kernel exports the contract demand of a render as typed data and fails closed; the operator stops walking it itself. | library#194, opm-operator#252 |
| i4 | The operator builds a fresh module-file source per Platform reconcile; the library checks a schema build's error before caching it, and the schema cache stays never-retry. | opm-operator#189, library#168 |
| i5 | The cli refuses a namespace flag or `OPM_NAMESPACE` that disagrees with the instance file (ADR-001). | cli#284 |
| j1 | The operator's Platform watch fires only on fields a render consumes, and enqueues only the instances that resolve against that Platform. | opm-operator#191, opm-operator#237 |
| j2 | Core moves its pins into a `src/pins/` subpackage; catalog_opm runs its fixture vet in CI and moves its fixtures behind a build tag. | core#103, catalog_opm#131 |
| j3 | Attachment map keys are bound to `metadata.fqn` as a hard vet error. | core#125 |
| j4 | An API-diff check warns until GA and blocks after; consumer builds of cli and opm-operator run on library PRs as a non-required job; `cuelang.org/go` moves only through a library release. | library#189, library#185, cli#312, opm-operator#245, workspace#28 |
| j5 | An instance's deletion plan is built from the persisted inventory plus the live objects, with no render and no stored plan; the operator records the last applied version in plain text. Rationale: ADR-008. | library#167, opm-operator#242 |

### 2. Archived changes are swept too

**Context:** the task for this change says every committed file that cites a walkthrough id cites the ADR instead. Archived OpenSpec changes cite walkthrough ids on 43 lines in the explicit form and 36 lines in the bare form ("decision d2", "task e3"). `CHANGELOG.md` cites none.
**Decision:** sweep every committed file, archived changes included, in a section of its own. Each citation becomes `ADR-013, decision <id>` (or `ADR-013, decisions <id> and <id>`), and a sentence that names the walkthrough without an id names ADR-013 instead. Only the citation words change; no archived requirement, task state or rationale changes. The gate (decision 3) keeps excluding `openspec/changes/`, because this change's own artifacts quote the old forms and archive with it; the archive sweep is checked by the same commands run over `openspec/changes/archive/` before this change is archived.
**Rationale:** the task names every committed file, and the owner's answer asks that the citations resolve ("cite that instead. Durable and resolvable"). The archive is a frozen record of what was planned, but a citation word that points nowhere is not part of what was planned. Rewording it keeps the record and makes it resolvable. The edit is a few dozen lines in one commit, so a reviewer can read it.

### 3. Citation form and the check

The form is `ADR-013, decision j4` (prose) and `Source: ADR-013, decision j4.` (specs), the form the task for this change gives. In a code comment the pointer goes in a non-doc comment or a test file, per the existing rule that published doc comments carry no `ADR-NNN` (the one site, `provides_parity_test.go`, is a test file).

The check, used by the tasks and by a new scenario, is two commands:

```
git grep -nE "[Oo]wner('s)? (walkthrough )?decisions? [a-j][1-5]\b|walkthrough (decisions?|tasks?) [a-j][1-5]\b|(beta\.1|kernel[ -]plan) walkthrough|owner decision 2026-10-0[23]" -- . ':!openspec/changes' ':!adr/013-*'
git grep -nE "\b(decisions?|tasks?) [a-j][1-5]\b" -- . ':!openspec/changes' ':!adr/013-*' | grep -v 'ADR-013, decisions\? '
```

The first matches every prefixed citation form found at 88afdfb, plural forms, and the date form "owner decision 2026-10-03" (how `kubernetes-tier` cites j5 today). The second catches a bare "decision g2" or "task e4" that does not go through ADR-013. Neither pattern matches its own text, so the spec that quotes them does not trip them once archived (the lesson of library#198). "by its walkthrough id" (`AGENTS.md:379`, the kernel-runtime requirement) is fixed by hand and checked by a separate grep in tasks, because a pattern for it would match itself. The check is not wired into `task check`; it joins the ungated one-home checks that library#198 tracks, and that issue should also cover it.

### 4. The api-diff-check Purpose is edited in place

A delta can add, modify, remove or rename requirements, but it cannot change a spec's Purpose. The Purpose of `api-diff-check` cites "owner decision j4 of the beta.1 walkthrough". Section 2 edits that one parenthesis in `openspec/specs/api-diff-check/spec.md` directly, to "(ADR-013, decision j4)". It is the only direct main-spec edit. Every requirement change goes through the deltas.

### 5. Spec deltas

- `kernel-runtime`: MODIFIED "Each runtime contract has one home". The source list says "a decision ADR-013 records, cited as `ADR-013, decision <id>`" in place of "an owner decision by its walkthrough id", and the Source line becomes `ADR-013, decision c4`. All four scenarios are kept unchanged, and two are added: "A walkthrough decision resolves through ADR-013" (every id cited as `ADR-013, decision <id>` has exactly one row with a non-empty Landed cell) and "No walkthrough id without ADR-013" (the decision 3 commands).
- `api-diff-check`: MODIFIED "Pull requests show breaking changes to the public API" (six scenarios kept) and "The base tag decides between warning and failing" (four kept): only the Source lines change.
- `consumer-build`: MODIFIED, all four requirements (scenarios kept): only the Source lines change.
- `kubernetes-tier`: MODIFIED "Deletion plans come from the inventory and the live objects" (Source becomes `ADR-008; ADR-013, decision j5; the deletion protocol is 0012:D4`) and "Readiness evaluation is a pure function over fetched objects" (Source becomes `0012:D3; ADR-013, decision f5`), all scenarios kept.

The delta bodies are copied from the main specs at 88afdfb and differ only in those words. Section 3 re-copies any requirement that a round-8 change modified first.

## Risks / Trade-offs

- [The round-8 library changes merge first and may add walkthrough-id citations or modify the same requirements] → section 3 merges `origin/main`, re-runs the decision 3 grep, and re-copies any delta whose main-spec requirement changed, keeping that change's text.
- [The Landed column goes stale as frontend halves merge] → the ADR says the column is a snapshot at acceptance; later PRs cite the ADR, and the ADR is not edited per PR. A row that says "pending" stays true as a record of the state at acceptance.
- [An ADR that records cli and opm-operator decisions lives in the library] → the owner chose one library ADR. The library is the kernel every frontend depends on, and most rows have a library half.
- [A reader of a frontend repo cannot open a library path] → the frontend sweeps cite it as `library` ADR-013 with its path in the library repository, which is public.
