# kernel-runtime Specification

## Purpose
The `Kernel` struct is the public anchor type for the OPM kernel runtime. It owns the `*cue.Context` and the schema cache used by every kernel operation, so downstream consumers (CLI, operator, Crossplane function) attach to a single mental anchor instead of importing the loader / module / render / validate packages individually. All future kernel-facing slices modify this capability.

## Requirements

### Requirement: Configuration Options

The Kernel SHALL accept configuration through functional options of type `Option`. The provided options SHALL be `WithSchemaLoader` and `WithRegistry`; the Kernel SHALL NOT expose an injection slot no kernel operation reads.

#### Scenario: Adding new options preserves backward compatibility

- **WHEN** a future slice adds a new option (e.g. `WithSchemaRegistry`)
- **THEN** existing callers of `kernel.New(...)` continue to compile and run unchanged

#### Scenario: No observability slots ahead of a reader

- **WHEN** a developer inspects the `Kernel` struct and its options after this change
- **THEN** no logger, tracer or clock field or option exists
- **AND** the injection surface for the execution half is introduced by enhancement 0009 together with its first reader (revised enhancement 0009 ("The execution half owns cancellation inside a stage and introduces its own injection surface"))

### Requirement: Goroutine Safety Contract
A single `Kernel` SHALL be safe for concurrent use across its own method calls: no operation shares evaluation state with another, because each creates its own `cue.Context` and drops its references to it on return, and the schema cache is memoized under synchronization. A consumer that needs concurrent operations uses one `Kernel` per process; the package documentation SHALL state this and SHALL NOT recommend one Kernel per goroutine.

#### Scenario: Documentation states the contract

- **WHEN** a developer reads the godoc for the `Kernel` type
- **THEN** it states that a single `Kernel` is safe for concurrent use across its method calls, shows one Kernel shared by concurrent goroutines, and states that every operation shares nothing

#### Scenario: Documentation retracts the shared-platform model

- **WHEN** a developer reads the godoc for the `Kernel` type
- **THEN** no shared materialized platform, held built value or mutex-serialised render appears as a supported shape; the retraction is recorded in ADR-002's supersession header and the shares-nothing rule in ADR-005 and ADR-007

#### Scenario: Repeated renders retain nothing

- **WHEN** `Render` is invoked repeatedly on one Kernel
- **THEN** each invocation builds in a fresh context, the results are byte-identical for identical inputs, and the kernel holds no built value between calls

#### Scenario: Concurrent acquisitions on one Kernel

- **WHEN** several goroutines acquire artifacts and synthesize instances on one Kernel at the same time under the race detector
- **THEN** every call succeeds with the same result as the sequential run and the race detector reports nothing

#### Scenario: Race detector runs on the render packages

- **WHEN** the repository test task runs
- **THEN** `opm/kernel` and `opm/internal/renderstage` are additionally run under `go test -race`

### Requirement: No Utility Methods on Kernel

The Kernel SHALL expose only the pipeline it runs (acquire, validate, synthesize, render) and SHALL NOT expose a finalization, constraint-stripping, raw-load or other value-utility method.

#### Scenario: No finalization method on the Kernel

- **WHEN** a consumer inspects the exported methods of `Kernel` and the exported identifiers of `opm/kernel`
- **THEN** neither `Finalize` nor `FinalizeValue` exists
- **AND** no method accepts a second, narrowed components value: the render build reads the imported instance's `components` once, for matching and execution alike

#### Scenario: No raw-load methods on the Kernel

- **WHEN** a consumer inspects the exported methods of `Kernel`
- **THEN** none of `LoadModulePackage`, `LoadPlatformPackage`, `LoadInstancePackage`, `NewModuleFromValue`, `NewPlatformFromValue` exists
- **AND** a caller that wants the raw value of an acquired artifact reads its `Package` field

### Requirement: Each runtime contract has one home
Each runtime contract of the library SHALL be stated in the doc comment of the package, type or function that owns it; the rationale behind a contract SHALL live in an ADR under `adr/`, and its testable obligations in a requirement under `openspec/specs/`. `README.md`, `AGENTS.md` and `docs/getting-started.md` SHALL link to a contract's home instead of restating it, and MAY keep a short orientation sentence that names the home. Source: ADR-013, decision c4.

#### Scenario: The render contract is stated once

- **WHEN** a developer looks for the render gate's cause order outside `opm/`
- **THEN** `README.md`, `AGENTS.md` and `docs/getting-started.md` link to the `opm/kernel` package doc or the `RenderError` doc for it, and none lists the order itself

#### Scenario: The env-override rule is stated in its homes

- **WHEN** a developer searches the non-test Go code under `opm/` for `os.Setenv`
- **THEN** the hits are the `opm/internal/cueenv` package doc and the `OCILoader` type doc; the acquire verbs link `[WithRegistry]` or the `opm/kernel` package doc, and the loader options link `[cueenv.Override]`, instead

#### Scenario: Published doc comments carry no ADR pointer

- **WHEN** a developer searches the doc comments of the exported packages under `opm/` for `ADR-`
- **THEN** none is found; ADR pointers appear only in non-doc comments

#### Scenario: No session-local citation

- **WHEN** a developer runs `git grep -nE '\bSD[0-9]+\b|[Ss]upervisor' -- . ':!openspec/changes'`
- **THEN** it prints nothing; each rule cites a source a reader can open

#### Scenario: A walkthrough decision resolves through ADR-013

- **WHEN** a developer collects every id cited as `ADR-013, decision <id>` or `ADR-013, decisions <id> and <id>` in the committed files outside `openspec/changes/`
- **THEN** each id has exactly one row in `adr/013-kernel-plan-walkthrough-decisions.md`, and that row has a non-empty Landed cell

#### Scenario: No walkthrough id without ADR-013

- **WHEN** a developer runs `git grep -nE "[Oo]wner('s)? (walkthrough )?([Dd]ecisions?|[Tt]asks?|[Ii]tems?) [a-j][1-5]\b|walkthrough ([Dd]ecisions?|[Tt]asks?|[Ii]tems?) [a-j][1-5]\b|(walkthrough|checklist) items? [a-j][1-5]\b|(beta\.1|kernel[ -]plan) walkthrough|owner decision 2026-10-0[23]|[Oo]wner('s)?,? [a-j][1-5]\b|(walkthrough|kernel[ -]plan),? \(?[a-j][1-5]\b" -- . ':!openspec/changes' ':!adr/013-*'` and `git grep -nE "\b([Dd]ecisions?|[Tt]asks?|[Ii]tems?) [a-j][1-5]\b" -- . ':!openspec/changes' ':!adr/013-*' | sed -E 's/ADR-013, decisions? [a-j][1-5]((, | and )[a-j][1-5])*//g' | grep -E "\b([Dd]ecisions?|[Tt]asks?|[Ii]tems?) [a-j][1-5]\b"`
- **THEN** both print nothing

### Requirement: Instance verbs refuse an unset required config value
`SynthesizeInstance` and `AcquireInstanceFromDir` SHALL refuse an instance whose effective values leave a required `#config` value unset, whether or not a component reads that value. Source: library#211, settled by the owner in the pull request that closes it; opm-operator#258 for the report of both checks.

#### Scenario: Synthesis refuses an unread required value

- **WHEN** `SynthesizeInstance` is called for a module whose `#config` declares `replicas: int | *1` and `image: string`, whose only component reads `replicas`, with one values source `replicas: 2`
- **THEN** the call returns no instance and an error framed `Kernel.SynthesizeInstance: instance "<name>": not fully concrete: `
- **AND** the error has a finding at the path `values.image` that carries the position of the `image` declaration in `#config`

#### Scenario: Directory acquisition refuses an unread required value

- **WHEN** `AcquireInstanceFromDir` is called on an instance package of that module whose own `values` set `replicas` and leave `image` unset, once with no sources and once with a source that sets an unrelated declared field
- **THEN** both calls return no instance and an error framed `Kernel.AcquireInstanceFromDir: instance "<name>": not fully concrete: ` with a finding at the path `values.image`

#### Scenario: A required field marker is refused when absent

- **WHEN** either verb is given values that omit a `#config` field declared `tag!: string` that no component reads
- **THEN** the call is refused with CUE's required-field error at the path `values.tag`, positioned at the `tag` declaration

#### Scenario: A field declared `_` is named by its path

- **WHEN** either verb is given values that omit a `#config` field declared `any: _` that no component reads
- **THEN** the call is refused with a finding at the path `values.any`

