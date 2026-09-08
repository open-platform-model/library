## Why

Two facts on `main` today, and one measurement that closes a third item:

1. **Synthesis builds the whole core schema to read a version string.** `SynthesizeInstance` calls `resolveCoreVersion` (`opm/kernel/synth.go:166-180`), which runs `schemaCache.Get` (a `load.Instances` plus `BuildInstance` of `opmodel.dev/core`, 18 ms on a warm cache and a registry fetch on a cold one), checks that `#ModuleInstance` exists in the result, and reads `ResolvedVersion()`. With the default loader that version is the pinned `schema.DefaultSchemaVersion()`, known before any I/O. The existence check duplicates the synth build's own failure: the synthesized package imports `core` and would fail with CUE's import error if the definition were missing. The operator pays this once per process; the cli pays it on every `opm render` of a registry module, and it is the only implicit network call on that path.
2. **Nothing else reads the cache on the render path.** After slice 2, `resolveCoreVersion` is the only library caller of `schemaCache.Get` outside tests; consumers call it for their own reasons (the cli publish gate, the operator's startup smoke check).
3. **The concreteness walk is not a cost.** The simplification plan's slice 6 card proposed skipping `processInstance`'s `Validate(cue.Concrete(true))` at acquisition and enforcing it inside the render build. Measured on 2026-09-08 against `testdata/parity/instance` on the warm workspace cache (three runs): `load.Instances` 13 to 28 ms, `BuildInstance` plus the gate lookups 34 to 38 ms, `Validate(Concrete)` 0.18 ms. The walk is under one percent of an acquisition; the cost is the load and build, which `Render` must repeat because it imports the instance by source. That item is dropped, and the decision is recorded in `design.md` so it is not re-proposed.

This is slice 6b of the simplification plan reviewed on 2026-09-05. Non-breaking.

**Scope statement (Principle VIII).** One package (`opm/schema`) gains one additive method; one function in `opm/kernel` changes; no verb changes its result.

## What Changes

**`opm/schema` (additive):**

- `OCILoader.PinnedVersion() (string, bool)`: the exact release named by `Module` (or by `DefaultSchemaModule` when `Module` is empty), and `true`, when the identifier names an exact release; `("", false)` for a bare major such as `opmodel.dev/core@v2`.

**`opm/kernel`:**

- `resolveCoreVersion` answers from `PinnedVersion()` when the kernel's loader is an `OCILoader` that pins an exact release (the default), with no schema load; otherwise it loads the cache and reads `ResolvedVersion()` as today. The `#ModuleInstance` existence check is removed; a schema without it fails the synth build with CUE's own import error naming the definition.

**Docs:** `CLAUDE.md` § Schema cache lifetime contract (which call first touches the schema), `opm/kernel/doc.go` (the `SynthesizeInstance` paragraph).

**Not in this change:** the acquisition concreteness walk (kept, see § Why 3); the values attribution passes in `AcquireInstanceFromDir` (kept, see `design.md`); anything about contexts (`kernel-owns-no-build-context`).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `schema-dispatch`: a pinned release is known without a load (new requirement).
- `kernel-runtime`: the synthesis requirement states which core release the synthesized package imports and that a pinned kernel loads no schema for it.

## Impact

**SemVer:** MINOR (Principle VI): one additive exported method; an internal call path changes. No consumer edit.

**Downstream:** none. The operator's startup smoke check and the cli publish gate still call `SchemaCache().Get()` for their own reasons and are unaffected. A cli `opm render` of a registry module no longer fetches or builds `opmodel.dev/core` outside the synth build itself.

**Library:** `opm/schema/loader.go` (+ test), `opm/kernel/synth.go` (+ test), two doc paragraphs.

**Complexity justification (Principle VII):** one ten-line accessor and one branch replace a schema build, an existence check and the `ErrSchemaUnavailable` path it produced; that sentinel keeps its remaining producer (a bare-major loader whose resolution reports no version).
