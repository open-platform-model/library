## MODIFIED Requirements

### Requirement: The catalog's committed dependency requirements are readable

The catalog artifact SHALL expose the requirements recorded in its own `cue.mod/module.cue`, as module path to version, so a consumer can compare them against another resolution without re-reading the artifact or reaching for CUE itself (enhancement 0015 D8, the committed-resolution comparison 0019 D18 defines). The library SHALL expose the requirements only; comparing them, and deciding what a mismatch means, belong to the consumer. The module file SHALL be read and parsed by the same reader the render stage uses for its input modules, in either source mode, so a module file the render stage refuses (a dependency carrying `replaceWith`, an empty module path, a file that does not parse) is refused here too, with an error naming the catalog's `cue.mod/module.cue`. A dependency a local replacement serves may carry no version and maps to the empty string.

#### Scenario: Declared requirements are readable off an acquired catalog

- **WHEN** a catalog whose `cue.mod/module.cue` requires a core version and a catalog version is acquired
- **THEN** both requirements are readable from the returned artifact, each as a path and a version
- **AND** the library reports no verdict about whether either is acceptable

#### Scenario: A module file the render stage would refuse is refused

- **WHEN** an acquired catalog's `cue.mod/module.cue` lists a dependency that carries `replaceWith`
- **THEN** reading its requirements fails with an error naming the catalog's `cue.mod/module.cue`, and no requirements are returned
