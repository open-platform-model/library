# kernel-runtime Delta

## MODIFIED Requirements

### Requirement: Kernel Type and Construction

The library SHALL expose a `Kernel` struct in `opm/kernel/` that serves as the single public anchor type for the OPM kernel runtime. The struct SHALL be constructible only via the `kernel.New(opts ...Option)` function.

#### Scenario: Default construction

- **WHEN** a caller invokes `kernel.New()` with no options
- **THEN** a non-nil `*Kernel` is returned with a private `*cue.Context` constructed via `cuecontext.New()` and a schema cache backed by the default OCI loader
- **AND** subsequent calls to `k.CueContext()` return the same `*cue.Context` instance for the lifetime of the Kernel

#### Scenario: Construction with options

- **WHEN** a caller invokes `kernel.New(WithSchemaLoader(myLoader), WithRegistry(mapping))`
- **THEN** the returned Kernel resolves the core schema through `myLoader` and uses `mapping` for catalog and module resolution

## ADDED Requirements

### Requirement: Configuration Options

The Kernel SHALL accept configuration through functional options of type `Option`. The provided options SHALL be `WithSchemaLoader` and `WithRegistry`; the Kernel SHALL NOT expose an injection slot no kernel operation reads.

#### Scenario: Adding new options preserves backward compatibility

- **WHEN** a future slice adds a new option (e.g. `WithSchemaRegistry`)
- **THEN** existing callers of `kernel.New(...)` continue to compile and run unchanged

#### Scenario: No observability slots ahead of a reader

- **WHEN** a developer inspects the `Kernel` struct and its options after this change
- **THEN** no logger, tracer or clock field or option exists
- **AND** the injection surface for the execution half is introduced by enhancement 0009 together with its first reader (revised 0009 D9)

## REMOVED Requirements

### Requirement: Functional Options Pattern

**Reason**: The requirement mandated `WithLogger`, `WithTracer` and `WithClock`, three slots stored by `New` and never read by any kernel operation since they were added; the tracer slot was the sole reason `go.opentelemetry.io/otel/trace` was a direct dependency. Enhancement 0009 D9, which reserved them for the execution half, has been revised: the execution half introduces the injection surface it needs together with its first reader instead of inheriting write-only slots. The options pattern itself survives as the ADDED "Configuration Options" requirement.

**Migration**: Callers passing `WithLogger`, `WithTracer` or `WithClock` delete the argument; no observable behavior changes because nothing ever read the injected values. The one known caller is `opm-operator`'s `cmd/main.go` (`WithLogger`), migrated in the same PR wave.