#### Scenario: Optional and defaulted fields are not required

- **WHEN** either verb is given values that leave unset only `#config` fields declared optional (`opt?: string`) or with a default (`replicas: int | *1`)
- **THEN** the call returns the instance

#### Scenario: The refused set matches ValidateConfigDetailed

- **WHEN** the same module and the same single values source are given to `ValidateConfigDetailed(mod.ConfigSchema(), []Source{src})` and to `SynthesizeInstance`, over values that set every field, leave an unread `string` field unset, leave an unread `foo!` field unset, leave an unread `_` field unset, leave an optional field unset, leave a defaulted field unset, leave a field a component reads only through a hidden field unset, set a key `#config` does not declare, set an unread field to a value of the wrong type, violate a constraint on such a hidden-read field, and the empty document `{}`
- **THEN** `SynthesizeInstance` refuses every case `ValidateConfigDetailed` refuses
- **AND** for a source that gives a defaulted field a bare type (`port: int` where `#config` declares `port: int | *80`), `ValidateConfigDetailed` accepts and `SynthesizeInstance` refuses, through the built-spec check

#### Scenario: Instances refused before keep their error

- **WHEN** an instance package's own `values` carry a non-concrete value (for example `replicas: >=1`) that the built spec exposes
- **THEN** `AcquireInstanceFromDir` refuses it with the same `not fully concrete` error the built-spec check reported before this requirement: a finding at the path `values.replicas` positioned in the package's own values file, and no finding positioned at the module's `#config` declaration

#### Scenario: A value the built spec refuses is reported once

- **WHEN** either verb is given values for a module whose `#config` declares `image: string`, `tag!: string` and `any: _`, none of them read by a component, where a source writes `image: string` and leaves `tag` and `any` unset
- **THEN** the call is refused with exactly one finding at the path `values.image`, the built-spec check's, positioned where the values carry it
- **AND** after it one finding each at `values.tag` and `values.any`, and no other finding

#### Scenario: A value a component reads is named at its field

- **WHEN** either verb is given values that leave unset a `#config` field `message: string` that a component reads into two of its regular fields, one of them through a string interpolation
- **THEN** the call is refused with exactly one finding, at the path `values.message`, positioned at the `message` declaration in `#config`
- **AND** no finding is at a `components` path or at a path of the module the instance unifies

#### Scenario: A value a catalog resource reads is named at its field

- **WHEN** `SynthesizeInstance` is given values that leave unset a `#config` field `note: string` of a module whose component is a catalog resource that reads `note` into three of the resource's fields
- **THEN** the call is refused with exactly one finding, at the path `values.note`, positioned at the `note` declaration in `#config`
- **AND** with the empty document the refusal has one finding for each required value and none at a `components` path

#### Scenario: Every unset value is named, read or not, nested or not

- **WHEN** `SynthesizeInstance` is given the empty document for a module whose `#config` declares `message: string` and `db: host: string`, both read by a component, `other: int`, read by none, and `port: int & >0 | *80`
- **THEN** the refusal has exactly three findings, at `values.message`, `values.other` and `values.db.host`, each positioned at its `#config` declaration
- **AND** the three paths are the fields `ValidateConfigDetailed` names at `#config.<field>` for the same source

#### Scenario: A component defect waits for complete values

- **WHEN** either verb is given values that leave `message` unset for a module whose component reads `message` and also declares a regular field `loose: string` that reads no `#config` value
- **THEN** the refusal has exactly one finding, at `values.message`
- **AND** with every value set, the refusal is the built-spec finding at the component's `loose` path, with the text it had before this requirement changed

#### Scenario: A written bare type hides the component findings it causes

- **WHEN** `SynthesizeInstance` is given a source that writes `message: string` for a field a component reads, with every other required value set
- **THEN** the refusal has exactly one finding, at `values.message`, positioned where the values carry it

#### Scenario: Other values defects keep their text

- **WHEN** `SynthesizeInstance` is given, for a module whose required values are otherwise all set, a value of the wrong type, a value that violates a constraint, or a key `#config` does not declare
- **THEN** each refusal has the text it had before this requirement changed, and none is framed `not fully concrete`

### Requirement: The Kernel is one struct that only kernel.New constructs
The library SHALL expose a `Kernel` struct in `opm/kernel/` that serves as the single public anchor type for the OPM kernel runtime. The struct SHALL be constructible only via the `kernel.New(opts ...Option)` function. Construction SHALL create no `cue.Context` and perform no CUE evaluation.

#### Scenario: Default construction

- **WHEN** a caller invokes `kernel.New()` with no options
- **THEN** a non-nil `*Kernel` is returned holding a schema cache backed by the default OCI loader reading the process environment
- **AND** the Kernel holds no `cue.Context` and has evaluated nothing

### Requirement: The schema cache is backed by the OCI loader of the kernel's registry mapping
Absent `WithSchemaLoader`, the schema cache SHALL be backed by an OCI loader whose registry mapping is the kernel's `WithRegistry` value (empty when the option is absent), so the schema and every other kernel operation resolve through one mapping. An explicit `WithSchemaLoader` SHALL take precedence regardless of option order.

#### Scenario: Registry option seeds the schema loader

- **WHEN** a caller invokes `kernel.New(WithRegistry(mapping))` with no `WithSchemaLoader`
- **THEN** the first schema-touching call resolves the core schema through `mapping`
- **AND** the process environment is not mutated

#### Scenario: Construction with options

- **WHEN** a caller invokes `kernel.New(WithSchemaLoader(myLoader), WithRegistry(mapping))` in either order
- **THEN** the returned Kernel resolves the core schema through `myLoader` and uses `mapping` for catalog and module resolution

### Requirement: The Kernel holds no cue.Context
The Kernel SHALL hold no `cue.Context`. The only long-lived evaluation state the Kernel owns is its schema cache, which holds a private context no accessor exposes.

#### Scenario: No context accessor

- **WHEN** a consumer inspects the exported methods of `Kernel`
- **THEN** no method returns or accepts a `*cue.Context`

### Requirement: Each build operation creates and drops its own cue.Context
Every kernel operation that builds an artifact (module, platform and instance acquisition, synthesis and render) SHALL create its own `cue.Context`, build in it, and let it go when the operation returns; validation (`ValidateConfigDetailed`) SHALL compile its sources in the context of the schema value it validates against, so the two unify in one runtime.

#### Scenario: Repeated acquisitions retain nothing

- **WHEN** a long-lived Kernel acquires the same platform directory twenty times and the caller drops each result
- **THEN** the process heap after the twentieth acquisition is within noise of the heap after the first, and no accessor on the Kernel reaches a built value

