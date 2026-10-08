## Context

See proposal.md for the motivation. Constraints: the base tag is `v1.0.0-beta.7`, a prerelease, so `task api:diff` warns and passes, and the migration note is the `BREAKING CHANGE:` footer (ADR-010). `opm/errors/classify.go` and its behaviour are out of scope. The main spec `config-validation` forbade any Go type around a CUE validation error; this change replaces that rule with "one marker, no projection".

## Goals / Non-Goals

**Goals:** fewer exported names; one receiver shape; one typed check for a validation failure; one name per type; a migration note per break.

**Non-Goals:** hiding `cue.Value`; `DefaultSchemaModule` as a function; new fetch sentinels; any change to `Classify`; the breadth or stability of `opm/k8s`; the acquire and synthesis values checks (see Open Questions).

## Research & Decisions

### Receiver shape of typed errors

**Context**: `IdentityError` has a value receiver; all eleven other types with an `Error` method in `opm/errors`, and `RenderError`, `ExportError` and `DuplicateIdentitiesError` outside it, use a pointer receiver.
**Explored**: `git grep ") Error() string"` over `opm/`; consumer uses with `git -C cli grep` and `git -C opm-operator grep` at `origin/main`.
**Decision**: pointer receivers everywhere. The loader returns `&oerrors.IdentityError{...}`.
**Rationale**: the majority shape, and with a pointer-only method set the value type no longer implements `error`, so a stale value-typed form is loud in place of missing silently: `errors.AsType[IdentityError]` fails to compile, and `errors.As(err, &v)` with a value-typed `v` is a `go vet` error and a run-time panic on every error that reaches it. Alternative: value receivers for all. Rejected: fourteen types change and both consumers break widely. Alternative: keep both method sets by adding nothing. Rejected: a value receiver puts the method in both method sets, so two `errors.As` targets work and callers diverge.

### Typed validation failure

**Context**: `ValidateConfigDetailed` returns the raw CUE tree. A caller cannot tell "values do not satisfy the schema" from another failure without walking CUE errors.
**Explored**: `cueerrors.Errors` and `cueerrors.Positions` (cuelang.org/go v0.17.1, `cue/errors/errors.go`) both use `errors.As`, so a wrapper with `Unwrap` keeps the tree reachable. All four consumer call sites already wrap the error with `%w`.
**Decision**: `type ConfigValidationError struct{ Err error }` in `opm/errors`, pointer receiver, `Error()` returns `Err.Error()`, `Unwrap()` returns `Err`. `validateValues` stays as it is; `ValidateConfigDetailed` wraps only the result of the validation step.

```go
func (k *Kernel) ValidateConfigDetailed(schema cue.Value, sources []Source) (cue.Value, error) {
    values, err := compileSources(...) // returned unwrapped
    ...
    v, err := validateCompiled(schema, values, true)
    if err != nil {
        return cue.Value{}, &oerrors.ConfigValidationError{Err: err}
    }
    return v, nil
}
```

**Rationale**: the type MUST NOT implement `cueerrors.Error` itself: `cueerrors.Errors` would then return the wrapper as one error for a single-error tree. The text is unchanged, so no frontend output moves. A compile or load failure of a source is not marked: a file-backed source can fail on a registry, which `Classify` owns. Alternatives: a sentinel `ErrConfigInvalid` joined to the tree (rejected: `errors.Join` changes the text and hides the tree from `errors.As` order guarantees); a marker interface (rejected: a second idiom beside `errors.As`). The name avoids the eight names the spec bans.

### One name per type

**Context**: four metadata types are declared in `opm/schema` and aliased in the artifact packages; `Source` is declared in `opm/module` and aliased in `opm/platform` and `opm/catalog`.
**Explored**: cli uses `module.ModuleMetadata` and `module.InstanceMetadata` at about sixty sites and never the `schema` names; opm-operator uses neither. `opm/schema` itself uses none of the four types. The decoders already live in the artifact packages.
**Decision**: move each declaration to its artifact package and delete it from `opm/schema`. Keep `module.Source`; delete the two aliases.
**Rationale**: zero consumer edits for the metadata types, and the type sits beside the struct field it types and the decoder that fills it. The alternative (keep the `schema` names) costs cli about sixty edits and keeps a type in a package that does not use it. For `Source`, `module.Source` is the declaration; `kernel.Source` is another type (a values source) and is untouched.

