# api-version-dispatch Specification

## Purpose
TBD - created by archiving change add-multi-apiversion-support. Update Purpose after archive.
## Requirements
### Requirement: Schema apiVersion is a concrete literal
Each OPM CUE schema under `apis/core/<version>/` MUST define `#ApiVersion` as a concrete string literal (e.g. `"opmodel.dev/v1alpha2"`). Artifacts that reference `#ApiVersion` (e.g. `#Module`, `#ModuleRelease`, `#Component`, `#ComponentTransformer`) MUST therefore carry the literal in their `apiVersion` field after CUE evaluation, without the user having to write it explicitly.

#### Scenario: Evaluated module carries concrete apiVersion
- **WHEN** an artifact authored against `apis/core/v1alpha2` is loaded and evaluated by the kernel
- **THEN** `apiVersion` on the artifact root resolves to the string `"opmodel.dev/v1alpha2"`

#### Scenario: User does not have to set apiVersion explicitly
- **WHEN** a user authors a `#Module` without setting `apiVersion` in their source
- **THEN** CUE unification with `#Module.apiVersion: #ApiVersion` fills the literal automatically and validation succeeds

### Requirement: Version detection from a CUE artifact
The kernel SHALL expose a `pkg/apiversion.Detect(cue.Value) (Version, error)` helper that reads the `apiVersion` field from any OPM artifact root and returns the matching `Version` constant. The helper MUST NOT mutate the input value and MUST be safe to call concurrently.

#### Scenario: Recognised version
- **WHEN** `Detect` is called with a CUE value whose root has `apiVersion: "opmodel.dev/v1alpha2"`
- **THEN** it returns `Version("opmodel.dev/v1alpha2")` and a nil error

#### Scenario: Unknown version literal
- **WHEN** `Detect` is called with a CUE value whose `apiVersion` is a string the kernel does not recognise
- **THEN** it returns the zero `Version` and an error that wraps `apiversion.ErrUnknownAPIVersion`

#### Scenario: Missing apiVersion field
- **WHEN** `Detect` is called with a CUE value that has no `apiVersion` field
- **THEN** it returns the zero `Version` and an error that wraps `apiversion.ErrUnknownAPIVersion`

#### Scenario: apiVersion is not a string
- **WHEN** `Detect` is called with a CUE value where `apiVersion` is present but not a string kind
- **THEN** it returns the zero `Version` and an error that wraps `apiversion.ErrUnknownAPIVersion`

### Requirement: Binding contract
The kernel SHALL expose a `pkg/api.Binding` interface that owns every version-specific fact: CUE path constants, decoded metadata shape, and transformer-context construction. Each schema version MUST contribute exactly one Binding implementation. The Binding interface MUST cover every CUE path the compile pipeline reads or writes — including release shape, component shape, transformer registry, and transformer context.

#### Scenario: Binding paths cover every compile-pipeline lookup
- **WHEN** the kernel renders a release end-to-end
- **THEN** every CUE path read from or written to a release, component, provider, or transformer value comes from `binding.Paths()` — no hardcoded path strings remain in `pkg/render`, `pkg/loader/{module,provider,release}.go`, or `pkg/module/parse.go`

#### Scenario: Binding owns transformer-context shape
- **WHEN** the renderer needs to fill `#context` on a transformer for a `(release, component)` pair
- **THEN** the value to fill is produced by `binding.BuildTransformerContext(...)` — the renderer does not construct the context shape itself

#### Scenario: Binding owns metadata decoding
- **WHEN** the loader or `module.ParseModuleRelease` needs to extract module, release, or provider metadata
- **THEN** the decode goes through `binding.DecodeModuleMetadata`, `binding.DecodeReleaseMetadata`, or `binding.DecodeProviderMetadata` — no version-specific JSON tags are inlined in `pkg/module` or `pkg/provider`

### Requirement: Binding registry
The kernel SHALL expose a process-wide `pkg/api` registry mapping `apiversion.Version` to `Binding`. Bindings MUST self-register from `init()` of their package. The registry MUST be read-only after process start: no public mutation API beyond `Register` is permitted.

#### Scenario: Lookup of a registered version
- **WHEN** `api.Lookup(apiversion.V1alpha2)` is called after `pkg/api/v1alpha2` has been imported
- **THEN** it returns the v1alpha2 binding and a nil error

#### Scenario: Lookup of an unregistered version
- **WHEN** `api.Lookup` is called with a version no imported package has registered
- **THEN** it returns a nil Binding and a non-nil error

