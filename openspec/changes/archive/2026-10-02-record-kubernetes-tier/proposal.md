## Why

Enhancement 0012 moves the Kubernetes decisions OPM makes (enhancement 0012 ("The Kubernetes runtime surface homes in the library's Kubernetes tier (supersedes only the placement conclusion of the CLI inventory and handoff enhancement)"): inventory entries, stale set, digests, prune and ownership guards, deletion ordering and the deletion hold) out of the cli and the operator and into the library, and admits `k8s.io/apimachinery` for them (enhancement 0012, "The library is written for Kubernetes; no portability abstraction is maintained on its behalf"). Both decisions were written using "kernel" for the library as a whole, before their 2026-10-02 revision. Enhancement 0012 ("The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module") narrows the placement to a tier beside `opm/kernel`, and both existing tiers rule themselves out. The kernel is "CUE in, verdicts out" and imports no Kubernetes package, and ADR-005, ADR-007 and ADR-008 reason about it in those terms. The helper tier is opt-in by definition, and a safety guard a frontend may skip is the shape enhancement 0006 ("Revert the shared `library/opm/inventory` package (the CLI and operator keep independent local inventory implementations); supersedes D13's shared-logic clause and D26") reverted.

The owner decided on 2026-10-02 to use a third tier, `opm/k8s/*`, beside the kernel: mandatory for a Kubernetes frontend, fenced from the kernel, in the same Go module. This change records that decision and builds its guardrails before any package exists, so that the first package change (`add-kubernetes-object-packages`) lands inside a fence that is already enforced.

## What Changes

- **ADR-011** (`adr/011-kubernetes-tier-beside-the-kernel.md`) records the placement, the fence, the obligation, the same-module choice, the apply/delete asymmetry, two-layer ordering, label stamping and the render digest, and the later move of `opm/helper/objectset`. It names four rejected alternatives: inside `opm/kernel`, under `opm/helper`, a nested Go module, and per-frontend copies (enhancement 0006, "Revert the shared `library/opm/inventory` package (the CLI and operator keep independent local inventory implementations); supersedes D13's shared-logic clause and D26"). These are enhancement 0012 ("The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module", "The library owns the whole deletion sequence but only the per-object verdict and the order for apply", "Ordering is kind-class order in the Kubernetes tier; no module-internal ordering is planned, and cross-module order belongs to a future Bundle" and "Labels are stamped by the CUE render; the shared render digest ignores the runtime-name label value").
- **ADR-008 clarified in place:** rule 4's "The kernel derives no ordering of its own" is clarified to mean no module-specific ordering. The tier's kind-class order falls outside it. **ADR-008 amended in place:** rule 3's allowance for opt-in executor backends under `opm/helper/` is narrowed to executor backends that perform no planned action against a cluster (ADR-013, decision e1, recorded in ADR-011 item 4). Nothing in the library performs a planned action against a cluster: no loop ships anywhere, no cluster-acting backend ships anywhere, and each frontend writes its own loop and performs those actions with its own client. 0009's non-cluster hosts stay allowed, and enhancement 0012 ("The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module") amends enhancement 0009 ("Executor backends ship in the library's opt-in layer; frontends compose them à la carte") for its cluster-acting Ops.
- **Constitution** (`CONSTITUTION.md` and the embedded text in `openspec/config.yaml`, kept identical in substance): Principle III gains the third tier, Principle IV's "MUST NOT import command, controller, or runtime-specific concerns" is narrowed (`opm/k8s` alone may import `k8s.io/apimachinery`, and no `opm/` package imports client-go, controller-runtime or Flux), and Principle I gains one sentence saying that Kubernetes vocabulary is not a runtime assumption.
- **Boundary prose:** `opm/helper/doc.go` (doc only), `README.md` § Helper boundary and `AGENTS.md` (repository rules and layout, with `opm/k8s/` marked as planned) stop saying that everything outside `opm/helper/` is kernel contract.
- **Lint fence** (`.golangci.yml`): six depguard rules beside the existing helper rule. `nothing-imports-k8s-tier` stops every other `opm/` package from importing `opm/k8s`. `k8s-tier-imports-no-runtime` stops `opm/k8s` from importing Flux, `opm/internal` or `opm/helper`. `k8s-tier-imports-only-apimachinery` lets `opm/k8s` import no Kubernetes module but `k8s.io/apimachinery`. `kernel-imports-no-kubernetes` and `helper-imports-no-kubernetes` stop every kernel and helper file from importing `k8s.io` or `sigs.k8s.io`. `opm-imports-no-cluster-runtime` stops every file under `opm/` from importing client-go, controller-runtime or Flux.
- **Specs:** a new `kubernetes-tier` capability, and the `helper-packages` boundary requirement rewritten so it no longer says that everything outside the helper tier is kernel contract.

## Not in this change

- Any Go code under `opm/k8s`, and any `go.mod` change. `k8s.io/apimachinery` arrives with `add-kubernetes-object-packages` (ADR-013, decision e2), which also moves `opm/helper/objectset` into `opm/k8s/object`.
- The cli and opm-operator lint rules that forbid their old local copies. Each frontend adds its rule when it migrates.
- The text of enhancement 0012. The decisions of that enhancement that the ADR cites are written into the entry separately.

## Classification

**No release.** Documentation, lint configuration and specs only. No exported symbol, signature or behaviour changes. Section commits are `docs(adr)`, `docs(constitution)` and `chore(lint)`, all of them types release-please hides. Complexity (Principle VII): six depguard rules, three of them guarding a tier that does not yet exist. They are justified because enhancement 0012 ("The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module") requires the fence to come before the code. A fence added in the same change as the first package would be reviewed as part of that package and could be loosened there without anyone noticing.

## Downstream consumers

- **cli** and **opm-operator**: none now. Once packages land, ADR-011 binds them to adopt each package and to delete their copy in the same release (enhancement 0012, "The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module").

## Capabilities

### New Capabilities

- `kubernetes-tier`: where the Kubernetes runtime surface lives, what may import it, what it may import, that the kernel imports no Kubernetes package, that no executor loop and no executor backend that performs a planned action against a cluster ships anywhere in the library, and that it binds a Kubernetes frontend.

### Modified Capabilities

- `helper-packages`: the boundary requirement stops saying that everything outside `opm/helper/` is kernel contract. It names `opm/k8s/` as the second non-kernel tier and lists `opm/catalog` among the kernel packages (missing since `acquire-catalog-artifact`).

## Impact

- `adr/011-kubernetes-tier-beside-the-kernel.md` (new), `adr/008-kernel-plans-caller-runs.md`.
- `CONSTITUTION.md`, `openspec/config.yaml`.
- `opm/helper/doc.go`, `README.md`, `AGENTS.md`.
- `.golangci.yml`.
