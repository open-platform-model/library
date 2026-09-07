# single-build-render Specification

## Purpose
The render path as one CUE build per render: the kernel stages the instance and platform source trees into a generated render module, evaluates it once in a context that dies with the render, and reads matching verdicts and rendered output off the built value. Covers input requirements, the promoted dependency list and its refusal invariant, version-skew detection and policy, in-build matching, and decode.
## Requirements

### Requirement: Render inputs are source-carrying artifacts

`Kernel.Render` SHALL accept a `*module.Instance` and a `*platform.Platform` that both carry a `Source` (staged tree or on-disk directory), plus a non-empty runtime name and a skew policy. It SHALL refuse an input whose `Source` is absent with an error naming the input; an evaluated `Package` alone is never sufficient, because the render build imports packages.

#### Scenario: Source-less platform refused

- **WHEN** `Render` is invoked with a platform constructed from a bare `cue.Value` (`Source == nil`)
- **THEN** it returns an error naming the platform's missing source, and no build is attempted

#### Scenario: Overlay-mode instance accepted

- **WHEN** `Render` is invoked with a synthesized instance whose `Source` is an overlay tree
- **THEN** the tree is materialized into the render's staging directory and the build proceeds

### Requirement: The render module's dependency list is derived by promotion

The generated render module's dependency list SHALL be derived by promotion: the platform module's tidied dependency list adopted whole, the instance module's list unioned in for paths only the instance carries, and the platform's entry winning every shared path. No tidy-equivalent and no registry consultation SHALL run at render time to compute the list. The two input trees SHALL enter the build through build-local directory replacements; neither input is fetched from a registry. Each input module's own path SHALL be listed in the render module's `cue.mod/module.cue` with a placeholder version of its own major and marked as the default major (unless a promoted entry already marks another major of the same root path default), so an unqualified import of the input's own subpackage from inside it resolves through the main module's default-major table exactly as it does when the input is the main module.

#### Scenario: Platform wins a shared path

- **WHEN** the instance module's `cue.mod` requires catalog build `1.3.0` and the platform module's `cue.mod` carries `1.2.0` for the same path
- **THEN** the render module lists `1.2.0`, and the build evaluates the platform's catalog bytes

#### Scenario: Instance-only paths survive

- **WHEN** the instance module depends on a path the platform module does not carry
- **THEN** the render module lists the instance's entry for that path, and the instance's import resolves

#### Scenario: An input's unqualified self-import resolves

- **WHEN** the instance module's root package imports one of its own subpackages without a major qualifier (an `identity` package, the shape `opm module init` writes) and the instance enters the build through a directory replacement
- **THEN** the render module marks the instance module's major default, and the build resolves the import from the replaced directory

### Requirement: A render refuses when promotion cannot cover an OPM path

After writing the render module's dependency list, the kernel SHALL verify that every OPM-namespace path required by either input resolves from the render module's own list, and SHALL refuse the render otherwise with an error identifying the uncovered path as a kernel defect. This refusal SHALL NOT be configurable by any caller.

#### Scenario: An uncovered path refuses

- **WHEN** the promotion produces a list missing an OPM-namespace path one input requires
- **THEN** `Render` fails before evaluation with an error naming the path, regardless of the configured skew policy

### Requirement: Version skew is detected from the two committed resolutions and the response is caller-configured

