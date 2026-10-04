## Why

The three front-door files a new reader opens first, `README.md`, `AGENTS.md` and `CONSTITUTION.md`, still say things about the kernel that stopped being true. At `origin/main` 58f8151:

- **Three kinds, then four.** `README.md:19` says the kernel "accepts only `Module`, `ModuleInstance`, and `Platform`", and `README.md:7` lists the same three. Four lines later `README.md:23` says "exactly four", with `Catalog`. ADR-009 admitted `Catalog`.
- **A schema label core does not use.** `README.md:27` labels `#Module` "(v1alpha2)", and `AGENTS.md:163` heads its table "Schema (`v1alpha2`)". The artifact roots carry no `apiVersion` (main spec `schema-dispatch`, "Single OPM schema, externally resolved, with no apiVersion field"; core `src/module.cue`, `src/module_instance.cue`, `src/platform.cue`, `src/catalog.cue` declare only `kind`). They are versioned by the core CUE module, `opmodel.dev/core@v2`. The `v1alpha1` and `v1beta1` labels in core belong to catalog member contracts, not to the artifact schemas.
- **"Floating major" as the mechanism.** `README.md:97`, `AGENTS.md:351` and `AGENTS.md:361` say additive schema changes are "absorbed by floating-major resolution". The default loader pins an exact release (`schema.DefaultSchemaModule`, `opm/schema/loader.go:43`), and a bare major is opt-in. `README.md:112` and `AGENTS.md:221-228` already say so, so the files contradict themselves. `AGENTS.md:353` also introduces the pinned-loader example as what an operator does "for reproducibility", which the default already gives.
- **A package that does not exist.** The `README.md` layout block lists `opm/core/` holding `Compiled` (`README.md:40`). `opm/` has no `core` package; `Compiled` lives in `opm/kernel/render.go:90`.
- **Old verbs and an old pipeline.** `CONSTITUTION.md:64` lists the kernel verbs as "acquire, load, process, validate, synthesize, render"; the load and process verbs are gone. `CONSTITUTION.md:65` calls `opm/module/` the "module and release model"; it is the module and instance model. The pipeline at `CONSTITUTION.md:78-80` (`loader -> module -> schema-validate -> render -> core`) names a validation stage and a `core` package that no longer exist.
- **An old principle name.** `README.md:144` summarizes the constitution with "small batches". Principle VIII is Mergeable Sections (`CONSTITUTION.md:20`).
- **A moved core file.** The site page `docs/site/diagnostics/colliding-contracts.md:35` tells the reader to check against `core/src/platform_contracts_pins.cue`. core moved its pins into `core/src/pins/` (core PR 103), so the file is now `core/src/pins/platform_contracts_pins.cue`.

The owner decided this in the kernel-plan walkthrough (task c2): a separate, small library docs PR for the README, AGENTS and CONSTITUTION front-door fixes, with "floating major" replaced by "pinned by default, bare major opt-in". The supervisor folded in wave-1 follow-up w1-05, the colliding-contracts pins path.

## What Changes

- **Four kinds everywhere.** `README.md:7` and `:19` name all four artifact types, `Catalog` included.
- **The schema is named by its module.** `README.md:27` drops "(v1alpha2)". The README table's schema column and the `AGENTS.md:163` header name `opmodel.dev/core@v2` instead of a version label.
- **Pinned by default, bare major opt-in.** `README.md:97` and `AGENTS.md:351`, `:353` and `:361` say that the default loader pins an exact core release (`schema.DefaultSchemaModule`) and that a bare major (`opmodel.dev/core@v2`) is opt-in. An additive change within a major still needs no Go API change, and the files say how it reaches a render: artifacts re-pin core in their own `cue.mod`, and `DefaultSchemaModule` moves separately in the cascade's `fix(deps)` PR. The pinned-loader code example at `AGENTS.md:356` stays.
- **README layout.** The `core/` row goes; the `kernel/` row names `Compiled`, the terminal output.
- **CONSTITUTION Principle III.** The `opm/kernel/` bullet lists the verbs as built (acquire, synthesize, validate, render). The `opm/module/` bullet says module and instance model. The pipeline block shows acquisition producing the four artifacts, synthesis producing an instance, and `Render` taking an instance and a platform through one CUE build to `[]*kernel.Compiled`; a `Catalog` is read and derived from, never rendered.
- **README further reading.** "small batches" becomes "mergeable sections".
- **Site page.** `docs/site/diagnostics/colliding-contracts.md:35` names `core/src/pins/platform_contracts_pins.cue`.

