## REMOVED Requirements

### Requirement: Subscription Resolution

**Reason**: `opm/materialize` is deleted (0019 D5/D9). A platform module's `cue.mod` names its catalog builds and the render build resolves them like any CUE dependency; there is no subscription scalar to resolve.

**Migration**: pin catalogs in the platform module's `cue.mod`; `Render` resolves them through the promoted render module.

### Requirement: Catalog Identity Verification

**Reason**: core's D5 tripwires replace it: the registry key binds to the embedded catalog's `modulePath`, and an expected `version` stamp unifies with the derived readout, so a wrong build is a build conflict naming the entry.

**Migration**: none; the check is structural in the schema.

### Requirement: Transformer Indexing

**Reason**: `#composedTransformers` is derived by core as a fold over enabled entries; the reverse index is built by the render glue from it (0019 D10/D17).

**Migration**: none.

### Requirement: MaterializedPlatform Output Shape

**Reason**: the type is deleted; nothing built is shared between renders (0019 D8).

**Migration**: hold a source-carrying `*platform.Platform` (the acquired module) instead; `Render` builds per call.

### Requirement: MaterializeError Diagnostic

**Reason**: no materialize step exists. Unresolvable catalog builds surface as `Render` staging or load errors naming the dependency; wrong bytes surface as the D5 conflict naming the entry.

**Migration**: match on the render error surface instead of `*oerrors.MaterializeError`.

### Requirement: Single-Provider Guard

**Reason**: relocated into the render build as an in-build diagnostics row and gate (`single-build-render`, "The single-provider guard runs inside the build"); the divergent-fulfilment arm is not carried because one build holds one set of catalog bytes.

**Migration**: the refusal moves from `Materialize` to `Render`, as a typed over-subscription error naming the key and the catalog paths.
