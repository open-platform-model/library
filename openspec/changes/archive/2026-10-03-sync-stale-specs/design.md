## Context

Every delta in this change describes code that already exists at `origin/main` (`cf79a5c`). Nothing is implemented. The work is to say in the specs what `opm/module`, `opm/platform`, `opm/kernel` and `opm/internal/synth` already do. The evidence for each correction was checked on 2026-10-03:

| Spec claim | Code at HEAD |
| --- | --- |
| `*catalog.Catalog` comes only from the two catalog acquire verbs | `opm/catalog/catalog.go` also exports `NewCatalogFromValue(v cue.Value)`, which the acquire verbs call |
| The synthesized package imports the kernel's pinned core release | `opm/internal/synth/render.go` writes `core@major(coreVersion)`; the release resolves from the module's own `cue.mod/module.cue` (`Kernel.SynthesizeInstance` godoc) |
| Partial validation runs only under `AcquireInstanceFromDir` | `opm/kernel/synth.go` also calls `validateSources(…, false)` from `SynthesizeInstance` |
| Artifacts carry `APIVersion apiversion.Version` | `module.Module`, `module.Instance`, `platform.Platform` and `catalog.Catalog` export exactly `Metadata`, `Package`, `Source`; no `opm/apiversion` package exists (`TestPlatform_FieldSet` pins the platform field set) |
| `Instance.ConfigSchema()` looks up a binding for `r.APIVersion` | `opm/module/instance.go`: `Package.LookupPath(schema.Module)`, then `LookupPath(schema.Config)`; zero value on nil receiver, missing `#module` or missing `#config` |
| Debug values read through `binding.Paths().DebugValues` | `opm/schema/paths.go` exports `DebugValues`; `opm/module/module.go` package doc says it is read off `Module.Package` via `schema.DebugValues` |
| `NewPlatformFromValue(k *kernel.Kernel, v)`, `apiversion.ErrUnknownAPIVersion` | `opm/platform/platform.go`: `NewPlatformFromValue(v cue.Value)`; error "platform metadata field is required" and nil platform when `metadata` is absent |
| `Source{Value: …}`, `sources[0].Value.Unify(…)` | `opm/kernel/source.go`: `Source{Origin string; Data []byte}`; `validateSources` compiles each `Data` with `cue.Filename(Origin)` in the schema's context, then unifies |
| `opm/errors` holds `TransformError` and sentinels only | `opm/errors` also holds the render verdict rows and gate causes (`UnresolvedDemand`, `OverSubscribedContract`, `ContractCollision`, `PlatformCoreTooOldError`, …); none projects a validation error |
| `Instance(ctx *cue.Context, in InstanceInput)`, built "through the path used by `LoadInstancePackage`" | `opm/internal/synth/instance.go`: `Instance(cueCtx *cue.Context, coreVersion string, in Input) (cue.Value, *module.Source, error)`, built through `loader.LoadDir(…, loader.InstanceSpec)` |
| "no `synth` package exists" | `opm/internal/synth` exists and exports `Instance` and `Input`; it is internal, so no consumer outside the module can import it |

## Goals / Non-Goals

**Goals:** every requirement this change touches matches HEAD, and `artifact-types` no longer contradicts `schema-dispatch`.

**Non-Goals:** a full audit of every library spec; any edit to Go code or godoc; renaming a requirement whose name still fits.

## Research & Decisions

### MODIFIED where every scenario name still fits, REMOVED + ADDED where one does not

**Context**: The owner's decision for this change says "REMOVED+ADDED deltas". OpenSpec 1.12 refuses a MODIFIED requirement that leaves out any scenario name the main spec has, and RENAMED does not get around that. A scenario whose name states something false (for example "Zero value on unregistered binding" or "SynthesizeInstance godoc points to LoadInstancePackage") cannot be kept.
**Explored**: REMOVED + ADDED for every touched requirement, or for only the ones where the 1.12 rule requires it.
**Decision**: Use REMOVED + ADDED under a new requirement name wherever a scenario name is stale or a requirement goes away: "APIVersion Field Stamped at Construction" (removed, with no replacement, because `schema-dispatch` already states the opposite), "Instance Config Schema Accessor" (removed; its zero-value cases move into the existing "Instance exposes its components and config schema", so the accessor is specified once in `artifact-types`), "Platform Constructor from cue.Value", "SynthesizeInstance is documented as the recommended in-memory entry point", and "Module and Instance Typed Convenience Methods" (three scenarios are titled for methods whose absence they assert; re-added as "Callers compose ConfigSchema with the one validation primitive", and no Go comment cites the old name). Use MODIFIED, keeping every scenario name and rewriting only the stale sentences, where all the names still fit: "Uniform Artifact Shape", "Kernel Artifact Type Set", "Instance exposes its components and config schema", "Single Kernel Validation Primitive", "No Custom Validation Error Types", "synth.Instance constructs the instance by single-build CUE evaluation", "Instance synthesis input", "Kernel.SynthesizeInstance method" (only its core-release sentence changes) and "Render is the kernel's sole render path".
**Rationale**: The owner's instruction comes from the 1.12 rule, and REMOVED + ADDED is how this change satisfies it wherever the rule applies. Keeping the names elsewhere preserves the references other specs and test comments make by name. For example, `opm/kernel/kernel_test.go` cites config-validation's "Single Kernel Validation Primitive", and `opm/module/module_test.go` cites artifact-types' "Constructor Helpers from cue.Value". For the same reason "Constructor Helpers from cue.Value" is left as it is, although one scenario name ("NewInstanceFromValue success path") is wrong: `opm/module/module_test.go` cites the requirement by name, OpenSpec refuses REMOVED and ADDED under one name, and this change edits no Go comment. The proposal lists it as a follow-up.

### Purpose lines are edited in place

**Context**: A delta spec carries requirements only. Archiving applies ADDED, MODIFIED and REMOVED blocks and never rewrites a main spec's `## Purpose`.
**Decision**: The stale Purpose lines of `artifact-types` and `config-validation` are rewritten directly in `openspec/specs/*/spec.md` in the section that syncs that spec. The archive, which rides the PR, applies the deltas.
**Rationale**: This is the only way to correct a Purpose line, and both edits ship in the same PR as the requirement deltas.

### schema-dispatch stays as it is

**Context**: The owner's decision names a "schema-dispatch conflict". `schema-dispatch` ("Module, Instance, Platform structs do not carry APIVersion") and `artifact-types` ("Uniform Artifact Shape") state opposite things.
**Decision**: Correct `artifact-types`. `schema-dispatch` matches the code and is not touched.

## Risks / Trade-offs

- [The code moves before this change merges] → each section begins by re-checking its rows of the table above against the worktree's HEAD, and changes the delta, not the code, if a row has moved.
- [Archive applies a delta differently than expected] → section 2 runs the archive in a scratch copy of `openspec/` and validates the resulting main specs before committing.
