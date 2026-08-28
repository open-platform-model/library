## MODIFIED Requirements

### Requirement: Compatibility Comparison

The library SHALL provide a pure comparison (`opm/compat.Check`) that, given a prior and a new definition of the same contract as `cue.Value`s, reports every violation of 0010 D27's additive-only rule: fields and options MAY be added and MUST NOT be removed; a newly added field MUST be optional or defaulted; an existing field's default MUST NOT change; an existing optional or defaulted field MUST NOT become required. The comparison SHALL be a field-wise walk — struct recursion for the removed-field, optional-or-defaulted and made-required rules, forward subsumption at leaves for the value domain, and explicit default comparison at every level — and SHALL NOT be implemented as a single subsumption call in either direction. Each violation SHALL carry the dotted path from the compared root and a stable kind discriminator; default-change violations SHALL carry the rendered old and new values.

A field is "required" for this rule when it carries the required constraint (`!`) or is a regular field with no default. A field that carried the optional constraint (`?`) in the prior definition and is required in the new one SHALL report kind `field made required` at its path, independently of whether its value domain also changed. A defaulted field that loses its default is the default rule's finding (`default removed`), not this one's. The reverse transition (required to optional, or an optional field gaining a default) SHALL NOT report this kind.

The walk SHALL apply 0010 D30's provenance denylist at every depth: the direct children `catalogVersion` and `description` of any field named `metadata` reached by the walk SHALL be neither compared nor reported, so a member reference embedded in another member (`appliesTo`, `composedResources`, `composedTraits`) does not report the referenced member's per-release provenance. Lists whose two sides have equal length SHALL be walked element-wise so their elements reach that rule; lists of unequal length SHALL be judged as a leaf. A leaf whose emitted syntax is identical on both sides SHALL report nothing.

#### Scenario: Field removal reported

- **WHEN** the new definition lacks a field the prior definition declares
- **THEN** `Check` reports one violation at that field's path with kind `field removed`

#### Scenario: Strict field addition reported

- **WHEN** the new definition adds a required field with no default
- **THEN** `Check` reports kind `field added without optional or default` at that path
- **AND** adding the same field as optional or defaulted reports nothing

#### Scenario: Field made required reported

- **WHEN** a field declared `y?: string` in the prior definition is declared `y!: string` in the new one
- **THEN** `Check` reports kind `field made required` at `y`
- **AND** declaring it `y: string` (regular, no default) reports the same kind
- **AND** declaring it `y!: =~"^[a-z]"` reports both `field made required` and `domain narrowed` at `y`
- **AND** the reverse (`y!: string` → `y?: string`, or `y: string` → `y: string | *"z"`) reports no `field made required`

#### Scenario: Default change reported with both values

- **WHEN** an existing field's default changes
- **THEN** `Check` reports kind `default changed` at that path carrying the old and new rendered values
- **AND** neither direction of whole-value subsumption would have detected it

#### Scenario: Domain narrowing reported at a leaf

- **WHEN** a leaf's value domain no longer accepts a value the prior domain accepted
- **THEN** `Check` reports kind `domain narrowed` at that path carrying the CUE subsumption diagnostic verbatim

#### Scenario: Additive widening passes

- **WHEN** the new definition only adds an option to a disjunction or adds an optional field
- **THEN** `Check` reports no violations

#### Scenario: Provenance inside a member reference is ignored

- **WHEN** a trait's `appliesTo` list references a resource whose `metadata.catalogVersion` default differs between the two builds and nothing else differs
- **THEN** `Check` reports no violations

#### Scenario: Unchanged guarded leaf is silent

- **WHEN** a field typed by a struct carrying `if`-guards over not-yet-concrete siblings (the `#Image` shape) or a `matchN` validator is byte-identical on both sides
- **THEN** `Check` reports no violations at that path

#### Scenario: Equal-length lists are walked

- **WHEN** a blueprint's `composedResources` list has the same length on both sides and one element removed a field
- **THEN** `Check` reports `field removed` at `composedResources[<i>].<field>`

#### Scenario: Unequal-length lists are a leaf

- **WHEN** the new definition drops an element from a list
- **THEN** `Check` reports `domain narrowed` at the list's path

Known limitation (measured 2026-08-16, cue v0.17.1; narrowed 2026-08-26): a leaf that both changed and carries a `matchN` validator or a pending comprehension is judged by the forward subsume alone, which cannot evaluate those constructs: narrowing `matchN`'s alternatives is not detected and widening them reports `domain narrowed` spuriously. Unchanged leaves no longer report. A comparator-level fix for the changed-leaf residue is unowned.
