## ADDED Requirements

### Requirement: One registry client per Kernel

A `Kernel` SHALL own one registry client, meaning the resolver and the OCI transport behind it and not a module cache, and SHALL hand it to every module load and registry fetch it runs for its own operations: registry acquisition (`AcquireModuleFromRegistry`, `AcquireCatalogFromRegistry`) including the build of the fetched artifact, directory acquisition (`AcquireModuleFromDir`, `AcquireCatalogFromDir`, `AcquirePlatformFromDir`, `AcquireInstanceFromDir`, values-layered or not), instance synthesis (`SynthesizeInstance`), the compilation of file-backed values sources (`ValidateConfigDetailed` and every verb that accepts `Source` values), and the render build (`Render`). The client SHALL be built from the kernel's registry mapping (`WithRegistry`, else the process `CUE_REGISTRY`) on its first use, never in `kernel.New`, so construction still evaluates nothing and fails on nothing. A failure to build the client SHALL be returned to the operation that needed it and SHALL NOT be kept: the next operation that needs a client SHALL try to build it again. Each operation SHALL wrap the client in a module cache of its own, over the cache directory read when that operation starts (`CUE_CACHE_DIR`, else the user cache directory), so that no fetch failure (a refused connection, an error status, a version not yet published, a cancelled or expired context) is served to a later operation, and a later change to `CUE_CACHE_DIR` reaches the next operation as it does today. A registry call that fails SHALL also drop the shared client, so that the next operation builds it again. The registry mapping and the credentials configuration SHALL be read when the client is built, and the `Kernel` documentation SHALL say so. The client SHALL NOT replace the schema cache's own loader: an `OCILoader` is configured on its own (it may override the cache directory) and the schema cache loads once per Kernel, and a caller-supplied `schema.Loader` is the caller's. The opt-in `helper/platformmodule` registry stays caller-built. No exported signature SHALL change for this: the client is not an option, a parameter or an exported field.

#### Scenario: Operations on one Kernel build one client

- **WHEN** one Kernel acquires a module from the registry, acquires a platform from a directory, synthesizes an instance with a file-backed values source and renders it
- **THEN** the registry client is constructed exactly once, and the render build and the fetch both resolve through it

#### Scenario: Construction builds no client

- **WHEN** `kernel.New(WithRegistry(mapping))` returns and no operation has run
- **THEN** no registry client has been constructed

#### Scenario: A failed construction is retried

- **WHEN** the first construction of a Kernel's registry client fails and a later operation needs the client
- **THEN** the first operation returns the construction error, and the later operation builds the client again and succeeds when the cause is gone

#### Scenario: A construction failure keeps its wording

- **WHEN** `AcquireModuleFromRegistry` runs on a Kernel whose registry client cannot be built
- **THEN** the error reads `building module registry resolver:` followed by the cause, and it is not classified as a fetch failure

#### Scenario: A transient fetch failure is not remembered

- **WHEN** the registry refuses the first request for a module version and serves every later one, and one Kernel acquires that module version twice
- **THEN** the first acquire returns the fetch failure, and the second acquire on the same Kernel fetches the module and succeeds

#### Scenario: A cancelled fetch is not remembered

- **WHEN** an acquire of a module version that is not in the module cache runs with an already-cancelled context, and a second acquire of the same version runs on the same Kernel with a live context
- **THEN** the first acquire returns the cancellation, and the second fetches the module and succeeds

#### Scenario: The cache directory is read for each operation

- **WHEN** a Kernel has run an operation and the process then points `CUE_CACHE_DIR` at an empty directory
- **THEN** the next registry acquire on that Kernel fetches into the new directory

#### Scenario: Concurrent operations share the client

- **WHEN** several goroutines acquire, synthesize and render on one Kernel at the same time under the race detector, the client's first use included
- **THEN** every call succeeds with the same result as the sequential run, the client is constructed once, and the race detector reports nothing

#### Scenario: The schema loader keeps its own client

- **WHEN** a Kernel's default schema cache loads the core schema
- **THEN** the load goes through the `OCILoader`'s own client, not through the Kernel's registry client
