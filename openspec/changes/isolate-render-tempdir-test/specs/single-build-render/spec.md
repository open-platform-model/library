## ADDED Requirements

### Requirement: Render staging assertions observe only a test-private temp root

A test that asserts on the render staging directories `Kernel.Render` creates SHALL first point the process temp directory at a directory owned by that test, and SHALL list only that directory. The result of such a test SHALL NOT depend on staging directories that other test processes, sharing the same `TMPDIR`, create or remove while it runs. Each test keeps its intent: after its renders or refusals, its private root holds no staging directory.

#### Scenario: Concurrent test processes share one TMPDIR

- **WHEN** several processes of the `opm/kernel` test binary run the staging-directory tests repeatedly at the same time with one shared `TMPDIR`
- **THEN** none of those tests fails because of a staging directory another process created or removed

#### Scenario: A render leaks its staging directory

- **WHEN** a change to `Kernel.Render` leaves a staging directory behind after a successful render or a refusal
- **THEN** the test covering that path fails, naming the leftover directory under its private root
