## Why

Enhancement 0019 Phase B makes the render one CUE build per render (D9): the build imports the instance and the platform as packages, so the kernel's inputs stop being evaluated `cue.Value`s and become source trees with a `cue.mod`. Today only `module.Module` carries its staged source (`module.Source`, stamped by the registry acquire path); `module.Instance` discards the staged tree `synth.Instance` builds, and `platform.Platform` has no source at all — `Materialize` consumes the evaluated value instead. This slice adds the source plumbing everywhere the render-build slice will consume it, with no behavior change: nothing reads the new fields yet.

This change is `library-platform-source`, the first library slice of 0019 Phase B (D9's input shape; groundwork for D5/D8/D10/D13). It lands against the current core pin and does not depend on core's `core-registry-import`.

## What Changes

- `module.Source` is broadened from "registry-staged module tree" to "the staged source tree an artifact was loaded or synthesized from": a new `Pkg` field (package directory relative to `Root`, empty means `.`), and a documented on-disk mode (`Overlay == nil` means the tree lives at `Root` on the real filesystem). Existing `HasSource()` semantics for the synth gate are unchanged.
- `module.Instance` gains `Source *module.Source`; `platform.Platform` gains `Source *platform.Source` (`platform.Source` is a type alias of `module.Source`, mirroring the existing metadata re-export pattern). Both are nil for value-constructed artifacts.
- **BREAKING (helper)** `synth.Instance` additionally returns the staged tree it built (the module's cloned overlay plus the synthesized instance package under its reserved subdirectory), instead of discarding it. `Kernel.SynthesizeInstance`'s signature is unchanged; it stamps the returned tree onto `Instance.Source`.
- `Kernel.AcquirePlatformFromDir` is added: load + shape-gate the platform package from a directory (existing `LoadPlatformPackage` path), construct via `NewPlatformFromValue`, stamp `Source{Root: <absDir>}`.
- `Kernel.AcquireInstanceFromDir` is added: load + shape-gate the instance package from a directory, process it via the validated entry point (`ProcessModuleInstance` with no extra values), stamp `Source{Root: <absDir>}`.
- Existing entry points are untouched in behavior: `LoadPlatformPackage`, `LoadInstancePackage`, `NewPlatformFromValue`, `NewInstanceFromValue`, `Materialize`, `Match`, `Compile` all work exactly as before, and nothing consumes `Source` yet.

## SemVer classification

MAJOR by the letter of Principle VI (a public helper signature changes: `synth.Instance`'s return values), pre-GA so no migration fragment is written (consumers migrate in the same PR wave; `migrations/` stays dormant per ADR-004). Everything else is additive (new fields, new kernel methods, a new alias). The kernel surface (`opm/kernel`) is purely additive.

## Affected packages and downstream consumers

- `opm/module` (Source broadened, Instance field), `opm/platform` (field + alias), `opm/helper/synth` (return plumbing), `opm/kernel` (stamping in `SynthesizeInstance`, two new acquire methods).
- **`cli` / `opm-operator`**: no action. Both go through `Kernel.SynthesizeInstance` / `Kernel.LoadPlatformPackage`, whose signatures are unchanged; a workspace grep for direct `synth.Instance` callers outside this repo is a verification task. The new acquire methods become the recommended entry points when the render-build slice lands.
- Later 0019 slices consume this one: `library-render-build` reads `Source` on both inputs for staging and `cue.mod` promotion (D13) and the skew comparison (D7/D18).

## Complexity justification

One broadened struct and two thin acquire methods, versus the alternative of the render-build slice re-deriving source locations out of band (re-loading directories it was already handed, re-staging a synth tree that was just discarded). Carrying the source on the artifact is the same call `add-registry-module-loader` already made for `module.Module` ("Carrying the staged source on the acquired *Module avoids a second registry fetch").

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `artifact-types`: artifacts gain the staged-source field (`Instance.Source`, with `module.Source` broadened); a validated instance acquire from a directory exists on the kernel.
- `platform-artifact`: `Platform` gains `Source`; a platform acquire from a directory exists on the kernel and returns a source-carrying artifact.
- `instance-synthesis`: synthesis surfaces the staged tree it builds; `Kernel.SynthesizeInstance` returns an instance whose `Source` names that tree.

## Impact

- Code: `opm/module/source.go`, `opm/module/instance.go`, `opm/platform/platform.go`, `opm/helper/synth/instance.go`, `opm/kernel/synth.go`, `opm/kernel/wrappers.go` (or a new `opm/kernel/acquire.go`), plus tests beside each.
- Docs: `opm/module/source.go` doc comments (broadened contract), `opm/helper/doc.go` and `opm/kernel/doc.go` untouched (no contract change yet); `CLAUDE.md` package notes untouched.
- No CUE, no schema pin, no fixture changes.
- Landed alongside, as an unrelated unblock in its own commit: `schema.DefaultSchemaModule` pinned to core 2.0.0-alpha.6, because 2.0.0-alpha.7 (released 2026-09-01) reshapes `#Platform.#registry` and broke the materialize tests on a clean `main`. The platform code follows the new shape in a later change, which restores the floating major.
