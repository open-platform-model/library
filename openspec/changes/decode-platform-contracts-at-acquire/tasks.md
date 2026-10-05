## 1. platform: record the core floor and the contract inventory at construction

Library tests run with an absolute private `TMPDIR`
(`export TMPDIR=$(mktemp -d -p <session scratchpad>)`), and the worktree's `.cue-cache` is a copy
of the main checkout's, never a symlink. Every commit task stages the files it names with
`git add <file>`.

- [ ] 1.1 `opm/platform/platform.go`: add the unexported `once sync.Once` and `recorded facts` fields and the `facts` type (design D1). `NewPlatformFromValue` decodes the metadata as today, then runs `p.once.Do` with the decode. A contracts refusal or decode error is stored and never returned from construction.
- [ ] 1.2 `opm/platform/contracts.go`: move the current `Contracts()` body into an unexported `decode(v cue.Value) facts`. It records `providedBy` presence, the inventory, a missing-field refusal as data (`field`, `since`), or the decode error (design D1, D3). `Contracts()` runs `p.once.Do` (the lazy path of design D2), then returns a deep copy of the inventory (an unexported `clone`, covering every map, slice and `Comparable` row) or a fresh `*oerrors.PlatformCoreTooOldError` built from the recorded data with `p.name()`. The clone keeps nil and empty apart (`maps.Clone` and `slices.Clone` do): a decoded empty map stays empty, a nil one stays nil. Add `(*Platform).CoreFloor() error` with the doc comment of design D4. Check that the error texts are byte-equal to today's.
- [ ] 1.3 Docs in this section (design D5, section 1 list):
  - the `Platform` doc: decoded once at construction, a cache like `Metadata`, re-run the constructor after changing `Package`, use through a pointer and never copy;
  - the `Contracts`, `ContractInventory` and `opm/platform` package docs: no "on demand", no "never at construction";
  - the `schema.Contracts` comment in `opm/schema/paths.go`;
  - the two `opm/catalog` comments that cite `Platform.Contracts` as their precedent (`catalog.go`, `provides.go`), which keep their own on-demand rule.
