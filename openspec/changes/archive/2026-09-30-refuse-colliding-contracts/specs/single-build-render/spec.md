## ADDED Requirements

### Requirement: A render refuses a platform whose enabled entries share contract keys

The generated glue SHALL read the platform's contract collisions from core's derived contract inventory and SHALL NOT compute them: for every key in `#Platform.#contracts.collisions`, in that (ascending) order, it SHALL emit one collision row naming the key and the registry keys `#contracts.collidingEntries` holds for it (path plus major, ascending). The glue SHALL emit the rows only when the platform value carries `collisions`, and SHALL emit none otherwise: every core release carrying `#contracts` without the report fails to evaluate a platform whose enabled entries share a key, so an absent report provably means no collision. The kernel SHALL decode the rows onto the render diagnostics and SHALL refuse the render, through the fail-closed gate, with one typed collision cause carrying every row unchanged. The refusal is platform-wide: it SHALL hold for every instance and whatever the caller's skip switch says. When a render refuses for several reasons, the collision cause SHALL be the first joined cause, ahead of unresolved demands, over-subscribed contracts and unmatched components, because a colliding key is absent from `definedBy`, `requiredBy` and the comparability report and the other rows are read against a distorted inventory. The cause's message SHALL name, per row, the key, the number of enabled registry entries defining it and those entries, and SHALL say that a contract key must have exactly one enabled definer until side-by-side catalog majors are supported, so all but one of the entries must be disabled. Because the render and the platform's `Contracts()` read the same report, the render's collision rows SHALL equal the inventory's `CollidingEntries` on every platform.

Source: enhancement 0026 OQ17 (the interim safety net ahead of 0026 D9), core change `fold-colliding-contract-keys`.

#### Scenario: A colliding bridge platform is refused instead of rendering twice

- **WHEN** the platform enables `maj@v0` 0.1.0 and `maj@v1` 1.4.0, which list the same container, expose and backup keys, and `maj@v1` ships a bridge transformer requiring `maj@v0`'s container, and an instance of a `maj@v0` module is rendered
- **THEN** `Render` fails with a `*RenderError` whose joined causes start with a typed collision cause carrying three rows (the container, backup and expose keys in ascending order, each naming `maj@v0` and `maj@v1`), no compiled object is returned, and the diagnostics carry the same three rows

#### Scenario: A collision refuses under the skip switch

- **WHEN** the same colliding platform is rendered with the skip switch on
- **THEN** `Render` fails with the same typed collision cause

#### Scenario: A collision and an over-subscription are both named, collision first

- **WHEN** the colliding platform also enables `bprov@v0` and `bprov@v1`, whose transformers both require the colliding provider-fulfilled backup trait
- **THEN** `Render` fails with a collision cause and an over-subscription cause for the backup key, reachable through `errors.As`, and the collision cause is the first joined cause

#### Scenario: Collision rows equal the inventory on every served platform

- **WHEN** the parity test renders every served render platform
- **THEN** the render's collision rows, keyed by contract, equal the platform inventory's `CollidingEntries`, and a platform with no collision carries no row

#### Scenario: A platform pinning a core without the report renders unchanged

- **WHEN** the served healthy platform, re-pinned to core `2.0.0-alpha.12`, is rendered
- **THEN** the render succeeds exactly as before this change, with no collision row and a routable diagnostic reading true

### Requirement: A render refuses a not-routable platform no row explains

The kernel SHALL decode core's `#contracts.routable` onto the render diagnostics, and SHALL refuse the render with a typed not-routable cause when it reads false and the diagnostics carry neither a collision row nor an over-subscription row. The cause SHALL be the last joined cause. It exists so that the kernel's decision agrees with the render module's own gate (which reads `routable`) even when a future core adds a term to `routable` that no row reports; on every platform whose `routable` is explained by its rows it SHALL NOT be raised.

#### Scenario: No served platform raises the catch-all

- **WHEN** the parity test renders every served render platform
- **THEN** no refusal carries the not-routable cause, and the decoded routable diagnostic equals the inventory's `Routable`

#### Scenario: An unexplained not-routable verdict refuses

