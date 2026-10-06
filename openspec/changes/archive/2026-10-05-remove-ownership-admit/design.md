## Context

`CanApply` and `CanDelete` each call one admission helper next to the managed-by check:

```go
if !opmManaged(in.Live) && !admittedForApply(in) { ... refuse foreign-object }
if !opmManaged(in.Live) && !admittedForDelete(in) { ... skip not-opm-managed }
```

`admittedForApply` uses `carriesNoOtherIdentity`; `admittedForDelete` uses `installDeletable`.
Nothing else in the library calls the four helpers.

## Goals / Non-Goals

**Goals:** the two verdicts decide exactly as they would for a caller that never set `Admit`; the
public surface loses the two fields.

**Non-Goals:** any other verdict change. The adopt annotation stays the only override.

## Decisions

- The condition becomes `if !opmManaged(in.Live)` in both verdicts, and the four helpers go. A
  verdict for `Admit: false` is unchanged, so every non-admission test passes untouched.
- The admission test cases are deleted, not rewritten: each one asserted only what admission lifts
  or does not lift. The behaviour they would assert without admission (a foreign object is refused
  on apply and skipped on delete, a terminating object is refused, a protected kind is skipped) is
  already covered by the remaining cases.
- The adopt-key citation moves from 0012:D8:R6 to 0012:D8. The decision text itself says neither
  frontend sets the annotation, and the withdrawal reshapes R6; citing the decision under-claims
  rather than naming a requirement that may no longer say it.

## Research & Decisions

### Whether to keep `Admit` as an unused seam

**Context**: the field is released public API (beta.5) with no caller.
**Explored**: keep the field as a seam for a future migration, or remove it.
**Decision**: remove it.
**Rationale**: the owner decided on 2026-10-05 to remove all legacy-migration code, since nobody
runs OPM yet (0032:D1 renames with no migration; 0012:D8 withdraws its install admission); a dead
public field is surface to support (Principle VII).

## Risks / Trade-offs

- A frontend outside the workspace that set `Admit` stops compiling. None is known; the
  `BREAKING CHANGE:` footer names the removal.
