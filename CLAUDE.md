# Library repository guide

## Purpose

This repo is the **OPM kernel** — the reference Go runtime for Open Platform Model. Consumed as a Go library by every front-end (`cli/`, `opm-operator/`, planned Crossplane composition fn). The repo ships no binary; the only `main` package is `cmd/flow-inspect`, an internal diagnostic CLI.

## Repository Rules

- `CONSTITUTION.md` is the human-readable principle source; `openspec/config.yaml` is normative. Read both before non-trivial changes.
- **Principle VIII (Small Batch Sizes) has a hard execution gate** that blocks oversized requests. If a request is too large (e.g. multi-package refactor, redesigning the compile pipeline in one go, design+implement+test a major feature in one go), respond with the gate phrase from `openspec/config.yaml` § Execution Gate and propose a split.
- **Kernel neutrality (Principle I).** The library is consumed by CLI, controller, and future runtimes. Do not introduce:
  - Global mutable state or package-level singletons hiding behavior.
  - `os.Exit`, direct logging output to stdout/stderr, shell invocation.
  - Hidden env lookups — config arrives explicitly via args.
  - Non-deterministic behavior given identical inputs.
- **Public surface = `opm/` only.** `opm/` packages MUST NOT import command/controller/runtime concerns. Output formatting and presentation stay outside the library. Everything under `opm/helper/` is opt-in — a frontend MAY skip it and call the kernel directly. Anything outside `helper/` is kernel contract. The helper-vs-kernel boundary matters when refactoring: moving a function across it changes SemVer obligations.
- I/O lives at edges (`helper/loader/*`, registry calls) and accepts caller-supplied config. Logging is caller-passed via parameter or `context.Context`.

## Entrypoint

Read these on entry:

- `CLAUDE.md` — repo working rules (this file).
- `CONSTITUTION.md` — design principles (full text).
- `openspec/config.yaml` — normative constitution + OpenSpec artifact rules.
- `README.md` — same big picture as below, slightly fuller prose.
- `MIGRATIONS.md` — every pre-release API break with migration recipe.
- `docs/getting-started.md` — end-to-end embedding walkthrough.
- `docs/design/kernel-validate-flow.md`, `docs/design/compile-pipeline-known-gaps.md` — flow notes.

## Repository Layout

```text
opm/
  apiversion/                 Version type + Detect(cue.Value) reads apiVersion off any artifact
  core/                       Platform-neutral primitives: Compiled, Resource, Identity
  errors/                     Sentinels + grouped CUE diagnostics (alias as oerrors in consumers)
  kernel/                     PUBLIC ENTRY POINT — Kernel struct, phase methods, validate helpers
  module/                     *module.Module / *module.Release types + value-validation accessors
  platform/                   *platform.Platform — kernel's sole match/execute input
  compile/                    finalize → match → execute → emit pipeline (no public entry; called via Kernel)
  schema/                     OPM core schema loader (OCILoader, Cache) + CUE paths + metadata decoders
  helper/                     OPT-IN convenience for frontends (a frontend MAY skip this entire tree)
    loader/file/              Filesystem loaders: LoadModulePackage, LoadReleasePackage, LoadPlatformFile
    loader/bytes/             In-memory loader — SKELETON ONLY, no exported funcs yet
    synth/                    Release(name, ns, ref, values, ...) → cue.Value (no files)
  internal/schematest/        Test-only helper for constructing *schema.Cache against the workspace cache
cmd/flow-inspect/             Internal diagnostic CLI (only main pkg in repo)
adr/                          Architecture decision records (use TEMPLATE.md)
enhancements/                 Long-form library proposals (000-TEMPLATE, 001..007). NOTE: per root CLAUDE.md these are frozen historical predecessors — cite via `legacy:NNN`, never edit, never fork. New cross-cutting OPM work goes in workspace-root enhancements/.
openspec/                     OpenSpec proposals/specs/archives (active change workflow)
modules/                      Test-only CUE modules (opm, opm_platform) — fixtures, not shipped
testdata/                     CUE module fixtures consumed by package tests (synth fixture + test cue.mod)
docs/getting-started.md       End-to-end embedding walkthrough
docs/design/                  Flow diagrams + pipeline gap notes
MIGRATIONS.md                 Pre-release API evolution + breaking-change recipes
.cue-cache/                   Gitignored workspace-local CUE module cache populated by tests
```

### Three artifact types — and nothing else

The kernel accepts exactly:

| Artifact         | Schema (`v1alpha2`)  | Go type              |
| ---------------- | -------------------- | -------------------- |
| `Module`         | `#Module`            | `*module.Module`     |
| `ModuleRelease`  | `#ModuleRelease`     | `*module.Release`    |
| `Platform`       | `#Platform`          | `*platform.Platform` |

`#ModuleDebug` was retired. `debugValues` is now a field on `Module`; whether the frontend layers it into the values stack is helper-layer policy. Don't reintroduce `ModuleDebug` as a top-level artifact.

## Environment Notes

