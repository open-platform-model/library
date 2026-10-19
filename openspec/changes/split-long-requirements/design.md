## Context

OpenSpec 1.14.1 measures the body of each requirement: the lines between `### Requirement:` and the first line that starts with `#`, with blank lines, fenced lines and `**Key**:` lines dropped, joined by newlines. Scenarios do not count. More than 500 UTF-16 code units is a WARNING, and `--strict` fails on it. The check runs on main specs and on ADDED blocks of a delta. It does not run on MODIFIED blocks. A MODIFIED block that stays over 500 therefore passes `validate <change> --strict`, archives, and then fails `validate --specs --strict` on the spec.

State on `main` (origin/main `cbe6e66b`, OpenSpec 1.14.1, run in a scratch copy of `openspec/`): `validate --all --strict --no-interactive` exits 1 with 153 WARNING lines. 152 are in 23 main specs; one is in the open change `publish-go-api-bundle`. Three specs are clean (`consumer-build`, `cue-regression-canary`, `transform-input-fill`). No library spec has a placeholder Purpose (a search for `TBD` finds none), so no Purpose edit is needed. The open change `publish-go-api-bundle` adds only the new spec `docs-bundle` and modifies no existing spec, so no batch collides with it.

## Goals / Non-Goals

**Goals**

- Every main spec passes `openspec validate --specs --strict` under 1.14.1.
- Every rewrite is a split or a move into scenarios. No rule, value, default, exit code, citation or RFC 2119 strength is lost or changed.
- A pinned gate keeps it that way.

**Non-Goals**

- Any change of meaning, any new rule, any code.
- The spec `docs-bundle` of `publish-go-api-bundle` and any other open change.
- A shorter spec for its own sake: a requirement at or under 500 is left as it is.

## Research & Decisions

### Which rewrite form each requirement gets

**Context**: A rewrite must shorten the requirement body itself. Three loopholes pass the length check and remove nothing, so they are forbidden: text moved into `**Key**:` lines, text under a sub-heading of any level (a `####` heading is a scenario), and text in a fenced block. In a REMOVED block only `**Reason**:` and `**Migration**:` are allowed.
**Explored**: The four forms the validator and the archive accept, each run in a scratch copy: (a) MODIFIED with a shorter body and more scenarios; (b) MODIFIED plus ADDED, the parent keeps its header and all its scenarios; (c2) REMOVED plus ADDED under different names; (d) RENAMED plus MODIFIED under the new name. Unsafe: dropping or renaming a scenario inside MODIFIED, moving a live scenario from a MODIFIED parent into an ADDED block, and these pairs on one name in one change: MODIFIED+ADDED, MODIFIED+REMOVED, ADDED+REMOVED, RENAMED FROM+REMOVED, RENAMED TO+ADDED. (The b form is MODIFIED on the parent plus ADDED under new names, which is safe because the names differ.)
**Decision**: Choose by what the excess is, in this order of preference:

| The excess is | Form |
| --- | --- |
| Examples, lists, formats or rationale around one behavior | a |
| Several behaviors, and the old scenarios fit the parent that stays | b |
| Several behaviors, and the old scenarios belong to different behaviors | c2 |
| A name that must change with no change of rule | d, alone |

Prefer a, then b, then c2. Use c2 only when the old scenarios must move, because OpenSpec checks no scenario under c2: an old scenario in no new requirement disappears without an error. The kind labels of the exploration inventory (`format`, `list`, `several behaviors`) are a hint; the text is read before the form is chosen. Reversibility: high (hours). A wrong form is caught by the gate and the checks below, and a later change can rewrite the spec again.
**Rationale**: a and b keep every old name and every old scenario in place, so OpenSpec compares them. c2 is the only form that may drop a scenario, so it carries the ledger and the byte check.

### How a split keeps one behavior per requirement

**Context**: A split into pieces that need each other to be read is not a fix.
**Decision**: Each new requirement states one behavior, holds SHALL or MUST in its body, has at least one scenario with a body, and measures 500 or less. A clause moves from a body to a scenario THEN line only when it is a default or an exact value of the same behavior; its SHALL moves with it. A clause that adds or changes a condition stays in the body. A separate behavior goes to b or c2. A new name states the behavior in words and keeps no enhancement id or number in the header.
**Rationale**: Testable one by one is the reason for the limit.

