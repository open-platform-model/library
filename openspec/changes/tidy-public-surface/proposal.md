## Why

The library goes to v1.0.0 soon. From then on every exported name under `opm/` is a contract, and a removal needs a new major and a new module path. The surface still holds a deprecated duplicate package, one error type whose receiver differs from all others, a validation failure a caller cannot test for by type, types under two names, and path variables and a constant that only the library and its tests read.

## What Changes

SemVer class: breaking, on the beta line (CONSTITUTION VI, pre-GA clause; ADR-010). It lands as `feat!` with a `BREAKING CHANGE:` footer, which is the migration note. The PR title is `feat!: tidy the public surface before v1.0.0`.

- **BREAKING** Remove `opm/helper/objectset` (five `Deprecated:` symbols). `opm/k8s/object` is the one home of the duplicate-identity check.
- **BREAKING** `errors.IdentityError` gets a pointer receiver, as every other typed error has. The library returns `*IdentityError`.
- Add `errors.ConfigValidationError`: `Kernel.ValidateConfigDetailed` returns it for values that fail the schema. It wraps the CUE error tree unchanged, so `cueerrors.Errors`, `Positions` and `Print` behave as before on a wrapped error. **BREAKING** only for a caller that type-asserts the returned error to `cueerrors.Error` without `errors.As`.
- **BREAKING** One name per type. The metadata types are declared in the package of their artifact (`module.ModuleMetadata`, `module.InstanceMetadata`, `platform.PlatformMetadata`, `catalog.CatalogMetadata`); `opm/schema` no longer declares them. `module.Source` is the one name of the staged source; the aliases `platform.Source` and `catalog.Source` go.
- **BREAKING** `opm/schema` exports three path variables (`Metadata`, `Module`, `CatalogProvides`), the ones a consumer reads. The other twelve move to `opm/internal/corepath`. The constant `CollisionsSince`, which no code reads, is deleted.

Kept on purpose: `cue.Value` in public structs (owner decision for v1); `schema.ProvidedBySince` and `schema.ProvidesSince` (the library's own packages and both consumers' tests read them); `DefaultSchemaModule`; `opm/errors/classify.go`; the `opm/k8s` tier.

### Downstream migration cost (cli a12373a5, opm-operator 7474778, both `origin/main`)

| Removal | cli | opm-operator |
| --- | --- | --- |
| `opm/helper/objectset` | none | none |
| `IdentityError` value receiver | none | `errors.AsType[oerrors.IdentityError]` becomes `errors.AsType[*oerrors.IdentityError]` at `internal/reconcile/resolution.go:27` and `internal/render/acquire_error_test.go:30`; four test literals take `&` (`moduleinstance_reconcile_test.go:2330`, `resolution_test.go:27`, `acquire_error_test.go:17`, `test/integration/reconcile/suite_test.go:206`) |
| `ConfigValidationError` | none (three call sites wrap with `%w`) | none (one call site wraps) |
| `schema.*Metadata` | none (uses `module.*Metadata` only) | none |
| `platform.Source`, `catalog.Source` | none | `catalog.Source` becomes `module.Source` at `internal/controller/transformerregistration_catalog_test.go:92,94` |
| twelve path variables, `CollisionsSince` | none | none |

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `duplicate-object-identities`: the requirement that keeps the deprecated helper copy is removed.
- `helper-packages`: the helper tier lists one subpackage.
- `config-validation`: one typed marker for a validation failure; every typed error matches through a pointer target.
- `schema-dispatch`: the path inventory is split between three exported paths and an internal package; metadata types live with their artifacts.
- `artifact-types`: accessor definitions name the internal paths.
- `platform-artifact`: `Platform.Source` is a `*module.Source`; no alias.

## Impact

- Packages: `opm/helper/objectset` (deleted), `opm/errors`, `opm/kernel`, `opm/module`, `opm/platform`, `opm/catalog`, `opm/schema`, `opm/internal/loader`, new `opm/internal/corepath`, `opm/k8s/object` (one parity test deleted).
- Docs: `AGENTS.md`, `README.md`, `CONSTITUTION.md`, `docs-kit.cue`, `.cascade-frozen`, package docs, ADR-015.
- Consumers: opm-operator needs the edits in the table when it takes the release. cli needs none.
- No dependency change, no core pin change, no kernel behaviour change.