Use the workspace env vars (`CUE_REGISTRY`, `OPM_REGISTRY`) from the root `CLAUDE.md`. A local CUE registry at `localhost:5000` is expected for publish/integration tests.

### Schema cache lifetime contract

The OPM core schema is fetched at runtime via `opm/schema.OCILoader` (resolves
`opmodel.dev/core@v0` against `CUE_REGISTRY`) and memoized in a
`*schema.Cache` owned by each `*kernel.Kernel`. Lifetime rules:

- **One Cache per Kernel.** Constructing two Kernels creates two Caches; they
  share the on-disk CUE module cache (`$CUE_CACHE_DIR`, by default
  `~/.cache/cuelang/mod/`) but not the in-process memoized `cue.Value`.
- **Long-running consumers (operator, server) MUST keep the Kernel alive
  across operations.** The schema fetch happens once per Kernel-instance on
  first `Cache.Get`; subsequent calls return the cached value with no
  registry round-trip.
- **Short-lived consumers (CLI, tests) pay one fetch per cold disk cache,
  then hit the warm CUE cache.** A repeated CLI invocation in the same
  process tree gets the same disk cache; a fresh checkout (or a deleted
  `$CUE_CACHE_DIR`) re-fetches once.
- The library auto-applies no `CUE_REGISTRY` default. Frontends (CLI,
  operator) MUST set `CUE_REGISTRY` (e.g. to `schema.PublicRegistry`,
  which maps `opmodel.dev` → `ghcr.io/open-platform-model`) before the
  first schema-touching Kernel call. Tests use the workspace-local cache
  via `opm/internal/schematest`.

### Materialize lifetime & registry contract

`Materialize` (`opm/materialize`, reachable as `(*Kernel).Materialize`)
resolves a `#Platform`'s `#registry` subscriptions into a sealed
`*MaterializedPlatform` (composed transformers + `#matchers` filled). Lifetime
and registry rules:

- **Explicit and caller-driven — the kernel holds no materialize cache**
  (Principle I). Every `Materialize` call performs registry I/O (version
  enumeration + OCI pulls). Long-running consumers that want memoization wire
  their own `opm/materialize/cache.MaterializeCache` (reference `LRU` +
  `Key(*platform.Platform)` over the `#registry` subtree). Invalidation policy
  is theirs: the operator keys it on a CR generation; the CLI opts out and
  relies on CUE's on-disk module cache.
- **Registry config mirrors the schema loader.** `(*Kernel).WithRegistry` sets
  the `CUE_REGISTRY` mapping for catalog (and the materialize-path schema)
  resolution; absent it, the kernel inherits process `CUE_REGISTRY` and
  auto-applies no default. The mapping is plumbed into `load.Config.Env` for
  the operation — never written back to the process environment.
- **Same `*cue.Context` throughout.** The owner's context builds the platform
  value AND every pulled catalog, so the filled `#composedTransformers` /
  `#matchers` share one context with the platform (cross-context values cannot
  be filled together).
- **Inputs are not mutated; failures fail-fast** as `*oerrors.MaterializeError`
  (`Kind: "catalog"`) naming the offending subscription path and version.
- Tests stand up an in-memory OCI registry (`mod/modregistrytest`) with inline
  `#Catalog` fixtures while resolving `opmodel.dev/core@v0` from the warm
  workspace cache — no test-only `Loader` backdoor; the production
  resolver→client→loader path runs unchanged.

## Build And Dev Commands

### Core commands

```bash
task fmt        # gofmt + goimports
task vet        # go vet ./...
task lint       # golangci-lint
task test       # go test ./...
task check      # all four (use before merge)
task check:fast # skips lint

task test:run TEST=TestName          # single Go test
task test:verbose                    # -v across all packages
task test:coverage                   # writes coverage.out + coverage.html

task build      # go build ./... (no binary produced)
task tidy       # go mod tidy
```

### CUE-module tasks

The repo vendors CUE modules under `modules/opm`, `modules/opm_platform`, and `testdata/modules/*` for tests and fixtures; production schema resolution is via `CUE_REGISTRY` against the published `opmodel.dev/core@v0`. Modules are auto-discovered via `CUE_MODULE_GLOBS` in `Taskfile.yml`.

```bash
task cue:discover            # list discovered modules + deps
task cue:fmt                 # cue fmt across all
task cue:vet                 # cue vet across all (CONCRETE=true for -c)
task cue:check               # fmt + vet
task cue:tidy                # cue mod tidy across all
task cue:publish:smart       # checksum-detect changes, bump, publish in dep order (DRY_RUN=true to preview)
task cue:publish PATH=modules/opm [VERSION=vX.Y.Z]
task cue:deps:update         # cue mod get + tidy across all
```

### Schema-fixture + flow tests

```bash
task cue:test                                   # runs TestSchemaFixtures (table-driven CUE fixture harness)
task cue:test:run CASE=<schemaCase.name>        # single fixture subtest
task cue:test:eval FIXTURE=<file.cue>           # bypass Go harness — `cue eval -t test ./testdata/<f>`
task cue:test:flow                              # plan→match→compile integration test (skips if registry unreachable; OPM_FLOW_TEST_FORCE=1 to require it)
task cue:test:flow:inspect [STAGES=plan,...]    # pretty-print each pipeline stage via cmd/flow-inspect
```

