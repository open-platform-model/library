## MODIFIED Requirements

### Requirement: In-Memory Load Without a Temporary Directory

The registry module loader SHALL load the fetched module in memory and SHALL NOT write the module's source to a temporary directory. It SHALL inject the fetched module's CUE files (every `.cue` file under the module root, the module's own `cue.mod/module.cue` included, and nothing else) via `load.Config.Overlay` under a deterministic synthetic root, leaving `load.Config.FS` nil so the module's transitive dependencies resolve through the registry and CUE module cache. The staged overlay the loader returns on the module's `Source` SHALL be that same set of files. The synthetic root SHALL be an absolute path on every operating system (on Windows it carries the current volume) that does not exist on disk. Because cue/load reads a real directory beneath an overlay root, the loader SHALL refuse the acquire, with an error naming the path and before anything is built, when anything (a directory, a file or a symlink) exists at the synthetic root. The same holds for every artifact kind the loader fetches from a registry, a catalog included.

#### Scenario: No temporary directory created

- **WHEN** a module is loaded from the registry
- **THEN** no temporary directory is created or left behind for the module's source
- **AND** the module's transitive dependencies still resolve

#### Scenario: Staged overlay carries the module's CUE files only

- **WHEN** a fetched module's archive contains `.cue` files, a `cue.mod/module.cue`, and non-CUE files such as a license or a readme
- **THEN** the staged overlay on `Module.Source` holds every `.cue` file and `cue.mod/module.cue`, each keyed under the synthetic root
- **AND** the non-CUE files are not present in the overlay
- **AND** a fetch that stages no `.cue` file fails the load with an error wrapping `ErrInvalidPackage` (a defensive branch: a module zip CUE's fetch accepts always carries `cue.mod/module.cue`, so the empty case is exercised at the walker, not through a registry)

#### Scenario: An existing synthetic root is refused

- **WHEN** a directory holding a `.cue` file of the module's package, or a plain file, exists at the synthetic root of a module version and that version is fetched from the registry
- **THEN** the load returns an error naming the synthetic root, and no value and no source
- **AND** once nothing exists at the root, the same fetch succeeds

#### Scenario: An existing synthetic root refuses a catalog acquire

- **WHEN** a directory holding a `.cue` file, or a plain file, exists at the synthetic root of a catalog version and that version is fetched from the registry with the catalog spec
- **THEN** the load returns an error naming the synthetic root, and no value and no source
- **AND** once nothing exists at the root, the same fetch succeeds
