## ADDED Requirements

### Requirement: Single embedded schema with no apiVersion field
The library SHALL ship exactly one OPM CUE schema, vendored under `library/apis/core/`. The schema MUST NOT define a top-level `#ApiVersion` constant. Artifact roots (`#Module`, `#ModuleRelease`, `#Component`, `#ComponentTransformer`, `#Platform`, `#Resource`, `#Trait`) MUST NOT carry an `apiVersion` field.

#### Scenario: Schema package declaration
- **WHEN** any `.cue` file under `library/apis/core/` is parsed
- **THEN** its package clause is `package core`
- **AND** no file declares an `#ApiVersion` constant

#### Scenario: Evaluated module has no apiVersion field
- **WHEN** an artifact authored against the library schema is loaded and evaluated
- **THEN** `apiVersion` on the artifact root does not exist

### Requirement: Path inventory exposed as package-level vars
The library SHALL expose every CUE path used by the kernel, matcher, and renderer as exported package-level `cue.Path` variables in `opm/schema`. The variable names MUST match the artifact-domain term (e.g. `Metadata`, `Components`, `Config`, `Module`, `ModuleMetadata`, `DebugValues`, `Transformers`, `Registry`, `KnownResources`, `KnownTraits`, `ComposedTransformers`, `Matchers`, `MatchersResources`, `MatchersTraits`, `Transform`, `TransformerRequiredLabels`, `TransformerRequiredResources`, `TransformerRequiredTraits`, `TransformerOptionalTraits`, `Component`, `Context`, `Output`, `MetadataLabels`, `MetadataAnnotations`, `MetadataFQN`, `ComponentResources`, `ComponentTraits`).

#### Scenario: Consumer references a path directly
- **WHEN** a kernel consumer needs the path to a release's `components` field
- **THEN** it imports `opm/schema` and references `schema.Components`
- **AND** does not call any `Paths()` method or look up a binding

### Requirement: Metadata decoders are free functions
The library SHALL expose `DecodeModuleMetadata`, `DecodeReleaseMetadata`, `DecodeProviderMetadata`, and `DecodePlatformMetadata` as free functions in `opm/schema`. Each function MUST accept a raw `cue.Value` at the artifact root and return the canonical decoded metadata struct or a non-nil error.

#### Scenario: Decoding a module artifact
- **WHEN** `schema.DecodeModuleMetadata(v)` is called with the root of a valid `#Module` value
- **THEN** it returns `*schema.ModuleMetadata` with `Name`, `ModulePath`, `Version`, `FQN`, `UUID`, `Labels`, `Annotations` populated and a nil error

#### Scenario: Missing metadata is fatal for module/release/platform
- **WHEN** `DecodeModuleMetadata`/`DecodeReleaseMetadata`/`DecodePlatformMetadata` is called with a value whose `metadata` field is absent
- **THEN** it returns nil and an error stating "metadata field is required"

#### Scenario: Provider metadata falls back to caller-supplied name
- **WHEN** `DecodeProviderMetadata(v, "fallback")` is called with a provider whose `metadata` field is absent
- **THEN** it returns `&ProviderMetadata{Name: "fallback"}` and a nil error

#### Scenario: Platform metadata hoists top-level type
- **WHEN** `DecodePlatformMetadata(v)` is called on a `#Platform` whose root has `type: "kubernetes"` alongside its `metadata` block
- **THEN** the returned `PlatformMetadata.Type` is `"kubernetes"`

### Requirement: Transformer-context builder
The library SHALL expose `schema.BuildTransformerContext(ctx, rel, compName, schemaComp, runtimeName)` that constructs the `#TransformerContext` value for a single (release, component, transformer) tuple. The caller's job is to fill the returned value at `schema.Context` on the unified transformer.

The function MUST accept any value implementing `schema.ReleaseView` (`ReleaseName/Namespace/ReleaseUUID/ModuleFQN/ModuleVersion/Labels/Annotations`). It MUST surface metadata-decode failures as non-fatal warnings rather than errors.

#### Scenario: Successful context construction
- **WHEN** `BuildTransformerContext` is called with a non-nil context, a valid `ReleaseView`, a non-empty `compName`, a schema-preserving component value, and a non-empty `runtimeName`
- **THEN** it returns a `cue.Value` carrying `#moduleReleaseMetadata`, `#componentMetadata`, `#runtimeName` and no error

