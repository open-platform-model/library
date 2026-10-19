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

**Decision**: This change does not add the Taskfile tasks or the CI job that run `openspec validate --all --strict`. The gate moves to a separate change after `publish-go-api-bundle` lands (owner decision of 2026-10-19), because until then `validate --all --strict` covers that open change and exits 1 on its one warning. The per-unit gate block above stays: it runs in a scratch copy and is not the CI gate. Reversibility: high.

### Batch library-B1: the form choices that later batches copy

**Context**: B1 holds `kubernetes-tier` (25 long requirements, 85 KB) and `api-diff-check` (3). Most long bodies hold several rules and a long `Source:` line.
**Decision**: Form a (MODIFIED) was not enough, because the excess was rules, not examples. 8 requirements use b (MODIFIED parent kept, split-off rules ADDED under new names) and 20 use c2 (REMOVED plus ADDED), where the old scenarios belong to different rules. A c2 split names each piece after the one rule it states, and every old scenario moves to the piece it tests, with its WHEN and THEN lines unchanged. Rule tables (the weights, the escape rules of the render digest, the field list of the label package, and the ordered conditions of the delete verdict, the apply verdict and the hold verdict) move into the THEN and AND lines of one new scenario; the body keeps the SHALL and the first-match rule, names the scenario and, for the delete and apply verdicts, lists the outcomes (owner ruling of 2026-10-19). A new requirement gets a restatement scenario, built from words of the old text, where the old scenarios cannot move to it (owner ruling of 2026-10-19). Where an old sentence had no SHALL and its new body needs one, the verb "is" becomes "SHALL be"; the ledger lists each such change in its clause table.
**Rationale**: One rule per requirement is the point of the limit. Reversibility: high.

### Source line placement

**Context**: A `Source:` line counts toward the 500 characters. Three `kubernetes-tier` requirements have a Source line of 495, 514 and 517 characters, which no body can carry whole, and other long Source lines leave little room for the rule.
**Decision**: A split block repeats the old requirement's whole `Source:` line when it fits (owner ruling of 2026-10-19). When it does not fit, the block carries the short form `Source: enhancement 0012.` (the enhancement that every line cites), or the title that supports its rule. Every title of an old Source line stays in at least one block of the same old requirement. Where the old text ties a Source line to a paragraph (the second paragraph of the apply-verdict requirement), the blocks of that paragraph keep that citation. Elsewhere the old text does not tie a title to a clause, so the assignment of a title to a block is a judgement: it follows the topic of the title (the deletion sequence, the order, the apply guard, the tier placement). The three requirements whose line cannot sit in one body split their titles; the table lists how each old requirement was handled.
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
- "A custom kind ending in Class falls back to the Class weight" in "A weight is looked up by group and version, then by group and kind, then by kind"
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
| A weight SHALL be looked up by group, version and kind first; then a CustomResourceDefinition o... | A weight is looked up by group and version, then by group and kind, then by kind | verbatim |
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
| An empty UUID on either side passes | Empty UUIDs, blank annotations and deleting objects pass the delete verdict, same name | no, byte for byte (checked) |
| Proceed carries the precondition for the judged object | A proceed verdict carries the judged object's identity, same name | no, byte for byte (checked) |
| A skip verdict yields no precondition | A skip verdict yields no precondition and carries a message, same name | no, byte for byte (checked) |
| A live object without a UID yields no precondition | A skip verdict yields no precondition and carries a message, same name | no, byte for byte (checked) |
| An object being deleted proceeds | Empty UUIDs, blank annotations and deleting objects pass the delete verdict, same name | no, byte for byte (checked) |
| An object annotated for another instance is left in place | A prune or an instance deletion leaves an object handed to another instance, same name | no, byte for byte (checked) |
| An annotation naming this instance changes nothing on delete | Empty UUIDs, blank annotations and deleting objects pass the delete verdict, same name | no, byte for byte (checked) |
| An empty instance UUID skips an annotated object | Empty UUIDs, blank annotations and deleting objects pass the delete verdict, same name | no, byte for byte (checked) |

New scenarios (`new:` lines of the byte check; each restates words of the old text or holds a moved table, and adds no case):

- "The delete verdict order" in "The delete verdict decides in a fixed order"

