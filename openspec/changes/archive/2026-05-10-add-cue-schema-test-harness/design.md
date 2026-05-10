## Context

The `apis/core/v1alpha2/` schemas mix pure types with non-trivial computation: `#PlatformBase` projects `#registry` into `#knownResources` / `#knownTraits` / `#composedTransformers`, builds the per-FQN candidate index in `#matchers`, and computes `_invalid` from sorted predicate signatures; `#Platform` then forces a hard fail through `_noMultiFulfiller`. `#TransformerContext` merges three label/annotation scopes deterministically; `#Module.metadata` derives `fqn` and the UUIDv5 `uuid` from authored fields. None of this is currently exercised by tests. `pkg/api/v1alpha2/platform_test.go` covers Go binding paths, not CUE-side computation. Today the only place a regression would fire is during downstream consumption (CLI, controller, opm-operator), which is far too late.

CUE itself does not yet have native test support. `research/cue/cli/inputs.md:26` states `*_test.cue` files are reserved and currently ignored by the loader; combined with the general rule that files starting with `_` are skipped, the name is doubly disqualified. The supported mechanism today is `@if(<tag>)` (`research/cue/cli/injection.md:43`) — a file-level attribute placed before `package` that includes the file only when the named tag is present. The Go SDK driver is `load.Config.Tags` (`research/cue/sdk/load.md:97-114`). This is the same mechanism CUE itself uses internally to keep environment-specific files out of default builds, and it is forward-compatible: when CUE eventually ships native testing, we can rename / reshape the layer without paying for the migration twice.

The existing `apis/core/embed.go` ships the schemas to Go consumers via `//go:embed cue.mod/module.cue v1alpha2/*.cue`. `pkg/api/v1alpha2/embed_test.go` (lines 79–87) asserts the embedded list equals the on-disk list under `v1alpha2/*.cue`. Anything we add into `v1alpha2/` therefore either (a) ships to consumers in `Schema embed.FS`, bloating their binary, or (b) breaks the embed assertion. Both are unacceptable. Test fixtures must live somewhere the embed pattern does not match.

## Goals / Non-Goals

**Goals:**

- Exercise the computed projections of `#PlatformBase` (registry → known* views, matcher candidate index) and `#Platform` (`_noMultiFulfiller` hard fail) end-to-end against representative inputs.
- Exercise the label/annotation merge behavior of `#TransformerContext` and the FQN/UUID derivation of `#Module.metadata`.
- Provide a stable, mechanical contract for adding both positive (equality on a computed value) and negative (expected error matching a regex) cases without touching production schemas or the embedded FS.
- Keep fixtures readable as plain CUE — `cue fmt` works, imports work, schema-side errors point to fixture lines.
- Run the harness as part of `task test` with no extra setup.

**Non-Goals:**

- Pure-CUE negative testing. Errors short-circuit evaluation; CUE has no `try`/`catch`. We accept the asymmetry: positive cases are easiest in pure CUE, negative cases are easiest from Go where we can inspect `error.Error()`. The harness covers both consistently.
- A bespoke testing DSL on top of CUE. We rely on the language's existing equality-via-unification and `Validate(cue.Concrete(true))` semantics. No new field conventions beyond `input:` and (optionally) `expect:`.
- Snapshot / golden-file infrastructure. If we want golden output later we can add it; the current schemas are small enough that inline `expect:` blocks are cheaper to read than separate goldens.
- Tooling for downstream module authors to write fixtures against their own modules. That is a future capability, not this change. The convention here is for the core schema team only.

## Decisions

### Decision 1: `@if(test)` file-level gating, not `*_test.cue` filenames

**Choice:** Each fixture file begins with `@if(test)` on the first line, before `package fixtures`.

