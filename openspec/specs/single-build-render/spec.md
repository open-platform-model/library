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
- **THEN** the tree is served to the build from memory through the load configuration's overlay, no file of it is written, and the build proceeds

### Requirement: The render module's dependency list is derived by promotion

The generated render module's dependency list SHALL be derived by promotion: the platform module's tidied dependency list adopted whole, the instance module's list unioned in for paths only the instance carries, and the platform's entry winning every shared path. No tidy-equivalent and no registry consultation SHALL run at render time to compute the list. The two input trees SHALL enter the build through build-local directory replacements; neither input is fetched from a registry. Each input module's own path SHALL be listed in the render module's `cue.mod/module.cue` with a placeholder version of its own major and marked as the default major (unless a promoted entry already marks another major of the same root path default), so an unqualified import of the input's own subpackage from inside it resolves through the main module's default-major table exactly as it does when the input is the main module.

When local replacements are enabled for the render, each input's own `cue.mod/local-module.cue` replacements SHALL be promoted under the same precedence as its dependencies: the platform's replacements whole, the instance's only for paths the platform's dependency list does not name. A promoted replacement SHALL be written into the render module's main-module view exactly as the input wrote it, except that a relative directory target SHALL be resolved against that input's own module root, since the render module's root is elsewhere. A replaced path the promoted dependency list does not carry SHALL be listed with a placeholder version of its major, so the coverage invariant holds for a replaced OPM-namespace path. An input dependency that carries no version and is covered by no promoted replacement SHALL be refused before staging with an error naming the path and the input.

#### Scenario: Platform wins a shared path

- **WHEN** the instance module's `cue.mod` requires catalog build `1.3.0` and the platform module's `cue.mod` carries `1.2.0` for the same path
- **THEN** the render module lists `1.2.0`, and the build evaluates the platform's catalog bytes

#### Scenario: Instance-only paths survive

- **WHEN** the instance module depends on a path the platform module does not carry
- **THEN** the render module lists the instance's entry for that path, and the instance's import resolves

#### Scenario: An input's unqualified self-import resolves

- **WHEN** the instance module's root package imports one of its own subpackages without a major qualifier (an `identity` package, the shape `opm module init` writes) and the instance enters the build through a directory replacement
- **THEN** the render module marks the instance module's major default, and the build resolves the import from the replaced directory

#### Scenario: A platform replacement of its catalog is honoured

- **WHEN** local replacements are enabled, the platform module's `cue.mod/local-module.cue` replaces its catalog path with a directory, and that directory holds a catalog whose transformer output differs from the published build
- **THEN** the render module's main-module view replaces the catalog path with that directory, the build evaluates the directory's transformer bytes, and the render's replacement rows name the path, the directory and the platform as its source

#### Scenario: An instance replacement on a platform-named path is inert

- **WHEN** local replacements are enabled and the instance module's `cue.mod/local-module.cue` replaces a path the platform module's dependency list carries
- **THEN** the render module carries no replacement for that path, the build evaluates the platform's pinned bytes, and no replacement row names the path

#### Scenario: An instance replacement on an instance-only path is honoured

- **WHEN** local replacements are enabled and the instance module's `cue.mod/local-module.cue` replaces a path the platform module does not carry with a relative directory
- **THEN** the render module's main-module view replaces that path with the directory resolved against the instance module's root, the build resolves the instance's import from it, and a replacement row names the path, the absolute directory and the instance as its source

#### Scenario: A replace-only dependency is listed with a placeholder

- **WHEN** local replacements are enabled and the instance module's `cue.mod/module.cue` lists a dependency with no version that its `cue.mod/local-module.cue` replaces with a directory
- **THEN** the render module's `cue.mod/module.cue` lists the path with a placeholder version of its major, the main-module view replaces it, and the render proceeds

#### Scenario: A version-less dependency without a replacement is refused

- **WHEN** an input's `cue.mod/module.cue` lists a dependency with no version and no promoted replacement covers the path
- **THEN** `Render` fails before staging with an error naming the path and the input, and no modfile formatting error is surfaced

