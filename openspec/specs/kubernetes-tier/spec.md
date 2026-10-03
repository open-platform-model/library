# kubernetes-tier Specification

## Purpose
The Kubernetes tier `opm/k8s/` beside the kernel (ADR-011, enhancement 0012:D3 to 0012:D6): where the Kubernetes decisions OPM makes live, how lint fences the tier from the kernel and from cluster runtimes, what it owns for apply and delete, and that every frontend targeting Kubernetes must use it.

## Requirements

### Requirement: The Kubernetes tier lives at opm/k8s beside the kernel

The library SHALL place the Kubernetes decisions OPM makes under `opm/k8s/`, a tier outside the kernel and outside the opt-in helper tier. The tier holds inventory entry construction, stale-set computation, digests, the prune and ownership guards at apply and delete time, deletion ordering, the deletion hold protocol, Compiled-to-object conversion, the label vocabulary, kind-class apply order and readiness evaluation. No package under `opm/kernel/` or `opm/helper/` SHALL implement any of these decisions once the `opm/k8s/` package that owns it exists. The tier SHALL be part of the library's Go module, `github.com/open-platform-model/library`, and SHALL NOT carry a `go.mod` of its own. ADR-011 records the placement. Source: 0012:D3.

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

### Requirement: The kernel imports no Kubernetes package and no library package imports a cluster runtime

No file under `opm/kernel`, `opm/module`, `opm/platform`, `opm/catalog`, `opm/schema`, `opm/errors` or `opm/internal/` SHALL import a package under `k8s.io/` or `sigs.k8s.io/`, test files included. No file under `opm/` SHALL import `k8s.io/client-go`, `sigs.k8s.io/controller-runtime` or a package under `github.com/fluxcd/`. Both rules SHALL be enforced by the repository lint gate before any package under `opm/k8s/` exists. Source: 0012:D2, 0012:D3.

#### Scenario: Lint refuses a Kubernetes import in the kernel

- **WHEN** a change adds an import of a `k8s.io/` or `sigs.k8s.io/` package to a file under a kernel package
- **THEN** `task lint` fails naming the forbidden import

#### Scenario: Lint refuses a cluster runtime anywhere under opm

- **WHEN** a change adds an import of `k8s.io/client-go`, `sigs.k8s.io/controller-runtime` or a `github.com/fluxcd/` package to any file under `opm/`, the helper tier included
- **THEN** `task lint` fails naming the forbidden import

### Requirement: The Kubernetes tier imports no cluster client or controller framework

Beyond the standard library and the CUE SDK that the kernel's output types carry, a file under `opm/k8s/` MAY import only the kernel's exported packages and `k8s.io/apimachinery`. No file under `opm/k8s/` SHALL import `k8s.io/client-go`, `sigs.k8s.io/controller-runtime`, any package under `github.com/fluxcd/`, any cluster client, or a package under `opm/internal/` or `opm/helper/`, and of the Kubernetes modules it SHALL import only `k8s.io/apimachinery`. The repository lint gate SHALL enforce these denials, including the refusal of every other `k8s.io` and `sigs.k8s.io` module. Source: 0012:D3.

#### Scenario: Lint refuses a cluster client in the tier

- **WHEN** a change adds an import of `k8s.io/client-go`, `sigs.k8s.io/controller-runtime` or a `github.com/fluxcd/` package to a file under `opm/k8s/`
- **THEN** `task lint` fails naming the forbidden import

#### Scenario: Lint refuses a Kubernetes module other than apimachinery in the tier

- **WHEN** a change adds an import of `k8s.io/api`, `k8s.io/utils`, `k8s.io/klog` or `sigs.k8s.io/yaml` to a file under `opm/k8s/`
- **THEN** `task lint` fails naming the forbidden import
- **AND** an import of `k8s.io/apimachinery` together with `opm/kernel` in the same file passes

#### Scenario: Lint refuses a Kubernetes module in the helper tier

- **WHEN** a change adds an import of `k8s.io/apimachinery` or any other `k8s.io` or `sigs.k8s.io` package to a file under `opm/helper/`
- **THEN** `task lint` fails naming the forbidden import

#### Scenario: Lint refuses a helper or internal import in the tier

- **WHEN** a change adds an import of a package under `opm/helper/` or `opm/internal/` to a file under `opm/k8s/`
- **THEN** `task lint` fails naming the forbidden import

### Requirement: Nothing in the library performs a cluster action

