# Design — library-acquire-and-subscription

## Overview

The materialize resolution block collapses from enumerate→filter→select→pull-each-survivor to read-scalar→check-major→pull-one. Identity verification lands at the two places the library holds both a fetched coordinate and decoded metadata. The synth write side moves in lockstep because it is the only in-repo producer of the subscription shape the kernel is about to start reading.

## Research & Decisions

### The resolution collapse

**Context**: `materialize.go:63-96` today: `decodeFilter` (always-empty on v2) → `enumerateVersions` → `filterVersions` (falls through to `highestStable`) → pull each survivor → `resolved[sub] = last survivor`.
**Decision**: Replace with:

```go
ver, err := subscriptionVersion(subVal)          // reads version!, error if absent/non-concrete
if err := majorAgrees(sub, ver); err != nil ...  // key's @vN suffix vs SemVer major of ver
cv, err := pullCatalog(octx, env, sub, "v"+ver)
if err != nil { /* lazy diagnostic: enumerateVersions for the error, see below */ }
verifyCatalogIdentity(cv, sub, ver)              // D11+D9, see below
builds = append(builds, catalogBuild{sub, ver, cv})
resolved[sub] = ver
```

`pullCatalog` already splits the `@vN` suffix and accepts a `v`-prefixed version — unchanged. The major-agreement check is `strings.SplitN(ver, ".", 2)[0]` vs the key suffix's digits, mirroring the target schema's `_majorAgrees` (`0010/schemas/target.cue:670-691`); a key with no `@vN` suffix is an error (core v2's `#ModulePathType` requires it; defensive, not a supported path).
**Rationale**: D14 verbatim. The check is "a consistency check on what the author wrote rather than a selection step" — it lives beside the subscription read, not in a file of its own.

### Enumeration's fate: lazy diagnostic

**Context**: after D14, `enumerateVersions` has no selection role. The target schema keeps `published` as a diagnostic input only ("a read-side gate uses it to report a named build that does not exist").
**Options considered**:
1. Delete; let `pullCatalog`'s load error speak. Cheapest, but the load error for a missing version is opaque and D14 explicitly reserves the better diagnostic.
2. Pre-flight existence check on every subscription. One extra registry round-trip per subscription on every materialize, paid on the happy path.
3. Enumerate only when the pull fails, to enrich the error with the published list.
**Decision**: Option 3. `enumerate.go` survives unexported with its doc rewritten (diagnostic-only role; the major-scoping behavior and its locking test stay — the published list shown to the user is scoped to the key's major).
**Rationale**: The diagnostic exists exactly when someone needs it, and the happy path performs one registry operation per subscription (the pull), which is the D14 floor.

### The typed identity error

**Context**: nothing identity-shaped exists in `opm/errors`. The two read sites live in packages that cannot share `MaterializeError` (the acquire site is not materialization; `errors` must stay dependency-free).
**Options considered**:
1. Two site-local error types with a shared predicate — two shapes for one rule.
2. A new `MaterializeError` Kind for the catalog site + something else for acquire — same objection, and routes one rule through two discriminators.
3. One type in `oerrors`, emitted by both sites.
**Decision**: Option 3:

```go
// opm/errors/identity.go
type IdentityError struct {
    Artifact string // "module" | "catalog"
    Field    string // "path" | "version"
    Declared string // what metadata says
    Fetched  string // the coordinate/tag it was fetched by
    Coordinate string // full fetched coordinate for context, e.g. "opmodel.dev/catalogs/opm@v2 v2.0.0-alpha.3"
}
```

Value receiver `Error()` naming both values (D11: "a typed error naming both"), no `Cause` (the condition is a comparison, not a wrapped failure). At the catalog site it is wrapped in `*MaterializeError` (`Kind: "catalog"`) as `Cause`, preserving the package's fail-fast contract and keeping `errors.As` reachability for frontends that route on `IdentityError`. The acquire site returns it bare.
**Rationale**: One rule, one type, two emitters — the strict reading of D11's "one implementation" applied to the part that is shareable (the comparison + the type), while each site keeps its package's error idiom.

### What each site compares

**Decision**: Both clauses at both sites, per `#FetchedArtifact`'s artifact-agnostic shape:
- *Module acquire* (`loader/registry/module.go`, after the shape gate at `:138`): declared `metadata.modulePath` == requested `modPath` (both full `@vN` paths, string equality); declared `metadata.version` == `strings.TrimPrefix(version, "v")`. The shape gate already guarantees both fields present and concrete (`ModuleSpec.RequiredConcreteFields`), so the check cannot misfire on absence.
- *Catalog materialize*: `metadata.modulePath` == subscription key (both `@vN`-suffixed — direct comparison, no recomposition; this is the operation `#ArtifactRef` exists to make one-field); `metadata.version` == the version just pulled. Partially redundant with the major-agreement check by design — that check validates the *author's* consistency before any I/O, this one validates the *artifact's* honesty after.
**Rationale**: D9 binds the version clause to "each read point D11 names". Skipping the version at the catalog site would leave the jellyfin defect class (stale `metadata.version` in a published artifact) undetected exactly where catalogs are consumed.

**Amendment (found at implementation)**: strict full-path equality at the acquire site collides with the untouched `registry-module-loading` scenario "Self-referential core@v0 metadata is preserved" and the pinned core-v1 regression fixtures (library#31): a core-v0/v1 module's schema cannot express the major-suffixed form — its `metadata.modulePath` is the major-free *parent* path the module is published under (`modulePath + "/" + snake_case(name)`, the enhancements/0003 convention). The path clause therefore dispatches on the declared shape: a declaration carrying `@` compares strictly (the designed v2 rule); a major-free declaration is verified against the publishing convention (fetched path must sit directly under the declared parent). Both shapes still refuse lying metadata; the version clause is line-independent and applies unchanged.

### The third read point collapses into materialize (recorded deviation)

**Context**: D11 names "platform subscription" as the earliest-firing read point so the platform author sees a broken catalog first. The library has no such site: the platform loader deliberately does not resolve subscriptions (`shape.PlatformSpec`'s documented boundary), and `PlatformMetadata` has no registry projection.
**Options considered**:
1. New exported `materialize.VerifySubscriptions(p)` — a second registry-touching entry point whose only caller would be frontends that could equally call `Materialize`.
2. Fold subscription verification into the platform loader — violates the loader's documented contract and puts registry I/O where file loading lives.
3. The materialize-time check *is* the library's subscription check; frontends deliver author-time firing by materializing at subscription time.
**Decision**: Option 3, recorded as a deviation in 0010 history on landing.
**Rationale**: The check's substance (fires, names the catalog, types the error) lands; *when* it fires is frontend workflow. The CLI's platform-edit path and the operator's Platform reconcile both already own a natural materialize call. Option 1 remains available later without breaking anything — deferral is free, per Principle VII.

### Synth moves in lockstep

**Context**: `synth.SubscriptionSpec{Enable, Filter}` has no `Version`; `writeRegistry`/`writeFilter` emit `filter:` blocks; `synth.Platform` does not validate concretely, so a versionless synthesized platform passes synth today and would materialize nothing once the kernel reads the scalar. `kernel/synth_platform_test.go` synthesizes empty specs and relies on the float.
**Decision**: `SubscriptionSpec` becomes `{Enable *bool; Version string}` with `Version` required — `Platform()` returns a validation error naming the subscription path when it is empty. `FilterSpec` and `writeFilter` are deleted. The transitional test `TestPlatform_SubscriptionFilterRejectedByV2Schema` is replaced by its post-seam counterpart (a filter field no longer *exists* to reject at the Go boundary; the CUE-schema rejection it pinned stays covered by the schema itself).
**Rationale**: Early validation (Constitution II) at the write side beats a silent empty materialize at the read side. The coupling is structural — this change makes the scalar load-bearing, so the only in-repo producer must produce it.

### D41 residue: the instance FQN in the transformer context

**Context**: `schema/context.go:59` fills `#moduleInstanceMetadata.fqn` with `inst.ModuleFQN()` — the module's FQN under an instance-shaped name. Core v2 defines the instance's own `metadata.fqn` (`registryPath:name:namespace`) and `uuid`. Measured free: no shipped transformer reads `.fqn` from that block. 02-design left `Version` "removed or repurposed".
**Decision**: `InstanceMetadata` gains `FQN` (decoded via the existing `schema.MetadataFQN` path from the instance value); `context.go` fills the block's `fqn` from it. `Version` **stays**, fed by `inst.ModuleVersion()` — post-this-change that value is acquire-verified, which *is* the "repurposed to the resolved coordinate" reading: declared and resolved are now proven equal, so the field needs no new source.
**Rationale**: Smallest true implementation of D41's one-line residue; no context shape change, no transformer breakage possible (measured zero readers).

### Cache key trim

**Decision**: `normSub` drops `Range`/`Allow`/`Deny`, the filter lookup block, both `sort.Strings` calls, and the `sort` import; doc comment loses the ordering clause. No invalidation concern: v2 keys are byte-identical (the fields were omitted-when-absent and a v2 subscription cannot express them); v1 platforms stopped loading at the retarget.

## Technical Notes

### Deletion inventory

`filter.go` (145), `filter_test.go` (160, minus the four `TestHighestStable` cases already moved by `library-compat-comparator`), `materialize.go:63-96` block + `decodeFilter` (`:127-147`), `enumerate_pull_test.go:23-52` rewritten against the scalar path, `doc.go` step 2 rewritten, `types.go:58-62` `Resolved` doc rewritten, `materialize_test.go` NOTE tombstone resolved and fixtures' authored `version:` becomes the asserted selection (the happy-path assertion flips from coincidence to contract).

### Spec deltas

- `platform-materialization`: MODIFIED Subscription Resolution (scalar + major agreement); REMOVED Version Enumeration and Filtering; ADDED Catalog Identity Verification; the named-build-missing diagnostic scenario.
- `registry-module-loading`: ADDED Module Identity Verification beside the shape-gate requirement.
- `platform-synthesis`: MODIFIED subscription synthesis (required `version`, no filter).

### Transitional scaffolding retirement

- `Taskfile.yml` `cue:catalog:drift`: the jq `highestStable` mirror (two sites) becomes "the pinned tag exists in the published list"; its header comment ("until library-acquire-and-subscription…") is resolved.
- `modules/opm_platform/platform.cue:29-39`: pin comment rewritten — the pin is load-bearing; bumping it is an ordinary fixture update, not a drift-check obligation.

### Open questions (carried into implementation, not blocking)

- Whether the acquire-site check should also run on the *file* loader path for a module that declares a registry-shaped `modulePath` (currently: no — there is no fetched coordinate to compare; D9's local-dir argument).
- Whether `MaterializeError` should gain a distinct `Kind` for identity (currently: no — `Kind: "catalog"` + `errors.As` on the wrapped `IdentityError` routes it; a new Kind is additive later if frontends want it).
