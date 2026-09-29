## Why

The single-provider rule (a contract declared `fulfilment: "provider"` has exactly one provider on a platform; 0010:D37, 0015:D2/D18) is counted twice today, and the two counts disagree. Core's `#Platform.#contracts` counts only contracts an enabled catalog defines and keys a provider by the transformer's stamped, major-free `metadata.modulePath`. The render glue's `guard._providerSuppliers` counts every enabled registry entry's transformers and keys them by registry key (path plus major). Two platforms fall between them, both measured in CUE:

- **Bug 1.** Two majors of one provider catalog enabled (k8up v2 plus v3, both requiring `backup`): the inventory counts one provider and reads routable, the render guard counts two and refuses every render on the platform.
- **Bug 2.** Two providers enabled while the catalog that defines the contract is disabled or absent: the inventory omits the contract and reads routable, the render guard counts two and refuses every render.

In both cases `opm platform check` and the operator's generation gate pass a platform the kernel then refuses on every render. The user approved (2026-09-29) making the render guard's count the only count: core change `count-providers-per-registry-entry` (change A of this set) recounts `#contracts` exactly like the guard, computes it once as `#contracts.providedBy`, and ships in core `2.0.0-alpha.12`. This change is change B: the library re-pins to that core, reads the count from core instead of computing its own, and refuses a platform whose core predates it, so the two counts cannot drift again.

## What Changes

- **A parity test, written first.** A new hermetic test in `opm/kernel` acquires every served render platform (each `testdata/render/platform*` directory, eight with the two new ones), reads `Contracts()` and renders the `instance` fixture against each, and asserts that the inventory's over-subscription verdict equals the render's over-subscription rows. Two new fixtures reproduce the bugs (`platform_two_majors`: `cat@v0` plus a new `cat@v1` catalog; `platform_definer_disabled`: `cat@v0` disabled, `cat2` 0.2.0 plus `cat@v1` enabled). It is run red on core `alpha.10` and turns green on core `alpha.12` with the glue unchanged, then stays as the tripwire against the glue ever growing its own count again.
- **Core pin to `v2.0.0-alpha.12`.** `schema.DefaultSchemaModule`, `registrytest.DefaultCoreVersion`, every `testdata/render/**` `cue.mod`, the modules `task cue:deps:update` discovers (`modules/opm_platform`, `testdata/parity`, `testdata/parity/opm_platform`, `testdata/modules/web_app`), `testdata/cue.mod` and the version literals in tests. A deliberate re-verification per the `DefaultSchemaModule` doc comment.
- **`ContractInventory.ProvidedBy`.** `Contracts()` decodes `#contracts.providedBy` (contract FQN to the sorted registry keys of the enabled entries whose transformers require it), the one fact the cli and the operator need to name the catalogs to disable when the defining catalog is absent (Bug 2 leaves `DefinedBy` and `RequiredBy` empty). The `OverSubscribed` and `Unfulfilled` doc comments say "registry entry (path plus major)" instead of "catalog".
- **BREAKING: an older core is refused, never degraded.** A new typed error `errors.PlatformCoreTooOldError{Platform, Field, Since}` is returned by `Contracts()` when `#contracts` lacks `providedBy`, and by `Kernel.Render` before staging when the platform package lacks it. No fallback count exists: a render never runs on a platform whose inventory disagrees with it.
- **The render reads core's count.** In `render.cue.tmpl`, `match.#providers` is `platform.#contracts.providedBy`, the over-subscription rows iterate `providedBy` filtered by `overSubscribed`, and the gate reads `platform.#contracts.routable`. `guard._providerSuppliers` is deleted. The row shape `{key, catalogs}`, `OverSubscribedContract` and the refusal text are unchanged.
- **Specs and docs** follow: the in-build guard requirement now reads core's count, the unprovided definition cites `providedBy`, the platform inventory gains the definer-disabled scenario, and the AGENTS render-pipeline prose, the oversubscribed-contracts diagnostics page and the `opm/errors`, `opm/kernel`, `opm/schema` doc comments say who counts.

