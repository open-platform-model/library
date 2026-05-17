# Experiments — Enhancement 004 (Module Context)

Self-contained proofs-of-concept validating specific claims in the post-slim 004 design (`#ctx` identity-only after D36). Per the template: one concept per experiment, copy-not-reference, disposable.

| #  | Name                          | Hypothesis                                                                                                                                                                                                                                                                                  | Status                     |
| -- | ----------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------- |
| 01 | names-cascade-and-injection   | `#ContextBuilder` produces `#ComponentNames` such that the four `dns.*` variants cascade from `resourceName`, `metadata.resourceName` overrides propagate, `_clusterDomain` self-defaults to `"cluster.local"`, and `#Component.#names == #ctx.runtime.components[<key>]` (D10/D13/D32/D36) | Concluded — 2026-05-15     |
| 02 | release-flow-ordering         | The 3-step `#ModuleRelease` flow is the only ordering that produces correct `#names` for dynamic components built from `#config`; identity propagates verbatim; inline-literal fixtures need `#ctx: _` / `#names: _` declarations (D11/D32/D34/D35)                                          | Concluded — 2026-05-15     |

Each experiment lives in its own subdirectory with its own `cue.mod/module.cue` so it evaluates standalone. To run an experiment, `cd` into its directory and follow the steps in that experiment's `README.md` (or run `./run.sh`).

Schemas under `schemas/` are **copied and trimmed** from `apis/core/v1alpha2/` plus the 004 design (03-schema.md). They are not authoritative — the production tree is. When 004 lands in `apis/core/v1alpha2/`, these experiments may be deleted.

A pre-slim precursor lives at `catalog/experiments/001-module-context/` in the sibling `catalog/` repo. Its layered `#Platform` / `#Environment` fixtures and cluster-domain disjunction (F1 / D33) are superseded by D36; the surviving findings (F2 / D34, F3 / D35) are re-grounded here against the slim schema. The old experiment is retained as historical provenance.
