#!/usr/bin/env bash
# test.sh: the deps:cascade scenarios (Phase 2 cascade contract §8; openspec
# change add-deps-cascade-task, design.md D7). Run it as
# `task -x deps:cascade:test`. Exit 0 when every selected scenario passes,
# 1 otherwise.
#
# Each scenario runs `task -x deps:cascade` in a throwaway copy of the tree
# (its own git repo; nothing touches this checkout) against the contract §7
# resolver stub, with a stub table built from the copy's own pins, so a pin
# move on main never breaks the test.
#
#   CASCADE_TEST_SET=offline  checks, S1 no-op, S3 resolver error, S6 dirty
#                             tree, S7 lagging tree, S8 catalog needs core,
#                             S9 frozen loader, S10 frozen lag needs no cue,
#                             S11 lagging docs
#   CASCADE_TEST_SET=all      (default) also S2 older pins, S4 frozen module
#                             and docs file (and another-major and prose
#                             releases in AGENTS.md), S4b tidy
#                             raises a frozen key and S5 title and body
#                             (network: cue mod get resolves the real older
#                             versions)
#
# S5 runs only when CASCADE_RESOLVER_REAL names an executable real resolver;
# otherwise it prints SKIP S5.
set -euo pipefail
here=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=SCRIPTDIR/lib.sh
. "$here/lib.sh"

STUB="$here/testdata/stub-resolve.sh"
STUB_SUM=970130f7d55c07f5b86d4f5b6f392330427ff923eb34f93553656bcd4b893d9c
OLDER="$here/testdata/older.tsv"
S1_CALLS="$here/testdata/s1-calls.txt"
TODAY=2026-10-03
# The loop module S4, S4b and S10 use: both OPM keys, no third-party dep (S4
# freezes both, S4b and S10 core only).
S4_FILE=testdata/modules/web_app/cue.mod/module.cue

SET="${CASCADE_TEST_SET:-all}"
case "$SET" in offline|all) ;; *) printf 'test.sh: CASCADE_TEST_SET must be offline or all, not %s\n' "$SET" >&2; exit 1 ;; esac

REPO=$(git -C "$here" rev-parse --show-toplevel)
# The main checkout (also from inside a .claude/worktrees/* worktree), whose
# .cue-cache/mod warms the network scenarios.
MAIN=$(dirname "$(git -C "$REPO" rev-parse --path-format=absolute --git-common-dir)")

# The sandboxes are their own repos: no user or system git config, a fixed
# identity, and none of the caller's cascade env.
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
# No automatic maintenance or gc: a detached run that git commit starts can
# still write into a sandbox .git while cleanup removes it.
export GIT_CONFIG_COUNT=2 GIT_CONFIG_KEY_0=maintenance.auto GIT_CONFIG_VALUE_0=false \
  GIT_CONFIG_KEY_1=gc.auto GIT_CONFIG_VALUE_1=0
export GIT_AUTHOR_NAME=cascade-test GIT_AUTHOR_EMAIL=cascade-test@example.invalid
export GIT_COMMITTER_NAME=cascade-test GIT_COMMITTER_EMAIL=cascade-test@example.invalid
unset CASCADE_ALLOW_DIRTY CASCADE_EXPECT CASCADE_BASE CASCADE_WARNINGS CASCADE_NOTES_FILE \
  CASCADE_SOURCE CASCADE_TAGS CASCADE_STUB_TABLE CASCADE_STUB_LOG

TMP=$(mktemp -d)
# A retry covers any late writer the config above does not stop.
# shellcheck disable=SC2317,SC2329 # invoked by the trap (SC2317 before 0.10)
cleanup() {
  local _
  for _ in 1 2 3; do
    chmod -R u+w "$TMP" 2>/dev/null || true
    rm -rf "$TMP" 2>/dev/null && return 0
    sleep 1
  done
  chmod -R u+w "$TMP"
  rm -rf "$TMP"
}
trap cleanup EXIT

failed=0
pass() { printf 'PASS %s\n' "$1"; }
fail() { printf 'FAIL %s: %s\n' "$1" "$2"; failed=1; }

# The tree as it is, tracked and untracked-but-not-ignored, copied once.
(cd "$REPO" && git ls-files -z --cached --others --exclude-standard | tar --null -T - -cf "$TMP/tree.tar")

