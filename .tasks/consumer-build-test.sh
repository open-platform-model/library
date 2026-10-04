#!/usr/bin/env bash
# consumer-build-test.sh: offline tests for .tasks/consumer-build.sh. Run it as
# `task consumer-build:test`. Exit 0 when every case passes, 1 otherwise.
#
# Each case builds synthetic git repos in a throwaway directory: a library
# module (example.test/lib, func Hello) and a consumer that requires it at
# v1.0.0 with no go.sum. The workspace the script writes supplies the library,
# so nothing is downloaded: the cases run under GOPROXY=off GOTOOLCHAIN=local.
#
#   pass       the consumer builds and vets; exit 0
#   renamed    the library renames Hello; exit 1 and the job summary names
#              `undefined:` (this case kills a script that drops pipefail,
#              where `go ... | tee` would return tee's status)
#   stray      a go wrapper drops a file into the consumer during vet; exit 1
#              and the tree check names the file (this case kills a script
#              that skips the consumer tree check)
#   not-git    the consumer directory is not a git checkout; exit 1
#
# The mutant section then runs the renamed and stray cases against copies of
# the script with pipefail dropped and the consumer tree check removed, and
# fails unless each mutant is caught.
set -euo pipefail

here=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
SCRIPT="$here/consumer-build.sh"

export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
export GIT_AUTHOR_NAME=consumer-build-test GIT_AUTHOR_EMAIL=consumer-build-test@example.invalid
export GIT_COMMITTER_NAME=consumer-build-test GIT_COMMITTER_EMAIL=consumer-build-test@example.invalid
export GOPROXY=off GOTOOLCHAIN=local GOFLAGS=
unset GOWORK GITHUB_STEP_SUMMARY CONSUMER_NAME

TMP=$(mktemp -d)
trap 'chmod -R u+w "$TMP" 2>/dev/null; rm -rf "$TMP"' EXIT

fails=0
pass() { printf 'PASS %s\n' "$1"; }
fail() { printf 'FAIL %s: %s\n' "$1" "$2" >&2; fails=$((fails + 1)); }

commit_all() {
  git -C "$1" add -A
  git -C "$1" commit -q -m init
}

# make_library <dir> <func name>
make_library() {
  mkdir -p "$1"
  git -C "$1" init -q
  printf 'module example.test/lib\n\ngo 1.21\n' >"$1/go.mod"
  printf 'package lib\n\n// %s returns a greeting.\nfunc %s() string { return "hello" }\n' "$2" "$2" >"$1/lib.go"
  commit_all "$1"
}

# make_consumer <dir>: one main package, so a build without -o /dev/null
# would write a binary into the checkout.
make_consumer() {
  mkdir -p "$1"
  git -C "$1" init -q
  printf 'module example.test/consumer\n\ngo 1.21\n\nrequire example.test/lib v1.0.0\n' >"$1/go.mod"
  printf 'package main\n\nimport (\n\t"fmt"\n\n\t"example.test/lib"\n)\n\nfunc main() { fmt.Println(lib.Hello()) }\n' >"$1/main.go"
  commit_all "$1"
}

# run_case <name> <script> [env...]: runs the script with a fresh work dir and
# a job summary file; sets rc, out and summary.
run_case() {
  local name=$1 script=$2
  shift 2
  local dir="$TMP/$name"
  : >"$dir/summary.md"
  rc=0
  out=$(env "$@" GITHUB_STEP_SUMMARY="$dir/summary.md" CONSUMER_NAME=consumer \
    bash "$script" "$dir/consumer" "$dir/library" "$dir/work" 2>&1) || rc=$?
  summary=$(cat "$dir/summary.md")
}

# setup <name> <library func>
setup() {
  mkdir -p "$TMP/$1"
  make_library "$TMP/$1/library" "$2"
  make_consumer "$TMP/$1/consumer"
}

