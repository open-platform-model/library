## ADDED Requirements

### Requirement: Synthesize a Platform value from typed inputs

The package `opm/helper/synth` SHALL expose `Platform(ctx *cue.Context, in PlatformInput) (cue.Value, error)`, a peer of `Release`, that builds a `#Platform` artifact value by unifying caller-supplied typed inputs against the `#Platform` definition resolved through `in.SchemaCache`. The returned `cue.Value` SHALL carry the caller-supplied identity and subscription fields and leave the kernel-filled materialization slots (`#composedTransformers`, `#matchers`) unset, since those are populated later by `Materialize`.

#### Scenario: Minimal valid platform

- **WHEN** `Platform` is called with `Name`, `Type`, and a `SchemaCache` resolving `#Platform`, and no subscriptions
- **THEN** it returns a `cue.Value` that unifies cleanly against `#Platform` with `metadata.name` and `type` set and `#registry` empty

#### Scenario: Materialization slots left unset

- **WHEN** `Platform` returns a value for any valid input
- **THEN** `#composedTransformers` and `#matchers` are absent on that value (synthesis never fills them)

### Requirement: Required inputs are validated with sentinel errors

`Platform` SHALL reject missing required inputs before touching the schema, returning a sentinel error matchable via `errors.Is`. The required inputs are `Name`, `Type`, and `SchemaCache`. When the resolved schema does not expose `#Platform`, `Platform` SHALL return a schema-unavailable sentinel.

#### Scenario: Missing name

- **WHEN** `Platform` is called with an empty `Name`
- **THEN** it returns an error satisfying `errors.Is(err, ErrMissingName)` and no schema fetch is attempted

#### Scenario: Missing type

- **WHEN** `Platform` is called with an empty `Type`
- **THEN** it returns an error satisfying `errors.Is(err, ErrMissingType)`

#### Scenario: Missing schema cache

- **WHEN** `Platform` is called with a nil `SchemaCache`
- **THEN** it returns an error satisfying `errors.Is(err, ErrMissingSchemaCache)`

#### Scenario: Schema without #Platform

- **WHEN** the supplied `SchemaCache` resolves but the package does not expose `#Platform`
- **THEN** `Platform` returns an error satisfying `errors.Is(err, ErrSchemaUnavailable)`

### Requirement: Subscriptions and filters map onto the registry

`PlatformInput` SHALL carry an optional typed `Subscriptions` map keyed by catalog module path. Each entry SHALL map onto one `#registry` subscription: an optional `Enable` (pointer-typed so an omitted value defers to the schema's `*true` default rather than forcing `false`) and an optional `Filter` carrying `Range`, `Allow`, and `Deny`. A path that violates `#ModulePathType` SHALL surface as a CUE unification error from `Platform`.

#### Scenario: Subscription with filter

- **WHEN** `Platform` is called with a subscription at `"opmodel.dev/catalogs/opm"` whose `Filter.Range` is `">=1.0.0 <2.0.0"`
- **THEN** the returned value has `#registry["opmodel.dev/catalogs/opm"].filter.range` equal to `">=1.0.0 <2.0.0"`

#### Scenario: Enable omitted defers to schema default

- **WHEN** a subscription is supplied with `Enable` left nil
- **THEN** the returned value's `enable` for that path resolves to the schema default `true`

#### Scenario: Enable explicitly false

- **WHEN** a subscription is supplied with `Enable` pointing to `false`
- **THEN** the returned value's `enable` for that path is `false`

#### Scenario: Invalid catalog path

- **WHEN** a subscription key does not satisfy `#ModulePathType`
- **THEN** `Platform` returns a non-nil error describing the unification failure

### Requirement: Kernel entry point returns a typed pre-materialize Platform

`*kernel.Kernel` SHALL expose `SynthesizePlatform(ctx context.Context, in synth.PlatformInput) (*platform.Platform, error)` as the recommended entry point. It SHALL default `in.SchemaCache` to the kernel-owned cache when nil, chain `synth.Platform` into `NewPlatformFromValue`, and return a typed `*platform.Platform`. It SHALL NOT call `Materialize`; resolving subscriptions into a `*MaterializedPlatform` remains a separate, explicit, caller-driven step.

#### Scenario: Synthesize returns decoded platform

- **WHEN** `SynthesizePlatform` is called with valid inputs on a kernel with a working schema cache
- **THEN** it returns a non-nil `*platform.Platform` whose `Metadata.Name` and `Metadata.Type` equal the inputs and whose `Package` is the synthesized value

#### Scenario: Schema cache defaulted from kernel

- **WHEN** `SynthesizePlatform` is called with `in.SchemaCache` nil
- **THEN** the kernel substitutes its own cache and synthesis succeeds without the caller threading a cache

#### Scenario: No registry I/O performed

- **WHEN** `SynthesizePlatform` completes
- **THEN** no registry materialization has occurred and `Package` carries `#registry` as authored with `#composedTransformers` / `#matchers` unset

#### Scenario: Synthesized platform materializes downstream

- **WHEN** the `*platform.Platform` returned by `SynthesizePlatform` is passed to `Kernel.Materialize`
- **THEN** materialization proceeds exactly as it would for a file-loaded platform of the same content
