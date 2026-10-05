## Context

Line numbers are at `origin/main` `a8bfc76`, after wave 2 round 1 (g4 `66e5b32` among it).

**Demand today.** The operator computes an instance's contract demand in Go
(`opm-operator/internal/render/demand.go:43-94` at operator `origin/main`). It iterates
`inst.Package.LookupPath(schema.Components)`, collects the field names of each component's
`#resources` and `#traits`, sorts them and drops duplicates. Three things in it are fail-open:

- an absent `components` reads as no demand (`:49-52`);
- an absent `#resources` or `#traits` contributes nothing (`contractKeys`, `:80-88`);
- the walk sees only the components the instance package declares.

It is called from `kernel_module_renderer.go` and `kernel_package_renderer.go`, and its list
becomes `Status.RequiredContracts`. The removal guard, the shrink refusal and the registration
watch filter trust that list.

**The render build.** The glue binds `_components: instance.components`
(`opm/internal/renderstage/render.cue.tmpl:48`), and that is the set the matcher iterates
(`:168-171`). It reads `comp.#resources` unguarded and `comp.#traits` behind `!= _|_`. Components
omitted under `SkipUnprovided` stay in `verdicts` and drop out of `pairs` only (`:382`). The kernel
reads `diagnostics` (`:534-557`) through `decodeRenderDiagnostics`
(`opm/kernel/render_decode.go:55-97`), which first forces all of it with
`dv.Validate(cue.Concrete(true))` (`:64`). `RenderDiagnostics` (`opm/kernel/render.go:155-225`)
rides on both `RenderResult.Diagnostics` (`:106-115`) and `RenderError.Diagnostics` (`:274-277`).

**Accessors today.** `*module.Instance` has `Components()` and `ConfigSchema()`
(`opm/module/instance.go:43-69`), and `*module.Module` has `ConfigSchema()`
(`opm/module/module.go:22-35`). The cli reads the rest raw:

- `inst.Package.LookupPath(schema.Module)` → decode to module metadata
  (`cli/internal/workflow/render/render.go`, `decodeModuleMetadata`, whose comment says the
  instance exposes no accessor for this subtree);
- `inst.Package.LookupPath(schema.Values)` → `decodeUnifiedValues`;
- `pkg.LookupPath(schema.DebugValues)` (`internal/workflow/render/values.go`,
  `internal/instinit/values.go`).

## Goals / Non-Goals

**Goals:**

- The render reports, as data on every result and every `*RenderError`, every contract key its
  components require, computed in the build from the same component set the matcher reads.
- The library fails closed on its own: if the demand cannot be computed, the render errors rather
  than reporting less.
- On every served render fixture, the reported list equals what the operator's walk returns.
- `Instance.ModuleMetadata()`, `Instance.Values()` and `Module.DebugValues()` exist, are nil-safe
  and follow the pattern of the existing accessors.
- The extra render cost is measured.

**Non-Goals:**

- Any operator or cli change. Those are op-i3g2 and cli-d2-accessors.
- A richer per-component demand row. 0013:D24 asks for a list, and a sibling field can be added
  later without breaking this one.
- Deprecating any read path. `Package` stays public (owner, d2).

## Decisions

### D1. The demand is computed in the glue over `_components`

`diagnostics` gains one field:

```cue
diagnostics: {
	...
	// Every contract key a component of this render requires: each
	// component's #resources keys and, when it attaches any, its #traits
	// keys, sorted and deduplicated. Every component counts, an omitted one
	// included. #resources is read unguarded, like the matcher reads it, so
	// a component without it fails the build instead of reading as no
	// demand.
	requiredContracts: list.SortStrings([for k, _ in _demand {k}])
}

_demand: {
	for _, comp in _components {
		for fqn, _ in comp.#resources {(fqn): true}
		if comp.#traits != _|_ {
			for fqn, _ in comp.#traits {(fqn): true}
		}
	}
}
```

The struct keys remove duplicates, and `list.SortStrings` sorts in byte order, which is the order
of the operator's `slices.Sort`. The computation is in the build, not in a Go walk of
`Instance.Components()`. 0013:D24 counts components the kernel synthesises, and those exist only
inside the render module. When 0013's synthesised components land, the change that adds them
SHALL make them part of the set this field iterates (today `_components`), and its tests SHALL
extend the parity fixtures.

