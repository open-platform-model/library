## 1. Spike: confirm `@if(test)` behavior end to end

- [x] 1.1 Create scratch fixture `apis/core/v1alpha2/testdata/_spike.cue` with `@if(test)` first line, `package fixtures`, and a trivial `input: core.#Module & {...}`
- [x] 1.2 Run `cue vet ./...` from `apis/core/v1alpha2/` and confirm zero output / zero load of the spike file
- [x] 1.3 In a Go scratch test, call `load.Instances([]string{"v1alpha2/testdata/_spike.cue"}, &load.Config{Dir: <apis/core>, Tags: []string{"test"}})` and confirm the fixture compiles and exposes `input`
- [x] 1.4 Re-run the same Go load *without* `Tags` and confirm the file is excluded
- [x] 1.5 Delete the spike fixture once behavior is confirmed (its purpose is to gate the rest of the work; if any of 1.2/1.3/1.4 disagrees with the design, stop and revisit Decision 1 before continuing)

> Spike findings (preserved for the README + design): conjunction syntax `core.#Module & {...}` is rejected by `metadata`'s self-referential closure idiom (`modulePath: metadata.modulePath`); fixtures MUST use the embed pattern `{ core.#Module; metadata: {...} }`, matching the authoring style in `modules/opm/module.cue`. CLI verification at `apis/core/v1alpha2/`: `cue eval ./testdata/spike_fixture.cue` (no tag) → exit 1 with literal `@if(test) did not match`; `cue eval -t test ./testdata/spike_fixture.cue` → exit 0 with computed FQN + UUIDv5. Step 1.3/1.4 SDK confirmation folded into the harness implementation in §6 — same loader, same Tags semantics.

## 2. Testdata layout + convention doc

- [x] 2.1 Create directory `apis/core/v1alpha2/testdata/`
- [x] 2.2 Add `apis/core/v1alpha2/testdata/README.md` documenting: `@if(test)` requirement, `package fixtures`, meaning of `input:` / `expect:`, location of the Go harness table, and step-by-step instructions for adding a new fixture + table row
- [x] 2.3 Confirm `apis/core/embed.go`'s `//go:embed cue.mod/module.cue v1alpha2/*.cue` pattern still does not match anything under `testdata/` (no edit; assertion only — pattern is non-recursive, so `testdata/*.cue` and `testdata/README.md` are both invisible to it)

## 3. Seed positive fixture: `#PlatformBase.#matchers`

- [x] 3.1 Create `apis/core/v1alpha2/testdata/platform_matchers_fixture.cue` with `@if(test)`, `package fixtures`, `import core "opmodel.dev/core/v1alpha2@v1"` (canonical import form, not the `@v1:v1alpha2` syntax — corrected during spike)
- [x] 3.2 Inside the fixture, build a minimal `#Module` exposing one `#Resource` (FQN `example.com/r/thing@v1`) and one `#ComponentTransformer` whose `requiredResources` references that FQN
- [x] 3.3 Build a `core.#Platform` value `input` that registers the module and is concrete enough to evaluate `#matchers`
- [x] 3.4 Add `expect:` block asserting `#matchers.resources` contains `"example.com/r/thing@v1"` keyed to a single-element list, `#matchers._invalid` is `{resources: [], traits: []}`, and `_noMultiFulfiller: 0`
- [x] 3.5 Confirm by hand: `cue vet -c -t test ./testdata/platform_matchers_fixture.cue + _assert: input & expect` returns exit 0 — concrete unification succeeds

## 4. Seed negative fixture: `_noMultiFulfiller`

- [x] 4.1 Create `apis/core/v1alpha2/testdata/multi_fulfiller_fixture.cue` with the same header conventions
- [x] 4.2 Build two `#ComponentTransformer` definitions on the same `#Module`, both keyed to identical-FQN-target with empty `requiredLabels` / `requiredTraits` so predicate signatures collide (signature `";"`)
- [x] 4.3 Register the module on a `core.#Platform` so `#matchers._invalid.resources` is non-empty and `_noMultiFulfiller` triggers
- [x] 4.4 Confirmed by hand: `cue vet -c -t test ./testdata/multi_fulfiller_fixture.cue` exit 1 with `input.#matchers._noMultiFulfiller: conflicting values 1 and 0` — regex anchor `_noMultiFulfiller`

## 5. Seed negative fixture: FQN collision across modules

- [x] 5.1 Create `apis/core/v1alpha2/testdata/fqn_collision_fixture.cue` with the same header conventions
- [x] 5.2 Build two `#Module`s, each with `#defines.resources` keyed to the *same* FQN but with conflicting `metadata.description` values
- [x] 5.3 Register both modules on a single `core.#Platform`; the projection into `#knownResources` surfaces a CUE bottom on the colliding key's `metadata.description`
- [x] 5.4 Confirmed by hand: `cue vet -c -t test ./testdata/fqn_collision_fixture.cue` exit 1 with `#knownResources.<FQN>.metadata.description: conflicting values` — regex anchor `conflicting values`

## 6. Go harness in `opm/api/v1alpha2/`

- [x] 6.1 Create `opm/api/v1alpha2/schema_fixture_test.go` with a `schemaCase` struct (fields: `name`, `fixture`, `assertField`, `assertValue`, `expectError`)
- [x] 6.2 Add a package-level `var schemaCases = []schemaCase{...}` referencing the three seed fixtures created in §3, §4, §5
- [x] 6.3 Implement `TestSchemaFixtures` resolving `apis/core` via `runtime.Caller` (mirrors `embed_test.go:22-28`), `load.Instances` with `Tags: ["test"]`, build via `cuecontext.New().BuildInstance`. Dispatches: positive default → assert `input.Unify(expect)` is concrete; positive `assertField` → decode + equality; negative `expectError` → regex against `built.Err()` OR `input.Validate(Concrete).Error()` (negative cases surface at build OR validate time, both accepted)
- [x] 6.4 Missing fixture → `require.Len(insts, 1)` + `require.NoErrorf(insts[0].Err, ...)` fails the subtest with the fixture filename in the message
- [x] 6.5 `go test ./opm/api/v1alpha2/... -run TestSchemaFixtures -v` — all three subtests pass

## 7. Embed-exclusion regression assertion

- [x] 7.1 Added `TestEmbeddedSchema_ExcludesTestdata` to `opm/api/v1alpha2/embed_test.go`, asserting no embedded path contains `testdata/`
- [x] 7.2 `go test ./opm/api/v1alpha2/... -run TestEmbeddedSchema` — green
- [x] 7.3 Sanity-checked: widened `apis/core/embed.go` to `v1alpha2/*.cue v1alpha2/testdata/*.cue`, test failed with explicit fixture filenames in error, reverted, test green

## 8. Validation gates

- [x] 8.1 `task fmt` — exit 0
- [x] 8.2 `task vet` — exit 0
- [x] 8.3 `task lint` — `0 issues.` exit 0
- [x] 8.4 `task test` — `opm/api/v1alpha2 0.033s ok`; `TestSchemaFixtures` (3 subtests) and `TestEmbeddedSchema_ExcludesTestdata` both green
- [x] 8.5 `cue vet ./...` from `apis/core/v1alpha2/` — exit 0, no fixture paths in output (fixtures excluded by `@if(test)`)
