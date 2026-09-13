# Design: read-contract-inventory

## Context

See `proposal.md` § Why. The decisions are `enhancements/0015` D1, D2 and D18 and the library integration points in its `02-design.md` (re-baselined on 0019): read the inventory off a built value, sharpen the missed-demand diagnostic through membership rather than prefix parsing, leave the match glue unchanged.

Current state, measured 2026-09-13:

- `schema.DefaultSchemaModule` is `opmodel.dev/core@v2.0.0-alpha.7`; `registrytest.DefaultCoreVersion` mirrors it; `modfile_test.go`, `stage_test.go`, `render_test.go` and `loader_test.go` carry the literal; `testdata/render/**` (twelve `cue.mod` files plus platform headers) pin it; the four modules `task cue:deps:update` discovers pin it in `cue.mod`. `closure_test.go` and `generate_test.go` are decoupled through their own `coreVersion` and need no edit.
- Nothing in `opm/` reads `#contracts`. `opm/platform.Platform` is `{Metadata, Package, Source}`; its doc says derived CUE views are not decoded into Go. `schema/paths.go` is the inventory of paths Go reads.
- The loader gate (`opm/internal/loader/shape.go`) walks `#registry` entries' regular fields only; `#contracts` is a hidden definition and is never visited, so alpha.9 meets no concreteness check at acquisition.
- The render glue (`render.cue.tmpl`) reads `platform.#composedTransformers` and, for the single-provider guard, `platform.#registry`; the unresolved rows carry `kind`, `fqn`, `alternatives`, `disqualified`; the kernel decodes `diagnostics` by JSON tag (`render_decode.go`) and derives the gate from the decoded rows.
- Core alpha.7 to alpha.9 is additive: `#Catalog` contract maps (alpha.8) and `#Platform.#contracts` (alpha.9). No shape the glue or the gate reads moved.

## Goals / Non-Goals

**Goals**

- One Go read of the inventory, on demand, off the artifact that already holds the built value.
- The unresolved-demand row names the defining catalog when one exists, and the refusal's wording says which of three cases applies. The gate is unchanged.
- The library's own fixtures exercise both against catalogs that list contracts, so nothing here is proven against empty maps.

**Non-Goals**

- Replacing the in-build single-provider guard with `#contracts.overSubscribed` (see Research & Decisions).
- Refusing platform-package generation on `routable: false` inside `platformmodule` (see Research & Decisions).
- Any rung change in the match glue.
- A CLI or operator surface: both are their own 0015 slices and read what this change exposes.

## Decisions

### The Go surface

`opm/schema/paths.go` gains the path; `opm/platform` gains the type and the accessor:

```go
// schema/paths.go
Contracts = cue.MakePath(cue.Def("contracts")) // #Platform.#contracts (0015 D1)

// platform/contracts.go
type ContractInventory struct {
	DefinedBy      map[string]string   // contract FQN -> registry key of the listing catalog
	RequiredBy     map[string][]string // contract FQN -> implementation FQNs requiring it
	Unfulfilled    []string
	OverSubscribed []string
	Fulfilled      bool
	Routable       bool
}

// Contracts decodes #contracts off Package. `defined` is not decoded: its
// values are the catalogs' member schemas, read off Package by a caller
// that wants one. Reports, never refusals: an over-subscribed platform
// returns Routable false and a nil error.
func (p *Platform) Contracts() (*ContractInventory, error)
```

`Contracts()` MUST read each decoded field by path from `Package.LookupPath(schema.Contracts)` and MUST return an error naming the missing field when the platform carries no `#contracts` (a value built against a core older than alpha.9). It MUST NOT be called by the loader gate, the render path or `NewPlatformFromValue`.

### The row

`oerrors.UnresolvedDemand` gains `DefinedBy string`. The glue's `unresolvedResources` and `unresolvedTraits` rows gain the field, filled from the inventory and defaulting to the empty string:

```cue
{
	kind:         "resource"
	fqn:          _f
	definedBy:    *"" | string
	if platform.#contracts.definedBy[_f] != _|_ {
		definedBy: platform.#contracts.definedBy[_f]
	}
	alternatives: (#alternatives & {universe: _bucketsResources, fqn: _f}).out
	disqualified: _resOutcome[_f].disqualified
}
```

`describe()` MUST word the three cases the spec names: defined by a catalog and unimplemented (catalog named, no alternatives); implemented at another apiVersion (alternatives); no enabled catalog defines it (neither). `gateErrors` and `glueDiagnostics` are unchanged: the row decodes by JSON tag and the gate keys on the row's presence, not its fields.

### Fixtures

`testdata/render/registry`'s `cat` and `cat2`, both builds, list every member they define in `#resources` and `#traits`, keyed by `metadata.fqn` in the `#transformers` idiom. The existing platforms then read as: `platform` (one catalog, its provider-fulfilled trait implemented or not per the fixture's own adapters), `platform_oversubscribed` (two catalogs supplying one provider-fulfilled key: `OverSubscribed` names it, `Routable` false, and the render still refuses through the unchanged guard), `platform_two`. Section 2's spike pins the exact expected values after listing, since the fixture catalogs' current fulfilment declarations decide them.

## Research & Decisions

### Where the inventory is read