### Requirement: Returned values keep their operation's runtime alive while the caller holds them
The values an operation returns (an artifact's `Package`, a validated value) SHALL keep that operation's runtime alive for exactly as long as the caller holds them.

#### Scenario: A held value keeps its operation's runtime alive

- **WHEN** a caller holds an artifact's `Package` or a validated value
- **THEN** the value keeps that operation's runtime alive for exactly as long as the caller holds it

### Requirement: No kernel method takes a cue.Value as a values input or returns one to unify across calls
No kernel method SHALL take a `cue.Value` from the caller as a values input (the schema `ValidateConfigDetailed` validates against is the one `cue.Value` a caller passes in), and no kernel method SHALL return a value the caller is expected to unify with a value from another kernel call.

#### Scenario: Artifacts cross Kernels

- **WHEN** a module is acquired with one Kernel, synthesized into an instance with a second, and rendered with a third against a platform acquired with a fourth
- **THEN** the render produces the same objects as the same sequence on one Kernel

### Requirement: A returned value keeps its operation's context alive as long as the caller holds it
The operation and its `cue.Context` are those of requirement "Goroutine Safety Contract". A value the operation returns (an artifact's `Package`, a `*kernel.Compiled`) SHALL keep that context alive for as long as the caller holds it, and no longer: the context's lifetime is bounded by its holder (ADR-007).

#### Scenario: A held value keeps its context alive

- **WHEN** a caller holds an artifact's `Package` or a `*kernel.Compiled`
- **THEN** the value keeps the context of its operation alive for as long as the caller holds it, and no longer

### Requirement: Kernel.Render shares nothing between renders
`Kernel.Render` SHALL share nothing between renders: each render is its own CUE build in a fresh `cue.Context` whose references the kernel drops when `Render` returns, and no built value is retained by the kernel. A `*kernel.Compiled` the caller holds keeps its render's build alive until the caller releases it.

#### Scenario: A render shares nothing with another render

- **WHEN** a caller invokes `Render` twice on one Kernel
- **THEN** each render is its own CUE build in a fresh `cue.Context` whose references the kernel drops when `Render` returns, and no built value is retained by the kernel

### Requirement: Concurrency is across operations, never within one
Concurrency is across operations, never within one; a consumer rendering from several goroutines SHALL call `Render` on one Kernel, with no shared materialized platform value and no mutex.

#### Scenario: Several goroutines render on one Kernel

- **WHEN** a consumer renders from several goroutines
- **THEN** it calls `Render` on one Kernel, with no shared materialized platform value and no mutex

### Requirement: The package documentation states the concurrency rule and the render pool size
The rule is the one of requirement "Concurrency is across operations, never within one". The package documentation SHALL state this, SHALL NOT present a value built into a shared context, or a shared materialized platform, as a supported shape, and SHALL state that a render pool is sized by memory (about 61 MB plus 7.75 MB per component per concurrent render, 0019 experiment 08) rather than by core count.

#### Scenario: The documentation sizes a render pool by memory

- **WHEN** a developer reads the package documentation
- **THEN** it states that a render pool is sized by memory (about 61 MB plus 7.75 MB per component per concurrent render, 0019 experiment 08) rather than by core count

### Requirement: The retracted shared-platform model is not a supported shape
The retracted shared-materialized-platform model and its mutex stopgap SHALL NOT appear as supported shapes.

#### Scenario: The retracted model does not appear

- **WHEN** a developer reads the package documentation
- **THEN** the retracted shared-materialized-platform model and its mutex stopgap do not appear as supported shapes

### Requirement: Layered validation and instance processing live on the Kernel
The canonical Go implementation of layered values validation and of instance processing (concreteness assertion and metadata decoding on a built instance spec) SHALL live on the `*Kernel` receiver in `opm/kernel/`. Instance processing SHALL be kernel-internal: it is reached only through `AcquireInstanceFromDir` and `SynthesizeInstance` and is not an exported method.

#### Scenario: ValidateConfigDetailed is a kernel method

- **WHEN** a caller invokes `k.ValidateConfigDetailed(schema, sources)`
- **THEN** the method unifies the sources in order, then validates the merged value with concreteness enforced and returns the merged `cue.Value` plus a CUE-native error
- **AND** no `opm/validate/` or `opm/helper/values/` import is required by callers

#### Scenario: ValidateConfig is a kernel method

- **WHEN** a consumer inspects the exported methods of `Kernel`
- **THEN** `ValidateConfig` does not exist; `ValidateConfigDetailed` is the one validation method and runs the full Tier-2 schema validation directly

#### Scenario: ValidateConfigPartial is a kernel method

- **WHEN** a consumer inspects the exported methods of `Kernel`
- **THEN** `ValidateConfigPartial` does not exist
- **AND** the partial-validation pass (type errors, disallowed fields and pattern violations on fields that are set, missing fields not flagged) is an unexported kernel internal used by `AcquireInstanceFromDir` to attribute extra-values errors to their sources

#### Scenario: ProcessModuleInstance is a kernel method

- **WHEN** a consumer inspects the exported methods of `Kernel`
- **THEN** `ProcessModuleInstance` does not exist; instance processing is an unexported kernel internal
- **AND** both `AcquireInstanceFromDir` and `SynthesizeInstance` assert concreteness on the built spec via `spec.Validate(cue.Concrete(true))` (CUE stdlib) and on the values unified with `#config` (see "Instance verbs refuse an unset required config value"), decode instance metadata, and return a `*module.Instance`

### Requirement: No standalone validate or ParseModuleInstance function remains
No standalone `validate.Config` / `validate.ConfigPartial` / `module.ParseModuleInstance` free functions SHALL remain in the library; the `opm/validate/` and `opm/helper/values/` packages SHALL NOT exist.

#### Scenario: opm/validate package is gone

- **WHEN** a developer runs `ls opm/validate/`
- **THEN** the directory does not exist

#### Scenario: opm/helper/values package is gone

- **WHEN** a developer runs `ls opm/helper/values/`
- **THEN** the directory does not exist

#### Scenario: module.ParseModuleInstance free function is gone

- **WHEN** a developer searches `opm/` for `ParseModuleInstance`
- **THEN** no free function and no method with that name exists

### Requirement: LoadSourceFromFile reads and parses a file
The `*Kernel.LoadSourceFromFile(path string)` method SHALL read the file at `path`, parse it for syntax, and return a `Source` whose `Data` is the file's bytes and whose `Origin` is the file's absolute path; `Source` carries no other label. The method SHALL NOT evaluate CUE.

#### Scenario: Syntax errors fail at load, schema errors at use

- **WHEN** a caller invokes `k.LoadSourceFromFile(path)` on a file that does not parse as CUE
- **THEN** the call returns an error positioned at the file's path and no `Source`
- **AND** a file that parses but violates a module's `#config` loads fine and fails, positioned at the file's path, in the operation that applies it

### Requirement: A compiled file source unwraps a top-level values field
The source is that of requirement "LoadSourceFromFile reads and parses a file". When a kernel operation compiles that source, it SHALL load it through cue/load at the file's directory (so imports resolve as they do for any CUE file), evaluate it in that operation's own context, and:
- If the evaluated value contains a top-level `values:` field whose `Exists()` is true and `Err()` is nil, the compiled source SHALL be that field.
- Otherwise the compiled source SHALL be the whole evaluated value.

#### Scenario: Values file is auto-unwrapped

- **WHEN** a caller invokes `k.LoadSourceFromFile("./values.cue")` against a file containing `values: { foo: "bar" }` and passes the source to `ValidateConfigDetailed`
- **THEN** the value validated is the inner `{ foo: "bar" }` value
- **AND** `Source.Origin` is the absolute path of `values.cue`

#### Scenario: File without values field passes through

- **WHEN** a caller invokes `k.LoadSourceFromFile("./flat.cue")` against a file with no top-level `values:` field and passes the source to `ValidateConfigDetailed`
- **THEN** the whole file value is validated
- **AND** `Source.Origin` is populated as above

### Requirement: LoadSourceFromFile does not depend on loaderfile.LoadValuesFile
The `*Kernel.LoadSourceFromFile(path string)` method SHALL NOT depend on `loaderfile.LoadValuesFile` (which is removed).

#### Scenario: The method does not depend on LoadValuesFile

- **WHEN** a developer inspects `Kernel.LoadSourceFromFile`
- **THEN** it does not depend on `loaderfile.LoadValuesFile`, which is removed

### Requirement: WithRegistry sets the mapping for the render build, registry acquisition and directory acquisition
The `Kernel` SHALL accept a `WithRegistry(string)` option that sets the one OCI registry mapping every kernel operation uses for catalog, module and schema resolution: the render build's catalog imports (`Render`), registry module acquisition (`AcquireModuleFromRegistry`), directory acquisition (`AcquireModuleFromDir`, `AcquirePlatformFromDir`, `AcquireInstanceFromDir`).

#### Scenario: Registry option used for resolution

- **WHEN** `kernel.New(WithRegistry("opmodel.dev=ghcr.io/open-platform-model"))` is called and `Render` runs against a platform whose `cue.mod` names a catalog under `opmodel.dev`
- **THEN** the catalog import resolves through that mapping
- **AND** the process environment is not mutated

#### Scenario: Directory acquisition uses the kernel mapping

- **WHEN** a kernel constructed with `WithRegistry(mapping)` acquires a platform or instance from a directory whose imports resolve from `opmodel.dev`
- **THEN** those imports resolve through `mapping` with no per-call argument

### Requirement: WithRegistry sets the mapping for synthesis, file-backed values sources and the default schema cache
The `Kernel` SHALL accept a `WithRegistry(string)` option that sets the one OCI registry mapping every kernel operation uses for catalog, module and schema resolution: instance synthesis (`SynthesizeInstance`), the compilation of file-backed values sources on every path that accepts `Source` values (`ValidateConfigDetailed`, `AcquireInstanceFromDir` with trailing values, `SynthesizeInstance`), and the default schema cache.

#### Scenario: Values source compilation uses the kernel mapping

- **WHEN** a kernel constructed with `WithRegistry(mapping)` compiles a file-backed `Source` whose file imports a package that only `mapping` routes
- **THEN** the import resolves through `mapping` on `ValidateConfigDetailed`, on `AcquireInstanceFromDir` with that source as a trailing value, and on `SynthesizeInstance` with it in `InstanceInput.Values`
- **AND** the process environment is not mutated

### Requirement: No acquire verb takes a per-call registry override
No acquire verb SHALL take a per-call registry override.

#### Scenario: No per-call registry parameter

- **WHEN** a consumer inspects the signatures of the acquire verbs and `SynthesizeInstance`
- **THEN** none takes a load-options or registry argument, and no `LoadOptions` type is exported from `opm/`

### Requirement: Absent WithRegistry the kernel inherits CUE_REGISTRY and applies no default
Absent the `WithRegistry(string)` option, the kernel SHALL inherit `CUE_REGISTRY` from the process environment and SHALL NOT auto-apply a built-in default registry.

#### Scenario: No default applied

- **WHEN** `kernel.New()` is called with no registry option
- **THEN** the kernel inherits the process `CUE_REGISTRY`
- **AND** applies no built-in default mapping

### Requirement: WithRegistry mutates no process environment
The `WithRegistry(string)` option MUST NOT mutate process environment state; the mapping is plumbed into each operation's load configuration.

#### Scenario: The option mutates no process environment

- **WHEN** a caller constructs a kernel with `WithRegistry`
- **THEN** the process environment is not mutated and the mapping is plumbed into each operation's load configuration

### Requirement: SynthesizeInstance takes an InstanceInput
The `*Kernel` type SHALL expose a method `SynthesizeInstance(ctx context.Context, in InstanceInput) (*module.Instance, error)`, where `InstanceInput` is declared in `opm/kernel` and carries `Module *module.Module` (required, source-carrying), `Name string` (required), `Namespace string` (required), `Values []Source` (optional; empty means "no values supplied"), `Labels map[string]string` and `Annotations map[string]string` (optional).

#### Scenario: SynthesizeInstance produces an end-to-end instance

- **WHEN** `k.SynthesizeInstance(ctx, kernel.InstanceInput{Module: mod, Name: "demo", Namespace: "default", Values: []kernel.Source{concrete}})` is called against an acquired module
- **THEN** the returned `*module.Instance` is non-nil
- **AND** `Instance.Metadata.Name` equals `"demo"`, `Instance.Metadata.Namespace` equals `"default"`
- **AND** `Instance.Metadata.UUID` equals `uuid.SHA1(OPMNamespace, "<module.uuid>:demo:default")`
- **AND** `Instance.Source` is non-nil

### Requirement: SynthesizeInstance builds the instance in one build and decodes its metadata
`Kernel.SynthesizeInstance` SHALL build the instance spec by single-build CUE evaluation inside the module's staged source with the unified values rendered into the package, check the values sources against the module's `#config` at their own positions after the build, assert concreteness on the built spec, decode instance metadata, and return the constructed `*module.Instance` with its `Source` populated.

#### Scenario: SynthesizeInstance rejects unconcrete result

- **WHEN** `k.SynthesizeInstance(ctx, in)` is called with empty `in.Values` against a module whose `#config` has required fields with no defaults
- **THEN** the returned error is non-nil, is framed `instance "<name>": …`, and wraps the concreteness diagnostic

### Requirement: The synthesized package imports core at the major of the kernel's schema release
The synthesized package SHALL import `core` at the major of the kernel's schema release (read from the loader's pin with no schema load when it names an exact release, resolved through the schema cache otherwise); the release that import resolves to comes from the module's own `cue.mod/module.cue`.

#### Scenario: The synthesized package imports the schema release's major

- **WHEN** a module is synthesized
- **THEN** the synthesized package imports `core` at the major of the kernel's schema release, and the release that import resolves to comes from the module's own `cue.mod/module.cue`

### Requirement: The synthesis build runs in a context the method creates and releases
The `SynthesizeInstance` build SHALL run in a context the method creates and releases; `SynthesizeInstance` SHALL NOT consult any additional values source.

#### Scenario: SynthesizeInstance uses the Kernel's cue.Context

- **WHEN** `k.SynthesizeInstance(ctx, in)` is called twice on one Kernel
- **THEN** the Kernel has no `cue.Context` to use: the two returned instances' `Package` values belong to two distinct contexts the method created, neither reachable through the Kernel

#### Scenario: Operator-shaped caller needs no cue.Context

- **WHEN** a frontend holds raw values bytes and their origin
- **THEN** it builds the input with `k.LoadSourceFromBytes(origin, raw)` and passes the result in `InstanceInput.Values`
- **AND** it needs no `cue.Context` of its own, since the kernel exposes none

### Requirement: A failed synthesis build is built again without the rendered values file
When the build fails and the call's merged values exist, `SynthesizeInstance` SHALL attribute the failure to the values sources the way `AcquireInstanceFromDir` does: it SHALL build the synthesized package again in the same context without the rendered values file, take `#config` from that build (never from the module's `Package`).

