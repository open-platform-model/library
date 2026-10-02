## MODIFIED Requirements

### Requirement: Generation is pure and deterministic

The helper SHALL render a platform module's two files (`cue.mod/module.cue`, `platform.cue`) from typed input only: the platform's name and type, its registry entries (major-qualified catalog module path, the catalog build as a SemVer string, enabled flag), the module path the generated module declares, and the resolved dependency list. It SHALL perform no I/O and read no environment. Entries and dependencies SHALL be emitted in sorted path order whatever order they arrive in, so identical input yields byte-identical files. `platform.cue` SHALL embed `core.#Platform`, declare `metadata.name` and `type`, import each entry's catalog under a positional alias, and write one `#registry` entry per subscription carrying `enable`, the entry's expected `version` (the 0019 D13 tripwire: it unifies with core's readout of the imported catalog) and `#catalog` bound to the import. `cue.mod/module.cue` SHALL declare the caller's module path, the language version `v0.17.0`, and every dependency without a default-major marker.

#### Scenario: Same input, same bytes

- **WHEN** the generator is invoked twice with the same entries and dependencies in different slice orders
- **THEN** both results are byte-identical, entries and dependencies sorted by path

#### Scenario: Two catalogs

- **WHEN** two enabled entries for `opmodel.dev/catalogs/opm@v4` and `example.com/catalogs/extra@v1` are generated with a closure pinning both
- **THEN** `platform.cue` imports both under distinct aliases and its `#registry` carries both keys with `enable: true`, the stamped versions and `#catalog` bound to the matching alias

#### Scenario: Disabled entry still imports its catalog

- **WHEN** an entry is generated with `enable` false
- **THEN** the catalog is still imported and the entry carries `enable: false`; the dependency list still pins it

#### Scenario: Invalid input is refused

- **WHEN** the input has an empty name or type, a duplicate registry path, an entry or dependency with an empty path or version, a dependency pinned twice at different versions, or a dependency list missing core or an entry's catalog
- **THEN** generation returns an error naming the offending field or path and emits no files

