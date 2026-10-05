# Tasks: record-the-walkthrough-decisions

Depends on: nothing unmerged. Merges last among the round-8 library changes (proposal.md, Order). Line numbers are at `origin/main` 88afdfb; re-read every site before editing. No identifier, signature, error text or test behaviour changes: in `.go`, `.sh` and `.yml` files only comment lines change (`git diff -U0 origin/main -- '*.go' '*.sh' '*.yml' | grep '^[-+][^-+]' | grep -v '^[-+]\s*//\|^[-+]\s*#'` prints nothing). Every section's gate: `TMPDIR=<absolute mktemp -d> task check` and `openspec validate record-the-walkthrough-decisions --strict` pass. Before the first gate run, copy (never symlink) the main checkout's `.cue-cache` into the worktree. No committed file may name an agent role, a session-local decision number, a workflow run or a session, and no file or commit message may contain the word git's guard refuses. No swept file may use the phrases "kernel-plan walkthrough" or "beta.1 walkthrough": the check below matches them.

The walkthrough-id check used below (design.md, decision 3), two commands that each print nothing when the tree is clean:

```
git grep -nE "[Oo]wner('s)? (walkthrough )?decisions? [a-j][1-5]\b|walkthrough (decisions?|tasks?) [a-j][1-5]\b|(beta\.1|kernel[ -]plan) walkthrough|owner decision 2026-10-0[23]" -- . ':!openspec/changes' ':!adr/013-*'
git grep -nE "\b(decisions?|tasks?) [a-j][1-5]\b" -- . ':!openspec/changes' ':!adr/013-*' | grep -v 'ADR-013, decisions\? '
```

The check is not wired into `task check`; library#198, which tracks the ungated one-home checks, should also cover it.

## 1. ADR-013 (adr)

