## MODIFIED Requirements

### Requirement: Instance construction shares one evaluate-and-shape-gate with the file loader

The kernel SHALL evaluate a synthesized instance package (in-memory overlay inside the module's staged tree) and a directory-acquired instance package (on-disk source) through one build-and-shape-gate routine. The two paths SHALL differ only in how the package source is supplied; the evaluation, shape gating, sentinel wrapping and values attribution SHALL be identical.

#### Scenario: Overlay and on-disk instances evaluate identically

- **WHEN** the same instance content is supplied once through `SynthesizeInstance` and once through `AcquireInstanceFromDir`
- **THEN** both produce a value of the same shape passing the same instance shape gate
- **AND** a malformed instance fails the shape gate identically in both paths, wrapping the same `opm/errors` sentinel

#### Scenario: A values conflict that fails the build is attributed identically

- **WHEN** the same violating values source (a value that breaks `#config` at a path a component consumes) is supplied once to `SynthesizeInstance` for a module and once to `AcquireInstanceFromDir` for an instance package of that module
- **THEN** both calls fail with a values error framed `instance "<name>": …` whose positions name the source's `Origin`, and neither returns an instance
