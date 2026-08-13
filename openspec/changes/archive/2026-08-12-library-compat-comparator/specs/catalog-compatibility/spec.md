# catalog-compatibility — Delta

## ADDED Requirements

### Requirement: Compatibility Comparison

The library SHALL provide a pure comparison (`opm/compat.Check`) that, given a prior and a new definition of the same contract as `cue.Value`s, reports every violation of 0010 D27's additive-only rule: fields and options MAY be added and MUST NOT be removed; a newly added field MUST be optional or defaulted; an existing field's default MUST NOT change. The comparison SHALL be a field-wise walk — struct recursion for the removed-field and optional-or-defaulted rules, forward subsumption at leaves for the value domain, and explicit default comparison at every level — and SHALL NOT be implemented as a single subsumption call in either direction. Each violation SHALL carry the dotted path from the compared root and a stable kind discriminator; default-change violations SHALL carry the rendered old and new values.

#### Scenario: Field removal reported

- **WHEN** the new definition lacks a field the prior definition declares
- **THEN** `Check` reports one violation at that field's path with kind `field removed`

#### Scenario: Strict field addition reported

- **WHEN** the new definition adds a required field with no default
- **THEN** `Check` reports kind `field added without optional or default` at that path
- **AND** adding the same field as optional or defaulted reports nothing

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

### Requirement: Level Classification

The library SHALL classify a primitive's `apiVersion` on the `vNalphaM | vNbetaM | vN` ladder (`opm/compat.ParseLevel`) using the exact grammar of core's `#APIVersionType`, and SHALL expose whether a level is enforced (`Enforced`: beta and GA yes, alpha no). The level-aware entry point (`CheckAtLevel`) SHALL return no violations and no error for an alpha `apiVersion` without evaluating the operands, and SHALL return an error — not a violation — for an `apiVersion` the grammar rejects. The library SHALL provide a total, transitive ordering over valid `apiVersion` strings (`CompareAPIVersions`).

#### Scenario: Alpha is not gated

- **WHEN** `CheckAtLevel` is called with `apiVersion: "v1alpha1"` and operands that differ incompatibly
- **THEN** it returns no violations and no error

#### Scenario: Beta and GA are gated

- **WHEN** `CheckAtLevel` is called with `"v1beta1"`, `"v1"`, or `"v2"` and incompatible operands
- **THEN** it returns the same violations `Check` reports

#### Scenario: Unparseable apiVersion is an error

- **WHEN** `CheckAtLevel` is called with an `apiVersion` core's `#APIVersionType` rejects (e.g. `"v1alpha"`, `"1.2.0"`)
- **THEN** it returns a non-nil error and no violations

#### Scenario: Ordering is transitive

- **WHEN** `CompareAPIVersions` orders `v1alpha1`, `v2`, and `v10`
- **THEN** the resulting order is the same regardless of input order

### Requirement: Predecessor Selection

The library SHALL provide predecessor selection over a published-version list (`opm/compat.HighestStable`): given the registry's `v`-prefixed SemVer-ascending list, it SHALL return the highest stable (non-prerelease) version, skipping unparseable entries, and SHALL fall back to the highest version overall when no stable version exists. Selection SHALL be pure — enumeration and fetching are the caller's.

#### Scenario: Stable preferred over higher prerelease

- **WHEN** the list contains `v1.2.0` and `v1.3.0-alpha.1`
- **THEN** `HighestStable` returns `v1.2.0`

#### Scenario: Prerelease-only fallback

- **WHEN** every entry is a prerelease
- **THEN** `HighestStable` returns the highest entry

### Requirement: Provenance Stripping

The library SHALL provide a provenance strip (`opm/compat.StripProvenance`) implementing 0010 D30's denylist — exactly `metadata.catalogVersion` and `metadata.description` — via a syntax round-trip that reaches the definition as well as the instance, preserves closedness, and inlines imports so the result is self-contained. All other fields, including identity fields and labels, SHALL remain in the value.

#### Scenario: Comparison across catalog releases

- **WHEN** two builds of an identical primitive differ only in `metadata.catalogVersion`
- **THEN** `Check` over the stripped operands reports no violations

#### Scenario: Definition-side strip

- **WHEN** the input value's definition declares `catalogVersion!`
- **THEN** the stripped value does not carry an unsatisfiable required field
