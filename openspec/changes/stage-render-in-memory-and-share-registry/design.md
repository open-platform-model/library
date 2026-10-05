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

- A render writes no staging file. The generated render module is served to the one
  build from `load.Config.Overlay` under a root that does not exist.
- One registry client (resolver and transport) per Kernel, built on first use, wrapped in a fresh
  module cache for each operation and handed to every load and fetch the kernel
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

`renderstage` gains the package variable `RenderRoot`: `filepath.Abs` of the separator-rooted
`opm-render`, so it is `/opm-render` on Unix and carries the current volume on Windows, where a
separator-rooted path without a volume is not absolute and cue/load refuses every overlay key
under it. `sourcetree.SyntheticRoot` gets the same treatment through the shared helper
`sourcetree.VolumeRoot`; before this change it was not absolute on Windows either. `Stage`
refuses with an error naming the path when `os.Lstat(RenderRoot)` finds anything, because
cue/load merges a real directory's entries into the overlay; a test points the root at an
existing directory holding an injected `.cue` file. `Stage` loses its `dir` parameter:

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

### D3. One resolver and transport per Kernel; a fresh module cache per operation

"Registry client" in this change means the resolver and the OCI transport behind it: the
`*modregistry.Client` that `modconfig.NewRegistry` builds from `modconfig.NewResolver`. It does
not mean the `*modcache.Cache` that `modconfig.NewRegistry` wraps around that client. The
distinction matters because the module cache remembers failures. Its `downloadZipCache` and
`modFileCache` are `par.ErrCache` values (`cuelang.org/go@v0.17.1/mod/modcache/fetch.go:52-53`),
and `ErrCache.Do` stores the error together with the value (`internal/par/work.go:117-122`). A
module cache shared for the life of an operator Kernel would therefore serve one 503, one refused
connection, one 404 before a version is published, or one cancelled context to every later fetch
of that `module@version` until the process restarts. The operator already avoids exactly this
for its platform source (opm-operator `internal/controller/platform_controller.go` builds a fresh source per reconcile for that
reason). So the Kernel shares only the part that holds no per-module state.

`opm/internal/cueenv` gains the shared client, since that package is already the one home for
how the kernel's registry environment reaches CUE:

```go
// Registry is a Kernel's one registry client: a resolver and its OCI
// transport, built on first use and shared by every operation. It holds no
// module cache; each operation gets its own (Operation).
type Registry struct {
	mapping string // the WithRegistry value; "" reads CUE_REGISTRY

	mu     sync.Mutex
	client *modregistry.Client

	newClient func(env []string) (*modregistry.Client, error) // nil: modconfig.NewResolver + modregistry.NewClientWithResolver
	wrapOp    func(modconfig.CachedRegistry) modconfig.CachedRegistry // nil: none; test-only
}

func NewRegistry(mapping string) *Registry

// Operation returns the registry one kernel operation hands to every load
// and fetch it runs. Its environment is read now (Override(mapping, "")),
// and on its first use it gets the shared client (building it if needed)
// and a fresh modcache.New(client, cacheDir), cacheDir from that
// environment by the rule cue uses: CUE_CACHE_DIR, else
// os.UserCacheDir()/cue.
func (r *Registry) Operation() *Operation

// Operation implements modconfig.CachedRegistry.
type Operation struct { ... }
func (o *Operation) Env() []string
func (o *Operation) Init() error
```

The rules the type keeps:

- **The client is built once, on first use, and a construction error is not kept.** `Init` (and
  every registry method, through it) builds the client under the mutex when there is none. A
  failure is returned to that operation and the next operation tries again.
  `modconfig.LazyRegistry` caches its error, which would keep an invalid `CUE_REGISTRY` value for
  the life of an operator process, while today the next operation tries again.
- **A failure is not remembered past its operation.** Each operation gets a fresh module cache,
  so a fetch failure is remembered only within the operation that saw it, as today. In addition,
  a `Fetch`, `ModFile` or `ModuleVersions` call that fails drops the shared client (when it is
  still the one the operation used), so the next operation builds a fresh resolver and transport.
  This covers the one failure the transport itself keeps: `cueLoginsTransport.init` records a
  credentials-file read error once per host (`mod/modconfig/modconfig.go`, `initOnce`). Dropping
  the client after a failure costs one rebuild, which is what every operation pays today.
  `FetchFromCache` reports a cache miss as an error during normal resolution, so its errors do
  not drop the client.
