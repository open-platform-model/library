# Design Package: Platform Registry Subscription

See [config.yaml](config.yaml) for metadata.

## Summary

Reshapes `#Platform.#registry` from a static map of `#Module` values into a map of registry **subscriptions** — each entry names a CUE module path, an enable flag, and a version filter. The kernel resolves each subscription against the OCI registry, pulls the selected build set, and materializes `#composedTransformers` containing every transformer from every selected version side-by-side. Primitive FQNs gain SemVer suffixes (was MAJOR-only), and `#Module.#defines` is retired — catalog modules are plain CUE packages that export primitives at top level, with `metadata.version` and `metadata.modulePath` sourced from a `Catalog` constant stamped into the package at publish time. Match-time pairing adds a unification step (`unify(MR_primitive, transformer.requiredResources[FQN])`) to catch schema drift behind a matching FQN.

<!--
When implementation lands (status → implemented, or implementation.status → partial+),
add an Implementation Status quote block here. Format:

  > **Implementation status (YYYY-MM-DD).** {One-paragraph summary of what shipped,
  > with file paths to landed code. If there are deliberate deviations from the
  > original design, point readers to the `## Deviations from Design` section below.}

The date in the block MUST match `config.yaml.implementation.date`.
`task enhancements:check` enforces presence for `status: implemented`.
-->

## Documents

1. [01-problem.md](01-problem.md) — Why the static `#registry: [Id]: #ModuleRegistration` shape from 003 can't subscribe to multiple primitive versions and forces MAJOR-only FQNs
2. [02-design.md](02-design.md) — Path-based `#registry`, kernel-side `Materialize` step over OCI, SemVer FQNs, removal of `#defines`, publish-time `Catalog` stamping, always-unify match
3. [03-decisions.md](03-decisions.md) — All design decisions with rationale and alternatives considered

<!-- Add optional and enhancement-specific documents below, numbered sequentially -->

## Applicability Checklist

Check each box that applies to this enhancement. When checked, create the corresponding numbered file and add it to the Documents list above.

- [ ] `NN-schema.md` — New or modified CUE definitions (planned; tracks core/ surface changes)
- [ ] `NN-pipeline-changes.md` — Go pipeline modifications (planned; `Materialize`, OCI pull, matcher rewrite)
- [ ] `NN-module-integration.md` — Impact on catalog authors (planned; `Catalog` constant, publish task)
- [ ] `NN-notes.md` — Deferred items and open questions
- [ ] `experiments/` — Self-contained proofs-of-concept validating ideas in this enhancement (see Experiments below)

Replace `NN` with the next available number in the sequence (starting from `04`).

## Scope

Concrete boundary of this enhancement. `task enhancements:check` requires this section starting at `status: accepted`. For design-time aspirations (what the solution must achieve), see `02-design.md` `## Design Goals`.

### In scope

- New `#Platform.#registry` shape: `[Id]: { path, enable, filter }`. `filter` carries a SemVer range plus optional `allow` / `deny` overrides.
- Removal of `#Module.#defines` and of the `#knownResources` / `#knownTraits` computed views. Catalog modules become plain CUE packages that export `#Resource`, `#Trait`, `#Blueprint`, `#ComponentTransformer` definitions at top level.
- `#FQNType` regex change: SemVer suffix (`@1.2.3`, `@1.2.3-rc.1`) replaces MAJOR-only (`@v1`). `metadata.version` on `#Resource` / `#Trait` / `#Blueprint` / `#ComponentTransformer` becomes `#VersionType` (SemVer) instead of `#MajorVersionType`.
- Catalog-side `Catalog` constant convention (`Catalog: { Version, ModulePath }`) declared in each catalog's root package, with `string | *"0.0.0-dev"` default for source-tree work and a publish task that overwrites with the concrete SemVer in a temp build dir before `cue mod publish`.
- Kernel `Materialize(Platform) → MaterializedPlatform` step: resolves filter, pulls every selected build via `cuelang.org/go/mod` (CUE_REGISTRY-driven; `Kernel.Registry` field defaults to GHCR), loads each package, indexes top-level `#ComponentTransformer` values by stamped FQN into a synthetic `#composedTransformers` map plus `#matchers.{resources,traits}` reverse index.
- Match algorithm rewrite: FQN-keyed lookup (now SemVer) followed by always-on `unify(MR_primitive, transformer.requiredResources[FQN])` before predicate evaluation; hard-fail on missing FQN unless the consumer primitive is optional, with one structured error per missing FQN.

### Out of scope

- The api binding / apiVersion machinery — a separate change is removing it from the library; nothing in this enhancement depends on `apiversion.Detect` / `api.Binding`.
- `#ctx` / `#PlatformContext` (owned by [004](../004-module-context/)).
- `#Claim` / `#ModuleTransformer` / module extension surface (owned by [005](../005-claims/)) and platform capabilities (owned by [006](../006-platform-capabilities/)). Their constructs continue to project through `#registry` once they exist.
- Renderer / `#transform` execution — unchanged from 001/003.
- Self-service catalog discovery UX (`opm catalog list`, web UI).
- Authentication / signing for catalogs in OCI — uses whatever `CUE_REGISTRY` is configured with.
- Migration of any third-party catalog modules; only the OPM core catalog at `library/modules/opm/` is in scope.

