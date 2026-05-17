# 02-release-flow-ordering

## Hypothesis

The 3-step `#ModuleRelease` flow specified in [004/03-schema.md §`#ModuleRelease` integration](../../03-schema.md#modulerelease-integration) — (1) unify values into `#config`, (2) feed the post-config `#components` to `#ContextBuilder`, (3) unify the builder's outputs back into the module — is the only ordering that produces correct `#names` for dynamic components built from `#config`. The release/module identity layer (D11) propagates verbatim. Inline-literal `v1.#Module & {…}` fixtures need `#ctx: _` / `#names: _` declarations to bring those identifiers into the inline literal's lexical scope (D35).

| Case | Scenario | Expected |
| ---- | -------- | -------- |
| 01-static-components | static component map; full `#ModuleRelease` flow | `release.*` / `module.*` identity verbatim; component gets `#names` |
| 02-dynamic-from-config | `for _srvName, _c in #config.servers { "server-\(_srvName)": ... }` with two server values | each dynamic component gets `#names.dns.fqdn`; lock-step holds vs `#ctx.runtime.components` |
| 03-config-after-builder-fails | run `#ContextBuilder` twice — once on bare-module `#components`, once on `_withConfig.#components` — same release | bare-module result has 0 components (silent failure); config-first result has both entries |
| 04-inline-literal-scope | inline `v1.#Module & {#ctx: _, #components: web: {#names: _, spec: url: "http://\(#names.dns.fqdn)..."}}` | component body's `#names.dns.fqdn` reference resolves through inline-declared identifiers; release concretizes the URL |

## Setup

Schemas are a copy of `01-names-cascade-and-injection/schemas/` (the "copy, never reference" convention — each experiment is self-contained). The post-slim `#ContextBuilder` and the 3-step `#ModuleRelease` flow are exactly the forms from [004/03-schema.md](../../03-schema.md).

Cases:

- **01-static-components** — full `#ModuleRelease` round-trip; asserts identity propagation and a single static component's `#names`.
- **02-dynamic-from-config** — module declares `#config.servers: [string]: {port: int}` and a `for` comprehension that materialises one component per server entry. Release values supply two servers. Per-component `#names.dns.fqdn` is asserted via literal-unification.
- **03-config-after-builder-fails** — counter-fixture that exercises the wrong ordering directly. `_wrongResult` calls `#ContextBuilder` with `_module.#components` (bare module, no values unified — comprehension produces zero entries because `#config.servers` is still the pattern constraint). `_rightResult` calls it with `(_module & {#config: values}).#components`. The case asserts `len(_wrongResult.ctx.runtime.components) == 0` and `len(_rightResult.ctx.runtime.components) == 2`. The 0-length assertion is the documented silent-failure shape — no CUE error, just missing data.
- **04-inline-literal-scope** — minimal inline-literal demonstrating the working form: `#ctx: _` at the module-literal level and `#names: _` at the component-literal level bring those identifiers into lexical scope. The component body uses the package-lexical identifier `#names.dns.fqdn` in a string interpolation. Removing either declaration line yields an `undefined field` CUE error at the reference site — see Findings below for the rationale.

## Run

```bash
cd enhancements/004-module-context/experiments/02-release-flow-ordering
./run.sh
```

`cue vet -c ./cases/<name>/...` per case. All four cases must pass.

## Outcome (2026-05-15)

All four cases produce the expected results.

```text
=== 01-static-components (expect pass) ===
[result] pass

=== 02-dynamic-from-config (expect pass) ===
[result] pass

=== 03-config-after-builder-fails (expect pass) ===
[result] pass

=== 04-inline-literal-scope (expect pass) ===
[result] pass
```

### Findings

- **D11 confirmed.** `release.*` (name, namespace) and `module.*` (name, version, fqn) propagate verbatim into `#ctx.runtime` with no transformation. Case 01 unifies the literal values against the runtime fields and produces no conflict.
- **D34 confirmed (positive form).** The 3-step `#ModuleRelease` flow produces correct `#names` for both dynamic and static components. Case 02's two `server-*` entries have their FQDNs computed correctly through the cascade.
- **D34 confirmed (silent-failure form).** Case 03 directly demonstrates the documented failure shape: invoking `#ContextBuilder` against `_module.#components` (where `#config.servers` is still a pattern constraint, not concrete data) yields a zero-length `ctx.runtime.components` map. There is no CUE error — only missing data — which is the failure-mode signature that motivated the ordering decision. Modules that read their own `#names.dns.fqdn` would render with broken self-references under the wrong order and tests pass for static-component modules but silently degrade for dynamic ones.
- **D35 confirmed.** Case 04 works because `#ctx: _` and `#names: _` are declared inside the inline literal at the same nesting level as their references. Lexically, CUE resolves the identifiers at the literal's scope — the case file's `package case04` doesn't export `#ctx` / `#names`, and CUE does not walk INTO the value being unified-against to discover them. In production module files (in the `v1alpha2` package alongside `#Module` / `#Component`), these declarations come free from package scope; the inline-literal form is a TEST-fixture cost only.
- **D32 (lock-step) confirmed for dynamic components.** Case 02's `_lockstepAlpha` / `_lockstepBeta` unify `release.components."server-alpha".#names` with `release.ctx.runtime.components."server-alpha"` and produce no conflict. The lock-step that 01-names-cascade-and-injection validated for static components also holds for dynamically-generated ones.

### Notes for 004 docs

- `04-decisions.md` D11, D34, D35 — confirmed under CUE v0.16.1; can be promoted from "expected behavior" to "validated".
- `02-design.md` §`#ModuleRelease` invokes `#ContextBuilder` — the 3-step ordering rationale is now empirically grounded. The silent-failure mode for static-component modules under the wrong ordering (`04-decisions.md:441`) is exactly what case 03 produces.
- D35 wording suggestion: 04-decisions.md could note that inline-literal `v1.#Module & {…}` fixtures (test files, examples in walkthroughs, docs) need to declare `#ctx: _` and any `#names: _` references at the literal's nesting level. This isn't a 004 design issue — it's idiomatic CUE — but worth surfacing in a "writing test fixtures" footnote to spare authors the lookup error.

## Status

Concluded — 2026-05-15. May be deleted once 004 lands in `apis/core/v1alpha2/`.
