## ADDED Requirements

### Requirement: Each render is its own in-memory build in its own context

`Render` SHALL create a fresh `cue.Context` for the render, evaluate the staged render module exactly once with it, and release its own references to it when `Render` returns; a `*kernel.Compiled` the caller holds keeps that build alive until the caller releases it. No built value SHALL be shared between renders, and the render SHALL NOT use any long-lived context. The render module SHALL be staged in memory under a fixed synthetic root that does not exist on disk: its generated `cue.mod/module.cue`, `cue.mod/local-module.cue` and glue file, and every file of an overlay-mode input, SHALL be served to the build from the load overlay, and an on-disk input SHALL be served from its own directory, so nothing of either input is copied and nothing is written. The coverage invariant SHALL be checked against the `cue.mod/module.cue` bytes the build is served.

#### Scenario: Repeated renders share nothing

- **WHEN** `Render` is invoked twice with the same inputs
- **THEN** each invocation stages, builds and decodes independently, and the results are byte-identical

#### Scenario: The render module is served from memory

- **WHEN** a render of an overlay-mode instance against an overlay-mode platform is staged
- **THEN** the staged overlay carries `cue.mod/module.cue`, `cue.mod/local-module.cue` and the glue file under the synthetic root, and every input file under that root's `instance/` or `platform/` directory
- **AND** the synthetic root does not exist on disk before or after the build

#### Scenario: Local replacements resolve from an in-memory render module

- **WHEN** a render module whose `cue.mod/local-module.cue` replaces input and catalog paths with on-disk directories is served from memory under the synthetic root
- **THEN** the one build resolves every replaced path from its directory
- **AND** the same build with `cue.mod/local-module.cue` left out of the overlay fails to resolve the replaced input, so the overlay's file is the one the build read

#### Scenario: Concurrent renders share the synthetic root

- **WHEN** several goroutines build render modules staged under the same synthetic root at the same time
- **THEN** every build succeeds with the same result as a sequential build, and the race detector reports nothing

### Requirement: A render writes no staging file

`Kernel.Render` SHALL write no staging file: it SHALL create, write or remove nothing under the process temp directory and nothing at the render module's synthetic root, not on a successful render, not on a refusal before staging, not on a refusal before evaluation, and not on a build failure. The CUE module cache is not staging: a render whose build fetches a dependency fills the module cache under `CUE_CACHE_DIR` exactly as any other load does. A test that asserts this SHALL point the process temp directory at a directory owned by that test and SHALL find it empty afterwards, so its result does not depend on other test processes sharing the same `TMPDIR`, and SHALL find that the synthetic root does not exist.

#### Scenario: A successful render leaves the temp directory untouched

- **WHEN** a test points `TMPDIR` at an empty directory it owns and renders a served instance against a served platform
- **THEN** the render succeeds, the directory is still empty, and the render module's synthetic root does not exist on disk

#### Scenario: A refusal leaves the temp directory untouched

- **WHEN** a test points `TMPDIR` at an empty directory it owns and `Render` refuses an older-core platform, a local replacement without the opt-in, and an uncovered OPM-namespace path
- **THEN** each call returns its refusal, the directory is still empty, and the synthetic root does not exist on disk

#### Scenario: A render that writes is caught

- **WHEN** a change to `Kernel.Render` writes a file or directory under the process temp directory
- **THEN** the test covering that path fails, naming what it found under its private root

## MODIFIED Requirements

### Requirement: Local replacements are honoured only when the caller opts in

`Kernel.Render` SHALL accept a per-render opt-in for local replacements, off by default. When it is off, an input whose `cue.mod/local-module.cue` carries at least one replacement SHALL be refused before staging with an error naming the input and the file; the file SHALL NOT be silently ignored. When it is on, promoted replacements SHALL be reported on the render diagnostics as rows carrying the replaced path, the absolute target (a directory, or a module path for a module replacement) and the input that supplied it, in path order; the kernel SHALL NOT render them as message strings. An input without the file SHALL behave identically under either setting.

#### Scenario: Replacement without opt-in refused

