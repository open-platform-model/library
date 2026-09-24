# ADR-009: A catalog is an acquired kind

## Status

Accepted (2026-09-15). Implemented by `acquire-catalog-artifact`.

## Context

The kernel accepts three artifact kinds and returns a typed Go value for each: `#Module` (`*module.Module`), `#ModuleInstance` (`*module.Instance`) and `#Platform` (`*platform.Platform`). That triple is a stated boundary, repeated in `AGENTS.md` and pinned by `opm/kernel/kernel_test.go`, and the surface around it has been shrinking rather than growing: ADR-007 removed the Kernel's shared `cue.Context` and its accessor, and eight simplification slices through 2026-09-14 deleted verbs, a raw value tier and three free-function entry points. A fourth kind runs against that direction and needs an argument rather than a convenience.

The pressure comes from a dynamic transformer path. A provider — a module shipping its own controller and its own rendering logic — installs a cluster-scoped claim naming a catalog it says implements a set of platform contracts, and asks a running platform to accept it. Accepting the claim means answering three questions about the *claimed catalog*, not about the platform:

- Is the artifact at that coordinate a catalog at all, or a module wearing the name?
- Does the catalog actually implement the contracts the claim lists — every contract required by one of its own transformers whose value carries `fulfilment: "provider"` — with no contract more and none fewer?
- What does the catalog's own `cue.mod/module.cue` commit to, so the acceptor can compare it against what the platform already resolved?

Each question is a read of a fetched, evaluated `#Catalog` value. The kernel can do none of it today: there is no catalog verb, no catalog type and no catalog entry in the loader's shape gate.

`Platform.Contracts()` is not a substitute, and the reason is structural rather than incidental. `#Platform.#contracts` is derived by core from the **enabled registry entries'** contract maps — the catalogs a platform already subscribes to. A claimed catalog is by construction not subscribed; if it were, the static path would already cover it and the dynamic path would have nothing to do. The two views answer different questions about different catalogs.

`CONSTITUTION.md` (Principle VII, closing line) sets the test this decision has to pass: "The library should grow when downstream needs prove it, not in anticipation."

## Decision

The kernel grows a fourth acquired kind. `opm/catalog` carries a `Catalog` artifact mirroring `module.Module` — `Metadata`, `Package`, `Source` — with `Provides()` and `Requires()` as on-demand derivations, and `opm/kernel` gains `AcquireCatalogFromRegistry` and `AcquireCatalogFromDir`, the registry and directory peers `AcquireModuleFrom*` already establishes. The loader gains a `CatalogSpec` beside `ModuleSpec`, `InstanceSpec` and `PlatformSpec`, and no new gating routine.

Four facts carry the decision.

**A catalog is already a kernel input, transitively.** Every platform build resolves and evaluates the catalogs the platform subscribes to; the render glue reads `#composedTransformers`, which is a fold over them. The kernel has always read catalogs — it has just never had one as an entry point. Naming the kind makes an existing dependency explicit rather than introducing one.

**The need is proven by one consumer that cannot proceed without it.** The controller implements the three reads above or the acceptance path does not ship. One honest consumer is what Principle VII asks for; it does not ask for two. This is deliberately *not* padded with a second: see the Consequences on `cli/internal/publish`.

**The alternative reverses a completed migration.** The consumer's own route is `cue/load.Instances` inside its `internal/` packages — CUE loading back in a controller that has none left after the kernel migration, and a third copy of fetch-an-OCI-module-and-evaluate-it. Slice 07 (five changes, 2026-09-13) spent its effort deleting exactly that.

**The boundary that actually matters is read versus judge, and it survives.** Acquisition returns a validated catalog and a derived set. Whether a derived set matches a claim, whether a competing claim exists, whether a resolved version is compatible, and what any refusal says all stay with the caller. That is the line `Platform.Contracts()` already draws — reports, never refusals — and this kind is admitted on the same terms.

Two alternatives were considered and rejected.

**A narrower verb: the kernel validates a claim and returns a verdict.** `VerifyRegistration(claim) → verdict` keeps the catalog type off the public surface entirely. Rejected because it moves policy into the library: the library would then own what a refusal says, which is precisely what the fourth argument above preserves for the caller. A smaller type surface bought with a larger policy surface is the wrong trade for a kernel.