#### Scenario: SynthesizeInstance attributes a values conflict that fails the build

- **WHEN** `k.SynthesizeInstance(ctx, in)` is called with a source (Origin `/values/bad.cue`) whose value violates the module's `#config` at a path a component of the module consumes, so the synthesized build fails
- **THEN** the returned error reads `Kernel.SynthesizeInstance: instance "<name>": …`, names the violating path, and at least one of its positions reports `/values/bad.cue`
- **AND** no instance is returned

### Requirement: A failed synthesis build validates the compiled values against the values-free #config
The values-free build and its `#config` are those of requirement "A failed synthesis build is built again without the rendered values file". When the build fails and the call's merged values exist, `SynthesizeInstance` SHALL attribute the failure to the values sources the way `AcquireInstanceFromDir` does: it SHALL validate the values it already compiled against it without concreteness (a field the values leave unset is not a conflict; concreteness is enforced on a build that succeeds).

#### Scenario: A failed build validates the compiled values without concreteness

- **WHEN** a synthesis build fails and the call's merged values exist
- **THEN** the values it already compiled are validated against the `#config` of the values-free build without concreteness

### Requirement: A values error is framed and names each violating source
A values error SHALL be returned framed `Kernel.SynthesizeInstance: instance "<name>": …`, with positions that name each violating source's `Origin`.

#### Scenario: SynthesizeInstance attributes a values violation to its source

- **WHEN** `k.SynthesizeInstance(ctx, in)` is called with a source whose value violates the module's `#config`
- **THEN** the returned error names the violating path and its positions report the source's `Origin`

### Requirement: Every other build failure is returned unchanged
In every other case (no values, the values-free build also fails, or the values are clean) `SynthesizeInstance` SHALL return the original build error unchanged.

#### Scenario: A build failure with clean values is returned unchanged

- **WHEN** `k.SynthesizeInstance(ctx, in)` is called with values that satisfy the module's `#config` against a module whose own component fails to evaluate
- **THEN** the returned error is the build error framed `Kernel.SynthesizeInstance: …`, not framed `instance "<name>": …`, and no instance is returned

#### Scenario: Incomplete values do not mask a build failure

- **WHEN** `k.SynthesizeInstance(ctx, in)` is called with values `replicas: 2` against a module whose `#config` also has a required `image: string` and whose component requires `#config.replicas & >5`, so the build fails
- **THEN** the returned error is the build error framed `Kernel.SynthesizeInstance: …`, not a missing-field error framed `instance "<name>": …`, and no instance is returned

### Requirement: A missing input fails with an opm/errors sentinel
A missing required input SHALL fail with an error wrapping the corresponding `opm/errors` sentinel (`ErrMissingModule`, `ErrMissingName`, `ErrMissingNamespace`); a module without staged source SHALL fail wrapping `ErrMissingSource`; a module whose `Source.Pkg` is non-empty SHALL fail with an error stating that a synthesizable module is its module's root package.

#### Scenario: SynthesizeInstance surfaces synth errors before validation

- **WHEN** `k.SynthesizeInstance(ctx, kernel.InstanceInput{Module: nil, Name: "x", Namespace: "y"})` is called
- **THEN** the returned error wraps `oerrors.ErrMissingModule` and no build runs

#### Scenario: SynthesizeInstance refuses a subpackage module

- **WHEN** `k.SynthesizeInstance` is called with a module acquired from a subdirectory of its CUE module (`Source.Pkg` non-empty)
- **THEN** the returned error states that the module must be its module's root package and no build runs

