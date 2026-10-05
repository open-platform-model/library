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

- **Refuse an existing synthetic root at acquire.** Before it stages a fetched artifact,
  `FetchArtifact` checks the synthetic root with `os.Lstat`. If anything exists there (a directory,
  a file, a symlink, even a dangling one), the acquire fails with an error that names the path, and
  nothing is built. A missing root passes. Any other `Lstat` error fails the acquire and wraps the
  cause. The check covers every artifact kind `FetchArtifact` serves (module and catalog), and so
  `Kernel.AcquireModuleFromRegistry` and `Kernel.AcquireCatalogFromRegistry`.
- **Refuse an existing synthetic root at synthesis.** A registry-acquired module keeps the synthetic
  root as its `Source.Root`, and `synth.Instance` builds the synthesized instance package beneath
  it (`<root>/opm-synth-instance`), possibly long after the acquire returned and, when values
  conflict, twice in one call. A directory created at the root after the acquire would join that
  build unchecked. So `synth.Instance` runs the same check before each build whose module root is a
  synthetic root (`sourcetree.IsSynthetic`). A module acquired from a directory is not checked: its
  root exists on disk by design.
- **One check, three callers.** The check moves into the internal `sourcetree` package as one
  helper, `CheckRootAbsent(role, served, root)`, that `renderstage.Stage` (for `RenderRoot`),
  `FetchArtifact` and `synth.Instance` (for the synthetic root) call. The role and the served thing are the phrases the
  message names, so the render refusal keeps its wording byte for byte:
  `render root <path> exists on disk; the render module is served from memory under it and the
  build would read what is there, so remove it`.
- **A test hook for the base.** The base directory every synthetic root lies under becomes an
  exported package variable of `opm/internal/sourcetree`, like `RenderRoot`, so a test can point
  it at a directory that exists. Nothing else assigns it. The package is internal, so this adds no
  public API.
- **Tests.** A loader test points the base at a test-owned directory, creates the synthetic root
  of a published fixture with an injected `.cue` file in it, and asserts that `FetchModule` refuses
  with the path in the error and returns no value. The same test covers a plain file at the root.
  A catalog test does the same for `FetchArtifact` with the catalog spec. A synthesis test fetches
  a published module, then creates its synthetic root with an injected `.cue` file, and asserts
  that `synth.Instance` refuses and, once the root is gone, succeeds. A `sourcetree` unit test
  covers the helper's absent, directory, file and dangling-symlink cases and `IsSynthetic`.
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
- `instance-synthesis`: synthesis from a registry-acquired module refuses, naming the path, when
  anything exists at the module's synthetic root.
- `single-build-render`: wording only. The render-root refusal becomes its own sentence; the
  behaviour does not change.

## Impact

- Code: `opm/internal/sourcetree/sourcetree.go` (the helper, `IsSynthetic`, the exported base
  variable, the `VolumeRoot` doc), `opm/internal/loader/registry.go` and
  `opm/internal/synth/instance.go` (the check), and `opm/internal/renderstage/stage.go` (calls the
  shared helper).
- Tests: `opm/internal/sourcetree/sourcetree_test.go`, `opm/internal/loader/registry_test.go`,
  `opm/internal/synth/instance_test.go`.
  The existing `TestStage_RefusesAnExistingRenderRoot` stays as it is and must still pass.
- Consumers: the cli and opm-operator need no change. The new error is a plain error, not a
  `*FetchError`, so `errors.Classify` and `ErrTransient` treat it as not transient, which is
  right: retrying does not remove the directory.
- Residual risk: each build under a synthetic root is checked immediately before it runs, but the
  check and the load are still separate steps, so a directory created between them is not caught.
  The window is one build per check site; closing it needs cue/load to stop reading the disk
  beneath an overlay root, which is upstream's choice.
- Release class: fix.
