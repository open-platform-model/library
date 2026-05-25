## REMOVED Requirements

### Requirement: Schema apiVersion is a concrete literal
**Reason**: Core schemas migrated to a dedicated `core` repo with `apiVersion` removed from every artifact root. The literal no longer exists in the schema.
**Migration**: None — library has no external consumers. Authoring code must not set `apiVersion` on any artifact.

### Requirement: Version detection from a CUE artifact
**Reason**: With no `apiVersion` field on artifacts, version detection has no input to read. Caller statically knows the artifact type at every site.
**Migration**: Delete `opm/apiversion` package. Callers that previously used `apiversion.Detect(v)` to dispatch on version delete the call; there is one schema.

### Requirement: Binding contract
**Reason**: With one schema, the `Binding` interface dispatched to one implementation. The indirection added no value.
**Migration**: Replaced by free functions in `opm/schema` (see `schema-dispatch::Path inventory exposed as package-level vars`, `schema-dispatch::Metadata decoders are free functions`, `schema-dispatch::Transformer-context builder`, `schema-dispatch::Cached embedded-schema loader`).

### Requirement: Binding registry
**Reason**: A registry mapping one key to one value is unnecessary structure.
**Migration**: Delete `opm/api` package. Remove the `_ "github.com/open-platform-model/library/opm/api/v1alpha2"` blank import from every main/cmd; nothing replaces it.

### Requirement: Loader surfaces detected version
**Reason**: No version to detect.
**Migration**: `file.Load{Module,Release,Platform}Package` return `(cue.Value, error)`. Kernel wrappers follow. Callers that destructured the version drop the variable.

### Requirement: Render dispatches via binding
**Reason**: No binding to dispatch through.
**Migration**: `kernel.ProcessModuleRelease` calls `schema.DecodeReleaseMetadata` directly. The apiVersion-mismatch check between release and platform is removed (the field does not exist on either).

### Requirement: Embedded schemas
**Reason**: Replaced by a single embedded schema with no per-version directory.
**Migration**: Replaced by `schema-dispatch::Single embedded schema with no apiVersion field` and `schema-dispatch::Cached embedded-schema loader`. The library vendors one flat copy of the schema at `apis/core/*.cue`; the `EmbeddedSchema(version)` API is gone.

### Requirement: New schema version is added without kernel edits
**Reason**: The library no longer supports multiple schema versions. New schema changes happen in the upstream `core` repo and are re-vendored as a single overwrite.
**Migration**: To roll a new schema, replace `library/apis/core/*.cue` and `library/apis/core/cue.mod/module.cue` from the new `core` repo and rebuild.

### Requirement: Binding exposes its loaded schema as a cue.Value
**Reason**: Replaced by the package-level `schema.SchemaValue(ctx)` function with identical caching contract.
**Migration**: Replace `binding.SchemaValue(ctx)` with `schema.SchemaValue(ctx)`. The per-binding cache becomes a package-level `sync.Once`; the contract (cached, concurrent-safe, errors returned but cached) is preserved.