**Leave it in the frontend.** The cheapest option today and an honest one. Rejected on where CUE loading belongs, not on deduplication — with a single consumer there is nothing to deduplicate. The rejection is weaker than it would be with two consumers, and this ADR says so rather than inflating the count.

**The test a fifth kind must pass, in writing, before it is admitted.** All four of:

1. It is a kind `core` defines, which the kernel already resolves and evaluates transitively as part of some existing build — so acquiring it names an existing dependency instead of adding one.
2. At least one consumer cannot ship a committed capability without it, named concretely, not a capability someone might build.
3. The alternative for that consumer is re-acquiring CUE machinery a completed migration removed from it.
4. The kernel reads and derives; every verdict about the value stays with the caller.

A candidate meeting three of the four does not qualify. If a future kind needs the bar relaxed, that is a new ADR amending this one, not a judgement call at the call site.

## Consequences

**Positive:** The consumer needs no CUE of its own. Restricting the dynamic path to actual catalogs costs no maintained rule — a module artifact is refused by the same shape gate that refuses a platform claimed as a module, wrapping the same `ErrWrongKind` sentinel. The provider-fulfilled set is derived once, in one place, in deterministic order, so two callers cannot disagree about what a catalog implements or about how to compare two derivations. The registry path needs no new plumbing: fetching and staging a published CUE module artifact reads no `kind`, so a catalog fetch is byte-identical work to a module fetch and one routine serves both.

**Negative:** The public SemVer surface grows by a package, a type, two methods and two kernel verbs, against a program that spent eight changes shrinking it, and the "three kinds and nothing else" line in `AGENTS.md` is no longer literally true. The change also lands before its consumer, as an additive MINOR nothing in this repo calls; the alternative, developing both repos against an unpublished pin, is worse.

**Trade-off — `loader.FetchModule` is now kind-agnostic in fact but not in spelling.** Fetching a published catalog is byte-identical work to fetching a published module: `modconfig` resolution of `path@version`, `reg.Fetch`, and the fetched `.cue` files staged as an overlay under a synthetic root. That is the one place a catalog and a module are genuinely the same thing, and it is why no second fetch path exists. Only the half after the staging was module-specific — the build gated through `ModuleSpec` and then ran the coordinate identity check — so the routine is parameterized rather than reused verbatim: `FetchArtifact` takes the `ArtifactSpec` and `FetchModule` becomes the one-line wrapper that passes `ModuleSpec` and keeps `verifyModuleIdentity`. Its signature, its behaviour and its one call site are unchanged. What changes is that the name now names only the first caller: read `FetchModule` as "fetch a published `#Module`", and `FetchArtifact` as the kind-parameterized routine underneath it.

The coordinate identity check deliberately does not follow. A `#Catalog` declares `metadata.modulePath` major-qualified and `metadata.version` bare exactly as a `#Module` does, so `verifyModuleIdentity` would work on one — but the requirement admitted here names only the kind refusal, and a kernel that refuses on something no requirement describes is a worse thing than an unchecked coordinate. Generalizing it is its own change, with its own scenario.

**Trade-off — `cli/internal/publish` is NOT a consumer of these verbs, and its loading is not a collapse candidate.** The resemblance is real enough to mislead a later reader, so it is recorded here rather than rediscovered. `internal/publish/compat.go`'s `loadPublishedPackage` also fetches through `cue/load.Instances` with a registry environment, but it is a different operation in four respects. It loads one published **subpackage** by import path (`.../resources/v1beta1@4.2.0`) to compare member shapes level-aware, where these verbs load a root package. It **gates to no kind at all** — a resources subpackage carries none — where these verbs require a concrete `kind: "Catalog"` and refuse everything else. It treats module-not-found as a **normal scan signal** (`found=false`), where these verbs return an error. And its directory peer, `load.go`'s `loadArtifact`, carries `packageClausePos`, `fieldPos` and `moduleLinePos` so publish refusals can point at source, where these verbs return no positions. Every compat call would fail this change's kind gate, and none of them wants what it returns. The shared `load.Instances` call is the whole of the similarity. A migration built on the resemblance cannot work; do not write one.