### Path table and constants

**Context**: fifteen exported `cue.Path` variables (assignable by any importer) and three constants.
**Explored**: consumer uses at `origin/main`: opm-operator reads `schema.Module` and `schema.Metadata` (`internal/render/version.go:27`) and `schema.CatalogProvides` and `schema.ProvidesSince` (`test/integration/reconcile/backup_fixture_test.go`); cli reads `schema.ProvidedBySince` (`internal/cmd/platform/check_test.go:800`, `tests/e2e/instance_build_test.go:279`). `CollisionsSince` has no reader in Go, only two doc links.
**Decision**: a new internal package `opm/internal/corepath` declares the twelve unread-by-consumer paths. `opm/schema` keeps `Metadata`, `Module`, `CatalogProvides`, `ProvidedBySince`, `ProvidesSince`. `CollisionsSince` is deleted; the two doc comments name the release.
**Rationale**: each path keeps exactly one declaration. The split costs one more import in five packages. The alternative, all fifteen internal with three re-exported, gives three values two names. Unexporting inside `opm/schema` is not possible: `opm/module`, `opm/platform`, `opm/catalog` and `opm/kernel` read them.

## Risks / Trade-offs

- [opm-operator `main` stops compiling against this library] → expected and listed; the non-required `Consumer build (opm-operator)` job turns red. The edits are in proposal.md; another task makes them with the pin bump.
- [A caller outside the workspace keeps `errors.As` with a value-typed `IdentityError` target: it compiles and panics at run time] → the footer names the panic and `go vet`.
- [A caller outside the workspace type-asserts the validation error] → the footer says to use `errors.As` or `cueerrors.Errors`.
- [The squash commit loses the footer] → the PR body carries the footer text; the reviewer checks the squash message (ADR-010).
- [Reversibility] → two-way until v1.0.0 is tagged; after that each restored name is additive, each further removal is a major.

## Migration Plan

One PR. Consumers migrate when they bump the library pin. Rollback: revert the squash commit; nothing persists outside Go source.

## Amendment (2026-10-08, after the first review of library#223)

**Context**: `Consumer build (opm-operator)` was red: the operator's `main` builds `oerrors.IdentityError{}` values as errors, matches with `errors.AsType[oerrors.IdentityError]`, and names `catalog.Source`.
**Decision**: `IdentityError` keeps the value receiver for now. The loader still returns `&IdentityError{}`. A method `As(target any) bool` on the value receiver fills a value target from a pointer in the chain and a pointer target from a value in the chain, so the old and the new form both match, with `errors.As` and with `errors.AsType`. `catalog.Source` returns as a Deprecated alias.
**Rationale**: the pointer and the value receiver cannot coexist on one type, so the receiver flip must wait. Returning the pointer now, with the bridge, lets the operator move to the pointer forms before the flip; without it the flip and the operator's edit would again have to land together. Alternative: return the value again and change nothing. Rejected: the pointer target would not match until the flip, so no consumer could migrate ahead of it.
**Follow-up**: plan task T9.24 flips the receiver and removes `As` and `catalog.Source` once the operator has migrated. The tests `TestTypedErrorsUsePointerReceivers` and `TestSurface_OneNamePerType` each hold a one-entry exception list that the follow-up empties.

The two open questions below are answered in part by the owner (2026-10-08, library#223): the four `opm/schema` names stay public for v1. The first question stays open.

## Open Questions

- Should the values checks inside `AcquireInstanceFromDir` and `SynthesizeInstance` also return `*ConfigValidationError`? Not in the brief; additive later.
- Should `Module`, `Metadata`, `CatalogProvides` and `ProvidesSince` stay public for good, or should opm-operator get accessors and drop them? Owner decision; additive or a later break.
