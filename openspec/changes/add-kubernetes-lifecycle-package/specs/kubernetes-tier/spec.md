## ADDED Requirements

### Requirement: The deletion plan orders inventory entries by kind-class delete order

The library SHALL provide `opm/k8s/lifecycle` with a deletion plan built from a list of inventory entries, a deletion policy (`Prune`, `ForceOrphan`) and the deleting instance's UUID. The plan SHALL order the entries by the tier's weight table in descending order, keeping the relative order of entries of equal weight, and SHALL NOT change the caller's list. Every CustomResourceDefinition and Namespace entry, as the tier's safety exclusion matches them, SHALL be marked safety-excluded in the plan before any live object is read. Building the plan SHALL NOT need a render or a stored plan: an uninstall builds it from the persisted inventory, and a prune builds it from the stale set the inventory package computes, through the same constructor. Source: 0012:D4, 0012:D1:R3/R6, ADR-008 (Deletion plans).

#### Scenario: Entries are ordered for deletion

- **WHEN** a plan is built from a ConfigMap, a Deployment, a Namespace and a ValidatingWebhookConfiguration, in that order
- **THEN** its steps are the ValidatingWebhookConfiguration, the Deployment, the ConfigMap and the Namespace, in that order
- **AND** the caller's list is unchanged

#### Scenario: Equal weights keep their order

- **WHEN** a plan is built from two Deployments `b` then `a`
- **THEN** its steps name `b` before `a`

#### Scenario: Protected kinds are marked up front

- **WHEN** a plan is built from an inventory holding a CustomResourceDefinition of `apiextensions.k8s.io` and a core Namespace
- **THEN** both steps are marked safety-excluded in the plan as built

#### Scenario: A prune hands its stale set to the same plan

- **WHEN** the stale set of a previous and a current inventory is passed to the plan constructor
- **THEN** the plan's steps are exactly the stale entries, in kind-class delete order

### Requirement: The deletion transition names one action per call

The deletion transition SHALL take the plan, the caller's state and the outcome of the action it last named, and SHALL return the next state and exactly one action: read one step's live object, delete one step's object, report one step as skipped, or done. It SHALL perform no action itself. A safety-excluded step SHALL be reported as skipped with the reason `safety-excluded` without a read. Every other step SHALL be read before it is judged, and the library's delete verdict SHALL judge it against the plan's instance UUID. A skip verdict SHALL be reported as skipped with the verdict's reason and message. A proceed verdict SHALL name a delete with Foreground propagation and the verdict's UID precondition. When the policy does not prune, the first call SHALL name done and no step SHALL be read or deleted. Every read, delete and skip action SHALL name a step of the plan, and the transition SHALL name no other kind of action. Source: 0012:D4:R1/R6, 0012:D1:R2/R4, ADR-008 rules 1 to 3.

#### Scenario: A protected kind is skipped without a read

- **WHEN** the next step is a Namespace
- **THEN** the action is a skip with reason `safety-excluded`
- **AND** no read is named for that step

#### Scenario: An owned object is read, then deleted with Foreground propagation

- **WHEN** a Deployment step is read and the live object is OPM-managed, carries the plan's instance UUID and has UID `u1`
- **THEN** the next action deletes that Deployment with Foreground propagation and a precondition on UID `u1`

#### Scenario: A foreign object is skipped with the verdict's reason

- **WHEN** a step's live object is not OPM-managed
- **THEN** the next action is a skip with reason `not-opm-managed` and the delete verdict's message

#### Scenario: Another instance's object is skipped

- **WHEN** a step's live object carries a different, non-empty instance UUID than the plan's non-empty owner UUID
- **THEN** the next action is a skip with reason `owner-mismatch`

#### Scenario: A plan that does not prune names done at once

- **WHEN** the policy's `Prune` is false and the transition is called with the zero state
- **THEN** the action is done
- **AND** no read or delete is named for any step

#### Scenario: The transition names only reads, deletions, skips and done

- **WHEN** a plan over a mixed inventory is advanced to done through every outcome kind
- **THEN** every action is a read, a delete, a skip or done
- **AND** every read, delete and skip names a step of the plan

### Requirement: Action outcomes are classified by the library

The caller SHALL hand back the raw error of each read or delete, and the library SHALL classify it with `k8s.io/apimachinery`'s API error helpers, which see through wrapping. A NotFound read, or a read that returns no object, SHALL count as already absent, and SHALL be skipped as `already-absent`. A NotFound delete SHALL be skipped as `already-absent`. A successful delete SHALL record the step as deleted. Any other error SHALL record the step as failed, with the failure class `forbidden`, `conflict` or `error` and the error's text. A failure SHALL NOT stop the plan: the next step's action SHALL follow. Source: 0012:D1:R1, 0012:D4:R1.

#### Scenario: A gone object is already absent

- **WHEN** a read returns a NotFound error
- **THEN** the next action is a skip with reason `already-absent`

#### Scenario: An object gone before its delete is already absent

- **WHEN** a delete returns a NotFound error
- **THEN** the next action is a skip with reason `already-absent`
- **AND** the step is not recorded as deleted

#### Scenario: A wrapped Forbidden error is classified

- **WHEN** a delete returns a Forbidden API error wrapped with `fmt.Errorf("deleting: %w", err)`
- **THEN** the step's outcome is failed with failure class `forbidden`

#### Scenario: A failed precondition is a conflict

- **WHEN** a delete returns a Conflict API error
- **THEN** the step's outcome is failed with failure class `conflict`
- **AND** it is not recorded as deleted

