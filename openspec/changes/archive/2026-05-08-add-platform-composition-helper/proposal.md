## Why

Slice 09 (`rewrite-match-around-platform`) made `*platform.Platform` the matcher's required input. But constructing a Platform — taking a "shell" Platform definition (metadata, type, ctx) and filling its `#registry` with a list of registered Modules so the computed views (`#composedTransformers`, `#matchers`) populate — is non-trivial CUE wiring. Each frontend (`cli`, `opm-operator`, planned Crossplane fn) would otherwise reinvent the same loop: load the shell, FillPath each Module into `#registry`, evaluate, return.

This is slice 10 of the kernel-redesign umbrella ([001-kernel-redesign-around-platform](../../../enhancements/001-kernel-redesign-around-platform/README.md)). It introduces `opm/helper/platform/Compose` so every frontend has a one-line composition path.

## What Changes

- Add `opm/helper/platform/` package with:
  - `func Compose(k *kernel.Kernel, shell *platform.Platform, modules []*module.Module) (*platform.Platform, error)` — takes a Platform shell (an unfilled Platform whose `#registry` is empty or partial) and a list of Modules to register, returns a fully-composed Platform with `#registry` populated and all computed views resolved.
  - Internally: for each `Module`, FillPath into `shell.Package` at `binding.Paths().Registry[<id>]` with a constructed `#ModuleRegistration` carrying the Module value; evaluate; check for unification errors (multi-fulfiller failures per catalog D13); return the new Platform.
- Add `(k *Kernel) ComposePlatform(shell *Platform, modules []*Module) (*Platform, error)` thin wrapper.
- This is a MINOR change — additive helper. No public API breakage.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `helper-packages`: Adds `opm/helper/platform/` and its `Compose` function to the helper layer.

## Impact

- **`opm/helper/platform/` (new)** — `Compose` function and any internal helpers.
- **`opm/kernel/`** — `(k *Kernel) ComposePlatform` wrapper.
- **`opm/api/v1alpha2/`** — confirms `Paths().Registry` resolves to the right path for FillPath; may add a small helper for building a `#ModuleRegistration` value.
- **Downstream consumers** — `cli` (in `opm-cli` workflows that construct platforms), `opm-operator` (when reconciling ModuleRelease CRs and updating the platform-level registry), and the future Crossplane fn each replace ~10 lines of CUE wiring with a single call.
- **Constitution Principle V (CUE-Native Module Resolution)** — composition uses CUE's FillPath natively; no parallel data structure.
- **Constitution Principle VII (YAGNI)** — this helper has three documented consumers; not speculative.