### Repo scenario and citation rules for new text

**Context**: `openspec/config.yaml` binds the specs of this repo.
**Decision**: The new text follows these rules of the file: RFC 2119 keywords in every body; a requirement that implements an enhancement names the enhancement id beside the reason in words, in the body (first sentence or a trailing `Source:` line), never in the `### Requirement:` header and never with a decision number; scenarios show observable behavior, such as a valid load against an invalid load or a render success against a render failure. A `Source:` line and an `ADR-NNN` pointer stay in the body of the block that carries the rule they support, and they count toward the 500. The `**Source**:` metadata form is not used, because it is the forbidden loophole. A rewrite adds no scenario that states a case absent from the old text; a case the old text implies but never states is a question for the human at review.
**Rationale**: These are the rules a reviewer applies to any spec change here.

### Scenario ledger

**Context**: OpenSpec does not compare scenario text, and under c2 it does not check scenario names.
**Decision**: Every rewritten requirement gets a ledger in this file, added in the same commit as its delta, under a heading `### Ledger: <spec>` per spec:

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |

and a second table for each clause that leaves a body:

| Old clause | New place |
| --- | --- |

Forms a, b, d: every old name stays under the same requirement, byte for byte. Form c2: every old name reappears in one new requirement, and its WHEN and THEN lines stay byte for byte. A moved scenario is proven by the command below, not by eye. No ledger rows exist yet; each batch fills its own.

Per requirement, before the unit commits, run this in the worktree root (set `SPEC`, `REQ` and `CH=split-long-requirements`). It prints `same:`, `new:`, `DIFF:` or `LOST:` for each scenario and exits 1 on any `DIFF` or `LOST`. A `DIFF` not in the clause table is blocking; a `new:` line needs a reason in the ledger.

```bash
scen() { awk -v n="#### Scenario: $2" '$0==n{p=1} p&&$0!=n&&/^(#|##|###|####) /{p=0} p' "$1" |
  sed -e :a -e '/^[[:space:]]*$/{$d;N;ba' -e '}'; }
OLD=$(mktemp); git show "main:openspec/specs/$SPEC/spec.md" > "$OLD"
NEW="openspec/changes/$CH/specs/$SPEC/spec.md"; rc=0
while IFS= read -r n; do
  grep -qxF "#### Scenario: $n" "$OLD" || { echo "new:  $n"; continue; }
  if [ "$(scen "$OLD" "$n")" = "$(scen "$NEW" "$n")" ]; then echo "same: $n"
  else echo "DIFF: $n"; diff <(scen "$OLD" "$n") <(scen "$NEW" "$n"); rc=1; fi
done < <(grep -E '^#### Scenario: ' "$NEW" | sed 's/^#### Scenario: //')
while IFS= read -r n; do
  grep -qxF "#### Scenario: $n" "$NEW" || { echo "LOST: $n"; rc=1; }
done < <(awk -v h="### Requirement: $REQ" '$0==h{p=1;next} p&&/^##+ /&&!/^#### /{p=0} p' "$OLD" |
  grep -E '^#### Scenario: ' | sed 's/^#### Scenario: //')
rm -f "$OLD"; [ "$rc" = 0 ]
```

Also per unit: search the repo for each old header before a rename, split or c2, because OpenSpec checks no reference: `grep -rn --exclude-dir=archive --exclude-dir=.git --exclude-dir=.cue-cache "<old header>" .`. On `main` the only non-skill files that name `openspec/specs` are `AGENTS.md` and the `kernel-runtime` spec itself; the search still runs for each header.

### The gate of each unit

**Context**: `validate <change> --strict` alone is not enough, because it skips MODIFIED text. The gate runs in a scratch copy of `openspec/`, never in the worktree, with the pinned CLI 1.14.1 and `jq`.
**Decision**: After each batch, run this block in one shell call. `WT` is the worktree, `CH` is `split-long-requirements`, `S` is a scratch folder outside the worktree, `N` is the number of long requirements the whole change has rewritten so far, earlier batches included (28, 56, 84, 112, 140, 152 in plan order). Set `LAST=1` only for the final unit. Exit 0 after `GATE PASS` is a pass; any other exit is a fail.

