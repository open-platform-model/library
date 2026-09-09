## Why

Two facts on `main` today:

1. **Every overlay-mode render writes the whole module tree to disk first.** `renderstage.Stage` serves an on-disk input by pointing the render module's `local-module.cue` replacement at the real directory, and an overlay input by writing every entry into the staging directory through `Source.WriteTo` (`opm/internal/renderstage/stage.go:161-177`), which cue/load then reads back. A synthesized instance is always overlay mode and carries the module's entire tree (the operator's every render), and an instance acquired with extra values is overlay mode too (the cli's every `-f values` render). On `modules/k8up` that is 1.4 MB written and re-read per render.
2. **cue/load can serve a replacement directory from memory.** A `replaceWith` directory is opened through the loader's own overlay-aware filesystem (`cue/load/config.go:621`, `c.fileSystem.ioFS(dir, ...)`), the same layer `load.Config.Overlay` feeds, and `opm/internal/loader.LoadDir` already builds the acquired artifacts that way. Only the render stage still materializes.

This is slice 6c of the simplification plan reviewed on 2026-09-05. Internal; no public surface moves, and the staging directory keeps holding the generated render module (its `cue.mod` and glue), which `VerifyCoverage` re-reads by design.

**Scope statement (Principle VIII).** Two functions in one internal package (`Stage`, `Build`) and the `Staged` struct between them. A spike opens the change because the claim rests on a cue/load behaviour the library has not exercised for replacements yet.

## What Changes

**`opm/internal/renderstage`:**

- `Stage` no longer writes an overlay-mode input. For each such input it re-keys the source's entries from `Source.Root` to `<dir>/instance` or `<dir>/platform` (the directory the replacement already names), collects them on `Staged.Overlay`, and points the replacement at that path exactly as today. On-disk inputs are unchanged.
- `Build` passes `Staged.Overlay` as `load.Config.Overlay` beside `Dir` and `ModuleRoot`.
- `Source.WriteTo` stays the library's one overlay writer; its remaining caller is a frontend (the cli's scaffold).

**Docs:** `CLAUDE.md` § Render contract (what the staging directory holds), `opm/internal/renderstage/doc.go`, `opm/kernel/doc.go` where it describes staging.

**Not in this change:** any change to promotion, skew, the glue or the decoder; the per-verb contexts (`kernel-owns-no-build-context`).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `single-build-render`: the inputs requirement's overlay scenario and the own-build requirement state that overlay-mode inputs are served from memory and the staging directory holds only the generated render module.
- `artifact-types`: the overlay-writer requirement no longer says the render stage materializes through `WriteTo`.

## Impact

**SemVer:** PATCH. No exported symbol changes; render output is byte-identical (the parity oracle and every render test pin it).

**Downstream:** none.

**Library:** `opm/internal/renderstage/stage.go` (+ `stage_test.go`), two doc paragraphs. Behaviour visible only as the absence of `<staging>/instance/**` and `<staging>/platform/**` on disk during a render.

**Complexity justification (Principle VII):** about 20 lines of re-keying replace a write loop per render; no new type, one new field on an internal struct.
