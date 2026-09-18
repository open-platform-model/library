# Design: read-comparable-predicates

## Context

See `proposal.md` § Why. The decision is `enhancements/0015` D5 (comparable predicates within one catalog-fulfilled bucket are refused, not arbitrated) with OQ9 resolved by core's `comparable-predicate-guard`: core derives the report, the generation step refuses, and the library's obligation is to read the report off a built value and to pin the shipped catalog against it.

Current state, measured 2026-09-18:

- `schema.DefaultSchemaModule` is `opmodel.dev/core@v2.0.0-alpha.9`; `registrytest.DefaultCoreVersion` mirrors it and `registrytest_test.go` asserts the two agree (the fixture-drift canary). The literal appears at 38 sites: `loader.go` (4), `loader_test.go` (2), `registrytest.go` (3), `registrytest_test.go` (2), `contracts.go` (2), `render_test.go` (8), `modfile_test.go` (10, some deliberately older skew rows), `stage_test.go` (3), `generate_test.go` (2), `requires_test.go` (2), `requires.go` and `modfile.go` doc comments, the four modules `task cue:deps:update` discovers, `testdata/cue.mod`, twelve `testdata/render/**/cue.mod`, `testdata/render/scenarios/cue.mod`, five `testdata/render/platform*/platform.cue` headers and `modules/opm_platform/platform.cue`. `closure_test.go` and `generate_test.go`'s graph use their own `coreVersion`.
- Core `alpha.9` to `alpha.10` is one commit (core PR 68): `#ContractInventory` gains `comparable` (rows of `broader`, `narrower`, `contracts`) and `discriminated`, derived inside `#Platform.#contracts` from `#composedTransformers`' three required demand maps. The loader gate visits `#registry` entries' regular fields only; the render glue reads `#composedTransformers`, `#registry` and `#contracts.definedBy`. Nothing either reads moved.
- `(*Platform).Contracts()` reads six fields by path off `Package.LookupPath(schema.Contracts)` and errors naming a missing field; its doc says "the six data fields".
- The parity harness's shipped group (`TestParity_ShippedCatalog`) acquires `testdata/parity/opm_platform`, which imports the published `catalogs/opm` build (4.3.1 today; 4.4.0 is the latest release, `derive-registration-provides`, registration contract only), gated by `testing.Short()` and `skipUnlessRegistry`, forced by `OPM_FLOW_TEST_FORCE=1` in CI. Seven case rows carry the catalog version literal.
- The served render fixtures: `cat` (0.1.0 and 0.2.0) ships `deployment-transformer` (container resource plus required label `render.test/workload: stateless`), `service-transformer` (container plus required `ExposeTrait`), `tiered-transformer` (its own resource plus a label) and one-resource-each transformers; `cat2` ships `mirror-transformer` (container resource alone) in both builds and, in 0.2.0, a second `gateway-transformer` over `cat`'s provider-fulfilled gateway. `platform_two` embeds cat 0.1.0 with cat2 0.1.0; `platform_oversubscribed` embeds cat 0.1.0 with cat2 0.2.0; `platform_disabled` disables cat beside cat2 0.1.0; `platform` and `platform_next` embed cat alone. The container resource is catalog-fulfilled (only the gateway resource declares `fulfilment: "provider"`).

## Goals / Non-Goals

**Goals**

- The two new report fields decode on the same accessor, by the same per-path rule, with the same refusal for an inventory that lacks them.
- The library's own fixtures exercise a non-empty `comparable` and a false `discriminated` without a new fixture.
- The published `catalogs/opm` build is asserted discriminated where the library already resolves it, so the catalog's new obligation has teeth outside the catalog repo.

**Non-Goals**

- Any refusal. Core reports; the generation step refuses (0015 D18's shape, D5's site); this change gates nothing (see Research & Decisions).
- A render-time tripwire on `discriminated` in the kernel's render gate.
- A new fixture catalog for the equal-predicate case; that derivation is core's, pinned in core.
- Any change to `platformmodule.Generate`, the in-build single-provider guard, or the match glue.

## Decisions

### The Go surface

`opm/platform/contracts.go` gains one row type and two fields; `Contracts()` reads two more paths:

