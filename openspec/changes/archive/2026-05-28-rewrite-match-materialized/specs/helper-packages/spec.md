## REMOVED Requirements

### Requirement: Compose Function

**Reason**: The Module-valued `#registry` composition `Compose` performs is obsolete under enhancement 0001's subscription model. `#Platform.#registry` is now path-keyed `[#ModulePathType]: #Subscription`, realized by `Materialize` against an OCI registry — not by FillPath-injecting `*module.Module` values into a shell. The function cannot produce a value that unifies against the subscription-shaped schema.

**Migration**: Build a `*platform.Platform` whose `#registry` carries the desired `#Subscription` entries directly, then call `k.Materialize(ctx, plat)` to obtain a `*MaterializedPlatform`.

### Requirement: Multi-Fulfiller Error Surface

**Reason**: `*MultiFulfillerError` was `Compose`'s structured error for same-FQN collisions across Module-valued registrations. With `Compose` removed, same-FQN handling moves: byte-identical bodies collapse during `Materialize`, divergent bodies surface as a `MaterializeError` (at materialize time) or a `UnifyError` (at match time, via the always-unify rung).

**Migration**: Handle `MaterializeError` returned by `Materialize` and `UnifyError` accumulated on the `MatchPlan` instead of `*MultiFulfillerError`.

### Requirement: Kernel Convenience Method

**Reason**: `(k *Kernel) ComposePlatform(...)` delegated to the removed `Compose`. It is superseded by `(k *Kernel) Materialize(ctx, *platform.Platform) (*MaterializedPlatform, error)`.

**Migration**: Replace `k.ComposePlatform(shell, modules)` with constructing a subscription-shaped `*platform.Platform` and calling `k.Materialize(ctx, plat)`.
