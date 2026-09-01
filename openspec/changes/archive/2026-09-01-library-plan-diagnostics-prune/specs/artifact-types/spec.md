# artifact-types Delta

## MODIFIED Requirements

### Requirement: Instance-Side Module Paths On Binding

The `api.Paths` inventory SHALL expose two CUE paths for instance-side lookup of the embedded source module:

- `Paths.Module` — the path under which the instance's CUE package carries its `#Module` reference (v1alpha2: `#module`).
- `Paths.ModuleMetadata` — the path under which the instance's CUE package carries the projected module metadata (v1alpha2: `#moduleMetadata`).

These paths SHALL be populated by every concrete binding so that kernel-internal call sites and `*Instance` accessor methods can read module identity from `Instance.Package` without ad-hoc path strings. The accessor set reading `Paths.ModuleMetadata` SHALL be `ModuleVersion()` alone; `Instance` SHALL NOT expose a `ModuleFQN()` accessor (no production code or consumer read it, and the transformer context carries the instance's own FQN, not the source module's).

#### Scenario: Instance reaches its source module via the binding

- **WHEN** a caller holds a `*Instance` whose `Package` carries a `#module` field
- **THEN** `rel.Package.LookupPath(b.Paths().Module).Exists()` is true (where `b` is the binding for `rel.APIVersion`)

#### Scenario: Instance accessors read module metadata via the binding

- **WHEN** a caller invokes `rel.ModuleVersion()`
- **THEN** the returned value is read from `rel.Package.LookupPath(b.Paths().ModuleMetadata)` via `api.Lookup(rel.APIVersion)`
- **AND** there is no cached `*Module` field on `Instance` carrying the same data

#### Scenario: ModuleFQN accessor removed

- **WHEN** a developer searches `opm/module` for `ModuleFQN`
- **THEN** no such accessor exists on `Instance`; the source module's identity remains reachable through `Instance.Package` at the module-metadata path
