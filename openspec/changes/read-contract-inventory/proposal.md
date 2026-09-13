## Why

Core `v2.0.0-alpha.9` derives `#Platform.#contracts`, the contract inventory enhancement 0015 D1 promised (defined contracts and their defining catalogs, required demands per contract, the `unfulfilled` and `overSubscribed` reports and the `fulfilled` and `routable` booleans, D2 and D18). The library pins `alpha.7` and nothing in it reads the inventory: a platform's unresolved-demand verdict still cannot say "this contract is defined by catalog X and nothing implements it" (0010 OQ3, the arm D18 names), and no consumer can ask a built platform whether it is fulfilled or routable without re-deriving the fold. 0015's cross-repo ordering (`06-operational.md`) puts `library` next after `core`: read the inventory off built values, sharpen the diagnostic, leave the match glue unchanged. The pin has to move first, and `schema-dispatch` says it moves only by a deliberate change that re-verifies the glue and the fixtures against the new release.

## What Changes

- **Core pin to `v2.0.0-alpha.9`.** `schema.DefaultSchemaModule`, `registrytest.DefaultCoreVersion`, the doc comments that cite the pinned identifier, and the pin assertions in `opm/schema/loader_test.go`; the four CUE modules `task cue:deps:update` discovers (`modules/opm_platform`, `testdata/modules/web_app`, `testdata/parity`, `testdata/parity/opm_platform`); the twelve `testdata/render/**/cue.mod/module.cue` files the render tests serve or load, their platform file headers, and the version literals in `opm/kernel/render_test.go`, `opm/internal/renderstage/modfile_test.go` and `stage_test.go`. No glue change is needed: alpha.7 to alpha.9 is additive (`#Catalog` contract maps, `#Platform.#contracts`), the loader gate visits regular fields only, and the glue references `#composedTransformers` and `#registry` alone.
- **The inventory is readable off a built platform.** `opm/platform` gains `ContractInventory` and `(*Platform).Contracts()`, decoding `#contracts` from `Package` on demand: `DefinedBy`, `RequiredBy`, `Unfulfilled`, `OverSubscribed`, `Fulfilled`, `Routable`. `defined` is deliberately not decoded: its values are the catalog's member schemas, not data, and a caller that wants one reads `Package`. `opm/schema` gains the `Contracts` path constant. Nothing decodes it at construction; `Package` stays the source of truth.
- **The unresolved-demand verdict names the defining catalog.** `oerrors.UnresolvedDemand` gains `DefinedBy`, the registry key of the enabled catalog listing the demanded key (empty when none does); the glue fills it from `platform.#contracts.definedBy`; the row's message distinguishes "defined by `<catalog>`, nothing implements it" from "implemented at a different apiVersion" from "no enabled catalog defines it". The gate does not change: an unresolved demand refuses exactly as before (0010 D28); only the wording of the refusal gains the catalog.
- **The served render fixtures list their contracts.** `testdata/render/registry`'s `cat` and `cat2` catalogs (both builds) gain `#resources` and `#traits` maps over the members they already define, so the accessor and the new arm are exercised against non-empty maps rather than vacuously (0015 `06-operational.md` names the empty-map case as the failure to guard against).
- **Not changed, on purpose.** The in-build single-provider guard stays demand-derived (`guard._providerSuppliers`) and is not replaced by `#contracts.overSubscribed`: the two disagree while any first-party catalog lists nothing, and the swap is a separate change gated on `catalog_opm` publishing populated maps (design.md). `platformmodule.Generate` stays a pure file writer: refusing generation on `routable: false` is the generation step's act (0019 D6), which for the operator is its own 0015 slice, reading `Contracts()` from this change. The matching rungs are untouched (0015 02-design.md: load-bearing that they are).

## Classification

**MINOR, additive** (Principle VI): one new exported type and method in `opm/platform`, one new path constant, one new field on a diagnostics row that JSON-decodes from a build that always emits it. No signature changes, no removed surface. The one observable text change: `UnresolvedDemandsError.Error()` gains the defining catalog for a defined-but-unimplemented demand, so a consumer test asserting the old sentence verbatim moves. Complexity added: one accessor and one row field, each with a named reader below (Principle VII).

## Downstream consumers

- **`cli`** (library `alpha.29`): picks up the next release through Dependabot. The 0015 `opm platform check` change is the first reader of `Contracts()`; the render-validation path that already counts providing catalogs keeps working unchanged. Measured at task 3.4 (cli built against this tree through a scratch `go.mod` replace): no cli test asserts the library's `UnresolvedDemandsError.Error()` sentence. The cli words the rows itself in `internal/cmdutil/output.go` `FormatUnresolvedDemands` (line 71, its own "nothing on this platform implements this contract" line, pinned by `internal/cmdutil/output_test.go` line 127), so the Dependabot bump changes no cli output; the cli's 0015 slice is where that presenter gains the `DefinedBy` arm.
- **`opm-operator`** (library `alpha.28`): built against this tree through a scratch `go.mod` replace at task 3.4, no signature it uses changed, and nothing in it asserts the unresolved-demand sentence.
- **`opm-operator`** (library `alpha.28`): the 0015 operator slice reads `Contracts().Unfulfilled` and `DefinedBy` for the non-gating `ContractsFulfilled` condition and `Routable` for the generation refusal (D18). Nothing until that slice.
- **`catalog_opm`**: independent. Its `catalog-contract-listing` change is what makes the inventory non-vacuous in production; this change's fixtures make it non-vacuous in the library's tests, so the two can land in either order.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `schema-dispatch`: the `DefaultSchemaModule` constant requirement moves the pinned release to `opmodel.dev/core@v2.0.0-alpha.9`, the first release carrying the contract inventory, and its scenarios follow.
- `platform-artifact`: the Platform Type Shape requirement gains the `Contracts()` accessor and the `ContractInventory` decoded view, and is restated against the artifact's current shape (`Metadata`, `Package`, `Source`; the spec still names an `APIVersion` field the type no longer has).
- `single-build-render`: the "Unresolved demands are diagnosed with alternatives" requirement gains the defining-catalog arm: the row carries `DefinedBy` from the platform's inventory, and the message distinguishes defined-but-unimplemented from unknown.

## Impact

- `opm/schema/loader.go`, `loader_test.go`, `paths.go`; doc comments in `opm/kernel` and `opm/schema` citing the pin.
- `opm/internal/registrytest/registrytest.go` (`DefaultCoreVersion`).
- `opm/platform/platform.go` (+ new `contracts.go`, `contracts_test.go`).
- `opm/errors/match.go` (`UnresolvedDemand.DefinedBy`, `describe`).
- `opm/internal/renderstage/render.cue.tmpl` (unresolved rows gain `definedBy`), `opm/kernel/render_decode.go` (no change expected: the row decodes by JSON tag), `render_test.go`.
- `modules/opm_platform`, `testdata/modules/web_app`, `testdata/parity`, `testdata/parity/opm_platform` cue.mod pins; `testdata/render/**` cue.mod pins, platform headers, and the `cat`/`cat2` catalogs' contract maps.
- `opm/internal/renderstage/modfile_test.go`, `stage_test.go`, `opm/kernel/render_test.go` version literals.
- Release: `feat` (library `v1.0.0-alpha.30`).
