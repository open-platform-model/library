## ADDED Requirements

### Requirement: Render is the kernel's sole render path

`Kernel.Render` SHALL be the only way the kernel renders an instance against a platform. The kernel SHALL expose no `Compile`, `Match` or `Materialize` method and no materialized-platform type; a dry run is `Render` with the rendered output discarded, since the build evaluates every matched pair regardless. The kernel's default core schema pin SHALL be a release carrying the D5 registry shape (`schema-dispatch`, "DefaultSchemaModule constant"), so every artifact the kernel synthesizes or validates is judged against that shape. A platform module importing its catalogs is the only platform shape the kernel accepts: the platform shape gate SHALL validate every `#registry` entry for completeness (`helper-packages`, "Loader shape gate validates identity and registry completeness"), so an entry that names no embedded catalog is refused at acquisition. Core derives the entry's `version` from the embedded catalog's stamped identity, and with no catalog that readout is a missing required field.

#### Scenario: Old entry points are gone

- **WHEN** a consumer inspects the exported identifiers of `opm/kernel`
- **THEN** none of `Compile`, `Match`, `Materialize`, `SynthesizePlatform`, `CompileInput`, `MatchInput`, `CompileResult`, `MatchPlan` exists, and no `opm/materialize` or `opm/compile` package exists

#### Scenario: A subscription-shaped platform is refused

- **WHEN** a platform package declares a registry entry with a `version` scalar and no embedded catalog
- **THEN** `AcquirePlatformFromDir` (and `LoadPlatformPackage`) fails with an error wrapping `ErrMissingRequiredField` that names the entry's `version` as a required field the embedded catalog would have supplied, and no render is attempted

### Requirement: The single-provider guard runs inside the build

The generated glue SHALL compute, over the platform's enabled `#registry` entries, every contract key declared `fulfilment: "provider"` on a required demand (`requiredResources` and `requiredTraits`) of any transformer in an entry's `#transformers`, and the set of registry keys (the catalog module paths, bound by core to each entry's `#catalog.metadata.modulePath`) whose transformers require it. Provenance is the registry key, never a value parsed out of an FQN or read off the transformer. A key supplied by more than one registry entry SHALL be reported as an over-subscription diagnostic row naming the key and every registry key, and SHALL refuse the render through the fail-closed gate with a decoded typed error. Keys with default (`catalog`) fulfilment MAY be supplied by any number of transformers from any number of catalogs.

#### Scenario: Second provider refused in-build

- **WHEN** two catalogs embedded in the platform each supply a transformer requiring a contract declared `fulfilment: "provider"`
- **THEN** `Render` fails with a typed over-subscription error naming the key and both registry keys, and the diagnostics remain readable beside the refusal

#### Scenario: Catalog-fulfilled plurality allowed

- **WHEN** many transformers across catalogs require a contract with default fulfilment
- **THEN** the render proceeds and every candidate participates in matching

### Requirement: Unresolved demands are diagnosed with alternatives

Every resource a component declares is a required demand. A demanded contract for which the platform holds no candidate, or for which every candidate is disqualified (by unification or by predicate), SHALL be reported as a typed unresolved-demand diagnostic carrying the component, the contract key, the same-base alternatives the platform does implement (sorted, so the diagnostic is deterministic), and, when candidates existed, the per-candidate disqualification naming the conflicting FQN. `Render` SHALL fail on any unresolved demand through the typed gate while returning the full diagnosis. The diagnostic SHALL distinguish "nothing on this platform implements this contract" (no alternatives) from "implemented at a different apiVersion" (alternatives listed).

#### Scenario: Undemandable resource fails the render

- **WHEN** a component demands a resource contract no embedded catalog implements
- **THEN** `Render` fails with an unresolved-demand error naming the component and key with no alternatives

#### Scenario: Different apiVersion named

- **WHEN** the platform implements the same contract base at a different apiVersion only
- **THEN** the unresolved-demand error lists those keys as alternatives

### Requirement: Trait posture governs unhandled traits

An unhandled trait's effect SHALL be governed by its effective `optional` value read from the component's trait attachment: effectively optional degrades to a warning on the result; effectively load-bearing fails exactly as an unresolved resource. An unstated posture (non-concrete `optional`) SHALL fail closed as a build error naming the trait's own `optional` field.

#### Scenario: Optional trait warns

- **WHEN** an unhandled trait's effective `optional` is true
- **THEN** the render proceeds and the trait is reported on the result's warnings

#### Scenario: Load-bearing trait fails

- **WHEN** an unhandled trait's effective `optional` is false
- **THEN** `Render` fails with an unresolved-demand error for the trait

### Requirement: The label predicate covers every admitted label type

The predicate rung SHALL evaluate a transformer's `requiredLabels` against the component's `matchLabels` by unification, so every value type `#LabelsAnnotationsType` admits participates, and a mismatch on any admitted type disqualifies the candidate.

#### Scenario: Non-string label value compared

- **WHEN** a transformer requires a label whose value is a non-string type the schema admits and the component carries a different value
- **THEN** the candidate is disqualified rather than silently skipped

## MODIFIED Requirements

### Requirement: Matching runs inside the build with verdicts as data

The generated glue SHALL express matching over the platform's derived `#composedTransformers` with the verdicts (matched pairs, missing FQNs, unify disqualifications, unresolved demands, unhandled-trait warnings) as data fields the kernel decodes into the existing structured diagnostic types. Matching semantics are those of the render-parity oracle: the pair set the glue reports SHALL equal the pair set plain predicate matching over the same inputs produces (`render-parity`, "Matched pair sets agree"). The always-unify rung SHALL be plain unification with no provenance exclusion. The fail-closed demand gate SHALL hold: an unresolved demand or unmatched component refuses the render while the diagnostics remain readable and decoded; an effectively-optional unhandled trait degrades to a warning; an unhandled trait with an UNSTATED optional posture refuses as a build error naming the trait's own `optional` field.

#### Scenario: Verdicts decode beside a failing gate

- **WHEN** a component demands a resource FQN no transformer on the platform requires
- **THEN** `Render` fails, and the returned error carries a decoded missing-FQN diagnostic for that (instance, component, FQN) with the platform's same-base alternatives

#### Scenario: Healthy pairs render beside a failing pair

- **WHEN** one matched pair's transformer errors while sibling pairs are healthy
- **THEN** the failing pair is reported as data naming the pair, and the sibling verdicts remain readable

## REMOVED Requirements

### Requirement: The existing pipeline is unaffected

**Reason**: the cutover deletes the existing pipeline; `Render` is the sole path (see the added requirement).

**Migration**: consumers of `Materialize`/`Match`/`Compile` call `AcquirePlatformFromDir` (or the generated platform module's acquire) plus `Render`; see `cli-render-switch` and `operator-render-switch`.
