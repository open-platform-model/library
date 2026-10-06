## Why

The Kubernetes controller `opm-operator` is renamed `opm-controller`: the repository, the Go module,
the image, and the word for the product (0032:D1, the owner's decision of 2026-10-05).
"Operator" in the Kubernetes sense means a controller carrying application-specific day-2
knowledge, and OPM's is a generic delivery engine. The library names its consumer in a CI workflow,
a script, its godoc, its lint messages, its docs and its specs. After the GitHub rename those names
point at a redirect, and the relative links into the sibling checkout break when the local
directory moves.

## What Changes

- CI: the consumer-build matrix value, step name, `if:` and checkout `repository:` name
  `opm-controller`, so the non-required check becomes `Consumer build (opm-controller)`.
  `.tasks/consumer-build.sh` comments follow. `.tasks/cascade/wiring-check.sh` is a byte-identical
  copy of the `.github` canonical file and changes only with the `.github` pin round.
- Godoc and comments under `opm/` name the controller where they named the operator, and the
  provenance notes name `opm-controller` at the same commits. `.golangci.yml` depguard messages,
  which a contributor reads, say "the controller's engine". Test assertion messages and one
  fixture comment follow; the hold-message test bans the word `controller` instead of `operator`.
- Prose: `README.md`, `AGENTS.md`, `CONSTITUTION.md`, `openspec/config.yaml`, `MIGRATIONS.md`,
  `migrations/README.md`, ADR bodies (dated material is renamed too), `docs/design/*` (links
  re-pointed at `../../../opm-controller/...`), `docs/site` diagnostics (the `Check against:`
  pointers and the "Controller conditions entry" name), the embedding page, the repo's two skills,
  and the active change `publish-go-api-bundle`.
- Specs: MODIFIED wording in `consumer-build`, `duplicate-object-identities` and
  `kubernetes-tier`. In `kernel-runtime` the requirement "Kernel.SynthesizeInstance method" is
  REMOVED and re-ADDED as "Kernel.SynthesizeInstance builds an instance from an acquired module",
  because its scenario "Operator-shaped caller needs no cue.Context" becomes "Controller-shaped
  caller ..." and OpenSpec refuses a MODIFIED that drops a scenario. The Purpose lines of
  `consumer-build`, `kernel-runtime` and `registry-module-loading` are edited by hand (a delta
  Purpose is ignored for an existing spec).

Unchanged on purpose: the human role ("operators in air-gapped environments", "operator
persona"), `CHANGELOG.md`, `openspec/changes/archive/**`, and ADR-001's "operators" (people).

Not in this change: the library release that carries it, and `.tasks/cascade/wiring-check.sh`
(the `.github` pin round replaces it).

## Impact

- Affected packages: none in behaviour. `opm/` changes are comments and test messages only.
- SemVer: PATCH-level by itself (docs, ci, test). It ships on the branch of
  `remove-ownership-admit`, whose break sets the release class.
- Downstream: none. The consumer-build check clones `open-platform-model/opm-controller`, which
  exists only after the GitHub rename; until then that (non-required) job fails to clone.
