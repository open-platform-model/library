# Design: core-version-without-schema-build

## Context

See `proposal.md` § Why. `resolveCoreVersion` exists because the synthesized `instance.cue` imports `core` at a major derived from the version (`internal/synth/render.go`, `major(version)`), so synthesis needs the release string before it renders the file. Today that string comes from the schema cache after a full build. Constraints: the bare-major loader form (`opmodel.dev/core@v2`, resolved to `.latest` by `OCILoader.Load`) has no version until a load resolves it; the parity fixtures pin an exact release; the kernel keeps its `schemaLoader` field, so it can inspect the configured loader.

## Goals / Non-Goals

**Goals:**

- A kernel whose loader pins an exact release synthesizes with no schema load.
- The bare-major form keeps working, through the cache as today.

**Non-Goals:**

- Removing the schema cache or changing `Cache.Get`.
- Any change to the acquisition or render evaluation (see the two decisions below that record why not).

## Decisions

### The pinned release is read off the loader, not built

**Context**: `OCILoader.Load` already parses the module identifier and treats a bare major specially (`isBareMajorVersion`); the exact-release case is the identifier's version suffix.
**Explored**: (A) resolve the version through `load.Instances` without `BuildInstance` (still a module-cache lookup and a fetch on a cold cache); (B) a `Cache.Version()` that special-cases its loader (puts loader knowledge in the cache); (C) `OCILoader.PinnedVersion()` and a kernel branch on the configured loader type.
**Decision**: C. `PinnedVersion` uses `ast.SplitPackageVersion` on the identifier (`DefaultSchemaModule` when empty) and reports the version only when it is a full release (`vN.N.N[-pre]`), `("", false)` otherwise. `resolveCoreVersion` checks `k.schemaLoader` (nil means the default `OCILoader{Registry: k.registry}`), returns the pinned release when present, and otherwise falls back to `Get()` and `ResolvedVersion()`.
**Rationale**: no I/O for the default and the pinned cases, which are every production configuration; the bare-major case is unchanged; the loader is the type that knows how it names releases.

### The existence check goes

**Decision**: the `#ModuleInstance` lookup in `resolveCoreVersion` is deleted. A test synthesizes against a served core that lacks `#ModuleInstance` and asserts the error names the definition and comes from the synth build.
**Rationale**: the check could only fire when a loaded schema was not the core schema, and the synth build reports that itself, at the import that fails.

### The acquisition concreteness walk stays

**Context**: slice 6's card proposed skipping `Validate(cue.Concrete(true))` at acquisition and enforcing concreteness in the render build.
**Measured (2026-09-08, `testdata/parity/instance`, warm cache, three runs)**: load 13 to 28 ms; build plus gate lookups 34 to 38 ms; `Validate(Concrete)` 0.17 to 0.18 ms.
**Decision**: no change. The walk is under one percent of an acquisition and is what gives `AcquireInstanceFromDir` and `SynthesizeInstance` their early, path-named concreteness error. The duplicated cost is the load and build, which `Render` repeats because the render module imports the instance by source; removing that duplication would mean rendering from a built value instead of a source, the shape ADR-005 and ADR-006 retired.

### The values attribution passes stay

**Context**: the same card counted up to three validations of layered values. `loader.LoadDir` reports only a root-level error (`val.Err()`), and a `#config` violation sits under `values`, so the post-build `validateSources` pass in `AcquireInstanceFromDir` (`acquire.go:284`) and `SynthesizeInstance` is the detector, not a re-check; the reload in `attributeValuesError` runs only when the build itself fails at the root.
**Decision**: no change. One Go pass on success, one attribution reload on a root failure. The alternative, overlaying each source verbatim so CUE positions are native, needs a package clause per source and shifts every line number by one.

## Risks / Trade-offs

- [A custom `Loader` that is not an `OCILoader`] → takes the cache path as today; nothing regresses.
- [A pinned identifier whose release the module's own `cue.mod` does not carry] → unchanged from today: the synth build resolves `core` through the module's dependency list; the kernel's pin only chooses the import major, as before.

## Migration Plan

One library PR; `task check` green; `task cue:test:flow` proves a registry-module render still resolves. No consumer step. Rollback is a revert.

## Open Questions

None.
