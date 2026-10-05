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

The catalog artifact SHALL expose the set of contracts it implements as a provider: every contract required by one of the catalog's own `#transformers` whose value carries `fulfilment: "provider"` (0015:D11). When the catalog was built against a core that derives `provides` (core `v2.0.0-beta.3` and later) and carries that field, the set SHALL be read from it. The set SHALL instead be derived from the catalog's `#transformers` by the library's fallback, which is deprecated and removed before GA, when the catalog's committed `cue.mod` pins a core older than `v2.0.0-beta.3`, whether or not the catalog authors a `provides` field, or when the catalog carries no `provides` field at all. For any catalog built against a core that derives `provides`, both sources SHALL yield the same set. The result SHALL be deterministically ordered and deduplicated, so two derivations of one catalog compare equal without the caller sorting. Derivation SHALL be a report: a catalog implementing no provider-fulfilled contract SHALL return an empty set and no error. A provider set that cannot be read SHALL be an error with no partial set returned, from either source: a `#transformers` that does not evaluate, even when a well-formed `provides` is present; a `provides` field that is not a concrete list of strings; a demand whose `fulfilment` is present but not a concrete string; or, for a catalog carrying its source tree, a committed module file that cannot be read or names a core version that is not SemVer.

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

#### Scenario: Core's provider set is read when the catalog carries it

- **WHEN** the derivation runs over a catalog that carries a `provides` field and does not pin a core older than `v2.0.0-beta.3`
- **THEN** the returned set is that field's contracts, sorted and deduplicated
- **AND** the catalog's `#transformers` are not walked to produce it

#### Scenario: Core's provider set and the fallback agree

- **WHEN** a provider catalog is built against the core the library pins, so it carries `provides`
- **THEN** the set read from `provides` equals, element for element, the set the fallback derives from the same catalog's `#transformers`

#### Scenario: A catalog built against an older core falls back

- **WHEN** the derivation runs over a catalog whose committed `cue.mod` pins a core older than `v2.0.0-beta.3`
- **THEN** it returns the set derived from the catalog's `#transformers`, with no error
- **AND** a `provides` field the catalog authors beside its embedded `#Catalog` is not read, because an older core does not derive it

#### Scenario: An unreadable provider set is reported

- **WHEN** the derivation runs over a catalog whose `provides` field is present but is not a concrete list of strings
- **THEN** it fails with an error naming the `provides` field, and no set is returned

#### Scenario: Transformers that do not evaluate are reported over a present field

- **WHEN** the derivation runs over a catalog that carries a well-formed `provides` field but whose `#transformers` do not evaluate
- **THEN** it fails with an error naming `#transformers`, and no set is returned

### Requirement: The catalog's committed dependency requirements are readable

The catalog artifact SHALL expose the requirements recorded in its own `cue.mod/module.cue`, as module path to version, so a consumer can compare them against another resolution without re-reading the artifact or reaching for CUE itself (0015:D8; the committed-resolution comparison 0019:D18 defines). The library SHALL expose the requirements only; comparing them, and deciding what a mismatch means, belong to the consumer. The module file SHALL be read and parsed by the same reader the render stage uses for its input modules, in either source mode, so a module file the render stage refuses (a dependency carrying `replaceWith`, an empty module path, a file that does not parse) is refused here too, with an error naming the catalog's `cue.mod/module.cue`. A dependency a local replacement serves may carry no version and maps to the empty string.

#### Scenario: Declared requirements are readable off an acquired catalog

- **WHEN** a catalog whose `cue.mod/module.cue` requires a core version and a catalog version is acquired
- **THEN** both requirements are readable from the returned artifact, each as a path and a version
- **AND** the library reports no verdict about whether either is acceptable

#### Scenario: A module file the render stage would refuse is refused

- **WHEN** an acquired catalog's `cue.mod/module.cue` lists a dependency that carries `replaceWith`
- **THEN** reading its requirements fails with an error naming the catalog's `cue.mod/module.cue`, and no requirements are returned
