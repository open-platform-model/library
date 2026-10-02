## Why

Enhancement 0012 moves the Kubernetes decisions OPM makes (0012:D1: inventory entries, stale set, digests, prune and ownership guards, deletion ordering and the deletion hold) out of the cli and the operator and into the library, and admits `k8s.io/apimachinery` for them (0012:D2). Both decisions use "kernel" for the library as a whole. 0012:D3 narrows the placement to a tier beside `opm/kernel`, and both existing tiers rule themselves out. The kernel is "CUE in, verdicts out" and imports no Kubernetes package, and ADR-005, ADR-007 and ADR-008 reason about it in those terms. The helper tier is opt-in by definition, and a safety guard a frontend may skip is the shape 0006:D31 reverted.

The owner decided on 2026-10-02 to use a third tier, `opm/k8s/*`, beside the kernel: mandatory for a Kubernetes frontend, fenced from the kernel, in the same Go module. This change records that decision and builds its guardrails before any package exists, so that the first package change (`add-kubernetes-object-packages`) lands inside a fence that is already enforced.

## What Changes

- **ADR-011** (`adr/011-kubernetes-tier-beside-the-kernel.md`) records the placement, the fence, the obligation, the same-module choice, the apply/delete asymmetry, two-layer ordering, label stamping and the render digest, and the later move of `opm/helper/objectset`. It names four rejected alternatives: inside `opm/kernel`, under `opm/helper`, a nested Go module, and per-frontend copies (0006:D31). These are 0012:D3, 0012:D4, 0012:D5 and 0012:D6.
- **ADR-008 clarified in place:** rule 4's "The kernel derives no ordering of its own" is clarified to mean no module-specific ordering. The tier's kind-class order falls outside it. Rule 3, which allows an opt-in executor under `opm/helper/`, is unchanged: ADR-011 bans an executor in `opm/k8s` and the kernel only.
- **Constitution** (`CONSTITUTION.md` and the embedded text in `openspec/config.yaml`, kept identical in substance): Principle III gains the third tier, Principle IV's "MUST NOT import command, controller, or runtime-specific concerns" is narrowed (`opm/k8s` alone may import `k8s.io/apimachinery`, and no `opm/` package imports client-go, controller-runtime or Flux), and Principle I gains one sentence saying that Kubernetes vocabulary is not a runtime assumption.
- **Boundary prose:** `opm/helper/doc.go` (doc only), `README.md` § Helper boundary and `AGENTS.md` (repository rules and layout, with `opm/k8s/` marked as planned) stop saying that everything outside `opm/helper/` is kernel contract.
- **Lint fence** (`.golangci.yml`): four depguard rules. `nothing-imports-k8s-tier` stops every other `opm/` package from importing `opm/k8s`. `k8s-tier-imports-no-runtime` stops `opm/k8s` from importing client-go, controller-runtime, Flux, `opm/internal` or `opm/helper`. `kernel-imports-no-kubernetes` stops every kernel file from importing `k8s.io` or `sigs.k8s.io`. `opm-imports-no-cluster-runtime` stops every file under `opm/` from importing client-go, controller-runtime or Flux.
- **Specs:** a new `kubernetes-tier` capability, and the `helper-packages` boundary requirement rewritten so it no longer says that everything outside the helper tier is kernel contract.

## Not in this change

- Any Go code under `opm/k8s`, and any `go.mod` change. `k8s.io/apimachinery` arrives with `add-kubernetes-object-packages` (checklist task e2), which also moves `opm/helper/objectset` into `opm/k8s/object`.
- The cli and opm-operator lint rules that forbid their old local copies. Each frontend adds its rule when it migrates.
- Enhancement 0012's own text. The decision numbers 0012:D3 to 0012:D6 are cited here and are written into `enhancements/0012/03-decisions.md` separately.

## Classification

**No release.** Documentation, lint configuration and specs only. No exported symbol, signature or behaviour changes. Section commits are `docs(adr)`, `docs(constitution)` and `chore(lint)`, all of them types release-please hides. Complexity (Principle VII): four depguard rules, two of them guarding a tier that does not yet exist. They are justified because 0012:D3 requires the fence to come before the code. A fence added in the same change as the first package would be reviewed as part of that package and could be loosened there without anyone noticing.

## Downstream consumers

- **cli** and **opm-operator**: none now. Once packages land, ADR-011 binds them to adopt each package and to delete their copy in the same release (0012:D3).

## Capabilities

### New Capabilities

- `kubernetes-tier`: where the Kubernetes runtime surface lives, what may import it, what it may import, that the kernel imports no Kubernetes package, that neither the tier nor the kernel ships an executor or does cluster I/O, and that it binds a Kubernetes frontend.

### Modified Capabilities

- `helper-packages`: the boundary requirement stops saying that everything outside `opm/helper/` is kernel contract. It names `opm/k8s/` as the second non-kernel tier and lists `opm/catalog` among the kernel packages (missing since `acquire-catalog-artifact`).

## Impact

- `adr/011-kubernetes-tier-beside-the-kernel.md` (new), `adr/008-kernel-plans-caller-runs.md`.
- `CONSTITUTION.md`, `openspec/config.yaml`.
- `opm/helper/doc.go`, `README.md`, `AGENTS.md`.
- `.golangci.yml`.
