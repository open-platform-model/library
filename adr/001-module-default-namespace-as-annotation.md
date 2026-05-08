# ADR-001: Express module default namespace as an annotation, not a typed field

## Status

Accepted

## Context

`#Module` (currently in `apis/core/v1alpha2/module.cue`) exposes a typed `defaultNamespace?: string` field on `metadata`. The same metadata block also carries an annotation slot, `module.opmodel.dev/defaultNamespace?: string`, so the same intent can be expressed two ways.

Namespace is a runtime concern. It belongs to the deployment context — the target cluster, the environment, the operator's policy — not to the portable module definition. Promoting a namespace field on `#Module` invites authors to think in terms of where their module will run, which is precisely the coupling the module/release split was introduced to remove. `#ModuleRelease` already owns the authoritative namespace at deploy time (`metadata.namespace!`).

At the same time, vendors and module authors have a legitimate need to suggest a default namespace — a hint to operators and tooling about where the module is conventionally deployed (e.g. `cert-manager`, `kube-system`, `ingress-nginx`). Removing the capability outright would push that hint into documentation and weaken the contract.

The annotation form already exists alongside the field. It carries the same information, is namespaced under `module.opmodel.dev/`, and is consistent with how other operator-facing hints are surfaced in OPM and in Kubernetes more broadly. The duplication is the cost we are paying for not having chosen.

## Decision

Remove `metadata.defaultNamespace` from `#Module`. Keep the annotation `module.opmodel.dev/defaultNamespace` as the sole way to express a module's suggested default namespace.

The annotation is advisory: tooling and operators MAY consult it to seed a default `metadata.namespace` on a `#ModuleRelease`, but `#ModuleRelease` and its consumers remain the authoritative owners of the actual deployed namespace.

Alternatives considered:

- **Keep the typed field, drop the annotation.** Stronger schema validation, but reinforces the wrong mental model — module authors get a first-class field for a runtime concern, and operators inherit a value that looks authoritative when it is in fact only a hint.
- **Keep both.** Status quo. Two ways to say the same thing diverge over time; downstream code has to consult both and reconcile conflicts. Rejected on grounds of simplicity (Constitution VII).
- **Drop the capability entirely.** Cleanest schema, but loses information vendors actually want to publish. Pushes the hint into prose docs and weakens the discovery story.

## Consequences

**Positive:** `#Module` schema becomes a clearer statement of authorial intent — what the module is, what it configures, what it composes — without leaking deployment context. The module/release boundary holds. The annotation form is consistent with other operator-facing hints and does not need schema-level enforcement to be useful.

**Negative:** Downstream Go code that decodes the typed field must change. `pkg/module/module.go` currently mirrors the CUE field with `ModuleMetadata.DefaultNamespace string`; it must be removed or rewritten to read the annotation. Any module already authored against the field will fail to compile against the new schema and must migrate to the annotation form. The catalog repository carries parallel `#Module` definitions in `core/v1alpha1` and `core/v1alpha2` that also expose `defaultNamespace?`; they need the same change to keep the two repos consistent, or an explicit note that the catalog copies are now downstream mirrors.

**Trade-off:** Annotations are less discoverable than typed fields — a module author has to know the annotation key exists to set it. We accept this in exchange for a cleaner schema and a stricter module/release boundary. Tooling (`opm mod`, docs generation, IDE schema hints) is the right place to make the annotation discoverable, not the type system.
