# helper-packages Specification

## Purpose
The `opm/helper/` subdirectory is the opt-in convenience boundary of the OPM library. Subpackages under `opm/helper/` are opinionated frontend conveniences that wrap kernel primitives for specific embedding patterns; a frontend MAY skip them and call the kernel directly. The library has three tiers. The kernel (`opm/kernel`, `opm/module`, `opm/platform`, `opm/catalog`, `opm/schema`, `opm/errors` and `opm/internal/**`) binds every frontend (CLI, controller, Crossplane composition function, future runtimes). The opt-in helper tier binds none. The Kubernetes tier, `opm/k8s/` (ADR-011, spec `kubernetes-tier`), binds every frontend that targets Kubernetes. This spec owns the helper boundary. It keeps the kernel's public surface small and lets the helper layer evolve independently. Each new helper subpackage requires its own change.

## Requirements

### Requirement: Helper Boundary at opm/helper/

The library SHALL maintain a `opm/helper/` subdirectory whose subpackages are opt-in, opinionated frontend conveniences. The packages outside `opm/helper/` SHALL fall into two tiers: the kernel (`opm/kernel`, `opm/module`, `opm/platform`, `opm/catalog`, `opm/schema`, `opm/errors` and `opm/internal/**`), whose contract every frontend MUST honour, and the Kubernetes tier under `opm/k8s/` (ADR-011, requirement "The Kubernetes tier binds a Kubernetes frontend" in `kubernetes-tier`), whose contract every frontend that targets Kubernetes MUST honour. Neither tier SHALL be described as opt-in. The helper boundary SHALL be real in the import graph: no kernel-tier package SHALL import a package under `opm/helper/` (the Kubernetes tier is held to the same rule by `kubernetes-tier`), no exported kernel signature SHALL name a type declared under `opm/helper/`, and no kernel operation SHALL return an error whose sentinel is declared under `opm/helper/`. The rule SHALL be enforced by the repository lint gate, not only by documentation. The exported packages outside `opm/helper/` and `opm/k8s/` SHALL be exactly `opm/kernel`, `opm/module`, `opm/platform`, `opm/catalog`, `opm/schema` and `opm/errors`: no publish-side or presentation package SHALL ship in the library.

#### Scenario: Helper boundary documented

- **WHEN** a developer reads `opm/helper/doc.go`
- **THEN** the file documents that anything under `opm/helper/` is opt-in and a frontend MAY skip it
- **AND** documents that the kernel outside `opm/helper/` binds every frontend and that the `opm/k8s/` tier binds every frontend that targets Kubernetes, without stating that everything outside `opm/helper/` is kernel contract

#### Scenario: Kernel imports nothing under the helper tree

- **WHEN** the dependency list of `opm/kernel` and of every package under `opm/internal/` is computed
- **THEN** no import path under `opm/helper/` appears in it

#### Scenario: A frontend that skips the helper tree can drive the whole pipeline

- **WHEN** a frontend imports only `opm/kernel`, `opm/module`, `opm/platform`, `opm/schema` and `opm/errors`
- **THEN** it can acquire a module from a directory or the registry, acquire a platform and an instance from directories, synthesize an instance, validate values and render, and branch on every failure class the kernel reports

#### Scenario: Lint gate refuses a helper import from the kernel

- **WHEN** a change adds an import of a package under `opm/helper/` to `opm/kernel` or `opm/internal/**`
- **THEN** the repository lint task fails naming the forbidden import

#### Scenario: No publish-side package in the library

- **WHEN** a consumer lists the exported packages under `opm/`
- **THEN** neither `opm/compat` nor `opm/core` exists, and the catalog compatibility comparator is reachable only through the cli

### Requirement: Helper Layout for Future Subpackages

Future opt-in helpers SHALL follow the `opm/helper/<name>/` convention. Subpackages SHALL be added by their owning slices and not as part of the originating slice that established the convention. Past examples of helper subpackages in this convention SHALL reflect the current package layout; subpackages that have been collapsed into the kernel (the previous `opm/helper/values/`, `opm/helper/loader/file/`, `opm/helper/loader/registry/` and `opm/helper/synth/`) SHALL NOT appear as exemplars. The current subpackages are `platformmodule` (platform module generation from catalog coordinates) and `objectset` (duplicate rendered object identities). `objectset` is Deprecated: the check's home is `opm/k8s/object` (ADR-011 item 9), and the helper copy stays only until both frontends have migrated (`duplicate-object-identities`).

#### Scenario: Platform helper landing place

- **WHEN** a frontend needs to generate a platform module from catalog coordinates
- **THEN** `opm/helper/platformmodule/` is the directory that helper occupies

#### Scenario: Duplicate-identity helper landing place

- **WHEN** a runtime needs to refuse a render whose objects share one apply identity
- **THEN** `opm/k8s/object` is the package that provides the check, the deprecated `opm/helper/objectset/` names it as the replacement, and the kernel calls neither

#### Scenario: Values helper subpackage no longer exists

- **WHEN** a developer searches `opm/helper/` for a `values` subpackage
- **THEN** no `opm/helper/values/` directory exists
- **AND** the canonical implementation of layered values validation lives at `Kernel.ValidateConfigDetailed` in `opm/kernel/`

#### Scenario: Loader and synth subpackages no longer exist

- **WHEN** a developer lists the subpackages of `opm/helper/`
- **THEN** exactly `platformmodule` and `objectset` are present, `objectset` marked Deprecated
- **AND** neither `opm/helper/loader/` nor `opm/helper/synth/` exists

### Requirement: No platform synthesis helper

The library SHALL expose no function that turns typed subscription inputs into a platform `cue.Value` without a module on disk, and the kernel SHALL expose no `SynthesizePlatform`. A platform is a CUE module on disk that imports its catalogs (0019 D5/D6); a frontend that starts from catalog coordinates generates that module through `opm/helper/platformmodule` (`platform-module-generation`) or writes it by hand, and acquires it with `AcquirePlatformFromDir`.

#### Scenario: Synth surface is instance-only

- **WHEN** a consumer inspects the exported methods of `Kernel`
- **THEN** `SynthesizeInstance` exists and no platform-synthesis method exists

#### Scenario: A frontend that synthesized platforms migrates to modules

- **WHEN** a frontend previously built a platform from typed subscription inputs
- **THEN** it generates or writes a platform CUE module (a `cue.mod` pinning the catalogs and their closure, a `platform.cue` importing them) and acquires it from that directory