# A go wrapper that drops stray.txt into the current consumer during vet.
mkdir -p "$TMP/bin"
real_go=$(command -v go)
cat >"$TMP/bin/go" <<EOF
#!/usr/bin/env bash
if [ "\${1:-}" = -C ] && [ "\${3:-}" = vet ]; then
  : >"\$2/stray.txt"
fi
exec "$real_go" "\$@"
EOF
chmod +x "$TMP/bin/go"

case_pass() {
  local script=$1 label=$2
  setup "$label" Hello
  run_case "$label" "$script"
  if [ "$rc" -ne 0 ]; then
    fail "$label" "want exit 0, got $rc: $out"
  elif [ -n "$(git -C "$TMP/$label/consumer" status --porcelain)" ]; then
    fail "$label" "the consumer checkout changed"
  else
    pass "$label"
  fi
}

# case_renamed <script> <label>: echoes caught/missed for the mutant section.
case_renamed() {
  local script=$1 label=$2
  setup "$label" Greet
  run_case "$label" "$script"
  if [ "$rc" -eq 1 ] && grep -q 'undefined:' <<<"$summary" && grep -q '::warning' <<<"$out"; then
    return 0
  fi
  printf 'exit %s\n%s\n' "$rc" "$out" >"$TMP/$label.why"
  return 1
}

case_stray() {
  local script=$1 label=$2
  setup "$label" Hello
  run_case "$label" "$script" PATH="$TMP/bin:$PATH"
  if [ "$rc" -eq 1 ] && grep -q 'stray.txt' <<<"$out"; then
    return 0
  fi
  printf 'exit %s\n%s\n' "$rc" "$out" >"$TMP/$label.why"
  return 1
}

case_pass "$SCRIPT" pass

# A mutant counts as killed only when the unmutated script passes the same
# case; otherwise its failure says nothing about the mutation.
renamed_ok="" stray_ok=""
if case_renamed "$SCRIPT" renamed; then pass renamed; renamed_ok=yes; else fail renamed "$(cat "$TMP/renamed.why")"; fi
if case_stray "$SCRIPT" stray; then pass stray; stray_ok=yes; else fail stray "$(cat "$TMP/stray.why")"; fi

mkdir -p "$TMP/not-git/consumer"
make_library "$TMP/not-git/library" Hello
printf 'module example.test/consumer\n\ngo 1.21\n' >"$TMP/not-git/consumer/go.mod"
run_case not-git "$SCRIPT"
if [ "$rc" -eq 1 ] && grep -q 'git status failed' <<<"$out"; then
  pass not-git
else
  fail not-git "want exit 1 and 'git status failed', got $rc: $out"
fi

# Mutants: each must be caught by its case. A mutation that no longer applies
# (the script changed shape) is a failure, never a silent skip.
mutate() {
  local name=$1 expr=$2 dst="$TMP/mutant-$1.sh"
  sed -e "$expr" "$SCRIPT" >"$dst"
  if cmp -s "$SCRIPT" "$dst"; then
    fail "mutant $name" "the mutation no longer applies to consumer-build.sh"
    return 1
  fi
  printf '%s' "$dst"
}

if [ -z "$renamed_ok" ]; then
  fail "mutant no-pipefail" "not run: the renamed case fails on the unmutated script"
elif m=$(mutate no-pipefail 's/^set -euo pipefail$/set -eu/'); then
  if case_renamed "$m" mutant-no-pipefail; then
    fail "mutant no-pipefail" "survived: the renamed case still passes"
  else
    pass "mutant no-pipefail killed"
  fi
fi

if [ -z "$stray_ok" ]; then
  fail "mutant no-tree-check" "not run: the stray case fails on the unmutated script"
elif m=$(mutate no-tree-check '/^check_tree consumer /d'); then
  if case_stray "$m" mutant-no-tree-check; then
    fail "mutant no-tree-check" "survived: the stray case still passes"
  else
    pass "mutant no-tree-check killed"
  fi
fi

if [ "$fails" -ne 0 ]; then
  printf '%d consumer-build case(s) failed\n' "$fails" >&2
  exit 1
fi
echo "consumer-build: all cases pass"
