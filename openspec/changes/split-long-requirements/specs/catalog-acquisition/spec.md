## MODIFIED Requirements

### Requirement: Catalog acquisition from a registry and from a directory
The kernel SHALL expose `AcquireCatalogFromRegistry(ctx, modPath, version)` and `AcquireCatalogFromDir(ctx, dirPath)`, returning a typed, source-carrying catalog artifact. Both SHALL reuse the existing acquisition plumbing: the registry path fetches through the same routine module acquisition uses, and both SHALL evaluate and shape-gate through the kernel's one evaluate-and-shape-gate routine.

#### Scenario: A published catalog is acquired by path and version

- **WHEN** `AcquireCatalogFromRegistry` is called with the path and version of a published catalog
- **THEN** the catalog is returned with its evaluated package and its source stamped
- **AND** its transitive dependencies resolve through the registry, with no temporary directory left behind

#### Scenario: A catalog directory is acquired

- **WHEN** `AcquireCatalogFromDir` is called on a directory holding a catalog package
- **THEN** the catalog is returned with its package evaluated and its source stamped in overlay mode
- **AND** the caller's directory is not written to

## REMOVED Requirements

### Requirement: The provider-fulfilled contract set is derived from the catalog
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The catalog artifact exposes the set of contracts it implements as a provider", "The provider set is read from provides when the catalog carries it", "The provider set falls back to a derivation from the catalog's #transformers for an older core or no provides field", "Both sources of the provider set yield the same set", "The provider set is deterministically ordered and deduplicated", "A catalog implementing no provider-fulfilled contract derives an empty set and no error" and "A provider set that cannot be read is an error with no partial set returned".

### Requirement: The catalog's committed dependency requirements are readable
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The catalog artifact exposes the requirements recorded in its own cue.mod/module.cue and the library exposes the requirements only", "The catalog's module file is read and parsed by the same reader the render stage uses for its input modules" and "A dependency a local replacement serves may carry no version and maps to the empty string".

## ADDED Requirements

### Requirement: The two catalog acquisition paths differ only in where the package files come from
The two paths (Requirement: Catalog acquisition from a registry and from a directory) SHALL differ only in where the package files come from.

#### Scenario: The two acquisition paths read the package files from different places

- **WHEN** a catalog is acquired from a registry and another from a directory
- **THEN** the two paths differ only in where the package files come from

### Requirement: Catalog acquisition writes nothing to the caller's directory and builds in a context created for the call
Neither `AcquireCatalogFromRegistry` nor `AcquireCatalogFromDir` SHALL write to the caller's directory, and each SHALL build in a `cue.Context` created for the call, per ADR-007.

#### Scenario: Catalog acquisition writes nothing to the caller's directory

- **WHEN** either catalog acquisition verb is called
- **THEN** neither verb writes to the caller's directory, and each builds in a `cue.Context` created for the call, per ADR-007

### Requirement: The catalog artifact exposes the set of contracts it implements as a provider
The catalog artifact SHALL expose the set of contracts it implements as a provider: every contract required by one of the catalog's own `#transformers` whose value carries `fulfilment: "provider"` (enhancement 0015, "Every field of the claim is derived or stamped; the only authored fact is the module's catalog dependency version").

#### Scenario: A provider catalog's contracts are derived

- **WHEN** the derivation runs over a catalog whose transformers require a contract carrying `fulfilment: "provider"`
- **THEN** that contract's FQN is in the returned set
- **AND** a contract required by those transformers that carries any other fulfilment is not

### Requirement: The provider set is read from provides when the catalog carries it
When the catalog was built against a core that derives `provides` (core `v2.0.0-beta.3` and later) and carries that field, the set (Requirement: The catalog artifact exposes the set of contracts it implements as a provider) SHALL be read from it.

#### Scenario: Core's provider set is read when the catalog carries it

- **WHEN** the derivation runs over a catalog that carries a `provides` field and does not pin a core older than `v2.0.0-beta.3`
- **THEN** the returned set is that field's contracts, sorted and deduplicated
- **AND** the catalog's `#transformers` are not walked to produce it

### Requirement: The provider set falls back to a derivation from the catalog's #transformers for an older core or no provides field
The set (Requirement: The provider set is read from provides when the catalog carries it) SHALL instead be derived from the catalog's `#transformers` by the library's fallback, which is deprecated and removed before GA, when the catalog's committed `cue.mod` pins a core older than `v2.0.0-beta.3`, whether or not the catalog authors a `provides` field, or when the catalog carries no `provides` field at all.

#### Scenario: A catalog built against an older core falls back

- **WHEN** the derivation runs over a catalog whose committed `cue.mod` pins a core older than `v2.0.0-beta.3`
- **THEN** it returns the set derived from the catalog's `#transformers`, with no error
- **AND** a `provides` field the catalog authors beside its embedded `#Catalog` is not read, because an older core does not derive it

