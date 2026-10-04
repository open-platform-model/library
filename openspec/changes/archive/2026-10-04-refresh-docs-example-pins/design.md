## Context

The cascade task (`.tasks/cascade/cascade.sh`, from `add-deps-cascade-task`, library PR 176) runs in three phases:

- **Phase A** resolves the targets. `D0` is the loader's core (`:106`). `D` is the core the run will have (`:107-113`).
- **Phase B** scopes the module loop.
- **Phase C** edits:
  - **C1** rewrites `DefaultSchemaModule` when `D != D0` (`:223-232`).
  - **C2** text-pins the test trees and calls `frozen` per file (`:236-246`).
  - **C3** is the module loop.
  - **C4** only warns about prose (`:288-296`). It greps `docs/getting-started.md` and `AGENTS.md` with `opmodel\.dev/core@\Kv[0-9]+\.[0-9]+\.[0-9]+[0-9A-Za-z.+-]*` and writes one warning, key `-`, per name that is not `D`.

The warnings go into the PR body (workspace `RELEASING.md`, "The cascade" › "Title from diff class"), and the receiver keeps them in its scratch warnings file (Phase 3 wiring contract §2.2).

As of `main` 8241250 there are three stale lines:

| File:line | Text | Matched by C4 |
| --- | --- | --- |
| `docs/getting-started.md:51` | `Module: "opmodel.dev/core@v2.0.0-beta.1",` | yes |
| `docs/getting-started.md:56` | `// → "v2.0.0-beta.1"` | no (no `opmodel.dev/core@` prefix) |
| `AGENTS.md:350` | `schema.OCILoader{Module: "opmodel.dev/core@v2.0.0-beta.1"}` | yes |

The loader is `opmodel.dev/core@v2.0.0-beta.2` (`opm/schema/loader.go:43`). `AGENTS.md:345` writes the placeholder `opmodel.dev/core@v2.X.Y[-pre]`, which the matcher never hits. No other line in either file names a core release.

The network scenario S2 (`.tasks/cascade/test.sh:311-335`) currently asserts that the docs warning appears (`:325`). It passes only because the docs are stale. A plain one-off bump would break S2 at that line: after the bump the docs name `D`, and `set_older` (`:168-180`) does not touch them.

## Goals / Non-Goals

**Goals:**

- The two examples name the loader's core now.
- After any cascade run that moves core, they name the new core in the same diff, with no human commit.
- A cascade run that does not move core never edits them.
- What the task cannot fix is still warned about.

**Non-Goals:**

- Generating the examples from Go. Markdown cannot reference a constant, and a doc-generation step is far more machinery than this needs (Principle VII).
- Covering any other file. These are the only prose files C4 watches. `docs/site/` pages carry no core literal today, and the docs bundle owns them.
- Changing the task's interface: exit codes, environment, or the title and body tasks (Phase 2 contract §5.1, §5.4).

## Decisions

### D1. The task edits the examples, not just warns

C4 becomes an edit step that runs only when core moved, and a warning for what remains.

### D2. Edit only in a run where C1 moved the loader

Condition: `D != D0`. A frozen loader keeps `D = D0` (`:108-110`), so it never edits.

Without this condition, a docs-only lag would be a diff on its own. Both files are `shipped` class (`.tasks/cascade/classes` lists only test paths), so the bot would open a `fix(deps)` PR that moves no pin. Gate G2 would then report a stale shipped pin on every release PR while the docs lag. G2 is `task -x deps:cascade` at the release head, and exit 0 with a shipped path means "behind" (Phase 3 wiring contract §8.2). Under the condition, a lag without a core move stays a warning and exit 3.

### D3. Rewrite the anchored examples of the same major; warn file-wide

When core moved, the step rewrites the version in every `Module: "opmodel.dev/core@<v>"` literal whose `<v>` is a release of `D`'s major, not only those equal to `D0`. A file that lagged before the run therefore catches up on the next core move instead of staying stuck.

