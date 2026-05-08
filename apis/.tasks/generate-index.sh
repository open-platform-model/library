#!/usr/bin/env bash
set -euo pipefail

# generate-index.sh — Print INDEX.md to stdout for a CUE module.
#
# Usage (run from library/apis/ directory):
#   bash .tasks/generate-index.sh core/v1alpha1
#   bash .tasks/generate-index.sh core/v1alpha2
#
# The caller redirects stdout to the desired output file:
#   bash .tasks/generate-index.sh core/v1alpha1 > core/v1alpha1/INDEX.md

MODULE_RELDIR="${1:?Error: module_dir argument required. Usage: bash .tasks/generate-index.sh core/v1alpha1}"
MODULE_DIR="${MODULE_RELDIR%/}"

# ── Fail fast: validate required paths exist ──────────────────────────────────

[[ -d "$MODULE_DIR" ]] \
    || { echo "Error: not a directory: $MODULE_DIR" >&2; exit 1; }
[[ -f "$MODULE_DIR/cue.mod/module.cue" ]] \
    || { echo "Error: missing $MODULE_DIR/cue.mod/module.cue" >&2; exit 1; }

# ── Parse module name from cue.mod/module.cue ─────────────────────────────────

MODULE_NAME=$(
    grep '^module:' "$MODULE_DIR/cue.mod/module.cue" \
    | sed 's/module:[[:space:]]*"\(.*\)"/\1/'
)

# Version label = trailing path component (e.g. v1alpha1, v1alpha2, v1)
MODULE_VERSION=$(basename "$MODULE_DIR")

# ── ASCII directory tree (dirs only, no cue.mod/, pure ASCII) ─────────────────

print_tree() {
    local dir="$1" prefix="$2"
    local subdirs=()
    mapfile -t subdirs < <(
        find "$dir" -maxdepth 1 -mindepth 1 -type d ! -name "cue.mod" | sort
    )
    local total="${#subdirs[@]}" idx=0
    for subdir in "${subdirs[@]}"; do
        idx=$((idx + 1))
        local name
        name=$(basename "$subdir")
        if [[ "$idx" -eq "$total" ]]; then
            printf '%s+-- %s/\n' "$prefix" "$name"
            print_tree "$subdir" "${prefix}    "
        else
            printf '%s+-- %s/\n' "$prefix" "$name"
            print_tree "$subdir" "${prefix}|   "
        fi
    done
}

# ── Extract top-level definitions from one CUE file ───────────────────────────
# Outputs DEF_NAME<TAB>DESCRIPTION per definition found at column 0.
# Description = first sentence of contiguous // comment lines immediately above
# the definition. Empty if no doc comment is present.

extract_definitions() {
    local file="$1"
    awk '
    BEGIN { comment = ""; has_comment = 0 }

    # Accumulate contiguous comment lines immediately above definitions
    /^\/\// {
        line = $0
        sub(/^\/\/ ?/, "", line)
        if (has_comment && length(line) > 0) {
            comment = comment " " line
        } else if (!has_comment) {
            comment = line
        }
        has_comment = 1
        next
    }

    # Top-level definition: must start at column 0, #UpperCase:
    /^#[A-Z][a-zA-Z0-9]*:/ {
        colon = index($0, ":")
        def = substr($0, 1, colon - 1)
        desc = comment
        # Take only the first sentence (up to first period)
        dot = index(desc, ".")
        if (dot > 0) desc = substr(desc, 1, dot - 1)
        gsub(/^[[:space:]]+|[[:space:]]+$/, "", desc)
        print def "\t" desc
        comment = ""; has_comment = 0
        next
    }

    # Any other line (blank, code, etc.) resets comment accumulation
    { comment = ""; has_comment = 0 }
    ' "$file"
}

# ── Collect all definitions across the module into one stream ─────────────────
# Output format per row (tab-separated):
#   TOP_DIR  SUB_DIR  FILE_REL  DEF_NAME  DESCRIPTION
#
# TOP_DIR  = immediate subdirectory of the module root
# SUB_DIR  = remaining path components under TOP_DIR (empty if file is in TOP_DIR)
# FILE_REL = path relative to MODULE_DIR