### Requirement: Both sources of the provider set yield the same set
For any catalog built against a core that derives `provides`, both sources (Requirement: The provider set is read from provides when the catalog carries it) (Requirement: The provider set falls back to a derivation from the catalog's #transformers for an older core or no provides field) SHALL yield the same set.

#### Scenario: Core's provider set and the fallback agree

- **WHEN** a provider catalog is built against the core the library pins, so it carries `provides`
- **THEN** the set read from `provides` equals, element for element, the set the fallback derives from the same catalog's `#transformers`

### Requirement: The provider set is deterministically ordered and deduplicated
The result (Requirement: The catalog artifact exposes the set of contracts it implements as a provider) SHALL be deterministically ordered and deduplicated, so two derivations of one catalog compare equal without the caller sorting.

#### Scenario: The derived order is stable

- **WHEN** the derivation runs twice over one catalog
- **THEN** the two results are equal element for element

### Requirement: A catalog implementing no provider-fulfilled contract derives an empty set and no error
Derivation SHALL be a report: a catalog implementing no provider-fulfilled contract SHALL return an empty set and no error.

#### Scenario: A catalog implementing no provider contract derives an empty set

- **WHEN** the derivation runs over a catalog none of whose transformers require a provider-fulfilled contract
- **THEN** it returns an empty set and no error, because implementing none is a fact about the catalog and not a malformed value

### Requirement: A provider set that cannot be read is an error with no partial set returned
A provider set that cannot be read SHALL be an error with no partial set returned, from either source: a `#transformers` that does not evaluate, even when a well-formed `provides` is present; a `provides` field that is not a concrete list of strings; a demand whose `fulfilment` is present but not a concrete string; or, for a catalog carrying its source tree, a committed module file that cannot be read or names a core version that is not SemVer.

#### Scenario: An unreadable provider set is reported

- **WHEN** the derivation runs over a catalog whose `provides` field is present but is not a concrete list of strings
- **THEN** it fails with an error naming the `provides` field, and no set is returned

#### Scenario: Transformers that do not evaluate are reported over a present field

- **WHEN** the derivation runs over a catalog that carries a well-formed `provides` field but whose `#transformers` do not evaluate
- **THEN** it fails with an error naming `#transformers`, and no set is returned

#### Scenario: A demand whose fulfilment is not a concrete string is reported

- **WHEN** the derivation runs over a catalog with a demand whose `fulfilment` is present but not a concrete string
- **THEN** it fails with an error, and no partial set is returned

#### Scenario: A committed module file that cannot be read is reported

- **WHEN** the derivation runs over a catalog carrying its source tree, with a committed module file that cannot be read or names a core version that is not SemVer
- **THEN** it fails with an error, and no partial set is returned

### Requirement: The catalog artifact exposes the requirements recorded in its own cue.mod/module.cue and the library exposes the requirements only
The catalog artifact SHALL expose the requirements recorded in its own `cue.mod/module.cue`, as module path to version, so a consumer can compare them against another resolution without re-reading the artifact or reaching for CUE itself. The library SHALL expose the requirements only; comparing them, and deciding what a mismatch means, belong to the consumer.
**Source**: enhancement 0015 ("A build-incompatible registration is refused at acceptance, by comparing committed resolutions under the stability discipline"); the committed-resolution comparison enhancement 0019 ("Skew policy defaults to warn-and-render; the comparison reads the two committed resolutions; older-than-platform is data") defines.

#### Scenario: Declared requirements are readable off an acquired catalog

- **WHEN** a catalog whose `cue.mod/module.cue` requires a core version and a catalog version is acquired
- **THEN** both requirements are readable from the returned artifact, each as a path and a version
- **AND** the library reports no verdict about whether either is acceptable

### Requirement: The catalog's module file is read and parsed by the same reader the render stage uses for its input modules
The module file (Requirement: The catalog artifact exposes the requirements recorded in its own cue.mod/module.cue and the library exposes the requirements only) SHALL be read and parsed by the same reader the render stage uses for its input modules, in either source mode, so a module file the render stage refuses (a dependency carrying `replaceWith`, an empty module path, a file that does not parse) is refused here too, with an error naming the catalog's `cue.mod/module.cue`.

#### Scenario: A module file the render stage would refuse is refused

- **WHEN** an acquired catalog's `cue.mod/module.cue` lists a dependency that carries `replaceWith`
- **THEN** reading its requirements fails with an error naming the catalog's `cue.mod/module.cue`, and no requirements are returned

### Requirement: A dependency a local replacement serves may carry no version and maps to the empty string
A dependency (Requirement: The catalog artifact exposes the requirements recorded in its own cue.mod/module.cue and the library exposes the requirements only) a local replacement serves may carry no version and SHALL map to the empty string.

#### Scenario: A dependency served by a local replacement maps to the empty string

- **WHEN** a dependency is served by a local replacement and carries no version
- **THEN** the requirement maps to the empty string
