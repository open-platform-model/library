## Why

Enhancement 0015 D3 gives transformers a second path onto a platform: a provider module ships a cluster-scoped `TransformerRegistration` claim naming a catalog it says implements platform contracts. `opm-operator` landed the CRD and its RBAC gate (PR 133); the acceptance path is the next slice, and three of its checks all need the same thing — **the claimed catalog artifact, fetched from the registry and evaluated**:

- **D10**: import the claimed artifact's root package and require a `#Catalog` value, so a module artifact is refused by shape rather than by a maintained rule.
- **D11**: re-derive `provides` as a fold over the catalog's own `#transformers` — every required contract whose value carries `fulfilment: "provider"` — and compare it to the claim for exact equality.
- **D8**: read the catalog's `cue.mod` requirements and compare them against the platform's resolved versions, per 0019 D18's committed-resolution discipline.

The kernel cannot do any of it. `opm/kernel` acquires `Module`, `Platform` and `Instance`; there is no catalog verb and no catalog type. `Platform.Contracts()` (alpha.30) is not a substitute: it decodes the inventory core derives from the platform's **enabled registry entries**, and a claimed catalog is by definition not subscribed yet — that is the whole point of the dynamic path.

`cli` is **not** a second consumer, though it looks like one. `internal/publish/compat.go`'s `loadPublishedPackage` also fetches through `cue/load.Instances` with a registry env, but it loads one published **subpackage** (`.../resources/v1beta1@4.2.0`) to compare member shapes level-aware, treats absence as a normal scan signal rather than an error, and gates to no kind at all — a resources subpackage carries none. Its directory peer builds an artifact carrying package-clause and field positions for publish refusal messages. Both would fail the kind gate this change adds, and neither wants what it returns. The resemblance is `load.Instances`, not the operation.

## What Changes

- `opm/catalog` (new package): a `Catalog` type — the typed, source-carrying artifact, mirroring `module.Module` — plus `NewCatalogFromValue`, and the `provides` derivation as a method.
- `opm/kernel`: `Kernel.AcquireCatalogFromRegistry(ctx, modPath, version)` and `Kernel.AcquireCatalogFromDir(ctx, dirPath)`, the registry and directory peers `AcquireModuleFrom*` already establishes.
- `opm/internal/loader`: a `CatalogSpec` shape gate beside `ModuleSpec`, `InstanceSpec` and `PlatformSpec`. `FetchModule` is already kind-agnostic — a catalog is a CUE module artifact — so the registry path needs no new plumbing.
- `adr/009`: why the kernel grows a fourth acquired kind after a program that deliberately shrank its surface.

**Not in this change**: every verdict. Whether a claim's `provides` matches, whether a competing claim exists, whether a build is compatible, and what any refusal says are the operator's, in `registration-acceptance`. This change hands back a validated catalog and its derived provider set; it judges nothing.

Also not in this change, and not a follow-up either: `cli/internal/publish`. Its loading is a different operation (see Why), so there is nothing there to collapse onto this verb.

## Capabilities

### New Capabilities

- `catalog-acquisition`: fetching a published catalog artifact or loading one from a directory, gating it to `#Catalog`, and deriving the provider-fulfilled contract set it implements.

### Modified Capabilities

None. The existing acquisition capabilities describe their own kinds; this adds a fourth beside them rather than restating them.

## Impact

- **`opm/` packages**: one new package (`opm/catalog`), two new `Kernel` methods, one new unexported loader spec. No existing symbol changes signature or behavior.
- **`opm-operator`**: the consumer this exists for. Picks up the pin and builds `registration-acceptance` on it.
- **`cli`**: nothing required. `internal/publish` keeps its own loading; the new verb is a later simplification candidate, not a migration this change forces.
- **SemVer: MINOR.** Additive public surface, no break. The library is pre-1.0 on the alpha line, so this lands as `feat` and cuts `v1.0.0-alpha.31`.

**Complexity justification (Principle VII).** `CONSTITUTION.md` line 138 is the governing test — the library grows "when downstream needs prove it, not in anticipation". The need is proven once, by one consumer that cannot proceed without it: the operator implements D8, D10 and D11 or 0015's acceptance path does not ship. One proven consumer is what line 138 asks for; it does not ask for two. The alternative is the operator calling `cue/load.Instances` in `internal/`, which reverses the kernel migration specifically so the controller can do CUE work again. A catalog is also already a kernel input transitively: every platform build resolves and evaluates the subscribed catalogs. This makes an existing implicit dependency explicit.

The countervailing cost is real and named in ADR-009: this is a fourth kind on a surface that ADR-007 and the kernel-diet slices deliberately shrank, and the `Module`/`ModuleInstance`/`Platform` triple is a stated boundary. The ADR is the gate — if its argument cannot be written convincingly, the boundary should win and the operator should get a narrower answer instead.
