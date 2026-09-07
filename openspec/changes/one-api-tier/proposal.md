## Why

The kernel says its helpers are opt-in, but nothing can call the kernel without them, and the one verb that is missing forces the cli to fake what the helpers do. Four facts on `main` today:

1. **The helper boundary is inverted.** `opm/kernel` imports `opm/helper/loader/file`, `opm/helper/loader/registry` and `opm/helper/synth` (`acquire.go:16`, `wrappers.go:8-9`, `synth.go:9`). Every `Acquire*` signature takes `loaderfile.LoadOptions`, `SynthesizeInstance` takes `synth.InstanceInput`, and the operator branches on `loaderfile.ErrWrongKind` returned by a kernel method (`opm-operator/internal/render/kernel_package_renderer.go:63`). The cli imports `loaderfile` at nine sites and the operator at four, only to spell `LoadOptions{Registry: r}` or match a sentinel. The helper tier is the contract; `CLAUDE.md`, `opm/helper/doc.go` and the `helper-packages` spec describe a boundary the code does not have, so the SemVer labels are wrong.
2. **A second raw-value tier sits beside `Acquire*` and is nearly unused.** `LoadModulePackage` / `LoadPlatformPackage` / `LoadInstancePackage` then `NewModuleFromValue` / `NewPlatformFromValue`. The operator uses none of it. The cli uses it at five sites, one of which (`cli/internal/publish/kernel_gate.go:30`) calls the helper directly with a bare `cue.Context` and bypasses the Kernel it built ten lines earlier; two (`scaffold.go:284`, `repair.go:251`) load a module only to read two metadata strings `Module.Metadata` already holds; one (`config/platform.go:95-99`) is `AcquirePlatformFromDir` minus the Source stamp.
3. **No verb produces a Source-carrying module from a directory.** Synthesis builds the instance inside the module's own tree and accepts only overlay-mode sources (`Module.HasSource()`, `opm/module/source.go:51-53`). So the cli walks the module directory into memory itself (`cli/internal/workflow/render/module.go:139-181`, `stageLocalModuleSource`, 45 lines with its own skip list), the hand-rolled shim Principle V forbids. Since `dedupe-internals` the library carries the same walker in `opm/internal/sourcetree`, unreachable from a consumer.
4. **The cli fetches a module twice to copy it.** `cli/internal/scaffold/scaffold.go:219-268` (`copyFetched`) runs `modconfig.NewRegistry`, `Fetch` and a filesystem walk to copy a template into place, ten lines after `AcquireModuleFromRegistry` returned that module with every file in `Source.Overlay`. The library's writer for exactly that tree (`sourcetree.WriteTo`, the one the render stage uses to materialize an overlay) is internal, so the cli cannot call it. Slice 5 listed the export; it lands here because this change already changes the overlay's element type and rewrites the scaffold's acquisition.

Around them: the registry mapping lives at three altitudes (`WithRegistry`, `LoadOptions.Registry`, `OCILoader.Registry`). The cli passes the same string to all three at every site; the operator sets only the first and third (`opm-operator/cmd/main.go:253`), so its schema fetch silently reads the process `CUE_REGISTRY` while its renders use the explicit mapping. And `synth.InstanceInput.Values` is a `cue.Value` the kernel immediately renders back to text (`opm/helper/synth/instance.go:216`), forcing the operator to call `Kernel.CueContext().CompileBytes` (`kernel_module_renderer.go:112`), a method documented as "typically tests".

This is slice 2 of the eight-slice simplification plan reviewed on 2026-09-05. `cut-dead-surface` (slice 1) and `dedupe-internals` (slice 3) deferred five items to it: `LoadInstancePackage`, `LoadSourceFromBytes`'s first consumer, the second `LoadOptions`, the two file loaders that bypass `buildAndShapeGate` with the `validate.go` alias file, and the `module.Source.Overlay` type. Pre-GA, so consumers re-pin once for the whole wave.

**Scope statement (Principle VIII).** The split into an additive change (02a) and a breaking wave (02b) was proposed and the owner chose one change for the whole slice. The tasks are ordered additive first, then breaking, so every commit is small, the tree is green between commits, and a reviewer can stop after the additive group and still have a shippable library. Both consumers migrate in the same PR wave.

## What Changes

**`opm/kernel` (additive first):**

