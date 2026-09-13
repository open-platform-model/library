## MODIFIED Requirements

### Requirement: Unresolved demands are diagnosed with alternatives

Every resource a component declares is a required demand. A demanded contract for which the platform holds no candidate, or for which every candidate is disqualified (by unification or by predicate), SHALL be reported as an unresolved-demand row carrying the component, the contract key, the kind, the same-base alternatives the platform does implement (sorted in the build on the apiVersion ladder, so the row is deterministic), the registry key of the enabled catalog that lists the demanded key in its contract maps (empty when no enabled catalog lists it; read inside the build from the platform's derived contract inventory, never parsed off the FQN), and, when candidates existed, each disqualified candidate's transformer with the FQNs it conflicted at. `Render` SHALL fail on any unresolved demand through the typed gate cause while returning the full diagnosis. The row SHALL distinguish three cases: "defined by `<catalog>` and nothing on this platform implements it" (defining catalog named, no alternatives), "implemented at a different apiVersion" (alternatives listed), and "no enabled catalog defines this contract" (no defining catalog, no alternatives). The defining catalog is diagnostic only (enhancement 0015 D18): its presence or absence SHALL NOT change whether the demand refuses.

#### Scenario: Undemandable resource fails the render

- **WHEN** a component demands a resource contract no embedded catalog implements or lists
- **THEN** `Render` fails with an unresolved-demands cause whose row names the component and key with no alternatives and no defining catalog, and the message says no enabled catalog defines the contract

#### Scenario: Different apiVersion named

- **WHEN** the platform implements the same contract base at `v1alpha1`, `v1` and `v1beta1` only, and the component demands `v2`
- **THEN** the unresolved-demand row lists the alternatives as `v1alpha1`, `v1beta1`, `v1`

#### Scenario: A defined but unimplemented contract names its catalog

- **WHEN** an enabled catalog lists a provider-fulfilled trait in its contract maps, no transformer on the platform requires it, and a component attaches it load-bearing
- **THEN** `Render` fails with an unresolved-demands cause whose row carries that catalog's registry key as the defining catalog and no alternatives, and the message names the catalog beside the component and key

#### Scenario: A disabled catalog defines nothing

- **WHEN** the only catalog listing the demanded key is a registry entry with `enable: false`
- **THEN** the unresolved-demand row carries no defining catalog
