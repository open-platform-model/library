# platform-synthesis — Delta

## REMOVED Requirements

### Requirement: Subscriptions and filters map onto the registry

**Reason**: 0010 D14 — the filter vocabulary does not exist in core v2, so the synthesis surface can no longer express one. Replaced by the Subscription Synthesis requirement below (required `version` scalar; Enable semantics and the invalid-path scenario carry over unchanged).

## ADDED Requirements

### Requirement: Subscription Synthesis

A synthesized platform's registry subscription SHALL carry a required scalar `version` naming the single build the subscription materializes, matching core v2's `#Subscription` shape. The synthesis input (`SubscriptionSpec`) SHALL require a non-empty `Version` and SHALL NOT be able to express a filter — the filter vocabulary (`range`, `allow`, `deny`) does not exist in core v2 and the synthesis surface no longer carries it. `Platform` SHALL refuse a subscription with an empty `Version` at synthesis time with an error naming the subscription path.

#### Scenario: Version emitted

- **WHEN** a platform is synthesized with a subscription `{Version: "2.0.0-alpha.3"}` for `opmodel.dev/catalogs/opm@v2`
- **THEN** the synthesized CUE carries `version: "2.0.0-alpha.3"` under that registry key
- **AND** the synthesized platform materializes exactly that build

#### Scenario: Empty version refused at synthesis

- **WHEN** a subscription is synthesized with an empty `Version`
- **THEN** `Platform` returns an error naming the subscription path
- **AND** no platform value is produced

#### Scenario: Enable omitted defers to schema default

- **WHEN** a subscription is supplied with `Enable` left nil
- **THEN** the returned value's `enable` for that path resolves to the schema default `true`

#### Scenario: Enable explicitly false

- **WHEN** a subscription is supplied with `Enable` pointing to `false`
- **THEN** the returned value's `enable` for that path is `false`

#### Scenario: Invalid catalog path

- **WHEN** a subscription key does not satisfy `#ModulePathType`
- **THEN** `Platform` returns a non-nil error describing the unification failure