```go
// ComparablePredicates is one row of the comparable-predicate report: two
// enabled transformers whose match predicates are comparable over at least
// one shared catalog-fulfilled contract (enhancement 0015 D5). Broader
// matches every component Narrower matches; Contracts names the shared
// contracts. A report row, never a refusal: the generation step refuses.
type ComparablePredicates struct {
	Broader   string   `json:"broader"`
	Narrower  string   `json:"narrower"`
	Contracts []string `json:"contracts"`
}

type ContractInventory struct {
	// ... the six existing fields ...
	Comparable    []ComparablePredicates `json:"comparable"`
	Discriminated bool                   `json:"discriminated"`
}
```

The per-path read table gains `{"comparable", &inv.Comparable, "2.0.0-alpha.10"}` and `{"discriminated", &inv.Discriminated, "2.0.0-alpha.10"}`, where the third column is the first release carrying the field and is appended to the missing-field error: `platform #contracts carries no "comparable" field (core derives it from release 2.0.0-alpha.10 on)`. The six existing fields carry `2.0.0-alpha.9` so the message is uniform. `Contracts()` MUST NOT return a partial inventory; a missing report field is the same refusal as a missing `#contracts`.

### Row order

`Comparable` decodes in the build's comprehension order, as `OverSubscribed` does. The accessor does not sort; a consumer that needs a stable order sorts. The tests use `ElementsMatch`.

### The parity pin is its own test function

`opm/kernel/parity_harness_test.go` gains `TestParity_ShippedCatalogDiscriminated`, gated identically to the shipped group (`testing.Short()` skip, `skipUnlessRegistry`, `flowRegistry()`), acquiring `testdata/parity/opm_platform` through the kernel and asserting `Contracts()` reads `Discriminated` true and an empty `Comparable`, with a failure message that prints every row as `broader -> narrower over [contracts]`. A separate function rather than an assertion inside `TestParity_ShippedCatalog`, so a catalog release that breaks discrimination fails by name under `-run`, and a render-case divergence and a discrimination break never mask each other.

### Fixture values pinned before the tests are written

Section 2 opened with a throwaway read through `contractsKernel` on every served platform fixture, recording the rows here. **Measured on core `2.0.0-alpha.10`, 2026-09-18; every value matches the prediction, nothing was corrected.** The FQN prefix `testing.opmodel.dev/library-render` is elided below.

| Fixture | `discriminated` | `comparable` |
| --- | --- | --- |
| `platform` | true | empty |
| `platform_next` | true | empty |
| `platform_two` | false | `cat2/transformers/mirror-transformer@0.1.0` broader against `cat/transformers/deployment-transformer@0.1.0` and against `cat/transformers/service-transformer@0.1.0`, each over `cat/resources/container@v1` |
| `platform_oversubscribed` | false | the same two rows with `mirror-transformer@0.2.0` broader |
| `platform_disabled` | true | empty |

`deployment` against `service` is incomparable (a required label against a required trait), so neither appears as a pair: the two rows per undiscriminated fixture are exactly the resource-only `mirror-transformer` against each of them. The cat2-only platform the accessor tests author in a temp dir lists nothing and reads `discriminated` true with an empty `comparable`, like `platform_disabled`.

### Files touched

| File | Change |
| --- | --- |
| `opm/schema/loader.go`, `loader_test.go` | the pin and its assertions |
| `opm/internal/registrytest/registrytest.go`, `registrytest_test.go` | `DefaultCoreVersion` and its doc |
| `opm/platform/contracts.go`, `contracts_test.go` | the row type, the two fields, the two reads, the release column; tests |
| `opm/kernel/parity_harness_test.go` | the discriminated pin; seven catalog literals |
| `opm/kernel/render_test.go`, `opm/internal/renderstage/{modfile,stage}_test.go`, `opm/helper/platformmodule/generate_test.go`, `opm/catalog/requires_test.go` | version literals |
| `opm/catalog/requires.go`, `opm/internal/renderstage/modfile.go` | doc comments citing the pin |
| the four discovered CUE modules, `testdata/cue.mod`, `testdata/render/**` | cue.mod pins and platform headers |

## Research & Decisions

### An absent report is an error, not a default

