## ADDED Requirements

### Requirement: The Kubernetes tier lives at opm/k8s beside the kernel

The library SHALL place the Kubernetes decisions OPM makes under `opm/k8s/`, a tier that is neither part of the kernel nor part of the opt-in helper tier. The tier holds inventory entry construction, stale-set computation, digests, the prune and ownership guards at apply and delete time, deletion ordering, the deletion hold protocol, Compiled-to-object conversion, the label vocabulary, kind-class apply order and readiness evaluation. No package under `opm/kernel/` or `opm/helper/` SHALL implement any of these decisions once the `opm/k8s/` package that owns it exists. The tier SHALL be part of the library's Go module, `github.com/open-platform-model/library`, and SHALL NOT carry a `go.mod` of its own. ADR-011 records the placement. Source: 0012:D3.

#### Scenario: The placement is recorded before any package exists

- **WHEN** a developer reads `adr/011-kubernetes-tier-beside-the-kernel.md`
- **THEN** it names `opm/k8s/*` as the Kubernetes tier, states that the tier is outside `opm/kernel` and outside `opm/helper`, and names placement inside the kernel, under the helper tier, in a nested Go module and in per-frontend copies as rejected alternatives with reasons

#### Scenario: The tier is not a nested module

- **WHEN** a developer lists every `go.mod` file tracked in the repository
- **THEN** exactly one is found, at the repository root, and none is under `opm/k8s/`

### Requirement: No other library package imports the Kubernetes tier

No package outside `opm/k8s/` SHALL import a package under `opm/k8s/`. This covers `opm/kernel`, `opm/module`, `opm/platform`, `opm/catalog`, `opm/schema`, `opm/errors`, every package under `opm/internal/` and every package under `opm/helper/`, test files included. The rule SHALL be enforced by the repository lint gate before any package under `opm/k8s/` exists. Source: 0012:D3.

#### Scenario: Lint refuses a kernel import of the tier

- **WHEN** a change adds an import of a package under `opm/k8s/` to a file under `opm/kernel/`, `opm/internal/` or any other `opm/` package outside `opm/k8s/`
- **THEN** `task lint` fails naming the forbidden import

#### Scenario: The kernel's dependency list is free of the tier

- **WHEN** the dependency list of `opm/kernel` and of every package under `opm/internal/` is computed
- **THEN** no import path under `opm/k8s/` and no path under `k8s.io/` appears in it

### Requirement: The Kubernetes tier imports no cluster client or controller framework

No file under `opm/k8s/` SHALL import `k8s.io/client-go`, `sigs.k8s.io/controller-runtime` or any package under `github.com/fluxcd/`, and none SHALL import a package under `opm/internal/` or `opm/helper/`. The only Kubernetes module the tier MAY import is `k8s.io/apimachinery`, and the only library packages it MAY import are the kernel's exported packages. The deny rules SHALL be enforced by the repository lint gate. Source: 0012:D3.

#### Scenario: Lint refuses a cluster client in the tier

- **WHEN** a change adds an import of `k8s.io/client-go`, `sigs.k8s.io/controller-runtime` or a `github.com/fluxcd/` package to a file under `opm/k8s/`
- **THEN** `task lint` fails naming the forbidden import

#### Scenario: Lint refuses a helper or internal import in the tier

- **WHEN** a change adds an import of a package under `opm/helper/` or `opm/internal/` to a file under `opm/k8s/`
- **THEN** `task lint` fails naming the forbidden import

### Requirement: The Kubernetes tier names actions and never performs them

The Kubernetes tier SHALL perform no cluster I/O and SHALL ship no loop that drives a plan to completion. A deletion plan SHALL advance one action per call from a serialisable state the caller owns, and the frontend SHALL perform each action with its own client. The tier SHALL own the whole deletion sequence (plan, transition, hold verdict), and for apply only the per-object verdict and the kind-class order, never an apply engine. Source: 0012:D3, 0012:D4. ADR-008 rules 1 to 3 apply to the tier unchanged.

#### Scenario: No executor ships in the library

- **WHEN** a developer searches the library for a function that takes a cluster client, a REST config or a controller-runtime client
- **THEN** none exists under `opm/`

#### Scenario: Apply engines stay with the frontends

- **WHEN** a developer reads ADR-011
- **THEN** it states that the operator keeps `fluxcd/pkg/ssa` and the CLI keeps its own server-side apply, and that the library owns the per-object apply verdict and the order but not the engine

### Requirement: The Kubernetes tier binds a Kubernetes frontend

The Kubernetes tier SHALL NOT be described as opt-in. A frontend that targets Kubernetes uses it, and when it adopts a package it deletes its own copy in the same release with no alias. The library's constitution and package documentation SHALL say so, and SHALL NOT state that everything outside `opm/helper/` is kernel contract. Source: 0012:D3.

#### Scenario: The constitution names three tiers

- **WHEN** a developer reads Principle III of `CONSTITUTION.md` and of the constitution embedded in `openspec/config.yaml`
- **THEN** both name the kernel, the opt-in `opm/helper/` tier and the `opm/k8s/` tier, and state that `opm/k8s/` is mandatory for a Kubernetes frontend and fenced from the kernel

#### Scenario: The runtime-concerns clause is narrowed, not dropped

- **WHEN** a developer reads Principle IV of `CONSTITUTION.md` and of the constitution embedded in `openspec/config.yaml`
- **THEN** both state that `opm/k8s/` alone may import `k8s.io/apimachinery` and that no `opm/` package imports `k8s.io/client-go`, `sigs.k8s.io/controller-runtime` or Flux

### Requirement: Kind-class order is a Kubernetes fact, not a kernel derivation

The library SHALL keep one kind-class apply order (a CustomResourceDefinition before its custom resources, a Namespace before namespaced objects), and it SHALL live in the Kubernetes tier, not in the kernel. Module-declared ordering (hooks, `dependsOn`, phases) SHALL remain data decoded off the CUE build. ADR-008's "The kernel derives no ordering of its own" SHALL be read as excluding module-specific ordering only, and ADR-008 SHALL say so in its own text. Source: 0012:D5.

#### Scenario: ADR-008 carries the clarification

- **WHEN** a developer reads `adr/008-kernel-plans-caller-runs.md`
- **THEN** its Status records the 2026-10-02 amendment pointing at ADR-011, and the rule ending "The kernel derives no ordering of its own" states that it excludes module-specific ordering only

### Requirement: Labels are stamped in CUE and the shared digest ignores the runtime name

Label stamping SHALL stay in CUE at render, where core's `#runtimeName` fills the managed-by label. The tier's shared render digest SHALL leave the managed-by label's value out of its input, so that the CLI and the operator, rendering the same instance, compute the same digest bytes. Source: 0012:D6, 0012:D1:R1.

#### Scenario: The digest rule is recorded

- **WHEN** a developer reads ADR-011
- **THEN** it states that labels stay stamped in CUE at render and that the shared digest excludes the managed-by label value
