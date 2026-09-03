# Per-version build isolation (C2) — future design note

**Status:** Moot (2026-09-03). Neither C1 nor C2 exists any more: `library-render-cutover` deleted
`opm/materialize` (`pullCatalog`, `indexCatalogs`, `MaterializedPlatform.Transformers` / `Matchers`) and the
Go executor. The premise, `Materialize` selecting several versions of one catalog `path@major`, was
made inexpressible by enhancement 0010 D14 (one build per subscription) and 0019 D5 (the platform is
a CUE module importing its catalogs, so minimum version selection admits one version per
`path@major` per build). The trigger below cannot fire: under `Kernel.Render` every transformer is
evaluated inside its own catalog's build by construction, because the whole render is one build.
Kept as the record of the question; see ADR-003's status and ADR-005.
**Related:** ADR-003, `openspec/changes/federate-materialize-transformers/design.md` (D5),
`docs/design/transformer-output-hidden-field-scope-bug.md` §14.

## Context

`Materialize` may select **multiple versions of the same catalog `path@major`** at once (a platform
subscribing to a `range` that admits `0.5.0` and `0.5.1`; Module-A authored against `0.5.0`, Module-B
against `0.5.1`, both live on one platform). CUE Minimal Version Selection admits only one version per
`path@major` per build, so the versions cannot be collapsed into a single CUE build.

The shipped approach is **C1 — one merged native map keyed by version-bearing FQN.** Each selected
version is pulled as its own build (`pullCatalog`), then `indexCatalogs` merges every selected
version's `#transformers` into one open `#composedTransformers` map and one `#matchers` reverse index,
exposed as `MaterializedPlatform.Transformers` / `Matchers`. Distinct version-bearing FQNs
(`…/deployment-transformer@0.5.0` vs `@0.5.1`) are distinct keys, so the merge is collision-free and
the matcher's exact-FQN lookup (demand-side version pinning, design D3) selects the right transformer
without the matcher ever choosing a version.

## Why C1 is sufficient today

- **Per-version dep isolation already holds.** Each version's transitive dependencies are resolved in
  its own `pullCatalog` build *before* the merge; the merge only collects the resulting
  `#ComponentTransformer` values by FQN.
- **Execution fills, it does not re-resolve.** `executePair` only `FillPath`s `#component` (finalized
  data) and `#context.*` into the looked-up `#transform`, then reads `output`. No current transform
  needs to be **re-evaluated against its full originating catalog build** at execution time.

## C2 — per-version build isolation (the future shape)

**C2** keeps each selected version as **its own build instance** for the life of the materialized
platform; the executor routes each matched transform back to *its* native build before rendering,
instead of reading from one merged map. `Transformers` / `Matchers` would grow from "one merged map"
into a `version → build` collection, while preserving the public "look up by FQN, render" contract.

## C1 → C2 trigger

Adopt C2 only when a transformer must **re-evaluate against its full native catalog build at
execution time** — i.e. when rendering a transform needs more of its originating catalog than the
self-contained `#ComponentTransformer` value carries (cross-references into sibling catalog
definitions resolved lazily at render time, version-specific `let`/hidden scaffolding outside the
transformer value, etc.). At that point the merged-map model (C1) is insufficient because the merge
discarded the per-version build context; the matched transform would need its own build to resolve
against. No current flow does this; until one does, C1 stands.
