## Why

Every `Kernel.Render` still touches disk. It creates a temporary directory
(`opm/kernel/render.go:376-380`), and `renderstage.Stage` writes three generated files into it:
`cue.mod/module.cue`, `cue.mod/local-module.cue` and the glue `render.cue`. It then reads
`module.cue` back for the 0019:D13 coverage tripwire (`opm/internal/renderstage/stage.go:126-184`).
The inputs themselves already reach the build from memory, through `load.Config.Overlay`. Only the
generated module still goes through the filesystem. In the operator that is a directory created
and removed on every reconcile. It is also the reason for a test-only spec requirement about
private temp roots, because test processes that share a `TMPDIR` saw each other's staging
directories.

The kernel also builds a new registry client for almost every load. `FetchArtifact` calls
`modconfig.NewRegistry` once per fetch (`opm/internal/loader/registry.go:94`). Every other
`load.Instances` call under the kernel leaves `load.Config.Registry` nil, so cue/load builds its
own lazy client for that one load (`cuelang.org/go@v0.17.1/cue/load/config.go:427-433`). That
covers the directory acquires, the registry build after a fetch, synthesis, the render build and
each file-backed values source. One operator reconcile therefore builds several resolvers, OCI
transports and auth configurations, uses each once and drops it.

This change is the non-breaking half of a two-part plan: in-memory render staging with no
temporary directory, and one registry client per Kernel. The other half (a platform overlay mode,
dropping `Platform.Package`, and with it the operator's lease system) is a later, breaking change
and is not in this one.

## What Changes

- **In-memory render staging.** `renderstage.Stage` no longer takes a directory. It stages the
  render module under a fixed synthetic root that never exists on disk, `/opm-render`
  (`renderstage.RenderRoot`). The three generated files go into `Staged.Overlay` next to the
  re-keyed overlay-mode inputs. An on-disk input is still referenced in place through its
  `local-module.cue` replacement. The 0019:D13 tripwire re-parses the `module.cue` bytes that the
  overlay serves, not the in-memory list. `Build` keeps `Dir = ModuleRoot = staged.Dir` and needs
  nothing on disk. `Kernel.Render` drops `os.MkdirTemp` and `os.RemoveAll`, together with the
  "creating render staging directory" error path. A render then writes nothing to the
  filesystem: not on success, not on a refusal, and not on a build failure.
- **One registry client per Kernel.** `kernel.New` gives the Kernel one registry client,
  built lazily. On its first use it is constructed with `modconfig.NewRegistry` from the kernel's
  load environment (`WithRegistry` applied through `cueenv.Override`). A construction error is
  returned to that call and not remembered, so a later call tries again. The kernel hands this one
  client to every load and fetch it runs:
  - `loader.LoadDir`, through a new `Registry` field on `loader.Options`;
  - `loader.FetchArtifact` and `FetchModule`, which now take `loader.Options` in place of the bare
    env slice and use the given client instead of building one;
  - `renderstage.Build`;
  - `synth.Instance`, whose `Input.Env` becomes `Input.Load loader.Options`;
  - the file-backed `compileSource` load.

  A loader or stage called without a client (internal tests, or a zero `Options`) builds one for
  itself, as it does today.
- **Scope of "one client per Kernel".** It covers the kernel's own module, catalog, platform,
  instance, values and render loads. Two clients stay outside it, on purpose:
  - The schema cache's `schema.OCILoader` keeps its own client. Its `CacheDir` field overrides
    `CUE_CACHE_DIR` for the schema load alone, so it cannot share a client built from the kernel
    environment. A caller-supplied `schema.Loader` was never the kernel's to configure.
  - `helper/platformmodule.NewRegistry` stays caller-built. It belongs to the opt-in helper tier and
    is not a kernel operation.
- **Docs.** The following say a render no longer stages to disk: the `opm/internal/renderstage`
  package doc, `Stage`, `Staged` and `Build`, the `Kernel.Render` doc, and the `opm/kernel` package
  doc (the sentences on the "staging directory" and on its removal on return). The `Kernel` type
  doc and the `WithRegistry` doc state the one-client rule, its scope and when the environment is
  read. ADR-005 and ADR-006 each get a dated amendment sentence. The AGENTS.md layout line for
  `internal/renderstage` changes from "temp-dir staging" to in-memory staging.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `single-build-render`:
  - REMOVED "Each render is its own build in its own context", and ADDED in its place "Each
    render is its own in-memory build in its own context". The old requirement's scenario "The
    staging directory holds no input files" names a directory that no longer exists, and OpenSpec
    1.12 does not let a MODIFIED requirement drop a main-spec scenario.
  - REMOVED "Render staging assertions observe only a test-private temp root", and ADDED "A
    render writes nothing to the filesystem".
  - MODIFIED "Local replacements are honoured only when the caller opts in" (the refusal stages
    nothing; scenario kept).
  - MODIFIED "A render refuses a platform whose core predates the provider count" (the refusal
    stages nothing; every scenario kept).
- `kernel-runtime`: ADDED "One registry client per Kernel".

## Impact

- Packages: `opm/internal/renderstage` (`stage.go`, `doc.go`, tests), `opm/kernel` (`render.go`,
  `kernel.go`, `acquire.go`, `synth.go`, `validate.go`, `source_loader.go`, `doc.go`, tests),
  `opm/internal/loader` (`load.go`, `registry.go`, tests), `opm/internal/synth` (`instance.go`,
  tests) and `opm/internal/cueenv` (the lazy client and its tests). Also `adr/005`, `adr/006`,
  `AGENTS.md` and `Taskfile.yml`, which runs `opm/internal/cueenv` under `-race`.
- Public surface: none. Every changed signature is under `opm/internal/`. `kernel.New`,
  `WithRegistry` and every verb keep their signatures. `Kernel` gains one unexported pointer
  field, so copying a `Kernel` value stays legal.
- Behaviour:
  - Render output, diagnostics and error causes are unchanged.
  - Positions in render build errors now name the deterministic `/opm-render/...` paths instead
    of a random temporary directory. Two renders of the same inputs therefore report identical
    positions.
  - The kernel reads the process environment (`CUE_CACHE_DIR`, `CUE_REGISTRY` when `WithRegistry`
    is absent, and the CUE config directory for registry logins) once, when its client is first
    built. A later change to that environment no longer reaches the kernel's loads. Both
    frontends set it before constructing their one Kernel: the operator sets `CUE_CACHE_DIR` in
    `cmd/main.go` before `kernel.New`, and the cli builds its Kernel per invocation. The `Kernel`
    and `WithRegistry` docs state the rule.
  - No memory saving is claimed. Peak memory is dominated by evaluation and conversion, not by
    staging or client setup. The verification section quotes a memprobe before and after for one
    render.
- Downstream: cli and opm-operator need no change. The consumer build is run against both
  `main`s.
- SemVer: PATCH (an internal refactor). Release class `refactor`.
