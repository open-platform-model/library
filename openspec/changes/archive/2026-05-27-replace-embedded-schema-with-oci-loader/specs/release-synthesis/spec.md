## MODIFIED Requirements

### Requirement: synth.Release function signature

The `opm/helper/synth/` package SHALL expose a function `Release(ctx *cue.Context, in ReleaseInput) (cue.Value, error)` that returns a `#ModuleRelease` artifact CUE value built by unifying the input fields against the `#ModuleRelease` schema definition resolved from the supplied `SchemaCache`.

The `ReleaseInput` struct SHALL carry: `Module *module.Module` (required), `Name string` (required), `Namespace string` (required), `SchemaCache *schema.Cache` (REQUIRED), `Values cue.Value` (optional; zero value means "no values supplied"), `Labels map[string]string` (optional), `Annotations map[string]string` (optional).

`synth.Release` MUST return a non-nil error when `SchemaCache == nil` (in addition to the existing required-field checks). The error message MUST name the missing field. The helper MUST NOT self-construct a `*schema.Cache` as a fallback; the caller is responsible for passing the cache it intends to share (typically `k.SchemaCache()` from its Kernel).

#### Scenario: Required inputs validated

- **WHEN** `synth.Release` is called with `Module == nil`, or `Name == ""`, or `Namespace == ""`, or `SchemaCache == nil`
- **THEN** it returns the zero `cue.Value` and a non-nil error naming the missing field

#### Scenario: Returned value is schema-unified

- **WHEN** `synth.Release` is called with valid inputs and a `SchemaCache` whose Loader resolves the OPM core schema
- **THEN** the returned `cue.Value` carries the `#ModuleRelease` shape at its root
- **AND** the value is unified with the schema's `#ModuleRelease` definition (the schema's structural constraints apply)

#### Scenario: Caller's Cache is reused, not replaced

- **WHEN** `synth.Release` is called with a `SchemaCache` that has already been warmed by a prior `Get`
- **THEN** the helper invokes `(*Cache).Get(ctx)` to retrieve the already-cached value
- **AND** no second schema load is triggered by the helper

### Requirement: Schema obtained through caller-supplied Cache

`synth.Release` SHALL obtain the `#ModuleRelease` definition by calling `in.SchemaCache.Get(ctx)` on the caller-supplied `*schema.Cache`, then `LookupPath("#ModuleRelease")` on the returned value. The helper MUST NOT call `load.Instances` directly, MUST NOT consult `os.Getenv("CUE_REGISTRY")`, MUST NOT read from the filesystem, and MUST NOT construct its own `*schema.Cache` or `Loader`.

#### Scenario: Helper delegates schema loading to the Cache

- **WHEN** `synth.Release` is called with a `SchemaCache` configured against a pre-seeded test cache
- **THEN** the call succeeds without any direct call to `load.Instances`, `os.Getenv`, or filesystem reads originating in `opm/helper/synth/`

#### Scenario: Schema load failure surfaces as a wrapped error

- **WHEN** `(*Cache).Get(ctx)` returns a non-nil error during a `synth.Release` invocation
- **THEN** `synth.Release` returns the zero `cue.Value` and an error wrapping the Cache's error

#### Scenario: No registry round-trip on warm cache

- **WHEN** `synth.Release` is called with a `SchemaCache` whose underlying CUE module cache is already warm
- **THEN** the call completes without contacting any external registry, regardless of `CUE_REGISTRY` value
