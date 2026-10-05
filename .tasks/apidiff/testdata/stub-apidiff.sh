#!/usr/bin/env bash
# stub-apidiff.sh: a stand-in for apidiff that .tasks/api-diff-test.sh passes
# as APIDIFF_BIN to test the wiring of main end to end.
#
#   -m -w <out> <module>          the "API" of the tree is its marker file
#                                 (tag, base or head), copied to <out>
#   -m -incompatible <old> <new>  prints a header and the canned diff for
#                                 <old>-><new>: tag->head is head.diff,
#                                 tag->base is base.diff; any other pair
#                                 (arguments swapped) fails
set -euo pipefail
data=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
[ "${1:-}" = -m ] || { echo "stub-apidiff: want -m first" >&2; exit 3; }
case "${2:-}" in
  -w) cp marker "$3" ;;
  -incompatible)
    pair="$(cat "$3")->$(cat "$4")"
    case "$pair" in
      'tag->head') f=head.diff ;;
      'tag->base') f=base.diff ;;
      *) echo "stub-apidiff: no canned diff for $pair" >&2; exit 3 ;;
    esac
    echo "Incompatible changes:"
    sed 's/^/- /' "$data/$f"
    ;;
  *) echo "stub-apidiff: unknown arguments: $*" >&2; exit 3 ;;
esac
