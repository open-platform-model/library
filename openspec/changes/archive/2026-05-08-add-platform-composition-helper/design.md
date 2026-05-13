## Context

Catalog 014 says a Platform's `#registry` is filled by either:

1. **Static path**: an admin-authored `platform.cue` file lists each `#ModuleRegistration` directly.
2. **Runtime path**: `opm-operator` reconciles `ModuleRelease` CRs and FillPaths the corresponding Module into `#registry[<id>].#module`.

Both paths produce the same end state: a Platform whose `#registry` is populated, whose computed views (`#composedTransformers`, `#matchers.{resources,traits}`, `#knownResources`, `#knownTraits`) are resolved.

`opm/helper/platform/Compose` implements the runtime path generically. It accepts:

- `shell *Platform` — a base Platform value with metadata, type, ctx, and possibly a partial `#registry`.
- `modules []*Module` — Modules to register.

And returns a new `*Platform` with `#registry` filled, computed views resolved, and the `APIVersion`/`Metadata`/`Package` fields stamped consistently. If unification fails (e.g. two Modules' transformers claim the same primitive FQN, violating catalog D13), the helper surfaces a clean Go error.

This slice depends on slice 08 (Platform type, binding paths) and slice 09 (Platform required by matcher). It does NOT depend on the static path being implemented separately — both paths converge to the same result.

## Goals / Non-Goals

**Goals:**
- One-line Platform composition for every frontend.
- Surfaces multi-fulfiller violations as Go errors with the offending FQN(s).
- Idempotent: composing the same `(shell, modules)` twice produces byte-equal Platforms.
- Returns a fresh `*Platform`; does not mutate inputs.

**Non-Goals:**
- Loading the shell from disk. Frontends do that with `LoadPlatformFile` (slice 08); the result is fed into `Compose`.
- Implementing per-Module registration metadata (categories, tags, presentation) — those are catalog 014 concepts captured in `#ModuleRegistration.presentation`. The helper sets only `#module` and `enabled: true`; presentation may be added in a follow-up slice if a real consumer asks.
- Conflict resolution policy. Multi-fulfiller is forbidden (catalog D13); the helper surfaces the failure verbatim.
- Reconciliation logic for the operator's runtime path (incremental Module additions on watch events). The operator wraps `Compose` in its reconcile loop.

## Decisions

**`Compose` returns a new Platform.** Reason: idempotency and reasoning ease. Inputs are not mutated; CUE's value model already favors this idiom.

**Module ID scheme: `module.Metadata.Name`.** Per catalog 014 D16: Id keys are kebab-case (`#NameType`), and convention is to set Id to `#module.metadata.name`. Mirror that exactly.

**FillPath one Module at a time.** Reason: errors from a specific Module's registration are easier to diagnose. Alternative (build all registrations as a single CUE value, FillPath the whole thing) loses per-Module error attribution.

**`enabled: true` set explicitly.** Even though `enabled` defaults to `true` per catalog 014, explicit setting makes the intent clear and survives any future schema default change.

**Binding-driven path access.** `binding.Paths().Registry` (from slice 08) provides the path. Helper does not hardcode `#registry`.

**`Compose` calls `*Kernel` for cue.Context.** The shell `Platform` and the Modules are already context-bound; the kernel's context is the one all subsequent operations share. Helper takes `*Kernel`.

**Multi-fulfiller error type.** Define `MultiFulfillerError` in the helper carrying the offending FQN(s), the conflicting Module names, and the conflicting transformer FQNs. Reason: frontends format this for users (CLI prose, operator status conditions, XR composition status). A bare CUE evaluation error is less actionable.

## Risks / Trade-offs

**Risk — large CUE values for big platforms.** A platform with hundreds of Modules has a sizeable `#registry`. Mitigation: not addressed in this slice; YAGNI. If real-world platforms hit this, optimize then.

**Risk — `MultiFulfillerError` shape may need to evolve.** Adding fields later is fine (additive); changing field types is breaking. Mitigation: keep the initial fields minimal and well-considered (FQN, Module names, transformer FQNs).

**Risk — `Compose` does not validate Module registration metadata.** A misconstructed Module could be registered without complaint until matcher time. Mitigation: assumed Modules have already been validated (e.g. by their loader). Helper is a composition tool, not a validation tool.

**Trade-off — ID scheme is fixed (Module name).** Frontends that want a different scheme (per-environment overrides, tenant-prefixed IDs) are not served. Defer to YAGNI; if a real consumer needs it, add an `IDFunc` option.
