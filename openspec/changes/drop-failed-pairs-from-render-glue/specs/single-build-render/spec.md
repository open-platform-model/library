## MODIFIED Requirements

### Requirement: Matching runs inside the build with verdicts as data

The generated glue SHALL express matching over the platform's derived `#composedTransformers` with the verdicts as data fields the kernel decodes into the structured diagnostic types without deriving, joining, grouping or re-sorting them: the matched pair set; every unresolved demand with the same-base alternatives the platform implements (sorted in the build on the `alpha < beta < GA`, then major, then minor apiVersion ladder), its unprovided marker, and each disqualified candidate with the FQNs it conflicted at; every demand skipped under the caller's switch (see "A caller may skip unprovided provider-fulfilled demands"); every candidate the always-unify rung refused, with its conflicting FQNs; every unmatched component with every candidate the demand walk reached for it and, for a predicate refusal, the required labels the component lacked or carried with a different value; the unhandled-trait table; the over-subscription rows. The always-unify and predicate rungs SHALL be evaluated only for a component's candidates (the transformers the demand walk reached for it), never for every transformer on the platform. Matching semantics are those of the render-parity oracle: with the skip switch off, the pair set the glue reports SHALL equal the pair set plain predicate matching over the same inputs produces (`render-parity`, "Matched pair sets agree"); with it on, the reported pair set SHALL be that set minus every pair of an omitted component. The always-unify rung SHALL be plain unification with no provenance exclusion. The fail-closed demand gate SHALL hold: an unresolved demand that was not skipped, or an unmatched component that was not omitted, refuses the render while the diagnostics remain readable and decoded; an effectively-optional unhandled trait is reported on the diagnostics' unhandled-trait table; an unhandled trait with an UNSTATED optional posture refuses as a build error naming the trait's own `optional` field.

Failed pairs are not a glue verdict. The glue SHALL unify each matched pair's transform once, in `rendered`, and SHALL NOT evaluate it a second time for diagnostics. The kernel SHALL name the failed pairs on `RenderDiagnostics.FailedPairs`, in pair order, from each matched pair's rendered output: a pair is failed when its output is an error, wherever inside the output the error arises. An incomplete (non-error) output SHALL NOT be listed; the per-pair concreteness check refuses it. The kernel SHALL fill `FailedPairs` on every `RenderError` raised after the build, a gate refusal included, and SHALL leave it empty on a successful render.

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

#### Scenario: Failed pairs are reported on a gate refusal

- **WHEN** one component's demand is unresolved, so the gate refuses the render, while another component's matched pair has a transformer whose output conflicts
- **THEN** `Render` fails with a `RenderError` whose cause is the gate's, and its diagnostics' `FailedPairs` names the conflicting pair and no other
