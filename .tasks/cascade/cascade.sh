#!/usr/bin/env bash
# cascade.sh: move the library's upstream pins in the working tree. Run it as
# `task -x deps:cascade`. Exit 0 when the working tree changed, 3 when there
# was nothing to do, anything else on error. It never commits, branches or
# pushes.
#
# Workspace RELEASING.md, sections "The receiver" and "What each repo's task
# moves"; Phase 2 cascade contract §5.2 (rules 1-15) and §6.2 (the library's
# order); openspec change add-deps-cascade-task, design.md D3.
#
#   0  setup: clean tree, state directory, steering files, registries
#   A  resolve every target (core, catalog, language check); no edits
#   B  compute the module loop's scope; check for cue when it is non-empty
#   C  edit: loader, text re-pin of the test trees, module loop, warnings
set -euo pipefail
die() { printf 'cascade: %s\n' "$1" >&2; exit "${2:-1}"; }
here=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=SCRIPTDIR/lib.sh
. "$here/lib.sh"

cd "$(git rev-parse --show-toplevel)"
R="${CASCADE_RESOLVER:?CASCADE_RESOLVER is not set; run task -x deps:cascade}"
[ -n "${CASCADE_MODULE_GLOBS:-}" ] || die "CASCADE_MODULE_GLOBS is not set; run task -x deps:cascade"

# --- Phase 0: setup ---------------------------------------------------------

# snapshot: the working tree's state, for CASCADE_ALLOW_DIRTY=1 (contract §5.2
# rule 1).
snapshot() {
  { git status --porcelain --untracked-files=all
    git diff HEAD --binary
    git ls-files -z --others --exclude-standard | xargs -0 -r sha256sum
  } | sha256sum
}

start=""
if [ "${CASCADE_ALLOW_DIRTY:-}" = 1 ]; then
  start=$(snapshot)
elif [ -n "$(git status --porcelain --untracked-files=all)" ]; then
  die "the working tree is not clean; commit or stash first, or set CASCADE_ALLOW_DIRTY=1"
fi

STATE="$(git rev-parse --absolute-git-dir)/cascade"
mkdir -p "$STATE"
: >"$STATE/warnings"
export CASCADE_WARNINGS="$STATE/warnings"

# warn KEY MESSAGE: one <pin-key>\t<message> line, as the resolver writes them.
warn() {
  printf '%s\t%s\n' "$1" "$2" >>"$CASCADE_WARNINGS"
  printf 'cascade: warning: %s\n' "$2" >&2
}

"$R" check-files --repo-root .

# The registries are set here and never inherited (contract §5.2 rule 4). The
# library resolves no testing.opmodel.dev fixture from a registry.
export CUE_REGISTRY=opmodel.dev=ghcr.io/open-platform-model,registry.cue.works
export OPM_REGISTRY=opmodel.dev=ghcr.io/open-platform-model,registry.cue.works

# --- Phase A: resolve -------------------------------------------------------

# resolve KIND COORD CURRENT: print the target (the newer version on resolver
# exit 0, CURRENT on exit 3) and return 0; on any other resolver exit, print a
# diagnostic and exit with that code. Callers use T=$(resolve ...) under
# set -e: the exit only leaves the command substitution's subshell, and set -e
# then stops the script on the assignment's non-zero status.
resolve() {
  local out rc=0 e
  local -a expect=()
  for e in ${CASCADE_EXPECT:-}; do
    [ "${e%%=*}" != "$2" ] || expect=(--expect "${e#*=}")
  done
  out=$("$R" newest "$1" "$2" --current "$3" --repo-root . "${expect[@]}") || rc=$?
  case "$rc" in
    0) printf '%s\n' "$out" ;;
    3) printf '%s\n' "$3" ;;
    *) printf 'cascade: newest %s %s failed (exit %s)\n' "$1" "$2" "$rc" >&2; exit "$rc" ;;
  esac
}

# cmp_v A B: set CMP to the resolver's semver-cmp of A and B (-1, 0 or 1).
# Called directly, never inside $(...), so a failure stops the script.
cmp_v() {
  CMP=$("$R" semver-cmp "$1" "$2") || die "semver-cmp $1 $2 failed"
  case "$CMP" in -1|0|1) ;; *) die "semver-cmp $1 $2 printed '$CMP'" ;; esac
}

# frozen FILE KEY: exit 0 when .cascade-frozen freezes KEY at FILE, 1 when not.
frozen() {
  local rc=0
  "$R" is-frozen "$1" "$2" --repo-root . || rc=$?
  case "$rc" in
    0) return 0 ;;
    3) return 1 ;;
    *) die "is-frozen $1 $2 failed (exit $rc)" "$rc" ;;
  esac
}

