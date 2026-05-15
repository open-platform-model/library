#!/usr/bin/env bash
# Driver for experiment 01-matcher-mechanics. Runs each fixture and captures
# the outcome. Positive cases must pass `cue vet -c`; negative cases (02, 03)
# must fail. The "Outcome" section of the experiment README pastes the actual
# stdout/stderr.

set -u
cd "$(dirname "$0")"

run_case() {
	local name="$1"
	local expect="$2" # "pass" or "fail"
	echo "=== ${name} (expect ${expect}) ==="
	# `cue vet -c` against the case dir validates everything including imports.
	if cue vet -c "./cases/${name}/..." 2>&1; then
		echo "[result] pass"
	else
		echo "[result] fail"
	fi
	echo
}

run_case "01-required-match"        "pass"
run_case "02-required-missing"      "fail"
run_case "03-required-mismatch"     "fail"
run_case "04-optional-match"        "pass"
run_case "05-optional-missing"      "pass"
run_case "06-platform-inheritance"  "pass"