# sandbox NAME: a fresh copy in $TMP/NAME/r with one commit, "base"; cd there.
# Sets SB (the scenario dir) and BASE0 (the base commit).
sandbox() {
  SB="$TMP/$1"
  mkdir -p "$SB/r"
  tar -xf "$TMP/tree.tar" -C "$SB/r"
  cd "$SB/r"
  git init -q -b main
  git add -A
  git commit -q -m base
  BASE0=$(git rev-parse HEAD)
  : >"$SB/log"
}

# setup_commit: commit the scenario's setup edits and pin CASCADE_BASE to that
# SHA for every run in the scenario (contract §8 step 4).
setup_commit() {
  git add -A
  git commit -q --allow-empty -m setup
  CASCADE_BASE=$(git rev-parse HEAD)
  export CASCADE_BASE
}

# run_cascade: `task -x deps:cascade` in the sandbox against the stub; sets RC.
run_cascade() {
  RC=0
  CASCADE_RESOLVER="$STUB" CASCADE_STUB_TABLE="$SB/table.tsv" CASCADE_STUB_LOG="$SB/log" \
    CASCADE_TODAY="$TODAY" task -x deps:cascade >"$SB/out" 2>&1 || RC=$?
}

# tree_pin KEY: the version pins.sh reports for KEY in the sandbox.
tree_pin() { .tasks/cascade/pins.sh WORKTREE | awk -F'\t' -v k="$1" '$1 == k {print $4}'; }

# older KEY: the older published version older.tsv lists for KEY.
older() { awk -F'\t' -v k="$1" '$1 == k && NF == 2 {print $2}' "$OLDER"; }

# table [CORE_NEWEST [CATALOG_NEWEST]]: write the stub table from the sandbox's
# own pins (the "current rows"), optionally overriding a newest row, plus the
# older catalog's real pin-of row.
table() {
  local core cat
  core=$(tree_pin "$CORE_KEY"); cat=$(tree_pin "$CATALOG_KEY")
  {
    printf 'newest\tcue\t%s\t%s\n' "$CORE_KEY" "${1:-$core}"
    printf 'newest\tcue\t%s\t%s\n' "$CATALOG_KEY" "${2:-$cat}"
    printf 'pin-of\t%s\t%s\t%s\t%s\n' "$CATALOG_KEY" "$cat" "$CORE_KEY" "$core"
    printf 'language-of\t%s\t%s\tv0.17.0\n' "$CORE_KEY" "$core"
    printf 'language-of\t%s\t%s\tv0.17.0\n' "$CATALOG_KEY" "$cat"
    awk -F'\t' '$1 == "pin-of"' "$OLDER"
  } >"$SB/table.tsv"
}

# status: the sandbox's porcelain status, untracked files included.
status() { git status --porcelain --untracked-files=all; }

# warnings: the run's warnings file.
warnings() { cat "$(git rev-parse --absolute-git-dir)/cascade/warnings"; }

# --- Checks before the scenarios ----------------------------------------------

check_stub() {
  local sum
  sum=$(sha256sum "$STUB" | cut -d' ' -f1)
  if [ "$sum" = "$STUB_SUM" ]; then pass checksum
  else fail checksum "$STUB has sha256 $sum, not the contract §7 $STUB_SUM"; fi
}

check_pins() {
  sandbox pins
  local w h
  w=$(.tasks/cascade/pins.sh WORKTREE); h=$(.tasks/cascade/pins.sh HEAD)
  if [ -z "$w" ] || [ "$(printf '%s\n' "$w" | wc -l)" != 2 ]; then
    fail pins "pins.sh WORKTREE printed $(printf '%s\n' "$w" | wc -l) rows, not 2"
  elif [ "$w" != "$h" ]; then
    fail pins "pins.sh WORKTREE and pins.sh HEAD differ on a clean copy"
  else pass pins; fi
}

check_older() {
  sandbox older
  local key o t ok=1
  for key in "$CORE_KEY" "$CATALOG_KEY"; do
    o=$(older "$key"); t=$(tree_pin "$key")
    if [ -z "$o" ] || [ "$(CASCADE_STUB_TABLE=/dev/null "$STUB" semver-cmp "$o" "$t")" != -1 ]; then
      fail older "\`older.tsv\` \`$key\` \`$o\` is not older than the tree's \`$t\`; pick an older published version"
      ok=0
    fi
  done
  [ "$ok" = 0 ] || pass older
}

