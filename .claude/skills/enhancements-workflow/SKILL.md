---
name: enhancements-workflow
description: Conventions for creating, editing, and tracking OPM library enhancements (design packages under enhancements/). Covers the TEMPLATE, the mandatory + optional documents, the config.yaml metadata contract, and the validator/list tasks.
user-invocable: true
---

# Enhancements Workflow

## When this applies

Any task touching `enhancements/`: creating a new enhancement, editing an existing design doc, updating implementation status as code lands, recording the outcome of an experiment, adding cross-references between enhancements, or archiving / marking one superseded.

## Directory layout

Enhancements live under `enhancements/NNN-<kebab-slug>/`.

- `NNN` is a zero-padded three-digit string, the next available number after the highest existing enhancement.
- The slug is kebab-case, matches `^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`, and is a short identifier — not a full sentence. Long enough to disambiguate, short enough to type.
- Both `id` (in `config.yaml`) and `slug` MUST match the directory name; `task enhancements:vet` enforces this.

## Start from the TEMPLATE — always

Use the scaffolding task. It auto-numbers the id from the highest existing enhancement, copies the template, fills `config.yaml` with real values (id, slug, title, today's date, author), and replaces the `{Enhancement Title}` placeholder in the README and the three mandatory document headers:

```bash
task enhancements:new SLUG=<kebab-slug> TITLE="<Human Title>"
```

Optional `AUTHOR="<name>"` overrides the default `OPM Contributors`. The task prints the new directory path and a checklist of remaining placeholders to fill in (`{Summary}`, per-document descriptions, Cross-References).

If you need to scaffold without the task (e.g., recovering from a botched run, or scaffolding into an unusual location), the manual fallback is `cp -r enhancements/000-TEMPLATE enhancements/NNN-<slug>` followed by hand-editing the placeholders. The task does this for you and is the preferred path; the manual route exists only as an escape hatch.

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
| `status`         | `draft` \| `accepted` \| `implemented` \| `superseded`                 | Lifecycle of the *design*. See [Status lifecycle](#status-lifecycle). |
| `semver`         | `major` \| `minor` \| `none`                                           | Public API impact per Constitution VI. Optional in `draft`; **required** at `accepted`/`implemented`/`superseded`. No `patch` — enhancements that small don't warrant a design package. |
| `created`        | `YYYY-MM-DD`                                                           | Set once, never changed.                                   |
| `updated`        | `YYYY-MM-DD`                                                           | Bumped on every meaningful edit (see RULES).               |
| `authors`        | list of strings, ≥1                                                    | Free-text. `OPM Contributors` is the current default.      |
| `implementation` | `{ status, date?, notes? }`                                            | See below. *Independent axis from `status`* — see lifecycle table. |
| `slices`         | list of OpenSpec change slugs (optional)                               | E.g. `["2026-05-08-add-kernel-struct"]`. Each must exist under `openspec/changes/archive/`. Use for umbrella enhancements that ship as multiple OpenSpec changes; omit for enhancements that land in a single direct edit. |
| `related`        | list of 3-digit ID strings                                             | Peer enhancements this one builds on or interacts with.    |
| `competes_with`  | list of 3-digit ID strings (optional)                                  | Alternative approaches solving the same problem. Symmetric — if A lists B, B MUST list A. Rendered as dashed-red edges in `GRAPH.md`. |
| `supersedes`     | list of 3-digit ID strings                                             | Enhancements this one fully replaces. Usually empty.       |
| `superseded_by`  | 3-digit ID string or `null`                                            | Set when a newer enhancement replaces this one. **Required non-null** at `status: superseded`. |

`implementation.status ∈ { not-started, in-progress, partial, complete }`. `implementation.date` is the canonical completion date — **required when `implementation.status == complete`, forbidden otherwise**. Snapshot dates on `partial`/`in-progress`/`not-started` go stale immediately; keep them out of structured metadata and put the date inline in `notes` if it matters. `implementation.notes` is a single-line summary; longer prose belongs in the README as a `> **Implementation status (YYYY-MM-DD).** …` quote block.

The full machine-readable contract lives in `enhancements/schema.cue` (`#EnhancementConfig`). Modify the schema only when the metadata contract genuinely changes; bumping `updated` does not touch the schema.

### `status` and `implementation.status` are orthogonal

The two axes — design lifecycle (`status`) and code lifecycle (`implementation.status`) — track independent realities. The schema constrains only the combinations that would be incoherent:

| `status` | Legal `implementation.status` | Reading |
| --- | --- | --- |
| `draft` | any | Design still forming. Scaffolding code may have landed (e.g., 005 today). |
| `accepted` | `not-started`, `in-progress`, `partial` | Design frozen, code in flight. Cannot be `complete` (that's `implemented`). |
| `implemented` | `complete` only | All design intent shipped. Deferred slots ⇒ carve them into a follow-up enhancement and keep this one `accepted`. |
| `superseded` | any | History. Don't retroactively edit. |

Distinction `partial` vs `complete`: `partial` means explicit design slots are deferred to a named follow-up enhancement. `complete` means everything in scope shipped, even if minor deviations are documented in the Deviations section.

## RULES — MUST follow

- **Update `updated` on every meaningful edit.** Any change to `01-problem.md`, `02-design.md`, `03-decisions.md`, the schema doc, an experiment, or the README body bumps `config.yaml.updated` to today's date. Skip only for typo fixes that do not change meaning.
- **Implementation status reflects reality.** When code lands, update `implementation.status` (and `implementation.notes`) to match. `implementation.date` is set the day `implementation.status` reaches `complete` — required at that point, forbidden before. The structured fields are the short summary; the multi-sentence prose lives as a `> **Implementation status (YYYY-MM-DD).** …` quote block at the top of the README, with the same date.
- **`id` matches directory prefix, `slug` matches directory suffix.** Always. After a copy-from-template, this is the most common drift; the validator catches it.
- **Cross-refs (`related`, `supersedes`, `superseded_by`, `competes_with`) point to existing enhancement IDs.** No dangling references. If you supersede an enhancement, set both sides — the newer one's `supersedes` AND the older one's `superseded_by`. If two enhancements compete, both MUST list each other in `competes_with`.
- **`updated >= created` and `updated >= implementation.date`.** ISO 8601 strings sort correctly. The validator catches dates that go backwards.
- **Never re-introduce a metadata table to README.** The table was removed by design; `config.yaml` is canonical.
- **No `{Placeholder}` strings left in mandatory docs.** `vet` greps for `{Capital...}` patterns outside code fences and hard-fails. Either fill them in or delete the section.

## Tasks (run from `library/`)

- `task enhancements:new SLUG=<slug> TITLE="<title>"` — scaffold a new enhancement from `000-TEMPLATE`. Auto-numbers the id, fills `config.yaml`, replaces title placeholders. Optional `AUTHOR="<name>"`.
- `task enhancements:list` — print a status table across all enhancements. Useful as the first thing to read when picking up unfamiliar work.
- `task enhancements:show ID=<id-or-slug>` — print full metadata + the document list for a single enhancement. `ID` accepts either the 3-digit id (`006`) or the slug (`platform-capabilities`).
- `task enhancements:vet` — **hard gate (CI).** Validate every `config.yaml` against `enhancements/schema.cue` plus cross-file consistency: id/slug ↔ dir, date ordering, dangling cross-refs, `competes_with` symmetry, `slices` existence under `openspec/changes/archive/`, and `{Placeholder}` strings in mandatory docs. Run before opening any PR that touches `enhancements/`.
- `task enhancements:vet:one ID=<id-or-slug>` — same gate, single enhancement. Use after editing one `config.yaml`.
- `task enhancements:check [ID=<id-or-slug>]` — **soft gate (pre-PR aid).** Audits prose conventions that don't fit a schema: scope section, decision headings, Open Questions block, implementation snapshot quote block (with matching date), Deviations section, superseded-by quote block. See [Per-status checklist](#per-status-checklist) for what fires at each status. Exits non-zero on warnings so it can wire into pre-commit hooks.
- `task enhancements:graph` — (over)write `enhancements/GRAPH.md` with a Mermaid relationship diagram (`related` undirected solid, `supersedes` directed solid, `competes_with` undirected dashed-red; nodes colored by status). Regenerate after editing any `config.yaml`; do not hand-edit.

All tasks live in `.tasks/enhancements.yaml` and are wired into the root Taskfile via the `enhancements:` include.

## Status lifecycle

`draft → accepted → implemented`. `superseded` is set when a newer enhancement replaces this one — paired with `superseded_by` on the older entry and `supersedes` on the newer one. Each transition tightens what `vet` and `check` require.

### Per-status checklist

Notation: **[H]** = hard-enforced by `vet` (PR-blocking). **[S]** = soft-enforced by `check` (warns, intended for pre-PR).

#### `draft`

The cheap-entry state. Be lenient — this is where ideas form.

- **[H]** `id`/`slug` match dir prefix/suffix
- **[H]** mandatory docs (`README.md`, `01-problem.md`, `02-design.md`, `03-decisions.md`) exist
- **[H]** no `{Capital…}` placeholder strings in mandatory docs (outside code fences)
- **[H]** `created` set, `updated >= created`, `updated >= implementation.date`
- **[H]** all cross-refs resolve (`related`, `supersedes`, `superseded_by`, `competes_with`)
- **[H]** `implementation.status ≠ complete` (that's reserved for `implemented`)

Not required at draft: `semver`, `slices`, scope section, decisions content, OQ list, impl block.

#### `accepted`

Design frozen, ready for slicing.

- Everything `draft` requires
- **[H]** `semver: major | minor | none` set
- **[H]** `competes_with` (if set) symmetric — each named peer lists this id back
- **[H]** `slices` (if set) — every listed slug exists under `openspec/changes/archive/`
- **[H]** `implementation.status ∈ {not-started, in-progress, partial}`
- **[S]** README contains `## Scope` with `### In scope` + `### Out of scope`
- **[S]** decisions file contains at least one `### DN:` heading
- **[S]** decisions file contains `## Open Questions` block (may say "None")

#### `implemented`

Code has landed. Status is retrospective — written when the last slice archived.

- Everything `accepted` requires
- **[H]** `implementation.status: complete` (the schema enforces this — `partial` keeps the enhancement at `accepted` with a follow-up enhancement named in the notes)
- **[H]** `implementation.date` set
- **[H]** `implementation.notes` set, non-empty
- **[S]** README contains `> **Implementation status (YYYY-MM-DD).**` quote block, date matching `implementation.date`
- **[S]** README contains `## ...Deviation...` section (may say "None")

#### `superseded`

Terminal state. Don't re-edit history; just check the handoff.

- **[H]** `superseded_by` set (non-null)
- **[H]** newer enhancement's `supersedes` includes this id (symmetry already checked by `vet`)
- **[S]** README has top-of-file `> **Superseded by NNN (YYYY-MM-DD).**` quote block linking the replacement
- **[S]** short migration paragraph in README

### Workflow: advancing status

When promoting an enhancement to a new status, run both gates before committing:

```bash
task enhancements:vet:one ID=<id>   # MUST pass — hard gate
task enhancements:check ID=<id>     # SHOULD pass — soft gate, document any warnings you choose to defer
```

If `vet` passes but `check` warns, either fix the warnings or document why they don't apply in the PR description. Don't silently ignore them — they exist because the prose convention helps future readers.
