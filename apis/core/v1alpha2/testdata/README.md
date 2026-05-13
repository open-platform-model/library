# `apis/core/v1alpha2/testdata/`

CUE fixtures exercising the v1alpha2 schemas (`#Platform`, `#PlatformBase`, `#Module`, `#ComponentTransformer`, `#TransformerContext`, `#Resource`, `#Trait`, …). Loaded by the Go harness in `opm/api/v1alpha2/schema_fixture_test.go`.

This directory does **not** ship to consumers. The `//go:embed` pattern in `apis/core/embed.go` (`v1alpha2/*.cue`) is non-recursive and excludes `testdata/`. `opm/api/v1alpha2/embed_test.go` enforces this.

## Why a separate convention

CUE's `*_test.cue` filename is reserved for future native test functionality and is **currently ignored by the loader** (`research/cue/cli/inputs.md:26`). We do not use that name; we use plain filenames gated by `@if(test)` (`research/cue/cli/injection.md:43`), which is the actively-supported file-level inclusion mechanism.

Without `-t test` (CLI) or `Config.Tags: []string{"test"}` (Go SDK), every fixture in this directory is excluded from the build. `cue vet ./...` from `apis/core/v1alpha2/` will not see them. Consumers of `apis.core.EmbeddedSchema` will not see them.

## File conventions

Every fixture file MUST:

1. Begin with the file-level attribute `@if(test)` on its first non-blank line, **before** the `package` clause.
2. Declare `package fixtures`.
3. Import the schemas via `core "opmodel.dev/core/v1alpha2@v1"` (the canonical import path used across the codebase — see `modules/opm/module.cue`).
4. Expose at least one of:
   - `input:` — the construction under test, typed against a v1alpha2 schema definition.
   - `expect:` — concrete equality target. The harness asserts positive equality by evaluating `input & expect` under `Validate(cue.Concrete(true))`.

Filename convention: `<topic>_fixture.cue` (e.g. `platform_matchers_fixture.cue`, `multi_fulfiller_fixture.cue`). Avoid the `_test.cue` suffix and any leading-underscore filename — both are loader-excluded by independent rules and would mask `@if(test)` regressions.

## Authoring `#Module`, `#Platform`, etc.

Use the **embed pattern**, not the `&` conjunction:

```cue
input: {
    core.#Module
    metadata: {
        name:       "demo"
        modulePath: "example.com/demo"
        version:    "0.1.0"
    }
}
```

The `&` form (`core.#Module & {metadata: {...}}`) is rejected by the schema's `metadata.modulePath: metadata.modulePath` self-reference idiom in closed-struct context. The embed form mirrors how live modules in `modules/opm/` author `#Module`.

## Adding a new fixture

1. Create `apis/core/v1alpha2/testdata/<topic>_fixture.cue` following the conventions above.
2. Verify locally:
   ```bash
   cd apis/core/v1alpha2
   cue eval -t test ./testdata/<topic>_fixture.cue        # should evaluate
   cue eval ./testdata/<topic>_fixture.cue                # should error: "@if(test) did not match"
   cue vet ./...                                           # should ignore the fixture
   ```
3. Add a row to the `schemaCases` table in `opm/api/v1alpha2/schema_fixture_test.go`. For:
   - **Positive equality** (`input` unifies with `expect`): leave `expectError` empty. The harness asserts `Validate(cue.Concrete(true))` succeeds on the unified value.
   - **Positive value extraction**: set `assertField` to a CUE path (e.g. `"input.metadata.uuid"`) and `assertValue` to the expected Go-decoded value.
   - **Negative**: set `expectError` to a regex matching the expected `error.Error()` substring.
4. Run `go test ./opm/api/v1alpha2/... -run TestSchemaFixtures`.

## Bundling multiple cases per fixture (`inputPath` override)

The default contract reads `input:` and `expect:` at the top level of each fixture. Set `inputPath` on a `schemaCase` to drive a different top-level field, letting one fixture file carry several independent cases. The harness then derives the paired positive-equality target as `<inputPath>_expect` (or stays at `expect` when `inputPath` is the default `"input"`).

Use this when:

- The cases are tiny (one-line type-regex assertions, presence/absence checks) and would proliferate if split across files.
- The cases share authoring context (same imports, same comment header).

Avoid it when:

- Cases need substantial fixture state (modules, transformers, registries) — splitting into separate files keeps each fixture readable.
- The cases produce different positive-equality `expect` shapes that don't share natural prefixes.

Canonical example: `type_regex_fixture.cue` declares `bad_name_uppercase`, `bad_fqn_no_version`, etc. The harness rows pair each with `expectError: "invalid value"` and a distinct `inputPath`.

## Caveat: `close()` does not forbid extra keys on open maps

Several v1alpha2 schemas (`#TransformerContext.labels`, `…annotations`, etc.) end with `...`, leaving them open to additional keys. Wrapping the `expect:` block in `close({...})` does NOT cause unification to fail when `input` carries an unexpected extra key, because the underlying input field is open. To assert "exactly N keys with these names and values," combine the positive-equality block with a length check using a hidden field:

```cue
expect: {
    labels: {
        "k1": "v1"
        "k2": "v2"
    }
    _labelKeyCount: len([for k, _ in input.labels {k}]) & 2
}
```

The `& 2` concrete-unifies the count; a regression that produces 3 keys fails with `conflicting values 3 and 2`. See `transformer_context_labels_fixture.cue` for a fully worked example.

## Layering reminder

Because the `//go:embed` pattern is non-recursive, files added to `testdata/` are automatically excluded from the embedded `Schema` filesystem. `embed_test.go` asserts this and will fail loudly if anyone broadens the embed pattern. Do **not** widen the embed glob to `v1alpha2/**/*.cue` — fixtures would ship to every consumer's binary.
