## MODIFIED Requirements

### Requirement: Source Type and Layered Input

The library SHALL expose a `Source` struct in `opm/kernel/` describing one values input for every values-taking kernel entry: `ValidateConfigDetailed`, the variadic values of `AcquireInstanceFromDir`, and `InstanceInput.Values` on `SynthesizeInstance`. A `Source` pairs the values payload, as CUE source bytes, with its stable origin; it SHALL carry no `cue.Value` and no display label, since a value is bound to the context that built it, presentation is outside the kernel's contract, and CUE positions carry the origin. Each kernel entry SHALL compile the sources it receives in the context of the schema they meet, with `cue.Filename(Origin)`, so every position in every error names the origin. No kernel entry SHALL take values in any other shape.

#### Scenario: Source struct shape

- **WHEN** a frontend constructs a `kernel.Source`
- **THEN** the struct exposes exactly two fields: `Data []byte` (the values payload as CUE source) and `Origin string` (the stable identifier CUE positions report)
- **AND** the godoc on `Source` states that the kernel compiles `Data` with `cue.Filename(Origin)` in the context of the operation that uses it

#### Scenario: Partial option

- **WHEN** a consumer inspects the exported identifiers of `opm/kernel`
- **THEN** neither a `ValidateOption` type nor a `Partial` constructor exists
- **AND** `ValidateConfigDetailed` takes exactly two arguments, `schema` and `sources`

#### Scenario: Stack ordering for layered inputs

- **WHEN** a frontend constructs `[]Source{a, b, c}` and passes it to `ValidateConfigDetailed`, as the trailing arguments of `AcquireInstanceFromDir`, or as `InstanceInput.Values`
- **THEN** unification proceeds `a → a∪b → a∪b∪c`
- **AND** field conflicts resolve to the layer that wrote them last

#### Scenario: One values shape everywhere

- **WHEN** a frontend has values as a file or as bytes
- **THEN** it wraps them once as `Source` values (`LoadSourceFromFile`, `LoadSourceFromBytes`, or a hand-built `Source`) and passes the same slice to validation, directory acquisition or synthesis without conversion
- **AND** it needs no `cue.Context` of its own to do so

#### Scenario: Positions survive compilation at use

- **WHEN** a source's `Data` violates a module's `#config` on its third line
- **THEN** the error the kernel returns positions the violation at `Origin`, line 3

### Requirement: Source Loader Helpers

The library SHALL expose two loader helpers on `*Kernel` that produce `Source` values: `LoadSourceFromFile` for a values file on disk and `LoadSourceFromBytes` for an in-memory payload. Both SHALL parse the payload for syntax and return a syntax error positioned at `Origin`; neither SHALL evaluate CUE, since evaluation happens in the context of the operation that uses the source. It SHALL NOT expose a string-input variant; a string is `[]byte(s)`.

#### Scenario: LoadSourceFromFile

- **WHEN** a caller invokes `k.LoadSourceFromFile(path string)`
- **THEN** the returned `Source` has `Origin` equal to the absolute path and `Data` equal to the file's bytes

#### Scenario: LoadSourceFromBytes

- **WHEN** a caller invokes `k.LoadSourceFromBytes(origin string, b []byte)`
- **THEN** the returned `Source` has `Origin = origin` and `Data = b`
- **AND** validation errors on the compiled source report `pos.Filename() == origin`

#### Scenario: Syntax errors fail at load

- **WHEN** either helper receives a payload that does not parse as CUE
- **THEN** it returns an error positioned at `Origin` and no `Source`

#### Scenario: LoadSourceFromString

- **WHEN** a consumer inspects the exported methods of `Kernel`
- **THEN** `LoadSourceFromString` does not exist
- **AND** a caller holding a string compiles it via `k.LoadSourceFromBytes(origin, []byte(s))`