# net_env: give the scenario its own CUE cache beside the sandbox repo (never
# inside it), seeded with a copy of the main checkout's .cue-cache/mod when it
# exists. Never a symlink: the extract tree is read-only and shared.
net_env() {
  mkdir -p "$SB/cue-cache"
  if [ -d "$MAIN/.cue-cache/mod" ]; then cp -a "$MAIN/.cue-cache/mod" "$SB/cue-cache/"; fi
  export CUE_CACHE_DIR="$SB/cue-cache"
}

# DOCS: the files whose `Module: "opmodel.dev/core@…"` example follows the
# loader (openspec change refresh-docs-example-pins).
DOCS="docs/getting-started.md AGENTS.md"

# set_example FILE VERSION: point FILE's `Module: "opmodel.dev/core@…"`
# examples of VERSION's major at VERSION; exit when the file has none.
# Examples of another major stay, as they do in the task (S4).
set_example() {
  local maj=${2%%.*}
  sed -i -E "s#(Module: \"opmodel\\.dev/core@)$maj\\.[0-9][^\"]*\"#\\1$2\"#g" "$1"
  grep -qF "Module: \"opmodel.dev/core@$2\"" "$1" || { printf 'test.sh: %s has no core example to set to %s\n' "$1" "$2" >&2; exit 1; }
}

# docs_warned: print the warnings that name a docs file.
docs_warned() {
  # shellcheck disable=SC2016 # the backticks are message text
  { warnings | grep -e '`docs/getting-started.md`' -e '`AGENTS.md`' || [ $? -eq 1 ]; }
}

# set_older WHAT: write older.tsv's versions into every location the task
# moves. WHAT is core, catalog or both. Core: the loader, the docs examples
# and the core pin of every tracked cue.mod/module.cue; catalog: the catalog
# pin of every tracked cue.mod/module.cue that has one.
set_older() {
  local oc ok f mods doc
  oc=$(older "$CORE_KEY"); ok=$(older "$CATALOG_KEY")
  if [ "$1" != catalog ]; then
    sed -i -E "s|^(const DefaultSchemaModule = \"opmodel\\.dev/core@)[^\"]+|\\1$oc|" "$LOADER"
    [ "$(loader_core <"$LOADER")" = "$oc" ] || { printf 'test.sh: cannot set the loader to %s\n' "$oc" >&2; exit 1; }
    for doc in $DOCS; do set_example "$doc" "$oc"; done
  fi
  mods=$(git ls-files '*cue.mod/module.cue')
  while IFS= read -r f; do
    if [ "$1" != catalog ] && [ -n "$(dep_v "$CORE_KEY" <"$f")" ]; then set_dep_v "$f" "$CORE_KEY" "$oc"; fi
    if [ "$1" != core ] && [ -n "$(dep_v "$CATALOG_KEY" <"$f")" ]; then set_dep_v "$f" "$CATALOG_KEY" "$ok"; fi
  done <<<"$mods"
}

# same_as_base: the tree equals the original copy, untracked files included.
same_as_base() { git diff --quiet "$BASE0" && [ -z "$(git ls-files --others --exclude-standard)" ]; }

# --- Offline scenarios ----------------------------------------------------------

s1_noop() {
  sandbox s1
  setup_commit
  table
  # shellcheck disable=SC2016 # the backticks are message text
  printf 'warn\tcue\t%s\tnew major available: `v3.0.0`\n' "$CORE_KEY" >>"$SB/table.tsv"
  run_cascade
  local got
  got=$(sed -E 's/v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?/V/g' "$SB/log" | LC_ALL=C sort)
  if [ "$RC" != 3 ]; then fail S1 "exit $RC, not 3: $(tail -n3 "$SB/out")"
  elif [ -n "$(status)" ]; then fail S1 "the tree changed: $(status | head -n3)"
  elif [ "$got" != "$(cat "$S1_CALLS")" ]; then fail S1 "the resolver calls differ from s1-calls.txt: $(diff <(printf '%s\n' "$got") "$S1_CALLS" | head -n5 | tr '\n' ' ')"
  elif [ -n "$(awk '/^newest / && !(/ --current v/ && / --repo-root \. ?/)' "$SB/log")" ]; then fail S1 "a newest call lacks --current or --repo-root"
  elif ! warnings | grep -qF "$CORE_KEY"$'\t'"new major available: \`v3.0.0\`"; then fail S1 "the resolver's new-major warning did not reach the warnings file"
  else pass S1; fi
}