- **WHEN** the gate is given decoded diagnostics whose routable reads false and which carry no collision or over-subscription row
- **THEN** the gate refuses with the typed not-routable cause, joined after every other cause present

## MODIFIED Requirements

### Requirement: The single-provider guard runs inside the build

The generated glue SHALL read the platform's provider count from core's derived contract inventory and SHALL NOT compute a count of its own. The count is `#Platform.#contracts.providedBy`: for every contract key declared `fulfilment: "provider"` on a required demand (`requiredResources` and `requiredTraits`) of any transformer of an enabled `#registry` entry, the sorted registry keys (the catalog module paths with their major, bound by core to each entry's `#catalog.metadata.modulePath`) of the entries whose transformers require it, whether or not any enabled entry defines the contract. Provenance is the registry key, never a value parsed out of an FQN or read off the transformer, so two majors of one catalog are two providers and two transformers of one entry are one. Every key core lists in `#contracts.overSubscribed` SHALL be reported as an over-subscription row naming the key and the registry keys `providedBy` holds for it, and the render SHALL refuse through the fail-closed gate with one typed over-subscription cause carrying every such row. The rows SHALL be built by iterating `providedBy` unconditionally, so a platform value lacking it fails the build instead of reading as zero providers. Keys with default (`catalog`) fulfilment MAY be supplied by any number of transformers from any number of catalogs. Because both the render and the platform's `Contracts()` read this one count, a render SHALL refuse on over-subscription exactly when the platform's inventory reads a non-empty `OverSubscribed`; `Routable` reads false exactly when the render carries an over-subscription row or a collision row.

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

- **WHEN** the parity test acquires every served render platform (each `testdata/render/platform*` directory), reads `Contracts()` and renders that platform's classified instance fixture against it
- **THEN** the sorted `OverSubscribed` list equals the keys of the render's over-subscription rows, every row's registry keys equal `ProvidedBy` for its key, `Routable` is true exactly when the render carries neither an over-subscription row nor a collision row, and the decoded routable diagnostic equals `Routable`, whether or not the render also refuses for an unrelated reason

### Requirement: Unresolved demands are diagnosed with alternatives

Every resource a component declares is a required demand. A demanded contract for which the platform holds no candidate, or for which every candidate is disqualified (by unification or by predicate), SHALL be reported as an unresolved-demand row carrying the component, the contract key, the kind, the same-base alternatives the platform does implement (sorted in the build on the apiVersion ladder, so the row is deterministic), the registry key of the enabled catalog that lists the demanded key in its contract maps (empty when no enabled catalog lists it, or when more than one does; read inside the build from the platform's derived contract inventory, never parsed off the FQN), the registry keys of the enabled entries defining the key when it is a collision (read from `#contracts.collidingEntries` when the platform carries it, empty otherwise), whether the demand is unprovided (its contract declares `fulfilment: "provider"` and no enabled registry entry carries a transformer requiring the key), and, when candidates existed, each disqualified candidate's transformer with the FQNs it conflicted at. The unprovided marker SHALL be computed on every render, whatever the skip switch says. `Render` SHALL fail on any unresolved demand that the caller's switch did not skip, through the typed gate cause, while returning the full diagnosis. The row SHALL distinguish four cases: "defined by `<catalog>` and nothing on this platform implements it" (defining catalog named, no alternatives), "implemented at a different apiVersion" (alternatives listed), "defined by more than one enabled registry entry" (no alternatives, the colliding entries named), and "no enabled catalog defines this contract" (no defining catalog, no colliding entries, no alternatives). An unprovided row's message SHALL additionally say that the contract is provider-fulfilled and nothing on the platform provides it, after the case text and before any disqualified-candidate count. The defining catalog and the colliding entries are diagnostic only (enhancement 0015 D18): their presence or absence SHALL NOT change whether the demand refuses.

#### Scenario: Undemandable resource fails the render

- **WHEN** a component demands a resource contract no embedded catalog implements or lists
- **THEN** `Render` fails with an unresolved-demands cause whose row names the component and key with no alternatives, no defining catalog and no colliding entries, and the message says no enabled catalog defines the contract

#### Scenario: Different apiVersion named

- **WHEN** the platform implements the same contract base at `v1alpha1`, `v1` and `v1beta1` only, and the component demands `v2`
- **THEN** the unresolved-demand row lists the alternatives as `v1alpha1`, `v1beta1`, `v1`

#### Scenario: A defined but unimplemented contract names its catalog

- **WHEN** an enabled catalog lists a trait in its contract maps (whatever its fulfilment), no transformer on the platform requires it, and a component attaches it load-bearing
- **THEN** `Render` fails with an unresolved-demands cause whose row carries that catalog's registry key as the defining catalog and no alternatives, and the message names the catalog beside the component and key

#### Scenario: A disabled catalog defines nothing

- **WHEN** the only catalog listing the demanded key is a registry entry with `enable: false`
- **THEN** the unresolved-demand row carries no defining catalog and no colliding entries

#### Scenario: An unprovided row says so

- **WHEN** a component attaches a load-bearing trait whose contract an enabled catalog lists and declares `fulfilment: "provider"`, no transformer on the platform requires it, and the skip switch is off
- **THEN** the unresolved-demand row is marked unprovided, and its message names the defining catalog and then states that the contract is provider-fulfilled and nothing on the platform provides it

#### Scenario: A catalog-fulfilled row is not marked

- **WHEN** a component attaches a load-bearing trait with default (`catalog`) fulfilment that no transformer handles
- **THEN** the unresolved-demand row is not marked unprovided, and its message carries no provider-fulfilled text

#### Scenario: A colliding contract row names its entries

- **WHEN** a component's demand on a colliding platform goes unresolved on a key two enabled entries define
- **THEN** the row carries no defining catalog, carries the colliding registry keys, and its message says the contract is defined by more than one enabled registry entry and names them, never that no enabled catalog defines it

### Requirement: A render result carries no presentation strings

A render result and a render refusal SHALL expose verdicts as structured rows only. The kernel SHALL NOT attach human-readable warning or advisory messages to a result; every advisory fact (an unhandled optional trait, a module requiring a newer build than the platform carries, a demand skipped under the caller's switch) SHALL be readable from the diagnostics rows, and the diagnostics type SHALL document where each advisory fact lives. Each row-carrying typed gate cause (colliding contracts, unresolved demands, unmatched components, over-subscribed contracts) SHALL carry the same rows the diagnostics carry, in the same order, and SHALL NOT synthesize causes of another kind. The not-routable cause carries no rows: it is raised only from the decoded routable diagnostic.

#### Scenario: Advisory facts are rows

- **WHEN** a render succeeds with one unhandled optional trait and one path in warn-mode skew
- **THEN** the result exposes the trait on the unhandled-trait table and the path on a resolved-versions row marked newer, and no field of the result holds a formatted message

#### Scenario: A gate cause is the diagnostics rows

- **WHEN** `Render` refuses on two unresolved demands, or on a platform with colliding contracts
- **THEN** the unresolved-demands cause, or the collision cause, reachable through `errors.As` carries exactly the rows the refusal's diagnostics carry, and unwrapping it yields no cause of a different kind

#### Scenario: A skipped demand is a row

- **WHEN** a render succeeds under the skip switch with one skipped trait demand
- **THEN** the result exposes the demand as a skipped-demand row, and no field of the result holds a formatted message for it

### Requirement: The render module's own gate agrees with the kernel's refusal

The generated render module SHALL carry a `gate` field that evaluates to an error exactly when the kernel refuses the render on a decoded verdict (an unresolved demand, an unmatched component, an over-subscribed provider-fulfilled contract, a contract collision, or a not-routable platform) and to `true` otherwise, so a staged render module refuses on its own under a plain CUE evaluation. The kernel's decoded refusal SHALL remain the authoritative one, since it carries the typed causes, and SHALL be decided from decoded rows and the decoded routable diagnostic only, never by reading `gate`.

#### Scenario: Refusal is visible in the module

- **WHEN** a render refuses on an unmatched component, or on a contract collision
- **THEN** the built value's `gate` field is an error and the diagnostics beside it decode

#### Scenario: Success is visible in the module

- **WHEN** a render passes the gate
- **THEN** the built value's `gate` field is `true`
