## MODIFIED Requirements

### Requirement: In-Memory Load Without a Temporary Directory

The registry module loader SHALL load the fetched module in memory and SHALL NOT write the module's source to a temporary directory. It SHALL inject the fetched module's CUE files (every `.cue` file under the module root, the module's own `cue.mod/module.cue` included, and nothing else) via `load.Config.Overlay` under a deterministic synthetic root, leaving `load.Config.FS` nil so the module's transitive dependencies resolve through the registry and CUE module cache. The staged overlay the loader returns on the module's `Source` SHALL be that same set of files.

#### Scenario: No temporary directory created

- **WHEN** a module is loaded from the registry
- **THEN** no temporary directory is created or left behind for the module's source
- **AND** the module's transitive dependencies still resolve

#### Scenario: Staged overlay carries the module's CUE files only

- **WHEN** a fetched module's archive contains `.cue` files, a `cue.mod/module.cue`, and non-CUE files such as a license or a readme
- **THEN** the staged overlay on `Module.Source` holds every `.cue` file and `cue.mod/module.cue`, each keyed under the synthetic root
- **AND** the non-CUE files are not present in the overlay
- **AND** an archive with no `.cue` file fails the load with an error wrapping `ErrInvalidPackage`

### Requirement: Module Identity Verification

After the artifact shape gate passes, the registry module loader SHALL verify the acquired artifact's declared identity against the coordinate it was fetched by: `metadata.modulePath` SHALL equal the requested module path (both in the full major-suffixed form, compared as strings without recomposition), and `metadata.version` SHALL equal the fetched tag with its `v` prefix stripped. A disagreement SHALL fail the load with a typed identity error naming both the declared and the fetched value. The verification SHALL sit on the shared load path so every entrypoint, and every frontend calling through the kernel, inherits one implementation.

There SHALL be no alternative verification for a major-free `metadata.modulePath`: the OPM schema the library consumes requires the major-suffixed form, so a declaration without the suffix cannot equal the fetched path and is refused with the same typed error. The library SHALL NOT compose a module address from a parent path and a name.

The kernel SHALL NOT write or correct either value: the schema declares identity, the reader verifies it (the kernel is a verifier, never a stamper).

#### Scenario: Address mismatch refused at acquire

- **WHEN** a published module is fetched by `opmodel.dev/modules/demo@v1` but declares `metadata.modulePath: "opmodel.dev/modules/other@v1"`
- **THEN** the load fails with a typed error carrying both paths

#### Scenario: Version mismatch refused at acquire

- **WHEN** a published module is fetched by tag `v2.0.2` but declares `metadata.version: "2.0.0"`
- **THEN** the load fails with a typed error carrying both versions

#### Scenario: Older-line parent-path declaration verified by convention

- **WHEN** a module fetched by `testing.opmodel.dev/x/modules/hello@v0` declares the major-free `metadata.modulePath: "testing.opmodel.dev/x/modules"` (the core-v0/v1 shape)
- **THEN** the load fails with the typed identity error for the `path` field, carrying the declared parent path and the fetched path
- **AND** no publishing-convention fallback is attempted

#### Scenario: Honest artifact loads

- **WHEN** the declared path and version match the fetched coordinate
- **THEN** the load proceeds exactly as before this requirement
