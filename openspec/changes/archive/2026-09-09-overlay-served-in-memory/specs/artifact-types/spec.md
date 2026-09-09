## MODIFIED Requirements

### Requirement: An overlay source can be written to a directory

`module.Source` SHALL expose `WriteTo(dir string) ([]string, error)`, which writes every entry of an overlay-mode source under `dir` at the entry's path relative to `Root`, creating parent directories as needed, and returns the dir-relative paths it wrote, sorted. The method SHALL validate before it writes: a nil receiver, an on-disk source (`Overlay` nil, since `Root` already is the directory) and any entry whose path is not under `Root` SHALL be refused with a plain error and nothing written. `WriteTo` SHALL be the library's one overlay writer, for frontends that need a fetched tree on disk; the kernel's render stage SHALL serve an overlay-mode input from memory and SHALL NOT write it.

#### Scenario: A fetched module is written to disk

- **WHEN** a frontend acquires a module from a registry and calls `Source.WriteTo(dest)` on it
- **THEN** every file of the fetched module is written under `dest` at its module-relative path, the returned list names each written path relative to `dest` in sorted order, and no registry fetch happens

#### Scenario: On-disk source refused

- **WHEN** `WriteTo` is called on a source with a nil `Overlay`
- **THEN** it returns an error and writes nothing

#### Scenario: Entry outside the root refused

- **WHEN** an overlay entry's absolute path is not under `Root`
- **THEN** `WriteTo` returns an error naming the entry and writes nothing, including entries that were valid

#### Scenario: Frontends stop copying fetched modules themselves

- **WHEN** a frontend scaffolds a new module from a published template
- **THEN** it writes the acquired module's `Source` into place with `WriteTo`
- **AND** it performs no second registry fetch and no filesystem walk of its own

#### Scenario: The render stage writes nothing for an overlay input

- **WHEN** `Kernel.Render` stages an overlay-mode instance or platform
- **THEN** `WriteTo` is not called and no entry of the overlay appears on disk
