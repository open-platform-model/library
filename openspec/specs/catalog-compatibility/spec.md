# catalog-compatibility Specification

## Purpose

Publish-side catalog compatibility as a pure library surface (`opm/compat`): enhancement 0010 D27's additive-only comparison walk, D34 contract-level classification, D9 predecessor selection, and the D30 provenance strip. Shared by the 0011 publish gate (`opm catalog publish`), `opm catalog registry check --compat`, and library-matching's unify rung. Pure `cue.Value` logic — no I/O, no schema-cache dependency; callers compose enumeration, fetching, and gate policy.

## Requirements

### Requirement: Compatibility Comparison

The library SHALL provide a pure comparison (`opm/compat.Check`) that, given a prior and a new definition of the same contract as `cue.Value`s, reports every violation of 0010 D27's additive-only rule: fields and options MAY be added and MUST NOT be removed; a newly added field MUST be optional or defaulted; an existing field's default MUST NOT change. The comparison SHALL be a field-wise walk — struct recursion for the removed-field and optional-or-defaulted rules, forward subsumption at leaves for the value domain, and explicit default comparison at every level — and SHALL NOT be implemented as a single subsumption call in either direction. Each violation SHALL carry the dotted path from the compared root and a stable kind discriminator; default-change violations SHALL carry the rendered old and new values.

The walk SHALL apply 0010 D30's provenance denylist at every depth: the direct children `catalogVersion` and `description` of any field named `metadata` reached by the walk SHALL be neither compared nor reported, so a member reference embedded in another member (`appliesTo`, `composedResources`, `composedTraits`) does not report the referenced member's per-release provenance. Lists whose two sides have equal length SHALL be walked element-wise so their elements reach that rule; lists of unequal length SHALL be judged as a leaf. A leaf whose emitted syntax is identical on both sides SHALL report nothing.

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

The library SHALL provide a stable-preferring version selector over a published-version list (`opm/compat.HighestStable`): given the registry's `v`-prefixed SemVer-ascending list, it SHALL return the highest stable (non-prerelease) version, skipping unparseable entries, and SHALL fall back to the highest version overall when no stable version exists. Selection SHALL be pure — enumeration and fetching are the caller's.

This selector is NOT the compatibility gate's predecessor selection. 0011 D23 (amending D9) settled that the publish gate's predecessor is found by the literal rule — scan published versions strictly below the effective version, same major, prereleases included, newest first, each member resolving against the newest build carrying its `name` at its `apiVersion` — implemented gate-side in the CLI over the registry's version enumeration; a stable-preferring selector coincides with that rule only on a prerelease-only history. `HighestStable` is the *float* selector: its first true caller is template resolution's version selection (`cli-template-modules`), which is why it stays.

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

Known limitation (measured 2026-08-16, cue v0.17.1): the inline-imports round-trip is not self-contained for members typed against core — the emitted syntax references core's hidden `#KebabToPascal` helper (reached via `metadata.#definitionName`) without inlining it, and the rebuild fails with `reference "#KebabToPascal" not found`. Every real catalog member is core-typed, so the strip currently works only on values whose imports carry no hidden-definition helpers. The CLI's publish gate applies the D30 denylist as a violation-path filter over `Check`'s output instead (same fields, same direct-children-of-metadata scope). The unify-rung consumer (library-matching) should measure before relying on the strip for core-typed members.