### Requirement: The kernel validates non-empty values against #config where they are applied
When values are non-empty, the kernel SHALL validate them against the Module's `#config` schema at the point they are applied to the instance, regardless of whether a Tier-1 helper validated them upstream.

#### Scenario: Kernel validates without Detailed

- **WHEN** a frontend skips `ValidateConfigDetailed` and supplies raw sources to `AcquireInstanceFromDir` or `SynthesizeInstance` directly
- **THEN** the kernel still produces correct schema-validation errors from the build, attributed to the sources' `Origin`

### Requirement: Both instance verbs check the sources against #config at their own positions
Values are applied inside a CUE build: `AcquireInstanceFromDir` renders the unified sources into the package overlay, and `SynthesizeInstance` renders them into the synthesized package; both SHALL then check the sources against `#config` so a violation is reported at the sources' own positions, and both assert concreteness on the whole built spec and on the values unified with `#config` (see "Instance verbs refuse an unset required config value").

#### Scenario: Kernel re-validates after Detailed

- **WHEN** a frontend validates sources with `k.ValidateConfigDetailed` and then supplies the same sources to `k.AcquireInstanceFromDir` or in `InstanceInput.Values`
- **THEN** the kernel validates them again where they are applied: the build unifies them with the package's `values`, and the sources are checked against `#config` at their own positions
- **AND** any schema violation produces a CUE-native error walkable via `cueerrors.Errors`, wrapped with the instance name, whose positions name the originating source's `Origin` rather than the rendered values file

### Requirement: Render performs no second validation pass
`Kernel.Render` SHALL NOT perform a second validation pass: the render build imports the instance as processed, which is already concrete.

#### Scenario: Render does not re-validate

- **WHEN** a caller invokes `k.Render` with an instance returned by `AcquireInstanceFromDir` or `SynthesizeInstance`
- **THEN** no `#config` validation runs inside `Render`; the staged instance package is imported by the build as it was processed

### Requirement: User values reach the kernel through exactly three public inputs
User values SHALL reach the kernel through exactly three public inputs, each carrying values as `Source` values: `ValidateConfigDetailed(schema, sources []Source)` for layered validation; `AcquireInstanceFromDir(ctx, dir, values ...Source)`, which unifies the sources inside the package build; and `SynthesizeInstance(ctx, in)` with `in.Values []Source`, which unifies the sources and renders the result into the synthesized package.

#### Scenario: A single pre-unified value is a one-element source list

- **WHEN** a caller holds a single pre-unified `cue.Value` to validate or apply
- **THEN** it passes it as a one-element `[]Source` (or one variadic `Source`) to the relevant entry
- **AND** there is no internal merge loop for a single source; the value is consumed as-is

#### Scenario: ValidateConfigDetailed takes a slice of Source

- **WHEN** a caller invokes `k.ValidateConfigDetailed(schema, sources)` with `sources` as `[]Source`
- **THEN** the method unifies the sources in order then validates the merged value against `schema`

#### Scenario: Extra values enter the instance build

- **WHEN** a caller invokes `k.AcquireInstanceFromDir(ctx, dir, a, b)`
- **THEN** the sources are unified and rendered into the package overlay so the schema's own values unification performs the merge in CUE
- **AND** no Go code fills the merged value into the evaluated instance

#### Scenario: Synthesized values are sources

- **WHEN** a caller invokes `k.SynthesizeInstance(ctx, in)` with `in.Values` holding one or more `Source` values
- **THEN** the sources are unified in order, the result is rendered into the synthesized package's values source and participates in the single build
- **AND** the method does not accept a bare `cue.Value`

#### Scenario: Empty values is the zero value

- **WHEN** a caller passes an empty `[]Source` to `k.ValidateConfigDetailed`, no trailing sources to `k.AcquireInstanceFromDir`, or an empty `in.Values` to `k.SynthesizeInstance`
- **THEN** the call succeeds with no values applied (the package or synthesized spec must already be concrete on every required field)
- **AND** the behavior is documented as "no values supplied"

### Requirement: The kernel accepts no cue.Value as a values argument
The kernel SHALL NOT accept `[]cue.Value` or a bare `cue.Value` as a values argument on any public method, and SHALL NOT expose a method that fills values into an already-built instance from Go.

#### Scenario: ProcessModuleInstance takes a single value

- **WHEN** a consumer inspects the exported methods of `Kernel`
- **THEN** `ProcessModuleInstance` does not exist
- **AND** values reach an instance only through the variadic sources of `AcquireInstanceFromDir` or `InstanceInput.Values` on `SynthesizeInstance`

### Requirement: The kernel retains no built value between calls
The kernel SHALL retain no built value between calls. Each `*kernel.Compiled` that `Render` returns carries a `cue.Value` into that render's build, so a caller that holds one keeps the build alive until it releases it; retention is bounded by what the caller holds, the rule ADR-007 states for an acquired artifact's `Package`.

#### Scenario: The kernel retains no built value

- **WHEN** a caller renders and holds the returned `*kernel.Compiled`
- **THEN** the kernel retains no built value between calls, and the held `*kernel.Compiled` keeps its render's build alive until the caller releases it

### Requirement: ADR-005 and the package documentation state the holder-bounded rule
Each `*kernel.Compiled` that `Render` returns carries a `cue.Value` into that render's build, so a caller that holds one keeps the build alive until it releases it; retention is bounded by what the caller holds, the rule ADR-007 states for an acquired artifact's `Package`. ADR-005 and the `opm/kernel` package documentation SHALL state this rule and SHALL NOT state that a caller cannot obtain a built value to hold.

#### Scenario: The documentation states the holder-bounded rule

- **WHEN** a developer reads rule 2 of `adr/005-shares-nothing-renders.md` and the Goroutine safety section of the `opm/kernel` package documentation
- **THEN** both state that the kernel holds no built value between calls and that a held `*Compiled` keeps its render's build alive until the caller releases it
- **AND** neither states that a caller cannot obtain a built value to hold

### Requirement: ADR-005 records that Render output moves to bytes before GA
ADR-005 SHALL record that Render output moves from `cue.Value` to bytes before GA at the latest. The bytes deadline is enhancement 0021 ("A beta line reaches GA only when its exit criteria hold"); the retention rule is ADR-007 rule 1.

#### Scenario: The move to bytes is recorded

- **WHEN** a developer reads the Status of `adr/005-shares-nothing-renders.md`
- **THEN** it states that Render output keeps `Compiled.Value` for now and moves to bytes before GA at the latest

### Requirement: The package documentation renders its verb list and examples as a list and code blocks
The `opm/kernel` package documentation SHALL present the `Surface` section's verbs as a Go doc list, with one item per verb group, and SHALL present each code example (the one-Kernel-per-process `renderAll` example, the diagnostics wording example and the replacements wording example) as a Go doc code block.

#### Scenario: go doc renders the Surface list as a list

- **WHEN** a developer runs `go doc ./opm/kernel`
- **THEN** each verb group under `Surface` (the module, catalog, platform and instance acquire verbs, `SynthesizeInstance`, `ValidateConfigDetailed` and `Render`) is printed as its own list item

#### Scenario: go doc renders each example as code

- **WHEN** a developer runs `go doc ./opm/kernel`
- **THEN** the `renderAll` example, the diagnostics wording example and the replacements wording example are printed as indented code with one statement per line, not as wrapped prose

### Requirement: A test parses the package documentation and fails when a list or code block is flattened
When parsed with `go/doc` and `go/doc/comment`, the package doc SHALL yield a `*comment.List` after the `Surface` heading and one `*comment.Code` block per example. A test in `opm/kernel` SHALL parse the package doc this way and SHALL fail when the list or any of the code blocks stops being one.

#### Scenario: Flattening the Surface list fails the guard test

- **WHEN** a change rewraps the `Surface` list in `opm/kernel/doc.go` into a single paragraph
- **THEN** `go test ./opm/kernel` fails, naming the `Surface` list and the block type it found instead

#### Scenario: Flattening a code example fails the guard test

- **WHEN** a change rewraps any of the three code examples in `opm/kernel/doc.go` into prose
- **THEN** `go test ./opm/kernel` fails, naming the missing code block

#### Scenario: Flattening one loop of an example fails the guard test

- **WHEN** a change rewraps a single loop of a code example in `opm/kernel/doc.go` into prose and leaves the rest of the example as code
- **THEN** `go test ./opm/kernel` fails, naming the code fragment it found outside a code block

