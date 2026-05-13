## Why

Today the kernel and its convenience helpers live in flat `opm/` packages: `opm/loader/` does filesystem-coupled loading; `opm/validate/` mixes Tier-2 kernel validation with what could be helper concerns; `opm/helper/values/` (slice 05) is the first explicitly-namespaced helper. Without a clear "kernel vs. helper" boundary, downstream consumers cannot tell at a glance which packages are essential to the kernel contract and which are opinionated, opt-in conveniences. The Crossplane composition function in particular cannot use the filesystem-coupled `opm/loader/`; it needs `opm/loader/bytes/`-style alternatives.

This is slice 07 of the kernel-redesign umbrella ([001-kernel-redesign-around-platform](../../../enhancements/001-kernel-redesign-around-platform/README.md)). It reorganizes the loader and any other opt-in conveniences under `opm/helper/`, making the boundary visible in import paths and giving frontends a clean menu of what to pick up.

## What Changes

- Move `opm/loader/*` → `opm/helper/loader/file/*` (the filesystem-coupled variant retains its current behavior; it is now explicitly the `file` flavor).
- Add `opm/helper/loader/bytes/` skeleton — a future-proof slot for in-memory loading (full implementation deferred to when a Crossplane fn or test consumer demands it).
- Existing import paths break: `github.com/open-platform-model/library/opm/loader` → `github.com/open-platform-model/library/opm/helper/loader/file`. **BREAKING** at import path level. Provide a forward shim in `opm/loader/` that re-exports the new package's symbols with `// Deprecated:` doc comments, so a single SemVer cycle bridges existing consumers.
- Confirm `opm/helper/values/` (from slice 05) is in the right place; no move needed.
- Document the helper boundary: anything under `opm/helper/` is opt-in, opinionated, and a frontend MAY skip it. Anything outside `opm/helper/` is part of the kernel contract.
- This is a MAJOR change at the import-path level; bump kernel module version.

## Capabilities

### New Capabilities

- `helper-packages`: The opt-in helper layer at `opm/helper/`. Defines the boundary between kernel core and frontend convenience code. Houses loaders, values layering (already shipped via slice 05), and (later) platform composition (slice 10).

### Modified Capabilities

None.

## Impact

- **`opm/loader/`** — moved to `opm/helper/loader/file/`. A shim file at the old path re-exports symbols with deprecation notices.
- **`opm/helper/loader/bytes/`** (new, skeleton only) — empty package with a doc comment describing the intent; full implementation in a follow-up slice when a consumer needs it.
- **`opm/helper/values/`** — already correctly located; no change.
- **`opm/kernel/`** — wrapper methods that previously called `loader.*` now call `helper/loader/file.*`. Wrapper signatures unchanged; deprecation aliases keep old downstream code compiling.
- **Downstream consumers** — `cli` and `opm-operator` see `// Deprecated:` warnings on existing imports. Migration is `gofmt`-clean: change one import path and the symbols are identical.
- **Constitution Principle III (Separation of Concerns)** — strengthens the kernel/helper boundary at the package layout level.
- **Constitution Principle VI (SemVer)** — MAJOR bump for import-path break.
