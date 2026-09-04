## MODIFIED Requirements

### Requirement: Imported-module render coverage exists

The library SHALL include a test that renders an instance whose module is referenced by import (not inlined) end-to-end through construction and `Kernel.Render`, against a D5-shaped platform (a platform module importing its catalog), producing concrete resources. This coverage SHALL exist for both the synth path and an authored-package path so that a regression in either surfaces. The synth-path coverage SHALL include a module that imports a **catalog subpackage** (e.g. a workload blueprint under `opmodel.dev/catalogs/opm/...`), so that a regression to a dependency-incomplete synthesis surfaces as a failing test.

#### Scenario: Real imported module renders to resources

- **WHEN** an instance referencing a published module by import is rendered through `Kernel.Render`
- **THEN** the rendered output contains the module's expected resources
- **AND** the test fails if import-based construction regresses to a `field not allowed` admission error

#### Scenario: Module importing a catalog subpackage synthesizes and renders

- **WHEN** `synth.Instance` is called with a published module whose source imports a transitive catalog subpackage (the library#31 shape), against a registry serving that catalog
- **THEN** synthesis succeeds without `cannot find module providing package opmodel.dev/catalogs/opm/...`
- **AND** the instance renders to the module's expected resources through `Kernel.Render`

#### Scenario: Single-build parity with an authored package

- **WHEN** `synth.Instance` builds an instance for module M with values V, and an authored `instance.cue` package imports the same M and sets the same V
- **THEN** both, passed through `Kernel.Render` against the same platform, produce the same rendered objects
