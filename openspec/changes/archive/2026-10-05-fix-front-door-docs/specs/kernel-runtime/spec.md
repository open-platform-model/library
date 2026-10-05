## ADDED Requirements

### Requirement: Front-door docs list the kernel's verbs and packages as built

Where `CONSTITUTION.md` Principle III lists the `Kernel` verbs, it SHALL list them as acquire, synthesize, validate and render; where it describes `opm/module/`, it SHALL describe the module and instance model. Where it shows a pipeline, the pipeline SHALL show acquisition or synthesis producing the artifacts and one render build over an instance and a platform, ending in `[]*kernel.Compiled`, and SHALL NOT send a `Catalog` into render. None of these SHALL name a load or process verb, a separate schema-validation stage, or a package that does not exist. Where `README.md` carries a layout block, it SHALL list only packages that exist under `opm/`, and SHALL NOT place `Compiled` outside `opm/kernel`; the front-door files MAY link to this spec instead of restating it.

#### Scenario: Principle III names the verbs as built

- **WHEN** a developer reads the `opm/kernel/` and `opm/module/` bullets of `CONSTITUTION.md` Principle III
- **THEN** the verbs read acquire, synthesize, validate and render, with no load or process verb
- **AND** `opm/module/` is described as the module and instance model

#### Scenario: The pipeline ends in kernel.Compiled

- **WHEN** a developer reads the pipeline block of `CONSTITUTION.md` Principle III
- **THEN** it runs from acquire or synthesize to one render build over an instance and a platform, ending in `[]*kernel.Compiled`
- **AND** it names no `core` package, no separate schema-validate stage, and no `Catalog` entering render

#### Scenario: The README layout names only real packages

- **WHEN** a developer compares the `README.md` layout block with the directories under `opm/`
- **THEN** every listed package exists, no `opm/core/` row appears, and `Compiled`, where named, sits on the `kernel/` row
