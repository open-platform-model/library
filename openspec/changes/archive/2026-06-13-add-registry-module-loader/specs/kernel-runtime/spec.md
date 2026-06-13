## ADDED Requirements

### Requirement: LoadModuleFromRegistry Method on Kernel

The `Kernel` SHALL expose `(k *Kernel) LoadModuleFromRegistry(ctx context.Context, modPath, version string) (cue.Value, error)` that loads a `#Module` published in an OCI registry, delegating to `opm/helper/loader/registry.LoadModulePackage` using the kernel's owned `*cue.Context` and configured registry (the `registry` field set via `WithRegistry`, inheriting `CUE_REGISTRY` from the process environment when unset). It SHALL return the raw module `cue.Value`, mirroring the existing `Kernel.LoadModulePackage` wrapper (which also returns `(cue.Value, error)`); callers decode it via `Kernel.NewModuleFromValue`. Adding this method SHALL NOT change the signatures of existing kernel methods.

#### Scenario: Delegates to the registry loader

- **WHEN** a caller invokes `k.LoadModuleFromRegistry(ctx, "testing.opmodel.dev/modules/hello@v0", "v0.0.2")`
- **THEN** it returns the `cue.Value` produced by `opm/helper/loader/registry.LoadModulePackage` using the kernel's registry and context
- **AND** the value decodes via `k.NewModuleFromValue` to a `*module.Module` with the author-set `metadata.name`, `metadata.version`, and `metadata.modulePath`

#### Scenario: Existing method signatures unchanged

- **WHEN** a developer reads `LoadModulePackage`, `Match`, `Plan`, and `Compile` after this slice
- **THEN** their signatures are unchanged
