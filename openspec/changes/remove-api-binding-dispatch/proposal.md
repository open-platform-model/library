## Why

The core OPM CUE schemas have moved to a dedicated `core` repo and `apiVersion` has been dropped from every artifact root. With a single, unversioned schema the entire per-version `Binding` indirection — registry, lookup, init-time registration, `apiversion.Version` type, `apiversion.Detect` helper — exists only to dispatch to its one and only implementation. Deleting it removes ~1k LoC of dead structure, simplifies every kernel consumer, and stops the codebase paying interface-tax for a problem it no longer has.

## What Changes

- **BREAKING (`opm/api`)** — Delete `opm/api` package entirely (`api.go`, `registry.go`, `embed.go`, `doc.go`, `v1alpha2/{binding,init,decode,context,consts,schemavalue}.go`, all tests).
- **BREAKING (`opm/apiversion`)** — Delete `opm/apiversion` package entirely (`apiversion.go`, tests). No more `Version` type, `Detect` helper, or `ErrUnknownAPIVersion` sentinel.
- **New (`opm/schema`)** — Single home for schema-side knowledge: `Paths` as package-level `var`s, `ModuleMetadata`/`ReleaseMetadata`/`ProviderMetadata`/`PlatformMetadata` types, `ReleaseView` interface, `Decode*Metadata` free functions, `BuildTransformerContext`, `SchemaValue` (cached embed loader), `AnnotationDefaultNamespace` const.
- **BREAKING (`opm/module`)** — `Module.APIVersion` and `Release.APIVersion` fields removed. `NewModuleFromValue` / `NewReleaseFromValue` call `schema.Decode*Metadata` directly; the `apiversion.Detect` step disappears.
- **BREAKING (`opm/platform`)** — `Platform.APIVersion` field removed. `NewPlatformFromValue` calls `schema.DecodePlatformMetadata` directly.
- **BREAKING (`opm/kernel`)** — `(*Kernel).DetectAPIVersion` deleted. `LoadModulePackage`/`LoadReleasePackage`/`LoadPlatformPackage` signatures lose the `apiversion.Version` return.
- **BREAKING (`opm/compile`)** — `Match` and `(*Module).Execute` no longer take an `api.Binding` parameter. Internal helpers (`candidateSatisfied`, `lookupCandidates`, `pairTransformer`, `extractComponentSummaries`) drop the `paths api.Paths` parameter and reference `opm/schema` package vars directly.
- **BREAKING (`opm/helper/loader/file`)** — `LoadModulePackage`/`LoadReleasePackage`/`LoadPlatformPackage` signatures lose the `apiversion.Version` return.
- **BREAKING (`opm/helper/platform`)** — `Compose` drops `shell.APIVersion` lookup; uses `opm/schema.Registry` directly.
- **BREAKING (`opm/helper/synth`)** — `Release` drops `api.Lookup(in.Module.APIVersion)`; calls `schema.SchemaValue(ctx)` directly. `ErrSchemaUnavailable` is retained.
- **CUE schema (`apis/core/`)** — Library's embedded copy is re-synced wholesale from `core/src/` in the new `core` repo: flat layout (no `v1alpha2/` subdir), package renamed from `v1alpha2` to `core`, `apiVersion: #ApiVersion` removed from every artifact root, `#ApiVersion` constant dropped, `cue.mod/module.cue` carries the new identity `opmodel.dev/core@v0`. The re-synced file set includes `catalog.cue` and `module_context.cue` — both introduced by enhancement 0001's core slice ahead of Part B; the Go kernel does not consume them yet, and copying the directory wholesale keeps Part B insulated from future core-slice file additions. The re-synced schema is post-0001 in `#Platform.#registry`, `#FQNType`, and `#Module.#ctx` shape; see `design.md` D10 for the consumer-fixture coordination.
- **BREAKING (`cmd/flow-inspect`)** — drops the blank import of `opm/api/v1alpha2`, the `apiVersion: "opmodel.dev/v1alpha2"` literal in the release skeleton, and the binding-Paths plumbing — reads directly from `opm/schema`.
- **Quarantine (consumer fixtures)** — `library/modules/opm_platform/platform.cue` and `library/testdata/modules/web_app/` use the pre-0001 schema shape. Part B quarantines them (delete or build-tag-gate) and `t.Skip`s `opm/kernel/{flow_integration_test,flow_synth_integration_test}.go`. Enhancement 0001's library slice rewrites the fixtures against `#Subscription` / `#ctx` / SemVer FQNs once it lands.

## Capabilities

### New Capabilities
- `schema-dispatch`: Single-schema artifact navigation and metadata decoding. Replaces the deleted `api-version-dispatch` capability. Covers the CUE path inventory, decoded metadata shapes, `ReleaseView` interface, free-function decoders for module/release/provider/platform metadata, the transformer-context builder, and the cached embedded-schema loader.

### Modified Capabilities
- `api-version-dispatch`: REMOVED. All requirements (concrete `#ApiVersion`, `Detect` helper, `Binding` interface, registry, embedded schema lookup) are dropped. Replaced by `schema-dispatch` operating on a single unversioned schema.

## Impact

- **Affected packages**: `opm/api` (deleted), `opm/apiversion` (deleted), `opm/schema` (new), `opm/module`, `opm/platform`, `opm/kernel`, `opm/compile`, `opm/helper/loader/file`, `opm/helper/platform`, `opm/helper/synth`, `cmd/flow-inspect`.
- **Affected schemas**: `apis/core/v1alpha2/` (deleted) → `apis/core/*.cue` (flat, re-synced from new `core` repo).
- **Downstream consumers**: Library has no external consumers yet (per user). No deprecation shim — every public surface that exposed `apiversion.Version` or `api.Binding` breaks at compile time, by design.
- **SemVer**: MAJOR. Every package listed above breaks API.
- **Tests**: `opm/api/v1alpha2/{binding,embed,schemavalue}_test.go` and `opm/api/registry_test.go` and `opm/apiversion/apiversion_test.go` deleted. Tests in `opm/module`, `opm/platform`, `opm/kernel` rewritten to drop the `_ "opm/api/v1alpha2"` blank imports, `apiversion.V1alpha2` references, and `apiVersion: "..."` literals in CUE fixtures.
