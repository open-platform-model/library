# Schema — Platform Capabilities

## Summary

One new file in `core/v1alpha2` (flat package, no subdirectories):

- `core/v1alpha2/capability.cue` — `#Capability`, `#CapabilityMap`

Four modifications:

- `core/v1alpha2/context_builder.cue` — `#ContextBuilder` gains `#platform` + `#consumes` inputs and an `out.consumes` output (matched provider specs unified back into `#consumes`)
- `core/v1alpha2/platform.cue` — `#Platform` gains `#provides`
- `core/v1alpha2/module.cue` — `#Module` gains `#consumes`
- `core/v1alpha2/module_release.cue` — `#ModuleRelease` gains kernel-populated `#platform: #Platform`; passes `#platform` / `#consumes` to `#ContextBuilder`; unifies `out.consumes` into `#module.#consumes`

`apis/core/v1alpha2/context.cue` is **untouched** — `#ModuleContext` stays single-layer (`{ runtime: #RuntimeContext }`); 006 introduces no `#ctx.capabilities` field and no `#CapabilitySet` type.

## `#Capability`

The FQN-identified, schema-bearing context primitive. Sibling to `#Resource` and `#Trait` — same identity-plus-schema shape, but a render *input* (nothing renders a `#Capability`).

```cue
// apis/core/v1alpha2/capability.cue
package v1alpha2

// #Capability: an FQN-identified, schema-bearing unit of platform-supplied
// context. Sibling to #Resource and #Trait — same identity + schema pattern,
// but a render *input* (a platform provides it, a module reads it) rather
// than a render *output* (no transformer renders a #Capability).
#Capability: {
	apiVersion: #ApiVersion
	kind:       "Capability"

	metadata: {
		name!: #NameType // Example: "route"
		#definitionName: (#KebabToPascal & {"in": name}).out

		modulePath!: #ModulePathType                              // Example: "opmodel.dev/opm/capabilities/routing"
		version!:    #MajorVersionType                            // Example: "v1"
		fqn:         #FQNType & "\(modulePath)/\(name)@\(version)" // Example: "opmodel.dev/opm/capabilities/routing/route@v1"

		description?: string
		labels?:      #LabelsAnnotationsType
		annotations?: #LabelsAnnotationsType
	}

	// The interface — the schema consumers program against and providers
	// populate. MUST be an OpenAPIv3-compatible schema.
	//
	// Unlike #Resource / #Trait, spec is NOT wrapped in a camelCase
	// single-field struct: capabilities are addressed by FQN in #consumes,
	// never flattened into a component's merged spec, so the wrapper buys
	// nothing here (D2).
	spec!: _
}

// #CapabilityMap: FQN-keyed map of capability definitions/instances.
// Used both for #Platform.#provides and for the two sub-maps under
// #Module.#consumes.
#CapabilityMap: [#FQNType]: #Capability
```

There is no separate `#CapabilitySet` type. `#consumes` itself doubles as the resolved read surface (D8) — the matcher unifies provider specs back into `#consumes.required` / `#consumes.optional`, so module bodies read straight from `#consumes`.

## `#Module.#consumes`

```cue
// apis/core/v1alpha2/module.cue
#Module: {
	// ... existing fields ...

	// Capabilities this module requires from the target platform.
	// Authored as a declaration (FQN + schema); after #ContextBuilder
	// matches against #platform.#provides, the matched concrete spec is
	// unified back into the entry. Module bodies read the resolved value
	// from this same field — declaration and read surface are co-located,
	// mirroring #Component.#resources (component.cue:23-55).
	//
	// required — a missing provider leaves the entry's spec! incomplete;
	//            `cue vet -c` reports the FQN as a release-time error (D6).
	// optional — a missing provider drops the entry entirely; module
	//            guards reads with `if #consumes.optional[<fqn>] != _|_`.
	//
	// Mirrors the transformer requiredResources / optionalResources split.
	// Non-optional pattern fields: an unpopulated map is {}, iterable by
	// #ContextBuilder without the optional-absent _|_ dance (D9).
	#consumes: {
		required: [FQN=#FQNType]: #Capability & {
			metadata: fqn: FQN
		}
		optional: [FQN=#FQNType]: #Capability & {
			metadata: fqn: FQN
		}
	}
}
```

## `#Platform.#provides`

