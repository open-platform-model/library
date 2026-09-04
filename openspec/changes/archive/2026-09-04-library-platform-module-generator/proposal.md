## Why

Under 0019 D5 a platform is a CUE module that imports its catalogs; the kernel's only platform input is a module directory acquired with `AcquirePlatformFromDir`. Every frontend that starts from a *coordinate* (the operator's Platform CR, the CLI's cluster-CR fallback, the CLI's seeded local default) therefore has to generate that module: emit `cue.mod/module.cue` with the full, tidied dependency closure and a `platform.cue` importing each catalog. The operator wrote that generator for D6 (`opm-operator/internal/platformmodule`: `Generate`, `Roots`, `Closure`) and its design recorded that the CLI needs the identical derivation and that the generator should be lifted into a library helper at that point. That point is now: `cli-render-switch` renders against the cluster CR and cannot import operator internals, and the CLI rule is that structure lives in the library, never in a frontend fork.

A second gap blocks the same CLI switch. `Kernel.Render` requires a source-carrying instance, but the CLI's instance-file commands layer extra values files (`-f`) onto an authored instance package, and `AcquireInstanceFromDir` is specified as "no extra values supplied". The kernel already has the mechanism (`synth.Instance` overlays a rendered `values.cue` into a staged package and builds through the shape gate); it is not reachable for an on-disk instance.

## What Changes

- **New helper `opm/helper/platformmodule`** (the `opm/helper/<name>/` convention): a pure, deterministic platform-module generator plus the dependency-closure derivation, lifted from the operator. Typed input (platform name, type, registry entries with path, version, enable; the module path the caller owns), the resolved dependency list, deterministic file bytes out (`cue.mod/module.cue` in modfile canonical format, `platform.cue` embedding `core.#Platform` and importing every catalog under positional aliases). `Closure` derives the module's full dependency list from the pinned modules' published module files (breadth-first over `ModFile`, maximum version per major-qualified path, roots participating, no default-major markers), the tidy a `cue mod tidy` would produce, computed once at generation (0019 D13). A `Files.WriteTo(dir)` writes the module into a caller-owned directory; directory lifecycle (generations, staging swaps, retention) stays with the frontend.
- **The core pin defaults from the library.** The generated module pins `opmodel.dev/core@v2` at the version carried by `schema.DefaultSchemaModule`, the release the render glue was verified against, unless the caller overrides it. Frontends stop carrying a core-pin constant of their own.
- **Kernel neutrality of the closure's registry access.** The module-file source is built from caller-supplied registry mapping, client type and process environment (`modconfig.Config.Env`), never from hidden lookups; the operator's hard-coded client type becomes an argument.
- **`AcquireInstanceFromDir` accepts extra values sources.** A new option layers caller-supplied values (the same `Source` type `ValidateConfigDetailed` takes) onto an on-disk instance package as an overlay `values.cue` in the package directory, built in one pass through the instance shape gate, and returns an instance whose `Source` is overlay mode (root and package as before, overlay carrying the on-disk files plus the rendered values). Without the option, behaviour is unchanged.
- `helper-packages`' "No platform synthesis helper" requirement keeps its substance (no `PlatformInput`, no synthesis into a value, no `SynthesizePlatform`) and drops the sentence that frontends generate the module *themselves*.

Not in scope: the operator's re-point onto the helper (`operator-platformmodule-lift`, deletes its copy and `CoreVersion`), the CLI's consumption (`cli-render-switch`, `cli-config-platform-module`), and the workspace pin tooling that bumps the operator constant (retired by the operator change).

## SemVer classification

MINOR. Two additive surfaces (`opm/helper/platformmodule`, an `AcquireInstanceFromDir` option). No existing signature changes; `AcquireInstanceFromDir` gains a variadic option so every current caller compiles unchanged.

## Complexity justification

The helper is a lift, not new machinery: the operator's package minus its process-policy half (`Layout`). The acquire option reuses `synth`'s overlay construction and the loader's overlay build; it adds one exported option and one private overlay builder. Both remove a fork the next consumer would otherwise write (Principle VII).

## Capabilities

### New Capabilities

- `platform-module-generation`: generating a D5-shaped platform CUE module (files and dependency closure) from typed registry entries, deterministic per input, with the core pin defaulting to the kernel's verified release.

### Modified Capabilities

- `helper-packages`: "No platform synthesis helper" no longer states that frontends generate the platform module themselves; the generator helper is the sanctioned way, hand-written modules remain valid.
- `artifact-types`: "Instance acquisition from a directory" gains the extra-values option and its overlay-mode `Source`.

## Impact

- New: `opm/helper/platformmodule/{doc,generate,closure,write}.go` + tests (`generate_test`, `closure_test` with a fixture module-file graph, an in-process `registrytest`-served build test proving the emitted module builds through `AcquirePlatformFromDir` and pins the catalog's transitive `cue.dev/x/k8s.io` dependency).
- `opm/kernel/acquire.go` (option + overlay path), `opm/helper/synth/instance.go` (values rendering shared, not duplicated), `opm/helper/loader/file/build.go` (no change expected; `BuildInstanceOverlayAt` already exists), `opm/helper/doc.go`, `CLAUDE.md` layout table, `docs/getting-started.md` (platform-from-coordinate example).
- Downstream: `opm-operator` deletes `internal/platformmodule/{generate,closure}.go` and `CoreVersion`; `cli` builds cluster-CR and local platforms through the helper and layers `-f` values through the acquire option. Neither is forced by this release (additive).
- `enhancement.yaml` declares 0019 D13 (the closure is the once-at-generation tidy D13 names) and D5 (the generated shape).
