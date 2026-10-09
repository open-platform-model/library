## Context

The rename is textual in this repo: no Go identifier, no exported symbol and no rendered output
names the operator. The managed-by value `opm-controller` and `labels.ManagedByController` were
already neutral and do not change.

## Goals / Non-Goals

**Goals:** every product mention says "controller" or `opm-controller`; every human-role mention
stays; nothing a test asserts on changes meaning.

**Non-Goals:** the `.github`-owned wiring-check copy; the CHANGELOG and archived changes (history,
kept as written).

## Decisions

- **Exact patterns only.** `opm-operator` is replaced everywhere in scope. The bare word is
  replaced line by line, after excluding "operator persona" (CONSTITUTION, openspec config),
  "Operators in air-gapped/restricted environments" (`docs/getting-started.md`,
  `opm/schema/loader.go`) and ADR-001 (people who deploy modules).
- **Provenance notes keep their commits.** "opm-operator 9b83611" becomes "opm-controller
  9b83611": the repository keeps its history across a GitHub rename, so the commit still
  resolves.
- **The kernel-runtime scenario is a product mention.** "Operator-shaped caller" described the
  controller's call path (raw bytes, no `cue.Context`), so it is renamed, not kept as generic.
- **Spec bodies move by delta, Purposes by hand.** MODIFIED blocks carry every scenario
  unchanged in name; the one retitled scenario takes a REMOVED + ADDED under a new requirement
  title.

## Research & Decisions

### Where the rename's sites are

**Context**: the sites had to be complete without a bare-word sed.
**Explored**: `git grep -nIiw 'operators?'` and the residue regex of task 3.6 over the
worktree.
**Decision**: rename every product hit; keep the four human-role hits above.
**Rationale**: the vocabulary rule of 0032:D1: "controller" is the product; the human role is not
renamed.

## Risks / Trade-offs

- Before the GitHub rename, `Consumer build (opm-controller)` cannot clone. The CI edits and the
  `consumer-build` spec wording are therefore the last, separate commit of the branch, applied
  after the GitHub rename; the commits before it keep `Consumer build (opm-operator)`.
