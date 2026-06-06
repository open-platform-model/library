## Context

`opm/helper/synth` builds OPM artifact values from typed in-memory inputs. Today it covers exactly one artifact: `synth.Release` (+ the `Kernel.SynthesizeRelease` wrapper that chains into `ProcessModuleRelease`). The `#Platform` artifact has only the file path: `LoadPlatformPackage` → `NewPlatformFromValue` → (explicit) `Materialize`.

The `#Platform` schema (`core/src/platform.cue`) is small and author-facing: `metadata.{name!, description?, labels?, annotations?}`, a required `type!` string discriminator, and a path-keyed `#registry: [#ModulePathType]: #Subscription` where `#Subscription = {enable: bool | *true, filter?: {range?, allow?, deny?}}`. The materialization slots `#composedTransformers` / `#matchers` are filled by the kernel's `Materialize` step, not by schema unification.

Constraints from `library/CLAUDE.md` / `CONSTITUTION.md`: kernel neutrality (no hidden I/O, no global state, deterministic), `opm/helper/` is opt-in convenience, `Materialize` is explicit and caller-driven (D14 — the kernel holds no materialize cache).

## Goals / Non-Goals

**Goals:**

- A `synth.Platform` helper that mirrors `synth.Release` in shape, sentinel-error style, and schema-cache usage.
- A `Kernel.SynthesizePlatform` recommended entry point returning a typed `*platform.Platform`.
- Fully typed inputs (no raw `cue.Value` escape hatch needed — platform inputs are plain data).

**Non-Goals:**

- No `Materialize` chaining inside the synth path (kept explicit and separate).
- No `core/` schema change.
- No operator/CLI wiring — additive library surface only; callers adopt later.
- No new concreteness-enforcement gate (a typed-input platform is already concrete; validation is what `NewPlatformFromValue`'s metadata decode + the unification in `synth.Platform` already provide).

## Decisions

### D1: Synthesis stops at `*platform.Platform`, not `*MaterializedPlatform`

`SynthesizePlatform` chains `synth.Platform` → `NewPlatformFromValue` and returns the pre-materialize twin. This is the structural mirror of `SynthesizeRelease`, whose downstream (`ProcessModuleRelease`) is pure (no I/O). `Materialize` performs registry I/O and is explicitly caller-driven (D14); folding it in would hide I/O behind a synth call and violate kernel neutrality.

*Alternative considered:* a combined `SynthesizeAndMaterializePlatform`. Rejected for now — it conflates a pure construction step with a registry round-trip and the materialize cache is the caller's concern. Can be added later as a thin convenience without breaking this surface.

### D2: Render-and-compile, not FillPath; and no `userModule` scope dance

`synth.Release` needs the `buildReleaseScope`/`userModule` workaround solely because `#Module` is a closed definition with a self-referential constraint (`modulePath: metadata.modulePath`) that `FillPath` cannot satisfy. `#Platform` has **no** such nested closed-artifact input — all inputs are plain scalars, maps, and lists. So `synth.Platform` renders a CUE source string (`platform: { #Platform, metadata: {...}, type: ..., #registry: {...} }`) and `CompileString`s it with the resolved schema package as `cue.Scope` (to resolve `#Platform`/`#Subscription` references). No overlay field, no value injection.

*Alternative considered:* `schemaPkg.LookupPath("#Platform").FillPath(...)`. Workable since there's no self-reference, but rendering source keeps the implementation visually parallel to `release.go` (`renderReleaseSource`) and sidesteps any FillPath-into-pattern-constraint edge cases on `#registry`.

*Implementation note (landed):* the rendered source unifies the definition explicitly — `platform: #Platform & { … }` — rather than embedding it as a struct field (`platform: { #Platform; … }`). Embedding a closed definition into an open struct literal relaxes its closedness, so an invalid `#registry` key (one violating `#ModulePathType`) would be silently accepted instead of surfacing as a unification error. `#Platform & {…}` preserves the closedness, satisfying the "Invalid catalog path" scenario. The mechanism is still render-source + `cue.Scope` as decided here; only embed-vs-`&` changed.

### D3: Fully typed `Subscriptions`, with `Enable *bool`

```
type PlatformInput struct {
    Name         string                       // required
    Type         string                       // required
    SchemaCache  *schema.Cache                // required
    Description  string                       // optional
    Labels       map[string]string            // optional
    Annotations  map[string]string            // optional
    Subscriptions map[string]SubscriptionSpec // optional; key = catalog module path
}
type SubscriptionSpec struct {
    Enable *bool       // nil → schema default (*true); non-nil → explicit
    Filter *FilterSpec // nil → no filter
}
type FilterSpec struct {
    Range string   // omitted when ""
    Allow []string // omitted when empty
    Deny  []string // omitted when empty
}
```

`Enable` is a pointer so "omitted" defers to the schema's `*true` default instead of forcing `false` — the same distinction `synth.Release` draws between an unset and a supplied value. Empty-string / empty-slice filter fields are simply not rendered, mirroring `writeStringMap` in `release.go`.

*Alternative considered:* a raw `cue.Value` for the registry subtree (as `Release` does for `Values`). Rejected — `Release` only used a raw value because `#config` shapes are open-ended per module; the platform registry shape is fixed and small, so full typing is more ergonomic and on-philosophy.

### D4: Sentinel errors mirror the Release set

New package-level sentinels: `ErrMissingType` (platform-specific). `ErrMissingName`, `ErrMissingSchemaCache`, and `ErrSchemaUnavailable` already exist in `release.go` but their messages are prefixed `synth.Release:`. To avoid misleading messages, introduce platform-scoped sentinels (e.g. `ErrMissingName` is release-worded today) — either add `ErrPlatformMissingName`-style names, or generalize the wording. **Decision:** add distinct platform sentinels with `synth.Platform:` wording; keep the existing release sentinels untouched to avoid churn on `synth.Release` callers. (Open to consolidating later.)

## Risks / Trade-offs

- **[Sentinel duplication]** Two near-identical sets of "missing name/cache/schema" errors → mild surface bloat. *Mitigation:* acceptable; messages stay accurate, and `errors.Is` call sites are explicit about which artifact failed. Revisit if a third artifact synth appears.
- **[`#registry` pattern-constraint rendering]** Emitting a map into `[#ModulePathType]: #Subscription` via rendered source could surprise if a key is malformed. *Mitigation:* that is desired behavior — invalid paths surface as a CUE unification error (covered by a spec scenario), not a silent drop.
- **[No concreteness gate]** Unlike `ProcessModuleRelease`, there's no explicit concreteness enforcement. *Mitigation:* `NewPlatformFromValue` already requires `metadata` to decode, and a typed-input platform cannot be non-concrete; if a future need arises, a `Validate`-style gate is additive.
- **[Doc drift]** `doc.go` currently says the package is ModuleRelease-only. *Mitigation:* update it in the same change (tracked as a task).

## Open Questions

- Should the platform sentinels be consolidated with the release ones into artifact-neutral errors (`ErrMissingName` shared) instead of duplicated? Deferred — D4 chooses duplication for now.
- Do we want a `SynthesizePlatform` variant that also materializes, once a concrete caller (operator Platform CRD) exists? Deferred to that caller's change.
