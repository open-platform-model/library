# ADR-015: One shape for the public surface before v1.0.0

## Status

Accepted (2026-10-08) on these two points, decided by the owner on library#223: the earlier rule "No Custom Validation Error Types" is reversed to one marker type with the CUE tree unchanged inside, and `schema.Module`, `schema.Metadata`, `schema.CatalogProvides` and `schema.ProvidesSince` stay public for v1. The rest was written with the change `tidy-public-surface` and is accepted with its merge. Reversibility: two-way until `v1.0.0` is tagged, one-way after it (a removal then needs a new major and a new module path).

## Context

From `v1.0.0` every exported name under `opm/` is a contract (CONSTITUTION VI), and the module path has no major suffix, so a later removal is a path change for every consumer. Four things on the surface had no rule behind them. `IdentityError` had a value receiver while every other typed error had a pointer receiver, so a caller needed two `errors.As` forms and a wrong one missed silently. `Kernel.ValidateConfigDetailed` returned a bare CUE error tree, so a caller could not test for "the values do not satisfy the schema" by type; the `config-validation` capability forbade any Go type around that tree, to keep the library from growing a second walking and formatting API. Five types had two exported names each: the four metadata types (declared in `opm/schema`, aliased in the artifact packages) and `module.Source` (aliased in `opm/platform` and `opm/catalog`). `opm/schema` exported fifteen assignable `cue.Path` variables, of which the consumers read three, and a constant no code read.

The options for each were to leave it (the status quo), to remove or unify it now while the line is a prerelease, or to deprecate now and remove after `v1.0.0` in a new major.

## Decision

The library applies four rules to its public surface.

A type that is an error declares `Error` on its pointer receiver. A caller matches every typed error with `errors.As` and a pointer target, and, once the receiver is a pointer, the value type is not an error. A stale value-typed form is then loud, not silent: `errors.AsType[IdentityError]` does not compile, and `errors.As(err, &v)` with a value-typed `v` is reported by `go vet` and panics at run time on every error that reaches it, because the target is checked before the chain is walked. A test in `opm/errors` scans the package for the rule. `IdentityError` reaches the rule in two steps, because opm-operator at `main` still builds it as a value and matches it with a value target, and the library deprecates before it removes (ADR-013, decision j4). In this change the library returns `*IdentityError`, the type keeps its value receiver, and an `As` method lets the pointer target and the deprecated value target both match. A later change gives it the pointer receiver and deletes `As`, after the operator has moved. Value receivers for all were rejected: fourteen types and both consumers change. Leaving `IdentityError` alone was rejected: the exception is permanent after `v1.0.0`.

A validation failure has one marker type, `errors.ConfigValidationError`, which wraps the CUE error tree and adds nothing: `Error` returns the tree's text and `Unwrap` returns the tree, so the `cuelang.org/go/cue/errors` helpers give the same result with or without it. The earlier rule keeps its purpose in a narrower form: the library projects no CUE diagnostic into a Go type and ships no walker or formatter. The marker does not implement the CUE `Error` interface, because `Errors` would then return the marker in place of a single-error tree. A sentinel joined to the tree was rejected: `errors.Join` changes the text. Only the validation step of `ValidateConfigDetailed` is marked; a source that fails to compile or load is not, because for a file-backed source that can be a registry failure.

An exported type has one name. Each metadata type is declared in the package of its artifact, next to its decoder and the field it types, which is also the name both consumers already use. `module.Source` is the one name of the staged source; `catalog.Source` stays as a Deprecated alias until the operator has moved, and goes in the same later change. Keeping the `opm/schema` names was rejected: it costs the cli about sixty edits and keeps four types in a package that uses none of them.

`opm/schema` exports a path only when a consumer reads it and no artifact accessor gives the field: `Metadata`, `Module` and `CatalogProvides`. The other paths live in `opm/internal/corepath`. A core-release constant is exported only when code outside the library's own tests reads it. A test in `opm/schema` pins both lists by name.

Removing after a deprecation period was rejected for all four: both consumers live in this workspace and migrate with their pin bump, and a deprecation that outlives `v1.0.0` can only be removed in a new major.

Security: the change adds no trust boundary, input, secret or outbound call. Moving the path variables into an internal package removes twelve process-wide variables an importer could reassign; three stay assignable and their doc comment says so. No residual risk is accepted.

## Consequences

**Positive:** Twenty-three exported names and one package leave the surface; one type and one deprecated method join it. The pointer target of `errors.As` works for every typed error. A frontend tests for a validation failure by type and keeps its CUE-native output. `task api:diff` lists a removal, and two tests hold the rules: one scans `opm/errors` for receivers, one pins the exported variables and core-release constants of `opm/schema` by name.

**Negative:** Two deprecated shims stay on the surface past this change (the value forms of `IdentityError` with its `As` method, and `catalog.Source`), and the receiver rule has one listed exception until they go. If they are not removed before `v1.0.0` they are permanent. When the receiver flips, an embedder that still calls `errors.As` with a value-typed `IdentityError` target compiles and then panics at that call, unless it runs `go vet`; the Deprecated note warns of it now. An embedder that type-asserts the validation error to the CUE `Error` interface must use `errors.As`. The path inventory now has two homes, so a maintainer adding a path chooses one.

**Operational:** The migration note is the `BREAKING CHANGE:` footer (ADR-010); it is lost if the squash message drops it, so the reviewer checks the squash message. Both consumers build against this change without an edit. The operator's eight edits (pointer targets, `&` literals, `module.Source`) come first, then the change that removes the shims.

**Trade-off:** `Metadata`, `Module`, `CatalogProvides` and `ProvidesSince` stay exported for v1, by the owner's decision, although an accessor could replace each. The acquire and synthesis values checks do not yet return the marker; adding it later is additive.
