# Design: move-compat-to-cli

## Context

See `proposal.md` § Why. The import graph once `cue-owned-verdicts` has landed:

```
  cli/internal/publish  ---> library/opm/compat   (ParseLevel, CheckAtLevel, Violation)
  cli/internal/scaffold ---> library/opm/compat   (HighestStable)
  library/opm/**        -/-> library/opm/compat   (no importer left; alternatives.go deleted by slice 4)
  library/opm/compat    ---> cuelang.org/go/cue, cue/format, Masterminds/semver
```

Constraints: `cue-owned-verdicts` has landed (task 1.1 checks it, since the tree still holds `alternatives.go` until then); no behaviour changes; the cli's `.golangci.yml` has no depguard rule to extend; `cli/internal/publish` already holds a file named `compat.go` (the gate), so that package cannot be the destination without merging the pure walk with gate policy, which 0011 keeps apart.

## Goals / Non-Goals

**Goals:**

- The comparator, the ladder and the float selector sit beside their only callers, so a comparator change is one cli PR.
- The library's exported surface holds kernel packages only: `kernel`, `module`, `platform`, `schema`, `errors`, and the one opt-in helper.
- The enhancements record says where the code lives.

**Non-Goals:**

- Changing the comparator, the ladder or the selector (0020's new rules land later, in the cli).
- Editing accepted decision text in 0011; decision text is immutable and the delivery log carries the relocation.
- A depguard rule in the cli.

## Decisions

### Destination is `cli/internal/compat`, package name unchanged

**Context**: three candidates. `cli/internal/publish` already holds `compat.go`, the gate; merging the pure walk into it joins the two concerns 0011 keeps apart (the walk reports, the gate decides). `cli/pkg/compat` stays importable by other Go programs and so preserves 0011 D9's "any CI action" sharing, at the cost of dragging the cli module's dependency graph into any importer. `cli/internal/compat` states that no non-cli consumer exists.
**Decision**: `cli/internal/compat`, the owner's choice on 2026-09-07. The package name stays `compat`, so the two call sites change only the import path and the four files move byte-for-byte; the package doc's consumer list drops library-matching.
**Rationale**: no non-cli consumer exists today; `internal` says so. The day one appears, promoting the directory to `cli/pkg/compat` is a rename inside the cli, and the code is pure and self-contained.

### `HighestStable` becomes `scaffold.highestStable`

**Context**: 0011 D23 struck `HighestStable` as the gate's predecessor selector and kept it only because template resolution (`cli-template-modules`) calls it. That caller is `scaffold.ResolveTemplateVersion`; nothing else calls it.
**Decision**: unexported `highestStable` in `cli/internal/scaffold`, with `predecessor.go`'s doc comment carried over (it explains why the float selector is not the gate's) and the four cases of `predecessor_test.go` in the scaffold tests.
**Alternative rejected**: keeping it in `cli/internal/compat` beside the ladder. It shares nothing with the comparator but the semver import, and a "compat" package holding a float selector is the confusion D23 had to explain away.

### The spec capability moves with the code

**Decision**: `catalog-compatibility` is REMOVED from the library's specs, each requirement's migration pointing at the cli; the cli PR adds the capability to `cli/openspec/specs/` restated against `cli/internal/compat` (three requirements, thirteen scenarios, unchanged semantics). The library's `helper-packages` boundary list drops `opm/compat` and, since `cue-owned-verdicts` deletes the package without a `helper-packages` delta, `opm/core`.
**Rationale**: a spec describes behaviour a repo ships; the library no longer ships this one.

### The enhancements record is amended, not contradicted

**Context**: 0011 D9's rationale and D23's note (accepted) place the code in the library; 0020 (draft) plans to extend it there and orders delivery "library before cli".
**Decision**: this change carries `enhancement.yaml` declaring 0011 D9, so `task enhancements:delivery:log` writes a line whose summary names the new home; 0020's `06-operational.md` item 3 and the four `schemas/target.cue` comments are edited in a separate `enhancements/` commit before this change merges (a draft entry is editable and `task vet` must pass). No decision text in 0011 changes.
**Rationale**: decision numbers and accepted text are immutable; a landed relocation belongs in the log; a future landing point belongs in the draft that plans it.

### The library keeps its semver dependency

**Decision**: no `go.mod` change in the library: `renderstage` imports `Masterminds/semver` for skew and promotion. The cli's `go.mod` promotes the same module to a direct requirement; `go mod tidy` does it.

## Risks / Trade-offs

- [A non-cli consumer of the comparator appears: the operator, or a Go CI action] → `cli/internal/compat` becomes `cli/pkg/compat` or returns to a library by directory rename; the package is pure and self-contained. The amended 0020 says so.
- [0020's authors assumed "library before cli" for the comparator] → the amendment collapses it to cli-only: one fewer hand-off and one fewer repo on the proof case's path.
- [`TestAPIVersionPatternCoreParity` pins a string copied from `core/src/types.cue:83`] → it moves unchanged; the pin is a string constant and needs no schema cache.
- [Two PRs must land in order] → the cli builds against the previous alpha until its PR, exactly as for slices 2 and 4; the wave re-pins once.

## Migration Plan

1. `enhancements/` commit: amend 0020 (task 1.2); `task vet` green there.
2. Library PR: delete the package, the lint entry, the doc lines and the specs; `task check` green; the consumer build against a `replace` proves the cli edits are the listed ones.
3. Release-please cuts the alpha that closes the wave (slices 2, 4 and 5).
4. cli PR: `internal/compat`, `scaffold.highestStable`, the import path, the re-pin, and the capability in the cli's specs.
5. Archive: `task enhancements:delivery:log FROM=<this change directory>` writes 0011's line.

Rollback: the cli pins the previous alpha; nothing persisted changes shape.

## Open Questions

None.