**Context**: after the bump, a platform module that pins core `alpha.9` in its own `cue.mod` still acquires (the kernel builds against the module's pins), and its `#contracts` carries six fields and no `comparable`.
**Explored**: (a) default `Discriminated` true and `Comparable` nil when absent; (b) default `Discriminated` false; (c) refuse, naming the field and the release.
**Decision**: (c).
**Rationale**: (a) turns "not checked" into "passed", the one error a gate cannot absorb, and (b) fails every platform on an older core for a report it never derived. The precedent rule for a missing `#contracts` is already (c); the release column makes the message actionable (re-pin the platform module). Both consumers generate their platform modules at the library's verified release, so only hand-authored platform modules can hit it, and they hit it in `opm platform check`.

### No render-time tripwire

**Context**: 0015 D5 fixes the guard at platform-package generation and calls render-build tripwires "defense in depth, not the gate". The kernel's render gate today raises `OverSubscribedContractsError` from render-time diagnostics (`RenderDiagnostics`), a different path from the platform inventory.
**Explored**: reading `Contracts().Discriminated` inside `Render` and refusing; leaving the refusal to the two consumers, each of which acquires the platform once.
**Decision**: no tripwire here.
**Rationale**: the property belongs to the platform, not to a render, and evaluating the inventory per render pays the pairwise fold on every instance for a fact the consumer can ask once at acquisition. The operator's generation gate and `opm platform check` are the named sites; the CLI's local-build path (a platform module directory, no generation step) is the one runtime with no gate today and decides in the cli slice whether to read the inventory at platform resolution. A kernel tripwire remains an additive follow-up if a runtime turns out to need it.

### Where the shipped-catalog pin lives

**Context**: core's proposal assigns the "shipped catalog stays discriminated" check to the library's parity harness, because core imports no catalog.
**Explored**: the parity shipped group (already acquires the published build under the GHCR gate); the flow test (`task cue:test:flow`, renders the web_app fixture through the published catalog); `catalog_opm`'s own `task vet`.
**Decision**: the parity harness, as its own gated test function.
**Rationale**: it is the one place the library resolves the published catalog into a `#Platform` and holds the version pin the check is about; the flow test asserts render output, not platform reports; `catalog_opm` cannot derive the report, which lives on `#Platform`, without authoring a platform fixture of its own, and that repo's gate is a cue vet, not a Go test.

### The catalog moves with the core pin

**Context**: `task cue:deps:update` bumps every direct dependency of the four discovered modules to its latest published version: core to `alpha.10` and `catalogs/opm` from 4.3.1 to 4.4.0. Seven literals in `parity_harness_test.go` name the catalog version, and a catalog bump has broken the shipped group on every previous run of the task.
**Explored**: moving only core by hand (`cue mod get opmodel.dev/core@v2.0.0-alpha.10` per module) and leaving the catalog at 4.3.1; running the task and moving the literals.
**Decision**: run the task; move the seven literals in the same fixture commit.
**Rationale**: the discriminated pin is worth most against the latest shipped catalog, which is the release a new platform embeds; 4.4.0 changes the registration contract only, so the parity cases' rendered bytes are expected unchanged, and the harness proves it. A divergence, if one appears, is a real finding recorded under Risks before the section closes, never papered over.

### Section 1 is a spike

**Context**: design.md carries one unverified assumption: that the suite is green on `alpha.10` and catalog 4.4.0 with no glue change.
**Decision**: section 1 lands the pin and re-runs `task check` and the gated GHCR tests before any reader is written; section 2 opens by measuring the fixture rows.

## Risks / Trade-offs

- [The fixture `tiered-transformer` declares a non-string required label value] → core's predicate fold interpolates the value into the token; `cue vet` on the served catalogs at `alpha.10` in section 1 is the check, and if the fold rejects it the fixture's value moves to a string, which changes no render assertion (the tiered resource never carries the label).
- [Catalog 4.4.0 changes a parity case's rendered bytes] → the harness names the case and the first differing path; recorded here as a finding, and the case table is re-pinned only if the difference is the catalog's, not the kernel's.
- [`Contracts()` on an `alpha.9`-pinned platform now errors] → named in the proposal's Classification; consumer-generated platform modules follow the pin, hand-authored ones re-pin on the message.
- [Cost of the pairwise fold on a `Contracts()` call] → measured by core at roughly +0.01 s on the shipped catalog's 8-transformer bucket; the accessor stays on demand, so a render pays nothing.
- [`opm-operator` and `cli` fixtures pin core `alpha.9`] → outside this change; the root `task deps:pins:fixtures` moves them when their repos bump the library.

## Migration Plan

One PR, three sections, squash title `feat(platform): read the comparable-predicate report off a built platform`. release-please cuts `v1.0.0-alpha.32`. Rollback before release is a revert; after, a later release. `cli` and `opm-operator` pick the release up through Dependabot and read the two fields in their own 0015 slices.

## Open Questions

None.
