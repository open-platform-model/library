#!/usr/bin/env bash
# Driver for experiment 01-names-cascade-and-injection. Every case is
# positive: cases assert via `_assertX: <literal> & <computed>` unification,
# so any cascade regression turns into a `conflicting values` CUE error.

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

run_case "01-default-cascade"       "pass"
run_case "02-resource-name-override" "pass"
run_case "03-cluster-domain-default" "pass"
run_case "04-names-lockstep"         "pass"
