## 1. Spike: the render module builds from an overlay under a root that does not exist, and the shared-client shape holds

Library tests run with an absolute private `TMPDIR`
(`export TMPDIR=$(mktemp -d -p <session scratchpad>)`), and the worktree's `.cue-cache` is a copy
of the main checkout's, never a symlink. Every commit task stages the files it names with
`git add <file>`.

This change rests on one assumption (design, Research). Planning already checked it with a scratch
spike, and all four cases passed. This section lands the spike as a committed test, so the
assumption stays pinned against a CUE bump. If any case fails here, stop after this section and
report it. Do not implement a fallback that keeps a temporary directory.

- [x] 1.1 `opm/internal/renderstage/stage_test.go`: add `TestStageBuild_RenderModuleServedFromMemory`, which builds today's `Stage` output after moving it into an overlay under a synthetic root that does not exist on disk (the staging directory's prefix is rewritten in the overlay keys and in `local-module.cue`, and the directory is removed before the build). Two subtests:
  - an overlay-mode instance with an on-disk platform; `_components` holds `web` and `config`;
  - both inputs overlay-mode, built by four goroutines at once from one `Staged`.

  Assert that the root does not exist before or after the build.
- [x] 1.2 Same file: add `TestStageBuild_LocalReplacementsServedFromMemory`, the hand-written render module of `TestStageBuild_LocalReplacementsResolveInOneBuild` served from an overlay under the synthetic root. The deployment carries the replaced catalog's label. Add a negative control: the same build with `cue.mod/local-module.cue` removed from the overlay fails to resolve the instance import.
- [x] 1.3 Spike the shared-client shape in a scratch test (not committed; section 3 lands it in `cueenv`): one `modregistry.NewClientWithResolver(modconfig.NewResolver(...))` shared by two operations, each with its own `modcache.New(client, cacheDir)`, against an in-process registry behind a proxy that refuses the first request. The first operation's fetch fails, the second operation's fetch succeeds, and a cue/load build given the per-operation cache as `Config.Registry` resolves its dependencies. If it fails, stop and report.
- [x] 1.4 Run `go test -race -count=1 -run 'TestStageBuild_' ./opm/internal/renderstage/` green, then `task check` green, then commit `test(renderstage): pin that cue/load serves the render module from an overlay`.

## 2. renderstage, kernel: stage the render module in memory

- [x] 2.1 `opm/internal/renderstage/stage.go`:
  - Add the exported constant `RenderRoot`: `/opm-render`, built with the OS separator (design D1).
  - `Stage` drops its `dir` parameter and stages under `RenderRoot`: the re-keyed inputs as today, plus `cue.mod/module.cue`, `cue.mod/local-module.cue` and `render.cue` as overlay entries. It writes nothing; delete every `os.MkdirAll` and `os.WriteFile`.
  - The 0019:D13 tripwire looks up the stored `module.cue` entry and passes those bytes to `VerifyCoverage` (design D2).
  - Update the `Staged`, `Staged.Dir`, `Staged.Overlay`, `Stage` and `Build` docs.
  - Rewrite the section 1 tests onto the new `Stage`. Drop the move-into-memory helper; the tests now assert `staged.Dir == RenderRoot` directly.
- [x] 2.2 `opm/internal/renderstage` tests:
  - Move `TestStage_ServesOverlayFromMemoryAndWritesRenderModule`, `TestStage_OverlayInputsLeaveOnlyTheRenderModule`, `TestStage_OnDiskInputsCarryNoOverlay`, `TestStageBuild_OverlayInstanceServedFromMemory`, the local-replacement tests (`readPair` reads the overlay entries now) and `skip_test.go`'s `buildScenario` to the dir-less `Stage`.
  - Replace "the staging directory holds only the generated render module" with assertions on the overlay keys under `RenderRoot`, and assert that `RenderRoot` does not exist on disk.
  - `TestStage_RefusesBadInputs`: keep every case; the refusals now leave nothing to inspect.
- [x] 2.3 `opm/kernel/render.go`: delete `os.MkdirTemp`, the deferred `os.RemoveAll` and the "creating render staging directory" error, and call `renderstage.Stage(in.Instance.Source, ...)`. Drop the `os` import if it is unused. Refusal order, wraps and causes stay as they are.
- [x] 2.4 `opm/kernel` tests:
  - Replace `stagingDirs` (`render_test.go:591-621`) with a helper that points `TMPDIR` at a test-owned directory and asserts it is empty. Use it in every test that lists staging directories today (`render_test.go:621, :1037, :1150`, `render_core_floor_test.go:50`).
  - Add a test that one success, one older-core refusal, one local-replacement refusal without the opt-in, one dependency covered by no version or replacement, one skew refusal and one build failure each leave (the 0019:D13 uncovered-path tripwire cannot be reached through `Render`: promotion never produces an uncovered path, and `TestVerifyCoverage_DoctoredPromotionRefuses` pins it in `renderstage`) that directory empty and leave `renderstage.RenderRoot` absent on disk ("A render writes no staging file").
  - Add a test that two renders of the same inputs whose build fails report error positions that are byte-identical and name `RenderRoot`.
