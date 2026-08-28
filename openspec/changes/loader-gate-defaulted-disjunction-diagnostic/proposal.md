## Why

The loader shape gate (`opm/helper/loader/internal/shape`) refuses a module whose `metadata.version` is a defaulted disjunction (`#VersionType | *"1.0.1"`, the form every `opm module init` scaffold and the whole `modules` v2 fleet carried until 2026-08-28) with `required field "metadata.version" is not concrete`. The verdict is right: a value with a default arm is a constraint with a suggestion, not the value a release moved, and every consumer of the field (`verifyModuleIdentity`, `module/instance.go`, `materialize`) reads it as a literal. The message is wrong in the way that matters: it names `metadata.version`, so the author opens `module.cue`, where the field is a plain `id.Version` reference that `cue vet -c` accepts; nothing points at `identity/identity.cue` or at the default arm. Measured 2026-08-28: diagnosing the fleet refusal took a Go probe of `IsConcrete()` versus `Default()` to find. Publish (`cli`) passes the same artifact because it reads through `String()`, which resolves defaults, so the loader is the first and only place the form is refused, and its message is the only clue.

## What Changes

- The shape gate's not-concrete refusal distinguishes a defaulted disjunction from other non-concrete values: when the field has a default, the error names the default arm and states that identity fields are literals, not defaults. The sentinel (`ErrMissingRequiredField`), the gate's verdict, the loader signatures and every other message are unchanged.
- The `helper-packages` spec states the rule the gate already applies: a defaulted disjunction is not concrete for the gate. Today the spec says only that required fields are "those the schema never defaults", which is the assumption an author-side default violates without contradicting.

No behavior change for any well-formed artifact; no change for `ErrWrongKind` or `ErrInvalidPackage`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `helper-packages`: the "Loader Shape Gate Validation" requirement gains the defaulted-disjunction clause and a scenario for its diagnostic.

## Impact

- `opm/helper/loader/internal/shape/shape.go` (`requireConcrete`): message text only. Internal package, no public API change.
- Downstream (CLI, controller, Crossplane function): unchanged branching (`errors.Is` on the same sentinel); a better message on the next library bump.
- SemVer: PATCH (`fix(loader)`).
- Complexity: one `Default()` call on the already-failed branch; no new types, options or symbols.
- Enhancement: none. Sibling changes in `cli` (scaffold and templates emit a literal; publish runs the loader) and `opm-operator` (fixtures) address the producers of the refused form; this change stands without them.
