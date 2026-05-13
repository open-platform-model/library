# Research: CUE `self` Identifier (aliasv2 experiment)

| Field       | Value            |
| ----------- | ---------------- |
| **Status**  | Research notes   |
| **Created** | 2026-05-01       |
| **Authors** | OPM Contributors |
| **Subject** | CUE language `self` identifier, postfix alias syntax, `aliasv2` experiment |

## Summary

CUE has an experimental `self` predeclared identifier, available since `v0.15.0-alpha.2`, that refers to the innermost surrounding struct or list. It ships behind a per-file experiment named `aliasv2`, alongside a unified postfix alias syntax (`~`) that replaces CUE's seven prior `=`-based alias forms. Both features are in **preview** as of v0.15.0 and may still change.

`self` was introduced because CUE previously had no way to refer to the current scope as a value — package roots, the enclosing struct, or non-identifier field names could not be referenced cleanly from within a definition.

## What `self` Is

Per the official how-to guide:

> The `aliasv2` experiment "enables the use of `self` identifier to refer to the enclosing struct and enables the postfix alias syntax (`~X` and `~(K,V)`)."

Per the proposal (`Postfix Aliases`, GitHub Discussion #4014):

> "We also propose introducing the `self` keyword to refer to the current scope."

`self` is a **predeclared identifier** (like `len`, `or`, `and`), not a keyword reserved everywhere — it only takes special meaning inside `aliasv2`-enabled files.

## How It Works

### Scoping rules

1. **Lexical scoping.** `self` refers to the value to the right of the innermost colon — i.e. the innermost CUE block (struct or list) that lexically contains the identifier.
2. **Curly-brace scopes.** Each `{}` opens a new scope.
3. **List scopes.** Each `[]` also opens a new scope, so `self` inside a list literal refers to the list itself.
4. **Implicit scopes count.** Shorthand nested-field syntax counts as a scope.

The implicit-scope rule has a notable consequence:

```cue
a: b: self.c   // self == a.b, so this refers to a.b.c, NOT top-level c
// Formatter rewrites to: a: { b: self.c } — the implicit struct counts
```

### Use cases

The official guide and proposal demonstrate four primary use cases:

#### 1. Reference fields with non-identifier names

CUE allows field labels that are not valid Go-style identifiers (e.g. `"-foo"`, `"a.b"`). Before `self`, these were unreachable from sibling expressions:

```cue
@experiment(aliasv2)

"-foo": 10
d: self["-foo"] + 1   // d == 11
```

#### 2. Bind a stable reference to the package root

Useful inside deeply nested fields where ordinary lookups would shadow or be ambiguous:

```cue
@experiment(aliasv2)

let Root = self

users: {
    alice: {
        // Reach back to package-level fields, even if shadowed:
        domain: Root.config.domain
    }
}

config: domain: "example.com"
```

#### 3. List self-reference

`self` inside a list literal binds to the list:

```cue
h: [10, self[0] + 1, self[1] * 2]
// h == [10, 11, 22]
```

#### 4. Self-referential constraints (definitions)

The proposal sketches usage inside definitions where the value being constrained is referenced as `self`:

```cue
#Even: {2 * div(self, 2)} | error("\(self) is not even")
```

### Postfix alias syntax (companion feature)

`aliasv2` also replaces all prior `=`-based alias forms with a single postfix `~` operator. This is a separate change that ships in the same experiment and is the reason the experiment is called `aliasv2` rather than `self`.

Forms:

- `X~V` — single form: alias `V` refers to the value of `X`
- `X~(L,V)` — dual form: alias `L` refers to the field label, `V` to the value
- `a~X~Y: int` — multiple aliases bound to the same field

Companion builtins:

- `keyOf(X)` — retrieve the field label
- `refOf(X)` — retrieve the field reference
- `valueOf(X)` — retrieve the value (also accessible by using `X` directly)

Example (from the proposal):

```cue
@experiment(aliasv2)

foo~X: {
    bar: 123
    baz: X.bar + 1   // X is the value of foo
}
```

When `aliasv2` is enabled in a file, **the old `=` prefix alias syntax is disallowed** in that file.

There is also an unrelated rename in the same experiment: the `fallback` keyword is replaced by `otherwise`.

## How to Enable

Two mechanisms, both per-file:

### 1. File-level attribute (manual)

Add at the top of any `.cue` file:

```cue
@experiment(aliasv2)
```

Multiple experiments combine on separate lines.

### 2. Automatic migration via `cue fix`

Rewrite an existing file or package to the new syntax and add the attribute:

```bash
cue fix --exp=aliasv2 ./path/to/package
```

`cue fix` translates `=`-based aliases into `~` postfix form and inserts the `@experiment(aliasv2)` attribute.

### Versioning behaviour

- Per-file experiments track the language version declared in the module's `cue.mod/module.cue`, falling back to the version reported by `cue version`.
- Targeting language version `v0.15.0` or later is required.
- The experiment is in **preview** — `cue fix` may need to be re-run after upgrading CUE versions.

## Status and Stability

| Aspect | State |
| --- | --- |
| First availability | `v0.15.0-alpha.2` |
| Current state | Preview in `v0.15.0` (and v0.15.x line) |
| Standardised in language spec? | **No.** The official CUE Language Specification (cuelang.org/docs/reference/spec/) does not yet mention `self`, `aliasv2`, or postfix aliases. Only prefix `=` aliases are specified. |
| Migration policy | Opt-in per file; old prefix aliases disallowed in opted-in files; auto-rewrite via `cue fix` |
| Known regressions | Issue [#4228](https://github.com/cue-lang/cue/issues/4228) — stack overflow on `v0.15.3` when `aliasv2` is enabled against Kubernetes definitions |
| Earlier experiment name | `aliasandself` (renamed to `aliasv2` during development) |

Open design questions noted in the proposal (#4014):

1. Whether the dual-binding form `~(K,V)` could be simplified to just `~K` (akin to Go range loops).
2. Whether prefix notation might be more discoverable than postfix.
3. Interaction with list element aliases and potential `let` support inside lists.
4. Recent (Sept 2025) refinement of `self` scoping rules — final rules still subject to change.

Because `self` and `aliasv2` are not yet in the language specification and have known regressions on real-world schemas, treating them as **not-yet-stable** is appropriate for any production OPM dependency choice.

## Relevance to Enhancement 004 (`#ctx`)

`self` is **not** a substitute for `#ctx`. The two operate on different problems:

| Concern | Mechanism |
| --- | --- |
| Lexical self-reference inside one struct or list | `self` (aliasv2) |
| Cross-document context injection (release -> module -> component) | `#ctx` (this enhancement) |

`#ctx` carries information that is not present in the module's source at all (release name, namespace, cluster domain, platform facts). No amount of intra-file self-reference solves that. `#ContextBuilder` unification remains required.

Where `self` *could* improve ergonomics in the schemas defined by 004:

- Inside `#ComponentNames` and `#ContextBuilder`, where derived fields currently restate full paths back to their containing struct, `self` could shorten expressions and remove fragile path duplication.
- For `#ctx.platform`-style open extension structs where platform teams may publish keys with non-identifier names (e.g. domain-style keys), `self["…"]` provides clean access without renaming.

These are **opportunistic ergonomic wins** for `03-schema.md`, not architectural changes. Adopting `aliasv2` in OPM CUE sources should wait until:

1. The experiment exits preview (specification text exists, no `--exp` flag required).
2. Issue #4228-class regressions are resolved on Kubernetes-shaped schemas (OPM transformers compile against `k8s.io/api/...` mirrors that resemble the trigger case).
3. `apis/core/cue.mod/module.cue` is willing to require the corresponding CUE version.

Until then: track the proposal, do not depend on `self` in shipped catalog modules.

## Sources

All sources are first-party (CUE developers) or hosted on the official `cue-lang` GitHub organisation.

- [Trying the "aliasv2" experiment](https://cuelang.org/docs/howto/try-aliasv2-experiment/) — official how-to, cuelang.org
- [`cue help experiments`](https://cuelang.org/docs/reference/command/cue-help-experiments/) — official CLI reference, cuelang.org
- [CUE Language Specification](https://cuelang.org/docs/reference/spec/) — official spec (notable: does not yet describe `self`/aliasv2)
- [cue-lang/cue release v0.15.0](https://github.com/cue-lang/cue/releases/tag/v0.15.0) — release notes that introduce the experiment
- [cue-lang/cue release v0.15.4](https://github.com/cue-lang/cue/releases/tag/v0.15.4) — subsequent point releases
- [Postfix Aliases proposal — Discussion #4014](https://github.com/cue-lang/cue/discussions/4014) — design proposal authored by CUE maintainers
- [cue-lang/proposal](https://github.com/cue-lang/proposal) — CUE Project Design Documents repository
- [Issue #4228 — stack overflow on v0.15.3 with aliasv2 and Kubernetes definitions](https://github.com/cue-lang/cue/issues/4228) — known regression report
