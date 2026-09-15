# Design: acquire-catalog-artifact

## Context

See `proposal.md` for motivation. Current state, read 2026-09-15 at alpha.30:

- `opm/kernel/acquire.go` holds four verbs: `AcquireModuleFromRegistry`, `AcquireModuleFromDir`, `AcquirePlatformFromDir`, `AcquireInstanceFromDir`. Each calls `cuecontext.New()` at entry (ADR-007), loads, shape-gates, and constructs a typed artifact.
- `opm/internal/loader/shape.go` defines `ArtifactSpec` and three instances — `ModuleSpec`, `InstanceSpec`, `PlatformSpec` — each naming the expected `kind` and the required metadata fields. One routine gates all three and wraps `ErrWrongKind`, `ErrInvalidPackage`, `ErrMissingRequiredField`.
- `opm/internal/loader/registry.go`'s `FetchModule` resolves `path@version` through CUE's native module machinery and stages the artifact's `.cue` files as an overlay. Its fetch and staging are **kind-agnostic** — any published CUE module artifact — but its build is not: it gates through `ModuleSpec` and then verifies the declared coordinate. See § The registry plumbing is kind-agnostic only up to the build.
- `opm/module/module.go`'s `Module` is the shape a typed artifact takes: `Metadata`, `Package cue.Value`, `Source`.
- `opm/platform/contracts.go` (alpha.30) decodes `#Platform.#contracts` on demand. It is the precedent for a core-defined derived view living in the library rather than in a consumer.
- There is no catalog type, no catalog verb, and no `CatalogSpec`.

## Goals / Non-Goals

**Goals**

- A catalog is acquirable by the same two routes a module is, with the same guarantees and the same sentinels.
- A non-catalog artifact is refused by shape, so 0015 D10's restriction costs no maintained rule.
- The provider-fulfilled set and the committed requirements are readable off the artifact, so the operator needs no CUE of its own.

**Non-Goals**

- Any verdict. Comparison, refusal and diagnostics are `registration-acceptance`'s.
- Migrating `cli/internal/publish` onto the new verb.
- A catalog *render* path. Catalogs are read here, never executed; transformers run through `Kernel.Render` as they already do.

## Decisions

### The kernel grows a fourth acquired kind, and ADR-009 records why

This is the load-bearing decision and the one a reviewer should attack. The kernel's accepted-kinds triple (`Module`, `ModuleInstance`, `Platform`) is a stated boundary, and ADR-007 plus the kernel-diet slices spent eight changes shrinking this surface. Adding to it needs an argument, not a convenience.

The argument, recorded in `adr/009-catalog-is-an-acquired-kind.md`:

1. **A catalog is already a kernel input, transitively.** Every platform build resolves and evaluates the subscribed catalogs; the kernel has always read catalogs, just never as an entry point. This makes an existing dependency explicit rather than introducing one.
2. **The need is proven by one consumer that cannot proceed without it**, which is the test `CONSTITUTION.md` line 138 sets — the operator implements D8, D10 and D11 or 0015's acceptance path does not ship. `cli/internal/publish` is deliberately NOT cited as a second consumer: its `loadPublishedPackage` probes published subpackages for member-shape compat, gating to no kind and treating absence as a scan signal, and its directory peer carries source positions for refusal messages. Both would fail this change's kind gate. One honest consumer is the argument; two inflated ones would be a worse one.
3. **The alternative reverses a completed migration.** The operator reaching for `cue/load.Instances` in `internal/` puts CUE loading back in the controller, which the kernel migration removed, and makes it the third copy of OCI-fetch-and-evaluate.
4. **The boundary that actually matters is read versus judge, and it is preserved.** The kernel returns a validated catalog and a derived set; every verdict stays with the consumer. That is the same line `Platform.Contracts()` draws — "Reports, never refusals".

**Alternative considered — a narrower answer: the operator asks the kernel to validate a claim.** A verb such as `VerifyRegistration(claim) → verdict` keeps the catalog type out of the public surface. Rejected: it moves 0015's *policy* into the library, which is the boundary this design is trying to hold. The library would then own what a refusal says, and D8, D11 and D12's diagnostics are explicitly left to the operator slice.

**Alternative considered — leave it in the frontends.** Honest and cheapest today. Rejected because the only frontend that would hold it is the operator, which has no CUE machinery left after the kernel migration; putting `load.Instances` back in `internal/` to serve one controller path is what slice 07 (five changes, 2026-09-13) spent its effort undoing. Note this is a weaker rejection than it would be with two consumers — with one, it turns entirely on where CUE loading belongs, not on deduplication.

### `opm/catalog` mirrors `opm/module`, and derivation is a method

```go
// opm/catalog
type Catalog struct {
	Metadata *CatalogMetadata
	Package  cue.Value
	Source   *module.Source
}

func NewCatalogFromValue(v cue.Value) (*Catalog, error)

// Provides returns the provider-fulfilled contract FQNs this catalog
// implements, deterministically ordered. A report, never a refusal.
func (c *Catalog) Provides() ([]string, error)

// Requires returns the module requirements committed in the catalog's own
// cue.mod/module.cue, as path -> version.
func (c *Catalog) Requires() (map[string]string, error)
```

