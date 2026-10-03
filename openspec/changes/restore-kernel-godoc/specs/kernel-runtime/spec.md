## ADDED Requirements

### Requirement: The kernel package doc renders its verb list and examples

The `opm/kernel` package documentation SHALL present the `Surface` section's verbs as a Go doc list, with one item per verb group, and SHALL present each code example (the one-Kernel-per-process `renderAll` example, the diagnostics wording example and the replacements wording example) as a Go doc code block. When parsed with `go/doc` and `go/doc/comment`, the package doc SHALL yield a `*comment.List` after the `Surface` heading and one `*comment.Code` block per example. A test in `opm/kernel` SHALL parse the package doc this way and SHALL fail when the list or any of the code blocks stops being one.

#### Scenario: go doc renders the Surface list as a list

- **WHEN** a developer runs `go doc ./opm/kernel`
- **THEN** each verb group under `Surface` (the module, catalog, platform and instance acquire verbs, `SynthesizeInstance`, `ValidateConfigDetailed` and `Render`) is printed as its own list item

#### Scenario: go doc renders each example as code

- **WHEN** a developer runs `go doc ./opm/kernel`
- **THEN** the `renderAll` example, the diagnostics wording example and the replacements wording example are printed as indented code with one statement per line, not as wrapped prose

#### Scenario: Flattening the Surface list fails the guard test

- **WHEN** a change rewraps the `Surface` list in `opm/kernel/doc.go` into a single paragraph
- **THEN** `go test ./opm/kernel` fails, naming the `Surface` list and the block type it found instead

#### Scenario: Flattening a code example fails the guard test

- **WHEN** a change rewraps any of the three code examples in `opm/kernel/doc.go` into prose
- **THEN** `go test ./opm/kernel` fails, naming the missing code block

#### Scenario: Flattening one loop of an example fails the guard test

- **WHEN** a change rewraps a single loop of a code example in `opm/kernel/doc.go` into prose and leaves the rest of the example as code
- **THEN** `go test ./opm/kernel` fails, naming the code fragment it found outside a code block
