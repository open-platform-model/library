# shellcheck shell=bash
# Shared helpers for the library's cascade scripts (pins.sh, cascade.sh,
# test.sh). Sourced, never run. Workspace RELEASING.md, section "The cascade";
# Phase 2 cascade contract §5 and §6.2.

# The constants are read by the scripts that source this file.
# shellcheck disable=SC2034

# A full v-prefixed SemVer (contract §2.2).
SEMVER_RE='^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$'

CORE_KEY='opmodel.dev/core@v2'
CATALOG_KEY='opmodel.dev/catalogs/opm@v4'
LOADER=opm/schema/loader.go
PARITY_MOD=testdata/parity/cue.mod/module.cue

# dep_v KEY: print the v: of the dependency block `"KEY": {` read from stdin,
# or nothing when the block is absent. Reads the canonical cue fmt layout
# every tracked cue.mod/module.cue has (one `v: "..."` line per block).
dep_v() {
  awk -v key="\"$1\": {" '
    { t = $0; sub(/^[ \t]+/, "", t); sub(/[ \t]+$/, "", t) }
    done { next }
    t == key { inb = 1; next }
    inb && t ~ /^v:[ \t]*"/ { v = t; sub(/^v:[ \t]*"/, "", v); sub(/".*$/, "", v); print v; done = 1; next }
    inb && t ~ /^}/ { done = 1 }
  '
}

# set_dep_v FILE KEY NEWV: rewrite only the v: inside the `"KEY": {` block of
# FILE, leaving every other byte alone. Returns 1 when the block or its v:
# line is missing. Writes through the existing inode (mode kept).
set_dep_v() {
  local f="$1" key="$2" nv="$3" out
  out=$(awk -v key="\"$key\": {" -v nv="$nv" '
    { t = $0; sub(/^[ \t]+/, "", t); sub(/[ \t]+$/, "", t) }
    t == key { inb = 1; print; next }
    inb && t ~ /^v:[ \t]*"/ { sub(/"[^"]*"/, "\"" nv "\""); inb = 0; done = 1; print; next }
    inb && t ~ /^}/ { inb = 0 }
    { print }
    END { exit done ? 0 : 1 }
  ' "$f" && printf x) || return 1
  printf '%s' "${out%x}" >"$f"
}

# loader_core: print the core version DefaultSchemaModule names, read from
# stdin, or nothing when the constant is absent (the Taskfile.yml idiom).
loader_core() {
  { grep -oP 'DefaultSchemaModule = "opmodel\.dev/core@\K[^"]+' || [ $? -eq 1 ]; } | head -n1
}
