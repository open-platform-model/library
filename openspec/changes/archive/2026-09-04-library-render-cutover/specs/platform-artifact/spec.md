## ADDED Requirements

### Requirement: Platform carries its render source

`platform.Platform` SHALL expose `Source *platform.Source`, where `platform.Source` is a type alias of `module.Source` (the same re-export pattern as `platform.PlatformMetadata`). `Source` SHALL be nil when the platform was constructed from a bare `cue.Value`. `Source` is the render input: `Render` imports the platform package from it, so a platform without `Source` cannot be rendered against (`single-build-render`, "Render inputs are source-carrying artifacts"). No other kernel operation reads the field.

#### Scenario: Value-constructed platform has no source

- **WHEN** a caller builds a platform via `NewPlatformFromValue`
- **THEN** the returned `*Platform` has `Source == nil`

#### Scenario: Source-less platform cannot render

- **WHEN** a platform built via `NewPlatformFromValue` is passed to `Render`
- **THEN** `Render` refuses it with an error naming the missing source, before any build is staged

## REMOVED Requirements

### Requirement: Optional Platform Field on Phase Inputs

**Reason**: the phase input structs and the materialized platform type are removed; `RenderInput.Platform` is a required source-carrying `*platform.Platform` (`single-build-render`).

**Migration**: pass the platform acquired from its module directory on `RenderInput.Platform`.

### Requirement: Platform carries its staged source

**Reason**: the requirement promised that `Materialize`, `Match` and `Compile` ignore `Source` and that the existing pipeline is unaffected; that pipeline is deleted and `Render` requires the field. Restated as "Platform carries its render source".

**Migration**: none; the field and its alias are unchanged.

### Requirement: Binding Path Constants for Platform Views

**Reason**: binding-era text (`Paths().Registry`, `KnownResources`, `Matchers`) retired by `schema-dispatch`, which forbids exporting platform view paths; after the cutover no Go code reads `#registry` or `#composedTransformers` (the render glue reads them in CUE), and `DecodePlatformMetadata` is specified under `schema-dispatch`, "Metadata decoders are free functions".

**Migration**: none.