| Old clause | New requirement | Change |
| --- | --- | --- |
| The delete verdict SHALL decide in this order and stop at the first match: skip as `safety-excl... | "The delete verdict decides in a fixed order" | the five skips and the proceed are listed in the THEN and AND lines of the new scenario "The delete verdict order", conditions verbatim; the body keeps the SHALL, the first-match rule and the six outcomes, and names the scenario |
| An empty UUID on either side SHALL pass the owner comparison. | Empty UUIDs, blank annotations and deleting objects pass the delete verdict | verbatim |
| A blank adopt annotation SHALL pass the adoption comparison, and with an empty instance UUID an... | Empty UUIDs, blank annotations and deleting objects pass the delete verdict | verbatim |
| An adopt annotation naming this instance SHALL change nothing. | Empty UUIDs, blank annotations and deleting objects pass the delete verdict | verbatim |
| An object being deleted SHALL NOT be skipped for that reason. | Empty UUIDs, blank annotations and deleting objects pass the delete verdict | verbatim |
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

**The deletion plan orders inventory entries by kind-class delete order**: form c2 (REMOVED plus ADDED); old body 1249 characters; new bodies 411, 493, 267, 231.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| Entries are ordered for deletion | The deletion plan orders its entries by delete weight, same name | no, byte for byte (checked) |
| Equal weights keep their order | The deletion plan orders its entries by delete weight, same name | no, byte for byte (checked) |
| Protected kinds are marked up front | Protected kinds are marked safety-excluded in the plan before any live read, same name | no, byte for byte (checked) |
| A prune hands its stale set to the same plan | A deletion plan needs no render and no stored plan, same name | no, byte for byte (checked) |
| Changing the listed steps does not change the plan | The plan constructor is the only way to build a plan with steps, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| The library SHALL provide `opm/k8s/lifecycle` with a deletion plan built from a list of invento... | The deletion plan orders its entries by delete weight | verbatim |
| The plan SHALL order the entries by the tier's weight table in descending order, keeping the re... | The deletion plan orders its entries by delete weight | verbatim |
| Every CustomResourceDefinition and Namespace entry, as the tier's safety exclusion matches them... | Protected kinds are marked safety-excluded in the plan before any live read | verbatim |
| Building the plan SHALL NOT need a render or a stored plan: an uninstall builds it from the per... | A deletion plan needs no render and no stored plan | verbatim |
| The constructor SHALL be the only way to build a plan with steps, so no frontend can hand the t... | The plan constructor is the only way to build a plan with steps | verbatim |
| Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object... | Protected kinds are marked safety-excluded in the plan before any live read | verbatim |

**The deletion transition names one action per call**: form c2 (REMOVED plus ADDED); old body 1561 characters; new bodies 460, 149, 449, 288, 338.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A protected kind is skipped without a read | A safety-excluded step is skipped without a read, same name | no, byte for byte (checked) |
| An owned object is read, then deleted with Foreground propagation | Every other step is read, judged, and then skipped or deleted, same name | no, byte for byte (checked) |
| A foreign object is skipped with the verdict's reason | Every other step is read, judged, and then skipped or deleted, same name | no, byte for byte (checked) |
| Another instance's object is skipped | Every other step is read, judged, and then skipped or deleted, same name | no, byte for byte (checked) |
| A plan that does not prune names done at once | A plan that does not prune names done at once, same name | no, byte for byte (checked) |
| A prune built with the zero policy deletes nothing | A plan that does not prune names done at once, same name | no, byte for byte (checked) |
| The transition names only reads, deletions, skips and done | The deletion transition returns the next state and one action, same name | no, byte for byte (checked) |
| A prune of a dropped object leaves it for the adopting instance | A prune leaves an object that another instance adopted, same name | no, byte for byte (checked) |

| Old clause | New requirement | Change |
| --- | --- | --- |
| The deletion transition SHALL take the plan, the caller's state and the outcome of the action i... | The deletion transition returns the next state and one action | verbatim |
| It SHALL perform no action itself. | The deletion transition returns the next state and one action | verbatim |
| A safety-excluded step SHALL be reported as skipped with the reason `safety-excluded` without a... | A safety-excluded step is skipped without a read | verbatim |
| Every other step SHALL be read before it is judged, and the library's delete verdict SHALL judg... | Every other step is read, judged, and then skipped or deleted | "Every other step" becomes "Every step that is not safety-excluded" (the negation of the sentence before it in the old body); the rest is verbatim |
| A skip verdict SHALL be reported as skipped with the verdict's reason and message. | Every other step is read, judged, and then skipped or deleted | verbatim |
| A proceed verdict SHALL name a delete with Foreground propagation and the verdict's UID precond... | Every other step is read, judged, and then skipped or deleted | verbatim |
| When the policy does not prune, the first call SHALL name done and no step SHALL be read or del... | A plan that does not prune names done at once | verbatim |
| Every read, delete and skip action SHALL name a step of the plan, and the transition SHALL name... | The deletion transition returns the next state and one action | verbatim |
| So a prune of an object the instance dropped because another instance adopted it leaves the obj... | "A prune leaves an object that another instance adopted" | "So a prune ... leaves" becomes "A prune ... SHALL leave" |
| Source: enhancement 0012 ("The library owns the whole deletion sequence but only the per-object... | see "Source line placement" | titles split: second to the non-pruning piece, first to the judging piece, third with the number to the adopted-object piece; the ADR clause to the first two pieces |

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

**The deletion state is a serialisable value the caller owns**: form c2 (REMOVED plus ADDED); old body 1065 characters; new bodies 389, 344, 430, 235.

| Old scenario name | New place (requirement, scenario name) | Changed text? |
| --- | --- | --- |
| A round-tripped state advances identically | A state written to JSON advances identically, same name | no, byte for byte (checked) |
| Advancing twice names the same action | A state written to JSON advances identically, same name | no, byte for byte (checked) |
| A state past the plan is refused | A state that cannot belong to the plan is refused, same name | no, byte for byte (checked) |
| A state awaiting something after the last step is refused | A state that cannot belong to the plan is refused, same name | no, byte for byte (checked) |
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
| The transition SHALL return an error, the input state unchanged and no action when the state ca... | A state that cannot belong to the plan is refused | verbatim |
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
