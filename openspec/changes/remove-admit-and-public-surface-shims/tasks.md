## 1. Remove the install admission (opm/k8s/ownership, opm/k8s/labels)

- [ ] 1.1 Add `TestVerdictInputsCarryNoOverride` (reflection: the field names of `ApplyInput` and `DeleteInput`); see it fail on `Admit`
- [ ] 1.2 Delete `ApplyInput.Admit`, `DeleteInput.Admit`, `admittedForApply`, `admittedForDelete`, `installDeletable`, `carriesNoOtherIdentity`; correct the `CanApply` and `CanDelete` docs; verify `git grep -n "Admit\|installDeletable" -- opm` is empty
- [ ] 1.3 Remove the table rows that set `Admit`; keep the earlier-manifest Namespace and Deployment rows with their refused verdicts; every other row is unedited
- [ ] 1.4 Cite 0012:D8:R3 in place of 0012:D8:R6 at `labels.AnnotationAdopt` and its no-writer test; correct the `labels/` and `ownership/` lines of `AGENTS.md`
- [ ] 1.5 `task check` green, then commit `feat(k8s)!: remove the install admission from the ownership verdicts`

## 2. Remove the two deprecated shims (opm/errors, opm/catalog, opm/schema, opm/internal/loader)

- [ ] 2.1 Empty the exception lists in `TestTypedErrorsUsePointerReceivers` and `TestSurface_OneNamePerType`, add the check that no type in `opm/errors` declares `As` and that the value type is not an error; see them fail
- [ ] 2.2 Give `IdentityError` its pointer receiver and delete `As`; delete `catalog.Source`; correct the docs of `IdentityError`, the `opm/errors` package and `module.Source`; drop the value-target assertions in `opm/errors` and `opm/internal/loader`
- [ ] 2.3 Correct the `errors/` line of `AGENTS.md` and `docs/site/diagnostics/identity-mismatch.md`
- [ ] 2.4 `task check` green, then commit `feat(errors)!: remove the deprecated IdentityError value forms and catalog.Source`

## 3. Record and verify

- [ ] 3.1 Add the closing note to `adr/015-one-shape-for-the-public-surface.md`; write `adr/016-no-install-admission-in-the-ownership-verdicts.md` from `adr/TEMPLATE.md`
- [ ] 3.2 Run `task api:diff`; it lists exactly the five removals of the proposal
- [ ] 3.3 Run `.tasks/consumer-build.sh` on throwaway copies: opm-operator at `origin/main` builds and vets; cli at `refactor/retire-legacy-migration` builds and vets; cli at `origin/main` fails only on `Admit`
- [ ] 3.4 `task check` green and `openspec validate remove-admit-and-public-surface-shims --strict`, then commit `docs(adr): record the end of the install admission and of the surface shims`
