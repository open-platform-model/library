## Context

`synth.Instance` today fabricates a `cue.mod/module.cue` declaring only `{core, module}` and loads an overlay under a throwaway synthetic root (`/opm-synth-instance`). Because `load.Instances` requires the **main** module to declare the full transitive closure (the closure `cue mod tidy` writes at publish time), any module importing a catalog subpackage fails synthesis (library#31, verified deterministic repro).

The `registry-module-loading` capability already solved the identical problem for plain module loads: `registry.LoadModulePackage` stages the fetched module **as the main module** (overlay under a synthetic root + the module's own `cue.mod/module.cue`), so the module's tidied deps drive transitive resolution (`module.go:81-100`, `overlayFromSource`). This change makes `synth.Instance` construct the instance inside that same staged root instead of fabricating a deps-incomplete module.

Constraints: kernel neutrality/determinism (Principle I); CUE plumbing lives only in the library (Principle V); `synth.Instance`'s public signature is contractual (`instance-synthesis` spec); small-batch delivery (Principle VIII) — implementation must land as discrete commits.

## Goals / Non-Goals

**Goals:**

- `synth.Instance` resolves a module's transitive (catalog) imports with no fabricated dep list and nothing to keep in sync with `cue mod tidy`.
- Reuse the module's already-tidied `cue.mod/module.cue` via the same main-module staging the registry loader uses.
- Keep `Instance(ctx, in InstanceInput) (cue.Value, error)` and the `InstanceInput` field set unchanged; keep results observably equivalent for inputs that already succeed.
- End-to-end regression coverage for a module that imports a catalog subpackage (the #31 shape).

**Non-Goals:**

- Calling/vendoring CUE's internal `modload.Tidy` (unimportable `internal/`) or reimplementing MVS/tidy in Go.
- Changing module *publishing* (modules are already tidied at publish).
- Re-deriving or rewriting the module's dependency set.
- Reworking `Kernel.Compile` / `ProcessModuleInstance`.

## Decisions

### D1 — The acquired `*module.Module` carries its staged source

The registry loader already builds `(synthRoot string, overlay map[string]load.Source)` via `overlayFromSource`, then discards both after `BuildInstance`. Instead, attach them to the acquired module so synth can reuse them without a second fetch.

- Add a typed carrier on `opm/module`: `type Source struct { Root string; Overlay map[string]load.Source }` (or store the raw `cuelang.org/go/mod/module.SourceLoc` and build the overlay lazily in synth via a shared helper). Expose it as an optional field on `*module.Module` (`Source *Source`), nil when the module was not acquired with source (e.g. constructed from a bare value in a unit test).
- `overlayFromSource` / `syntheticRoot` move to a location both `loader/registry` and `synth` can call (e.g. a small shared `loader/internal/` helper), so staging stays single-sourced (DRY, mirrors the shape-gate single-sourcing).

*Alternatives:* (a) synth re-fetches the module source itself — double registry I/O and reintroduces the registry/`CUE_REGISTRY` coupling the current synth spec forbids; rejected. (b) Thread source through `InstanceInput` as a new field — changes the contractual input struct for data that logically belongs to the module; rejected in favor of carrying it on the module.

### D2 — New source-carrying acquire entrypoint; existing `LoadModuleFromRegistry` stays additive

`Kernel.LoadModuleFromRegistry` returns `cue.Value` and the caller separately calls `NewModuleFromValue` — the source would be lost at that boundary. Add `Kernel.AcquireModuleFromRegistry(ctx, path, version string) (*module.Module, error)` that returns a module with `Source` populated. Keep `LoadModuleFromRegistry`'s signature unchanged (still returns `cue.Value`) so the change is **MINOR**. The operator/CLI migrate their acquire call to the new method to get transitive-dep-correct synthesis.

*Alternative:* change `LoadModuleFromRegistry` to return `*module.Module` — **MAJOR**, more downstream churn; rejected unless review prefers consolidation.

### D3 — The instance source imports the module's own (now-local) package

Synth overlays the instance source (and rendered values) into a synthetic subdirectory **under the module's staged root**, e.g. `<Root>/opm-synth-instance/instance.cue`, `package instance`. It imports:

- the module's own package by its registry path+major (`<modulePath>/<snakeName>@vMajor:<snakeName>`) — which now resolves **locally** because that path matches the staged main module's `module:` line (no fabricated dep, no registry round-trip for the module itself);
- `core` by major — resolved from the **module's own tidied deps**.

The existing `moduleImportPath` / `moduleSnakeName` / `major` helpers are **retained** (still needed to address the module's package). Only `renderModuleFile` and the dep-list pinning helpers (the fabricated `deps:` block, `normalizeVersion`, `corePath` as a dep) are removed. The subdir name is a non-`_`-prefixed reserved segment so CUE does not treat it as an ignored directory and it cannot collide with a real module package.

### D4 — `#ModuleInstance` and `core` resolve from the module's deps; `SchemaCache` validates, no longer pins

Consequence of D3: `core.#ModuleInstance` now comes from the core version in the **module's** tidied `cue.mod/module.cue`, not from `SchemaCache.ResolvedVersion()`. The fabricated `core` dep pin is gone.

- `SchemaCache` is still **required** and still consulted to confirm `#ModuleInstance` is resolvable and to surface a resolved version, but it no longer pins the synth build's core.
- Within `core@v1` (additive major), the module's core and the kernel's resolved core unify downstream in `ProcessModuleInstance`/`Compile`. Cross-major (module on `@v0`, kernel on `@v1`) is an already-broader incompatibility, out of scope.

This is the most significant semantic shift and is reflected in the `instance-synthesis` spec deltas (the "core version pinned from SchemaCache" requirement is replaced).

### D5 — Single build-and-shape-gate path preserved

Synth continues to evaluate through the same build-and-shape-gate routine `LoadInstancePackage` uses (existing `instance-synthesis` requirement), via a new `BuildInstanceOverlayAt(ctx, moduleRoot, pkg, overlay, opts)` that loads the instance package from a subdirectory of the module root (the old `.`-rooted `BuildInstanceOverlay` had a single caller — synth — and is removed). Only the overlay contents and root change (module-staged root + extra instance files vs. fabricated bare root).

## Risks / Trade-offs

- **Core-version skew between module deps and kernel SchemaCache (D4)** → Within-major additive unification makes this safe in practice; add a test asserting an instance built against the module's core still passes `Kernel.Compile` under the kernel's schema. Document the skew in `MIGRATIONS.md`.
- **Module acquired without source (`Source == nil`)** → synth cannot stage-in-root. Decide between a clear typed error ("module was not acquired with source; use AcquireModuleFromRegistry") vs. a fallback. Default: **error** (deterministic, surfaces misuse) — the operator/CLI always acquire from the registry.
- **`load.Source` on the `opm/module` contract type couples it to `cue/load`** → `module.Module` already holds `cue.Value`; the coupling is consistent. If unwanted, store `module.SourceLoc` (from `mod/module`) instead and build the overlay in synth.
- **Subdir name collision / CUE ignore rules (D3)** → choose a non-`_` reserved segment; add a test that a module with an unrelated subpackage still synthesizes.
- **Scope is multi-package** → land as sequenced commits (loader surfaces source → module carrier + shared staging helper → new acquire method → synth rewrite → operator/CLI migration → regression test), each independently buildable.

## Migration Plan

1. Library: add source carrier + `AcquireModuleFromRegistry` (additive, MINOR); rewrite synth construction; keep old `LoadModuleFromRegistry`.
2. `opm-operator`: switch `moduleacquire.Acquire` to `AcquireModuleFromRegistry` so synth gets source. CLI same where it synthesizes.
3. `MIGRATIONS.md`: note the new acquire method, the `Source` field, and the core-version-source change.

- **Rollback:** the old fabricated-deps path is removed; rollback = revert the change set. Low risk — synth resolves a strict superset of inputs.

## Open Questions

- **D2:** add `AcquireModuleFromRegistry` (MINOR, recommended) or change `LoadModuleFromRegistry`'s return (MAJOR, fewer methods)? Defaulting to additive.
- **D4: LOCKED IN (2026-06-28).** The synth build's `core`/`#ModuleInstance` come from the module's own pinned core (its tidied `cue.mod/module.cue`), not the kernel's `SchemaCache`. `SchemaCache` stays required for `#ModuleInstance`-availability confirmation but no longer pins the build. Within-`core@v1` additive unification makes downstream `Compile` safe; cross-major is out of scope. Guarded by the skew-safety test (task 6.2).
- **D1 carrier shape:** `{Root, Overlay}` vs. raw `module.SourceLoc` on `*module.Module` — pick during implementation by which keeps the staging helper cleanest.
