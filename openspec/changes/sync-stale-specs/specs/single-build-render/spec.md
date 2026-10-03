## MODIFIED Requirements

### Requirement: Render is the kernel's sole render path

`Kernel.Render` SHALL be the only way the kernel renders an instance against a platform. The kernel SHALL expose no `Compile`, `Match` or `Materialize` method and no materialized-platform type; a dry run is `Render` with the rendered output discarded, since the build evaluates every matched pair regardless. The kernel's default core schema pin SHALL be a release carrying the 0019:D5 registry shape (`schema-dispatch`, "DefaultSchemaModule constant"), so every artifact the kernel synthesizes or validates is judged against that shape. A platform module importing its catalogs is the only platform shape the kernel accepts: the platform shape gate SHALL validate every `#registry` entry for completeness (`helper-packages`, "Loader shape gate validates identity and registry completeness"), so an entry that names no embedded catalog is refused at acquisition. Core derives the entry's `version` from the embedded catalog's stamped identity, and with no catalog that readout is a missing required field.

#### Scenario: Old entry points are gone

- **WHEN** a consumer inspects the exported identifiers of `opm/kernel`
- **THEN** none of `Compile`, `Match`, `Materialize`, `SynthesizePlatform`, `CompileInput`, `MatchInput`, `CompileResult`, `MatchPlan` exists, and no `opm/materialize` or `opm/compile` package exists

#### Scenario: A subscription-shaped platform is refused

- **WHEN** a platform package declares a registry entry with a `version` scalar and no embedded catalog
- **THEN** `AcquirePlatformFromDir` fails with an error wrapping `ErrMissingRequiredField` that names the entry's `version` as a required field the embedded catalog would have supplied, and no render is attempted
