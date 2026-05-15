# Experiments — Enhancement 006 (Platform Capabilities)

Self-contained proofs-of-concept validating specific claims in the 006 design. Per the template: one concept per experiment, copy-not-reference, disposable.

| #  | Name                          | Hypothesis                                                                                                                                                  | Status |
| -- | ----------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| 01 | matcher-mechanics             | `#ContextBuilder._matched` produces correct outputs for the required/optional × match/missing/mismatch matrix; OQ6 platform inheritance via CUE unification | Draft  |
| 02 | read-portability-fillpath     | `#consumes.required[fqn].spec.X` interpolation concretizes through the 3-step `#ModuleRelease` flow; unbound release surfaces a clean diagnostic; `#`-prefix excludes `#platform` from `cue export`; `cue.Value.FillPath` populates `#platform` per D13 | Draft  |

Each experiment lives in its own subdirectory with its own `cue.mod/module.cue` so it evaluates standalone. To run an experiment, `cd` into its directory and follow the steps in that experiment's `README.md` (or run `./run.sh`).

Schemas under `schemas/` are **copied and trimmed** from `apis/core/v1alpha2/` plus the 006 design (03-schema.md). They are not authoritative — the production tree is. When 006 lands, these experiments may be deleted.
