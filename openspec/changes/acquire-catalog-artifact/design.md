# Design: acquire-catalog-artifact

## Context

See `proposal.md` for motivation. Current state, read 2026-09-15 at alpha.30:

- `opm/kernel/acquire.go` holds four verbs: `AcquireModuleFromRegistry`, `AcquireModuleFromDir`, `AcquirePlatformFromDir`, `AcquireInstanceFromDir`. Each calls `cuecontext.New()` at entry (ADR-007), loads, shape-gates, and constructs a typed artifact.
- `opm/internal/loader/shape.go` defines `ArtifactSpec` and three instances — `ModuleSpec`, `InstanceSpec`, `PlatformSpec` — each naming the expected `kind` and the required metadata fields. One routine gates all three and wraps `ErrWrongKind`, `ErrInvalidPackage`, `ErrMissingRequiredField`.
- `opm/internal/loader/registry.go`'s `FetchModule` resolves `path@version` through CUE's native module machinery and stages the artifact's `.cue` files as an overlay. It is **kind-agnostic**: it fetches a CUE module artifact and returns a value, with no assumption that the value is a `#Module`.
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
2. **The need is proven twice, which is the test `CONSTITUTION.md` line 138 sets.** The operator cannot implement D8, D10 or D11 without it, and `cli/internal/publish` already hand-rolled the operation (`compat.go:212-223`).
3. **The alternative reverses a completed migration.** The operator reaching for `cue/load.Instances` in `internal/` puts CUE loading back in the controller, which the kernel migration removed, and makes it the third copy of OCI-fetch-and-evaluate.
4. **The boundary that actually matters is read versus judge, and it is preserved.** The kernel returns a validated catalog and a derived set; every verdict stays with the consumer. That is the same line `Platform.Contracts()` draws — "Reports, never refusals".

**Alternative considered — a narrower answer: the operator asks the kernel to validate a claim.** A verb such as `VerifyRegistration(claim) → verdict` keeps the catalog type out of the public surface. Rejected: it moves 0015's *policy* into the library, which is the boundary this design is trying to hold. The library would then own what a refusal says, and D8, D11 and D12's diagnostics are explicitly left to the operator slice.

**Alternative considered — leave it in the frontends.** Honest and cheapest today; rejected on the duplication the slice-07 program (five changes, 2026-09-13) just finished removing between `cli` and `opm-operator`.

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

No new gating routine. The registry path calls the existing `FetchModule` unchanged, because it already fetches a CUE module artifact without assuming its kind — the one place where a catalog and a module are genuinely the same thing.

## Research & Decisions

### The registry plumbing needs no change

**Context**: whether fetching a catalog artifact needs a second fetch path.
**Explored**: `opm/internal/loader/registry.go:53` — `FetchModule(ctx, cueCtx, modPath, version, env) (cue.Value, *Source, error)`.
**Decision**: reuse it as-is; add only the shape gate and the typed constructor.
**Rationale**: the function resolves `path@version` and stages the artifact's CUE files. Nothing in it reads `kind`; the kind assumption lives entirely in the caller's choice of `ArtifactSpec`. A catalog published by `opm catalog publish` is an ordinary CUE module artifact, so the fetch is byte-identical work. The name `FetchModule` becomes slightly misleading and is worth a doc note, not a rename this change pays for.

### `Platform.Contracts()` cannot answer this

**Context**: alpha.30 already decodes a contract inventory; is a new verb needed at all?
**Explored**: `opm/platform/contracts.go` and its doc comment.
**Decision**: it is not a substitute; the new verb stands.
**Rationale**: `#Platform.#contracts` is derived by core from the **enabled registry entries' contract maps** — the catalogs the platform already subscribes to. A registration claims a catalog that is by construction not subscribed; if it were, the static path would already cover it and D3 would have nothing to do. The two answer different questions about different catalogs.

## Risks / Trade-offs

- [A fourth acquired kind invites a fifth] -> ADR-009 states the test that admitted this one (a core kind the kernel already evaluates transitively, with two proven consumers, where the library reads and the caller judges). A fifth has to pass the same test in writing.
- [`Provides` derivation drifts from what core means by `fulfilment: "provider"`] -> the derivation is a fold over values core defines, and `catalog_opm` 4.1.0 populated the contract maps it reads. A core change to the meaning breaks the fold loudly rather than silently, because the field's absence is distinguishable from `"provider"`.
- [`Requires` exposes `cue.mod` shape to consumers] -> it returns path-to-version, not a parsed module file, so the consumer never sees CUE's module types. D8's comparison is version arithmetic on strings within a major.
- [The change lands before its only consumer] -> it ships as an unused MINOR on the alpha line, and `registration-acceptance` picks it up in the same enhancement. The alternative, developing both repos against an unpublished pin, is worse.

## Migration Plan

Additive; no consumer migration. Ships as `v1.0.0-alpha.31`, then `opm-operator` bumps its pin and builds `registration-acceptance` on it. Rollback before release is a revert; after release, an unused additive symbol costs nothing to leave in place.

## Open Questions

None.
