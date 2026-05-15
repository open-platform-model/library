#!/usr/bin/env bash
# Driver for experiment 02. Validates the four claims (D7 read surface, the
# unbound-release diagnostic, D13 #-prefix export exclusion, D13 FillPath).

set -u
cd "$(dirname "$0")"

div() { printf '\n=== %s ===\n' "$*"; }

div "Claim 1 — read surface concretizes (bound release)"
cue eval ./cases/jellyfin/ -e 'ReleaseBound.components' 2>&1
echo "[exit=$?]"

div "Claim 2 — unbound release diagnostic"
cue vet -c ./cases/jellyfin/release-unbound.cue ./cases/jellyfin/module.cue ./cases/jellyfin/platform-prod.cue 2>&1
echo "[exit=$?]"

div "Claim 3 — #-prefix exclusion (cue export)"
out=$(cue export ./cases/jellyfin/ -e 'ReleaseBound' --out yaml 2>&1)
echo "$out"
if echo "$out" | grep -E '^\s*#?(platform|module|consumes):' >/dev/null; then
	echo "[result] LEAK — definition field appeared in export"
else
	echo "[result] OK — no definition fields in export"
fi

div "Claim 4 — Go harness FillPath kernel writeback"
( cd cmd/fillpath && go run . 2>&1 )
echo "[exit=$?]"
