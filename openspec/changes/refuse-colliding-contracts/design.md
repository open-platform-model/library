## Context

See proposal.md for why. The facts the approach rests on, all read on library `30f08c1` (release `1.0.0-alpha.35`):

- `opm/kernel/render_decode.go` `gateErrors` refuses only on decoded `Unresolved`, `OverSubscribed` and `Unmatched` rows. It never reads `routable`, and it never reads the build's `gate` field. The glue's `gate` (`match.resolved & platform.#contracts.routable & true`) already reads `routable`, so on A's core a colliding platform's own gate is bottom while the kernel renders it. Measured by the planning mapper on a scratch copy with A's patched core: `Kernel.Render` succeeds on a two-major colliding platform carrying a bridge transformer and returns two Deployments named `probe-maj0-web` (built by `0.1.0` and `bridge-1.4.0`). Enhancement 0026 experiment 07 measured the same bridge shape on the shipped core with disjoint keys.
- `opm/platform/contracts.go` `Contracts()` decodes nine fields by name, each behind a since-guard that refuses an absent field with `PlatformCoreTooOldError`, and its doc says a missing report is never defaulted in either direction. The floor is `schema.ProvidedBySince = "2.0.0-alpha.12"`.
- `opm/errors/match.go` `UnresolvedDemand.describe` falls back to "no enabled catalog defines this contract" when `DefinedBy` is empty; on A's core a colliding key is absent from `definedBy`, so that text would be wrong.
- `opm/internal/registrytest` `TestDefaultCoreVersion_IsTheDefaultSchemaRelease` asserts every `testdata/render/**/cue.mod` pins `registrytest.DefaultCoreVersion`, which must equal `schema.DefaultSchemaVersion()`. Moving the default therefore re-pins every render fixture, and an older-core case is a copied and re-pinned fixture (`acquireOlderCorePlatform` in `opm/kernel/render_core_floor_test.go` is the pattern).
- Core A's surface (orchestration.md, interface): `#ContractInventory.collisions: [...#ContractFQNType]` (ascending), `collidingEntries: [#ContractFQNType]: [...#ModulePathType]` (ascending registry keys), `routable: len(overSubscribed) == 0 && len(collisions) == 0`. A colliding key is in none of `defined`, `definedBy`, `requiredBy`, `unfulfilled` or `comparable`; `providedBy` and `overSubscribed` are unaffected.

## Goals / Non-Goals

**Goals:**

- The kernel refuses a colliding platform from decoded rows, in `Render`, with a typed cause a frontend can word, and `Contracts()` reports the same collision.
- The render refuses exactly when the module's own `gate` does, as a runtime property for `routable`, not only as a test invariant.
- No platform that acquires and renders today changes outcome.

**Non-Goals:**

