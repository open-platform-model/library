## MODIFIED Requirements

### Requirement: The single-provider guard runs inside the build

The generated glue SHALL read the platform's provider count from core's derived contract inventory and SHALL NOT compute a count of its own. The count is `#Platform.#contracts.providedBy`: for every contract key declared `fulfilment: "provider"` on a required demand (`requiredResources` and `requiredTraits`) of any transformer of an enabled `#registry` entry, the sorted registry keys (the catalog module paths with their major, bound by core to each entry's `#catalog.metadata.modulePath`) of the entries whose transformers require it, whether or not any enabled entry defines the contract. Provenance is the registry key, never a value parsed out of an FQN or read off the transformer, so two majors of one catalog are two providers and two transformers of one entry are one. Every key core lists in `#contracts.overSubscribed` SHALL be reported as an over-subscription row naming the key and the registry keys `providedBy` holds for it, and the render SHALL refuse through the fail-closed gate, which reads `#contracts.routable`, with one typed over-subscription cause carrying every such row. The rows SHALL be built by iterating `providedBy` unconditionally, so a platform value lacking it fails the build instead of reading as zero providers. Keys with default (`catalog`) fulfilment MAY be supplied by any number of transformers from any number of catalogs. Because both the render and the platform's `Contracts()` read this one count, a render SHALL refuse on over-subscription exactly when the platform's inventory reads `Routable: false`.

Source: 0010:D37 and 0015:D2/D18, as recounted by core change `count-providers-per-registry-entry`.

#### Scenario: Second provider refused in-build

- **WHEN** two catalogs embedded in the platform each supply a transformer requiring a contract declared `fulfilment: "provider"`
- **THEN** `Render` fails with a typed over-subscription cause whose row names the key and both registry keys, and the diagnostics remain readable beside the refusal

#### Scenario: Catalog-fulfilled plurality allowed

- **WHEN** many transformers across catalogs require a contract with default fulfilment
- **THEN** the render proceeds and every candidate participates in matching

#### Scenario: Two majors of one provider catalog are two providers

- **WHEN** the platform enables `cat@v0` and `cat@v1`, and a transformer of each requires `cat@v0`'s provider-fulfilled gateway contract
- **THEN** `Render` fails with one over-subscription row naming the gateway key and the registry keys `cat@v0` and `cat@v1`, and the platform's `Contracts()` reads the same key in `OverSubscribed` and `Routable` false

#### Scenario: Providers count when the defining catalog is disabled

- **WHEN** the platform carries the defining `cat@v0` entry with `enable: false` and enables `cat2` 0.2.0 and `cat@v1`, whose transformers both require the gateway contract
- **THEN** `Render` fails with one over-subscription row naming the gateway key and the registry keys `cat2@v0` and `cat@v1`, and the platform's `Contracts()` reads the same key in `OverSubscribed` and `Routable` false

#### Scenario: Inventory and render agree on every served platform

- **WHEN** the parity test acquires every served render platform (each `testdata/render/platform*` directory), reads `Contracts()` and renders the `instance` fixture against it
- **THEN** the sorted `OverSubscribed` list equals the keys of the render's over-subscription rows, `Routable` is true exactly when there are no rows, and every row's registry keys equal `ProvidedBy` for its key, whether or not the render also refuses for an unrelated reason

### Requirement: A caller may skip unprovided provider-fulfilled demands

`Kernel.Render` SHALL accept a per-render switch, off by default, that asks the render to skip every **unprovided** demand: a resource or trait demand whose contract declares `fulfilment: "provider"` and for which no enabled registry entry carries a transformer requiring that contract key (the key is absent from core's `#contracts.providedBy`, the count the single-provider guard reads). With the switch off, the render's verdicts, refusals and rendered output SHALL be exactly those of a render without the switch. With the switch on:

- A skipped trait demand SHALL NOT refuse the render. Its component SHALL render every pair it matched, and nothing SHALL render for the trait.
- A skipped resource demand SHALL NOT refuse the render, and its component SHALL be omitted: no pair of that component SHALL render, and the component SHALL NOT be reported as unmatched. A partly satisfied component never renders.
- Every skipped demand SHALL be reported as a skipped-demand row carrying the component, the contract key, the kind (`resource` or `trait`), the defining catalog's registry key (empty when no enabled catalog lists the key), the same-base alternatives the platform implements, and whether the component was omitted. The omission flag SHALL be set on every skipped row of an omitted component, trait rows included.
- Every other refusal SHALL stand: a catalog-fulfilled unresolved demand, a provider-fulfilled demand for which a provider exists but did not match or was disqualified, an over-subscribed provider-fulfilled contract, and an unmatched component (other than an omitted one) SHALL refuse the render as without the switch. The skipped-demand rows SHALL be readable on the diagnostics beside such a refusal.
- An unhandled trait whose effective `optional` is true SHALL remain an unhandled-trait table entry, never a skipped row.
- The generated render module SHALL carry the switch as a literal, so its own `gate` field agrees with the kernel's verdict under both values.

Source: core `SPEC.md` §2.1 and §3.1 and capability `contract-fulfilment`, as amended by core changes `skip-unprovided-provider-demands` and `count-providers-per-registry-entry`.

#### Scenario: Switch off refuses an unprovided trait as before

- **WHEN** a component attaches a load-bearing trait whose contract declares `fulfilment: "provider"`, no transformer on the platform requires that contract, and the render runs with the switch off
- **THEN** `Render` fails with an unresolved-demands cause whose row for the trait is marked unprovided, and the diagnostics carry no skipped-demand row

#### Scenario: An unprovided trait is skipped and its component renders

- **WHEN** the same render runs with the switch on
- **THEN** `Render` succeeds, every pair the component matched renders, the diagnostics carry one skipped-demand row naming the component, the trait's contract key, kind `trait`, the defining catalog and an unset omission flag, and the unresolved-demand list is empty

#### Scenario: An unprovided resource omits its component

- **WHEN** one component declares a resource whose contract declares `fulfilment: "provider"` and has no provider on the platform, a sibling component is fully satisfied, and the render runs with the switch on
- **THEN** `Render` succeeds, no object renders for the first component although its other resources have matching transformers, the sibling's objects render, the first component is not reported as unmatched, and its skipped-demand row carries kind `resource` and a set omission flag

#### Scenario: A catalog-fulfilled demand still refuses

- **WHEN** a component attaches a load-bearing trait whose contract has default (`catalog`) fulfilment and no transformer handles it, and the render runs with the switch on
- **THEN** `Render` fails with an unresolved-demands cause carrying that row, marked not unprovided

#### Scenario: A provider that exists but does not match still refuses

- **WHEN** a component attaches a load-bearing, provider-fulfilled trait, one enabled catalog carries a transformer requiring it, that transformer is disqualified for the component, and the render runs with the switch on
- **THEN** `Render` fails with an unresolved-demands cause carrying that row, marked not unprovided, and no skipped-demand row is reported for it

#### Scenario: Over-subscription still refuses

- **WHEN** two enabled catalogs each carry a transformer requiring the same provider-fulfilled contract and the render runs with the switch on
- **THEN** `Render` fails with the typed over-subscription cause, exactly as with the switch off

#### Scenario: The module's own gate agrees under the switch

- **WHEN** a render whose only unresolved demand is unprovided runs with the switch on
- **THEN** the built value's `gate` field is `true`, and with the switch off the same render's `gate` field is an error

## ADDED Requirements

### Requirement: A render refuses a platform whose core predates the provider count

`Kernel.Render` SHALL check, before staging, that the platform's `Package` carries `#contracts.providedBy`. When it does not (the platform module pins a core release older than `2.0.0-alpha.12`, or carries no `#contracts` at all), `Render` SHALL return an error wrapping the typed `PlatformCoreTooOldError` that names the platform, the missing field and the first core release carrying it. The refusal SHALL NOT be a `*RenderError`, SHALL leave no staging directory behind and SHALL NOT fall back to a count of the kernel's own: a render never runs on a platform whose inventory would disagree with it. The check is sound because the render build evaluates the platform module's own core pin, the same one its `Package` was built from. The check SHALL only read the platform's `Package` (a path lookup and a presence test, never a unification or fill), so one acquired platform stays shareable, as data, across concurrent `Render` calls on one Kernel.

#### Scenario: An older-core platform is refused before staging

- **WHEN** a platform module identical to a served fixture but pinning core `2.0.0-alpha.10` is acquired and rendered
- **THEN** `Render` returns an error from which `errors.As` extracts a `PlatformCoreTooOldError` naming the platform, the field `providedBy` and the release `2.0.0-alpha.12`, the error is not a `*RenderError`, no result is returned, and no render staging directory remains

#### Scenario: A current-core platform is not affected

- **WHEN** a platform module pinning core `2.0.0-alpha.12` or later is rendered
- **THEN** the core floor raises no error and the render proceeds to staging

#### Scenario: A platform shared by concurrent renders stays race-free

- **WHEN** one acquired current-core platform is shared by several goroutines, each calling `Render` on one Kernel
- **THEN** every render passes the core floor and produces the same objects, with no data race reported under the race detector
