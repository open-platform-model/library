#!/usr/bin/env bash
# pins.sh WORKTREE|<ref>: the library's upstream pin report (Phase 2 cascade
# contract §4.1 and §6.2). One TSV row per pin:
#   <pin-key> <display> <class> <version> <labels>
# WORKTREE reads the files on disk; any other argument is a git ref read with
# git show. A pin whose file or block is missing at that ref is omitted. No
# cue: the report works offline and on a runner without cue.
set -euo pipefail
die() { printf 'pins.sh: %s\n' "$1" >&2; exit 1; }
here=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=SCRIPTDIR/lib.sh
. "$here/lib.sh"

[ $# -eq 1 ] || die "usage: pins.sh WORKTREE|<ref>"
ref="$1"
cd "$(git rev-parse --show-toplevel)"
if [ "$ref" != WORKTREE ]; then
  git rev-parse -q --verify "$ref^{commit}" >/dev/null || die "unknown ref $ref"
fi

# have PATH: the file exists at the ref.
have() {
  if [ "$ref" = WORKTREE ]; then [ -f "$1" ]; else git cat-file -e "$ref:$1" 2>/dev/null; fi
}
# show PATH: print the file at the ref.
show() {
  if [ "$ref" = WORKTREE ]; then cat "$1"; else git show "$ref:$1"; fi
}
# row KEY DISPLAY CLASS VERSION LABELS: print one row after checking VERSION.
row() {
  [ -n "$4" ] || return 0
  [[ "$4" =~ $SEMVER_RE ]] || die "$1 at $ref reads '$4', not a v-prefixed version"
  printf '%s\t%s\t%s\t%s\t%s\n' "$1" "$2" "$3" "$4" "$5"
}

core=""
if have "$LOADER"; then core=$(show "$LOADER" | loader_core); fi
catalog=""
if have "$PARITY_MOD"; then catalog=$(show "$PARITY_MOD" | dep_v "$CATALOG_KEY"); fi

row "$CORE_KEY" core shipped "$core" need-human-review
row "$CATALOG_KEY" "opm catalog" test "$catalog" ""