#### Scenario: A failure does not stop the plan

- **WHEN** the read of the first of two steps fails with an Internal error
- **THEN** the next action reads the second step

### Requirement: The deletion state is a serialisable value the caller owns

The deletion state SHALL be a plain value with a defined JSON encoding, holding the next step, what the state awaits and one outcome per finished step. The zero state SHALL be the start of a plan. The library SHALL keep nothing between calls. A state written to JSON and read back SHALL advance identically to the state held in memory, and advancing the same plan, state and outcome twice SHALL name the same action and return equal states. The transition SHALL return an error, the input state unchanged and no action when the state cannot belong to the plan or the live object handed back names a different object than the step being read. Source: 0012:D4:R5, ADR-008 rule 2.

#### Scenario: A round-tripped state advances identically

- **WHEN** a plan is advanced to done once with an in-memory state and once with a state written to JSON and read back before every call, through the same outcomes
- **THEN** both runs name the same actions in the same order
- **AND** their final states are equal

#### Scenario: Advancing twice names the same action

- **WHEN** the same plan, state and outcome are advanced twice
- **THEN** both calls return equal states and equal actions

#### Scenario: A live object for another step is refused

- **WHEN** the state awaits the read of `Deployment/app/web` and the live object handed back is `Deployment/app/api`
- **THEN** the transition returns an error and the input state unchanged

#### Scenario: A state past the plan is refused

- **WHEN** the state's next step is beyond the plan's last step
- **THEN** the transition returns an error

### Requirement: The hold verdict is decided from the policy and the plan's outcome

The library SHALL decide whether an instance's deletion hold may be released from the plan's policy, its steps, the state's outcomes and whether the caller can act as the deleting identity, and from nothing that names the hold's bearer or the frontend. It SHALL check in this order and stop at the first match. When the policy does not prune, it SHALL release as `prune-disabled`. When the plan has no steps, it SHALL release as `inventory-empty`. When the deleting identity is missing and the policy sets force-orphan, it SHALL release as `force-orphan`. When the identity is missing or could not be obtained, it SHALL hold as `identity-unavailable`. When the plan is not finished, it SHALL hold as `cleanup-incomplete`. When any step failed as Forbidden, it SHALL hold as `cleanup-forbidden`. When any other step failed, it SHALL hold as `cleanup-incomplete`. Otherwise it SHALL release as `cleanup-complete`, with skipped steps counting as complete. Each verdict SHALL carry the reason as the contract's literal and a message the library words. Source: 0012:D1:R5, 0012:D4:R1, the 0012 contract's `#HoldVerdict`.

#### Scenario: Pruning disabled releases

- **WHEN** the policy does not prune
- **THEN** the verdict is release with reason `prune-disabled`, whatever the identity and the state

#### Scenario: An empty inventory releases

- **WHEN** the policy prunes and the plan has no steps
- **THEN** the verdict is release with reason `inventory-empty`

#### Scenario: Force-orphan releases only for a missing identity

- **WHEN** the policy prunes and sets force-orphan, the plan has steps and the identity is missing
- **THEN** the verdict is release with reason `force-orphan`
- **AND** with the identity failed instead of missing, the verdict is hold with reason `identity-unavailable`

#### Scenario: A missing identity without force-orphan holds

- **WHEN** the policy prunes without force-orphan, the plan has steps and the identity is missing
- **THEN** the verdict is hold with reason `identity-unavailable`

#### Scenario: An unfinished plan holds

- **WHEN** the identity is available and the state has not reached done
- **THEN** the verdict is hold with reason `cleanup-incomplete`

#### Scenario: A Forbidden failure wins over other failures

- **WHEN** the plan is finished and one step failed as `forbidden` and another as `error`
- **THEN** the verdict is hold with reason `cleanup-forbidden`

#### Scenario: Force-orphan does not lift a failed cleanup

- **WHEN** the policy sets force-orphan, the identity is available, and the finished plan has a failed step
- **THEN** the verdict is a hold

#### Scenario: A finished plan with skips releases

- **WHEN** the plan is finished, every step is deleted or skipped, and the identity is available
- **THEN** the verdict is release with reason `cleanup-complete`

#### Scenario: Reasons are the contract's literals

- **WHEN** a test reads the hold reason constants
- **THEN** they are `prune-disabled`, `inventory-empty`, `cleanup-complete`, `force-orphan`, `cleanup-incomplete`, `cleanup-forbidden` and `identity-unavailable`

### Requirement: The lifecycle package is pure and carries no hook semantics

`opm/k8s/lifecycle` SHALL perform no cluster read or write, read no clock or environment, start no goroutine and log nothing. It SHALL NOT change the entries, the live objects or the errors it is given. It SHALL name no step a module declares around a deletion: its plan holds only the inventory entries it was given. Its package documentation SHALL carry no ADR or enhancement reference, since it publishes into the Library reference. Source: 0012:D4:R6, ADR-008 rule 3, ADR-011.

#### Scenario: The package imports no clock, environment, logger or sync

- **WHEN** the direct imports of `opm/k8s/lifecycle` are listed
- **THEN** none of them is `os`, `time`, `log`, `log/slog` or `sync`

#### Scenario: The package starts no goroutine

- **WHEN** the package's non-test Go files are parsed
- **THEN** none contains a `go` statement

#### Scenario: The live object is not modified

- **WHEN** a live object is handed to the transition
- **THEN** it is deeply equal to a copy taken before the call
