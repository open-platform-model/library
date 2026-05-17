#!/usr/bin/env bash
# Driver for experiment 02-release-flow-ordering. Every case is positive:
# cases 01/02/04 demonstrate the correct working forms; case 03 demonstrates
# the SILENT-FAILURE mode of wrong ordering as a `len(... ) & 0` assertion,
# so it passes by producing the documented (empty) shape.

set -u
cd "$(dirname "$0")"

run_case() {
	local name="$1"
	local expect="$2" # "pass" or "fail"
	echo "=== ${name} (expect ${expect}) ==="
	if cue vet -c "./cases/${name}/..." 2>&1; then
		echo "[result] pass"
	else
		echo "[result] fail"
	fi
	echo
}

run_case "01-static-components"         "pass"
run_case "02-dynamic-from-config"       "pass"
run_case "03-config-after-builder-fails" "pass"
run_case "04-inline-literal-scope"      "pass"
