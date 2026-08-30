## Context

See proposal.md, Why. The 2026-08-30 kernel review swept exported identifiers under `opm/` (comments stripped) against library production code, library tests, `cli` and `opm-operator`. The identifiers this change removes each have zero production references and zero consumer references; the only readers are their own tests and, for some, a spec requirement that still names them.

Constraints that shape the approach:

- Principle VI: every removal of an exported identifier is MAJOR, even with no known caller. Under alpha the cost is a `MIGRATIONS.md` entry per identifier group, not a compatibility shim.
- Principle VII: the removal list is bounded by "zero readers today", not by taste. An identifier with one production reader stays, even if that reader is thin.
- Enhancement 0009 D9 owns the kernel's cancellation path and the logger, tracer and clock slots. They are unread today and are not touched here.
- `library-phase-and-values-prune` owns `Plan`, `Kernel.Validate`, `CompileInput.Values` and the typed validation wrappers. Not touched here.

## Goals / Non-Goals

**Goals:**

- Delete every identifier in the proposal's list, its tests, and the spec text that requires it, in one change with one migration entry.
- Leave every consumer compiling unchanged (verified by grep before and by building both consumers after).
- Retire two spec requirements that describe symbols which never existed on `main` (`ParseModuleInstance` alias, `ModuleResult` alias scenario) so the spec stops promising phantom surface.

**Non-Goals:**

- Any behaviour change. No rendered value, no error message reachable from a consumer, no match verdict changes.
- Any change to `Kernel` fields or options (0009 D9).
- Cleaning up the stale binding-era prose in `platform-artifact` (`Paths()`, slice numbers); that spec has no requirement this change affects and gets its own hygiene pass.

## Decisions

### D1: Remove the value-only registry loader together with its kernel wrapper

**Context**: `registry.LoadModulePackage` returns `StagedSource.Value` and drops the staged source; `Kernel.LoadModuleFromRegistry` wraps it. Since `synth-instance-in-module-root`, `synth.Instance` refuses a source-free module, so the value-only path leads nowhere a consumer can use it.

**Explored**: Keeping `registry.LoadModulePackage` as the helper's documented entry and removing only the kernel wrapper. Rejected: it leaves a helper entry point whose only caller would be its own test, which is the shape this change exists to remove.

**Decision**: Remove both. `LoadModulePackageWithSource` becomes the package's single entry and `Kernel.AcquireModuleFromRegistry` the single kernel entry.

**Rationale**: One acquisition path, one identity verification site, one shape gate call. A caller that wants only the value reads `StagedSource.Value` or `mod.Package`.

### D2: Remove `StripProvenance` although the compat spec names it

**Context**: `opm/compat.StripProvenance` implements 0010 D30 via a syntax round-trip. The matcher's comment records why it does not use it (the round-trip cannot rebuild kernel-side schema-derived operands and opens closed definitions), and the comparison walk skips provenance at every depth on its own. Zero production callers.

**Explored**: Keeping it for a future publish-gate consumer in `cli`. Rejected: the `cli` gate already calls `compat.CheckAtLevel`, whose walk carries the skip, so the future consumer already exists and does not use it.

**Decision**: Remove it and the `catalog-compatibility` requirement that specifies it. D30 stays implemented by the walk and by the matcher's verdict filter; the enhancement decision is not reopened.

**Rationale**: Two implementations of one rule, one of them unused and lossy (positions discarded), is a maintenance trap. The remaining implementation is the one both real consumers exercise.

### D3: Delete `materialize/cache` rather than keep it as a reference implementation

**Context**: The package was designed as opt-in memoization the kernel would not hold. The operator wrote a single generation-keyed slot instead; the CLI relies on CUE's disk cache.

**Explored**: Keeping `Key` alone as a stable content hash of a platform's registry. Rejected: no caller, and 0019 D5 changes what a registry entry carries (the catalog by import), so the key's normalised shape is about to be wrong anyway.

**Decision**: Delete the package. The Principle I statement ("the kernel holds no materialize cache") moves to the `Materialize` method's doc comment, where it already appears.

### D4: Retire phantom spec requirements in the same change

**Context**: `kernel-runtime` requires a deprecated `ParseModuleInstance` alias and a `ModuleResult` alias scenario. The former never existed on `main`; the latter is removed here.

**Decision**: Remove both from the spec via REMOVED / MODIFIED deltas in this change, and add negative scenarios so the absence is pinned.

**Rationale**: A spec that requires surface the tree does not have is the same defect as surface the spec does not know about. Both are corrected by the sweep whose whole subject is "what exists versus what is recorded".

### D5: Pin absence with negative scenarios, not with tests that grep source

**Decision**: Each removed group gets a scenario stating the identifier does not exist. Where the archived `library-finalize-removal` change added a reflect-based test for a removed method, this change adds one reflect-based test for the removed `Kernel` method (`LoadModuleFromRegistry`) and relies on the compiler for package-level removals (a deleted package cannot be imported).

## Risks / Trade-offs

- [A consumer branch not yet on `main` references a removed identifier] → The grep was run against `cli` and `opm-operator` `main` on 2026-08-30. `MIGRATIONS.md` names every identifier and its replacement; the fix on such a branch is mechanical.
- [`AnnotationDefaultNamespace` was the only Go-side spelling of an ADR-001 key] → The key is declared in core's `#Module` and is a schema fact, not a library one. ADR-001 is unchanged; the migration entry says where the key lives.
- [Removing `StripProvenance` looks like reopening 0010 D30] → The design records that D30 has two live implementations that stay. The `catalog-compatibility` delta says so in its Reason.
- [Three MAJOR removals in a row (`library-finalize-removal`, this, `library-phase-and-values-prune`) accumulate under one `Unreleased, Breaking` bucket] → Each carries its own `MIGRATIONS.md` entry keyed by change slug; release-please graduates them together, which is the intended alpha cadence.

## Migration Plan

1. Land this change as one PR after `library-finalize-removal` is on `main` (it is, as of 2026-08-30).
2. `MIGRATIONS.md` entry under `## Unreleased — Breaking`, one bullet per identifier group, replacement named for each.
3. Rollback is a revert; nothing downstream depends on the removals.
