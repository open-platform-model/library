## MODIFIED Requirements

### Requirement: Registry access is caller-configured
The module-file source the closure reads through SHALL be constructed from caller-supplied configuration: the CUE registry mapping, the client type reported to registries, and the process environment the CUE module cache location is read from. Tests SHALL be able to supply a fixture module-file graph without any registry.

#### Scenario: Fixture graph without a registry

- **WHEN** a test supplies an in-memory module-file source
- **THEN** the closure derives the list from it with no network or cache access

#### Scenario: Registry source carries the caller's configuration

- **WHEN** a frontend constructs the registry-backed source with its registry mapping, client type and environment
- **THEN** module files resolve through that mapping and cache, and registry requests report that client type

## REMOVED Requirements

### Requirement: Generation is pure and deterministic
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Generation renders a platform module's two files from typed input only", "Generation performs no I/O and reads no environment", "Entries and dependencies are emitted in sorted path order", "platform.cue embeds core.#Platform, declares the name and type and imports each entry's catalog under a positional alias", "platform.cue writes one registry entry per subscription" and "cue.mod/module.cue declares the caller's module path, the language version and every dependency".

### Requirement: The core pin defaults to the kernel's verified release
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The generated module file pins core at the kernel's default schema module version and Roots derives that pin plus every entry's catalog" and "Roots takes no option and a caller assembles another core pin directly".

### Requirement: The dependency closure is derived from published module files
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The helper derives the dependency list by a breadth-first walk over published module files and the walk honours context cancellation", "The result is the tidied list a cue mod tidy would write for the same roots" and "A requirement naming an unpublished build fails with the module path and version".

## ADDED Requirements

### Requirement: Generation renders a platform module's two files from typed input only
The helper SHALL render a platform module's two files (`cue.mod/module.cue`, `platform.cue`) from typed input only: the platform's name and type, its registry entries (major-qualified catalog module path, the catalog build as a SemVer string, enabled flag), the module path the generated module declares, and the resolved dependency list.

#### Scenario: Invalid input is refused

- **WHEN** the input has an empty name or type, a duplicate registry path, an entry or dependency with an empty path or version, a dependency pinned twice at different versions, or a dependency list missing core or an entry's catalog
- **THEN** generation returns an error naming the offending field or path and emits no files

### Requirement: Generation performs no I/O and reads no environment
The helper SHALL perform no I/O and read no environment.

#### Scenario: The helper does no I/O

- **WHEN** the helper renders the two files
- **THEN** it performs no I/O and reads no environment

### Requirement: Entries and dependencies are emitted in sorted path order
Entries and dependencies SHALL be emitted in sorted path order whatever order they arrive in, so identical input yields byte-identical files.

#### Scenario: Same input, same bytes

- **WHEN** the generator is invoked twice with the same entries and dependencies in different slice orders
- **THEN** both results are byte-identical, entries and dependencies sorted by path

### Requirement: platform.cue embeds core.#Platform, declares the name and type and imports each entry's catalog under a positional alias
`platform.cue` SHALL embed `core.#Platform`, declare `metadata.name` and `type`, import each entry's catalog under a positional alias.

#### Scenario: Disabled entry still imports its catalog

- **WHEN** an entry is generated with `enable` false
- **THEN** the catalog is still imported and the entry carries `enable: false`; the dependency list still pins it

### Requirement: platform.cue writes one registry entry per subscription
`platform.cue` SHALL write one `#registry` entry per subscription carrying `enable`, the entry's expected `version` (the enhancement 0019 ("The render module's `cue.mod` is derived by promotion, never computed, and a render refuses when derivation cannot cover a path") tripwire: it unifies with core's readout of the imported catalog) and `#catalog` bound to the import.

#### Scenario: Two catalogs

- **WHEN** two enabled entries for `opmodel.dev/catalogs/opm@v4` and `example.com/catalogs/extra@v1` are generated with a closure pinning both
- **THEN** `platform.cue` imports both under distinct aliases and its `#registry` carries both keys with `enable: true`, the stamped versions and `#catalog` bound to the matching alias