# A1. Core, from DefaultSchemaModule. A frozen loader keeps core where it is
# before anything reads D, so the catalog check and the language check below
# judge the core the run will really have (contract §6.2 step 3).
D0=$(loader_core <"$LOADER")
[[ "$D0" =~ $SEMVER_RE ]] || die "cannot read DefaultSchemaModule from $LOADER"
if frozen "$LOADER" "$CORE_KEY"; then
  warn "$CORE_KEY" "\`$LOADER\` is frozen for \`$CORE_KEY\`; \`DefaultSchemaModule\` stays at \`$D0\`"
  D="$D0"
else
  D=$(resolve cue "$CORE_KEY" "$D0")
fi

# A2. The opm catalog, from the parity module. It moves only as far as the
# loader's core allows (contract §6.2 step 3).
P0=$(dep_v "$CATALOG_KEY" <"$PARITY_MOD")
[[ "$P0" =~ $SEMVER_RE ]] || die "cannot read the $CATALOG_KEY pin from $PARITY_MOD"
t=$(resolve cue "$CATALOG_KEY" "$P0")
K="$P0"
if [ "$t" != "$P0" ]; then
  rc=0
  c=$("$R" pin-of "$CATALOG_KEY" "$t" "$CORE_KEY") || rc=$?
  case "$rc" in
    0)
      cmp_v "$c" "$D"
      if [ "$CMP" = 1 ]; then
        warn "$CATALOG_KEY" "catalog \`$t\` needs core \`$c\`, newer than \`DefaultSchemaModule\` \`$D\`; advance core first"
      else
        K="$t"
      fi ;;
    3) K="$t" ;;
    *) die "pin-of $CATALOG_KEY $t $CORE_KEY failed (exit $rc)" "$rc" ;;
  esac
fi

# A3. language.version of every moved CUE upstream against the CUE version the
# required CUE job installs (contract §5.2 rule 10). A warning, never an edit.
CUE_PIN_FILE=.github/workflows/cue.yml
lang_check() { # lang_check KEY TARGET
  local lv rc=0
  lv=$("$R" language-of "$1" "$2") || rc=$?
  case "$rc" in
    0) ;;
    3) return 0 ;;
    *) die "language-of $1 $2 failed (exit $rc)" "$rc" ;;
  esac
  cmp_v "$lv" "$cue_pin"
  if [ "$CMP" = 1 ]; then
    warn "$1" "\`$1\` \`$2\` declares \`language.version\` \`$lv\`, newer than the CUE \`$cue_pin\` that \`$CUE_PIN_FILE\` installs"
  fi
}
if [ "$D" != "$D0" ] || [ "$K" != "$P0" ]; then
  cue_pin=""
  if [ -f "$CUE_PIN_FILE" ] &&
     ! cue_pin=$(yq -r '.jobs.cue.steps[] | select((.uses // "") | test("setup-cue")) | .with.version' "$CUE_PIN_FILE"); then
    cue_pin=""
  fi
  if [[ "$cue_pin" =~ $SEMVER_RE ]]; then
    [ "$D" = "$D0" ] || lang_check "$CORE_KEY" "$D"
    [ "$K" = "$P0" ] || lang_check "$CATALOG_KEY" "$K"
  else
    warn - "cannot read the setup-cue version from \`$CUE_PIN_FILE\`; \`language.version\` not checked"
  fi
fi

# --- Phase B: module loop scope ----------------------------------------------

# below_k V: exit 0 when catalog version V is below K. Equal strings need no
# resolver call, so a run with nothing to move asks nothing.
below_k() {
  [ -n "$1" ] && [ "$1" != "$K" ] || return 1
  cmp_v "$1" "$K"
  [ "$CMP" = -1 ]
}

# A module is in scope when its core differs from D, or when it pins the
# catalog below K. A catalog above K is never named, so never lowered.
scope=()
while IFS= read -r glob; do
  [ -n "$glob" ] || continue
  # shellcheck disable=SC2086 # the glob is meant to expand
  for dir in $glob; do
    f="$dir/cue.mod/module.cue"
    [ -f "$f" ] || continue
    mcore=$(dep_v "$CORE_KEY" <"$f")
    mcat=$(dep_v "$CATALOG_KEY" <"$f")
    if { [ -n "$mcore" ] && [ "$mcore" != "$D" ]; } || below_k "$mcat"; then
      scope+=("$dir")
    fi
  done
done <<<"$CASCADE_MODULE_GLOBS"

if [ "${#scope[@]}" -gt 0 ] && ! command -v cue >/dev/null 2>&1; then
  die "cue is required: ${#scope[@]} module(s) need cue mod get and tidy"
fi

# --- Phase C: edit -----------------------------------------------------------

