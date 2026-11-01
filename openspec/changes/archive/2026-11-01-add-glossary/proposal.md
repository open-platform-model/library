## Why

Agents and the owner use loose words for the library's concepts: "provider" for a Transformer, "release" for a Module Instance, "compile" for Render. The repo has no one place that names its domain terms. The skills that design and refactor read a root `GLOSSARY.md` and treat its terms as domain terms, so this repo needs one.

## What Changes

- New root file `GLOSSARY.md` with the domain terms of the library: the Kernel and its tiers, the four artifact kinds, the render vocabulary, the core schema and the Kubernetes tier. Each term has a definition and the loose words to avoid.
- `AGENTS.md` gets one rule line that points to `GLOSSARY.md`. `CONTEXT.md` gets one router row for naming a concept. No other line of either file changes.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. This change has no spec delta, and `.openspec.yaml` sets `skip_specs: true`, as the archived change `add-openspec-strict-gate` does.

## Impact

**SemVer: no release.** No `opm/` code changes, and no downstream consumer is affected. The commits are `docs`, which release-please hides in this repo. No change is breaking.

**Risk:** a definition drifts from the code. Each term traces to a package doc, an ADR or `AGENTS.md`, and a reviewer can check it there.

## Enhancement

None.
