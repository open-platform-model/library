## ADDED Requirements

### Requirement: A held Render output keeps its build alive

The kernel SHALL retain no built value between calls. Each `*kernel.Compiled` that `Render` returns carries a `cue.Value` into that render's build, so a caller that holds one keeps the build alive until it releases it; retention is bounded by what the caller holds, the rule ADR-007 states for an acquired artifact's `Package`. ADR-005 and the `opm/kernel` package documentation SHALL state this rule and SHALL NOT state that a caller cannot obtain a built value to hold. ADR-005 SHALL record that Render output moves from `cue.Value` to bytes before GA at the latest. Source: 0021:D8:R11.

#### Scenario: The documentation states the holder-bounded rule

- **WHEN** a developer reads rule 2 of `adr/005-shares-nothing-renders.md` and the Goroutine safety section of the `opm/kernel` package documentation
- **THEN** both state that the kernel holds no built value between calls and that a held `*Compiled` keeps its render's build alive until the caller releases it
- **AND** neither states that a caller cannot obtain a built value to hold

#### Scenario: The move to bytes is recorded

- **WHEN** a developer reads the Status of `adr/005-shares-nothing-renders.md`
- **THEN** it states that Render output keeps `Compiled.Value` for now and moves to bytes before GA at the latest