s3_error() {
  sandbox s3
  setup_commit
  table ERROR
  run_cascade
  if [ "$RC" = 0 ] || [ "$RC" = 3 ]; then fail S3 "exit $RC on a resolver error"
  elif [ -n "$(status)" ]; then fail S3 "the tree changed: $(status | head -n3)"
  else pass S3; fi
}

s6_dirty() {
  sandbox s6
  setup_commit
  table
  printf 'x\n' >cascade-test-untracked
  run_cascade
  if [ "$RC" != 1 ]; then fail S6 "exit $RC on a dirty tree, not 1"
  elif [ "$(status)" != "?? cascade-test-untracked" ]; then fail S6 "something else changed: $(status | head -n3)"
  else pass S6; fi
}

s7_lagging_tree() {
  sandbox s7
  local f
  f=$(find testdata/render -path '*/cue.mod/module.cue' | LC_ALL=C sort | head -n1)
  set_dep_v "$f" "$CORE_KEY" "$(older "$CORE_KEY")"
  setup_commit
  table
  run_cascade
  if [ "$RC" != 0 ]; then fail S7 "exit $RC, not 0: $(tail -n3 "$SB/out")"
  elif [ "$(status)" != " M $f" ]; then fail S7 "expected only $f to change: $(status | head -n3)"
  elif ! git diff --quiet "$BASE0" -- "$f"; then fail S7 "$f is not back to its original bytes"
  else pass S7; fi
}

s8_catalog_needs_core() {
  sandbox s8
  setup_commit
  table "" v4.999.0
  printf 'pin-of\t%s\tv4.999.0\t%s\tv2.999.0\n' "$CATALOG_KEY" "$CORE_KEY" >>"$SB/table.tsv"
  run_cascade
  if [ "$RC" != 3 ]; then fail S8 "exit $RC, not 3: $(tail -n3 "$SB/out")"
  elif [ -n "$(status)" ]; then fail S8 "the tree changed: $(status | head -n3)"
  elif ! warnings | grep -q 'advance core first'; then fail S8 "no \"advance core first\" warning"
  else pass S8; fi
}

# S9: a frozen loader is decided before the catalog and language checks, so
# a catalog that needs the newer core waits ("advance core first") and no
# language.version warning names a core move that never happens.
s9_frozen_loader() {
  sandbox s9
  [ -f .cascade-frozen ] || printf 'frozen:\n' >.cascade-frozen
  printf '  - path: %s\n    pins: ["%s"]\n    reason: "cascade test S9"\n' \
    "$LOADER" "$CORE_KEY" >>.cascade-frozen
  setup_commit
  table v2.999.0 v4.999.0
  printf 'pin-of\t%s\tv4.999.0\t%s\tv2.999.0\n' "$CATALOG_KEY" "$CORE_KEY" >>"$SB/table.tsv"
  printf 'language-of\t%s\tv2.999.0\tv0.99.0\n' "$CORE_KEY" >>"$SB/table.tsv"
  run_cascade
  if [ "$RC" != 3 ]; then fail S9 "exit $RC, not 3: $(tail -n3 "$SB/out")"
  elif [ -n "$(status)" ]; then fail S9 "the tree changed: $(status | head -n3)"
  elif grep -q "^newest cue $CORE_KEY " "$SB/log"; then fail S9 "core was resolved although the loader is frozen"
  elif ! warnings | grep -q 'is frozen for'; then fail S9 "no warning that the loader is frozen"
  elif ! warnings | grep -q 'advance core first'; then fail S9 "no \"advance core first\" warning"
  elif warnings | grep -q 'language.version'; then fail S9 "a language.version warning for a core move that never happens"
  else pass S9; fi
}

