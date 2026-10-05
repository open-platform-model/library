## MODIFIED Requirements

### Requirement: Helper Layout for Future Subpackages

Future opt-in helpers SHALL follow the `opm/helper/<name>/` convention. Subpackages SHALL be added by their owning slices and not as part of the originating slice that established the convention. Past examples of helper subpackages in this convention SHALL reflect the current package layout; subpackages that have been collapsed into the kernel (the previous `opm/helper/values/`, `opm/helper/loader/file/`, `opm/helper/loader/registry/` and `opm/helper/synth/`) SHALL NOT appear as exemplars. The current subpackages are `platformmodule` (platform module generation from catalog coordinates) and `objectset` (duplicate rendered object identities). `objectset` is Deprecated: the check's home is `opm/k8s/object` (ADR-011 item 9), and the helper copy stays only until both frontends have migrated (`duplicate-object-identities`).

#### Scenario: Platform helper landing place

- **WHEN** a frontend needs to generate a platform module from catalog coordinates
- **THEN** `opm/helper/platformmodule/` is the directory that helper occupies

#### Scenario: Duplicate-identity helper landing place

- **WHEN** a runtime needs to refuse a render whose objects share one apply identity
- **THEN** `opm/k8s/object` is the package that provides the check, the deprecated `opm/helper/objectset/` names it as the replacement, and the kernel calls neither

#### Scenario: Values helper subpackage no longer exists

- **WHEN** a developer searches `opm/helper/` for a `values` subpackage
- **THEN** no `opm/helper/values/` directory exists
- **AND** the canonical implementation of layered values validation lives at `Kernel.ValidateConfigDetailed` in `opm/kernel/`

#### Scenario: Loader and synth subpackages no longer exist

- **WHEN** a developer lists the subpackages of `opm/helper/`
- **THEN** exactly `platformmodule` and `objectset` are present, `objectset` marked Deprecated
- **AND** neither `opm/helper/loader/` nor `opm/helper/synth/` exists
