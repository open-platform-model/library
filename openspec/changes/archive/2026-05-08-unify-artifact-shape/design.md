## Context

The kernel will eventually accept four artifact types: `Module`, `ModuleRelease`, `Platform`, and (deferred) Claim-related artifacts. With each having a custom Go shape, the kernel's input contract becomes a moving target — every new artifact type adds a new contract. Multi-version dispatch (`add-multi-apiversion-support`) compounds the problem because each shape must be decoded per version.

Collapsing everything to `(APIVersion, Metadata, Package cue.Value)` makes the contract one rule. `Package` carries the whole loaded CUE; the kernel reads field paths through the version binding's `Paths()` accessor. Typed `Metadata` remains as a cheap projection for hot-path access (log fields, name lookups), but the CUE package is authoritative — when the two disagree, `Package` wins.

This slice depends on `add-multi-apiversion-support`. The binding interface provides:

- `apiversion.Detect(cue.Value) (Version, error)` — read the `apiVersion` field from a raw artifact.
- `api.Lookup(Version) (Binding, error)` — get the binding for that version.
- `binding.Paths()` — the path constants (Spec, Config, Components, etc.).
- `binding.DecodeModuleMetadata(cue.Value)` and equivalents — typed metadata decoders.

## Goals / Non-Goals

**Goals:**
- One Go contract for every OPM artifact type accepted by the kernel: `(APIVersion, Metadata, Package)`.
- Constructor helpers that take a raw `cue.Value` and return the typed artifact, performing version detection and metadata decode in one place.
- `Package cue.Value` is authoritative; `Metadata` is a cache for ergonomic access.
- Remove `Spec`, `Config`, `Values`, embedded `Module` from `module.Module` and `module.Release`. Internal callers migrate to `Package.LookupPath` via the binding.
- Provide a clean migration path for downstream consumers (CLI, operator) — small per-call-site change, well-documented.

**Non-Goals:**
- Refactoring `provider.Provider`. Provider is retired in slice 09 in favor of `Platform` (slice 08).
- Introducing the `Platform` type — that is slice 08, which builds on this slice's shape.
- Changing the loader's external interface — loaders still return `cue.Value`; a new helper layer builds typed artifacts on top.
- Changing values-input handling — that is slice 04.

## Decisions

**Constructor helpers, not constructors.** `NewModuleFromValue(k *Kernel, v cue.Value) (*Module, error)` rather than `NewModule(...)`. Reason: clearer that input is an existing CUE value being adapted, not a fresh build. Pairs naturally with loader output.

**`Metadata` is a pointer, not embedded.** `Metadata *ModuleMetadata` (not `Metadata ModuleMetadata`). Reason: explicit nil case for "metadata could not be decoded yet" during error paths; avoids zero-value confusion.

**Constructor signature accepts a `CueContextOwner` interface, not `*Kernel` directly.** The shipped signature is `module.NewModuleFromValue(k CueContextOwner, v cue.Value)` where `CueContextOwner` is `interface { CueContext() *cue.Context }` defined in `opm/module`. `*kernel.Kernel` satisfies the interface so call-site ergonomics match (`k.NewModuleFromValue(v)` via the `opm/kernel` wrapper, or `module.NewModuleFromValue(k, v)` direct). The interface indirection avoids a `opm/module → opm/kernel` import cycle; `apiversion.Detect` and `api.Lookup` are process-globals so the kernel parameter is currently unused but reserved for future kernel-scoped state (logger, tracer) without an API break.

**`Package` field name, not `Cue` or `Spec` or `Document`.** Reason: matches the umbrella decision (D10) and the user's mental model from CUE — a Module is a CUE package. Avoids the v1alpha1 connotation of `Spec`.

**`APIVersion` is denormalized.** It lives on the struct AND in `Package` via `apiVersion: ...`. Constructor stamps the field from `Package` at construction time. After construction the two are in sync; if `Package` is mutated, the field is stale. The doc comment states: "`APIVersion` is set at construction; do not mutate `Package`'s `apiVersion` field afterward."

**Internal callers migrate in this slice.** `opm/render/match.go`, `opm/render/execute.go`, `opm/module/parse.go`, `opm/validate/config.go` — every read of `mod.Spec`, `rel.Values`, etc. is rewritten to `Package.LookupPath(binding.Paths()....)`. The binding interface from `add-multi-apiversion-support` makes this a mechanical edit.

**Tests update in lockstep.** Test fixtures that construct `module.Module{Spec: ..., Config: ...}` directly are rewritten to use the constructor or to set `Package` directly. The umbrella's "fixtures live in `library/testdata/`" note is honored — fixtures stay shareable across CLI / operator tests.

## Risks / Trade-offs

**Risk — large mechanical churn.** Every site that read the removed fields must migrate. Mitigation: do the migration in this slice (all in one place) rather than letting it fan out across later slices. The `add-multi-apiversion-support` binding makes the new access pattern uniform.

**Risk — `Metadata` drifting from `Package`.** A future slice mutates `Package` without updating `Metadata`. Mitigation: clear doc comment; avoid mutation pattern in kernel internals; one migration path away from this is to drop the typed cache entirely (rejected for ergonomics — D3).

**Risk — downstream consumers cannot migrate quickly.** `cli` and `opm-operator` may still reference removed fields. Mitigation: ship this slice with a CHANGELOG entry showing the before/after for each removed field. The binding paths from `add-multi-apiversion-support` give every consumer the same access pattern.

**Trade-off — `Package cue.Value` is opaque.** Consumers can no longer write `mod.Spec.LookupPath(...)` directly; they need a binding to know the path. This is intentional — it forces every reader to go through version-aware dispatch and keeps single-version assumptions out of consumer code.