For each OPM-namespace path, the kernel SHALL compare the instance module's `cue.mod` requirement against the platform module's tidied entry (never the render module's promoted list). When the instance requires a NEWER build than the platform carries, the configured policy decides: warn-and-render (the default when no policy is supplied) marks that path's resolved-versions row as newer and proceeds; refuse fails the render before evaluation. A module requiring an OLDER build SHALL produce no such mark; the per-path resolved-versions comparison SHALL always be present in the result as plain data with no severity. The kernel SHALL NOT render the skew as a message string; a frontend formats the row.

#### Scenario: Newer module warns and renders by default

- **WHEN** the instance requires catalog `1.3.0`, the platform carries `1.2.0`, and no policy is supplied
- **THEN** the render proceeds against `1.2.0` and the result's resolved-versions row for that path names both versions and is marked newer, and the result carries no message string for it

#### Scenario: Refuse policy stops the render

- **WHEN** the same skew exists and the caller configured the refuse policy
- **THEN** `Render` fails before evaluation with an error naming the path and both versions

#### Scenario: Older module is data, not a warning

- **WHEN** the instance requires `1.1.0` and the platform carries `1.2.0`
- **THEN** the render proceeds, the resolved-versions row for that path is present in the result's diagnostics, and it is not marked newer

### Requirement: Each render is its own build in its own context

`Render` SHALL create a fresh `cue.Context` for the render, evaluate the staged render module exactly once with it, and release it when `Render` returns. No built value SHALL be shared between renders, and the render SHALL NOT use the Kernel's long-lived context. The staging directory SHALL be removed when the render completes.

#### Scenario: Repeated renders share nothing

- **WHEN** `Render` is invoked twice with the same inputs
- **THEN** each invocation stages, builds and decodes independently, and the results are byte-identical

### Requirement: Matching runs inside the build with verdicts as data

The generated glue SHALL express matching over the platform's derived `#composedTransformers` with the verdicts as data fields the kernel decodes into the structured diagnostic types without deriving, joining, grouping or re-sorting them: the matched pair set; every unresolved demand with the same-base alternatives the platform implements (sorted in the build on the `alpha < beta < GA`, then major, then minor apiVersion ladder) and each disqualified candidate with the FQNs it conflicted at; every candidate the always-unify rung refused, with its conflicting FQNs; every unmatched component with every candidate the demand walk reached for it and, for a predicate refusal, the required labels the component lacked or carried with a different value; the unhandled-trait table; the over-subscription rows. Matching semantics are those of the render-parity oracle: the pair set the glue reports SHALL equal the pair set plain predicate matching over the same inputs produces (`render-parity`, "Matched pair sets agree"). The always-unify rung SHALL be plain unification with no provenance exclusion. The fail-closed demand gate SHALL hold: an unresolved demand or unmatched component refuses the render while the diagnostics remain readable and decoded; an effectively-optional unhandled trait is reported on the diagnostics' unhandled-trait table; an unhandled trait with an UNSTATED optional posture refuses as a build error naming the trait's own `optional` field.

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

### Requirement: Rendered output decodes with provenance and per-pair concreteness

On a passing gate, `Render` SHALL decode `rendered` into `[]*kernel.Compiled`, each carrying instance, component and transformer provenance, in the build's deterministic order. `Compiled` SHALL be declared in `opm/kernel`, beside the verb that produces it, with exactly the fields `Value cue.Value`, `Instance`, `Component` and `Transformer`; no other package under `opm/` SHALL declare a compiled-output type. The kernel SHALL validate per-pair output concreteness itself: an incomplete (non-error) pair output SHALL fail the render at a path naming the pair.

#### Scenario: Provenance on every object

- **WHEN** a render of two matched pairs succeeds
- **THEN** every returned `*kernel.Compiled` names its instance, component and transformer FQN

#### Scenario: Incomplete pair output refuses

- **WHEN** a transformer's output evaluates non-concrete without erroring
- **THEN** `Render` fails with an error whose path names the (component, transformer) pair

#### Scenario: One compiled-output type

- **WHEN** a consumer inspects the packages under `opm/`
- **THEN** `Compiled` is declared in `opm/kernel` only and no `opm/core` package exists

### Requirement: Render is the kernel's sole render path

`Kernel.Render` SHALL be the only way the kernel renders an instance against a platform. The kernel SHALL expose no `Compile`, `Match` or `Materialize` method and no materialized-platform type; a dry run is `Render` with the rendered output discarded, since the build evaluates every matched pair regardless. The kernel's default core schema pin SHALL be a release carrying the D5 registry shape (`schema-dispatch`, "DefaultSchemaModule constant"), so every artifact the kernel synthesizes or validates is judged against that shape. A platform module importing its catalogs is the only platform shape the kernel accepts: the platform shape gate SHALL validate every `#registry` entry for completeness (`helper-packages`, "Loader shape gate validates identity and registry completeness"), so an entry that names no embedded catalog is refused at acquisition. Core derives the entry's `version` from the embedded catalog's stamped identity, and with no catalog that readout is a missing required field.

#### Scenario: Old entry points are gone

- **WHEN** a consumer inspects the exported identifiers of `opm/kernel`
- **THEN** none of `Compile`, `Match`, `Materialize`, `SynthesizePlatform`, `CompileInput`, `MatchInput`, `CompileResult`, `MatchPlan` exists, and no `opm/materialize` or `opm/compile` package exists

#### Scenario: A subscription-shaped platform is refused

- **WHEN** a platform package declares a registry entry with a `version` scalar and no embedded catalog
- **THEN** `AcquirePlatformFromDir` (and `LoadPlatformPackage`) fails with an error wrapping `ErrMissingRequiredField` that names the entry's `version` as a required field the embedded catalog would have supplied, and no render is attempted

### Requirement: The single-provider guard runs inside the build

The generated glue SHALL compute, over the platform's enabled `#registry` entries, every contract key declared `fulfilment: "provider"` on a required demand (`requiredResources` and `requiredTraits`) of any transformer in an entry's `#transformers`, and the set of registry keys (the catalog module paths, bound by core to each entry's `#catalog.metadata.modulePath`) whose transformers require it. Provenance is the registry key, never a value parsed out of an FQN or read off the transformer. A key supplied by more than one registry entry SHALL be reported as an over-subscription row naming the key and every registry key, and SHALL refuse the render through the fail-closed gate with one typed over-subscription cause carrying every such row. Keys with default (`catalog`) fulfilment MAY be supplied by any number of transformers from any number of catalogs.

#### Scenario: Second provider refused in-build

- **WHEN** two catalogs embedded in the platform each supply a transformer requiring a contract declared `fulfilment: "provider"`
- **THEN** `Render` fails with a typed over-subscription cause whose row names the key and both registry keys, and the diagnostics remain readable beside the refusal

#### Scenario: Catalog-fulfilled plurality allowed

- **WHEN** many transformers across catalogs require a contract with default fulfilment
- **THEN** the render proceeds and every candidate participates in matching

### Requirement: Unresolved demands are diagnosed with alternatives

Every resource a component declares is a required demand. A demanded contract for which the platform holds no candidate, or for which every candidate is disqualified (by unification or by predicate), SHALL be reported as an unresolved-demand row carrying the component, the contract key, the kind, the same-base alternatives the platform does implement (sorted in the build on the apiVersion ladder, so the row is deterministic), and, when candidates existed, each disqualified candidate's transformer with the FQNs it conflicted at. `Render` SHALL fail on any unresolved demand through the typed gate cause while returning the full diagnosis. The row SHALL distinguish "nothing on this platform implements this contract" (no alternatives) from "implemented at a different apiVersion" (alternatives listed).

#### Scenario: Undemandable resource fails the render

- **WHEN** a component demands a resource contract no embedded catalog implements
- **THEN** `Render` fails with an unresolved-demands cause whose row names the component and key with no alternatives

#### Scenario: Different apiVersion named

- **WHEN** the platform implements the same contract base at `v1alpha1`, `v1` and `v1beta1` only, and the component demands `v2`
- **THEN** the unresolved-demand row lists the alternatives as `v1alpha1`, `v1beta1`, `v1`

### Requirement: Trait posture governs unhandled traits

An unhandled trait's effect SHALL be governed by its effective `optional` value read from the component's trait attachment: effectively optional is reported on the diagnostics' unhandled-trait table and the render proceeds; effectively load-bearing fails exactly as an unresolved resource. An unstated posture (non-concrete `optional`) SHALL fail closed as a build error naming the trait's own `optional` field. The kernel SHALL NOT render an unhandled trait as a message string; a frontend formats the table.

#### Scenario: Optional trait warns

- **WHEN** an unhandled trait's effective `optional` is true
- **THEN** the render proceeds, the diagnostics' unhandled-trait table maps the component to that trait, and the result carries no message string for it

#### Scenario: Load-bearing trait fails

- **WHEN** an unhandled trait's effective `optional` is false
- **THEN** `Render` fails with an unresolved-demands cause carrying a row for the trait

### Requirement: The label predicate covers every admitted label type

The predicate rung SHALL evaluate a transformer's `requiredLabels` against the component's `matchLabels` by unification, so every value type `#LabelsAnnotationsType` admits participates, and a mismatch on any admitted type disqualifies the candidate.

#### Scenario: Non-string label value compared

- **WHEN** a transformer requires a label whose value is a non-string type the schema admits and the component carries a different value
- **THEN** the candidate is disqualified rather than silently skipped
### Requirement: A render result carries no presentation strings

A render result and a render refusal SHALL expose verdicts as structured rows only. The kernel SHALL NOT attach human-readable warning or advisory messages to a result; every advisory fact (an unhandled optional trait, a module requiring a newer build than the platform carries) SHALL be readable from the diagnostics rows, and the diagnostics type SHALL document where each advisory fact lives. Each typed gate cause (unresolved demands, unmatched components, over-subscribed contracts) SHALL carry the same rows the diagnostics carry, in the same order, and SHALL NOT synthesize causes of another kind.

#### Scenario: Advisory facts are rows

- **WHEN** a render succeeds with one unhandled optional trait and one path in warn-mode skew
- **THEN** the result exposes the trait on the unhandled-trait table and the path on a resolved-versions row marked newer, and no field of the result holds a formatted message

#### Scenario: A gate cause is the diagnostics rows

- **WHEN** `Render` refuses on two unresolved demands
- **THEN** the unresolved-demands cause reachable through `errors.As` carries exactly the two rows the refusal's diagnostics carry, and unwrapping it yields no cause of a different kind

### Requirement: The render module's own gate agrees with the kernel's refusal

The generated render module SHALL carry a `gate` field that evaluates to an error exactly when the kernel refuses the render on a decoded verdict (an unresolved demand, an unmatched component, an over-subscribed provider-fulfilled contract) and to `true` otherwise, so a staged render module refuses on its own under a plain CUE evaluation. The kernel's decoded refusal SHALL remain the authoritative one, since it carries the typed causes.

#### Scenario: Refusal is visible in the module

- **WHEN** a render refuses on an unmatched component
- **THEN** the built value's `gate` field is an error and the diagnostics beside it decode

#### Scenario: Success is visible in the module

- **WHEN** a render passes the gate
- **THEN** the built value's `gate` field is `true`
