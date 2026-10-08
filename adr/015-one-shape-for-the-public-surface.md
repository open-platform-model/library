# ADR-015: One shape for the public surface before v1.0.0

## Status

Proposed (2026-10-08). Written with the change `tidy-public-surface`; the owner accepts it by merging that change. Reversibility: two-way until `v1.0.0` is tagged, one-way after it (a removal then needs a new major and a new module path).

## Context

From `v1.0.0` every exported name under `opm/` is a contract (CONSTITUTION VI), and the module path has no major suffix, so a later removal is a path change for every consumer. Four things on the surface had no rule behind them. `IdentityError` had a value receiver while every other typed error had a pointer receiver, so a caller needed two `errors.As` forms and a wrong one missed silently. `Kernel.ValidateConfigDetailed` returned a bare CUE error tree, so a caller could not test for "the values do not satisfy the schema" by type; the `config-validation` capability forbade any Go type around that tree, to keep the library from growing a second walking and formatting API. Five types had two exported names each: the four metadata types (declared in `opm/schema`, aliased in the artifact packages) and `module.Source` (aliased in `opm/platform` and `opm/catalog`). `opm/schema` exported fifteen assignable `cue.Path` variables, of which the consumers read three, and a constant no code read.

The options for each were to leave it (the status quo), to remove or unify it now while the line is a prerelease, or to deprecate now and remove after `v1.0.0` in a new major.

## Decision

The library applies four rules to its public surface.

A type that is an error declares `Error` on its pointer receiver. A caller matches every typed error with `errors.As` and a pointer target, and the value type is not an error. A stale value-typed form is loud, not silent: `errors.AsType[IdentityError]` does not compile, and `errors.As(err, &v)` with a value-typed `v` is reported by `go vet` and panics at run time on every error that reaches it, because the target is checked before the chain is walked. A test in `opm/errors` scans the package for the rule. Value receivers for all were rejected: fourteen types and both consumers change. Leaving `IdentityError` alone was rejected: the exception is permanent after `v1.0.0`.

A validation failure has one marker type, `errors.ConfigValidationError`, which wraps the CUE error tree and adds nothing: `Error` returns the tree's text and `Unwrap` returns the tree, so the `cuelang.org/go/cue/errors` helpers give the same result with or without it. The earlier rule keeps its purpose in a narrower form: the library projects no CUE diagnostic into a Go type and ships no walker or formatter. The marker does not implement the CUE `Error` interface, because `Errors` would then return the marker in place of a single-error tree. A sentinel joined to the tree was rejected: `errors.Join` changes the text. Only the validation step of `ValidateConfigDetailed` is marked; a source that fails to compile or load is not, because for a file-backed source that can be a registry failure.

An exported type has one name. Each metadata type is declared in the package of its artifact, next to its decoder and the field it types, which is also the name both consumers already use. `module.Source` is the one name of the staged source. Keeping the `opm/schema` names was rejected: it costs the cli about sixty edits and keeps four types in a package that uses none of them.

`opm/schema` exports a path only when a consumer reads it and no artifact accessor gives the field: `Metadata`, `Module` and `CatalogProvides`. The other paths live in `opm/internal/corepath`. A core-release constant is exported only when code outside the library's own tests reads it. A test in `opm/schema` pins both lists by name.

Removing after a deprecation period was rejected for all four: both consumers live in this workspace and migrate with their pin bump, and a deprecation that outlives `v1.0.0` can only be removed in a new major.

Security: the change adds no trust boundary, input, secret or outbound call. Moving the path variables into an internal package removes twelve process-wide variables an importer could reassign; three stay assignable and their doc comment says so. No residual risk is accepted.

## Consequences

**Positive:** Twenty-four exported names and one package leave the surface, and one name joins it. One `errors.As` form for every typed error. A frontend tests for a validation failure by type and keeps its CUE-native output. `task api:diff` lists a removal, and two tests hold the rules: one scans `opm/errors` for receivers, one pins the exported variables and core-release constants of `opm/schema` by name.

**Negative:** opm-operator at `main` does not compile against this library until it changes eight lines (the `IdentityError` forms and one `catalog.Source`); the `Consumer build (opm-operator)` job is red on the pull request. An embedder outside the workspace that still calls `errors.As` with a value-typed `IdentityError` target compiles and then panics at that call, unless it runs `go vet`. One that type-asserts the validation error to the CUE `Error` interface must use `errors.As`. The path inventory now has two homes, so a maintainer adding a path chooses one.

**Operational:** The migration note is the `BREAKING CHANGE:` footer (ADR-010); it is lost if the squash message drops it, so the reviewer checks the squash message. The operator's edits ride its next library pin bump.

**Trade-off:** `Metadata`, `Module`, `CatalogProvides` and `ProvidesSince` stay exported because opm-operator reads them, although an accessor could replace each. Whether they stay for good is the owner's call before `v1.0.0`. The acquire and synthesis values checks do not yet return the marker; adding it later is additive.
