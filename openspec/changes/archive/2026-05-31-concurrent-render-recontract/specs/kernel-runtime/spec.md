## ADDED Requirements

### Requirement: Compile sources its cue.Context from the caller Kernel

The compile pipeline (Finalize → Match → Execute, driven by `Kernel.Compile`) SHALL build every value it constructs — the finalized data components, the per-pair transformer `#context.*` view, and the rendered output — using the **caller Kernel's** owned `*cue.Context` (the instance returned by `k.CueContext()`). It SHALL NOT derive the build context from the materialized platform (`mp.Package.Context()` / `platform.Package.Context()`). The materialized platform's `Package` is read as input (the `FillPath` argument and cross-read source), not as the owner of the build context.

#### Scenario: Compiled output builds in the Kernel's cue.Context

- **WHEN** a developer calls `k.Compile(ctx, in)` and inspects the `*cue.Context` underlying a rendered value in the returned `CompileResult.Compiled`
- **THEN** that context is the same instance returned by `k.CueContext()`

#### Scenario: Pipeline does not call Value.Context on the platform

- **WHEN** the compile pipeline finalizes components and executes transformers
- **THEN** it obtains its `*cue.Context` from the caller Kernel
- **AND** it does not call `Value.Context()` on the materialized platform's `Package`

#### Scenario: Behavior preserved for single-Kernel callers

- **WHEN** a single Kernel materializes a platform and then compiles a release against it (the platform's `Package` was built in that same Kernel's `*cue.Context`)
- **THEN** the rendered output is identical to the prior platform-context-sourced behavior, because the caller context and the platform context are the same instance
