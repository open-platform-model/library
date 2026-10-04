#!/usr/bin/env bash
# task api:diff: list the incompatible changes to the module's exported Go
# API (every non-internal package; all of them live under opm/) since the
# last release tag, and decide from that tag whether they warn or fail.
#
#   base tag  nearest v[0-9]* tag reachable from the base commit (never the
#             highest tag, so v0.7.0 does not decide while the beta line runs)
#   base      API_DIFF_BASE_REF (CI: the pull request's base SHA), else the
#             merge base of HEAD with origin/main, else HEAD
#   inherited an entry tag->base already lists: on the base branch before
#             this change, shown but never charged to it
#   mode      warn on a prerelease tag (a '-' suffix), block on a release;
#             warn exits 0, block exits 1 when the change adds an entry
#
# BASE=<tag> overrides the base tag for a local run; CI ignores it, so no
# variable or input changes the mode there. Any failure of the check itself
# (tool build, export, no tag) exits non-zero in both modes.
set -euo pipefail

ROOT=$(git rev-parse --show-toplevel)
cd "$ROOT"

in_ci() { [ -n "${GITHUB_ACTIONS:-}" ]; }
die() {
  if in_ci; then echo "::error title=API diff::$*"; fi
  echo "api:diff: $*" >&2
  exit 2
}

TMP=$(mktemp -d "${TMPDIR:-/tmp}/api-diff.XXXXXX")
# shellcheck disable=SC2329 # run by the EXIT trap
cleanup() { chmod -R u+w "$TMP" 2>/dev/null || true; rm -rf "$TMP"; }
trap cleanup EXIT

# Builds and package loads ignore any go.work above the checkout and any
# GOFLAGS of the caller: -mod=readonly makes go.sum the trust root.
export GOWORK=off GOFLAGS=-mod=readonly

# --- The tool -----------------------------------------------------------------
APIDIFF="$TMP/bin/apidiff"
go build -C .tasks/apidiff -o "$APIDIFF" golang.org/x/exp/cmd/apidiff ||
  die "building apidiff from .tasks/apidiff failed (a go.sum mismatch refuses the build)"

# --- The base commit and the base tag -----------------------------------------
if [ -n "${API_DIFF_BASE_REF:-}" ]; then
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

case "$tag" in
  v[0-9]*.[0-9]*.[0-9]*-*) mode=warn ;;
  *) mode=block ;;
esac

# --- Export the three APIs ----------------------------------------------------
MODULE=$(go list -m)

# export_at <commit> <out>: the API of a clean copy of <commit>.
export_at() {
  local dir="$TMP/src-$1"
  mkdir -p "$dir"
  git archive "$1" | tar -x -C "$dir"
  (cd "$dir" && "$APIDIFF" -m -w "$2" "$MODULE" 2>"$2.log") ||
    { cat "$2.log" >&2; die "exporting the API at $1 failed"; }
}

export_at "$tag_commit" "$TMP/tag.export"
("$APIDIFF" -m -w "$TMP/head.export" "$MODULE" 2>"$TMP/head.log") ||
  { cat "$TMP/head.log" >&2; die "exporting the API of the work tree failed"; }

# entries <old> <new>: one incompatible change per line, "- " prefix dropped.
# apidiff prints "Ignoring internal package" lines to stderr; they go to a
# log that is shown only when the comparison fails.
entries() {
  "$APIDIFF" -m -incompatible "$1" "$2" 2>"$TMP/compare.log" | sed -n 's/^- //p' | LC_ALL=C sort -u ||
    { cat "$TMP/compare.log" >&2; return 1; }
}

entries "$TMP/tag.export" "$TMP/head.export" >"$TMP/head.diff" || die "comparing $tag with the work tree failed"
if [ "$base_commit" = "$tag_commit" ]; then
  : >"$TMP/base.diff"
else
  export_at "$base_commit" "$TMP/base.export"
  entries "$TMP/tag.export" "$TMP/base.export" >"$TMP/base.diff" || die "comparing $tag with $base_commit failed"
fi

LC_ALL=C comm -12 "$TMP/head.diff" "$TMP/base.diff" >"$TMP/inherited"
LC_ALL=C comm -23 "$TMP/head.diff" "$TMP/base.diff" >"$TMP/new"
n_new=$(wc -l <"$TMP/new" | tr -d ' ')
n_inherited=$(wc -l <"$TMP/inherited" | tr -d ' ')

# --- Report ---------------------------------------------------------------------
if [ "$mode" = warn ]; then
  mode_line="mode: warn ($tag is a prerelease: incompatible changes warn and pass)"
  remedy="A breaking change needs a \`feat!\` commit with a \`BREAKING CHANGE:\` footer that is its migration note (ADR-010)."
  level=warning
else
  mode_line="mode: block ($tag is a release: incompatible changes fail the check)"
  remedy="A breaking change is MAJOR (CONSTITUTION VI) and needs a migration fragment under \`migrations/unreleased/\` per \`migrations/README.md\` (ADR-004)."
  level=error
fi

report() {
  echo "## API diff against $tag"
  echo
  echo "$mode_line"
  echo
  if [ "$n_new" -eq 0 ]; then
    echo "The API is compatible with $tag: this change adds no incompatible change."
  else
    echo "Incompatible changes in this change ($n_new):"
    echo
    sed 's/^/- /' "$TMP/new"
    echo
    echo "$remedy"
  fi
  if [ "$n_inherited" -gt 0 ]; then
    echo
    echo "Already on the base branch since $tag ($n_inherited, not charged to this change):"
    echo
    sed 's/^/- /' "$TMP/inherited"
  fi
}

report
if in_ci; then
  while IFS= read -r e; do
    e=${e//%/%25}
    echo "::$level title=Incompatible API change::$e"
  done <"$TMP/new"
  [ -z "${GITHUB_STEP_SUMMARY:-}" ] || report >>"$GITHUB_STEP_SUMMARY"
fi

if [ "$mode" = block ] && [ "$n_new" -gt 0 ]; then
  exit 1
fi
exit 0
