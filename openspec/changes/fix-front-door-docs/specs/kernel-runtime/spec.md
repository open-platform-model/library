## ADDED Requirements

### Requirement: Front-door docs list the kernel's verbs and packages as built

`CONSTITUTION.md` Principle III SHALL list the `Kernel` verbs as acquire, synthesize, validate and render, and SHALL describe `opm/module/` as the module and instance model. Its pipeline block SHALL show acquisition or synthesis producing the artifacts, one render build, and `[]*kernel.Compiled` as the output; it SHALL NOT name a load or process verb, a separate schema-validation stage, or a package that does not exist. The `README.md` layout block SHALL list only packages that exist under `opm/`, and SHALL place `Compiled` in `opm/kernel`.

#### Scenario: Principle III names the verbs as built

- **WHEN** a developer reads the `opm/kernel/` and `opm/module/` bullets of `CONSTITUTION.md` Principle III
- **THEN** the verbs read acquire, synthesize, validate and render, with no load or process verb
- **AND** `opm/module/` is described as the module and instance model

#### Scenario: The pipeline ends in kernel.Compiled

- **WHEN** a developer reads the pipeline block of `CONSTITUTION.md` Principle III
- **THEN** it runs from acquire or synthesize through the four artifacts and one render build to `[]*kernel.Compiled`
- **AND** it names no `core` package and no separate schema-validate stage

#### Scenario: The README layout names only real packages

- **WHEN** a developer compares the `README.md` layout block with the directories under `opm/`
- **THEN** every listed package exists, no `opm/core/` row appears, and the `kernel/` row names `Compiled`
