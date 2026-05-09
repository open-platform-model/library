## MODIFIED Requirements

### Requirement: Helper Layout for Future Subpackages

Future opt-in helpers SHALL follow the `pkg/helper/<name>/` convention. Subpackages SHALL be added by their owning slices (e.g. `pkg/helper/platform/` by slice 10) and not as part of the originating slice that established the convention. Past examples of helper subpackages in this convention SHALL reflect the current package layout; subpackages that have been collapsed into the kernel (such as the previous `pkg/helper/values/`) SHALL NOT appear as exemplars.

#### Scenario: Platform helper landing place

- **WHEN** slice 10 (`add-platform-composition-helper`) is implemented
- **THEN** `pkg/helper/platform/` is the directory it occupies
- **AND** the convention is consistent with `pkg/helper/loader/file/`

#### Scenario: Values helper subpackage no longer exists

- **WHEN** a developer searches `pkg/helper/` for a `values` subpackage
- **THEN** no `pkg/helper/values/` directory exists
- **AND** the canonical implementation of layered values validation lives at `Kernel.ValidateConfigDetailed` in `pkg/kernel/`