## Classification

**MAJOR (pre-GA, released as `feat(kernel)!`).** One exported field (`ContractInventory.ProvidedBy`) and one exported error type (`errors.PlatformCoreTooOldError`) are additive, but two behaviours break:

- `Kernel.Render` and `Platform.Contracts()` refuse any platform module pinning core older than `2.0.0-alpha.12`, where they rendered or decoded before.
- Bug 1 and Bug 2 platforms now read `Routable: false` from `Contracts()`. Every render on them was already refused, so render outcomes do not change; only the inventory verdict does.

Pre-GA, so no migration fragment (`migrations/README.md`). Complexity: net negative in the glue (one comprehension deleted), plus one typed error with two call sites (Principle VII).

## Downstream consumers

- **Generated platforms move automatically.** The operator and the cli's cluster and deps sources generate platform modules pinned to `schema.DefaultSchemaVersion()`, so bumping `DefaultSchemaModule` re-pins all of them on upgrade. The operator empties `--platform-dir` at start and regenerates a held package whose `Contracts()` errors.
- **Hand-written `--platform` modules pinned to `alpha.10` or `alpha.11` are refused until re-pinned.** In the workspace: `cli/hack/platform` and `cli/examples` (through the workspace `task deps:update`), this repo's testdata and `modules/opm_platform` (this change), and the personal `opm-kind-demo`. Outside `task deps:update`'s reach: `opm-suite-installer/platform` pins core `v2.0.0-alpha.10` and vendors core under `vendor/`. It is safe until its checksum-pinned `opm` (`OPM_VERSION` in its `Containerfile`, today `v1.0.0-alpha.20`) moves past D's release; that bump must re-pin and re-vendor core at the same time.
- **`opm-operator`** (change C) and **`cli`** (change D) consume this change's release: they name `ProvidedBy[contract]` in their over-subscription messages, word `PlatformCoreTooOldError` as a re-pin hint, and the operator's registration acceptance reads `ProvidedBy` instead of its own count.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `single-build-render`: the in-build single-provider guard reads core's `#contracts.providedBy`, `overSubscribed` and `routable` instead of computing its own count; the unprovided definition cites `providedBy`; the render refuses a platform whose core predates `providedBy` before staging.
- `platform-artifact`: `Contracts()` decodes `ProvidedBy`, counts providers per registry entry (two majors of one catalog are two providers; a defining catalog need not be enabled), refuses a pre-`alpha.12` inventory with the typed error, and gains the two-majors and definer-disabled scenarios.
- `schema-dispatch`: `DefaultSchemaModule` moves to `opmodel.dev/core@v2.0.0-alpha.12`.

## Impact

- `opm/internal/renderstage/render.cue.tmpl` (the guard, `#providers`, rows, gate).
- `opm/kernel/render.go` (pre-staging core floor), `doc.go`; new `opm/kernel/render_inventory_parity_test.go`; `render_test.go`, `render_skip_test.go`.
- `opm/platform/contracts.go`, `contracts_test.go`; `opm/errors` (new `PlatformCoreTooOldError`, `oversubscribed.go` doc); `opm/schema/loader.go`, `paths.go`, `loader_test.go`.
- `opm/internal/registrytest` (`DefaultCoreVersion`); every test literal naming `alpha.10` (the deliberately older skew rows stay).
- `testdata/render/registry/testing.opmodel.dev_library-render_cat_v1.0.0/` (new), `testdata/render/platform_two_majors/`, `testdata/render/platform_definer_disabled/` (new), `testdata/render/scenarios/README.md`, every `testdata/render/**` and discovered module `cue.mod`.
- `AGENTS.md`, `docs/site/diagnostics/oversubscribed-contracts.md`.
- Release: `feat(kernel)!` overall; waits on core `2.0.0-alpha.12` (change A) being resolvable from GHCR.
