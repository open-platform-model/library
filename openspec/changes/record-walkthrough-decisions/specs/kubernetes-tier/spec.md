## MODIFIED Requirements

### Requirement: Kind-class order is a Kubernetes fact that lives in the tier

The library SHALL keep one kind-class apply order (a CustomResourceDefinition before its custom resources, a Namespace before namespaced objects). It SHALL live in the Kubernetes tier, and the kernel SHALL hold none. No module-internal ordering (`dependsOn` edges, phases within one module) is planned: the library SHALL add no module-specific ordering of its own, and any module-internal ordering added later SHALL come off the CUE build as data. Lifecycle hooks remain the parked 0009 work, carried as build data under ADR-008 rule 4. In the OPM model, ordering across modules belongs to a future Bundle definition, which uses the order its modules are defined in; the library SHALL add none before then. A frontend's readiness gating between its own custom resources is outside this rule. ADR-008 rule 4's "The kernel derives no ordering of its own" SHALL be read as excluding module-specific ordering only, and ADR-008 and ADR-011 SHALL say so in their own text. Source: 0012:D5.

#### Scenario: ADR-008 carries the clarification

- **WHEN** a developer reads `adr/008-kernel-plans-caller-runs.md`
- **THEN** its Status records the 2026-10-02 amendment pointing at ADR-011, and the rule ending "The kernel derives no ordering of its own" states that it excludes module-specific ordering only

#### Scenario: No module-internal ordering is planned and cross-module order belongs to a Bundle

- **WHEN** a developer reads rule 4 of ADR-008 and item 7 of ADR-011
- **THEN** both state that no module-internal ordering (`dependsOn` edges, phases) is planned and that any added later comes off the build, that lifecycle hooks remain the parked 0009 work, and that in the OPM model ordering across modules belongs to a future Bundle definition using the order its modules are defined in
- **AND** neither presents module-internal `dependsOn` edges or phases as planned work

## ADDED Requirements

### Requirement: Deletion plans come from the inventory and the live objects

A deletion plan SHALL be built from the persisted inventory plus the live objects in the cluster. Building it SHALL NOT need a render and SHALL NOT read a plan stored at apply time. ADR-008 SHALL record this, and SHALL record that planning from a render and lifecycle transition detection (install, upgrade, reconfigure, no-op, uninstall) are deferred to the lifecycle hook work of enhancement 0009. This deferral does not touch the deletion step transition, which the tier owns as part of the deletion sequence. Source: 0012:D5 (as amended 2026-10-03).

#### Scenario: ADR-008 records the deletion-plan source

- **WHEN** a developer reads `adr/008-kernel-plans-caller-runs.md`
- **THEN** it states that deletion plans are built from the persisted inventory plus the live objects, with no render and no stored plan
- **AND** it lists planning from a render and lifecycle transition detection among the questions it does not decide, deferred to the 0009 hook work, distinct from the deletion step transition the tier owns