**Alternatives considered.** A top-level `demand` field beside `diagnostics`: rejected, because it
would need its own concreteness pass and its own decode path. Inside `diagnostics`, it shares the
existing `Validate(cue.Concrete(true))` and fails closed with the rest. A Go walk in the kernel:
rejected for the synthesised-component reason above.

### D2. The field lives on `RenderDiagnostics`

```go
// RequiredContracts is every contract key a component of the instance
// requires: each component's #resources and #traits keys, sorted and
// deduplicated (0013:D24). It counts every component of the render, one
// omitted under RenderInput.SkipUnprovided included, and it is not
// narrowed to provider-fulfilled contracts. Empty, never nil, for an
// instance with no components.
RequiredContracts []string
```

The plan entry named `RenderResult.RequiredContracts` and also asked for it on
`RenderError.Diagnostics`. A single field on `RenderDiagnostics` satisfies both: callers read it as
`res.Diagnostics.RequiredContracts` on success and `rerr.Diagnostics.RequiredContracts` on a
refusal, and there is one value to keep correct instead of two. `RenderDiagnostics` already holds
everything the build reports as data. This reading follows the owner's decision ("exports contract
demand as typed data") over the plan's literal field placement.

`decodeRenderDiagnostics` decodes it with the other glue fields. If the decode yields nil, it
normalises the field to `[]string{}`, so that a status write does not alternate between absent and
empty (the operator's existing guarantee).

**The contract op-i3g2 consumes.** The operator reads `out.Diagnostics.RequiredContracts` from a
`RenderResult`, and `rerr.Diagnostics.RequiredContracts` from a `*RenderError` if it wants demand on
refusals. There is no `RenderResult.RequiredContracts`. The PR body names this field path.

**Where demand is not reported.** A render that fails before evaluation returns a plain error,
not a `*RenderError`: a missing input or Source, the core floor, staging, uncovered paths, skew
under `SkewRefuse`, or a cancelled context. A build error or a diagnostics decode failure also
returns a plain error. None of these carries demand. A caller that tracks demand keeps its last
known list on a plain error. The operator already does this today when the render fails before
`resultFromRender`. The godoc on the field says so.

### D3. Fail-closed boundary

- `#resources` is read with no presence guard. If a component has no `#resources`, or it does not
  evaluate, `diagnostics` is not concrete. The decode then refuses with the existing "a matching
  verdict did not evaluate" error, and the render returns a plain error with no `RenderResult`.
  Core's `#Component` declares `#resources: #ResourceMap`, so this cannot happen for valid input.
  Today the matcher's unguarded `_resFqns` already makes such a render fail. Section 1 pins that
  behaviour before the new field exists, so the field does not loosen it.
- `#traits` is guarded with `!= _|_`, the same guard the matcher (`:171`) and `traitPostures`
  (`:551`) use. An absent `#traits` is allowed (core declares it `#traits?`). The guard cannot tell
  an absent `#traits` from an errored one. Demand therefore sees exactly the trait set that
  matching sees, and this change does not make it stricter than the matcher. Making both strict is
  a separate decision about the matcher.
- An errored `components` struct fails the build, as it does today.

- A `#traits` that exists but is bottom (a conflict) is the one place where the guard could be
  looser than the operator walk. The walk tests `Exists()`, which is true for a bottom value, then
  calls `Fields()`, which errors, so the walk fails closed. The glue's `!= _|_` drops the value.
  The second spike in section 1 (`bad_traits`) pins what `Render` does at the base for a
  component whose `#traits` is a top-level conflict. If the render already refuses, this
  difference is unreachable through a successful render, and the field's godoc says so. If the
  render returns a result, the implementer stops before section 2 and reports to the supervisor,
  because the demand would then be looser than `demand.go`.

The spike in section 1 checks the first point. It renders a test-only instance whose component has
no `#resources`, using a `*module.Instance` struct literal whose `Source` names a plain CUE package
that is not a core `#ModuleInstance`. `Render` reads only `Source` and `Metadata.Name`, so the
acquisition shape gate is not in the way. If that build cannot be staged, the spike records why in
this section, and the fail-closed test moves to a glue-level test on `RenderForTest`'s built value.

### D4. Parity with the operator walk

A test-only Go helper, `walkDeclaredContracts`, copies the operator's `declaredContracts` and
`contractKeys` (opm-operator `internal/render/demand.go` at its `origin/main`, cited by commit in
the helper's comment). It uses `cue.MakePath(cue.Def("resources"))` and `cue.Def("traits")` locally
in the test file. For every instance the `opm/kernel` render tests acquire (every
`testdata/render/scenarios` package, the happy-path instance, and the parity harness instances),
the test asserts that `walkDeclaredContracts(inst)` equals `Diagnostics.RequiredContracts`. It
reads the field from the `RenderResult` or from the `*RenderError`, whichever the render returned.
Scenarios that fail before the diagnostics decode are excluded by name (`unstated`). The exclusion
list is asserted, so a new scenario has to be classified.

### D5. No `opm/schema` paths for `#resources` and `#traits`

The plan entry said to add them, following a comment in the operator (`demand.go:17-24`) that
notes they are missing. This change leaves them out:

- After the change, no Go code in the library reads them, and once op-i3g2 lands the operator does
  not either. The parity helper is test code and builds its paths locally.
- The `schema-dispatch` requirement "Path inventory exposed as package-level vars" says that a path
  with no reader is removed, not kept for a possible consumer. Its scenario "Matcher and
  transformer paths are gone" names `ComponentResources` and `ComponentTraits` as identifiers that
  must not exist.

This is the reading that matches the owner's decision, which is that the operator drops the walk.

### D6. `Instance.ModuleMetadata()` decodes on call

`ModuleMetadata()` is all or nothing: on any decode failure it returns nil, with no partial fill.
The cli's local `decodeModuleMetadata` is best-effort and keeps the fields that decoded, so
cli-d2-accessors maps nil to the zero metadata, as its missing-`#module` path does today.

```go
// ModuleMetadata returns the metadata of the module this instance was
// built from, decoded from the embedded #module on Package. It returns nil
// for a nil receiver, an instance with no #module, or metadata that does
// not decode. Each call decodes afresh, and the result is a cache like
// Metadata: Package wins when they disagree.
func (r *Instance) ModuleMetadata() *ModuleMetadata
```

The plan said "decoded once in processInstance". That would need state the kernel package can set
on a `module.Instance`. The options are an exported field, which the owner's "accessors" and the
"Uniform Artifact Shape" three-field rule both rule out, or a new exported constructor or setter in
`opm/module`, which is new API with no other use. It would also leave a struct-literal `Instance`
(as frontends' tests build them, and as `opm/kernel/validate_test.go:187` does) returning nil while
its `Package` carries the metadata, which contradicts "Package is the source of truth". Decoding on
call costs one lookup and one decode of a small struct. It reuses `opm/module`'s existing
unexported `decodeModuleMetadata` (`module.go:94`) on `Package.LookupPath(schema.Module)`. The
"Metadata decoders are free functions" requirement still holds, because the decoder lives in the
package of its callers.

Acquisition is unchanged: `processInstance` does not start refusing instances whose embedded
module metadata does not decode.

### D7. `Instance.Values()` and `Module.DebugValues()`

```go
// Values returns the instance's merged values at schema.Values, as
// evaluated. The zero value for a nil receiver or an instance with no
// values field.
func (r *Instance) Values() cue.Value

// DebugValues returns the module's author-supplied debugValues at
// schema.DebugValues. The zero value for a nil receiver or a module that
// declares none; callers test Exists(). Whether a frontend layers them into
// its values stack is its own policy.
func (m *Module) DebugValues() cue.Value
```

Both mirror `Components()` and `ConfigSchema()`. They return `cue.Value` on the artifact, as the
existing accessors do. 0021:D8:R11 (no `cue.Value` in public output) is about render output, and
these accessors are not on `RenderResult`. There is no `Module.InitValues()` (SD9).

### D8. Spec and docs folds (SD14)

- In artifact-types, the scenario "NewInstanceFromValue success path" repeats the scenario "No
  instance constructor": the same WHEN, and a THEN that differs only in wording. Removing a scenario under MODIFIED is refused by
  OpenSpec 1.12. So the requirement "Constructor Helpers from cue.Value" is REMOVED and re-ADDED as
  "Module and platform constructors from cue.Value", with the same text and every other scenario.
- "Instance exposes its components and config schema" says that the instance "SHALL expose ... no
  accessor over the module-metadata projection". `ModuleMetadata()` reverses that clause, and the
  name no longer covers the requirement. It is REMOVED and re-ADDED as "Instance exposes its
  components, config schema, values and module metadata", with all six scenarios kept and new ones
  added.
- "Kernel Artifact Type Set" is MODIFIED. Its scenario keeps the name "debugValues accessible via
  Module.Package", and its THEN now names `Module.DebugValues()`, which reads `Package`.
- `README.md:32` (the line was at `:22` when the plan was written) says "read
  `mod.Package.LookupPath(schema.DebugValues)`". It changes to "read `mod.DebugValues()`". The
  `opm/module` package doc and the `schema.DebugValues` comment change the same way.

## Risks / Trade-offs

- **Render cost.** The comprehension iterates each component's demand maps once more on every
  render. The matcher already forces the same maps, so the extra cost should be small. Section 4
  measures it with the memprobe setup that g4 used (cert_manager `r1-nil`, `RUNS=5`). If
  `render_peak_heap` or `user_s` rises by more than 5% against the base, the implementer stops and
  reports to the supervisor. `render_peak_heap` is the hard stop. `user_s` includes about 2 s of
  fixed startup and moves with host load, so a `user_s` rise over 5% is re-measured with base and
  head run back to back before it is reported.
- **Errored `#traits` is not fail-closed** (design D3). It matches the matcher, and it is stated in the
  field's godoc.
- **Merge order.** lib-d1d3 also edits `render.go`, in disjoint hunks. lib-h4 rebases onto this
  change (SD14). Whichever of this change and lib-d1d3 merges second takes the merge.
- **Demand absent on plain errors** (design D2). This is unchanged for the operator, which already keeps
  its last status on those paths. op-i3g2 must keep doing that.

## Research & Decisions

### Where the demand field lives

**Context**: The plan entry named `RenderResult.RequiredContracts` and also asked for demand on
`RenderError.Diagnostics`.

**Explored**: `RenderDiagnostics` rides on both `RenderResult.Diagnostics` and
`RenderError.Diagnostics` (`opm/kernel/render.go`), and is decoded once by
`decodeRenderDiagnostics` under one concreteness check.

**Decision**: One field, `RenderDiagnostics.RequiredContracts` (D2). op-i3g2 reads
`Diagnostics.RequiredContracts`.

**Rationale**: One value to keep correct instead of two, and it meets 0013:D24 on both success and
refusal. It follows the owner's "exports contract demand as typed data" over the plan's literal
placement.

### Whether to add `opm/schema` paths for `#resources` and `#traits`

**Context**: The operator's `demand.go:17-24` notes that the paths are missing, and the plan entry
listed them.

**Explored**: The `schema-dispatch` requirement "Path inventory exposed as package-level vars" and
its scenario "Matcher and transformer paths are gone", which names `ComponentResources` and
`ComponentTraits` as identifiers that must not exist; the glue, which reads both in CUE; the
operator walk, which op-i3g2 deletes.

**Decision**: No new paths (D5). The parity test builds its paths locally.

**Rationale**: A path with no production reader is removed under that requirement, and once the
operator drops its walk nothing reads them.

### Decode module metadata on call, or once at acquisition

**Context**: The plan said "decoded once in processInstance".

**Explored**: Storing a decode needs an exported field or setter on `module.Instance` that the
kernel package can write, and the "Uniform Artifact Shape" rule allows three fields only. A
struct-literal `Instance` (frontends' tests, `opm/kernel/validate_test.go`) would return nil while
its `Package` holds the metadata. Also read: the cli's raw reads (`render.go`, `values.go`,
`instinit/values.go`) and its best-effort `decodeModuleMetadata`.

**Decision**: Decode on call with the existing unexported `decodeModuleMetadata` (D6), all or
nothing.

**Rationale**: No new API besides the accessor, `Package` stays the source of truth, and the cost
is one lookup and one decode of a small struct.

### Fail-closed boundary of the demand

**Context**: Owner i3: the library "fails closed itself". The operator walk refuses on a bottom
`#traits` (`Exists()` then `Fields()`).

**Explored**: The matcher's unguarded `#resources` read and its `!= _|_` guard on `#traits`, and
the two spikes in section 1 (`no_resources`, `bad_traits`).

**Decision**: `#resources` unguarded, and `#traits` behind the matcher's guard (D3), subject to the
`bad_traits` spike.

**Rationale**: Demand sees exactly what matching sees, and the spikes show whether the render
refuses where the walk would have.

## Measurements

Filled in by tasks 1.1 and 4.1.

## Verification

Filled in by tasks 2.x and 4.2.
