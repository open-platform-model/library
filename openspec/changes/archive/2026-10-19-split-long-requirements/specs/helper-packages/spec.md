## MODIFIED Requirements

### Requirement: Helper Boundary at opm/helper/
The helper boundary SHALL be real in the import graph: no kernel-tier package (Requirement: Two tiers lie outside opm/helper) SHALL import a package under `opm/helper/` (the Kubernetes tier is held to the same rule by `kubernetes-tier`), no exported kernel signature SHALL name a type declared under `opm/helper/`, and no kernel operation SHALL return an error whose sentinel is declared under `opm/helper/`. The rule SHALL be enforced by the repository lint gate, not only by documentation.

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

## REMOVED Requirements

### Requirement: Helper Layout for Future Subpackages
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Future opt-in helpers follow the opm/helper/<name>/ convention and are added by their owning slices", "Collapsed subpackages do not appear as exemplars of the helper convention", "The one current helper subpackage is platformmodule" and "The earlier opm/helper/objectset does not exist and the check lives in opm/k8s/object".

## ADDED Requirements

### Requirement: The library maintains an opm/helper subdirectory of opt-in frontend conveniences
The library SHALL maintain a `opm/helper/` subdirectory whose subpackages are opt-in, opinionated frontend conveniences.

#### Scenario: The helper subdirectory is opt-in

- **WHEN** the library is read
- **THEN** it maintains a `opm/helper/` subdirectory whose subpackages are opt-in, opinionated frontend conveniences

### Requirement: Two tiers lie outside opm/helper
The packages outside `opm/helper/` SHALL fall into two tiers: the kernel (`opm/kernel`, `opm/module`, `opm/platform`, `opm/catalog`, `opm/schema`, `opm/errors` and `opm/internal/**`), whose contract every frontend MUST honour, and the Kubernetes tier under `opm/k8s/` (ADR-011, requirement "The Kubernetes tier binds a Kubernetes frontend" in `kubernetes-tier`), whose contract every frontend that targets Kubernetes MUST honour. Neither tier SHALL be described as opt-in.

#### Scenario: The packages outside the helper tree fall into two tiers

- **WHEN** the packages outside `opm/helper/` are read
- **THEN** they fall into two tiers: the kernel, whose contract every frontend MUST honour, and the Kubernetes tier under `opm/k8s/`, whose contract every frontend that targets Kubernetes MUST honour, and neither tier is described as opt-in

### Requirement: The exported packages outside opm/helper and opm/k8s are exactly the six kernel packages
The exported packages outside `opm/helper/` and `opm/k8s/` SHALL be exactly `opm/kernel`, `opm/module`, `opm/platform`, `opm/catalog`, `opm/schema` and `opm/errors`: no publish-side or presentation package SHALL ship in the library.

#### Scenario: The exported packages outside the helper and Kubernetes trees are the six kernel packages

- **WHEN** the exported packages outside `opm/helper/` and `opm/k8s/` are listed
- **THEN** they are exactly `opm/kernel`, `opm/module`, `opm/platform`, `opm/catalog`, `opm/schema` and `opm/errors`, and no publish-side or presentation package ships in the library

### Requirement: Future opt-in helpers follow the opm/helper/<name>/ convention and are added by their owning slices
Future opt-in helpers SHALL follow the `opm/helper/<name>/` convention. Subpackages SHALL be added by their owning slices and not as part of the originating slice that established the convention.

#### Scenario: A future helper follows the convention

- **WHEN** a future opt-in helper is added
- **THEN** it follows the `opm/helper/<name>/` convention and is added by its owning slice, not as part of the originating slice that established the convention

### Requirement: Collapsed subpackages do not appear as exemplars of the helper convention
Past examples of helper subpackages in this convention (Requirement: Future opt-in helpers follow the opm/helper/<name>/ convention and are added by their owning slices) SHALL reflect the current package layout; subpackages that have been collapsed into the kernel (the previous `opm/helper/values/`, `opm/helper/loader/file/`, `opm/helper/loader/registry/` and `opm/helper/synth/`) SHALL NOT appear as exemplars.

#### Scenario: Values helper subpackage no longer exists

- **WHEN** a developer searches `opm/helper/` for a `values` subpackage
- **THEN** no `opm/helper/values/` directory exists
- **AND** the canonical implementation of layered values validation lives at `Kernel.ValidateConfigDetailed` in `opm/kernel/`

### Requirement: The one current helper subpackage is platformmodule
The one current subpackage (Requirement: Future opt-in helpers follow the opm/helper/<name>/ convention and are added by their owning slices) SHALL be `platformmodule` (platform module generation from catalog coordinates).

#### Scenario: Platform helper landing place

- **WHEN** a frontend needs to generate a platform module from catalog coordinates
- **THEN** `opm/helper/platformmodule/` is the directory that helper occupies

#### Scenario: Loader and synth subpackages no longer exist

- **WHEN** a developer lists the subpackages of `opm/helper/`
- **THEN** exactly `platformmodule` is present
- **AND** neither `opm/helper/loader/` nor `opm/helper/synth/` exists

### Requirement: The earlier opm/helper/objectset does not exist and the check lives in opm/k8s/object
The earlier `opm/helper/objectset` (duplicate rendered object identities) SHALL NOT exist: the check's one home is `opm/k8s/object` (ADR-011 item 9).

#### Scenario: Duplicate-identity helper landing place

- **WHEN** a runtime needs to refuse a render whose objects share one apply identity
- **THEN** `opm/k8s/object` is the package that provides the check, no `opm/helper/objectset/` directory exists, and the kernel does not call the check
