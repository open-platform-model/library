## Why

`opm/k8s/ownership` carries an install admission: `ApplyInput.Admit` and `DeleteInput.Admit`, plus
the unexported `installDeletable` kind set. A caller sets `Admit` for an object it has proven came
from an earlier release's install manifest, so that the first module-based install of the
controller could take over a manifest-installed one (the earlier text of 0012:D8:R6/R7). The owner
decided on 2026-10-05, as part of renaming the operator to the controller, to remove every piece of
legacy-migration code: nobody runs OPM yet, so there is no manifest-installed controller to take
over. The enhancement withdraws 0012:D8:R6/R7. No frontend ever set `Admit`; the cli removes its
own migration (its `inventory.AdmitSet`) in a change of its own.

`Admit` has no caller. Removing it now is free; once a frontend calls it, removing it is a real
break.

## What Changes

- **BREAKING** `opm/k8s/ownership`: remove the `Admit` field from `ApplyInput` and `DeleteInput`,
  and the unexported `admittedForApply`, `admittedForDelete`, `installDeletable` and
  `carriesNoOtherIdentity` that only admission used. `CanApply` refuses every live object outside
  the inventory that OPM does not manage, unless its adopt annotation names this instance.
  `CanDelete` skips every live object OPM does not manage.
- The tests drop the admission cases. The fixtures that named the earlier install's objects
  (`opm-operator-system`, `opm-operator-manager`) go with them.
- The adopt-annotation citations (`opm/k8s/labels`, its no-writer test, `AGENTS.md`) cite 0012:D8
  instead of 0012:D8:R6, since the requirement they cited changes. The package map line in
  `AGENTS.md` drops the install admission.
- Spec `kubernetes-tier`: REMOVED the requirement "The operator install admission lifts only a
  proven object's ownership refusal" and its ten scenarios; MODIFIED the apply verdict and the
  adopt-key requirements to drop the admission and the R6 citation.
- Library #220 (the adopted-elsewhere refusal, 0012:D8:R8) merged first and built on the
  admission: it added an admitted terminating row, an admitted adopted-elsewhere row on each side,
  "unless admitted" wording and two admission scenarios. Those go with the rest. The
  adopted-elsewhere refusal itself and its 0012:D8:R8 citations stay; R8 is live.

Not in this change: renaming the operator to the controller in the rest of the library (the
follow-up change `follow-controller-rename` on the same branch), and the enhancement text itself
(the enhancements repo).

## Impact

- Affected packages: `opm/k8s/ownership` (public fields removed), `opm/k8s/labels` (comment only).
- SemVer: a break to `opm/` types. On the beta line it lands as a `!` commit with a
  `BREAKING CHANGE:` footer and advances the `-beta.N` counter (Principle VI, ADR-010).
- Downstream migration cost: none. The cli and the controller pin `v1.0.0-beta.6`, which carries
  the field, but neither imports `opm/k8s/ownership` yet, and the cli deletes its migration before
  it adopts the package. A caller that set `Admit: false` deletes the field.
- Complexity: removes code; no addition.