- [ ] 1.4 Tests in `opm/platform` (`contracts_test.go`, `platform_test.go`), one per `platform-artifact` scenario that needs no kernel:
  - zeroing `Package` after `NewPlatformFromValue` leaves `Contracts()` equal and `CoreFloor()` nil;
  - mutating a returned inventory (a map entry, a truncated slice, a `Comparable` row's `Contracts`) does not change the next call's result, and the copy keeps nil and empty apart;
  - `&platform.Platform{}` returns the `#contracts` refusal, and its `CoreFloor()` returns the `providedBy` refusal;
  - a struct literal over a valid value returns the same inventory as the constructed platform;
  - a value lacking `providedBy` constructs without error, and `Contracts()` and `CoreFloor()` both return the typed refusal with field `providedBy`;
  - a value whose `#contracts` fails to evaluate but carries `providedBy` constructs without error, `Contracts()` returns the "did not evaluate" error, and `CoreFloor()` returns nil;
  - concurrency: goroutines calling `Contracts()` and `CoreFloor()` on one constructed platform and on one struct-literal platform at the same time, run under `-race`.

  `Taskfile.yml` `test`: move `./opm/platform/...` from the plain `go test` line to the `-race` line, so `task test` (which CI runs) checks the concurrency scenario on every push.

  Update the `TestPlatform_FieldSet` comment: the exported set is unchanged, and the inventory is recorded in unexported fields.
- [ ] 1.5 `task check` green (its `test` step now runs `opm/platform` under the race detector), then commit `feat(platform): decode the core floor and contract inventory at construction`.

## 2. kernel: Render reads the recorded core floor

- [ ] 2.1 `opm/kernel/render.go`: replace the `Package` lookup with `in.Platform.CoreFloor()`, wrapped as `render refused before staging: %w`, at the same point: after the `ctx.Err()` check and before the staging directory (design D4). Delete `platformMetadataName` if nothing else reads it. Drop the `opm/schema` import if it is unused.
- [ ] 2.2 Docs in this section (design D5, section 2 list):
  - the `Render` doc and the floor comment in `render.go`;
  - the `opm/kernel` package doc (`doc.go`: the "with one exception: Render reads ... Package" sentence in "Every operation shares nothing", the Goroutine safety sentence that a render "reads the shared Package only for the core floor", which now says a render reads no `Package` and the floor reads the fact recorded at construction, and the core-floor paragraph);
  - the `schema.ContractsProvidedBy` comment;
  - a dated amendment sentence on ADR-007's Status paragraph, in the form of the existing one: "Amended 2026-10-05 by `decode-platform-contracts-at-acquire` (owner decision h4 of the beta.1 kernel checklist walkthrough): `Render` reads no `Package`; the core floor reads the fact the platform recorded at construction." The 2026-09-30 sentence stays as history.

  AGENTS.md is not edited: its `platform/` line stays true, and the `Platform` godoc is the contract's one home.
- [ ] 2.3 Tests in `opm/kernel`:
  - `render_core_floor_test.go`: keep `TestRender_OlderCorePlatformRefusedBeforeStaging` and `TestRender_UnnamedOlderCorePlatformSameMessageAsContracts` unchanged; the struct-literal case is now the pin for the hand-built scenario. Add a test that an acquired current-core platform with its `Package` replaced by the zero `cue.Value` renders the same objects as the unchanged platform ("A render reads no platform Package"). Add a test that acquiring the alpha.10-pinned platform yields equal typed errors from `CoreFloor()` and from `Render`'s unwrapped cause. Update the race-test comments (`TestRender_SharedPlatformConcurrentRenders` and its cold sibling): the floor reads recorded Go fields, not `Package`.
  - `render_core_floor_test.go` `acquireOlderCorePlatform`: reword the helper comment and the require message ("acquisition does not read the inventory") to say that acquisition records the refusal and does not return it.
  - `render_collision_test.go`: the hand-built old-core colliding platform still constructs, passes the floor and fails in the build. Update its comment ("construction reads metadata only") to the new construction.
- [ ] 2.4 `task check` green, then commit `feat(kernel): read the recorded core floor in Render`.

## 3. Full suite, consumer builds and api diff

- [ ] 3.1 Run the full non-short suite with the network tests forced: `OPM_FLOW_TEST_FORCE=1 go test -race ./opm/...`. `TestParity_*`, `TestRender_InventoryParity` and the flow test must run, not skip. Record the result in design.md "Verification".
- [ ] 3.2 Run the consumer build against fresh clones of cli `main` and opm-operator `main` (`GOTOOLCHAIN=local bash .tasks/consumer-build.sh <clone> . <scratch work dir>`). Both must build and vet. `go vet`'s copylocks must report no copy of a `Platform` (design D2). Also run opm-operator's `internal/controller` and `internal/platform` unit tests against this tree (a `replace` in a scratch copy, never committed), including the `platform_inventory_test.go` zero-platform test. Record the consumer commits and results in design.md "Verification".
- [ ] 3.3 Run `task api:diff`. It must list `(*Platform).CoreFloor` as a compatible addition and no incompatible change charged to this branch. Record the output in design.md "Verification".
- [ ] 3.4 `task check` green, then commit `chore(openspec): record decode-platform-contracts-at-acquire verification`.

## 4. Verify and archive

- [ ] 4.1 Run `openspec verify` for `decode-platform-contracts-at-acquire` (the repo's openspec-verify-change skill). Verify: no CRITICAL finding.
- [ ] 4.2 At PR time, not in the implement stage: run `openspec archive decode-platform-contracts-at-acquire --yes`, then check that `openspec validate --specs --strict` passes and that `platform-artifact` carries "A platform records its core floor and contract inventory at construction".
- [ ] 4.3 Run the gates green, then commit `chore(openspec): archive decode-platform-contracts-at-acquire` (the archive rides the implementing PR).
