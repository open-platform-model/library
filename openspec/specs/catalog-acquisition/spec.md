# catalog-acquisition Specification

## Purpose
Give the library a first-class primitive for reading a `#Catalog`: fetched from an OCI registry by `path@version`, or loaded from a directory. The kernel already owns module-acquisition plumbing (Principle V), and a catalog is a CUE module artifact carrying a different core kind, so the two paths differ only in the shape they gate to and the type they return. The capability also derives the provider-fulfilled contract set a catalog implements, because that set is a function of the catalog value and a core-defined concept, not a consumer's policy.

This capability reads and derives. It renders no verdict: whether a derived set matches a claim, whether a version is compatible, and what any refusal says belong to the consumer (enhancement 0015 D8, D11, D12).
## Requirements

### Requirement: Catalog acquisition from a registry and from a directory

The kernel SHALL expose `AcquireCatalogFromRegistry(ctx, modPath, version)` and `AcquireCatalogFromDir(ctx, dirPath)`, returning a typed, source-carrying catalog artifact. Both SHALL reuse the existing acquisition plumbing: the registry path fetches through the same routine module acquisition uses, and both SHALL evaluate and shape-gate through the kernel's one evaluate-and-shape-gate routine. The two paths SHALL differ only in where the package files come from. Neither SHALL write to the caller's directory, and each SHALL build in a `cue.Context` created for the call, per ADR-007.

#### Scenario: A published catalog is acquired by path and version

- **WHEN** `AcquireCatalogFromRegistry` is called with the path and version of a published catalog
- **THEN** the catalog is returned with its evaluated package and its source stamped
- **AND** its transitive dependencies resolve through the registry, with no temporary directory left behind

#### Scenario: A catalog directory is acquired

- **WHEN** `AcquireCatalogFromDir` is called on a directory holding a catalog package
- **THEN** the catalog is returned with its package evaluated and its source stamped in overlay mode
- **AND** the caller's directory is not written to

### Requirement: A non-catalog artifact is refused by shape

Acquisition SHALL gate the evaluated value to a concrete `kind` of `"Catalog"` and SHALL refuse anything else with an error wrapping `ErrWrongKind`, naming the kind it found. A module artifact is therefore refused structurally rather than by a maintained rule (enhancement 0015 D10). Acquisition SHALL NOT perform full schema validation, which remains the kernel's contract, and SHALL NOT return a partial catalog on any gate failure.

#### Scenario: A module artifact claimed as a catalog is refused

- **WHEN** acquisition resolves an artifact whose concrete `kind` is `"Module"`
- **THEN** it fails with an error wrapping `ErrWrongKind` naming both the expected and the found kind
- **AND** no catalog value is returned

#### Scenario: An artifact carrying no kind is refused

- **WHEN** the resolved artifact has no `kind` field, or a `kind` that is not a concrete string
- **THEN** it fails with an error wrapping `ErrWrongKind`

### Requirement: The provider-fulfilled contract set is derived from the catalog

The catalog artifact SHALL expose the set of contracts it implements as a provider: every contract required by one of the catalog's own `#transformers` whose value carries `fulfilment: "provider"` (enhancement 0015 D11). The result SHALL be deterministically ordered, so two derivations of one catalog compare equal without the caller sorting. Derivation SHALL be a report: a catalog implementing no provider-fulfilled contract SHALL return an empty set and no error.

#### Scenario: A provider catalog's contracts are derived

- **WHEN** the derivation runs over a catalog whose transformers require a contract carrying `fulfilment: "provider"`
- **THEN** that contract's FQN is in the returned set
- **AND** a contract required by those transformers that carries any other fulfilment is not

#### Scenario: A catalog implementing no provider contract derives an empty set

- **WHEN** the derivation runs over a catalog none of whose transformers require a provider-fulfilled contract
- **THEN** it returns an empty set and no error, because implementing none is a fact about the catalog and not a malformed value

#### Scenario: The derived order is stable

- **WHEN** the derivation runs twice over one catalog
- **THEN** the two results are equal element for element

### Requirement: The catalog's committed dependency requirements are readable

The catalog artifact SHALL expose the requirements recorded in its own `cue.mod/module.cue`, as module path to version, so a consumer can compare them against another resolution without re-reading the artifact or reaching for CUE itself (enhancement 0015 D8, the committed-resolution comparison 0019 D18 defines). The library SHALL expose the requirements only; comparing them, and deciding what a mismatch means, belong to the consumer.

#### Scenario: Declared requirements are readable off an acquired catalog

- **WHEN** a catalog whose `cue.mod/module.cue` requires a core version and a catalog version is acquired
- **THEN** both requirements are readable from the returned artifact, each as a path and a version
- **AND** the library reports no verdict about whether either is acceptable
