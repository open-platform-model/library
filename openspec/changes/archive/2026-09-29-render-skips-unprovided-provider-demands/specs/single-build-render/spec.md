## ADDED Requirements

### Requirement: A caller may skip unprovided provider-fulfilled demands

`Kernel.Render` SHALL accept a per-render switch, off by default, that asks the render to skip every **unprovided** demand: a resource or trait demand whose contract declares `fulfilment: "provider"` and for which no enabled registry entry carries a transformer requiring that contract key (the same per-key count the single-provider guard computes). With the switch off, the render's verdicts, refusals and rendered output SHALL be exactly those of a render without the switch. With the switch on:

- A skipped trait demand SHALL NOT refuse the render. Its component SHALL render every pair it matched, and nothing SHALL render for the trait.
- A skipped resource demand SHALL NOT refuse the render, and its component SHALL be omitted: no pair of that component SHALL render, and the component SHALL NOT be reported as unmatched. A partly satisfied component never renders.
- Every skipped demand SHALL be reported as a skipped-demand row carrying the component, the contract key, the kind (`resource` or `trait`), the defining catalog's registry key (empty when no enabled catalog lists the key), the same-base alternatives the platform implements, and whether the component was omitted. The omission flag SHALL be set on every skipped row of an omitted component, trait rows included.
- Every other refusal SHALL stand: a catalog-fulfilled unresolved demand, a provider-fulfilled demand for which a provider exists but did not match or was disqualified, an over-subscribed provider-fulfilled contract, and an unmatched component (other than an omitted one) SHALL refuse the render as without the switch. The skipped-demand rows SHALL be readable on the diagnostics beside such a refusal.
- An unhandled trait whose effective `optional` is true SHALL remain an unhandled-trait table entry, never a skipped row.
- The generated render module SHALL carry the switch as a literal, so its own `gate` field agrees with the kernel's verdict under both values.

Source: core `SPEC.md` §2.1 and §3.1 and capability `contract-fulfilment`, as amended by core change `skip-unprovided-provider-demands`.

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

## MODIFIED Requirements

### Requirement: Matching runs inside the build with verdicts as data

The generated glue SHALL express matching over the platform's derived `#composedTransformers` with the verdicts as data fields the kernel decodes into the structured diagnostic types without deriving, joining, grouping or re-sorting them: the matched pair set; every unresolved demand with the same-base alternatives the platform implements (sorted in the build on the `alpha < beta < GA`, then major, then minor apiVersion ladder), its unprovided marker, and each disqualified candidate with the FQNs it conflicted at; every demand skipped under the caller's switch (see "A caller may skip unprovided provider-fulfilled demands"); every candidate the always-unify rung refused, with its conflicting FQNs; every unmatched component with every candidate the demand walk reached for it and, for a predicate refusal, the required labels the component lacked or carried with a different value; the unhandled-trait table; the over-subscription rows. Matching semantics are those of the render-parity oracle: with the skip switch off, the pair set the glue reports SHALL equal the pair set plain predicate matching over the same inputs produces (`render-parity`, "Matched pair sets agree"); with it on, the reported pair set SHALL be that set minus every pair of an omitted component. The always-unify rung SHALL be plain unification with no provenance exclusion. The fail-closed demand gate SHALL hold: an unresolved demand that was not skipped, or an unmatched component that was not omitted, refuses the render while the diagnostics remain readable and decoded; an effectively-optional unhandled trait is reported on the diagnostics' unhandled-trait table; an unhandled trait with an UNSTATED optional posture refuses as a build error naming the trait's own `optional` field.

#### Scenario: Verdicts decode beside a failing gate

- **WHEN** a component demands a resource FQN no transformer on the platform requires
- **THEN** `Render` fails, and the returned error carries a decoded unresolved-demand row for that (instance, component, FQN) with the platform's same-base alternatives in ladder order

#### Scenario: Healthy pairs render beside a failing pair

- **WHEN** one matched pair's transformer errors while sibling pairs are healthy
- **THEN** the failing pair is reported as data naming the pair, and the sibling verdicts remain readable

#### Scenario: An unmatched component carries its candidates on the diagnostics

- **WHEN** a component's only candidate is refused by the label predicate on one required label
- **THEN** the diagnostics list that component as unmatched with the candidate's transformer, an unmatched verdict and the missing label, and the typed unmatched-components cause carries the same row

#### Scenario: A unify refusal names its conflicts once

- **WHEN** a candidate's required primitive body conflicts with the component's body at two FQNs
- **THEN** the diagnostics carry one unify-refusal row for that (component, transformer) listing both FQNs, and the demand's disqualified list names the same transformer with the same FQNs

#### Scenario: An omitted component leaves the pair set

