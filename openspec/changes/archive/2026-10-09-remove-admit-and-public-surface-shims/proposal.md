## Why

Two things on the public surface must go before `v1.0.0`, or they are contract for good (CONSTITUTION VI: the module path has no major suffix, so a later removal is a path change). The owner wants both in one breaking beta, so the count of releases without a breaking change (0021:D8:R14) restarts once.

1. The two deprecated shims ADR-015 kept for opm-operator: the value forms of `errors.IdentityError` with its `As` method, and the alias `catalog.Source`. opm-operator left them in opm-operator#273; the cli never used them.
2. The install admission: `Admit` on the ownership verdict inputs. The enhancement record withdrew it (0012:D8:R6 and 0012:D8:R7 removed on 2026-10-09, enhancements#109). The cli, its only caller, stops setting it in cli#357. The adopt rule 0012:D8:R8 stays and does not change.

## What Changes

**BREAKING**, SemVer MAJOR by rule, shipped as a prerelease bump on the beta line (`feat!` with a `BREAKING CHANGE:` footer, ADR-010).

Removed exported names, with their place on `origin/main` (c0a89cf, `v1.0.0-beta.8`) and what a consumer writes instead:

| Removed | Place on `origin/main` | A consumer writes |
| --- | --- | --- |
| `ownership.ApplyInput.Admit` (field) | `opm/k8s/ownership/apply.go:63` | Nothing: delete the field from the literal. The verdict is the one `Admit: false` gave. A user who wants an instance to take over an existing object sets `opmodel.dev/adopt` on it to the instance UUID. |
| `ownership.DeleteInput.Admit` (field) | `opm/k8s/ownership/delete.go:50` | Nothing: delete the field from the literal. An object OPM does not manage is skipped as `not-opm-managed`; a user deletes it by hand. |
| `errors.IdentityError.As` (method) | `opm/errors/identity.go:63` | `errors.As(err, &target)` with `var target *errors.IdentityError`, or `errors.AsType[*errors.IdentityError](err)`. |
| `errors.IdentityError.Error` on the value receiver (the value type stops implementing `error`; `(*IdentityError).Error` stays) | `opm/errors/identity.go:50` | `&errors.IdentityError{...}` where an `error` is needed; a pointer target where one is matched. |
| `catalog.Source` (type alias) | `opm/catalog/catalog.go:67` | `module.Source`. |

Removed unexported code that existed only for the admission: `admittedForApply` (`apply.go:151`), `admittedForDelete` (`delete.go:126`), `installDeletable` (`delete.go:134`), `carriesNoOtherIdentity` (`ownership.go:73`), and the tests that set `Admit`.

Nothing is added to the exported surface. `task api:diff` is expected to list the five rows above and nothing else.

### Verdict before and after, for an input that does not set `Admit`

`Admit` is read in two expressions only: `!opmManaged(in.Live) && !admittedForApply(in)` (`apply.go:109`) and `!opmManaged(in.Live) && !admittedForDelete(in)` (`delete.go:103`). Both helpers start with `in.Admit &&`, so with `Admit` false each expression is `!opmManaged(in.Live)`, which is what the change leaves. Every row is therefore unchanged:

`CanApply`, checked top to bottom, first match wins:

| Input (no `Admit`) | Before | After |
| --- | --- | --- |
| No live object | apply | apply |
| Live object has a deletion timestamp | `terminating` | `terminating` |
| Adopt annotation equals the non-empty instance UUID | apply | apply |
| In inventory, instance UUID empty or no adopt annotation | apply | apply |
| In inventory, adopt annotation names another instance | `adopted-elsewhere` | `adopted-elsewhere` |
| Outside inventory, not OPM-managed | `foreign-object` | `foreign-object` |
| Outside inventory, OPM-managed, annotation equals the live UUID label | `adopted-elsewhere` | `adopted-elsewhere` |
| Outside inventory, OPM-managed, live UUID set and differs | `other-instance` | `other-instance` |
| Outside inventory, OPM-managed, annotation set | `adopted-elsewhere` | `adopted-elsewhere` |
| Outside inventory, OPM-managed, none of the above | apply | apply |

`CanDelete`, checked top to bottom, first match wins:

| Input (no `Admit`) | Before | After |
| --- | --- | --- |
| Namespace or CustomResourceDefinition | `safety-excluded` | `safety-excluded` |
| No live object | `already-absent` | `already-absent` |
| Not OPM-managed | `not-opm-managed` | `not-opm-managed` |
| Live UUID and instance UUID both set and differ | `owner-mismatch` | `owner-mismatch` |
| Adopt annotation set and not this instance | `adopted-elsewhere` | `adopted-elsewhere` |
| Otherwise | proceed, with UID and resourceVersion | proceed, with UID and resourceVersion |

The messages are unchanged too. The only inputs whose verdict changes are the ones that set `Admit: true`, which no longer compile: an admitted object outside the inventory that OPM does not manage and that carries no other instance's UUID was applied and is now `foreign-object`; an admitted `apps` Deployment, RoleBinding or ClusterRoleBinding that OPM does not manage and that carries no UUID label was deleted and is now `not-opm-managed`.

### Downstream migration cost

| Consumer | State | Cost when it adopts the release |
| --- | --- | --- |
| opm-operator `origin/main` (7cf939d, pins beta.8) | Sets no `Admit` (`internal/apply/prune.go:194` is a comment), uses `*oerrors.IdentityError` and `&oerrors.IdentityError{}` only, no `catalog.Source` | The pin bump. One stale comment to correct. |
| cli `refactor/retire-legacy-migration` (9f158d22, cli#357, pins beta.8) | No `Admit` left | The pin bump. |
| cli `origin/main` (03ec4374) | Sets `Admit` in `internal/inventory/guard.go:127` and `internal/operator/migration_plan.go:225` | Does not build against this change until cli#357 merges. This change merges after cli#357. |

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `kubernetes-tier`: the requirement on the operator install admission is removed; the apply verdict requirement loses its two admission clauses; the adopt annotation requirement cites 0012:D8:R3 in place of the withdrawn 0012:D8:R6; a requirement is added that the verdict inputs carry no field whose only effect is to lift a refusal.
- `config-validation`: the requirement that held both the validation marker rule and the receiver rule is split in two; the receiver rule loses its `IdentityError` exception and the scenario for the deprecated value target.
- `platform-artifact`: `opm/catalog` no longer exports the alias `Source`.

## Impact

- Packages: `opm/k8s/ownership`, `opm/errors`, `opm/catalog`; doc comments in `opm/k8s/labels`, `opm/module`; tests in `opm/schema`, `opm/internal/loader`.
- Docs: `AGENTS.md` (layout lines for `errors/`, `labels/`, `ownership/`), `docs/site/diagnostics/identity-mismatch.md`, ADR-015 (closing note), a new ADR-016 for the admission removal.
- Not touched: the kernel, any rename, the unexported `providesFold` fallback in `opm/catalog` (deprecated by ADR-013, decision h2, before library#223; its own removal).
- Security: the change removes the one input whose only effect was to lift an ownership refusal on the caller's word. The caller still supplies inventory membership (`InInventory`) and the instance UUID, which the library trusts as the frontend's ownership record; that is unchanged. It adds no trust boundary, input or secret.
- No `enhancement.yaml`: the change delivers no decision. It removes what 0012:D8 withdrew, and the delivery log has no entry kind for a withdrawal.