### Requirement: The documentation names SynthesizeInstance as the typed-input entry point
The package documentation and the `Kernel.SynthesizeInstance` godoc SHALL state that `SynthesizeInstance` is the entry point for building an instance from typed inputs, mirroring `Kernel.AcquireInstanceFromDir` for a directory-based CUE package, and that the module it takes comes from `AcquireModuleFromRegistry` or `AcquireModuleFromDir`.

#### Scenario: Documentation directs callers to the kernel method

- **WHEN** a developer reads the godoc on `opm/kernel`
- **THEN** the documentation states that `Kernel.SynthesizeInstance` is the entry point for typed-input synthesis, names the two acquire verbs that produce its module, and states that the synthesized package imports `core` at the major of the kernel's schema release while the release that import resolves to comes from the module's own `cue.mod/module.cue`
- **AND** no reference to `synth.Instance`, `opm/helper/synth` or `Kernel.LoadInstancePackage` remains

#### Scenario: SynthesizeInstance godoc names its directory mirror and module sources

- **WHEN** a developer reads the `Kernel.SynthesizeInstance` godoc
- **THEN** the directory-driven mirror it names is `Kernel.AcquireInstanceFromDir`, and the module sources it names are `AcquireModuleFromRegistry` and `AcquireModuleFromDir`
- **AND** no reference to `Kernel.LoadInstancePackage`, `Kernel.ProcessModuleInstance` or `synth.Instance` remains

### Requirement: The documentation states which core release the synthesized package imports
The package documentation and the `Kernel.SynthesizeInstance` godoc SHALL state that the synthesized package imports `core` at the major of the kernel's schema release (read from the configured loader without a schema load when that loader pins an exact release, and resolved through the kernel's schema cache otherwise), and that the release that import resolves to comes from the module's own `cue.mod/module.cue`.

#### Scenario: Pinned kernel synthesizes without touching the schema cache

- **WHEN** a frontend constructs a kernel with the default loader and synthesizes an instance
- **THEN** no schema load runs for the synthesis; the module's own dependency list resolves `core` inside the build

### Requirement: The documentation presents no helper-level composition
The package documentation and the `Kernel.SynthesizeInstance` godoc SHALL NOT present a helper-level composition that reaches synthesis or instance processing directly, since neither is exported.

#### Scenario: No helper-level composition is presented

- **WHEN** a developer reads the package documentation and the `Kernel.SynthesizeInstance` godoc
- **THEN** neither presents a helper-level composition that reaches synthesis or instance processing directly, since neither is exported

### Requirement: CONSTITUTION Principle III lists the Kernel verbs as built
Where `CONSTITUTION.md` Principle III lists the `Kernel` verbs, it SHALL list them as acquire, synthesize, validate and render; where it describes `opm/module/`, it SHALL describe the module and instance model. None of these SHALL name a load or process verb, a separate schema-validation stage, or a package that does not exist.

#### Scenario: Principle III names the verbs as built

- **WHEN** a developer reads the `opm/kernel/` and `opm/module/` bullets of `CONSTITUTION.md` Principle III
- **THEN** the verbs read acquire, synthesize, validate and render, with no load or process verb
- **AND** `opm/module/` is described as the module and instance model

### Requirement: A front-door pipeline ends in kernel.Compiled
Where `CONSTITUTION.md` shows a pipeline, the pipeline SHALL show acquisition or synthesis producing the artifacts and one render build over an instance and a platform, ending in `[]*kernel.Compiled`, and SHALL NOT send a `Catalog` into render. None of these SHALL name a load or process verb, a separate schema-validation stage, or a package that does not exist.

#### Scenario: The pipeline ends in kernel.Compiled

- **WHEN** a developer reads the pipeline block of `CONSTITUTION.md` Principle III
- **THEN** it runs from acquire or synthesize to one render build over an instance and a platform, ending in `[]*kernel.Compiled`
- **AND** it names no `core` package, no separate schema-validate stage, and no `Catalog` entering render

### Requirement: The README layout lists only packages that exist
Where `README.md` carries a layout block, it SHALL list only packages that exist under `opm/`, and SHALL NOT place `Compiled` outside `opm/kernel`; the front-door files MAY link to this spec instead of restating it.

#### Scenario: The README layout names only real packages

- **WHEN** a developer compares the `README.md` layout block with the directories under `opm/`
- **THEN** every listed package exists, no `opm/core/` row appears, and `Compiled`, where named, sits on the `kernel/` row

### Requirement: Verbs check their context between stages
`Kernel.AcquireModuleFromDir`, `Kernel.AcquireCatalogFromDir`, `Kernel.AcquirePlatformFromDir`, `Kernel.AcquireInstanceFromDir` and `Kernel.SynthesizeInstance` SHALL check their context after their argument checks and between their stages: after the directory is read, after the values sources are merged. Source: enhancement 0009 ("The execution half owns cancellation inside a stage and introduces its own injection surface"), as revised on 2026-10-03.

#### Scenario: A cancelled context stops each directory verb

- **WHEN** `AcquireModuleFromDir`, `AcquireCatalogFromDir`, `AcquirePlatformFromDir` or `AcquireInstanceFromDir` is called with a context that is already cancelled and a valid directory
- **THEN** it returns `context.Canceled` and a nil artifact

#### Scenario: A cancelled context stops synthesis

- **WHEN** `SynthesizeInstance` is called with a context that is already cancelled and a complete `InstanceInput`
- **THEN** it returns `context.Canceled` and a nil instance

#### Scenario: Argument errors come first

- **WHEN** `SynthesizeInstance` is called with a cancelled context and an `InstanceInput` with no `Module`
- **THEN** the error wraps `oerrors.ErrMissingModule`

### Requirement: Verbs check their context after the build
`Kernel.AcquireModuleFromDir`, `Kernel.AcquireCatalogFromDir`, `Kernel.AcquirePlatformFromDir`, `Kernel.AcquireInstanceFromDir` and `Kernel.SynthesizeInstance` SHALL check their context after their argument checks and between their stages: after the package is built and before the next stage starts. Source: enhancement 0009 ("The execution half owns cancellation inside a stage and introduces its own injection surface"), as revised on 2026-10-03.

#### Scenario: A verb checks its context after the package is built

- **WHEN** `AcquireModuleFromDir`, `AcquireCatalogFromDir`, `AcquirePlatformFromDir`, `AcquireInstanceFromDir` or `SynthesizeInstance` has built the package
- **THEN** it checks its context after the package is built and before the next stage starts

### Requirement: The registry acquire verbs and Render check their context
The registry acquire verbs SHALL check their context after the registry fetch returns and after the package is built, and `Kernel.Render` SHALL check its context after its input checks, after staging and after the render build. Source: enhancement 0009 ("The execution half owns cancellation inside a stage and introduces its own injection surface"), as revised on 2026-10-03.

#### Scenario: Registry verbs and Render check the context

- **WHEN** `Render` runs or a registry acquire verb runs
- **THEN** `Render` checks the context after its input checks, after staging and after the render build, and a registry acquire verb checks it after the registry fetch returns and after the package is built

### Requirement: A cancelled check of a directory or synthesis verb returns the context's own error and no artifact
The checks are those of requirements "Verbs check their context between stages" and "Verbs check their context after the build". When one of these checks finds the context done, the verb SHALL return the context's own error unwrapped (so `errors.Is(err, context.Canceled)` or `errors.Is(err, context.DeadlineExceeded)` holds) and no artifact. Source: enhancement 0009 ("The execution half owns cancellation inside a stage and introduces its own injection surface"), as revised on 2026-10-03.

#### Scenario: A done context returns its own error and no artifact

- **WHEN** one of these checks finds the context done
- **THEN** the verb returns the context's own error unwrapped, so `errors.Is(err, context.Canceled)` or `errors.Is(err, context.DeadlineExceeded)` holds, and no artifact

### Requirement: A cancelled check of a registry acquire verb or Render returns the context's own error and no artifact
The checks are those of requirement "The registry acquire verbs and Render check their context". When one of these checks finds the context done, the verb SHALL return the context's own error unwrapped (so `errors.Is(err, context.Canceled)` or `errors.Is(err, context.DeadlineExceeded)` holds) and no artifact. Source: enhancement 0009 ("The execution half owns cancellation inside a stage and introduces its own injection surface"), as revised on 2026-10-03.