# C1. The loader (shipped). One anchored literal edit, labelled for review. A
# frozen loader never gets here: phase A kept D at D0.
if [ "$D" != "$D0" ]; then
  before=$(cat "$LOADER"; printf x)
  sed -i -E "s|^(const DefaultSchemaModule = \"opmodel\\.dev/core@)[^\"]+(\")|\\1${D}\\2|" "$LOADER"
  changed=$({ diff <(printf '%s' "${before%x}") "$LOADER" || [ $? -eq 1 ]; } | { grep -c '^>' || [ $? -eq 1 ]; })
  if [ "$changed" != 1 ] || [ "$(loader_core <"$LOADER")" != "$D" ]; then
    die "the DefaultSchemaModule rewrite in $LOADER changed $changed line(s), not exactly one"
  fi
  warn "$CORE_KEY" "\`DefaultSchemaModule\` moved from \`$D0\` to \`$D\`; need-human-review: re-verify the glue (\`$LOADER\`) before merging"
fi

# C2. Text re-pin of the trees cue cannot resolve (contract §6.2 step 2): the
# testdata module root and every render tree a walk finds. Runs on every run,
# so a lagging tree catches up; never runs cue there.
[ -d testdata/render ] || die "testdata/render is missing"
trees=$(find testdata/render -path '*/cue.mod/module.cue' | LC_ALL=C sort)
while IFS= read -r f; do
  [ -n "$f" ] || continue
  v=$(dep_v "$CORE_KEY" <"$f")
  [ -n "$v" ] && [ "$v" != "$D" ] || continue
  if frozen "$f" "$CORE_KEY"; then continue; fi
  set_dep_v "$f" "$CORE_KEY" "$D" || die "cannot rewrite the $CORE_KEY block in $f"
done <<<"$(printf '%s\n%s\n' testdata/cue.mod/module.cue "$trees")"

# dep_keys FILE: every dependency key in a cue.mod/module.cue.
dep_keys() {
  { grep -oP '^\s*"\K[^"]+(?=":\s*\{\s*$)' "$1" || [ $? -eq 1 ]; }
}

# C3. The module loop: one explicit-version get and one tidy per module in
# scope. Third-party pins are never named.
for dir in "${scope[@]}"; do
  f="$dir/cue.mod/module.cue"
  mcore=$(dep_v "$CORE_KEY" <"$f")
  mcat=$(dep_v "$CATALOG_KEY" <"$f")
  args=(); frozen_keys=()
  if [ -n "$mcore" ]; then
    if frozen "$f" "$CORE_KEY"; then frozen_keys+=("$CORE_KEY"); else args+=("opmodel.dev/core@$D"); fi
  fi
  if [ -n "$mcat" ]; then
    if frozen "$f" "$CATALOG_KEY"; then
      frozen_keys+=("$CATALOG_KEY")
    elif below_k "$mcat"; then
      args+=("opmodel.dev/catalogs/opm@$K")
    fi
  fi
  [ "${#args[@]}" -gt 0 ] || continue

  declare -A was=()
  keys=$(dep_keys "$f")
  while IFS= read -r k; do
    [ -n "$k" ] || continue
    was["$k"]=$(dep_v "$k" <"$f")
  done <<<"$keys"

  printf 'cascade: %s: cue mod get %s\n' "$dir" "${args[*]}" >&2
  if ! out=$(cd "$dir" && cue mod get "${args[@]}" 2>&1 && cue mod tidy 2>&1); then
    die "$(printf '%s: cue mod get/tidy failed:\n%s' "$dir" "$out")"
  fi

  for k in "${frozen_keys[@]}"; do
    if [ "$(dep_v "$k" <"$f")" != "${was[$k]}" ]; then
      die "$f: tidy changed the frozen $k pin from ${was[$k]}; freeze the whole module, or hold the upstream"
    fi
  done
  for k in "${!was[@]}"; do
    case "$k" in opmodel.dev/*|testing.opmodel.dev/*) continue ;; esac
    now=$(dep_v "$k" <"$f")
    if [ "$now" != "${was[$k]}" ]; then
      warn - "tidy raised \`$k\` from \`${was[$k]}\` to \`$now\` in \`$f\`"
    fi
  done
  unset was
done

# C4. Prose that names a core release other than the loader's: a warning only.
for doc in docs/getting-started.md AGENTS.md; do
  [ -f "$doc" ] || continue
  named=$({ grep -oP 'opmodel\.dev/core@\Kv[0-9]+\.[0-9]+\.[0-9]+[0-9A-Za-z.+-]*' "$doc" || [ $? -eq 1 ]; } | LC_ALL=C sort -u)
  while IFS= read -r v; do
    [ -n "$v" ] && [ "$v" != "$D" ] || continue
    warn - "\`$doc\` still names \`opmodel.dev/core@$v\`"
  done <<<"$named"
done

# --- Result -------------------------------------------------------------------

if [ -n "$start" ]; then
  [ "$(snapshot)" != "$start" ] || exit 3
elif [ -z "$(git status --porcelain --untracked-files=all)" ]; then
  exit 3
fi
exit 0
