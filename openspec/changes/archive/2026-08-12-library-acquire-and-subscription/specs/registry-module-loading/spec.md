# registry-module-loading — Delta

## ADDED Requirements

### Requirement: Module Identity Verification

After the artifact shape gate passes, the registry module loader SHALL verify the acquired artifact's declared identity against the coordinate it was fetched by: `metadata.modulePath` SHALL equal the requested module path (both in the full major-suffixed form, compared as strings without recomposition), and `metadata.version` SHALL equal the fetched tag with its `v` prefix stripped. A disagreement SHALL fail the load with a typed identity error naming both the declared and the fetched value. The verification SHALL sit on the shared load path so every entrypoint — and every frontend calling through the kernel — inherits one implementation.

A major-free `metadata.modulePath` (the core-v0/v1 metadata shape, which cannot express the major-suffixed form) SHALL instead be verified against the publishing convention: the fetched module path, with its major suffix stripped, SHALL sit directly under the declared parent path (`metadata.modulePath + "/" + <leaf>`). This preserves the "Self-referential core@v0 metadata is preserved" scenario while still refusing a declaration that does not match the fetched coordinate.

The kernel SHALL NOT write or correct either value: the schema declares identity, the reader verifies it (the kernel is a verifier, never a stamper).

#### Scenario: Address mismatch refused at acquire

- **WHEN** a published module is fetched by `opmodel.dev/modules/demo@v1` but declares `metadata.modulePath: "opmodel.dev/modules/other@v1"`
- **THEN** the load fails with a typed error carrying both paths

#### Scenario: Version mismatch refused at acquire

- **WHEN** a published module is fetched by tag `v2.0.2` but declares `metadata.version: "2.0.0"`
- **THEN** the load fails with a typed error carrying both versions

#### Scenario: Older-line parent-path declaration verified by convention

- **WHEN** a module fetched by `testing.opmodel.dev/x/modules/hello@v0` declares the major-free `metadata.modulePath: "testing.opmodel.dev/x/modules"` (core-v0/v1 shape)
- **THEN** the load succeeds
- **AND** a major-free declaration that is not the fetched path's parent fails with the same typed error

#### Scenario: Honest artifact loads

- **WHEN** the declared path and version match the fetched coordinate
- **THEN** the load proceeds exactly as before this requirement