**Context**: 0015 02-design.md reduces library's obligation to "reading it off a built value".
**Explored**: (a) decode into a `Platform.Contracts` field at construction; (b) a kernel verb `Kernel.PlatformContracts(p)`; (c) an on-demand accessor on `*Platform` reading `Package`.
**Decision**: (c).
**Rationale**: `Package` is the source of truth and the type's doc already refuses decoded derived views (a); the value is built and the kernel holds no context or state the read needs (ADR-007), so a verb adds a surface with nothing behind it (b). On demand also keeps the fold's cost off every acquire: the operator's readiness loop and `opm platform check` ask for it, a render does not.

### `defined` is not decoded

**Context**: `#contracts.defined` carries the member primitives whole, with non-concrete `spec` schemas.
**Explored**: decoding it into `map[string]any`; decoding the whole `#contracts` struct in one `Decode`; reading the six data fields by path and leaving `defined` on `Package`.
**Decision**: read the six data fields by path.
**Rationale**: a non-concrete schema has no Go value a caller could use, and one `Decode` over the struct would have to skip `defined` by convention rather than by type; per-path reads make the decoded set the type's field list, nothing more. `DefinedBy` already carries the key set of `defined`.

### The defining catalog rides the existing row

**Context**: 0010 OQ3 asks how a render says "defined by a subscribed catalog, unimplemented"; D18 says the refusal names the defining catalog beside the demanding component.
**Explored**: a new diagnostics table keyed by contract; membership in `#contracts.unfulfilled`; a `definedBy` field on the unresolved row read from `#contracts.definedBy`.
**Decision**: the field on the row.
**Rationale**: the row is what a frontend words (0019 D10: rows as data, no joins in Go), so the fact belongs on it; `unfulfilled` covers provider-fulfilled keys only, while a catalog-fulfilled key a catalog lists and no transformer implements is the same diagnostic; `definedBy` is the registry key core binds to the catalog's identity, never a prefix parsed off the FQN (0010 D17).

### The in-build guard stays, and is not switched to `#contracts.overSubscribed`

**Context**: the proposal for core's inventory said the render validation "can replace its own over-subscription count with `#contracts.overSubscribed`". The glue's `guard._providerSuppliers` is that count.
**Explored**: swapping the guard's source now; keeping the guard and adding the accessor beside it.
**Decision**: keep the guard unchanged.
**Rationale**: the two differ. The guard refuses when a *transformer* declares its required copy provider-fulfilled; the inventory refuses only when an enabled catalog *lists* the key. While every first-party catalog lists nothing (`catalog_opm` pins alpha.6, its listing change is planned), the inventory's report is vacuously empty and a swap would silently delete the shipped refusal. D18 also moves the over-subscription refusal to platform-package generation, so the right follow-up is not a swap inside the render but a decision, once the operator's generation refusal exists, whether the render-time guard is retired. Recorded as a follow-up change gated on `catalog_opm` publishing populated maps.

### Generation stays pure

**Context**: D18 says only over-subscription refuses platform-package generation; `platformmodule.Generate` writes files and never builds.
**Explored**: building the generated module inside `Generate` and refusing on `routable: false`; leaving `Generate` pure and letting the caller that already builds the module (the operator's generation step, its own 0015 slice) read `Contracts().Routable` and refuse.
**Decision**: leave `Generate` pure.
**Rationale**: a build inside `Generate` adds a registry fetch to a step whose contract is "pure files", duplicates the build the operator performs anyway, and would make the CLI's seeded-platform generation fetch a catalog to write a `cue.mod`. The refusal needs the accessor this change adds and nothing else from library.

### Pin bump commit shape

**Context**: CLAUDE.md types a shipped pin bump `fix(deps)` and a fixture pin bump `test(fixtures)`, never mixed in one commit; `registrytest.DefaultCoreVersion` is a test-only Go constant.
**Explored**: one commit; two commits inside section 1; two sections.
**Decision**: two commits inside section 1, shipped pin first (`loader.go`, `loader_test.go`, doc comments), then every fixture pin (`registrytest.go`, the CUE modules, `testdata/render`, the test literals).
**Rationale**: two sections would need the first to end green on its own, and a fixture-drift canary (the served fixtures declare the default release) is red between the two commits; one section closes green after the second commit and the PR squashes as `feat`.

### Section 1 is a spike

**Context**: design.md carries two unverified assumptions: that the render suite is green on alpha.9 with no glue change, and that `Contracts()` reads cleanly through `LookupPath` on the hidden definition.
**Decision**: section 1 lands the pin and re-runs `task check`; section 2 opens with the read measured on the `platform` fixture before the accessor is written.

## Risks / Trade-offs

- [A consumer asserts the old unresolved-demand sentence verbatim] → the cli's render tests may; named in the proposal's downstream section so the Dependabot bump re-pins the sentence.
- [`#contracts` cost on the operator's readiness loop] → the fold is roughly ten thousand string compares at `catalog_opm`'s size (core proposal, measured); `Contracts()` is on demand, so a render pays nothing.
- [A platform built against core before alpha.9] → `Contracts()` returns an error naming the missing field; nothing else in library changes behaviour on such a value.
- [Fixture listing changes what `platform_oversubscribed` and `platform` report] → intended; section 2's spike pins the values before the accessor tests are written, and the render assertions those fixtures already carry are unchanged because the guard is unchanged.
- [The served fixtures' private caches] → per-test private caches extract served coordinates fresh, so an edited `cue.mod` under a fixed fixture version is picked up (`test-fixture-registry`).

## Migration Plan

One PR, three sections, squash title `feat(platform): read the contract inventory off a built platform`. release-please cuts `v1.0.0-alpha.30`. Rollback before release is a revert; after, a later release. `cli` and `opm-operator` pick the release up through Dependabot; neither reads `Contracts()` until its own 0015 slice.