Nothing in the library SHALL perform a cluster action, and the Kubernetes tier SHALL perform no cluster I/O. No executor (a loop that drives a plan to completion) and no executor backend that performs a planned action against a cluster SHALL ship anywhere under `opm/`: not in the kernel, not under `opm/k8s/` and not under `opm/helper/`. ADR-008 rule 3's allowance for opt-in executor backends under `opm/helper/` is narrowed to executor backends that perform no planned action against a cluster (0009's wasm, HTTP, `cue.eval` and local container hosts; 0012:D3 amends 0009:D4 so its cluster-acting Ops are performed by the frontend). A deletion plan SHALL advance one action per call from a serialisable state the caller owns, and the frontend SHALL perform each action with its own client, in a loop it writes itself. The tier SHALL own the whole deletion sequence (plan, transition, hold verdict). For apply, it SHALL own the per-object verdict (CanApply) and the order, never an apply engine. Each frontend SHALL submit objects in the library's order. An engine's own staging MAY refine that order, for example by sorting within a stage, and SHALL NOT contradict it. ADR-008 rules 1 to 3 apply to the tier. Source: 0012:D3, 0012:D4, 0012:D5.

#### Scenario: No executor loop or cluster backend ships anywhere in the library

- **WHEN** a developer searches every package under `opm/` (the kernel, `opm/k8s/` and `opm/helper/`) for a function that takes a cluster client, a REST config or a controller-runtime client, that performs a planned action against a cluster, or that drives a plan to completion
- **THEN** none exists

#### Scenario: ADR-008 allows only non-cluster executor backends

- **WHEN** a developer reads rule 3 of ADR-008
- **THEN** it states that no executor backend that performs a planned action against a cluster ships in the library, `opm/helper/` included, that each frontend performs such an action with its own client, and that opt-in executor backends that perform no planned action against a cluster may ship under `opm/helper/`

#### Scenario: Apply engines stay with the frontends

- **WHEN** a developer reads ADR-011
- **THEN** it states that the operator keeps `fluxcd/pkg/ssa` and the CLI keeps its own server-side apply, that the library owns the per-object apply verdict and the order and never an apply engine, and that an engine's staging may refine the library's order and never contradict it

### Requirement: The Kubernetes tier binds a Kubernetes frontend

The Kubernetes tier SHALL NOT be described as opt-in. A frontend that targets Kubernetes uses it, and when it adopts a package it deletes its own copy in the same release with no alias. The library's constitution and package documentation SHALL say so, and SHALL NOT state that everything outside `opm/helper/` is kernel contract. Source: 0012:D3.

#### Scenario: The constitution names three tiers

- **WHEN** a developer reads Principle III of `CONSTITUTION.md` and of the constitution embedded in `openspec/config.yaml`
- **THEN** both name the kernel, the opt-in `opm/helper/` tier and the `opm/k8s/` tier, and state that `opm/k8s/` is mandatory for a Kubernetes frontend and fenced from the kernel

#### Scenario: The runtime-concerns clause names what is admitted

- **WHEN** a developer reads Principle IV of `CONSTITUTION.md` and of the constitution embedded in `openspec/config.yaml`
- **THEN** both state that `opm/k8s/` alone may import `k8s.io/apimachinery` and that no `opm/` package imports `k8s.io/client-go`, `sigs.k8s.io/controller-runtime` or Flux

### Requirement: Kind-class order is a Kubernetes fact that lives in the tier

The library SHALL keep one kind-class apply order (a CustomResourceDefinition before its custom resources, a Namespace before namespaced objects). It SHALL live in the Kubernetes tier, and the kernel SHALL hold none. Module-declared ordering (hooks, `dependsOn`, phases) SHALL remain data decoded off the CUE build. ADR-008 rule 4's "The kernel derives no ordering of its own" SHALL be read as excluding module-specific ordering only, and ADR-008 SHALL say so in its own text. Source: 0012:D5.

#### Scenario: ADR-008 carries the clarification

- **WHEN** a developer reads `adr/008-kernel-plans-caller-runs.md`
- **THEN** its Status records the 2026-10-02 amendment pointing at ADR-011, and the rule ending "The kernel derives no ordering of its own" states that it excludes module-specific ordering only

### Requirement: Labels are stamped in CUE and the shared digest ignores the runtime name

Label stamping SHALL stay in CUE at render, where core's `#runtimeName` fills the managed-by label. The tier's shared render digest SHALL leave the managed-by label's value out of its input, so that the CLI and the operator, rendering the same instance, compute the same digest bytes. Source: 0012:D6, 0012:D1:R1.

#### Scenario: The digest rule is recorded

- **WHEN** a developer reads ADR-011
- **THEN** it states that labels stay stamped in CUE at render and that the shared digest excludes the managed-by label value
