#!/usr/bin/env bash
# api-diff-test.sh: offline tests of .tasks/api-diff.sh (task api:diff:test).
# No network, no Go build: the functions run against the fixture diffs in
# .tasks/apidiff/testdata, and the exit-2 paths fail before the tool build.
#
#   head.diff        tag->head: an inherited entry, the allowed core pin, an
#                    edited entry (moved again, so new) and a new entry
#   base.diff        tag->base for the same pull request, plus a synthetic
#                    line containing a new head entry (only a whole-line
#                    match makes an entry inherited)
#   compatible.diff  a head that adds nothing to base.diff
#
# Exit 0 when every check passes, 1 otherwise.
set -euo pipefail
here=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=SCRIPTDIR/api-diff.sh
. "$here/api-diff.sh"
data="$here/apidiff/testdata"

# Run the checks outside CI unless a check sets it itself.
unset GITHUB_ACTIONS GITHUB_STEP_SUMMARY BASE API_DIFF_BASE_REF

tmp=$(mktemp -d "${TMPDIR:-/tmp}/api-diff-test.XXXXXX")
trap 'rm -rf "$tmp"' EXIT

fails=0
ok() { printf 'ok   %s\n' "$1"; }
bad() { printf 'FAIL %s\n' "$1"; fails=$((fails + 1)); }
check() { # check <name> <command...>
  local name=$1
  shift
  if "$@"; then ok "$name"; else bad "$name"; fi
}
not() { ! "$@"; }
same() { # same <file> <expected lines...>
  local f=$1
  shift
  if [ $# -eq 0 ]; then [ ! -s "$f" ]; else diff -u <(printf '%s\n' "$@") "$f"; fi
}
count() { # count <file> <fixed string> <n>
  [ "$(grep -cF -- "$2" "$1" || true)" -eq "$3" ]
}

# --- mode_of ----------------------------------------------------------------
for t in v1.0.0-beta.4:warn v2.1.0-rc.1:warn v1.0.0:block v0.7.0:block v1.2.3:block; do
  check "mode_of ${t%%:*} is ${t##*:}" test "$(mode_of "${t%%:*}")" = "${t##*:}"
done

# --- allowed: a fixed prefix, nothing wider ---------------------------------
check "allowed: the core pin value change" \
  allowed './opm/schema.DefaultSchemaModule: value changed from "a" to "b"'
check "not allowed: the core pin removed" \
  not allowed './opm/schema.DefaultSchemaModule: removed'
check "not allowed: a longer name with the same start" \
  not allowed './opm/schema.DefaultSchemaModuleV2: value changed from "a" to "b"'
check "not allowed: the prefix elsewhere in the line" \
  not allowed './opm/x.Y: removed; ./opm/schema.DefaultSchemaModule: value changed from '

# --- split ------------------------------------------------------------------
mkdir "$tmp/split"
split "$data/head.diff" "$data/base.diff" "$tmp/split"
check "split: new entries (an edited entry counts as new)" same "$tmp/split/new" \
  './opm/x.C: value changed from 1 to 3' \
  './opm/y.F: removed'
check "split: inherited entries" same "$tmp/split/inherited" \
  './opm/helper/objectset: removed'
check "split: allowed entries" same "$tmp/split/allowed" \
  './opm/schema.DefaultSchemaModule: value changed from "opmodel.dev/core@v2.0.0-beta.2" to "opmodel.dev/core@v2.0.0-beta.4"'

mkdir "$tmp/compat"
split "$data/compatible.diff" "$data/base.diff" "$tmp/compat"
check "split: a compatible head adds nothing" same "$tmp/compat/new"

mkdir "$tmp/fresh"
split "$data/head.diff" /dev/null "$tmp/fresh"
check "split: with no base diff the core pin is still allowed" count "$tmp/fresh/allowed" DefaultSchemaModule 1
check "split: with no base diff every other entry is new" count "$tmp/fresh/new" './opm/' 3

# --- conclude: output and exit codes -----------------------------------------
# conclude_run <name> <tag> <dir> <ci:0|1>: runs conclude in a subshell and
# leaves its stdout in $tmp/<name>.out, the summary in $tmp/<name>.sum and
# its exit code in $tmp/<name>.rc.
conclude_run() {
  local name=$1 tag=$2 dir=$3 ci=$4 rc=0
  : >"$tmp/$name.sum"
  (
    if [ "$ci" = 1 ]; then export GITHUB_ACTIONS=true GITHUB_STEP_SUMMARY="$tmp/$name.sum"; fi
    conclude "$tag" "$dir"
  ) >"$tmp/$name.out" 2>&1 || rc=$?
  echo "$rc" >"$tmp/$name.rc"
}
rc_is() { [ "$(cat "$tmp/$1.rc")" = "$2" ]; }

conclude_run warn v1.0.0-beta.4 "$tmp/split" 1
check "warn with new entries exits 0" rc_is warn 0
check "warn names the mode" count "$tmp/warn.out" 'mode: warn' 1
check "warn annotates each new entry" count "$tmp/warn.out" '::warning title=Incompatible API change::' 2
check "warn never annotates an error" count "$tmp/warn.out" '::error' 0
check "warn names the feat! remedy" count "$tmp/warn.out" 'BREAKING CHANGE:' 1
check "warn lists the inherited entry" count "$tmp/warn.out" '- ./opm/helper/objectset: removed' 1
check "warn lists the allowed entry" count "$tmp/warn.out" 'Allowed (' 1
check "warn writes the job summary" count "$tmp/warn.sum" '## API diff against v1.0.0-beta.4' 1

conclude_run block v1.0.0 "$tmp/split" 1
check "block with new entries exits 1" rc_is block 1
check "block annotates each new entry as an error" count "$tmp/block.out" '::error title=Incompatible API change::' 2
check "block names the migration fragment" count "$tmp/block.out" 'migrations/unreleased/' 1

conclude_run compat v1.0.0 "$tmp/compat" 1
check "block with only inherited and allowed entries exits 0" rc_is compat 0
check "a compatible change says it adds none" count "$tmp/compat.out" 'This change adds no incompatible change since v1.0.0.' 1
check "a compatible change names no remedy" count "$tmp/compat.out" 'migrations/unreleased/' 0
check "a compatible change annotates nothing" count "$tmp/compat.out" '::' 0

conclude_run local v1.0.0 "$tmp/split" 0
check "outside CI block still exits 1" rc_is local 1
check "outside CI nothing is annotated" count "$tmp/local.out" '::' 0

mkdir "$tmp/many"
: >"$tmp/many/inherited"
: >"$tmp/many/allowed"
for i in $(seq -w 1 11); do echo "./opm/z.F$i: removed"; done >"$tmp/many/new"
conclude_run many v1.0.0-beta.4 "$tmp/many" 1
check "more than 10 annotations add a pointer to the summary" \
  count "$tmp/many.out" '11 incompatible changes; the job summary lists them all' 1
grep -F '::warning' "$tmp/many.out" >"$tmp/many.ann" || true
check "the pointer is the first annotation" \
  grep -qF '11 incompatible changes; the job summary lists them all' <(head -n 1 "$tmp/many.ann")
check "at most 10 annotations of the level are written" count "$tmp/many.ann" '::warning' 10
check "the first 9 entries follow the pointer" count "$tmp/many.ann" './opm/z.F0' 9

for i in $(seq -w 1 10); do echo "./opm/z.F$i: removed"; done >"$tmp/many/new"
conclude_run ten v1.0.0-beta.4 "$tmp/many" 1
check "exactly 10 entries are all annotated, with no pointer" count "$tmp/ten.out" '::warning' 10
check "exactly 10 entries add no pointer" count "$tmp/ten.out" 'the job summary lists them all' 0

# --- main: a failure of the check itself exits 2 (before any build) ----------
script="$here/api-diff.sh"
run_main() { # run_main <name> <dir> <env...>: exit code into $tmp/<name>.rc
  local name=$1 dir=$2 rc=0
  shift 2
  (cd "$dir" && env -u GITHUB_ACTIONS -u BASE -u API_DIFF_BASE_REF "$@" bash "$script") \
    >"$tmp/$name.out" 2>&1 || rc=$?
  echo "$rc" >"$tmp/$name.rc"
}

run_main badref "$here" API_DIFF_BASE_REF=0000000000000000000000000000000000000000
check "an unknown base commit exits 2" rc_is badref 2
check "an unknown base commit says to fetch it" count "$tmp/badref.out" 'is not in this clone' 1

run_main badtag "$here" BASE=v0.0.0-no-such-tag
check "a BASE that is not a tag exits 2" rc_is badtag 2

repo="$tmp/untagged"
git init -q "$repo"
git -C "$repo" -c user.name=t -c user.email=t@example.invalid commit -q --allow-empty -m init
run_main notag "$repo" GITHUB_ACTIONS=true
check "no reachable tag exits 2" rc_is notag 2
check "no reachable tag annotates the fetch-tags hint" count "$tmp/notag.out" '::error title=API diff::no v[0-9]* tag is reachable' 1

# In CI on a merge commit the base is its first parent, not the event's SHA.
git -C "$repo" checkout -q -b topic
git -C "$repo" -c user.name=t -c user.email=t@example.invalid commit -q --allow-empty -m topic
git -C "$repo" checkout -q -
git -C "$repo" -c user.name=t -c user.email=t@example.invalid commit -q --allow-empty -m main
first_parent=$(git -C "$repo" rev-parse HEAD)
git -C "$repo" -c user.name=t -c user.email=t@example.invalid merge -q --no-ff -m merge topic
run_main merge "$repo" GITHUB_ACTIONS=true API_DIFF_BASE_REF=0000000000000000000000000000000000000000
check "in CI a merge commit's first parent is the base" grep -qF -- "reachable from $first_parent" "$tmp/merge.out"

if [ "$fails" -gt 0 ]; then
  printf 'api-diff-test: %d check(s) failed\n' "$fails" >&2
  exit 1
fi
echo "api-diff-test: all checks passed"
