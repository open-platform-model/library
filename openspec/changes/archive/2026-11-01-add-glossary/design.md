## Context

The library has `AGENTS.md`, `CONTEXT.md`, `CONSTITUTION.md`, the package map in `references/` and ADRs. None of them is a list of terms with the words to avoid. The workspace docs-form rule allows `GLOSSARY.md` at a repo root.

## Goals / Non-Goals

**Goals**

- One file where each domain term has one definition and a list of words to avoid.
- Routing to it from `AGENTS.md` and `CONTEXT.md` in the form the docs-form rule gives.

**Non-Goals**

- Any change of meaning of a rule, a package doc or an ADR. The glossary names what they already say.
- General programming terms.

## Research & Decisions

### Which terms

**Context**: A glossary lists only terms specific to the domain.
**Explored**: The `opm/kernel`, `opm/platform`, `opm/errors` and `opm/k8s` package docs, `AGENTS.md`, `references/package-map.md` and the ADRs.
**Decision**: Terms in five clusters: kernel and tiers, artifacts, rendering, schema, Kubernetes tier. A term with no source in the repo is left out.
**Rationale**: Every definition can be checked against a file the reader can open.

### Where the pointers go

**Context**: The docs-form rule makes `AGENTS.md` hold facts and `CONTEXT.md` hold a router with no facts.
**Decision**: One rule line in the repository rules of `AGENTS.md`, and one router row in `CONTEXT.md`.
**Rationale**: Each file keeps its role, and `CONTEXT.md` stays far below 80 lines.