- [x] 2.5 Docs in this section (design D5, section 2 list):
  - the renderstage package doc;
  - the `Kernel.Render` doc (`opm/kernel/render.go:307-308`, `:318`, and "with no staging directory created" at `:328-329`);
  - `opm/kernel/doc.go`: the staging-directory sentences (`:157`, `:185`);
  - a dated amendment sentence on the Status of ADR-005 and of ADR-006, in the form of ADR-007's, naming this change;
  - the AGENTS.md layout line for `internal/renderstage`.
- [x] 2.6 `task check` green, then commit `refactor(render): stage the render module in memory`.

## 3. cueenv, loader, synth, kernel: one registry client per Kernel

- [x] 3.1 `opm/internal/cueenv`: add `Registry` (`NewRegistry(mapping)`), holding only the resolver and transport (`*modregistry.Client`), and `Operation` (`(*Registry).Operation()`, with `Env()`, `Init()` and the four `modconfig.CachedRegistry` methods), which wraps the shared client in a fresh `modcache.New` over the cache directory read from that operation's environment (design D3). Client construction is lazy, under a mutex. Success is kept; a construction error is returned to that call and not kept. A failed `Fetch`, `ModFile` or `ModuleVersions` drops the shared client; `FetchFromCache` errors do not. Unit tests, with a counting constructor through the unexported field:
  - one construction across many operations;
  - an error, then a retry that succeeds;
  - concurrent first use constructs once;
  - a transient failure is not remembered: against an in-process registry behind a proxy that refuses the first request, the first operation's fetch fails and a second operation's fetch of the same version succeeds;
  - a cancelled fetch is not remembered: a fetch with a cancelled context fails, and a second operation with a live context succeeds;
  - each operation reads `CUE_CACHE_DIR` when it starts.

  Add `./opm/internal/cueenv/...` to the `-race` line of `Taskfile.yml` `test` and remove it from the plain line.
- [x] 3.2 `opm/internal/loader`:
  - `Options` gains `Registry modconfig.Registry`; `LoadDir` sets `cfg.Registry`.
  - `FetchArtifact` and `FetchModule` take `opts Options` in place of `env []string`. A given registry is used for the fetch and passed on to `LoadDir`. With none, a client is built as today. A given registry with an `Init() error` method whose `Init` fails is reported as `building module registry resolver: %w`, unclassified, before the fetch.
  - Update the docs and the loader tests' call sites.
- [x] 3.3 `opm/internal/renderstage.Build` takes `loader.Options` and sets `Env` and `Registry`. In `opm/internal/synth`, `Input.Env` becomes `Input.Load loader.Options`, and `Instance` passes it to `LoadDir`. Update the call sites in the tests.
- [x] 3.4 `opm/kernel`:
  - `Kernel` gains the unexported `*cueenv.Registry` field, set in `New` from `k.registry`.
  - Add `loadOptions()`, which starts one operation: `{Env: op.Env(), Registry: op}`, or `{Env: k.loadEnv()}` with no registry on a Kernel not built by `New` (never a typed-nil interface). Each verb calls it once and routes every `LoadDir`, `FetchModule`, `FetchArtifact`, `renderstage.Build` and `synth.Input` call of that verb through the result.
  - `compileSource`, `compileSources`, `mergeSources` and `validateSources` take `loader.Options`, and the file-backed load sets `cfg.Registry` when it is set.
  - `kernel.New` still constructs no client.
