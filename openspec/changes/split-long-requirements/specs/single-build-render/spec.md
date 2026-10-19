## MODIFIED Requirements

### Requirement: Local replacements are honoured only when the caller opts in
`Kernel.Render` SHALL accept a per-render opt-in for local replacements, off by default. When it is off, an input whose `cue.mod/local-module.cue` carries at least one replacement SHALL be refused before staging with an error naming the input and the file; the file SHALL NOT be silently ignored. An input without the file SHALL behave identically under either setting.

#### Scenario: Replacement without opt-in refused

- **WHEN** `Render` is invoked with the opt-in off and the platform module's `cue.mod/local-module.cue` replaces its catalog path
- **THEN** it returns an error naming the platform and `cue.mod/local-module.cue`, no render module is staged, and no staging file is written

#### Scenario: Inputs without the file are unaffected

- **WHEN** `Render` is invoked with the opt-in on and neither input carries `cue.mod/local-module.cue`
- **THEN** the render's output and diagnostics are byte-identical to the same render with the opt-in off, and the replacement rows are empty

#### Scenario: Replacement rows are data

- **WHEN** a render with the opt-in on honours one platform replacement and one instance replacement
- **THEN** the result's diagnostics carry exactly two replacement rows, sorted by path, each naming its source input, and no field of the result holds a formatted message about them

### Requirement: Trait posture governs unhandled traits
An unhandled trait's effect SHALL be governed by its effective `optional` value read from the component's trait attachment: effectively optional is reported on the diagnostics' unhandled-trait table and the render proceeds; effectively load-bearing fails exactly as an unresolved resource. The kernel SHALL NOT render an unhandled trait as a message string; a frontend formats the table.

#### Scenario: Optional trait warns

- **WHEN** an unhandled trait's effective `optional` is true
- **THEN** the render proceeds, the diagnostics' unhandled-trait table maps the component to that trait, and the result carries no message string for it

#### Scenario: Load-bearing trait fails

- **WHEN** an unhandled trait's effective `optional` is false
- **THEN** `Render` fails with an unresolved-demands cause carrying a row for the trait

### Requirement: The render module's own gate agrees with the kernel's refusal
The generated render module SHALL carry a `gate` field that evaluates to an error exactly when the kernel refuses the render on a decoded verdict (an unresolved demand, an unmatched component, an over-subscribed provider-fulfilled contract, a contract collision, or a not-routable platform) and to `true` otherwise, so a staged render module refuses on its own under a plain CUE evaluation.

#### Scenario: Refusal is visible in the module

- **WHEN** a render refuses on an unmatched component, or on a contract collision
- **THEN** the built value's `gate` field is an error and the diagnostics beside it decode

#### Scenario: Success is visible in the module

- **WHEN** a render passes the gate
- **THEN** the built value's `gate` field is `true`

