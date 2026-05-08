## 1. Pre-Slice Snapshot

- [x] 1.1 Run the full test suite and capture rendered output for every fixture; commit snapshots to a temporary location for byte-comparison post-rewrite
- [x] 1.2 List every test fixture currently using `*provider.Provider`; this is the migration worklist for tasks group 6

## 2. Match Algorithm Rewrite

- [x] 2.1 Refactor `pkg/render/match.go` to walk `Platform.#matchers` via `binding.Paths().Matchers` instead of iterating a Provider's transformer list
- [x] 2.2 Implement Resource demand walking: collect FQN keys from each component's `#resources` map
- [x] 2.3 Implement Trait demand walking: collect FQN keys from each component's `#traits` map
- [x] 2.4 For each demanded FQN, look up the candidate list and route to matched / unmatched / ambiguous
- [x] 2.5 Preserve `requiredLabels` matching behavior — the existing label-match code stays in place
- [x] 2.6 Remove every reference to `*provider.Provider` from `pkg/render/match.go`

## 3. Execute Phase Rewrite

- [x] 3.1 Update `pkg/render/execute.go` to resolve transformers by FQN from `Platform.#composedTransformers` via `binding.Paths().ComposedTransformers`
- [x] 3.2 Remove every reference to `*provider.Provider` from `pkg/render/execute.go`
- [x] 3.3 Confirm FillPath / decode / emit behavior is otherwise unchanged

## 4. Runtime Helper Update

- [x] 4.1 Update `render.Module` (in `pkg/render/module.go`) to hold `*platform.Platform` instead of `*provider.Provider`
- [x] 4.2 Update `render.NewModule` signature: `func NewModule(plat *platform.Platform, runtimeName string) *Module`
- [x] 4.3 Update internal field accesses
- [x] 4.4 Update `process_module.go` / `compile_module.go` (renamed in slice 06) to construct via the new signature

## 5. Phase Input Struct Updates

- [x] 5.1 Remove `Provider` field from `MatchInput`, `PlanInput`, `CompileInput`
- [x] 5.2 Remove godoc disclaimers stating `Platform` is optional today; mark required
- [x] 5.3 Update kernel methods (`Match`, `Plan`, `Compile`) to read from `in.Platform` only

## 6. Fixture Migration

- [x] 6.1 For each fixture in the worklist (task 1.2), construct an equivalent `platform.cue` artifact carrying a `#registry` with the pre-existing transformers
- [x] 6.2 Update test loaders: replace `loaderfile.LoadProvider(...)` with `loaderfile.LoadPlatformFile(...)` + `platform.NewPlatformFromValue(...)`
- [x] 6.3 Run each fixture's test; compare rendered output to pre-slice snapshots from task 1.1; assert byte-equal
- [x] 6.4 For any fixture that diverges, investigate and document the cause before proceeding

## 7. Provider Package Deletion

- [x] 7.1 Delete `pkg/provider/` directory entirely
- [x] 7.2 Delete `pkg/helper/loader/file/provider.go`
- [x] 7.3 Remove `LoadProvider` re-export from the `pkg/loader/` deprecation shim
- [x] 7.4 `grep -rn "provider.Provider" pkg/` and confirm zero hits
- [x] 7.5 `grep -rn "LoadProvider" pkg/` and confirm zero hits

## 8. Kernel Wrapper Cleanup

- [x] 8.1 Remove `(k *Kernel) NewRenderModule` wrapper if it took `*Provider`
- [x] 8.2 Remove `(k *Kernel) LoadProvider` wrapper
- [x] 8.3 Confirm kernel public surface contains no `Provider` mentions

## 9. Documentation

- [x] 9.1 CHANGELOG entry: MAJOR bump; Provider retired; match consumes Platform; phase inputs require Platform; migration recipe pointing at slice 10's `helper/platform.Compose`
- [x] 9.2 Update `library/README.md` Quick Start to use Platform throughout
- [x] 9.3 Update umbrella enhancement to mark Provider as fully retired
- [x] 9.4 `pkg/render/match.go` package doc: describe the algorithm clearly; cross-reference catalog 014's `#PlatformMatch` as the conceptual source

## 10. Validation

- [x] 10.1 Run `task fmt`
- [x] 10.2 Run `task vet`
- [x] 10.3 Run `task lint`
- [x] 10.4 Run `task test`; investigate every failure
- [x] 10.5 Run `task check`
- [x] 10.6 `go build ./...` from repo root
- [x] 10.7 Final byte-equal snapshot comparison: every fixture's rendered output matches the pre-slice snapshot from task 1.1