- `Kernel.AcquireModuleFromDir(ctx, dir) (*module.Module, error)`: loads a `#Module` package from a directory through the shape gate, constructs the typed module and stamps `Source` in overlay mode (`Root` = the enclosing module root, `Pkg` = the package directory relative to it, `Overlay` = every `.cue` file under `Root`), so the module is a valid `SynthesizeInstance` input. Synthesis refuses a module whose `Pkg` is non-empty: the synthesized package imports the module by its module path, which resolves to the root package only.
- `New` seeds the default schema loader from the registry: absent `WithSchemaLoader`, the cache is backed by `schema.OCILoader{Registry: <WithRegistry value>}`. An explicit `WithSchemaLoader` still wins. Behaviour fix for the operator; the cli may drop its duplicate option.
- `Kernel.InstanceInput` becomes the synthesis input type, kernel-owned: `Module`, `Name`, `Namespace`, `Values []Source`, `Labels`, `Annotations`. No `SchemaCache` field: the kernel's cache is the only one.

**`opm/kernel` (BREAKING, `refactor(kernel)!:`):**

- **BREAKING** `SynthesizeInstance(ctx, kernel.InstanceInput)`. `Values` are unified in stack order, rendered into the synthesized package, and after the build checked against the module's `#config` at the sources' own positions, exactly as `AcquireInstanceFromDir` does for its extra values. The two values paths share one merge and one attribution pass.
- **BREAKING** `AcquirePlatformFromDir(ctx, dir)` and `AcquireInstanceFromDir(ctx, dir, values ...Source)` drop the `LoadOptions` parameter; the registry mapping is the kernel's (`WithRegistry`). `AcquireOption` and `WithValues` are removed.
- **BREAKING** The raw tier is removed: `Kernel.LoadModulePackage`, `LoadPlatformPackage`, `LoadInstancePackage`, `NewModuleFromValue`, `NewPlatformFromValue`. `module.NewModuleFromValue` and `platform.NewPlatformFromValue` stay as the Source-less constructors for a value a frontend already holds.

**`opm/helper` (BREAKING):**

- **BREAKING** `opm/helper/loader/file`, `opm/helper/loader/registry` and `opm/helper/loader/internal/shape` fold into `opm/internal/loader`; `opm/helper/synth` folds into `opm/internal/synth`. `opm/helper` keeps `platformmodule` only: both consumers use it and it has no kernel dependency. The fold clears the slice 1 and 3 deferrals: the two loaders that inline the build-and-gate sequence, the `validate.go` alias file, the second `LoadOptions` copy.
- **BREAKING** The sentinels move to `opm/errors`: `ErrInvalidPackage`, `ErrWrongKind`, `ErrMissingRequiredField` (shape gate); `ErrMissingModule`, `ErrMissingName`, `ErrMissingNamespace`, `ErrMissingSource`, `ErrSchemaUnavailable` (synthesis). `ErrMissingSchemaCache` is deleted with its field. Both consumers already import `opm/errors`.

**`opm/module` (BREAKING):**

- **BREAKING** `module.Source.Overlay` becomes `map[string][]byte`. Every overlay the library builds comes from bytes; `load.FromBytes` is applied at the two build sites, and the reflection in `sourcetree.Bytes` is deleted.
- `(*module.Source).WriteTo(dir) ([]string, error)` writes an overlay-mode source into a directory and returns the dir-relative paths written, sorted (the shape `platformmodule.Files.WriteTo` already has). On-disk mode is refused: `Root` already is the directory. `sourcetree.WriteTo` is deleted and the render stage calls the method, so the library has one overlay writer and a frontend can reach it.

**Lint and docs:**

- A `depguard` rule in `.golangci.yml` forbids `opm/kernel`, `opm/internal/**`, `opm/module`, `opm/platform`, `opm/schema`, `opm/errors` from importing `opm/helper/**`, so the boundary is enforced by `task lint` rather than described.
- `opm/helper/doc.go`, `opm/kernel/doc.go`, `CLAUDE.md` (Repository Layout, Kernel API surface, Working Style), `README.md`, `docs/getting-started.md` describe the resulting surface.

**Not in this change:** `Kernel.CueContext()` (its last production use, `SchemaCache().Get(k.CueContext())`, is a `schema.Cache` API question for a later slice); the "Root plus layered overlay" Source model and a per-call `cue.Context` for acquisition (slice 6); the cli's own copies of kernel code (slice 7).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `helper-packages`: the boundary requirement gains "the kernel imports nothing under `opm/helper/`"; the layout requirement lists `platformmodule` as the only subpackage; the loader, registry-loader, shared-gate and gate-behaviour requirements are removed (the gate behaviour moves to `artifact-types`); the no-platform-synthesis requirement is restated without a synth package.
- `artifact-types`: adds module acquisition from a directory and the acquisition shape gate (sentinels in `opm/errors`); constructors are Source-less and have no kernel wrappers; `Source.Overlay` carries bytes and can be written to a directory through `Source.WriteTo`; instance acquisition takes values as a variadic `Source` list and no load options.
- `kernel-runtime`: default construction seeds the schema loader from the registry; the registry option is the one mapping for every operation; the wrapper requirement is removed; the values-input, `SynthesizeInstance`, documentation and Tier-2 requirements are restated for `kernel.InstanceInput` with `Values []Source`; the utility-methods requirement drops "load".
- `instance-synthesis`: the helper-location and caller-supplied-cache requirements are removed; the input requirement is restated as `kernel.InstanceInput`; the staged-source requirement accepts a module from either acquire verb and refuses a subpackage; the shared-gate and staged-tree requirements are restated against directory acquisition and byte overlays.
- `platform-artifact`: the platform loader requirement is removed; directory acquisition takes no load options and wraps the `opm/errors` sentinels.
- `registry-module-loading`: the public loader requirement is removed; the acquire requirement no longer promises a value-only entry point.
- `schema-dispatch`: the loader-signature requirement is removed.
- `config-validation`: `Source` is the values input for all three values-taking kernel entries.

