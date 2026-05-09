## ADDED Requirements

### Requirement: Release Config Schema Accessor

`*module.Release` SHALL expose a `ConfigSchema() cue.Value` accessor that returns the embedded source module's `#config` schema. The accessor SHALL look up the schema via the binding registered for `r.APIVersion` using `Paths().Module` followed by `Paths().Config`. The accessor SHALL return the zero `cue.Value` (not an error) when the binding is unregistered, the receiver is `nil`, or the path does not exist.

#### Scenario: Schema reachable on a well-formed release

- **WHEN** a caller invokes `rel.ConfigSchema()` on a `*Release` whose `Package` carries an embedded `#module` with a `#config` definition
- **THEN** the returned `cue.Value` exists (`v.Exists() == true`)
- **AND** the returned value is identical to `rel.Package.LookupPath(b.Paths().Module).LookupPath(b.Paths().Config)` where `b` is the binding for `rel.APIVersion`

#### Scenario: Zero value on unregistered binding

- **WHEN** a caller invokes `rel.ConfigSchema()` on a `*Release` whose `APIVersion` has no registered binding
- **THEN** the returned `cue.Value` is the zero value (`v.Exists() == false`)
- **AND** no error is returned

#### Scenario: Zero value on missing #config path

- **WHEN** a caller invokes `rel.ConfigSchema()` on a `*Release` whose embedded `#module` does not declare a `#config` definition
- **THEN** the returned `cue.Value` is the zero value (`v.Exists() == false`)

#### Scenario: Nil receiver safety

- **WHEN** a caller invokes `(*Release)(nil).ConfigSchema()`
- **THEN** the returned `cue.Value` is the zero value
- **AND** no panic occurs
