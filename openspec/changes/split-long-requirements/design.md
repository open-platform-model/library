## Context

OpenSpec 1.14.1 measures the body of each requirement: the lines between `### Requirement:` and the first line that starts with `#`, with blank lines, fenced lines and `**Key**:` lines dropped, joined by newlines. Scenarios do not count. More than 500 UTF-16 code units is a WARNING, and `--strict` fails on it. The check runs on main specs and on ADDED blocks of a delta. It does not run on MODIFIED blocks. A MODIFIED block that stays over 500 therefore passes `validate <change> --strict`, archives, and then fails `validate --specs --strict` on the spec.

State on `main` (origin/main `cbe6e66b`, OpenSpec 1.14.1, run in a scratch copy of `openspec/`): `validate --all --strict --no-interactive` exits 1 with 153 WARNING lines. 152 are in 23 main specs; one is in the open change `publish-go-api-bundle`. Three specs are clean (`consumer-build`, `cue-regression-canary`, `transform-input-fill`). No library spec has a placeholder Purpose (a search for `TBD` finds none), so no Purpose edit is needed. The open change `publish-go-api-bundle` adds only the new spec `docs-bundle` and modifies no existing spec, so no batch collides with it.

## Goals / Non-Goals

**Goals**

- Every main spec passes `openspec validate --specs --strict` under 1.14.1.
- Every rewrite is a split or a move into scenarios. No rule, value, default, exit code, citation or RFC 2119 strength is lost or changed.
- A per-unit gate proves each rewrite; the CI gate that keeps it that way is a separate change.

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

Forms a, b, d: every old name stays under the same requirement, byte for byte. Form c2: every old name reappears in one new requirement, and its WHEN and THEN lines stay byte for byte. A moved scenario is proven by the command below, not by eye. The ledgers are in the section "Ledger" at the end of this file; each batch adds its own.

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
Last unit (`LAST=1`, the sixth): it judges the main specs and the change, not other open changes. It passes when `validate split-long-requirements --strict` exits 0, `validate --specs --strict` exits 0 after the archive, `gate.out` has no "Archive would refuse" line, and no ERROR item is new. `validate --all --strict` may still exit 1 for the one warning in `publish-go-api-bundle`; the block prints that and does not fail on it.
**Rationale**: Each unit proves the count fell by exactly what it rewrote, with no new ERROR and no broken open change.

### Batch order

**Context**: 23 specs, 152 requirements. No spec is in two batches, so the batches are independent: no rewrite in one batch depends on another.
**Explored**: (1) Largest first, B1 to B6 as the exploration planned them (first-fit decreasing, cap 28 per unit). (2) Smallest first, to prove the method on twelve requirements. (3) By risk: workflow specs (`cascade-wiring`, `release-pipeline`, `api-diff-check`) first, because they describe CI files that other repos read.
**Decision**: Option 1, B1 to B6. B1 holds `kubernetes-tier`, the biggest spec (25 long of 32 requirements), where the form choice is practised most, and the three `api-diff-check` requirements. Mistakes of form show early, in the unit with the most rewrites, and later units copy the settled pattern. B6 is last and smallest. Reversibility: high; the human may reorder, because the batches do not depend on each other. `N` in the gate follows the order actually run.
**Rationale**: The plan order is simple to check against the inventory; option 3 mixes risk and size without a measured reason, and option 2 leaves the hardest spec for last.

### The CI gate moves to a separate change

**Decision**: This change does not add the Taskfile tasks or the CI job that run `openspec validate --all --strict`. The gate moves to a separate change after `publish-go-api-bundle` lands, because until then `validate --all --strict` covers that open change and exits 1 on its one warning. The per-unit gate block above stays: it runs in a scratch copy and is not the CI gate. Reversibility: high.

### Batch library-B1: the form choices that later batches copy

**Context**: B1 holds `kubernetes-tier` (25 long requirements, 85 KB) and `api-diff-check` (3). Most long bodies hold several rules and a long `Source:` line.
**Decision**: Form a (MODIFIED) was not enough, because the excess was rules, not examples. 8 requirements use b (MODIFIED parent kept, split-off rules ADDED under new names) and 20 use c2 (REMOVED plus ADDED), where the old scenarios belong to different rules. A c2 split names each piece after the one rule it states, and every old scenario moves to the piece it tests, with its WHEN and THEN lines unchanged. Rule tables (the weights, the escape rules of the render digest, the field list of the label package, and the ordered conditions of the delete verdict, the apply verdict and the hold verdict) move into the THEN and AND lines of one new scenario; the body keeps the SHALL and the first-match rule, names the scenario and, for the delete and apply verdicts, lists the outcomes. A new requirement gets a restatement scenario, built from words of the old text, where the old scenarios cannot move to it. Where an old sentence had no SHALL and its new body needs one, the verb "is" becomes "SHALL be"; the ledger lists each such change in its clause table.
**Rationale**: One rule per requirement is the point of the limit. Reversibility: high.

### Source line placement

**Context**: A `Source:` line counts toward the 500 characters. Three `kubernetes-tier` requirements have a Source line of 495, 514 and 517 characters, which no body can carry whole, and other long Source lines leave little room for the rule.
**Decision**: A split block repeats the old requirement's whole `Source:` line when it fits. When it does not fit, the block carries the short form `Source: enhancement 0012.` (the enhancement that every line cites), or the title that supports its rule. Every title of an old Source line stays in at least one block of the same old requirement. Where the old text ties a Source line to a paragraph (the second paragraph of the apply-verdict requirement), the blocks of that paragraph keep that citation. Elsewhere the old text does not tie a title to a clause, so the assignment of a title to a block is a judgement: it follows the topic of the title (the deletion sequence, the order, the apply guard, the tier placement). The three requirements whose line cannot sit in one body split their titles; the table lists how each old requirement was handled.
**Rationale**: The 500-character limit and the rule that the citation stays in the body of the block that carries the rule leave no other choice. A title beside the wrong block is a citation error, not a rule change. Reversibility: high.

| Old requirement | Old Source line (characters) | Whole line repeated | Short form `Source: enhancement 0012.` or inline |
| --- | --- | --- | --- |
| The Kubernetes tier lives at opm/k8s beside the kernel | 54 | 3 of 3 pieces | 0 |
| The Kubernetes tier imports no cluster client or controller framework | 54 | 5 of 5 pieces | 0 |
| Nothing in the library performs a cluster action | 397 | none (split by titles) | 6 |
| Kind-class order is a Kubernetes fact that lives in the tier | 172 | 3 of 4 pieces | 1 |
| Deletion plans come from the inventory and the live objects | 186 | 3 of 3 pieces | 0 |
| The label vocabulary names labels and never stamps them | 250 | 3 of 4 pieces | 1 |
| Compiled objects convert to Kubernetes objects through one export | 308 | 1 of 5 pieces | 4 |
| One weight table orders apply and delete | 172 | 3 of 4 pieces | 1 |
| Apply stages follow the weight table | 279 | 1 of 3 pieces | 2 |
| Readiness evaluation is a pure function over fetched objects | 166 | 8 of 11 pieces | 3 |
| The ownership verdicts are pure | 148 | 3 of 3 pieces | 0 |
| The delete verdict skips with a reason or proceeds with the judged objec | 495 | none (split by titles) | 5 |
| The adopt annotation is the only override and the refusal names it | 208 | 5 of 5 pieces | 0 |
| Inventory entries are plain values built from objects | 286 | 2 of 3 pieces | 1 |
| The stale set is component-blind | 286 | 2 of 3 pieces | 1 |
| The inventory digest hashes a canonical field encoding | 286 | 1 of 3 pieces | 2 |
| The render digest ignores only the managed-by label value | 111 | 8 of 8 pieces | 0 |
| The deletion plan orders inventory entries by kind-class delete order | 320 | 1 of 4 pieces | 3 |
| The deletion transition names one action per call | 517 | none (split by titles) | 5 |
| Action outcomes are classified by the library | 294 | 4 of 6 pieces | 2 |
| The deletion state is a serialisable value the caller owns | 146 | 3 of 4 pieces | 1 |
| The hold verdict is decided from the policy and the plan's outcome | 330 | 1 of 2 pieces | 1 |
| The weight table never contradicts Flux's staged apply order | 172 | 4 of 4 pieces | 0 |
| The apply verdict refuses terminating, foreign, other-instance and adopt | 514+227 | 1 of 6 pieces | 5 |
| The ownership verdicts take no input that only lifts a refusal | 222 | 3 of 3 pieces | 0 |
| Pull requests show breaking changes to the public API | 29 | 4 of 4 pieces | 0 |
| The base tag decides between warning and failing | 64 | 3 of 3 pieces | 0 |
| The check runs read-only with a committed tool checksum | 70 | 4 of 4 pieces | 0 |

### Scenario names that two requirements share

**Context**: The old `kubernetes-tier` spec uses some scenario names in two requirements (for example "The live object is not modified", "The encoding is the one defined", "Reasons are the contract's literals", "A foreign object is skipped"). The byte-equality command above reads a name from the whole file, so it reports a false `DIFF` for these names.
**Decision**: The unit also runs the same comparison per old requirement: for each old requirement, each of its scenarios must appear in the delta with identical lines. That per-requirement run prints no `DIFF` and no `LOST`. The shared names stay as they are: a scenario name is unique inside a requirement, and the delta moves each copy with its own requirement.
**Rationale**: A rename would break the byte check for a kept scenario. Reversibility: high.

### Batch library-B2: the form choices and the limits of the verbatim rule

**Context**: B2 holds `single-build-render` (19 long requirements) and `schema-dispatch` (9). Eleven bodies run past 1400 characters, and several sentences run past 500 on their own.
**Decision**: 5 requirements use form b because another requirement or a code comment cites the header ("Local replacements are honoured only when the caller opts in", "Trait posture governs unhandled traits", "The render module's own gate agrees with the kernel's refusal", "A caller may skip unprovided provider-fulfilled demands", "Path inventory exposed as package-level vars"). 23 use c2. Every new body is built from whole sentences of the old body, copied unchanged. A sentence over 500 characters on its own is cut at a list boundary (the lead is repeated, the last "," or ";" of the list becomes "."). The words added are only these: a subject or condition copied from the same old requirement (the clause tables name each), the keyword SHALL where a sentence has none (the strength table), the lead "With the switch on:" copied from the old list, and a pointer to a scenario where a rule table moved into THEN lines (the Path inventory readers and the retired matcher paths). Two citations of a renamed requirement are updated in place: "DefaultSchemaModule constant" and "OCILoader is the only public Loader". Seven code comments cite a removed requirement name, four of them wrapped over two lines ("Unresolved demands are diagnosed with alternatives" in `opm/errors/match_test.go` and `opm/kernel/render_definedby_test.go`, "Matching runs inside the build with verdicts as data" in `opm/kernel/render_decode_internal_test.go`, "A render reports every contract its instance requires" in `opm/kernel/render_demand_test.go`, "DefaultSchemaModule constant" in `opm/schema/loader_test.go`, "A render writes no staging file" in `opm/kernel/render_no_staging_test.go:20-21` and "A pinned schema release is known without a load" in `opm/schema/loader_test.go:211-212`); the last unit updates them. Those four cited headers use c2 although they are cited: form b would keep the scenarios of every other rule in the parent and add one restatement scenario per split piece, and the comment update is a known, listed task. Reversibility: high.
**Rationale**: A header that others cite stays (form b); where the old scenarios belong to different rules they move with their rule (form c2) and need no new scenario.

Source lines of B2 (the whole line is repeated where it fits; a title drops whole, never paraphrased):

| Old requirement | Old Source line (characters) | Handling |
| --- | --- | --- |
| The single-provider guard runs inside the build | 430 | Three titles. `Source: enhancement 0010 and enhancement 0015.` on two blocks; `Source: enhancement 0015.` on the block with the longest body; the two titles of enhancement 0015 and the core change on the block "Provenance is the registry key, so two majors of one catalog are two providers"; the title of enhancement 0010 and the core change on "Keys with default fulfilment may be supplied by any number of transformers". Every title stays in one block. |
| A caller may skip unprovided provider-fulfilled demands | 178 | The whole line on all 12 blocks, with its clause "as amended by core changes ...". Where the whole line did not fit, the body was split further (the parenthetical of the parent moved to "An unprovided demand is one no enabled registry entry provides"; "Every other refusal stands with the switch on" became three blocks). |
| A render refuses a platform whose enabled entries share contract keys | 257 | The whole line on 9 of 11 blocks. Two blocks ("The glue emits no collision row when the platform carries no collision report" and "The collision cause comes first because a colliding key distorts the inventory") carry the first title with its parenthetical, whole, and drop the title "core change `fold-colliding-contract-keys`", which stays on the other blocks. No clause inside a title is dropped. |
| A render reports every contract its instance requires | 114 | The whole line on the block that holds the first paragraph, because the old text ties it to that paragraph. |

### Batch library-B3: the form choices and the limits of the verbatim rule

**Context**: B3 holds `kernel-runtime` (17 long requirements) and `artifact-types` (11). Nine bodies run past 1500 characters, and four sentences run past 500 on their own.
**Decision**: 4 requirements use form b because another file cites the header: "Goroutine Safety Contract" (ADR-002), "Each runtime contract has one home" (ADR-013), "An overlay source can be written to a directory" (a comment in `opm/module/source_test.go`) and "Instance verbs refuse an unset required config value" (two requirements of `kernel-runtime`, one of them a scenario that stays byte for byte). 24 use c2. No other removed header is cited outside `openspec/` (grep over the repository, archive left out), so B3 adds no stale code-comment name to the seven listed above. Every new body is built from whole sentences of the old body, copied unchanged. A sentence over 500 characters on its own is cut at a list boundary and its lead is repeated (the register option, `AcquireInstanceFromDir`, `AcquireModuleFromDir`, the single registry client, the values sources of `AcquireInstanceFromDir`, the failed synthesis build, the committed-file citation rule and the `SynthesizeInstance` refusal parity sentence). The words added are only these: a keyword (the strength table), a subject or referent copied from the same old requirement (the clause tables name each), and a pointer to another requirement of the same spec by its exact header, where a body used "this", "that build" or "these checks" and the old content cannot fit. The Source lines are repeated whole in every block, with their clauses ("as revised on 2026-10-03"; both clauses of the refusal line). A body that did not fit beside its whole line was split further at a list or clause boundary (the cancellation list and the cancelled-check sentence, the deletion-state conditions). Reversibility: high.
**Rationale**: The 500-character limit, the rule that a citation stays in the body of the block that carries the rule, and the rule that no word is dropped leave no other choice.

### Batch library-B4, first half: the form choices and the limits of the verbatim rule

**Context**: The first half of B4 holds `deps-cascade` (8 long requirements) and `instance-synthesis` (7). The other three specs of B4 (`cascade-wiring`, `config-validation`, `workflow-hardening`) follow in the second half, with its own ledger.
**Decision**: 2 requirements use form b because another file cites the header: "The docs examples follow the loader's core" (a sentence of "The cascade task never touches release or steering files" names it, and that sentence keeps the citation in its new block) and "Instance synthesis input" (a comment in `opm/kernel/synth_test.go`). 13 use c2. No other removed header is cited outside `openspec/` (grep over the repository, archive left out). Every new body is built from whole sentences of the old body, copied unchanged. The words added are only these: a keyword (the strength table), a subject copied from the same old requirement ("The task" or `task -x deps:cascade` for "It", `InstanceInput` for "It"), the lead sentence of the old body repeated where a later sentence needs its subject or its referent, the closing ";" of a list bullet turned into ".", and a pointer to another requirement of the same spec by its exact header where the old content cannot fit beside the whole Source line (the target and the modules in scope of the catalog requirement, the files and literals of the docs examples requirement, the fields of "Instance synthesis input", the overlay of the staged tree requirement, the synthetic-root check). The Source lines are repeated whole in every block. A body that did not fit beside its whole line was split further at a sentence boundary (the core pin and the docs examples).
**Rationale**: The 500-character limit, the rule that a citation stays in the body of the block that carries the rule, and the rule that no word is dropped leave no other choice. Reversibility: high.

One more code comment cites a requirement whose rule moved: `opm/kernel/synth_test.go:168` names "Instance synthesis input" for "InstanceInput carries no schema cache", which is now "InstanceInput carries no schema cache and no cue.Context". The last unit updates it with the comments listed above.

### Batch library-B4, second half: the form choices and the limits of the verbatim rule

**Context**: The second half of B4 holds `cascade-wiring` (6 long requirements), `config-validation` (6) and `workflow-hardening` (1): 13 in all, which completes the 28 of B4.
**Decision**: 2 requirements use form b because another text cites the header: "Single Kernel Validation Primitive" (a comment in `opm/kernel/kernel_test.go`) and "File-Backed Sources Resolve Imports Through the Kernel Mapping" (a sentence of the primitive's old body names it, and that sentence keeps the citation in its new block). In both, the parent keeps all its scenarios and a shorter body; the cut sentences go to ADDED blocks. 11 use c2. No other removed header is cited outside `openspec/` (grep over the repository, archive left out). Every new body is built from whole sentences of the old body, copied unchanged. The words added are only these: a keyword (the strength table), a subject copied from the same old requirement (a file path, a type name or a key name for "It", "Its" or "The key"; `publish` in "The `publish` job's `if:`"), and a pointer "(Requirement: <exact header>)" to another requirement of the same spec where a sentence needs a referent that the old requirement gave in a sibling sentence ("this", "these references", "the option", "the rule", "So", "the marker"). One long sentence is cut at a list boundary with subject and verb repeated (the wiring-check script sentence that checks, runs offline and runs in the `Go tests` job). The citation of each split block is the whole old `Source:` line as a `**Source**:` metadata line (the human's decision of 2026-10-20); the rule text stays in the body. Em-dashes of the old bodies became `,` or `:`.
**Rationale**: The 500-character limit, the rule that a citation stays with the rule it supports, and the rule that no word is dropped leave no other choice. Reversibility: high.

### Batch library-B5, first half: the form choices and the limits of the verbatim rule

**Context**: The first half of B5 holds `fetch-error-classification` (5 long requirements), `platform-artifact` (5) and `platform-module-generation` (4): 14 in all. The other four specs of B5 (`registry-module-loading`, `render-parity`, `catalog-acquisition`, `fixture-pin-maintenance`) follow in the second half, with its own ledger. "Platform Type Shape" runs to 4656 characters, and its inventory sentence runs to about 2400 on its own.
**Decision**: 6 requirements use form b because another text cites the header or only one sentence has to leave: "A fetch or resolution failure is a typed error" (a sentence of the token-endpoint requirement names it for the forbidden tag lookup, and that sentence keeps the citation in its new block, with a pointer to the block that now holds the tag-lookup rule), "The library classifies every fetch and resolution failure it returns" (the unchanged requirement "The library returns author-defect resolution failures typed" names it), "Platform constructor takes a bare value" (a sentence of "A platform records its core floor and contract inventory at construction" names it), "A platform records its core floor and contract inventory at construction" (the `single-build-render` delta names it twice, for `CoreFloor()` and for the decode on a platform the constructor did not build, and so did "Platform Type Shape"; the new blocks that hold those rules carry a pointer, and the two `single-build-render` lines are left to the audit of the whole change because an earlier unit wrote them), "Platform carries its render source" (519 characters: the last sentence leaves) and "Registry access is caller-configured" (590: the sentence on the process environment leaves). In each, the parent keeps all its scenarios and a shorter body, and the cut sentences go to ADDED blocks with a restatement scenario. 8 use c2. No removed header is cited outside `openspec/` (grep over the repository, archive left out) and none is cited inside a kept text except as listed, so no code comment goes stale in this unit. Every new body is built from whole sentences of the old body, copied unchanged. The words added are only these: a keyword (the strength table below), a subject copied from the same old requirement (`Classify` or the constructor for "It", `Contracts()` accessor for "The accessor", the verb `AcquirePlatformFromDir(ctx, dir)` for "the verb", "the fetch forms" for "They"), a pointer "(Requirement: <exact header>)" to the requirement of the same spec that holds the referent ("This", "It", "this requirement", "that form", "the recorded floor", "these facts", "step 1 below", "step 2 below", "The result", "A missing"), and the list numbers "1." and "2." of the step list. Sentences over 500 characters on their own are cut at a list boundary with the lead repeated and the closing "," or "and" turned into ".": the inventory sentence of "Platform Type Shape" (the lead up to "`ContractInventory`", then one block per field or pair of fields), the stamped fields of `AcquirePlatformFromDir` (`Overlay` and `Root` in one block, `Pkg` in the other), the constructor's decode list (`Metadata` and `type` stay in the parent, `Package` and `Source` go to the new block) and the `platform.cue` sentence (embed, declare and import in one block, the `#registry` entry in the other, "`platform.cue` SHALL" repeated). The Source line of the three specs that have one ("Source: enhancement 0021 (the GA exit criteria).") is repeated whole as a `**Source**:` line in every block of the old requirement that carried it (every block of "Classify recognises ...", including the ones from its second paragraph). "Platform Type Shape" carries its citations inside two sentences, as a parenthesis: the four titles of enhancement 0015 after "`#Platform.#contracts`" and the two after "all are reports". A body cannot hold the lead and its first parenthesis together (the lead plus the parenthesis is 760 characters), so that parenthesis moved whole, word for word, to the `**Source**:` line of every block that repeats its lead (the human's decision of 2026-10-20 allows a citation there and nothing else). The second parenthesis fits, and stays in its body. The two parentheses that cite enhancement 0019 in `platform-module-generation` ("Generation is pure and deterministic", "The dependency closure is derived from published module files") hold rule text ("tripwire: it unifies with core's readout of the imported catalog"; "tidying happens once, at generation"), so they stay in their bodies. Reversibility: high.
**Rationale**: The 500-character limit, the rule that a citation stays with the rule it supports, and the rule that no word is dropped leave no other choice.

### Batch library-B5, second half: the form choices and the limits of the verbatim rule

**Context**: The second half of B5 holds `registry-module-loading` (4 long requirements), `render-parity` (4), `catalog-acquisition` (3) and `fixture-pin-maintenance` (3): 14 in all, which completes the 28 of B5. "Registry acquisition accepts bare and v-prefixed versions" in `registry-module-loading` measures 492 characters and is not long, so it is not rewritten. "Acquire a module from the registry" runs to 1608 characters, and one sentence of it runs to 731 on its own.
**Decision**: 3 requirements use form b: "The shipped catalog stays discriminated" (a comment in `opm/kernel/parity_harness_test.go` names the header), "Pure-CUE unification is the render oracle" (the last sentence leaves) and "Catalog acquisition from a registry and from a directory" (the last two sentences leave). In each, the parent keeps all its scenarios and a shorter body, and the cut sentences go to ADDED blocks with a restatement scenario; the scenario "A comparable pair in a catalog release fails the harness" stays in the parent, because a live scenario cannot move to an ADDED block. 11 use c2. No other removed header is cited outside `openspec/` (grep over the repository, archive left out). The comment at `opm/kernel/parity_harness_test.go:115` cites the kept header "The shipped catalog stays discriminated" for rules that now sit in the ADDED blocks "The discrimination read lives in a test function of its own beside the render cases" and "The discrimination assertion is gated as the shipped group is"; the last unit updates it with the code comments listed above. Every new body is built from whole sentences of the old body, copied unchanged. The words added are only these: a keyword (the strength table below), a subject copied from the same old requirement ("The registry module loader" for "It"; `AcquireCatalogFromRegistry` and `AcquireCatalogFromDir` for "Neither"; "The shipped parity group" repeated at the cut; "the update task" for "the task"), and a pointer "(Requirement: <exact header>)" to the requirement of the same spec that holds the referent ("The verification", "the set", "both sources", "The result", "The staged source", "The registry mapping", "that entry point", "The proof", "the default", "the held resolution", "that pin", "The assertion", "a listed location", "A listed file", "the same typed error", "either value", "the inventory", "The module file", "A dependency", "The two paths"). Sentences over 500 characters on their own are cut at a list boundary with the lead repeated and the closing "," or ";" turned into ".": the entry-point sentence of "Acquire a module from the registry" (one block per item of its list, "is" becomes "SHALL be") and the sentence of "The shipped catalog stays discriminated" (cut at ", and SHALL assert", the subject repeated). The citation parenthesis of the first sentence of "The catalog's committed dependency requirements are readable" (two enhancement titles) moved whole, word for word, to a `**Source**:` line, because the sentence with it measures 573 characters; the rule text stays in the body. Two descriptive sentences keep no keyword, because their block carries SHALL: "That module is the one the oracle builds in." and "A listed file can still read the derived version in other places, for example to find the pin it replaces with an old one." The first commit of this unit gave the two `single-build-render` lines that cite "A platform records its core floor and contract inventory at construction" for rules now in other blocks exact-header pointers into `platform-artifact`: the check of the core floor cites "CoreFloor returns nil when the recorded floor is met and otherwise the typed PlatformCoreTooOldError", and the line on the recorded floor cites that header in its body; its second clause, the decode on a platform the constructor did not build, moved to a block of its own with a pointer to "A platform the constructor did not build decodes Package once, on its first CoreFloor or Contracts call" (a pointer is rule text and does not sit in a `**Source**:` line). Reversibility: high.
**Rationale**: The 500-character limit, the rule that a citation stays with the rule it supports, and the rule that no word is dropped leave no other choice.

### Batch library-B6: the form choices and the limits of the verbatim rule

**Context**: B6 holds `migration-docs` (3 long requirements), `release-pipeline` (3), `duplicate-object-identities` (2), `helper-packages` (2) and `test-fixture-registry` (2): 12 in all, which completes the 152 of the change. After it the only WARNING left on `validate --all --strict` is the one in the open change `publish-go-api-bundle`.
**Decision**: 2 requirements use form b because a file outside `openspec/` cites the header: "Helper Boundary at opm/helper/" (a comment in `.golangci.yml`) and "The opmodel.dev namespace resolves through the shared tier" (a comment in `opm/internal/schematest/schematest_test.go`). In each, the parent keeps all its scenarios and a shorter body, and the cut sentences go to ADDED blocks with a restatement scenario. The parent of the helper boundary keeps the import-graph rule and the lint gate rule; the sentence on the `opm/helper/` subdirectory, the tiers and the six exported packages leave, so the last unit updates the `.golangci.yml` comment to cite the block that now holds the opt-in sentence. 10 use c2. Every new body is built from whole sentences of the old body, copied unchanged. The words added are only these: a keyword (the strength table below), a subject copied from the same old requirement ("The release-pin check" for "The check", "The `notify-downstream` job" for "The job" and "it", "The release-please job" for "The job", `secrets.CASCADE_APP_PRIVATE_KEY` before "the key"), and a pointer "(Requirement: <exact header>)" to the requirement of the same spec that holds the referent ("this policy", "These mechanisms", "kernel-tier package", "Rows", "The same check", "that private cache", "The private cache", "the shared tier", "this error", "the message", "the identity", "The one current subpackage", "in this convention", "The exported identity type", "return no rows"). The list sentence of "Release PRs refuse dev and replaced pins" measures 500 characters with its old subject and 512 with "The release-pin check", so it is cut at a list boundary (items 1 to 3 and items 4 and 5) with the lead repeated and the closing ";" of item 3 turned into ".". The Source lines are repeated whole as `**Source**:` lines in every block of the old requirement that carried them. The em-dash of the escape-hatch trailer in "Enforcement design is recorded and armed at GA" became a comma, so the trailer reads `Migration: none, <reason>` in the new block; `migrations/README.md` and ADR-004 keep their own wording. Reversibility: high.
**Rationale**: The 500-character limit, the rule that a citation stays with the rule it supports, and the rule that no word is dropped leave no other choice.

## Security

Assets: none new; the change edits spec text only. No trust boundary, dependency or workflow changes, so no threat model applies. No sensitive detail is in this record.

## Risks / Trade-offs

- **A rewrite silently drops a rule or a scenario.** The ledger, the byte-equality command and the gate make it visible; the reviewer repeats all three per batch. Trade-off: extra tables in this file, which grows by up to about 150 ledger entries.
- **A split spreads one rule over several requirements.** One behavior per requirement is the point of the limit, but a reader now follows several headers. The rule above (a split piece must stand alone) limits it.
- **Six sections in one change.** Each ends green and leaves `main` releasable, because a spec edit touches no code. If the human wants a smaller cut, the batches can ship as sequential changes.
- **An open change edits a requirement that c2 removes.** Today only `publish-go-api-bundle` is open and it edits no existing requirement. The gate fails on any "Archive would refuse" line, and each unit lists `openspec/changes/` first.

## Ledger

One block per rewritten requirement: form, old and new body lengths (characters, by the rule of the guide), the scenario ledger, the new scenarios and the clause table. "verbatim" means the old sentence appears word for word in the new body. Lengths of the new bodies are listed in the order of the replacing requirements.

### Ledger: kubernetes-tier

**The Kubernetes tier lives at opm/k8s beside the kernel**: form c2 (REMOVED plus ADDED); old body 779 characters; new bodies 476, 477, 221.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| The placement is recorded before any package exists | The Kubernetes tier is part of the library's Go module, same name | no, byte for byte (checked) |
| The tier is not a nested module | The Kubernetes tier is part of the library's Go module, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text or holds a moved table, and adds no case):

- "The tier sits beside the kernel and the helper tier" in "The Kubernetes tier lives under opm/k8s outside the kernel and the helper tier"
- "A kernel or helper package does not decide for the tier" in "No kernel or helper package implements a decision of the Kubernetes tier"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The library SHALL place the Kubernetes decisions OPM makes under `opm/k8s/`, a tier outside the... | The Kubernetes tier lives under opm/k8s outside the kernel and the helper tier | verbatim |
| The tier holds inventory entry construction, stale-set computation, digests, the prune and owne... | The Kubernetes tier lives under opm/k8s outside the kernel and the helper tier | verbatim |
| No package under `opm/kernel/` or `opm/helper/` SHALL implement any of these decisions once the... | "No kernel or helper package implements a decision of the Kubernetes tier" | "these decisions" is restated as the list of the first requirement, in parentheses; the rest is verbatim |
| The tier SHALL be part of the library's Go module, `github.com/open-platform-model/library`, an... | The Kubernetes tier is part of the library's Go module | verbatim |
| ADR-011 records the placement. | The Kubernetes tier is part of the library's Go module | verbatim |
| Source: enhancement 0012 (the fenced Kubernetes tier). | The Kubernetes tier lives under opm/k8s outside the kernel and the helper tier; No kernel or helper package im... | verbatim |

**The Kubernetes tier imports no cluster client or controller framework**: form b (MODIFIED parent plus ADDED); old body 1240 characters; new bodies 388, 216, 418, 438, 359.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Lint refuses a cluster client in the tier | The Kubernetes tier imports no cluster client or controller framework, same name | no, byte for byte (checked) |
| Lint refuses a Kubernetes module other than apimachinery in the tier | The Kubernetes tier imports no cluster client or controller framework, same name | no, byte for byte (checked) |
| Lint refuses a Kubernetes module in the helper tier | The Kubernetes tier imports no cluster client or controller framework, same name | no, byte for byte (checked) |
| Lint refuses a helper or internal import in the tier | The Kubernetes tier imports no cluster client or controller framework, same name | no, byte for byte (checked) |
| Lint refuses an unlisted third-party import in the tier | The Kubernetes tier imports no cluster client or controller framework, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text or holds a moved table, and adds no case):

- "The tier imports only apimachinery of the Kubernetes modules" in "The Kubernetes tier imports no internal or helper package and one Kubernetes module"
- "Lint refuses every other Kubernetes module" in "The lint gate enforces the import denials of the Kubernetes tier"
- "Lint refuses a sibling CUE module in the tier" in "Non-test files of the Kubernetes tier import from a strict allow list"
- "The parity test file is the one exception" in "One test file of the Kubernetes tier imports the deprecated objectset"

| Old clause | New requirement | Change |
| --- | --- | --- |
| Beyond the standard library and the CUE SDK that the kernel's output types carry, a file under... | The Kubernetes tier imports no cluster client or controller framework | verbatim |
| No file under `opm/k8s/` SHALL import `k8s.io/client-go`, `sigs.k8s.io/controller-runtime`, any... | parent (cluster client and controller framework) and "The Kubernetes tier imports no internal or helper package and one Kubernetes module" | one sentence cut at "any cluster client"; the words are unchanged |
| The repository lint gate SHALL enforce these denials, including the refusal of every other `k8s... | "The lint gate enforces the import denials of the Kubernetes tier" | "these denials" is restated as the list of denials, in parentheses |
| For non-test files the gate SHALL also enforce the allowed set as a strict allow list: the Go s... | "Non-test files of the Kubernetes tier import from a strict allow list" | "the gate" becomes "the repository lint gate" and the scope "under `opm/k8s/`" is named; the rest is verbatim |
| The one exception is the test file `opm/k8s/object/objectset_parity_test.go`, which imports the... | "One test file of the Kubernetes tier imports the deprecated objectset" | "is" becomes "SHALL be", the keyword an ADDED block needs; the denial it is an exception to is named |
| Source: enhancement 0012 (the fenced Kubernetes tier). | The Kubernetes tier imports no cluster client or controller framework; The Kubernetes tier imports no internal... | verbatim |

**Nothing in the library performs a cluster action**: form c2 (REMOVED plus ADDED); old body 1838 characters; new bodies 483, 248, 318, 436, 232, 346.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| No executor loop or cluster backend ships anywhere in the library | The library performs no cluster action and ships no cluster executor, same name | no, byte for byte (checked) |
| ADR-008 allows only non-cluster executor backends | ADR-008 allows only executor backends that perform no cluster action, same name | no, byte for byte (checked) |
| Apply engines stay with the frontends | The library owns the apply verdict and the order and never an apply engine, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text or holds a moved table, and adds no case):

- "A cluster-acting Op is performed by the frontend" in "The frontend performs the cluster-acting Ops of enhancement 0009"
- "The caller performs each deletion action" in "The frontend performs each deletion action and the tier owns the deletion sequence"
- "An engine sorts within a stage" in "Each frontend submits objects in the library order"

| Old clause | New requirement | Change |
| --- | --- | --- |
| Nothing in the library SHALL perform a cluster action, and the Kubernetes tier SHALL perform no... | The library performs no cluster action and ships no cluster executor | verbatim |
| No executor (a loop that drives a plan to completion) and no executor backend that performs a p... | The library performs no cluster action and ships no cluster executor | verbatim |
| ADR-008 rule 3's allowance for opt-in executor backends under `opm/helper/` is narrowed to exec... | "ADR-008 allows only executor backends that perform no cluster action" and "The frontend performs the cluster-acting Ops of enhancement 0009" | "is narrowed" becomes "SHALL be narrowed"; the parenthesis is cut in two at the semicolon, the first half (0009's hosts) stays, the amend clause moves with both titles; "are performed" becomes "SHALL be performed" |
| A deletion plan SHALL advance one action per call from a serialisable state the caller owns, an... | The frontend performs each deletion action and the tier owns the deletion sequence | verbatim |
| The tier SHALL own the whole deletion sequence (plan, transition, hold verdict). | The frontend performs each deletion action and the tier owns the deletion sequence | verbatim |
| For apply, it SHALL own the per-object verdict (CanApply) and the order, never an apply engine. | "The library owns the apply verdict and the order and never an apply engine" | "it" becomes "the tier" |
| Each frontend SHALL submit objects in the library's order. | Each frontend submits objects in the library order | verbatim |
| An engine's own staging MAY refine that order, for example by sorting within a stage, and SHALL... | Each frontend submits objects in the library order | verbatim |
| ADR-008 rules 1 to 3 apply to the tier. | The frontend performs each deletion action and the tier owns the deletion sequence | verbatim |
| Source: enhancement 0012 ("The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, b... | see "Source line placement" | the three titles go to the pieces they support: first title to the cluster-action piece and the amend piece, second to the deletion and apply-verdict pieces, third to the order piece |

**Kind-class order is a Kubernetes fact that lives in the tier**: form c2 (REMOVED plus ADDED); old body 1297 characters; new bodies 389, 403, 399, 475.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| ADR-008 carries the clarification | ADR-008 rule 4 excludes module-specific ordering only, same name | no, byte for byte (checked) |
| No module-internal ordering is planned and cross-module order belongs to a Bundle | No module-internal ordering is planned, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text or holds a moved table, and adds no case):

- "The kernel holds no apply order" in "The kind-class apply order lives in the Kubernetes tier"
- "The library adds no cross-module ordering" in "Ordering across modules belongs to a future Bundle"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The library SHALL keep one kind-class apply order (a CustomResourceDefinition before its custom... | The kind-class apply order lives in the Kubernetes tier | verbatim |
| It SHALL live in the Kubernetes tier, and the kernel SHALL hold none. | The kind-class apply order lives in the Kubernetes tier | verbatim |
| No module-internal ordering (`dependsOn` edges, phases within one module) is planned: the libra... | No module-internal ordering is planned | verbatim |
| Lifecycle hooks remain the parked 0009 work (an open question of enhancement 0009), carried as... | ADR-008 rule 4 excludes module-specific ordering only | verbatim |
| In the OPM model, ordering across modules belongs to a future Bundle definition, which uses the... | Ordering across modules belongs to a future Bundle | verbatim |
| A frontend's readiness gating between its own custom resources, such as the operator's ModulePa... | Ordering across modules belongs to a future Bundle | verbatim |
| ADR-008 rule 4's "The kernel derives no ordering of its own" SHALL be read as excluding module-... | ADR-008 rule 4 excludes module-specific ordering only | verbatim |
| Source: enhancement 0012 ("Ordering is kind-class order in the Kubernetes tier; no module-inter... | The kind-class apply order lives in the Kubernetes tier; No module-internal ordering is planned; ADR-008 rule... | verbatim |

**Deletion plans come from the inventory and the live objects**: form b (MODIFIED parent plus ADDED); old body 815 characters; new bodies 419, 278, 499.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| ADR-008 records the deletion-plan source | Deletion plans come from the inventory and the live objects, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text or holds a moved table, and adds no case):

- "A prune builds the plan from the stale set" in "A prune hands its stale set to the deletion plan"
- "ADR-008 records the deferral" in "ADR-008 records the deferral of render planning and transition detection"

| Old clause | New requirement | Change |
| --- | --- | --- |
| An instance's deletion plan (uninstall) SHALL be built from the persisted inventory plus the li... | Deletion plans come from the inventory and the live objects | verbatim |
| Building it SHALL NOT need a render and SHALL NOT read a plan stored at apply time. | Deletion plans come from the inventory and the live objects | verbatim |
| A prune's stale set SHALL be computed by the inventory package and handed to the same plan. | A prune hands its stale set to the deletion plan | verbatim |
| ADR-008 SHALL record this, and SHALL record that planning from a render and lifecycle transitio... | parent ("ADR-008 SHALL record this.") and "ADR-008 records the deferral of render planning and transition detection" | one sentence cut in two; "SHALL record that" becomes "SHALL also record that" |
| This deferral does not touch the deletion step transition, which the tier owns as part of the d... | ADR-008 records the deferral of render planning and transition detection | verbatim |
| Source: ADR-008; ADR-013, decision j5; the deletion protocol is enhancement 0012 ("The library... | Deletion plans come from the inventory and the live objects; A prune hands its stale set to the deletion plan;... | verbatim |

**The label vocabulary names labels and never stamps them**: form c2 (REMOVED plus ADDED); old body 1305 characters; new bodies 459, 347, 484, 381.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Every OPM managed-by value is recognised | A predicate recognises the OPM managed-by values, same name | no, byte for byte (checked) |
| The vocabulary matches the frontends' copies | The labels package declares the label keys and values both frontends use, same name | no, byte for byte (checked) |
| Lint keeps the label package dependency-free | The labels package imports only the standard library, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text or holds a moved table, and adds no case):

- "The package declares these keys and values" in "The labels package declares the label keys and values both frontends use"
- "No function in the labels package sets a label" in "The labels package never stamps labels"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The library SHALL provide `opm/k8s/labels`, which imports nothing outside the Go standard libra... | The labels package imports only the standard library | verbatim |
| It SHALL declare the label keys and values both Kubernetes frontends read or write, with values... | "The labels package declares the label keys and values both frontends use" | the colon list moves to the THEN lines of the new scenario "The package declares these keys and values", values verbatim; the body keeps the SHALL and names the scenario |
| The constant for `opmodel.dev/component` SHALL be documented as the OPM object category, not a... | The labels package declares the label keys and values both frontends use | verbatim |
| It SHALL provide a predicate that reports whether a managed-by value identifies an OPM runtime,... | "A predicate recognises the OPM managed-by values" | "It" becomes `opm/k8s/labels`; the values of the two runtimes and the legacy value are added in parentheses from the list |
| No function in the package SHALL set a label on an object. | "The labels package never stamps labels" | "the package" becomes `opm/k8s/labels` |
| Labels on rendered objects are stamped by the CUE build at render. | The labels package never stamps labels | verbatim |
| The repository lint gate SHALL refuse any non-test import into the package from outside the sta... | The labels package imports only the standard library | verbatim |
| Source: enhancement 0012 ("Labels are stamped by the CUE render; the shared render digest ignor... | The labels package imports only the standard library; A predicate recognises the OPM managed-by values; The la... | verbatim |

**Compiled objects convert to Kubernetes objects through one export**: form c2 (REMOVED plus ADDED); old body 1460 characters; new bodies 292, 271, 235, 470, 292.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| One export feeds every consumer of the object | One export returns each Resource's JSON, object and provenance, same name | no, byte for byte (checked) |
| A failing export names the resource and the step | A failing export names the Resource, its position and the step, same name | no, byte for byte (checked) |
| A value that is not an object fails at the decode | A failing export names the Resource, its position and the step, same name | no, byte for byte (checked) |
| The input survives the export | The export leaves the input Resources alone, same name | no, byte for byte (checked) |
| Accessors are best-effort | A Resource exposes the object's fields and its JSON and unstructured forms, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text or holds a moved table, and adds no case):

- "A nil Compiled yields no Resource" in "A Resource carries a rendered object and its provenance"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The library SHALL provide, in `opm/k8s/object`, a `Resource` that carries a rendered object's C... | A Resource carries a rendered object and its provenance | verbatim |
| A nil `Compiled` SHALL yield no Resource. | A Resource carries a rendered object and its provenance | verbatim |
| `Resource` SHALL expose the object's kind, name, namespace, apiVersion, group-version-kind, lab... | A Resource exposes the object's fields and its JSON and unstructured forms | verbatim |
| It SHALL also expose its JSON form and its `*unstructured.Unstructured` form. | A Resource exposes the object's fields and its JSON and unstructured forms | verbatim |
| The library SHALL provide an export of a list of Resources that exports each Resource's value f... | One export returns each Resource's JSON, object and provenance | verbatim |
| The export SHALL NOT change or release the input Resources. | The export leaves the input Resources alone | verbatim |
| A failure SHALL name the failing Resource, its position, and whether the CUE export or the JSON... | A failing export names the Resource, its position and the step | verbatim |
| A value that exports to JSON that is not an object, `null` included, SHALL fail as a decode fai... | A failing export names the Resource, its position and the step | verbatim |
| A Resource keeps its whole CUE build alive; releasing it after export is the caller's responsib... | The export leaves the input Resources alone | verbatim |
| Source: enhancement 0012 ("The Kubernetes runtime surface homes in the library's Kubernetes tie... | The export leaves the input Resources alone | verbatim |

**One weight table orders apply and delete**: form b (MODIFIED parent plus ADDED); old body 1706 characters; new bodies 319, 354, 270, 438.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Apply order puts definitions before their users | One weight table orders apply and delete, same name | no, byte for byte (checked) |
| Delete order is the reverse and stable | One weight table orders apply and delete, same name | no, byte for byte (checked) |
| An unknown version of a known kind falls back to the kind | One weight table orders apply and delete, same name | no, byte for byte (checked) |
| Webhook configurations come after custom resources | One weight table orders apply and delete, same name | no, byte for byte (checked) |
| Class kinds come right after ClusterRoles | One weight table orders apply and delete, same name | no, byte for byte (checked) |
| Quotas and limits come before the workloads they constrain | One weight table orders apply and delete, same name | no, byte for byte (checked) |
| Kinds Flux does not list weigh the default | One weight table orders apply and delete, same name | no, byte for byte (checked) |
| A definition kind name in another group is not a definition | One weight table orders apply and delete, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text or holds a moved table, and adds no case):

- "An engine applying through Flux only refines the table" in "The weight table agrees with the staged apply order of Flux"
- "Equal weights keep their relative order" in "The library sorts by weight with a stable sort"
- "A custom kind ending in Class falls back to the Class weight" in "A weight falls back to the kind alone, then to the Class weight, then to 1000"
- "The weights of the table" in "One weight table orders apply and delete"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The library SHALL keep one kind-class weight table in `opm/k8s/object`, and it SHALL agree with... | parent and "The weight table agrees with the staged apply order of Flux" | one sentence cut at the comma; "it SHALL agree" becomes "The weight table SHALL agree" |
| A CustomResourceDefinition weighs -100. | scenario "The weights of the table" (THEN line) | the weight table moves to the THEN and AND lines; the article is lower case; the body keeps the SHALL and names the scenario |
| A Namespace weighs 0. | scenario "The weights of the table" (AND line) | as above |
| ClusterRoles weigh 5. | scenario "The weights of the table" | verbatim |
| PriorityClasses, RuntimeClasses, StorageClasses, VolumeSnapshotClasses, IngressClasses, Gateway... | scenario "The weights of the table" | verbatim |
| ClusterRoleBindings weigh 7. | scenario "The weights of the table" | verbatim |
| ResourceQuotas weigh 8. | scenario "The weights of the table" | verbatim |
| ServiceAccounts, Roles and RoleBindings weigh 10. | scenario "The weights of the table" | verbatim |
| Secrets and ConfigMaps weigh 15. | scenario "The weights of the table" | verbatim |
| Services weigh 50. | scenario "The weights of the table" | verbatim |
| LimitRanges weigh 60. | scenario "The weights of the table" | verbatim |
| Deployments and StatefulSets weigh 100. | scenario "The weights of the table" | verbatim |
| CronJobs weigh 105. | scenario "The weights of the table" | verbatim |
| PodDisruptionBudgets weigh 108. | scenario "The weights of the table" | verbatim |
| Validating and mutating webhook configurations weigh 2000. | scenario "The weights of the table" | verbatim |
| Every other kind, among them PersistentVolumes, PersistentVolumeClaims, DaemonSets, ReplicaSets... | scenario "The weights of the table" (AND line) | as above; "Every" is lower case |
| A weight SHALL be looked up by group, version and kind first; then a CustomResourceDefinition o... | A weight is looked up by group, version and kind first; A weight falls back to the kind alone, then to the Class weight, then to 1000 | verbatim; the sentence is cut at the list boundary "; then by kind alone", the lead "A weight SHALL be looked up" is repeated and the second block points to the first by its header |
| The library SHALL provide a stable sort by weight, ascending for apply and descending for delet... | The library sorts by weight with a stable sort | verbatim |
| Source: enhancement 0012 ("Ordering is kind-class order in the Kubernetes tier; no module-inter... | One weight table orders apply and delete; The weight table agrees with the staged apply order of Flux; The lib... | verbatim |

**Apply stages follow the weight table**: form b (MODIFIED parent plus ADDED); old body 1023 characters; new bodies 425, 395, 251.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Stages for a typical module | Apply stages follow the weight table, same name | no, byte for byte (checked) |
| A Namespace kind in another group is not a cluster definition | Apply stages follow the weight table, same name | no, byte for byte (checked) |
| No cluster definitions, no definition stage | Apply stages follow the weight table, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text or holds a moved table, and adds no case):

- "Stages keep input order and are not empty" in "Apply stages keep the stable sort order and are never empty"
- "A frontend submits one stage in one call" in "A Flux frontend may submit a whole set or one stage"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The library SHALL provide a function that returns a sorted copy of an apply set cut into stages... | Apply stages follow the weight table | verbatim |
| The first stage holds the cluster definitions, which are every CustomResourceDefinition of `api... | Apply stages follow the weight table | verbatim |
| It is omitted when there are none. | Apply stages follow the weight table | verbatim |
| After it comes one stage per distinct weight of the remaining objects, in ascending weight. | Apply stages follow the weight table | verbatim |
| Order within a stage SHALL be the stable sort order. | Apply stages keep the stable sort order and are never empty | verbatim |
| The input SHALL NOT be reordered, and no stage SHALL be empty. | Apply stages keep the stable sort order and are never empty | verbatim |
| Because the weight table never contradicts Flux's staged apply order, a frontend whose apply en... | "A Flux frontend may submit a whole set or one stage" | the MAY clause is kept word for word; the SHALL comes from "The weight table SHALL NOT contradict Flux's staged apply order", the old "never contradicts" as a rule (the Flux test requirement states the same rule) |
| Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object... | Apply stages keep the stable sort order and are never empty | verbatim |

**Readiness evaluation is a pure function over fetched objects**: form c2 (REMOVED plus ADDED); old body 3347 characters; new bodies 466, 289, 493, 494, 435, 499, 447, 473, 283, 375, 372.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| The status strings match the cli's output | Readiness statuses and evaluation rules are the cli's, same name | no, byte for byte (checked) |
| A Deployment mid-rollout is not ready | A Deployment is ready only when its rollout is complete, same name | no, byte for byte (checked) |
| A Deployment past its progress deadline is not ready | A Deployment is ready only when its rollout is complete, same name | no, byte for byte (checked) |
| A stalled Deployment is told apart from one rolling out | ProgressDeadlineExceeded tells a stalled rollout from one in progress, same name | no, byte for byte (checked) |
| A partitioned StatefulSet is ready at its partition | A StatefulSet is ready by its update strategy, same name | no, byte for byte (checked) |
| A pending claim reports its phase | A claim reports its phase and any other kind reports its Ready condition, same name | no, byte for byte (checked) |
| A custom resource without a Ready condition counts as applied | A claim reports its phase and any other kind reports its Ready condition, same name | no, byte for byte (checked) |
| Missing and unreadable objects count against the aggregate | The aggregate counts missing and unreadable objects against ready, same name | no, byte for byte (checked) |
| Nothing to aggregate is unknown | The aggregate counts missing and unreadable objects against ready, same name | no, byte for byte (checked) |
| The package performs no cluster I/O | Lint holds the health package to the standard library and apimachinery, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text or holds a moved table, and adds no case):

- "The package reads no cluster" in "Readiness is judged by a pure function over fetched objects"
- "A DaemonSet with all its pods available is ready" in "A DaemonSet and a Job are judged by their counts and conditions"
- "A Job completes before it fails" in "A DaemonSet and a Job are judged by their counts and conditions"
- "A passive kind is applied" in "A CronJob and the passive kinds are applied"
- "Only four statuses are healthy" in "Four statuses are healthy and every other string is unhealthy"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The library SHALL provide `opm/k8s/health`, which judges the readiness of one Kubernetes object... | Readiness is judged by a pure function over fetched objects | verbatim |
| Nothing in the package SHALL read a cluster, wait or poll; the frontend fetches each live objec... | Readiness is judged by a pure function over fetched objects | verbatim |
| A status SHALL be one of the strings `Ready`, `NotReady`, `Complete`, `Unknown`, `Missing`, `Ap... | Readiness statuses and evaluation rules are the cli's | verbatim |
| The evaluation rules SHALL be the cli's at that commit, unchanged: | "Readiness statuses and evaluation rules are the cli's" | the closing colon becomes a full stop, because the bullets became requirements |
| - A Deployment is `Ready` only when `status.observedGeneration` has reached `metadata.generatio... | "A Deployment is ready only when its rollout is complete" | the bullet becomes a sentence; "is" becomes "SHALL be" (or "reports" becomes "SHALL report"), the keyword an ADDED block needs; the condition text is verbatim |
| - A StatefulSet is `Ready` when the generation is observed and the ready replicas equal `spec.r... | "A StatefulSet is ready by its update strategy" | the bullet becomes a sentence; "is" becomes "SHALL be" (or "reports" becomes "SHALL report"), the keyword an ADDED block needs; the condition text is verbatim |
| Otherwise `NotReady`. | A StatefulSet is ready by its update strategy | verbatim |
| - A DaemonSet is `Ready` when the generation is observed and the updated and available counts e... | "A DaemonSet and a Job are judged by their counts and conditions" | the bullet becomes a sentence; "is" becomes "SHALL be" (or "reports" becomes "SHALL report"), the keyword an ADDED block needs; the condition text is verbatim |
| - A Job is `Complete` when, in condition order, a `Complete` condition with status `True` appea... | "A DaemonSet and a Job are judged by their counts and conditions" | the bullet becomes a sentence; "is" becomes "SHALL be" (or "reports" becomes "SHALL report"), the keyword an ADDED block needs; the condition text is verbatim |
| - A CronJob, and every kind of the fixed passive set (ConfigMap, Secret, Service, ServiceAccoun... | "A CronJob and the passive kinds are applied" | the bullet becomes a sentence; "is" becomes "SHALL be" (or "reports" becomes "SHALL report"), the keyword an ADDED block needs; the condition text is verbatim |
| - A PersistentVolumeClaim reports its `status.phase` as the status (`Bound` for a bound claim),... | "A claim reports its phase and any other kind reports its Ready condition" | the bullet becomes a sentence; "is" becomes "SHALL be" (or "reports" becomes "SHALL report"), the keyword an ADDED block needs; the condition text is verbatim |
| - Any other kind is `Ready` when its `Ready` condition is `True`, `NotReady` when that conditio... | "A claim reports its phase and any other kind reports its Ready condition" | the bullet becomes a sentence; "is" becomes "SHALL be" (or "reports" becomes "SHALL report"), the keyword an ADDED block needs; the condition text is verbatim |
| `Ready`, `Applied`, `Complete` and `Bound` SHALL be the healthy statuses, and every other strin... | Four statuses are healthy and every other string is unhealthy | verbatim |
| The aggregate of a list of statuses and a count of tracked objects that have no status (missing... | The aggregate counts missing and unreadable objects against ready | verbatim |
| The package SHALL also report, through `ProgressDeadlineExceeded`, whether a Deployment whose c... | "ProgressDeadlineExceeded tells a stalled rollout from one in progress" | "The package" becomes `opm/k8s/health`, so the block names its subject |
| A strict lint rule SHALL hold the package's non-test files to the Go standard library and `k8s.... | "Lint holds the health package to the standard library and apimachinery" | "the package's non-test files" becomes "the non-test files of `opm/k8s/health`" |
| Source: enhancement 0012 ("The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, b... | Readiness is judged by a pure function over fetched objects; Lint holds the health package to the standard lib... | verbatim |

**The ownership verdicts are pure**: form c2 (REMOVED plus ADDED); old body 769 characters; new bodies 365, 296, 404.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| The live object is not modified | The ownership package performs no I/O and never modifies the live object, same name | no, byte for byte (checked) |
| The package imports no clock, environment or logger | The ownership package performs no I/O and never modifies the live object, same name | no, byte for byte (checked) |
| Reasons are the contract's literals | The ownership verdicts are pure functions of explicit inputs, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text or holds a moved table, and adds no case):

- "An input names the object and carries the live object" in "Ownership inputs name the object and carry the live object"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The library SHALL provide `opm/k8s/ownership` with a per-object delete verdict and a per-object... | The ownership verdicts are pure functions of explicit inputs | verbatim |
| Each input SHALL name the object by group, kind, namespace and name and SHALL carry the live ob... | Ownership inputs name the object and carry the live object | verbatim |
| The package SHALL perform no cluster read or write, read no clock or environment, and log nothi... | The ownership package performs no I/O and never modifies the live object | verbatim |
| It SHALL NOT modify the live object it is given. | The ownership package performs no I/O and never modifies the live object | verbatim |
| Each reason SHALL be a string equal to the contract literal for it. | The ownership verdicts are pure functions of explicit inputs | verbatim |
| A frontend reads the live object with its own client. | Ownership inputs name the object and carry the live object | verbatim |
| Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object... | The ownership verdicts are pure functions of explicit inputs; The ownership package performs no I/O and never... | verbatim |

**The delete verdict skips with a reason or proceeds with the judged object's identity**: form c2 (REMOVED plus ADDED); old body 2071 characters; new bodies 375, 433, 373, 414, 235.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A protected kind is skipped without a live read | The delete verdict decides in a fixed order, same name | no, byte for byte (checked) |
| A gone object is skipped as already absent | The delete verdict decides in a fixed order, same name | no, byte for byte (checked) |
| A foreign object is skipped | The delete verdict decides in a fixed order, same name | no, byte for byte (checked) |
| Another instance's object is skipped | The delete verdict decides in a fixed order, same name | no, byte for byte (checked) |
| An empty UUID on either side passes | Empty UUIDs and blank annotations pass the delete verdict, same name | no, byte for byte (checked) |
| Proceed carries the precondition for the judged object | A proceed verdict carries the judged object's identity, same name | no, byte for byte (checked) |
| A skip verdict yields no precondition | A skip verdict yields no precondition and carries a message, same name | no, byte for byte (checked) |
| A live object without a UID yields no precondition | A skip verdict yields no precondition and carries a message, same name | no, byte for byte (checked) |
| An object being deleted proceeds | An annotation naming this instance and a deleting object pass the delete verdict, same name | no, byte for byte (checked) |
| An object annotated for another instance is left in place | A prune or an instance deletion leaves an object handed to another instance, same name | no, byte for byte (checked) |
| An annotation naming this instance changes nothing on delete | An annotation naming this instance and a deleting object pass the delete verdict, same name | no, byte for byte (checked) |
| An empty instance UUID skips an annotated object | Empty UUIDs and blank annotations pass the delete verdict, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text or holds a moved table, and adds no case):

- "The delete verdict order" in "The delete verdict decides in a fixed order"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The delete verdict SHALL decide in this order and stop at the first match: skip as `safety-excl... | "The delete verdict decides in a fixed order" | the five skips and the proceed are listed in the THEN and AND lines of the new scenario "The delete verdict order", conditions verbatim; the body keeps the SHALL, the first-match rule and the six outcomes, and names the scenario |
| An empty UUID on either side SHALL pass the owner comparison. | Empty UUIDs and blank annotations pass the delete verdict | verbatim |
| A blank adopt annotation SHALL pass the adoption comparison, and with an empty instance UUID an... | Empty UUIDs and blank annotations pass the delete verdict | verbatim |
| An adopt annotation naming this instance SHALL change nothing. | An annotation naming this instance and a deleting object pass the delete verdict | verbatim |
| An object being deleted SHALL NOT be skipped for that reason. | An annotation naming this instance and a deleting object pass the delete verdict | verbatim |
| A proceed verdict SHALL carry the UID and the resourceVersion of the live object it judged. | A proceed verdict carries the judged object's identity | verbatim |
| Its DELETE precondition SHALL name that UID and SHALL NOT name a resourceVersion. | A proceed verdict carries the judged object's identity | verbatim |
| A caller MAY add the carried resourceVersion itself. | A proceed verdict carries the judged object's identity | verbatim |
| A skip verdict, and a proceed verdict whose judged UID is empty, SHALL yield no precondition at... | A skip verdict yields no precondition and carries a message | verbatim |
| A skip verdict SHALL carry a message naming the object and the reason. | A skip verdict yields no precondition and carries a message | verbatim |
| So a prune or an instance deletion never deletes an object the user is handing to another insta... | "A prune or an instance deletion leaves an object handed to another instance" | "never deletes" becomes "SHALL NOT delete" |
| Source: enhancement 0012 ("The Kubernetes runtime surface homes in the library's Kubernetes tie... | see "Source line placement" | first title to the proceed-identity piece, second to the order piece, third with the number to the handed-over piece |

**The adopt annotation is the only override and the refusal names it**: form c2 (REMOVED plus ADDED); old body 1177 characters; new bodies 500, 347, 403, 421, 338.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Adoption lifts a foreign refusal | An adopt annotation naming the instance lifts an ownership refusal, same name | no, byte for byte (checked) |
| Adoption lifts an other-instance refusal | An adopt annotation naming the instance lifts an ownership refusal, same name | no, byte for byte (checked) |
| An annotation naming another instance lifts nothing | An adopt annotation with any other value lifts nothing, same name | no, byte for byte (checked) |
| Surrounding whitespace in the annotation value is ignored | An adopt annotation naming the instance lifts an ownership refusal, same name | no, byte for byte (checked) |
| A blank annotation counts as none | An adopt annotation with any other value lifts nothing, same name | no, byte for byte (checked) |
| The other-instance remedy moves the object only after its owner lets go | The other-instance refusal asks to remove the object from its owner first, same name | no, byte for byte (checked) |
| The refusal tells the user how to adopt | A refusal names the adopt annotation key and the UUID to set, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text or holds a moved table, and adds no case):

- "A refusal message names no other remedy and no enhancement" in "No refusal message names another way past the refusal"

| Old clause | New requirement | Change |
| --- | --- | --- |
| An adopt annotation whose value equals the instance UUID SHALL lift the `foreign-object`, `othe... | An adopt annotation naming the instance lifts an ownership refusal | verbatim |
| The annotation value SHALL be compared with surrounding whitespace trimmed, and a value that is... | An adopt annotation naming the instance lifts an ownership refusal | verbatim |
| An adopt annotation with any other value SHALL lift nothing, and the refusal message SHALL say... | An adopt annotation with any other value lifts nothing | the subject restates the old condition: "with any other value" becomes "whose value does not equal the instance UUID" (the plain negation of the first sentence of the old body); the rest is verbatim |
| The refusal message for `foreign-object` and `other-instance` SHALL name the object, the annota... | A refusal names the adopt annotation key and the UUID to set | verbatim |
| The `other-instance` message SHALL ask the user to remove the object from the instance that own... | The other-instance refusal asks to remove the object from its owner first | verbatim |
| No message SHALL name any other way past a refusal, such as a command-line flag. | No refusal message names another way past the refusal | verbatim |
| No message SHALL carry an enhancement reference. | No refusal message names another way past the refusal | verbatim |
| Source: enhancement 0012 ("The apply guard runs on every apply, refusing foreign objects outsid... | An adopt annotation naming the instance lifts an ownership refusal; An adopt annotation with any other value l... | verbatim |

**Inventory entries are plain values built from objects**: form c2 (REMOVED plus ADDED); old body 898 characters; new bodies 243, 494, 472.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| An entry reads the object's identity and component | The entry constructor reads identity from the object and component from its label, same name | no, byte for byte (checked) |
| A core-group, cluster-scoped object without the label | The entry constructor reads identity from the object and component from its label, same name | no, byte for byte (checked) |
| The entry type carries no tags | The entry type carries no struct tags, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text or holds a moved table, and adds no case):

- "The entry has six fields and an object constructor" in "An inventory entry has exactly six fields and a constructor from an object"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The library SHALL provide, in `opm/k8s/inventory`, an inventory entry with exactly the fields g... | An inventory entry has exactly six fields and a constructor from an object | verbatim |
| The constructor SHALL read group, version and kind, namespace and name from the object, and the... | The entry constructor reads identity from the object and component from its label | verbatim |
| The entry type SHALL carry no struct tags, so that no serialisation the library chooses can bec... | The entry type carries no struct tags | verbatim |
| Each frontend maps entries to its own CRD or record fields. | The entry type carries no struct tags | verbatim |
| Source: enhancement 0012 ("The Kubernetes runtime surface homes in the library's Kubernetes tie... | The entry constructor reads identity from the object and component from its label; The entry type carries no s... | verbatim |

**The stale set is component-blind**: form c2 (REMOVED plus ADDED); old body 864 characters; new bodies 300, 432, 480.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A component rename is not stale | The stale set compares group, kind, namespace and name only, same name | no, byte for byte (checked) |
| An API version change is not stale | The stale set compares group, kind, namespace and name only, same name | no, byte for byte (checked) |
| Removed objects are stale in previous order | The stale set keeps the previous order and is never nil, same name | no, byte for byte (checked) |
| The package exports no other comparison | The package exports one identity comparison and no component-aware one, same name | no, byte for byte (checked) |
| Nothing stale is an empty list, not nil | The stale set keeps the previous order and is never nil, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| The library SHALL provide, in `opm/k8s/inventory`, one stale-set function. | The stale set compares group, kind, namespace and name only | verbatim |
| It SHALL return every entry of the previous inventory for which no entry of the current invento... | The stale set compares group, kind, namespace and name only | verbatim |
| The result SHALL keep the previous inventory's order, SHALL be a non-nil empty list when nothin... | The stale set keeps the previous order and is never nil | verbatim |
| The same identity comparison SHALL be exported on its own. | "The package exports one identity comparison and no component-aware one" | "The same identity comparison" becomes "The identity comparison of the stale set", and "in the package" becomes `opm/k8s/inventory`, so the block names its subject |
| No component-aware comparison and no separate component-rename filter SHALL exist in the packag... | The package exports one identity comparison and no component-aware one | verbatim |
| Source: enhancement 0012 ("The stale set is component-blind, and the inventory digest hashes a... | The stale set keeps the previous order and is never nil; The package exports one identity comparison and no co... | verbatim |

**The inventory digest hashes a canonical field encoding**: form c2 (REMOVED plus ADDED); old body 1155 characters; new bodies 497, 435, 354.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| The encoding is the one defined | The inventory digest is sha256 over a tagged, length-prefixed encoding, same name | no, byte for byte (checked) |
| Input order does not matter | The inventory digest depends only on the entries' field values, same name | no, byte for byte (checked) |
| Every field counts | The inventory digest depends only on the entries' field values, same name | no, byte for byte (checked) |
| Empty and nil inventories agree | The empty inventory hashes the tag line and a new encoding uses a new tag line, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| The library SHALL provide, in `opm/k8s/inventory`, one inventory digest of the form `sha256:` f... | The inventory digest is sha256 over a tagged, length-prefixed encoding | verbatim |
| It SHALL hash the tag line `opm-inventory-v1` with a newline, followed by every entry in order... | The inventory digest is sha256 over a tagged, length-prefixed encoding | verbatim |
| Each entry SHALL be written as its group, kind, namespace, name, version and component, in that... | The inventory digest is sha256 over a tagged, length-prefixed encoding | verbatim |
| The digest SHALL depend only on the entries' field values: not on their input order and not on... | The inventory digest depends only on the entries' field values | verbatim |
| Two inventories that differ in their entries, or in any field of an entry, SHALL produce differ... | The inventory digest depends only on the entries' field values | verbatim |
| The empty inventory SHALL hash the tag line alone. | The empty inventory hashes the tag line and a new encoding uses a new tag line | "The empty inventory" becomes "The inventory digest of the empty inventory", and the tag line `opm-inventory-v1` is named (both from the old body); the rest is verbatim |
| The function SHALL leave its input unchanged. | The empty inventory hashes the tag line and a new encoding uses a new tag line | "The function" becomes "The inventory digest function"; the rest is verbatim |
| A change to this encoding SHALL use a new tag line. | The empty inventory hashes the tag line and a new encoding uses a new tag line | "this encoding" becomes "the encoding of the inventory digest"; the rest is verbatim |
| Source: enhancement 0012 ("The stale set is component-blind, and the inventory digest hashes a... | The empty inventory hashes the tag line and a new encoding uses a new tag line | the first title stays here; the second title ("The Kubernetes runtime surface homes ...") moved whole to the block "The inventory digest depends only on the entries' field values", so the line fits 500 |

**The render digest ignores only the managed-by label value**: form c2 (REMOVED plus ADDED); old body 2177 characters; new bodies 415, 421, 275, 426, 221, 389, 274, 264.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| The two runtimes digest one render equally | The cli and the operator compute the same render digest for one render, same name | no, byte for byte (checked) |
| Any other label counts | Any other difference changes the render digest, same name | no, byte for byte (checked) |
| Adding or removing the managed-by label counts | Any other difference changes the render digest, same name | no, byte for byte (checked) |
| Large integers are not rounded | The render digest blanks the managed-by value and sorts object keys, same name | no, byte for byte (checked) |
| The encoding is the one defined | The render digest orders the encoded objects and hashes them after a tag line, same name | no, byte for byte (checked) |
| The input is not changed | The render digest leaves its input unchanged and fails on a malformed object with its posi..., same name | no, byte for byte (checked) |
| A malformed object fails with its position | The render digest leaves its input unchanged and fails on a malformed object with its posi..., same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text or holds a moved table, and adds no case):

- "The render digest reads no CUE value" in "The render digest reads only exported JSON"
- "Strings are escaped as defined" in "The render digest encodes strings as Go's JSON encoder does with HTML escaping off"
- "Sets that differ only in the managed-by value digest equally" in "Sets that differ only in the managed-by value have the same render digest"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The library SHALL provide, in `opm/k8s/inventory`, one render digest over the objects that `opm... | The render digest reads only exported JSON | verbatim |
| It SHALL read only each object's exported JSON, never a CUE value, so a caller may release its... | The render digest reads only exported JSON | verbatim |
| For each object it SHALL decode the JSON keeping every number's literal, replace the value of t... | "The render digest blanks the managed-by value and sorts object keys" | "it" becomes "the render digest" |
| Strings SHALL be encoded as Go's `encoding/json` Encoder writes them with HTML escaping off: `"... | "The render digest encodes strings as Go's JSON encoder does with HTML escaping off" | the colon list moves to the THEN and AND lines of the new scenario "Strings are escaped as defined", verbatim; the body keeps the SHALL and names the scenario |
| It SHALL hash the tag line `opm-render-v1` with a newline, followed by the encoded objects in o... | "The render digest orders the encoded objects and hashes them after a tag line" | "It" becomes "The render digest" |
| Two exported object sets that differ only in the value of the managed-by label SHALL produce th... | Sets that differ only in the managed-by value have the same render digest | verbatim |
| The cli (`opm-cli`) and the operator (`opm-controller`) therefore compute the same digest for o... | "The cli and the operator compute the same render digest for one render" | "therefore compute" becomes "SHALL compute"; "therefore" goes because the sentence stands alone |
| Two object sets that differ in any other label, in whether the managed-by label is present, or... | Any other difference changes the render digest | verbatim |
| The empty set SHALL hash the tag line alone. | The render digest orders the encoded objects and hashes them after a tag line | verbatim |
| An object whose JSON does not decode to a single JSON object SHALL fail with an error naming it... | The render digest leaves its input unchanged and fails on a malformed object with its position | verbatim |
| The function SHALL leave its input unchanged. | The render digest leaves its input unchanged and fails on a malformed object with its position | verbatim |
| A change to this encoding SHALL use a new tag line. | The render digest orders the encoded objects and hashes them after a tag line | verbatim |
| Source: enhancement 0012 (labels are stamped by the CUE render and the shared digest ignores th... | The render digest reads only exported JSON; The render digest blanks the managed-by value and sorts object key... | verbatim |

**The deletion plan orders inventory entries by kind-class delete order**: form c2 (REMOVED plus ADDED); old body 1249 characters; new bodies 308, 330, 493, 372, 500.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Entries are ordered for deletion | The deletion plan orders its entries by delete weight, same name | no, byte for byte (checked) |
| Equal weights keep their order | The deletion plan orders its entries by delete weight, same name | no, byte for byte (checked) |
| Protected kinds are marked up front | Protected kinds are marked safety-excluded in the plan before any live read, same name | no, byte for byte (checked) |
| A prune hands its stale set to the same plan | A deletion plan needs no render and no stored plan, same name | no, byte for byte (checked) |
| Changing the listed steps does not change the plan | The plan constructor is the only way to build a plan with steps, same name | no, byte for byte (checked) |

New scenario (restates the first sentence of the old body): "A deletion plan is built from entries, a policy and a UUID" in "The library provides a deletion plan built from inventory entries, a policy and a UUID".

| Old clause | New requirement | Change |
| --- | --- | --- |
| The library SHALL provide `opm/k8s/lifecycle` with a deletion plan built from a list of invento... | The library provides a deletion plan built from inventory entries, a policy and a UUID | verbatim |
| The plan SHALL order the entries by the tier's weight table in descending order, keeping the re... | The deletion plan orders its entries by delete weight | verbatim; "The plan" gets a pointer to the requirement that holds the first sentence |
| Every CustomResourceDefinition and Namespace entry, as the tier's safety exclusion matches them... | Protected kinds are marked safety-excluded in the plan before any live read | verbatim |
| Building the plan SHALL NOT need a render or a stored plan: an uninstall builds it from the per... | A deletion plan needs no render and no stored plan | verbatim |
| The constructor SHALL be the only way to build a plan with steps, so no frontend can hand the t... | The plan constructor is the only way to build a plan with steps | verbatim |
| Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object... | Protected kinds are marked safety-excluded in the plan before any live read (whole line); the pieces that carried `Source: enhancement 0012; ADR-008 (Deletion plans).` now carry one title of it and the ADR clause (the first sentence was split from the ordering sentence to make room), the last with both titles | verbatim, whole titles only; no clause inside a title dropped |

**The deletion transition names one action per call**: form c2 (REMOVED plus ADDED); old body 1561 characters; new bodies 428, 348, 273, 471, 288, 338.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A protected kind is skipped without a read | A safety-excluded step is skipped without a read, same name | no, byte for byte (checked) |
| An owned object is read, then deleted with Foreground propagation | Every other step is read, judged, and then skipped or deleted, same name | no, byte for byte (checked) |
| A foreign object is skipped with the verdict's reason | Every other step is read, judged, and then skipped or deleted, same name | no, byte for byte (checked) |
| Another instance's object is skipped | Every other step is read, judged, and then skipped or deleted, same name | no, byte for byte (checked) |
| A plan that does not prune names done at once | A plan that does not prune names done at once, same name | no, byte for byte (checked) |
| A prune built with the zero policy deletes nothing | A plan that does not prune names done at once, same name | no, byte for byte (checked) |
| The transition names only reads, deletions, skips and done | The deletion transition returns the next state and one action, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "A call returns the next state and one action" in "The deletion transition returns the next state and one action"
- "The transition performs no action and names no other kind of action" in "The deletion transition performs no action and names only steps of the plan"

| A prune of a dropped object leaves it for the adopting instance | A prune leaves an object that another instance adopted, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| The deletion transition SHALL take the plan, the caller's state and the outcome of the action i... | The deletion transition returns the next state and one action | verbatim |
| It SHALL perform no action itself. | The deletion transition performs no action and names only steps of the plan | "It" becomes "The deletion transition", the subject of the first sentence of the old body; the rest is verbatim |
| A safety-excluded step SHALL be reported as skipped with the reason `safety-excluded` without a... | A safety-excluded step is skipped without a read | verbatim |
| Every other step SHALL be read before it is judged, and the library's delete verdict SHALL judg... | Every other step is read, judged, and then skipped or deleted | "Every other step" becomes "Every step that is not safety-excluded" (the negation of the sentence before it in the old body); the rest is verbatim |
| A skip verdict SHALL be reported as skipped with the verdict's reason and message. | Every other step is read, judged, and then skipped or deleted | verbatim |
| A proceed verdict SHALL name a delete with Foreground propagation and the verdict's UID precond... | Every other step is read, judged, and then skipped or deleted | verbatim |
| When the policy does not prune, the first call SHALL name done and no step SHALL be read or del... | A plan that does not prune names done at once | verbatim |
| Every read, delete and skip action SHALL name a step of the plan, and the transition SHALL name... | The deletion transition performs no action and names only steps of the plan | verbatim |
| So a prune of an object the instance dropped because another instance adopted it leaves the obj... | "A prune leaves an object that another instance adopted" | "So a prune ... leaves" becomes "A prune ... SHALL leave" |
| Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object... | see "Source line placement" | titles split: second to the non-pruning piece, first to the judging piece, third with the number to the adopted-object piece; the ADR clause to the first two pieces. The pieces "The deletion transition returns the next state and one action", "The deletion transition performs no action and names only steps of the plan" and "A safety-excluded step is skipped without a read" carry the first title, `(enhancements#103)` and the ADR clause, all verbatim (the whole line does not fit) |

**Action outcomes are classified by the library**: form c2 (REMOVED plus ADDED); old body 1361 characters; new bodies 469, 474, 494, 274, 366, 233.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A gone object is already absent | A NotFound read or delete is already absent, same name | no, byte for byte (checked) |
| An object gone before its delete is already absent | A NotFound read or delete is already absent, same name | no, byte for byte (checked) |
| A wrapped Forbidden error is classified | The library classifies the raw error of each read or delete, same name | no, byte for byte (checked) |
| A failed precondition is a conflict | A successful delete is recorded as deleted and any other error as failed, same name | no, byte for byte (checked) |
| A failure does not stop the plan | A failure does not stop the plan, same name | no, byte for byte (checked) |
| A live object for another step is not deleted | A mismatched live object is never judged or deleted, same name | no, byte for byte (checked) |
| An empty live namespace or kind is not a mismatch | A mismatched live object is never judged or deleted, same name | no, byte for byte (checked) |
| One call can finish two steps | Each call appends the outcomes of the steps it finished, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| The caller SHALL hand back the raw error of each read or delete, and the library SHALL classify... | The library classifies the raw error of each read or delete | verbatim |
| A NotFound read, or a read that returns no object, SHALL count as already absent, and SHALL be... | A NotFound read or delete is already absent | verbatim |
| A NotFound delete SHALL be skipped as `already-absent`. | A NotFound read or delete is already absent | verbatim |
| A successful delete SHALL record the step as deleted. | A successful delete is recorded as deleted and any other error as failed | verbatim |
| Any other error SHALL record the step as failed, with the failure class `forbidden`, `conflict`... | "A successful delete is recorded as deleted and any other error as failed" | "Any other error" becomes "Any error that is not a NotFound", the same set after the two NotFound cases |
| A live object handed back for a read that names a different object than the step, in any identi... | A mismatched live object is never judged or deleted | verbatim |
| A failure SHALL NOT stop the plan: the next step's action SHALL follow. | A failure does not stop the plan | verbatim |
| Each call SHALL append, in step order, the outcomes of the steps it finished and only those, an... | Each call appends the outcomes of the steps it finished | verbatim |
| Source: enhancement 0012 ("The Kubernetes runtime surface homes in the library's Kubernetes tie... | The library classifies the raw error of each read or delete; A NotFound read or delete is already absent; A su... | verbatim |

**The deletion state is a serialisable value the caller owns**: form c2 (REMOVED plus ADDED); old body 1065 characters; new bodies 389, 344, 452, 349, 235.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A round-tripped state advances identically | A state written to JSON advances identically, same name | no, byte for byte (checked) |
| Advancing twice names the same action | A state written to JSON advances identically, same name | no, byte for byte (checked) |
| A state past the plan is refused | A state that cannot belong to the plan is refused, same name | no, byte for byte (checked) |
| A state awaiting something after the last step is refused | A state that awaits something it cannot await is refused, same name | no, byte for byte (checked) |
| A state whose outcomes do not match its next step is refused | A state that cannot belong to the plan is refused, same name | no, byte for byte (checked) |
| A state with an out-of-order or unknown outcome is refused | A state that cannot belong to the plan is refused, same name | no, byte for byte (checked) |
| The JSON encoding is fixed | The JSON encoding of the deletion state is fixed field by field, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text or holds a moved table, and adds no case):

- "The zero state starts a plan" in "The deletion state is a plain value the caller owns"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The deletion state SHALL be a plain value with a defined JSON encoding, holding the next step,... | The deletion state is a plain value the caller owns | verbatim |
| The zero state SHALL be the start of a plan. | The deletion state is a plain value the caller owns | verbatim |
| The library SHALL keep nothing between calls. | The deletion state is a plain value the caller owns | verbatim |
| A state written to JSON and read back SHALL advance identically to the state held in memory, an... | A state written to JSON advances identically | verbatim |
| The transition SHALL return an error, the input state unchanged and no action when the state ca... | A state that cannot belong to the plan is refused; A state that awaits something it cannot await is refused | the sentence and the title of its Source line do not fit together in 500 characters: cut at the list boundary ", it awaits something after the last step", the lead repeated, the last "," of the first list becomes "."; the conditions of each part are verbatim |
| The JSON encoding SHALL be fixed field by field, so that renaming a field fails a check. | The JSON encoding of the deletion state is fixed field by field | verbatim |
| Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object... | The deletion state is a plain value the caller owns; A state written to JSON advances identically; The JSON en... | verbatim |

**The hold verdict is decided from the policy and the plan's outcome**: form c2 (REMOVED plus ADDED); old body 1519 characters; new bodies 380, 425.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Pruning disabled releases | The hold verdict is decided from the policy, the plan's outcome and the identity, same name | no, byte for byte (checked) |
| An empty inventory releases | The hold verdict is decided from the policy, the plan's outcome and the identity, same name | no, byte for byte (checked) |
| Force-orphan releases only for a missing identity | The hold verdict is decided from the policy, the plan's outcome and the identity, same name | no, byte for byte (checked) |
| A missing identity without force-orphan holds | The hold verdict is decided from the policy, the plan's outcome and the identity, same name | no, byte for byte (checked) |
| An unfinished plan holds | The hold verdict is decided from the policy, the plan's outcome and the identity, same name | no, byte for byte (checked) |
| A state that cannot belong to the plan holds | The hold verdict is decided from the policy, the plan's outcome and the identity, same name | no, byte for byte (checked) |
| A Forbidden failure wins over other failures | The hold verdict is decided from the policy, the plan's outcome and the identity, same name | no, byte for byte (checked) |
| Force-orphan does not lift a failed cleanup | The hold verdict is decided from the policy, the plan's outcome and the identity, same name | no, byte for byte (checked) |
| A finished plan with skips releases | The hold verdict is decided from the policy, the plan's outcome and the identity, same name | no, byte for byte (checked) |
| Reasons are the contract's literals | Each hold verdict carries its reason and a message, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text or holds a moved table, and adds no case):

- "The hold verdict order" in "The hold verdict is decided from the policy, the plan's outcome and the identity"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The library SHALL decide whether an instance's deletion hold may be released from the plan's po... | The hold verdict is decided from the policy, the plan's outcome and the identity | verbatim |
| It SHALL check in this order and stop at the first match. | "The hold verdict is decided from the policy, the plan's outcome and the identity" and scenario "The hold verdict order" | the ordered conditions move to the THEN and AND lines of the new scenario, wording verbatim; the body keeps the SHALL and the first-match rule and names the scenario |
| When the policy does not prune, it SHALL release as `prune-disabled`. | "The hold verdict is decided from the policy, the plan's outcome and the identity" and scenario "The hold verdict order" | the ordered conditions move to the THEN and AND lines of the new scenario, wording verbatim; the body keeps the SHALL and the first-match rule and names the scenario |
| When the plan has no steps, it SHALL release as `inventory-empty`. | "The hold verdict is decided from the policy, the plan's outcome and the identity" and scenario "The hold verdict order" | the ordered conditions move to the THEN and AND lines of the new scenario, wording verbatim; the body keeps the SHALL and the first-match rule and names the scenario |
| When the deleting identity is missing and the policy sets force-orphan, it SHALL release as `fo... | "The hold verdict is decided from the policy, the plan's outcome and the identity" and scenario "The hold verdict order" | the ordered conditions move to the THEN and AND lines of the new scenario, wording verbatim; the body keeps the SHALL and the first-match rule and names the scenario |
| When the identity is missing or could not be obtained, it SHALL hold as `identity-unavailable`. | "The hold verdict is decided from the policy, the plan's outcome and the identity" and scenario "The hold verdict order" | the ordered conditions move to the THEN and AND lines of the new scenario, wording verbatim; the body keeps the SHALL and the first-match rule and names the scenario |
| When the state cannot belong to the plan, for any reason the transition refuses it, it SHALL ho... | "The hold verdict is decided from the policy, the plan's outcome and the identity" and scenario "The hold verdict order" | the ordered conditions move to the THEN and AND lines of the new scenario, wording verbatim; the body keeps the SHALL and the first-match rule and names the scenario |
| When the plan is not finished, it SHALL hold as `cleanup-incomplete`. | "The hold verdict is decided from the policy, the plan's outcome and the identity" and scenario "The hold verdict order" | the ordered conditions move to the THEN and AND lines of the new scenario, wording verbatim; the body keeps the SHALL and the first-match rule and names the scenario |
| When any step failed as Forbidden, it SHALL hold as `cleanup-forbidden`. | "The hold verdict is decided from the policy, the plan's outcome and the identity" and scenario "The hold verdict order" | the ordered conditions move to the THEN and AND lines of the new scenario, wording verbatim; the body keeps the SHALL and the first-match rule and names the scenario |
| When any other step failed, it SHALL hold as `cleanup-incomplete`. | "The hold verdict is decided from the policy, the plan's outcome and the identity" and scenario "The hold verdict order" | the ordered conditions move to the THEN and AND lines of the new scenario, wording verbatim; the body keeps the SHALL and the first-match rule and names the scenario |
| Otherwise it SHALL release as `cleanup-complete`, with skipped steps counting as complete. | "The hold verdict is decided from the policy, the plan's outcome and the identity" and scenario "The hold verdict order" | the ordered conditions move to the THEN and AND lines of the new scenario, wording verbatim; the body keeps the SHALL and the first-match rule and names the scenario |
| Each verdict SHALL carry the reason as the contract's literal and a message the library words. | Each hold verdict carries its reason and a message | verbatim |
| Source: enhancement 0012 ("The Kubernetes runtime surface homes in the library's Kubernetes tie... | Each hold verdict carries its reason and a message | verbatim |

**The weight table never contradicts Flux's staged apply order**: form b (MODIFIED parent plus ADDED); old body 1260 characters; new bodies 495, 442, 311, 388.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| The shipped table contradicts Flux nowhere | The weight table never contradicts Flux's staged apply order, same name | no, byte for byte (checked) |
| A weight edit that creates a contradiction is caught | The weight table never contradicts Flux's staged apply order, same name | no, byte for byte (checked) |
| Flux's tie-break between unlisted kinds is counted | The weight table never contradicts Flux's staged apply order, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text or holds a moved table, and adds no case):

- "The universe holds copies in groups before and after the built-in groups" in "The test universe of the Flux model covers the table, Flux's kinds and both group orders"
- "A contradicting pair fails the test and is named" in "The Flux model test fails on any contradicting pair and names it"
- "Both orders put a CustomResourceDefinition before a Namespace" in "A second test shows that the comparison reports an injected contradiction"
- "The model of Flux's stages" in "The weight table never contradicts Flux's staged apply order"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The library SHALL keep a test that models the staged apply of Flux's `ssa` package at the versi... | parent and scenario "The model of Flux's stages" | the stage list in parentheses moves to the THEN and AND lines; the pinned version phrase and the rank and tie-break clauses stay in the body |
| The test's universe SHALL hold the library's table, Flux's listed kinds, a copy of each of thos... | The test universe of the Flux model covers the table, Flux's kinds and both group orders | verbatim |
| The test SHALL fail when the library weighs any pair of the universe strictly opposite to the o... | The Flux model test fails on any contradicting pair and names it | verbatim |
| A second test SHALL show that the comparison reports a contradiction injected through a modifie... | A second test shows that the comparison reports an injected contradiction | verbatim |
| Source: enhancement 0012 ("Ordering is kind-class order in the Kubernetes tier; no module-inter... | The weight table never contradicts Flux's staged apply order; The test universe of the Flux model covers the t... | verbatim |

**The apply verdict refuses terminating, foreign, other-instance and adopted-elsewhere objects**: form c2 (REMOVED plus ADDED); old body 2959 characters; new bodies 343, 387, 365, 409, 389, 488.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A new object is applied | The apply verdict decides in a fixed order, same name | no, byte for byte (checked) |
| A terminating object is refused everywhere | Nothing lifts a terminating refusal, same name | no, byte for byte (checked) |
| An inventoried foreign object is applied | The apply verdict decides in a fixed order, same name | no, byte for byte (checked) |
| An inventoried object of this instance is applied | The apply verdict decides in a fixed order, same name | no, byte for byte (checked) |
| An inventoried object annotated for another instance is refused | The adopted-elsewhere message names the object, the adopter and the way back, same name | no, byte for byte (checked) |
| An inventoried object labelled for another instance without an annotation is applied | An instance whose UUID changed keeps applying its own inventoried objects, same name | no, byte for byte (checked) |
| An inventoried object the adopter has taken is refused | The apply verdict decides in a fixed order, same name | no, byte for byte (checked) |
| An annotation naming this instance takes an inventoried object back | An instance whose UUID changed keeps applying its own inventoried objects, same name | no, byte for byte (checked) |
| An empty instance UUID applies an inventoried object | The apply verdict decides in a fixed order, same name | no, byte for byte (checked) |
| A foreign object outside the inventory is refused | The apply verdict decides in a fixed order, same name | no, byte for byte (checked) |
| Another instance's object outside the inventory is refused | The apply verdict decides in a fixed order, same name | no, byte for byte (checked) |
| A handed-over object stays adopted-elsewhere after the adopter applies | The adopted-elsewhere message names the object, the adopter and the way back, same name | no, byte for byte (checked) |
| An annotation naming another instance than the label keeps other-instance | The apply verdict decides in a fixed order, same name | no, byte for byte (checked) |
| A dropped object is not taken back on the next apply | The adopted-elsewhere message names the object, the adopter and the way back, same name | no, byte for byte (checked) |
| An OPM object without a UUID label is applied | The apply verdict decides in a fixed order, same name | no, byte for byte (checked) |
| An OPM object of this instance outside the inventory is applied | The apply verdict decides in a fixed order, same name | no, byte for byte (checked) |
| An empty instance UUID fails closed outside the inventory | Outside the inventory an empty instance UUID fails closed, same name | no, byte for byte (checked) |
| An empty instance UUID refuses an annotated object outside the inventory | Outside the inventory an empty instance UUID fails closed, same name | no, byte for byte (checked) |
| The docs state the frontend's part of the hand-over | The docs state the frontend's part of the hand-over, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text or holds a moved table, and adds no case):

- "The apply verdict order" in "The apply verdict decides in a fixed order"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The apply verdict SHALL decide in this order and stop at the first match. | "The apply verdict decides in a fixed order" and scenario "The apply verdict order"; terminating also in "Nothing lifts a terminating refusal"; UUID change in "... | the ordered conditions move to the THEN and AND lines of the new scenario, wording verbatim; the body keeps the SHALL, the first-match rule and the outcomes, and names the scenario |
| When there is no live object, it SHALL apply. | "The apply verdict decides in a fixed order" and scenario "The apply verdict order"; terminating also in "Nothing lifts a terminating refusal"; UUID change in "... | the ordered conditions move to the THEN and AND lines of the new scenario, wording verbatim; the body keeps the SHALL, the first-match rule and the outcomes, and names the scenario |
| When the live object has a deletion timestamp, it SHALL refuse as `terminating`, whether or not... | "The apply verdict decides in a fixed order" and scenario "The apply verdict order"; terminating also in "Nothing lifts a terminating refusal"; UUID change in "... | the ordered conditions move to the THEN and AND lines of the new scenario, wording verbatim; the body keeps the SHALL, the first-match rule and the outcomes, and names the scenario |
| When the live object's adopt annotation equals the non-empty instance UUID, it SHALL apply. | "The apply verdict decides in a fixed order" and scenario "The apply verdict order"; terminating also in "Nothing lifts a terminating refusal"; UUID change in "... | the ordered conditions move to the THEN and AND lines of the new scenario, wording verbatim; the body keeps the SHALL, the first-match rule and the outcomes, and names the scenario |
| When the object is in the instance's recorded inventory: with an empty instance UUID it SHALL a... | "The apply verdict decides in a fixed order" and scenario "The apply verdict order"; terminating also in "Nothing lifts a terminating refusal"; UUID change in "... | the ordered conditions move to the THEN and AND lines of the new scenario, wording verbatim; the body keeps the SHALL, the first-match rule and the outcomes, and names the scenario |
| Outside the inventory: when the live managed-by label is not an OPM runtime's value, it SHALL r... | "The apply verdict decides in a fixed order" and scenario "The apply verdict order"; terminating also in "Nothing lifts a terminating refusal"; UUID change in "... | the ordered conditions move to the THEN and AND lines of the new scenario, wording verbatim; the body keeps the SHALL, the first-match rule and the outcomes, and names the scenario |
| Outside the inventory an empty instance UUID SHALL never match an adopt annotation, and any non... | Outside the inventory an empty instance UUID fails closed | verbatim |
| Source: enhancement 0012 ("The apply guard runs on every apply, refusing foreign objects outsid... | see "Source line placement" | the first-paragraph Source (514 characters) is split by title; the second-paragraph Source is kept as `Source: enhancement 0012 (enhancements#103).` on both blocks that hold the second paragraph (the title is dropped whole, the issue reference stays) |
| The `ApplyInput.InInventory` doc and the package doc SHALL state that a frontend drops an objec... | The docs state the frontend's part of the hand-over | verbatim |
| The `adopted-elsewhere` message SHALL name the object and the instance its adopt annotation nam... | The adopted-elsewhere message names the object, the adopter and the way back | verbatim |
| For an inventoried object it SHALL say that this instance no longer applies the object and drop... | The adopted-elsewhere message names the object, the adopter and the way back | verbatim |
| Source: enhancement 0012 ("The apply guard runs on every apply, refusing foreign objects outsid... | Outside the inventory an empty instance UUID fails closed | verbatim |

**The ownership verdicts take no input that only lifts a refusal**: form b (MODIFIED parent plus ADDED); old body 965 characters; new bodies 448, 486, 429.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| The verdict inputs carry no admission field | The ownership verdicts take no input that only lifts a refusal, same name | no, byte for byte (checked) |
| An earlier install manifest's object is refused on apply | The ownership verdicts take no input that only lifts a refusal, same name | no, byte for byte (checked) |
| An earlier install manifest's Deployment is not deleted | The ownership verdicts take no input that only lifts a refusal, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text or holds a moved table, and adds no case):

- "The caller supplies two facts and the operator installer is judged as any instance" in "The caller supplies the inventory fact and the instance UUID"
- "Nothing else lifts an ownership refusal" in "Only an adopt annotation outside the inventory lifts an ownership refusal"

| Old clause | New requirement | Change |
| --- | --- | --- |
| `ownership.ApplyInput` SHALL have exactly the fields `Object`, `Live`, `InInventory` and `Insta... | parent, naming the unchanged scenario "The verdict inputs carry no admission field" | the field lists stay in that scenario, byte for byte; the body keeps the SHALL |
| Neither SHALL carry a field whose only effect is to lift a refusal or a skip. | The ownership verdicts take no input that only lifts a refusal | verbatim |
| The caller still supplies two facts the library does not check, as the frontend's own ownership... | The caller supplies the inventory fact and the instance UUID | verbatim |
| Outside the inventory, the adopt annotation on the live object naming the applying instance SHA... | Only an adopt annotation outside the inventory lifts an ownership refusal | verbatim |
| On delete, nothing SHALL lift the `not-opm-managed` skip. | Only an adopt annotation outside the inventory lifts an ownership refusal | verbatim |
| A frontend that installs the operator is judged as any other instance. | "The caller supplies the inventory fact and the instance UUID" | "is judged" becomes "SHALL be judged" |
| Source: enhancement 0012 ("The apply guard runs on every apply, refusing foreign objects outsid... | The ownership verdicts take no input that only lifts a refusal; The caller supplies the inventory fact and the... | verbatim |

### Ledger: api-diff-check

**Pull requests show breaking changes to the public API**: form c2 (REMOVED plus ADDED); old body 820 characters; new bodies 313, 188, 204, 239.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A pull request removes an exported function | Pull requests compare the exported API against the base release tag, same name | no, byte for byte (checked) |
| A pull request changes only internal packages | Pull requests compare the exported API against the base release tag, same name | no, byte for byte (checked) |
| An older release tag exists | The base release tag is the nearest reachable v tag, same name | no, byte for byte (checked) |
| A release-cascade pull request moves core | A move of the core pin constant is allowed, same name | no, byte for byte (checked) |
| A pull request removes the core pin constant | A move of the core pin constant is allowed, same name | no, byte for byte (checked) |
| A break merged earlier since the tag | Inherited incompatible changes do not count as a change of the pull request, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| A pull request that changes Go code or the module files SHALL run a check that compares the exp... | Pull requests compare the exported API against the base release tag | verbatim |
| The base release tag SHALL be the nearest tag matching `v[0-9]*` reachable from the pull reques... | The base release tag is the nearest reachable v tag | verbatim |
| An incompatible change that the base commit already carries relative to that tag SHALL be liste... | "Inherited incompatible changes do not count as a change of the pull request" | "that tag" becomes "the base release tag" |
| A value change of `schema.DefaultSchemaModule`, which the release cascade makes on every core m... | "A move of the core pin constant is allowed" | "SHALL NOT count either" becomes "SHALL NOT count as a change of the pull request" |
| Source: ADR-013, decision j4. | Pull requests compare the exported API against the base release tag; The base release tag is the nearest reach... | verbatim |

**The base tag decides between warning and failing**: form b (MODIFIED parent plus ADDED); old body 754 characters; new bodies 404, 318, 167.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A breaking change on the beta line | The base tag decides between warning and failing, same name | no, byte for byte (checked) |
| A breaking change after GA | The base tag decides between warning and failing, same name | no, byte for byte (checked) |
| A compatible pull request | The base tag decides between warning and failing, same name | no, byte for byte (checked) |
| No release tag is reachable | The base tag decides between warning and failing, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text or holds a moved table, and adds no case):

- "Each mode states its remedy" in "The check states the remedy of its mode"
- "A tool build error fails the job" in "A failure of the check itself fails the job in both modes"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The check SHALL derive its outcome from the base release tag alone: when the tag has a prerelea... | The base tag decides between warning and failing | verbatim |
| In both modes it SHALL name the base tag and each change. | The base tag decides between warning and failing | verbatim |
| In warn mode it SHALL state that a breaking change needs a `feat!` commit with a `BREAKING CHAN... | "The check states the remedy of its mode" | "In warn mode it" becomes "In warn mode the check"; the rest is verbatim |
| No repository variable or input SHALL change the mode in CI. | The base tag decides between warning and failing | verbatim |
| A failure of the check itself (a tool build error, no reachable tag) SHALL fail the job in both... | A failure of the check itself fails the job in both modes | verbatim |
| Source: ADR-013, decision j4; ADR-010; ADR-004; CONSTITUTION VI. | The base tag decides between warning and failing; The check states the remedy of its mode; A failure of the ch... | verbatim |

**The check runs read-only with a committed tool checksum**: form b (MODIFIED parent plus ADDED); old body 626 characters; new bodies 392, 130, 133, 182.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A same-repo pull request runs the check | The check runs read-only with a committed tool checksum, same name | no, byte for byte (checked) |
| A tampered tool module | The check runs read-only with a committed tool checksum, same name | no, byte for byte (checked) |
| A maintainer checks a release-tag base locally | The check runs read-only with a committed tool checksum, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text or holds a moved table, and adds no case):

- "The library module does not require the tool" in "The library's go.mod does not require the diff tool"
- "The check is not a required status check" in "The check is not a required status check by this change"
- "CI ignores the BASE override" in "The check runs locally through task api:diff"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The check's workflow SHALL declare `permissions: contents: read`, SHALL check out with `persist... | The check runs read-only with a committed tool checksum | verbatim |
| The diff tool SHALL be built from a tools module in this repository whose committed `go.sum` th... | parent and "The library's go.mod does not require the diff tool" | one sentence cut at the last comma; "the library's own `go.mod` SHALL NOT require the tool" becomes "...the diff tool" |
| The check SHALL NOT be a required status check by this change. | The check is not a required status check by this change | verbatim |
| The same check SHALL run locally through `task api:diff`, with an optional `BASE` tag override... | The check runs locally through task api:diff | verbatim |
| Source: library#181 (the workflow hardening pass); workflow-hardening. | The check runs read-only with a committed tool checksum; The library's go.mod does not require the diff tool;... | verbatim |

### Ledger: single-build-render

**The render module's dependency list is derived by promotion**: form c2 (REMOVED plus ADDED); old body 1800 characters; new bodies 357, 127, 428, 284, 253, 183, 162.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Platform wins a shared path | The platform's dependency list wins in the render module, same name | no, byte for byte (checked) |
| Instance-only paths survive | The platform's dependency list wins in the render module, same name | no, byte for byte (checked) |
| An input's unqualified self-import resolves | Each input module's own path is listed as a default major, same name | no, byte for byte (checked) |
| A platform replacement of its catalog is honoured | Promoted local replacements follow the precedence of the dependencies, same name | no, byte for byte (checked) |
| An instance replacement on a platform-named path is inert | Promoted local replacements follow the precedence of the dependencies, same name | no, byte for byte (checked) |
| An instance replacement on an instance-only path is honoured | A promoted replacement is written as the input wrote it with a relative target resolved against the input, same name | no, byte for byte (checked) |
| A replace-only dependency is listed with a placeholder | A replaced path that the promoted list does not carry is listed with a placeholder version, same name | no, byte for byte (checked) |
| A version-less dependency without a replacement is refused | A version-less dependency without a promoted replacement is refused before staging, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "Inputs enter through directory replacements" in "The two input trees enter the build through directory replacements"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The generated render module's dependency list SHALL be derived by promotion: the platform modul... | The platform's dependency list wins in the render module | verbatim |
| No tidy-equivalent and no registry consultation SHALL run at render time to compute the list. | The platform's dependency list wins in the render module | verbatim |
| The two input trees SHALL enter the build through build-local directory replacements; neither i... | The two input trees enter the build through directory replacements | verbatim |
| Each input module's own path SHALL be listed in the render module's `cue.mod/module.cue` with a... | Each input module's own path is listed as a default major | verbatim |
| When local replacements are enabled for the render, each input's own `cue.mod/local-module.cue`... | Promoted local replacements follow the precedence of the dependencies | verbatim |
| A promoted replacement SHALL be written into the render module's main-module view exactly as th... | A promoted replacement is written as the input wrote it with a relativ... | verbatim |
| A replaced path the promoted dependency list does not carry SHALL be listed with a placeholder ... | A replaced path that the promoted list does not carry is listed with a... | verbatim |
| An input dependency that carries no version and is covered by no promoted replacement SHALL be ... | A version-less dependency without a promoted replacement is refused be... | verbatim |

**Local replacements are honoured only when the caller opts in**: form b (MODIFIED parent plus ADDED); old body 663 characters; new bodies 369, 335.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Replacement without opt-in refused | Local replacements are honoured only when the caller opts in, same name | no, byte for byte (checked) |
| Inputs without the file are unaffected | Local replacements are honoured only when the caller opts in, same name | no, byte for byte (checked) |
| Replacement rows are data | Local replacements are honoured only when the caller opts in, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "Honoured replacements are rows" in "Honoured local replacements are reported as rows, not as message strings"

| Old clause | New requirement | Change |
| --- | --- | --- |
| `Kernel.Render` SHALL accept a per-render opt-in for local replacements, off by default. When i... | Local replacements are honoured only when the caller opts in | verbatim |
| An input without the file SHALL behave identically under either setting. | Local replacements are honoured only when the caller opts in | verbatim |
| When the per-render opt-in for local replacements is on, | Honoured local replacements are reported as rows, not as message strin... | added words (not in the old body) |
| promoted replacements SHALL be reported on the render diagnostics as rows carrying the replaced... | Honoured local replacements are reported as rows, not as message strin... | verbatim |

**Version skew is detected from the two committed resolutions and the response is caller-configured**: form c2 (REMOVED plus ADDED); old body 859 characters; new bodies 337, 347, 214.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Prerelease builds compare by SemVer precedence | The skew check compares the instance's requirement with the platform's tidied entry by SemVer 2 precedence, same name | no, byte for byte (checked) |
| Newer module warns and renders by default | A newer instance requirement is warned or refused by the caller's policy, same name | no, byte for byte (checked) |
| Refuse policy stops the render | A newer instance requirement is warned or refused by the caller's policy, same name | no, byte for byte (checked) |
| Older module is data, not a warning | An older instance requirement is plain data, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| For each OPM-namespace path, the kernel SHALL compare the instance module's `cue.mod` requireme... | The skew check compares the instance's requirement with the platform's... | verbatim |
| Newer and older SHALL follow SemVer 2 precedence, prerelease builds included (`2.0.0-beta.2` is... | The skew check compares the instance's requirement with the platform's... | verbatim |
| When the instance requires a NEWER build than the platform carries, the configured policy decid... | A newer instance requirement is warned or refused by the caller's poli... | verbatim |
| The kernel SHALL NOT render the skew as a message string; a frontend formats the row. | A newer instance requirement is warned or refused by the caller's poli... | verbatim |
| A module requiring an OLDER build SHALL produce no such mark; the per-path resolved-versions co... | An older instance requirement is plain data | "no such mark" names the mark of the sibling sentence, as the old sentence before it states |

**Matching runs inside the build with verdicts as data**: form c2 (REMOVED plus ADDED); old body 2778 characters; new bodies 389, 497, 422, 350, 264, 333, 414, 165, 496, 156.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A unify refusal names its conflicts once | The glue reports the matched pairs, the unify refusals, the unhandled-trait table and the over-subscription rows as data, same name | no, byte for byte (checked) |
| Verdicts decode beside a failing gate | The glue reports each unresolved demand with its same-base alternatives as data, same name | no, byte for byte (checked) |
| An unmatched component carries its candidates on the diagnostics | The glue reports each unmatched component with its candidates as data, same name | no, byte for byte (checked) |
| An omitted component leaves the pair set | The reported pair set agrees with plain predicate matching, same name | no, byte for byte (checked) |
| Healthy pairs render beside a failing pair | The kernel names the failed pairs from each matched pair's rendered output, same name | no, byte for byte (checked) |
| Failed pairs are reported on a gate refusal | FailedPairs is filled on every RenderError raised after the build, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "A skipped demand is a verdict as data" in "The glue reports each demand skipped under the caller's switch as data"
- "A rung sees only a component's candidates" in "The rungs are evaluated only for a component's candidates"
- "The gate refuses beside readable diagnostics" in "The fail-closed demand gate holds while the diagnostics stay readable"
- "A matched pair is unified once" in "The glue unifies each matched pair's transform once"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The generated glue SHALL express matching over the platform's derived `#composedTransformers` w... | The glue reports the matched pairs, the unify refusals, the unhandled-... | verbatim |
| the matched pair set; | The glue reports the matched pairs, the unify refusals, the unhandled-... | verbatim |
| every candidate the always-unify rung refused, with its conflicting FQNs; | The glue reports the matched pairs, the unify refusals, the unhandled-... | verbatim |
| the unhandled-trait table; the over-subscription rows. | The glue reports the matched pairs, the unify refusals, the unhandled-... | verbatim |
| The generated glue SHALL express matching over the platform's derived `#composedTransformers` w... | The glue reports each unresolved demand with its same-base alternative... | verbatim |
| every unresolved demand with the same-base alternatives the platform implements (sorted in the ... | The glue reports each unresolved demand with its same-base alternative... | the final ";" of the list becomes "." |
| The generated glue SHALL express matching over the platform's derived `#composedTransformers` w... | The glue reports each unmatched component with its candidates as data | verbatim |
| every unmatched component with every candidate the demand walk reached for it and, for a predic... | The glue reports each unmatched component with its candidates as data | the final ";" of the list becomes "." |
| The generated glue SHALL express matching over the platform's derived `#composedTransformers` w... | The glue reports each demand skipped under the caller's switch as data | verbatim |
| every demand skipped under the caller's switch (see "A caller may skip unprovided provider-fulf... | The glue reports each demand skipped under the caller's switch as data | the final ";" of the list becomes "." |
| The always-unify and predicate rungs SHALL be evaluated only for a component's candidates (the ... | The rungs are evaluated only for a component's candidates | verbatim |
| The always-unify rung SHALL be plain unification with no provenance exclusion. | The rungs are evaluated only for a component's candidates | verbatim |
| Matching semantics are those of the render-parity oracle: with the skip switch off, the pair se... | The reported pair set agrees with plain predicate matching | verbatim |
| The fail-closed demand gate SHALL hold: an unresolved demand that was not skipped, or an unmatc... | The fail-closed demand gate holds while the diagnostics stay readable | verbatim |
| Failed pairs are not a glue verdict. | The glue unifies each matched pair's transform once | verbatim |
| The glue SHALL unify each matched pair's transform once, in `rendered`, and SHALL NOT evaluate ... | The glue unifies each matched pair's transform once | verbatim |
| The kernel SHALL name the failed pairs on `RenderDiagnostics.FailedPairs`, in pair order, from ... | The kernel names the failed pairs from each matched pair's rendered ou... | verbatim |
| The kernel SHALL fill `FailedPairs` on every `RenderError` raised after the build, a gate refus... | FailedPairs is filled on every RenderError raised after the build | verbatim |

**Rendered output decodes with provenance and per-pair concreteness**: form c2 (REMOVED plus ADDED); old body 560 characters; new bodies 176, 233, 149.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Provenance on every object | Rendered output decodes with provenance in the build's deterministic order, same name | no, byte for byte (checked) |
| One compiled-output type | Compiled is declared in opm/kernel only, same name | no, byte for byte (checked) |
| Incomplete pair output refuses | The kernel validates per-pair output concreteness itself, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| On a passing gate, `Render` SHALL decode `rendered` into `[]*kernel.Compiled`, each carrying in... | Rendered output decodes with provenance in the build's deterministic o... | verbatim |
| `Compiled` SHALL be declared in `opm/kernel`, beside the verb that produces it, with exactly th... | Compiled is declared in opm/kernel only | verbatim |
| The kernel SHALL validate per-pair output concreteness itself: an incomplete (non-error) pair o... | The kernel validates per-pair output concreteness itself | verbatim |

**Render is the kernel's sole render path**: form c2 (REMOVED plus ADDED); old body 1124 characters; new bodies 308, 366, 467.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Old entry points are gone | Render is the only way the kernel renders an instance against a platform, same name | no, byte for byte (checked) |
| A subscription-shaped platform is refused | A platform module importing its catalogs is the only platform shape the kernel accepts, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "The default pin carries the registry shape" in "The kernel's default core pin carries the registry shape of the platform import"

| Old clause | New requirement | Change |
| --- | --- | --- |
| `Kernel.Render` SHALL be the only way the kernel renders an instance against a platform. | Render is the only way the kernel renders an instance against a platfo... | verbatim |
| The kernel SHALL expose no `Compile`, `Match` or `Materialize` method and no materialized-platf... | Render is the only way the kernel renders an instance against a platfo... | verbatim |
| The kernel's default core schema pin SHALL be a release carrying the enhancement 0019 ("A platf... | The kernel's default core pin carries the registry shape of the platfo... | the citation names the renamed requirement of schema-dispatch |
| A platform module importing its catalogs is the only platform shape the kernel accepts: the pla... | A platform module importing its catalogs is the only platform shape th... | verbatim |
| Core derives the entry's `version` from the embedded catalog's stamped identity, and with no ca... | A platform module importing its catalogs is the only platform shape th... | verbatim |

**The single-provider guard runs inside the build**: form c2 (REMOVED plus ADDED); old body 2056 characters; new bodies 482, 493, 482, 481, 467.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Inventory and render agree on every served platform | The render and the platform's contract inventory read one provider count, same name | no, byte for byte (checked) |
| Providers count when the defining catalog is disabled | The provider count lists the registry keys that require each provider-fulfilled contract, same name | no, byte for byte (checked) |
| Two majors of one provider catalog are two providers | Provenance is the registry key, so two majors of one catalog are two providers, same name | no, byte for byte (checked) |
| Second provider refused in-build | An over-subscribed contract is reported as a row and refuses the render, same name | no, byte for byte (checked) |
| Catalog-fulfilled plurality allowed | Keys with default fulfilment may be supplied by any number of transformers, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| The generated glue SHALL read the platform's provider count from core's derived contract invent... | The render and the platform's contract inventory read one provider cou... | verbatim |
| Because both the render and the platform's `Contracts()` read this one count, a render SHALL re... | The render and the platform's contract inventory read one provider cou... | verbatim |
| Source: enhancement 0010 and enhancement 0015. | The render and the platform's contract inventory read one provider cou... | added words (not in the old body) |
| The count is `#Platform.#contracts.providedBy`: for every contract key declared `fulfilment: "p... | The provider count lists the registry keys that require each provider-... | "The count is" becomes "The provider count SHALL be": the subject is the phrase of the first sentence, and the verb gets the keyword an ADDED block needs |
| Source: enhancement 0015. | The provider count lists the registry keys that require each provider-... | added words (not in the old body) |
| Provenance is the registry key, never a value parsed out of an FQN or read off the transformer,... | Provenance is the registry key, so two majors of one catalog are two p... | the verb "is" becomes "SHALL be", the keyword an ADDED block needs |
| Source: enhancement 0015 (^ | Provenance is the registry key, so two majors of one catalog are two p... | added words (not in the old body) |
| "One provider per contract stands; provider classes are rejected in this entry" | Provenance is the registry key, so two majors of one catalog are two p... | verbatim |
| and | Provenance is the registry key, so two majors of one catalog are two p... | added words (not in the old body) |
| "An unfulfilled contract is reported, never refused; only over-subscription refuses platform-pa... | Provenance is the registry key, so two majors of one catalog are two p... | verbatim |
| ^), | Provenance is the registry key, so two majors of one catalog are two p... | added words (not in the old body) |
| as recounted by core change `count-providers-per-registry-entry`. | Provenance is the registry key, so two majors of one catalog are two p... | verbatim |
| Every key core lists in `#contracts.overSubscribed` SHALL be reported as an over-subscription r... | An over-subscribed contract is reported as a row and refuses the rende... | verbatim |
| The rows SHALL be built by iterating `providedBy` unconditionally, so a platform value lacking ... | An over-subscribed contract is reported as a row and refuses the rende... | verbatim |
| Source: enhancement 0010 and enhancement 0015. | An over-subscribed contract is reported as a row and refuses the rende... | added words (not in the old body) |
| The generated glue SHALL read the platform's provider count from core's derived contract invent... | Keys with default fulfilment may be supplied by any number of transfor... | verbatim |
| Keys with default (`catalog`) fulfilment MAY be supplied by any number of transformers from any... | Keys with default fulfilment may be supplied by any number of transfor... | verbatim |
| Source: enhancement 0010 (^ | Keys with default fulfilment may be supplied by any number of transfor... | added words (not in the old body) |
| "A contract declares where its fulfilment comes from, and a provider-fulfilled contract admits ... | Keys with default fulfilment may be supplied by any number of transfor... | verbatim |
| ^), | Keys with default fulfilment may be supplied by any number of transfor... | added words (not in the old body) |
| as recounted by core change `count-providers-per-registry-entry`. | Keys with default fulfilment may be supplied by any number of transfor... | verbatim |

**Unresolved demands are diagnosed with alternatives**: form c2 (REMOVED plus ADDED); old body 2237 characters; new bodies 371, 451, 470, 437, 196, 207, 436, 299.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Different apiVersion named | An unresolved demand is reported with its component, key, kind and alternatives, same name | no, byte for byte (checked) |
| A defined but unimplemented contract names its catalog | An unresolved-demand row names the enabled catalog that lists the demanded key, same name | no, byte for byte (checked) |
| A disabled catalog defines nothing | An unresolved-demand row names the enabled catalog that lists the demanded key, same name | no, byte for byte (checked) |
| A colliding contract row names its entries | An unresolved-demand row names the colliding entries and each disqualified candidate, same name | no, byte for byte (checked) |
| A catalog-fulfilled row is not marked | An unresolved-demand row marks an unprovided demand on every render, same name | no, byte for byte (checked) |
| An unprovided row says so | An unprovided row's message says that the contract is provider-fulfilled, same name | no, byte for byte (checked) |
| Undemandable resource fails the render | The unresolved-demand row distinguishes four cases, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "An unskipped unresolved demand fails the render" in "Render fails on any unresolved demand that the caller's switch did not skip"
- "Diagnostic fields do not change the refusal" in "The defining catalog and the colliding entries of a row are diagnostic only"

| Old clause | New requirement | Change |
| --- | --- | --- |
| A demanded contract for which the platform holds no candidate, or for which every candidate is ... | An unresolved demand is reported with its component, key, kind and alt... | verbatim |
| the component, the contract key, the kind, | An unresolved demand is reported with its component, key, kind and alt... | verbatim |
| the same-base alternatives the platform does implement (sorted in the build on the apiVersion l... | An unresolved demand is reported with its component, key, kind and alt... | the final "," of the list becomes "." |
| A demanded contract for which the platform holds no candidate, or for which every candidate is ... | An unresolved-demand row names the enabled catalog that lists the dema... | verbatim |
| the registry key of the enabled catalog that lists the demanded key in its contract maps (empty... | An unresolved-demand row names the enabled catalog that lists the dema... | the final "," of the list becomes "." |
| A demanded contract for which the platform holds no candidate, or for which every candidate is ... | An unresolved-demand row names the colliding entries and each disquali... | verbatim |
| the registry keys of the enabled entries defining the key when it is a collision (read from `#c... | An unresolved-demand row names the colliding entries and each disquali... | verbatim |
| and, when candidates existed, each disqualified candidate's transformer with the FQNs it confli... | An unresolved-demand row names the colliding entries and each disquali... | verbatim |
| A demanded contract for which the platform holds no candidate, or for which every candidate is ... | An unresolved-demand row marks an unprovided demand on every render | verbatim |
| whether the demand is unprovided (its contract declares `fulfilment: "provider"` and no enabled... | An unresolved-demand row marks an unprovided demand on every render | the final "," of the list becomes "." |
| The unprovided marker SHALL be computed on every render, whatever the skip switch says. | An unresolved-demand row marks an unprovided demand on every render | verbatim |
| An unprovided row's message SHALL additionally say that the contract is provider-fulfilled and ... | An unprovided row's message says that the contract is provider-fulfill... | verbatim |
| Every resource a component declares is a required demand. | Render fails on any unresolved demand that the caller's switch did not... | verbatim |
| `Render` SHALL fail on any unresolved demand that the caller's switch did not skip, through the... | Render fails on any unresolved demand that the caller's switch did not... | verbatim |
| The row SHALL distinguish four cases: "defined by `<catalog>` and nothing on this platform impl... | The unresolved-demand row distinguishes four cases | "The row" becomes "The unresolved-demand row", the phrase of the first sentence |
| The defining catalog and the colliding entries are diagnostic only (enhancement 0015, "An unful... | The defining catalog and the colliding entries of a row are diagnostic... | names the row they belong to |

**Trait posture governs unhandled traits**: form b (MODIFIED parent plus ADDED); old body 509 characters; new bodies 387, 121.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Optional trait warns | Trait posture governs unhandled traits, same name | no, byte for byte (checked) |
| Load-bearing trait fails | Trait posture governs unhandled traits, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "An unstated posture fails closed" in "An unstated trait posture fails closed as a build error"

| Old clause | New requirement | Change |
| --- | --- | --- |
| An unhandled trait's effect SHALL be governed by its effective `optional` value read from the c... | Trait posture governs unhandled traits | verbatim |
| The kernel SHALL NOT render an unhandled trait as a message string; a frontend formats the tabl... | Trait posture governs unhandled traits | verbatim |
| An unstated posture (non-concrete `optional`) SHALL fail closed as a build error naming the tra... | An unstated trait posture fails closed as a build error | verbatim |

**A render result carries no presentation strings**: form c2 (REMOVED plus ADDED); old body 778 characters; new bodies 439, 338.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Advisory facts are rows | A render result and a render refusal expose verdicts as structured rows only, same name | no, byte for byte (checked) |
| A skipped demand is a row | A render result and a render refusal expose verdicts as structured rows only, same name | no, byte for byte (checked) |
| A gate cause is the diagnostics rows | A typed gate cause carries the same rows as the diagnostics, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| A render result and a render refusal SHALL expose verdicts as structured rows only. | A render result and a render refusal expose verdicts as structured row... | verbatim |
| The kernel SHALL NOT attach human-readable warning or advisory messages to a result; every advi... | A render result and a render refusal expose verdicts as structured row... | verbatim |
| Each row-carrying typed gate cause (colliding contracts, unresolved demands, unmatched componen... | A typed gate cause carries the same rows as the diagnostics | verbatim |
| The not-routable cause carries no rows: it is raised only from the decoded routable diagnostic. | A typed gate cause carries the same rows as the diagnostics | verbatim |

**The render module's own gate agrees with the kernel's refusal**: form b (MODIFIED parent plus ADDED); old body 596 characters; new bodies 390, 205.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Refusal is visible in the module | The render module's own gate agrees with the kernel's refusal, same name | no, byte for byte (checked) |
| Success is visible in the module | The render module's own gate agrees with the kernel's refusal, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "The decoded refusal decides" in "The kernel's decoded refusal is authoritative and never reads the gate field"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The generated render module SHALL carry a `gate` field that evaluates to an error exactly when ... | The render module's own gate agrees with the kernel's refusal | verbatim |
| The kernel's decoded refusal SHALL remain the authoritative one, since it carries the typed cau... | The kernel's decoded refusal is authoritative and never reads the gate... | verbatim |

**A caller may skip unprovided provider-fulfilled demands**: form b (MODIFIED parent plus ADDED); old body 2172 characters; new bodies 469, 306, 388, 340, 427, 491, 296, 411, 381, 463, 318, 319.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Switch off refuses an unprovided trait as before | A caller may skip unprovided provider-fulfilled demands, same name | no, byte for byte (checked) |
| An unprovided trait is skipped and its component renders | A caller may skip unprovided provider-fulfilled demands, same name | no, byte for byte (checked) |
| An unprovided resource omits its component | A caller may skip unprovided provider-fulfilled demands, same name | no, byte for byte (checked) |
| A catalog-fulfilled demand still refuses | A caller may skip unprovided provider-fulfilled demands, same name | no, byte for byte (checked) |
| A provider that exists but does not match still refuses | A caller may skip unprovided provider-fulfilled demands, same name | no, byte for byte (checked) |
| Over-subscription still refuses | A caller may skip unprovided provider-fulfilled demands, same name | no, byte for byte (checked) |
| The module's own gate agrees under the switch | A caller may skip unprovided provider-fulfilled demands, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "The switch off changes nothing" in "With the switch off a render is as without the switch"
- "A skipped trait leaves its pairs" in "A skipped trait demand does not refuse the render"
- "A skipped resource omits the component" in "A skipped resource demand omits its component"
- "A skipped demand row carries the component and the kind" in "Every skipped demand is reported as a skipped-demand row"
- "The flag is set on trait rows too" in "The omission flag is set on every skipped row of an omitted component"
- "An unprovided demand is one no enabled registry entry provides" in "An unprovided demand is one no enabled registry entry provides"
- "A catalog-fulfilled or unmatched provider-fulfilled demand still refuses" in "Every other refusal stands with the switch on"
- "An over-subscribed contract or an unmatched component still refuses" in "An over-subscribed contract and an unmatched component still refuse with the switch on"
- "Skipped rows are readable beside a refusal" in "The skipped-demand rows are readable beside a refusal with the switch on"
- "An optional trait stays in its table" in "An unhandled optional trait is never a skipped row"
- "The switch is a literal" in "The render module carries the switch as a literal"

| Old clause | New requirement | Change |
| --- | --- | --- |
| `Kernel.Render` SHALL accept a per-render switch, off by default, that asks the render to skip ... | A caller may skip unprovided provider-fulfilled demands | verbatim, cut at the colon (the whole Source line does not fit beside the definition) |
| a resource or trait demand whose contract declares ... (the key is absent from core's `#contracts.providedBy`, the count the single-provider guard reads) | An unprovided demand is one no enabled registry entry provides | verbatim from the colon on, with the lead "Every **unprovided** demand SHALL be" (subject copied from the sentence; the colon takes the keyword, strength table) |
| Source: core `SPEC.md` §2.1 and §3.1 and capability `contract-fulfilment`, as amended by core changes ... | A caller may skip unprovided provider-fulfilled demands; An unprovided demand is one no enabled registry entry provides | verbatim, the whole line in both blocks (the earlier blocks lacked the clause "as amended by ...") |
| With the switch off, the render's verdicts, refusals and rendered output SHALL be exactly those... | With the switch off a render is as without the switch | verbatim |
| Source: core `SPEC.md` §2.1 and §3.1 and capability `contract-fulfilment`, as amended by core c... | With the switch off a render is as without the switch | verbatim |
| With the switch on: | A skipped trait demand does not refuse the render | verbatim |
| A skipped trait demand SHALL NOT refuse the render. Its component SHALL render every pair it ma... | A skipped trait demand does not refuse the render | verbatim |
| Source: core `SPEC.md` §2.1 and §3.1 and capability `contract-fulfilment`, as amended by core c... | A skipped trait demand does not refuse the render | verbatim |
| With the switch on: | A skipped resource demand omits its component | verbatim |
| A skipped resource demand SHALL NOT refuse the render, and its component SHALL be omitted: no p... | A skipped resource demand omits its component | verbatim |
| Source: core `SPEC.md` §2.1 and §3.1 and capability `contract-fulfilment`, as amended by core c... | A skipped resource demand omits its component | verbatim |
| Every skipped demand SHALL be reported as a skipped-demand row carrying the component, the cont... | Every skipped demand is reported as a skipped-demand row | verbatim |
| Source: core `SPEC.md` §2.1 and §3.1 and capability `contract-fulfilment`, as amended by core c... | Every skipped demand is reported as a skipped-demand row | verbatim |
| With the switch on: | The omission flag is set on every skipped row of an omitted component | verbatim |
| The omission flag SHALL be set on every skipped row of an omitted component, trait rows include... | The omission flag is set on every skipped row of an omitted component | verbatim |
| Source: core `SPEC.md` §2.1 and §3.1 and capability `contract-fulfilment`, as amended by core c... | The omission flag is set on every skipped row of an omitted component | verbatim |
| With the switch on: | Every other refusal stands with the switch on; An over-subscribed contract and an unmatched component still refuse with the switch on; The skipped-demand rows are readable beside a refusal with the switch on | verbatim, in each of the three blocks |
| Every other refusal SHALL stand: a catalog-fulfilled unresolved demand, a provider-fulfilled de... | Every other refusal stands with the switch on; An over-subscribed contract and an unmatched component still refuse with the switch on | the sentence and the whole Source line do not fit together in 500 characters: cut at the list boundary ", an over-subscribed provider-fulfilled contract", the lead repeated, the predicate "SHALL refuse the render as without the switch" repeated; no word is dropped and the comma between the first two items stays |
| The skipped-demand rows SHALL be readable on the diagnostics beside such a refusal. | The skipped-demand rows are readable beside a refusal with the switch on | verbatim; "such a refusal" gets a pointer to the two requirements that hold the refusals |
| Source: core `SPEC.md` §2.1 and §3.1 and capability `contract-fulfilment`, as amended by core changes ... | Every other refusal stands with the switch on; An over-subscribed contract and an unmatched component still refuse with the switch on; The skipped-demand rows are readable beside a refusal with the switch on | verbatim, the whole line (the earlier block lacked the clause "as amended by ...") |
| With the switch on: | An unhandled optional trait is never a skipped row | verbatim |
| An unhandled trait whose effective `optional` is true SHALL remain an unhandled-trait table ent... | An unhandled optional trait is never a skipped row | verbatim |
| Source: core `SPEC.md` §2.1 and §3.1 and capability `contract-fulfilment`, as amended by core c... | An unhandled optional trait is never a skipped row | verbatim |
| The generated render module SHALL carry the switch as a literal, so its own `gate` field agrees... | The render module carries the switch as a literal | verbatim |
| Source: core `SPEC.md` §2.1 and §3.1 and capability `contract-fulfilment`, as amended by core c... | The render module carries the switch as a literal | verbatim |

**A render refuses a platform whose core predates the provider count**: form c2 (REMOVED plus ADDED); old body 1487 characters; new bodies 455, 325, 232, 416, 310, 192.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A current-core platform is not affected | A render checks the core floor of the platform before staging, same name | no, byte for byte (checked) |
| An older-core platform is refused before staging | The refusal of a platform that predates the provider count stages nothing and does not fall back, same name | no, byte for byte (checked) |
| A render reads no platform Package | The check reads the recorded floor and not the platform's Package, same name | no, byte for byte (checked) |
| A platform shared by concurrent renders stays race-free | One acquired platform stays shareable across concurrent renders, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "An older core raises the typed error" in "A platform that predates the provider count is refused with PlatformCoreTooOldError"
- "A platform the constructor did not build decodes Package on its first call" in "The check decodes Package once on a platform the constructor did not build"

| Old clause | New requirement | Change |
| --- | --- | --- |
| `Kernel.Render` SHALL check, before staging, that the platform's core floor is met: that the pl... | A render checks the core floor of the platform before staging | verbatim |
| The check is sound because the render build evaluates the platform module's own core pin, the s... | A render checks the core floor of the platform before staging | verbatim |
| When it does not (the platform module pins a core release older than `2.0.0-alpha.12`, or carri... | A platform that predates the provider count is refused with PlatformCo... | "it does not" names the floor test of the first sentence: the platform's `#contracts` carries `providedBy` |
| The refusal SHALL NOT be a `*RenderError`, SHALL stage no render module and SHALL NOT fall back... | The refusal of a platform that predates the provider count stages noth... | "The refusal" names the error of the sentence before it |
| The check SHALL read the recorded floor through `CoreFloor()`, and on a platform the constructo... | The check reads the recorded floor and not the platform's Package | "The check" names the check of the first sentence of the old requirement |
| a platform the constructor did not build decodes `Package` once, on its first `CoreFloor()` or `Contracts()` call (the second half of the same sentence) | The check decodes Package once on a platform the constructor did not build | the sentence is repeated whole with the subject "A platform the constructor did not build" and the keyword SHALL in place of "decodes", and the pointer to its exact header |
| Apart from the floor, `Render` reads only the platform's `Metadata` and `Source`. | One acquired platform stays shareable across concurrent renders | the verb "reads" gets the keyword SHALL, as an ADDED block needs |
| One acquired platform therefore stays shareable, as data, across concurrent `Render` calls on o... | One acquired platform stays shareable across concurrent renders | verbatim |

**A render refuses a platform whose enabled entries share contract keys**: form c2 (REMOVED plus ADDED); old body 1969 characters; new bodies 389, 494, 378, 497, 448, 442, 434, 462, 387, 452, 425.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A colliding bridge platform is refused instead of rendering twice | The glue emits one collision row for each key in the collision report, same name | no, byte for byte (checked) |
| A platform pinning a core without the report renders unchanged | The glue emits no collision row when the platform carries no collision report, same name | no, byte for byte (checked) |
| A collision refuses under the skip switch | The kernel refuses a colliding platform for every instance and skip switch, same name | no, byte for byte (checked) |
| A collision and an over-subscription are both named, collision first | The collision cause is the first joined cause, same name | no, byte for byte (checked) |
| Collision rows equal the inventory on every served platform | The render's collision rows equal the inventory's CollidingEntries, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "The glue reads the collisions and does not compute them" in "The glue reads the collisions from core's derived contract inventory"
- "The glue emits the rows only when the platform carries collisions" in "The glue emits collision rows only with a collision report"
- "The kernel decodes the rows and refuses through the gate" in "The kernel decodes the collision rows and refuses the render"
- "A colliding key distorts the inventory the other rows are read against" in "The collision cause comes first because a colliding key distorts the inventory"
- "The message names the keys and the definers" in "The collision cause message names each key and its definers"
- "The message says how to remedy a collision" in "The collision cause message says how to remedy a collision"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The generated glue SHALL read the platform's contract collisions from core's derived contract i... | The glue reads the collisions from core's derived contract inventory | verbatim up to the colon, which becomes "." (the whole Source line does not fit beside the rest) |
| for every key in `#Platform.#contracts.collisions`, in that (ascending) order, it SHALL emit one... | The glue emits one collision row for each key in the collision report | "it" becomes "the generated glue", the subject of the first sentence; the clause starts the sentence ("For every key") |
| The glue SHALL emit the rows only when the platform value carries `collisions`, and SHALL emit ... | The glue emits collision rows only with a collision report | "the rows" names the collision rows of the first sentence; the first block ends at ", and SHALL emit none otherwise" (the colon becomes "."), the second block repeats "The glue SHALL emit none otherwise" and keeps the reason after the colon |
| and SHALL emit none otherwise: every core release carrying `#contracts` without the report fail... | The glue emits no collision row when the platform carries no collision report | verbatim; "otherwise" gets a pointer to the first block; the Source line is the first title with its parenthetical, whole; the title "core change `fold-colliding-contract-keys`" is dropped here and stays on every other block |
| The kernel SHALL decode the rows onto the render diagnostics and SHALL refuse the render, throu... | The kernel decodes the collision rows and refuses the render | "the rows" names the collision rows of the glue sentence |
| The refusal is platform-wide: it SHALL hold for every instance and whatever the caller's skip s... | The kernel refuses a colliding platform for every instance and skip switch | verbatim; "The refusal" gets a pointer to the requirement that states it |
| When a render refuses for several reasons, the collision cause SHALL be the first joined cause,... | The collision cause is the first joined cause; The collision cause comes first because a colliding key distorts the inventory | cut at the clause boundary ", because": the first block keeps ", ahead of unresolved demands, over-subscribed contracts and unmatched components", the second keeps ", because a colliding key is absent from ..."; the lead is repeated and the Source line is the first title with its parenthetical, whole, in the second block |
| The cause's message SHALL name, per row, the key, the number of enabled registry entries defini... | The collision cause message names each key and its definers | "The cause" names the collision cause of the sentences before it ("The collision cause's message"); cut before ", and SHALL say" |
| and SHALL say that a contract key must have exactly one enabled definer until side-by-side catal... | The collision cause message says how to remedy a collision | verbatim; the subject "The collision cause's message" is repeated |
| Because the render and the platform's `Contracts()` read the same report, the render's collision... | The render's collision rows equal the inventory's CollidingEntries | verbatim |
| Source: an open question of enhancement 0026 (the interim safety net ahead of that enhancement, ... | every block of this requirement except "The glue emits no collision row when the platform carries no collision report" and "The collision cause comes first because a colliding key distorts the inventory" | verbatim, the whole line |
| Source: an open question of enhancement 0026 (the interim safety net ahead of that enhancement, ... | "The glue emits no collision row when the platform carries no collision report"; "The collision cause comes first because a colliding key distorts the inventory" | the first title with its parenthetical, whole; the second title is dropped (the whole line does not fit) |

**A render refuses a not-routable platform no row explains**: form c2 (REMOVED plus ADDED); old body 544 characters; new bodies 283, 280.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| An unexplained not-routable verdict refuses | The kernel refuses a not-routable platform that no row explains, same name | no, byte for byte (checked) |
| No served platform raises the catch-all | The not-routable cause is never raised where the rows explain routable, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| The kernel SHALL decode core's `#contracts.routable` onto the render diagnostics, and SHALL ref... | The kernel refuses a not-routable platform that no row explains | verbatim |
| The cause SHALL be the last joined cause. | The kernel refuses a not-routable platform that no row explains | verbatim |
| It exists so that the kernel's decision agrees with the render module's own gate (which reads `... | The not-routable cause is never raised where the rows explain routable | "It" names the not-routable cause of the sentences before it |

**Matching stays in the glue and derived rules move into core one at a time**: form c2 (REMOVED plus ADDED); old body 810 characters; new bodies 395, 414.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| ADR-012 records where matching lives | The matching algorithm stays in the library's render glue, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "A derived rule moves with a parity test" in "A derived rule moves into core one at a time with a parity test"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The matching algorithm (which transformers a component is paired with, the demand buckets, and ... | The matching algorithm stays in the library's render glue | verbatim |
| ADR-012 records this. | The matching algorithm stays in the library's render glue | verbatim |
| A single derived rule MAY move into core when core can compute it from core shapes alone. Each ... | A derived rule moves into core one at a time with a parity test | verbatim |

**A render reports every contract its instance requires**: form c2 (REMOVED plus ADDED); old body 1438 characters; new bodies 450, 460, 251, 437.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Demand on a successful render | A render reports the instance's contract demand as data on RequiredContracts, same name | no, byte for byte (checked) |
| A component without traits | A render reports the instance's contract demand as data on RequiredContracts, same name | no, byte for byte (checked) |
| A component with traits and no resources | A render reports the instance's contract demand as data on RequiredContracts, same name | no, byte for byte (checked) |
| The demand equals the declared contracts of the instance | The contract demand is computed inside the build from the matcher's component set, same name | no, byte for byte (checked) |
| An omitted component's demand is reported | The contract demand is computed inside the build from the matcher's component set, same name | no, byte for byte (checked) |
| No components | The contract demand is computed inside the build from the matcher's component set, same name | no, byte for byte (checked) |
| Demand on a refusal | The field is set on every RenderResult and every RenderError, same name | no, byte for byte (checked) |
| A component without resources refuses | Computing the demand fails closed, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| The render SHALL report the instance's contract demand as data on `RenderDiagnostics.RequiredCo... | A render reports the instance's contract demand as data on RequiredCon... | verbatim |
| Source: enhancement 0013 ("The kernel reports every contract a render required, synthesised com... | A render reports the instance's contract demand as data on RequiredCon... | verbatim |
| The render SHALL report the instance's contract demand as data on `RenderDiagnostics.RequiredCo... | The contract demand is computed inside the build from the matcher's co... | verbatim |
| The demand SHALL be computed inside the build, from the same component set the matcher iterates... | The contract demand is computed inside the build from the matcher's co... | verbatim |
| The field SHALL be set on every `RenderResult` and on every `*RenderError`. A render that retur... | The field is set on every RenderResult and every RenderError | "The field" names the field of the first sentence |
| Computing the demand SHALL fail closed. A component's `#resources` is read with no presence tes... | Computing the demand fails closed | "the demand" names the instance's contract demand of the first sentence |

**Each render is its own in-memory build in its own context**: form c2 (REMOVED plus ADDED); old body 1109 characters; new bodies 367, 332, 408.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Repeated renders share nothing | Each render creates a fresh context, evaluates once and shares no built value, same name | no, byte for byte (checked) |
| Concurrent renders share the synthetic root | Each render creates a fresh context, evaluates once and shares no built value, same name | no, byte for byte (checked) |
| The synthetic roots are absolute | The render module is staged in memory under a synthetic root that does not exist on disk, same name | no, byte for byte (checked) |
| An existing render root is refused | The render module is staged in memory under a synthetic root that does not exist on disk, same name | no, byte for byte (checked) |
| The render module is served from memory | The render module and the overlay-mode inputs are served from the load overlay, same name | no, byte for byte (checked) |
| Local replacements resolve from an in-memory render module | The render module and the overlay-mode inputs are served from the load overlay, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| `Render` SHALL create a fresh `cue.Context` for the render, evaluate the staged render module e... | Each render creates a fresh context, evaluates once and shares no buil... | verbatim |
| The render module SHALL be staged in memory under a fixed synthetic root, an absolute path on e... | The render module is staged in memory under a synthetic root that does... | verbatim |
| The render module's generated `cue.mod/module.cue`, `cue.mod/local-module.cue` and glue file, a... | The render module and the overlay-mode inputs are served from the load... | verbatim |

**A render writes no staging file**: form c2 (REMOVED plus ADDED); old body 720 characters; new bodies 450, 308.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A successful render leaves the temp directory untouched | Kernel.Render writes no staging file, same name | no, byte for byte (checked) |
| A refusal leaves the temp directory untouched | Kernel.Render writes no staging file, same name | no, byte for byte (checked) |
| A render that writes is caught | A test that asserts that a render writes no staging file owns its temp directory, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| `Kernel.Render` SHALL write no staging file: it SHALL create, write or remove nothing under the... | Kernel.Render writes no staging file | verbatim |
| A test that asserts this SHALL point the process temp directory at a directory owned by that te... | A test that asserts that a render writes no staging file owns its temp... | "this" names the rule of the first sentence |

### Ledger: schema-dispatch

**DefaultSchemaModule constant**: form c2 (REMOVED plus ADDED); old body 3861 characters; new bodies 428, 450, 432, 259, 280, 291, 337, 291, 275, 338, 184, 396.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Empty Module resolves the v2 default | DefaultSchemaModule names an exact core release, same name | no, byte for byte (checked) |
| ResolvedVersion reports the v2 resolution | DefaultSchemaModule names an exact core release, same name | no, byte for byte (checked) |
| No doc comment cites a deleted package or the floating major | Doc comments cite the default by the constant's name, same name | no, byte for byte (checked) |
| A default move edits no test file | The constant is the only Go source that a default move edits, same name | no, byte for byte (checked) |
| The pin assertion checks shape, not the release | The schema tests assert the shape of the constant, same name | no, byte for byte (checked) |
| Served fixtures pin the default release | Every served test fixture declares the default core version, same name | no, byte for byte (checked) |
| An older-core test survives a default move | Every served test fixture declares the default core version, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "The documentation names the release of the collision report" in "2.0.0-alpha.13 is the first release that reports contract collisions"
- "The default reports the earlier inventories" in "2.0.0-alpha.13 reports the provider count, the comparable report and the contract inventory"
- "The default reports the registry shape" in "2.0.0-alpha.13 reports the registry shape and the context projection"
- "A platform between the floor and the default renders" in "The default is not the render floor"
- "The default advances through a reviewed change" in "The default advances only through a deliberate change that re-verifies the glue and fixtures"
- "A test reads the current core from the harness" in "No test spells the default release where it means the current core"
- "A synthetic literal is declared frozen" in "Synthetic, floor and skew literals stay literal and are declared frozen"

| Old clause | New requirement | Change |
| --- | --- | --- |
| `schema.DefaultSchemaModule` SHALL name an exact core release, not the floating `opmodel.dev/co... | DefaultSchemaModule names an exact core release | verbatim |
| It is the release the kernel's render glue, fixtures and parity oracle were verified against. | DefaultSchemaModule names an exact core release | verbatim |
| The constant is the only record of which release that is: this specification, the doc comments ... | DefaultSchemaModule names an exact core release | verbatim |
| `OCILoader.Load` with an empty `Module` field SHALL resolve this identifier. | DefaultSchemaModule names an exact core release | verbatim |
| `2.0.0-alpha.13` remains the first release that reports contract collisions on the derived `#Pl... | 2.0.0-alpha.13 is the first release that reports contract collisions | verbatim |
| The default's documentation SHALL keep naming `2.0.0-alpha.13` as that release and SHALL NOT at... | 2.0.0-alpha.13 is the first release that reports contract collisions | verbatim |
| It reports them on top of | 2.0.0-alpha.13 reports the provider count, the comparable report ... | "It reports them" becomes "`2.0.0-alpha.13` SHALL report contract collisions": the subject and the object are the words of the sentence before it, and the verb gets the keyword an ADDED block needs |
| the per-registry-entry provider count (`providedBy`), | 2.0.0-alpha.13 reports the provider count, the comparable report ... | verbatim |
| the comparable-predicate report (`comparable` and `discriminated`; enhancement 0015), | 2.0.0-alpha.13 reports the provider count, the comparable report ... | verbatim |
| the contract inventory (enhancement 0015: contracts are published as members beside transformer... | 2.0.0-alpha.13 reports the provider count, the comparable report ... | the final "," of the list becomes "." |
| It reports them on top of | 2.0.0-alpha.13 reports the registry shape and the context project... | "It reports them" becomes "`2.0.0-alpha.13` SHALL report contract collisions": the subject and the object are the words of the sentence before it, and the verb gets the keyword an ADDED block needs |
| the registry shape (enhancement 0019: a platform imports its catalog and embeds it whole into t... | 2.0.0-alpha.13 reports the registry shape and the context project... | verbatim |
| and the context projection (enhancement 0019: the kernel fills `#runtimeName` alone). | 2.0.0-alpha.13 reports the registry shape and the context project... | verbatim |
| The default is not the render floor: `Kernel.Render` and `Platform.Contracts()` accept every co... | The default is not the render floor | the verb "accept" gets the keyword SHALL, as an ADDED block needs |
| The constant advances only through a deliberate change that re-verifies the glue and fixtures a... | The default advances only through a deliberate change that re-verifies... | the verb "advances" gets the keyword SHALL, as an ADDED block needs |
| The deliberate re-verification is the review of the change that advances it. | The default advances only through a deliberate change that re-verifies... | verbatim |
| Doc comments that cite the default module identifier (`opm/kernel`, `opm/schema`) SHALL cite it... | Doc comments cite the default by the constant's name | verbatim |
| The constant SHALL be the only Go source that the default move edits. | The constant is the only Go source that a default move edits | verbatim |
| The fixture harness's declared core version SHALL be derived from it, never kept as a second li... | The constant is the only Go source that a default move edits | verbatim |
| Advancing the default therefore edits the constant and the fixture `cue.mod` pins, and no test ... | The constant is the only Go source that a default move edits | verbatim |
| No test SHALL spell the default release where it means the current core: a test that expects th... | No test spells the default release where it means the current core | verbatim |
| Synthetic, floor and skew literals that happen to equal or predate the default are not "the cur... | Synthetic, floor and skew literals stay literal and are declared froze... | the verb "stay" gets the keyword SHALL, as an ADDED block needs |
| Test files that keep an older or synthetic core literal on purpose are declared as frozen (capa... | Synthetic, floor and skew literals stay literal and are declared froze... | verbatim |
| The schema tests SHALL assert the constant's shape: an exact release on the `v2` major whose ve... | The schema tests assert the shape of the constant | verbatim |
| They SHALL NOT repeat the release as a literal. | The schema tests assert the shape of the constant | verbatim |
| The core version every served test fixture declares SHALL be the same release, so a fixture nev... | Every served test fixture declares the default core version | verbatim |
| A test that exercises an older core copies a fixture and re-pins the copy. | Every served test fixture declares the default core version | verbatim |
| It never commits a fixture pinning another release, and it reads the fixture's current pin from... | Every served test fixture declares the default core version | verbatim |

**Schema Loader interface**: form c2 (REMOVED plus ADDED); old body 574 characters; new bodies 342, 249.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Loader is the contract not the implementation | The Loader interface has a single method Load, same name | no, byte for byte (checked) |
| No second public Loader | The library exports no Loader implementation other than OCILoader, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| The library SHALL define `opm/schema.Loader` as an interface with a single method `Load(ctx *cu... | The Loader interface has a single method Load | verbatim |
| The library SHALL NOT export any `Loader` implementation other than `OCILoader` (see "OCILoader... | The library exports no Loader implementation other than OCILoader | the citation names the renamed requirement |

**OCILoader is the only public Loader**: form c2 (REMOVED plus ADDED); old body 1349 characters; new bodies 239, 345, 292, 362, 165.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Zero-value OCILoader resolves defaults | OCILoader is the sole public implementation of Loader, same name | no, byte for byte (checked) |
| Explicit overrides take precedence over env | OCILoader.Load resolves its empty fields, same name | no, byte for byte (checked) |
| Load failures are wrapped | OCILoader.Load builds the instance and wraps its errors, same name | no, byte for byte (checked) |
| A module that loads but does not build is a load failure | OCILoader.Load treats a built value that carries an error as a load failure, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "Load mutates no process-global state" in "OCILoader.Load uses no custom OCI client and mutates no process-global state"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The library SHALL expose `opm/schema.OCILoader` as the sole public implementation of `Loader`. ... | OCILoader is the sole public implementation of Loader | verbatim |
| `OCILoader.Load(ctx)` SHALL: | OCILoader.Load resolves its empty fields | verbatim |
| Resolve `Module` to `DefaultSchemaModule` when the field is empty. | OCILoader.Load resolves its empty fields | verbatim |
| Resolve `Registry` to the value derived from `os.Environ`'s `CUE_REGISTRY` when the field is em... | OCILoader.Load resolves its empty fields | verbatim |
| Resolve `CacheDir` to the value derived from `os.Environ`'s `CUE_CACHE_DIR` (or CUE's default w... | OCILoader.Load resolves its empty fields | verbatim |
| `OCILoader.Load(ctx)` SHALL: | OCILoader.Load builds the instance and wraps its errors | verbatim |
| Invoke `cuelang.org/go/cue/load.Instances([]string{module}, &load.Config{Env: derivedEnv})` wit... | OCILoader.Load builds the instance and wraps its errors | verbatim |
| Call `ctx.BuildInstance` on the returned instance and return the resulting `cue.Value` and any ... | OCILoader.Load builds the instance and wraps its errors | verbatim |
| `OCILoader.Load(ctx)` SHALL: | OCILoader.Load treats a built value that carries an error as a load fa... | verbatim |
| Treat a built value that carries an error (`Value.Err()` non-nil, e.g. an unresolved reference ... | OCILoader.Load treats a built value that carries an error as a load fa... | verbatim |
| `OCILoader.Load` MUST NOT use any custom OCI client (e.g., `oras-go`), MUST NOT bypass CUE's mo... | OCILoader.Load uses no custom OCI client and mutates no process-global... | verbatim |

**A pinned schema release is known without a load**: form c2 (REMOVED plus ADDED); old body 615 characters; new bodies 276, 338.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Default loader is pinned | OCILoader reports its pinned core release without any I/O, same name | no, byte for byte (checked) |
| Bare major is not pinned | OCILoader reports its pinned core release without any I/O, same name | no, byte for byte (checked) |
| Synthesis on a pinned kernel loads no schema | A kernel derives the synthesis release from the loader's pin or from its schema cache, same name | no, byte for byte (checked) |
| Synthesis on a bare-major kernel resolves through the cache | A kernel derives the synthesis release from the loader's pin or from its schema cache, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| `OCILoader` SHALL report, without any I/O, the exact core release its module identifier names: ... | OCILoader reports its pinned core release without any I/O | verbatim |
| A kernel whose configured loader pins an exact release SHALL derive the core release its synthe... | A kernel derives the synthesis release from the loader's pin or from i... | verbatim |

**Schema Cache memoizes a single Load per instance**: form c2 (REMOVED plus ADDED); old body 1435 characters; new bodies 489, 446, 324, 173.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Repeated Get returns the cached value | Cache.Get invokes the Loader exactly once per Cache, same name | no, byte for byte (checked) |
| Concurrent first Get is safe | Cache.Get invokes the Loader exactly once per Cache, same name | no, byte for byte (checked) |
| A zero value from the Loader is an unusable schema | The Cache treats a loaded value that carries an error as a load failure, same name | no, byte for byte (checked) |
| Loader errors are cached | The Cache does not retry a failed Load, same name | no, byte for byte (checked) |
| An errored schema build is memoised as an error | The Cache does not retry a failed Load, same name | no, byte for byte (checked) |
| The schema context is private | The Cache does not retry a failed Load, same name | no, byte for byte (checked) |
| Two Cache instances do not share state | No Loader result is cached at package scope, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| The library SHALL expose `opm/schema.Cache` as a struct with at minimum a `Loader Loader` field... | Cache.Get invokes the Loader exactly once per Cache | verbatim |
| `(*Cache).Get() (cue.Value, error)` SHALL invoke `Loader.Load(ctx)` exactly once per `Cache` in... | Cache.Get invokes the Loader exactly once per Cache | verbatim |
| Subsequent calls, including the call that loses the race, SHALL return the cached `cue.Value`... | Cache.Get invokes the Loader exactly once per Cache | verbatim |
| Whatever `Loader` it wraps, the Cache SHALL treat a loaded value that carries an error (`Value.... | The Cache treats a loaded value that carries an error as a load failur... | verbatim |
| That covers both a value whose build failed and the zero `cue.Value` a Loader returns with a ni... | The Cache treats a loaded value that carries an error as a load failur... | verbatim |
| The Cache SHALL NOT retry a failed Load: an error, including one for a schema whose build faile... | The Cache does not retry a failed Load | verbatim |
| A caller that must compile a value against the schema obtains the schema's context from the ret... | The Cache does not retry a failed Load | verbatim |
| The library MUST NOT cache the `Loader`'s result at package scope. There SHALL be no package-le... | No Loader result is cached at package scope | verbatim |

**Path inventory exposed as package-level vars**: form b (MODIFIED parent plus ADDED); old body 3777 characters; new bodies 466, 479, 178, 260, 448.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Consumer references a path directly | Path inventory exposed as package-level vars, same name | no, byte for byte (checked) |
| Paths with an accessor are not exported | Path inventory exposed as package-level vars, same name | no, byte for byte (checked) |
| Matcher and transformer paths are gone | Path inventory exposed as package-level vars, same name | no, byte for byte (checked) |
| Platform view and context sub-paths are not exported | Path inventory exposed as package-level vars, same name | no, byte for byte (checked) |
| Instance and module reads go through the inventory | Path inventory exposed as package-level vars, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "The exported paths and their readers" in "Path inventory exposed as package-level vars"
- "The internal paths and their readers" in "Every path other than the exported three lives in opm/internal/corepath"
- "The inventory is exactly the listed paths" in "The inventory is exactly the exported and the internal paths"
- "The core-release constants are exported" in "opm/schema exports the core-release constants ProvidedBySince and ProvidesSince"
- "A path no kernel code reads is removed" in "The paths of the retired matcher, executor and context builder stay removed"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The library SHALL keep every CUE path the kernel's Go code reads on an OPM artifact as one pack... | Path inventory exposed as package-level vars | verbatim |
| `opm/schema` SHALL export exactly the three paths a frontend reads on an artifact `Package` for... | Path inventory exposed as package-level vars | verbatim |
| The paths and their readers are listed in scenario "The exported paths and their readers". | Path inventory exposed as package-level vars | added words (not in the old body) |
| Every other path SHALL live in the internal package `opm/internal/corepath`, where no consumer ... | Every path other than the exported three lives in opm/internal/corepat... | "other" names the three exported paths of the sibling sentence |
| The paths and their readers are listed in scenario "The internal paths and their readers". | Every path other than the exported three lives in opm/internal/corepat... | added words (not in the old body) |
| The paths and their readers are listed in scenarios "The exported paths and their readers" and ... | The inventory is exactly the exported and the internal paths | added words (not in the old body) |
| The inventory SHALL be exactly these paths. | The inventory is exactly the exported and the internal paths | verbatim |
| `opm/schema` SHALL export the core-release constants `ProvidedBySince` (the render floor) and `... | opm/schema exports the core-release constants ProvidedBySince and Prov... | verbatim |
| The paths the retired Go matcher, executor and context builder read | The paths of the retired matcher, executor and context builder stay re... | verbatim |
| (listed in scenario "Matcher and transformer paths are gone") | The paths of the retired matcher, executor and context builder stay re... | added words (not in the old body) |
| and `ModuleMetadataPath` SHALL stay removed: the render build reads the instance's components a... | The paths of the retired matcher, executor and context builder stay re... | verbatim |
| A path that no kernel code reads, by variable or relative to an inventory path, is removed, not... | The paths of the retired matcher, executor and context builder stay re... | verbatim |

**Metadata decoders are free functions**: form c2 (REMOVED plus ADDED); old body 1222 characters; new bodies 310, 416, 494.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Decoders are not exported | Each artifact's metadata decodes through one unexported free function in the package of its caller, same name | no, byte for byte (checked) |
| Provider metadata falls back to caller-supplied name | Each artifact's metadata decodes through one unexported free function in the package of its caller, same name | no, byte for byte (checked) |
| Decoding a module artifact | Each decoder reads a raw cue.Value and returns the canonical metadata struct or an error, same name | no, byte for byte (checked) |
| Decoding a catalog artifact | Each decoder reads a raw cue.Value and returns the canonical metadata struct or an error, same name | no, byte for byte (checked) |
| Platform metadata hoists top-level type | Each decoder reads a raw cue.Value and returns the canonical metadata struct or an error, same name | no, byte for byte (checked) |
| Each metadata type has one name | Each decoder reads a raw cue.Value and returns the canonical metadata struct or an error, same name | no, byte for byte (checked) |
| Missing metadata is fatal for module/instance/platform | Consumers reach decoded metadata through constructors and a constructor adds no error prefix, same name | no, byte for byte (checked) |
| Missing metadata is fatal for a catalog | Consumers reach decoded metadata through constructors and a constructor adds no error prefix, same name | no, byte for byte (checked) |
| A decode failure names its artifact once | Consumers reach decoded metadata through constructors and a constructor adds no error prefix, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| The library SHALL decode each artifact's metadata through one unexported free function per arti... | Each artifact's metadata decodes through one unexported free function ... | verbatim |
| Each decoder MUST accept a raw `cue.Value` at the artifact root, read it through the `opm/schem... | Each decoder reads a raw cue.Value and returns the canonical metadata ... | verbatim |
| Consumers reach decoded metadata only through the artifact constructors and the kernel's acquis... | Consumers reach decoded metadata through constructors and a constructo... | verbatim |
| A constructor (`NewModuleFromValue`, `NewPlatformFromValue`, `NewCatalogFromValue`) SHALL retur... | Consumers reach decoded metadata through constructors and a constructo... | verbatim |

**Front-door docs describe the default core as an exact pin**: form c2 (REMOVED plus ADDED); old body 621 characters; new bodies 321, 319.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| The API stability section says pinned by default | Front-door docs state an exact pin for the default core, same name | no, byte for byte (checked) |
| The schema versioning section says pinned by default | Front-door docs state an exact pin for the default core, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "An additive change is explained" in "Front-door docs say how an additive schema change reaches a render"

| Old clause | New requirement | Change |
| --- | --- | --- |
| Where `README.md` or `AGENTS.md` states which core the default schema loader uses, it SHALL sta... | Front-door docs state an exact pin for the default core | verbatim |
| Neither SHALL present floating-major resolution as the mechanism that absorbs additive schema c... | Front-door docs state an exact pin for the default core | verbatim |
| Where either says that an additive schema change within a major needs no Go API change, it SHAL... | Front-door docs say how an additive schema change reaches a render | "either" names the two files of the first sentence |

**Single OPM schema, externally resolved and pinned by default**: form c2 (REMOVED plus ADDED); old body 730 characters; new bodies 218, 168, 112, 229.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Schema resolved via the default module identifier | The library consumes one OPM CUE schema resolved through CUE's module system, same name | no, byte for byte (checked) |
| Caller-pinned earlier major still loads | The library consumes one OPM CUE schema resolved through CUE's module system, same name | no, byte for byte (checked) |
| Default resolves the pinned release | The default loader resolves the exact release and the bare major is opt-in, same name | no, byte for byte (checked) |
| Bare major resolves within the v2 major (opt-in) | The default loader resolves the exact release and the bare major is opt-in, same name | no, byte for byte (checked) |
| No in-tree schema source | The library vendors or embeds no schema source, same name | no, byte for byte (checked) |
| Evaluated module has no apiVersion field | The schema defines no top-level ApiVersion and artifact roots carry no apiVersion field, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| The library SHALL consume exactly one OPM CUE schema package: `opmodel.dev/core@v2` (or a calle... | The library consumes one OPM CUE schema resolved through CUE's module ... | verbatim |
| The default loader SHALL resolve the exact release `schema.DefaultSchemaModule` names; the bare... | The default loader resolves the exact release and the bare major is op... | verbatim |
| The library MUST NOT vendor or embed the schema source under `library/apis/core/` or any other ... | The library vendors or embeds no schema source | verbatim |
| The schema package MUST NOT define a top-level `#ApiVersion` constant. | The schema defines no top-level ApiVersion and artifact roots carry no... | verbatim |
| Artifact roots (`#Module`, `#ModuleInstance`, `#Component`, `#ComponentTransformer`, `#Platform... | The schema defines no top-level ApiVersion and artifact roots carry no... | verbatim |

### Ledger: fix pass of the units before library-B3

The should-fix findings of the review of library-B2. Titles are shortened by whole titles only. "T1" to "T4" and "TL" name these titles of enhancement 0012: T1 "Ordering is kind-class order in the Kubernetes tier; ...", T2 "The Kubernetes runtime decisions live in a fenced tier, ...", T3 "The library owns the whole deletion sequence but only the per-object verdict and the order for apply", T4 "The stale set is component-blind, and the inventory digest hashes a canonical field encoding", TL "Labels are stamped by the CUE render; ...".

| Block | Source line now | Handling |
| --- | --- | --- |
| kubernetes-tier: Apply stages follow the weight table | T3 | The sentence "After it comes one stage per distinct weight of the remaining objects, in ascending weight." moves to the new "Stages after the first follow ascending weight" (T1), which names the parent by its header and takes SHALL. |
| kubernetes-tier: Ordering across modules belongs to a future Bundle | T1 | The sentence on a frontend's readiness gating moves to the new "A frontend's readiness gating is outside the cross-module ordering rule", which names the parent by its header; "is outside this rule" becomes "SHALL be outside this rule". |
| kubernetes-tier: A weight is looked up by group, version and kind first | T1 | The old sentence is cut at the list boundary "; then by kind alone". The second block "A weight falls back to the kind alone, then to the Class weight, then to 1000" repeats the lead "A weight SHALL be looked up" and names the first by its header. The first block is renamed (it no longer holds the whole lookup); no other file cites the old name. |
| kubernetes-tier: A StatefulSet is ready by its update strategy | T2 | The sentence is cut at the list boundary "; otherwise when". The new "A StatefulSet is ready when its updated replicas and revisions agree" repeats the lead and names the first by its header; only the second block keeps "Otherwise `NotReady`.", because it follows all three cases |
| kubernetes-tier: Empty UUIDs and blank annotations pass the delete verdict | T3 | The sentences "An adopt annotation naming this instance SHALL change nothing." and "An object being deleted SHALL NOT be skipped for that reason." move to "An annotation naming this instance and a deleting object pass the delete verdict", which names the first by its header for "the owner comparison" and "the adoption comparison". The two scenarios on these sentences move with them. |
| kubernetes-tier: The inventory digest is sha256 over a tagged, length-prefixed encoding | T4 | The sentence "Each entry SHALL be written as ..." moves to "Each entry of the inventory digest is written with length-prefixed fields", which repeats the first sentence of the parent. |
| kubernetes-tier: The adopted-elsewhere message names the object, the adopter and the way back | T3 and enhancements#103 | The sentence on an inventoried object moves to "The adopted-elsewhere message for an inventoried object says it is dropped"; "it" becomes "the `adopted-elsewhere` message", words of the first sentence. |
| kubernetes-tier: ten blocks with `Source: enhancement 0012.` | T1, T2, T3, T4 or TL | One title that supports the rule is added; no body changes. enhancements#103 stays where the old text had it. |
| single-build-render: The render and the platform's contract inventory read one provider count | 0010 (t10) and 0015 (t15a) | The second sentence is cut at its list boundary "; `Routable`": "A render refuses on over-subscription exactly when the platform's inventory reads it" and "Routable reads false exactly when the render carries an over-subscription row or a collision row" repeat "Because both the render and the platform's `Contracts()` read this one count," and name the first by its header for "this one count". "reads false" becomes "SHALL read false". |
| single-build-render: An over-subscribed contract is reported as a row and refuses the render | 0015 (t15b) | The sentence "The rows SHALL be built by iterating `providedBy` unconditionally, ..." moves to "The over-subscription rows are built by iterating providedBy unconditionally" (0010 and 0015); "The rows" becomes "The over-subscription rows", words of the first sentence. |
| single-build-render: The provider count lists the registry keys that require each provider-fulfilled contract | 0015 (t15a) | The parenthetical "(the catalog module paths with their major, ...)" and the clause ", whether or not any enabled entry defines the contract" move to "The registry keys of the provider count are the catalog module paths with their major" and "The provider count lists a contract whether or not any enabled entry defines it"; both name the first by its header. The old scenario "Providers count when the defining catalog is disabled" moves to the second. |
| single-build-render: An older instance requirement is plain data | none | "such mark" has its referent: the sentence "When the instance requires a NEWER build than the platform carries, the configured policy decides: ..." is copied from the old requirement before it. |
| single-build-render: An unstated trait posture fails closed as a build error | none | The first clause of the sibling sentence "An unhandled trait's effect SHALL be governed by its effective `optional` value read from the component's trait attachment" is copied as context; its colon becomes a full stop. |
| schema-dispatch: 2.0.0-alpha.13 reports ... (two blocks) | none | The headers and the WHEN lines of "The default reports the earlier inventories" and "The default reports the registry shape" name `2.0.0-alpha.13`, the only release the old text says it of. |
| schema-dispatch: three blocks that began with "The constant" or "The schema tests" | none | "The constant" is followed by `schema.DefaultSchemaModule`; the schema-tests block starts with the first sentence of the old requirement, which names it. |

Strength rows of this pass: "reads false" to "SHALL read false" in "Routable reads false exactly when the render carries an over-subscription row or a collision row"; "is outside this rule" to "SHALL be outside this rule" in "A frontend's readiness gating is outside the cross-module ordering rule"; "After it comes" to "After it SHALL come" in "Stages after the first follow ascending weight".

### Ledger: fix pass of library-B3 (T4.13-u3-fix)

The findings of the review of library-B3 that the first fix left open, under the rule that a new body is built from whole old sentences. Each touched body was re-read against `origin/main`. A Source line is whole; where it did not fit, the body was split at a list or clause boundary. The other rows of the ledgers above that changed carry the new text in their clause tables.

| Block | Old text on `origin/main` | New text | Row |
| --- | --- | --- | --- |
| single-build-render: A caller may skip unprovided provider-fulfilled demands | "...requiring that contract key (the key is absent from core's `#contracts.providedBy`, the count the single-provider guard reads). ... Source: core `SPEC.md` ..., as amended by core changes ..." | the parenthetical moves to "An unprovided demand is one no enabled registry entry provides"; both blocks carry the whole Source line | verbatim except the pointer subject and "SHALL be" (strength table) |
| single-build-render: Every other refusal stands with the switch on (three blocks) | one bullet: "Every other refusal SHALL stand: <four items> SHALL refuse the render as without the switch. The skipped-demand rows SHALL be readable on the diagnostics beside such a refusal." | cut at the list boundary before the third item and at the sentence boundary; each block has the lead "With the switch on:", the predicate repeated, and the whole Source line | verbatim; pointer in the third block |
| single-build-render: five collision blocks | one body of six sentences with the Source line "an open question of enhancement 0026 (the interim safety net ..., ...), core change ..." | each block carries the whole line, or (two blocks) the first title with its parenthetical; the sentence is cut at the colon, ", and SHALL emit none otherwise", the sentence boundary, ", because" and ", and SHALL say" | verbatim; "it" becomes "the generated glue"; pointers in two blocks |
| kernel-runtime: Verbs check their context between stages / after the build | "...after the directory is read, after the values sources are merged, after the package is built and before the next stage starts. ..." | cut at ", after the package is built"; the lead repeated; the whole Source line in both | verbatim |
| kernel-runtime: The registry acquire verbs and Render check their context | "The registry acquire verbs SHALL check it ... and `Kernel.Render` SHALL check it ..." | "it" becomes "their context" and "its context" ("their context" is in the sentence before it) | pronoun replaced by the antecedent |
| kernel-runtime: two cancelled-check blocks | "When one of these checks finds the context done, the verb SHALL return ..." | the sentence is repeated once for each group of checks, each with a pointer to the exact headers | verbatim; pointers |
| kernel-runtime: A compiled file source unwraps a top-level values field | "...and return a `Source` ...  When a kernel operation compiles that source, ..." | the paraphrase "The source is the one that `Kernel.LoadSourceFromFile` returns." is replaced by a pointer to "LoadSourceFromFile reads and parses a file" | pointer |
| kernel-runtime: two blocks "When they leave one or more unset" | "When they leave one or more unset" (they = the values of the sentence before it) | "When the values leave one or more unset" | subject copied |
| kernel-runtime: SynthesizeInstance may refuse more ... | Source: library#211, settled by the owner in the pull request that closes it; opm-operator#258 for the report of both checks. | the whole line | verbatim |
| artifact-types: A layered instance's Source is in overlay mode | "With one or more values sources (the `Source` type `ValidateConfigDetailed` accepts), the kernel SHALL unify them ... The returned instance's `Source` SHALL then be overlay mode" | "With one or more values sources (...), the returned instance's `Source` SHALL then be overlay mode" | condition copied from the sentence before it |
| artifact-types: A registry entry with no embedded catalog fails as a missing required field | "... fails here as a missing required field naming the entry." | "... SHALL fail here, at the gate, as a missing required field naming the entry." | "the gate" copied from the old requirement |
| kubernetes-tier: The deletion plan orders its entries by delete weight; A deletion plan needs no render and no stored plan; The plan constructor is the only way to build a plan with steps | Source: enhancement 0012 ("The library owns ..." and "The Kubernetes runtime surface homes ..."), ADR-008 (Deletion plans). | one or both titles of that line, whole, plus the ADR clause where it fits | whole titles only |
| kubernetes-tier: The deletion transition returns the next state and one action; The deletion transition performs no action and names only steps of the plan; A safety-excluded step is skipped without a read | Source: enhancement 0012 (three titles) (enhancements#103), ADR-008 rules 1 to 3. | the first title, `(enhancements#103)` and the ADR clause | whole titles only; the first body is cut at the sentence boundary |
| kubernetes-tier: A state that cannot belong to the plan is refused; A state that awaits something it cannot await is refused | "...cannot belong to the plan: its next step lies outside the plan, ..., it awaits something after the last step, or what it awaits is not a defined value." with "Source: enhancement 0012 (\"The library owns ...\"), ADR-008 rule 2." | cut at the list boundary ", it awaits something after the last step", the lead repeated | verbatim; whole Source line in both |
| kubernetes-tier: A weight is looked up by group, version and kind first | header "A weight is looked up by group and kind first" over a body that says "by group, version and kind first" | the header says "group, version and kind" | header only |

### Ledger: kernel-runtime

**Kernel Type and Construction**: form c2 (REMOVED plus ADDED); old body 621 characters; new bodies 292, 328.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Default construction | The Kernel is one struct that only kernel.New constructs, same name | no, byte for byte (checked) |
| Registry option seeds the schema loader | The schema cache is backed by the OCI loader of the kernel's registry mapping, same name | no, byte for byte (checked) |
| Construction with options | The schema cache is backed by the OCI loader of the kernel's registry mapping, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| The library SHALL expose a `Kernel` struct in `opm/kernel/` that serves as the single public an... | The Kernel is one struct that only kernel.New constructs | verbatim |
| The struct SHALL be constructible only via the `kernel.New(opts ...Option)` function. | The Kernel is one struct that only kernel.New constructs | verbatim |
| Absent `WithSchemaLoader`, the schema cache SHALL be backed by an OCI loader whose registry map... | The schema cache is backed by the OCI loader of the kernel's registry mapping | verbatim |
| An explicit `WithSchemaLoader` SHALL take precedence regardless of option order. | The schema cache is backed by the OCI loader of the kernel's registry mapping | verbatim |
| Construction SHALL create no `cue.Context` and perform no CUE evaluation. | The Kernel is one struct that only kernel.New constructs | verbatim |

**The Kernel owns no build context**: form c2 (REMOVED plus ADDED); old body 969 characters; new bodies 164, 357, 164, 287.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Repeated acquisitions retain nothing | Each build operation creates and drops its own cue.Context, same name | no, byte for byte (checked) |
| Artifacts cross Kernels | No kernel method takes a cue.Value as a values input or returns one to unify across calls, same name | no, byte for byte (checked) |
| No context accessor | The Kernel holds no cue.Context, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "A held value keeps its operation's runtime alive" in "Returned values keep their operation's runtime alive while the caller holds them"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The Kernel SHALL hold no `cue.Context`. | The Kernel holds no cue.Context | verbatim |
| Every kernel operation that builds an artifact (module, platform and instance acquisition, synt... | Each build operation creates and drops its own cue.Context | verbatim |
| The values an operation returns (an artifact's `Package`, a validated value) keep that operatio... | Returned values keep their operation's runtime alive while the caller holds them | SHALL added: the old sentence has no keyword |
| No kernel method SHALL take a `cue.Value` from the caller as a values input (the schema `Valida... | No kernel method takes a cue.Value as a values input or returns one to unify across calls | verbatim |
| The only long-lived evaluation state the Kernel owns is its schema cache, which holds a private... | The Kernel holds no cue.Context | verbatim |

**Goroutine Safety Contract**: form b (MODIFIED parent plus ADDED); old body 1578 characters; new bodies 439, 307, 312, 188, 408, 108.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Documentation states the contract | Goroutine Safety Contract, same name | no, byte for byte (checked) |
| Documentation retracts the shared-platform model | Goroutine Safety Contract, same name | no, byte for byte (checked) |
| Repeated renders retain nothing | Goroutine Safety Contract, same name | no, byte for byte (checked) |
| Concurrent acquisitions on one Kernel | Goroutine Safety Contract, same name | no, byte for byte (checked) |
| Race detector runs on the render packages | Goroutine Safety Contract, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "A held value keeps its context alive" in "A returned value keeps its operation's context alive as long as the caller holds it"
- "A render shares nothing with another render" in "Kernel.Render shares nothing between renders"
- "Several goroutines render on one Kernel" in "Concurrency is across operations, never within one"
- "The documentation sizes a render pool by memory" in "The package documentation states the concurrency rule and the render pool size"
- "The retracted model does not appear" in "The retracted shared-platform model is not a supported shape"

| Old clause | New requirement | Change |
| --- | --- | --- |
| A single `Kernel` SHALL be safe for concurrent use across its own method calls: no operation sh... | Goroutine Safety Contract | verbatim |
| A value the operation returns (an artifact's `Package`, a `*kernel.Compiled`) keeps that contex... | A returned value keeps its operation's context alive as long as the caller holds it | SHALL added: the old sentence has no keyword |
| A consumer that needs concurrent operations uses one `Kernel` per process; the package document... | Goroutine Safety Contract | verbatim |
| `Kernel.Render` SHALL share nothing between renders: each render is its own CUE build in a fres... | Kernel.Render shares nothing between renders | verbatim |
| A `*kernel.Compiled` the caller holds keeps its render's build alive until the caller releases ... | Kernel.Render shares nothing between renders | verbatim |
| Concurrency is across operations, never within one; a consumer rendering from several goroutine... | Concurrency is across operations, never within one | SHALL added: the old clause has no keyword |
| The package documentation SHALL state this, SHALL NOT present a value built into a shared conte... | The package documentation states the concurrency rule and the render pool size | verbatim |
| The retracted shared-materialized-platform model and its mutex stopgap SHALL NOT appear as supp... | The retracted shared-platform model is not a supported shape | verbatim |
| (added words) | A returned value keeps its operation's context alive as long as the caller holds it | pointer to the parent for 'that context' |
| (added words) | The package documentation states the concurrency rule and the render pool size | pointer for 'this' |

**Canonical Implementations Live on Kernel**: form c2 (REMOVED plus ADDED); old body 573 characters; new bodies 368, 204.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| ValidateConfigDetailed is a kernel method | Layered validation and instance processing live on the Kernel, same name | no, byte for byte (checked) |
| ValidateConfig is a kernel method | Layered validation and instance processing live on the Kernel, same name | no, byte for byte (checked) |
| ValidateConfigPartial is a kernel method | Layered validation and instance processing live on the Kernel, same name | no, byte for byte (checked) |
| ProcessModuleInstance is a kernel method | Layered validation and instance processing live on the Kernel, same name | no, byte for byte (checked) |
| opm/validate package is gone | No standalone validate or ParseModuleInstance function remains, same name | no, byte for byte (checked) |
| opm/helper/values package is gone | No standalone validate or ParseModuleInstance function remains, same name | no, byte for byte (checked) |
| module.ParseModuleInstance free function is gone | No standalone validate or ParseModuleInstance function remains, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| The canonical Go implementation of layered values validation and of instance processing (concre... | Layered validation and instance processing live on the Kernel | verbatim |
| Instance processing SHALL be kernel-internal: it is reached only through `AcquireInstanceFromDi... | Layered validation and instance processing live on the Kernel | verbatim |
| No standalone `validate.Config` / `validate.ConfigPartial` / `module.ParseModuleInstance` free ... | No standalone validate or ParseModuleInstance function remains | verbatim |

**Kernel.LoadSourceFromFile auto-unwraps the values field**: form c2 (REMOVED plus ADDED); old body 774 characters; new bodies 275, 499, 120.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Values file is auto-unwrapped | A compiled file source unwraps a top-level values field, same name | no, byte for byte (checked) |
| File without values field passes through | A compiled file source unwraps a top-level values field, same name | no, byte for byte (checked) |
| Syntax errors fail at load, schema errors at use | LoadSourceFromFile reads and parses a file, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "The method does not depend on LoadValuesFile" in "LoadSourceFromFile does not depend on loaderfile.LoadValuesFile"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The `*Kernel.LoadSourceFromFile(path string)` method SHALL read the file at `path`, parse it fo... | LoadSourceFromFile reads and parses a file | verbatim |
| The method SHALL NOT evaluate CUE. | LoadSourceFromFile reads and parses a file | verbatim |
| When a kernel operation compiles that source, it SHALL load it through cue/load at the file's d... | A compiled file source unwraps a top-level values field | verbatim |
| - If the evaluated value contains a top-level `values:` field whose `Exists()` is true and `Err... | A compiled file source unwraps a top-level values field | verbatim |
| - Otherwise the compiled source SHALL be the whole evaluated value. | A compiled file source unwraps a top-level values field | verbatim |
| The method SHALL NOT depend on `loaderfile.LoadValuesFile` (which is removed). | LoadSourceFromFile does not depend on loaderfile.LoadValuesFile | subject copied from the first sentence |
| (added words) | A compiled file source unwraps a top-level values field | pointer: 'The source is that of requirement "LoadSourceFromFile reads and parses a file".' for 'that source'; the earlier sentence 'The source is the one that `Kernel.LoadSourceFromFile` returns.' is gone |

**Registry Configuration Option**: form c2 (REMOVED plus ADDED); old body 950 characters; new bodies 373, 426, 56, 164, 141.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Registry option used for resolution | WithRegistry sets the mapping for the render build, registry acquisition and directory acquisition, same name | no, byte for byte (checked) |
| Directory acquisition uses the kernel mapping | WithRegistry sets the mapping for the render build, registry acquisition and directory acquisition, same name | no, byte for byte (checked) |
| Values source compilation uses the kernel mapping | WithRegistry sets the mapping for synthesis, file-backed values sources and the default schema cache, same name | no, byte for byte (checked) |
| No per-call registry parameter | No acquire verb takes a per-call registry override, same name | no, byte for byte (checked) |
| No default applied | Absent WithRegistry the kernel inherits CUE_REGISTRY and applies no default, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "The option mutates no process environment" in "WithRegistry mutates no process environment"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The `Kernel` SHALL accept a `WithRegistry(string)` option that sets the one OCI registry mappin... | WithRegistry sets the mapping for the render build, registry acquisition and directory acquisition | verbatim part |
| The `Kernel` SHALL accept a `WithRegistry(string)` option that sets the one OCI registry mappin... | WithRegistry sets the mapping for the render build, registry acquisition and directory acquisition | verbatim list part (tail '.') |
| The `Kernel` SHALL accept a `WithRegistry(string)` option that sets the one OCI registry mappin... | WithRegistry sets the mapping for synthesis, file-backed values sources and the default schema cache | verbatim part |
| The `Kernel` SHALL accept a `WithRegistry(string)` option that sets the one OCI registry mappin... | WithRegistry sets the mapping for synthesis, file-backed values sources and the default schema cache | verbatim list part (tail '.') |
| No acquire verb SHALL take a per-call registry override. | No acquire verb takes a per-call registry override | verbatim |
| Absent the option, the kernel SHALL inherit `CUE_REGISTRY` from the process environment and SHA... | Absent WithRegistry the kernel inherits CUE_REGISTRY and applies no default | referent copied from the first sentence |
| The option MUST NOT mutate process environment state; the mapping is plumbed into each operatio... | WithRegistry mutates no process environment | referent copied from the first sentence |

**Kernel.SynthesizeInstance method**: form c2 (REMOVED plus ADDED); old body 2412 characters; new bodies 438, 409, 299, 155, 333, 491, 149, 160, 364.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| SynthesizeInstance produces an end-to-end instance | SynthesizeInstance takes an InstanceInput, same name | no, byte for byte (checked) |
| SynthesizeInstance rejects unconcrete result | SynthesizeInstance builds the instance in one build and decodes its metadata, same name | no, byte for byte (checked) |
| SynthesizeInstance attributes a values violation to its source | A values error is framed and names each violating source, same name | no, byte for byte (checked) |
| SynthesizeInstance attributes a values conflict that fails the build | A failed synthesis build is built again without the rendered values file, same name | no, byte for byte (checked) |
| A build failure with clean values is returned unchanged | Every other build failure is returned unchanged, same name | no, byte for byte (checked) |
| Incomplete values do not mask a build failure | Every other build failure is returned unchanged, same name | no, byte for byte (checked) |
| SynthesizeInstance surfaces synth errors before validation | A missing input fails with an opm/errors sentinel, same name | no, byte for byte (checked) |
| SynthesizeInstance refuses a subpackage module | A missing input fails with an opm/errors sentinel, same name | no, byte for byte (checked) |
| SynthesizeInstance uses the Kernel's cue.Context | The synthesis build runs in a context the method creates and releases, same name | no, byte for byte (checked) |
| Operator-shaped caller needs no cue.Context | The synthesis build runs in a context the method creates and releases, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "The synthesized package imports the schema release's major" in "The synthesized package imports core at the major of the kernel's schema release"
- "A failed build validates the compiled values without concreteness" in "A failed synthesis build validates the compiled values against the values-free #config"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The `*Kernel` type SHALL expose a method `SynthesizeInstance(ctx context.Context, in InstanceIn... | SynthesizeInstance takes an InstanceInput | verbatim |
| It SHALL build the instance spec by single-build CUE evaluation inside the module's staged sour... | SynthesizeInstance builds the instance in one build and decodes its metadata | subject copied from the first sentence |
| The synthesized package imports `core` at the major of the kernel's schema release (read from t... | The synthesized package imports core at the major of the kernel's schema release | SHALL added: the old sentence has no keyword |
| The build SHALL run in a context the method creates and releases; the method SHALL NOT consult ... | The synthesis build runs in a context the method creates and releases | subject copied from the first sentence |
| When the build fails and the call's merged values exist, the method SHALL attribute the failure... | A failed synthesis build is built again without the rendered values file | subject copied from the first sentence |
| When the build fails and the call's merged values exist, the method SHALL attribute the failure... | A failed synthesis build is built again without the rendered values file | verbatim list part (tail '.') |
| When the build fails and the call's merged values exist, the method SHALL attribute the failure... | A failed synthesis build validates the compiled values against the values-free #config | subject copied from the first sentence |
| When the build fails and the call's merged values exist, the method SHALL attribute the failure... | A failed synthesis build validates the compiled values against the values-free #config | verbatim list part (tail '.') |
| A values error SHALL be returned framed `Kernel.SynthesizeInstance: instance "<name>": …`, with... | A values error is framed and names each violating source | verbatim |
| In every other case (no values, the values-free build also fails, or the values are clean) the ... | Every other build failure is returned unchanged | subject copied from the first sentence |
| A missing required input SHALL fail with an error wrapping the corresponding `opm/errors` senti... | A missing input fails with an opm/errors sentinel | verbatim |
| (added words) | A failed synthesis build validates the compiled values against the values-free #config | pointer for 'that build' and 'it' |
| (added words) | A failed synthesis build validates the compiled values against the values-free #config | keyword copied from the old sentence |

**Tier-2 validation runs where values are applied**: form c2 (REMOVED plus ADDED); old body 790 characters; new bodies 205, 451, 138.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Kernel re-validates after Detailed | Both instance verbs check the sources against #config at their own positions, same name | no, byte for byte (checked) |
| Kernel validates without Detailed | The kernel validates non-empty values against #config where they are applied, same name | no, byte for byte (checked) |
| Render does not re-validate | Render performs no second validation pass, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| When values are non-empty, the kernel SHALL validate them against the Module's `#config` schema... | The kernel validates non-empty values against #config where they are applied | verbatim |
| Values are applied inside a CUE build: `AcquireInstanceFromDir` renders the unified sources int... | Both instance verbs check the sources against #config at their own positions | SHALL added: the old sentence has no keyword |
| `Kernel.Render` SHALL NOT perform a second validation pass: the render build imports the instan... | Render performs no second validation pass | verbatim |

**Values enter as sources**: form c2 (REMOVED plus ADDED); old body 628 characters; new bodies 433, 194.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A single pre-unified value is a one-element source list | User values reach the kernel through exactly three public inputs, same name | no, byte for byte (checked) |
| ProcessModuleInstance takes a single value | The kernel accepts no cue.Value as a values argument, same name | no, byte for byte (checked) |
| ValidateConfigDetailed takes a slice of Source | User values reach the kernel through exactly three public inputs, same name | no, byte for byte (checked) |
| Extra values enter the instance build | User values reach the kernel through exactly three public inputs, same name | no, byte for byte (checked) |
| Synthesized values are sources | User values reach the kernel through exactly three public inputs, same name | no, byte for byte (checked) |
| Empty values is the zero value | User values reach the kernel through exactly three public inputs, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| User values SHALL reach the kernel through exactly three public inputs, each carrying values as... | User values reach the kernel through exactly three public inputs | verbatim |
| The kernel SHALL NOT accept `[]cue.Value` or a bare `cue.Value` as a values argument on any pub... | The kernel accepts no cue.Value as a values argument | verbatim |

**A held Render output keeps its build alive**: form c2 (REMOVED plus ADDED); old body 706 characters; new bodies 327, 417, 234.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| The documentation states the holder-bounded rule | ADR-005 and the package documentation state the holder-bounded rule, same name | no, byte for byte (checked) |
| The move to bytes is recorded | ADR-005 records that Render output moves to bytes before GA, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "The kernel retains no built value" in "The kernel retains no built value between calls"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The kernel SHALL retain no built value between calls. | The kernel retains no built value between calls | verbatim |
| Each `*kernel.Compiled` that `Render` returns carries a `cue.Value` into that render's build, s... | The kernel retains no built value between calls | verbatim |
| Each `*kernel.Compiled` that `Render` returns carries a `cue.Value` into that render's build, s... | ADR-005 and the package documentation state the holder-bounded rule | verbatim |
| ADR-005 and the `opm/kernel` package documentation SHALL state this rule and SHALL NOT state th... | ADR-005 and the package documentation state the holder-bounded rule | verbatim |
| ADR-005 SHALL record that Render output moves from `cue.Value` to bytes before GA at the latest... | ADR-005 records that Render output moves to bytes before GA | verbatim |
| The bytes deadline is enhancement 0021 ("A beta line reaches GA only when its exit criteria hol... | ADR-005 records that Render output moves to bytes before GA | verbatim |

**The kernel package doc renders its verb list and examples**: form c2 (REMOVED plus ADDED); old body 606 characters; new bodies 309, 296.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| go doc renders the Surface list as a list | The package documentation renders its verb list and examples as a list and code blocks, same name | no, byte for byte (checked) |
| go doc renders each example as code | The package documentation renders its verb list and examples as a list and code blocks, same name | no, byte for byte (checked) |
| Flattening the Surface list fails the guard test | A test parses the package documentation and fails when a list or code block is flattened, same name | no, byte for byte (checked) |
| Flattening a code example fails the guard test | A test parses the package documentation and fails when a list or code block is flattened, same name | no, byte for byte (checked) |
| Flattening one loop of an example fails the guard test | A test parses the package documentation and fails when a list or code block is flattened, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| The `opm/kernel` package documentation SHALL present the `Surface` section's verbs as a Go doc ... | The package documentation renders its verb list and examples as a list and code blocks | verbatim |
| When parsed with `go/doc` and `go/doc/comment`, the package doc SHALL yield a `*comment.List` a... | A test parses the package documentation and fails when a list or code block is flattened | verbatim |
| A test in `opm/kernel` SHALL parse the package doc this way and SHALL fail when the list or any... | A test parses the package documentation and fails when a list or code block is flattened | verbatim |

**SynthesizeInstance is documented as the typed-input entry point**: form c2 (REMOVED plus ADDED); old body 853 characters; new bodies 340, 416, 195.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Documentation directs callers to the kernel method | The documentation names SynthesizeInstance as the typed-input entry point, same name | no, byte for byte (checked) |
| SynthesizeInstance godoc names its directory mirror and module sources | The documentation names SynthesizeInstance as the typed-input entry point, same name | no, byte for byte (checked) |
| Pinned kernel synthesizes without touching the schema cache | The documentation states which core release the synthesized package imports, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "No helper-level composition is presented" in "The documentation presents no helper-level composition"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The package documentation and the `Kernel.SynthesizeInstance` godoc SHALL state that `Synthesiz... | The documentation names SynthesizeInstance as the typed-input entry point | verbatim |
| The documentation SHALL state that the synthesized package imports `core` at the major of the k... | The documentation states which core release the synthesized package imports | subject copied from the first sentence |
| The documentation SHALL NOT present a helper-level composition that reaches synthesis or instan... | The documentation presents no helper-level composition | subject copied from the first sentence |

**Front-door docs list the kernel's verbs and packages as built**: form c2 (REMOVED plus ADDED); old body 775 characters; new bodies 329, 363, 215.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Principle III names the verbs as built | CONSTITUTION Principle III lists the Kernel verbs as built, same name | no, byte for byte (checked) |
| The pipeline ends in kernel.Compiled | A front-door pipeline ends in kernel.Compiled, same name | no, byte for byte (checked) |
| The README layout names only real packages | The README layout lists only packages that exist, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| Where `CONSTITUTION.md` Principle III lists the `Kernel` verbs, it SHALL list them as acquire, ... | CONSTITUTION Principle III lists the Kernel verbs as built | verbatim |
| Where it shows a pipeline, the pipeline SHALL show acquisition or synthesis producing the artif... | A front-door pipeline ends in kernel.Compiled | referent copied from the first sentence |
| None of these SHALL name a load or process verb, a separate schema-validation stage, or a packa... | CONSTITUTION Principle III lists the Kernel verbs as built | verbatim |
| None of these SHALL name a load or process verb, a separate schema-validation stage, or a packa... | A front-door pipeline ends in kernel.Compiled | verbatim |
| Where `README.md` carries a layout block, it SHALL list only packages that exist under `opm/`, ... | The README layout lists only packages that exist | verbatim |

**Kernel verbs check cancellation at entry and between stages**: form c2 (REMOVED plus ADDED); old body 1246 characters; new bodies 454, 449, 376, 491, 459, 254, 351.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A cancelled context stops each directory verb | Verbs check their context between stages, same name | no, byte for byte (checked) |
| A cancelled context stops synthesis | Verbs check their context between stages, same name | no, byte for byte (checked) |
| Argument errors come first | Verbs check their context between stages, same name | no, byte for byte (checked) |
| A fetch served from the cache still observes cancellation | A cancellation inside the registry fetch is still a context error, same name | no, byte for byte (checked) |
| A cancellation during a stage lands at the next check | A running load or build is not interrupted, same name | no, byte for byte (checked) |
| The godoc states where cancellation lands | A running load or build is not interrupted, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "Registry verbs and Render check the context" in "The registry acquire verbs and Render check their context"
- "A verb checks its context after the package is built" in "Verbs check their context after the build"
- "A done context returns its own error and no artifact" in "A cancelled check of a directory or synthesis verb returns the context's own error and no artifact"
- "A done check of a registry acquire verb or Render returns its own error and no artifact" in "A cancelled check of a registry acquire verb or Render returns the context's own error and no artifact"

| Old clause | New requirement | Change |
| --- | --- | --- |
| `Kernel.AcquireModuleFromDir`, `Kernel.AcquireCatalogFromDir`, `Kernel.AcquirePlatformFromDir`,... | Verbs check their context between stages; Verbs check their context after the build | the sentence and the whole Source line do not fit together in 500 characters: cut at the list boundary ", after the package is built", the lead repeated, the last "," of the first part becomes "."; every word of each part is verbatim |
| The registry acquire verbs SHALL check it after the registry fetch returns and after the packag... | The registry acquire verbs and Render check their context | "it" becomes "their context" for the registry acquire verbs and "its context" for `Kernel.Render`; "their context" is copied from the first sentence of the old body |
| When one of these checks finds the context done, the verb SHALL return the context's own error ... | A cancelled check of a directory or synthesis verb returns the context's own error and no artifact; A cancelled check of a registry acquire verb or Render returns the context's own error and no artifact | verbatim; the sentence is repeated once for each group of checks, each with its pointer, because the pointer to three requirements and the whole Source line do not fit one body |
| A cancellation observed inside the registry fetch SHALL still satisfy `errors.Is(err, context.C... | A cancellation inside the registry fetch is still a context error | verbatim |
| A running `cue/load` or build SHALL NOT be interrupted; cancellation lands at the next stage bo... | A running load or build is not interrupted | verbatim |
| The `opm/kernel` package doc SHALL say so. | A running load or build is not interrupted | verbatim |
| Cancellation inside a stage is outside this requirement. | A running load or build is not interrupted | verbatim |
| Source: enhancement 0009 ("The execution half owns cancellation inside a stage and introduces i... | every block of this requirement | the whole line, with "as revised on 2026-10-03", in all seven blocks |
| (added words) | A cancelled check of a directory or synthesis verb returns the context's own error and no artifact; A cancelled check of a registry acquire verb or Render returns the context's own error and no artifact | pointer for 'these checks' to the exact headers |

**Each runtime contract has one home**: form b (MODIFIED parent plus ADDED); old body 1191 characters; new bodies 461, 204, 247, 432.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| The render contract is stated once | Each runtime contract has one home, same name | no, byte for byte (checked) |
| The env-override rule is stated in its homes | Each runtime contract has one home, same name | no, byte for byte (checked) |
| Published doc comments carry no ADR pointer | Each runtime contract has one home, same name | no, byte for byte (checked) |
| No session-local citation | Each runtime contract has one home, same name | no, byte for byte (checked) |
| A walkthrough decision resolves through ADR-013 | Each runtime contract has one home, same name | no, byte for byte (checked) |
| No walkthrough id without ADR-013 | Each runtime contract has one home, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "A published doc comment has no ADR pointer" in "A published doc comment carries no ADR pointer"
- "A session-local decision is not cited as a source" in "No committed file cites a session-local decision as a source"
- "A committed file states the rule and cites a source a reader can open" in "A committed file cites a source a reader can open"

| Old clause | New requirement | Change |
| --- | --- | --- |
| Each runtime contract of the library SHALL be stated in the doc comment of the package, type or... | Each runtime contract has one home | verbatim |
| `README.md`, `AGENTS.md` and `docs/getting-started.md` SHALL link to a contract's home instead ... | Each runtime contract has one home | verbatim |
| A doc comment that the docs bundle publishes (any exported package under `opm/`) SHALL NOT carr... | A published doc comment carries no ADR pointer | verbatim |
| No committed file outside `openspec/changes/` SHALL cite a session-local decision, one recorded... | No committed file cites a session-local decision as a source | verbatim list part (tail '.') |
| No committed file outside `openspec/changes/` SHALL cite a session-local decision, one recorded... | A committed file cites a source a reader can open | keyword and subject: the old clause continues the sentence about a committed file |
| Source: ADR-013, decision c4. | Each runtime contract has one home | verbatim |
| Source: ADR-013, decision c4. | A published doc comment carries no ADR pointer | verbatim |
| Source: ADR-013, decision c4. | No committed file cites a session-local decision as a source | verbatim |
| Source: ADR-013, decision c4. | A committed file cites a source a reader can open | verbatim |

**One registry client per Kernel**: form c2 (REMOVED plus ADDED); old body 2131 characters; new bodies 334, 356, 402, 362, 179, 423, 114, 245, 180.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Operations on one Kernel build one client | A Kernel hands its registry client to registry acquisition, same name | no, byte for byte (checked) |
| A file-backed values source resolves through the client | A Kernel hands its registry client to synthesis, values sources and the render build, same name | no, byte for byte (checked) |
| Construction builds no client | The registry client is built on first use and its settings are read then, same name | no, byte for byte (checked) |
| A failed construction is retried | A failure to build the client is returned and not kept, same name | no, byte for byte (checked) |
| A construction failure keeps its wording | A failure to build the client is returned and not kept, same name | no, byte for byte (checked) |
| A transient fetch failure is not remembered | Each operation wraps the client in a module cache of its own, same name | no, byte for byte (checked) |
| A cancelled fetch is not remembered | Each operation wraps the client in a module cache of its own, same name | no, byte for byte (checked) |
| The cache directory is read for each operation | Each operation wraps the client in a module cache of its own, same name | no, byte for byte (checked) |
| Concurrent operations share the client | A Kernel hands its registry client to registry acquisition, same name | no, byte for byte (checked) |
| The schema loader keeps its own client | The registry client does not replace the schema cache's loader, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "Directory acquisition uses the one client" in "A Kernel hands its registry client to directory acquisition"
- "A failed registry call drops the client" in "A failed registry call drops the shared client"
- "No exported signature changes for the client" in "The helper platformmodule registry stays caller-built and no exported signature changes"

| Old clause | New requirement | Change |
| --- | --- | --- |
| A `Kernel` SHALL own one registry client, meaning the resolver and the OCI transport behind it ... | A Kernel hands its registry client to registry acquisition | verbatim part |
| A `Kernel` SHALL own one registry client, meaning the resolver and the OCI transport behind it ... | A Kernel hands its registry client to registry acquisition | verbatim list part (tail '.') |
| A `Kernel` SHALL own one registry client, meaning the resolver and the OCI transport behind it ... | A Kernel hands its registry client to directory acquisition | verbatim part |
| A `Kernel` SHALL own one registry client, meaning the resolver and the OCI transport behind it ... | A Kernel hands its registry client to directory acquisition | verbatim list part (tail '.') |
| A `Kernel` SHALL own one registry client, meaning the resolver and the OCI transport behind it ... | A Kernel hands its registry client to synthesis, values sources and the render build | verbatim part |
| A `Kernel` SHALL own one registry client, meaning the resolver and the OCI transport behind it ... | A Kernel hands its registry client to synthesis, values sources and the render build | verbatim list part (tail '.') |
| The client SHALL be built from the kernel's registry mapping (`WithRegistry`, else the process ... | The registry client is built on first use and its settings are read then | referent copied from the first sentence |
| A failure to build the client SHALL be returned to the operation that needed it and SHALL NOT b... | A failure to build the client is returned and not kept | referent copied from the first sentence |
| Each operation SHALL wrap the client in a module cache of its own, over the cache directory rea... | Each operation wraps the client in a module cache of its own | referent copied from the first sentence |
| A registry call that fails SHALL also drop the shared client, so that the next operation builds... | A failed registry call drops the shared client | referent copied from the first sentence |
| The registry mapping and the credentials configuration SHALL be read when the client is built, ... | The registry client is built on first use and its settings are read then | verbatim |
| The client SHALL NOT replace the schema cache's own loader: an `OCILoader` is configured on its... | The registry client does not replace the schema cache's loader | referent copied from the first sentence |
| The opt-in `helper/platformmodule` registry stays caller-built. | The helper platformmodule registry stays caller-built and no exported signature changes | verbatim |
| No exported signature SHALL change for this: the client is not an option, a parameter or an exp... | The helper platformmodule registry stays caller-built and no exported signature changes | referent copied from the first sentence |

**Instance verbs refuse an unset required config value**: form b (MODIFIED parent plus ADDED); old body 3121 characters; new bodies 308, 323, 447, 423, 348, 387, 376, 455, 418, 298, 359, 394, 470.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Synthesis refuses an unread required value | Instance verbs refuse an unset required config value, same name | no, byte for byte (checked) |
| Directory acquisition refuses an unread required value | Instance verbs refuse an unset required config value, same name | no, byte for byte (checked) |
| A required field marker is refused when absent | Instance verbs refuse an unset required config value, same name | no, byte for byte (checked) |
| A field declared `_` is named by its path | Instance verbs refuse an unset required config value, same name | no, byte for byte (checked) |
| Optional and defaulted fields are not required | Instance verbs refuse an unset required config value, same name | no, byte for byte (checked) |
| The refused set matches ValidateConfigDetailed | Instance verbs refuse an unset required config value, same name | no, byte for byte (checked) |
| Instances refused before keep their error | Instance verbs refuse an unset required config value, same name | no, byte for byte (checked) |
| A value the built spec refuses is reported once | Instance verbs refuse an unset required config value, same name | no, byte for byte (checked) |
| A value a component reads is named at its field | Instance verbs refuse an unset required config value, same name | no, byte for byte (checked) |
| A value a catalog resource reads is named at its field | Instance verbs refuse an unset required config value, same name | no, byte for byte (checked) |
| Every unset value is named, read or not, nested or not | Instance verbs refuse an unset required config value, same name | no, byte for byte (checked) |
| A component defect waits for complete values | Instance verbs refuse an unset required config value, same name | no, byte for byte (checked) |
| A written bare type hides the component findings it causes | Instance verbs refuse an unset required config value, same name | no, byte for byte (checked) |
| Other values defects keep their text | Instance verbs refuse an unset required config value, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "The effective values are the built spec's values" in "The effective values are the built spec's values"
- "A required value is one that concreteness refuses" in "A required value is one that concreteness refuses"
- "Both verbs run both checks in one step" in "Both concreteness checks run in one instance processing step"
- "The refusal is CUE's own error tree" in "The refusal is CUE's own error tree framed with the verb and the instance"
- "The refusal holds the findings of the values only" in "The refusal holds the value findings of the built-spec check"
- "The refusal names every other unset value" in "The refusal names every other unset required value at values.<field>"
- "A finding carries the position of its declaration" in "A finding carries the position of the field's #config declaration"
- "Component findings are left out of the refusal" in "Findings outside values are left out of the refusal"
- "Positions decide nothing" in "The kernel does not decide from positions whether an unset value causes a finding"
- "A caller finds the tree with errors.As" in "A caller identifies the refusal by CUE's error type"
- "SynthesizeInstance refuses what ValidateConfigDetailed refuses" in "SynthesizeInstance refuses what ValidateConfigDetailed refuses"
- "SynthesizeInstance may refuse more" in "SynthesizeInstance may refuse more only through the concreteness check on the built spec"

| Old clause | New requirement | Change |
| --- | --- | --- |
| `SynthesizeInstance` and `AcquireInstanceFromDir` SHALL refuse an instance whose effective valu... | Instance verbs refuse an unset required config value | verbatim |
| The effective values are the built spec's `values`: the package's own `values` plus the trailin... | The effective values are the built spec's values | SHALL added: the old sentence has no keyword |
| A required value is one that `cue.Concrete(true)` refuses on those values unified with `#config... | A required value is one that concreteness refuses | SHALL added: the old sentence has no keyword |
| An optional field (`foo?:`) and a field with a default are not required. | A required value is one that concreteness refuses | verbatim |
| Both concreteness checks, the one on the built spec and the one on the values unified with `#co... | Both concreteness checks run in one instance processing step | verbatim |
| The refusal SHALL be CUE's own error tree, framed `Kernel.<Verb>: instance "<name>": not fully ... | The refusal is CUE's own error tree framed with the verb and the instance | verbatim |
| When the values leave no required value unset, the refusal SHALL be the error of the built-spec... | The refusal is CUE's own error tree framed with the verb and the instance | verbatim |
| When they leave one or more unset, the refusal SHALL hold findings about the values only: | The refusal holds the value findings of the built-spec check | "they" becomes "the values", the subject of the sentence before it in the old body |
| When they leave one or more unset, the refusal SHALL hold findings about the values only: | The refusal names every other unset required value at values.<field> | "they" becomes "the values", the subject of the sentence before it in the old body |
| 1. every finding of the built-spec check at a path under `values` (a value the instance's value... | The refusal holds the value findings of the built-spec check | verbatim |
| 2. one finding for every other unset required value, at the path `values.<field>` (the full pat... | The refusal names every other unset required value at values.<field> | verbatim |
| A finding SHALL carry the position of the field's `#config` declaration wherever CUE records on... | A finding carries the position of the field's #config declaration | verbatim |
| A path SHALL be reported once. | A finding carries the position of the field's #config declaration | verbatim |
| Every finding of the built-spec check outside `values` SHALL be left out of that refusal: the c... | Findings outside values are left out of the refusal | verbatim |
| Such a field that is still incomplete once the values are complete is then refused with the bui... | Findings outside values are left out of the refusal | verbatim |
| The kernel SHALL NOT decide from the positions of a finding whether an unset value causes it. | The kernel does not decide from positions whether an unset value causes a finding | verbatim |
| A caller identifies the refusal as it did before this requirement changed: `errors.As` with CUE... | A caller identifies the refusal by CUE's error type | SHALL added: the old sentence has no keyword |
| When a required value is unset, every finding of the tree is a values finding. | The kernel does not decide from positions whether an unset value causes a finding | verbatim |
| For the same module and the same single values source, `SynthesizeInstance` SHALL refuse every ... | SynthesizeInstance refuses what ValidateConfigDetailed refuses | verbatim list part (tail '.') |
| For the same module and the same single values source, `SynthesizeInstance` SHALL refuse every ... | SynthesizeInstance may refuse more only through the concreteness check on the built spec | verbatim list part (tail ';') |
| For the same module and the same single values source, `SynthesizeInstance` SHALL refuse every ... | SynthesizeInstance may refuse more only through the concreteness check on the built spec | verbatim list part (tail '.') |
| Source: library#211, settled by the owner in the pull request that closes it; opm-operator#258 ... | Instance verbs refuse an unset required config value | verbatim |
| Source: library#211, settled by the owner in the pull request that closes it; opm-operator#258 ... | The effective values are the built spec's values | verbatim |
| Source: library#211, settled by the owner in the pull request that closes it; opm-operator#258 ... | A required value is one that concreteness refuses | verbatim |
| Source: library#211, settled by the owner in the pull request that closes it; opm-operator#258 ... | Both concreteness checks run in one instance processing step | verbatim |
| Source: library#211, settled by the owner in the pull request that closes it; opm-operator#258 ... | The refusal is CUE's own error tree framed with the verb and the instance | verbatim |
| Source: library#211, settled by the owner in the pull request that closes it; opm-operator#258 ... | The refusal holds the value findings of the built-spec check | verbatim |
| Source: library#211, settled by the owner in the pull request that closes it; opm-operator#258 ... | The refusal names every other unset required value at values.<field> | verbatim |
| Source: library#211, settled by the owner in the pull request that closes it; opm-operator#258 ... | A finding carries the position of the field's #config declaration | verbatim |
| Source: library#211, settled by the owner in the pull request that closes it; opm-operator#258 ... | Findings outside values are left out of the refusal | verbatim |
| Source: library#211, settled by the owner in the pull request that closes it; opm-operator#258 ... | The kernel does not decide from positions whether an unset value causes a finding | verbatim |
| Source: library#211, settled by the owner in the pull request that closes it; opm-operator#258 ... | A caller identifies the refusal by CUE's error type | verbatim |
| Source: library#211, settled by the owner in the pull request that closes it; opm-operator#258 ... | SynthesizeInstance refuses what ValidateConfigDetailed refuses | verbatim |
| (added words) | A finding carries the position of the field's #config declaration | pointer for the subject: the old sentence is part of that list item |
| (added words) | SynthesizeInstance may refuse more only through the concreteness check on the built spec | none now: the block carries the whole old Source line (the earlier clause-only line is gone) |

### Ledger: artifact-types

**Uniform Artifact Shape**: form c2 (REMOVED plus ADDED); old body 632 characters; new bodies 385, 251.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Module type shape | Every artifact type is a struct with exactly three exported fields, same name | no, byte for byte (checked) |
| ModuleInstance type shape | The Metadata, Package and Source fields hold a projection, the source of truth and the staged source, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| Every OPM artifact type accepted by the kernel (`module.Module`, `module.Instance`, `platform.P... | Every artifact type is a struct with exactly three exported fields | verbatim |
| The `Metadata` pointer holds a decoded ergonomic projection; the `Package` field carries the so... | The Metadata, Package and Source fields hold a projection, the source of truth and the staged source | SHALL added: the old sentence has no keyword |
| No artifact struct SHALL carry an API version field (`schema-dispatch`, "Module, Instance, Plat... | Every artifact type is a struct with exactly three exported fields | verbatim |

**Kernel Artifact Type Set**: form c2 (REMOVED plus ADDED); old body 760 characters; new bodies 213, 224, 327.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| No top-level ModuleDebug type | ModuleDebug is not a kernel artifact type, same name | no, byte for byte (checked) |
| debugValues accessible via Module.Package | ModuleDebug is not a kernel artifact type, same name | no, byte for byte (checked) |
| Documentation explicitly retires the construct | ModuleDebug is not a kernel artifact type, same name | no, byte for byte (checked) |
| The fourth type is a read, never a render | Catalog is admitted on the terms of ADR-009 and a catalog is never rendered, same name | no, byte for byte (checked) |
| The enumerated set is stated once and agrees everywhere | The kernel accepts exactly four artifact types, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| The kernel SHALL accept exactly four artifact types: `Module`, `ModuleInstance`, `Platform` and... | The kernel accepts exactly four artifact types | verbatim |
| `#ModuleDebug` SHALL NOT be a kernel artifact type. | ModuleDebug is not a kernel artifact type | verbatim |
| Debug values are carried as a `debugValues` field within `Module.Package`; whether they partici... | ModuleDebug is not a kernel artifact type | verbatim |
| `Catalog` is admitted on the terms ADR-009 records and on no others: the kernel acquires it, re... | Catalog is admitted on the terms of ADR-009 and a catalog is never rendered | SHALL added: the old sentence has no keyword |
| A catalog is never rendered and never executed; the transformers it carries reach a render only... | Catalog is admitted on the terms of ADR-009 and a catalog is never rendered | verbatim |
| The set SHALL NOT grow again without the candidate meeting, in writing, the four-part test ADR-... | The kernel accepts exactly four artifact types | verbatim |

**Artifacts carry their staged source**: form c2 (REMOVED plus ADDED); old body 1020 characters; new bodies 374, 126, 208, 309.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Overlay entries are bytes | module.Source describes a staged source tree in two modes, same name | no, byte for byte (checked) |
| Value-constructed instance has no source | Every instance the kernel returns carries a non-nil Source, same name | no, byte for byte (checked) |
| Value-constructed module has no source | A module's Source is populated by acquisition and nil for a bare value, same name | no, byte for byte (checked) |
| Directory-acquired module has an overlay source | A module's Source is populated by acquisition and nil for a bare value, same name | no, byte for byte (checked) |
| On-disk source mode | module.Source describes a staged source tree in two modes, same name | no, byte for byte (checked) |
| Synth gate unchanged | A module's Source is populated by acquisition and nil for a bare value, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "Pkg names the package directory" in "module.Source carries a Pkg field"

| Old clause | New requirement | Change |
| --- | --- | --- |
| `module.Source` SHALL describe the staged source tree an artifact was loaded or synthesized fro... | module.Source describes a staged source tree in two modes | verbatim |
| `Overlay` SHALL map each absolute file path to that file's contents as bytes. | module.Source describes a staged source tree in two modes | verbatim |
| `module.Source` SHALL carry a `Pkg` field naming the package directory relative to `Root`; empt... | module.Source carries a Pkg field | verbatim |
| `module.Instance` SHALL expose `Source *module.Source`; every instance the kernel returns carri... | Every instance the kernel returns carries a non-nil Source | verbatim |
| `module.Module.Source` SHALL be populated in overlay mode by both `AcquireModuleFromRegistry` a... | A module's Source is populated by acquisition and nil for a bare value | verbatim |

**Instance acquisition from a directory**: form c2 (REMOVED plus ADDED); old body 1752 characters; new bodies 271, 386, 79, 176, 61, 123, 358, 231, 409, 112.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Acquired instance carries its source | An acquired instance's Source describes the directory, same name | no, byte for byte (checked) |
| Acquired subpackage names its module root | A directory with no enclosing module is its own root, same name | no, byte for byte (checked) |
| Validation failures propagate | AcquireInstanceFromDir loads a package through the shape gate and processes it, same name | no, byte for byte (checked) |
| Loader failures propagate | AcquireInstanceFromDir loads a package through the shape gate and processes it, same name | no, byte for byte (checked) |
| Registry mapping is the kernel's | The registry mapping for an instance's imports is the kernel's, same name | no, byte for byte (checked) |
| Extra values layer onto the package | Values sources are unified in order, same name | no, byte for byte (checked) |
| Layered instance renders | The unified values are rendered as a values file beside the package's files, same name | no, byte for byte (checked) |
| Conflicting extra values fail at acquisition | The package is built in one pass through the instance shape gate, same name | no, byte for byte (checked) |
| No option type for values | Values sources are unified in order, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "Without values the overlay is nil" in "An instance acquired without values has an on-disk source"
- "A layered instance carries an overlay source" in "A layered instance's Source is in overlay mode"
- "Acquisition writes nothing and fills nothing from Go" in "Instance acquisition writes nothing into the caller's directory"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The kernel SHALL expose `AcquireInstanceFromDir(ctx, dir, values ...Source)`, which loads a `#M... | AcquireInstanceFromDir loads a package through the shape gate and processes it | verbatim list part (tail '.') |
| The kernel SHALL expose `AcquireInstanceFromDir(ctx, dir, values ...Source)`, which loads a `#M... | An acquired instance's Source describes the directory | verbatim part |
| The kernel SHALL expose `AcquireInstanceFromDir(ctx, dir, values ...Source)`, which loads a `#M... | An acquired instance's Source describes the directory | verbatim list part (tail '.') |
| A directory with no enclosing module is its own root with an empty `Pkg`. | A directory with no enclosing module is its own root | SHALL added: the old sentence has no keyword |
| The registry mapping used for the instance's imports SHALL be the kernel's (`WithRegistry`), ap... | The registry mapping for an instance's imports is the kernel's | verbatim |
| With no values, `Source.Overlay` is nil (on-disk mode). | An instance acquired without values has an on-disk source | SHALL added: the old sentence has no keyword |
| With one or more values sources (the `Source` type `ValidateConfigDetailed` accepts), the kerne... | Values sources are unified in order | verbatim part |
| With one or more values sources (the `Source` type `ValidateConfigDetailed` accepts), the kerne... | Values sources are unified in order | verbatim list part (tail '.') |
| With one or more values sources (the `Source` type `ValidateConfigDetailed` accepts), the kerne... | The unified values are rendered as a values file beside the package's files | verbatim part |
| With one or more values sources (the `Source` type `ValidateConfigDetailed` accepts), the kerne... | The unified values are rendered as a values file beside the package's files | verbatim list part (tail '.') |
| With one or more values sources (the `Source` type `ValidateConfigDetailed` accepts), the kerne... | The package is built in one pass through the instance shape gate | verbatim part |
| With one or more values sources (the `Source` type `ValidateConfigDetailed` accepts), the kerne... | The package is built in one pass through the instance shape gate | verbatim list part (tail '.') |
| The returned instance's `Source` SHALL then be overlay mode: same `Root` and `Pkg`, `Overlay` c... | A layered instance's Source is in overlay mode | the condition "With one or more values sources (the `Source` type `ValidateConfigDetailed` accepts)," is copied from the sentence before it in the old body; the rest is verbatim |
| The kernel MUST NOT write into the caller's directory and MUST NOT fill values into the evaluat... | Instance acquisition writes nothing into the caller's directory | verbatim |
| (added words) | A layered instance's Source is in overlay mode | pointer for 'then' |

**Internal call sites use schema paths**: form c2 (REMOVED plus ADDED); old body 554 characters; new bodies 361, 197.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Render build reads artifacts by import | The render path reads no artifact sub-value in Go, same name | no, byte for byte (checked) |
| Instance processing uses schema paths | Kernel-internal call sites read an artifact's Package through the path inventory, same name | no, byte for byte (checked) |
| Metadata reads go through the metadata path | Kernel-internal call sites read an artifact's Package through the path inventory, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| Every kernel-internal Go call site that reads a sub-value of an artifact's `Package` SHALL read... | Kernel-internal call sites read an artifact's Package through the path inventory | verbatim |
| The render path reads no artifact sub-value in Go: the instance and the platform enter the rend... | The render path reads no artifact sub-value in Go | SHALL added: the old sentence has no keyword |

**Module acquisition from a directory**: form c2 (REMOVED plus ADDED); old body 1081 characters; new bodies 252, 402, 239, 157, 174, 54.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Acquired module carries an overlay source | An acquired module's Source has a Root and a Pkg, same name | no, byte for byte (checked) |
| Acquired module synthesizes like a registry module | An acquired module can be passed to SynthesizeInstance without further staging, same name | no, byte for byte (checked) |
| Shape-gate failures propagate | AcquireModuleFromDir loads a module package through the shape gate and constructs the typed module, same name | no, byte for byte (checked) |
| Frontends stop staging module sources themselves | An acquired module can be passed to SynthesizeInstance without further staging, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "The overlay holds every cue file under Root" in "An acquired module's Source Overlay holds every cue file under Root"
- "The module's imports use the kernel's mapping" in "The registry mapping for a module's imports is the kernel's"
- "Module acquisition writes nothing" in "Module acquisition writes nothing into the caller's directory"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The kernel SHALL expose `AcquireModuleFromDir(ctx, dir) (*module.Module, error)`, which loads a... | AcquireModuleFromDir loads a module package through the shape gate and constructs the typed module | verbatim list part (tail '.') |
| The kernel SHALL expose `AcquireModuleFromDir(ctx, dir) (*module.Module, error)`, which loads a... | An acquired module's Source has a Root and a Pkg | verbatim part |
| The kernel SHALL expose `AcquireModuleFromDir(ctx, dir) (*module.Module, error)`, which loads a... | An acquired module's Source has a Root and a Pkg | verbatim list part (tail '.') |
| The kernel SHALL expose `AcquireModuleFromDir(ctx, dir) (*module.Module, error)`, which loads a... | An acquired module's Source Overlay holds every cue file under Root | verbatim part |
| The kernel SHALL expose `AcquireModuleFromDir(ctx, dir) (*module.Module, error)`, which loads a... | An acquired module's Source Overlay holds every cue file under Root | verbatim part |
| The kernel SHALL expose `AcquireModuleFromDir(ctx, dir) (*module.Module, error)`, which loads a... | An acquired module's Source Overlay holds every cue file under Root | verbatim list part (tail '.') |
| The acquired module SHALL satisfy the synthesis precondition (`Module.HasSource()` true) so it ... | An acquired module can be passed to SynthesizeInstance without further staging | verbatim |
| The registry mapping used for the module's imports SHALL be the kernel's (`WithRegistry`), appl... | The registry mapping for a module's imports is the kernel's | verbatim |
| The kernel MUST NOT write into the caller's directory. | Module acquisition writes nothing into the caller's directory | verbatim |

**Acquisition shape gate**: form c2 (REMOVED plus ADDED); old body 2434 characters; new bodies 458, 421, 234, 344, 458, 359, 129, 195.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Wrong artifact type rejected | The gate rejects a non-struct root, a wrong kind and more than one instance, same name | no, byte for byte (checked) |
| Missing identity field rejected | The gate rejects a package whose required identity field is absent or not concrete, same name | no, byte for byte (checked) |
| Defaulted identity field rejected with the default named | A defaulted identity field is not concrete for the gate, same name | no, byte for byte (checked) |
| Instance embedding a non-module rejected | The required identity fields are those the schema never defaults, same name | no, byte for byte (checked) |
| Registry entry with no embedded catalog rejected | A registry entry with no embedded catalog fails as a missing required field, same name | no, byte for byte (checked) |
| Non-struct root rejected | The gate rejects a non-struct root, a wrong kind and more than one instance, same name | no, byte for byte (checked) |
| Conflicting package clauses rejected | The gate rejects a non-struct root, a wrong kind and more than one instance, same name | no, byte for byte (checked) |
| Registry and directory gate identically | A directory-acquired and a registry-acquired artifact are gated identically, same name | no, byte for byte (checked) |
| Sentinels live in one package | opm/errors exports the three gate sentinels, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "The gate runs after the build and judges structure only" in "A shape gate runs right after the package is built"

| Old clause | New requirement | Change |
| --- | --- | --- |
| Every kernel acquisition from a directory or a registry (`AcquireModuleFromDir`, `AcquireModule... | A shape gate runs right after the package is built | verbatim |
| The gate validates structural identity only; it SHALL NOT perform full schema validation of the... | A shape gate runs right after the package is built | verbatim |
| The gate SHALL reject a package when any of the following hold, returning an error that wraps t... | The gate rejects a non-struct root, a wrong kind and more than one instance | verbatim |
| The gate SHALL reject a package when any of the following hold, returning an error that wraps t... | The gate rejects a package whose required identity field is absent or not concrete | verbatim |
| - The built root value is not a struct, or the package load resolved other than exactly one ins... | The gate rejects a non-struct root, a wrong kind and more than one instance | verbatim |
| - The concrete `kind` literal does not match the artifact the verb acquires (`"Module"`, `"Modu... | The gate rejects a non-struct root, a wrong kind and more than one instance | verbatim |
| - A required identity field is absent or not concrete: wraps `ErrMissingRequiredField`. | The gate rejects a package whose required identity field is absent or not concrete | verbatim |
| Required identity fields are those the schema never defaults: | The required identity fields are those the schema never defaults | SHALL added: the old sentence has no keyword |
| - Module: `metadata.name`, `metadata.modulePath`, `metadata.version`. | The required identity fields are those the schema never defaults | verbatim |
| - Instance: `metadata.name`, `metadata.namespace`, and `#module` present with `#module.kind == ... | The required identity fields are those the schema never defaults | verbatim |
| - Platform: `metadata.name`, `type`, and every `#registry` entry complete under `cue.Concrete(t... | The required identity fields are those the schema never defaults | verbatim |
| Core derives an entry's `version` from its embedded catalog's stamped `metadata.version` (enhan... | A registry entry with no embedded catalog fails as a missing required field | SHALL added: the old sentence has no keyword; ", at the gate," follows "fails here" (the subject "the gate" is copied from the old requirement; "here" stays) |
| This is the refusal `single-build-render` relies on for a subscription-shaped platform. | A registry entry with no embedded catalog fails as a missing required field | verbatim |
| A required identity field declared as a disjunction with a default (for example `#VersionType /... | A defaulted identity field is not concrete for the gate | verbatim |
| When the gate rejects such a field, the error SHALL name the default arm's value and state that... | A defaulted identity field is not concrete for the gate | verbatim |
| A directory-acquired artifact and a registry-acquired artifact SHALL be gated identically and f... | A directory-acquired and a registry-acquired artifact are gated identically | verbatim |
| `opm/errors` SHALL export `ErrInvalidPackage`, `ErrWrongKind` and `ErrMissingRequiredField` so ... | opm/errors exports the three gate sentinels | verbatim |

**An overlay source can be written to a directory**: form b (MODIFIED parent plus ADDED); old body 696 characters; new bodies 264, 233, 196.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A fetched module is written to disk | An overlay source can be written to a directory, same name | no, byte for byte (checked) |
| On-disk source refused | An overlay source can be written to a directory, same name | no, byte for byte (checked) |
| Entry outside the root refused | An overlay source can be written to a directory, same name | no, byte for byte (checked) |
| Frontends stop copying fetched modules themselves | An overlay source can be written to a directory, same name | no, byte for byte (checked) |
| The render stage writes nothing for an overlay input | An overlay source can be written to a directory, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "WriteTo refuses before it writes" in "WriteTo validates before it writes"
- "WriteTo is the one overlay writer" in "WriteTo is the library's one overlay writer"

| Old clause | New requirement | Change |
| --- | --- | --- |
| `module.Source` SHALL expose `WriteTo(dir string) ([]string, error)`, which writes every entry ... | An overlay source can be written to a directory | verbatim |
| The method SHALL validate before it writes: a nil receiver, an on-disk source (`Overlay` nil, s... | WriteTo validates before it writes | subject copied from the first sentence |
| `WriteTo` SHALL be the library's one overlay writer, for frontends that need a fetched tree on ... | WriteTo is the library's one overlay writer | verbatim |

**Directory acquisition reads its tree once and builds from that read**: form c2 (REMOVED plus ADDED); old body 1291 characters; new bodies 338, 355, 327, 163, 126, 169.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A missing directory fails before any tree read | A directory acquire verb checks the directory and a missing path fails, same name | no, byte for byte (checked) |
| A file path is refused as not a directory | A directory acquire verb refuses a path that is not a directory, same name | no, byte for byte (checked) |
| Module and catalog are built from the overlay they carry | An overlay-mode verb reads the cue files once and builds from that overlay, same name | no, byte for byte (checked) |
| Values attribution reuses the authored read | A failed layered build is attributed from the overlay already read, same name | no, byte for byte (checked) |
| A module embedding a non-CUE file acquires from a directory | A package that embeds a non-CUE file acquires from a directory, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "Platform acquisition keeps the on-disk mode" in "Platform acquisition and instance acquisition without values keep the on-disk mode"

| Old clause | New requirement | Change |
| --- | --- | --- |
| Every directory acquire verb (`AcquireModuleFromDir`, `AcquireCatalogFromDir`, `AcquirePlatform... | A directory acquire verb checks the directory and a missing path fails | verbatim |
| Every directory acquire verb (`AcquireModuleFromDir`, `AcquireCatalogFromDir`, `AcquirePlatform... | A directory acquire verb refuses a path that is not a directory | verbatim |
| A path that does not exist SHALL fail with `accessing <label> directory "<absolute path>"` wrap... | A directory acquire verb checks the directory and a missing path fails | verbatim list part (tail '.') |
| A path that does not exist SHALL fail with `accessing <label> directory "<absolute path>"` wrap... | A directory acquire verb refuses a path that is not a directory | capital letter only |
| A verb that stamps its `Source` in overlay mode (`AcquireModuleFromDir`, `AcquireCatalogFromDir... | An overlay-mode verb reads the cue files once and builds from that overlay | verbatim |
| When the layered instance build fails, the kernel SHALL attribute the failure to the values sou... | A failed layered build is attributed from the overlay already read | verbatim |
| Platform acquisition and instance acquisition without values sources keep the on-disk mode thei... | Platform acquisition and instance acquisition without values keep the on-disk mode | SHALL added: the old sentence has no keyword |
| A package that embeds a non-CUE file stored beside it SHALL acquire from a directory as it does... | A package that embeds a non-CUE file acquires from a directory | verbatim |

**Module and platform constructors from cue.Value**: form c2 (REMOVED plus ADDED); old body 727 characters; new bodies 359, 49, 251, 137.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| NewModuleFromValue success path | Constructor helpers build module and platform artifacts from a cue.Value, same name | no, byte for byte (checked) |
| NewModuleFromValue with missing metadata | Constructor helpers build module and platform artifacts from a cue.Value, same name | no, byte for byte (checked) |
| NewModuleFromValue with unknown apiVersion | Constructor helpers build module and platform artifacts from a cue.Value, same name | no, byte for byte (checked) |
| NewPlatformFromValue hoists the platform type | Constructor helpers build module and platform artifacts from a cue.Value, same name | no, byte for byte (checked) |
| Constructors take no context owner | Constructor helpers build module and platform artifacts from a cue.Value, same name | no, byte for byte (checked) |
| No kernel constructor wrappers | The kernel does not wrap the constructors, same name | no, byte for byte (checked) |
| No instance constructor | No instance constructor is exported, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "A constructed artifact has no Source" in "A constructed artifact carries no Source"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The library SHALL provide constructor helpers that build the module and platform artifacts from... | Constructor helpers build module and platform artifacts from a cue.Value | verbatim |
| Each constructor SHALL take only the artifact value, decode `Metadata` from the value's `metada... | Constructor helpers build module and platform artifacts from a cue.Value | verbatim |
| The constructed artifact carries no `Source`. | A constructed artifact carries no Source | SHALL added: the old sentence has no keyword |
| The kernel SHALL NOT wrap these constructors: a frontend holding a value calls the package cons... | The kernel does not wrap the constructors | referent copied from the first sentence |
| No instance constructor SHALL be exported: an instance is produced only by the kernel's acquisi... | No instance constructor is exported | verbatim |

**Instance exposes its components, config schema, values and module metadata**: form c2 (REMOVED plus ADDED); old body 1538 characters; new bodies 270, 196, 197, 262, 274, 183, 321.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Components accessor | Instance exposes Components and Values, same name | no, byte for byte (checked) |
| No metadata-mirroring accessors | No accessor consults a version or mirrors a metadata field, same name | no, byte for byte (checked) |
| Config schema reachable on a well-formed instance | Instance exposes ConfigSchema, same name | no, byte for byte (checked) |
| Config schema is the zero value on a missing #module | ConfigSchema and Values return the zero value rather than an error, same name | no, byte for byte (checked) |
| Config schema is the zero value on a missing #config | ConfigSchema and Values return the zero value rather than an error, same name | no, byte for byte (checked) |
| Config schema on a nil instance is the zero value | ConfigSchema and Values return the zero value rather than an error, same name | no, byte for byte (checked) |
| Values accessor | Instance exposes Components and Values, same name | no, byte for byte (checked) |
| Values on a nil instance is the zero value | ConfigSchema and Values return the zero value rather than an error, same name | no, byte for byte (checked) |
| Module metadata of an acquired instance | Instance exposes ModuleMetadata, same name | no, byte for byte (checked) |
| Module metadata is nil when it cannot be read | ModuleMetadata returns nil when it cannot be read and decodes afresh, same name | no, byte for byte (checked) |
| Module metadata reads the package | ModuleMetadata returns nil when it cannot be read and decodes afresh, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "The render build reads no accessor" in "The render build reads none of the instance accessors"

| Old clause | New requirement | Change |
| --- | --- | --- |
| `*module.Instance` SHALL expose the following accessors: | Instance exposes Components and Values | verbatim |
| `*module.Instance` SHALL expose the following accessors: | Instance exposes ConfigSchema | verbatim |
| `*module.Instance` SHALL expose the following accessors: | Instance exposes ModuleMetadata | verbatim |
| - `Components()`: the instance's evaluated components value, definition fields included, read t... | Instance exposes Components and Values | verbatim |
| - `ConfigSchema()`: the embedded module's `#config`, read as `Package.LookupPath(schema.Module)... | Instance exposes ConfigSchema | verbatim |
| - `Values()`: the instance's merged values as evaluated, read through `corepath.Values`. | Instance exposes Components and Values | verbatim |
| - `ModuleMetadata()`: the metadata of the module the instance was built from, decoded from `Pac... | Instance exposes ModuleMetadata | verbatim |
| `ConfigSchema()` and `Values()` SHALL return the zero `cue.Value` (not an error) when the recei... | ConfigSchema and Values return the zero value rather than an error | verbatim |
| `ConfigSchema()` returns it as well when the instance carries no `#module` or the embedded modu... | ConfigSchema and Values return the zero value rather than an error | verbatim |
| `ModuleMetadata()` SHALL return nil when the receiver is `nil`, when the instance carries no `#... | ModuleMetadata returns nil when it cannot be read and decodes afresh | verbatim |
| Each call decodes afresh from `Package`; the returned struct is a copy, and mutating it does no... | ModuleMetadata returns nil when it cannot be read and decodes afresh | verbatim |
| No accessor SHALL consult a version or binding. | No accessor consults a version or mirrors a metadata field | verbatim |
| The instance SHALL expose no accessor that mirrors a single field of its own decoded `Metadata`... | No accessor consults a version or mirrors a metadata field | verbatim |
| The render build reads none of these accessors: the transformer context is projected by core (e... | The render build reads none of the instance accessors | SHALL added: the old sentence has no keyword; referent copied from the first sentence |

### Ledger: deps-cascade

**The cascade task reports its result by exit code**: form c2 (REMOVED plus ADDED); old body 806 characters; new bodies 282, 239, 419.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Nothing to move | The cascade task changes the working tree only and reports its result by exit code, same name | no, byte for byte (checked) |
| Resolver error | A failing step ends the cascade task with a non-zero code other than 3, same name | no, byte for byte (checked) |
| Dirty tree | The cascade task refuses a dirty working tree unless told otherwise, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| `task -x deps:cascade` SHALL move the library's upstream pins in the working tree only. | The cascade task changes the working tree only and reports its result by exit code | verbatim |
| It SHALL NOT commit, branch or push. | The cascade task changes the working tree only and reports its result by exit code | verbatim |
| It SHALL exit 0 when the working tree changed, 3 when nothing changed, and any other code on error. | The cascade task changes the working tree only and reports its result by exit code | verbatim |
| Source: workspace `RELEASING.md`, section "The receiver". | The cascade task changes the working tree only and reports its result by exit code | the whole Source line |
| It SHALL refuse a dirty working tree with exit 1 unless `CASCADE_ALLOW_DIRTY=1` is set, in which... | The cascade task refuses a dirty working tree unless told otherwise | subject copied from the first sentence |
| Source: workspace `RELEASING.md`, section "The receiver". | The cascade task refuses a dirty working tree unless told otherwise | the whole Source line |
| A failing resolver call, `cue mod get`, `cue mod tidy` or file edit SHALL end the task with a non... | A failing step ends the cascade task with a non-zero code other than 3 | verbatim |
| The task SHALL NOT fall back to an older or guessed version, and SHALL NOT discard the failing co... | A failing step ends the cascade task with a non-zero code other than 3 | verbatim |
| Every resolver call that decides a target SHALL run before the first file edit, so a resolver err... | A failing step ends the cascade task with a non-zero code other than 3 | verbatim |
| Source: workspace `RELEASING.md`, section "The receiver". | A failing step ends the cascade task with a non-zero code other than 3 | the whole Source line |

**Core moves with DefaultSchemaModule and is labelled for human review**: form c2 (REMOVED plus ADDED); old body 960 characters; new bodies 361, 458, 464, 376, 280.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A newer core is published | The core pin moves to the newest published release the resolver returns, same name | no, byte for byte (checked) |
| The resolver keeps core where it is | The core pin never moves backwards or across a major, stops at a hold and counts prereleases while it is a prerelease, same name | no, byte for byte (checked) |
| A frozen loader holds back the catalog | A frozen loader keeps the core pin and every later step uses that core, same name | no, byte for byte (checked) |
| The resolver warns of a new core major | The core pin never moves backwards or across a major, stops at a hold and counts prereleases while it is a prerelease, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "A core move is recorded in a warning" in "A core move records a warning that names it and need-human-review"
- "The core row carries the label" in "The pin report labels the core row need-human-review"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The task SHALL move the core pin `DefaultSchemaModule` in `opm/schema/loader.go` to the newest pu... | The core pin moves to the newest published release the resolver returns | verbatim |
| It SHALL edit only that string literal. | The core pin moves to the newest published release the resolver returns | verbatim |
| It SHALL NOT edit any other Go file. | The core pin moves to the newest published release the resolver returns | verbatim |
| Source: owner selection 9 ("Bot proposes, glue review", label `need-human-review`); workspace `RE... | The core pin moves to the newest published release the resolver returns | the whole Source line |
| The task SHALL move the core pin `DefaultSchemaModule` in `opm/schema/loader.go` to the newest pu... | The core pin never moves backwards or across a major, stops at a hold and counts prereleases while it is a... | verbatim |
| Prereleases SHALL count while the current pin is a prerelease. | The core pin never moves backwards or across a major, stops at a hold and counts prereleases while it is a... | verbatim |
| The pin SHALL NOT move backwards, SHALL NOT cross a major, and SHALL stop at an in-date `.cascade... | The core pin never moves backwards or across a major, stops at a hold and counts prereleases while it is a... | verbatim |
| Source: owner selection 9 ("Bot proposes, glue review", label `need-human-review`); workspace `RE... | The core pin never moves backwards or across a major, stops at a hold and counts prereleases while it is a... | the whole Source line |
| The task SHALL move the core pin `DefaultSchemaModule` in `opm/schema/loader.go` to the newest pu... | A frozen loader keeps the core pin and every later step uses that core | verbatim |
| When `.cascade-frozen` lists `opm/schema/loader.go` for `opmodel.dev/core@v2`, the pin SHALL stay... | A frozen loader keeps the core pin and every later step uses that core | verbatim |
| Source: owner selection 9 ("Bot proposes, glue review", label `need-human-review`); workspace `RE... | A frozen loader keeps the core pin and every later step uses that core | the whole Source line |
| The task SHALL move the core pin `DefaultSchemaModule` in `opm/schema/loader.go` to the newest pu... | A core move records a warning that names it and need-human-review | verbatim |
| When it moves, the task SHALL record a warning that names the move and `need-human-review`. | A core move records a warning that names it and need-human-review | verbatim |
| Source: owner selection 9 ("Bot proposes, glue review", label `need-human-review`); workspace `RE... | A core move records a warning that names it and need-human-review | the whole Source line |
| The pin report SHALL carry the label `need-human-review` on the core row, so the PR body's `casca... | The pin report labels the core row need-human-review | verbatim |
| Source: owner selection 9 ("Bot proposes, glue review", label `need-human-review`); workspace `RE... | The pin report labels the core row need-human-review | the whole Source line |

**Text-pinned test trees follow the loader**: form c2 (REMOVED plus ADDED); old body 571 characters; new bodies 386, 406.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Every render tree follows | The text-pinned test trees take the core version DefaultSchemaModule names, same name | no, byte for byte (checked) |
| A lagging tree catches up | The text-pinned test trees take the core version DefaultSchemaModule names, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "A render tree keeps its other dependencies" in "The text-pinned test trees keep every other dependency and run no cue command"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The task SHALL set the core `v:` of `testdata/cue.mod/module.cue` and of every `testdata/render/*... | The text-pinned test trees take the core version DefaultSchemaModule names | verbatim |
| It SHALL rewrite only the `v:` inside the `"opmodel.dev/core@v2"` block. | The text-pinned test trees take the core version DefaultSchemaModule names | verbatim |
| A file that `.cascade-frozen` lists for `opmodel.dev/core@v2` SHALL be left byte-unchanged. | The text-pinned test trees take the core version DefaultSchemaModule names | verbatim |
| The task SHALL set the core `v:` of `testdata/cue.mod/module.cue` and of every `testdata/render/*... | The text-pinned test trees keep every other dependency and run no cue command | verbatim |
| It SHALL NOT change any other dependency in those files, including the synthetic `testing.opmodel... | The text-pinned test trees keep every other dependency and run no cue command | verbatim |

**The opm catalog moves only as far as the loader's core allows**: form c2 (REMOVED plus ADDED); old body 1159 characters; new bodies 274, 309, 360, 429, 437, 485.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Catalog-only move | The opm catalog moves to the newest published v4 release the resolver returns, same name | no, byte for byte (checked) |
| Catalog needs a newer core | The catalog stays when the core its release pins is newer than the loader's, same name | no, byte for byte (checked) |
| Frozen module | A frozen key is left out of the get and the task exits 1 if tidy changed it, same name | no, byte for byte (checked) |
| Tidy raises a frozen key | A frozen key is left out of the get and the task exits 1 if tidy changed it, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "A module whose core differs is in scope" in "A module is in scope when its core differs from the loader value or its catalog is below the target"
- "A module that pins the catalog below the target is in scope" in "A module is in scope when its core differs from the loader value or its catalog is below the target"
- "A module where nothing moved is left untouched" in "A module is in scope when its core differs from the loader value or its catalog is below the target"
- "One get and one tidy per module in scope" in "One cue mod get and one cue mod tidy move a module in scope and name the catalog only when it is below the target"
- "A third-party dependency is not named" in "The get names no third-party dependency and a dependency that tidy raises is a warning"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The task SHALL move `opmodel.dev/catalogs/opm@v4`, read from `testdata/parity/cue.mod/module.cue`... | The opm catalog moves to the newest published v4 release the resolver returns | verbatim |
| Prereleases SHALL NOT count while the current pin is a release. | The opm catalog moves to the newest published v4 release the resolver returns | verbatim |
| A catalog pin above the target SHALL never be lowered. | The opm catalog moves to the newest published v4 release the resolver returns | verbatim |
| The task SHALL move `opmodel.dev/catalogs/opm@v4`, read from `testdata/parity/cue.mod/module.cue`... | The catalog stays when the core its release pins is newer than the loader's | verbatim |
| When the core that catalog release pins is newer than `DefaultSchemaModule`, the catalog SHALL st... | The catalog stays when the core its release pins is newer than the loader's | verbatim |
| The task SHALL move `opmodel.dev/catalogs/opm@v4`, read from `testdata/parity/cue.mod/module.cue`... | A module is in scope when its core differs from the loader value or its catalog is below the target | verbatim |
| Each module of the `CUE_MODULE_GLOBS` set SHALL be in scope when its core differs from the loader... | A module is in scope when its core differs from the loader value or its catalog is below the target | verbatim |
| A module where nothing moved SHALL be left untouched. | A module is in scope when its core differs from the loader value or its catalog is below the target | verbatim |
| For each module in scope, the task SHALL run one `cue mod get` that names core at the loader valu... | One cue mod get and one cue mod tidy move a module in scope and name the catalog only when it is below the... | verbatim |
| The catalog SHALL be named in a module's get only when that module's catalog is below the target. | One cue mod get and one cue mod tidy move a module in scope and name the catalog only when it is below the... | verbatim |
| (added pointer) | One cue mod get and one cue mod tidy move a module in scope and name the catalog only when it is below the... | pointer to the requirement that holds the target and the scope |
| For each module in scope, the task SHALL run one `cue mod get` that names core at the loader valu... | The get names no third-party dependency and a dependency that tidy raises is a warning | verbatim |
| Third-party dependencies SHALL NOT be named. | The get names no third-party dependency and a dependency that tidy raises is a warning | verbatim |
| When tidy raises one, that SHALL be a warning, not a revert. | The get names no third-party dependency and a dependency that tidy raises is a warning | verbatim |
| (added pointer) | The get names no third-party dependency and a dependency that tidy raises is a warning | pointer to the requirement that holds the target and the scope |
| For each module in scope, the task SHALL run one `cue mod get` that names core at the loader valu... | A frozen key is left out of the get and the task exits 1 if tidy changed it | verbatim |
| A key that `.cascade-frozen` lists for a module's `cue.mod/module.cue` SHALL be left out of the g... | A frozen key is left out of the get and the task exits 1 if tidy changed it | verbatim |
| (added pointer) | A frozen key is left out of the get and the task exits 1 if tidy changed it | pointer to the requirement that holds the target and the scope |

**The cascade task never touches release or steering files**: form c2 (REMOVED plus ADDED); old body 780 characters; new bodies 276, 43, 163, 313.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Stale getting-started prose | The cascade task warns when the docs name a core release other than the loader's, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "No release or steering file changes" in "The cascade task modifies no release or steering file"
- "No publish to a registry" in "The cascade task publishes to no registry"
- "A newer language version only warns" in "The cascade task warns of a newer language version and edits nothing"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The task SHALL NOT modify these: | The cascade task modifies no release or steering file | verbatim |
| - `.cascade-frozen` and `.cascade-hold`; | The cascade task modifies no release or steering file | verbatim |
| - `.release-please-manifest.json`, `release-please-config.json` and `CHANGELOG.md`; | The cascade task modifies no release or steering file | verbatim |
| - anything under `.github/`; | The cascade task modifies no release or steering file | verbatim |
| - `.opm-docs-version` and `docs-kit.cue`; | The cascade task modifies no release or steering file | verbatim |
| - `cue-versions.yml`; | The cascade task modifies no release or steering file | verbatim |
| - any `language.version`. | The cascade task modifies no release or steering file | verbatim |
| It SHALL NOT publish to any registry. | The cascade task publishes to no registry | subject copied from the first sentence |
| It SHALL warn, without editing, when an upstream it moved declares a `language.version` newer tha... | The cascade task warns of a newer language version and edits nothing | subject copied from the first sentence |
| It SHALL warn when `docs/getting-started.md` or `AGENTS.md` still names a core release other than... | The cascade task warns when the docs name a core release other than the loader's | subject copied from the first sentence |

**Title and body come from the shared resolver**: form c2 (REMOVED plus ADDED); old body 592 characters; new bodies 298, 293.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Core and catalog moved | Title and body come from the shared resolver with the repo's class map, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "pins.sh reports two pins" in "pins.sh reports the core and catalog pins and reads them without running cue"

| Old clause | New requirement | Change |
| --- | --- | --- |
| `task -x deps:cascade:title` and `task -x deps:cascade:body` SHALL call the shared resolver's `ti... | Title and body come from the shared resolver with the repo's class map | verbatim |
| The class map SHALL classify `testdata/`, `modules/` and `*_test.go` as `test`, and every other p... | Title and body come from the shared resolver with the repo's class map | verbatim |
| `pins.sh` SHALL report two pins: | pins.sh reports the core and catalog pins and reads them without running cue | verbatim |
| - `opmodel.dev/core@v2`, read from the loader, class `shipped`, label `need-human-review`; | pins.sh reports the core and catalog pins and reads them without running cue | verbatim |
| - `opmodel.dev/catalogs/opm@v4`, read from the parity module file, class `test`. | pins.sh reports the core and catalog pins and reads them without running cue | verbatim |
| `pins.sh` SHALL read them from the working tree or from a git ref without running `cue`. | pins.sh reports the core and catalog pins and reads them without running cue | verbatim |

**The cascade task is tested offline and over the network**: form c2 (REMOVED plus ADDED); old body 1365 characters; new bodies 210, 427, 318, 443, 371, 279.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Idempotent catch-up | The full set runs the older-pins scenario, same name | no, byte for byte (checked) |
| Lagging docs without a core move | The lagging-docs scenario expects exit 3, an unchanged tree and a warning naming the file, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "The stub checksum is asserted" in "The cascade test task runs against the canonical resolver stub and passes only when every selected scenario passes"
- "The offline set needs no network" in "The offline set needs no network and runs in the required CI job, and a non-required workflow runs the full set"
- "A frozen module stays byte-unchanged" in "The full set runs the frozen-module scenario"
- "Title and body are checked with a real resolver" in "The full set runs the title and body scenarios when a real resolver is given"

| Old clause | New requirement | Change |
| --- | --- | --- |
| `task -x deps:cascade:test` SHALL run the task in throwaway copies of the tree against the canoni... | The cascade test task runs against the canonical resolver stub and passes only when every selected scenario... | verbatim |
| It SHALL assert the stub's checksum. | The cascade test task runs against the canonical resolver stub and passes only when every selected scenario... | verbatim |
| It SHALL exit 0 only when every selected scenario passes. | The cascade test task runs against the canonical resolver stub and passes only when every selected scenario... | verbatim |
| `task -x deps:cascade:test` SHALL run the task in throwaway copies of the tree against the canoni... | The offline set needs no network and runs in the required CI job, and a non-required workflow runs the full... | verbatim |
| With `CASCADE_TEST_SET=offline`, it SHALL run the no-op, resolver-error, dirty-tree, lagging-tree... | The offline set needs no network and runs in the required CI job, and a non-required workflow runs the full... | verbatim |
| A non-required workflow SHALL run the full set. | The offline set needs no network and runs in the required CI job, and a non-required workflow runs the full... | verbatim |
| `task -x deps:cascade:test` SHALL run the task in throwaway copies of the tree against the canoni... | The lagging-docs scenario expects exit 3, an unchanged tree and a warning naming the file | verbatim |
| The lagging-docs scenario SHALL give the example in `docs/getting-started.md` an older same-major... | The lagging-docs scenario expects exit 3, an unchanged tree and a warning naming the file | verbatim |
| `task -x deps:cascade:test` SHALL run the task in throwaway copies of the tree against the canoni... | The full set runs the older-pins scenario | verbatim |
| With the full set, it SHALL also run these scenarios: | The full set runs the older-pins scenario | verbatim |
| - **older pins:** every location set to an older published version, the two docs examples include... | The full set runs the older-pins scenario | list bullet; the closing ";" becomes "." |
| `task -x deps:cascade:test` SHALL run the task in throwaway copies of the tree against the canoni... | The full set runs the frozen-module scenario | verbatim |
| With the full set, it SHALL also run these scenarios: | The full set runs the frozen-module scenario | verbatim |
| - **frozen module:** a module listed in `.cascade-frozen` stays byte-unchanged; in the same run,... | The full set runs the frozen-module scenario | list bullet; the closing ";" becomes "." |
| `task -x deps:cascade:test` SHALL run the task in throwaway copies of the tree against the canoni... | The full set runs the title and body scenarios when a real resolver is given | verbatim |
| With the full set, it SHALL also run these scenarios: | The full set runs the title and body scenarios when a real resolver is given | verbatim |
| - **title and body**, when a real resolver is given, for a core-and-catalog move and for a catalo... | The full set runs the title and body scenarios when a real resolver is given | verbatim |

**The docs examples follow the loader's core**: form b (MODIFIED parent plus ADDED); old body 884 characters; parent body 457 characters; new bodies 308, 354, 240, 274.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Core moves and the examples follow | The docs examples follow the loader's core, same name | no, byte for byte (checked) |
| A lagging example catches up on the next core move | The docs examples follow the loader's core, same name | no, byte for byte (checked) |
| No core move, no docs edit | The docs examples follow the loader's core, same name | no, byte for byte (checked) |
| Another major is left alone | The docs examples follow the loader's core, same name | no, byte for byte (checked) |
| A release named in prose is left alone | The docs examples follow the loader's core, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "Nothing else in the docs files is rewritten" in "The docs examples change nothing else in the docs files"
- "A run without a core move produces no docs diff" in "The docs examples are not edited in a run that does not move DefaultSchemaModule"
- "A release of another major is not rewritten" in "The docs examples leave a release of another major alone"
- "A frozen docs file is left byte-unchanged" in "A frozen docs file stays byte-unchanged"

| Old clause | New requirement | Change |
| --- | --- | --- |
| In a run that moves `DefaultSchemaModule`, the task SHALL rewrite the version in every `Module: "... | The docs examples follow the loader's core | verbatim |
| Source: workspace `RELEASING.md`, sections "What each repo's task moves" and "Title from diff cla... | The docs examples follow the loader's core | the whole Source line |
| It SHALL change nothing else in those files: a core release named anywhere else, in prose include... | The docs examples change nothing else in the docs files | subject copied from the first sentence |
| (added pointer) | The docs examples change nothing else in the docs files | pointer to the parent requirement that holds the files |
| Source: workspace `RELEASING.md`, sections "What each repo's task moves" and "Title from diff cla... | The docs examples change nothing else in the docs files | the whole Source line |
| It SHALL NOT edit either file in a run that does not move `DefaultSchemaModule`, a run with a fro... | The docs examples are not edited in a run that does not move DefaultSchemaModule | subject copied from the first sentence |
| (added pointer) | The docs examples are not edited in a run that does not move DefaultSchemaModule | pointer to the parent requirement that holds the files |
| Source: workspace `RELEASING.md`, sections "What each repo's task moves" and "Title from diff cla... | The docs examples are not edited in a run that does not move DefaultSchemaModule | the whole Source line |
| It SHALL NOT rewrite a release of another major. | The docs examples leave a release of another major alone | subject copied from the first sentence |
| (added pointer) | The docs examples leave a release of another major alone | pointer to the parent requirement that holds the literals |
| Source: workspace `RELEASING.md`, sections "What each repo's task moves" and "Title from diff cla... | The docs examples leave a release of another major alone | the whole Source line |
| A file that `.cascade-frozen` lists for `opmodel.dev/core@v2` SHALL be left byte-unchanged. | A frozen docs file stays byte-unchanged | verbatim |
| (added pointer) | A frozen docs file stays byte-unchanged | pointer to the parent requirement that holds the files |
| Source: workspace `RELEASING.md`, sections "What each repo's task moves" and "Title from diff cla... | A frozen docs file stays byte-unchanged | the whole Source line |

### Ledger: instance-synthesis

**Values field is caller-supplied with no implicit fallback**: form c2 (REMOVED plus ADDED); old body 611 characters; new bodies 243, 367.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Caller-supplied values participate in the build | Caller-supplied Values are unified in stack order, rendered into the build and checked against #config, same name | no, byte for byte (checked) |
| Layered values unify in order | Caller-supplied Values are unified in stack order, rendered into the build and checked against #config, same name | no, byte for byte (checked) |
| Zero Values is not replaced by debugValues | No implicit source replaces an empty Values, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| `SynthesizeInstance` SHALL NOT consult `Module.debugValues` or any other implicit source when `In... | No implicit source replaces an empty Values | verbatim |
| When `Values` is empty, the kernel SHALL omit the values source and enforce concreteness on the b... | No implicit source replaces an empty Values | verbatim |
| When `Values` holds one or more sources, the kernel SHALL unify them in stack order, render the r... | Caller-supplied Values are unified in stack order, rendered into the build and checked against #config | verbatim |

**Optional labels and annotations are filled into instance metadata**: form c2 (REMOVED plus ADDED); old body 723 characters; new bodies 225, 191, 305.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Caller labels merged with schema-stamped labels | Caller labels cannot remove schema-stamped labels, same name | no, byte for byte (checked) |
| Annotations are passed through unchanged | Caller labels and annotations fill instance metadata, same name | no, byte for byte (checked) |
| Label and annotation order is stable | The staged instance source is deterministic and emits entries in ascending key order, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| When `InstanceInput.Labels` is non-empty, `synth.Instance` SHALL fill `metadata.labels` with thos... | Caller labels and annotations fill instance metadata | verbatim |
| When `InstanceInput.Annotations` is non-empty, `synth.Instance` SHALL fill `metadata.annotations`... | Caller labels and annotations fill instance metadata | verbatim |
| The schema's label stamping (Requirement: Derived fields come from schema unification) unifies wi... | Caller labels cannot remove schema-stamped labels | verbatim |
| The staged instance source SHALL be deterministic: for identical inputs, the bytes synthesis emit... | The staged instance source is deterministic and emits entries in ascending key order | verbatim |
| Entries SHALL be emitted in ascending key order. | The staged instance source is deterministic and emits entries in ascending key order | verbatim |

**Imported-module render coverage exists**: form c2 (REMOVED plus ADDED); old body 646 characters; new bodies 409, 236.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Real imported module renders to resources | A test renders an instance whose module is referenced by import, on the synth path and an authored-package path, same name | no, byte for byte (checked) |
| Module importing a catalog subpackage synthesizes and renders | The synth-path coverage includes a module that imports a catalog subpackage, same name | no, byte for byte (checked) |
| Single-build parity with an authored package | A test renders an instance whose module is referenced by import, on the synth path and an authored-package path, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| The library SHALL include a test that renders an instance whose module is referenced by import (n... | A test renders an instance whose module is referenced by import, on the synth path and an authored-package... | verbatim |
| This coverage SHALL exist for both the synth path and an authored-package path so that a regressi... | A test renders an instance whose module is referenced by import, on the synth path and an authored-package... | verbatim |
| The synth-path coverage SHALL include a module that imports a **catalog subpackage** (e.g. a work... | The synth-path coverage includes a module that imports a catalog subpackage | verbatim |

**synth.Instance constructs the instance by single-build CUE evaluation**: form c2 (REMOVED plus ADDED); old body 2503 characters; new bodies 340, 458, 414, 259, 414, 194, 434.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| No scope or Go pre-merge in the construction path | The schema merges the values and the Go code fills only caller-supplied inputs, same name | no, byte for byte (checked) |
| No fabricated module file | The instance source is overlaid into the module's staged source tree and no module file is fabricated, same name | no, byte for byte (checked) |
| Values merged in-build by the schema | The schema merges the values and the Go code fills only caller-supplied inputs, same name | no, byte for byte (checked) |
| Imported-module construction succeeds | The instance source imports the module's package and core, and the module import resolves locally, same name | no, byte for byte (checked) |
| Single-build parity with an authored package | synth.Instance builds the instance in a single build through the shape gate, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "The values file is overlaid beside the instance source" in "A values file rendered from the input's Values is overlaid beside the instance source"
- "synth.Instance lives off the public surface" in "synth.Instance lives in opm/internal/synth with a fixed Go signature"
- "The cueCtx and coreVersion of synth.Instance" in "The cueCtx and coreVersion parameters of synth.Instance are the call context and the core release"
- "The Input of synth.Instance" in "The Input parameter of synth.Instance is the package-internal input"

| Old clause | New requirement | Change |
| --- | --- | --- |
| `synth.Instance` SHALL construct the `#ModuleInstance` value by synthesizing an in-memory CUE pac... | synth.Instance builds the instance in a single build through the shape gate | verbatim |
| The instance source SHALL be overlaid **into the acquired module's own staged source tree** (unde... | The instance source is overlaid into the module's staged source tree and no module file is fabricated | verbatim |
| `synth.Instance` SHALL NOT fabricate a `cue.mod/module.cue` and SHALL NOT declare a dependency se... | The instance source is overlaid into the module's staged source tree and no module file is fabricated | verbatim |
| The instance source SHALL **import** the module's own package by its module path and write `#modu... | The instance source imports the module's package and core, and the module import resolves locally | verbatim |
| Because the instance is built inside the module's main module, the module import resolves **local... | The instance source imports the module's package and core, and the module import resolves locally | SHALL added: the old sentence has no keyword |
| A values file rendered from the input's `Values` (the unified value of the caller's `InstanceInpu... | A values file rendered from the input's Values is overlaid beside the instance source | verbatim |
| `synth.Instance` SHALL NOT inject the module via `cue.Scope` / a `userModule` field, and SHALL NO... | The schema merges the values and the Go code fills only caller-supplied inputs | verbatim |
| The values merge SHALL be performed by the schema in CUE (`#ModuleInstance`'s `unifiedModule = #m... | The schema merges the values and the Go code fills only caller-supplied inputs | verbatim |
| The Go code SHALL fill only caller-supplied inputs and let CUE derive `metadata.uuid`, `component... | The schema merges the values and the Go code fills only caller-supplied inputs | verbatim |
| `synth.Instance` lives in `opm/internal/synth`, off the public surface; its Go signature is `Inst... | synth.Instance lives in opm/internal/synth with a fixed Go signature | SHALL added: the old sentence has no keyword |
| `cueCtx` is the context `Kernel.SynthesizeInstance` created for the call, `coreVersion` is the ke... | The cueCtx and coreVersion parameters of synth.Instance are the call context and the core release | SHALL added: the old sentence has no keyword; the subject `synth.Instance` and a colon are added, the sentence is cut at the list boundary before `Input` |
| `Input` is the package-internal input carrying `Module`, `Name`, `Namespace`, the unified `Values`, `Labels`, `Annotations` and the kernel's registry en... | The Input parameter of synth.Instance is the package-internal input | SHALL added: the old sentence has no keyword; the subject `synth.Instance` and a colon are added, the clause is cut at the list boundary |

**synth.Instance requires the module's staged source**: form c2 (REMOVED plus ADDED); old body 1229 characters; new bodies 187, 486, 467, 383, 197.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Module without staged source is rejected | A module without staged source is refused and the kernel fetches nothing itself, same name | no, byte for byte (checked) |
| Module acquired with source synthesizes | Synthesis constructs the instance inside the acquired module's staged source tree, same name | no, byte for byte (checked) |
| Module acquired from a directory synthesizes | Synthesis constructs the instance inside the acquired module's staged source tree, same name | no, byte for byte (checked) |
| Subpackage module is refused | A module acquired from a subdirectory of its CUE module is refused, same name | no, byte for byte (checked) |
| An existing synthetic root refuses a synthesis | An existing synthetic root refuses a synthesis of a registry-acquired module, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "A module acquired from a directory is not checked" in "A module acquired from a directory is not checked for a synthetic root"

| Old clause | New requirement | Change |
| --- | --- | --- |
| `SynthesizeInstance` SHALL construct the instance inside the acquired module's staged source tree... | Synthesis constructs the instance inside the acquired module's staged source tree | verbatim |
| `SynthesizeInstance` SHALL construct the instance inside the acquired module's staged source tree... | A module without staged source is refused and the kernel fetches nothing itself | verbatim |
| When `InstanceInput.Module` carries no staged source (it was constructed from a bare value), the... | A module without staged source is refused and the kernel fetches nothing itself | verbatim |
| The kernel SHALL NOT perform a registry fetch or a directory walk of its own to obtain the module... | A module without staged source is refused and the kernel fetches nothing itself | verbatim |
| `SynthesizeInstance` SHALL construct the instance inside the acquired module's staged source tree... | A module acquired from a subdirectory of its CUE module is refused | verbatim |
| When the module's `Source.Pkg` is non-empty (the module was acquired from a subdirectory of its C... | A module acquired from a subdirectory of its CUE module is refused | verbatim |
| When the module was acquired from a registry, its staged root is a synthetic root that does not e... | An existing synthetic root refuses a synthesis of a registry-acquired module | verbatim |
| A module acquired from a directory has a real root on disk and is not checked. | A module acquired from a directory is not checked for a synthetic root | SHALL NOT added: the old sentence has no keyword |
| (added pointer) | A module acquired from a directory is not checked for a synthetic root | pointer to the requirement that holds the check |

**Synthesis surfaces its staged tree**: form c2 (REMOVED plus ADDED); old body 533 characters; new bodies 374, 261.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| The staged tree matches the build | Synthesis stamps the staged source tree on the returned instance, same name | no, byte for byte (checked) |
| Failure returns no tree | Synthesis stamps the staged source tree on the returned instance, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "The overlay is a clone" in "The stamped overlay is a clone"

| Old clause | New requirement | Change |
| --- | --- | --- |
| `SynthesizeInstance` SHALL stamp on the returned instance the staged source tree the single build... | Synthesis stamps the staged source tree on the returned instance | verbatim |
| The overlay SHALL be a clone, so repeated synthesis from one module remains safe and the caller m... | The stamped overlay is a clone | verbatim |
| (added pointer) | The stamped overlay is a clone | pointer to the requirement that holds the overlay |

**Instance synthesis input**: form b (MODIFIED parent plus ADDED); old body 928 characters; parent body 392 characters; new bodies 158, 242, 210.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Required inputs validated | Instance synthesis input, same name | no, byte for byte (checked) |
| One synthesis entry point | Instance synthesis input, same name | no, byte for byte (checked) |
| Returned value is schema-unified | Instance synthesis input, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "InstanceInput holds no schema cache and no context" in "InstanceInput carries no schema cache and no cue.Context"
- "A missing required field fails before any build" in "A missing required field fails before any build runs"
- "No second synthesis entry point or input type" in "No public package exports a second synthesis entry point or input type"

| Old clause | New requirement | Change |
| --- | --- | --- |
| Instance synthesis SHALL be reached only through `Kernel.SynthesizeInstance(ctx, kernel.InstanceI... | Instance synthesis input | verbatim |
| `InstanceInput` SHALL carry `Module *module.Module` (required, source-carrying), `Name string` (r... | Instance synthesis input | verbatim |
| It SHALL carry no schema cache and no `cue.Context`: the kernel owns the schema cache, and synthe... | InstanceInput carries no schema cache and no cue.Context | subject copied from the first sentence |
| A missing required field SHALL fail with an error wrapping the matching `opm/errors` sentinel (`E... | A missing required field fails before any build runs | verbatim |
| (added pointer) | A missing required field fails before any build runs | pointer to the parent requirement that holds the fields |
| No public package under `opm/` SHALL export a second synthesis entry point or input type; the bui... | No public package exports a second synthesis entry point or input type | verbatim |

### Ledger: cascade-wiring

**The receiver runs on dispatch, daily and by hand**: form c2 (REMOVED plus ADDED); old body 1050 characters; new bodies 380, 373, 237.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| An upstream release arrives | The receiver workflow calls cascade-receive.yml and runs on dispatch, on a schedule and by hand, same name | no, byte for byte (checked) |
| The daily sweep | The receiver workflow calls cascade-receive.yml and runs on dispatch, on a schedule and by hand, same name | no, byte for byte (checked) |
| A gates-only run does not displace a pending real run | The receiver workflow uses a concurrency group by run kind, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "The workflow keys and the cascade job grants are fixed" in "The receiver workflow has fixed top-level keys and its cascade job has fixed grants"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The library SHALL have a workflow `.github/workflows/deps-cascade.yml` whose job `cascade` calls... | The receiver workflow calls cascade-receive.yml and runs on dispatch, on a schedule and by hand | verbatim |
| It SHALL run on `repository_dispatch` of type `upstream-released`, on a daily schedule at `17 5 *... | The receiver workflow calls cascade-receive.yml and runs on dispatch, on a schedule and by hand | verbatim |
| Its top-level keys SHALL be exactly `name`, `on`, `permissions`, `concurrency` and `jobs`, with `... | The receiver workflow has fixed top-level keys and its cascade job has fixed grants | verbatim; `Its` becomes `.github/workflows/deps-cascade.yml`: its (the path of the first sentence, a colon, the pronoun in lower case) |
| The `cascade` job SHALL be granted only `contents: read`, `pull-requests: read` and `statuses: wr... | The receiver workflow has fixed top-level keys and its cascade job has fixed grants | verbatim |
| It SHALL use the concurrency group `deps-cascade` without cancelling for real runs on `main`, `de... | The receiver workflow uses a concurrency group by run kind | verbatim; `It` becomes `.github/workflows/deps-cascade.yml`: it (the path of the first sentence, a colon, the pronoun in lower case) |
| Source: workspace `RELEASING.md`, sections "The receiver" and "Concurrency"; Phase 3... | The receiver workflow calls cascade-receive.yml and runs on dispatch, on a schedule and by hand | the whole Source line, as a `**Source**:` line |
| Source: workspace `RELEASING.md`, sections "The receiver" and "Concurrency"; Phase 3... | The receiver workflow has fixed top-level keys and its cascade job has fixed grants | the whole Source line, as a `**Source**:` line |
| Source: workspace `RELEASING.md`, sections "The receiver" and "Concurrency"; Phase 3... | The receiver workflow uses a concurrency group by run kind | the whole Source line, as a `**Source**:` line |

**The receiver publishes only from its own caller-owned publish job**: form c2 (REMOVED plus ADDED); old body 1319 characters; new bodies 463, 168, 479, 77.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A live run with a change to push | The receiver has a publish job of a fixed shape with one step, same name | no, byte for byte (checked) |
| Compute plans nothing to publish | The receiver has a publish job of a fixed shape with one step, same name | no, byte for byte (checked) |
| A reusable call is given the secrets | Only the caller-owned notify-downstream or publish job reads the App key and no reusable call passes secrets, same name | no, byte for byte (checked) |
| Another job reaches the key through a variant spelling | No other job in any library workflow declares the cascade Environment, same name | no, byte for byte (checked) |
| A gates-only run never reaches publish | The publish job never starts for a gates-only run, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| `deps-cascade.yml` SHALL have a job `publish` that needs `cascade`, runs on `ubuntu-latest`, decl... | The receiver has a publish job of a fixed shape with one step | verbatim |
| The job's `if:` SHALL require `inputs.gates_only != true`, read from the caller's own input and n... | The publish job never starts for a gates-only run | verbatim; `publish` added after `The` to name the job (copied from the same old requirement) |
| The key SHALL be read only in the caller-owned `notify-downstream` or `publish` job, which declar... | Only the caller-owned notify-downstream or publish job reads the App key and no reusable call passes secrets | verbatim; the prefix `secrets.CASCADE_APP_PRIVATE_KEY`: names the key and `The` becomes `the` |
| No other job in any library workflow SHALL declare the `cascade` Environment. | No other job in any library workflow declares the cascade Environment | verbatim |
| Source: Phase 3 wiring contract (version 3.1) §2.2, §5 and §10.1 item 9; workspace `R... | The receiver has a publish job of a fixed shape with one step | the whole Source line, as a `**Source**:` line |
| Source: Phase 3 wiring contract (version 3.1) §2.2, §5 and §10.1 item 9; workspace `R... | The publish job never starts for a gates-only run | the whole Source line, as a `**Source**:` line |
| Source: Phase 3 wiring contract (version 3.1) §2.2, §5 and §10.1 item 9; workspace `R... | Only the caller-owned notify-downstream or publish job reads the App key and no reusable call passes secrets | the whole Source line, as a `**Source**:` line |
| Source: Phase 3 wiring contract (version 3.1) §2.2, §5 and §10.1 item 9; workspace `R... | No other job in any library workflow declares the cascade Environment | the whole Source line, as a `**Source**:` line |

**The receiver pushes nothing until CASCADE_DRY_RUN is false**: form c2 (REMOVED plus ADDED); old body 1092 characters; new bodies 378, 461, 228.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| The variable is unset | The receiver treats an unset or other CASCADE_DRY_RUN value as a dry run, same name | no, byte for byte (checked) |
| Live, but a manual dry run | The receiver publishes only when CASCADE_DRY_RUN is false and the run is not a manual dry run, same name | no, byte for byte (checked) |
| Nothing to move on main | A dry run writes its plan to the job summary and changes nothing, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "The dry-run expression reaches both consumers" in "The receiver treats an unset or other CASCADE_DRY_RUN value as a dry run"
- "The gate statuses are still posted in a dry run" in "A dry run writes its plan to the job summary and changes nothing"

| Old clause | New requirement | Change |
| --- | --- | --- |
| `deps-cascade.yml` SHALL pass `${{ inputs.dry_run == true \|\| vars.CASCADE_DRY_RUN != 'false' }}... | The receiver treats an unset or other CASCADE_DRY_RUN value as a dry run | verbatim |
| So the receiver SHALL publish only when the repo variable `CASCADE_DRY_RUN` is `false` (GitHub co... | The receiver publishes only when CASCADE_DRY_RUN is false and the run is not a manual dry run | verbatim; the body adds before it `(Requirement: The receiver treats an unset or other CASCADE_DRY_RUN value as a dry run)` (a subject or a pointer to another requirement of the same spec) |
| An unset, deleted or any other value SHALL mean a dry run. | The receiver treats an unset or other CASCADE_DRY_RUN value as a dry run | verbatim |
| In a dry run the receiver computes the diff, title, body and labels and writes them to the job su... | A dry run writes its plan to the job summary and changes nothing | SHALL added: the old sentence has no keyword; the verbs take the infinitive after `SHALL` |
| The gate statuses SHALL still be posted in a dry run. | A dry run writes its plan to the job summary and changes nothing | verbatim |
| Source: workspace `RELEASING.md`, section "Stop switches"; Phase 3 wiring contract (v... | The receiver treats an unset or other CASCADE_DRY_RUN value as a dry run | the whole Source line, as a `**Source**:` line |
| Source: workspace `RELEASING.md`, section "Stop switches"; Phase 3 wiring contract (v... | The receiver publishes only when CASCADE_DRY_RUN is false and the run is not a manual dry run | the whole Source line, as a `**Source**:` line |
| Source: workspace `RELEASING.md`, section "Stop switches"; Phase 3 wiring contract (v... | A dry run writes its plan to the job summary and changes nothing | the whole Source line, as a `**Source**:` line |

**A core bump reaches the cascade PR as need-human-review**: form c2 (REMOVED plus ADDED); old body 730 characters; new bodies 328, 407.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Core moved | A core move labels the cascade PR need-human-review, a catalog-only move does not, and the bot never removes the label, same name | no, byte for byte (checked) |
| Only the opm catalog moved | A core move labels the cascade PR need-human-review, a catalog-only move does not, and the bot never removes the label, same name | no, byte for byte (checked) |
| Dry run of a core move | A core move labels the cascade PR need-human-review, a catalog-only move does not, and the bot never removes the label, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "The publish step leaves label creation to the action" in "The need-human-review label comes from the pin report's core row and from the cascade-publish action"

| Old clause | New requirement | Change |
| --- | --- | --- |
| When a cascade run moves `DefaultSchemaModule`, the resulting PR SHALL carry the label `need-huma... | A core move labels the cascade PR need-human-review, a catalog-only move does not, and the bot never remove... | verbatim |
| The library SHALL get this from its pin report's core row label and from the `cascade-publish` ac... | The need-human-review label comes from the pin report's core row and from the cascade-publish action | verbatim; the body adds before it `(Requirement: A core move labels the cascade PR need-human-review, a catalog-only move...` (a subject or a pointer to another requirement of the same spec) |
| A catalog-only move SHALL NOT add the label. | A core move labels the cascade PR need-human-review, a catalog-only move does not, and the bot never remove... | verbatim |
| The bot SHALL NOT remove the label once set. | A core move labels the cascade PR need-human-review, a catalog-only move does not, and the bot never remove... | verbatim |
| Source: owner selection 9 ("Bot proposes, glue review", label `need-human-review`); w... | A core move labels the cascade PR need-human-review, a catalog-only move does not, and the bot never remove... | the whole Source line, as a `**Source**:` line |
| Source: owner selection 9 ("Bot proposes, glue review", label `need-human-review`); w... | The need-human-review label comes from the pin report's core row and from the cascade-publish action | the whole Source line, as a `**Source**:` line |

**Every PR carries the cascade gate statuses**: form c2 (REMOVED plus ADDED); old body 775 characters; new bodies 476, 205.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| An ordinary PR | The cascade-gates.yml workflow runs on pull_request_target with fixed grants and checks out no PR code, same name | no, byte for byte (checked) |
| A release PR in warn mode | The cascade-gates.yml workflow runs on pull_request_target with fixed grants and checks out no PR code, same name | no, byte for byte (checked) |
| A shipped pin is behind | The cascade-gates.yml workflow runs on pull_request_target with fixed grants and checks out no PR code, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "A newer run cancels an older run of the same PR" in "The cascade gates cancel an older run per PR and are no required check"
- "The gate statuses are not required checks" in "The cascade gates cancel an older run per PR and are no required check"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The library SHALL have a workflow `.github/workflows/cascade-gates.yml` that runs on `pull_reques... | The cascade-gates.yml workflow runs on pull_request_target with fixed grants and checks out no PR code | verbatim |
| It SHALL check out no PR code. | The cascade-gates.yml workflow runs on pull_request_target with fixed grants and checks out no PR code | verbatim |
| The concurrency group SHALL be per PR number, cancelling an older run. | The cascade gates cancel an older run per PR and are no required check | verbatim; the body adds before it ``.github/workflows/cascade-gates.yml`:` (a subject or a pointer to another requirement of the same spec) |
| Neither `cascade/freshness` nor `cascade/settled` SHALL become a required check in this change. | The cascade gates cancel an older run per PR and are no required check | verbatim |
| Source: workspace `RELEASING.md`, section "Gates" (G2, G3); owner selections 11 and 1... | The cascade-gates.yml workflow runs on pull_request_target with fixed grants and checks out no PR code | the whole Source line, as a `**Source**:` line |
| Source: workspace `RELEASING.md`, section "Gates" (G2, G3); owner selections 11 and 1... | The cascade gates cancel an older run per PR and are no required check | the whole Source line, as a `**Source**:` line |

**The cascade references carry one .github SHA, checked on every PR**: form c2 (REMOVED plus ADDED); old body 2633 characters; new bodies 496, 185, 177, 434, 432, 445, 318.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| The wiring is as the contract gives it | The wiring check runs in the required Go tests job on every PR and confirms the pin and the copy, same name | no, byte for byte (checked) |
| One reference moves alone | Every cascade reference names the same full .github main SHA and no branch or tag, same name | no, byte for byte (checked) |
| A key-holding job gains an env | The wiring check refuses shapes of this capability and runs offline through task cascade:wiring:check, same name | no, byte for byte (checked) |
| A .github cascade change lands | Every cascade reference names the same full .github main SHA and no branch or tag, same name | no, byte for byte (checked) |
| A pin that is not on .github main | The wiring check runs in the required Go tests job on every PR and confirms the pin and the copy, same name | no, byte for byte (checked) |
| The copy drifts from .github | The wiring-check script is a byte-identical copy of the .github script and moves only with the pin, same name | no, byte for byte (checked) |
| CI catches a copy that differs from the pin | The wiring check runs in the required Go tests job on every PR and confirms the pin and the copy, same name | no, byte for byte (checked) |
| A run step before the wiring step | The test.yml workflow keeps a narrow env and only pinned actions before the wiring step, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "A missing resolver fails the task" in "cascade-task.yml fails, not skips S5, when the resolver is missing"
- "Dependabot leaves the .github pin alone" in "Dependabot ignores the .github repository and waits 7 days on github-actions"

| Old clause | New requirement | Change |
| --- | --- | --- |
| Every cascade reference in the library's workflows SHALL name the same full 40-character SHA of a... | Every cascade reference names the same full .github main SHA and no branch or tag | verbatim |
| No reference SHALL name a branch or a tag. | Every cascade reference names the same full .github main SHA and no branch or tag | verbatim |
| `cascade-task.yml` SHALL fail, not skip S5, when the resolver is missing at that commit. | cascade-task.yml fails, not skips S5, when the resolver is missing | verbatim; the body adds before it `(Requirement: Every cascade reference names the same full .github main SHA and no branc...` (a subject or a pointer to another requirement of the same spec) |
| `.github/dependabot.yml` SHALL ignore `open-platform-model/.github*` under `github-actions`. | Dependabot ignores the .github repository and waits 7 days on github-actions | verbatim |
| `.github/dependabot.yml` SHALL give the `github-actions` entry a cooldown of 7 days. | Dependabot ignores the .github repository and waits 7 days on github-actions | verbatim |
| The script `.tasks/cascade/wiring-check.sh` SHALL be a byte-identical copy of `.github/scripts/ca... | The wiring-check script is a byte-identical copy of the .github script and moves only with the pin | verbatim |
| It SHALL check these references and the shapes of this capability, SHALL refuse any `release.yml`... | The wiring check refuses shapes of this capability and runs offline through task cascade:wiring:check | verbatim up to the list boundary; `It` becomes `.tasks/cascade/wiring-check.sh`: it, the pointer (Requirement: ...) follows `these references` and a closing `.` ends the sentence |
| and, in the required `Go tests` job on every PR, as the step "Verify the cascade wiring" with `GH... | The wiring check runs in the required Go tests job on every PR and confirms the pin and the copy | verbatim from the list boundary; `and,` becomes `.tasks/cascade/wiring-check.sh`: it SHALL run, (the subject and the verb repeated from the cut sentence); the pointer (Requirement: ...) follows `confirms the SHA` |
| In `test.yml`, the workflow's and the `test` job's `env` SHALL name only `CUE_*`, `OPM_*`, `REGIS... | The test.yml workflow keeps a narrow env and only pinned actions before the wiring step | verbatim |
| The copy moves only with the pin. | The wiring-check script is a byte-identical copy of the .github script and moves only with the pin | verbatim |
| Source: owner decision 24, extended to the two reusable workflows and the resolver ch... | Every cascade reference names the same full .github main SHA and no branch or tag | the whole Source line, as a `**Source**:` line |
| Source: owner decision 24, extended to the two reusable workflows and the resolver ch... | cascade-task.yml fails, not skips S5, when the resolver is missing | the whole Source line, as a `**Source**:` line |
| Source: owner decision 24, extended to the two reusable workflows and the resolver ch... | Dependabot ignores the .github repository and waits 7 days on github-actions | the whole Source line, as a `**Source**:` line |
| Source: owner decision 24, extended to the two reusable workflows and the resolver ch... | The wiring-check script is a byte-identical copy of the .github script and moves only with the pin | the whole Source line, as a `**Source**:` line |
| Source: owner decision 24, extended to the two reusable workflows and the resolver ch... | The wiring check refuses shapes of this capability and runs offline through task cascade:wiring:check | the whole Source line, as a `**Source**:` line |
| Source: owner decision 24, extended to the two reusable workflows and the resolver ch... | The wiring check runs in the required Go tests job on every PR and confirms the pin and the copy | the whole Source line, as a `**Source**:` line |
| Source: owner decision 24, extended to the two reusable workflows and the resolver ch... | The test.yml workflow keeps a narrow env and only pinned actions before the wiring step | the whole Source line, as a `**Source**:` line |

### Ledger: config-validation

**Source Type and Layered Input**: form c2 (REMOVED plus ADDED); old body 752 characters; new bodies 306, 269, 175.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Source struct shape | A Source pairs CUE source bytes with its origin and carries no cue.Value and no display label, same name | no, byte for byte (checked) |
| Partial option | The Source struct is the one values input of every values-taking kernel entry, same name | no, byte for byte (checked) |
| Stack ordering for layered inputs | The Source struct is the one values input of every values-taking kernel entry, same name | no, byte for byte (checked) |
| One values shape everywhere | The Source struct is the one values input of every values-taking kernel entry, same name | no, byte for byte (checked) |
| Positions survive compilation at use | Each kernel entry compiles its sources with cue.Filename(Origin) in the context of the schema they meet, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| The library SHALL expose a `Source` struct in `opm/kernel/` describing one values input for every... | The Source struct is the one values input of every values-taking kernel entry | verbatim |
| A `Source` pairs the values payload, as CUE source bytes, with its stable origin; it SHALL carry... | A Source pairs CUE source bytes with its origin and carries no cue.Value and no display label | verbatim |
| Each kernel entry SHALL compile the sources it receives in the context of the schema they meet, w... | Each kernel entry compiles its sources with cue.Filename(Origin) in the context of the schema they meet | verbatim |
| No kernel entry SHALL take values in any other shape. | The Source struct is the one values input of every values-taking kernel entry | verbatim |

**File-Backed Sources Resolve Imports Through the Kernel Mapping**: form b (MODIFIED plus ADDED); old body 683 characters; new bodies 267, 493.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Values file importing a registry module | File-Backed Sources Resolve Imports Through the Kernel Mapping, same name | no, byte for byte (checked) |
| Same mapping on every compiling path | File-Backed Sources Resolve Imports Through the Kernel Mapping, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "Absent the option the load reads CUE_REGISTRY" in "A file-backed source without the option reads the process CUE_REGISTRY and the process environment is not mutated"
- "A source compiled from bytes is unaffected" in "A file-backed source without the option reads the process CUE_REGISTRY and the process environment is not mutated"
- "The process environment is not mutated" in "A file-backed source without the option reads the process CUE_REGISTRY and the process environment is not mutated"

| Old clause | New requirement | Change |
| --- | --- | --- |
| When a `Source` is file-backed (its `Origin` is an absolute path naming an existing file), the op... | File-Backed Sources Resolve Imports Through the Kernel Mapping | verbatim |
| Absent the option, the load SHALL read the process `CUE_REGISTRY` unchanged. | A file-backed source without the option reads the process CUE_REGISTRY and the process environment is not m... | verbatim; the body adds before it `(Requirement: File-Backed Sources Resolve Imports Through the Kernel Mapping)` (a subject or a pointer to another requirement of the same spec) |
| This SHALL hold on every path that compiles sources: `ValidateConfigDetailed`, `AcquireInstanceFr... | File-Backed Sources Resolve Imports Through the Kernel Mapping | verbatim |
| A source compiled from bytes carries no imports and is unaffected. | A file-backed source without the option reads the process CUE_REGISTRY and the process environment is not m... | verbatim |
| The process environment SHALL NOT be mutated. | A file-backed source without the option reads the process CUE_REGISTRY and the process environment is not m... | verbatim |

**Internal Closed-Schema Workaround**: form c2 (REMOVED plus ADDED); old body 731 characters; new bodies 268, 486.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Disallowed field in closed schema produces positioned error | The disallowed-field error Path returns one selector segment per field label and a label with dots is one segment, same name | no, byte for byte (checked) |
| A label containing dots is one path segment | The disallowed-field error Path returns one selector segment per field label and a label with dots is one segment, same name | no, byte for byte (checked) |
| Internal types not exported | The closed-schema walk is private to opm/kernel/validate.go and its error implements cueerrors.Error, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "The disallowed-field error is a cueerrors.Error" in "The closed-schema walk is private to opm/kernel/validate.go and its error implements cueerrors.Error"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The library SHALL retain `walkDisallowed` and `fieldNotAllowedError` as private internals of `opm... | The closed-schema walk is private to opm/kernel/validate.go and its error implements cueerrors.Error | verbatim |
| The error type SHALL implement `cuelang.org/go/cue/errors.Error` so that disallowed-field diagnos... | The closed-schema walk is private to opm/kernel/validate.go and its error implements cueerrors.Error | verbatim |
| The error's `Path()` SHALL return `values` followed by one segment per field label on the way to... | The disallowed-field error Path returns one selector segment per field label and a label with dots is one s... | verbatim; the body adds before it ``fieldNotAllowedError`:` (a subject or a pointer to another requirement of the same spec) |
| A label that contains dots SHALL be one segment, so joining the segments with `.` gives the field... | The disallowed-field error Path returns one selector segment per field label and a label with dots is one s... | verbatim |

**Single Kernel Validation Primitive**: form b (MODIFIED plus ADDED); old body 1448 characters; new bodies 431, 420, 300, 467.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| ValidateConfigDetailed signature and behavior | Single Kernel Validation Primitive, same name | no, byte for byte (checked) |
| No single-value or partial variant is exported | Single Kernel Validation Primitive, same name | no, byte for byte (checked) |
| Empty inputs short-circuit to success | Single Kernel Validation Primitive, same name | no, byte for byte (checked) |
| Errors carry source positions when filename was set at compile time | Single Kernel Validation Primitive, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "Sources are compiled in the schema's context and unified in stack order" in "ValidateConfigDetailed compiles each source in the schema's context, unifies in stack order, walks disallowed fields and asserts concreteness"
- "A failed validation step is a ConfigValidationError" in "A failure of the validation step is a ConfigValidationError that wraps the CUE error tree unchanged"
- "A source that fails to load is returned without the marker" in "A source that fails to compile or to load is returned without the ConfigValidationError marker"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The library SHALL expose exactly one validation method on `*Kernel` in `opm/kernel/`: `ValidateCo... | Single Kernel Validation Primitive | verbatim |
| (cue.Value, error)`. | Single Kernel Validation Primitive | verbatim; the body adds before it `The library SHALL expose exactly one validation method on `*Kernel` in `opm/kernel/`: `...` (a subject or a pointer to another requirement of the same spec) |
| It SHALL compile each source in the schema's own context so that every position names the source'... | ValidateConfigDetailed compiles each source in the schema's context, unifies in stack order, walks disallow... | verbatim; `It` becomes `ValidateConfigDetailed`: it |
| A failure of the validation step (the disallowed-field walk, the unification with the schema, or... | A failure of the validation step is a ConfigValidationError that wraps the CUE error tree unchanged | verbatim; the body adds before it ``ValidateConfigDetailed`:` (a subject or a pointer to another requirement of the same spec) |
| SHALL be returned as a `*ConfigValidationError` (`opm/errors`) that wraps the CUE-native error tr... | A failure of the validation step is a ConfigValidationError that wraps the CUE error tree unchanged | verbatim; the body adds before it ``ValidateConfigDetailed`: A failure of the validation step (the disallowed-field walk,...` (a subject or a pointer to another requirement of the same spec) |
| A source that fails to compile or to load is returned as before, without the marker: such a failu... | A source that fails to compile or to load is returned without the ConfigValidationError marker | SHALL added: the old sentence has no keyword; `is returned` becomes `SHALL be returned`; the pointer (Requirement: ...) is added before the sentence |
| The kernel SHALL NOT expose a single-value or a partial-mode variant: a single value is a one-ele... | Single Kernel Validation Primitive | verbatim |

**A validation failure has one marker type and no projection**: form c2 (REMOVED plus ADDED); old body 674 characters; new bodies 306, 392.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| opm/errors carries no validation projections | ConfigValidationError projects nothing and the library has no parallel walking API and none of the retired error names, same name | no, byte for byte (checked) |
| Frontends rely on cuelang.org/go/cue/errors | ConfigValidationError projects nothing and the library has no parallel walking API and none of the retired error names, same name | no, byte for byte (checked) |
| A validation failure is matched by type | ConfigValidationError in opm/errors is the one Go type of a validation failure and carries the CUE error tree, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| The library SHALL define exactly one Go type for a configuration validation failure: `ConfigValid... | ConfigValidationError in opm/errors is the one Go type of a validation failure and carries the CUE error tree | verbatim |
| It SHALL carry the tree in its `Err` field, return it from `Unwrap`, and return the tree's own te... | ConfigValidationError in opm/errors is the one Go type of a validation failure and carries the CUE error tree | verbatim |
| It SHALL NOT project, group, re-order or reword the CUE errors, and the library SHALL NOT provide... | ConfigValidationError projects nothing and the library has no parallel walking API and none of the retired... | verbatim; `It` becomes `ConfigValidationError`: it |
| The names `ConfigError`, `ValidationError`, `FieldError`, `ErrorLocation`, `GroupedError`, `Multi... | ConfigValidationError projects nothing and the library has no parallel walking API and none of the retired... | verbatim |

**Every typed error has a pointer receiver**: form c2 (REMOVED plus ADDED); old body 550 characters; new bodies 227, 407.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Every typed error matches through a pointer target | Every error type in opm/errors declares Error on its pointer receiver, same name | no, byte for byte (checked) |
| No typed error keeps a value receiver | Every error type in opm/errors declares Error on its pointer receiver, same name | no, byte for byte (checked) |
| The value type is not an error | IdentityError follows the pointer-receiver rule and a value-typed IdentityError target does not match, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "IdentityError is returned as a pointer" in "IdentityError follows the pointer-receiver rule and a value-typed IdentityError target does not match"

| Old clause | New requirement | Change |
| --- | --- | --- |
| Every error type in `opm/errors` that has an `Error` method SHALL declare it on the pointer recei... | Every error type in opm/errors declares Error on its pointer receiver | verbatim |
| `IdentityError` SHALL follow the rule: the library returns `*IdentityError`, the type declares `E... | IdentityError follows the pointer-receiver rule and a value-typed IdentityError target does not match | verbatim; the pointer (Requirement: ...) is added after `the rule` and the colon stays |
| A value-typed `IdentityError` target SHALL NOT match: `errors.AsType[IdentityError]` does not com... | IdentityError follows the pointer-receiver rule and a value-typed IdentityError target does not match | verbatim |

### Ledger: workflow-hardening

**Every workflow states least-privilege token grants**: form c2 (REMOVED plus ADDED); old body 634 characters; new bodies 280, 288.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| The repo default token becomes read-only | Every workflow declares permissions at the workflow level and no pull_request job holds a write grant, same name | no, byte for byte (checked) |
| A same-repo PR runs the tests | Every workflow declares permissions at the workflow level and no pull_request job holds a write grant, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "A checkout does not persist credentials" in "Every checkout step sets persist-credentials false"

| Old clause | New requirement | Change |
| --- | --- | --- |
| Every workflow file under `.github/workflows/` SHALL declare `permissions:` at the workflow level... | Every workflow declares permissions at the workflow level and no pull_request job holds a write grant | verbatim |
| No job that runs on `pull_request` SHALL hold a write grant. | Every workflow declares permissions at the workflow level and no pull_request job holds a write grant | verbatim |
| Every checkout step in this repository's workflow files SHALL set `persist-credentials: false`, s... | Every checkout step sets persist-credentials false | verbatim |
| (A reusable workflow called from another repository, such as docs-kit's `publish.yml`, owns its o... | Every checkout step sets persist-credentials false | verbatim |
| Source: owner selection 30; security pass findings N4 and GOV-3. | Every workflow declares permissions at the workflow level and no pull_request job holds a write grant | the whole Source line, as a `**Source**:` line |
| Source: owner selection 30; security pass findings N4 and GOV-3. | Every checkout step sets persist-credentials false | the whole Source line, as a `**Source**:` line |

### Ledger: fetch-error-classification

**A fetch or resolution failure is a typed error**: form b (MODIFIED parent kept with a shorter body and all its scenarios, split-off rules ADDED); old body 1307 characters; new bodies 447, 326, 355, 138.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| The message is the cause's message | A fetch or resolution failure is a typed error, kept in place | no, byte for byte (checked) |
| The cause stays reachable | A fetch or resolution failure is a typed error, kept in place | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "A refused credential is unauthorized" in "FetchNotFound and FetchUnauthorized classify an absent module and a refused credential"
- "A 403 on a module tag lookup classifies as not found" in "FetchNotFound and FetchUnauthorized classify an absent module and a refused credential"
- "A refused connection is unreachable" in "FetchUnreachable and FetchOther classify the other failed registry interactions and Status holds the answered code"
- "A 429 answer or an archive that does not unzip is another failed interaction" in "FetchUnreachable and FetchOther classify the other failed registry interactions and Status holds the answered code"
- "A resolution failure no registry interaction caused" in "An author-defect resolution failure is a ResolutionError and never a FetchError"

| Old clause | New requirement | Change |
| --- | --- | --- |
| `opm/errors` SHALL export `FetchKind` with the values `FetchOther`, `FetchNotFound`, `FetchUnauth... | A fetch or resolution failure is a typed error | verbatim |
| `(*FetchError).Error()` SHALL return the text of `Err` unchanged, and `Unwrap` SHALL return `Err`... | A fetch or resolution failure is a typed error | verbatim |
| Source: enhancement 0021 (the GA exit criteria). | A fetch or resolution failure is a typed error | the whole citation, as a `**Source**:` line |
| `FetchNotFound` means the module, version or package is absent. | FetchNotFound and FetchUnauthorized classify an absent module and a refused credential | verbatim |
| `FetchUnauthorized` means the registry refused the credentials (401), or refused the access (403)... | FetchNotFound and FetchUnauthorized classify an absent module and a refused credential | verbatim |
| CUE's registry client reports a 403 on a module tag lookup as not-found, so that case SHALL class... | FetchNotFound and FetchUnauthorized classify an absent module and a refused credential | verbatim |
| Source: enhancement 0021 (the GA exit criteria). | FetchNotFound and FetchUnauthorized classify an absent module and a refused credential | the whole citation, as a `**Source**:` line |
| `FetchUnreachable` means no HTTP response was received: a refused connection, a DNS or TLS failur... | FetchUnreachable and FetchOther classify the other failed registry interactions and Status holds the answered code | "`FetchUnreachable` means" becomes "`FetchUnreachable` SHALL mean" |
| `FetchOther` is any other failed registry interaction, such as a 429 answer or a fetched archive... | FetchUnreachable and FetchOther classify the other failed registry interactions and Status holds the answered code | verbatim |
| `Status` is the HTTP status the registry answered with, or 0 when there was none or it is unknown. | FetchUnreachable and FetchOther classify the other failed registry interactions and Status holds the answered code | verbatim |
| Source: enhancement 0021 (the GA exit criteria). | FetchUnreachable and FetchOther classify the other failed registry interactions and Status holds the answered code | the whole citation, as a `**Source**:` line |
| An author-defect resolution failure, which no failed registry interaction caused, is a `*Resoluti... | An author-defect resolution failure is a ResolutionError and never a FetchError | "is a `*ResolutionError`" becomes "SHALL be a `*ResolutionError`" |
| Source: enhancement 0021 (the GA exit criteria). | An author-defect resolution failure is a ResolutionError and never a FetchError | the whole citation, as a `**Source**:` line |

**A token endpoint's answer is a registry answer**: form c2 (REMOVED plus ADDED); old body 1425 characters; new bodies 484, 320, 351, 188, 327, 359.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A push whose token endpoint refuses the credentials | A token endpoint's answer classifies the same on a typed chain and on a chain flattened into text, same name | no, byte for byte (checked) |
| A push whose token endpoint refuses the permission | A token endpoint's answer classifies by its status and never as FetchUnreachable, same name | no, byte for byte (checked) |
| A directory module whose dependency registry refuses the token | A token endpoint's answer classifies the same on a typed chain and on a chain flattened into text, same name | no, byte for byte (checked) |
| A registry fetch whose token endpoint refuses the credentials | A token endpoint's answer classifies by its status and never as FetchUnreachable, same name | no, byte for byte (checked) |
| A refused token refresh on a direct call | A refused token refresh is classified by the status the token endpoint answered, same name | no, byte for byte (checked) |
| A token endpoint that fails stays transient | A token endpoint's answer classifies by its status and never as FetchUnreachable, same name | no, byte for byte (checked) |
| A token endpoint that throttles is not transient | A token endpoint's answer classifies by its status and never as FetchUnreachable, same name | no, byte for byte (checked) |
| A forbidden token on a version lookup is not found | A 403 or a 404 on a module version lookup classifies as FetchNotFound, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "Status tells a refused credential from a refused permission" in "A caller tells a refused credential from a refused permission by Status"
- "A 403 or a 404 on a module version lookup drops the status" in "A 403 or a 404 on a module version lookup drops the status, also for the token endpoint's answer"

| Old clause | New requirement | Change |
| --- | --- | --- |
| When a registry uses token authentication and its token endpoint answers the token request with a... | A token endpoint's answer classifies by its status and never as FetchUnreachable | verbatim |
| It SHALL NOT classify as `FetchUnreachable`, and it SHALL satisfy `errors.Is(err, ErrTransient)`... | A token endpoint's answer classifies by its status and never as FetchUnreachable | verbatim |
| This SHALL hold on a chain that keeps the typed cause and on a chain that the embedded CUE flatte... | A token endpoint's answer classifies the same on a typed chain and on a chain flattened into text | "This SHALL" becomes "This (Requirement: A token endpoint's answer classifies by its status and never as FetchUnreachable) SHALL" |
| It SHALL also hold when the client was refreshing a token it held: the embedded registry client t... | A refused token refresh is classified by the status the token endpoint answered | "It SHALL" becomes "It (Requirement: A token endpoint's answer classifies by its status and never as FetchUnreachable) SHALL" |
| A caller tells a refused credential from a refused permission by `Status` (401 or 403). | A caller tells a refused credential from a refused permission by Status | "A caller tells" becomes "(Requirement: A token endpoint's answer classifies by its status and never as FetchUnreachable) A caller SH..." |
| One case is outside this requirement. | A 403 or a 404 on a module version lookup drops the status, also for the token endpoint's answer | "One case is outside this requirement" becomes "One case SHALL be outside this requirement (Requirement: A token endpoint's answer classifies by its status..." |
| The embedded CUE's registry client reports a 403 or a 404 on a module version lookup as `modregis... | A 403 or a 404 on a module version lookup drops the status, also for the token endpoint's answer | verbatim |
| That case SHALL classify as `FetchNotFound`, as the requirement "A fetch or resolution failure is... | A 403 or a 404 on a module version lookup classifies as FetchNotFound | "That case" becomes "That case (Requirement: A 403 or a 404 on a module version lookup drops the status, also for the token endp..."; "for a forbidden tag lookup." becomes "for a forbidden tag lookup (Requirement: FetchNotFound and FetchUnauthorized classify an absent module and..." |

**The library classifies every fetch and resolution failure it returns**: form b (MODIFIED parent kept with a shorter body and all its scenarios, split-off rules ADDED); old body 845 characters; new bodies 498, 274, 190.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Acquiring an unpublished version | The library classifies every fetch and resolution failure it returns, kept in place | no, byte for byte (checked) |
| A directory module whose dependency registry is down | The library classifies every fetch and resolution failure it returns, kept in place | no, byte for byte (checked) |
| A values file whose import registry is down | The library classifies every fetch and resolution failure it returns, kept in place | no, byte for byte (checked) |
| A malformed package is not a fetch failure | The library classifies every fetch and resolution failure it returns, kept in place | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "A classified failure keeps its wrap text and its sentinels" in "The wrap text and the existing sentinels of a classified failure stay unchanged"
- "A shape-gate failure is not classified" in "An evaluation error is not classified"

| Old clause | New requirement | Change |
| --- | --- | --- |
| Every library path that returns a registry fetch failure or a `cue/load` dependency resolution fa... | The library classifies every fetch and resolution failure it returns | verbatim |
| These paths are the registry acquire verbs (`FetchArtifact`, with `Coordinate` set to the fetched... | The library classifies every fetch and resolution failure it returns | verbatim |
| Source: enhancement 0021 (the GA exit criteria). | The library classifies every fetch and resolution failure it returns | the whole citation, as a `**Source**:` line |
| The wrap text SHALL be unchanged, and every existing sentinel (`ErrInvalidPackage`, `ErrWrongKind... | The wrap text and the existing sentinels of a classified failure stay unchanged | "The wrap text" becomes "(Requirement: The library classifies every fetch and resolution failure it returns) The wrap text" |
| Source: enhancement 0021 (the GA exit criteria). | The wrap text and the existing sentinels of a classified failure stay unchanged | the whole citation, as a `**Source**:` line |
| An evaluation error (a built value's error, a shape-gate or concreteness failure) SHALL NOT be cl... | An evaluation error is not classified | "An evaluation error" becomes "(Requirement: The library classifies every fetch and resolution failure it returns) An evaluation error" |
| Source: enhancement 0021 (the GA exit criteria). | An evaluation error is not classified | the whole citation, as a `**Source**:` line |

**An author-defect resolution failure is a typed error**: form c2 (REMOVED plus ADDED); old body 1245 characters; new bodies 308, 302, 102, 165, 173, 156.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| The message is the cause's message | A ResolutionError returns the text of its cause and unwraps to it, same name | no, byte for byte (checked) |
| An author defect is never transient | A ResolutionError is never transient and holds a FetchError only when its cause does, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "ResolutionOther is never built" in "opm/errors exports ResolutionKind and ResolutionError and Classify never builds ResolutionOther"
- "An import of a package no module of the build provides" in "ResolutionImportUnprovided means no module of the build provides an imported package"
- "An imported package that more than one module of the build provides" in "ResolutionImportAmbiguous means more than one module of the build provides an imported package"
- "A dependency module file that does not parse" in "ResolutionModuleFileInvalid means a dependency's module file does not parse"

| Old clause | New requirement | Change |
| --- | --- | --- |
| `opm/errors` SHALL export `ResolutionKind` with the values `ResolutionOther`, `ResolutionImportUn... | opm/errors exports ResolutionKind and ResolutionError and Classify never builds ResolutionOther | verbatim |
| `ResolutionOther` is the zero value, and `Classify` SHALL NOT build it. | opm/errors exports ResolutionKind and ResolutionError and Classify never builds ResolutionOther | verbatim |
| Source: enhancement 0021 (the GA exit criteria). | opm/errors exports ResolutionKind and ResolutionError and Classify never builds ResolutionOther | the whole citation, as a `**Source**:` line |
| `ResolutionImportUnprovided` means no module of the build provides an imported package. | ResolutionImportUnprovided means no module of the build provides an imported package | "`ResolutionImportUnprovided` means" becomes "`ResolutionImportUnprovided` SHALL mean" |
| It covers an import of a package missing from the main module's own path, an import of a module t... | ResolutionImportUnprovided means no module of the build provides an imported package | verbatim |
| Source: enhancement 0021 (the GA exit criteria). | ResolutionImportUnprovided means no module of the build provides an imported package | the whole citation, as a `**Source**:` line |
| `ResolutionImportAmbiguous` means more than one module of the build provides an imported package. | ResolutionImportAmbiguous means more than one module of the build provides an imported package | "`ResolutionImportAmbiguous` means" becomes "`ResolutionImportAmbiguous` SHALL mean" |
| Source: enhancement 0021 (the GA exit criteria). | ResolutionImportAmbiguous means more than one module of the build provides an imported package | the whole citation, as a `**Source**:` line |
| `ResolutionModuleFileInvalid` means a dependency's module file does not parse, whether the depend... | ResolutionModuleFileInvalid means a dependency's module file does not parse | "`ResolutionModuleFileInvalid` means" becomes "`ResolutionModuleFileInvalid` SHALL mean" |
| Source: enhancement 0021 (the GA exit criteria). | ResolutionModuleFileInvalid means a dependency's module file does not parse | the whole citation, as a `**Source**:` line |
| `(*ResolutionError).Error()` SHALL return the text of `Err` unchanged, and `Unwrap` SHALL return... | A ResolutionError returns the text of its cause and unwraps to it | verbatim |
| Source: enhancement 0021 (the GA exit criteria). | A ResolutionError returns the text of its cause and unwraps to it | the whole citation, as a `**Source**:` line |
| A `*ResolutionError` SHALL NOT satisfy `errors.Is(err, ErrTransient)`, and `errors.As` SHALL NOT... | A ResolutionError is never transient and holds a FetchError only when its cause does | verbatim |
| Source: enhancement 0021 (the GA exit criteria). | A ResolutionError is never transient and holds a FetchError only when its cause does | the whole citation, as a `**Source**:` line |

**Classify recognises fetch failures and author-defect resolution failures, and leaves every other error unchanged**: form c2 (REMOVED plus ADDED); old body 3522 characters; new bodies 258, 233, 145, 259, 171, 264, 302, 236, 165, 262, 315, 323, 261, 74, 395, 209, 110.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| An author defect stays a plain error | Classify returns nil for nil and leaves an already classified error, a cancellation and an unrecognised error unchanged, same name | no, byte for byte (checked) |
| A flattened not-found form is classified | The module not found and exact-version package forms are fetch forms, same name | no, byte for byte (checked) |
| An exact version the registry does not provide is not found | The module not found and exact-version package forms are fetch forms, same name | no, byte for byte (checked) |
| An unresolvable import is a typed author defect | Classify returns a ResolutionError that wraps the error for an author-defect resolution failure no failed registry interaction caused, same name | no, byte for byte (checked) |
| A dependency whose module file does not parse is a typed author defect | cannot expand module graph is not matched on its own, same name | no, byte for byte (checked) |
| A dependency module file that does not parse on the direct import path is a typed author defect | The author-defect forms are matched in a fixed order, same name | no, byte for byte (checked) |
| An ambiguous import is a typed author defect | The author-defect forms are matched in a fixed order, same name | no, byte for byte (checked) |
| The first author-defect form decides | The first author-defect form decides when the text carries several, same name | no, byte for byte (checked) |
| A registry failure beside an unprovided import is a fetch failure | A fetch form wins over an author-defect form in the same text, same name | no, byte for byte (checked) |
| An archive that does not unzip beside an unprovided import stays a fetch failure | A fetch form wins over an author-defect form in the same text, same name | no, byte for byte (checked) |
| Cancellation is not a fetch failure | Classify returns nil for nil and leaves an already classified error, a cancellation and an unrecognised error unchanged, same name | no, byte for byte (checked) |
| Classifying twice changes nothing | Classify returns nil for nil and leaves an already classified error, a cancellation and an unrecognised error unchanged, same name | no, byte for byte (checked) |
| A flattened token refusal is not unreachable | The answered token request classifies by its status and never as FetchUnreachable, same name | no, byte for byte (checked) |
| A request that got no response stays unreachable | Any other cannot do HTTP request form is FetchUnreachable and a flattened HTTP status keeps its class, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "A failed registry interaction is returned as a FetchError" in "Classify returns a FetchError that wraps the error for a failed registry interaction"
- "The typed chain is read before the text" in "Classify reads the typed chain first and matches text only when no typed cause is found"
- "A net.Error is unreachable" in "A net.Error is FetchUnreachable except for the answered token request"
- "Only the text fallback matches error text" in "The text fallback is the only place in the library that matches the text of a registry or cue/load error"
- "The fetch forms come before the author-defect forms" in "Classify covers the fetch forms before the author-defect forms"
- "A request line that ends in an HTTP status of 400 or higher" in "The answered token request is the first fetch form"
- "The generic cannot fetch form is FetchOther" in "The generic cannot fetch form is FetchOther"
- "A changed CUE form fails the test" in "A test produces each covered form through the embedded CUE"

| Old clause | New requirement | Change |
| --- | --- | --- |
| `opm/errors` SHALL export `Classify(err error) error`. | Classify returns nil for nil and leaves an already classified error, a cancellation and an unrecognised error unchanged | verbatim |
| It SHALL return nil for nil. | Classify returns nil for nil and leaves an already classified error, a cancellation and an unrecognised error unchanged | verbatim |
| It SHALL return `err` unchanged when the chain already holds a `*FetchError` or a `*ResolutionErr... | Classify returns nil for nil and leaves an already classified error, a cancellation and an unrecognised error unchanged | verbatim |
| Source: enhancement 0021 (the GA exit criteria). | Classify returns nil for nil and leaves an already classified error, a cancellation and an unrecognised error unchanged | the whole citation, as a `**Source**:` line |
| It SHALL return a `*FetchError` that wraps `err` for a failed registry interaction: a fetch that... | Classify returns a FetchError that wraps the error for a failed registry interaction | "It SHALL" becomes "`Classify` SHALL" |
| Source: enhancement 0021 (the GA exit criteria). | Classify returns a FetchError that wraps the error for a failed registry interaction | the whole citation, as a `**Source**:` line |
| It SHALL return a `*ResolutionError` that wraps `err` for an author-defect resolution failure tha... | Classify returns a ResolutionError that wraps the error for an author-defect resolution failure no failed registry interaction caused | "It SHALL" becomes "`Classify` SHALL" |
| Source: enhancement 0021 (the GA exit criteria). | Classify returns a ResolutionError that wraps the error for an author-defect resolution failure no failed registry interaction caused | the whole citation, as a `**Source**:` line |
| When the text carries both a fetch form and an author-defect form, the fetch form SHALL win, the... | A fetch form wins over an author-defect form in the same text | verbatim |
| Source: enhancement 0021 (the GA exit criteria). | A fetch form wins over an author-defect form in the same text | the whole citation, as a `**Source**:` line |
| When the text carries several author-defect forms, the first in the order of step 2 below SHALL d... | The first author-defect form decides when the text carries several | "step 2 below" becomes "step 2 below (Requirement: The author-defect forms are matched in a fixed order)" |
| Source: enhancement 0021 (the GA exit criteria). | The first author-defect form decides when the text carries several | the whole citation, as a `**Source**:` line |
| `Classify` SHALL read the typed chain first: `context.DeadlineExceeded`, an `ociregistry.HTTPErro... | Classify reads the typed chain first and matches text only when no typed cause is found | verbatim |
| Only when no typed cause is found SHALL it match text. | Classify reads the typed chain first and matches text only when no typed cause is found | verbatim |
| Source: enhancement 0021 (the GA exit criteria). | Classify reads the typed chain first and matches text only when no typed cause is found | the whole citation, as a `**Source**:` line |
| A `net.Error` is `FetchUnreachable`, except when the text of the chain is the answered token requ... | A net.Error is FetchUnreachable except for the answered token request | "A `net.Error` is" becomes "A `net.Error` SHALL be"; "step 1 below" becomes "step 1 below (Requirement: The answered token request is the first fetch form)" |
| Source: enhancement 0021 (the GA exit criteria). | A net.Error is FetchUnreachable except for the answered token request | the whole citation, as a `**Source**:` line |
| The text fallback, with that one `net.Error` exception, SHALL be the only place in the library th... | The text fallback is the only place in the library that matches the text of a registry or cue/load error | "that one `net.Error` exception" becomes "that one `net.Error` exception (Requirement: A net.Error is FetchUnreachable except for the answered token..." |
| Source: enhancement 0021 (the GA exit criteria). | The text fallback is the only place in the library that matches the text of a registry or cue/load error | the whole citation, as a `**Source**:` line |
| It SHALL cover the forms the embedded CUE version produces when `cue/load` flattens a failure, in... | Classify covers the fetch forms before the author-defect forms | "It SHALL" becomes "`Classify` SHALL"; list numbers "1.", "2." added |
| The fetch forms. | Classify covers the fetch forms before the author-defect forms | verbatim; list numbers "1.", "2." added |
| The author-defect forms, in this order: | Classify covers the fetch forms before the author-defect forms | "The author-defect forms, in this order:" becomes "The author-defect forms."; list numbers "1.", "2." added |
| Source: enhancement 0021 (the GA exit criteria). | Classify covers the fetch forms before the author-defect forms | the whole citation, as a `**Source**:` line |
| The fetch forms. | The answered token request is the first fetch form | verbatim |
| First is the answered token request: `cannot do HTTP request: ` followed on the same line by a re... | The answered token request is the first fetch form | "First is" becomes "First SHALL be" |
| Source: enhancement 0021 (the GA exit criteria). | The answered token request is the first fetch form | the whole citation, as a `**Source**:` line |
| The embedded registry client writes that form only when the token endpoint of a token-authenticat... | The answered token request classifies by its status and never as FetchUnreachable | "that form" becomes "that form (Requirement: The answered token request is the first fetch form)"; "so it classifies" becomes "so it SHALL classify" |
| Source: enhancement 0021 (the GA exit criteria). | The answered token request classifies by its status and never as FetchUnreachable | the whole citation, as a `**Source**:` line |
| The fetch forms. | Any other cannot do HTTP request form is FetchUnreachable and a flattened HTTP status keeps its class | verbatim |
| Then come `cannot do HTTP request` in any other form, which is `FetchUnreachable`, and an HTTP st... | Any other cannot do HTTP request form is FetchUnreachable and a flattened HTTP status keeps its class | "in any other form" becomes "in any other form (Requirement: The answered token request is the first fetch form)"; "which is `FetchUnreachable`" becomes "which SHALL be `FetchUnreachable`" |
| Source: enhancement 0021 (the GA exit criteria). | Any other cannot do HTTP request form is FetchUnreachable and a flattened HTTP status keeps its class | the whole citation, as a `**Source**:` line |
| They also include `module not found`, and `cannot find module providing package` followed by a pa... | The module not found and exact-version package forms are fetch forms | "They also include" becomes "The fetch forms SHALL also include" |
| Only a standalone `path@version` load produces that form, after it asks the registry for that ver... | The module not found and exact-version package forms are fetch forms | verbatim |
| Source: enhancement 0021 (the GA exit criteria). | The module not found and exact-version package forms are fetch forms | the whole citation, as a `**Source**:` line |
| Last among them is `cannot fetch`, which is `FetchOther`. | The generic cannot fetch form is FetchOther | "Last among them" becomes "Last among the fetch forms"; "which is" becomes "which SHALL be" |
| Source: enhancement 0021 (the GA exit criteria). | The generic cannot fetch form is FetchOther | the whole citation, as a `**Source**:` line |
| The author-defect forms, in this order: | The author-defect forms are matched in a fixed order | "The author-defect forms, in this order:" becomes "The author-defect forms SHALL be, in this order:" |
| - `cannot find module providing package` followed by a path without an exact version, which is `R... | The author-defect forms are matched in a fixed order | verbatim |
| - `ambiguous import`, which is `ResolutionImportAmbiguous`; | The author-defect forms are matched in a fixed order | verbatim |
| - `cannot parse module file`, or `import failed: ` followed directly by a module coordinate at an... | The author-defect forms are matched in a fixed order | verbatim |
| Source: enhancement 0021 (the GA exit criteria). | The author-defect forms are matched in a fixed order | the whole citation, as a `**Source**:` line |
| `cannot expand module graph` SHALL NOT be matched on its own, because it wraps both fetch failure... | cannot expand module graph is not matched on its own | verbatim |
| It classifies through the form it carries. | cannot expand module graph is not matched on its own | "It classifies" becomes "`cannot expand module graph` classifies" |
| Source: enhancement 0021 (the GA exit criteria). | cannot expand module graph is not matched on its own | the whole citation, as a `**Source**:` line |
| A test SHALL produce each covered form through the embedded CUE and SHALL fail when a CUE version... | A test produces each covered form through the embedded CUE | verbatim |
| Source: enhancement 0021 (the GA exit criteria). | A test produces each covered form through the embedded CUE | the whole citation, as a `**Source**:` line |

### Ledger: platform-artifact

**Platform Type Shape**: form c2 (REMOVED plus ADDED); old body 4656 characters; new bodies 323, 152, 208, 297, 392, 396, 356, 451, 406, 299, 302, 48, 191, 135, 489, 306, 264, 494.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Platform struct fields | Platform has the uniform artifact shape of Metadata, Package and Source, same name | no, byte for byte (checked) |
| PlatformMetadata fields | Platform has the uniform artifact shape of Metadata, Package and Source, same name | no, byte for byte (checked) |
| The inventory of a healthy platform reads as fulfilled and routable | Contracts returns a decoded ContractInventory, decoded once at construction, same name | no, byte for byte (checked) |
| An over-subscribed platform is reported, not refused | The inventory reports Unfulfilled and OverSubscribed, same name | no, byte for byte (checked) |
| An undiscriminated platform is reported, not refused | The inventory reports Comparable, same name | no, byte for byte (checked) |
| An unlisted demand leaves the inventory empty | The accessor does not decode defined, same name | no, byte for byte (checked) |
| An inventory that predates the comparable-predicate report is refused | The accessor does not default a report the value does not carry, same name | no, byte for byte (checked) |
| Two majors of one provider catalog are two providers | Providers are counted per registry entry, same name | no, byte for byte (checked) |
| A disabled definer does not hide over-subscription | A disabled entry never counts as a definer, same name | no, byte for byte (checked) |
| An inventory that predates the provider count is refused | A missing providedBy is refused with PlatformCoreTooOldError, same name | no, byte for byte (checked) |
| Two majors sharing contract keys are reported as collisions | The inventory reports Collisions and CollidingEntries, same name | no, byte for byte (checked) |
| A colliding key is left out of the other reports | A colliding key is left out of the other reports and no caller reads Fulfilled or Discriminated as safe while Collisions is non-empty, same name | no, byte for byte (checked) |
| Collision and over-subscription are reported together | The inventory carries the booleans Fulfilled, Routable and Discriminated, and Routable needs no over-subscription and no collision, same name | no, byte for byte (checked) |
| An inventory that predates the collision report reads no collision | An absent collision report decodes as empty and a present one that fails to decode is refused, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "No exported field holds a decoded derived view" in "No exported field of Platform holds a decoded derived CUE view"
- "A platform the constructor did not build decodes Package on its first call" in "A platform the constructor did not build decodes Package once, on its first CoreFloor or Contracts call"
- "DefinedBy and RequiredBy read the enabled entries" in "The inventory reports DefinedBy and RequiredBy"
- "ProvidedBy maps a required contract to the registry keys" in "The inventory reports ProvidedBy"
- "A report is not a refusal" in "The accessor does not refuse on a report"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The library SHALL expose `Platform` in `opm/platform/` with the uniform artifact shape: `{ Metada... | Platform has the uniform artifact shape of Metadata, Package and Source | verbatim |
| `Package` is the source of truth; `Metadata` is a decoded cache of the platform-level metadata; `... | Platform has the uniform artifact shape of Metadata, Package and Source | verbatim |
| No exported field SHALL hold a decoded derived CUE view (`#composedTransformers`, `#contracts`),... | No exported field of Platform holds a decoded derived CUE view | verbatim |
| The contract inventory core derives on `#Platform.#contracts` | Contracts returns a decoded ContractInventory, decoded once at construction | verbatim; list punctuation adjusted at the cut (a ":" or "." added, a "," or "and" dropped) |
| SHALL be readable through a `Contracts()` accessor returning a decoded `ContractInventory`, decod... | Contracts returns a decoded ContractInventory, decoded once at construction | verbatim; list punctuation adjusted at the cut (a ":" or "." added, a "," or "and" dropped) |
| enhancement 0015, "`#Catalog` publishes its contracts as members, beside its transformers", "One... | Contracts returns a decoded ContractInventory, decoded once at construction | the whole citation, as a `**Source**:` line |
| a platform the constructor did not build decodes `Package` once, on its first `CoreFloor()` or `C... | A platform the constructor did not build decodes Package once, on its first CoreFloor or Contracts call | "a platform the constructor did not build decodes" becomes "A platform the constructor did not build SHALL decode"; "contract inventory at construction")" becomes "contract inventory at construction") (Requirement: A Platform the constructor did not build decodes the sam..."; list punctuation adjusted at the cut (a ":" or "." added, a "," or "and" dropped) |
| The contract inventory core derives on `#Platform.#contracts` | The inventory reports DefinedBy and RequiredBy | verbatim; list punctuation adjusted at the cut (a ":" or "." added, a "," or "and" dropped) |
| SHALL be readable through a `Contracts()` accessor returning a decoded `ContractInventory` | The inventory reports DefinedBy and RequiredBy | verbatim; list punctuation adjusted at the cut (a ":" or "." added, a "," or "and" dropped) |
| `DefinedBy` (contract FQN to the registry key of the enabled catalog listing it, for keys exactly... | The inventory reports DefinedBy and RequiredBy | verbatim; list punctuation adjusted at the cut (a ":" or "." added, a "," or "and" dropped) |
| `RequiredBy` (each such defined contract FQN to the implementation FQNs of the enabled transforme... | The inventory reports DefinedBy and RequiredBy | verbatim; list punctuation adjusted at the cut (a ":" or "." added, a "," or "and" dropped) |
| enhancement 0015, "`#Catalog` publishes its contracts as members, beside its transformers", "One... | The inventory reports DefinedBy and RequiredBy | the whole citation, as a `**Source**:` line |
| The contract inventory core derives on `#Platform.#contracts` | The inventory reports ProvidedBy | verbatim; list punctuation adjusted at the cut (a ":" or "." added, a "," or "and" dropped) |
| SHALL be readable through a `Contracts()` accessor returning a decoded `ContractInventory` | The inventory reports ProvidedBy | verbatim; list punctuation adjusted at the cut (a ":" or "." added, a "," or "and" dropped) |
| `ProvidedBy` (every provider-fulfilled contract FQN some enabled transformer requires, whether or... | The inventory reports ProvidedBy | verbatim; list punctuation adjusted at the cut (a ":" or "." added, a "," or "and" dropped) |
| enhancement 0015, "`#Catalog` publishes its contracts as members, beside its transformers", "One... | The inventory reports ProvidedBy | the whole citation, as a `**Source**:` line |
| The contract inventory core derives on `#Platform.#contracts` | The inventory reports Unfulfilled and OverSubscribed | verbatim; list punctuation adjusted at the cut (a ":" or "." added, a "," or "and" dropped) |
| SHALL be readable through a `Contracts()` accessor returning a decoded `ContractInventory` | The inventory reports Unfulfilled and OverSubscribed | verbatim; list punctuation adjusted at the cut (a ":" or "." added, a "," or "and" dropped) |
| `Unfulfilled` (defined provider-fulfilled contract FQNs no enabled entry provides) | The inventory reports Unfulfilled and OverSubscribed | verbatim; list punctuation adjusted at the cut (a ":" or "." added, a "," or "and" dropped) |
| `OverSubscribed` (provider-fulfilled contract FQNs `ProvidedBy` maps to two or more registry entr... | The inventory reports Unfulfilled and OverSubscribed | verbatim; list punctuation adjusted at the cut (a ":" or "." added, a "," or "and" dropped) |
| enhancement 0015, "`#Catalog` publishes its contracts as members, beside its transformers", "One... | The inventory reports Unfulfilled and OverSubscribed | the whole citation, as a `**Source**:` line |
| The contract inventory core derives on `#Platform.#contracts` | The inventory reports Comparable | verbatim; list punctuation adjusted at the cut (a ":" or "." added, a "," or "and" dropped) |
| SHALL be readable through a `Contracts()` accessor returning a decoded `ContractInventory` | The inventory reports Comparable | verbatim; list punctuation adjusted at the cut (a ":" or "." added, a "," or "and" dropped) |
| `Comparable` (every pair of enabled transformers whose match predicates are comparable over at le... | The inventory reports Comparable | verbatim; list punctuation adjusted at the cut (a ":" or "." added, a "," or "and" dropped) |
| enhancement 0015, "`#Catalog` publishes its contracts as members, beside its transformers", "One... | The inventory reports Comparable | the whole citation, as a `**Source**:` line |
| The contract inventory core derives on `#Platform.#contracts` | The inventory reports Collisions and CollidingEntries | verbatim; list punctuation adjusted at the cut (a ":" or "." added, a "," or "and" dropped) |
| SHALL be readable through a `Contracts()` accessor returning a decoded `ContractInventory` | The inventory reports Collisions and CollidingEntries | verbatim; list punctuation adjusted at the cut (a ":" or "." added, a "," or "and" dropped) |
| `Collisions` (every contract FQN that two or more enabled registry entries' catalogs list in thei... | The inventory reports Collisions and CollidingEntries | verbatim; list punctuation adjusted at the cut (a ":" or "." added, a "," or "and" dropped) |
| `CollidingEntries` (each `Collisions` key to the ascending registry keys, path plus major, of the... | The inventory reports Collisions and CollidingEntries | verbatim; list punctuation adjusted at the cut (a ":" or "." added, a "," or "and" dropped) |
| enhancement 0015, "`#Catalog` publishes its contracts as members, beside its transformers", "One... | The inventory reports Collisions and CollidingEntries | the whole citation, as a `**Source**:` line |
| The contract inventory core derives on `#Platform.#contracts` | The inventory carries the booleans Fulfilled, Routable and Discriminated, and Routable needs no over-subscription and no collision | verbatim; list punctuation adjusted at the cut (a ":" or "." added, a "," or "and" dropped) |
| SHALL be readable through a `Contracts()` accessor returning a decoded `ContractInventory` | The inventory carries the booleans Fulfilled, Routable and Discriminated, and Routable needs no over-subscription and no collision | verbatim; list punctuation adjusted at the cut (a ":" or "." added, a "," or "and" dropped) |
| the booleans `Fulfilled`, `Routable` and `Discriminated` | The inventory carries the booleans Fulfilled, Routable and Discriminated, and Routable needs no over-subscription and no collision | verbatim; list punctuation adjusted at the cut (a ":" or "." added, a "," or "and" dropped) |
| `Routable` SHALL be true exactly when `OverSubscribed` and `Collisions` are both empty. | The inventory carries the booleans Fulfilled, Routable and Discriminated, and Routable needs no over-subscription and no collision | verbatim; list punctuation adjusted at the cut (a ":" or "." added, a "," or "and" dropped) |
| enhancement 0015, "`#Catalog` publishes its contracts as members, beside its transformers", "One... | The inventory carries the booleans Fulfilled, Routable and Discriminated, and Routable needs no over-subscription and no collision | the whole citation, as a `**Source**:` line |
| A colliding key is in none of `DefinedBy`, `RequiredBy`, `Unfulfilled` or `Comparable`, so `Fulfi... | A colliding key is left out of the other reports and no caller reads Fulfilled or Discriminated as safe while Collisions is non-empty | "the accessor's documentation" becomes "the `Contracts()` accessor's documentation" |
| A disabled entry never counts as a definer. | A disabled entry never counts as a definer | "never counts" becomes "SHALL never count" |
| Providers are counted per registry entry, the same count the render's single-provider guard reads... | Providers are counted per registry entry | "Providers are counted" becomes "Providers SHALL be counted" |
| The accessor SHALL NOT decode `defined`: its values are the catalogs' member schemas, which a cal... | The accessor does not decode defined | "The accessor SHALL" becomes "The `Contracts()` accessor SHALL" |
| The accessor SHALL NOT refuse on `Fulfilled: false`, `Routable: false`, `Discriminated: false` or... | The accessor does not refuse on a report | "The accessor SHALL" becomes "The `Contracts()` accessor SHALL"; list punctuation adjusted at the cut (a ":" or "." added, a "," or "and" dropped) |
| (enhancement 0015, "An unfulfilled contract is reported, never refused; only over-subscription re... | The accessor does not refuse on a report | verbatim; list punctuation adjusted at the cut (a ":" or "." added, a "," or "and" dropped) |
| whether a generation step withholds a platform package on them is that step's decision, outside t... | The accessor does not refuse on a report | verbatim; list punctuation adjusted at the cut (a ":" or "." added, a "," or "and" dropped) |
| The accessor SHALL NOT default a report the value does not carry: a platform whose `#contracts` p... | The accessor does not default a report the value does not carry | "The accessor SHALL" becomes "The `Contracts()` accessor SHALL" |
| A missing `providedBy` SHALL be refused with the typed `PlatformCoreTooOldError` (field `provided... | A missing providedBy is refused with PlatformCoreTooOldError | "A missing" becomes "(Requirement: The accessor does not default a report the value does not carry) A missing" |
| The one exception is the collision report: an absent `collisions` or `collidingEntries` SHALL dec... | An absent collision report decodes as empty and a present one that fails to decode is refused | "The one exception" becomes "(Requirement: The accessor does not default a report the value does not carry) The one exception" |
| A present collision field that fails to decode SHALL still be refused. | An absent collision report decodes as empty and a present one that fails to decode is refused | verbatim |

**Platform acquisition from a directory returns a source-carrying artifact**: form c2 (REMOVED plus ADDED); old body 799 characters; new bodies 420, 413, 256.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Acquired platform carries its source | AcquirePlatformFromDir stamps the source of a platform with no overlay and the module root as its Root, same name | no, byte for byte (checked) |
| Acquired subpackage names its module root | AcquirePlatformFromDir stamps Pkg as the package directory relative to Root, same name | no, byte for byte (checked) |
| Registry override honored | The registry mapping of platform acquisition is the kernel's and never touches the process environment, same name | no, byte for byte (checked) |
| Shape-gate failures propagate | AcquirePlatformFromDir stamps the source of a platform with no overlay and the module root as its Root, same name | no, byte for byte (checked) |
| Frontends stop composing load and construct | AcquirePlatformFromDir stamps the source of a platform with no overlay and the module root as its Root, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| The kernel SHALL expose `AcquirePlatformFromDir(ctx, dir)`, which loads a `#Platform` CUE package... | AcquirePlatformFromDir stamps the source of a platform with no overlay and the module root as its Root | verbatim; list punctuation adjusted at the cut (a ":" or "." added, a "," or "and" dropped) |
| `Overlay` nil | AcquirePlatformFromDir stamps the source of a platform with no overlay and the module root as its Root | verbatim; list punctuation adjusted at the cut (a ":" or "." added, a "," or "and" dropped) |
| `Root` = the absolute path of the enclosing module root (the nearest ancestor holding `cue.mod/mo... | AcquirePlatformFromDir stamps the source of a platform with no overlay and the module root as its Root | verbatim; list punctuation adjusted at the cut (a ":" or "." added, a "," or "and" dropped) |
| The kernel SHALL expose `AcquirePlatformFromDir(ctx, dir)`, which loads a `#Platform` CUE package... | AcquirePlatformFromDir stamps Pkg as the package directory relative to Root | verbatim; list punctuation adjusted at the cut (a ":" or "." added, a "," or "and" dropped) |
| `Pkg` = the package directory relative to `Root` (empty for the root package) | AcquirePlatformFromDir stamps Pkg as the package directory relative to Root | verbatim; list punctuation adjusted at the cut (a ":" or "." added, a "," or "and" dropped) |
| A directory with no enclosing module is its own root with an empty `Pkg`. | AcquirePlatformFromDir stamps Pkg as the package directory relative to Root | "is its own root" becomes "SHALL be its own root"; list punctuation adjusted at the cut (a ":" or "." added, a "," or "and" dropped) |
| The registry mapping used for the platform's catalog imports SHALL be the kernel's (`WithRegistry... | The registry mapping of platform acquisition is the kernel's and never touches the process environment | "the verb takes" becomes "the verb `AcquirePlatformFromDir(ctx, dir)` takes" |

**Platform carries its render source**: form b (MODIFIED parent kept with a shorter body and all its scenarios, split-off rules ADDED); old body 519 characters; new bodies 476, 106.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Value-constructed platform has no source | Platform carries its render source, kept in place | no, byte for byte (checked) |
| The source type has one name | Platform carries its render source, kept in place | no, byte for byte (checked) |
| Source-less platform cannot render | Platform carries its render source, kept in place | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "Only Render reads the source" in "No kernel operation other than Render reads the Source field of a platform"

| Old clause | New requirement | Change |
| --- | --- | --- |
| `platform.Platform` SHALL expose `Source *module.Source`. The source type has one name, `module.S... | Platform carries its render source | verbatim |
| No other kernel operation reads the field. | No kernel operation other than Render reads the Source field of a platform | "No other kernel operation reads the field" becomes "(Requirement: Platform carries its render source) No other kernel operation SHALL read the `Source` field" |

**Platform constructor takes a bare value**: form b (MODIFIED parent kept with a shorter body and all its scenarios, split-off rules ADDED); old body 566 characters; new bodies 430, 170.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Successful construction | Platform constructor takes a bare value, kept in place | no, byte for byte (checked) |
| Missing metadata is refused | Platform constructor takes a bare value, kept in place | no, byte for byte (checked) |
| The constructor takes one argument | Platform constructor takes a bare value, kept in place | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "The constructor keeps Package and leaves Source nil" in "The constructor keeps Package unchanged, leaves Source nil and performs no API version detection and no binding lookup"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The library SHALL expose `func NewPlatformFromValue(v cue.Value) (*Platform, error)` in `opm/plat... | Platform constructor takes a bare value | verbatim; list punctuation adjusted at the cut (a ":" or "." added, a "," or "and" dropped) |
| When `metadata` is absent or does not decode, or the root `type` is present but not a string, it... | Platform constructor takes a bare value | verbatim; list punctuation adjusted at the cut (a ":" or "." added, a "," or "and" dropped) |
| It SHALL decode `Metadata` from the value's `metadata` field, hoist the root-level `type` into `M... | The constructor keeps Package unchanged, leaves Source nil and performs no API version detection and no binding lookup | "It SHALL decode `Metadata` from the value's `metadata` field, hoist..." becomes "The constructor SHALL set" |
| It SHALL perform no API version detection and no binding lookup. | The constructor keeps Package unchanged, leaves Source nil and performs no API version detection and no binding lookup | "It SHALL" becomes "The constructor SHALL" |

**A platform records its core floor and contract inventory at construction**: form b (MODIFIED parent kept with a shorter body and all its scenarios, split-off rules ADDED); old body 2059 characters; new bodies 421, 371, 447, 436, 463, 379.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| An older-core platform constructs and is refused later | A platform records its core floor and contract inventory at construction, kept in place | no, byte for byte (checked) |
| Contracts and the floor no longer read Package | A platform records its core floor and contract inventory at construction, kept in place | no, byte for byte (checked) |
| Each call returns its own inventory | A platform records its core floor and contract inventory at construction, kept in place | no, byte for byte (checked) |
| Concurrent reads of one platform are race-free | A platform records its core floor and contract inventory at construction, kept in place | no, byte for byte (checked) |
| A hand-built platform keeps its results | A platform records its core floor and contract inventory at construction, kept in place | no, byte for byte (checked) |
| A contracts value that does not evaluate still constructs | A platform records its core floor and contract inventory at construction, kept in place | no, byte for byte (checked) |
| The zero platform reports no contracts | A platform records its core floor and contract inventory at construction, kept in place | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "A contracts refusal does not fail construction" in "A contracts refusal does not fail construction, and recording the facts once lets Render and Contracts stop reading Package"
- "A call returns a value no other call shares" in "Contracts returns the recorded inventory or the recorded refusal without reading Package, and each call returns its own value"
- "CoreFloor reads the recorded floor" in "CoreFloor returns nil when the recorded floor is met and otherwise the typed PlatformCoreTooOldError"
- "A struct-literal platform decodes its facts once" in "A Platform the constructor did not build decodes the same facts from its Package once"
- "A caller that changes Package re-runs the constructor" in "Package stays the source of truth and the Platform documentation says the recorded facts are a cache"

| Old clause | New requirement | Change |
| --- | --- | --- |
| `platform.NewPlatformFromValue`, and therefore `Kernel.AcquirePlatformFromDir`, SHALL read the pl... | A platform records its core floor and contract inventory at construction | verbatim |
| Where the inventory is refused, it SHALL record the refusal (a missing `#contracts` or report fie... | A platform records its core floor and contract inventory at construction | verbatim |
| A contracts refusal SHALL NOT fail construction: the constructor still fails only for the metadat... | A contracts refusal does not fail construction, and recording the facts once lets Render and Contracts stop reading Package | verbatim |
| Recording these facts once is what lets `Render` and `Contracts()` stop reading `Package`, so tha... | A contracts refusal does not fail construction, and recording the facts once lets Render and Contracts stop reading Package | "these facts" becomes "these facts (Requirement: A platform records its core floor and contract inventory at construction)" |
| On a constructed platform, `Contracts()` SHALL return the recorded inventory or the recorded refu... | Contracts returns the recorded inventory or the recorded refusal without reading Package, and each call returns its own value | "the recorded inventory" becomes "the recorded inventory (Requirement: A platform records its core floor and contract inventory at construction)" |
| Each call SHALL return a value no other call shares: an inventory whose maps and slices a caller... | Contracts returns the recorded inventory or the recorded refusal without reading Package, and each call returns its own value | verbatim |
| The library SHALL expose `(*Platform).CoreFloor() error`, which SHALL return nil when the recorde... | CoreFloor returns nil when the recorded floor is met and otherwise the typed PlatformCoreTooOldError | "the recorded floor" becomes "the recorded floor (Requirement: A platform records its core floor and contract inventory at construction)" |
| It SHALL NOT read `Package` on a constructed platform. | CoreFloor returns nil when the recorded floor is met and otherwise the typed PlatformCoreTooOldError | "It SHALL" becomes "`(*Platform).CoreFloor()` SHALL" |
| A `Platform` that the constructor did not build (a struct literal or the zero value) SHALL decode... | A Platform the constructor did not build decodes the same facts from its Package once | "the same facts" becomes "the same facts (Requirement: A platform records its core floor and contract inventory at construction)" |
| So a hand-built platform keeps today's results, and the zero `Platform` returns the `PlatformCore... | A Platform the constructor did not build decodes the same facts from its Package once | verbatim |
| `Package` stays the source of truth: the recorded facts are a cache stamped at construction, like... | Package stays the source of truth and the Platform documentation says the recorded facts are a cache | "the recorded facts" becomes "the recorded facts (Requirement: A platform records its core floor and contract inventory at construction)" |
| The `Platform` documentation SHALL say this, and SHALL say that a `Platform` is used through a po... | Package stays the source of truth and the Platform documentation says the recorded facts are a cache | verbatim |

### Ledger: platform-module-generation

**Generation is pure and deterministic**: form c2 (REMOVED plus ADDED); old body 1165 characters; new bodies 338, 56, 141, 134, 371, 145.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Same input, same bytes | Entries and dependencies are emitted in sorted path order, same name | no, byte for byte (checked) |
| Two catalogs | platform.cue writes one registry entry per subscription, same name | no, byte for byte (checked) |
| Disabled entry still imports its catalog | platform.cue embeds core.#Platform, declares the name and type and imports each entry's catalog under a positional alias, same name | no, byte for byte (checked) |
| Invalid input is refused | Generation renders a platform module's two files from typed input only, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "The helper does no I/O" in "Generation performs no I/O and reads no environment"
- "The module file declares the module path, the language version and every dependency" in "cue.mod/module.cue declares the caller's module path, the language version and every dependency"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The helper SHALL render a platform module's two files (`cue.mod/module.cue`, `platform.cue`) from... | Generation renders a platform module's two files from typed input only | verbatim |
| It SHALL perform no I/O and read no environment. | Generation performs no I/O and reads no environment | "It SHALL" becomes "The helper SHALL" |
| Entries and dependencies SHALL be emitted in sorted path order whatever order they arrive in, so... | Entries and dependencies are emitted in sorted path order | verbatim |
| `platform.cue` SHALL embed `core.#Platform`, declare `metadata.name` and `type`, import each entr... | platform.cue embeds core.#Platform, declares the name and type and imports each entry's catalog under a positional alias | verbatim; list punctuation adjusted at the cut (a ":" or "." added, a "," or "and" dropped) |
| `platform.cue` SHALL | platform.cue writes one registry entry per subscription | verbatim |
| write one `#registry` entry per subscription carrying `enable`, the entry's expected `version` (t... | platform.cue writes one registry entry per subscription | verbatim |
| `cue.mod/module.cue` SHALL declare the caller's module path, the language version `v0.17.0`, and... | cue.mod/module.cue declares the caller's module path, the language version and every dependency | verbatim |

**The core pin defaults to the kernel's verified release**: form c2 (REMOVED plus ADDED); old body 515 characters; new bodies 330, 184.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Default core pin | The generated module file pins core at the kernel's default schema module version and Roots derives that pin plus every entry's catalog, same name | no, byte for byte (checked) |
| Explicit core pin | Roots takes no option and a caller assembles another core pin directly, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| The generated `cue.mod/module.cue` SHALL pin `opmodel.dev/core@v2` at the version carried by the... | The generated module file pins core at the kernel's default schema module version and Roots derives that pin plus every entry's catalog | verbatim |
| The dependency roots `Roots` derives SHALL be that core pin plus every entry's catalog, disabled... | The generated module file pins core at the kernel's default schema module version and Roots derives that pin plus every entry's catalog | verbatim |
| `Roots` SHALL take no option: a caller that needs another core build assembles its `[]Dep` roots... | Roots takes no option and a caller assembles another core pin directly | verbatim |

**The dependency closure is derived from published module files**: form c2 (REMOVED plus ADDED); old body 828 characters; new bodies 387, 466, 121.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Transitive dependency pinned | The helper derives the dependency list by a breadth-first walk over published module files and the walk honours context cancellation, same name | no, byte for byte (checked) |
| Root wins a shared path | The helper derives the dependency list by a breadth-first walk over published module files and the walk honours context cancellation, same name | no, byte for byte (checked) |
| Unpublished build refused | A requirement naming an unpublished build fails with the module path and version, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "The result is the list a cue mod tidy would write" in "The result is the tidied list a cue mod tidy would write for the same roots"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The helper SHALL derive the full dependency list from the roots by a breadth-first walk over each... | The helper derives the dependency list by a breadth-first walk over published module files and the walk honours context cancellation | verbatim |
| The walk SHALL honour context cancellation. | The helper derives the dependency list by a breadth-first walk over published module files and the walk honours context cancellation | verbatim |
| The result SHALL be the tidied list a `cue mod tidy` would write for the same roots (enhancement... | The result is the tidied list a cue mod tidy would write for the same roots | "The result SHALL" becomes "The result (Requirement: The helper derives the dependency list by a breadth-first walk over published modu..." |
| A root or transitive requirement naming an unpublished build SHALL fail with an error naming the... | A requirement naming an unpublished build fails with the module path and version | verbatim |

**Registry access is caller-configured**: form b (MODIFIED parent kept with a shorter body and all its scenarios, split-off rules ADDED); old body 590 characters; new bodies 323, 266.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Fixture graph without a registry | Registry access is caller-configured, kept in place | no, byte for byte (checked) |
| Registry source carries the caller's configuration | Registry access is caller-configured, kept in place | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "A nil environment is the caller's choice" in "The helper reads no process environment and hard-codes no client type, and a nil environment is the caller's choice"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The module-file source the closure reads through SHALL be constructed from caller-supplied config... | Registry access is caller-configured | verbatim |
| Tests SHALL be able to supply a fixture module-file graph without any registry. | Registry access is caller-configured | verbatim |
| The helper SHALL NOT read the process environment or hard-code a client type itself; a caller tha... | The helper reads no process environment and hard-codes no client type, and a nil environment is the caller's choice | verbatim |

### Ledger: registry-module-loading

**In-Memory Load Without a Temporary Directory**: form c2 (REMOVED plus ADDED); old body 1046 characters; new bodies 132, 458, 478.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| No temporary directory created | The registry module loader loads the fetched module in memory and writes no temporary directory, same name | no, byte for byte (checked) |
| Staged overlay carries the module's CUE files only | The loader injects the fetched module's CUE files under a deterministic synthetic root and leaves load.Config.FS nil, same name | no, byte for byte (checked) |
| An existing synthetic root is refused | The synthetic root is an absolute path that does not exist on disk and the loader refuses the acquire when anything exists at it, same name | no, byte for byte (checked) |
| An existing synthetic root refuses a catalog acquire | The synthetic root is an absolute path that does not exist on disk and the loader refuses the acquire when anything exists at it, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| The registry module loader SHALL load the fetched module in memory and SHALL NOT write the module's... | The registry module loader loads the fetched module in memory and writes no temporary directory | verbatim |
| It SHALL inject the fetched module's CUE files (every `.cue` file under the module root, the module'... | The loader injects the fetched module's CUE files under a deterministic synthetic root and leaves load.Config.FS nil | "It SHALL" becomes "The registry module loader SHALL" |
| The staged overlay the loader returns on the module's `Source` SHALL be that same set of files. | The loader injects the fetched module's CUE files under a deterministic synthetic root and leaves load.Config.FS nil | verbatim |
| The synthetic root SHALL be an absolute path on every operating system (on Windows it carries the cu... | The synthetic root is an absolute path that does not exist on disk and the loader refuses the acquire when anything exists at it | verbatim |
| Because cue/load reads a real directory beneath an overlay root, the loader SHALL refuse the acquire... | The synthetic root is an absolute path that does not exist on disk and the loader refuses the acquire when anything exists at it | verbatim |
| The same holds for every artifact kind the loader fetches from a registry, a catalog included. | The synthetic root is an absolute path that does not exist on disk and the loader refuses the acquire when anything exists at it | verbatim |

**Registry-Loaded Modules Pass the Module Shape Gate**: form c2 (REMOVED plus ADDED); old body 631 characters; new bodies 401, 267, 105.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Wrong artifact kind rejected | The registry module loader evaluates and shape-gates the fetched module through the kernel's one routine, same name | no, byte for byte (checked) |
| Missing identity field rejected | The registry module loader evaluates and shape-gates the fetched module through the kernel's one routine, same name | no, byte for byte (checked) |
| Registry and directory acquisition fail identically | The registry and directory acquisition paths differ only in where the package files come from, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "Full schema validation stays the Kernel's contract" in "The registry module loader performs no full schema validation"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The registry module loader SHALL evaluate and shape-gate the fetched module through the kernel's one... | The registry module loader evaluates and shape-gates the fetched module through the kernel's one routine | verbatim |
| The two acquisition paths SHALL differ only in where the package files come from: a fetched overlay... | The registry and directory acquisition paths differ only in where the package files come from | pointer (Requirement: <exact header>) added after "The two acquisition paths" |
| It SHALL NOT perform full schema validation, which remains the Kernel's contract. | The registry module loader performs no full schema validation | "It SHALL NOT" becomes "The registry module loader SHALL NOT" |

**Module Identity Verification**: form c2 (REMOVED plus ADDED); old body 1610 characters; new bodies 496, 267, 384, 223, 433, 244.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Address mismatch refused at acquire | The loader verifies a module's declared identity against its fetched coordinate, same name | no, byte for byte (checked) |
| Version mismatch refused at acquire | The loader verifies a module's declared identity against its fetched coordinate, same name | no, byte for byte (checked) |
| Honest artifact loads | The loader verifies a module's declared identity against its fetched coordinate, same name | no, byte for byte (checked) |
| A non-module kind traverses the shared fetch unchecked | The identity verification is scoped to modules, same name | no, byte for byte (checked) |
| Older-line parent-path declaration verified by convention | A major-free metadata.modulePath has no alternative verification and no module address is composed from a parent path and a name, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "Every frontend inherits one verification" in "The identity verification sits on the single module entrypoint of the shared fetch routine"
- "The check is found in the module entrypoint" in "The identity check is distinguishable in the code rather than by convention"
- "The kernel does not stamp the declared identity" in "The kernel is a verifier and never a stamper of the declared identity"

| Old clause | New requirement | Change |
| --- | --- | --- |
| After the artifact shape gate passes, the registry module loader SHALL verify the acquired artifact'... | The loader verifies a module's declared identity against its fetched coordinate | verbatim |
| A disagreement SHALL fail the load with a typed identity error naming both the declared and the fetc... | The loader verifies a module's declared identity against its fetched coordinate | verbatim |
| The verification SHALL sit on the single module entrypoint of the shared fetch routine, so every fro... | The identity verification sits on the single module entrypoint of the shared fetch routine | pointer (Requirement: <exact header>) added after "The verification" |
| The verification SHALL be scoped to modules. | The identity verification is scoped to modules | pointer (Requirement: <exact header>) added after "The verification" |
| The kind-parameterized fetch routine underneath the module entrypoint SHALL NOT run it, so an artifa... | The identity verification is scoped to modules | verbatim |
| The two behaviours SHALL be distinguishable in the code rather than by convention: the check lives i... | The identity check is distinguishable in the code rather than by convention | pointer (Requirement: <exact header>) added after "The two behaviours" |
| There SHALL be no alternative verification for a major-free `metadata.modulePath`: the OPM schema th... | A major-free metadata.modulePath has no alternative verification and no module address is composed from a parent path and a name | pointer (Requirement: <exact header>) added after "the same typed error" |
| The library SHALL NOT compose a module address from a parent path and a name. | A major-free metadata.modulePath has no alternative verification and no module address is composed from a parent path and a name | verbatim |
| The kernel SHALL NOT write or correct either value: the schema declares identity, the reader verifie... | The kernel is a verifier and never a stamper of the declared identity | pointer (Requirement: <exact header>) added after "either value" |

**Acquire a module from the registry**: form c2 (REMOVED plus ADDED); old body 1608 characters; new bodies 89, 235, 270, 429, 301, 423, 208, 245, 69, 219.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| One fetch routine serves every kind | One fetch routine serves every artifact kind, same name | no, byte for byte (checked) |
| Acquired module carries reusable staged source | AcquireModuleFromRegistry returns the module with its staged source attached, same name | no, byte for byte (checked) |
| Registry mapping is the kernel's | The registry mapping of module acquisition is the kernel's, same name | no, byte for byte (checked) |
| No value-only loader | No value-only loader is exported from any package under opm, same name | no, byte for byte (checked) |
| Staged source drives transitive resolution during synthesis | The acquired module's staged source is consumable by SynthesizeInstance, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "Each artifact kind has one registry acquisition entry point" in "Each artifact kind has one registry acquisition entry point"
- "Acquisition fetches a published module" in "AcquireModuleFromRegistry fetches a published module via CUE's native module machinery"
- "Acquisition loads the module in memory as the main module" in "AcquireModuleFromRegistry loads the module in memory as the main module"
- "Acquisition runs the shape gate and verifies the identity" in "AcquireModuleFromRegistry runs the shape gate and verifies the declared identity"
- "The staged source is the one used to build the module value" in "The staged source is the tree used to build the module value"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The library SHALL provide exactly one registry acquisition entry point per artifact kind. | Each artifact kind has one registry acquisition entry point | verbatim |
| Every kind's entry point SHALL be served by one fetch routine, parameterized by the shape it gates t... | One fetch routine serves every artifact kind | verbatim |
| Fetching and staging a published CUE module artifact reads no `kind`, so a second fetch implementati... | One fetch routine serves every artifact kind | verbatim |
| For modules that entry point is `Kernel.AcquireModuleFromRegistry(ctx, path, version string) (*modul... | AcquireModuleFromRegistry fetches a published module via CUE's native module machinery | cut at a list boundary (one item of the list), "is" becomes "SHALL be", pointer (Requirement: <exact header>) added after "that entry point" |
| For modules that entry point is `Kernel.AcquireModuleFromRegistry(ctx, path, version string) (*modul... | AcquireModuleFromRegistry loads the module in memory as the main module | cut at a list boundary (one item of the list), "is" becomes "SHALL be", pointer (Requirement: <exact header>) added after "that entry point" |
| For modules that entry point is `Kernel.AcquireModuleFromRegistry(ctx, path, version string) (*modul... | AcquireModuleFromRegistry runs the shape gate and verifies the declared identity | cut at a list boundary (one item of the list), "is" becomes "SHALL be", pointer (Requirement: <exact header>) added after "that entry point" |
| For modules that entry point is `Kernel.AcquireModuleFromRegistry(ctx, path, version string) (*modul... | AcquireModuleFromRegistry returns the module with its staged source attached | cut at a list boundary (one item of the list), "is" becomes "SHALL be", pointer (Requirement: <exact header>) added after "that entry point" |
| The staged source SHALL be the same tree used to build the module value, so no second fetch is requi... | The staged source is the tree used to build the module value | pointer (Requirement: <exact header>) added after "The staged source" |
| The registry mapping SHALL be the kernel's (`WithRegistry`), applied through the resolver and load c... | The registry mapping of module acquisition is the kernel's | pointer (Requirement: <exact header>) added after "The registry mapping" |
| No value-only loader SHALL be exported from any package under `opm/`. | No value-only loader is exported from any package under opm | verbatim |
| The acquired module's staged source SHALL be consumable by `SynthesizeInstance` to construct an inst... | The acquired module's staged source is consumable by SynthesizeInstance | verbatim |

### Ledger: render-parity

**Pure-CUE unification is the render oracle**: form b (MODIFIED parent kept with a shorter body and all its scenarios, split-off rules ADDED); old body 586 characters; new bodies 452, 139.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Agreement on a shipped transformer | Pure-CUE unification is the render oracle, kept in place | no, byte for byte (checked) |
| Disagreement is attributed to the kernel | Pure-CUE unification is the render oracle, kept in place | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "The harness catches a reintroduced Go-side transformation" in "The harness is the standing tripwire against a Go-side transformation being reintroduced"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The test suite SHALL include a differential parity harness that renders each parity case twice, once... | Pure-CUE unification is the render oracle (kept) | verbatim |
| When the two disagree the harness SHALL report the kernel as the defective side, naming the case, th... | Pure-CUE unification is the render oracle (kept) | verbatim |
| The harness is the standing tripwire against any Go-side transformation of a component, instance or... | The harness is the standing tripwire against a Go-side transformation being reintroduced | "The harness is" becomes "The harness SHALL be" |

**The old path was proven equal to Render before deletion**: form c2 (REMOVED plus ADDED); old body 516 characters; new bodies 400, 228.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Proof gated the deletion | The proof of the comparison lives in the archive and after deletion the harness compares Render against the oracle only, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "Both renderers were compared per case before the removal" in "The old render path was compared with Render on every parity case before it was removed"

| Old clause | New requirement | Change |
| --- | --- | --- |
| Before the old render path was removed, the harness SHALL have rendered every parity case through bo... | The old render path was compared with Render on every parity case before it was removed | verbatim |
| The proof lives in the archive as the record; after deletion the harness compares `Render` against t... | The proof of the comparison lives in the archive and after deletion the harness compares Render against the oracle only | pointer (Requirement: <exact header>) added after "The proof"; "lives" becomes "SHALL live" and "compares" becomes "SHALL compare" |

**The shipped catalog stays discriminated**: form b (MODIFIED parent kept with a shorter body and all its scenarios, split-off rules ADDED); old body 1251 characters; new bodies 446, 442, 225, 310.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| The published catalog is discriminated | The shipped catalog stays discriminated, kept in place | no, byte for byte (checked) |
| A comparable pair in a catalog release fails the harness | The shipped catalog stays discriminated, kept in place | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "The inventory is read in a test function of its own" in "The discrimination read lives in a test function of its own beside the render cases"
- "The discrimination assertion is skipped where the shipped group is skipped" in "The discrimination assertion is gated as the shipped group is"
- "A catalog build with two comparable transformers is attributed to the catalog release" in "A catalog build that ships comparable transformers fails the harness naming both transformers and the shared contracts"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The shipped parity group SHALL, in a test function of its own beside the one rendering its cases, re... | The shipped catalog stays discriminated (kept) | cut at the list boundary ", and SHALL assert" (second half of the sentence); the subject "The shipped parity group" repeated; a pointer after "the inventory" |
| The shipped parity group SHALL, in a test function of its own beside the one rendering its cases, re... | The discrimination read lives in a test function of its own beside the render cases | cut at the list boundary ", and SHALL assert" (first half of the sentence); "." closes it |
| The read SHALL NOT live inside the render-case function: a catalog release that breaks discriminatio... | The discrimination read lives in a test function of its own beside the render cases | verbatim |
| The assertion SHALL be gated exactly as the shipped group is: skipped under `-short` and when the pu... | The discrimination assertion is gated as the shipped group is | pointer (Requirement: <exact header>) added after "The assertion" |
| A published catalog build that ships two transformers whose predicates are comparable over a shared... | A catalog build that ships comparable transformers fails the harness naming both transformers and the shared contracts | verbatim |

**The shipped case table names the catalog build the parity module pins**: form c2 (REMOVED plus ADDED); old body 610 characters; new bodies 376, 383.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A catalog bump edits only module files | The shipped case table names each transformer by its catalog-relative name and the harness builds each id from the parity module's pin, same name | no, byte for byte (checked) |
| A pin naming no shipped transformer | The shipped case table names each transformer by its catalog-relative name and the harness builds each id from the parity module's pin, same name | no, byte for byte (checked) |
| The oracle reports another build than the pin | The harness checks that the catalog build the oracle reports equals the pin, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| The shipped case table SHALL name each transformer by its catalog-relative name only. | The shipped case table names each transformer by its catalog-relative name and the harness builds each id from the parity module's pin | verbatim |
| The harness SHALL build every transformer id from the `opmodel.dev/catalogs/opm` version the parity... | The shipped case table names each transformer by its catalog-relative name and the harness builds each id from the parity module's pin | verbatim |
| That module is the one the oracle builds in. | The shipped case table names each transformer by its catalog-relative name and the harness builds each id from the parity module's pin | verbatim |
| No case row, helper or assertion in the parity harness SHALL spell a `catalogs/opm` version as a lit... | The shipped case table names each transformer by its catalog-relative name and the harness builds each id from the parity module's pin | verbatim |
| The harness SHALL also check, before comparing any case, that the catalog build the oracle reports e... | The harness checks that the catalog build the oracle reports equals the pin | pointer (Requirement: <exact header>) added after "equals that pin" |

### Ledger: catalog-acquisition

**Catalog acquisition from a registry and from a directory**: form b (MODIFIED parent kept with a shorter body and all its scenarios, split-off rules ADDED); old body 588 characters; new bodies 396, 141, 178.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A published catalog is acquired by path and version | Catalog acquisition from a registry and from a directory, kept in place | no, byte for byte (checked) |
| A catalog directory is acquired | Catalog acquisition from a registry and from a directory, kept in place | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "The two acquisition paths read the package files from different places" in "The two catalog acquisition paths differ only in where the package files come from"
- "Catalog acquisition writes nothing to the caller's directory" in "Catalog acquisition writes nothing to the caller's directory and builds in a context created for the call"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The kernel SHALL expose `AcquireCatalogFromRegistry(ctx, modPath, version)` and `AcquireCatalogFromD... | Catalog acquisition from a registry and from a directory (kept) | verbatim |
| Both SHALL reuse the existing acquisition plumbing: the registry path fetches through the same routi... | Catalog acquisition from a registry and from a directory (kept) | verbatim |
| The two paths SHALL differ only in where the package files come from. | The two catalog acquisition paths differ only in where the package files come from | pointer (Requirement: <exact header>) added after "The two paths" |
| Neither SHALL write to the caller's directory, and each SHALL build in a `cue.Context` created for t... | Catalog acquisition writes nothing to the caller's directory and builds in a context created for the call | "Neither SHALL" becomes "Neither `AcquireCatalogFromRegistry` nor `AcquireCatalogFromDir` SHALL" |

**The provider-fulfilled contract set is derived from the catalog**: form c2 (REMOVED plus ADDED); old body 1629 characters; new bodies 333, 246, 409, 313, 232, 123, 448.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A provider catalog's contracts are derived | The catalog artifact exposes the set of contracts it implements as a provider, same name | no, byte for byte (checked) |
| Core's provider set is read when the catalog carries it | The provider set is read from provides when the catalog carries it, same name | no, byte for byte (checked) |
| A catalog built against an older core falls back | The provider set falls back to a derivation from the catalog's #transformers for an older core or no provides field, same name | no, byte for byte (checked) |
| Core's provider set and the fallback agree | Both sources of the provider set yield the same set, same name | no, byte for byte (checked) |
| The derived order is stable | The provider set is deterministically ordered and deduplicated, same name | no, byte for byte (checked) |
| A catalog implementing no provider contract derives an empty set | A catalog implementing no provider-fulfilled contract derives an empty set and no error, same name | no, byte for byte (checked) |
| An unreadable provider set is reported | A provider set that cannot be read is an error with no partial set returned, same name | no, byte for byte (checked) |
| Transformers that do not evaluate are reported over a present field | A provider set that cannot be read is an error with no partial set returned, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "A demand whose fulfilment is not a concrete string is reported" in "A provider set that cannot be read is an error with no partial set returned"
- "A committed module file that cannot be read is reported" in "A provider set that cannot be read is an error with no partial set returned"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The catalog artifact SHALL expose the set of contracts it implements as a provider: every contract r... | The catalog artifact exposes the set of contracts it implements as a provider | verbatim |
| When the catalog was built against a core that derives `provides` (core `v2.0.0-beta.3` and later) a... | The provider set is read from provides when the catalog carries it | pointer (Requirement: <exact header>) added after "the set" |
| The set SHALL instead be derived from the catalog's `#transformers` by the library's fallback, which... | The provider set falls back to a derivation from the catalog's #transformers for an older core or no provides field | pointer (Requirement: <exact header>) added after "The set" |
| For any catalog built against a core that derives `provides`, both sources SHALL yield the same set. | Both sources of the provider set yield the same set | pointer (Requirement: <exact header>) added after "both sources" |
| The result SHALL be deterministically ordered and deduplicated, so two derivations of one catalog co... | The provider set is deterministically ordered and deduplicated | pointer (Requirement: <exact header>) added after "The result" |
| Derivation SHALL be a report: a catalog implementing no provider-fulfilled contract SHALL return an... | A catalog implementing no provider-fulfilled contract derives an empty set and no error | verbatim |
| A provider set that cannot be read SHALL be an error with no partial set returned, from either sourc... | A provider set that cannot be read is an error with no partial set returned | verbatim |

**The catalog's committed dependency requirements are readable**: form c2 (REMOVED plus ADDED); old body 1124 characters; new bodies 361, 481, 241.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Declared requirements are readable off an acquired catalog | The catalog artifact exposes the requirements recorded in its own cue.mod/module.cue and the library exposes the requirements only, same name | no, byte for byte (checked) |
| A module file the render stage would refuse is refused | The catalog's module file is read and parsed by the same reader the render stage uses for its input modules, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "A dependency served by a local replacement maps to the empty string" in "A dependency a local replacement serves may carry no version and maps to the empty string"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The catalog artifact SHALL expose the requirements recorded in its own `cue.mod/module.cue`, as modu... | The catalog artifact exposes the requirements recorded in its own cue.mod/module.cue and the library exposes the requirements only | the parenthesis "(enhancement 0015 ... defines)" moved whole, word for word, to the Source line; "." closes the sentence |
| The library SHALL expose the requirements only; comparing them, and deciding what a mismatch means,... | The catalog artifact exposes the requirements recorded in its own cue.mod/module.cue and the library exposes the requirements only | verbatim |
| The module file SHALL be read and parsed by the same reader the render stage uses for its input modu... | The catalog's module file is read and parsed by the same reader the render stage uses for its input modules | pointer (Requirement: <exact header>) added after "The module file" |
| A dependency a local replacement serves may carry no version and maps to the empty string. | A dependency a local replacement serves may carry no version and maps to the empty string | pointer (Requirement: <exact header>) added after "A dependency a"; "and maps" becomes "and SHALL map" |

### Ledger: fixture-pin-maintenance

**Test module pins update in one resolution pass**: form c2 (REMOVED plus ADDED); old body 612 characters; new bodies 359, 406.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Nothing to update | The update task updates each test module with one cue mod get that names every direct opmodel.dev dependency, followed by one cue mod tidy, same name | no, byte for byte (checked) |
| A newer catalog raises a third-party pin only as far as it requires | A non-core OPM dependency is requested at its newest version within its major and a third-party dependency is not named in the get, same name | no, byte for byte (checked) |
| A third-party pin is not chased | A non-core OPM dependency is requested at its newest version within its major and a third-party dependency is not named in the get, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| The library's dependency-update task SHALL update every test CUE module it discovers (the `CUE_MODUL... | The update task updates each test module with one cue mod get that names every direct opmodel.dev dependency, followed by one cue mod tidy | verbatim |
| That one call SHALL name every direct `opmodel.dev/*` dependency of the module, followed by one `cue... | The update task updates each test module with one cue mod get that names every direct opmodel.dev dependency, followed by one cue mod tidy | verbatim |
| A module whose OPM dependencies are several releases stale SHALL therefore update in a single run. | The update task updates each test module with one cue mod get that names every direct opmodel.dev dependency, followed by one cue mod tidy | verbatim |
| Each non-core OPM dependency SHALL be requested at the newest published version within its major. | A non-core OPM dependency is requested at its newest version within its major and a third-party dependency is not named in the get | verbatim |
| A third-party dependency (such as `cue.dev/x/k8s.io`) SHALL NOT be named in the get; it moves only w... | A non-core OPM dependency is requested at its newest version within its major and a third-party dependency is not named in the get | pointer (Requirement: <exact header>) added after "in the get" |

**Core is held at the kernel default**: form c2 (REMOVED plus ADDED); old body 713 characters; new bodies 299, 449, 237.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Default advanced first | The update task requests core at the release the kernel default names and core moves only together with the default, same name | no, byte for byte (checked) |
| A catalog built on a newer core | A held resolution that fails because a dependency requires a newer core exits non-zero and names the dependency, same name | no, byte for byte (checked) |
| Unreadable default | The update task exits non-zero when the default cannot be read from the Go source, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| The update task SHALL request `opmodel.dev/core@v2` at the release `schema.DefaultSchemaModule` name... | The update task requests core at the release the kernel default names and core moves only together with the default | verbatim |
| Core in the test modules moves only together with the default, never on its own. | The update task requests core at the release the kernel default names and core moves only together with the default | "Core in the test modules moves" becomes "Core in the test modules SHALL move" |
| When the held resolution fails because a dependency requires a newer core, the task SHALL exit non-z... | A held resolution that fails because a dependency requires a newer core exits non-zero and names the dependency | pointer (Requirement: <exact header>) added after "When the held resolution fails"; "the task SHALL" becomes "the update task SHALL" |
| The failure SHALL print the module directory and CUE's own error, and SHALL name each dependency who... | A held resolution that fails because a dependency requires a newer core exits non-zero and names the dependency | verbatim |
| When the default cannot be read from the Go source, the task SHALL exit non-zero with a message sayi... | The update task exits non-zero when the default cannot be read from the Go source | pointer (Requirement: <exact header>) added after "the default cannot" |

**Intentionally old version literals are declared**: form c2 (REMOVED plus ADDED); old body 1033 characters; new bodies 324, 423, 466, 211.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Floor test keeps its old core | The repository root carries a .cascade-frozen file listing every test file that keeps a version literal on purpose, same name | no, byte for byte (checked) |
| Synthetic literal is listed | A file qualifies for the frozen list when its literal names an older release a test needs or is synthetic, same name | no, byte for byte (checked) |
| Current-core expectation is not frozen | An expectation of the current core or of the pinned catalog build is not a frozen literal, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "Release-cascade tooling leaves a listed literal alone" in "Release-cascade tooling does not rewrite a version literal at a listed location"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The repository root SHALL carry a `.cascade-frozen` file listing every test file that keeps a versio... | The repository root carries a .cascade-frozen file listing every test file that keeps a version literal on purpose | verbatim |
| Each entry SHALL name a repo-relative `path`, the `pins` (module paths) frozen at that location, and... | The repository root carries a .cascade-frozen file listing every test file that keeps a version literal on purpose | verbatim |
| A file qualifies when its literal names an older release a floor, skew or pre-collision test needs,... | A file qualifies for the frozen list when its literal names an older release a test needs or is synthetic | pointer (Requirement: <exact header>) added after "A file qualifies" |
| An expectation of the current core or of the pinned catalog build, and a module file a test authors... | An expectation of the current core or of the pinned catalog build is not a frozen literal | verbatim |
| A listed file can still read the derived version in other places, for example to find the pin it rep... | An expectation of the current core or of the pinned catalog build is not a frozen literal | pointer (Requirement: <exact header>) added after "A listed file" |
| Release-cascade tooling SHALL NOT rewrite a version literal at a listed location. | Release-cascade tooling does not rewrite a version literal at a listed location | pointer (Requirement: <exact header>) added after "at a listed location" |


### Ledger: migration-docs

**Migration documentation lives in per-change fragments under migrations/**: form c2 (REMOVED plus ADDED); old body 538 characters; new bodies 390, 147.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Policy is discoverable at the directory root | Migration fragments are one per breaking change and carry no version number when authored, same name | no, byte for byte (checked) |
| The old monolith redirects | The repository maintains no monolithic migrations document and MIGRATIONS.md is a tombstone, same name | no, byte for byte (checked) |
| No version numbers at authoring time | Migration fragments are one per breaking change and carry no version number when authored, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| Migration documentation SHALL live in the `migrations/` directory as one fragment per breaking chang... | Migration fragments are one per breaking change and carry no version number when authored | verbatim |
| Fragments SHALL be authored under `migrations/unreleased/` and SHALL NOT carry version numbers in th... | Migration fragments are one per breaking change and carry no version number when authored | verbatim |
| The repository SHALL NOT maintain a monolithic migrations document; `MIGRATIONS.md` remains only as ... | The repository maintains no monolithic migrations document and MIGRATIONS.md is a tombstone | verbatim |

**Pre-GA dormancy**: form c2 (REMOVED plus ADDED); old body 674 characters; new bodies 242, 280, 416.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Breaking change during alpha | No migration fragment is required before GA and the breaking-change record is the CHANGELOG plus the OpenSpec archive, same name | no, byte for byte (checked) |
| Repo guidance matches the policy | Repo guidance states the pre-GA migration policy, same name | no, byte for byte (checked) |
| Breaking change during beta | From the first beta a breaking change lands only as a feat! commit whose BREAKING CHANGE footer is the migration note, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| Until the library's first GA release, through its alpha and beta lines alike, no migration fragment ... | No migration fragment is required before GA and the breaking-change record is the CHANGELOG plus the OpenSpec archive | verbatim |
| The breaking-change record pre-GA SHALL be `CHANGELOG.md` (release-please) plus the OpenSpec archive... | No migration fragment is required before GA and the breaking-change record is the CHANGELOG plus the OpenSpec archive | verbatim |
| From the first beta on, a breaking change SHALL land only as a `feat!` commit whose `BREAKING CHANGE... | From the first beta a breaking change lands only as a feat! commit whose BREAKING CHANGE footer is the migration note | verbatim |
| Repo guidance (`AGENTS.md`, `README.md`, `migrations/README.md`) SHALL state this policy rather than... | Repo guidance states the pre-GA migration policy | verbatim; pointer (Requirement: <exact header>) added after "state this policy" |

**Enforcement design is recorded and armed at GA**: form c2 (REMOVED plus ADDED); old body 787 characters; new bodies 429, 156, 317.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Arming condition is written down | The GA-time enforcement design is recorded in ADR-004 and summarized in migrations/README.md before GA, same name | no, byte for byte (checked) |
| No gate runs pre-GA | The enforcement mechanisms do not run before GA, same name | no, byte for byte (checked) |
| The GA exit criteria are linked | ADR-004, README.md and migrations/README.md name enhancement 0021 as the GA exit criteria, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| The GA-time enforcement design SHALL be recorded in ADR-004 and summarized in `migrations/README.md`... | The GA-time enforcement design is recorded in ADR-004 and summarized in migrations/README.md before GA | the em-dash of "`Migration: none — <reason>`" becomes a comma: "`Migration: none, <reason>`" |
| These mechanisms SHALL NOT run pre-GA. | The enforcement mechanisms do not run before GA | verbatim; pointer (Requirement: <exact header>) added after "These mechanisms" |
| ADR-004, `README.md` and `migrations/README.md` SHALL name enhancement 0021 ("A beta line reaches GA... | ADR-004, README.md and migrations/README.md name enhancement 0021 as the GA exit criteria | verbatim |

### Ledger: release-pipeline

**Release PRs refuse dev and replaced pins**: form c2 (REMOVED plus ADDED); old body 1026 characters; new bodies 187, 311, 292, 84, 159, 139.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Clean release PR passes | The release-pin check runs as a step of the Go tests job, same name | no, byte for byte (checked) |
| Replace directive on a release PR | The release-pin check fails the Go tests check on a replace directive, an OPM Go pseudo-version or a dev CUE pin, same name | no, byte for byte (checked) |
| OPM Go module at a pseudo-version | The release-pin check fails the Go tests check on a replace directive, an OPM Go pseudo-version or a dev CUE pin, same name | no, byte for byte (checked) |
| Third-party pseudo-versions are allowed | The release-pin check fails the Go tests check on a replace directive, an OPM Go pseudo-version or a dev CUE pin, same name | no, byte for byte (checked) |
| Dev CUE pin in a cue.mod | The release-pin check fails the Go tests check on a replace directive, an OPM Go pseudo-version or a dev CUE pin, same name | no, byte for byte (checked) |
| Dev core pin in the shipped default | The release-pin check fails the Go tests check on a dev core literal in Go source or a tracked cue.mod/local-module.cue, same name | no, byte for byte (checked) |
| Tracked CUE local replacement | The release-pin check fails the Go tests check on a dev core literal in Go source or a tracked cue.mod/local-module.cue, same name | no, byte for byte (checked) |
| Ordinary PR is not gated | The library's CI runs a release-pin check on every release PR, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "The failing check names each offender" in "The release-pin check names every offending file and line or module"
- "The check runs locally through the Taskfile" in "The release-pin check is runnable locally through the Taskfile"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The library's CI SHALL run a release-pin check on every release PR, identified by a branch name (`gi... | The library's CI runs a release-pin check on every release PR | verbatim |
| Source: workspace RELEASING.md, section "Gates" (G1). | The library's CI runs a release-pin check on every release PR | the whole Source line, as a `**Source**:` line |
| The check SHALL fail the PR's `Go tests` check when the PR head carries any of: - a `replace` direct... | The release-pin check fails the Go tests check on a replace directive, an OPM Go pseudo-version or a dev CUE pin | "The check" becomes "The release-pin check" (the name the first sentence gives it); the list is cut at a list boundary (items 1 to 3), the lead repeated, the closing ";" of item 3 becomes "." |
| Source: workspace RELEASING.md, section "Gates" (G1). | The release-pin check fails the Go tests check on a replace directive, an OPM Go pseudo-version or a dev CUE pin | the whole Source line, as a `**Source**:` line |
| The check SHALL fail the PR's `Go tests` check when the PR head carries any of: - a `replace` direct... | The release-pin check fails the Go tests check on a dev core literal in Go source or a tracked cue.mod/local-module.cue | "The check" becomes "The release-pin check"; the list is cut at a list boundary (items 4 and 5), the lead repeated; the wrapped line of item 4 is joined |
| Source: workspace RELEASING.md, section "Gates" (G1). | The release-pin check fails the Go tests check on a dev core literal in Go source or a tracked cue.mod/local-module.cue | the whole Source line, as a `**Source**:` line |
| On failure the check SHALL name every offending file and line or module. | The release-pin check names every offending file and line or module | "the check" becomes "the release-pin check" |
| Source: workspace RELEASING.md, section "Gates" (G1). | The release-pin check names every offending file and line or module | the whole Source line, as a `**Source**:` line |
| The check SHALL run as a step of the existing `Go tests` job, not as a job of its own, so it can nev... | The release-pin check runs as a step of the Go tests job | "The check" becomes "The release-pin check" |
| Source: workspace RELEASING.md, section "Gates" (G1). | The release-pin check runs as a step of the Go tests job | the whole Source line, as a `**Source**:` line |
| The same check SHALL be runnable locally through the Taskfile. | The release-pin check is runnable locally through the Taskfile | verbatim; pointer (Requirement: <exact header>) added after "The same check" |
| Source: workspace RELEASING.md, section "Gates" (G1). | The release-pin check is runnable locally through the Taskfile | the whole Source line, as a `**Source**:` line |

**A release notifies the library's downstream repos**: form c2 (REMOVED plus ADDED); old body 1317 characters; new bodies 248, 291, 488, 72, 136.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A release is cut | The release workflow has a notify-downstream job that runs cascade-notify with the release tag, same name | no, byte for byte (checked) |
| No release is cut | The notify-downstream job depends on the release-please job only and runs only for a release with CASCADE_NOTIFY not off, same name | no, byte for byte (checked) |
| Notify is switched off | The notify-downstream job depends on the release-please job only and runs only for a release with CASCADE_NOTIFY not off, same name | no, byte for byte (checked) |
| The docs bundle fails | The documentation-bundle job does not gate the notify-downstream job, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "The key is passed only to the cascade action" in "Only the caller-owned notify-downstream job reads the cascade App key"
- "Dispatch targets and the Go proxy wait belong to the action" in "Which repos notify-downstream dispatches to and the Go proxy wait are the action's behaviour"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The release workflow SHALL have a job `notify-downstream`, the last job of the workflow, that runs t... | The release workflow has a notify-downstream job that runs cascade-notify with the release tag | verbatim |
| Source: workspace `RELEASING.md`, sections "Notify after publish" and "repository_dispatch"; owner d... | The release workflow has a notify-downstream job that runs cascade-notify with the release tag | the whole Source line, as a `**Source**:` line |
| The job SHALL depend on the release-please job only, SHALL run only when that job's `releases_create... | The notify-downstream job depends on the release-please job only and runs only for a release with CASCADE_NOTIFY not off | "The job" becomes "The `notify-downstream` job" |
| Source: workspace `RELEASING.md`, sections "Notify after publish" and "repository_dispatch"; owner d... | The notify-downstream job depends on the release-please job only and runs only for a release with CASCADE_NOTIFY not off | the whole Source line, as a `**Source**:` line |
| The key SHALL be read only in the caller-owned `notify-downstream` job, which declares `environment:... | Only the caller-owned notify-downstream job reads the cascade App key | "The key SHALL" becomes "`secrets.CASCADE_APP_PRIVATE_KEY`: the key SHALL" (the key name is copied from the same sentence, as in `cascade-wiring`) |
| Source: workspace `RELEASING.md`, sections "Notify after publish" and "repository_dispatch"; owner d... | Only the caller-owned notify-downstream job reads the cascade App key | the whole Source line, as a `**Source**:` line |
| The documentation-bundle job SHALL NOT gate it. | The documentation-bundle job does not gate the notify-downstream job | "it" becomes "the `notify-downstream` job" |
| Source: workspace `RELEASING.md`, sections "Notify after publish" and "repository_dispatch"; owner d... | The documentation-bundle job does not gate the notify-downstream job | the whole Source line, as a `**Source**:` line |
| Which repos it dispatches to, and the Go proxy wait before it does, are the action's behaviour, not ... | Which repos notify-downstream dispatches to and the Go proxy wait are the action's behaviour | "it" (first) becomes `notify-downstream`; "are" becomes "SHALL be", as the block needs a keyword |
| Source: workspace `RELEASING.md`, sections "Notify after publish" and "repository_dispatch"; owner d... | Which repos notify-downstream dispatches to and the Go proxy wait are the action's behaviour | the whole Source line, as a `**Source**:` line |

**The release App key is read only in the release Environment**: form c2 (REMOVED plus ADDED); old body 753 characters; new bodies 353, 357.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A release run on main | The release workflow declares no workflow-level permissions and the release-please job mints a narrow App token, same name | no, byte for byte (checked) |
| A workflow on another branch reaches for the key | Every job that reads RELEASE_APP_PRIVATE_KEY declares the release Environment, same name | no, byte for byte (checked) |
| The secret is not yet in the Environment | Every job that reads RELEASE_APP_PRIVATE_KEY declares the release Environment, same name | no, byte for byte (checked) |
| The organization secret still reaches the library | Every job that reads RELEASE_APP_PRIVATE_KEY declares the release Environment, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| Every job that reads `RELEASE_APP_PRIVATE_KEY` SHALL declare `environment: release`, the Environment... | Every job that reads RELEASE_APP_PRIVATE_KEY declares the release Environment | verbatim |
| Today that is the release workflow's release-please job, which SHALL be granted no `GITHUB_TOKEN` pe... | Every job that reads RELEASE_APP_PRIVATE_KEY declares the release Environment | verbatim |
| Source: owner selection 29; security pass finding GOV-2. | Every job that reads RELEASE_APP_PRIVATE_KEY declares the release Environment | the whole Source line, as a `**Source**:` line |
| The release workflow SHALL declare `permissions: {}` at the workflow level, so each of its jobs carr... | The release workflow declares no workflow-level permissions and the release-please job mints a narrow App token | verbatim |
| The job SHALL mint the App token for this repository only (`owner`, `repositories`) with `contents: ... | The release workflow declares no workflow-level permissions and the release-please job mints a narrow App token | "The job" becomes "The release-please job" |
| Source: owner selection 29; security pass finding GOV-2. | The release workflow declares no workflow-level permissions and the release-please job mints a narrow App token | the whole Source line, as a `**Source**:` line |

### Ledger: duplicate-object-identities

**Duplicate rendered identities are detected with their producers**: form c2 (REMOVED plus ADDED); old body 1364 characters; new bodies 190, 404, 483, 250, 167, 201.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Two components render one cluster-scoped object | Each returned row carries the identity of the first object rendered with it and every producer of it, same name | no, byte for byte (checked) |
| Three objects, two identities | A function in opm/k8s/object returns every apply identity that two or more rendered objects share, same name | no, byte for byte (checked) |
| A clean render returns nothing | A render with no shared identity returns no rows, same name | no, byte for byte (checked) |
| One object under two versions of its group | Two objects share an apply identity when their API group, kind, namespace and name agree and the version part of apiVersion does not distinguish them, same name | no, byte for byte (checked) |
| An object with no apiVersion falls in the core group | Two objects share an apply identity when their API group, kind, namespace and name agree and the version part of apiVersion does not distinguish them, same name | no, byte for byte (checked) |
| One kind and name in two groups stays distinct | Two objects share an apply identity when their API group, kind, namespace and name agree and the version part of apiVersion does not distinguish them, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "Rows follow the order of the first render of each identity" in "Rows are returned in the order the first object of each identity was rendered"
- "The identity type keeps its four fields" in "The exported identity type keeps its fields"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The library SHALL provide, in the Kubernetes tier package `opm/k8s/object`, a function that scans a ... | A function in opm/k8s/object returns every apply identity that two or more rendered objects share | verbatim |
| Source: enhancement 0015 ("One registration per module" and "The rendered CR's name is instance-deri... | A function in opm/k8s/object returns every apply identity that two or more rendered objects share | the whole Source line, as a `**Source**:` line |
| Two objects SHALL share an apply identity when they carry the same API group (the part of `apiVersio... | Two objects share an apply identity when their API group, kind, namespace and name agree and the version part of apiVersion does not distinguish them | verbatim |
| Source: enhancement 0015 ("One registration per module" and "The rendered CR's name is instance-deri... | Two objects share an apply identity when their API group, kind, namespace and name agree and the version part of apiVersion does not distinguish them | the whole Source line, as a `**Source**:` line |
| Each returned row SHALL carry the identity of the first object rendered with it (its `apiVersion` ve... | Each returned row carries the identity of the first object rendered with it and every producer of it | verbatim; pointer (Requirement: <exact header>) added after "carry the identity" |
| Source: enhancement 0015 ("One registration per module" and "The rendered CR's name is instance-deri... | Each returned row carries the identity of the first object rendered with it and every producer of it | the whole Source line, as a `**Source**:` line |
| Rows SHALL be returned in the order the first object of each identity was rendered, so the result is... | Rows are returned in the order the first object of each identity was rendered | verbatim; pointer (Requirement: <exact header>) added after "Rows" |
| Source: enhancement 0015 ("One registration per module" and "The rendered CR's name is instance-deri... | Rows are returned in the order the first object of each identity was rendered | the whole Source line, as a `**Source**:` line |
| A render with no shared identity SHALL return no rows. | A render with no shared identity returns no rows | verbatim; pointer (Requirement: <exact header>) added after "return no rows" |
| Source: enhancement 0015 ("One registration per module" and "The rendered CR's name is instance-deri... | A render with no shared identity returns no rows | the whole Source line, as a `**Source**:` line |
| The exported identity type SHALL keep its fields (apiVersion, kind, namespace, name). | The exported identity type keeps its fields | verbatim; pointer (Requirement: <exact header>) added after "exported identity type" |
| Source: enhancement 0015 ("One registration per module" and "The rendered CR's name is instance-deri... | The exported identity type keeps its fields | the whole Source line, as a `**Source**:` line |

**The refusal is worded once**: form c2 (REMOVED plus ADDED); old body 596 characters; new bodies 231, 359, 202.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| The message names both carrying components | opm/k8s/object provides an error type that aggregates duplicate rows in one wording, same name | no, byte for byte (checked) |
| The message names a version mismatch | The aggregated error names each producer's apiVersion only when the producers of a row differ, same name | no, byte for byte (checked) |
| A same-version row carries no versions | The aggregated error names each producer's apiVersion only when the producers of a row differ, same name | no, byte for byte (checked) |
| A missing apiVersion is worded | The aggregated error names each producer's apiVersion only when the producers of a row differ, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "The kernel leaves the aggregated error to the runtime" in "The kernel does not return the aggregated duplicate-identity error"

| Old clause | New requirement | Change |
| --- | --- | --- |
| `opm/k8s/object` SHALL provide an error type aggregating duplicate rows, whose message names each id... | opm/k8s/object provides an error type that aggregates duplicate rows in one wording | verbatim |
| When the producers of a row do not all carry the same `apiVersion`, the message SHALL name each prod... | The aggregated error names each producer's apiVersion only when the producers of a row differ | verbatim; pointer (Requirement: <exact header>) added after "`apiVersion`, the message" |
| The kernel SHALL NOT return this error: it is raised by the runtime that calls the check, before app... | The kernel does not return the aggregated duplicate-identity error | verbatim; pointer (Requirement: <exact header>) added after "return this error" |

### Ledger: helper-packages

**Helper Boundary at opm/helper/**: form b (MODIFIED parent kept with a shorter body and all its scenarios, split-off rules ADDED); old body 1270 characters; new bodies 491, 120, 472, 232.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Helper boundary documented | Helper Boundary at opm/helper/, kept in place | no, byte for byte (checked) |
| Kernel imports nothing under the helper tree | Helper Boundary at opm/helper/, kept in place | no, byte for byte (checked) |
| A frontend that skips the helper tree can drive the whole pipeline | Helper Boundary at opm/helper/, kept in place | no, byte for byte (checked) |
| Lint gate refuses a helper import from the kernel | Helper Boundary at opm/helper/, kept in place | no, byte for byte (checked) |
| No publish-side package in the library | Helper Boundary at opm/helper/, kept in place | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "The helper subdirectory is opt-in" in "The library maintains an opm/helper subdirectory of opt-in frontend conveniences"
- "The packages outside the helper tree fall into two tiers" in "Two tiers lie outside opm/helper"
- "The exported packages outside the helper and Kubernetes trees are the six kernel packages" in "The exported packages outside opm/helper and opm/k8s are exactly the six kernel packages"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The helper boundary SHALL be real in the import graph: no kernel-tier package SHALL import a package... | Helper Boundary at opm/helper/ (kept) | verbatim; pointer (Requirement: <exact header>) added after "no kernel-tier package" |
| The rule SHALL be enforced by the repository lint gate, not only by documentation. | Helper Boundary at opm/helper/ (kept) | verbatim |
| The library SHALL maintain a `opm/helper/` subdirectory whose subpackages are opt-in, opinionated fr... | The library maintains an opm/helper subdirectory of opt-in frontend conveniences | verbatim |
| The packages outside `opm/helper/` SHALL fall into two tiers: the kernel (`opm/kernel`, `opm/module`... | Two tiers lie outside opm/helper | verbatim |
| Neither tier SHALL be described as opt-in. | Two tiers lie outside opm/helper | verbatim |
| The exported packages outside `opm/helper/` and `opm/k8s/` SHALL be exactly `opm/kernel`, `opm/modul... | The exported packages outside opm/helper and opm/k8s are exactly the six kernel packages | verbatim |

**Helper Layout for Future Subpackages**: form c2 (REMOVED plus ADDED); old body 746 characters; new bodies 195, 413, 222, 149.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Platform helper landing place | The one current helper subpackage is platformmodule, same name | no, byte for byte (checked) |
| Duplicate-identity helper landing place | The earlier opm/helper/objectset does not exist and the check lives in opm/k8s/object, same name | no, byte for byte (checked) |
| Values helper subpackage no longer exists | Collapsed subpackages do not appear as exemplars of the helper convention, same name | no, byte for byte (checked) |
| Loader and synth subpackages no longer exist | The one current helper subpackage is platformmodule, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "A future helper follows the convention" in "Future opt-in helpers follow the opm/helper/<name>/ convention and are added by their owning slices"

| Old clause | New requirement | Change |
| --- | --- | --- |
| Future opt-in helpers SHALL follow the `opm/helper/<name>/` convention. | Future opt-in helpers follow the opm/helper/<name>/ convention and are added by their owning slices | verbatim |
| Subpackages SHALL be added by their owning slices and not as part of the originating slice that esta... | Future opt-in helpers follow the opm/helper/<name>/ convention and are added by their owning slices | verbatim |
| Past examples of helper subpackages in this convention SHALL reflect the current package layout; sub... | Collapsed subpackages do not appear as exemplars of the helper convention | verbatim; pointer (Requirement: <exact header>) added after "in this convention" |
| The one current subpackage is `platformmodule` (platform module generation from catalog coordinates)... | The one current helper subpackage is platformmodule | "is" becomes "SHALL be", as the block needs a keyword; pointer (Requirement: <exact header>) added after "one current subpackage" |
| The earlier `opm/helper/objectset` (duplicate rendered object identities) SHALL NOT exist: the check... | The earlier opm/helper/objectset does not exist and the check lives in opm/k8s/object | verbatim |

### Ledger: test-fixture-registry

**Served fixtures resolve from a per-test module cache**: form c2 (REMOVED plus ADDED); old body 615 characters; new bodies 292, 337, 208.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Committed fixture edited under a fixed coordinate | A served coordinate is fetched into the private cache on first use and no earlier bytes are visible, same name | no, byte for byte (checked) |
| Two packages serve the same coordinate concurrently | Every fixture registry the harness stands up gives the test a private CUE module cache directory, same name | no, byte for byte (checked) |
| Same test name in two packages | Every fixture registry the harness stands up gives the test a private CUE module cache directory, same name | no, byte for byte (checked) |
| Private cache is cleaned up | The private cache is removed when the test ends, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| Every fixture registry the harness stands up (inline catalog fixtures, inline module fixtures, commi... | Every fixture registry the harness stands up gives the test a private CUE module cache directory | verbatim |
| A coordinate the registry serves SHALL be fetched and extracted into that private cache on first use... | A served coordinate is fetched into the private cache on first use and no earlier bytes are visible | verbatim; pointer (Requirement: <exact header>) added after "that private cache" |
| The private cache SHALL be removed when the test ends, read-only extracted directories included. | The private cache is removed when the test ends | verbatim; pointer (Requirement: <exact header>) added after "The private cache" |

**The opmodel.dev namespace resolves through the shared tier**: form b (MODIFIED parent kept with a shorter body and all its scenarios, split-off rules ADDED); old body 521 characters; new bodies 460, 134.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Warm core is reused | The opmodel.dev namespace resolves through the shared tier, kept in place | no, byte for byte (checked) |
| Cold core is fetched once | The opmodel.dev namespace resolves through the shared tier, kept in place | no, byte for byte (checked) |
| Concurrent cold fetch | The opmodel.dev namespace resolves through the shared tier, kept in place | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text and adds no case):

- "The harness writes nothing into the shared tier" in "The harness itself writes nothing into the shared tier"

| Old clause | New requirement | Change |
| --- | --- | --- |
| Within a private cache, the `opmodel.dev` module namespace (`opmodel.dev/core`, `opmodel.dev/catalog... | The opmodel.dev namespace resolves through the shared tier (kept) | verbatim |
| The harness itself SHALL write nothing into the shared tier. | The harness itself writes nothing into the shared tier | verbatim; pointer (Requirement: <exact header>) added after "the shared tier" |

## Strength table

Sentences of unit 2 that gained SHALL: the old sentence has no keyword and an ADDED block needs one. The units before this one list theirs in the clause tables ("is" becomes "SHALL be"); the last unit collects every row here.

| Old sentence | New sentence | Block |
| --- | --- | --- |
| Provenance is the registry key, never a value parsed out of an FQN or read off the transformer, so two majors of one catalog are two providers and two transformers of one entry are one. | Provenance SHALL be the registry key, never a value parsed out of an FQN or read off the transformer, so two majors of one catalog are two providers and two transformers of one entry are one. | Provenance is the registry key, so two majors of one catalog are two providers |
| The count is `#Platform.#contracts.providedBy`: for every contract key declared ... | The provider count SHALL be `#Platform.#contracts.providedBy`: for every contract key declared ... | The provider count lists the registry keys that require each provider-fulfilled contract |
| Apart from the floor, `Render` reads only the platform's `Metadata` and `Source`. | Apart from the floor, `Render` SHALL read only the platform's `Metadata` and `Source`. | One acquired platform stays shareable across concurrent renders |
| A platform the constructor did not build decodes `Package` once, on its first `CoreFloor()` or `Contracts()` call. | A platform the constructor did not build SHALL decode `Package` once, on its first `CoreFloor()` or `Contracts()` call. | The check decodes Package once on a platform the constructor did not build |
| It reports them on top of ... | `2.0.0-alpha.13` SHALL report contract collisions on top of ... | 2.0.0-alpha.13 reports the provider count, the comparable report and the contract inventory; 2.0.0-alpha.13 reports the registry shape and the context projection |
| ... `Kernel.Render` and `Platform.Contracts()` accept every core from `schema.ProvidedBySince` (`2.0.0-alpha.12`) on ... | ... `Kernel.Render` and `Platform.Contracts()` SHALL accept every core from `schema.ProvidedBySince` (`2.0.0-alpha.12`) on ... | The default is not the render floor |
| The constant advances only through a deliberate change ... | The constant SHALL advance only through a deliberate change ... | The default advances only through a deliberate change that re-verifies the glue and fixtures |
| ... they stay literal and are declared in `.cascade-frozen` ... | ... they SHALL stay literal and are declared in `.cascade-frozen` ... | Synthetic, floor and skew literals stay literal and are declared frozen |
| `Kernel.Render` ... skip every **unprovided** demand: a resource or trait demand whose contract declares ... (the key is absent from ...) | Every **unprovided** demand SHALL be a resource or trait demand whose contract declares ... (the key is absent from ...). | An unprovided demand is one no enabled registry entry provides |

Sentences of unit 3 that gained SHALL (the old sentence or clause has no keyword), or changed a verb form to take it:

| Old sentence | New sentence | Block |
| --- | --- | --- |
| The values an operation returns (an artifact's `Package`, a validated value) keep that operation's runtime alive for exactly as long as the caller holds them. | The values an operation returns (an artifact's `Package`, a validated value) SHALL keep that operation's runtime alive for exactly as long as the caller holds them. | Returned values keep their operation's runtime alive while the caller holds them |
| A value the operation returns (an artifact's `Package`, a `*kernel.Compiled`) keeps that context alive for as long as the caller holds it, and no longer: the context's lifetime is bounded by its holder (ADR-007). | A value the operation returns (an artifact's `Package`, a `*kernel.Compiled`) SHALL keep that context alive for as long as the caller holds it, and no longer: the context's lifetime is bounded by its holder (ADR-007). | A returned value keeps its operation's context alive as long as the caller holds it |
| Concurrency is across operations, never within one; a consumer rendering from several goroutines calls `Render` on one Kernel, with no shared materialized platform value and no mutex. | Concurrency is across operations, never within one; a consumer rendering from several goroutines SHALL call `Render` on one Kernel, with no shared materialized platform value and no mutex. | Concurrency is across operations, never within one |
| The method SHALL NOT depend on `loaderfile.LoadValuesFile` (which is removed). | The `*Kernel.LoadSourceFromFile(path string)` method SHALL NOT depend on `loaderfile.LoadValuesFile` (which is removed). | LoadSourceFromFile does not depend on loaderfile.LoadValuesFile |
| Absent the option, the kernel SHALL inherit `CUE_REGISTRY` from the process environment and SHALL NOT auto-apply a built-in default registry. | Absent the `WithRegistry(string)` option, the kernel SHALL inherit `CUE_REGISTRY` from the process environment and SHALL NOT auto-apply a built-in default registry. | Absent WithRegistry the kernel inherits CUE_REGISTRY and applies no default |
| The option MUST NOT mutate process environment state; the mapping is plumbed into each operation's load configuration. | The `WithRegistry(string)` option MUST NOT mutate process environment state; the mapping is plumbed into each operation's load configuration. | WithRegistry mutates no process environment |
| It SHALL build the instance spec by single-build CUE evaluation inside the module's staged source with the unified values rendered into the package, check the values sources against the module's `#config` at their own positions after the build, assert concreteness on the built spec, decode instance metadata, and return the constructed `*module.Instance` with its `Source` populated. | `Kernel.SynthesizeInstance` SHALL build the instance spec by single-build CUE evaluation inside the module's staged source with the unified values rendered into the package, check the values sources against the module's `#config` at their own positions after the build, assert concreteness on the built spec, decode instance metadata, and return the constructed `*module.Instance` with its `Source` populated. | SynthesizeInstance builds the instance in one build and decodes its metadata |
| The synthesized package imports `core` at the major of the kernel's schema release (read from the loader's pin with no schema load when it names an exact release, resolved through the schema cache otherwise); the release that import resolves to comes from the module's own `cue.mod/module.cue`. | The synthesized package SHALL import `core` at the major of the kernel's schema release (read from the loader's pin with no schema load when it names an exact release, resolved through the schema cache otherwise); the release that import resolves to comes from the module's own `cue.mod/module.cue`. | The synthesized package imports core at the major of the kernel's schema release |
| The build SHALL run in a context the method creates and releases; the method SHALL NOT consult any additional values source. | The `SynthesizeInstance` build SHALL run in a context the method creates and releases; `SynthesizeInstance` SHALL NOT consult any additional values source. | The synthesis build runs in a context the method creates and releases |
| When the build fails and the call's merged values exist, the method SHALL attribute the failure to the values sources the way `AcquireInstanceFromDir` does: | When the build fails and the call's merged values exist, `SynthesizeInstance` SHALL attribute the failure to the values sources the way `AcquireInstanceFromDir` does: | A failed synthesis build is built again without the rendered values file |
| When the build fails and the call's merged values exist, the method SHALL attribute the failure to the values sources the way `AcquireInstanceFromDir` does: | When the build fails and the call's merged values exist, `SynthesizeInstance` SHALL attribute the failure to the values sources the way `AcquireInstanceFromDir` does: | A failed synthesis build validates the compiled values against the values-free #config |
| In every other case (no values, the values-free build also fails, or the values are clean) the method SHALL return the original build error unchanged. | In every other case (no values, the values-free build also fails, or the values are clean) `SynthesizeInstance` SHALL return the original build error unchanged. | Every other build failure is returned unchanged |
| Values are applied inside a CUE build: `AcquireInstanceFromDir` renders the unified sources into the package overlay, and `SynthesizeInstance` renders them into the synthesized package; both then check the sources against `#config` so a violation is reported at the sources' own positions, and both assert concreteness on the whole built spec and on the values unified with `#config` (see "Instance verbs refuse an unset required config value"). | Values are applied inside a CUE build: `AcquireInstanceFromDir` renders the unified sources into the package overlay, and `SynthesizeInstance` renders them into the synthesized package; both SHALL then check the sources against `#config` so a violation is reported at the sources' own positions, and both assert concreteness on the whole built spec and on the values unified with `#config` (see "Instance verbs refuse an unset required config value"). | Both instance verbs check the sources against #config at their own positions |
| The documentation SHALL state that the synthesized package imports `core` at the major of the kernel's schema release (read from the configured loader without a schema load when that loader pins an exact release, and resolved through the kernel's schema cache otherwise), and that the release that import resolves to comes from the module's own `cue.mod/module.cue`. | The package documentation and the `Kernel.SynthesizeInstance` godoc SHALL state that the synthesized package imports `core` at the major of the kernel's schema release (read from the configured loader without a schema load when that loader pins an exact release, and resolved through the kernel's schema cache otherwise), and that the release that import resolves to comes from the module's own `cue.mod/module.cue`. | The documentation states which core release the synthesized package imports |
| The documentation SHALL NOT present a helper-level composition that reaches synthesis or instance processing directly, since neither is exported. | The package documentation and the `Kernel.SynthesizeInstance` godoc SHALL NOT present a helper-level composition that reaches synthesis or instance processing directly, since neither is exported. | The documentation presents no helper-level composition |
| Where it shows a pipeline, the pipeline SHALL show acquisition or synthesis producing the artifacts and one render build over an instance and a platform, ending in `[]*kernel.Compiled`, and SHALL NOT send a `Catalog` into render. | Where `CONSTITUTION.md` shows a pipeline, the pipeline SHALL show acquisition or synthesis producing the artifacts and one render build over an instance and a platform, ending in `[]*kernel.Compiled`, and SHALL NOT send a `Catalog` into render. | A front-door pipeline ends in kernel.Compiled |
| it states the rule and cites a source a reader can open (an ADR; a decision ADR-013 records, cited as `ADR-013, decision <id>`; an enhancement decision such as enhancement 0012 ("The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module"); a pull request; or an archived change) | A committed file outside `openspec/changes/` that cites a source SHALL state the rule and cite a source a reader can open (an ADR; a decision ADR-013 records, cited as `ADR-013, decision <id>`; an enhancement decision such as enhancement 0012 ("The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module"); a pull request; or an archived change) | A committed file cites a source a reader can open |
| The client SHALL be built from the kernel's registry mapping (`WithRegistry`, else the process `CUE_REGISTRY`) on its first use, never in `kernel.New`, so construction still evaluates nothing and fails on nothing. | The registry client SHALL be built from the kernel's registry mapping (`WithRegistry`, else the process `CUE_REGISTRY`) on its first use, never in `kernel.New`, so construction still evaluates nothing and fails on nothing. | The registry client is built on first use and its settings are read then |
| A failure to build the client SHALL be returned to the operation that needed it and SHALL NOT be kept: the next operation that needs a client SHALL try to build it again. | A failure to build the registry client SHALL be returned to the operation that needed it and SHALL NOT be kept: the next operation that needs a client SHALL try to build it again. | A failure to build the client is returned and not kept |
| Each operation SHALL wrap the client in a module cache of its own, over the cache directory read when that operation starts (`CUE_CACHE_DIR`, else the user cache directory), so that no fetch failure (a refused connection, an error status, a version not yet published, a cancelled or expired context) is served to a later operation, and a later change to `CUE_CACHE_DIR` reaches the next operation as it does today. | Each operation SHALL wrap the registry client in a module cache of its own, over the cache directory read when that operation starts (`CUE_CACHE_DIR`, else the user cache directory), so that no fetch failure (a refused connection, an error status, a version not yet published, a cancelled or expired context) is served to a later operation, and a later change to `CUE_CACHE_DIR` reaches the next operation as it does today. | Each operation wraps the client in a module cache of its own |
| A registry call that fails SHALL also drop the shared client, so that the next operation builds it again. | A registry call that fails SHALL also drop the shared registry client, so that the next operation builds it again. | A failed registry call drops the shared client |
| The client SHALL NOT replace the schema cache's own loader: an `OCILoader` is configured on its own (it may override the cache directory) and the schema cache loads once per Kernel, and a caller-supplied `schema.Loader` is the caller's. | The registry client SHALL NOT replace the schema cache's own loader: an `OCILoader` is configured on its own (it may override the cache directory) and the schema cache loads once per Kernel, and a caller-supplied `schema.Loader` is the caller's. | The registry client does not replace the schema cache's loader |
| No exported signature SHALL change for this: the client is not an option, a parameter or an exported field. | No exported signature SHALL change for this: the registry client is not an option, a parameter or an exported field. | The helper platformmodule registry stays caller-built and no exported signature changes |
| The effective values are the built spec's `values`: the package's own `values` plus the trailing sources on `AcquireInstanceFromDir`, the merged `InstanceInput.Values` on `SynthesizeInstance`. | The effective values SHALL be the built spec's `values`: the package's own `values` plus the trailing sources on `AcquireInstanceFromDir`, the merged `InstanceInput.Values` on `SynthesizeInstance`. | The effective values are the built spec's values |
| A required value is one that `cue.Concrete(true)` refuses on those values unified with `#config`: a required field (`foo!:`) that is absent, or a field that resolves to no concrete value (a bare type such as `string` or `_`, with no default). | A required value SHALL be one that `cue.Concrete(true)` refuses on those values unified with `#config`: a required field (`foo!:`) that is absent, or a field that resolves to no concrete value (a bare type such as `string` or `_`, with no default). | A required value is one that concreteness refuses |
| A caller identifies the refusal as it did before this requirement changed: `errors.As` with CUE's error type finds the tree, and a finding whose path starts with `values` is a values finding; the kernel adds no error type for it. | A caller SHALL identify the refusal as it did before this requirement changed: `errors.As` with CUE's error type finds the tree, and a finding whose path starts with `values` is a values finding; the kernel adds no error type for it. | A caller identifies the refusal by CUE's error type |
| The `Metadata` pointer holds a decoded ergonomic projection; the `Package` field carries the source-of-truth CUE value; `Source` is the staged source tree the artifact was acquired or synthesized from, nil for an artifact built from a bare value. | The `Metadata` pointer SHALL hold a decoded ergonomic projection; the `Package` field carries the source-of-truth CUE value; `Source` is the staged source tree the artifact was acquired or synthesized from, nil for an artifact built from a bare value. | The Metadata, Package and Source fields hold a projection, the source of truth and the staged source |
| `Catalog` is admitted on the terms ADR-009 records and on no others: the kernel acquires it, reads it and derives from it, and every verdict about what it reads stays with the caller. | `Catalog` SHALL be admitted on the terms ADR-009 records and on no others: the kernel acquires it, reads it and derives from it, and every verdict about what it reads stays with the caller. | Catalog is admitted on the terms of ADR-009 and a catalog is never rendered |
| A directory with no enclosing module is its own root with an empty `Pkg`. | A directory with no enclosing module SHALL be its own root with an empty `Pkg`. | A directory with no enclosing module is its own root |
| With no values, `Source.Overlay` is nil (on-disk mode). | With no values, `Source.Overlay` SHALL be nil (on-disk mode). | An instance acquired without values has an on-disk source |
| The render path reads no artifact sub-value in Go: the instance and the platform enter the render build by import, and the generated glue reads `components` and `#composedTransformers` in CUE. | The render path SHALL read no artifact sub-value in Go: the instance and the platform enter the render build by import, and the generated glue reads `components` and `#composedTransformers` in CUE. | The render path reads no artifact sub-value in Go |
| Required identity fields are those the schema never defaults: | Required identity fields SHALL be those the schema never defaults: | The required identity fields are those the schema never defaults |
| Core derives an entry's `version` from its embedded catalog's stamped `metadata.version` (enhancement 0019, "A platform imports its catalog and embeds it whole into the registry entry, which derives everything else"), so an entry with no embedded catalog, or one embedding an unstamped catalog, fails here as a missing required field naming the entry. | Core derives an entry's `version` from its embedded catalog's stamped `metadata.version` (enhancement 0019, "A platform imports its catalog and embeds it whole into the registry entry, which derives everything else"), so an entry with no embedded catalog, or one embedding an unstamped catalog, SHALL fail here, at the gate, as a missing required field naming the entry. | A registry entry with no embedded catalog fails as a missing required field |
| The method SHALL validate before it writes: a nil receiver, an on-disk source (`Overlay` nil, since `Root` already is the directory) and any entry whose path is not under `Root` SHALL be refused with a plain error and nothing written. | `WriteTo` SHALL validate before it writes: a nil receiver, an on-disk source (`Overlay` nil, since `Root` already is the directory) and any entry whose path is not under `Root` SHALL be refused with a plain error and nothing written. | WriteTo validates before it writes |
| a path that is not a directory SHALL fail with `<label> path "<absolute path>" is not a directory`, where `<label>` is `module`, `catalog`, `platform` or `instance` | A path that is not a directory SHALL fail with `<label> path "<absolute path>" is not a directory`, where `<label>` is `module`, `catalog`, `platform` or `instance` | A directory acquire verb refuses a path that is not a directory |
| Platform acquisition and instance acquisition without values sources keep the on-disk mode their own requirements state. | Platform acquisition and instance acquisition without values sources SHALL keep the on-disk mode their own requirements state. | Platform acquisition and instance acquisition without values keep the on-disk mode |
| The constructed artifact carries no `Source`. | The constructed artifact SHALL carry no `Source`. | A constructed artifact carries no Source |
| The kernel SHALL NOT wrap these constructors: a frontend holding a value calls the package constructor directly; a frontend that wants a source-carrying artifact uses an acquire verb. | The kernel SHALL NOT wrap the constructors `module.NewModuleFromValue(v)` and `platform.NewPlatformFromValue(v)`: a frontend holding a value calls the package constructor directly; a frontend that wants a source-carrying artifact uses an acquire verb. | The kernel does not wrap the constructors |
| The render build reads none of these accessors: the transformer context is projected by core (enhancement 0019, "`#TransformerContext` becomes a projection of the other two inputs; the kernel fills `#runtimeName` alone"), and the glue reads the instance by import. | The render build SHALL read none of `Components()`, `ConfigSchema()`, `Values()` and `ModuleMetadata()`: the transformer context is projected by core (enhancement 0019, "`#TransformerContext` becomes a projection of the other two inputs; the kernel fills `#runtimeName` alone"), and the glue reads the instance by import. | The render build reads none of the instance accessors |

Sentences of unit 4 (library-B4 first half) that gained SHALL, or changed a verb form to take it:

| Old sentence | New sentence | Block |
| --- | --- | --- |
| Because the instance is built inside the module's main module, the module import resolves **locally** (no fabricated dependency, no registry round-trip for the module itself). | Because the instance is built inside the module's main module, the module import SHALL resolve **locally** (no fabricated dependency, no registry round-trip for the module itself). | The instance source imports the module's package and core, and the module import resolves locally |
| `synth.Instance` lives in `opm/internal/synth`, off the public surface; its Go signature is `Instance(cueCtx *cue.Context, coreVersion string, in Input) (cue.Value, *module.Source, error)`. | `synth.Instance` SHALL live in `opm/internal/synth`, off the public surface; its Go signature is `Instance(cueCtx *cue.Context, coreVersion string, in Input) (cue.Value, *module.Source, error)`. | synth.Instance lives in opm/internal/synth with a fixed Go signature |
| `cueCtx` is the context `Kernel.SynthesizeInstance` created for the call, `coreVersion` is the kernel's core release, from which the synthesized `core` import takes its major (the release the import resolves to comes from the module's own `cue.mod/module.cue`), and `Input` is the package-internal input carrying `Module`, `Name`, `Namespace`, the unified `Values`, `Labels`, `Annotations` and the... | `synth.Instance`: `cueCtx` SHALL the context `Kernel.SynthesizeInstance` created for the call, `coreVersion` is the kernel's core release, from which the synthesized `core` import takes its major (the release the import resolves to comes from the module's own `cue.mod/module.cue`). | The cueCtx and coreVersion parameters of synth.Instance are the call context and the core release |
| `cueCtx` is the context `Kernel.SynthesizeInstance` created for the call, `coreVersion` is the kernel's core release, from which the synthesized `core` import takes its major (the release the import resolves to comes from the module's own `cue.mod/module.cue`), and `Input` is the package-internal input carrying `Module`, `Name`, `Namespace`, the unified `Values`, `Labels`, `Annotations` and the... | `synth.Instance`: `Input` SHALL be the package-internal input carrying `Module`, `Name`, `Namespace`, the unified `Values`, `Labels`, `Annotations` and the... | The Input parameter of synth.Instance is the package-internal input |
| A module acquired from a directory has a real root on disk and is not checked. | A module acquired from a directory has a real root on disk and SHALL NOT be checked. | A module acquired from a directory is not checked for a synthetic root |

Sentences of unit 5 (library-B4 second half) that gained SHALL, or changed a verb form to take it:

| Old sentence | New sentence | Block |
| --- | --- | --- |
| In a dry run the receiver computes the diff, title, body and labels and writes them to the job summary, and pushes nothing, opens or edits no PR, and adds no label or comment. | In a dry run the receiver SHALL compute the diff, title, body and labels and write them to the job summary, and push nothing, open or edit no PR, and add no label or comment. | A dry run writes its plan to the job summary and changes nothing |
| A source that fails to compile or to load is returned as before, without the marker: such a failure can be a registry or a file failure, which is not a statement about the values. | A source that fails to compile or to load SHALL be returned as before, without the marker: such a failure can be a registry or a file failure, which is not a statement about the values. | A source that fails to compile or to load is returned without the ConfigValidationError marker |

Sentences of unit 6 (library-B5 first half) that gained SHALL, or changed a verb form to take it. The pointer, subject and number additions of the decision above are not listed here, because they add no keyword:

| Old sentence | New sentence | Block |
| --- | --- | --- |
| `FetchUnreachable` means no HTTP response was received: a refused connection, a DNS or TLS failure, a timeout or an expired deadline. | `FetchUnreachable` SHALL mean no HTTP response was received: a refused connection, a DNS or TLS failure, a timeout or an expired deadline. | FetchUnreachable and FetchOther classify the other failed registry interactions and Status holds the answered code |
| An author-defect resolution failure, which no failed registry interaction caused, is a `*ResolutionError` and never a `*FetchError`. | An author-defect resolution failure, which no failed registry interaction caused, SHALL be a `*ResolutionError` and never a `*FetchError`. | An author-defect resolution failure is a ResolutionError and never a FetchError |
| A caller tells a refused credential from a refused permission by `Status` (401 or 403). | (Requirement: A token endpoint's answer classifies by its status and never as FetchUnreachable) A caller SHALL tell a refused credential from a refused permission by `Status` (401 or 403). | A caller tells a refused credential from a refused permission by Status |
| One case is outside this requirement. | One case SHALL be outside this requirement (Requirement: A token endpoint's answer classifies by its status and never as FetchUnreachable). | A 403 or a 404 on a module version lookup drops the status, also for the token endpoint's answer |
| `ResolutionImportUnprovided` means no module of the build provides an imported package. | `ResolutionImportUnprovided` SHALL mean no module of the build provides an imported package. | ResolutionImportUnprovided means no module of the build provides an imported package |
| `ResolutionImportAmbiguous` means more than one module of the build provides an imported package. | `ResolutionImportAmbiguous` SHALL mean more than one module of the build provides an imported package. | ResolutionImportAmbiguous means more than one module of the build provides an imported package |
| `ResolutionModuleFileInvalid` means a dependency's module file does not parse, whether the dependency is published or served from a local replacement directory. | `ResolutionModuleFileInvalid` SHALL mean a dependency's module file does not parse, whether the dependency is published or served from a local replacement directory. | ResolutionModuleFileInvalid means a dependency's module file does not parse |
| A `net.Error` is `FetchUnreachable`, except when the text of the chain is the answered token request of step 1 below, which classifies by its status: the embedded registry client leaves no typed status for a refused token refresh. | A `net.Error` SHALL be `FetchUnreachable`, except when the text of the chain is the answered token request of step 1 below (Requirement: The answered token request is the first fetch form), which classifies by its status: the embedded registry client leaves no typed status for a refused token refresh. | A net.Error is FetchUnreachable except for the answered token request |
| First is the answered token request: `cannot do HTTP request: ` followed on the same line by a request and `: <code> <status text>` at the end of the line, where the status text is the one HTTP gives the code and the code is 400 or higher. | First SHALL be the answered token request: `cannot do HTTP request: ` followed on the same line by a request and `: <code> <status text>` at the end of the line, where the status text is the one HTTP gives the code and the code is 400 or higher. | The answered token request is the first fetch form |
| The embedded registry client writes that form only when the token endpoint of a token-authenticated registry answered the token request with that status, so it classifies by the status, as a registry answer does, and never as `FetchUnreachable`. | The embedded registry client writes that form (Requirement: The answered token request is the first fetch form) only when the token endpoint of a token-authenticated registry answered the token request with that status, so it SHALL classify by the status, as a registry answer does, and never as `FetchUnreachable`. | The answered token request classifies by its status and never as FetchUnreachable |
| Then come `cannot do HTTP request` in any other form, which is `FetchUnreachable`, and an HTTP status in the form `<code> <status text>: `, which the embedded CUE keeps for 401, 429 and 5xx answers, so a flattened 5xx stays transient. | Then come `cannot do HTTP request` in any other form (Requirement: The answered token request is the first fetch form), which SHALL be `FetchUnreachable`, and an HTTP status in the form `<code> <status text>: `, which the embedded CUE keeps for 401, 429 and 5xx answers, so a flattened 5xx stays transient. | Any other cannot do HTTP request form is FetchUnreachable and a flattened HTTP status keeps its class |
| They also include `module not found`, and `cannot find module providing package` followed by a package path at an exact version (`P@vX.Y.Z`). | The fetch forms SHALL also include `module not found`, and `cannot find module providing package` followed by a package path at an exact version (`P@vX.Y.Z`). | The module not found and exact-version package forms are fetch forms |
| Last among them is `cannot fetch`, which is `FetchOther`. | Last among the fetch forms is `cannot fetch`, which SHALL be `FetchOther`. | The generic cannot fetch form is FetchOther |
| The author-defect forms, in this order: | The author-defect forms SHALL be, in this order: | The author-defect forms are matched in a fixed order |
| a platform the constructor did not build decodes `Package` once, on its first `CoreFloor()` or `Contracts()` call ("A platform records its core floor and contract inventory at construction") | A platform the constructor did not build SHALL decode `Package` once, on its first `CoreFloor()` or `Contracts()` call ("A platform records its core floor and contract inventory at construction") (Requirement: A Platform the constructor did not build decodes the same facts from its Package once) | A platform the constructor did not build decodes Package once, on its first CoreFloor or Contracts call |
| A disabled entry never counts as a definer. | A disabled entry SHALL never count as a definer. | A disabled entry never counts as a definer |
| Providers are counted per registry entry, the same count the render's single-provider guard reads: two majors of one catalog are two providers, and two transformers of one entry are one. | Providers SHALL be counted per registry entry, the same count the render's single-provider guard reads: two majors of one catalog are two providers, and two transformers of one entry are one. | Providers are counted per registry entry |
| A directory with no enclosing module is its own root with an empty `Pkg`. | A directory with no enclosing module SHALL be its own root with an empty `Pkg`. | AcquirePlatformFromDir stamps Pkg as the package directory relative to Root |
| No other kernel operation reads the field. | (Requirement: Platform carries its render source) No other kernel operation SHALL read the `Source` field. | No kernel operation other than Render reads the Source field of a platform |

Sentences of unit 7 (library-B5 second half) that gained SHALL, or changed a verb form to take it. The pointer and subject additions of the decision above add no keyword and are not listed here:

| Old sentence | New sentence | Block |
| --- | --- | --- |
| For modules that entry point is `Kernel.AcquireModuleFromRegistry(ctx, path, version string) (*module.Module, error)`, which fetches a published `#Module` via CUE's native module machinery, loads it in memory as the main... | For modules that entry point (Requirement: Each artifact kind has one registry acquisition entry point) SHALL be `Kernel.AcquireModuleFromRegistry(ctx, path, version string) (*module.Module, error)`, which fetches a published `#Module` via CUE's native module... | AcquireModuleFromRegistry fetches a published module via CUE's native module machinery |
| For modules that entry point is `Kernel.AcquireModuleFromRegistry(ctx, path, version string) (*module.Module, error)`, which fetches a published `#Module` via CUE's native module machinery, loads it in memory as the main... | For modules that entry point (Requirement: Each artifact kind has one registry acquisition entry point) SHALL be `Kernel.AcquireModuleFromRegistry(ctx, path, version string) (*module.Module, error)`, which loads it in memory as the main module (its own `cue.mo... | AcquireModuleFromRegistry loads the module in memory as the main module |
| For modules that entry point is `Kernel.AcquireModuleFromRegistry(ctx, path, version string) (*module.Module, error)`, which fetches a published `#Module` via CUE's native module machinery, loads it in memory as the main... | For modules that entry point (Requirement: Each artifact kind has one registry acquisition entry point) SHALL be `Kernel.AcquireModuleFromRegistry(ctx, path, version string) (*module.Module, error)`, which runs the acquisition shape gate, verifies the declared... | AcquireModuleFromRegistry runs the shape gate and verifies the declared identity |
| For modules that entry point is `Kernel.AcquireModuleFromRegistry(ctx, path, version string) (*module.Module, error)`, which fetches a published `#Module` via CUE's native module machinery, loads it in memory as the main... | For modules that entry point (Requirement: Each artifact kind has one registry acquisition entry point) SHALL be `Kernel.AcquireModuleFromRegistry(ctx, path, version string) (*module.Module, error)`, which returns a `*module.Module` whose staged source is atta... | AcquireModuleFromRegistry returns the module with its staged source attached |
| A dependency a local replacement serves may carry no version and maps to the empty string. | A dependency (Requirement: The catalog artifact exposes the requirements recorded in its own cue.mod/module.cue and the library exposes the requirements only) a local replacement serves may carry no version and SHALL map to the empty string. | A dependency a local replacement serves may carry no version and maps to the empty string |
| The harness is the standing tripwire against any Go-side transformation of a component, instance or context value being reintroduced. | The harness SHALL be the standing tripwire against any Go-side transformation of a component, instance or context value being reintroduced. | The harness is the standing tripwire against a Go-side transformation being reintroduced |
| The proof lives in the archive as the record; after deletion the harness compares `Render` against the oracle only. | The proof (Requirement: The old render path was compared with Render on every parity case before it was removed) SHALL live in the archive as the record; after deletion the harness SHALL compare `Render` against the oracle only. | The proof of the comparison lives in the archive and after deletion the harness compares Render against the oracle only |
| Core in the test modules moves only together with the default, never on its own. | Core in the test modules SHALL move only together with the default, never on its own. | The update task requests core at the release the kernel default names and core moves only together with the default |
| A file qualifies when its literal names an older release a floor, skew or pre-collision test needs, or when the literal is synthetic: a default move does not break it, because it is parsed or compiled in memory, or coupl... | A file (Requirement: The repository root carries a .cascade-frozen file listing every test file that keeps a version literal on purpose) SHALL qualify when its literal names an older release a floor, skew or pre-collision test needs, or when the literal is syn... | A file qualifies for the frozen list when its literal names an older release a test needs or is synthetic |
| The one current subpackage is `platformmodule` (platform module generation from catalog coordinates). | The one current subpackage SHALL be `platformmodule` (platform module generation from catalog coordinates). | The one current helper subpackage is platformmodule |
| Which repos it dispatches to, and the Go proxy wait before it does, are the action's behaviour, not the library's. | Which repos `notify-downstream` dispatches to, and the Go proxy wait before it does, SHALL be the action's behaviour, not the library's. | Which repos notify-downstream dispatches to and the Go proxy wait are the action's behaviour |
