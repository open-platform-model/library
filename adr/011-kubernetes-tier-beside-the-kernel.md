# ADR-011: A Kubernetes tier beside the kernel

## Status

Accepted (2026-10-02). Records 0012:D3, 0012:D4, 0012:D5 and 0012:D6 (workspace root, `enhancements/0012/03-decisions.md`) as library rules. Places the Kubernetes runtime surface that 0012:D1 moves into the library and that 0012:D2 admits `k8s.io/apimachinery` for. Amends ADR-008 rule 4 (see Decision 7). Recorded by `record-kubernetes-tier`, which ships the boundary and its lint fence and no package. The first package change is `add-kubernetes-object-packages`.

## Context

Enhancement 0012 moves the Kubernetes decisions OPM makes out of the two first-party frontends and into the library. 0012:D1 lists them: inventory entry construction, stale-set computation, digests, prune and ownership guards at apply and delete time, deletion ordering and the deletion hold protocol. Converting a rendered object to a Kubernetes object, the label vocabulary, the order in which kinds are applied and readiness evaluation belong with them, because each is written in both frontends today. 0012:D2 admits `k8s.io/apimachinery` as a library dependency and excludes `client-go`, `controller-runtime` and Flux. Neither decision says where in the library the code goes. The library has two tiers today, and each one constrains the answer.

The kernel (`opm/kernel`, `opm/module`, `opm/platform`, `opm/catalog`, `opm/schema`, `opm/errors` and `opm/internal/**`) takes CUE in and returns verdicts and decoded values. It imports no Kubernetes package. ADR-005 and ADR-007 state its evaluation and retention rules in terms of CUE builds and `cue.Context` lifetimes. ADR-008 states what it may do with time: it plans, names actions and keeps no run state.

The helper tier (`opm/helper/**`) is opt-in by definition. Its package doc says a frontend MAY skip it, and a depguard rule in `.golangci.yml` keeps every kernel package from importing it, so that the claim stays true. It already holds one Kubernetes-specific package, `opm/helper/objectset`, which finds rendered objects that share one apply identity.

0006:D31 is the precedent for code that a frontend may skip. On 2026-07-01 a shared `library/opm/inventory` package was reverted, and both frontends kept their own copies. By 2026-07-27, inspection had found three single-actor defects in those copies: the CLI deletes CustomResourceDefinitions, the CLI prunes without checking live ownership, and the operator force-applies over foreign objects. The duplication 0012 measured on 2026-09-14 is a byte-identical object conversion and label helpers that differ only in comments. A render-digest function exists in both frontends with a comment asking maintainers to keep them in sync by hand.

The two frontends differ in how they apply objects. The operator applies through `fluxcd/pkg/ssa`, which brings controller-runtime with it. The CLI has its own server-side apply, and it must not inherit controller-runtime. Deletion carries no such framework opinion. Ordering, fetching, guarding and deleting are plain Kubernetes steps.

Labels are stamped in CUE at render. Core's transformer composes the label set, and the runtime fills `#runtimeName`, which becomes the managed-by label value (`opm-cli` or `opm-controller`). The two frontends therefore render the same instance into objects that differ in exactly one label value. 0012:D1:R1 requires them to compute the same digest.

Both first-party frontends already require `k8s.io/apimachinery`, and both pin the same library version.

## Decision

Nine rules. Items 1 to 5 and 9 record 0012:D3, item 6 records 0012:D4, item 7 records 0012:D5, and item 8 records 0012:D6.