```cue
// apis/core/v1alpha2/platform.cue
#Platform: {
	// ... existing fields (#registry, #knownResources, #matchers, …) ...

	// Concrete capability instances this platform supplies.
	// FQN-keyed; specs MUST be concrete. Single source for capability
	// values — there is no second provider tier (D4).
	//
	// Per-platform variation is handled by CUE unification of #Platform
	// values (#KindDev: #KindBase & {#provides: {...}}) — see OQ6.
	#provides: [FQN=#FQNType]: #Capability & {
		metadata: fqn: FQN
	}
}
```

## `#ModuleRelease.#platform` (kernel-populated)

```cue
// apis/core/v1alpha2/module_release.cue
#ModuleRelease: {
	// ... existing fields (metadata, #module, values) ...

	// Target platform — kernel-populated (D13). End-users authoring a
	// release MUST NOT write this field; the runtime/CLI/operator unifies
	// #platform: <chosen> into the release at apply time via FillPath.
	// The release artifact stays portable; the platform binding is a
	// runtime decision.
	//
	// Same precedent as #TransformerContext.#runtimeName! (transformer.cue:121):
	// "Mandatory — CUE evaluation fails if the runtime forgets to fill
	// this." `cue vet -c` against an unbound release errors as expected;
	// apply-time evaluation supplies #platform and everything concretizes.
	//
	// #-prefixed: excluded from `cue export`; #Platform values do not leak
	// into rendered manifests.
	#platform: #Platform

	// ... let bindings + #ContextBuilder invocation + components extraction
	//     (see "#ModuleRelease integration" below) ...
}
```

## `#ContextBuilder` matching step

004's slimmed `#ContextBuilder` has three inputs (`#release`, `#module`, `#components`) and produces `out.ctx.runtime` plus `out.injections`. 006 adds two inputs (`#platform`, `#consumes`) and one output (`out.consumes`). The 004 parts (`_componentNames`, `runtime`, `injections`) are unchanged and abbreviated here.

```cue
// apis/core/v1alpha2/context_builder.cue  (modified)
#ContextBuilder: {
	// ---- 004 inputs (unchanged) ----
	#release:    {name: #NameType, namespace: string, uuid: #UUIDType}
	#module:     {name: #NameType, version: #VersionType, fqn: string, uuid: #UUIDType}
	#components: [string]: _

	// ---- 006 inputs ----
	// #platform comes from #ModuleRelease.#platform, which is kernel-populated.
	#platform: #Platform
	// #consumes comes from the config-resolved module (see #ModuleRelease).
	#consumes: {
		required: [#FQNType]: #Capability
		optional: [#FQNType]: #Capability
	}

	// ... 004 let: _componentNames ...

	// ---- 006: capability matching ----
	// For each consumed FQN, unify #platform.#provides[fqn] into the
	// consumed #Capability.
	//
	// required: if no provider exists, cap.spec! stays incomplete and
	//   `cue vet -c` reports out.consumes.required.<fqn>.spec — a
	//   release-time error naming the missing capability (D6).
	// optional: the comprehension `if` drops the entry entirely when no
	//   provider exists, so out.consumes.optional[fqn] is absent.
	// Schema mismatches (provider's spec violates capability schema) become
	// CUE bottoms at the FQN — release-time errors either way.
	let _matched = {
		required: {
			for fqn, cap in #consumes.required {
				(fqn): cap & {
					if #platform.#provides[fqn] != _|_ {
						#platform.#provides[fqn]
					}
				}
			}
		}
		optional: {
			for fqn, cap in #consumes.optional
			if #platform.#provides[fqn] != _|_ {
				(fqn): cap & #platform.#provides[fqn]
			}
		}
	}

	out: {
		ctx: #ModuleContext & {
			runtime: #RuntimeContext & {
				// ... 004: release, module, components ...
			}
			// Note: no `capabilities` field on #ModuleContext (D8).
		}

		injections: {
			// ... 004: per-component #names ...
		}

		// 006 — matched specs, unified back into #module.#consumes by
		// #ModuleRelease alongside out.ctx and out.injections.
		consumes: _matched
	}
}
```

### Why a conditional struct, not a `*` default disjunction

Same reasoning as 004 D33: `#platform.#provides[fqn]` is a map lookup that yields `_|_` for absent keys. A `*` default disjunction (`*#platform.#provides[fqn] | <fallback>`) does not fall through cleanly on `_|_`; the `!= _|_` conditional-struct guard does. Because `#provides` is a **non-optional** pattern field (D9), an unpopulated map is `{}` and `{}[fqn]` is a clean `_|_` — no "cannot reference optional field" surprise.

## `#ModuleRelease` integration

