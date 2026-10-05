## MODIFIED Requirements

### Requirement: Each render is its own in-memory build in its own context

`Render` SHALL create a fresh `cue.Context` for the render, evaluate the staged render module exactly once with it, and release its own references to it when `Render` returns; a `*kernel.Compiled` the caller holds keeps that build alive until the caller releases it. No built value SHALL be shared between renders, and the render SHALL NOT use any long-lived context. The render module SHALL be staged in memory under a fixed synthetic root, an absolute path on every operating system (on Windows it carries a volume), that does not exist on disk. Because cue/load reads a real directory beneath an overlay root, staging SHALL refuse, with an error naming the path, when anything exists at that root. The render module's generated `cue.mod/module.cue`, `cue.mod/local-module.cue` and glue file, and every file of an overlay-mode input, SHALL be served to the build from the load overlay, and an on-disk input SHALL be served from its own directory, so nothing of either input is copied and nothing is written. The coverage invariant SHALL be checked against the `cue.mod/module.cue` bytes the build is served.

#### Scenario: Repeated renders share nothing

- **WHEN** `Render` is invoked twice with the same inputs
- **THEN** each invocation stages, builds and decodes independently, and the results are byte-identical

#### Scenario: The render module is served from memory

- **WHEN** a render of an overlay-mode instance against an overlay-mode platform is staged
- **THEN** the staged overlay carries `cue.mod/module.cue`, `cue.mod/local-module.cue` and the glue file under the synthetic root, and every input file under that root's `instance/` or `platform/` directory
- **AND** the synthetic root does not exist on disk before or after the build

#### Scenario: An existing render root is refused

- **WHEN** a directory holding a `.cue` file of the render package, or a plain file, exists at the synthetic root and a render module is staged
- **THEN** staging returns an error naming the root, and nothing is staged or built

#### Scenario: The synthetic roots are absolute

- **WHEN** the render root and a registry module's synthetic root are computed on any operating system
- **THEN** both are absolute paths, so cue/load accepts the overlay keys beneath them

#### Scenario: Local replacements resolve from an in-memory render module

- **WHEN** a render module whose `cue.mod/local-module.cue` replaces input and catalog paths with on-disk directories is served from memory under the synthetic root
- **THEN** the one build resolves every replaced path from its directory
- **AND** the same build with `cue.mod/local-module.cue` left out of the overlay fails to resolve the replaced input, so the overlay's file is the one the build read

#### Scenario: Concurrent renders share the synthetic root

- **WHEN** several goroutines build render modules staged under the same synthetic root at the same time
- **THEN** every build succeeds with the same result as a sequential build, and the race detector reports nothing
