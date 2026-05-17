---
name: enhancements-workflow
description: Conventions for creating, editing, and tracking OPM library enhancements (design packages under library/enhancements/). Covers the TEMPLATE, the mandatory + optional documents, the config.yaml metadata contract, and the validator/list tasks.
user-invocable: true
---

# Enhancements Workflow

## When this applies

Any task touching `library/enhancements/`: creating a new enhancement, editing an existing design doc, updating implementation status as code lands, recording the outcome of an experiment, adding cross-references between enhancements, or archiving / marking one superseded.

## Directory layout

Enhancements live under `library/enhancements/NNN-<kebab-slug>/`.

- `NNN` is a zero-padded three-digit string, the next available number after the highest existing enhancement.
- The slug is kebab-case, matches `^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`, and is a short identifier — not a full sentence. Long enough to disambiguate, short enough to type.
- Both `id` (in `config.yaml`) and `slug` MUST match the directory name; `task enhancements:vet` enforces this.

## Start from the TEMPLATE — always

Use the scaffolding task. It auto-numbers the id from the highest existing enhancement, copies the template, fills `config.yaml` with real values (id, slug, title, today's date, author), and replaces the `{Enhancement Title}` placeholder in the README and the three mandatory document headers:

```bash
task enhancements:new SLUG=<kebab-slug> TITLE="<Human Title>"
```

Optional `AUTHOR="<name>"` overrides the default `OPM Contributors`. The task prints the new directory path and a checklist of remaining placeholders to fill in (`{Summary}`, per-document descriptions, Cross-References).

If you need to scaffold without the task (e.g., recovering from a botched run, or scaffolding into an unusual location), the manual fallback is `cp -r library/enhancements/000-TEMPLATE library/enhancements/NNN-<slug>` followed by hand-editing the placeholders. The task does this for you and is the preferred path; the manual route exists only as an escape hatch.

The template carries the canonical structure (README + the three mandatory documents), the Applicability Checklist for optional docs, the agent-instruction HTML comments at the bottom of the README (do not strip these when copying), and a placeholder `config.yaml`.

## Documents

Three documents are mandatory and always present:

1. `01-problem.md` — the problem being solved
2. `02-design.md` — the proposed design
3. `03-decisions.md` — design decisions with rationale and alternatives

Optional documents listed in the Applicability Checklist are created sequentially from `04` onward when they apply (`NN-schema.md`, `NN-pipeline-changes.md`, `NN-module-integration.md`, `NN-notes.md`, `experiments/`). Check the box in the Applicability Checklist when the file lands; uncheck if removed. Keep the `## Documents` list in `README.md` in sync with the actual files in the directory.

Experiments are *optional* and usually appear part-way through an enhancement's life — once a specific claim emerges that benefits from a runnable proof. Do not create `experiments/` upfront when copying the template. Rules for experiments are documented in `000-TEMPLATE/README.md`; follow them.

## The `config.yaml` — metadata contract

`config.yaml` is the **sole source** of enhancement metadata. The README no longer carries a metadata table. The contract:

| Field            | Type / values                                                          | Notes                                                      |
| ---------------- | ---------------------------------------------------------------------- | ---------------------------------------------------------- |
| `id`             | 3-digit string (`"006"`)                                               | Must match dir prefix.                                     |
| `slug`           | kebab-case string                                                      | Must match dir suffix.                                     |
| `title`          | string                                                                 | Human title. Backticks for code identifiers are fine.      |
| `status`         | `draft` \| `accepted` \| `implemented` \| `superseded`                 | Lifecycle of the *design*, not the implementation.         |
| `created`        | `YYYY-MM-DD`                                                           | Set once, never changed.                                   |
| `updated`        | `YYYY-MM-DD`                                                           | Bumped on every meaningful edit (see RULES).               |
| `authors`        | list of strings, ≥1                                                    | Free-text. `OPM Contributors` is the current default.      |
| `implementation` | `{ status, date?, notes? }`                                            | See below.                                                 |
| `related`        | list of 3-digit ID strings                                             | Peer enhancements this one builds on or interacts with.    |
| `supersedes`     | list of 3-digit ID strings                                             | Enhancements this one fully replaces. Usually empty.       |
| `superseded_by`  | 3-digit ID string or `null`                                            | Set when a newer enhancement replaces this one.            |

`implementation.status ∈ { not-started, in-progress, partial, complete }`. `implementation.date` is the date of the latest implementation snapshot (omit while `not-started`). `implementation.notes` is a single-line summary; longer prose belongs in the README as a `> **Implementation status (YYYY-MM-DD).** …` quote block.

The full machine-readable contract lives in `library/enhancements/schema.cue` (`#EnhancementConfig`). Modify the schema only when the metadata contract genuinely changes; bumping `updated` does not touch the schema.

## RULES — MUST follow

- **Update `updated` on every meaningful edit.** Any change to `01-problem.md`, `02-design.md`, `03-decisions.md`, the schema doc, an experiment, or the README body bumps `config.yaml.updated` to today's date. Skip only for typo fixes that do not change meaning.
- **Implementation status reflects reality.** When code lands or an experiment concludes, update `implementation.status` and set `implementation.date` to today. The structured fields are the short summary; the multi-sentence prose lives as a `> **Implementation status (YYYY-MM-DD).** …` quote block at the top of the README. Keep prose and structured fields in sync — same date in both.
- **`id` matches directory prefix, `slug` matches directory suffix.** Always. After a copy-from-template, this is the most common drift; the validator catches it.
- **Cross-refs (`related`, `supersedes`, `superseded_by`) point to existing enhancement IDs.** No dangling references. If you supersede an enhancement, set both sides — the newer one's `supersedes` AND the older one's `superseded_by`.
- **`updated >= created` and `updated >= implementation.date`.** ISO 8601 strings sort correctly. The validator catches dates that go backwards.
- **Never re-introduce a metadata table to README.** The table was removed by design; `config.yaml` is canonical.

## Tasks (run from `library/`)

- `task enhancements:new SLUG=<slug> TITLE="<title>"` — scaffold a new enhancement from `000-TEMPLATE`. Auto-numbers the id, fills `config.yaml`, replaces title placeholders. Optional `AUTHOR="<name>"`.
- `task enhancements:list` — print a status table across all enhancements. Useful as the first thing to read when picking up unfamiliar work.
- `task enhancements:show ID=<id-or-slug>` — print full metadata + the document list for a single enhancement. `ID` accepts either the 3-digit id (`006`) or the slug (`platform-capabilities`).
- `task enhancements:vet` — validate every `config.yaml` against `enhancements/schema.cue` plus the cross-file consistency checks (id/slug ↔ dir, date ordering, dangling cross-refs). Run this before opening a PR that touches `enhancements/`.
- `task enhancements:vet:one ID=<id-or-slug>` — same validation, but for one enhancement only. Useful after editing a single config.yaml to confirm it still passes before running the full sweep.
- `task enhancements:graph` — (over)write `enhancements/GRAPH.md` with a Mermaid relationship diagram (`related` undirected, `supersedes` directed; nodes colored by status). GRAPH.md is fully generated — regenerate after editing any `config.yaml`, do not hand-edit.

All six tasks live in `library/.tasks/enhancements.yaml` and are wired into the root Taskfile via the `enhancements:` include.

## Status lifecycle

`draft → accepted → implemented`. `superseded` is set when a newer enhancement replaces this one — paired with `superseded_by` on the older entry and `supersedes` on the newer one (validator checks both directions exist; symmetry is your responsibility).
