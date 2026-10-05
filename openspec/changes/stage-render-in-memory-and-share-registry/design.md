## Context

Line numbers are at `origin/main` `84b71ac` (library v1.0.0-beta.6 plus #208 and #209).

**Render staging today.** `Kernel.render` checks the inputs and the context, then the core floor
recorded at construction (`in.Platform.CoreFloor()`, `opm/kernel/render.go:372`, from #208). Only
after that does it create `os.MkdirTemp("", "opm-render-")` (`:376-380`) and call
`renderstage.Stage(dir, ...)` (`:382`). `Stage` (`opm/internal/renderstage/stage.go:68-187`) does
the following:

- re-keys each overlay-mode input under `<dir>/instance` or `<dir>/platform` onto
  `Staged.Overlay`, and leaves an on-disk input where it is (`serveDir`, `:228-249`);
- reads the two module files and the optional `local-module.cue` views, and refuses an unopted
  replacement;
- promotes the dependency lists;
- writes `cue.mod/module.cue` and `cue.mod/local-module.cue` (`:126-145`);
- reads `module.cue` back from disk for the 0019:D13 tripwire `VerifyCoverage` (`:147-154`);
- compares skew, renders the glue and writes `render.cue` (`:181-184`).

`Build` (`:197-220`) runs `load.Instances(".")` with `Dir = ModuleRoot = staged.Dir` and the
overlay. The registry mapping reaches it as `cueenv.Override(k.registry, "")` (`render.go:406`).

**Registry clients today.**

| Load | Where | Client |
| --- | --- | --- |
| registry fetch | `loader/registry.go:94` | `modconfig.NewRegistry` per call |
| build after the fetch | `loader/registry.go:134` → `LoadDir` | cue/load's own lazy client |
| directory acquires, values-layered instance, attribution rebuild | `kernel/acquire.go:158, :441, :495` → `LoadDir` | cue/load's own |
| synthesis | `synth/instance.go:191` → `LoadDir` | cue/load's own |
| file-backed values source | `kernel/source_loader.go:101-111` | cue/load's own |
| render build | `renderstage/stage.go:201-212` | cue/load's own |
| schema load | `schema/loader.go:158` | cue/load's own, env from `Override(l.Registry, l.CacheDir)` |
| platform closure | `helper/platformmodule/closure.go:45` | caller-built `modconfig.NewRegistry` |

When `load.Config.Registry` is nil, cue/load (v0.17.1, `cue/load/config.go:427-433`) wraps
`modconfig.NewRegistry(&modconfig.Config{Env: c.Env})` in a `modconfig.LazyRegistry` for that one
load. `Config.Registry` is a public field, marked experimental. When the main module carries
replacements, cue/load wraps the given registry in a per-load replacing registry
(`config.go:603-624`); it never mutates the registry it was given.

## Goals / Non-Goals

**Goals:**

- A render writes nothing to the filesystem. The generated render module is served to the one
  build from `load.Config.Overlay` under a root that does not exist.
- One registry client per Kernel, built on first use, handed to every load and fetch the kernel
  runs.
- No change to any exported signature, to render output, diagnostics or error causes, or to the
  render's refusal order.

**Non-Goals:**

- The breaking half of the plan: the platform overlay mode, dropping `Platform.Package`, the
  operator's lease system. That is a later change.
- Folding the schema `OCILoader` or `helper/platformmodule.NewRegistry` into the kernel's client
  (D4).
- Any memory claim (see Risks).
- Any cli or opm-operator change.

## Decisions

### D1. The render module is staged under a fixed synthetic root

`renderstage` gains the constant `RenderRoot`, the absolute path `/opm-render` built with the OS
separator, like `sourcetree.SyntheticRoot`. `Stage` loses its `dir` parameter:

```go
func Stage(instance, platform *module.Source, runtimeName string, opts StageOptions) (*Staged, error)
```

It re-keys overlay-mode inputs under `RenderRoot/instance` and `RenderRoot/platform` exactly as
`serveDir` does today. It places three entries in `Staged.Overlay`:
`RenderRoot/cue.mod/module.cue`, `RenderRoot/cue.mod/local-module.cue` and
`RenderRoot/render.cue`. `Staged.Dir` is `RenderRoot`. `Build` is unchanged apart from its options
(D3): `Dir = ModuleRoot = staged.Dir`, with every overlay entry wrapped by `load.FromBytes`. Since
`Staged.Overlay` now always holds the generated files, its doc changes from "Empty when both
inputs are on disk" to "always carries the generated module".

The root is the same for every render. That is safe because an overlay belongs to one
`load.Instances` call, and nothing in cue/load keys process-wide state on the main module's root:
the module cache keys by `module@version`. The spike (Research) built four renders concurrently on
one root under `-race`.

A fixed root also makes positions in render build errors deterministic (`/opm-render/render.cue`
instead of `/tmp/opm-render-123/render.cue`).

### D2. The tripwire re-parses the bytes the build is served

The 0019:D13 tripwire exists so that coverage is checked on what cue/load will read, not on the
promotion's in-memory list. Today "what cue/load reads" is the file re-read from disk. After this
change it is the overlay entry. `Stage` therefore stores `promotion.ModuleFile()`'s bytes at
`RenderRoot/cue.mod/module.cue` first, then looks that entry up in `Staged.Overlay` and passes
those bytes to `VerifyCoverage`, with that path as the position name. The comment changes from
"re-read what was written" to "re-parse the bytes the build is served, never the in-memory
list". `TestVerifyCoverage_DoctoredPromotionRefuses` keeps the refusal pinned.

### D3. One lazily built client per Kernel, retried on a construction error

`opm/internal/cueenv` gains the client, since that package is already the one home for how the
kernel's registry environment reaches CUE:

```go
// Registry is a modconfig.CachedRegistry built on first use from env and
// shared by every load handed it. A construction error is returned to that
// call and not kept: the next call tries again.
type Registry struct {
	env []string
	mu  sync.Mutex
	reg modconfig.CachedRegistry
	new func(*modconfig.Config) (modconfig.CachedRegistry, error) // nil: modconfig.NewRegistry
}

func NewRegistry(env []string) *Registry
func (r *Registry) Client() (modconfig.CachedRegistry, error)
// ModFile, Fetch, ModuleVersions, FetchFromCache delegate to Client().
```

`kernel.New` stores `cueenv.NewRegistry(k.loadEnv())` behind a pointer field on `Kernel`. The env
slice is computed once in `New`: `cueenv.Override` copies `os.Environ()` only when `WithRegistry`
is set, and otherwise returns nil. The client itself is built on first use. With a nil env,
`modconfig.NewRegistry` reads the process environment at that moment. In both cases the process
environment is read no later than the first use of the client, which is the rule the docs state.

A new unexported `(*Kernel).loadOptions() loader.Options` returns `{Env: k.loadEnv(), Registry:
k.registryClient}` (the second field is nil on a zero `Kernel`). It replaces every
`loader.Options{Env: k.loadEnv()}` and every bare `k.loadEnv()` argument:

- `loader.Options` gains `Registry modconfig.Registry`. `LoadDir` sets `cfg.Registry` from it.
- `FetchArtifact` and `FetchModule` take `opts loader.Options` in place of `env []string`.
  - With `opts.Registry` set, `FetchArtifact` uses it.
  - Otherwise it builds a client with `modconfig.NewRegistry(&modconfig.Config{Env: opts.Env})` as
    today.
  - A construction failure keeps today's wording, `building module registry resolver: %w`, and
    stays unclassified. For a `*cueenv.Registry`, `FetchArtifact` first calls `Client()` and
    reports its error in that wording, before the fetch.
  - The staged build passes the same options to `LoadDir`.
- `renderstage.Build(cueCtx, staged, opts loader.Options)` sets `Env` and `Registry`.
  `renderstage` importing `loader` creates no cycle: `loader` imports neither `renderstage` nor
  `synth`.
- `synth.Input.Env []string` becomes `synth.Input.Load loader.Options`.
- `compileSource`, `compileSources`, `mergeSources` and `validateSources` take `loader.Options`
  in place of `env`, and the file-backed load sets `cfg.Registry`.

The client is built lazily, so `kernel.New` still evaluates nothing and fails on nothing, as the
"Kernel Type and Construction" requirement asks. The pointer field means a copied `Kernel` shares
its client, and `go vet` copylocks has nothing to flag.

A construction error is not cached. `modconfig.LazyRegistry` caches it, but then a long-running
operator would keep an invalid logins file or an unreadable cache directory for the life of the
process, while today the next operation tries again. Not caching keeps today's recovery.

Tests reach the constructor through `cueenv.Registry`'s unexported `new` field: a unit test in
`cueenv` covers counting, retry and concurrent first use. In `opm/kernel` an `export_test.go`
adds a test-only option that installs a counting constructor. It is not part of the shipped
surface, so the "Configuration Options" requirement still holds.

### D4. Scope: the kernel's own loads; the schema loader and the helper keep theirs

"One registry client per Kernel" names no exceptions. This change reads it as the kernel's own
loads: every row of the Context table except the last two.

- **`schema.OCILoader`** carries `CacheDir`, which overrides `CUE_CACHE_DIR` for the schema load
  alone (`schema/loader.go:158`). A client built from the kernel's environment would ignore that
  override. A caller-supplied `schema.Loader` is not the kernel's to configure at all. Giving
  `OCILoader` an unexported client hook would add a second way to configure the schema load, to
  save one client per process: the schema cache loads once per Kernel, and on a pinned kernel no
  verb loads it. The loader keeps its own client.
- **`helper/platformmodule.NewRegistry`** is the opt-in helper tier's caller-built client for
  closure derivation. It is not a kernel operation, and the helper tier must not reach into the
  kernel.

The proposal, the `Kernel` doc and the new `kernel-runtime` requirement state this scope, so "one
client per Kernel" is not read as "one client per process".

### D5. Docs move with the code they describe

Section 2 (staging):

- the `opm/internal/renderstage` package doc ("stages the generated render module into a
  directory", "The directory holds only the generated module");
- the `Stage`, `Staged`, `Staged.Dir`, `Staged.Overlay` and `Build` docs;
- `Kernel.Render`'s doc ("in a per-render temporary directory", "The staging directory is removed
  on return");
- the `opm/kernel` package doc (`doc.go:157` "the per-render staging directory holds only the
  generated module", `:185` "the staging directory is removed on return", and the core-floor
  sentence "with no staging directory created");
- ADR-005: a dated amendment sentence on its Status, saying rule 1's "the staging directory" no
  longer exists because a render stages in memory and writes nothing;
- ADR-006: a dated amendment sentence on its Status, saying the "Staging touches disk" negative
  is retired;
- the AGENTS.md layout line for `internal/renderstage` ("temp-dir staging" becomes "in-memory
  staging"). This is an orientation line, not a contract.

Section 3 (client):

- the `Kernel` type doc and `New` doc: one client, built on first use, its scope (D4), and that
  the process environment is read no later than that first use;
- the `WithRegistry` doc: one sentence linking the `Kernel` doc;
- the `opm/internal/cueenv` package doc;
- the `loader.Options`, `LoadDir`, `FetchArtifact` and `synth.Input` docs.

## Research & Decisions

### Can cue/load serve the whole render module from an overlay under a root that does not exist?

**Context**: The whole change depends on it. If cue/load v0.17.1 read
`cue.mod/local-module.cue` only from disk, or needed the replacement directories to exist, an
in-memory render would be impossible without keeping a temporary directory, and the change would
stop at the spike.

**Explored**: cue/load v0.17.1 source. `loadModule` opens both `cue.mod/module.cue` and
`cue.mod/local-module.cue` through `c.fileSystem.openFile` (`cue/load/config.go:581-600`), and
that is the overlay-aware filesystem. Replacement directories are served through
`c.fileSystem.ioFS(dir, ...)` (`:616-623`), which is also overlay-aware.
`checkReplaceDirModulePath` reads the replacement's `module.cue` through the same filesystem.
Then a spike run in the worktree during planning (a scratch test in `opm/internal/renderstage`,
not committed; section 1 lands it as a test). Each case built from an overlay keyed under
`/opm-render-spike/render`, a path that does not exist:

1. The current `Stage` output (overlay-mode instance, on-disk platform) moved into memory with
   its directory prefix rewritten, and the staging directory deleted first. Built, and
   `_components` held `web` and `config`.
2. Both inputs overlay-mode, built four times concurrently from one `Staged` value under `-race`.
   All passed, with no race report.
3. The hand-written render module of `TestStageBuild_LocalReplacementsResolveInOneBuild`, served
   from memory, with four on-disk directory replacements including a replaced catalog. The
   deployment carried the replaced catalog's label.
4. Negative control: case 1 with `local-module.cue` removed from the overlay. The build failed
   with `cannot find package "testing.opmodel.dev/library-render/instance@v0"`, which shows that
   the overlay's `local-module.cue` is what the build read.

`/opm-render-spike` did not exist after the run.

**Decision**: Stage the whole render module in memory (D1). No fallback is needed.

**Rationale**: Every file cue/load reads for the main module, and every replacement directory, goes
through the overlay-aware filesystem. That is the same mechanism `FetchArtifact` has used since
`add-registry-module-loader`, with `sourcetree.SyntheticRoot`, which also never exists on disk.

### Where the shared client lives and what it covers

**Context**: "One registry client per Kernel" leaves open whether it includes the schema
`OCILoader`, which has its own `CacheDir` override.

**Explored**: every `load.Config` and `modconfig.NewRegistry` site under `opm/` (Context table);
`modconfig.LazyRegistry` (`mod/modconfig/modconfig.go:375-420`), which caches a construction
error for the life of the value; cue/load's handling of a caller-supplied `Config.Registry`.

**Decision**: one `cueenv.Registry` per Kernel, built lazily and retried on error (D3). The scope
is the kernel's own loads, not the schema `OCILoader` and not the helper's closure client (D4).

**Rationale**: The kernel's module, catalog, platform, instance, values and render loads share one
client. The `OCILoader` keeps its own because its `CacheDir` override cannot be honoured by a
client built from the kernel environment. `cueenv` already owns how the kernel's registry environment reaches
CUE. Not caching the error keeps today's per-operation recovery.

### Spec delta shape

**Context**: OpenSpec 1.12 refuses a MODIFIED requirement that drops a main-spec scenario.

**Decision**:

- "Each render is its own build in its own context" is REMOVED and ADDED as "Each render is its
  own in-memory build in its own context". Its scenario "The staging directory holds no input
  files" names a directory that no longer exists, so a MODIFIED that keeps the name would state
  something false. The new requirement keeps "Repeated renders share nothing" word for word.
- "Render staging assertions observe only a test-private temp root" is REMOVED: there is no
  staging directory left to observe. It is replaced by "A render writes nothing to the
  filesystem", which keeps its intent (a test owns its temp root, and a leak fails that test).
- The two requirements whose prose says "no staging directory" are MODIFIED with every scenario
  kept, and only the wording changed.
- The client rule is a new `kernel-runtime` requirement, since "Registry Configuration Option" is
  about the mapping, not the client.

## Risks / Trade-offs

- **The environment is read once per Kernel.** `CUE_CACHE_DIR`, `CUE_REGISTRY` (when no
  `WithRegistry`) and the registry login configuration are read when the client is first built.
  A process that changes them later, or rotates a credentials file in place, keeps the old view
  until it constructs a new Kernel. Today each operation reads them again. Neither frontend
  depends on this: the operator sets `CUE_CACHE_DIR` before `kernel.New` and configures no
  registry credentials, and the cli builds one Kernel per invocation. The `Kernel` doc states the
  rule. Library tests set `CUE_REGISTRY` and `CUE_CACHE_DIR` through `registrytest` and
  `schematest` before they construct their Kernel, so the planning audit found no test that sets
  them after first use. Section 3 audits again, after the change, by running the suite.
- **A fixed root shared with the host filesystem.** cue/load reads a file the overlay does not
  carry from disk beneath the overlay root. A real `/opm-render` directory holding `.cue` files
  could therefore leak into the render package. `sourcetree.SyntheticRoot` (`/opm-registry-module`)
  has had the same exposure since `add-registry-module-loader`. A test pins that a render with no
  such directory reads nothing from disk. Detecting a hostile host directory is out of scope.
- **Sharing one client across concurrent loads.** `modcache.Cache`, the resolver
  (`registries` under a mutex) and the transport (`initOnce`, `loginsMu`, `mu`) are built for
  shared use; the `cue` command shares one client across its loads. Section 3 runs
  `TestKernel_ConcurrentAcquireAndSynth`, the two concurrent-render tests and the cold
  concurrent-render test under `-race`, now through one client.
- **No memory claim.** Staging wrote three small files, and a client is a resolver plus a
  transport. Peak memory is dominated by evaluation and by the operator's conversion step. The PR
  claims no memory saving. The verification section quotes one memprobe render before and after,
  for the record.

## Verification

Filled in by section 4.
