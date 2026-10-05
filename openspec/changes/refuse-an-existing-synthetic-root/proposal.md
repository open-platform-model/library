## Why

Library PR #212 put a volume on the synthetic roots (`sourcetree.VolumeRoot`), so on Windows an
overlay key is an absolute path that cue/load accepts. That fix made registry acquisition reachable
on Windows for the first time. It also exposed a gap that the render path closed in the same PR
and the registry path did not.

`FetchArtifact` (`opm/internal/loader/registry.go`) stages every fetched module or catalog in
memory under `sourcetree.SyntheticRoot(path, version)`, which lies under `VolumeRoot("opm-registry-module")`:
`/opm-registry-module/<path>_<version>` on Unix and `C:\opm-registry-module\<path>_<version>` on
Windows. The path is deterministic, so anyone can compute it. cue/load merges a real directory's
entries beneath an overlay root into the load. On Unix, creating a directory under `/` needs root.
On Windows, any local user can create `C:\opm-registry-module`. A `.cue` file placed at a
predictable synthetic root would then join the package of every acquire of that module version, in
every process on the machine, and the acquired value would not be what the registry served.

`renderstage.Stage` already refuses when anything exists at `RenderRoot`, for exactly this reason.
`SyntheticRoot` has no such guard; PR #212 recorded it as a follow-up. It has to land before the
next library release, because that release is the first one in which a Windows frontend reaches
this path.

Two wording faults in PR #212 are fixed here too: the `VolumeRoot` doc writes the Windows result
as `\\<name>`, which reads as a UNC path, and one sentence of the single-build-render requirement
puts the refusal clause in front of the colon that introduces the list of served files.

## What Changes

- **Refuse an existing synthetic root.** Before it stages a fetched artifact, `FetchArtifact`
  checks the synthetic root with `os.Lstat`. If anything exists there (a directory, a file, a
  symlink, even a dangling one), the acquire fails with an error that names the path, and nothing
  is built. A missing root passes. Any other `Lstat` error fails the acquire and wraps the cause.
  The check covers every artifact kind `FetchArtifact` serves (module and catalog), and so
  `Kernel.AcquireModuleFromRegistry` and `Kernel.AcquireCatalogFromRegistry`.
- **One check, two callers.** The check moves into the internal `sourcetree` package as one helper,
  `CheckRootAbsent(role, root)`, that both `renderstage.Stage` (for `RenderRoot`) and
  `FetchArtifact` (for the synthetic root) call. The render refusal keeps its wording (`render root <path> exists on disk ...`).
- **A test hook for the base.** The base directory every synthetic root lies under becomes an
  exported package variable of `opm/internal/sourcetree`, like `RenderRoot`, so a test can point
  it at a directory that exists. Nothing else assigns it. The package is internal, so this adds no
  public API.
- **Tests.** A loader test points the base at a test-owned directory, creates the synthetic root
  of a published fixture with an injected `.cue` file in it, and asserts that `FetchModule` refuses
  with the path in the error and returns no value. The same test covers a plain file at the root.
  A `sourcetree` unit test covers the helper's absent, directory, file and dangling-symlink cases.
- **Doc fixes.** The `VolumeRoot` doc writes the Windows result as `C:\<name>` on the current
  volume. The single-build-render requirement's long sentence is split in two, so the refusal is
  its own sentence and the colon introduces only the list of served files.

No exported `opm/` API changes. A frontend sees a new refusal only when something exists at a
synthetic root, which nothing in OPM creates.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `registry-module-loading`: the in-memory load stages under a synthetic root that is absolute and
  absent on disk, and refuses, naming the path, when anything exists there.
- `single-build-render`: wording only. The render-root refusal becomes its own sentence; the
  behaviour does not change.

## Impact

- Code: `opm/internal/sourcetree/sourcetree.go` (the helper, the exported base variable, the
  `VolumeRoot` doc), `opm/internal/loader/registry.go` (the check), and
  `opm/internal/renderstage/stage.go` (calls the shared helper).
- Tests: `opm/internal/sourcetree/sourcetree_test.go`, `opm/internal/loader/registry_test.go`.
  The existing `TestStage_RefusesAnExistingRenderRoot` stays as it is and must still pass.
- Consumers: the cli and opm-operator need no change. The new error is a plain error, not a
  `*FetchError`, so `errors.Classify` and `ErrTransient` treat it as not transient, which is
  right: retrying does not remove the directory.
- Residual risk: the check and the load are separate steps, so a directory created between them
  is not caught. The window is the length of one build; closing it needs cue/load to stop reading
  the disk beneath an overlay root, which is upstream's choice.
- Release class: fix.
