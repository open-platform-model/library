# Design: overlay-served-in-memory

## Context

See `proposal.md` § Why. The staging directory of one render today:

```
  <tmp>/opm-render-XXXX/
    cue.mod/module.cue        promoted deps (written, re-read by VerifyCoverage)
    cue.mod/local-module.cue  replaceWith: instance -> <dir>/instance, platform -> <dir>/platform (or real dirs)
    render.cue                the glue
    instance/**               WRITTEN from Source.Overlay when the instance is overlay mode
    platform/**               WRITTEN from Source.Overlay when the platform is overlay mode
```

Constraints: cue/load's replacement directory must sit inside `ModuleRoot` for the overlay filesystem to serve it; `ReadModFile` and `PackageName` already read overlay sources from memory through `sourcetree`, so `Stage`'s promotion and skew inputs do not depend on the write; `VerifyCoverage` re-reads the written `cue.mod/module.cue` on purpose (a D13 tripwire) and keeps doing so.

## Goals / Non-Goals

**Goals:**

- No file of an overlay-mode input touches disk during a render.
- Render output and every verdict are byte-identical to today.

**Non-Goals:**

- Serving the generated render module itself from memory; `VerifyCoverage`'s re-read is deliberate.
- Changing how on-disk inputs are served.

## Decisions

### Entries are re-keyed under the staging directory, not served at the synthetic root

**Context**: an overlay source's keys sit under `Source.Root`, a synthetic absolute path for a fetched module (`sourcetree.SyntheticRoot`) or a real directory for a layered instance. cue/load's overlay applies inside `ModuleRoot`.
**Explored**: (A) point the replacement at `Source.Root` and hand the overlay through unchanged (keys outside `ModuleRoot`; a real-directory root would mix on-disk files with the overlay's); (B) re-key every entry from `Source.Root` to `<dir>/<name>` and point the replacement there, as the write does today.
**Decision**: B. `Stage` builds one `map[string][]byte` for both inputs, keyed under the staging directory; `Build` sets `cfg.Overlay` from it with `load.FromBytes`. `Staged` gains `Overlay map[string][]byte`.
**Rationale**: the replacement path is the same one the write produced, so `local-module.cue`, promotion and the skew rows are untouched; the overlay never shadows a real file because the staging directory holds nothing under `instance/` or `platform/`.

### The spike is the first task and decides the change

**Decision**: task 1.1 is a unit test in `stage_test.go` that stages an overlay-mode instance against an on-disk platform, asserts `<dir>/instance` does not exist on disk, builds, and asserts the built value's `_components` resolve. If cue/load refuses to serve the replacement from the overlay, the test names the error, the change is archived with that finding recorded here, and the write stays.
**Rationale**: the proposal's claim is a reading of `cue/load/config.go:621`, not a measurement; the test turns it into one before anything else changes.

## Risks / Trade-offs

- [cue/load reads the replaced module's `cue.mod/module.cue` from disk rather than the overlay] → `checkReplaceDirModulePath` opens it through the same filesystem the spike exercises; a failure shows up in task 1.1, not in production.
- [A consumer relied on the staged tree being on disk during a render] → none can: the directory is private to the call and removed on return.

## Migration Plan

One library PR; `task check` green including the race pass and the parity harness; `task cue:test:flow` green. Rollback is a revert.

## Open Questions

None.