collect_all_definitions() {
    local module_dir="$1"
    local cue_files=()
    mapfile -t cue_files < <(
        find "$module_dir" -name "*.cue" -not -path "*/cue.mod/*" | sort
    )

    for abs_file in "${cue_files[@]}"; do
        local file_rel="${abs_file#${module_dir}/}"
        local dir_rel
        dir_rel="$(dirname "$file_rel")"
        [[ "$dir_rel" == "." ]] && dir_rel=""

        local top_dir sub_dir
        if [[ -z "$dir_rel" ]]; then
            top_dir=""; sub_dir=""
        else
            top_dir="${dir_rel%%/*}"
            if [[ "$dir_rel" == "$top_dir" ]]; then
                sub_dir=""
            else
                sub_dir="${dir_rel#${top_dir}/}"
            fi
        fi

        while IFS=$'\t' read -r def desc; do
            [[ -n "$def" ]] || continue
            printf '%s\t%s\t%s\t%s\t%s\n' \
                "$top_dir" "$sub_dir" "$file_rel" "$def" "$desc"
        done < <(extract_definitions "$abs_file")
    done
}

# ── Capitalize first letter of a string ───────────────────────────────────────

capitalize_first() {
    echo "$1" | awk '{print toupper(substr($0,1,1)) substr($0,2)}'
}

# ── Print a markdown definition table for a given top_dir + sub_dir ──────────
# Prints nothing (not even the header row) when no matching rows exist.

print_definition_table() {
    local data_file="$1" top_dir="$2" sub_dir="$3"
    local rows
    rows=$(
        awk -F'\t' -v td="$top_dir" -v sd="$sub_dir" \
            '$1 == td && $2 == sd' "$data_file" \
        | sort -t$'\t' -k3,3 -k4,4
    )
    [[ -z "$rows" ]] && return
    echo '| Definition | File | Description |'
    echo '|---|---|---|'
    printf '%s\n' "$rows" | awk -F'\t' '{
        printf "| `%s` | `%s` | %s |\n", $4, $3, $5
    }'
    echo ""
}

# ── Main ──────────────────────────────────────────────────────────────────────

TMPFILE=$(mktemp)
trap 'rm -f "$TMPFILE"' EXIT

collect_all_definitions "$MODULE_DIR" > "$TMPFILE"

# ── Header ────────────────────────────────────────────────────────────────────

echo "# ${MODULE_VERSION} — Definition Index"
echo ""
echo "CUE module: \`${MODULE_NAME}\`"
echo ""
echo "---"
echo ""

# ── Project Structure ─────────────────────────────────────────────────────────

echo "## Project Structure"
echo ""
echo '```'
print_tree "$MODULE_DIR" ""
echo '```'
echo ""
echo "---"
echo ""

# ── Definition tables grouped by directory ────────────────────────────────────

[[ ! -s "$TMPFILE" ]] && exit 0

mapfile -t top_dirs < <(awk -F'\t' '{print $1}' "$TMPFILE" | sort -u)

for top_dir in "${top_dirs[@]}"; do
    section_name=$(capitalize_first "$top_dir")
    echo "## ${section_name}"
    echo ""

    # Definitions in files directly inside this top-level dir (no sub_dir)
    print_definition_table "$TMPFILE" "$top_dir" ""

    # Nested subdirectories under this top-level dir
    mapfile -t sub_dirs < <(
        awk -F'\t' -v td="$top_dir" '$1 == td && $2 != "" {print $2}' "$TMPFILE" \
        | sort -u
    )
    for sub_dir in "${sub_dirs[@]}"; do
        echo "### ${sub_dir}"
        echo ""
        print_definition_table "$TMPFILE" "$top_dir" "$sub_dir"
    done

    echo "---"
    echo ""
done