#### Scenario: A done check of a registry acquire verb or Render returns its own error and no artifact

- **WHEN** one of the checks of requirement "The registry acquire verbs and Render check their context" finds the context done
- **THEN** the verb returns the context's own error unwrapped, so `errors.Is(err, context.Canceled)` or `errors.Is(err, context.DeadlineExceeded)` holds, and no artifact

### Requirement: A cancellation inside the registry fetch is still a context error
A cancellation observed inside the registry fetch SHALL still satisfy `errors.Is(err, context.Canceled)`. Source: enhancement 0009 ("The execution half owns cancellation inside a stage and introduces its own injection surface"), as revised on 2026-10-03.

#### Scenario: A fetch served from the cache still observes cancellation

- **WHEN** `AcquireModuleFromRegistry` is called with a cancelled context for a module the local cache already holds, so the fetch returns without a network request
- **THEN** it returns an error for which `errors.Is(err, context.Canceled)` holds, and no module

### Requirement: A running load or build is not interrupted
A running `cue/load` or build SHALL NOT be interrupted; cancellation lands at the next stage boundary. The `opm/kernel` package doc SHALL say so. Cancellation inside a stage is outside this requirement. Source: enhancement 0009 ("The execution half owns cancellation inside a stage and introduces its own injection surface"), as revised on 2026-10-03.

#### Scenario: A cancellation during a stage lands at the next check

- **WHEN** any of these verbs, or `Render`, is called with a context that becomes done at one of its stage checks, the first or any later one
- **THEN** it returns the bare `context.Canceled` and no artifact, even when every stage before that check succeeded

#### Scenario: The godoc states where cancellation lands

- **WHEN** a developer reads `go doc ./opm/kernel`
- **THEN** a Cancellation section says that the context is checked at entry and between stages, and that a running load or build is not interrupted

### Requirement: A published doc comment carries no ADR pointer
A doc comment that the docs bundle publishes (any exported package under `opm/`) SHALL NOT carry an `ADR-NNN` pointer; a package that wants one keeps it in a non-doc comment. Source: ADR-013, decision c4.

#### Scenario: A published doc comment has no ADR pointer

- **WHEN** a doc comment of an exported package under `opm/` is read
- **THEN** it carries no `ADR-NNN` pointer, and a package that wants one keeps it in a non-doc comment

### Requirement: No committed file cites a session-local decision as a source
No committed file outside `openspec/changes/` SHALL cite a session-local decision, one recorded only in an agent run's own log (a numbered decision kept in that log, or an agent role named as the source), as a source. Source: ADR-013, decision c4.

#### Scenario: A session-local decision is not cited as a source

- **WHEN** a committed file outside `openspec/changes/` is read
- **THEN** it cites no decision recorded only in an agent run's own log

### Requirement: A committed file cites a source a reader can open
A committed file outside `openspec/changes/` that cites a source SHALL state the rule and cite a source a reader can open (an ADR; a decision ADR-013 records, cited as `ADR-013, decision <id>`; an enhancement decision such as enhancement 0012 ("The Kubernetes runtime decisions live in a fenced tier, `opm/k8s`, beside the kernel and inside the library module"); a pull request; or an archived change). Source: ADR-013, decision c4.

#### Scenario: A committed file states the rule and cites a source a reader can open

- **WHEN** a committed file outside `openspec/changes/` cites a source
- **THEN** the source is an ADR, a decision ADR-013 records, an enhancement decision, a pull request or an archived change

### Requirement: A Kernel hands its registry client to registry acquisition
A `Kernel` SHALL own one registry client, meaning the resolver and the OCI transport behind it and not a module cache, and SHALL hand it to every module load and registry fetch it runs for its own operations: registry acquisition (`AcquireModuleFromRegistry`, `AcquireCatalogFromRegistry`) including the build of the fetched artifact.

#### Scenario: Operations on one Kernel build one client

- **WHEN** one Kernel acquires a module from the registry, acquires a platform from a directory, synthesizes an instance and renders it
- **THEN** the registry client is constructed exactly once, and each of the four is one operation that resolves through it: the registry acquire both for its fetch and for the build after it, the platform acquire for its catalog dependencies, the synthesis for the module's dependencies, and the render build

#### Scenario: Concurrent operations share the client

- **WHEN** several goroutines acquire, synthesize and render on one Kernel at the same time under the race detector, the client's first use included
- **THEN** every call succeeds with the same result as the sequential run, the client is constructed once, and the race detector reports nothing

### Requirement: A Kernel hands its registry client to directory acquisition
A `Kernel` SHALL own one registry client, meaning the resolver and the OCI transport behind it and not a module cache, and SHALL hand it to every module load and registry fetch it runs for its own operations: directory acquisition (`AcquireModuleFromDir`, `AcquireCatalogFromDir`, `AcquirePlatformFromDir`, `AcquireInstanceFromDir`, values-layered or not).

#### Scenario: Directory acquisition uses the one client

- **WHEN** a Kernel runs a directory acquisition verb
- **THEN** the verb receives the registry client the Kernel owns, values-layered or not

### Requirement: A Kernel hands its registry client to synthesis, values sources and the render build
A `Kernel` SHALL own one registry client, meaning the resolver and the OCI transport behind it and not a module cache, and SHALL hand it to every module load and registry fetch it runs for its own operations: instance synthesis (`SynthesizeInstance`), the compilation of file-backed values sources (`ValidateConfigDetailed` and every verb that accepts `Source` values), and the render build (`Render`).

#### Scenario: A file-backed values source resolves through the client

- **WHEN** a values file whose module imports a package served only by the Kernel's registry is compiled by a Kernel verb
- **THEN** the import resolves through that Kernel's registry client

### Requirement: The registry client is built on first use and its settings are read then
The registry client SHALL be built from the kernel's registry mapping (`WithRegistry`, else the process `CUE_REGISTRY`) on its first use, never in `kernel.New`, so construction still evaluates nothing and fails on nothing. The registry mapping and the credentials configuration SHALL be read when the client is built, and the `Kernel` documentation SHALL say so.

#### Scenario: Construction builds no client

- **WHEN** `kernel.New(WithRegistry(mapping))` returns and no operation has run
- **THEN** no registry client has been constructed

### Requirement: A failure to build the client is returned and not kept
A failure to build the registry client SHALL be returned to the operation that needed it and SHALL NOT be kept: the next operation that needs a client SHALL try to build it again.

#### Scenario: A failed construction is retried

- **WHEN** the first construction of a Kernel's registry client fails and a later operation needs the client
- **THEN** the first operation returns the construction error, and the later operation builds the client again and succeeds when the cause is gone

#### Scenario: A construction failure keeps its wording

- **WHEN** `AcquireModuleFromRegistry` runs on a Kernel whose registry client cannot be built
- **THEN** the error reads `building module registry resolver:` followed by the cause, and it is not classified as a fetch failure

### Requirement: Each operation wraps the client in a module cache of its own
Each operation SHALL wrap the registry client in a module cache of its own, over the cache directory read when that operation starts (`CUE_CACHE_DIR`, else the user cache directory), so that no fetch failure (a refused connection, an error status, a version not yet published, a cancelled or expired context) is served to a later operation, and a later change to `CUE_CACHE_DIR` reaches the next operation as it does today.

#### Scenario: A transient fetch failure is not remembered

- **WHEN** the registry refuses the first request for a module version and serves every later one, and one Kernel acquires that module version twice
- **THEN** the first acquire returns the fetch failure, and the second acquire on the same Kernel fetches the module and succeeds

#### Scenario: A cancelled fetch is not remembered

- **WHEN** an acquire of a module version that is not in the module cache runs with an already-cancelled context, and a second acquire of the same version runs on the same Kernel with a live context
- **THEN** the first acquire returns the cancellation, and the second fetches the module and succeeds

#### Scenario: The cache directory is read for each operation

- **WHEN** a Kernel has run an operation and the process then points `CUE_CACHE_DIR` at an empty directory
- **THEN** the next registry acquire on that Kernel fetches into the new directory

### Requirement: A failed registry call drops the shared client
A registry call that fails SHALL also drop the shared registry client, so that the next operation builds it again.

#### Scenario: A failed registry call drops the client

- **WHEN** a registry call fails
- **THEN** the shared client is dropped, so that the next operation builds it again