**Rationale:** `*_test.cue` is reserved for CUE's future native test feature and is currently ignored by the loader (`research/cue/cli/inputs.md:26`). Adopting that name would force a rename when CUE ships its real implementation, and would mean we are loading files specifically by working around the loader's exclusion rule. `@if(test)` is the actively supported mechanism: documented, exercised by CUE's own SDK examples (`research/cue/sdk/load.md:97-114`), and with clear behavior — the file is excluded from any build that does not pass `-t test` / `Tags: []string{"test"}`. `cue vet ./...` from `apis/core/v1alpha2` will not see the fixtures; the Go harness, which sets the tag, will.

**Alternatives considered:**

- **`_test.cue` filenames.** Rejected: doubly excluded (filename starts with `_` *and* `*_test.cue` reserved), forward-incompatible with native testing, and indirectly relies on us paying attention to a moving target.
- **Underscore-prefixed directory `_testdata/`.** Workable — CUE's general rule excludes `_`-prefixed dirs — but combines two implicit exclusion mechanisms (dir prefix + filename) with no positive declaration of intent. `@if(test)` makes intent explicit, file-by-file.
- **Whole-package isolation under `apis/core/testdata/v1alpha2/`.** Symmetrically clean from CUE's perspective but requires harness fixtures to redeclare module import paths and complicates IDE/tooling. Keeping fixtures within the schema package's directory tree (just one level deeper, in `testdata/`) preserves import ergonomics.

### Decision 2: Fixtures live in `apis/core/v1alpha2/testdata/`

**Choice:** New directory `apis/core/v1alpha2/testdata/`. Fixtures named `<topic>_fixture.cue`. A `README.md` documents the convention.

**Rationale:** The `//go:embed v1alpha2/*.cue` pattern in `apis/core/embed.go` is non-recursive — `testdata/` is invisible to the embed machinery. Go's `go test` already treats `testdata` as a special, ignored directory at compile time, so dropping non-Go files there is a known idiom. The fixtures still resolve `import core "opmodel.dev/core@v1:v1alpha2"` because they sit inside the `apis/core` CUE module. We preserve the embed contract and keep `embed_test.go` honest by extending its disk-vs-embed comparison to assert `testdata/` is absent — if anyone broadens the embed pattern to `v1alpha2/**/*.cue` later, the test fails loudly.

**Alternatives considered:**

- **`apis/core/v1alpha2-tests/`** as a sibling package. Rejected: forces a separate `cue.mod` or duplicates module setup; cross-package imports work but feel heavyweight for what amounts to a fixture cupboard.
- **`pkg/api/v1alpha2/testdata/`** next to the harness. Rejected: places CUE fixtures away from the schemas they exercise, and forces the harness to walk back to `apis/core` for the module root.

### Decision 3: Fixture file shape

**Choice:** Each fixture declares `package fixtures` and exposes one or both of:

- `input:` — the construction under test, typed against the relevant schema definition (`core.#Platform`, `core.#Module`, etc.).
- `expect:` — concrete equality target. Unifying `input` with `expect` reduces to `input` if equal, `_|_` otherwise. The harness asserts `Validate(cue.Concrete(true))` on a top-level `_assert: input & expect` field for positive equality cases.

For negative cases the harness ignores `expect:` and relies on `Validate(cue.Concrete(true))` on `input:` returning an error whose message matches the case's regex.

**Rationale:** Two named fields cover both modes. CUE's built-in unification *is* the equality assertion — we are not inventing a comparison primitive, just naming the target. The shape is small enough to memorize and matches how schema authors already mentally model "inputs vs computed outputs."

**Alternatives considered:**

- **Per-case attribute markers (`@case(positive,name=...)`)**. Rejected as YAGNI. We have a Go table for case metadata; encoding the same data in CUE attributes adds parsing without paying for itself.
- **One fixture file per case.** Rejected: drives file count up and fragments shared imports / setup. One file per *topic* (per schema area being exercised) groups related cases readably.

### Decision 4: Go harness shape — table-driven, single test function

