## MODIFIED Requirements

### Requirement: Instance verbs refuse an unset required config value

`SynthesizeInstance` and `AcquireInstanceFromDir` SHALL refuse an instance whose effective values leave a required `#config` value unset, whether or not a component reads that value. The effective values are the built spec's `values`: the package's own `values` plus the trailing sources on `AcquireInstanceFromDir`, the merged `InstanceInput.Values` on `SynthesizeInstance`. A required value is one that `cue.Concrete(true)` refuses on those values unified with `#config`: a required field (`foo!:`) that is absent, or a field that resolves to no concrete value (a bare type such as `string` or `_`, with no default). An optional field (`foo?:`) and a field with a default are not required. Both concreteness checks, the one on the built spec and the one on the values unified with `#config`, SHALL run in the kernel's one instance processing step, with `#config` read off the built spec and never off `Module.Package`, so that both verbs refuse the same instances with the same findings.

The refusal SHALL be CUE's own error tree, framed `Kernel.<Verb>: instance "<name>": not fully concrete: …`. When the values leave no required value unset, the refusal SHALL be the error of the built-spec check, unchanged. When they leave one or more unset, the refusal SHALL hold findings about the values only:

1. every finding of the built-spec check at a path under `values` (a value the instance's values carry in a non-concrete form), positioned where the values carry it;
2. one finding for every other unset required value, at the path `values.<field>` (the full path for a nested field), whether or not a component reads it. A finding SHALL carry the position of the field's `#config` declaration wherever CUE records one; a field CUE records no position for (a field declared `_`) is identified by its path only.

A path SHALL be reported once. Every finding of the built-spec check outside `values` SHALL be left out of that refusal: the component fields that read an unset value, and any other incomplete field of the module. Such a field that is still incomplete once the values are complete is then refused with the built-spec error. The kernel SHALL NOT decide from the positions of a finding whether an unset value causes it.

A caller identifies the refusal as it did before this requirement changed: `errors.As` with CUE's error type finds the tree, and a finding whose path starts with `values` is a values finding; the kernel adds no error type for it. When a required value is unset, every finding of the tree is a values finding.

For the same module and the same single values source, `SynthesizeInstance` SHALL refuse every value that `ValidateConfigDetailed(#config, sources)` refuses, and SHALL name at `values.<field>` every unset field that `ValidateConfigDetailed` names at `#config.<field>`; it MAY refuse more only through the concreteness check on the built spec, which refuses a non-concrete value the instance's values carry even where a `#config` default would complete it. Source: library#211, settled by the owner in the pull request that closes it; opm-operator#258 for the report of both checks.

#### Scenario: Synthesis refuses an unread required value

- **WHEN** `SynthesizeInstance` is called for a module whose `#config` declares `replicas: int | *1` and `image: string`, whose only component reads `replicas`, with one values source `replicas: 2`
- **THEN** the call returns no instance and an error framed `Kernel.SynthesizeInstance: instance "<name>": not fully concrete: `
- **AND** the error has a finding at the path `values.image` that carries the position of the `image` declaration in `#config`

#### Scenario: Directory acquisition refuses an unread required value

- **WHEN** `AcquireInstanceFromDir` is called on an instance package of that module whose own `values` set `replicas` and leave `image` unset, once with no sources and once with a source that sets an unrelated declared field
- **THEN** both calls return no instance and an error framed `Kernel.AcquireInstanceFromDir: instance "<name>": not fully concrete: ` with a finding at the path `values.image`

#### Scenario: A required field marker is refused when absent

- **WHEN** either verb is given values that omit a `#config` field declared `tag!: string` that no component reads
- **THEN** the call is refused with CUE's required-field error at the path `values.tag`, positioned at the `tag` declaration

#### Scenario: A field declared `_` is named by its path

- **WHEN** either verb is given values that omit a `#config` field declared `any: _` that no component reads
- **THEN** the call is refused with a finding at the path `values.any`

#### Scenario: Optional and defaulted fields are not required

- **WHEN** either verb is given values that leave unset only `#config` fields declared optional (`opt?: string`) or with a default (`replicas: int | *1`)
- **THEN** the call returns the instance

#### Scenario: The refused set matches ValidateConfigDetailed

- **WHEN** the same module and the same single values source are given to `ValidateConfigDetailed(mod.ConfigSchema(), []Source{src})` and to `SynthesizeInstance`, over values that set every field, leave an unread `string` field unset, leave an unread `foo!` field unset, leave an unread `_` field unset, leave an optional field unset, leave a defaulted field unset, leave a field a component reads only through a hidden field unset, set a key `#config` does not declare, set an unread field to a value of the wrong type, violate a constraint on such a hidden-read field, and the empty document `{}`
- **THEN** `SynthesizeInstance` refuses every case `ValidateConfigDetailed` refuses
- **AND** for a source that gives a defaulted field a bare type (`port: int` where `#config` declares `port: int | *80`), `ValidateConfigDetailed` accepts and `SynthesizeInstance` refuses, through the built-spec check

#### Scenario: Instances refused before keep their error

- **WHEN** an instance package's own `values` carry a non-concrete value (for example `replicas: >=1`) that the built spec exposes
- **THEN** `AcquireInstanceFromDir` refuses it with the same `not fully concrete` error the built-spec check reported before this requirement: a finding at the path `values.replicas` positioned in the package's own values file, and no finding positioned at the module's `#config` declaration

#### Scenario: A value the built spec refuses is reported once

- **WHEN** either verb is given values for a module whose `#config` declares `image: string`, `tag!: string` and `any: _`, none of them read by a component, where a source writes `image: string` and leaves `tag` and `any` unset
- **THEN** the call is refused with exactly one finding at the path `values.image`, the built-spec check's, positioned where the values carry it
- **AND** after it one finding each at `values.tag` and `values.any`, and no other finding

#### Scenario: A value a component reads is named at its field

- **WHEN** either verb is given values that leave unset a `#config` field `message: string` that a component reads into two of its regular fields, one of them through a string interpolation
- **THEN** the call is refused with exactly one finding, at the path `values.message`, positioned at the `message` declaration in `#config`
- **AND** no finding is at a `components` path or at a path of the module the instance unifies

#### Scenario: A value a catalog resource reads is named at its field

- **WHEN** `SynthesizeInstance` is given values that leave unset a `#config` field `note: string` of a module whose component is a catalog resource that reads `note` into three of the resource's fields
- **THEN** the call is refused with exactly one finding, at the path `values.note`, positioned at the `note` declaration in `#config`
- **AND** with the empty document the refusal has one finding for each required value and none at a `components` path

#### Scenario: Every unset value is named, read or not, nested or not

- **WHEN** `SynthesizeInstance` is given the empty document for a module whose `#config` declares `message: string` and `db: host: string`, both read by a component, `other: int`, read by none, and `port: int & >0 | *80`
- **THEN** the refusal has exactly three findings, at `values.message`, `values.other` and `values.db.host`, each positioned at its `#config` declaration
- **AND** the three paths are the fields `ValidateConfigDetailed` names at `#config.<field>` for the same source

#### Scenario: A component defect waits for complete values

- **WHEN** either verb is given values that leave `message` unset for a module whose component reads `message` and also declares a regular field `loose: string` that reads no `#config` value
- **THEN** the refusal has exactly one finding, at `values.message`
- **AND** with every value set, the refusal is the built-spec finding at the component's `loose` path, with the text it had before this requirement changed

#### Scenario: A written bare type hides the component findings it causes

- **WHEN** `SynthesizeInstance` is given a source that writes `message: string` for a field a component reads, with every other required value set
- **THEN** the refusal has exactly one finding, at `values.message`, positioned where the values carry it

#### Scenario: Other values defects keep their text

- **WHEN** `SynthesizeInstance` is given, for a module whose required values are otherwise all set, a value of the wrong type, a value that violates a constraint, or a key `#config` does not declare
- **THEN** each refusal has the text it had before this requirement changed, and none is framed `not fully concrete`