- [x] 1.1 Write `adr/013-kernel-plan-walkthrough-decisions.md`, title "ADR-013: The kernel-plan walkthrough decisions", in the `adr/TEMPLATE.md` shape, as design.md decision 1 lays out: Status Accepted (2026-10-05), naming this change; Context (the beta.1 kernel plan, the owner's answer to each task on 2026-10-02 and 2026-10-03, ids cited across library, cli and opm-operator with no in-repo record); Decision (this ADR is the record, the citation form `ADR-013, decision <id>`, then the table); Consequences as bold-labelled paragraphs (ids resolve; the Landed column is a snapshot at acceptance and is not edited per later PR; committed files cite the ADR, while merged commit messages and PR bodies keep bare ids, which the table resolves; cli and opm-operator cite it as `library` ADR-013).
- [x] 1.2 The table: one row per decided id from design.md decision 1 (`a1`, `a3` to `a5`, `b1` to `b5`, `c1`, `c2`, `c4`, `c5`, `d1` to `d5`, `e1` to `e5`, `f1` to `f5`, `g1` to `g5`, `h1` to `h4`, `i1` to `i5`, `j1` to `j5`), with the final answer only and "(replaces an earlier answer)" where the owner replaced one. Before writing, check each Landed entry with `gh pr view <n> -R open-platform-model/<repo> --json title,state,mergedAt` and drop or correct any PR that is not merged or does not carry that decision. The Landed column lists merged PRs only; an issue or a closed proposal goes in the decision text. No row restates an ADR's or enhancement's rationale; it names it.
- [x] 1.3 Verify: the ADR names no agent role, session-local decision number, workflow id or session (`grep -nE '[Ss]upervisor|\bSD[0-9]+\b|wf_' adr/013-*.md` prints nothing); every id `a1` to `j5` except `a2` and `c3` has exactly one row; every Landed entry has the `repo#N` form.
- [x] 1.4 Gates green, then commit `docs(adr): add ADR-013, the kernel-plan walkthrough decisions`.

## 2. Cite ADR-013 (library sweep)

- [x] 2.1 `AGENTS.md`: `:297` "owner decision j4" and `:299` "owner decision j4 of the beta.1 walkthrough" become `ADR-013, decision j4`; `:370-371` "owner decision c4 of the beta.1 walkthrough" becomes `ADR-013, decision c4`; `:379` "an owner decision by its walkthrough id" becomes "a decision ADR-013 records, cited as `ADR-013, decision <id>`", matching the MODIFIED `kernel-runtime` requirement.
- [x] 2.2 `.github/workflows/consumer-build.yml:3` and `.tasks/consumer-build.sh:10-11`: the comment cites `ADR-013, decision j4` in place of the owner decision and walkthrough date. Comment lines only.
- [x] 2.3 `adr/012-matching-stays-in-the-library-glue.md:5`: "Records the owner's answer to where the matching algorithm lives, given in the beta.1 kernel plan walkthrough" becomes "Records ADR-013, decision d5: where the matching algorithm lives." The rest of the Status line stays.
- [x] 2.4 `opm/catalog/provides_parity_test.go:64-65`: "(owner decision j3, beta.1 walkthrough)" becomes "(ADR-013, decision j3)", rewrapped. Comment lines only.
- [x] 2.5 `openspec/specs/api-diff-check/spec.md:4` (Purpose, in place per design.md decision 4): "(owner decision j4 of the beta.1 walkthrough)" becomes "(ADR-013, decision j4)". No other main-spec line is edited by hand.
- [x] 2.6 Verify: the walkthrough-id check with `':!openspec/specs'` added to both commands prints nothing. Without that exclusion it prints only the main-spec lines this change's deltas replace (`consumer-build` four, `api-diff-check` two, `kernel-runtime` one, `kubernetes-tier` two). `git grep -n 'by its walkthrough id' -- . ':!openspec/changes'` prints only the `kernel-runtime` main-spec line.
- [x] 2.7 Gates green, then commit `docs: cite ADR-013 for the kernel-plan walkthrough decisions`.

## 3. Cite ADR-013 (archived changes)

- [x] 3.1 In every file under `openspec/changes/archive/` that the walkthrough-id check (both commands, run over `openspec/changes/archive` instead of `.`) prints, the citation cites ADR-013: "owner decision j4 (beta.1 walkthrough)", "walkthrough task f5", "owner decision 2026-10-03" and the bare "decision d2" or "task e3" become `ADR-013, decision <id>`; a sentence that names the walkthrough without an id, or in another wording ("walkthrough item i1", "the kernel checklist walkthrough"), names ADR-013 instead; an out-of-repo log path is dropped. Only citation words change: no requirement, scenario, task box state or rationale changes, and no archived file is moved or deleted.
- [x] 3.2 Verify: both commands, run over `openspec/changes/archive`, print nothing except `2026-10-02-prepare-release-cascade/proposal.md:35`, whose "owner decision 2026-10-02" is a release-cascade decision that cites `RELEASING.md`, not a walkthrough id; `git grep -niE walkthrough -- openspec/changes/archive` shows no citation of the walkthrough that names neither ADR-013 nor a change name; `git diff --stat origin/main -- openspec/changes/archive` touches only files the 3.1 check printed; `openspec validate --all --strict` passes.
- [x] 3.3 Gates green, then commit `docs(openspec): cite ADR-013 in the archived changes`.

## 4. Absorb main and sweep again

- [ ] 4.1 After the other round-8 library changes merge: `git fetch origin` and `git merge origin/main` (the squash drops the merge commit). Keep their text; where one of them changed a requirement this change's deltas copy, re-copy the delta from the new main spec and change only the Source words.
- [ ] 4.2 Re-run the walkthrough-id check, the 2.6 greps and the 3.2 check over the merged tree. Apply sections 2 and 3's form to any new citation the merged changes added (for example a comment citing g2, d1 or e4, or a newly archived change).
- [ ] 4.3 Refresh ADR-013's Landed column against the PRs merged by now in every repo it names, including the round-8 library PRs (the g2, d1 and e4 refinements) and any frontend half that has merged. Keep "pending" where a decided half has not merged.
- [ ] 4.4 Cross-check: `task api:diff` reports no incompatible change of this branch. Run `.tasks/consumer-build.sh` against fresh clones of cli and opm-operator `main` (each in its own work directory under the scratch dir), and both build and vet green. `openspec validate --all --strict` passes.
- [ ] 4.5 Gates green. If 4.1 to 4.3 changed anything, commit `docs(adr): bring ADR-013 up to date with main`; otherwise record "no sweep changes" in this box.

## 5. Verify and archive

- [ ] 5.1 The repo's verify skill (`openspec verify`) reports no CRITICAL finding.
- [ ] 5.2 `openspec archive record-the-walkthrough-decisions -y`, then the walkthrough-id check (without the `openspec/specs` exclusion) prints nothing and `openspec validate --all --strict` passes. The archive rides the PR.
- [ ] 5.3 Commit `chore(openspec): archive record-the-walkthrough-decisions`.
