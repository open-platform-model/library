## MODIFIED Requirements

### Requirement: No platform synthesis helper

`opm/helper/synth` SHALL expose no `Platform` function and no `PlatformInput`/`SubscriptionSpec` types, and the kernel SHALL expose no `SynthesizePlatform`. A platform is a CUE module on disk that imports its catalogs (0019 D5/D6); a frontend that starts from catalog coordinates generates that module through `opm/helper/platformmodule` (`platform-module-generation`) or writes it by hand, and acquires it with `AcquirePlatformFromDir`. No helper SHALL turn typed subscription inputs into a platform `cue.Value` without a module on disk.

#### Scenario: Synth surface is instance-only

- **WHEN** a consumer inspects the exported identifiers of `opm/helper/synth`
- **THEN** `Instance` and its input types exist and no platform-synthesis identifier exists

#### Scenario: A frontend that synthesized platforms migrates to modules

- **WHEN** a frontend previously built a platform from typed subscription inputs
- **THEN** it generates or writes a platform CUE module (a `cue.mod` pinning the catalogs and their closure, a `platform.cue` importing them) and acquires it from that directory

### Requirement: Helper Layout for Future Subpackages

Future opt-in helpers SHALL follow the `opm/helper/<name>/` convention. Subpackages SHALL be added by their owning slices and not as part of the originating slice that established the convention. Past examples of helper subpackages in this convention SHALL reflect the current package layout; subpackages that have been collapsed into the kernel (such as the previous `opm/helper/values/`) SHALL NOT appear as exemplars. The current subpackages are `loader/file`, `loader/registry`, `synth` and `platformmodule`.

#### Scenario: Platform helper landing place

- **WHEN** a frontend needs to generate a platform module from catalog coordinates
- **THEN** `opm/helper/platformmodule/` is the directory that helper occupies, consistent with `opm/helper/loader/file/` and `opm/helper/synth/`

#### Scenario: Values helper subpackage no longer exists

- **WHEN** a developer searches `opm/helper/` for a `values` subpackage
- **THEN** no `opm/helper/values/` directory exists
- **AND** the canonical implementation of layered values validation lives at `Kernel.ValidateConfigDetailed` in `opm/kernel/`
