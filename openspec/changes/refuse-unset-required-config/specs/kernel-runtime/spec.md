## ADDED Requirements

### Requirement: Instance verbs refuse an unset required config value

`SynthesizeInstance` and `AcquireInstanceFromDir` SHALL refuse an instance whose effective values leave a required `#config` value unset, whether or not a component reads that value. The effective values are the built spec's `values`: the package's own `values` plus the trailing sources on `AcquireInstanceFromDir`, the merged `InstanceInput.Values` on `SynthesizeInstance`. A required value is one that `cue.Concrete(true)` refuses on `#config` unified with those values: a required field (`foo!:`) that is absent, or a field that resolves to no concrete value (a bare type such as `string` or `_`, with no default). An optional field (`foo?:`) and a field with a default are not required. The check SHALL run in the kernel's one instance processing step, after the concreteness check on the built spec, with `#config` read off the built spec and never off `Module.Package`, so that both verbs refuse the same instances and every instance refused before this check existed is refused with the same error as before. The refusal SHALL be CUE's own error tree, framed `Kernel.<Verb>: instance "<name>": not fully concrete: …`, positioned at the field's `#config` declaration, and also at the `Origin` of a values source that wrote the non-concrete value. For the same module and the same values, the set of instances this check refuses SHALL equal the set `ValidateConfigDetailed(#config, sources)` refuses for missing or non-concrete values. The failure-path attribution of a build that fails SHALL stay without concreteness. Source: library#211.

#### Scenario: Synthesis refuses an unread required value

- **WHEN** `SynthesizeInstance` is called for a module whose `#config` declares `replicas: int | *1` and `image: string`, whose only component reads `replicas`, with one values source `replicas: 2`
- **THEN** the call returns no instance and an error framed `Kernel.SynthesizeInstance: instance "<name>": not fully concrete: `
- **AND** the error names the `image` field and carries the position of its `#config` declaration

#### Scenario: Directory acquisition refuses an unread required value

- **WHEN** `AcquireInstanceFromDir` is called on an instance package of that module whose own `values` set `replicas` and leave `image` unset, once with no sources and once with a source that sets an unrelated declared field
- **THEN** both calls return no instance and an error framed `Kernel.AcquireInstanceFromDir: instance "<name>": not fully concrete: ` naming the `image` field

#### Scenario: A required field marker is refused when absent

- **WHEN** either verb is given values that omit a `#config` field declared `tag!: string` that no component reads
- **THEN** the call is refused with CUE's required-field error for `tag`

#### Scenario: Optional and defaulted fields are not required

- **WHEN** either verb is given values that leave unset only `#config` fields declared optional (`opt?: string`) or with a default (`replicas: int | *1`)
- **THEN** the call returns the instance

#### Scenario: A source that writes a non-concrete value is named

- **WHEN** a values source with `Origin` `/values/a.cue` writes `image: string` for a `#config` field no component reads
- **THEN** the refusal carries a position naming `/values/a.cue`

#### Scenario: The refused set matches ValidateConfigDetailed

- **WHEN** the same module and the same single values source are given to `ValidateConfigDetailed(mod.ConfigSchema(), []Source{src})` and to `SynthesizeInstance`, over values that set every field, leave an unread `string` field unset, leave an unread `foo!` field unset, leave an unread `_` field unset, leave an optional field unset, leave a defaulted field unset, and leave a field a component reads unset
- **THEN** `SynthesizeInstance` refuses exactly the cases `ValidateConfigDetailed` refuses

#### Scenario: Instances refused before keep their error

- **WHEN** an instance package's own `values` carry a non-concrete value (for example `replicas: >=1`) that the built spec exposes
- **THEN** `AcquireInstanceFromDir` refuses it with the same `not fully concrete` error the built-spec check reported before this requirement, and no values error

#### Scenario: A build failure still returns the build error

- **WHEN** a build fails for a reason the values do not explain, while the values leave a required `#config` field unset, through either verb
- **THEN** the call returns its build error, not the required-field refusal