### Requirement: The registry client does not replace the schema cache's loader
The registry client SHALL NOT replace the schema cache's own loader: an `OCILoader` is configured on its own (it may override the cache directory) and the schema cache loads once per Kernel, and a caller-supplied `schema.Loader` is the caller's.

#### Scenario: The schema loader keeps its own client

- **WHEN** a Kernel's default schema cache loads the core schema
- **THEN** the load goes through the `OCILoader`'s own client, not through the Kernel's registry client

### Requirement: The helper platformmodule registry stays caller-built and no exported signature changes
The opt-in `helper/platformmodule` registry stays caller-built. No exported signature SHALL change for this: the registry client is not an option, a parameter or an exported field.

#### Scenario: No exported signature changes for the client

- **WHEN** a developer inspects the exported signatures of the Kernel
- **THEN** the client is not an option, a parameter or an exported field, and the opt-in `helper/platformmodule` registry stays caller-built

### Requirement: The effective values are the built spec's values
The effective values SHALL be the built spec's `values`: the package's own `values` plus the trailing sources on `AcquireInstanceFromDir`, the merged `InstanceInput.Values` on `SynthesizeInstance`. Source: library#211, settled by the owner in the pull request that closes it; opm-operator#258 for the report of both checks.

#### Scenario: The effective values are the built spec's values

- **WHEN** an instance is refused for an unset required value
- **THEN** the effective values are the built spec's `values`: the package's own `values` plus the trailing sources on `AcquireInstanceFromDir`, the merged `InstanceInput.Values` on `SynthesizeInstance`

### Requirement: A required value is one that concreteness refuses
A required value SHALL be one that `cue.Concrete(true)` refuses on those values unified with `#config`: a required field (`foo!:`) that is absent, or a field that resolves to no concrete value (a bare type such as `string` or `_`, with no default). An optional field (`foo?:`) and a field with a default are not required. Source: library#211, settled by the owner in the pull request that closes it; opm-operator#258 for the report of both checks.

#### Scenario: A required value is one that concreteness refuses

- **WHEN** the effective values are unified with `#config`
- **THEN** a required field (`foo!:`) that is absent, or a field that resolves to no concrete value, is required, and an optional field (`foo?:`) and a field with a default are not required

### Requirement: Both concreteness checks run in one instance processing step
Both concreteness checks, the one on the built spec and the one on the values unified with `#config`, SHALL run in the kernel's one instance processing step, with `#config` read off the built spec and never off `Module.Package`, so that both verbs (Requirement: Instance verbs refuse an unset required config value) refuse the same instances with the same findings. Source: library#211, settled by the owner in the pull request that closes it; opm-operator#258 for the report of both checks.

#### Scenario: Both verbs run both checks in one step

- **WHEN** `SynthesizeInstance` or `AcquireInstanceFromDir` processes an instance
- **THEN** both concreteness checks run in the kernel's one instance processing step, with `#config` read off the built spec and never off `Module.Package`

### Requirement: The refusal is CUE's own error tree framed with the verb and the instance
The refusal (Requirement: Instance verbs refuse an unset required config value) SHALL be CUE's own error tree, framed `Kernel.<Verb>: instance "<name>": not fully concrete: …`. When the values leave no required value unset, the refusal SHALL be the error of the built-spec check, unchanged. Source: library#211, settled by the owner in the pull request that closes it; opm-operator#258 for the report of both checks.

#### Scenario: The refusal is CUE's own error tree

- **WHEN** an instance is refused for an unset required value
- **THEN** the refusal is CUE's own error tree, framed `Kernel.<Verb>: instance "<name>": not fully concrete: …`

### Requirement: The refusal holds the value findings of the built-spec check
When the values leave one or more unset, the refusal (Requirement: Instance verbs refuse an unset required config value) SHALL hold findings about the values only:
1. every finding of the built-spec check at a path under `values` (a value the instance's values carry in a non-concrete form), positioned where the values carry it; Source: library#211, settled by the owner in the pull request that closes it; opm-operator#258 for the report of both checks.

#### Scenario: The refusal holds the findings of the values only

- **WHEN** the values leave one or more required values unset
- **THEN** the refusal holds every finding of the built-spec check at a path under `values`, positioned where the values carry it

### Requirement: The refusal names every other unset required value at values.<field>
When the values leave one or more unset, the refusal (Requirement: Instance verbs refuse an unset required config value) SHALL hold findings about the values only:
2. one finding for every other unset required value, at the path `values.<field>` (the full path for a nested field), whether or not a component reads it. Source: library#211, settled by the owner in the pull request that closes it; opm-operator#258 for the report of both checks.

#### Scenario: The refusal names every other unset value

- **WHEN** the values leave one or more required values unset
- **THEN** the refusal holds one finding for every other unset required value, at the path `values.<field>`, whether or not a component reads it

### Requirement: A finding carries the position of the field's #config declaration
The findings are those of requirement "The refusal names every other unset required value at values.<field>". A finding SHALL carry the position of the field's `#config` declaration wherever CUE records one; a field CUE records no position for (a field declared `_`) is identified by its path only. A path SHALL be reported once. Source: library#211, settled by the owner in the pull request that closes it; opm-operator#258 for the report of both checks.

#### Scenario: A finding carries the position of its declaration

- **WHEN** a finding names an unset field
- **THEN** it carries the position of the field's `#config` declaration wherever CUE records one, and a path is reported once

### Requirement: Findings outside values are left out of the refusal
Every finding of the built-spec check outside `values` SHALL be left out of that refusal (Requirement: Instance verbs refuse an unset required config value): the component fields that read an unset value, and any other incomplete field of the module. Such a field that is still incomplete once the values are complete is then refused with the built-spec error. Source: library#211, settled by the owner in the pull request that closes it; opm-operator#258 for the report of both checks.

#### Scenario: Component findings are left out of the refusal

- **WHEN** the values leave a required value unset
- **THEN** every finding of the built-spec check outside `values` is left out of that refusal

### Requirement: The kernel does not decide from positions whether an unset value causes a finding
The kernel SHALL NOT decide from the positions of a finding whether an unset value causes it. When a required value is unset, every finding of the tree (Requirement: The refusal is CUE's own error tree framed with the verb and the instance) is a values finding. Source: library#211, settled by the owner in the pull request that closes it; opm-operator#258 for the report of both checks.

#### Scenario: Positions decide nothing

- **WHEN** a finding is judged
- **THEN** the kernel does not decide from the positions of a finding whether an unset value causes it, and when a required value is unset every finding of the tree is a values finding

### Requirement: A caller identifies the refusal by CUE's error type
A caller SHALL identify the refusal (Requirement: Instance verbs refuse an unset required config value) as it did before this requirement changed: `errors.As` with CUE's error type finds the tree, and a finding whose path starts with `values` is a values finding; the kernel adds no error type for it. Source: library#211, settled by the owner in the pull request that closes it; opm-operator#258 for the report of both checks.

#### Scenario: A caller finds the tree with errors.As

- **WHEN** a caller handles the refusal
- **THEN** `errors.As` with CUE's error type finds the tree, and the kernel adds no error type for it

### Requirement: SynthesizeInstance refuses what ValidateConfigDetailed refuses
For the same module and the same single values source, `SynthesizeInstance` SHALL refuse every value that `ValidateConfigDetailed(#config, sources)` refuses, and SHALL name at `values.<field>` every unset field that `ValidateConfigDetailed` names at `#config.<field>`. Source: library#211, settled by the owner in the pull request that closes it; opm-operator#258 for the report of both checks.

#### Scenario: SynthesizeInstance refuses what ValidateConfigDetailed refuses

- **WHEN** the same module and the same single values source are checked by both
- **THEN** `SynthesizeInstance` refuses every value that `ValidateConfigDetailed(#config, sources)` refuses, and names at `values.<field>` every unset field that `ValidateConfigDetailed` names at `#config.<field>`

### Requirement: SynthesizeInstance may refuse more only through the concreteness check on the built spec
For the same module and the same single values source, `SynthesizeInstance` SHALL refuse every value that `ValidateConfigDetailed(#config, sources)` refuses; it MAY refuse more only through the concreteness check on the built spec, which refuses a non-concrete value the instance's values carry even where a `#config` default would complete it. Source: library#211, settled by the owner in the pull request that closes it; opm-operator#258 for the report of both checks.

#### Scenario: SynthesizeInstance may refuse more

- **WHEN** the built spec carries a non-concrete value in the instance's values
- **THEN** `SynthesizeInstance` may refuse it even where a `#config` default would complete it
