## MODIFIED Requirements

### Requirement: Render inputs are source-carrying artifacts

`Kernel.Render` SHALL accept a `*module.Instance` and a `*platform.Platform` that both carry a `Source` (staged tree or on-disk directory), plus a non-empty runtime name and a skew policy. It SHALL refuse an input whose `Source` is absent with an error naming the input; an evaluated `Package` alone is never sufficient, because the render build imports packages.

#### Scenario: Source-less platform refused

- **WHEN** `Render` is invoked with a platform constructed from a bare `cue.Value` (`Source == nil`)
- **THEN** it returns an error naming the platform's missing source, and no build is attempted

#### Scenario: Overlay-mode instance accepted

- **WHEN** `Render` is invoked with a synthesized instance whose `Source` is an overlay tree
- **THEN** the tree is served to the build from memory through the load configuration's overlay, no file of it is written, and the build proceeds

### Requirement: Each render is its own build in its own context

`Render` SHALL create a fresh `cue.Context` for the render, evaluate the staged render module exactly once with it, and release it when `Render` returns. No built value SHALL be shared between renders, and the render SHALL NOT use any long-lived context. The staging directory SHALL hold only the generated render module (its `cue.mod` and glue); an overlay-mode input SHALL be served from memory and an on-disk input from its own directory, so nothing of either input is copied. The staging directory SHALL be removed when the render completes.

#### Scenario: Repeated renders share nothing

- **WHEN** `Render` is invoked twice with the same inputs
- **THEN** each invocation stages, builds and decodes independently, and the results are byte-identical

#### Scenario: The staging directory holds no input files

- **WHEN** a render of an overlay-mode instance against an overlay-mode platform is staged
- **THEN** the staging directory contains `cue.mod/module.cue`, `cue.mod/local-module.cue` and the glue file, and no `instance/` or `platform/` directory