```bash
export OPENSPEC_TELEMETRY=0
errs() { local j; j=$(openspec validate --all --strict --json --no-interactive 2>/dev/null)
  printf '%s' "$j" | jq -e '.items | type == "array"' > /dev/null || return 1
  printf '%s' "$j" |
  jq -r '.items[] | .type as $t | .id as $i | .issues[]? | select(.level == "ERROR") | "\($t) \($i) \(.path)"' | sort -u; }
rm -rf "$S"; mkdir -p "$S/base"
git -C "$WT" archive main openspec | tar -x -C "$S/base"
cp -r "$WT/openspec" "$S/openspec"
(cd "$S/base" && openspec validate --all --strict --no-interactive > base.out 2>&1)
(cd "$S/base" && errs > base.err) || { echo "GATE FAIL: no ERROR list for main (is jq installed?)"; exit 1; }
before=$(grep -c WARNING "$S/base/base.out")
cd "$S" || exit 1
openspec validate "$CH" --strict --no-interactive || { echo "GATE FAIL: validate $CH"; exit 1; }
openspec archive "$CH" --yes                         || { echo "GATE FAIL: archive $CH"; exit 1; }
openspec validate --all --strict --no-interactive > gate.out 2>&1; all=$?
errs > gate.err || { echo "GATE FAIL: no ERROR list for the copy (is jq installed?)"; exit 1; }
cat gate.out
grep -F "Archive would refuse" gate.out && { echo "GATE FAIL: Archive would refuse"; exit 1; }
after=$(grep -c WARNING gate.out)
echo "validate --all exit $all; WARNING lines: before $before, after $after, rewritten $N"
new=$(comm -13 base/base.err gate.err) || exit 1
[ -z "$new" ] || { echo "GATE FAIL: new ERROR item:"; echo "$new"; exit 1; }
if [ -n "$LAST" ]; then openspec validate --specs --strict --no-interactive || { echo "GATE FAIL: validate --specs"; exit 1; }
else [ "$after" -eq $((before - N)) ] || { echo "GATE FAIL: warnings did not fall by $N"; exit 1; }; fi
echo "GATE PASS"
```

Earlier unit: it passes when no ERROR item is new and the WARNING count fell by `N` (from 153 on `main`: 125, 97, 69, 41, 13 after B1 to B5 in plan order). For the batch's specs also run `openspec validate <spec> --type spec --strict --json 2>&1` and read it: no WARNING or ERROR for a rewritten requirement. `N` counts old requirements rewritten, so a c2 split that adds requirements still lowers the count by one per old requirement.
Last unit (`LAST=1`): it judges the main specs and the change, not other open changes. It passes when `validate split-long-requirements --strict` exits 0, `validate --specs --strict` exits 0 after the archive, `gate.out` has no "Archive would refuse" line, and no ERROR item is new. `validate --all --strict` may still exit 1 for the one warning in `publish-go-api-bundle`; the block prints that and does not fail on it.
**Rationale**: Each unit proves the count fell by exactly what it rewrote, with no new ERROR and no broken open change.

### Batch order

**Context**: 23 specs, 152 requirements. No spec is in two batches, so the batches are independent: no rewrite in one batch depends on another.
**Explored**: (1) Largest first, B1 to B6 as the exploration planned them (first-fit decreasing, cap 28 per unit). (2) Smallest first, to prove the method on twelve requirements. (3) By risk: workflow specs (`cascade-wiring`, `release-pipeline`, `api-diff-check`) first, because they describe CI files that other repos read.
**Decision**: Option 1, B1 to B6. B1 holds `kubernetes-tier`, the biggest spec (25 long of 32 requirements), where the form choice is practised most, and the three `api-diff-check` requirements. Mistakes of form show early, in the unit with the most rewrites, and later units copy the settled pattern. B6 is last and smallest. Reversibility: high; the human may reorder, because the batches do not depend on each other. `N` in the gate follows the order actually run.
**Rationale**: The plan order is simple to check against the inventory; option 3 mixes risk and size without a measured reason, and option 2 leaves the hardest spec for last.

