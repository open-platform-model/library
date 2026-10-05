## Why

The operator learns which contracts an instance depends on by walking the instance itself.
`opm-operator/internal/render/demand.go` reads `#resources` and `#traits` off every component of
`inst.Package` and stores the sorted keys on `ModuleInstance.status.requiredContracts`. The
removal guard, the shrink refusal and the registration watch filter all trust that list. The walk
copies kernel knowledge into a frontend, and it fails open: a missing `components` or `#resources`
reads as no demand. It also cannot see a component that exists only inside the render. 0013:D24
says the render result lists every contract the render required, synthesised components included,
and frontends read demand from there. The render build already holds the full component set
(`_components`, `opm/internal/renderstage/render.cue.tmpl:48`), so the kernel is the one place
that can report it.

The cli reads three instance and module sub-values raw because the artifacts expose no accessor
for them: the instance's embedded module metadata (`schema.Module`), its merged values
(`schema.Values`) and the module's `debugValues` (`schema.DebugValues`).

The owner decided both in the kernel checklist walkthrough:

- i3: "Kernel export only (no operator stop-gap). Library exports contract demand as typed data in
  0013:D24 shape, fails closed itself; operator drops demand.go walk. ... fail-open stays until
  operator migrates."
- d2: "Additive accessors now (Instance module metadata, merged values, Module debugValues),
  bundled with the i3 demand export in one library change ... Package stays public."

There is no `Module.InitValues()` accessor, because d2 names `debugValues` only. Two spec and
docs fixes from the beta.1 kernel plan's gate review are folded into this change: the duplicate artifact-types scenario
"NewInstanceFromValue success path" and the README's `debugValues` migration line.

## What Changes

- `render.cue.tmpl`: `diagnostics` gains `requiredContracts`, the sorted, deduplicated contract
  keys of every component in the build (`#resources` read unguarded, `#traits` when present). It
  covers components omitted under `SkipUnprovided`. 0013's target schema adds synthesised components
  through the render glue, not through the instance's `components`, so the change that adds them
  must add them to the set this field iterates.
- `opm/kernel`: `RenderDiagnostics.RequiredContracts []string`, decoded under the existing
  concreteness check. It is set on every `RenderResult` and on every `*RenderError`, and it is an
  empty, non-nil slice for an instance with no components. A component whose `#resources` is
  missing or errored makes the render fail with an error. A missing `#traits` is allowed.
- `opm/module`: three additive accessors, each nil-safe:
  - `(*Instance).ModuleMetadata() *ModuleMetadata`: the embedded module's metadata, decoded from
    `Package`. It returns nil when there is no `#module` or it does not decode.
  - `(*Instance).Values() cue.Value`: the instance's merged values at `schema.Values`.
  - `(*Module).DebugValues() cue.Value`: the module's `debugValues`, or the zero value when the
    module declares none.
- Docs: the `opm/module` package doc, the `schema.DebugValues` comment and the README migration
  line (`README.md:32`) point at `Module.DebugValues()`. The `opm/kernel` package doc names the
  demand field.
- Specs: `single-build-render` gains the demand requirement. `artifact-types` gains the three
  accessors, drops the duplicate constructor scenario, and the `debugValues` scenario names the
  accessor.

Not in this change:

- 0013:D24:R1 (an instance using a secret source depends on that source's contract). The
  synthesised secret component exists only in the 0013 target schema, and core has no such
  component yet. The demand field covers whatever `_components` holds, but this change cannot
  prove the secret-source case end to end.
- The operator migration off `demand.go` (op-i3g2) and the cli migration onto the accessors
  (cli-d2-accessors). Until the operator migrates, its walk stays fail-open.
- `Module.InitValues()` (d2 names `debugValues` only).
- `opm/schema` paths for a component's `#resources` and `#traits`. The plan entry listed them, but
  no Go code reads them: the glue reads both in CUE, and the operator drops its walk. The
  `schema-dispatch` spec removes a path with no reader and names `ComponentResources` and
  `ComponentTraits` among the paths that must not exist. design.md D5 records the reasoning.
- The `schema-dispatch` requirement "Path inventory exposed as package-level vars" still lists
  the readers of `Values`, `Module` and `DebugValues` as before. After this change they also have
  production readers in `Instance.Values()`, `Instance.ModuleMetadata()` and `Module.DebugValues()`,
  so the list is stale. This change does not MODIFY that requirement, because lib-h2 (round 2,
  `CatalogProvides`) changes the same list. The fix belongs to lib-c4, which merges after both.
- Removing or deprecating any existing API. `Package` stays public, and `schema.DebugValues`,
  `schema.Values` and `schema.Module` stay.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `single-build-render`: ADDED "A render reports every contract its instance requires".
- `artifact-types`:
  - "Constructor Helpers from cue.Value" is replaced by "Module and platform constructors from
    cue.Value". The requirement text is the same, and every scenario is kept except the duplicate
    "NewInstanceFromValue success path". Its content is the kept scenario "No instance
    constructor".
  - "Instance exposes its components and config schema" is replaced by "Instance exposes its
    components, config schema, values and module metadata". Every scenario is kept, and new ones
    are added for `Values()` and `ModuleMetadata()`.
  - "Kernel Artifact Type Set" is MODIFIED: the scenario "debugValues accessible via
    Module.Package" names `Module.DebugValues()`.
  - ADDED "Module exposes its debug values".

## Impact

- Packages: `opm/internal/renderstage` (the glue), `opm/kernel` (`render.go`, `render_decode.go`,
  `doc.go`, tests), `opm/module` (`instance.go`, `module.go`, tests), `opm/schema` (one doc
  comment), `testdata/render/scenarios` (new packages), `README.md`.
- Public surface: additive only. There is one new field, `RenderDiagnostics.RequiredContracts`,
  and three new methods. No signature changes and nothing is removed.
- Downstream: neither frontend needs a code change to keep building. The operator (op-i3g2)
  replaces `declaredContracts` with `Diagnostics.RequiredContracts`. The cli (cli-d2-accessors)
  moves its raw reads (`internal/workflow/render/render.go`, `values.go`, `instinit/values.go`)
  onto the accessors.
- Behaviour: a render whose component lacks `#resources` errors. That is unreachable for valid
  input, because core's `#Component` declares `#resources: #ResourceMap`, and the matcher already
  iterates it unguarded. The new comprehension runs on every render, and section 4 measures its
  cost.
- SemVer: MINOR (additive). Release class `feat`.
