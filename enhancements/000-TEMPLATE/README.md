# Design Package: {Enhancement Title}

| Field       | Value            |
| ----------- | ---------------- |
| **Status**  | Draft            |
| **Created** | {YYYY-MM-DD}     |
| **Authors** | OPM Contributors |

## Summary

{One to three sentences describing what this enhancement introduces and why it matters.}

## Documents

1. [01-problem.md](01-problem.md) — {One-line description of the problem being solved}
2. [02-design.md](02-design.md) — {One-line description of the proposed design}
3. [03-decisions.md](03-decisions.md) — All design decisions with rationale and alternatives considered

<!-- Add optional and enhancement-specific documents below, numbered sequentially -->

## Applicability Checklist

Check each box that applies to this enhancement. When checked, create the corresponding numbered file and add it to the Documents list above.

- [ ] `NN-schema.md` — New or modified CUE definitions
- [ ] `NN-pipeline-changes.md` — Go pipeline modifications
- [ ] `NN-module-integration.md` — Impact on module authors
- [ ] `NN-notes.md` — Deferred items and open questions
- [ ] `experiments/` — Self-contained proofs-of-concept validating ideas in this enhancement (see Experiments below)

Replace `NN` with the next available number in the sequence (starting from `04`).

## Experiments

Experiments are **optional** and usually appear **after the enhancement has been in design for a while** — once specific claims emerge that benefit from a runnable proof. Do not create `experiments/` upfront when copying this template; add it the first time you actually need to validate an idea. If the enhancement reaches implementation without ever needing one, that is fine.

When an idea does need to be tested or showcased before adoption, place proofs-of-concept under `experiments/` inside this enhancement directory. Experiments live with the enhancement so reviewers can find them next to the design that motivated them.

### Rules

- **One concept per experiment.** Each experiment proves a single idea. If two ideas are entangled, split them into two experiments.
- **Self-contained.** An experiment must run without modifying anything outside its own directory. No edits to `opm/`, `catalog/`, runtime packages, or any sibling experiment.
- **Copy, never reference, source-of-truth artifacts.** CUE schemas, traits, transformers, Go fixtures — copy them into the experiment directory and modify the copies. Never import from or mutate the originals.
- **Disposable.** Experiments are not production code. They may be deleted once the enhancement is implemented or rejected. Do not build infrastructure that other code depends on.
- **Languages.** Go is preferred for runtime/pipeline experiments; CUE for schema experiments; shell scripts or other languages where they fit.

### Layout

When you add the first experiment, create `experiments/README.md` as an index, then add per-experiment subdirectories alongside it.

```
experiments/
├── README.md                  # Index of experiments + how to run them
├── 01-{concept-name}/
│   ├── README.md              # What this proves, how to run, expected outcome
│   ├── ...                    # Copied schemas, Go code, fixtures, etc.
│   └── ...
└── 02-{concept-name}/
    └── ...
```

The `experiments/README.md` is a thin index — list each experiment, its hypothesis, and its current status (Draft / Running / Concluded). Per-experiment READMEs carry the detail.

### Per-experiment README

Each experiment's README must answer:

1. **Hypothesis** — What claim from the design is this validating?
2. **Setup** — What was copied in, from where, and what was modified.
3. **Run** — Exact commands to reproduce the result.
4. **Outcome** — What was observed; whether the hypothesis held.

Update the per-experiment README in place as the experiment evolves. Once concluded, record the outcome and link the result back into `02-design.md` or `03-decisions.md` so the enhancement carries the evidence.

## Cross-References

| Document | Purpose |
| -------- | ------- |
| `CONSTITUTION.md` (repo root) | Core design principles governing all changes in this repository |
| {path} | {purpose} |

<!--
## Agent Instructions

This directory is a template for new OPM enhancements. To create a new enhancement:

1. Copy this entire `000-TEMPLATE/` directory to `NNN-kebab-case-title/` using the next available number (three-digit, zero-padded).
2. Fill in all `{placeholder}` values.
3. The three mandatory files (01-problem.md, 02-design.md, 03-decisions.md) must always be present and populated.
4. Review the Applicability Checklist. For each checked item, create the corresponding file numbered sequentially from 04 onward and add it to the Documents list.
5. All files must use the `NN-name.md` numbering convention.
6. Keep the Documents list in README.md in sync with the actual files in the directory.
7. Update the Applicability Checklist as the enhancement evolves — check boxes when files are added, uncheck if removed.
8. Do not create `experiments/` upfront. Add it only when a specific claim in the design needs a runnable proof (often partway through the enhancement's life). Check the `experiments/` box in the Applicability Checklist at that point.

### Status Lifecycle

- **Draft** — initial design, actively being written
- **Accepted** — design agreed upon, ready for implementation
- **Implemented** — design has been realized in code
- **Superseded by NNN** — replaced by a newer enhancement
-->