- **WHEN** the skip switch is on and a component is omitted for an unprovided resource demand while its other resources match transformers
- **THEN** the reported pair set carries no pair of that component, and no rendered output is decoded for it

### Requirement: Unresolved demands are diagnosed with alternatives

Every resource a component declares is a required demand. A demanded contract for which the platform holds no candidate, or for which every candidate is disqualified (by unification or by predicate), SHALL be reported as an unresolved-demand row carrying the component, the contract key, the kind, the same-base alternatives the platform does implement (sorted in the build on the apiVersion ladder, so the row is deterministic), the registry key of the enabled catalog that lists the demanded key in its contract maps (empty when no enabled catalog lists it; read inside the build from the platform's derived contract inventory, never parsed off the FQN), whether the demand is unprovided (its contract declares `fulfilment: "provider"` and no enabled registry entry carries a transformer requiring the key), and, when candidates existed, each disqualified candidate's transformer with the FQNs it conflicted at. The unprovided marker SHALL be computed on every render, whatever the skip switch says. `Render` SHALL fail on any unresolved demand that the caller's switch did not skip, through the typed gate cause, while returning the full diagnosis. The row SHALL distinguish three cases: "defined by `<catalog>` and nothing on this platform implements it" (defining catalog named, no alternatives), "implemented at a different apiVersion" (alternatives listed), and "no enabled catalog defines this contract" (no defining catalog, no alternatives). An unprovided row's message SHALL additionally say that the contract is provider-fulfilled and nothing on the platform provides it, after the case text and before any disqualified-candidate count. The defining catalog is diagnostic only (enhancement 0015 D18): its presence or absence SHALL NOT change whether the demand refuses.

#### Scenario: Undemandable resource fails the render

- **WHEN** a component demands a resource contract no embedded catalog implements or lists
- **THEN** `Render` fails with an unresolved-demands cause whose row names the component and key with no alternatives and no defining catalog, and the message says no enabled catalog defines the contract

#### Scenario: Different apiVersion named

- **WHEN** the platform implements the same contract base at `v1alpha1`, `v1` and `v1beta1` only, and the component demands `v2`
- **THEN** the unresolved-demand row lists the alternatives as `v1alpha1`, `v1beta1`, `v1`

#### Scenario: A defined but unimplemented contract names its catalog

- **WHEN** an enabled catalog lists a trait in its contract maps (whatever its fulfilment), no transformer on the platform requires it, and a component attaches it load-bearing
- **THEN** `Render` fails with an unresolved-demands cause whose row carries that catalog's registry key as the defining catalog and no alternatives, and the message names the catalog beside the component and key

#### Scenario: A disabled catalog defines nothing

- **WHEN** the only catalog listing the demanded key is a registry entry with `enable: false`
- **THEN** the unresolved-demand row carries no defining catalog

#### Scenario: An unprovided row says so

- **WHEN** a component attaches a load-bearing trait whose contract an enabled catalog lists and declares `fulfilment: "provider"`, no transformer on the platform requires it, and the skip switch is off
- **THEN** the unresolved-demand row is marked unprovided, and its message names the defining catalog and then states that the contract is provider-fulfilled and nothing on the platform provides it

#### Scenario: A catalog-fulfilled row is not marked

- **WHEN** a component attaches a load-bearing trait with default (`catalog`) fulfilment that no transformer handles
- **THEN** the unresolved-demand row is not marked unprovided, and its message carries no provider-fulfilled text

### Requirement: A render result carries no presentation strings

A render result and a render refusal SHALL expose verdicts as structured rows only. The kernel SHALL NOT attach human-readable warning or advisory messages to a result; every advisory fact (an unhandled optional trait, a module requiring a newer build than the platform carries, a demand skipped under the caller's switch) SHALL be readable from the diagnostics rows, and the diagnostics type SHALL document where each advisory fact lives. Each typed gate cause (unresolved demands, unmatched components, over-subscribed contracts) SHALL carry the same rows the diagnostics carry, in the same order, and SHALL NOT synthesize causes of another kind.

#### Scenario: Advisory facts are rows

- **WHEN** a render succeeds with one unhandled optional trait and one path in warn-mode skew
- **THEN** the result exposes the trait on the unhandled-trait table and the path on a resolved-versions row marked newer, and no field of the result holds a formatted message

#### Scenario: A gate cause is the diagnostics rows

- **WHEN** `Render` refuses on two unresolved demands
- **THEN** the unresolved-demands cause reachable through `errors.As` carries exactly the two rows the refusal's diagnostics carry, and unwrapping it yields no cause of a different kind

#### Scenario: A skipped demand is a row

- **WHEN** a render succeeds under the skip switch with one skipped trait demand
- **THEN** the result exposes the demand as a skipped-demand row, and no field of the result holds a formatted message for it
