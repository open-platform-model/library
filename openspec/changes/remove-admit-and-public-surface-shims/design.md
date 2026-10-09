## Context

ADR-015 gave typed errors one receiver shape and types one name, and left two listed exceptions for opm-operator: `IdentityError` (value receiver, `As` method) and `catalog.Source`. Two tests carry the exception lists (`TestTypedErrorsUsePointerReceivers` in `opm/errors`, `TestSurface_OneNamePerType` in `opm/schema`). opm-operator#273 removed the last uses.

`opm/k8s/ownership` took an `Admit` input for `opm operator install` over an operator installed from a manifest (0012:D8:R6/R7). The owner withdrew that rule on 2026-10-09; the cli removes its migration in cli#357.

## Goals / Non-Goals

**Goals:** remove the five exported names in one breaking beta; change no verdict for an input without `Admit`; leave the adopt rule (0012:D8:R8) as it is; correct every text that names a removed thing.

**Non-Goals:** any rename (`opm-operator` stays in every string, workflow and script); the kernel; the `providesFold` fallback; a new override of any kind.

## Research & Decisions

### Where `Admit` is read

**Context**: The change is safe only if `Admit` false and "no `Admit`" are the same function.
**Explored**: `git grep -n Admit -- opm` on c0a89cf: two fields, two helpers, tests. `admittedForApply` and `admittedForDelete` both start with `in.Admit &&`; `carriesNoOtherIdentity` and `installDeletable` are called only by them.
**Decision**: Delete the two fields, the two helpers and the two functions only they call. The two conditions become `!opmManaged(in.Live)`.
**Rationale**: `X && !(false && Y)` is `X`. No other line changes, so no verdict and no message changes. The remaining table tests run unedited as the proof.

```go
// before
if !opmManaged(in.Live) && !admittedForApply(in) { /* foreign-object */ }
// after
if !opmManaged(in.Live) { /* foreign-object */ }
```

### Pinning the absence

**Context**: After `v1.0.0` an added field is a MINOR change, so an override could return without a break.
**Explored**: The package's tests are table tests over the verdicts; none pins the input shape.
**Decision**: One reflection test lists the fields of `ApplyInput` and `DeleteInput` by name. Two table rows keep the earlier-manifest cases with their new, refused verdicts.
**Rationale**: The rule "the annotation on the live object is the only override" (0012:D8:R3) is then a test, and a new input needs a spec change.

### The `IdentityError` receiver

**Context**: ADR-015 planned the flip in two steps.
**Explored**: Callers on c0a89cf and in both consumers build `&IdentityError{}` and match `*IdentityError`. With a pointer receiver and no `As`, `errors.As` with a `*IdentityError` target matches by assignability, as for every other type here.
**Decision**: `func (e *IdentityError) Error() string`; delete `As`; empty the exception list in `TestTypedErrorsUsePointerReceivers` and make the test fail on any value receiver; drop the value-target assertions.
**Rationale**: This is the planned flip and nothing more. The behaviour ADR-015 named stays as named: a stale value target does not compile under `errors.AsType`, and `errors.As` with one is reported by `go vet` and panics.

### `catalog.Source`

**Decision**: Delete the alias; empty `deprecatedAliases` in `TestSurface_OneNamePerType` by removing the mechanism, so any alias fails.
**Rationale**: The field `Catalog.Source` is already typed `*module.Source`.

### Records

**Decision**: ADR-015 gets a dated closing note under Status (the two exceptions are gone). A new ADR-016 records the admission removal with 0012:D8:R6/R7 and what stays (0012:D8:R8). No `enhancement.yaml`.
**Rationale**: AGENTS.md "Where a statement lives": rationale goes in an ADR. The delivery log records delivered decisions, and this change delivers none.

## Risks / Trade-offs

- cli `main` does not build against this change until cli#357 merges → merge order: cli#357 first. `Consumer build (cli)` is red on this PR until then; it is not a required check.
- An embedder outside the workspace that sets `Admit` or uses a value target breaks at compile time (or under `go vet` for `errors.As`) → the `BREAKING CHANGE:` footer says what to write.
- An operator installed from the earlier manifest is no longer taken over by `opm operator install` → decided by the owner; the user annotates or deletes the objects (cli docs, not this repo).
- The squash message can drop the footer → the reviewer checks it (ADR-010).

## Migration Plan

Merge after cli#357. Release as the next beta. Then both consumers bump their pin; neither needs a code edit beyond one stale comment in opm-operator. Rollback is a revert before the release; after it, the next beta.

## Open Questions

None.
