## MODIFIED Requirements

### Requirement: The core pin defaults to the kernel's verified release

The generated `cue.mod/module.cue` SHALL pin `opmodel.dev/core@v2` at the version carried by the kernel's default schema module identifier. The dependency roots `Roots` derives SHALL be that core pin plus every entry's catalog, disabled entries included, with bare SemVer canonicalised to the `v`-prefixed form `cue.mod` requires. `Roots` SHALL take no option: a caller that needs another core build assembles its `[]Dep` roots directly and hands them to `Closure` and `Generate`, which pin whatever they are given.

#### Scenario: Default core pin

- **WHEN** roots are derived from entries with `Roots`
- **THEN** the roots contain `opmodel.dev/core@v2` at the version of `schema.DefaultSchemaModule` and one root per entry

#### Scenario: Explicit core pin

- **WHEN** the caller assembles roots pinning core at another version and passes them as `Input.Deps`
- **THEN** the generated module file reflects that version
- **AND** no `WithCoreVersion` option or `RootOption` type exists in the package
