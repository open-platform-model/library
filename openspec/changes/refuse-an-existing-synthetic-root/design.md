## Context

cue/load serves `load.Config.Overlay` on top of the real filesystem: for a directory it lists, it
merges the overlay entries with whatever exists on disk at that path. Both in-memory build paths
in the library key their overlay under a fixed or deterministic root that is meant not to exist:

| Root | Who stages under it | Guard today |
| --- | --- | --- |
| `renderstage.RenderRoot` (`/opm-render`, `C:\opm-render`) | `renderstage.Stage`, every render | `checkRootAbsent` refuses when `os.Lstat` finds anything (`opm/internal/renderstage/stage.go`) |
| `sourcetree.SyntheticRoot(path, version)` (under `/opm-registry-module`, `C:\opm-registry-module`) | `loader.FetchArtifact`, every registry acquire of a module or catalog | none |

Before library PR #212 the Windows synthetic root had no volume, cue/load refused the overlay keys
as not absolute, and registry acquisition did not work on Windows at all. With the volume it works,
and the missing guard matters: any local Windows user can create a directory under `C:\`.

## Goals / Non-Goals

**Goals:**

- A registry acquire refuses, naming the path, when anything exists at its synthetic root.
- One implementation of the check for both roots.
- A test that shows the refusal with a real fetched fixture and an injected `.cue` file.
- Fix the two #212 wording faults.

**Non-Goals:**

- Closing the window between the check and the load (see Risks).
- Changing where the roots lie, or randomising them. Deterministic roots keep error positions and
  staged sources identical from one acquire to the next, which `registry-module-loading` requires
  (bare and v-prefixed versions stage under the same root).
- Refusing when only the base directory (`/opm-registry-module`) exists. cue/load reads the disk
  beneath the module root it is given, and the base lies above every synthetic root, so a stray
  base directory with no matching child is harmless. A symlink at the base is still caught,
  because `os.Lstat` of the child follows every component but the last.

## Decisions

### D1: The check lives in `sourcetree`, and both callers use it

`sourcetree` already owns `VolumeRoot` and `SyntheticRoot`, and `renderstage` already imports it.
`CheckRootAbsent(role, root string) error` returns nil when `os.Lstat(root)` reports
`fs.ErrNotExist`, an error naming the role and the path when it finds anything, and a wrapped error
for any other `Lstat` failure. The message keeps the render form:
`<role> <root> exists on disk; it is served from memory and the build would read what is there, so remove it`.
`renderstage.checkRootAbsent` is deleted and `Stage` calls the helper with role `render root`, so
`TestStage_RefusesAnExistingRenderRoot` (which asserts `render root <path> exists on disk`) passes
unchanged. `FetchArtifact` calls it with role `synthetic root`.

Alternative: a second copy of the ten lines in `loader`. Rejected: the two guards protect against
the same cue/load behaviour and should not drift apart.

### D2: The check runs after the fetch, right before staging

`FetchArtifact` computes the synthetic root after the fetch today. The check sits immediately after
that and before `OverlayFromFS`, wrapped as `staging <label> <module@version>: <cause>`, the prefix
the other staging failures there already use. Running it before the fetch would save one network
call on a refusal, but it would widen the window between the check and the load by the length of
the fetch, and a refusal here is rare. The cancellation check after the fetch stays first, so a
cancelled acquire still reports the cancellation.

### D3: The base becomes an exported variable for tests

`syntheticBase` becomes `SyntheticBase`, documented like `RenderRoot`: it is a variable only so a
test can point it at an existing directory, and nothing else assigns it. `opm/internal/sourcetree`
is internal, so no consumer can reach it and `task api:diff` sees no change. The loader test sets
it to `t.TempDir()`, restores it in `t.Cleanup`, and does not run in parallel. No test in
`opm/internal/loader` calls `t.Parallel` today.

Alternative: a function-valued hook in `loader`. Rejected: it would add a second way to compute
the root, and the render side already set the precedent with `RenderRoot`.

### D4: The error is plain, not typed

The refusal is not a fetch failure, so it is not a `*FetchError` and `errors.Classify` does not
mark it transient. A frontend that retries transient failures does not retry this one, which is
correct, because retrying does not remove the directory. No sentinel is added; nothing in the cli
or opm-operator needs to tell this refusal apart from other permanent acquire failures.

## Risks / Trade-offs

- [The check and the load are two steps] A directory created after `Lstat` and before cue/load
  reads the root is not caught. → The window is one build. Closing it needs cue/load to stop
  reading the disk beneath an overlay root; nothing in the library can close it. The guard turns a
  silent, persistent injection into a refusal in every case except a race won during one build.
- [A leftover directory blocks acquires] If something does create a synthetic root (a debugging
  session, a tool), every acquire of that module version fails until it is removed. → The error
  names the path and says to remove it.
