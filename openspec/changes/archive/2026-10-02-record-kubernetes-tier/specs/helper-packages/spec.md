## MODIFIED Requirements

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