### Requirement: Local replacements are honoured only when the caller opts in

`Kernel.Render` SHALL accept a per-render opt-in for local replacements, off by default. When it is off, an input whose `cue.mod/local-module.cue` carries at least one replacement SHALL be refused before staging with an error naming the input and the file; the file SHALL NOT be silently ignored. When it is on, promoted replacements SHALL be reported on the render diagnostics as rows carrying the replaced path, the absolute target (a directory, or a module path for a module replacement) and the input that supplied it, in path order; the kernel SHALL NOT render them as message strings. An input without the file SHALL behave identically under either setting.

#### Scenario: Replacement without opt-in refused

- **WHEN** `Render` is invoked with the opt-in off and the platform module's `cue.mod/local-module.cue` replaces its catalog path
- **THEN** it returns an error naming the platform and `cue.mod/local-module.cue`, and no staging directory is written

#### Scenario: Inputs without the file are unaffected

- **WHEN** `Render` is invoked with the opt-in on and neither input carries `cue.mod/local-module.cue`
- **THEN** the render's output and diagnostics are byte-identical to the same render with the opt-in off, and the replacement rows are empty

#### Scenario: Replacement rows are data

- **WHEN** a render with the opt-in on honours one platform replacement and one instance replacement
- **THEN** the result's diagnostics carry exactly two replacement rows, sorted by path, each naming its source input, and no field of the result holds a formatted message about them

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

`Render` SHALL create a fresh `cue.Context` for the render, evaluate the staged render module exactly once with it, and release it when `Render` returns. No built value SHALL be shared between renders, and the render SHALL NOT use any long-lived context. The staging directory SHALL hold only the generated render module (its `cue.mod` and glue); an overlay-mode input SHALL be served from memory and an on-disk input from its own directory, so nothing of either input is copied. The staging directory SHALL be removed when the render completes.

#### Scenario: Repeated renders share nothing

- **WHEN** `Render` is invoked twice with the same inputs
- **THEN** each invocation stages, builds and decodes independently, and the results are byte-identical

#### Scenario: The staging directory holds no input files

- **WHEN** a render of an overlay-mode instance against an overlay-mode platform is staged
- **THEN** the staging directory contains `cue.mod/module.cue`, `cue.mod/local-module.cue` and the glue file, and no `instance/` or `platform/` directory

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

`Kernel.Render` SHALL be the only way the kernel renders an instance against a platform. The kernel SHALL expose no `Compile`, `Match` or `Materialize` method and no materialized-platform type; a dry run is `Render` with the rendered output discarded, since the build evaluates every matched pair regardless. The kernel's default core schema pin SHALL be a release carrying the 0019:D5 registry shape (`schema-dispatch`, "DefaultSchemaModule constant"), so every artifact the kernel synthesizes or validates is judged against that shape. A platform module importing its catalogs is the only platform shape the kernel accepts: the platform shape gate SHALL validate every `#registry` entry for completeness (`helper-packages`, "Loader shape gate validates identity and registry completeness"), so an entry that names no embedded catalog is refused at acquisition. Core derives the entry's `version` from the embedded catalog's stamped identity, and with no catalog that readout is a missing required field.

#### Scenario: Old entry points are gone

- **WHEN** a consumer inspects the exported identifiers of `opm/kernel`
- **THEN** none of `Compile`, `Match`, `Materialize`, `SynthesizePlatform`, `CompileInput`, `MatchInput`, `CompileResult`, `MatchPlan` exists, and no `opm/materialize` or `opm/compile` package exists

#### Scenario: A subscription-shaped platform is refused

- **WHEN** a platform package declares a registry entry with a `version` scalar and no embedded catalog
- **THEN** `AcquirePlatformFromDir` fails with an error wrapping `ErrMissingRequiredField` that names the entry's `version` as a required field the embedded catalog would have supplied, and no render is attempted

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
