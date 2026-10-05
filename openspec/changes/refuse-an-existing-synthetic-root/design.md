## Context

cue/load serves `load.Config.Overlay` on top of the real filesystem: for a directory it lists, it
merges the overlay entries with whatever exists on disk at that path. Both in-memory build paths
in the library key their overlay under a fixed or deterministic root that is meant not to exist:

| Root | Who stages under it | Guard today |
| --- | --- | --- |
| `renderstage.RenderRoot` (`/opm-render`, `C:\opm-render`) | `renderstage.Stage`, every render | `checkRootAbsent` refuses when `os.Lstat` finds anything (`opm/internal/renderstage/stage.go`) |
| `sourcetree.SyntheticRoot(path, version)` (under `/opm-registry-module`, `C:\opm-registry-module`) | `loader.FetchArtifact`, every registry acquire of a module or catalog; then `synth.Instance`, every synthesis from a registry-acquired module, which builds `<root>/opm-synth-instance` | none |

Before library PR #212 the Windows synthetic root had no volume, cue/load refused the overlay keys
as not absolute, and registry acquisition did not work on Windows at all. With the volume it works,
and the missing guard matters: any local Windows user can create a directory under `C:\`.

## Goals / Non-Goals

**Goals:**

- A registry acquire refuses, naming the path, when anything exists at its synthetic root.
- A synthesis from a registry-acquired module refuses the same way, checked again at synthesis
  time, because it builds beneath the same root after the acquire has returned.
- One implementation of the check for both roots and all three build sites.
- A test that shows the refusal with a real fetched fixture and an injected `.cue` file.
- Fix the two #212 wording faults.

**Non-Goals:**

- Closing the window between the check and the load (see Risks).
- Changing where the roots lie, or randomising them. Deterministic roots keep error positions and
  staged sources identical from one acquire to the next, which `registry-module-loading` requires
  (bare and v-prefixed versions stage under the same root).
- Refusing when only the base directory (`/opm-registry-module`) exists. cue/load reads the disk
  beneath the module root it is given, and the base lies above every synthetic root, so a stray
  base directory with no matching child is harmless. A symlinked base is caught whenever the child
  root resolves to something, because `os.Lstat` follows every component but the last.

## Decisions

### D1: The check lives in `sourcetree`, and both callers use it

`sourcetree` already owns `VolumeRoot` and `SyntheticRoot`, and `renderstage` already imports it.
`CheckRootAbsent(role, served, root string) error` returns nil when `os.Lstat(root)` reports
`fs.ErrNotExist`, an error naming the role and the path when it finds anything, and a wrapped error
for any other `Lstat` failure. The message is
`<role> <root> exists on disk; the <what> is served from memory under it and the build would read what is there, so remove it`,
where the caller passes both phrases. `renderstage.checkRootAbsent` is deleted and `Stage` calls
the helper with `render root` and `render module`, so the render refusal stays byte-identical and
`TestStage_RefusesAnExistingRenderRoot` passes unchanged. `FetchArtifact` passes `synthetic root`
and the artifact label (`module`, `catalog`); `synth.Instance` passes `synthetic root` and
`module and its synthesized instance package`.

Alternative: a second copy of the ten lines in `loader`. Rejected: the two guards protect against
the same cue/load behaviour and should not drift apart.

### D2: The check runs after the fetch, right before staging

`FetchArtifact` computes the synthetic root after the fetch today. The check sits immediately after
that and before `OverlayFromFS`, wrapped as `staging <label> <module@version> in overlay: <cause>`,
the prefix the other staging failures there already use. Running it before the fetch would save one network
call on a refusal, but it would widen the window between the check and the load by the length of
the fetch, and a refusal here is rare. The cancellation check after the fetch stays first, so a
cancelled acquire still reports the cancellation.

### D3: Synthesis checks again, only under a synthetic root

`synth.Instance` builds the instance package beneath the acquired module's `Source.Root`. For a
registry-acquired module that root is the synthetic root, and the build runs whenever the caller
synthesizes, which can be long after the acquire and, on a values conflict, twice in one
`Kernel.SynthesizeInstance` call. The acquire-time check does not cover those builds, so
`synth.Instance` calls the helper before its own `LoadDir`, every time. It does so only when
`sourcetree.IsSynthetic(root)` reports that the root lies directly under `SyntheticBase`: a module
acquired from a directory has a real root on disk by design and must stay allowed. The refusal is
wrapped `instance synthesis: <cause>`, like the other build failures there, and returns no tree.

Alternative: put the check in `loader.LoadDir` for every overlay-mode root under the base. Rejected
for now: `LoadDir` serves every artifact kind and acquisition mode, and naming the two build sites
that stage under the synthetic root keeps the guard where the reason for it is documented.

### D4: The base becomes an exported variable for tests

`syntheticBase` becomes `SyntheticBase`, documented like `RenderRoot`: it is a variable only so a
test can point it at an existing directory, and nothing else assigns it. `opm/internal/sourcetree`
is internal, so no consumer can reach it and `task api:diff` sees no change. The loader test sets
it to `t.TempDir()`, restores it in `t.Cleanup`, and does not run in parallel. No test in
`opm/internal/loader` calls `t.Parallel` today.

Alternative: a function-valued hook in `loader`. Rejected: it would add a second way to compute
the root, and the render side already set the precedent with `RenderRoot`.

### D5: The error is plain, not typed

The refusal is not a fetch failure, so it is not a `*FetchError` and `errors.Classify` does not
mark it transient. A frontend that retries transient failures does not retry this one, which is
correct, because retrying does not remove the directory. No sentinel is added; nothing in the cli
or opm-operator needs to tell this refusal apart from other permanent acquire failures.

## Risks / Trade-offs

- [The check and the load are two steps] A directory created after `Lstat` and before cue/load
  reads the root is not caught. → Each build under a synthetic root (the acquire's and every
  synthesis's) is checked immediately before it runs, so the window is one build per check site. Closing it needs cue/load to stop
  reading the disk beneath an overlay root; nothing in the library can close it. The guard turns a
  silent, persistent injection into a refusal in every case except a race won during one build.
- [A leftover directory blocks acquires] If something does create a synthetic root (a debugging
  session, a tool), every acquire of that module version fails until it is removed. → The error
  names the path and says to remove it.
