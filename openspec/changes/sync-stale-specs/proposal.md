## Why

Agents and reviewers load the main specs as the source of truth, and six of them still describe code that is gone. `artifact-types` requires an `APIVersion apiversion.Version` field on every artifact and a binding lookup in `Instance.ConfigSchema()`, but the `opm/apiversion` package and the binding registry no longer exist, and the structs carry a `Source` field the spec does not list. That requirement also contradicts `schema-dispatch` ("Module, Instance, Platform structs do not carry APIVersion"), which matches the code. `platform-artifact` specifies `NewPlatformFromValue(k *kernel.Kernel, v)` with `apiversion.ErrUnknownAPIVersion`, while the real constructor takes the value alone. `config-validation` opens with a Purpose naming `ValidateConfig`, `ValidateConfigPartial` and typed wrappers that its own requirements say do not exist, and two of its scenarios build a `Source` with a `Value` field it no longer has. `instance-synthesis` states a wrong signature for `synth.Instance` and names the removed `LoadInstancePackage`. Two stragglers remain: a `kernel-runtime` scenario titled "SynthesizeInstance godoc points to LoadInstancePackage" whose body says the opposite, and a `single-build-render` scenario that still names `LoadPlatformPackage`.

`openspec validate --specs --strict` passes on all of this, so no gate catches it. The owner decided on 2026-10-02 to fix it as a spec-only library change that lands first, before the later contract changes add their deltas on top of these specs.

## What Changes

- **artifact-types:** the uniform shape becomes `{Metadata, Package, Source}` with no `APIVersion`, matching `schema-dispatch`. The "APIVersion Field Stamped at Construction" requirement is removed. The binding-based "Instance Config Schema Accessor" is removed; the existing "Instance exposes its components and config schema", which already reads `#config` through `schema.Module` and `schema.Config`, takes over its zero-value cases (nil receiver, missing `#module`, missing `#config`). A catalog is produced by its two acquire verbs or `catalog.NewCatalogFromValue`. The debug-values scenarios read `schema.DebugValues` instead of a version binding. The Purpose line is rewritten in place.
- **platform-artifact:** "Platform Constructor from cue.Value" is removed and replaced by a requirement for the one-argument constructor as it exists: metadata decoded with `type` hoisted, `Package` unchanged, no `Source`, no version detection, and an error with no partial platform when `metadata` is missing.
- **config-validation:** the Purpose line is rewritten in place to name the one validation primitive, the two `ConfigSchema()` accessors and the two source loaders. Scenarios that build or unify `Source.Value` are rewritten to `Origin` and `Data`, with the kernel compiling each source. The `opm/errors` scenario stops claiming that package holds only `TransformError` and sentinels.
- **instance-synthesis:** the single-build requirement names the shared build step (`loader.LoadDir` with the instance shape gate, the step `AcquireInstanceFromDir` runs), the internal signature `Instance(cueCtx *cue.Context, coreVersion string, in Input)`, and the values file rendered from the input's unified `Values`. The synthesis-input requirement says that no public package exports a second entry point, because `opm/internal/synth` exists and exports `Instance` and `Input`.
- **kernel-runtime:** the godoc requirement is removed and re-added under a new name so that the misnamed scenario can be renamed. Its sentence on the imported core release is corrected: the synthesized package imports `core` at the major of the kernel's schema release, and the release that import resolves to comes from the module's own `cue.mod/module.cue`. The other scenarios are unchanged.
- **single-build-render:** the subscription-shaped refusal scenario drops "(and `LoadPlatformPackage`)".
- **schema-dispatch:** unchanged. It already matches the code, and the conflict is resolved by correcting `artifact-types`.

## Not in this change

- Any Go code, godoc, README or AGENTS text. Those belong to the front-door doc fix and the godoc contract pass.
- Stale text outside the six named specs, or in requirements of these specs that this change does not touch, goes to the PR body as follow-up.

Known follow-ups, for the PR body:

- `README.md:19` still says the kernel accepts only `Module`, `ModuleInstance` and `Platform`, so "The enumerated set is stated once and agrees everywhere" fails until the front-door doc fix lands.
- `opm/kernel/doc.go` says the core release the synthesized package imports is the kernel's pinned schema release; the corrected "SynthesizeInstance is documented as the typed-input entry point" needs it to name the major only and the module's own `cue.mod/module.cue` (the `Kernel.SynthesizeInstance` godoc already does).
- `kernel-runtime` "Kernel.SynthesizeInstance method" (untouched here) carries the same core-release claim.
- `artifact-types` "Constructor Helpers from cue.Value" keeps a scenario titled "NewInstanceFromValue success path" whose body repeats "No instance constructor". Renaming it takes a REMOVED + ADDED under a new requirement name, which would orphan the citation in `opm/module/module_test.go`; it waits for a change that may edit Go comments.
- `schema-dispatch` Purpose line, flagged during planning.
- No test covers `Instance.ConfigSchema()` on an instance with no `#module` (`opm/module/instance_test.go` covers the reachable, missing-`#config` and nil cases).

## Classification

**No release.** Specs only. No exported symbol, signature or behaviour changes. Both section commits are `docs(openspec)`, a type release-please hides. Complexity (Principle VII): none added.

## Downstream consumers

None. The cli and opm-operator read no library spec.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `artifact-types`: uniform shape without `APIVersion`; the instance accessor requirement carries the config schema's zero-value cases; debug values read `schema.DebugValues`; catalogs also come from `catalog.NewCatalogFromValue`.
- `platform-artifact`: the bare-value constructor takes one argument and detects no version.
- `config-validation`: `Source` is `{Origin, Data}` in every scenario; the `opm/errors` scenario describes the package as it is.
- `instance-synthesis`: the real internal signature and build step; no second public entry point.
- `kernel-runtime`: the godoc scenario is renamed to what it asserts, and the imported core release is stated as the code has it.
- `single-build-render`: the subscription-shaped refusal names only `AcquirePlatformFromDir`.

## Impact

- `openspec/specs/{artifact-types,platform-artifact,config-validation,instance-synthesis,kernel-runtime,single-build-render}/spec.md` (via the deltas at archive; the two Purpose lines directly).