## Coding Standards

### Kernel API surface

`*kernel.Kernel` is the single entry point. Four phase-explicit methods map to frontend subcommands:

- `Kernel.Validate` — vet
- `Kernel.Match` — match components ↔ transformers via `Platform.#matchers`
- `Kernel.Plan` — plan / preview
- `Kernel.Compile` — apply / render → `*kernel.CompileResult` carrying `[]*core.Compiled`

The free-function entry points (`compile.CompileModuleRelease`, `compile.ProcessModuleRelease`, `module.ParseModuleRelease`) have been removed — construct a `Kernel` and call its methods. There is no standalone `opm/validate/` package; validation lives on the `Kernel` (`ValidateConfig`, `ValidateConfigPartial`, `ValidateConfigDetailed`, plus typed shortcuts in `kernel/validate_typed.go`).

`*core.Compiled` is terminal output — adapters in downstream impls wrap each one with a platform-specific `core.Resource` filling `core.Identity`. Don't push platform-native identity into the kernel.

### Compile pipeline (per release)

```
loaderfile.LoadReleasePackage  → cue.Value
Kernel.ProcessModuleRelease    → *module.Release (validated, concrete)
Kernel.Compile                 → *kernel.CompileResult
        compile.FinalizeValue       strip schema constraints from components
        compile.Match               component ↔ transformer pairing
        compile.Module.Execute      per-pair transformer execution
              FillPath #component, #context.{moduleReleaseMetadata, componentMetadata, runtimeName}
              decode `output` (ListKind | StructKind dispatch)
              emit []*core.Compiled with Release/Component/Transformer FQN provenance
```

### OPM schema versioning

The schema lives in the `opmodel.dev/core` CUE module, resolved at runtime via `CUE_REGISTRY` and cached per-Kernel in `*schema.Cache`. Versioning is per-OCI-module-version: `opmodel.dev/core@v0` for the floating major, `opmodel.dev/core@v0.X.Y` for a pinned release.

Operators wanting reproducibility pin the schema version explicitly:

```go
k := kernel.New(kernel.WithSchemaLoader(schema.OCILoader{Module: "opmodel.dev/core@v0.3.0"}))
```

Inspect what got resolved at runtime via `k.SchemaCache().ResolvedVersion()` after the first schema-touching call.

A shape-breaking schema change is a coordinated event: the `core` repo publishes the new shape, the library's Go code in `opm/schema` and `opm/compile` adapts to the new paths, and downstream consumers re-pin. Within a major (`@v0`), additive schema changes are absorbed transparently by floating-major resolution.

Two independent compat tracks, never confuse:

- **Go-module SemVer** — Go types/signatures consumed by binaries. Breaking change → MAJOR library bump.
- **OPM schema versioning** — CUE module versions resolved via `CUE_REGISTRY`. Within a major, kernel MUST adapt to additive schema changes. A shape break in the schema is itself a library-breaking event.

### Imports + style

Standard Go grouping with blank lines between groups: stdlib → external (incl. `cuelang.org/go`) → `github.com/open-platform-model/library/...`. Let `gofmt`/`goimports` handle it. Accept interfaces, return concrete structs. Propagate `context.Context` through I/O and CUE evaluation. Wrap errors: `fmt.Errorf("loading module: %w", err)`. Reuse `opm/errors` types.

### Commit style

Conventional Commits v1: `type(scope): description`. Scopes match packages: `core`, `loader`, `module`, `provider`, `render`, `kernel`, `errors` (plus `api`, `apiversion`, `compile`, `helper`).

## Working Style for Agents

- Apply the small-batch hard gate before starting work — split oversized requests using `openspec/config.yaml` § Execution Gate phrasing.
- Pick the right destination for new work:
  - **Cross-cutting OPM design** (spans `core/`, `library/`, `catalog/`, `opm-operator/`, etc.) — workspace-root `enhancements/`, never `library/enhancements/`.
  - **Library-scoped slice of a cross-cutting enhancement** — OpenSpec change under `openspec/changes/` here.
  - **Architecture decision purely about library internals** — `adr/<NNN>-<slug>.md` (use `adr/TEMPLATE.md`).
  - **Schema change** — almost always `core/`. Catalog primitives built on top → `catalog/`. Editing `core/*.cue` requires the `core-schema-edit` skill (`core/.claude/skills/core-schema-edit/SKILL.md`) — SPEC.md co-update is pre-commit-gated.
- Run `task check:fast` for iterative work, `task check` before merge.
- When changing kernel-exposed signatures, check downstream impact in `cli/` and `opm-operator/` consumers and update `MIGRATIONS.md`.
- Don't reintroduce removed top-level artifacts (`#ModuleDebug`) or free-function entry points (`compile.CompileModuleRelease`, etc.).