# no_cue_path: a directory of symlinks to the tools a stubbed run needs, and
# nothing else, so the run proves it never needs cue. A tool missing here is
# skipped; the run then fails loudly on it rather than passing by accident.
NO_CUE_TOOLS="task yq jq git bash sh env sed grep awk find sort uniq diff cmp
sha256sum xargs mktemp cat tr head tail cut wc dirname basename ls mkdir rm cp
mv touch date"
no_cue_path() {
  local t p
  mkdir -p "$SB/bin"
  for t in $NO_CUE_TOOLS; do
    p=$(command -v "$t" 2>/dev/null) || continue
    case "$p" in /*) ln -sf "$p" "$SB/bin/$t" ;; esac
  done
  printf '%s\n' "$SB/bin"
}

# S10: a loop module whose core is frozen below the loader has nothing to move,
# so a run is a no-op that needs no cue on PATH.
s10_frozen_lag_needs_no_cue() {
  sandbox s10
  set_dep_v "$S4_FILE" "$CORE_KEY" "$(older "$CORE_KEY")"
  [ -f .cascade-frozen ] || printf 'frozen:\n' >.cascade-frozen
  printf '  - path: %s\n    pins: ["%s"]\n    reason: "cascade test S10"\n' \
    "$S4_FILE" "$CORE_KEY" >>.cascade-frozen
  setup_commit
  table
  local p
  p=$(no_cue_path)
  PATH="$p" run_cascade
  if [ "$RC" != 3 ]; then fail S10 "exit $RC, not 3: $(tail -n3 "$SB/out")"
  elif [ -n "$(status)" ]; then fail S10 "the tree changed: $(status | head -n3)"
  else pass S10; fi
}

# S11: an example that lags while core does not move is warned about and left
# alone, so a docs lag on its own is never a diff (refresh-docs-example-pins,
# design.md D2).
s11_lagging_docs() {
  sandbox s11
  set_example docs/getting-started.md "$(older "$CORE_KEY")"
  setup_commit
  table
  run_cascade
  local want
  # shellcheck disable=SC2016 # the backticks are message text
  want='`docs/getting-started.md` still names `opmodel.dev/core@'"$(older "$CORE_KEY")"'`'
  if [ "$RC" != 3 ]; then fail S11 "exit $RC, not 3: $(tail -n3 "$SB/out")"
  elif [ -n "$(status)" ]; then fail S11 "the tree changed: $(status | head -n3)"
  elif ! warnings | grep -qF "$want"; then fail S11 "no warning names the lagging docs/getting-started.md"
  else pass S11; fi
}

# --- Network scenarios -----------------------------------------------------------

S2_DIR=""; S2_BASE=""
# Older than older.tsv's core; check_older does not cover it, so keep it a
# real v2 release below that one.
S2_DOC_CORE=v2.0.0-alpha.12

s2_older_pins() {
  sandbox s2
  net_env
  table
  # The moved catalog declares a language.version above the pinned CUE, so
  # the run must warn (and never edit a language.version).
  sed -i -E "s|^(language-of\t$CATALOG_KEY\t[^\t]+\t).*|\1v0.99.0|" "$SB/table.tsv"
  set_older both
  # One example lags further than the loader, so the run proves it catches
  # up on the next core move.
  set_example docs/getting-started.md "$S2_DOC_CORE"
  setup_commit
  run_cascade
  if [ "$RC" != 0 ]; then fail S2 "exit $RC, not 0: $(tail -n5 "$SB/out")"; return; fi
  # shellcheck disable=SC2016 # the backticks are message text
  if ! warnings | grep -q 'declares `language.version` `v0.99.0`'; then fail S2 "no language.version warning for the moved catalog"; return; fi
  if [ -n "$(docs_warned)" ]; then fail S2 "a warning names a docs file: $(docs_warned | head -n2 | tr '\n' ' ')"; return; fi
  # The library has no version-advance paths: the golden list is empty, so
  # the result must equal the original tree exactly.
  if ! same_as_base; then fail S2 "the result differs from the original tree: $(git diff --name-only "$BASE0" | head -n5 | tr '\n' ' ')"; return; fi
  git add -A
  git commit -q -m run1
  run_cascade
  if [ "$RC" != 3 ]; then fail S2 "the second run exited $RC, not 3: $(tail -n5 "$SB/out")"
  elif [ -n "$(status)" ]; then fail S2 "the second run changed the tree: $(status | head -n3)"
  else pass S2; S2_DIR="$SB"; S2_BASE="$CASCADE_BASE"; fi
}

# S4_PROSE: a prose line in AGENTS.md naming another major and an older v2
# release outside any `Module: "…"` example; the run must leave it and warn.
# S4 also freezes docs/getting-started.md for core, so its older example
# must stay byte-unchanged and be warned about.
# shellcheck disable=SC2016 # the backticks are prose text
S4_PROSE='Cascade test S4 prose: `opmodel.dev/core@v1.0.0` and `opmodel.dev/core@v2.0.0-alpha.12`.'
# S4_OTHER_MAJOR: an anchored example of another major; the run must leave
# it byte-unchanged (only the loader's major moves) and warn.
S4_OTHER_MAJOR='    Module: "opmodel.dev/core@v1.0.0",'

s4_frozen() {
  sandbox s4
  net_env
  table
  printf '\n%s\n%s\n' "$S4_PROSE" "$S4_OTHER_MAJOR" >>AGENTS.md
  cp AGENTS.md "$SB/agents.want"
  set_older both
  # docs/getting-started.md is frozen for core, so its older example stays.
  cp docs/getting-started.md "$SB/gs.want"
  [ -f .cascade-frozen ] || printf 'frozen:\n' >.cascade-frozen
  printf '  - path: %s\n    pins: ["%s", "%s"]\n    reason: "cascade test S4"\n' \
    "$S4_FILE" "$CORE_KEY" "$CATALOG_KEY" >>.cascade-frozen
  printf '  - path: %s\n    pins: ["%s"]\n    reason: "cascade test S4 docs"\n' \
    docs/getting-started.md "$CORE_KEY" >>.cascade-frozen
  setup_commit
  cp "$S4_FILE" "$SB/frozen.before"
  run_cascade
  local other w1 w2 w3
  other=$(git diff --name-only "$BASE0" | grep -v -x -e "$S4_FILE" -e .cascade-frozen -e AGENTS.md -e docs/getting-started.md || [ $? -eq 1 ])
  # shellcheck disable=SC2016 # the backticks are message text
  w1='`AGENTS.md` still names `opmodel.dev/core@v1.0.0`' w2='`AGENTS.md` still names `opmodel.dev/core@v2.0.0-alpha.12`'
  w3="\`docs/getting-started.md\` still names \`opmodel.dev/core@$(older "$CORE_KEY")\`"
  if [ "$RC" != 0 ]; then fail S4 "exit $RC, not 0: $(tail -n5 "$SB/out")"
  elif ! cmp -s "$S4_FILE" "$SB/frozen.before"; then fail S4 "the frozen $S4_FILE changed"
  elif [ -n "$other" ]; then fail S4 "other files differ from the original tree: $(printf '%s' "$other" | head -n5 | tr '\n' ' ')"
  elif ! cmp -s AGENTS.md "$SB/agents.want"; then fail S4 "AGENTS.md is not its setup copy with the example current: $(diff "$SB/agents.want" AGENTS.md | head -n4 | tr '\n' ' ')"
  elif ! cmp -s docs/getting-started.md "$SB/gs.want"; then fail S4 "the frozen docs/getting-started.md changed: $(diff "$SB/gs.want" docs/getting-started.md | head -n4 | tr '\n' ' ')"
  elif ! warnings | grep -qF "$w1"; then fail S4 "no warning for the v1 release in AGENTS.md"
  elif ! warnings | grep -qF "$w2"; then fail S4 "no warning for the older v2 release in AGENTS.md prose"
  elif ! warnings | grep -qF "$w3"; then fail S4 "no warning for the frozen docs/getting-started.md example"
  else pass S4; fi
}

# S4b: a key frozen for one module, and tidy raises it by MVS when the other
# key moves. The task must refuse (contract §5.2 rule 8), naming the key.
# web_app's core is frozen at v2.0.0-alpha.12 and its catalog is older.tsv's,
# whose core (v2.0.0-beta.1) is newer, so moving the catalog raises core.
S4B_CORE=v2.0.0-alpha.12

s4b_tidy_raises_frozen() {
  sandbox s4b
  net_env
  set_dep_v "$S4_FILE" "$CORE_KEY" "$S4B_CORE"
  set_dep_v "$S4_FILE" "$CATALOG_KEY" "$(older "$CATALOG_KEY")"
  [ -f .cascade-frozen ] || printf 'frozen:\n' >.cascade-frozen
  printf '  - path: %s\n    pins: ["%s"]\n    reason: "cascade test S4b"\n' \
    "$S4_FILE" "$CORE_KEY" >>.cascade-frozen
  setup_commit
  table
  run_cascade
  local other
  other=$(git diff --name-only HEAD | grep -v -x -e "$S4_FILE" || [ $? -eq 1 ])
  if [ "$RC" = 0 ] || [ "$RC" = 3 ]; then fail S4b "exit $RC, not a refusal: $(tail -n3 "$SB/out")"
  elif ! grep -qF "tidy changed the frozen $CORE_KEY pin from $S4B_CORE" "$SB/out"; then fail S4b "the refusal does not name the frozen key: $(tail -n3 "$SB/out")"
  elif [ -n "$other" ]; then fail S4b "files other than $S4_FILE changed: $(printf '%s' "$other" | head -n5 | tr '\n' ' ')"
  else pass S4b; fi
}

# real VERB: `task -x deps:cascade:VERB` against the real resolver; sets OUT, RC.
real() {
  RC=0
  OUT=$(CASCADE_RESOLVER="$CASCADE_RESOLVER_REAL" task -x "deps:cascade:$1" 2>"$SB/real.err") || RC=$?
}

s5_title_body() {
  if [ -z "${CASCADE_RESOLVER_REAL:-}" ] || [ ! -x "$CASCADE_RESOLVER_REAL" ]; then
    printf 'SKIP S5: CASCADE_RESOLVER_REAL does not name an executable resolver\n'
    return
  fi
  if [ -z "$S2_DIR" ]; then fail S5 "S2 did not pass, so there is no diff to title"; return; fi
  SB="$S2_DIR"; cd "$SB/r"
  export CASCADE_BASE="$S2_BASE"
  local core cat want
  core=$(tree_pin "$CORE_KEY"); cat=$(tree_pin "$CATALOG_KEY")
  want="fix(deps): bump core to $core and opm catalog to $cat"
  real title
  if [ "$RC" != 0 ] || [ "$OUT" != "$want" ]; then fail S5 "title exited $RC and printed '$OUT', not '$want'"; return; fi
  real body
  if [ "$RC" != 0 ]; then fail S5 "body exited $RC: $(tail -n3 "$SB/real.err")"; return; fi
  if ! grep -qxF "<!-- cascade-title: $want -->" <<<"$OUT"; then fail S5 "body lacks the cascade-title marker"; return; fi
  if ! grep -qx '<!-- cascade-labels: need-human-review -->' <<<"$OUT"; then fail S5 "body's cascade-labels marker is not need-human-review"; return; fi
  if [ "$(grep -c -e '^| core (' -e '^| opm catalog (' <<<"$OUT")" != 2 ]; then fail S5 "body does not have one row per moved pin"; return; fi
  if [ "$(grep '^## ' <<<"$OUT" | tail -n1)" != "## Notes" ]; then fail S5 "## Notes is not the last section"; return; fi

  # Catalog-only variant: only the catalog moves, so the diff is test-class.
  sandbox s5-catalog
  net_env
  table
  set_older catalog
  setup_commit
  run_cascade
  if [ "$RC" != 0 ]; then fail S5 "catalog-only run exited $RC, not 0: $(tail -n5 "$SB/out")"; return; fi
  want="test(fixtures): bump opm catalog to $cat"
  real title
  if [ "$RC" != 0 ] || [ "$OUT" != "$want" ]; then fail S5 "catalog-only title exited $RC and printed '$OUT', not '$want'"; return; fi
  real body
  if [ "$RC" != 0 ] || ! grep -qE '^<!-- cascade-labels: *-->$' <<<"$OUT"; then fail S5 "catalog-only body exited $RC or carries a label"; return; fi
  pass S5
}

check_stub
check_pins
check_older
s1_noop
s3_error
s6_dirty
s7_lagging_tree
s8_catalog_needs_core
s9_frozen_loader
s10_frozen_lag_needs_no_cue
s11_lagging_docs
if [ "$SET" = all ]; then
  s2_older_pins
  s4_frozen
  s4b_tidy_raises_frozen
  s5_title_body
fi

exit "$failed"