004's slimmed `#ModuleRelease` already does the three-step config-first dance (004 D34): unify values, invoke `#ContextBuilder` inline, unify outputs back. 006 extends every step:

1. The kernel populates `#platform`.
2. The builder call adds `#platform: #platform` and `#consumes: _withConfig.#consumes` to its inputs.
3. The unify-back step adds `#consumes: _builderOut.consumes` alongside `#ctx` and `#components`.

```cue
// apis/core/v1alpha2/module_release.cue  (modified)
#ModuleRelease: {
	apiVersion: "opmodel.dev/core/v1alpha2"
	kind:       "ModuleRelease"

	metadata: {
		name!:      #NameType
		namespace!: string
		uuid!:      #UUIDType
		...
	}

	#module: #Module
	values:  _

	// 006 — kernel-populated; end-users do not author this field.
	#platform: #Platform

	// Step 1 — unify values into #config so dynamic #components materialise (004 D34).
	let _withConfig = #module & {#config: values}
	let _moduleMetadata = _withConfig.metadata

	// Step 2 — feed the post-config component map AND the post-config
	// #consumes AND the kernel-populated #platform to the builder.
	let _builderOut = (#ContextBuilder & {
		#release: {
			name:      metadata.name
			namespace: metadata.namespace
			uuid:      metadata.uuid
		}
		#module: {
			name:    _moduleMetadata.name
			version: _moduleMetadata.version
			fqn:     _moduleMetadata.fqn
			uuid:    _moduleMetadata.uuid
		}
		#components: _withConfig.#components
		#platform:   #platform                 // 006
		#consumes:   _withConfig.#consumes      // 006
	}).out

	// Step 3 — unify the builder's outputs back into the (config-resolved) module.
	let unifiedModule = _withConfig & {
		#ctx:        _builderOut.ctx
		#components: _builderOut.injections
		#consumes:   _builderOut.consumes       // 006 — matched specs land here
	}

	components: {
		for name, comp in unifiedModule.#components {(name): comp}
	}
}
```

By the time `components` are extracted, `#consumes` has its matched specs unified in and component bodies that read `#consumes.required[fqn].spec.X` resolve concretely. If the kernel hasn't filled `#platform` (e.g. `cue vet -c` against the release alone), the `_matched` lookup against `#platform.#provides[fqn]` is non-concrete and the relevant entries surface as incomplete values — the correct diagnostic for an unbound release.

## Field documentation

### `#Capability`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `metadata.name` | `#NameType` | yes | Capability name, e.g. `route` |
| `metadata.modulePath` | `#ModulePathType` | yes | Owning package path |
| `metadata.version` | `#MajorVersionType` | yes | Major version, e.g. `v1` |
| `metadata.fqn` | `#FQNType` | computed | `\(modulePath)/\(name)@\(version)` |
| `spec` | `_` (OpenAPIv3) | yes | The interface — schema providers populate and consumers read |

### `#Module.#consumes`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `required` | `[#FQNType]: #Capability` | yes (defaults `{}`) | Capabilities that must be provided; missing → release error. Doubles as read surface once matched. |
| `optional` | `[#FQNType]: #Capability` | yes (defaults `{}`) | Capabilities used if provided; missing → entry absent. Doubles as read surface once matched. |

### `#Platform.#provides`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `#provides` | `[#FQNType]: #Capability` | yes (defaults `{}`) | Concrete capability instances; sole source of capability values |

### `#ModuleRelease.#platform`

| Field | Type | Required | Who sets it | Description |
|-------|------|----------|-------------|-------------|
| `#platform` | `#Platform` | yes | Kernel/runtime (NOT end-user) | The chosen target platform; runtime fills via `FillPath` at apply time |

## File locations

### New files

| Path | Purpose |
|------|---------|
| `apis/core/v1alpha2/capability.cue` | `#Capability`, `#CapabilityMap` |

### Modified files

| Path | Change |
|------|--------|
| `apis/core/v1alpha2/context_builder.cue` | `#ContextBuilder` gains `#platform` + `#consumes` inputs; adds `_matched` capability-matching let; adds `out.consumes` output |
| `apis/core/v1alpha2/platform.cue` | `#Platform` gains `#provides` |
| `apis/core/v1alpha2/module.cue` | `#Module` gains `#consumes` |
| `apis/core/v1alpha2/module_release.cue` | `#ModuleRelease` gains kernel-populated `#platform: #Platform`; passes `#platform` / `#consumes` to `#ContextBuilder`; unifies `out.consumes` into `#module.#consumes` |

`apis/core/v1alpha2/context.cue` is untouched.
