# 01-matcher-mechanics

## Hypothesis

The `#ContextBuilder` capability-matching block specified in [006/03-schema.md §`#ContextBuilder` matching step](../../03-schema.md#contextbuilder-matching-step) produces correct outputs under `cue vet -c` for all five required/optional × match/missing/mismatch outcomes, and per-platform variation via plain CUE unification of `#Platform` values (006 D4 / OQ6) works as the design claims.

| Case | Scenario | Expected |
| ---- | -------- | -------- |
| 01-required-match | required FQN, provider present, schema matches | concrete spec, `cue vet -c` passes |
| 02-required-missing | required FQN, no provider | spec incomplete, `cue vet -c` fails naming `out.consumes.required.<fqn>.spec.<field>` |
| 03-required-mismatch | required FQN, provider violates capability schema constraint | CUE bottom localized at the offending field |
| 04-optional-match | optional FQN, provider present | concrete spec, passes |
| 05-optional-missing | optional FQN, no provider | entry absent entirely, passes |
| 06-platform-inheritance | `#KindDev: #KindBase & {#provides: {…}}` | base entries inherited, dev entries added; module consuming both resolves cleanly |

## Setup

Copied / trimmed from `apis/core/v1alpha2/`:

- `schemas/types.cue` — only the named types the matcher touches.

Authored fresh per [006/03-schema.md](../../03-schema.md):

- `schemas/capability.cue` — `#Capability`, `#CapabilityMap`.
- `schemas/module.cue` — trimmed `#Module` carrying just `metadata`, `#components` (stub), `#config`, and the new `#consumes` block.
- `schemas/platform.cue` — trimmed `#Platform` carrying just `metadata` and the new `#provides` block.
- `schemas/context_builder.cue` — `#ContextBuilder` with only the 006 inputs (`#platform`, `#consumes`) and the `_matched` block. The 004 runtime-context outputs are out of scope here and elided.
- `schemas/capabilities.cue` — two concrete capability definitions (`#Route`, `#StorageClass`) plus FQN constants. `#Route.spec.domain` carries a regex (`=~"^[a-z0-9.-]+\\.[a-z]+$"`) so the mismatch case can produce a constraint violation localized at one field.

Each fixture lives in its own package under `cases/NN-name/main.cue`.

## Run

```bash
cd enhancements/006-platform-capabilities/experiments/01-matcher-mechanics
./run.sh
```

The script invokes `cue vet -c ./cases/<name>/...` for each case and reports pass/fail. Positive cases (01, 04, 05, 06) must pass; negative cases (02, 03) must fail with the expected diagnostic.

## Outcome (2026-05-15)

All six cases produce the expected results. Captured output:

```text
=== 01-required-match (expect pass) ===
[result] pass

=== 02-required-missing (expect fail) ===
result.consumes.required."opmodel.dev/exp/caps/routing/route@v1".spec.domain: incomplete value =~"^[a-z0-9.-]+\\.[a-z]+$":
    ./schemas/capabilities.cue:19:20
[result] fail

=== 03-required-mismatch (expect fail) ===
result.consumes.required."opmodel.dev/exp/caps/routing/route@v1".spec.domain: invalid value "no-tld" (out of bound =~"^[a-z0-9.-]+\\.[a-z]+$"):
    ./schemas/capabilities.cue:19:20
    ./cases/03-required-mismatch/main.cue:36:17
    ./schemas/capabilities.cue:19:11
[result] fail

=== 04-optional-match (expect pass) ===
[result] pass

=== 05-optional-missing (expect pass) ===
[result] pass

=== 06-platform-inheritance (expect pass) ===
[result] pass
```

### Findings

- **D6 confirmed.** The required-missing diagnostic names the precise FQN-keyed path: `result.consumes.required."opmodel.dev/exp/caps/routing/route@v1".spec.domain: incomplete value …`. No surrounding error noise, no path obscuration. Modules will get an actionable message naming the capability FQN that needs a provider.
- **D6 alternative branch (schema mismatch) confirmed.** Constraint violations surface as `invalid value "no-tld" (out of bound =~…)` at the same FQN-keyed path. The localization is exact — only the offending field's spec is reported.
- **Conditional struct embedding inside a `for` comprehension works.** The design's recipe — `(fqn): cap & { if #platform.#provides[fqn] != _|_ { #platform.#provides[fqn] } }` — evaluates correctly under CUE v0.16.1. The `*` default disjunction alternative noted in 03-schema.md §"Why a conditional struct…" was not tested as a counter-fixture here; the positive form is what the design ships.
- **D9 confirmed.** Empty `#provides: {}` (case 02, 05) iterates cleanly through the `for fqn, cap in #consumes.required` comprehension. No `_|_` guard at the use site needed.
- **D7 + D8 confirmed (structurally).** `#consumes` is the read surface — case 01's assertion `_assertDomain: "apps.example.com" & result.consumes.required[v1.RouteFQN].spec.domain` reads through the matcher's output cleanly. Component-body string interpolation is exp 02's job.
- **OQ6 confirmed.** `#KindDev: #KindBase & {#provides: {…}}` works as plain CUE unification with one caveat: literal values on `#KindBase` (e.g. `metadata.name: "kind-base"`) conflict with overrides on derived platforms. The fix is the standard CUE pattern — use `*` default disjunctions for fields that derived platforms are expected to override (`metadata: name: *"kind-base" | _`). This is idiomatic CUE; no new construct needed. Tooling-driven inheritance graph (the optional `#extends: #Platform` metadata field hinted at in OQ6) remains a separate question and is not blocked by this finding.

### Notes for 006 docs

- 03-schema.md §"Why a conditional struct…" — the positive form is validated; a counter-fixture showing the `*` default form does fail would strengthen the section but is not strictly necessary.
- 04-decisions.md D6 — confirmed; can be promoted from "expected behavior" to "validated under cue v0.16.1".
- 04-decisions.md D9 — confirmed.
- 04-decisions.md D4 / OQ6 — confirmed with the override-via-default caveat. The OQ6 entry could note that derived platforms must use default disjunctions for any base-pinned field they want to override; this is a small piece of guidance worth surfacing if `#extends` formalization is ever proposed.

## Status

Concluded — 2026-05-15. May be deleted once 006 lands in `apis/core/v1alpha2/`.
