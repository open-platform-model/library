#!/usr/bin/env bash
set -euo pipefail

# consumer-build.sh — build and vet one library consumer against a library tree.
#
# The library's consumers, cli and opm-operator, require a released library
# version in their go.mod. This script points one consumer at a library tree
# through a go.work written in a throwaway work directory and selected with
# GOWORK, then runs `go build ./...` and `go vet ./...` in the consumer. No
# replace, go.work or go.work.sum is written into either checkout
# (ADR-013, decision j4). .github/workflows/consumer-build.yml runs it once per
# consumer on a library pull request.
#
# Usage:
#   bash .tasks/consumer-build.sh <consumer-dir> <library-dir> [<work-dir>]
#
#   <consumer-dir>  a git checkout of cli or opm-operator
#   <library-dir>   a git checkout of this library (a worktree with
#                   uncommitted edits is fine)
#   <work-dir>      where go.work and the step logs go; must not already hold a
#                   go.work; default: a new temporary directory
#   CONSUMER_NAME   the name the report uses; default: <consumer-dir>'s basename
#
# Run it with GOTOOLCHAIN=local to build with the installed Go, as CI does, so a
# library go line above the consumer's fails instead of downloading a toolchain.
#
# Steps: snapshot `git status --porcelain` of both checkouts; `go work init`;
# `go build ./...`; `go vet ./...` (which compiles the consumer's _test.go files
# too). The first failing step stops the build. On a failure the script prints
# the compiler's file:line:col lines (or, when none match, the log's last 30
# lines); with GITHUB_STEP_SUMMARY set it also appends them to the job summary
# with the consumer commit and prints a ::warning annotation. On every path it
# then compares both checkouts' status with the snapshot and fails if either
# changed. Exit 0 when the consumer builds and vets and both trees are
# unchanged, 1 otherwise, 2 on a usage error.

usage() {
  echo "usage: $0 <consumer-dir> <library-dir> [<work-dir>]" >&2
  exit 2
}

[ $# -ge 2 ] && [ $# -le 3 ] || usage

abs_dir() {
  [ -d "$1" ] || { echo "consumer-build: not a directory: $1" >&2; exit 2; }
  (cd "$1" && pwd -P)
}

consumer=$(abs_dir "$1")
library=$(abs_dir "$2")
if [ $# -eq 3 ]; then
  mkdir -p "$3"
  work=$(abs_dir "$3")
else
  work=$(mktemp -d)
fi
name=${CONSUMER_NAME:-$(basename "$consumer")}

if [ -e "$work/go.work" ]; then
  echo "consumer-build: $work already holds a go.work; use an empty work directory" >&2
  exit 2
fi

# A git failure must stop the script, never read as an empty status: set -e
# fails a plain assignment whose command substitution fails, and the explicit
# check makes that visible.
tree_status() {
  local out
  if ! out=$(git -C "$1" status --porcelain); then
    echo "consumer-build: git status failed in $1" >&2
    exit 1
  fi
  printf '%s' "$out"
}

before_consumer=$(tree_status "$consumer")
before_library=$(tree_status "$library")
consumer_commit=$(git -C "$consumer" rev-parse HEAD)

failed=""

echo "consumer-build: $name ($consumer_commit) against $library, work dir $work"

# Set before init: go work init writes to GOWORK when it is set, so a GOWORK
# from the caller's environment would put the file outside the work directory.
export GOWORK="$work/go.work"
if ! (cd "$work" && go work init "$consumer" "$library") 2>&1 | tee "$work/init.log"; then
  failed=init
fi

for step in build vet; do
  [ -z "$failed" ] || break
  echo "consumer-build: go $step ./... in $name"
  # -o /dev/null: a consumer whose ./... is one main package would otherwise
  # get a binary written into its checkout and fail the tree check.
  if [ "$step" = build ]; then
    args=(build -o /dev/null ./...)
  else
    args=(vet ./...)
  fi
  if ! go -C "$consumer" "${args[@]}" 2>&1 | tee "$work/$step.log"; then
    failed=$step
  fi
done

report() {
  local step=$1 log="$work/$1.log" lines command hint
  case $step in
    init)
      command="go work init"
      hint="The consumer and this library tree cannot share a workspace. A common cause is a library \`go\` line above the consumer's (the job runs with GOTOOLCHAIN=local)."
      ;;
    *)
      command="go $step ./..."
      hint="A red run means this PR changes API that $name's \`main\` still uses. Deprecate it instead (a \`Deprecated:\` comment, the API kept), and remove it in a later change once both consumers have migrated."
      ;;
  esac
  lines=$(grep -E '[^[:space:]]+\.go:[0-9]+(:[0-9]+)?: ' "$log" | head -n 50 || true)
  if [ -z "$lines" ]; then
    lines=$(tail -n 30 "$log")
  fi
  echo "consumer-build: $name failed at $command against this library tree:" >&2
  printf '%s\n' "$lines" >&2
  if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
    {
      echo "### Consumer build: $name fails \`$command\` against this library"
      echo
      echo "- Consumer commit: \`$consumer_commit\`"
      echo "- Failing step: \`$command\`"
      echo
      echo "$hint"
      echo
      echo '```text'
      printf '%s\n' "$lines"
      echo '```'
    } >>"$GITHUB_STEP_SUMMARY"
    echo "::warning title=Consumer build ($name)::$name fails $command against this library tree; see the job summary"
  fi
}

[ -z "$failed" ] || report "$failed"

changed=""
check_tree() {
  local label=$1 dir=$2 before=$3 after
  after=$(tree_status "$dir")
  if [ "$after" != "$before" ]; then
    echo "consumer-build: the $label checkout changed during the build ($dir):" >&2
    diff <(printf '%s\n' "$before") <(printf '%s\n' "$after") >&2 || true
    changed=yes
  fi
}
check_tree consumer "$consumer" "$before_consumer"
check_tree library "$library" "$before_library"

if [ -n "$failed" ] || [ -n "$changed" ]; then
  exit 1
fi
echo "consumer-build: $name builds and vets against this library tree"