- [x] 3.5 `opm/kernel` tests. In `export_test.go` add `(*Kernel).SetRegistryHooksForTest`, which installs a counting constructor (wrapping the real one) on the Kernel's `cueenv.Registry` and a per-operation wrapper that counts the calls each operation makes through the client. Add tests for:
  - "Operations on one Kernel build one client": a registry module acquire, a platform directory acquire, a synthesis with a file-backed values source and a render, giving one construction, and `Fetch` calls seen from both the fetch and the render build;
  - "Construction builds no client";
  - "A failed construction is retried": a failing constructor, then a succeeding one;
  - "A construction failure keeps its wording";
  - "The schema loader keeps its own client": `SchemaCache().Get()` makes no call through the counted client;
  - "Concurrent operations share the client": extend `TestKernel_ConcurrentAcquireAndSynth`, or add a sibling with a render, asserting one construction under `-race`;
  - "A transient fetch failure is not remembered" and "A cancelled fetch is not remembered": two `AcquireModuleFromRegistry` calls on one Kernel, the first failing (a proxy refusing once; a cancelled context on a cold cache), the second succeeding;
  - "The cache directory is read for each operation": after one operation, point `CUE_CACHE_DIR` at an empty directory; the next registry acquire fills it;
  - a zero `Kernel` value still renders (no typed-nil registry reaches cue/load).

  Audit every test that sets `CUE_REGISTRY` or `CUE_CACHE_DIR` after it constructs a Kernel that has already run an operation. Planning found none. Fix any that the suite shows.
- [x] 3.6 Docs in this section (design D5, section 3 list):
  - the `Kernel` type and `New` docs: one client (resolver and transport), built on first use, its scope, a fresh module cache per operation, which parts of the environment are read when the client is built and which at each operation, and that no failure is remembered past its operation;
  - the `WithRegistry` doc;
  - the `cueenv` package doc;
  - the `loader.Options`, `LoadDir`, `FetchArtifact` and `synth.Input` docs;
  - the AGENTS.md layout line for `internal/cueenv`.
- [x] 3.7 `task check` green, then commit `refactor(kernel): share one registry client per kernel`.

## 4. Full suite, consumers, api diff and memprobe

- [ ] 4.1 Run the full non-short suite with the network tests forced: `OPM_FLOW_TEST_FORCE=1 go test -race ./opm/...`. `TestParity_*`, `TestRender_InventoryParity`, `TestRender_SharedPlatformConcurrentRenders`, `TestRender_SharedPlatformConcurrentRendersCold`, `TestRender_ConcurrentKernelsShareNothing` and the flow test must run, not skip. Record the result in design.md "Verification".
- [ ] 4.2 Run the consumer build against fresh clones of cli `main` and opm-operator `main` (`GOTOOLCHAIN=local bash .tasks/consumer-build.sh <clone> . <scratch work dir>`); both must build and vet. Also run, against this tree, with a `replace` in a scratch copy that is never committed:
  - the cli unit packages that construct a Kernel (`internal/config`, `internal/workflow/render`, `internal/cmd/module`, `internal/cmd/instance`);
  - the opm-operator `internal/render`, `internal/controller` and `internal/platform` unit tests;
  - the opm-operator registry-backed `test/integration/reconcile` specs (`platform_transient_failure`, `platform_recovery`, `concurrent_render`), with `LOCAL_REGISTRY` set so they run rather than skip. The transient-failure spec constructs its Kernel and then points `CUE_CACHE_DIR` at an empty directory, which is the per-operation cache-directory read this change must keep.

  Cluster suites run only under `flock <session scratchpad>/kind-opm-dev.lock`. Record the consumer commits and results in design.md "Verification".
- [ ] 4.3 Run `task api:diff`. It must charge no incompatible change to this branch (every changed signature is under `opm/internal/`). Record the output in design.md "Verification".
- [ ] 4.4 Memprobe (`claude-stuff/kernel-plan-beta1/memprobe`, outside the repo). Copy it to the session scratchpad, point the copy's `replace` first at a clean checkout of `origin/main` and then at this worktree, and run the render case of each with `RUNS=3`. Quote the median heap and peak RSS for one render, before and after, in design.md "Verification". Claim no saving unless the numbers show one beyond run-to-run noise. Skip this task with a one-line note if the memprobe module cache is cold and offline.
- [ ] 4.5 `task check` green, then commit `chore(openspec): record stage-render-in-memory-and-share-registry verification`.

## 5. Verify and archive

- [ ] 5.1 Run `openspec verify` for `stage-render-in-memory-and-share-registry` (the repo's openspec-verify-change skill). Verify: no CRITICAL finding.
- [ ] 5.2 At PR time, not in the implement stage: run `openspec archive stage-render-in-memory-and-share-registry --yes`, then check that `openspec validate --specs --strict` passes, that `single-build-render` carries "Each render is its own in-memory build in its own context" and "A render writes no staging file", and that `kernel-runtime` carries "One registry client per Kernel".
- [ ] 5.3 Run the gates green, then commit `chore(openspec): archive stage-render-in-memory-and-share-registry` (the archive rides the implementing PR).