The rewrite is anchored on `Module: "`, the way C1 anchors on `const DefaultSchemaModule = "`. Both real examples share that anchor (`docs/getting-started.md:51`, `AGENTS.md:350`). A core release named anywhere else, prose included, is never rewritten: `AGENTS.md` is a long steering file, and a sentence that names a release on purpose ("the floor is `opmodel.dev/core@v2.0.0-alpha.12`") must not move on each core release. `.cascade-frozen` can only freeze a whole file, so it is no tool for that.

A different major is never rewritten. Crossing a major is a hand-made PR (`RELEASING.md`, "Runbook" › "Handling a cascade PR", "new major available"), and the warning stays.

The rewrite matcher, in `sed -E`, is `(Module: "opmodel\.dev/core@)v<maj>\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?"`. The closing quote is part of the match, so build metadata (`v2.0.0+meta`) and trailing digits (`v2.0.01`) are never half-rewritten; they are left alone and warned about. `D` is checked against `SEMVER_RE` before it enters the `sed` program, so it carries no `#`, `&` or `\`.

The warning matcher stays file-wide, so whatever the rewrite leaves is still reported: `opmodel\.dev/core@\Kv[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]*[0-9A-Za-z])?(\+[0-9A-Za-z.-]*[0-9A-Za-z])?`. It ends on an alphanumeric, so a sentence-final period is never part of the version (the old class `[0-9A-Za-z.+-]*` swallowed it), and it keeps build metadata so `v2.0.0+meta` is warned about as itself.

### D4. One section, one `ci(cascade)` commit, the doc bump inside it

S2 is the test that proves the edit: it sets the examples to the older core and expects them back at the original bytes. It holds only when the original bytes name `D`.

- With the bump in its own earlier commit, S2 fails at `:325` in between, because no warning is raised once the docs are current.
- With the task change first, S2 fails because the base still names beta.1 and the run writes beta.2.

Both commits would be hidden types (`docs`, `ci`), so one commit loses nothing in the changelog and keeps every commit green under the full test set.

### D5. Line 56 carries no literal

`// → "v2.0.0-beta.1"` becomes `// → the version after the "@" in Module`. The output comment is a second copy of the same fact that no matcher anchors on. Removing it is cheaper than teaching the task a second pattern.

### D6. `.cascade-frozen` applies to the two files

This uses the existing `frozen FILE KEY` helper (`:92-101`), with key `opmodel.dev/core@v2`. C2 already calls it the same way in phase C. A frozen doc is left byte-unchanged and warned about. Nothing freezes them today. The rule exists so that the frozen mechanism means the same thing for every file the task edits. Because a frozen doc is a valid state that warns, no required test asserts that `main` carries no docs warning: the no-op scenario S1 makes no such assertion (plan review, finding 1, option a). S2 and S11 cover the behaviour, and S4 covers a frozen doc.

## Research & Decisions

### How the examples follow the loader

**Context**: The examples name a literal core release. The task warns about it on every run, and every core move brings the warning back.

**Explored** (sources: `.tasks/cascade/cascade.sh:288-296`; `openspec/specs/deps-cascade/spec.md:89-105`; Phase 3 follow-ups research, section (3); Phase 2 contract §6.2 step 5):

1. **One-off bump, warning kept.** This is the smallest change. The warning comes back on every core move, and a human must push a commit to `deps/cascade` to clear it. The research note leaned this way because the warning works as designed. The cost is a recurring manual step on every library core PR.
2. **Placeholder in the examples** (`opmodel.dev/core@v2.X.Y-pre`, as `AGENTS.md:345` does in prose). It never warns, but the example stops being copy-pasteable, and it drops the "update to the current release" goal.
3. **Skip `OCILoader{` lines in C4.** This hides the drift instead of fixing it.
4. **Edit with the loader** (D1 to D3). It is about 15 lines of shell, reuses the frozen and warn helpers, and adds no new file.

**Decision**: Option 4.

