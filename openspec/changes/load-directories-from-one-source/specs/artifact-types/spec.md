## ADDED Requirements

### Requirement: Directory acquisition reads its tree once and builds from that read

Every directory acquire verb (`AcquireModuleFromDir`, `AcquireCatalogFromDir`,
`AcquirePlatformFromDir`, `AcquireInstanceFromDir`) SHALL check the directory before reading
anything from it. A path that does not exist SHALL fail with `accessing <label> directory
"<absolute path>"` wrapping the stat error (so it matches `fs.ErrNotExist`), and a path that
is not a directory SHALL fail with `<label> path "<absolute path>" is not a directory`, where
`<label>` is `module`, `catalog`, `platform` or `instance`. A verb that stamps its `Source` in
overlay mode (`AcquireModuleFromDir`, `AcquireCatalogFromDir`, and `AcquireInstanceFromDir`
with values sources) SHALL read the `.cue` files under the module root once, before the
build, and SHALL build the package from that overlay, so `Package` and `Source.Overlay` come
from the same read. When the layered instance build fails, the kernel SHALL attribute the
failure to the values sources by building the authored package from the overlay already read,
without reading the directory again. `AcquirePlatformFromDir` and `AcquireInstanceFromDir`
without values sources SHALL keep building from disk and stamping on-disk mode (`Overlay`
nil). A package that embeds a non-CUE file stored beside it SHALL acquire from a directory as
it does from disk, because files outside the overlay are read from the directory.

#### Scenario: A missing directory fails before any tree read

- **WHEN** a caller invokes any of the four directory acquire verbs on a path that does not exist
- **THEN** the error reads `accessing <label> directory "<absolute path>": ...` with the verb's label and matches `fs.ErrNotExist`
- **AND** it does not read `reading module tree`, and no artifact is returned

#### Scenario: A file path is refused as not a directory

- **WHEN** a caller invokes any of the four directory acquire verbs on the path of a regular file
- **THEN** the error reads `<label> path "<absolute path>" is not a directory`, and no artifact is returned

#### Scenario: Module and catalog are built from the overlay they carry

- **WHEN** a caller acquires a module with `AcquireModuleFromDir` or a catalog with `AcquireCatalogFromDir`
- **THEN** the package is built from the `.cue` bytes held in the returned `Source.Overlay`, read once before the build
- **AND** the directory is not walked a second time to stamp the `Source`

#### Scenario: Platform and authored instance stay on-disk

- **WHEN** a caller acquires a platform with `AcquirePlatformFromDir`, or an instance with `AcquireInstanceFromDir` and no values sources
- **THEN** the returned `Source` has a nil `Overlay`, `Root` the enclosing module root and `Pkg` the package directory relative to it

#### Scenario: Values attribution reuses the authored read

- **WHEN** an instance acquired with values sources fails to build because a source conflicts with the package's values or the module's `#config`
- **THEN** the error is attributed to the source, as before
- **AND** the authored package used for the attribution is built from the overlay read for the layered build, without the rendered values file and without a further read of the directory

#### Scenario: A module embedding a non-CUE file acquires from a directory

- **WHEN** a module package uses the embed attribute to embed a JSON file stored beside it, and a caller invokes `AcquireModuleFromDir` on its directory
- **THEN** acquisition succeeds and `Package` carries the embedded data
- **AND** `Source.Overlay` still holds only the `.cue` files under the module root