- **WHEN** `Render` is invoked with the opt-in off and the platform module's `cue.mod/local-module.cue` replaces its catalog path
- **THEN** it returns an error naming the platform and `cue.mod/local-module.cue`, no render module is staged, and no staging file is written

#### Scenario: Inputs without the file are unaffected

- **WHEN** `Render` is invoked with the opt-in on and neither input carries `cue.mod/local-module.cue`
- **THEN** the render's output and diagnostics are byte-identical to the same render with the opt-in off, and the replacement rows are empty

#### Scenario: Replacement rows are data

- **WHEN** a render with the opt-in on honours one platform replacement and one instance replacement
- **THEN** the result's diagnostics carry exactly two replacement rows, sorted by path, each naming its source input, and no field of the result holds a formatted message about them

### Requirement: A render refuses a platform whose core predates the provider count

`Kernel.Render` SHALL check, before staging, that the platform's core floor is met: that the platform's `#contracts` carries `providedBy`, as recorded when the platform was constructed and reported by `Platform.CoreFloor()` (`platform-artifact`, "A platform records its core floor and contract inventory at construction"). When it does not (the platform module pins a core release older than `2.0.0-alpha.12`, or carries no `#contracts` at all), `Render` SHALL return an error wrapping the typed `PlatformCoreTooOldError` that names the platform, the missing field and the first core release carrying it. The refusal SHALL NOT be a `*RenderError`, SHALL stage no render module and SHALL NOT fall back to a count of the kernel's own: a render never runs on a platform whose inventory would disagree with it. The check is sound because the render build evaluates the platform module's own core pin, the same one its `Package` was built from. The check SHALL read the recorded floor through `CoreFloor()`, and on a platform the constructor built it SHALL NOT read the platform's `Package`; a platform the constructor did not build decodes `Package` once, on its first `CoreFloor()` or `Contracts()` call (`platform-artifact`, "A platform records its core floor and contract inventory at construction"). Apart from the floor, `Render` reads only the platform's `Metadata` and `Source`. One acquired platform therefore stays shareable, as data, across concurrent `Render` calls on one Kernel.

#### Scenario: An older-core platform is refused before staging

- **WHEN** a platform module identical to a served fixture but pinning core `2.0.0-alpha.10` is acquired and rendered
- **THEN** `Render` returns an error from which `errors.As` extracts a `PlatformCoreTooOldError` naming the platform, the field `providedBy` and the release `2.0.0-alpha.12`, the error is not a `*RenderError`, no result is returned, and no staging file is written

#### Scenario: A current-core platform is not affected

- **WHEN** a platform module pinning core `2.0.0-alpha.12` or later is rendered
- **THEN** the core floor raises no error and the render proceeds to staging

#### Scenario: A platform shared by concurrent renders stays race-free

- **WHEN** one acquired current-core platform is shared by several goroutines, each calling `Render` on one Kernel
- **THEN** every render passes the core floor and produces the same objects, with no data race reported under the race detector

#### Scenario: A render reads no platform Package

- **WHEN** a current-core platform is acquired, its `Package` is replaced with the zero `cue.Value`, and it is rendered with an acquired instance
- **THEN** the core floor raises no error and the render produces the same objects as a render of the unchanged platform

## REMOVED Requirements

### Requirement: Each render is its own build in its own context

**Reason**: A render no longer stages into a directory, so the requirement's "staging directory" sentences and its scenario "The staging directory holds no input files" name something that does not exist. OpenSpec does not let a MODIFIED requirement drop a main-spec scenario.

**Migration**: Replaced by "Each render is its own in-memory build in its own context", which keeps "Repeated renders share nothing" word for word and states the in-memory staging. Nothing changes for a caller: `Render`'s signature, output and diagnostics are the same.

### Requirement: Render staging assertions observe only a test-private temp root

**Reason**: A render creates no staging directory, so there is nothing to observe. The concern behind the requirement (a test whose result depends on another test process sharing `TMPDIR`) still applies to the stronger claim that a render writes no staging file.

**Migration**: Replaced by "A render writes no staging file", whose tests point `TMPDIR` at a directory they own and find it empty after their renders and refusals.