#### Scenario: Empty runtimeName is fatal
- **WHEN** `BuildTransformerContext` is called with `runtimeName=""`
- **THEN** it returns the zero `cue.Value` and an error

#### Scenario: Bad metadata.labels surfaces as warning
- **WHEN** the supplied `schemaComp` has a `metadata.labels` field that cannot be decoded as `map[string]string`
- **THEN** the returned warnings slice contains a message naming the component and the labels field, and no error is returned

### Requirement: Cached embedded-schema loader
The library SHALL expose `schema.SchemaValue(ctx *cue.Context) (cue.Value, error)` that loads the embedded CUE schema package into a `cue.Value`. The loader MUST cache the result so repeated calls amortise the load cost. The cache MUST be safe for concurrent first-call invocation. Schema-load failures MUST be returned wrapped; subsequent calls MUST return the cached error rather than retrying.

#### Scenario: Repeated calls return the same value
- **WHEN** `SchemaValue(ctx)` is called twice with the same `*cue.Context`
- **THEN** both calls return the same underlying instance and the second call does not re-execute `load.Instances`

#### Scenario: Returned value exposes #ModuleRelease
- **WHEN** `SchemaValue(ctx)` is called and the returned value is queried with `LookupPath(cue.ParsePath("#ModuleRelease"))`
- **THEN** the result exists

#### Scenario: No registry round-trip required
- **WHEN** `SchemaValue(ctx)` is called in an environment with `CUE_REGISTRY` unset and no network access
- **THEN** the call succeeds and returns a non-zero `cue.Value`

#### Scenario: Concurrent first calls are safe
- **WHEN** two goroutines call `SchemaValue(ctx)` simultaneously before the cache is warmed
- **THEN** exactly one schema load runs and both goroutines receive the same `cue.Value`

### Requirement: Default-namespace annotation key
The library SHALL expose `schema.AnnotationDefaultNamespace = "module.opmodel.dev/default-namespace"` as the canonical key for the advisory default-namespace annotation defined by ADR-001.

#### Scenario: Constant value
- **WHEN** code references `schema.AnnotationDefaultNamespace`
- **THEN** it resolves to the string `"module.opmodel.dev/default-namespace"`

### Requirement: Loader helpers return only the loaded value
`opm/helper/loader/file.LoadModulePackage`, `LoadReleasePackage`, and `LoadPlatformPackage` MUST have the signature `(ctx *cue.Context, dirPath string, opts LoadOptions) (cue.Value, error)`. The previous `apiversion.Version` return is removed. Their `(*Kernel)` wrappers MUST follow the same signature.

#### Scenario: LoadModulePackage signature
- **WHEN** a caller invokes `file.LoadModulePackage(ctx, dir, opts)`
- **THEN** it returns exactly two values: a `cue.Value` and an `error`

### Requirement: Module, Release, Platform structs do not carry APIVersion
`opm/module.Module`, `opm/module.Release`, and `opm/platform.Platform` MUST NOT have an `APIVersion` field. Their constructors (`NewModuleFromValue`, `NewReleaseFromValue`, `NewPlatformFromValue`) MUST call the appropriate `schema.Decode*Metadata` function directly without consulting any binding registry.

#### Scenario: Module struct has no APIVersion field
- **WHEN** Go code references `module.Module{}`
- **THEN** the literal compiles without an `APIVersion` field
- **AND** there is no exported `apiversion.Version`-typed accessor

#### Scenario: NewModuleFromValue decodes metadata directly
- **WHEN** `NewModuleFromValue(k, v)` is called with a valid `#Module` value
- **THEN** the returned `*module.Module` has `Metadata` populated, no version-dispatch lookup occurs, and `Package == v`

### Requirement: Match drops the binding parameter
`opm/compile.Match` MUST have signature `Match(components cue.Value, plat *platform.Platform) (*MatchPlan, error)`. `(*compile.Module).Execute` MUST drop its binding lookup; internal helpers reference `opm/schema` package vars directly.

#### Scenario: Match signature
- **WHEN** Go code calls `compile.Match(components, plat)`
- **THEN** it compiles and returns the match plan or an error
- **AND** no `api.Binding` parameter is required

#### Scenario: Cross-artifact apiVersion checks are gone
- **WHEN** `kernel.Match` or `kernel.Compile` runs against a release and platform
- **THEN** no apiVersion mismatch check fires (the field does not exist)