- **The cache directory is read for each operation.** `Operation` reads `CUE_CACHE_DIR` when the
  kernel starts the operation, as every load does today. What the Kernel reads once, on the
  client's first build, is the registry mapping (`CUE_REGISTRY` when `WithRegistry` is absent)
  and the credentials configuration.
- **One environment slice per operation.** `Operation.Env()` is the slice the cache directory
  was read from, and the kernel passes that same slice as `loader.Options.Env`, so there are not
  two snapshots in one operation. cue/load reads `Env` only to build a registry of its own
  (`cue/load/config.go:339-343`), so with a registry given it is inert, but a zero `Options`
  still uses it.

`kernel.New` stores `cueenv.NewRegistry(k.registry)` behind a pointer field on `Kernel`. A new
unexported `(*Kernel).loadOptions() loader.Options` starts one operation: with a registry it
returns `{Env: op.Env(), Registry: op}`. On a `Kernel` not built by `New` (the zero value, which
`Render` accepts today) it returns `{Env: k.loadEnv()}` and leaves `Registry` unset. It never
puts a nil pointer into the interface field, so cue/load still builds its own registry there. Each
verb calls `loadOptions()` once and passes the result to every load and fetch it runs. It replaces
every `loader.Options{Env: k.loadEnv()}` and every bare `k.loadEnv()` argument:

- `loader.Options` gains `Registry modconfig.Registry`. `LoadDir` sets `cfg.Registry` from it
  when it is set.
- `FetchArtifact` and `FetchModule` take `opts loader.Options` in place of `env []string`.
  - With `opts.Registry` set, `FetchArtifact` uses it.
  - Otherwise it builds a client with `modconfig.NewRegistry(&modconfig.Config{Env: opts.Env})` as
    today.
  - A construction failure keeps today's wording, `building module registry resolver: %w`, and
    stays unclassified. When the given registry has an `Init() error` method (a
    `*cueenv.Operation`), `FetchArtifact` calls it first and reports its error in that wording,
    before the fetch.
  - The staged build passes the same options to `LoadDir`.
- `renderstage.Build(cueCtx, staged, opts loader.Options)` sets `Env` and `Registry`.
  `renderstage` importing `loader` creates no cycle: `loader` imports neither `renderstage` nor
  `synth`.
- `synth.Input.Env []string` becomes `synth.Input.Load loader.Options`.
- `compileSource`, `compileSources`, `mergeSources` and `validateSources` take `loader.Options`
  in place of `env`, and the file-backed load sets `cfg.Registry` when it is set.

The client is built lazily, so `kernel.New` still evaluates nothing and fails on nothing, as the
"Kernel Type and Construction" requirement asks. A load that needs no registry never builds it,
as today with cue/load's own lazy registry. The pointer field means a copied `Kernel` shares its
client, and `go vet` copylocks has nothing to flag.

Tests reach the constructor through `cueenv.Registry`'s unexported fields: unit tests in `cueenv`
cover counting, retry, concurrent first use, and that neither a transient failure nor a
cancelled fetch is remembered. In `opm/kernel`, `export_test.go` adds
`(*Kernel).SetRegistryHooksForTest`, which installs a counting constructor and a per-operation
wrapper that counts the calls each operation makes through the client. Test-only seams end in
`ForTest` (`opm/kernel/kernel_test.go`, `TestKernel_ExportedSurface`), and the seam is not part of
the shipped surface, so the "Configuration Options" requirement still holds.


### D4. Scope: the kernel's own loads; the schema loader and the helper keep theirs

"One registry client per Kernel" names no exceptions. This change reads it as the kernel's own
loads: every row of the Context table except the last two.

- **`schema.OCILoader`** is configured on its own: it may override the cache directory
  (`CacheDir`, applied at `schema/loader.go:158`), and the kernel's default loader is simply
  `schema.OCILoader{Registry: k.registry}` with no override (`opm/kernel/kernel.go:68`). A
  caller-supplied `schema.Loader` is not the kernel's to configure at all. Giving `OCILoader` an
  unexported client hook would add a second way to configure the schema load, to save one client
  per Kernel: the schema cache loads once per Kernel, and on a pinned kernel no verb loads it. The
  loader keeps its own client.
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
- `Kernel.Render`'s doc (`opm/kernel/render.go:307-308` "in a per-render temporary directory",
  `:318` "The staging directory is removed on return", and the core-floor sentence at `:328-329`
  "with no staging directory created");
