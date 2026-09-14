## Context

See proposal.md for motivation. The facts the approach rests on, as read at alpha.30:

- `opm/internal/synth/render.go` `writeStringMap` ranges a `map[string]string` while writing `metadata.labels` and `metadata.annotations` into the staged `instance.cue`. The bytes land on `Instance.Source.Overlay` and are served to the render build unchanged.
- `opm/kernel/source_loader.go` `compileSource` has two branches. The file-backed branch builds a `load.Config{Dir, Overlay}` with no `Env`; the bytes branch uses `CompileBytes`. `compileSources`, `mergeSources` (acquire.go) and `validateSources` (validate.go) are free functions with no access to the receiver, so `k.loadEnv()` never reaches the load. Every caller is a `*Kernel` method: `ValidateConfigDetailed`, `loadInstanceWithValues`, `attributeValuesError`, `SynthesizeInstance`.
- `opm/internal/loader/registry.go` `FetchModule` wraps the fetched overlay into `load.Source` values, calls `load.Instances`, checks for exactly one instance, builds, checks `Err`, and runs `gate(val, ModuleSpec)`. `opm/internal/loader/load.go` `LoadDir` does the same sequence for an overlay under a root and is documented as the one evaluate-and-shape-gate step. The only differences are error-message framing and where the overlay comes from.
- The other two `load.Instances` call sites are `renderstage.Build` (no gate, must return a value whose evaluation errored because the glue's `gate` field is bottom on refusal) and `schema.OCILoader.loadVersioned` (loads an import path, not a directory). Neither is a `LoadDir` shape.

## Goals / Non-Goals

**Goals:**

- Identical `InstanceInput` produces identical staged bytes.
- One registry mapping on every load the kernel performs, values files included.
- One evaluate-and-shape-gate routine for the two module acquisition paths.

**Non-Goals:**

- Changing any exported signature or the `Source` type.
- Folding `renderstage.Build` or `OCILoader` onto `LoadDir`; their load semantics differ on purpose.
- Deduplicating the values double-compile (merge, then validate); that is a separate efficiency change.

## Decisions

### D1: Thread the environment as a parameter, not by promoting the helpers to methods

`compileSource`, `compileSources`, `mergeSources` and `validateSources` gain an `env []string` parameter; each `*Kernel` caller passes `k.loadEnv()`. The file-backed branch sets `load.Config.Env = env`. The bytes branch ignores it.

Alternative: make the four functions methods on `*Kernel`. Rejected because they stay testable without a `Kernel`, and the parameter makes the dependency visible at every call site, which is how the omission would have been caught.

Go sketch:

```go
func compileSource(ctx *cue.Context, s Source, env []string) (cue.Value, error) {
    // ...
    if isFileBacked(s.Origin) {
        cfg := &load.Config{
            Dir:     filepath.Dir(s.Origin),
            Overlay: map[string]load.Source{s.Origin: load.FromBytes(s.Data)},
            Env:     env, // the kernel's WithRegistry mapping, nil to inherit the process env
        }
        // ...
    }
}
```

### D2: Sort keys in Go, keep the emitter a string builder

`writeStringMap` iterates `slices.Sorted(maps.Keys(m))`. Ascending byte order is stable, needs no new dependency, and the module's Go floor (1.25) already has both packages.

Alternative: build the metadata block as a CUE AST and print it with `format.Node`, as `valuesfile` does for caller values. Rejected: the labels and annotations are identity strings the kernel already quotes with `literal.String.Quote`, so the injection-safety argument that justifies the AST path for values does not apply, and the AST route adds code for no observable gain.

### D3: `FetchModule` calls `LoadDir`

After the empty-overlay refusal, `FetchModule` calls `LoadDir(cueCtx, synthRoot, ".", overlay, env, ModuleSpec)` and then `verifyModuleIdentity` as today. The `load.Source` wrapping, the instance count check, the build, and the gate all move out. The wrapped error text changes from "loading module package `<mv>`" to `LoadDir`'s "loading module package from `<root>`" family; the sentinels and the identity error are unchanged.

Alternative: leave `FetchModule` as is and fix only the values load. Rejected: the duplicated sequence is what let the `Env` omission survive review, and the instance-synthesis spec already requires one routine for its two paths; the registry loader is the remaining copy.

### D4: The other `load.Instances` sites stay

`renderstage.Build` and `OCILoader.loadVersioned` are different operations, not copies. The file-backed values load stays its own call as well: it loads one file at its directory, not a package under a module root, so `LoadDir`'s root-and-package contract does not fit. Each keeps its call and the reason is recorded in the proposal.

## Research & Decisions

### Which loads consult the registry mapping today

**Context**: The `WithRegistry` doc claims one mapping for every operation; the audit found a load that ignores it.
**Explored**: Every `load.Instances` and `modconfig.NewRegistry` call under `opm/` (five load sites, one resolver site) and whether each receives `cueenv.Override`.
**Decision**: Four of five already pass the environment; the file-backed values load is the one that does not, and it is reachable from three public verbs.
**Rationale**: The fix is one field on one config struct plus parameter plumbing; no new mechanism.

### Whether sorted output changes any consumer's rendered objects

**Context**: A byte change in `instance.cue` could reorder `metadata.labels` in rendered objects.
**Explored**: `InstanceInput.Labels` and `Annotations` call sites in cli and opm-operator.
**Decision**: Neither consumer populates either field; the change is observable only to a future caller.
**Rationale**: No migration note is needed.

## Risks / Trade-offs

- [A consumer relied on the process `CUE_REGISTRY` routing a values-file import while passing a narrower `WithRegistry`] → `cueenv.Override` replaces the variable wholesale, so that import now fails at compile with a CUE import error naming the path. This is the documented contract; both first-party consumers set the two to the same mapping.
- [`FetchModule`'s wrapped error text changes] → Sentinels unchanged. The operator's string matchers match "loading package" and "resolving", neither of which the old or new text contains. Loader and kernel tests that assert message text are adjusted in the same section.
- [The registry test for values-file imports needs a served module whose prefix the process environment does not route] → `opm/internal/registrytest` serves modules in-process, returns the mapping, and sets the process `CUE_REGISTRY` to it. The test then sets `CUE_REGISTRY` to `schema.PublicRegistry` with `t.Setenv`, so core still resolves from the shared cache while the served prefix routes only through `WithRegistry`, and asserts the negative case on a kernel without the option.

## Migration Plan

None. PATCH release; consumers pick it up on their next library bump.
