#!/usr/bin/env bash
# task api:diff: list the incompatible changes to the module's exported Go
# API (every non-internal package; all of them live under opm/) since the
# last release tag, and decide from that tag whether they warn or fail.
#
#   base tag  nearest v[0-9]* tag reachable from the base commit (never the
#             highest tag, so v0.7.0 does not decide while the beta line runs)
#   base      in CI the first parent of the checked-out merge commit, else
#             API_DIFF_BASE_REF (CI: the pull request's base SHA), else the
#             merge base of HEAD with origin/main, else HEAD
#   inherited an entry tag->base already lists: on the base branch before
#             this change, shown but never charged to it
#   allowed   an entry starting with a line of ALLOW (fixed prefixes, not
#             patterns): shown, never charged
#   mode      warn on a prerelease tag (a '-' suffix), block on a release;
#             warn exits 0, block exits 1 when the change adds an entry
#
# BASE=<tag> overrides the base tag for a local run; CI ignores it, so no
# variable or input changes the mode there. APIDIFF_BIN=<path> replaces the
# built tool for a local run (the offline tests' stub); CI ignores it too. Any failure of the check itself
# (tool build, export, no tag) exits 2 in both modes.
#
# Sourcing this file defines the functions only; .tasks/api-diff-test.sh
# tests them offline against fixture diffs.
set -euo pipefail

# Accepted incompatible entries, matched as fixed line prefixes. The release
# cascade rewrites this constant on every core move (.tasks/cascade/cascade.sh,
# phase C1); supervisor decision SD17 of the beta.1 walkthrough.
ALLOW=(
  './opm/schema.DefaultSchemaModule: value changed from '
)

in_ci() { [ -n "${GITHUB_ACTIONS:-}" ]; }
die() {
  if in_ci; then echo "::error title=API diff::$*"; fi
  echo "api:diff: $*" >&2
  exit 2
}

# mode_of <tag>: warn for a prerelease tag, block for a release.
mode_of() {
  case "$1" in
    v[0-9]*.[0-9]*.[0-9]*-*) echo warn ;;
    *) echo block ;;
  esac
}

# allowed <entry>: true when the entry starts with a line of ALLOW.
allowed() {
  local p
  for p in "${ALLOW[@]}"; do
    [[ "$1" == "$p"* ]] && return 0
  done
  return 1
}

# split <head.diff> <base.diff> <dir>: writes <dir>/new, <dir>/inherited and
# <dir>/allowed. Both inputs hold one entry per line. An allowed
# entry goes to allowed; an entry of head also in base (line for line) is
# inherited; the rest is new.
split() {
  local e
  : >"$3/new"
  : >"$3/inherited"
  : >"$3/allowed"
  while IFS= read -r e; do
    if allowed "$e"; then
      echo "$e" >>"$3/allowed"
    elif grep -qxF -- "$e" "$2"; then
      echo "$e" >>"$3/inherited"
    else
      echo "$e" >>"$3/new"
    fi
  done <"$1"
}

# report <tag> <mode> <dir>: the text for stdout and the job summary.
report() {
  local tag=$1 mode=$2 dir=$3 n_new n_inherited n_allowed
  n_new=$(wc -l <"$dir/new" | tr -d ' ')
  n_inherited=$(wc -l <"$dir/inherited" | tr -d ' ')
  n_allowed=$(wc -l <"$dir/allowed" | tr -d ' ')
  echo "## API diff against $tag"
  echo
  if [ "$mode" = warn ]; then
    echo "mode: warn ($tag is a prerelease: incompatible changes warn and pass)"
  else
    echo "mode: block ($tag is a release: incompatible changes fail the check)"
  fi
  echo
  if [ "$n_new" -eq 0 ]; then
    echo "This change adds no incompatible change since $tag."
  else
    echo "Incompatible changes in this change ($n_new):"
    echo
    sed 's/^/- /' "$dir/new"
    echo
    if [ "$mode" = warn ]; then
      echo "A breaking change needs a \`feat!\` commit with a \`BREAKING CHANGE:\` footer that is its migration note (ADR-010)."
    else
      echo "The check stays red. A breaking change is MAJOR (CONSTITUTION VI) and needs a migration fragment under \`migrations/unreleased/\` per \`migrations/README.md\` (ADR-004); a deliberate break also needs the owner's sign-off, since the post-GA route is still open."
    fi
  fi
  if [ "$n_inherited" -gt 0 ]; then
    echo
    echo "Already on the base branch since $tag ($n_inherited, not charged to this change):"
    echo
    sed 's/^/- /' "$dir/inherited"
  fi
  if [ "$n_allowed" -gt 0 ]; then
    echo
    echo "Allowed (the release cascade's core pin, never charged):"
    echo
    sed 's/^/- /' "$dir/allowed"
  fi
}