### Requirement: A caller may skip unprovided provider-fulfilled demands
`Kernel.Render` SHALL accept a per-render switch, off by default, that asks the render to skip every **unprovided** demand: a resource or trait demand whose contract declares `fulfilment: "provider"` and for which no enabled registry entry carries a transformer requiring that contract key (the key is absent from core's `#contracts.providedBy`, the count the single-provider guard reads). Source: core `SPEC.md` §2.1 and §3.1 and capability `contract-fulfilment`.

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

## REMOVED Requirements

### Requirement: The render module's dependency list is derived by promotion
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The platform's dependency list wins in the render module", "The two input trees enter the build through directory replacements", "Each input module's own path is listed as a default major", "Promoted local replacements follow the precedence of the dependencies", "A promoted replacement is written as the input wrote it with a relative target resolved against the input", "A replaced path that the promoted list does not carry is listed with a placeholder version" and "A version-less dependency without a promoted replacement is refused before staging".

### Requirement: Version skew is detected from the two committed resolutions and the response is caller-configured
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The skew check compares the instance's requirement with the platform's tidied entry by SemVer 2 precedence", "A newer instance requirement is warned or refused by the caller's policy" and "An older instance requirement is plain data".

### Requirement: Matching runs inside the build with verdicts as data
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The glue reports the matched pairs, the unify refusals, the unhandled-trait table and the over-subscription rows as data", "The glue reports each unresolved demand with its same-base alternatives as data", "The glue reports each unmatched component with its candidates as data", "The glue reports each demand skipped under the caller's switch as data", "The rungs are evaluated only for a component's candidates", "The reported pair set agrees with plain predicate matching", "The fail-closed demand gate holds while the diagnostics stay readable", "The glue unifies each matched pair's transform once", "The kernel names the failed pairs from each matched pair's rendered output" and "FailedPairs is filled on every RenderError raised after the build".

### Requirement: Rendered output decodes with provenance and per-pair concreteness
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Rendered output decodes with provenance in the build's deterministic order", "Compiled is declared in opm/kernel only" and "The kernel validates per-pair output concreteness itself".

### Requirement: Render is the kernel's sole render path
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Render is the only way the kernel renders an instance against a platform", "The kernel's default core pin carries the registry shape of the platform import" and "A platform module importing its catalogs is the only platform shape the kernel accepts".

### Requirement: The single-provider guard runs inside the build
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The render and the platform's contract inventory read one provider count", "A render refuses on over-subscription exactly when the platform's inventory reads it", "Routable reads false exactly when the render carries an over-subscription row or a collision row", "The provider count lists the registry keys that require each provider-fulfilled contract", "The registry keys of the provider count are the catalog module paths with their major", "The provider count lists a contract whether or not any enabled entry defines it", "Provenance is the registry key, so two majors of one catalog are two providers", "An over-subscribed contract is reported as a row and refuses the render", "The over-subscription rows are built by iterating providedBy unconditionally" and "Keys with default fulfilment may be supplied by any number of transformers".

### Requirement: Unresolved demands are diagnosed with alternatives
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "An unresolved demand is reported with its component, key, kind and alternatives", "An unresolved-demand row names the enabled catalog that lists the demanded key", "An unresolved-demand row names the colliding entries and each disqualified candidate", "An unresolved-demand row marks an unprovided demand on every render", "An unprovided row's message says that the contract is provider-fulfilled", "Render fails on any unresolved demand that the caller's switch did not skip", "The unresolved-demand row distinguishes four cases" and "The defining catalog and the colliding entries of a row are diagnostic only".

### Requirement: A render result carries no presentation strings
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "A render result and a render refusal expose verdicts as structured rows only" and "A typed gate cause carries the same rows as the diagnostics".

### Requirement: A render refuses a platform whose core predates the provider count
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "A render checks the core floor of the platform before staging", "A platform that predates the provider count is refused with PlatformCoreTooOldError", "The refusal of a platform that predates the provider count stages nothing and does not fall back", "The check reads the recorded floor and not the platform's Package" and "One acquired platform stays shareable across concurrent renders".

### Requirement: A render refuses a platform whose enabled entries share contract keys
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The glue emits one collision row for each key in the collision report", "The glue emits no collision row when the platform carries no collision report", "The kernel refuses a colliding platform for every instance and skip switch", "The collision cause is the first joined cause", "The collision cause message names each key, its definers and the remedy" and "The render's collision rows equal the inventory's CollidingEntries".

### Requirement: A render refuses a not-routable platform no row explains
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The kernel refuses a not-routable platform that no row explains" and "The not-routable cause is never raised where the rows explain routable".

### Requirement: Matching stays in the glue and derived rules move into core one at a time
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The matching algorithm stays in the library's render glue" and "A derived rule moves into core one at a time with a parity test".

### Requirement: A render reports every contract its instance requires
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "A render reports the instance's contract demand as data on RequiredContracts", "The contract demand is computed inside the build from the matcher's component set", "The field is set on every RenderResult and every RenderError" and "Computing the demand fails closed".

### Requirement: Each render is its own in-memory build in its own context
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Each render creates a fresh context, evaluates once and shares no built value", "The render module is staged in memory under a synthetic root that does not exist on disk" and "The render module and the overlay-mode inputs are served from the load overlay".

### Requirement: A render writes no staging file
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Kernel.Render writes no staging file" and "A test that asserts that a render writes no staging file owns its temp directory".

## ADDED Requirements

### Requirement: The platform's dependency list wins in the render module
The generated render module's dependency list SHALL be derived by promotion: the platform module's tidied dependency list adopted whole, the instance module's list unioned in for paths only the instance carries, and the platform's entry winning every shared path. No tidy-equivalent and no registry consultation SHALL run at render time to compute the list.

#### Scenario: Platform wins a shared path

- **WHEN** the instance module's `cue.mod` requires catalog build `1.3.0` and the platform module's `cue.mod` carries `1.2.0` for the same path
- **THEN** the render module lists `1.2.0`, and the build evaluates the platform's catalog bytes

#### Scenario: Instance-only paths survive

- **WHEN** the instance module depends on a path the platform module does not carry
- **THEN** the render module lists the instance's entry for that path, and the instance's import resolves

### Requirement: The two input trees enter the build through directory replacements
The two input trees SHALL enter the build through build-local directory replacements; neither input is fetched from a registry.

#### Scenario: Inputs enter through directory replacements

- **WHEN** the render module is built
- **THEN** the two input trees enter the build through build-local directory replacements; neither input is fetched from a registry

### Requirement: Each input module's own path is listed as a default major
Each input module's own path SHALL be listed in the render module's `cue.mod/module.cue` with a placeholder version of its own major and marked as the default major (unless a promoted entry already marks another major of the same root path default), so an unqualified import of the input's own subpackage from inside it resolves through the main module's default-major table exactly as it does when the input is the main module.

#### Scenario: An input's unqualified self-import resolves

- **WHEN** the instance module's root package imports one of its own subpackages without a major qualifier (an `identity` package, the shape `opm module init` writes) and the instance enters the build through a directory replacement
- **THEN** the render module marks the instance module's major default, and the build resolves the import from the replaced directory

### Requirement: Promoted local replacements follow the precedence of the dependencies
When local replacements are enabled for the render, each input's own `cue.mod/local-module.cue` replacements SHALL be promoted under the same precedence as its dependencies: the platform's replacements whole, the instance's only for paths the platform's dependency list does not name.

#### Scenario: A platform replacement of its catalog is honoured

- **WHEN** local replacements are enabled, the platform module's `cue.mod/local-module.cue` replaces its catalog path with a directory, and that directory holds a catalog whose transformer output differs from the published build
- **THEN** the render module's main-module view replaces the catalog path with that directory, the build evaluates the directory's transformer bytes, and the render's replacement rows name the path, the directory and the platform as its source

#### Scenario: An instance replacement on a platform-named path is inert

- **WHEN** local replacements are enabled and the instance module's `cue.mod/local-module.cue` replaces a path the platform module's dependency list carries
- **THEN** the render module carries no replacement for that path, the build evaluates the platform's pinned bytes, and no replacement row names the path

### Requirement: A promoted replacement is written as the input wrote it with a relative target resolved against the input
A promoted replacement SHALL be written into the render module's main-module view exactly as the input wrote it, except that a relative directory target SHALL be resolved against that input's own module root, since the render module's root is elsewhere.

#### Scenario: An instance replacement on an instance-only path is honoured

- **WHEN** local replacements are enabled and the instance module's `cue.mod/local-module.cue` replaces a path the platform module does not carry with a relative directory
- **THEN** the render module's main-module view replaces that path with the directory resolved against the instance module's root, the build resolves the instance's import from it, and a replacement row names the path, the absolute directory and the instance as its source

### Requirement: A replaced path that the promoted list does not carry is listed with a placeholder version
A replaced path the promoted dependency list does not carry SHALL be listed with a placeholder version of its major, so the coverage invariant holds for a replaced OPM-namespace path.

#### Scenario: A replace-only dependency is listed with a placeholder

- **WHEN** local replacements are enabled and the instance module's `cue.mod/module.cue` lists a dependency with no version that its `cue.mod/local-module.cue` replaces with a directory
- **THEN** the render module's `cue.mod/module.cue` lists the path with a placeholder version of its major, the main-module view replaces it, and the render proceeds

### Requirement: A version-less dependency without a promoted replacement is refused before staging
An input dependency that carries no version and is covered by no promoted replacement SHALL be refused before staging with an error naming the path and the input.

#### Scenario: A version-less dependency without a replacement is refused

- **WHEN** an input's `cue.mod/module.cue` lists a dependency with no version and no promoted replacement covers the path
- **THEN** `Render` fails before staging with an error naming the path and the input, and no modfile formatting error is surfaced

### Requirement: Honoured local replacements are reported as rows, not as message strings
When the per-render opt-in for local replacements is on, promoted replacements SHALL be reported on the render diagnostics as rows carrying the replaced path, the absolute target (a directory, or a module path for a module replacement) and the input that supplied it, in path order; the kernel SHALL NOT render them as message strings.

#### Scenario: Honoured replacements are rows

- **WHEN** the per-render opt-in for local replacements is on
- **THEN** promoted replacements SHALL be reported on the render diagnostics as rows carrying the replaced path, the absolute target (a directory, or a module path for a module replacement) and the input that supplied it, in path order

### Requirement: The skew check compares the instance's requirement with the platform's tidied entry by SemVer 2 precedence
For each OPM-namespace path, the kernel SHALL compare the instance module's `cue.mod` requirement against the platform module's tidied entry (never the render module's promoted list). Newer and older SHALL follow SemVer 2 precedence, prerelease builds included (`2.0.0-beta.2` is older than `2.0.0-beta.10`, which is older than `2.0.0`).

#### Scenario: Prerelease builds compare by SemVer precedence

- **WHEN** the skew comparison compares an instance requiring `2.0.0-beta.10` of a path against a platform carrying `2.0.0-beta.2`, and against a platform carrying `2.0.0`
- **THEN** the row against `2.0.0-beta.2` is marked newer and the row against `2.0.0` is not

### Requirement: A newer instance requirement is warned or refused by the caller's policy
When the instance requires a NEWER build than the platform carries, the configured policy decides: warn-and-render (the default when no policy is supplied) marks that path's resolved-versions row as newer and proceeds; refuse fails the render before evaluation. The kernel SHALL NOT render the skew as a message string; a frontend formats the row.

#### Scenario: Newer module warns and renders by default

- **WHEN** the instance requires catalog `1.3.0`, the platform carries `1.2.0`, and no policy is supplied
- **THEN** the render proceeds against `1.2.0` and the result's resolved-versions row for that path names both versions and is marked newer, and the result carries no message string for it

#### Scenario: Refuse policy stops the render

- **WHEN** the same skew exists and the caller configured the refuse policy
- **THEN** `Render` fails before evaluation with an error naming the path and both versions

### Requirement: An older instance requirement is plain data
When the instance requires a NEWER build than the platform carries, the configured policy decides: warn-and-render (the default when no policy is supplied) marks that path's resolved-versions row as newer and proceeds; refuse fails the render before evaluation. A module requiring an OLDER build SHALL produce no such mark; the per-path resolved-versions comparison SHALL always be present in the result as plain data with no severity.

#### Scenario: Older module is data, not a warning

- **WHEN** the instance requires `1.1.0` and the platform carries `1.2.0`
- **THEN** the render proceeds, the resolved-versions row for that path is present in the result's diagnostics, and it is not marked newer

### Requirement: The glue reports the matched pairs, the unify refusals, the unhandled-trait table and the over-subscription rows as data
The generated glue SHALL express matching over the platform's derived `#composedTransformers` with the verdicts as data fields the kernel decodes into the structured diagnostic types without deriving, joining, grouping or re-sorting them: the matched pair set; every candidate the always-unify rung refused, with its conflicting FQNs; the unhandled-trait table; the over-subscription rows.

#### Scenario: A unify refusal names its conflicts once

- **WHEN** a candidate's required primitive body conflicts with the component's body at two FQNs
- **THEN** the diagnostics carry one unify-refusal row for that (component, transformer) listing both FQNs, and the demand's disqualified list names the same transformer with the same FQNs

### Requirement: The glue reports each unresolved demand with its same-base alternatives as data
The generated glue SHALL express matching over the platform's derived `#composedTransformers` with the verdicts as data fields the kernel decodes into the structured diagnostic types without deriving, joining, grouping or re-sorting them: every unresolved demand with the same-base alternatives the platform implements (sorted in the build on the `alpha < beta < GA`, then major, then minor apiVersion ladder), its unprovided marker, and each disqualified candidate with the FQNs it conflicted at.

#### Scenario: Verdicts decode beside a failing gate

- **WHEN** a component demands a resource FQN no transformer on the platform requires
- **THEN** `Render` fails, and the returned error carries a decoded unresolved-demand row for that (instance, component, FQN) with the platform's same-base alternatives in ladder order

### Requirement: The glue reports each unmatched component with its candidates as data
The generated glue SHALL express matching over the platform's derived `#composedTransformers` with the verdicts as data fields the kernel decodes into the structured diagnostic types without deriving, joining, grouping or re-sorting them: every unmatched component with every candidate the demand walk reached for it and, for a predicate refusal, the required labels the component lacked or carried with a different value.

#### Scenario: An unmatched component carries its candidates on the diagnostics

- **WHEN** a component's only candidate is refused by the label predicate on one required label
- **THEN** the diagnostics list that component as unmatched with the candidate's transformer, an unmatched verdict and the missing label, and the typed unmatched-components cause carries the same row

### Requirement: The glue reports each demand skipped under the caller's switch as data
The generated glue SHALL express matching over the platform's derived `#composedTransformers` with the verdicts as data fields the kernel decodes into the structured diagnostic types without deriving, joining, grouping or re-sorting them: every demand skipped under the caller's switch (see "A caller may skip unprovided provider-fulfilled demands").

#### Scenario: A skipped demand is a verdict as data

- **WHEN** a demand is skipped under the caller's switch
- **THEN** it is among the verdicts as data fields the kernel decodes into the structured diagnostic types without deriving, joining, grouping or re-sorting them

### Requirement: The rungs are evaluated only for a component's candidates
The always-unify and predicate rungs SHALL be evaluated only for a component's candidates (the transformers the demand walk reached for it), never for every transformer on the platform. The always-unify rung SHALL be plain unification with no provenance exclusion.

#### Scenario: A rung sees only a component's candidates

- **WHEN** a component's candidates are evaluated
- **THEN** The always-unify and predicate rungs SHALL be evaluated only for a component's candidates (the transformers the demand walk reached for it), never for every transformer on the platform.

### Requirement: The reported pair set agrees with plain predicate matching
Matching semantics are those of the render-parity oracle: with the skip switch off, the pair set the glue reports SHALL equal the pair set plain predicate matching over the same inputs produces (`render-parity`, "Matched pair sets agree"); with it on, the reported pair set SHALL be that set minus every pair of an omitted component.

#### Scenario: An omitted component leaves the pair set

- **WHEN** the skip switch is on and a component is omitted for an unprovided resource demand while its other resources match transformers
- **THEN** the reported pair set carries no pair of that component, and no rendered output is decoded for it

### Requirement: The fail-closed demand gate holds while the diagnostics stay readable
The fail-closed demand gate SHALL hold: an unresolved demand that was not skipped, or an unmatched component that was not omitted, refuses the render while the diagnostics remain readable and decoded; an effectively-optional unhandled trait is reported on the diagnostics' unhandled-trait table; an unhandled trait with an UNSTATED optional posture refuses as a build error naming the trait's own `optional` field.

#### Scenario: The gate refuses beside readable diagnostics

- **WHEN** an unresolved demand was not skipped
- **THEN** the gate refuses the render while the diagnostics remain readable and decoded

### Requirement: The glue unifies each matched pair's transform once
Failed pairs are not a glue verdict. The glue SHALL unify each matched pair's transform once, in `rendered`, and SHALL NOT evaluate it a second time for diagnostics.

#### Scenario: A matched pair is unified once

- **WHEN** the glue evaluates a matched pair
- **THEN** The glue SHALL unify each matched pair's transform once, in `rendered`, and SHALL NOT evaluate it a second time for diagnostics.

### Requirement: The kernel names the failed pairs from each matched pair's rendered output
The kernel SHALL name the failed pairs on `RenderDiagnostics.FailedPairs`, in pair order, from each matched pair's rendered output: a pair is listed when its output's `Value.Err()` is non-nil, wherever inside the output the error arises, and including an output whose root is itself incomplete (its value cannot be resolved to a struct or a list). An output whose root is a struct or a list whose only defect is non-concrete fields SHALL NOT be listed; the per-pair concreteness check refuses it.

#### Scenario: Healthy pairs render beside a failing pair

- **WHEN** one matched pair's transformer errors while sibling pairs are healthy
- **THEN** the failing pair is reported as data naming the pair, and the sibling verdicts remain readable

### Requirement: FailedPairs is filled on every RenderError raised after the build
The kernel SHALL fill `FailedPairs` on every `RenderError` raised after the build, a gate refusal included, and SHALL leave it empty on a successful render.

#### Scenario: Failed pairs are reported on a gate refusal

- **WHEN** one component's demand is unresolved, so the gate refuses the render, while another component's matched pair has a transformer whose output conflicts
- **THEN** `Render` fails with a `RenderError` whose cause is the gate's, and its diagnostics' `FailedPairs` names the conflicting pair and no other

### Requirement: Rendered output decodes with provenance in the build's deterministic order
On a passing gate, `Render` SHALL decode `rendered` into `[]*kernel.Compiled`, each carrying instance, component and transformer provenance, in the build's deterministic order.

#### Scenario: Provenance on every object

- **WHEN** a render of two matched pairs succeeds
- **THEN** every returned `*kernel.Compiled` names its instance, component and transformer FQN

### Requirement: Compiled is declared in opm/kernel only
`Compiled` SHALL be declared in `opm/kernel`, beside the verb that produces it, with exactly the fields `Value cue.Value`, `Instance`, `Component` and `Transformer`; no other package under `opm/` SHALL declare a compiled-output type.

#### Scenario: One compiled-output type

- **WHEN** a consumer inspects the packages under `opm/`
- **THEN** `Compiled` is declared in `opm/kernel` only and no `opm/core` package exists

### Requirement: The kernel validates per-pair output concreteness itself
The kernel SHALL validate per-pair output concreteness itself: an incomplete (non-error) pair output SHALL fail the render at a path naming the pair.

#### Scenario: Incomplete pair output refuses

- **WHEN** a transformer's output evaluates non-concrete without erroring
- **THEN** `Render` fails with an error whose path names the (component, transformer) pair

### Requirement: Render is the only way the kernel renders an instance against a platform
`Kernel.Render` SHALL be the only way the kernel renders an instance against a platform. The kernel SHALL expose no `Compile`, `Match` or `Materialize` method and no materialized-platform type; a dry run is `Render` with the rendered output discarded, since the build evaluates every matched pair regardless.

#### Scenario: Old entry points are gone

- **WHEN** a consumer inspects the exported identifiers of `opm/kernel`
- **THEN** none of `Compile`, `Match`, `Materialize`, `SynthesizePlatform`, `CompileInput`, `MatchInput`, `CompileResult`, `MatchPlan` exists, and no `opm/materialize` or `opm/compile` package exists

### Requirement: The kernel's default core pin carries the registry shape of the platform import
The kernel's default core schema pin SHALL be a release carrying the enhancement 0019 ("A platform imports its catalog and embeds it whole into the registry entry, which derives everything else") registry shape (`schema-dispatch`, "DefaultSchemaModule names an exact core release"), so every artifact the kernel synthesizes or validates is judged against that shape.

#### Scenario: The default pin carries the registry shape

- **WHEN** the kernel's default core schema pin is read
- **THEN** The kernel's default core schema pin SHALL be a release carrying the enhancement 0019 ("A platform imports its catalog and embeds it whole into the registry entry, which derives everything else") registry shape (`schema-dispatch`, "DefaultSchemaModule names an exact core release"), so every artifact the kernel synthesizes or validates is judged against that shape.

### Requirement: A platform module importing its catalogs is the only platform shape the kernel accepts
A platform module importing its catalogs is the only platform shape the kernel accepts: the platform shape gate SHALL validate every `#registry` entry for completeness (`helper-packages`, "Loader shape gate validates identity and registry completeness"), so an entry that names no embedded catalog is refused at acquisition. Core derives the entry's `version` from the embedded catalog's stamped identity, and with no catalog that readout is a missing required field.

#### Scenario: A subscription-shaped platform is refused

- **WHEN** a platform package declares a registry entry with a `version` scalar and no embedded catalog
- **THEN** `AcquirePlatformFromDir` fails with an error wrapping `ErrMissingRequiredField` that names the entry's `version` as a required field the embedded catalog would have supplied, and no render is attempted

### Requirement: The render and the platform's contract inventory read one provider count
The generated glue SHALL read the platform's provider count from core's derived contract inventory and SHALL NOT compute a count of its own. Source: enhancement 0010 ("A contract declares where its fulfilment comes from, and a provider-fulfilled contract admits exactly one provider") and enhancement 0015 ("One provider per contract stands; provider classes are rejected in this entry").

#### Scenario: Inventory and render agree on every served platform

- **WHEN** the parity test acquires every served render platform (each `testdata/render/platform*` directory), reads `Contracts()` and renders that platform's classified instance fixture against it
- **THEN** the sorted `OverSubscribed` list equals the keys of the render's over-subscription rows, every row's registry keys equal `ProvidedBy` for its key, `Routable` is true exactly when the render carries neither an over-subscription row nor a collision row, and the decoded routable diagnostic equals `Routable`, whether or not the render also refuses for an unrelated reason

### Requirement: A render refuses on over-subscription exactly when the platform's inventory reads it
The one count is the provider count of requirement "The render and the platform's contract inventory read one provider count". Because both the render and the platform's `Contracts()` read this one count, a render SHALL refuse on over-subscription exactly when the platform's inventory reads a non-empty `OverSubscribed`. Source: enhancement 0015 ("An unfulfilled contract is reported, never refused; only over-subscription refuses platform-package generation").

#### Scenario: A render refuses when the inventory reads an over-subscription

- **WHEN** the platform's inventory reads a non-empty `OverSubscribed`
- **THEN** a render refuses on over-subscription

### Requirement: Routable reads false exactly when the render carries an over-subscription row or a collision row
The one count is the provider count of requirement "The render and the platform's contract inventory read one provider count". Because both the render and the platform's `Contracts()` read this one count, `Routable` SHALL read false exactly when the render carries an over-subscription row or a collision row. Source: enhancement 0015 ("An unfulfilled contract is reported, never refused; only over-subscription refuses platform-package generation").

#### Scenario: Routable reads false when the render carries a row

- **WHEN** the render carries an over-subscription row or a collision row
- **THEN** `Routable` reads false

### Requirement: The provider count lists the registry keys that require each provider-fulfilled contract
The provider count SHALL be `#Platform.#contracts.providedBy`: for every contract key declared `fulfilment: "provider"` on a required demand (`requiredResources` and `requiredTraits`) of any transformer of an enabled `#registry` entry, the sorted registry keys of the entries whose transformers require it. Source: enhancement 0015 ("One provider per contract stands; provider classes are rejected in this entry").

#### Scenario: The provider count lists the registry keys that require a contract

- **WHEN** a contract key is declared `fulfilment: "provider"` on a required demand of a transformer of an enabled `#registry` entry
- **THEN** the provider count lists the sorted registry keys of the entries whose transformers require it

### Requirement: The registry keys of the provider count are the catalog module paths with their major
The registry keys of requirement "The provider count lists the registry keys that require each provider-fulfilled contract" SHALL be the catalog module paths with their major, bound by core to each entry's `#catalog.metadata.modulePath`. Source: enhancement 0015 ("One provider per contract stands; provider classes are rejected in this entry").

#### Scenario: A registry key is a catalog module path with its major

- **WHEN** the provider count lists a registry key
- **THEN** the key is the catalog module path with its major, bound by core to the entry's `#catalog.metadata.modulePath`

### Requirement: The provider count lists a contract whether or not any enabled entry defines it
The provider count of requirement "The provider count lists the registry keys that require each provider-fulfilled contract" SHALL list those registry keys whether or not any enabled entry defines the contract. Source: enhancement 0015 ("One provider per contract stands; provider classes are rejected in this entry").

#### Scenario: Providers count when the defining catalog is disabled

- **WHEN** the platform carries the defining `cat@v0` entry with `enable: false` and enables `cat2` 0.2.0 and `cat@v1`, whose transformers both require the gateway contract
- **THEN** `Render` fails with one over-subscription row naming the gateway key and the registry keys `cat2@v0` and `cat@v1`, and the platform's `Contracts()` reads the same key in `OverSubscribed` and `Routable` false

### Requirement: Provenance is the registry key, so two majors of one catalog are two providers
Provenance SHALL be the registry key, never a value parsed out of an FQN or read off the transformer, so two majors of one catalog are two providers and two transformers of one entry are one. Source: enhancement 0015 ("One provider per contract stands; provider classes are rejected in this entry" and "An unfulfilled contract is reported, never refused; only over-subscription refuses platform-package generation"), as recounted by core change `count-providers-per-registry-entry`.

#### Scenario: Two majors of one provider catalog are two providers

- **WHEN** the platform enables `cat@v0` and `cat@v1`, and a transformer of each requires `cat@v0`'s provider-fulfilled gateway contract
- **THEN** `Render` fails with one over-subscription row naming the gateway key and the registry keys `cat@v0` and `cat@v1`, and the platform's `Contracts()` reads the same key in `OverSubscribed` and `Routable` false

### Requirement: An over-subscribed contract is reported as a row and refuses the render
Every key core lists in `#contracts.overSubscribed` SHALL be reported as an over-subscription row naming the key and the registry keys `providedBy` holds for it, and the render SHALL refuse through the fail-closed gate with one typed over-subscription cause carrying every such row. Source: enhancement 0015 ("An unfulfilled contract is reported, never refused; only over-subscription refuses platform-package generation").

#### Scenario: Second provider refused in-build

- **WHEN** two catalogs embedded in the platform each supply a transformer requiring a contract declared `fulfilment: "provider"`
- **THEN** `Render` fails with a typed over-subscription cause whose row names the key and both registry keys, and the diagnostics remain readable beside the refusal

### Requirement: The over-subscription rows are built by iterating providedBy unconditionally
The over-subscription rows SHALL be built by iterating `providedBy` unconditionally, so a platform value lacking it fails the build instead of reading as zero providers. Source: enhancement 0010 ("A contract declares where its fulfilment comes from, and a provider-fulfilled contract admits exactly one provider") and enhancement 0015 ("One provider per contract stands; provider classes are rejected in this entry").

#### Scenario: The rows iterate providedBy unconditionally

- **WHEN** the over-subscription rows are built
- **THEN** they are built by iterating `providedBy` unconditionally, so a platform value lacking it fails the build instead of reading as zero providers

### Requirement: Keys with default fulfilment may be supplied by any number of transformers
The generated glue SHALL read the platform's provider count from core's derived contract inventory and SHALL NOT compute a count of its own. Keys with default (`catalog`) fulfilment MAY be supplied by any number of transformers from any number of catalogs. Source: enhancement 0010 ("A contract declares where its fulfilment comes from, and a provider-fulfilled contract admits exactly one provider"), as recounted by core change `count-providers-per-registry-entry`.

#### Scenario: Catalog-fulfilled plurality allowed

- **WHEN** many transformers across catalogs require a contract with default fulfilment
- **THEN** the render proceeds and every candidate participates in matching

### Requirement: An unresolved demand is reported with its component, key, kind and alternatives
A demanded contract for which the platform holds no candidate, or for which every candidate is disqualified (by unification or by predicate), SHALL be reported as an unresolved-demand row carrying the component, the contract key, the kind, the same-base alternatives the platform does implement (sorted in the build on the apiVersion ladder, so the row is deterministic).

#### Scenario: Different apiVersion named

- **WHEN** the platform implements the same contract base at `v1alpha1`, `v1` and `v1beta1` only, and the component demands `v2`
- **THEN** the unresolved-demand row lists the alternatives as `v1alpha1`, `v1beta1`, `v1`

### Requirement: An unresolved-demand row names the enabled catalog that lists the demanded key
A demanded contract for which the platform holds no candidate, or for which every candidate is disqualified (by unification or by predicate), SHALL be reported as an unresolved-demand row carrying the registry key of the enabled catalog that lists the demanded key in its contract maps (empty when no enabled catalog lists it, or when more than one does; read inside the build from the platform's derived contract inventory, never parsed off the FQN).

#### Scenario: A defined but unimplemented contract names its catalog

- **WHEN** an enabled catalog lists a trait in its contract maps (whatever its fulfilment), no transformer on the platform requires it, and a component attaches it load-bearing
- **THEN** `Render` fails with an unresolved-demands cause whose row carries that catalog's registry key as the defining catalog and no alternatives, and the message names the catalog beside the component and key

#### Scenario: A disabled catalog defines nothing

- **WHEN** the only catalog listing the demanded key is a registry entry with `enable: false`
- **THEN** the unresolved-demand row carries no defining catalog and no colliding entries

### Requirement: An unresolved-demand row names the colliding entries and each disqualified candidate
A demanded contract for which the platform holds no candidate, or for which every candidate is disqualified (by unification or by predicate), SHALL be reported as an unresolved-demand row carrying the registry keys of the enabled entries defining the key when it is a collision (read from `#contracts.collidingEntries` when the platform carries it, empty otherwise), and, when candidates existed, each disqualified candidate's transformer with the FQNs it conflicted at.

#### Scenario: A colliding contract row names its entries

- **WHEN** a component's demand on a colliding platform goes unresolved on a key two enabled entries define
- **THEN** the row carries no defining catalog, carries the colliding registry keys, and its message says the contract is defined by more than one enabled registry entry and names them, never that no enabled catalog defines it

### Requirement: An unresolved-demand row marks an unprovided demand on every render
A demanded contract for which the platform holds no candidate, or for which every candidate is disqualified (by unification or by predicate), SHALL be reported as an unresolved-demand row carrying whether the demand is unprovided (its contract declares `fulfilment: "provider"` and no enabled registry entry carries a transformer requiring the key). The unprovided marker SHALL be computed on every render, whatever the skip switch says.

#### Scenario: A catalog-fulfilled row is not marked

- **WHEN** a component attaches a load-bearing trait with default (`catalog`) fulfilment that no transformer handles
- **THEN** the unresolved-demand row is not marked unprovided, and its message carries no provider-fulfilled text

### Requirement: An unprovided row's message says that the contract is provider-fulfilled
An unprovided row's message SHALL additionally say that the contract is provider-fulfilled and nothing on the platform provides it, after the case text and before any disqualified-candidate count.

#### Scenario: An unprovided row says so

- **WHEN** a component attaches a load-bearing trait whose contract an enabled catalog lists and declares `fulfilment: "provider"`, no transformer on the platform requires it, and the skip switch is off
- **THEN** the unresolved-demand row is marked unprovided, and its message names the defining catalog and then states that the contract is provider-fulfilled and nothing on the platform provides it

### Requirement: Render fails on any unresolved demand that the caller's switch did not skip
Every resource a component declares is a required demand. `Render` SHALL fail on any unresolved demand that the caller's switch did not skip, through the typed gate cause, while returning the full diagnosis.

#### Scenario: An unskipped unresolved demand fails the render

- **WHEN** a component declares a resource and its demand is unresolved and not skipped
- **THEN** `Render` SHALL fail on any unresolved demand that the caller's switch did not skip, through the typed gate cause, while returning the full diagnosis.

### Requirement: The unresolved-demand row distinguishes four cases
The unresolved-demand row SHALL distinguish four cases: "defined by `<catalog>` and nothing on this platform implements it" (defining catalog named, no alternatives), "implemented at a different apiVersion" (alternatives listed), "defined by more than one enabled registry entry" (no alternatives, the colliding entries named), and "no enabled catalog defines this contract" (no defining catalog, no colliding entries, no alternatives).

#### Scenario: Undemandable resource fails the render

- **WHEN** a component demands a resource contract no embedded catalog implements or lists
- **THEN** `Render` fails with an unresolved-demands cause whose row names the component and key with no alternatives, no defining catalog and no colliding entries, and the message says no enabled catalog defines the contract

### Requirement: The defining catalog and the colliding entries of a row are diagnostic only
The defining catalog and the colliding entries of an unresolved-demand row are diagnostic only (enhancement 0015, "An unfulfilled contract is reported, never refused; only over-subscription refuses platform-package generation"): their presence or absence SHALL NOT change whether the demand refuses.

#### Scenario: Diagnostic fields do not change the refusal

- **WHEN** an unresolved-demand row carries or lacks a defining catalog or colliding entries
- **THEN** their presence or absence SHALL NOT change whether the demand refuses

### Requirement: An unstated trait posture fails closed as a build error
An unhandled trait's effect SHALL be governed by its effective `optional` value read from the component's trait attachment. An unstated posture (non-concrete `optional`) SHALL fail closed as a build error naming the trait's own `optional` field.

#### Scenario: An unstated posture fails closed

- **WHEN** an unhandled trait has an unstated posture (non-concrete `optional`)
- **THEN** the posture SHALL fail closed as a build error naming the trait's own `optional` field

### Requirement: A render result and a render refusal expose verdicts as structured rows only
A render result and a render refusal SHALL expose verdicts as structured rows only. The kernel SHALL NOT attach human-readable warning or advisory messages to a result; every advisory fact (an unhandled optional trait, a module requiring a newer build than the platform carries, a demand skipped under the caller's switch) SHALL be readable from the diagnostics rows, and the diagnostics type SHALL document where each advisory fact lives.

#### Scenario: Advisory facts are rows

- **WHEN** a render succeeds with one unhandled optional trait and one path in warn-mode skew
- **THEN** the result exposes the trait on the unhandled-trait table and the path on a resolved-versions row marked newer, and no field of the result holds a formatted message

#### Scenario: A skipped demand is a row

- **WHEN** a render succeeds under the skip switch with one skipped trait demand
- **THEN** the result exposes the demand as a skipped-demand row, and no field of the result holds a formatted message for it

### Requirement: A typed gate cause carries the same rows as the diagnostics
Each row-carrying typed gate cause (colliding contracts, unresolved demands, unmatched components, over-subscribed contracts) SHALL carry the same rows the diagnostics carry, in the same order, and SHALL NOT synthesize causes of another kind. The not-routable cause carries no rows: it is raised only from the decoded routable diagnostic.

#### Scenario: A gate cause is the diagnostics rows

- **WHEN** `Render` refuses on two unresolved demands, or on a platform with colliding contracts
- **THEN** the unresolved-demands cause, or the collision cause, reachable through `errors.As` carries exactly the rows the refusal's diagnostics carry, and unwrapping it yields no cause of a different kind

### Requirement: The kernel's decoded refusal is authoritative and never reads the gate field
The kernel's decoded refusal SHALL remain the authoritative one, since it carries the typed causes, and SHALL be decided from decoded rows and the decoded routable diagnostic only, never by reading `gate`.

#### Scenario: The decoded refusal decides

- **WHEN** the kernel decides a refusal
- **THEN** The kernel's decoded refusal SHALL remain the authoritative one, since it carries the typed causes, and SHALL be decided from decoded rows and the decoded routable diagnostic only, never by reading `gate`.

### Requirement: With the switch off a render is as without the switch
With the switch off, the render's verdicts, refusals and rendered output SHALL be exactly those of a render without the switch. Source: core `SPEC.md` §2.1 and §3.1 and capability `contract-fulfilment`, as amended by core changes `skip-unprovided-provider-demands` and `count-providers-per-registry-entry`.

#### Scenario: The switch off changes nothing

- **WHEN** the switch is off
- **THEN** the render's verdicts, refusals and rendered output SHALL be exactly those of a render without the switch

### Requirement: A skipped trait demand does not refuse the render
With the switch on: A skipped trait demand SHALL NOT refuse the render. Its component SHALL render every pair it matched, and nothing SHALL render for the trait. Source: core `SPEC.md` §2.1 and §3.1 and capability `contract-fulfilment`, as amended by core changes `skip-unprovided-provider-demands` and `count-providers-per-registry-entry`.

#### Scenario: A skipped trait leaves its pairs

- **WHEN** the switch is on and a trait demand is skipped
- **THEN** Its component SHALL render every pair it matched, and nothing SHALL render for the trait.

### Requirement: A skipped resource demand omits its component
With the switch on: A skipped resource demand SHALL NOT refuse the render, and its component SHALL be omitted: no pair of that component SHALL render, and the component SHALL NOT be reported as unmatched. A partly satisfied component never renders. Source: core `SPEC.md` §2.1 and §3.1 and capability `contract-fulfilment`, as amended by core changes `skip-unprovided-provider-demands` and `count-providers-per-registry-entry`.

#### Scenario: A skipped resource omits the component

- **WHEN** the switch is on and a resource demand is skipped
- **THEN** its component SHALL be omitted: no pair of that component SHALL render, and the component SHALL NOT be reported as unmatched

### Requirement: Every skipped demand is reported as a skipped-demand row
Every skipped demand SHALL be reported as a skipped-demand row carrying the component, the contract key, the kind (`resource` or `trait`), the defining catalog's registry key (empty when no enabled catalog lists the key), the same-base alternatives the platform implements, and whether the component was omitted. Source: core `SPEC.md` §2.1 and §3.1 and capability `contract-fulfilment`, as amended by core changes `skip-unprovided-provider-demands` and `count-providers-per-registry-entry`.

#### Scenario: A skipped demand row carries the component and the kind

- **WHEN** a demand is skipped
- **THEN** Every skipped demand SHALL be reported as a skipped-demand row carrying the component, the contract key, the kind (`resource` or `trait`)

### Requirement: The omission flag is set on every skipped row of an omitted component
With the switch on: The omission flag SHALL be set on every skipped row of an omitted component, trait rows included. Source: core `SPEC.md` §2.1 and §3.1 and capability `contract-fulfilment`, as amended by core changes `skip-unprovided-provider-demands` and `count-providers-per-registry-entry`.

#### Scenario: The flag is set on trait rows too

- **WHEN** a component is omitted and has a skipped trait row
- **THEN** The omission flag SHALL be set on every skipped row of an omitted component, trait rows included.

### Requirement: Every other refusal stands with the switch on
With the switch on: Every other refusal SHALL stand: a catalog-fulfilled unresolved demand, a provider-fulfilled demand for which a provider exists but did not match or was disqualified, an over-subscribed provider-fulfilled contract, and an unmatched component (other than an omitted one) SHALL refuse the render as without the switch. The skipped-demand rows SHALL be readable on the diagnostics beside such a refusal. Source: core `SPEC.md` §2.1 and §3.1 and capability `contract-fulfilment`.

#### Scenario: Skipped rows are readable beside a refusal

- **WHEN** the render refuses for another reason with the switch on
- **THEN** The skipped-demand rows SHALL be readable on the diagnostics beside such a refusal.

### Requirement: An unhandled optional trait is never a skipped row
With the switch on: An unhandled trait whose effective `optional` is true SHALL remain an unhandled-trait table entry, never a skipped row. Source: core `SPEC.md` §2.1 and §3.1 and capability `contract-fulfilment`, as amended by core changes `skip-unprovided-provider-demands` and `count-providers-per-registry-entry`.

#### Scenario: An optional trait stays in its table

- **WHEN** the switch is on and an unhandled trait's effective `optional` is true
- **THEN** An unhandled trait whose effective `optional` is true SHALL remain an unhandled-trait table entry, never a skipped row.

### Requirement: The render module carries the switch as a literal
The generated render module SHALL carry the switch as a literal, so its own `gate` field agrees with the kernel's verdict under both values. Source: core `SPEC.md` §2.1 and §3.1 and capability `contract-fulfilment`, as amended by core changes `skip-unprovided-provider-demands` and `count-providers-per-registry-entry`.

#### Scenario: The switch is a literal

- **WHEN** the render module is generated
- **THEN** The generated render module SHALL carry the switch as a literal, so its own `gate` field agrees with the kernel's verdict under both values.

### Requirement: A render checks the core floor of the platform before staging
`Kernel.Render` SHALL check, before staging, that the platform's core floor is met: that the platform's `#contracts` carries `providedBy`, as recorded when the platform was constructed and reported by `Platform.CoreFloor()` (`platform-artifact`, "A platform records its core floor and contract inventory at construction"). The check is sound because the render build evaluates the platform module's own core pin, the same one its `Package` was built from.

#### Scenario: A current-core platform is not affected

- **WHEN** a platform module pinning core `2.0.0-alpha.12` or later is rendered
- **THEN** the core floor raises no error and the render proceeds to staging

### Requirement: A platform that predates the provider count is refused with PlatformCoreTooOldError
When the platform's `#contracts` does not carry `providedBy` (the platform module pins a core release older than `2.0.0-alpha.12`, or carries no `#contracts` at all), `Render` SHALL return an error wrapping the typed `PlatformCoreTooOldError` that names the platform, the missing field and the first core release carrying it.

#### Scenario: An older core raises the typed error

- **WHEN** the platform module pins a core release older than `2.0.0-alpha.12`
- **THEN** `Render` SHALL return an error wrapping the typed `PlatformCoreTooOldError` that names the platform, the missing field and the first core release carrying it.

### Requirement: The refusal of a platform that predates the provider count stages nothing and does not fall back
The refusal with `PlatformCoreTooOldError` SHALL NOT be a `*RenderError`, SHALL stage no render module and SHALL NOT fall back to a count of the kernel's own: a render never runs on a platform whose inventory would disagree with it.

#### Scenario: An older-core platform is refused before staging

- **WHEN** a platform module identical to a served fixture but pinning core `2.0.0-alpha.10` is acquired and rendered
- **THEN** `Render` returns an error from which `errors.As` extracts a `PlatformCoreTooOldError` naming the platform, the field `providedBy` and the release `2.0.0-alpha.12`, the error is not a `*RenderError`, no result is returned, and no staging file is written

### Requirement: The check reads the recorded floor and not the platform's Package
The check of the platform's core floor SHALL read the recorded floor through `CoreFloor()`, and on a platform the constructor built it SHALL NOT read the platform's `Package`; a platform the constructor did not build decodes `Package` once, on its first `CoreFloor()` or `Contracts()` call (`platform-artifact`, "A platform records its core floor and contract inventory at construction"). Apart from the floor, `Render` SHALL read only the platform's `Metadata` and `Source`.

#### Scenario: A render reads no platform Package

- **WHEN** a current-core platform is acquired, its `Package` is replaced with the zero `cue.Value`, and it is rendered with an acquired instance
- **THEN** the core floor raises no error and the render produces the same objects as a render of the unchanged platform

### Requirement: One acquired platform stays shareable across concurrent renders
Apart from the floor, `Render` SHALL read only the platform's `Metadata` and `Source`. One acquired platform therefore stays shareable, as data, across concurrent `Render` calls on one Kernel.

#### Scenario: A platform shared by concurrent renders stays race-free

- **WHEN** one acquired current-core platform is shared by several goroutines, each calling `Render` on one Kernel
- **THEN** every render passes the core floor and produces the same objects, with no data race reported under the race detector

### Requirement: The glue emits one collision row for each key in the collision report
The generated glue SHALL read the platform's contract collisions from core's derived contract inventory and SHALL NOT compute them: for every key in `#Platform.#contracts.collisions`, in that (ascending) order, it SHALL emit one collision row naming the key and the registry keys `#contracts.collidingEntries` holds for it (path plus major, ascending). Source: an open question of enhancement 0026, core change `fold-colliding-contract-keys`.

#### Scenario: A colliding bridge platform is refused instead of rendering twice

- **WHEN** the platform enables `maj@v0` 0.1.0 and `maj@v1` 1.4.0, which list the same container, expose and backup keys, and `maj@v1` ships a bridge transformer requiring `maj@v0`'s container, and an instance of a `maj@v0` module is rendered
- **THEN** `Render` fails with a `*RenderError` whose joined causes start with a typed collision cause carrying three rows (the container, backup and expose keys in ascending order, each naming `maj@v0` and `maj@v1`), no compiled object is returned, and the diagnostics carry the same three rows

### Requirement: The glue emits no collision row when the platform carries no collision report
The glue SHALL emit the collision rows only when the platform value carries `collisions`, and SHALL emit none otherwise: every core release carrying `#contracts` without the report fails to evaluate a platform whose enabled entries share a key, so an absent report provably means no collision. Source: an open question of enhancement 0026, core change `fold-colliding-contract-keys`.

#### Scenario: A platform pinning a core without the report renders unchanged

- **WHEN** the served healthy platform, re-pinned to core `2.0.0-alpha.12`, is rendered
- **THEN** the render succeeds exactly as before this change, with no collision row and a routable diagnostic reading true

### Requirement: The kernel refuses a colliding platform for every instance and skip switch
The kernel SHALL decode the collision rows onto the render diagnostics and SHALL refuse the render, through the fail-closed gate, with one typed collision cause carrying every row unchanged. The refusal is platform-wide: it SHALL hold for every instance and whatever the caller's skip switch says. Source: an open question of enhancement 0026, core change `fold-colliding-contract-keys`.

#### Scenario: A collision refuses under the skip switch

- **WHEN** the same colliding platform is rendered with the skip switch on
- **THEN** `Render` fails with the same typed collision cause

### Requirement: The collision cause is the first joined cause
When a render refuses for several reasons, the collision cause SHALL be the first joined cause, ahead of unresolved demands, over-subscribed contracts and unmatched components, because a colliding key is absent from `definedBy`, `requiredBy` and the comparability report and the other rows are read against a distorted inventory. Source: an open question of enhancement 0026, core change `fold-colliding-contract-keys`.

#### Scenario: A collision and an over-subscription are both named, collision first

- **WHEN** the colliding platform also enables `bprov@v0` and `bprov@v1`, whose transformers both require the colliding provider-fulfilled backup trait
- **THEN** `Render` fails with a collision cause and an over-subscription cause for the backup key, reachable through `errors.As`, and the collision cause is the first joined cause

### Requirement: The collision cause message names each key, its definers and the remedy
The collision cause's message SHALL name, per row, the key, the number of enabled registry entries defining it and those entries, and SHALL say that a contract key must have exactly one enabled definer until side-by-side catalog majors are supported, so all but one of the entries must be disabled. Source: an open question of enhancement 0026, core change `fold-colliding-contract-keys`.

#### Scenario: The message names the keys and the definers

- **WHEN** the render refuses a colliding platform
- **THEN** the cause's message SHALL name, per row, the key, the number of enabled registry entries defining it and those entries

### Requirement: The render's collision rows equal the inventory's CollidingEntries
Because the render and the platform's `Contracts()` read the same report, the render's collision rows SHALL equal the inventory's `CollidingEntries` on every platform. Source: an open question of enhancement 0026 (the interim safety net ahead of that enhancement, in which a platform may admit several majors of one catalog and each resolution holds exactly one major per catalog), core change `fold-colliding-contract-keys`.

#### Scenario: Collision rows equal the inventory on every served platform

- **WHEN** the parity test renders every served render platform
- **THEN** the render's collision rows, keyed by contract, equal the platform inventory's `CollidingEntries`, and a platform with no collision carries no row

### Requirement: The kernel refuses a not-routable platform that no row explains
The kernel SHALL decode core's `#contracts.routable` onto the render diagnostics, and SHALL refuse the render with a typed not-routable cause when it reads false and the diagnostics carry neither a collision row nor an over-subscription row. The cause SHALL be the last joined cause.

#### Scenario: An unexplained not-routable verdict refuses

- **WHEN** the gate is given decoded diagnostics whose routable reads false and which carry no collision or over-subscription row
- **THEN** the gate refuses with the typed not-routable cause, joined after every other cause present

### Requirement: The not-routable cause is never raised where the rows explain routable
The not-routable cause exists so that the kernel's decision agrees with the render module's own gate (which reads `routable`) even when a future core adds a term to `routable` that no row reports; on every platform whose `routable` is explained by its rows it SHALL NOT be raised.

#### Scenario: No served platform raises the catch-all

- **WHEN** the parity test renders every served render platform
- **THEN** no refusal carries the not-routable cause, and the decoded routable diagnostic equals the inventory's `Routable`

### Requirement: The matching algorithm stays in the library's render glue
The matching algorithm (which transformers a component is paired with, the demand buckets, and the verdicts the glue emits) SHALL stay in the library's render glue, as enhancement 0019 ("Matching moves into the render build; verdicts are data; the D30 carve-out is deleted" and "`#Platform.#matchers` is removed, and the render build's glue owns the reverse index") decide. ADR-012 records this.

#### Scenario: ADR-012 records where matching lives

- **WHEN** a developer reads `adr/012-matching-stays-in-the-library-glue.md`
- **THEN** it states that the matching algorithm stays in the library glue under enhancement 0019 ("Matching moves into the render build; verdicts are data; the D30 carve-out is deleted" and "`#Platform.#matchers` is removed, and the render build's glue owns the reverse index"), that single derived rules move into core one at a time with a parity test, and that `#contracts.providedBy` is the precedent
- **AND** it names moving `#Match` whole into core (core#62) and a core reverse index as rejected alternatives with reasons

### Requirement: A derived rule moves into core one at a time with a parity test
A single derived rule MAY move into core when core can compute it from core shapes alone. Each such move SHALL be one rule at a time, SHALL be an additive core release, and SHALL ship with a parity test proving that the core field and the derivation it replaces agree over the served fixtures. `#contracts.providedBy`, read by the glue and checked by `opm/kernel/render_inventory_parity_test.go`, is the precedent.

#### Scenario: A derived rule moves with a parity test

- **WHEN** a single derived rule moves into core
- **THEN** Each such move SHALL be one rule at a time, SHALL be an additive core release, and SHALL ship with a parity test

### Requirement: A render reports the instance's contract demand as data on RequiredContracts
The render SHALL report the instance's contract demand as data on `RenderDiagnostics.RequiredContracts`. The field holds every contract key that a component of the render requires: the keys of each component's `#resources`, plus the keys of its `#traits` when it attaches any. The keys are sorted in byte order with duplicates removed. Source: enhancement 0013 ("The kernel reports every contract a render required, synthesised components included").

#### Scenario: Demand on a successful render

- **WHEN** an instance renders whose components declare resources and traits, and two components declare the same resource key
- **THEN** `RenderResult.Diagnostics.RequiredContracts` lists every resource and trait key once, sorted

#### Scenario: A component without traits

- **WHEN** an instance renders whose component attaches no `#traits`
- **THEN** that component's resource keys are listed and the render does not fail on the absent traits

#### Scenario: A component with traits and no resources

- **WHEN** an instance renders whose component has an empty `#resources` map and attaches `#traits`
- **THEN** that component's trait keys are listed on `RequiredContracts`

### Requirement: The contract demand is computed inside the build from the matcher's component set
The render SHALL report the instance's contract demand as data on `RenderDiagnostics.RequiredContracts`. The demand SHALL be computed inside the build, from the same component set the matcher iterates. A component that is omitted under `RenderInput.SkipUnprovided` still counts. The list SHALL NOT be narrowed to provider-fulfilled contracts or to contracts the platform implements. The list SHALL be an empty, non-nil slice for an instance with no components.

#### Scenario: The demand equals the declared contracts of the instance

- **WHEN** any served render fixture whose instance has no synthesised components renders and the build reaches its diagnostics
- **THEN** `RequiredContracts` equals the sorted, deduplicated `#resources` and `#traits` keys of the instance package's own components

#### Scenario: An omitted component's demand is reported

- **WHEN** `SkipUnprovided` is set and a component is omitted for an unprovided resource demand
- **THEN** that component's resource and trait keys are on `RequiredContracts`, although it renders nothing

#### Scenario: No components

- **WHEN** an instance with no components renders
- **THEN** the render succeeds and `RequiredContracts` is an empty, non-nil slice

### Requirement: The field is set on every RenderResult and every RenderError
The field `RenderDiagnostics.RequiredContracts` SHALL be set on every `RenderResult` and on every `*RenderError`. A render that returns a plain error carries no demand. That covers a refusal before evaluation and a build or diagnostics-decode failure.

#### Scenario: Demand on a refusal

- **WHEN** the gate refuses a render because a resource demand is unresolved
- **THEN** the returned `*RenderError`'s `Diagnostics.RequiredContracts` lists every contract key the instance's components declare, the unresolved one included

### Requirement: Computing the demand fails closed
Computing the instance's contract demand SHALL fail closed. A component's `#resources` is read with no presence test, so a component whose `#resources` is missing or does not evaluate makes the render return an error and no result. It never reads as a component with no demand. An absent `#traits` contributes nothing. `#traits` is read with the same presence test the matcher uses, so the demand covers exactly the traits matching sees.

#### Scenario: A component without resources refuses

- **WHEN** a render's instance package has a component that carries no `#resources`
- **THEN** `Render` returns an error and no result, and does not report the component as one with no demand

### Requirement: Each render creates a fresh context, evaluates once and shares no built value
`Render` SHALL create a fresh `cue.Context` for the render, evaluate the staged render module exactly once with it, and release its own references to it when `Render` returns; a `*kernel.Compiled` the caller holds keeps that build alive until the caller releases it. No built value SHALL be shared between renders, and the render SHALL NOT use any long-lived context.

#### Scenario: Repeated renders share nothing

- **WHEN** `Render` is invoked twice with the same inputs
- **THEN** each invocation stages, builds and decodes independently, and the results are byte-identical

#### Scenario: Concurrent renders share the synthetic root

- **WHEN** several goroutines build render modules staged under the same synthetic root at the same time
- **THEN** every build succeeds with the same result as a sequential build, and the race detector reports nothing

### Requirement: The render module is staged in memory under a synthetic root that does not exist on disk
The render module SHALL be staged in memory under a fixed synthetic root, an absolute path on every operating system (on Windows it carries a volume), that does not exist on disk. Because cue/load reads a real directory beneath an overlay root, staging SHALL refuse, with an error naming the path, when anything exists at that root.

#### Scenario: The synthetic roots are absolute

- **WHEN** the render root and a registry module's synthetic root are computed on any operating system
- **THEN** both are absolute paths, so cue/load accepts the overlay keys beneath them

#### Scenario: An existing render root is refused

- **WHEN** a directory holding a `.cue` file of the render package, or a plain file, exists at the synthetic root and a render module is staged
- **THEN** staging returns an error naming the root, and nothing is staged or built

### Requirement: The render module and the overlay-mode inputs are served from the load overlay
The render module's generated `cue.mod/module.cue`, `cue.mod/local-module.cue` and glue file, and every file of an overlay-mode input, SHALL be served to the build from the load overlay, and an on-disk input SHALL be served from its own directory, so nothing of either input is copied and nothing is written. The coverage invariant SHALL be checked against the `cue.mod/module.cue` bytes the build is served.

#### Scenario: The render module is served from memory

- **WHEN** a render of an overlay-mode instance against an overlay-mode platform is staged
- **THEN** the staged overlay carries `cue.mod/module.cue`, `cue.mod/local-module.cue` and the glue file under the synthetic root, and every input file under that root's `instance/` or `platform/` directory
- **AND** the synthetic root does not exist on disk before or after the build

#### Scenario: Local replacements resolve from an in-memory render module

- **WHEN** a render module whose `cue.mod/local-module.cue` replaces input and catalog paths with on-disk directories is served from memory under the synthetic root
- **THEN** the one build resolves every replaced path from its directory
- **AND** the same build with `cue.mod/local-module.cue` left out of the overlay fails to resolve the replaced input, so the overlay's file is the one the build read

### Requirement: Kernel.Render writes no staging file
`Kernel.Render` SHALL write no staging file: it SHALL create, write or remove nothing under the process temp directory and nothing at the render module's synthetic root, not on a successful render, not on a refusal before staging, not on a refusal before evaluation, and not on a build failure. The CUE module cache is not staging: a render whose build fetches a dependency fills the module cache under `CUE_CACHE_DIR` exactly as any other load does.

#### Scenario: A successful render leaves the temp directory untouched

- **WHEN** a test points `TMPDIR` at an empty directory it owns and renders a served instance against a served platform
- **THEN** the render succeeds, the directory is still empty, and the render module's synthetic root does not exist on disk

#### Scenario: A refusal leaves the temp directory untouched

- **WHEN** a test points `TMPDIR` at an empty directory it owns and `Render` refuses an older-core platform, a local replacement without the opt-in, an instance dependency that carries no version and no replacement, and catalog skew under `SkewRefuse`, and fails to load a render module whose instance imports a module nothing serves
- **THEN** each call returns its refusal, the directory is still empty, and the synthetic root does not exist on disk

### Requirement: A test that asserts that a render writes no staging file owns its temp directory
A test that asserts that `Kernel.Render` writes no staging file SHALL point the process temp directory at a directory owned by that test and SHALL find it empty afterwards, so its result does not depend on other test processes sharing the same `TMPDIR`, and SHALL find that the synthetic root does not exist.

#### Scenario: A render that writes is caught

- **WHEN** a change to `Kernel.Render` writes a file or directory under the process temp directory
- **THEN** the test covering that path fails, naming what it found under its private root
