## Why

OpenSpec 1.14.1 raises "Requirement text is very long (>500 characters)" from INFO to WARNING. `openspec validate --strict` fails on any warning, and no config key or flag turns it off: the limit is a constant in the validator. The warning is right on the merits. A long requirement usually holds several behaviors, and one behavior per requirement makes each one testable by its own scenarios.

On `main`, 152 requirements in 23 of the 26 main specs are over the limit, so `openspec validate --specs --strict` fails today (153 WARNING lines in `openspec validate --all --strict`; the 153rd sits in the open change `publish-go-api-bundle`, which is not ours). The library cannot turn `--strict` back on as a gate until they are rewritten. This change rewrites them so that `--strict` can pass again with no rule lost or changed, and then adds the gate so the problem cannot return.

## What Changes

The rewrites are split or moves into scenarios. They never change the meaning of a requirement or a scenario. The delta specs are written batch by batch in the apply phase, one batch per tasks.md section; this proposal copies no requirement text. Counts are the long requirements of each spec on `main` (checked with `openspec validate --all --strict --json`).

- **library-B1 (28):** `kubernetes-tier` (25), `api-diff-check` (3).
- **library-B2 (28):** `single-build-render` (19), `schema-dispatch` (9).
- **library-B3 (28):** `kernel-runtime` (17), `artifact-types` (11).
- **library-B4 (28):** `deps-cascade` (8), `instance-synthesis` (7), `cascade-wiring` (6), `config-validation` (6), `workflow-hardening` (1).
- **library-B5 (28):** `fetch-error-classification` (5), `platform-artifact` (5), `platform-module-generation` (4), `registry-module-loading` (4), `render-parity` (4), `catalog-acquisition` (3), `fixture-pin-maintenance` (3).
- **library-B6 (12):** `migration-docs` (3), `release-pipeline` (3), `duplicate-object-identities` (2), `helper-packages` (2), `test-fixture-registry` (2).
- **Gate:** a Taskfile task pinned to OpenSpec 1.14.1 that runs `openspec validate --all --strict --no-interactive`, wired into `task check`, and a CI job that runs the same task. The library has no OpenSpec task or CI job today.

Not touched: the specs `consumer-build`, `cue-regression-canary` and `transform-input-fill` (no long requirement), the open change `publish-go-api-bundle` and its new spec `docs-bundle`, and every Go and CUE file.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

The 23 specs named above change their requirement structure, not their rules. Each is rewritten in the apply phase, by MODIFIED, MODIFIED plus ADDED, or REMOVED plus ADDED deltas, as design.md chooses per requirement. No capability gains or loses a rule.

## Impact

**SemVer: no bump, no release of its own.** No `opm/` identifier, signature or behavior changes, and no Go code is touched. Spec commits are `docs:` and the gate is `ci:`, both hidden from release-please. No downstream consumer (`cli`, `opm-operator`) has anything to do.

**Complexity (Principle VII):** the gate adds one Taskfile task pair and one CI job, and no dependency to the Go module. It pins an npm CLI at an exact version; design.md gives the reasons.

**Risk:** a rewrite that silently drops a rule or a scenario. OpenSpec does not compare scenario text and does not check scenarios under REMOVED plus ADDED. The scenario ledger and the byte-equality check in design.md cover that, and the reviewer repeats them per batch.

**Delivery order:** the gate section is the last commit. A PR that carries it passes only after the open change `publish-go-api-bundle` lands without long-requirement warnings, because `validate --all --strict` covers open changes too.

**Mergeable sections:** the work is six batch sections and one gate section, seven in all. The `tasks` rule of `openspec/config.yaml` allows about seven; the `proposal` rule says about five. The human accepts seven sections at the proposal gate or asks for another cut. The six batches cannot merge without passing the 28-requirement cap of one unit.

## Enhancement

None.
