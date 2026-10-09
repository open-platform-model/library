## MODIFIED Requirements

### Requirement: Platform carries its render source

`platform.Platform` SHALL expose `Source *module.Source`. The source type has one name, `module.Source`. `opm/platform` SHALL NOT re-export it. `opm/catalog` SHALL NOT re-export it either. `Source` SHALL be nil when the platform was constructed from a bare `cue.Value`. `Source` is the render input: `Render` imports the platform package from it, so a platform without `Source` cannot be rendered against (`single-build-render`, "Render inputs are source-carrying artifacts"). No other kernel operation reads the field.

#### Scenario: Value-constructed platform has no source

- **WHEN** a caller builds a platform via `NewPlatformFromValue`
- **THEN** the returned `*Platform` has `Source == nil`

#### Scenario: The source type has one name

- **WHEN** a developer inspects the exported identifiers of `opm/platform` and `opm/catalog`
- **THEN** neither package exports a `Source` type, and the `Source` field of `Platform` and of `Catalog` is a `*module.Source`

#### Scenario: Source-less platform cannot render

- **WHEN** a platform built via `NewPlatformFromValue` is passed to `Render`
- **THEN** `Render` refuses it with an error naming the missing source, before any build is staged