**Choice:** One file `pkg/api/v1alpha2/schema_fixture_test.go`. A single `TestSchemaFixtures` runs through a `[]schemaCase` table. Each case names: a fixture file, an optional CUE path + decode target for positive equality, and an optional regex for negative error matching. The harness uses `cuelang.org/go/cue/load` with `Config.Dir = apis/core` and `Tags = []string{"test"}`. Compilation goes through `cuecontext.New().BuildInstance(insts[0])`, evaluation via `Validate(cue.Concrete(true))`.

**Rationale:** Mirrors the existing style in `pkg/api/v1alpha2/platform_test.go` — same `cuecontext` / `cue.Path` plumbing the team already reads fluently. A single table keeps the schema-test inventory in one scrollable place. The test file lives in `pkg/api/v1alpha2/` (not `apis/core/...`) because Go test files must sit in a Go package; `pkg/api/v1alpha2/` is the natural neighbor.

**Alternatives considered:**

- **One Go file per fixture.** Rejected: duplicates loader plumbing across files; reading the test inventory means opening 5+ files instead of one.
- **`txtar` / `cuetxtar`-style harness.** Rejected as YAGNI. CUE itself uses `cuetxtar` because its test inputs are arbitrary mixes of files, registry state, and stdout/stderr expectations. Our inputs are a single `.cue` file plus a regex; a flat Go table is enough.
- **Per-case `t.Run` discovery via filesystem walk.** Rejected: loses the explicit table that makes "what is currently tested" obvious. Fixture files without a corresponding table entry would silently never run.

### Decision 5: Embed-exclusion regression assertion

**Choice:** Extend `pkg/api/v1alpha2/embed_test.go` (specifically `TestEmbeddedSchema_FileSetMatchesDisk`) to assert no path under `testdata/` appears in the embedded FS, even after a future broadening of the embed pattern.

**Rationale:** The whole `@if(test)` story relies on fixtures *not* shipping to consumers. If `apis/core/embed.go` is later changed (well-intentioned: "embed all `.cue` files"), the test fixtures would silently bloat every consumer binary and surface in their CUE evaluation. A loud test failure is the cheap insurance.

**Alternatives considered:**

- **Trust reviewer vigilance.** Rejected: subtle, easy to miss, fails open.
- **Comment in `embed.go` only.** Rejected: comments are not enforcement.

## Risks / Trade-offs

- **`@if(test)` predicate edge cases** → Mitigation: seed three fixtures (one positive, two negative) up front. Run both `cue vet ./...` (must ignore them) and `go test ./pkg/api/v1alpha2/...` (must execute them) before merging. If a corner case bites, we contain it before scaling.
- **Fixture drift from production schemas** → Mitigation: fixtures are typed against the live schema definitions (`core.#Platform`, `core.#Module`). Schema-breaking refactors will fail the fixture build, surfacing the impact at the same time as the change.
- **Asymmetry between positive (CUE) and negative (regex) assertions** → Accepted. Pure-CUE negative testing requires either `_|_`-as-value gymnastics or `tool/exec` with shell-side exit-code inversion; both are worse than a regex. Negative tests are rarer than positive, and a regex against `error.Error()` is the same shape used elsewhere in the codebase (e.g. `pkg/kernel/validate_test.go`).
- **Harness invisible to non-Go contributors** → Mitigation: `apis/core/v1alpha2/testdata/README.md` documents how to add a fixture and how to add a corresponding row to the Go table. The README is the contract for non-Go contributors — they author CUE, ping a reviewer to add the table row, and the harness picks it up.

## Migration Plan

Not applicable — purely additive test infrastructure. No production code is modified, no schema requirement changes, no public Go API changes.

## Open Questions

- Should the harness emit a "no case matched" error if `apis/core/v1alpha2/testdata/*.cue` contains a fixture file that is not referenced by any `schemaCase`? Leaning yes (catches drop-and-forget mistakes), but defer to a follow-up if it complicates the seed implementation.
- Whether to expose a `TestMain` that lints fixtures with `cue fmt --check` against `testdata/` to enforce formatting consistency. Likely yes, but small enough to add later.
