## Why

`add-platform-materialize` produces a `MaterializedPlatform` carrying the composed transformer map and `#matchers` reverse index, but the current matcher (`opm/compile/match.go`) only checks FQN *set-membership* — it never verifies that a consumer component's primitive body agrees with the transformer's `requiredResources[FQN]` / `requiredTraits[FQN]` body. Under the SemVer-FQN model (enhancement 0001, D5/D6), schema agreement must be enforced for every paired primitive, a missing FQN must be a hard structured diagnostic (D20), and the matcher must consume the materialized platform rather than a raw `#Platform`. This slice rewrites the algorithm to *FQN-lookup → always-unify → predicate* and moves the kernel entry points onto `*MaterializedPlatform`.

## What Changes

This change is **BREAKING (MAJOR)** — it changes kernel method signatures and removes a helper.

- **BREAKING** — `Kernel.Match` / `Plan` / `Compile` take a `*MaterializedPlatform` instead of `*platform.Platform`. `MatchInput.Platform`, `PlanInput.Platform`, `CompileInput.Platform` are retyped accordingly. Callers must `Materialize` before matching.
- **MODIFIED `opm/compile/match.go`** — between FQN lookup and predicate evaluation, insert an always-on unify rung: `unify(component.#resources[FQN], transformer.requiredResources[FQN])` and the analogous traits step (D6). No `--strict` mode; unification runs every time (cost is a few CUE evaluations per paired primitive). Same-SemVer byte-identical bodies collapse cleanly; divergent bodies fail.
- **NEW `MissingFQN`** in `opm/errors` (D20) — one structured diagnostic per `(release, component, fqn)` triple, shape `{release, component, fqn, alternatives}`. `alternatives` is a prefix-match on `modulePath`/`name` across the materialized FQN universe at adjacent SemVers. Match accumulates every miss in one pass — no fail-fast.
- **NEW `UnifyError`** in `opm/errors` — shape `{component, fqn, cause}`, where `cause` is the CUE error tree (walkable via `errors.As` → `cuelang.org/go/cue/errors.Error`). The CUE `conflicting values "X" and "Y": file:line file:line` message is surfaced **verbatim** with no Go-side formatting.
- **MODIFIED `MatchPlan`** — grows `Missing []MissingFQN` and `Unify []UnifyError` fields, replacing the inline `MatchResult.MissingResources` / `MissingTraits` set-membership semantics. The existing `(*MatchPlan, error)` return shape is preserved.
- **REMOVED `opm/helper/platform/compose.go`** — the Module-valued `#registry` composition `Compose` performs is obsolete under the subscription model; callers build `*Platform` directly and `Materialize` it. Deleted outright (no deprecated stub), consistent with the no-back-compat stance for the single in-repo consumer.
- **MIGRATED** — `cmd/flow-inspect`, `opm/compile` tests, and `opm/helper/platform` fixtures rewired to the `Materialize → *MaterializedPlatform → Match` path, reusing the `modregistrytest` fixtures introduced by `add-platform-materialize`.

Multi-fulfiller behavior is unchanged (D17): `#matchers[FQN]` stays a list; predicate disambiguation (labels + extra required primitives) is still the tie-breaker; workload-type discrimination (`stateless → Deployment`, etc.) continues to work. The SemVer-FQN expansion only narrows average bucket size.

Out of scope: the catalog repackage to the D19 `#Catalog` shape and its `@0.1.0` publish (independent), and the `modules/` publish task.

## Capabilities

### New Capabilities

<!-- None. MissingFQN / UnifyError are new observable behaviors of the existing platform-matching capability, not a separate capability. -->

### Modified Capabilities

- `platform-matching`: adds the always-unify rung between FQN lookup and predicate; `MissingFQN` becomes a hard structured diagnostic per `(release, component, fqn)`; `UnifyError` surfaces CUE divergence verbatim; `MatchPlan` accumulates both in one pass. Match consumes `*MaterializedPlatform`.
- `kernel-runtime`: `Match` / `Plan` / `Compile` signatures take `*MaterializedPlatform` (**BREAKING**).
- `helper-packages`: `Compose` is removed.

## Impact

- **`opm/compile/match.go`**: rewritten (lookup → unify → predicate).
- **`opm/errors/`**: adds `MissingFQN`, `UnifyError`.
- **`opm/kernel/`**: `inputs.go` retypes the `Platform` field on `MatchInput` / `PlanInput` / `CompileInput`; `phases.go` updates `Match` / `Plan` / `Compile` bodies.
- **`opm/helper/platform/`**: `compose.go` + `compose_test.go` deleted.
- **`cmd/flow-inspect/`**: callsite migrated to build → materialize → match.
- **Tests**: reuse the `modregistrytest` in-memory registry + inline `#Catalog` fixtures from `add-platform-materialize`.
- **`MIGRATIONS.md`**: documents the `Match(*Platform)` → `Materialize` + `Match(*MaterializedPlatform)` recipe and the `Compose` removal.
- **SemVer**: **MAJOR**. Breaks `Kernel.Match` / `Plan` / `Compile` signatures and removes `Compose`. Downstream `cli/` and `opm-operator/` must build `*Platform`, call `Materialize`, and pass `*MaterializedPlatform` into the phase methods; any `Compose` callsite must be replaced. Migration cost is concentrated at the platform-construction boundary, not throughout the pipeline.
- **Sequencing**: depends on `add-platform-materialize` (the `MaterializedPlatform` type and `Materialize` must exist first). Independent of the catalog repackage; the end-to-end flow test wiring the real `opmodel.dev/catalogs/opm@0.1.0` catalog comes once both this and the repackage have landed.