`Provides` and `Requires` are methods on the artifact rather than kernel verbs, following `Platform.Contracts()`: the value is already built, and a caller that never asks pays nothing. Both decode on demand and neither is called at construction.

**Ordering is part of the contract, not an implementation detail.** D11 compares the derived set against a claimed list for exact equality, and a set whose order varies forces every caller to sort before comparing — the kind of thing one caller forgets. Sorting once, here, is why the spec states it as a requirement.

### `CatalogSpec` joins the three existing shape gates

```go
CatalogSpec = ArtifactSpec{ Kind: "Catalog", /* required metadata fields */ }
```

No new gating routine. The registry path goes through the loader's one fetch routine with `CatalogSpec` in place of `ModuleSpec` — the fetch and the staging are the one place where a catalog and a module are genuinely the same thing, and the spec is the only thing that differs.

## Research & Decisions

### The registry plumbing is kind-agnostic only up to the build, and is parameterized

**Context**: whether fetching a catalog artifact needs a second fetch path.
**Explored**: `opm/internal/loader/registry.go` — `FetchModule(ctx, cueCtx, modPath, version, env) (cue.Value, *Source, error)`.
**Decision**: split the routine — `FetchArtifact(..., spec ArtifactSpec)` carries the body, `FetchModule` becomes a one-line wrapper passing `ModuleSpec`. No call site and no module behaviour changes. Do NOT write a second fetch path in the kernel.
**Rationale**: an earlier reading of this function (recorded here, now withdrawn) called it kind-agnostic. Only its first half is. The fetch and the staging are indeed byte-identical work for any published CUE module artifact — `modconfig` resolution, `reg.Fetch`, `OverlayFromFS` under a synthetic root — but the build is not: it calls `LoadDir(..., ModuleSpec)` and then `verifyModuleIdentity`. A catalog fetched through it fails the kind gate and the `metadata.name` requirement, which a `#Catalog` does not carry. Parameterizing the spec keeps the one fetch implementation the whole design rests on; the alternative (a second fetch in the kernel) is the duplication ADR-009 argues against.

**The coordinate identity check stays module-only.** `verifyModuleIdentity` would work on a catalog — a `#Catalog` declares `metadata.modulePath` major-qualified and `metadata.version` bare, exactly as a `#Module` does — but this change's delta spec names only the kind refusal, and a refusal the spec does not describe is out of scope. `FetchModule` keeps the check; `FetchArtifact` does not run it. Generalizing it is its own change.

### `cli/internal/publish` is not a consumer of this verb

**Context**: the first draft of this design cited `cli` as a second proven consumer and listed `compat.go` as a later collapse candidate. Both claims needed checking before an implementer acted on them.
**Explored**: `cli/internal/publish/compat.go` `loadPublishedPackage` and `load.go` `loadArtifact`/`loadPackage`.
**Decision**: `cli` is not a consumer, now or later. The claims are withdrawn from the proposal and from ADR-009.
**Rationale**: `loadPublishedPackage` loads ONE published subpackage by import path (`.../resources/v1beta1@4.2.0`) to compare member shapes level-aware. It gates to no kind — a resources subpackage has none — and returns `found=false` on module-not-found, because absence is the scan's negative signal rather than a failure. `loadArtifact` carries `packageClausePos`, `fieldPos` and `moduleLinePos` so publish refusals can point at source. This verb fetches a root package, requires `kind: "Catalog"`, errors on absence, and returns no positions. Every compat call would fail its gate. The shared `load.Instances` call is the only resemblance.

### `Platform.Contracts()` cannot answer this

**Context**: alpha.30 already decodes a contract inventory; is a new verb needed at all?
**Explored**: `opm/platform/contracts.go` and its doc comment.
**Decision**: it is not a substitute; the new verb stands.
**Rationale**: `#Platform.#contracts` is derived by core from the **enabled registry entries' contract maps** — the catalogs the platform already subscribes to. A registration claims a catalog that is by construction not subscribed; if it were, the static path would already cover it and D3 would have nothing to do. The two answer different questions about different catalogs.

## Risks / Trade-offs

- [A fourth acquired kind invites a fifth] -> ADR-009 states the test that admitted this one (a core kind the kernel already evaluates transitively, with at least one proven consumer that cannot ship without it, whose alternative is re-acquiring CUE machinery a migration removed, where the library reads and the caller judges). A fifth has to pass the same test in writing.
- [`Provides` derivation drifts from what core means by `fulfilment: "provider"`] -> the derivation is a fold over values core defines, and `catalog_opm` 4.1.0 populated the contract maps it reads. A core change to the meaning breaks the fold loudly rather than silently, because the field's absence is distinguishable from `"provider"`.
- [`Requires` exposes `cue.mod` shape to consumers] -> it returns path-to-version, not a parsed module file, so the consumer never sees CUE's module types. D8's comparison is version arithmetic on strings within a major.
- [The change lands before its only consumer] -> it ships as an unused MINOR on the alpha line, and `registration-acceptance` picks it up in the same enhancement. The alternative, developing both repos against an unpublished pin, is worse.

## Migration Plan

Additive; no consumer migration. Ships as `v1.0.0-alpha.31`, then `opm-operator` bumps its pin and builds `registration-acceptance` on it. Rollback before release is a revert; after release, an unused additive symbol costs nothing to leave in place.

## Open Questions

None.