### Requirement: cue.mod/module.cue declares the caller's module path, the language version and every dependency
`cue.mod/module.cue` SHALL declare the caller's module path, the language version `v0.17.0`, and every dependency without a default-major marker.

#### Scenario: The module file declares the module path, the language version and every dependency

- **WHEN** `cue.mod/module.cue` is generated
- **THEN** it declares the caller's module path, the language version `v0.17.0`, and every dependency without a default-major marker

### Requirement: The generated module file pins core at the kernel's default schema module version and Roots derives that pin plus every entry's catalog
The generated `cue.mod/module.cue` SHALL pin `opmodel.dev/core@v2` at the version carried by the kernel's default schema module identifier. The dependency roots `Roots` derives SHALL be that core pin plus every entry's catalog, disabled entries included, with bare SemVer canonicalised to the `v`-prefixed form `cue.mod` requires.

#### Scenario: Default core pin

- **WHEN** roots are derived from entries with `Roots`
- **THEN** the roots contain `opmodel.dev/core@v2` at the version of `schema.DefaultSchemaModule` and one root per entry

### Requirement: Roots takes no option and a caller assembles another core pin directly
`Roots` SHALL take no option: a caller that needs another core build assembles its `[]Dep` roots directly and hands them to `Closure` and `Generate`, which pin whatever they are given.

#### Scenario: Explicit core pin

- **WHEN** the caller assembles roots pinning core at another version and passes them as `Input.Deps`
- **THEN** the generated module file reflects that version
- **AND** no `WithCoreVersion` option or `RootOption` type exists in the package

### Requirement: The helper derives the dependency list by a breadth-first walk over published module files and the walk honours context cancellation
The helper SHALL derive the full dependency list from the roots by a breadth-first walk over each reachable module version's published `cue.mod/module.cue`, obtained through a caller-supplied module-file source, selecting the maximum version per major-qualified path with the roots participating in the maximum and skipping local replacements. The walk SHALL honour context cancellation.

#### Scenario: Transitive dependency pinned

- **WHEN** the closure is derived for a catalog whose published module file requires `cue.dev/x/k8s.io@v0`
- **THEN** the result pins `cue.dev/x/k8s.io@v0` at the maximum version any reachable module file names, alongside the roots

#### Scenario: Root wins a shared path

- **WHEN** a root pins a path at a version higher than any transitive requirement names
- **THEN** the root's version is selected

### Requirement: The result is the tidied list a cue mod tidy would write for the same roots
The result (Requirement: The helper derives the dependency list by a breadth-first walk over published module files and the walk honours context cancellation) SHALL be the tidied list a `cue mod tidy` would write for the same roots (enhancement 0019 ("The render module's `cue.mod` is derived by promotion, never computed, and a render refuses when derivation cannot cover a path"): tidying happens once, at generation), minus the prune of modules no import reaches.

#### Scenario: The result is the list a cue mod tidy would write

- **WHEN** the closure is derived for the same roots
- **THEN** the result is the tidied list a `cue mod tidy` would write, minus the prune of modules no import reaches

### Requirement: A requirement naming an unpublished build fails with the module path and version
A root or transitive requirement naming an unpublished build SHALL fail with an error naming the module path and version.

#### Scenario: Unpublished build refused

- **WHEN** a root or a transitive requirement names a version the source cannot resolve
- **THEN** the closure returns an error naming that module path and version and no partial list

### Requirement: The helper reads no process environment and hard-codes no client type, and a nil environment is the caller's choice
The helper SHALL NOT read the process environment or hard-code a client type itself; a caller that passes a nil environment selects the CUE module system's own default (the current process environment) explicitly, and that choice is the caller's, never the helper's.

#### Scenario: A nil environment is the caller's choice

- **WHEN** a caller passes a nil environment
- **THEN** the CUE module system's own default (the current process environment) is selected explicitly, and that choice is the caller's, never the helper's
