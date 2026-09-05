# platform-module-generation Specification

## Purpose

Generating a D5-shaped platform CUE module (a `cue.mod/module.cue` carrying the tidied dependency closure and a `platform.cue` importing every catalog) from typed registry entries, so a frontend that starts from catalog coordinates (a Platform CR, a seeded local default) obtains a directory `AcquirePlatformFromDir` accepts without writing its own generator.

## Requirements

### Requirement: Generation is pure and deterministic

The helper SHALL render a platform module's two files (`cue.mod/module.cue`, `platform.cue`) from typed input only: the platform's name and type, its registry entries (major-qualified catalog module path, the catalog build as a SemVer string, enabled flag), the module path the generated module declares, and the resolved dependency list. It SHALL perform no I/O and read no environment. Entries and dependencies SHALL be emitted in sorted path order whatever order they arrive in, so identical input yields byte-identical files. `platform.cue` SHALL embed `core.#Platform`, declare `metadata.name` and `type`, import each entry's catalog under a positional alias, and write one `#registry` entry per subscription carrying `enable`, the entry's expected `version` (the 0019 D13 tripwire: it unifies with core's readout of the imported catalog) and `#catalog` bound to the import. `cue.mod/module.cue` SHALL declare the caller's module path, the language version `v0.17.0`, and every dependency without a default-major marker.

#### Scenario: Same input, same bytes

- **WHEN** the generator is invoked twice with the same entries and dependencies in different slice orders
- **THEN** both results are byte-identical, entries and dependencies sorted by path

#### Scenario: Two catalogs

- **WHEN** two enabled entries for `opmodel.dev/catalogs/opm@v4` and `opmodel.dev/catalogs/k8s@v1` are generated with a closure pinning both
- **THEN** `platform.cue` imports both under distinct aliases and its `#registry` carries both keys with `enable: true`, the stamped versions and `#catalog` bound to the matching alias

#### Scenario: Disabled entry still imports its catalog

- **WHEN** an entry is generated with `enable` false
- **THEN** the catalog is still imported and the entry carries `enable: false`; the dependency list still pins it

#### Scenario: Invalid input is refused

- **WHEN** the input has an empty name or type, a duplicate registry path, an entry or dependency with an empty path or version, a dependency pinned twice at different versions, or a dependency list missing core or an entry's catalog
- **THEN** generation returns an error naming the offending field or path and emits no files

### Requirement: The core pin defaults to the kernel's verified release

The generated `cue.mod/module.cue` SHALL pin `opmodel.dev/core@v2` at the version carried by the kernel's default schema module identifier. The dependency roots `Roots` derives SHALL be that core pin plus every entry's catalog, disabled entries included, with bare SemVer canonicalised to the `v`-prefixed form `cue.mod` requires. `Roots` SHALL take no option: a caller that needs another core build assembles its `[]Dep` roots directly and hands them to `Closure` and `Generate`, which pin whatever they are given.

#### Scenario: Default core pin

- **WHEN** roots are derived from entries with `Roots`
- **THEN** the roots contain `opmodel.dev/core@v2` at the version of `schema.DefaultSchemaModule` and one root per entry

#### Scenario: Explicit core pin

- **WHEN** the caller assembles roots pinning core at another version and passes them as `Input.Deps`
- **THEN** the generated module file reflects that version
- **AND** no `WithCoreVersion` option or `RootOption` type exists in the package

### Requirement: The dependency closure is derived from published module files

The helper SHALL derive the full dependency list from the roots by a breadth-first walk over each reachable module version's published `cue.mod/module.cue`, obtained through a caller-supplied module-file source, selecting the maximum version per major-qualified path with the roots participating in the maximum and skipping local replacements. The result SHALL be the tidied list a `cue mod tidy` would write for the same roots (0019 D13: tidying happens once, at generation), minus the prune of modules no import reaches. A root or transitive requirement naming an unpublished build SHALL fail with an error naming the module path and version. The walk SHALL honour context cancellation.

#### Scenario: Transitive dependency pinned

- **WHEN** the closure is derived for a catalog whose published module file requires `cue.dev/x/k8s.io@v0`
- **THEN** the result pins `cue.dev/x/k8s.io@v0` at the maximum version any reachable module file names, alongside the roots

#### Scenario: Root wins a shared path

- **WHEN** a root pins a path at a version higher than any transitive requirement names
- **THEN** the root's version is selected

#### Scenario: Unpublished build refused

- **WHEN** a root or a transitive requirement names a version the source cannot resolve
- **THEN** the closure returns an error naming that module path and version and no partial list

### Requirement: Registry access is caller-configured

The module-file source the closure reads through SHALL be constructed from caller-supplied configuration: the CUE registry mapping, the client type reported to registries, and the process environment the CUE module cache location is read from. The helper SHALL NOT read the process environment or hard-code a client type itself; a caller that passes a nil environment selects the CUE module system's own default (the current process environment) explicitly, and that choice is the caller's, never the helper's. Tests SHALL be able to supply a fixture module-file graph without any registry.

#### Scenario: Fixture graph without a registry

- **WHEN** a test supplies an in-memory module-file source
- **THEN** the closure derives the list from it with no network or cache access

#### Scenario: Registry source carries the caller's configuration

- **WHEN** a frontend constructs the registry-backed source with its registry mapping, client type and environment
- **THEN** module files resolve through that mapping and cache, and registry requests report that client type

### Requirement: The emitted module builds through the kernel

A module generated from a closure derived against a registry serving core and a catalog SHALL build through `AcquirePlatformFromDir` with no further tidy: the platform's shape gate passes, every `#registry` entry's derived `version` equals the stamped version, and the catalog's transitive dependency is pinned in the module file.

#### Scenario: End-to-end against the in-process registry

- **WHEN** the test registry serves core and a catalog, the closure is derived for one entry and the generated files are written to a directory
- **THEN** `AcquirePlatformFromDir` on that directory returns a source-carrying platform whose `#registry` entry reads the stamped version, and the module file pins the catalog's transitive dependency

### Requirement: Files are written into a caller-owned directory

The helper SHALL offer a write operation that places the generated files under a caller-supplied directory, creating parent directories, refusing any file path that escapes the directory. It SHALL own no directory lifecycle: no generation naming, staging swap, retention or reset; those remain frontend policy.

#### Scenario: Write into an empty directory

- **WHEN** the generated files are written into an empty directory
- **THEN** `cue.mod/module.cue` and `platform.cue` exist under it with the generated bytes

#### Scenario: Path escape refused

- **WHEN** a file name would resolve outside the target directory
- **THEN** the write fails naming the file and writes nothing