# conclude <tag> <dir>: prints the report, writes annotations and the job
# summary under GITHUB_ACTIONS, and returns 1 in block mode with new
# entries, 0 otherwise.
conclude() {
  local tag=$1 dir=$2 mode level n_new n_shown e
  mode=$(mode_of "$tag")
  n_new=$(wc -l <"$dir/new" | tr -d ' ')
  report "$tag" "$mode" "$dir"
  if in_ci; then
    if [ "$mode" = warn ]; then level=warning; else level=error; fi
    # The runner keeps only the first 10 annotations of a level per step
    # (actions/runner ExecutionContext, _maxCountPerIssueType) and drops the
    # rest. Past 10 entries the pointer to the summary goes first and only 9
    # entries follow it, so the pointer is never the one dropped.
    n_shown=$n_new
    if [ "$n_new" -gt 10 ]; then
      echo "::$level title=Incompatible API change::$n_new incompatible changes; the job summary lists them all"
      n_shown=9
    fi
    head -n "$n_shown" "$dir/new" | while IFS= read -r e; do
      e=${e//%/%25}
      echo "::$level title=Incompatible API change::$e"
    done
    [ -z "${GITHUB_STEP_SUMMARY:-}" ] || report "$tag" "$mode" "$dir" >>"$GITHUB_STEP_SUMMARY"
  fi
  if [ "$mode" = block ] && [ "$n_new" -gt 0 ]; then
    return 1
  fi
  return 0
}

main() {
  local ROOT TMP APIDIFF MODULE base_ref base_commit tag tag_commit
  ROOT=$(git rev-parse --show-toplevel)
  cd "$ROOT"

  # --- The base commit and the base tag (git only, before any build) -------
  if in_ci && git rev-parse -q --verify 'HEAD^2' >/dev/null; then
    # actions/checkout checks out refs/pull/N/merge: its first parent is the
    # base branch the pull request was merged onto, which the event's base
    # SHA may lag.
    base_ref=$(git rev-parse 'HEAD^1')
  elif [ -n "${API_DIFF_BASE_REF:-}" ]; then
    base_ref=$API_DIFF_BASE_REF
  elif git rev-parse -q --verify origin/main >/dev/null; then
    base_ref=$(git merge-base HEAD origin/main) || die "no merge base of HEAD and origin/main"
  else
    base_ref=HEAD
  fi
  base_commit=$(git rev-parse -q --verify "${base_ref}^{commit}") ||
    die "base commit $base_ref is not in this clone; fetch it (CI checks out with fetch-depth: 0)"

  if [ -n "${BASE:-}" ] && ! in_ci; then
    tag=$BASE
    git rev-parse -q --verify "refs/tags/$tag" >/dev/null || die "BASE=$tag is not a tag in this clone"
  else
    [ -z "${BASE:-}" ] || echo "api:diff: BASE is ignored in CI; the base tag comes from the base commit" >&2
    tag=$(git describe --tags --abbrev=0 --match 'v[0-9]*' "$base_commit" 2>/dev/null) ||
      die "no v[0-9]* tag is reachable from $base_commit; fetch the tags (git fetch --tags, or --unshallow in a shallow clone)"
  fi
  tag_commit=$(git rev-parse "${tag}^{commit}")

  TMP=$(mktemp -d "${TMPDIR:-/tmp}/api-diff.XXXXXX")
  # shellcheck disable=SC2064 # expand TMP now: it is local to main
  trap "chmod -R u+w '$TMP' 2>/dev/null || true; rm -rf '$TMP'" EXIT

  # Builds and package loads ignore any go.work above the checkout and any
  # GOFLAGS of the caller: -mod=readonly makes go.sum the trust root.
  export GOWORK=off GOFLAGS=-mod=readonly

  # --- The tool ---------------------------------------------------------------
  if [ -n "${APIDIFF_BIN:-}" ] && ! in_ci; then
    # Outside CI only: .tasks/api-diff-test.sh points this at a stub to test
    # the wiring below end to end without the tool.
    APIDIFF=$APIDIFF_BIN
  else
    [ -z "${APIDIFF_BIN:-}" ] || echo "api:diff: APIDIFF_BIN is ignored in CI; the tool is built from .tasks/apidiff" >&2
    APIDIFF="$TMP/bin/apidiff"
    go build -C .tasks/apidiff -o "$APIDIFF" golang.org/x/exp/cmd/apidiff ||
      die "building apidiff from .tasks/apidiff failed (a go.sum mismatch refuses the build)"
  fi

  # --- Export the three APIs --------------------------------------------------
  MODULE=$(go list -m)

  # export_at <commit> <out>: the API of a clean copy of <commit>.
  export_at() {
    local dir="$TMP/src-$1"
    mkdir -p "$dir"
    { git archive "$1" | tar -x -C "$dir"; } || die "extracting $1 failed"
    (cd "$dir" && "$APIDIFF" -m -w "$2" "$MODULE" 2>"$2.log") ||
      { cat "$2.log" >&2; die "exporting the API at $1 failed"; }
  }

  # entries <old> <new>: one incompatible change per line, "- " prefix
  # dropped. apidiff prints "Ignoring internal package" lines to stderr; they
  # go to a log that is shown only when the comparison fails.
  entries() {
    "$APIDIFF" -m -incompatible "$1" "$2" 2>"$TMP/compare.log" | sed -n 's/^- //p' | LC_ALL=C sort -u ||
      { cat "$TMP/compare.log" >&2; return 1; }
  }

  export_at "$tag_commit" "$TMP/tag.export"
  ("$APIDIFF" -m -w "$TMP/head.export" "$MODULE" 2>"$TMP/head.log") ||
    { cat "$TMP/head.log" >&2; die "exporting the API of the work tree failed"; }

  entries "$TMP/tag.export" "$TMP/head.export" >"$TMP/head.diff" || die "comparing $tag with the work tree failed"
  if [ "$base_commit" = "$tag_commit" ]; then
    : >"$TMP/base.diff"
  else
    export_at "$base_commit" "$TMP/base.export"
    entries "$TMP/tag.export" "$TMP/base.export" >"$TMP/base.diff" || die "comparing $tag with $base_commit failed"
  fi

  split "$TMP/head.diff" "$TMP/base.diff" "$TMP"
  conclude "$tag" "$TMP"
}

if [ "${BASH_SOURCE[0]}" = "$0" ]; then
  main "$@"
fi
