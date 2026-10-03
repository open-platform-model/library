## Why

The kernel's two registry verbs, `Kernel.AcquireModuleFromRegistry` and
`Kernel.AcquireCatalogFromRegistry`, refuse a bare SemVer version. Both go through
`loader.FetchArtifact` (`opm/internal/loader/registry.go:82`), which hands the caller's version
straight to `module.NewVersion`. In `cuelang.org/go` v0.17.1, `NewVersion` checks the version
with Go's `semver.IsValid`, which needs the `v` prefix, so `"1.0.0"` fails as "not well formed".

The operator's `TransformerRegistration` documents `spec.version` as bare SemVer (`"1.0.0"`) and
passes it unchanged to `AcquireCatalogFromRegistry`, so every real claim is refused as
`CatalogUnresolved`. A `v`-prefixed claim is not a workaround. The same value is stamped raw as
the registry entry's `version:` in the generated platform module
(`opm/helper/platformmodule/generate.go:189`), where it unifies with the catalog's bare
`metadata.version`. A `"v1.0.0"` claim would pass acquisition and then conflict in the platform
build. No claim value works on both paths, so the claim pipeline is unusable end to end.

The kernel already normalises the version in two places: `verifyModuleIdentity` strips the
prefix (`registry.go:168`), and `platformmodule.canonicalVersion` (`generate.go:197-204`) adds
it for `Roots`. The registry verbs are the one entry that still demands a single spelling, and
their doc comments (`opm/kernel/acquire.go:39-40, 130-131`) say `"v4.3.0"` while the operator
says `"1.0.0"`.

Owner decision (walkthrough item i1, 2026-10-02): normalise in the kernel, so both `1.0.0` and
`v1.0.0` are accepted, through one shared `canonicalVersion` helper. This change is the library
half. The operator e2e test that drives a real claim through acceptance and the platform build
comes in a later change, after the library release.

## What Changes

- **One shared canonicalisation helper.** `platformmodule.canonicalVersion` moves to a new
  internal package, `opm/internal/modversion`, as `Canonical(v string) string`, with the same
  behaviour: a bare SemVer gains the `v` prefix; an already-prefixed or empty string comes back
  unchanged. `platformmodule.Roots` and its tests call it. A later change (walkthrough item b3)
  reuses this package for the library's other version helpers.
- **The registry verbs accept both spellings.** `loader.FetchArtifact` canonicalises the version
  before `module.NewVersion`, and `loader.FetchModule` canonicalises the version it hands the
  identity check, so every coordinate after the parse step that reaches an error message or the identity check is
  in canonical form. A malformed version (`"not-a-version"`) is still refused with the wrapped
  parse error, which names the version as the caller wrote it.
- **The synthetic root is the same for both spellings.** `sourcetree.SyntheticRoot`
  canonicalises the version it is given. `Source.Root` of an artifact fetched as `1.0.0` equals
  the one fetched as `v1.0.0`.
- **Doc comments state the contract.** The godoc of `Kernel.AcquireModuleFromRegistry`,
  `Kernel.AcquireCatalogFromRegistry` and `loader.FetchArtifact` say the version may be written
  either way.
- **Tests** cover both spellings through the `registrytest` fixture: at the loader for a module
  and a catalog, and at the kernel for `AcquireCatalogFromRegistry`.

Not **BREAKING**. No exported symbol changes. A caller that passed a `v`-prefixed version sees no
difference; a bare version that was refused now loads.

SemVer class: PATCH. The release-bearing commit is `fix(loader)`; the helper lift is a
`refactor`. The PR title is `fix(loader): accept bare semver in the registry verbs`.

Not in this change: the operator pin bump and the operator e2e test (later wave), consolidating
the library's other version helpers and the Masterminds dependency (b3), and any change to how
`platformmodule.Generate` stamps the entry version. Generate stamps the subscription's version as
given, so a `v`-prefixed claim still conflicts at platform build; the operator keeps the bare form
its CRD documents (or the owner decides on Generate normalisation separately). Both spellings are
accepted by the registry verbs only, not by the claim pipeline end to end (design Risks).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `registry-module-loading`: a new requirement that the registry acquisition verbs accept a
  version with or without the `v` prefix, resolve both to the same artifact, and stage it under
  the same synthetic root.

## Impact

- Packages: `opm/internal/modversion` (new, internal), `opm/internal/loader` (`registry.go`),
  `opm/internal/sourcetree` (`SyntheticRoot`), `opm/helper/platformmodule` (`generate.go`, the
  closure test), and godoc in `opm/kernel/acquire.go`. The helper tier may import
  `opm/internal/*`; the depguard rule forbids only the reverse.
- Public surface under `opm/`: no signature change. Behaviour widens: two verbs accept input they
  refused before.
- Downstream: opm-operator's `TransformerRegistration` claims start resolving once it bumps to
  the release carrying this fix. The cli passes `v`-prefixed versions and is unaffected.
- No `enhancement.yaml`: the owner decision comes from the beta-1 kernel-plan walkthrough, not
  from an enhancement decision.
