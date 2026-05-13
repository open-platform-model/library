## Why

The `apis/core/v1alpha2/` CUE schemas have grown past pure typing. `#PlatformBase` derives `#knownResources` / `#knownTraits` / `#composedTransformers` from `#registry`, builds a per-FQN candidate index in `#matchers`, and exposes `_invalid` to flag duplicate-fulfiller transformers; `#Platform` then promotes that diagnostic to a hard fail via `_noMultiFulfiller`. `#ComponentTransformer` and `#TransformerContext` inherit, sort, and merge labels/annotations across module / component / controller scopes. `#Module` derives `metadata.fqn` and `metadata.uuid` (UUIDv5/SHA1) from authored fields. None of this is currently exercised by tests — regressions in the projection logic would reach downstream consumers (CLI, controller) before any signal fires. The kernel-side Go tests in `opm/api/v1alpha2/` cover Go binding paths, not CUE-level computation.

We need fixture-based tests that (a) confirm computed outputs match expected values for representative inputs and (b) confirm guard rails (`_noMultiFulfiller`, FQN collisions, type unifications) actually reject the inputs they're meant to reject. CUE's `_test.cue` filename is reserved for future native testing (`research/cue/cli/inputs.md:26`) and `*_test.cue` files are currently ignored by the loader, so we cannot rely on it. The supported mechanism today is `@if(<tag>)` file-level inclusion (`research/cue/cli/injection.md:43`), which the Go SDK drives via `load.Config.Tags` (`research/cue/sdk/load.md:97`). Pairing `@if(test)`-gated fixtures with a Go-side table-driven harness gives us both positive equality assertions and negative error-regex assertions without polluting normal `cue vet` runs or the embedded schema FS.

## What Changes

- Add `apis/core/v1alpha2/testdata/` directory holding `.cue` fixtures, each prefixed with the file-level `@if(test)` attribute so they are excluded from `cue vet ./...`, `cue eval`, and any consumer that does not opt in via `-t test`.
- Add fixture authoring conventions: each fixture imports `opmodel.dev/core@v1:v1alpha2`, declares `package fixtures`, and exposes a top-level `input:` (the construction under test) plus optional `expect:` (concrete equality target unified with `input` for positive cases).
- Add a Go test harness `opm/api/v1alpha2/schema_fixture_test.go` that table-drives a list of cases over the fixtures: each case names a fixture file, an optional CUE path + decode target for positive equality, and an optional regex for negative error-message assertions.
- The harness loads fixtures via `cuelang.org/go/cue/load` with `Config.Tags: []string{"test"}` so the `@if(test)` gate opens. Without the tag the loader returns no files — protecting against stale fixtures sneaking into a release build.
- Confirm `apis/core/embed.go`'s `//go:embed` pattern (`v1alpha2/*.cue`) does not match `v1alpha2/testdata/`; extend `embed_test.go`'s disk-vs-embed comparison to assert the test fixtures are explicitly excluded so a future broadening of the embed pattern fails loudly.
- Seed three fixtures + cases as proof of mechanism, covering one positive (`#Platform.#matchers.resources` populated for a single transformer) and two negative (`_noMultiFulfiller` triggered by duplicate predicate signatures; `#defines` FQN collision across two `#Module`s registered on one `#Platform`).
- Document the convention in a short `apis/core/v1alpha2/testdata/README.md` so future contributors know how to add cases without reading the harness.

## Capabilities

### New Capabilities

- `schema-testing`: The `@if(test)`-gated fixture convention plus the Go table-driven harness that loads, evaluates, and asserts against `apis/core/v1alpha2/` CUE schemas. Owns every requirement governing fixture placement, fixture syntax, harness inputs, and pass/fail assertion semantics.

### Modified Capabilities

(none — purely additive test infrastructure; no public Go API or schema requirement changes)

## Impact

**Affected library packages**

- `apis/core/v1alpha2/testdata/`: new directory holding fixtures + README. Outside the embed pattern.
- `opm/api/v1alpha2/`: new `schema_fixture_test.go` and (likely) a small `schema_fixture_helpers_test.go` for shared load/decode plumbing. No production code touched.
- `apis/core/embed.go`: unchanged. `opm/api/v1alpha2/embed_test.go` gains one assertion that `testdata/` is absent from `Schema embed.FS`.

**Affected downstream consumers**

None. Test-only addition; no public type, signature, or behavior change.

**SemVer**

PATCH. Test infrastructure only; no `opm/` surface change.

**Dependencies**

No new external dependencies. Uses `cuelang.org/go/cue`, `cuelang.org/go/cue/cuecontext`, `cuelang.org/go/cue/load`, `cuelang.org/go/cue/errors` — all already imported.

**Risk**

Low. Fixtures live behind `@if(test)` so a misconfigured loader silently drops them rather than poisoning a real evaluation. The harness is a single test file; failure surface is contained. Main risk is that `@if(test)` predicate semantics surprise us on a corner case (e.g. multi-tag combinations) — mitigated by seeding three diverse fixtures up front and running both `cue vet ./...` (must ignore them) and `go test ./opm/api/v1alpha2/...` (must execute them) before merge.