- Side-by-side majors (0026 D9's per-resolution builds). A collision is refused, never resolved.
- Keying `requiredBy` or `comparable` over colliding keys. Core A deferred it (colliding members can disagree across majors, and `routable` false already withholds generation); the library documents the blind spot and does not paper over it.
- A new core floor (see Decision 2).
- Frontend wording. The operator (change C) and cli (change D) own it.

## Decisions

### 1. The glue reads the report; the kernel decides from decoded rows

`render.cue.tmpl` gains, beside `guard.overSubscribed`:

```cue
guard: {
	overSubscribed: /* unchanged, iterates providedBy unconditionally */
	// Core's collision report, read and never recomputed. Guarded on
	// presence: a core without the report cannot evaluate a colliding
	// platform, so absence is provably no collision.
	collisions: [
		if platform.#contracts.collisions != _|_
		for _k in platform.#contracts.collisions {{
			key:      _k
			catalogs: platform.#contracts.collidingEntries[_k]
		}},
	]
}

diagnostics: {
	// ...existing rows...
	collisions: guard.collisions
	// Present on every accepted core (since 2.0.0-alpha.9); read
	// unconditionally, so it is concrete or the decode refuses.
	routable: platform.#contracts.routable
}
```

`collisions` is already ascending in core, so the rows are emitted in key order without a sort. `#Match` gains `#collidingEntries: [string]: [...string]`, bound to `platform.#contracts.collidingEntries` when present and `{}` otherwise, and each unresolved resource and trait row gains:

```cue
colliding: *[] | [...string]
if #collidingEntries[_f] != _|_ {
	colliding: #collidingEntries[_f]
}
```

The `gate` expression is unchanged: it already reads `routable`.

Go side (`render_decode.go`, `render.go`):

```go
type glueDiagnostics struct {
	// ...existing fields...
	Collisions []oerrors.ContractCollision `json:"collisions"`
	Routable   bool                        `json:"routable"`
}

// RenderDiagnostics gains:
//   Collisions []oerrors.ContractCollision  // key-sorted; any row refuses
//   Routable   bool                         // core's #contracts.routable, as decoded

func gateErrors(diag RenderDiagnostics) error {
	var gate []error
	if len(diag.Collisions) > 0 {
		gate = append(gate, &oerrors.ContractCollisionsError{Contracts: diag.Collisions})
	}
	if len(diag.Unresolved) > 0 { /* unchanged */ }
	if len(diag.OverSubscribed) > 0 { /* unchanged */ }
	if len(diag.Unmatched) > 0 { /* unchanged */ }
	if !diag.Routable && len(diag.OverSubscribed) == 0 && len(diag.Collisions) == 0 {
		gate = append(gate, &oerrors.NotRoutableError{})
	}
	if len(gate) == 0 {
		return nil
	}
	return errors.Join(gate...)
}
```

`opm/errors` gains (new file `collision.go`):

```go
// ContractCollision is one row: a contract key more than one enabled
// registry entry's catalog lists. Data, not an error.
type ContractCollision struct {
	Key      string   `json:"key"`
	Catalogs []string `json:"catalogs"` // sorted registry keys, path@major
}

type ContractCollisionsError struct{ Contracts []ContractCollision } // pointer receiver, wraps nothing

type NotRoutableError struct{} // pointer receiver, wraps nothing
```

`ContractCollisionsError.Error()` follows `OverSubscribedContractsError`: a header `%d colliding contract(s):` then one line per row, `contract %q is defined by %d enabled registry entries (%q, %q); a contract key must have exactly one enabled definer until side-by-side catalog majors are supported, so disable all but one of these entries` (the quoted entries joined by `, `, as many as the row carries). `NotRoutableError.Error()` reads `platform is not routable: its contract inventory reads routable false and reports no over-subscribed or colliding contract to explain it`.

`UnresolvedDemand` gains `Colliding []string` (diagnostic only), and `describe` gains a case after the alternatives and defined-by cases and before the default: `: defined by more than one enabled registry entry (%v)`. The `RenderError` doc lists both new causes.

**Why these choices.** The collision cause is first because a colliding key has left `definedBy`, `requiredBy` and `comparable`, so every other row is read against a distorted inventory, and its fix (disable a major) comes before any other. The refusal is platform-wide and ignores `SkipUnprovided`, like over-subscription, because it is a fact about the platform, not about the instance. The name `Catalogs` mirrors `OverSubscribedContract{Key, Catalogs}`, whose values are also registry keys.

### 2. No new floor: absence is decoded as empty

`Contracts()` adds two rows to its per-field table that are read with an absent-as-empty rule instead of a since-guard; a present field that fails to decode still returns the decode error. `schema.CollisionsSince = "2.0.0-alpha.13"` (A's tag) documents the first release carrying the report and is used by tests; it is not a floor, and `ProvidedBySince` is unchanged.

The exception is sound for every value acquired through the loader, by proof plus measurement:

1. Every core carrying `#contracts` (`alpha.9` to `alpha.12`) folds `definedBy: {for path, entry in #registry if entry.enable {for fqn, _ in ... {(fqn): path}}}`. Registry keys are distinct strings, so two enabled definers of one key always conflict in `definedBy` (and in `defined`, on `metadata.catalogVersion`). Measured: 0026 experiment 01 case A, experiment 06's control and the mapper's four colliding fixtures on `alpha.12` all fail at acquire in `opm/internal/loader/load.go` (`val.Err()` after the build): `#contracts.defined."maj/resources/container@v1".metadata.catalogVersion: conflicting values "1.0.0" and "0.1.0"`.
2. Cores before `alpha.9` are already refused by `PlatformCoreTooOldError`, and the render floor stays `alpha.12`.
3. Measured: the guarded glue row on the unmodified `alpha.12` core yields `diagnostics.collisions: []`, and the full `./opm/kernel` and `./opm/internal/renderstage` suites pass.

This differs from the `providedBy` precedent on purpose. There, absence meant an unknown count, so the glue iterates unconditionally and fails closed. Here, absence means provably none.

The rewritten `Contracts()` doc states the exception and cites the `definedBy` conflict. The `Routable` doc becomes "true exactly when OverSubscribed and Collisions are both empty"; the `DefinedBy`, `RequiredBy`, `Unfulfilled`, `Comparable`, `Fulfilled` and `Discriminated` docs say a colliding key is excluded, so `Fulfilled` and `Discriminated` can read true while `Collisions` is non-empty.

**Residual hole, kept honest.** A hand-built `Platform` (exported `Package` and `Source`, no `AcquirePlatformFromDir`) is outside the proof. Measured in section 3 (`TestRender_HandBuiltOlderCoreCollidingPlatformNeverRenders`): `platform_collide` re-pinned to core `2.0.0-alpha.12` is refused by the loader (`#contracts.defined."…/maj/resources/container@v1".metadata.catalogVersion: conflicting values "1.4.0" and "0.1.0"`). The same package built by hand with `cue/load` and wrapped by `platform.NewPlatformFromValue` constructs (metadata decodes), carries `#contracts.providedBy` and so passes the render's core floor, and carries no collision report. `Kernel.Render` then stages and builds it, and the build fails when the diagnostics are decoded, on the same conflict (`match.#definedBy."…/maj/resources/container@v1": conflicting values "…/maj@v1" and "…/maj@v0"`). The result is a plain error (not a `*RenderError`, no typed collision cause) and no compiled object: the collision is not named as a collision, but it never renders. The catch-all (Decision 3) closes the new-core case where `collidingEntries` is present but a guarded row came out empty.

### 3. The not-routable catch-all

On `alpha.12` and on A's core, `routable` false always carries a `providedBy`-derived over-subscription row or a collision row, so `NotRoutableError` fires on no platform today, and a test asserts that over every served platform. It makes the kernel's verdict agree with the module's `gate` on `routable` by construction: a future core that adds a term to `routable` without a row the kernel decodes is refused instead of rendered. It is decoded from `routable`, which every accepted core carries.

### 4. Paths and pins

- `opm/schema/paths.go`: `ContractsCollisions` and `ContractsCollidingEntries` (`#contracts.collisions`, `#contracts.collidingEntries`), `CollisionsSince`. The `Contracts` path comment says eleven data fields.
- `schema.DefaultSchemaModule` = `opmodel.dev/core@v2.0.0-alpha.13` (A's tag), with its doc comment rewritten (the default is past the floor), `registrytest.DefaultCoreVersion` with it, every `testdata/render/**/cue.mod`, and `task cue:deps:update` over the discovered modules (`modules/opm_platform`, `testdata/modules/web_app`, `testdata/parity`, `testdata/parity/opm_platform`). `acquireOlderCorePlatform` asserts the new literal before rewriting it to `alpha.10`.

### 5. Fixtures and tests

Served fixtures, ported from 0026 experiment 07's overlay (same shapes, library naming, pinned to A's tag):

- `testdata/render/registry/testing.opmodel.dev_library-render_maj_v0.1.0`: catalog `maj@v0` listing `resources/container@v1`, `traits/expose@v1` and the provider-fulfilled `traits/backup@v1`, with deployment and service transformers.
- `..._maj_v1.4.0`: catalog `maj@v1` at `1.4.0`, experiment 07's `1.3.0` plus the three `maj@v0` keys in its own contract maps (so they collide) and `container@v2`; transformers deployment, service, deployment-v2 and the deployment bridge requiring `maj@v0`'s container.
- `..._bprov_v0.1.0` (built on `maj@v0`) and `..._bprov_v1.0.0` (built on `maj@v1`, dep pinned to `1.4.0`): each one backup transformer requiring its major's backup trait, no contracts of their own.
- `..._app_maj0_v0.1.0` (a `maj@v0` container component with expose) and `..._app_bk0_v0.1.0` (a `maj@v0` component with the load-bearing backup trait); instances `testdata/render/instance_maj0` and `testdata/render/instance_bk0`.
- Platforms: `platform_collide` (`maj@v0` plus `maj@v1`), `platform_collide_provider` (plus `bprov@v0`), `platform_collide_oversubscribed` (plus `bprov@v0` and `bprov@v1`). Each header states its role, as the existing platform fixtures do.

Tests:

- `opm/platform/contracts_test.go`: the three platforms' `Collisions`, `CollidingEntries`, `DefinedBy`, `RequiredBy`, `ProvidedBy`, `OverSubscribed`, `Fulfilled`, `Discriminated`, `Routable`; empty collisions on every existing case; the `alpha.12` copy decoding empty; a `CompileString` value whose `collisions` is present and ill-typed returning a decode error. The limitation assertions carry a comment saying they pin a known blind spot, so a future core fix updates them deliberately.
- `opm/kernel/render_inventory_parity_test.go`: the table classifies each platform with its instance (`instance` by default, `instance_maj0` for the collide platforms) and its expected over-subscription and collision rows; it asserts collision rows equal `CollidingEntries`, `diag.Routable == inv.Routable`, `inv.Routable == (no over-subscription row && no collision row)`, and that no refusal carries `NotRoutableError`.
- A new `opm/kernel/render_collision_test.go`: the bridge refusal (typed cause first, three rows, nil result, `assertGateAgrees(t, built, true)`), the same under `SkipUnprovided`, the joined order on `platform_collide_oversubscribed`, the colliding unresolved row on `instance_bk0` over `platform_collide`, and the `alpha.12` copy rendering unchanged with no collision row and `Routable` true.
- `opm/kernel` internal test of `gateErrors` for the catch-all (fires only on `Routable` false with no over-subscription or collision row; joined last).
- `opm/errors`: message tests for `ContractCollisionsError`, `NotRoutableError` and the colliding `UnresolvedDemand` case.

Red first, never committed red: the bridge refusal test is written and run on the unchanged kernel with A's core, and must fail by returning success with two `probe-maj0-web` Deployments; the output is recorded for the worker report.

### 6. Section plan

1. `fix(deps)`: pin A's tag everywhere, glue unchanged; the whole suite green proves a one-major platform reads the same on A's core.
2. `feat(platform)`: paths, `Contracts()` decode and docs, the collision fixtures, contracts tests, the parity table extended with the new platforms and the routable assertion widened to collisions.
3. `feat(kernel)`: error types, glue rows, decode, `gateErrors`, render tests (red first), the hand-built measurement, consumer builds.
4. `docs(kernel)`: diagnostics entry, AGENTS and package doc.

## Research & Decisions

### Floor or no floor

**Context**: `Contracts()` never defaults a missing report; the question is whether a missing `collisions` is refused (a `CollisionsSince` since-guard) or read as empty.
**Explored**: the fold in every cached core carrying `#contracts` (alpha.9 to alpha.12); experiment 01 case A, experiment 06's control and four colliding fixtures on `alpha.12`; a guarded glue row on `alpha.12` with the kernel and renderstage suites.
**Decision**: absent reads as empty, documented as the one exception (Decision 2).
**Rationale**: absence provably means none, and a since-guard would refuse every `alpha.12` platform at the operator's held-package and generation gate, at TransformerRegistration acceptance and in `opm platform check` until re-pinned, a real cost bought for nothing.

### Keep the catch-all

**Context**: whether to refuse `routable` false with no explaining row.
**Explored**: every served platform's `routable` against its rows on `alpha.12`; the module `gate` expression.
**Decision**: keep `NotRoutableError` (planning question for the user; recommended).
**Rationale**: one branch turns gate/kernel parity into a runtime property; dropping it keeps the surface one type smaller but lets a future `routable` term render silently, which is exactly the class of bug this change fixes.

### Fixture source

**Context**: the mapper's colliding fixtures lived in a scratch copy.
**Explored**: 0026 experiment 07's overlay (`maj` 0.1.0, 1.0.0, 1.1.0, 1.3.0; `bprov` 0.1.0 and 1.0.0; `app_maj0`, `app_bk0`).
**Decision**: port experiment 07's shapes into `testdata/render`, with `maj` 1.4.0 as the colliding bridge major.
**Rationale**: measured shapes, already served by `registrytest`, and they reproduce the duplicate-Deployment hazard exactly.

## Risks / Trade-offs

- [A's tag differs from `2.0.0-alpha.13`] -> every literal in this change (constants, fixtures, spec scenarios, `CollisionsSince`) uses A's reported tag, and the worker reports the difference under `deviations`.
- [An old kernel renders a colliding platform pinned to A's core] -> not fixable here. Ordering is the mitigation: ship right after A, move `DefaultSchemaModule` only in this change, and the workspace `task deps:update` waits for this release.
- [The ported fixtures' render of the parity `instance` on the collide platforms fails outside `RenderError`] -> the parity table renders `instance_maj0` on them; if that also fails to decode, the collide platforms are classified with a non-`RenderError` expectation and the finding is written here.
- [`task cue:deps:update` moves `opmodel.dev/catalogs/opm` as well as core] -> move the parity harness literals with it and run `OPM_FLOW_TEST_FORCE=1 go test ./opm/kernel -run 'TestParity|TestFlow'`, as the previous re-pin did.
- [Limitation: `Fulfilled` and `Discriminated` read true on a colliding platform] -> documented on every affected field and pinned by tests; consumers (C, D) are told to treat a non-empty `Collisions` as overriding both.
- [A hand-built platform over an old-core colliding value] -> measured in section 3; the finding is written into Decision 2 and reported.