#### Scenario: Duplicate registration is fatal
- **WHEN** `api.Register` is called twice for the same `apiversion.Version` (e.g. two packages claim `v1alpha2`)
- **THEN** the second call panics, surfacing the misconfiguration at process start rather than at first use

#### Scenario: Convenience lookup from a CUE value
- **WHEN** `api.For(v cue.Value)` is called with an OPM artifact root
- **THEN** it returns the binding registered for the artifact's detected version, or an error wrapping `apiversion.ErrUnknownAPIVersion` if detection fails or no binding is registered

### Requirement: Loader surfaces detected version
The loader functions in `pkg/loader` (`LoadModulePackage`, `LoadReleaseFile`, `LoadProvider`) MUST detect the schema version of every successfully loaded artifact and surface it to callers. The `module.Module`, `module.Release`, and `provider.Provider` types MUST carry an `APIVersion apiversion.Version` field populated at load time.

#### Scenario: Loaded release carries its apiVersion
- **WHEN** `LoadReleaseFile` returns successfully
- **THEN** the resulting CUE value has a detectable `apiVersion` and the value passed to `module.ParseModuleRelease` populates `Release.APIVersion`

#### Scenario: Loaded provider carries its apiVersion
- **WHEN** `LoadProvider` returns a `*provider.Provider`
- **THEN** `provider.APIVersion` is populated with the detected version

#### Scenario: Loader rejects artifact with unknown apiVersion
- **WHEN** an artifact is loaded whose `apiVersion` is unrecognised
- **THEN** the loader returns an error wrapping `apiversion.ErrUnknownAPIVersion` and does not produce a partial result

### Requirement: Render dispatches via binding
`render.ProcessModuleRelease` MUST resolve the binding for the supplied release via `api.Lookup(rel.APIVersion)` and pass it through to `Match` and the transform-execution phase. The signature of `render.ProcessModuleRelease` MUST remain `(ctx, *module.Release, *provider.Provider, runtimeName string) (*ModuleResult, error)` — no binding parameter at the public boundary.

#### Scenario: Public render entry point unchanged
- **WHEN** existing downstream code calls `render.ProcessModuleRelease(ctx, rel, p, "opm-cli")`
- **THEN** it compiles and runs without source changes after the refactor

#### Scenario: Mismatched binding between release and provider
- **WHEN** a `*module.Release` with `APIVersion=v1alpha2` is rendered against a `*provider.Provider` with `APIVersion=v1alpha1`
- **THEN** `ProcessModuleRelease` returns an error explaining the version mismatch and does not invoke any transformer

#### Scenario: Unregistered binding for the release version
- **WHEN** `ProcessModuleRelease` is called with a release whose `APIVersion` has no registered binding
- **THEN** it returns an error wrapping the lookup failure and does not invoke any transformer

### Requirement: Embedded schemas
Each `apis/core/<version>/` directory MUST expose its CUE source files via a `go:embed`-backed `embed.FS`. The kernel SHALL provide a helper that returns the embedded filesystem for a given `apiversion.Version`, enabling deterministic offline schema validation.

#### Scenario: Embedded schema available without registry round-trip
- **WHEN** the kernel validates an artifact against the embedded schema for a known version
- **THEN** validation completes without network access and without consulting the user's `CUE_REGISTRY`

#### Scenario: Embedded files match shipped source
- **WHEN** a Go test enumerates the embedded `embed.FS` for `v1alpha2`
- **THEN** the file list and byte contents match the on-disk `apis/core/v1alpha2/**/*.cue` and `cue.mod/module.cue`

### Requirement: New schema version is added without kernel edits
A new OPM schema version `<vN>` MUST be addable by:
1. Creating `apis/core/<vN>/` with the CUE schema and a pinned `#ApiVersion` literal.
2. Creating `pkg/api/<vN>/` implementing `api.Binding` and registering itself in `init()`.
3. Adding a constant `apiversion.V<N>` to `pkg/apiversion`.

No edits SHALL be required in `pkg/render`, `pkg/loader/{module,provider,release}.go`, `pkg/module/parse.go`, `pkg/validate`, `pkg/core`, or `pkg/errors` to support the new version.

#### Scenario: Adding a new version touches only version-scoped packages
- **WHEN** a hypothetical `v1beta1` is added following the three steps above
- **THEN** `git diff` for the change shows modifications only under `apis/core/v1beta1/` and `pkg/api/v1beta1/` plus one constant in `pkg/apiversion`

#### Scenario: Two versions coexist at runtime
- **WHEN** a single kernel build imports both `pkg/api/v1alpha2` and a hypothetical `pkg/api/v1alpha1`
- **THEN** loading and rendering an artifact of either version succeeds within the same process without rebuild