**Not in this change:**

- The `opm/k8s/` "planned" lines (`README.md:118`, `AGENTS.md:134`, `CONSTITUTION.md:70`). The Kubernetes-tier change (wave-2 `lib-e2e5`) owns them, and this change lands before it.
- The `AGENTS.md` "Release cascade task" paragraph (`:303`) and the `AGENTS.md` repository layout block (`:124-157`), which other wave-2 changes edit.
- The intro sentence "loading, processing, validating, and rendering OPM `#Module`s" in `CONSTITUTION.md:7` and its normative copy in `openspec/config.yaml`, and the "module/release semantics" step of normative Principle II at `openspec/config.yaml:31` (all routed to wave-2 `lib-c4`), the frozen-enhancement pointers at `README.md:55`, `:127` and `:149`, and the `README.md:32` debugValues migration line (wave-2 `lib-i3d2` owns it).
- The nonexistent package scopes `core`, `provider` and `validate` in the commit-scope and task-group lists at `CONSTITUTION.md:134` and `:273` and the `core` scope at `AGENTS.md:374` (same class as the package map fixed here; no wave-2 change owns them yet, a follow-up for the supervisor).
- Godoc and the one-statement-per-fact pass (wave-2 `lib-c4`, which runs after this change).
- Two stale scenarios in the main spec `schema-dispatch` that still describe a bare-major default: "Schema resolved via module identifier" (default `"opmodel.dev/core@v2"`) and "Default resolves within the v2 major". They are spec text, not front-door files; the supervisor routed both to wave-2 `lib-c4` (triage SD25), which owns the SHALL requirements in specs and fixes both (REMOVED plus ADDED under a new name, since a MODIFIED that drops a scenario is refused).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `artifact-types`: adds a requirement that, where the front-door files list the accepted kinds or label the schema, they name all four artifact types and name the schema by its core module instead of a version label.
- `schema-dispatch`: adds a requirement that, where the front-door files say which core the default loader uses, they describe it as an exact pin with the bare major opt-in, or link to this spec.
- `kernel-runtime`: adds a requirement that, where the front-door files list the kernel's verbs and packages, they list them as built.

## Impact

**SemVer: no release.** Nothing under `opm/` changes. Every commit is `docs`, which release-please hides (`AGENTS.md`, "Commit style").

**Affected files:** `README.md`, `AGENTS.md`, `CONSTITUTION.md`, `docs/site/diagnostics/colliding-contracts.md`. No `opm/` package, no test, no task.

**Downstream consumers:**

| Consumer | What it has to do |
| --- | --- |
| cli, opm-operator | Nothing. No Go surface changes. |
| opmodel.dev | Nothing. `docs/site/` ships in the library docs bundle; the changed line is an HTML comment for authors, so the rendered page does not change. |
| Release cascade | Nothing. The cascade warns about core releases that `AGENTS.md` names in prose (main spec `deps-cascade`); the new wording names no release literal. |

**Order and conflicts:**

- Lands before wave-2 `lib-e2e5`, which edits the same three files, and before `lib-c4`, so the comment pass edits fresh text.
- Open change `publish-go-api-bundle` (another session) touches none of these files.
- `AGENTS.md` is a hot file. The hunks here sit at `:163` and `:351-361`, away from the layout block and the cascade paragraph. Whichever PR merges second rebases.
