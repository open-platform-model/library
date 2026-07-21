# cue-regression-canary

## Purpose

Hermetic evaluator-level guard for the CUE closedness regression (spurious `field not allowed` on comprehension-guarded nested struct fields; present in cuelang.org/go v0.17.0+, mitigated by the catalog_opm hoisted-guard authoring rule). Replaces the detection the workaround itself disabled.

## ADDED Requirements

### Requirement: Trigger-form canary pins the known evaluator defect

The test suite SHALL evaluate the minimal trigger-form reproducer (from `docs/design/cue-closedness-regression-alpha2.md`) through the module's own `cuelang.org/go` version and SHALL assert that validation produces a `field not allowed` error at path `x.out.a.b`. When validation instead succeeds, the test SHALL fail with a message naming the running cuelang.org/go version and directing the reader to re-run the regression matrix and deliberately re-evaluate the hoisted-guard authoring rule.

#### Scenario: Defect present (current pinned CUE)

- **WHEN** the trigger form is compiled and validated under a cuelang.org/go version carrying the defect
- **THEN** validation SHALL yield `field not allowed` at `x.out.a.b` and the canary passes

#### Scenario: Defect fixed upstream

- **WHEN** the trigger form validates clean under a future cuelang.org/go version
- **THEN** the canary SHALL fail with the re-evaluation instructions and the CUE version in the message

### Requirement: Hoisted-guard form must remain clean

The test suite SHALL evaluate the hoisted-guard variant of the reproducer (the shipped workaround pattern) and SHALL assert it validates without error under the module's `cuelang.org/go` version.

#### Scenario: Workaround valid on the current evaluator

- **WHEN** the hoisted form is compiled and validated
- **THEN** validation SHALL succeed; a failure blocks any CUE version bump

### Requirement: Canary is hermetic

Both canary tests SHALL run with no network, no registry, no filesystem module loading, and no `cue.mod` — fixtures are embedded strings evaluated via `cuecontext` — and SHALL run under `go test -short`.

#### Scenario: Offline short run

- **WHEN** the package tests run offline with `-short`
- **THEN** both canary tests SHALL execute (not skip)