**Rationale**: The examples stay real and copy-pasteable. A core cascade PR carries its own docs fix, so the `need-human-review` reviewer looks only at the glue. The body's warnings block stays empty in the normal case, so a warning there means something again. The spec change is narrow: one added requirement, and one sentence and one scenario rewritten in an existing requirement. If the supervisor prefers option 1, sections and specs shrink to the bump plus the S2 assertion change, and the rest of this design falls away.

### C4, as it will read

```bash
# C4. The Module: "…" examples follow the loader when core moved; every
# other release named in the file warns.
[[ "$D" =~ $SEMVER_RE ]] || die "the core target '$D' is not a SemVer"
maj=${D%%.*}; maj=${maj#v}
for doc in docs/getting-started.md AGENTS.md; do
  [ -f "$doc" ] || continue
  if [ "$D" != "$D0" ] && ! frozen "$doc" "$CORE_KEY"; then
    sed -i -E "s#(Module: \"opmodel\\.dev/core@)v$maj\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?\"#\\1$D\"#g" "$doc"
    # assert: every anchored same-major example now names D
  fi
  # warn "-" for each release the file-wide matcher finds that is not D
done
```

The assert dies in the same spirit as C1's exactly-one-line check (`:227-231`). It cannot fire unless `sed` itself misbehaves, so the spec does not name it and no scenario tests it (plan review, finding 7).

### Tests

- **`set_older core|both`** also rewrites `Module: "opmodel.dev/core@<current loader>"` in the two docs to the older core.
- **S2**:
  - Drop the `:325` assertion.
  - Set the `docs/getting-started.md` example to `v2.0.0-alpha.12`, older than `older.tsv`'s core, so the run proves a lagging example catches up.
  - Add "no warning names `docs/getting-started.md` or `AGENTS.md`".
  - The existing `same_as_base` check proves that the docs came back.
- **S4** (network, already moves core):
  - Before `set_older`, add one prose line to `AGENTS.md` naming `opmodel.dev/core@v1.0.0` and `opmodel.dev/core@v2.0.0-alpha.12`.
  - Also append an anchored `    Module: "opmodel.dev/core@v1.0.0",` line, and freeze `docs/getting-started.md` for `opmodel.dev/core@v2`.
  - Expect `AGENTS.md` equal to that setup copy (the example back at the current core), `docs/getting-started.md` byte-unchanged after `set_older`, and warnings for both prose releases (a warning names `opmodel.dev/core@v1.0.0` in `AGENTS.md`; C4 dedupes versions per file, so the `cmp` against the setup copy is what proves the v1 example survives) and the frozen file's older example.
  - This proves "Another major is left alone", "A release named in prose is left alone" and the frozen-docs rule. `set_example` rewrites only the target version's major, so the v1 line survives the setup.
- **S11 (offline, new)**:
  - Set the `docs/getting-started.md` example to an older same-major core, then run `setup_commit` and run the task with the stub answering core as current.
  - Expect exit 3, an empty `status`, and a warning naming the file.
  - This proves D2 offline.

The network workflow's path filter gains `docs/getting-started.md` and `AGENTS.md`, since S2 and S4 now read them; otherwise only the weekly cron would catch an edit there that breaks S2.

## Risks / Trade-offs

- **[A human edits the examples into a form the matcher misses]**: then they are neither edited nor warned about. This is the same blind spot today's C4 has, and line 56 is the instance it already misses. Mitigation: D5 removes that instance, and the "Release cascade task" paragraph in `AGENTS.md` names the two files.
- **[The docs edit adds two paths to every core cascade PR]**: The reviewer sees a prose diff beside the loader. It does not change the title, because the loader already makes the diff `shipped`.
- **[Option 4 amends a Phase 2 contract step]**: The contract (§6.2 step 5) said warn only. This change rewrites the library spec, and the archived contract stays as history. The supervisor is told in the hand-off.
- **[A hand-made core bump leaves the examples behind]**: the next cascade run that does not move core warns about them, and the next core move brings them current. No required check fails in between (D6).

## Migration Plan

None. The next core cascade run uses the new C4. A rollback is a revert of the squash commit. The examples then stay at beta.2 and warn again on the next core move.
