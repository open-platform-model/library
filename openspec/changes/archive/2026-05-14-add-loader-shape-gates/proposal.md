## Why

`LoadModulePackage`, `LoadReleasePackage`, and `LoadPlatformPackage` build a CUE package, detect its apiVersion, and return the raw `cue.Value` — but never check that the package is actually the artifact type the caller asked for. Pointing `LoadModulePackage` at a Platform directory succeeds silently; the mistake surfaces far downstream, detached from its cause. A cheap shape gate at the loader boundary turns "wrong directory" into an immediate, well-attributed error instead of a confusing failure three layers up.

## What Changes

- Add a **shape gate** to all three package loaders, run immediately after the package builds and before the value is returned:
  - **Root is a struct** — reject packages that evaluate to a scalar, list, or bottom.
  - **`kind` discriminator** — assert the artifact's concrete `kind` literal matches the loader (`"Module"` / `"ModuleRelease"` / `"Platform"`). This is the missing piece: `apiVersion` identifies the schema *version*, `kind` identifies the artifact *type*.
  - **Required identity fields are concrete** — not merely present. The schema is embedded in authored packages, so every field "exists"; the meaningful check is concreteness of fields the schema never defaults:
    - Module: `metadata.name`, `metadata.modulePath`, `metadata.version`.
    - Release: `metadata.name`, `metadata.namespace`; `#module` present and itself shaped as a Module (`#module.kind == "Module"`).
    - Platform: `metadata.name`, `type`; each `#registry[id].#module` present and shaped as a Module.
- Tighten the instance-count guard from `len(instances) == 0` to `len(instances) != 1`, so a `"."` load that ever returns more than one instance is asserted rather than silently taking `[0]`.
- Add sentinel errors so frontends can branch programmatically — `ErrWrongKind`, `ErrMissingRequiredField`, `ErrInvalidPackage` — mirroring `apiversion.ErrUnknownAPIVersion`.
- **Out of scope:** full schema validation. The loader stays a shape gate; auditing `#config`/`#components`/`debugValues`/`values` against the v1alpha2 definitions remains the Kernel/Binding's job. The loader does not re-implement CUE's unification in Go.

This is **MINOR** per SemVer — additive validation, no signature change. It does tighten behavior: directories that previously loaded despite being the wrong artifact type now error. Those inputs were already malformed for the caller's purpose; the change moves the failure earlier, not into new territory.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `helper-packages`: the "Loader Reorganization Under Helper" requirement gains shape-gate behavior — the three package loaders now reject wrong-kind packages, non-concrete identity fields, and non-struct roots before returning, and expose sentinel errors for these failures.

## Impact

- **Code:** `opm/helper/loader/file/module.go`, `release.go`, `platform.go` — shared validation logic, likely extracted into a common helper to avoid copy-paste across the three near-identical functions. New sentinel errors in the same package (or `opm/errors`).
- **Tests:** new fixtures for wrong-kind directories, missing identity fields, conflicting `package` clauses, non-struct roots.
- **Downstream:** CLI (`opm module vet` and friends) and the controller call these loaders; both gain earlier, clearer errors. No call-site changes required — signatures are unchanged.
