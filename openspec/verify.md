# Verify rules of this repo

OpenSpec never writes this file; `openspec update` leaves it alone. The rules add to the
generated `openspec-verify-change` skill. Apply them in the same verify run and put their
findings in the same report.

## A fourth dimension: Cleanliness

The report has four dimensions: Completeness, Correctness, Coherence, and **Cleanliness**
(dead code left behind by the change).

## Task Hygiene (part of Completeness)

- Flag any task that is a delivery operation other than a section-closing commit: pushing, branching, opening/merging PRs, tagging, cutting releases, commit-message or PR-body hygiene checks.
- Add WARNING: "Delivery-operation task: <description>: remove it rather than checking it off; delivery past the section commit happens under the repo's normal git workflow".
- A section-closing commit task (the last checkbox of a `## N.` section, naming a Conventional Commit) is satisfied when that commit exists in `git log`; add WARNING "Section without a commit task: <heading>" for any `## N.` section that lacks one.
- Exception: skip this check when the change's stated deliverable is itself a release or publishing operation; there those steps are the implementation.

## Verify Cleanliness (required, before the report)

This step is part of every verify run. A report-only run includes it, because the analysis reads files and changes none. A change with little code includes it too.

Before you write the report, load the `openspec-cleanup-change` skill in change-aware mode (pass the change name), do every step of it, and use its report as the Cleanliness dimension.

- High-confidence candidates: WARNING "Dead code candidate: `<symbol>` in `<file>:<line>`: <reason>".
- Medium or low-confidence candidates: SUGGESTION with the same form.
- No candidates: note "No dead code detected" in the Summary.
- Skip only when no task is done yet, and note "Cleanliness check skipped: no implementation to analyze."

The step is done when the Cleanliness row of the Summary holds one of these results: the number of candidates, "No dead code detected", or the skip note.

A report without one of these results is a failed verify run. "Not verified" is not a result for this dimension. If you cannot produce one (for example, the harness refuses the skill or a tool that the skill needs), add CRITICAL "Cleanliness not run: <the reason; a refusal word for word>" and start the Final Assessment with "Verify run failed: no Cleanliness result."

## Issue ids

Number the issues by level: CRITICAL as C1, C2; WARNING as W1, W2; SUGGESTION as S1, S2.