1. **The Kubernetes tier is `opm/k8s/*`, beside the kernel.** The decisions 0012:D1 lists live there, together with Compiled-to-object conversion, the label vocabulary, kind-class apply order and readiness evaluation. The package split is indicative and later changes settle it: `labels`, `object`, `inventory`, `ownership`, `lifecycle` and `health`. The tier sits neither inside `opm/kernel` nor under `opm/helper`.
1. **The tier is fenced in both directions.** No package outside `opm/k8s` imports it: not `opm/kernel`, `opm/module`, `opm/platform`, `opm/catalog`, `opm/schema`, `opm/errors`, `opm/internal/**` or `opm/helper/**`. A package under `opm/k8s` may import the kernel's public output types and `k8s.io/apimachinery`. It never imports `k8s.io/client-go`, `sigs.k8s.io/controller-runtime`, `github.com/fluxcd/*` or any cluster client, and it does not import `opm/internal/**` or `opm/helper/**`. depguard enforces the fence from this change on, before any package under `opm/k8s` exists.
1. **The tier is an obligation for a Kubernetes frontend, not an option.** A frontend that targets Kubernetes uses `opm/k8s`. When it adopts a package, it deletes its own copy in the same release, with no aliases. After it migrates, it adds a lint rule that forbids its old local copy (for example `pkg/core` in cli and opm-operator). Deletion is a step protocol under ADR-008: the frontend makes progress only by asking for the next action, so it has no way to skip a guard.
1. **No cluster I/O and no loop.** ADR-008 rules 1, 2 and 3 apply to the tier unchanged. It names actions, the frontend performs them with its own client, and the state between steps belongs to the caller. No executor ships in the library, under `opm/k8s` or anywhere else.
1. **The tier is in the same Go module.** `opm/k8s` is part of `github.com/open-platform-model/library`, not a nested module. The `k8s.io/apimachinery` requirement arrives with the first package change, not with this one.
1. **Apply and delete are asymmetric.** The library owns the whole deletion sequence: the plan, the transition and the hold verdict. For apply, it owns the per-object verdict (`CanApply`) and the order, but not the engine. The operator keeps `fluxcd/pkg/ssa`, and the CLI keeps its own server-side apply. This answers the half of 0012:OQ1 that ADR-008 left open.
1. **Ordering has two layers.** Kind-class order is a Kubernetes fact: a CustomResourceDefinition before its custom resources, a Namespace before the objects in it, and so on. It lives in one weight table in `opm/k8s/object`. The CLI's `cli/pkg/resourceorder` table moves there in a later change. Module-declared order (hooks, `dependsOn`, phases) is data that comes off the CUE build, under ADR-008 rule 4. That rule's sentence "The kernel derives no ordering of its own" means no module-specific ordering. The tier's kind-class order is not a kernel derivation, and ADR-008 is amended in place to say so.
1. **Label stamping stays in CUE at render.** Core's `#runtimeName` keeps filling the managed-by label. The shared render digest in `opm/k8s` leaves the managed-by label's value out of its input, so the CLI and the operator digest the same inputs to the same bytes (0012:D1:R1). This answers 0012:OQ11.
1. **`opm/helper/objectset` moves into `opm/k8s/object`.** Apply identity is Kubernetes-specific. It moves in the first package change, not in this one.

Rejected alternatives:

- **Inside `opm/kernel`.** One tier fewer, and the kernel would gain a Kubernetes dependency. The kernel today is "CUE in, verdicts out", and ADR-005, ADR-007 and ADR-008 reason about it in exactly those terms. Object conversion, digests and deletion plans take Kubernetes objects as input, not CUE builds. Putting them in the kernel would make every kernel package link apimachinery for decisions no kernel verb makes. It would also make the next reader's question, "does this verb build CUE or decide about a cluster object?", harder to answer from the import graph alone.
- **Under `opm/helper`.** The tier's job is to hold the safety guards a Kubernetes frontend must not skip, and the helper tier is the tier a frontend MAY skip, by its own definition. Placing the guards there would repeat the shape 0006:D31 reverted, a package nothing forces a frontend to use, and would leave the helper doc's central claim true in word and false in effect.
- **A nested Go module (`library/opm/k8s` with its own `go.mod`).** It keeps apimachinery out of the kernel module's requirements. Both first-party frontends already require apimachinery, though, so the floor costs them nothing. A nested module adds a hop to the release cascade and lets the kernel and the tier drift to different versions in one frontend's `go.sum`, the version skew the tier exists to remove. Revisit only if an embedder appears that renders without Kubernetes.
- **Per-frontend copies, as 0006:D31 decided.** This is the state today. It produced the three single-actor defects listed in the Context, and nothing in the frontends' build fails when their copies drift.

## Consequences

**Positive:** Each Kubernetes decision gets one definition that both frontends execute. A guard added to the tier reaches both frontends on their next library bump, and a frontend cannot drift from it by forgetting to port it. The kernel's import graph and its ADRs stay as they are: no kernel package gains a Kubernetes import, and the rules of ADR-005, ADR-007 and ADR-008 hold without revision. The fence is lint-enforced before the first package lands, so the first package change cannot quietly let the kernel import the tier. Deletion safety becomes testable without a cluster, because a plan and its transitions are pure functions of their inputs.

**Negative:** The library becomes Kubernetes-aware. An embedder that renders without Kubernetes still gets `k8s.io/apimachinery` in its module graph once the first package lands, which is the cost 0012:D2 accepted. The public surface grows by a tier of packages, each of them SemVer surface under Principle VI, against a program that spent eight changes shrinking it. Two frontends must delete code they own and add a lint rule against its return, which is coordinated work across three repos for each package.

**Trade-off:** Apply stays split. The library decides whether each object may be applied and in what order, while each frontend keeps its own apply engine. A defect in how an engine performs an apply that the verdict allowed is still a per-frontend defect. The alternative, one engine, would pull controller-runtime into the CLI through Flux. For deletion, the library owns the sequence because no framework opinion is involved.

**Trade-off:** Leaving the managed-by value out of the digest means a change that touches only that label does not move the digest. That is the intended behaviour: the label names which runtime applied the object, and a handoff between the two runtimes must not read as a content change.

**Relation to ADR-008:** ADR-008 settles that the library plans and the caller runs. This ADR places the Kubernetes plans the library will emit and fences them. Rule 4 of ADR-008 is narrowed so that a fixed kind-class order, which is a fact about Kubernetes and not about any module, does not count as ordering the kernel derives.

**Relation to ADR-005 and ADR-007:** neither is touched. The tier evaluates no CUE build and holds no `cue.Context`. Every input it reads comes from what `Kernel.Render` returns.