- the `opm/kernel` package doc (`doc.go:157` "the per-render staging directory holds only the
  generated module", `:185` "the staging directory is removed on return");
- ADR-005: a dated amendment sentence on its Status, saying rule 1's "the staging directory" no
  longer exists because a render stages in memory and writes no staging file;
- ADR-006: a dated amendment sentence on its Status, saying the "Staging touches disk" negative
  is retired;
- the AGENTS.md layout line for `internal/renderstage` ("temp-dir staging" becomes "in-memory
  staging"). This is an orientation line, not a contract.

Section 3 (client):

- the `Kernel` type doc and `New` doc: one client (resolver and transport), built on first use,
  its scope (D4), a fresh module cache per operation, that the registry mapping and credentials
  are read when the client is first built and the cache directory at each operation, and that a
  failure is not remembered past its operation;
- the `WithRegistry` doc: one sentence linking the `Kernel` doc;
- the `opm/internal/cueenv` package doc;
- the `loader.Options`, `LoadDir`, `FetchArtifact` and `synth.Input` docs;
- the AGENTS.md layout line for `internal/cueenv`, which today says the package is only the
  `CUE_REGISTRY` / `CUE_CACHE_DIR` override; it also owns the shared client now.

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
error for the life of the value; cue/load's handling of a caller-supplied `Config.Registry`;
`modconfig.NewRegistry` (`mod/modconfig/modconfig.go:362-372`), which wraps the resolver's client
in a `*modcache.Cache` whose `par.ErrCache` fields keep every fetch error with its key
(`mod/modcache/fetch.go:52-53`, `internal/par/work.go:117-122`); and the transport's
`initOnce`, which keeps a credentials read error per host.

**Decision**: one `cueenv.Registry` per Kernel, holding the resolver and transport, built lazily,
retried on a construction error and dropped after a failed registry call; a fresh module cache per
operation over it (D3). The scope is the kernel's own loads, not the schema `OCILoader` and not
the helper's closure client (D4).

**Rationale**: The kernel's module, catalog, platform, instance, values and render loads share one
resolver and transport. Sharing the module cache as well would remember a transient fetch failure
for the life of an operator Kernel, which today's per-operation client never does, and the owner
asked for this half of the plan to be non-breaking. The `OCILoader` keeps its own client because
it is configured on its own. `cueenv` already owns how the kernel's registry environment reaches
CUE. Not caching any error past its operation keeps today's per-operation recovery.

### Spec delta shape

**Context**: OpenSpec 1.12 refuses a MODIFIED requirement that drops a main-spec scenario.

**Decision**:

- "Each render is its own build in its own context" is REMOVED and ADDED as "Each render is its
  own in-memory build in its own context". Its scenario "The staging directory holds no input
  files" names a directory that no longer exists, so a MODIFIED that keeps the name would state
  something false. The new requirement keeps "Repeated renders share nothing" word for word.
- "Render staging assertions observe only a test-private temp root" is REMOVED: there is no
  staging directory left to observe. It is replaced by "A render writes no staging
  file", which keeps its intent (a test owns its temp root, and a leak fails that test).
- The two requirements whose prose says "no staging directory" are MODIFIED with every scenario
  kept, and only the wording changed.
- The client rule is a new `kernel-runtime` requirement, since "Registry Configuration Option" is
  about the mapping, not the client.

## Risks / Trade-offs

- **Part of the environment is read once per Kernel.** `CUE_REGISTRY` (when no `WithRegistry`)
  and the registry credentials configuration are read when the client is first built. A process
  that changes them later, or rotates a credentials file in place, keeps the old view until the
  client is dropped after a failed registry call or a new Kernel is constructed. Today each
  operation reads them again. `CUE_CACHE_DIR` is still read for every operation, so a consumer
  that points it somewhere else between operations is unaffected. opm-operator relies on that:
  `test/integration/reconcile/platform_transient_failure_test.go` constructs a Kernel and then
  points `CUE_CACHE_DIR` at an empty directory, expecting a real registry round trip; section 4
  runs that suite against this tree. The operator configures no registry credentials, and the cli
  builds one Kernel per invocation. The `Kernel` doc states the rule. Library tests set
  `CUE_REGISTRY` through `registrytest` and `schematest` before they construct their Kernel;
  section 3 audits that by running the suite.
- **A failure is remembered only within its operation.** One operation shares one module cache,
  so a fetch that failed is not retried inside the same operation, which is today's behaviour too
  (cue/load's own registry lives exactly one load). Across operations nothing is kept: the module
  cache is fresh, and the client is rebuilt after any failed `Fetch`, `ModFile` or
  `ModuleVersions`.
- **A fixed root shared with the host filesystem.** cue/load reads a file the overlay does not
  carry from disk beneath the overlay root. A real `/opm-render` directory holding `.cue` files
  could therefore leak into the render package, so `Stage` refuses when anything exists at
  `RenderRoot` (a check-then-build window remains; a host that can create the directory between
  the two can also edit the inputs). `sourcetree.SyntheticRoot` (`/opm-registry-module`) has had
  the same exposure since `add-registry-module-loader` and is not guarded by this change.
- **Sharing one client across concurrent operations.** The resolver (`registries` under a
  mutex) and the transport (`initOnce`, `loginsMu`, `mu`) are built for shared use. Each
  operation's module cache is its own, and concurrent operations share only the on-disk cache, as
  concurrent Kernels do today (the module cache takes a lock file per version). Section 3 runs
  `TestKernel_ConcurrentAcquireAndSynth`, the two concurrent-render tests and the cold
  concurrent-render test under `-race`, now through one client.
- **No memory claim.** Staging wrote three small files, and a client is a resolver plus a
  transport. Peak memory is dominated by evaluation and by the operator's conversion step. The PR
  claims no memory saving. The verification section quotes one memprobe render before and after,
  for the record.

## Verification

Run on 2026-10-05 against `origin/main` `84b71ac`, branch head after section 3 (`6764b2e`).

- **Full suite (4.1).** `OPM_FLOW_TEST_FORCE=1 go test -race -count=1 -v ./opm/...` with an
  absolute private `TMPDIR`: exit 0, 600 top-level tests passed, none skipped, no race report.
  Ran and passed: `TestParity_ShippedCatalog`, `TestParity_ShippedCatalogDiscriminated`,
  `TestParity_Probes`, `TestRender_InventoryParity`, `TestRender_SharedPlatformConcurrentRenders`,
  `TestRender_SharedPlatformConcurrentRendersCold`, `TestRender_ConcurrentKernelsShareNothing`,
  and the flow tests (`TestFlow_WebApp_OnOpmPlatform`, `TestFlow_ImportedModule_SynthToRender`,
  `TestFlow_ImportedModule_CatalogSubpackageImport_SynthToRender`). No test needed a change for
  the environment-read rule: the suite was green after section 3 with no fix.
- **Consumers (4.2).** Fresh clones of cli `main` `e5b8c50` and opm-operator `main` `2fe3c66`,
  `GOTOOLCHAIN=local bash .tasks/consumer-build.sh <clone> . <work dir>`: both build and vet
  against this tree. Through the same `go.work` (nothing written into either checkout):
  - cli `internal/config`, `internal/workflow/render`, `internal/cmd/module`,
    `internal/cmd/instance`: pass.
  - opm-operator `internal/render`, `internal/controller`, `internal/platform`: pass.
  - opm-operator `test/integration/reconcile` (envtest, run under the cluster lock), with the GHCR
    mapping as `CUE_REGISTRY`, an empty `CUE_CACHE_DIR` and `OPM_TEST_REGISTRY_FORCE=1` so no
    registry-backed spec can skip: 70 passed, 0 failed, 1 skipped (a spec owned by another
    change). Among them "Platform transient registry failure (registry-backed) recovers from a
    transient registry failure on the same reconciler", "Platform build recovery
    (registry-backed)" and both concurrent-render specs.
- **API diff (4.3).** `task api:diff`: "This change adds no incompatible change since
  v1.0.0-beta.6."
- **Memprobe (4.4).** The harness copied to the session scratchpad, built once against an export
  of `origin/main` `84b71ac` and once against this worktree, case `-scenario render -n 1
  -hold=true -platform two` (one cert-manager render, 42 objects), three runs each, medians in
  MiB:

  | | render peak heap | render peak live | apply retained | peak RSS (VmHWM) | render wall (ms) |
  | --- | --- | --- | --- | --- | --- |
  | before | 291.9 | 232.6 | 225.8 | 349.8 | 489 |
  | after | 295.8 | 238.1 | 225.8 | 350.1 | 481 |

  The differences are inside run-to-run spread (render peak heap ranged 272.9 to 296.7 before and
  291.6 to 307.6 after). No memory saving is claimed, as the proposal says.
