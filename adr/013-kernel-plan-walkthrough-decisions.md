# ADR-013: The kernel-plan walkthrough decisions

## Status

Accepted (2026-10-05). Records the decisions the owner gave on 2026-10-02 and 2026-10-03, going through the beta.1 kernel plan task by task. Recorded by `record-the-walkthrough-decisions`, which also makes every committed library file cite this ADR instead of a bare walkthrough id; it is not the 2026-10-03 change `record-walkthrough-decisions` (library#167), which amended ADR-004, ADR-005, ADR-008 and ADR-011 and added ADR-012 with some of the same answers. The cli and opm-operator are to make the same change in their own repositories.

## Context

Before the first library beta, the kernel plan for beta.1 listed 49 tasks across the library, the cli and opm-operator: fixes, consolidations, a Kubernetes tier, the GA criteria and the CI checks around the library. On 2026-10-02 and 2026-10-03 the owner walked through it and gave an answer to each task. Each task had a short id, a letter for its group and a number (`a1` to `j5`), and the changes that followed cite the answers by that id: commit messages, PR bodies, code comments, `AGENTS.md`, OpenSpec specs and archived changes in all three repositories.

The answers themselves were written down only in a planning log outside every repository. A citation such as "owner decision j4" therefore named a source no reader of the repository could open, although the `kernel-runtime` requirement "Each runtime contract has one home" listed a walkthrough id as one. Some answers already have their own home for the rationale (ADR-005, ADR-008, ADR-011, ADR-012, and the enhancement decisions 0012:D3 to 0012:D8 and 0021:D8), and some refine an earlier answer or were refined later. Nothing listed them all in one place, with where each one landed.

## Decision

This ADR is the record of the walkthrough decisions. A committed file cites one as `ADR-013, decision <id>`, and several as `ADR-013, decisions <id> and <id>`. A bare walkthrough id, or one that names the walkthrough in place of this ADR, is not a citation a committed file may use.

The table below holds one row per decided task. A row states the final answer: where the owner replaced an earlier answer, the row gives the replacement and says "(replaces an earlier answer)"; where a later owner answer refined it, or a later change amended it, the row adds that refinement or amendment with its date and its own source. Where the rationale has its own ADR or enhancement decision, the row names it and does not restate it. The Landed column lists the merged pull requests that carry the decision, as `repo#N` in the `open-platform-model` organisation, and says "pending" for a decided half that had not merged on the date of acceptance, or "deferred" for a decision whose work waits on another piece of work by design.

Two tasks have no row: `a2` and `c3` were done before the walkthrough, and the owner gave no answer for them. The research items added to the plan afterwards were not walked and are not decisions.

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
| d1 | Fetch failures are typed: `ErrTransient` and `*FetchError`, read with `errors.Is` and `errors.As`, and `Classify` for raw CUE, modconfig and module errors, so the frontends stop matching message text. Refined 2026-10-05 (0021:D8:R12): author-defect resolution errors get a typed kind too, and the cli drops its last text match. | library#205, opm-operator#249, cli#325; refinement pending |
| d2 | Additive accessors (instance module metadata, merged values, module debug values), and the cli reads through them. | library#194, cli#311, cli#327 |
| d3 | Every kernel verb checks cancellation at entry and between stages; 0009:D9 is revised to allow it ahead of the 0009 work. | enhancements#88, library#205 |
| d4 | Coverage and skew checks cover every catalog a platform lists, delivered inside the 0026 work with h1. | pending |
| d5 | The matching algorithm stays in the library glue; single derived rules move into core one at a time, each with a parity test. core#62 (move `#Match` into core) is closed as not now. Rationale: ADR-012. | library#167 |
| e1 | A Kubernetes tier `opm/k8s/` sits beside the kernel in the same Go module. Nothing in the library performs a planned action against a cluster; non-cluster executor backends may ship opt-in under `opm/helper/` (replaces an earlier answer). Each frontend submits in the library's order, which an engine's staging may refine but never contradict. Rationale: ADR-011, ADR-008. | enhancements#82, library#159 |
| e2 | The frontends' `pkg/core` moves into `opm/k8s/labels` and `opm/k8s/object`, and each frontend deletes its copy. It ships on its own (replaces the pairing with g1). | library#196, opm-operator#253, cli#328 |
| e3 | `opm/k8s/inventory` holds the entry, the stale-set function and both digests; the inventory digest hashes a canonical field encoding, so each frontend's stored digest changes once. | library#203, opm-operator#255, cli#329 |
| e4 | `opm/k8s/ownership` gives pure apply and delete verdicts, used on every apply, prune and delete path. The apply guard judges objects not already in the instance inventory; the override is a per-object adopt annotation; ModulePackage gets a persisted UUID. Refined 2026-10-05 (0012:D8): the apply guard also refuses an in-inventory object whose live adopt annotation names another instance, and the refusing instance drops it from its own next inventory. The annotation alone decides: a foreign UUID label without it does not refuse, so a module-path move, which changes the instance UUID, keeps working. | library#202; refinement and frontends pending |
| e5 | The weight table moves from the cli into `opm/k8s/object` with e2, and the cli deletes its copy. The operator pre-sorts by the library weights and hands the set to Flux, whose staging may refine the order but never contradicts it. Amended 2026-10-05 by `order-kinds-like-flux` (library#214; ADR-011; 0012:D5:R1): the table agrees with Flux's staged apply order, the operator keeps Flux's staged apply, and a parity test proves Flux's order never contradicts the library order. | library#196, library#214, cli#328; operator parity test pending |
| f1 | No ordering within a module is planned; ordering across modules belongs to a future Bundle definition, in the order its modules are defined (the Bundle idea is filed as enhancements#87). Kind-class staging stays. Rationale: ADR-008, ADR-011, 0012:D5. | library#167, enhancements#88 |
| f2 | The deletion protocol lives in `opm/k8s/lifecycle` (plan, serialisable state, one transition per call, hold verdict), used by both frontends' delete paths. No hook semantics. | library#209; frontends pending |
| f3 | Deferred until hook objects exist, with the 0009 hook work. | deferred |
| f4 | 0009 is parked, with a note that its in-module step ordering conflicts with f1; 0025:OQ13 stays open. | enhancements#88 |
| f5 | The readiness evaluator moves into `opm/k8s/health` as a pure function over fetched objects; the cli switches, and the operator adds a separate Healthy condition. | library#199; frontends pending |
| g1 | Render output keeps `Compiled.Value` for now and moves to bytes before GA at the latest, earlier if render output is cached (replaces an earlier answer). Rationale: ADR-005. The operator memory package, answered alongside g1: the operator drops held render results after conversion, shares one render limit and sets GOMEMLIMIT to about 80% of the pod limit, and operator memory is measured before and after j2, the dropped render results, the shared limit and g4. | library#167, opm-operator#192, library#186 |
| g2 | `SynthesizeInstance` attributes values failures the way directory acquisition does; the operator deletes its own pre-check after that. Refined 2026-10-05: both verbs refuse an unset required `#config` value at its position (library#211). | library#197; refinement and operator half pending |
| g3 | The operator skips a render whose inputs are unchanged, and re-renders on a throttle to catch drift. | opm-operator#246 |
| g4 | Failed pairs leave the render glue and are filled in Go, and the matching steps key on candidates, measured first. | library#186 |
| g5 | Part A: the render module is staged in memory and each kernel shares one registry client. Part B (platform overlay mode, dropping `Platform.Package`) follows h4 and the 0026 reshape. | library#212; Part B pending |
| h1 | A kernel verb inside the 0026 work; catalog dependencies are recognised by what the module is, not by a path prefix. | pending |
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

## Consequences

**Positive:** Every walkthrough id cited in a committed file resolves to one row here, with the decision in plain words and the pull requests that carry it. Once the cli and opm-operator have made the same change, a reader there finds the same record, cited as `library` ADR-013 (`adr/013-kernel-plan-walkthrough-decisions.md` in the library repository). Commit messages and PR bodies merged before this ADR keep their bare ids, and the table resolves them as well.

**Negative:** One library ADR records decisions whose halves live in the cli, opm-operator, core, catalog_opm and the enhancements. A frontend reader has to open the library repository to resolve them. The library is the kernel every frontend depends on, and most rows have a library half, so the record sits there.

**Trade-off:** The Landed column is a snapshot on the date of acceptance. A later pull request that finishes a pending half cites this ADR; this ADR is not edited to list it. A row that says "pending" stays true as a record of the state when the decisions were written down, and the pull request history shows what landed after.
