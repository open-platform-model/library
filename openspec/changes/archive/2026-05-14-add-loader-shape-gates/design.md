## Context

The three package loaders in `opm/helper/loader/file/` (`module.go`, `release.go`, `platform.go`) are near-identical: resolve the directory, stat it, `load.Instances`, `ctx.BuildInstance`, `apiversion.Detect`, return. They return the *raw* `cue.Value` by design — deep validation belongs to the Kernel/Binding layer.

Two facts shape this design:

1. **Authored packages embed the schema.** A module file does `m.#Module` at the package root, so the built `cue.Value` is already unified with the full v1alpha2 `#Module` definition. Every schema-defined field — `#config`, `#components`, `debugValues` — therefore reports `.Exists() == true` regardless of what the author wrote. A presence-only check validates nothing.
2. **`apiVersion` alone is not the artifact identity.** `apiversion.Detect` confirms the *schema version* but not the *artifact type*. `LoadModulePackage` will happily build and return a Platform directory. The `kind` field (`"Module"` / `"ModuleRelease"` / `"Platform"`) is the missing discriminator.

## Goals / Non-Goals

**Goals:**

- Fail fast at the loader boundary when a directory is the wrong artifact type, or is missing identity fields the schema never defaults.
- Give frontends sentinel errors to branch on, mirroring `apiversion.ErrUnknownAPIVersion`.
- Collapse the copy-paste across the three loaders into one shared, table-driven path.

**Non-Goals:**

- Full schema validation. The loader does not audit `#config`/`#components`/`values` against the v1alpha2 definitions — that is the Kernel/Binding's contract and CUE does it better than Go would.
- Changing any public signature. `Load*Package` keep `(cue.Value, apiversion.Version, error)`.
- Validating `#defines` presence — it is `#defines?:` (optional) in the schema; a pure-config module legitimately has none.

## Decisions

### Shape gate vs schema validation

The loader runs a **shape gate**: cheap, structural, fast-fail, with a clear "you pointed me at the wrong directory" message. It does *not* re-implement unification-against-schema in Go. Rationale: the v1alpha2 `#Module`/`#ModuleRelease`/`#Platform` definitions already express the full contract; duplicating a subset of it in Go drifts out of sync and adds SemVer surface for no gain (Principle VII). The gate is limited to the minimal discriminator set — root is a struct, `kind` matches, identity fields are concrete.

*Alternative considered:* validate every field the proposal's original field lists named (`#config`, `#components`, etc.). Rejected — the embedded schema makes `.Exists()` always true, so the check is theatre; and a concreteness check on `_`-typed fields like `#config` is meaningless at load time.

### Existence vs concreteness

The meaningful primitive is **concrete non-empty**, not "exists". `metadata.name!` is a required field the schema never defaults; an author who omits it leaves the field present-but-non-concrete. So the gate checks `value.LookupPath(p)` for `Exists() && IsConcrete()` and (for strings) non-empty — the same shape as `apiversion.Detect`'s `kind != StringKind` guard.

### `kind` as the discriminator

Each loader asserts the concrete `kind` literal matches its artifact. For Release and Platform, the embedded/registered `#module` is recursively shape-checked (`#module.kind == "Module"`) — a release wrapping a non-module, or a registry entry pointing at a Platform, fails at load.

*Alternative considered:* infer the artifact type from `kind` and dispatch instead of asserting. Rejected — the caller already chose `LoadModulePackage`; honoring that choice and rejecting mismatches is clearer than silently loading whatever was found.

### Shared helper, table-driven

Extract the common path into one internal helper parameterized by an artifact spec: `{expectedKind string, requiredConcreteFields []string, moduleRefPaths []string}`. The three public functions become thin wrappers. This kills the existing triplication and means new checks land once. Helper stays unexported — no new public surface (Principle IV).

### Instance-count assertion

Change `len(instances) == 0` to `len(instances) != 1`. A `"."` load resolves exactly one package; CUE already errors on conflicting `package` clauses across files (surfaced via `instances[0].Err`). The `!= 1` form asserts the invariant the code currently *assumes*, catching any future loader-config change that widens the arg set. The conflicting-`package` case needs no new code — only a regression test fixture.

### Sentinel errors

Add `ErrInvalidPackage` (root not a struct / instance-count violation), `ErrWrongKind`, and `ErrMissingRequiredField` in the `file` package, alongside the existing `apiversion.ErrUnknownAPIVersion` it already wraps. Each returned error wraps the relevant sentinel via `%w` so CLI (human message) and controller (programmatic branch) frontends can react differently.

## Risks / Trade-offs

- **Behavior tightening** → directories that previously loaded despite being the wrong artifact type now error. Mitigation: those inputs were already malformed for the caller's purpose; the failure moves earlier, not into new territory. Documented as MINOR-with-note in the proposal.
- **`kind` may be absent if the author never embedded the schema** → handled: a missing/non-string/non-concrete `kind` is itself a shape-gate failure (`ErrWrongKind` or `ErrMissingRequiredField`), which is the correct outcome — that package is not an OPM artifact.
- **Recursive `#module` check could be costly on large registries** → the check is a single `LookupPath` + concrete-string read per entry, not a deep walk; negligible against the build cost already paid.

## Open Questions

- Should the sentinels live in `opm/errors` (`oerrors`) rather than the `file` package? The constitution points structured errors there. Leaning `file`-local for now since they are loader-specific and `apiversion` already keeps its own sentinel; revisit if a second package needs them.