## Experiments

Experiments are **optional** and usually appear **after the enhancement has been in design for a while** — once specific claims emerge that benefit from a runnable proof. Do not create `experiments/` upfront when copying this template; add it the first time you actually need to validate an idea. If the enhancement reaches implementation without ever needing one, that is fine.

When an idea does need to be tested or showcased before adoption, place proofs-of-concept under `experiments/` inside this enhancement directory. Experiments live with the enhancement so reviewers can find them next to the design that motivated them.

### Rules

- **One concept per experiment.** Each experiment proves a single idea. If two ideas are entangled, split them into two experiments.
- **Self-contained.** An experiment must run without modifying anything outside its own directory. No edits to `opm/`, `catalog/`, runtime packages, or any sibling experiment.
- **Copy, never reference, source-of-truth artifacts.** CUE schemas, traits, transformers, Go fixtures — copy them into the experiment directory and modify the copies. Never import from or mutate the originals.
- **Disposable.** Experiments are not production code. They may be deleted once the enhancement is implemented or rejected. Do not build infrastructure that other code depends on.
- **Languages.** Go is preferred for runtime/pipeline experiments; CUE for schema experiments; shell scripts or other languages where they fit.

### Layout

When you add the first experiment, create `experiments/README.md` as an index, then add per-experiment subdirectories alongside it.

```
experiments/
├── README.md                  # Index of experiments + how to run them
├── 01-{concept-name}/
│   ├── README.md              # What this proves, how to run, expected outcome
│   ├── ...                    # Copied schemas, Go code, fixtures, etc.
│   └── ...
└── 02-{concept-name}/
    └── ...
```

The `experiments/README.md` is a thin index — list each experiment, its hypothesis, and its current status (Draft / Running / Concluded). Per-experiment READMEs carry the detail.

### Per-experiment README

Each experiment's README must answer:

1. **Hypothesis** — What claim from the design is this validating?
2. **Setup** — What was copied in, from where, and what was modified.
3. **Run** — Exact commands to reproduce the result.
4. **Outcome** — What was observed; whether the hypothesis held.

Update the per-experiment README in place as the experiment evolves. Once concluded, record the outcome and link the result back into `02-design.md` or `03-decisions.md` so the enhancement carries the evidence.

## Deviations from Design

None at this stage. Update this section when implementation lands and any deliberate divergences from the design need to be documented. `task enhancements:check` enforces presence for `status: implemented` (the section may say "None"; it just has to exist).

## Cross-References

| Document | Purpose |
| -------- | ------- |
| `CONSTITUTION.md` (repo root) | Core design principles governing all changes in this repository |
| [`003-platform-construct/`](../003-platform-construct/) | Superseded by this enhancement. 003's `#Platform`, `#ModuleRegistration`, `#composedTransformers`, and `#matchers` shapes are the starting point this design refactors. |
| [`001-kernel-redesign-around-platform/`](../001-kernel-redesign-around-platform/) | Defines the kernel surface that grows the new `Materialize` step. |
| [`004-module-context/`](../004-module-context/), [`005-claims/`](../005-claims/), [`006-platform-capabilities/`](../006-platform-capabilities/) | Sibling enhancements that project through `#registry`; this redesign preserves their integration points (their primitives, once shipped, are surfaced via the same path-based subscription model). |
| `../../core/platform.cue` | Target of the `#Platform` / `#registry` rewrite. |
| `../../core/transformer.cue` | Target of the `#ComponentTransformer` SemVer-FQN change. |
| `../../core/module.cue` | Target of the `#Module.#defines` removal. |
| `../../core/types.cue` | Target of the `#FQNType` regex change. |
| `../../opm/compile/match.go` | Target of the matcher rewrite (FQN+unify+predicate). |
| `../../opm/platform/platform.go`, `../../opm/kernel/kernel.go` | Target of the new `Materialize` step and `Kernel.Registry` field. |

<!--
## Agent Instructions

This directory is a template for new OPM enhancements. To create a new enhancement:

1. Copy this entire `000-TEMPLATE/` directory to `NNN-kebab-case-title/` using the next available number (three-digit, zero-padded).
2. Fill in all `{placeholder}` values.
3. The three mandatory files (01-problem.md, 02-design.md, 03-decisions.md) must always be present and populated.
4. Review the Applicability Checklist. For each checked item, create the corresponding file numbered sequentially from 04 onward and add it to the Documents list.
5. All files must use the `NN-name.md` numbering convention.
6. Keep the Documents list in README.md in sync with the actual files in the directory.
7. Update the Applicability Checklist as the enhancement evolves — check boxes when files are added, uncheck if removed.
8. Do not create `experiments/` upfront. Add it only when a specific claim in the design needs a runnable proof (often partway through the enhancement's life). Check the `experiments/` box in the Applicability Checklist at that point.

### Status Lifecycle

- **Draft** — initial design, actively being written
- **Accepted** — design agreed upon, ready for implementation
- **Implemented** — design has been realized in code
- **Superseded by NNN** — replaced by a newer enhancement
-->
