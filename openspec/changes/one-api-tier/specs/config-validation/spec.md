## MODIFIED Requirements

### Requirement: Source Type and Layered Input

The library SHALL expose a `Source` struct in `opm/kernel/` describing one values input for every values-taking kernel entry: `ValidateConfigDetailed`, the variadic values of `AcquireInstanceFromDir`, and `InstanceInput.Values` on `SynthesizeInstance`. A `Source` pairs the values payload with its stable origin; it SHALL carry no display label, since presentation is outside the kernel's contract and CUE positions carry the origin. No kernel entry SHALL take values in any other shape.

#### Scenario: Source struct shape

- **WHEN** a frontend constructs a `kernel.Source`
- **THEN** the struct exposes exactly two fields: `Value cue.Value` (the values payload) and `Origin string` (the stable identifier CUE positions report)
- **AND** the godoc on `Source.Value` states that the value MUST have been compiled with `cue.Filename(Origin)` for per-source attribution to flow through into errors

#### Scenario: Partial option

- **WHEN** a consumer inspects the exported identifiers of `opm/kernel`
- **THEN** neither a `ValidateOption` type nor a `Partial` constructor exists
- **AND** `ValidateConfigDetailed` takes exactly two arguments, `schema` and `sources`

#### Scenario: Stack ordering for layered inputs

- **WHEN** a frontend constructs `[]Source{a, b, c}` and passes it to `ValidateConfigDetailed`, as the trailing arguments of `AcquireInstanceFromDir`, or as `InstanceInput.Values`
- **THEN** unification proceeds `a → a∪b → a∪b∪c`
- **AND** field conflicts resolve to the layer that wrote them last

#### Scenario: One values shape everywhere

- **WHEN** a frontend has values as a file, as bytes, or as a `cue.Value` it compiled with a filename
- **THEN** it wraps them once as `Source` values (`LoadSourceFromFile`, `LoadSourceFromBytes`, or a hand-built `Source`) and passes the same slice to validation, directory acquisition or synthesis without conversion
