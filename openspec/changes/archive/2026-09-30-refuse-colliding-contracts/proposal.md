## Why

Core change `fold-colliding-contract-keys` (change A of this set, core `2.0.0-alpha.13` expected) makes a platform that enables two majors of one catalog sharing contract keys evaluate instead of failing: `#Platform.#contracts` folds only the keys with exactly one enabled definer and reports the rest as `collisions` and `collidingEntries`, with `routable` false. Before it, such a platform failed at acquire (`#contracts.definedBy` conflicts), so the kernel never saw one. After it, the kernel does, and the kernel decides a render only from Go-decoded rows (`gateErrors` in `opm/kernel/render_decode.go`), never from the in-build `gate`. Measured on library `30f08c1` (release `1.0.0-alpha.35`) with the patched core: the in-build gate is bottom, yet `Kernel.Render` succeeds on a colliding platform carrying a bridge transformer and returns two Deployments named `probe-maj0-web` (built by `0.1.0` and by the bridge). That breaks the rule that the module's own gate agrees with the kernel's refusal, and it renders duplicate objects.

This change gives a collision a typed, Go-decoded refusal in both `Platform.Contracts()` and `Kernel.Render`, so the kernel refuses exactly what the inventory reads as not routable. It is the interim safety net recommended by enhancement 0026 OQ17, not side-by-side major support (0026 D9 later makes two majors legitimate through per-resolution builds). No 0026 decision is fully delivered, so the change carries no `enhancement.yaml`.

## What Changes

- **Core pin to A's tag.** `schema.DefaultSchemaModule` and `registrytest.DefaultCoreVersion` move to `opmodel.dev/core@v2.0.0-alpha.13` (A's reported tag wins), every `testdata/render/**` fixture re-pins, and `task cue:deps:update` moves the discovered modules. Generated platforms (operator, cli platform-less path) then pin the fold core and name a collision instead of failing at acquire. The render floor `schema.ProvidedBySince` stays `2.0.0-alpha.12`: no platform that works today is refused.
- **`ContractInventory.Collisions` and `CollidingEntries`.** `Contracts()` decodes `#contracts.collisions` (ascending contract keys with more than one enabled definer) and `#contracts.collidingEntries` (each such key to the sorted registry keys, path plus major, of the entries listing it). An ABSENT field decodes as empty, a documented exception to the rule that a missing report is never defaulted: every core carrying `#contracts` before A fails to evaluate a colliding platform (the `definedBy` conflict), so absence provably means none. A present field that fails to decode still refuses. `Routable` is documented as true exactly when `OverSubscribed` and `Collisions` are both empty; the `DefinedBy`, `RequiredBy`, `Unfulfilled`, `Comparable`, `Fulfilled` and `Discriminated` docs state that a colliding key is excluded from them, so `Fulfilled` and `Discriminated` can read true while `Collisions` is non-empty.
- **A typed collision refusal in the render.** New `errors.ContractCollision{Key, Catalogs}` rows and the gate cause `*errors.ContractCollisionsError`; the glue emits `diagnostics.collisions` (guarded on presence, in key order) and `diagnostics.routable`; `RenderDiagnostics` gains `Collisions` and `Routable`. `gateErrors` raises the collision cause first (the other rows are distorted under a collision), then unresolved, over-subscribed and unmatched as today. The refusal is platform-wide and stands under `SkipUnprovided`.
- **A not-routable catch-all.** New `*errors.NotRoutableError`, raised only when the decoded `routable` is false and no over-subscription or collision row explains it. It fires on no platform today; it turns the gate/kernel parity (a test invariant until now) into a runtime property, so a future core term in `routable` cannot silently render.
- **Unresolved rows name a collision.** `UnresolvedDemand` gains diagnostic-only `Colliding []string`; its message says "defined by more than one enabled registry entry" instead of the misleading "no enabled catalog defines this contract" (a colliding key is absent from `definedBy`).
- **`schema.ContractsCollisions`, `ContractsCollidingEntries` and `CollisionsSince`** (documentation and tests only; not a floor).
- **Docs**: a `docs/site/diagnostics/colliding-contracts.md` entry, the AGENTS render-pipeline block and the `opm/kernel` package doc.

## Classification

**MINOR (pre-GA, released as `feat(kernel)`), not breaking.** Every addition is an exported field, type or constant. The only behaviour change is on platforms that could not be acquired before (a colliding platform failed at acquire on every core up to `2.0.0-alpha.12`), and no floor moves, so a platform pinning `2.0.0-alpha.12` decodes and renders exactly as before, with `Collisions` empty. Section commits: `fix(deps)` (the shipped `DefaultSchemaModule`), `feat(platform)`, `feat(kernel)`, `docs(kernel)`. No migration fragment (pre-GA, no floor). Complexity (Principle VII): two error types and one row type, justified by the kernel deciding only from decoded rows; the catch-all is one branch.

## Downstream consumers

- **`opm-operator`** (change C) names the collision in its platform readiness (`ContractCollisions` reason) from `Collisions` and `CollidingEntries`, and fails closed on `Routable` false with no rows.
- **`cli`** (change D) prints a colliding-contracts section in `opm platform check`, counts collisions in the routable verdict and the exit message, and prints the collision row first in render validation.
- **Rollout hazard no library change can close:** an OLD kernel (this repo `1.0.0-alpha.35` and earlier) given a colliding platform pinned to A's core renders it, duplicates included. Mitigations are ordering: this change ships right after A; generated platforms move to A's core only with this change; the workspace `task deps:update` waits for this release.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `platform-artifact`: `Contracts()` decodes `Collisions` and `CollidingEntries` (absent reads as empty), `Routable` covers collisions, and the reports blind to a colliding key say so.
- `single-build-render`: a render refuses a platform whose enabled entries share contract keys with a typed collision cause ahead of every other cause, refuses a not-routable platform no row explains, names a collision on unresolved rows, and the module's gate, the parity test and the inventory agree on collisions.
- `schema-dispatch`: `DefaultSchemaModule` moves to A's tag, and the default no longer equals the render floor.

## Impact

- `opm/errors` (new collision rows and causes, `UnresolvedDemand.Colliding`), `opm/schema/paths.go`, `opm/schema/loader.go`, `opm/platform/contracts.go`.
- `opm/internal/renderstage/render.cue.tmpl`, `opm/kernel/render_decode.go`, `render.go`, `doc.go`; tests in `opm/kernel`, `opm/platform`, `opm/errors`, `opm/internal/registrytest`.
- New served fixtures under `testdata/render/registry` (catalog `maj` at `0.1.0` and `1.4.0`, provider `bprov` at `0.1.0` and `1.0.0`, module `app_maj0`), `testdata/render/instance_maj0` and three `testdata/render/platform_collide*` platforms; every `testdata/render/**` and discovered module `cue.mod` re-pinned.
- `AGENTS.md`, `docs/site/diagnostics/colliding-contracts.md`.
- Waits on core change A's release resolving from GHCR.
