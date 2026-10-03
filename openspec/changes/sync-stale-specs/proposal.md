## Why

Agents and reviewers load the main specs as the source of truth, and six of them still describe code that is gone. `artifact-types` requires an `APIVersion apiversion.Version` field on every artifact and a binding lookup in `Instance.ConfigSchema()`, but the `opm/apiversion` package and the binding registry no longer exist, and the structs carry a `Source` field the spec does not list. That requirement also contradicts `schema-dispatch` ("Module, Instance, Platform structs do not carry APIVersion"), which matches the code. `platform-artifact` specifies `NewPlatformFromValue(k *kernel.Kernel, v)` with `apiversion.ErrUnknownAPIVersion`, while the real constructor takes the value alone. `config-validation` opens with a Purpose naming `ValidateConfig`, `ValidateConfigPartial` and typed wrappers that its own requirements say do not exist, and two of its scenarios build a `Source` with a `Value` field it no longer has. `instance-synthesis` states a wrong signature for `synth.Instance` and names the removed `LoadInstancePackage`. Two stragglers remain: a `kernel-runtime` scenario titled "SynthesizeInstance godoc points to LoadInstancePackage" whose body says the opposite, and a `single-build-render` scenario that still names `LoadPlatformPackage`.

`openspec validate --specs --strict` passes on all of this, so no gate catches it. The owner decided (checklist task c1, 2026-10-02) to fix it as a spec-only library change in wave 1, before the b4, g4 and d-group changes add deltas on top of these specs.

## What Changes

- **artifact-types:** the uniform shape becomes `{Metadata, Package, Source}` with no `APIVersion`, matching `schema-dispatch`. The "APIVersion Field Stamped at Construction" requirement is removed. The binding-based "Instance Config Schema Accessor" is removed and replaced by a requirement that reads `#config` through `schema.Module` and `schema.Config`. The debug-values scenarios read `schema.DebugValues` instead of a version binding. The Purpose line is rewritten in place.
- **platform-artifact:** "Platform Constructor from cue.Value" is removed and replaced by a requirement for the one-argument constructor as it exists: metadata decoded with `type` hoisted, `Package` unchanged, no `Source`, no version detection, and an error with no partial platform when `metadata` is missing.
- **config-validation:** the Purpose line is rewritten in place to name the one validation primitive, the two `ConfigSchema()` accessors and the two source loaders. Scenarios that build or unify `Source.Value` are rewritten to `Origin` and `Data`, with the kernel compiling each source. The `opm/errors` scenario stops claiming that package holds only `TransformError` and sentinels.
- **instance-synthesis:** the single-build requirement names the shared build step (`loader.LoadDir` with the instance shape gate, the step `AcquireInstanceFromDir` runs), the internal signature `Instance(cueCtx *cue.Context, coreVersion string, in Input)`, and the values file rendered from the input's unified `Values`. The synthesis-input requirement says that no public package exports a second entry point, because `opm/internal/synth` exists and exports `Instance` and `Input`.
- **kernel-runtime:** the godoc requirement is removed and re-added under a new name so that the misnamed scenario can be renamed. The requirement text and the other scenarios are unchanged.
- **single-build-render:** the subscription-shaped refusal scenario drops "(and `LoadPlatformPackage`)".
- **schema-dispatch:** unchanged. It already matches the code, and the conflict is resolved by correcting `artifact-types`.

## Not in this change

- Any Go code, godoc, README or AGENTS text. Front-door doc fixes are checklist task c2, and godoc contract placement is c4.
- Stale text in other specs or in other parts of these specs that the c1 review did not name. Anything found while writing this change is listed in the PR body as follow-up, not fixed here.

## Classification

**No release.** Specs only. No exported symbol, signature or behaviour changes. Both section commits are `docs(openspec)`, a type release-please hides. Complexity (Principle VII): none added.

## Downstream consumers

None. The cli and opm-operator read no library spec.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `artifact-types`: uniform shape without `APIVersion`; config schema accessor reads schema paths; debug values read `schema.DebugValues`.
- `platform-artifact`: the bare-value constructor takes one argument and detects no version.
- `config-validation`: `Source` is `{Origin, Data}` in every scenario; the `opm/errors` scenario describes the package as it is.
- `instance-synthesis`: the real internal signature and build step; no second public entry point.
- `kernel-runtime`: the godoc scenario is renamed to what it asserts.
- `single-build-render`: the subscription-shaped refusal names only `AcquirePlatformFromDir`.

## Impact

- `openspec/specs/{artifact-types,platform-artifact,config-validation,instance-synthesis,kernel-runtime,single-build-render}/spec.md` (via the deltas at archive; the two Purpose lines directly).