### The gate task and the CI job

**Context**: The library has no OpenSpec task and no OpenSpec CI job. Its `task check` runs fmt, vet, lint, test, docs bundle, cascade wiring, cite-lint, consumer-build tests and api-diff tests. `cli` has `openspec:install` and `openspec:check` that install a global npm package at a fixed version.
**Explored**: (1) Copy the `cli` form: `npm install -g @fission-ai/openspec@<exact>`, then `openspec validate --all --strict --no-interactive`. (2) Install into the repo's gitignored `.bin/` with `npm install --prefix`, `--ignore-scripts` and an exact version, so the task touches no global state. (3) Run the CLI with `npx` at an exact version at each call. (4) No gate.
**Decision**: Option 2. Tasks `openspec:install` (exact `@fission-ai/openspec@1.14.1`, `--ignore-scripts`, into `.bin/`, reused when `openspec --version` already prints 1.14.1) and `openspec:check` (`openspec validate --all --strict --no-interactive` from that binary, with a precondition that the binary exists). `task check` runs `openspec:check` after `cite-lint`; `check:fast` stays as it is. One CI job, in a new workflow `openspec.yml` on `push` to `main` and on `pull_request`, with `permissions: contents: read`, `actions/checkout` with `persist-credentials: false`, the same SHA-pinned `actions/setup-node` and `go-task/setup-task` the repo or `cli` already pins (SHAs copied from existing workflows, none is new), and a step that runs `task openspec:install openspec:check`. The AGENTS.md row of `task check` gains the new step. Reversibility: high (hours).
**Rationale**: This is a supply-chain choice, so it follows the repo's habit for tools (a pinned version, no install script from a mutable ref, `.bin/` gitignored). Option 1 writes global state and runs install scripts. Option 3 hides the version in the call. Option 4 lets the 500-character limit slip back in; 1.14.1 has no flag to turn the limit off, so a gate is the only guard. Two facts are not verified here and the first gate task checks them: that the package works with `--ignore-scripts`, and that 1.14.1 has been published for the seven-day cooldown that the workspace supply-chain rule asks of a new pin. If either fails, the task stops and the human decides (drop `--ignore-scripts` with the reason recorded, or wait).

### Delivery order of the gate

**Decision**: The gate section is the last commit of the branch, and the PR that carries it is opened after `publish-go-api-bundle` lands without long-requirement warnings. Before that, `validate --all --strict` covers that open change and exits 1 on one warning, so the new CI job would fail its own PR. The six batch sections need no such wait; the supervisor may deliver them as an earlier PR and the gate as a second one. Reversibility: high.

## Security

Assets: none new; the change edits spec text and adds one dev tool. Trust boundary: one new, the npm registry, crossed in `openspec:install` and in the CI job. Baseline: the workspace supply-chain rules (exact pin, no install script, cooldown, read-only workflow token, SHA-pinned actions). Threat: a hijacked release of the pinned package runs code on a developer machine or a runner; the exact pin and `--ignore-scripts` narrow it, and the token is read-only with no secret in the job. Residual risk: the package is trusted when it runs; the owner is the library maintainer, who bumps the pin by hand. No sensitive detail is in this record.

## Risks / Trade-offs

- **A rewrite silently drops a rule or a scenario.** The ledger, the byte-equality command and the gate make it visible; the reviewer repeats all three per batch. Trade-off: extra tables in this file, which grows by up to about 150 ledger entries.
- **A split spreads one rule over several requirements.** One behavior per requirement is the point of the limit, but a reader now follows several headers. The rule above (a split piece must stand alone) limits it.
- **Seven sections in one change.** Each ends green and leaves `main` releasable, because a spec edit touches no code. If the human wants a smaller cut, the batches can ship as sequential changes.
- **An open change edits a requirement that c2 removes.** Today only `publish-go-api-bundle` is open and it edits no existing requirement. The gate fails on any "Archive would refuse" line, and each unit lists `openspec/changes/` first.
- **Operational cost.** One new CI job and one Taskfile pair, owned by the library maintainer. The pin lives in the Taskfile only; the CI job calls the task.
