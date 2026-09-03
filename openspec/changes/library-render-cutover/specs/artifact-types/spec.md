## ADDED Requirements

### Requirement: Internal call sites use schema paths

Every kernel-internal Go call site that reads a sub-value of an artifact's `Package` SHALL read it through the `opm/schema` path variables (`schema-dispatch`, "Path inventory exposed as package-level vars"), never through a removed struct field or an ad-hoc path literal. The render path reads no artifact sub-value in Go: the instance and the platform enter the render build by import, and the generated glue reads `components` and `#composedTransformers` in CUE.

#### Scenario: Render build reads artifacts by import

- **WHEN** `Kernel.Render` runs
- **THEN** no Go code looks up the instance's components or the platform's transformers by path; the staged render module imports both packages and the glue reads them

#### Scenario: Instance processing uses schema paths

- **WHEN** `Kernel.ProcessModuleInstance` reads a module's `#config` schema or fills validated values
- **THEN** the reads and the fill go through `schema.Config` and `schema.Values`
- **AND** there is no direct dereference of a removed field

## REMOVED Requirements

### Requirement: Internal Call Sites Use Binding Paths

**Reason**: binding-era text (`binding.Paths()`, the `opm/compile/` and `opm/validate/` pipelines) whose referents are all gone: bindings were retired by `schema-dispatch`, `opm/validate/` by the kernel consolidation, and `opm/compile/` by this change. Restated as "Internal call sites use schema paths".

**Migration**: none.