## Impact

**SemVer:** MAJOR on the alpha line (Principle VI): exported methods, a type, two option constructors and three packages leave `opm/`; a public struct field changes type; a Kernel method's input type changes package. Pre-GA, so no migration fragment (ADR-004).

**Downstream migration cost, `cli` (13 non-test sites, 1 test file):**

- `internal/workflow/render/module.go`: `LoadModulePackage` + `NewModuleFromValue` + `stageLocalModuleSource` become `AcquireModuleFromDir`; the walker and its `load`/`fs` imports are deleted. `synth.InstanceInput` becomes `kernel.InstanceInput`.
- `internal/config/platform.go`: `LoadPlatformPackage` + `NewPlatformFromValue` become `AcquirePlatformFromDir`; sentinel checks read `oerrors`.
- `internal/scaffold/scaffold.go`, `internal/scaffold/repair.go`: `AcquireModuleFromDir`, then `Module.Metadata.ModulePath` / `.Version` instead of `LookupPath`. `scaffold.go` also keeps the module `AcquireModuleFromRegistry` returns and writes it into place with `Source.WriteTo(dest)`; `copyFetched` and its `modconfig`, `mod/module` and `io/fs` imports are deleted.
- `internal/publish/kernel_gate.go` and `publish.Options`: the gate receives the `*kernel.Kernel` `RunPublish` already builds and calls `AcquireModuleFromDir`; sentinel mapping reads `oerrors`.
- `internal/workflow/render/kernel.go`, `render.go`: drop `LoadOptions` and `WithValues`; `NewKernel` may drop its duplicate `WithSchemaLoader`.
- `internal/cmd/module/vet.go`, `internal/cmdutil/publish.go`: may drop the duplicate `WithSchemaLoader`.
- `tests/integration/{platform-build,render-parity}/main.go`, `internal/platform/seed_test.go`: same mechanical edits.

**Downstream migration cost, `opm-operator` (4 non-test sites, 1 test file):**

- `internal/render/kernel_module_renderer.go`: values become `[]kernel.Source` via `LoadSourceFromBytes(origin, values.Raw)`; the `CueContext().CompileBytes` call goes; `synth.InstanceInput` becomes `kernel.InstanceInput`.
- `internal/render/kernel_package_renderer.go`, `internal/controller/platform_controller.go`: drop `LoadOptions`; `ErrWrongKind` reads `oerrors`.
- `test/integration/reconcile/registry_helpers_test.go`: drop `LoadOptions`.
- Behaviour change to note in the operator PR: the core schema now resolves through the `WithRegistry` mapping instead of the process `CUE_REGISTRY`. The operator sets both to the same value today, so no observable change is expected; the startup smoke check (`verifyCoreSchema`) proves it.

**Library:** `opm/kernel` (`acquire.go`, `synth.go`, `wrappers.go`, `kernel.go`, `source.go`, `doc.go`), `opm/module/source.go`, `opm/errors` (new sentinels file), `opm/internal/loader` (new, from `opm/helper/loader/**`), `opm/internal/synth` (new, from `opm/helper/synth`), `opm/internal/sourcetree`, `opm/internal/renderstage/stage.go`, `.golangci.yml`, docs. Tests: about 75 `LoadOptions{}` sites, 24 `InstanceInput{}` sites and 31 raw-loader calls move onto the acquire verbs; the file and registry loader tests move under `opm/internal/loader`; `TestKernel_PrunedSurface` gains the removed names. `catalog_opm`, `modules`, `core`: no impact.

**Complexity justification (Principle VII):** net deletion of roughly 350 non-test lines (the cli walker, the cli's second fetch-and-copy, two inlined loader sequences, the alias file, the second `LoadOptions`, the five kernel wrappers, the `SchemaCache` plumbing, the overlay reflection) against one new verb of about 30 lines, one overlay writer of about 15 lines that replaces an internal one, and one merge-and-attribute helper shared by both values paths. No new option type, interface or injection slot.
