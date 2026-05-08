## Context

The kernel's contract after slice 04 is "give me one validated `cue.Value`." That is a clean kernel input but a poor UX: when a user sets `memory: "tw0gb"` in their values file, the kernel's error says "values do not satisfy #config" — it cannot say "in user-values.cue line 14: memory must match GiB regex." Position-rich diagnostics need the per-source values BEFORE unification.

Tier-1 validation lives outside the kernel. The helper validates each `Layer` independently against the schema, recording any per-layer errors with their CUE source positions. Then it unifies in order. The result is one `cue.Value` plus a `MultiSourceError` collecting per-layer diagnostics.

Three frontends consume this helper:

- **CLI**: `Stack{ {Name: "defaults", ...}, {Name: "values.cue", ...}, {Name: "-f overlay.cue", ...} }`. Each layer's CUE positions point to its file.
- **Operator**: `Stack{ {Name: "defaults", ...}, {Name: "ConfigMap/foo", ...}, {Name: "Secret/bar", ...}, {Name: "ModuleRelease/spec.values", ...} }`. Each layer's "position" is the K8s object name.
- **Crossplane fn**: usually a single layer, since the composition delivers one structured input. The helper still works (single layer, validates that one).

Naming: `ValidateAndUnify` is verbose but unambiguous. Considered `Resolve`, but rejected — collides with claim `#resolution` from catalog 015.

## Goals / Non-Goals

**Goals:**
- One implementation of layering + per-source validation, shared by every frontend.
- Source-positioned diagnostics: the user sees "in `<source>` at line N: <message>" for every error.
- Order is explicit (slice index = override priority); no "policy" baked in.
- Kernel does not depend on the helper; the helper depends on the kernel (it uses `k.CueContext()` to evaluate against the same context as later kernel calls).

**Non-Goals:**
- Defining what layers each frontend should use. That is per-frontend convention. (CLI: `defaults → -f stack`; operator: `defaults → ConfigMap → Secret → CR`; XR: usually one layer.)
- Replacing the kernel's Tier-2 safety net. Both tiers run.
- Per-field source attribution after unification. CUE positions are per-source-file; once unified, a field's "winning" position is whichever layer wrote it last.

## Decisions

**`Layer` is a struct with three fields.** `Name` is human-friendly ("user-values.cue", "CRD spec"). `Source` is a stable identifier (file path, K8s object name) for machine-readable error correlation. `Value` is the raw `cue.Value`. Reason: lets frontends present errors with rich context without special-casing per-frontend.

**`Stack` is `[]Layer` ordered later-overrides-earlier.** No magic. Reason: matches CLI mental model (`-f a -f b -f c` → c overrides b overrides a). Operator/XR explicitly construct in their preferred order.

**`MultiSourceError` aggregates per-layer errors.** Each `Layer` validated independently produces zero or more `*oerrors.ConfigError` instances. The aggregate carries them all so the user sees every problem at once, not one-per-pass.

**Tier-1 stops at validation; does not unify on error.** If any layer has a Tier-1 error, the helper returns the `MultiSourceError` and `cue.Value{}` (zero) for the unified value. Reason: unifying with broken layers produces cascading garbage errors.

**`ValidateAndUnify` does NOT call kernel Tier-2.** It returns the unified value to the caller, who then passes it to the kernel (or invokes the kernel themselves). Reason: separation of concerns; helper does layering + Tier-1, kernel does Tier-2. Frontends that prefer one-call ergonomics can write a thin local helper.

**Helper uses `k *kernel.Kernel`.** Reason: helper-built `cue.Value`s must share the kernel's `cue.Context` (umbrella D8 — CUE values are context-bound). Helper takes `*Kernel`; reaches in for the context.

**Public method on Kernel as ergonomic shortcut.** Add `(k *Kernel) ValidateAndUnify(schema cue.Value, layers Stack) (cue.Value, *MultiSourceError)` delegating to the helper. Reason: discovery — readers who started with `kernel` find layering through the same anchor.

**Removal of `validate.UnifyAndValidate` is in this slice.** No deprecation gap. Reason: that helper was a temporary bridge in slice 04; once Tier-1 lands, leaving the bridge is technical debt.

## Risks / Trade-offs

**Risk — frontends skip the helper.** A consumer in a hurry merges values themselves and feeds the kernel directly, getting Tier-2-only diagnostics. Mitigation: Tier-2 is correct, just less helpful. CHANGELOG and docs strongly recommend the helper. Tooling (linters, examples) prefer it.

**Risk — `MultiSourceError` is verbose.** With many layers each having issues, the error becomes a wall of text. Mitigation: aggregate exposes structured fields so frontends can format per their UX (CLI: pretty-print; operator: condense to top-N in CRD status; XR: compact).

**Risk — helper layout becomes a precedent for sprawling helpers.** Other concerns (loading, platform composition) get their own helper packages later (slices 07, 10). Mitigation: that's intentional. Each helper is small, focused, and frontends pick what they want.

**Trade-off — two validation passes (Tier-1 helper, Tier-2 kernel).** Cost: roughly 2x validation work when both run. CUE validation is cheap relative to render; not measurable at typical sizes.

**Trade-off — frontends now own ordering.** They had to anyway (each had its own stack). The benefit is the helper standardizes error shape and saves boilerplate.
