## Why

Two rendered objects with the same apiVersion, kind, namespace and name reach apply as two writes to one object, and the last one wins silently: nothing in the kernel, the CLI or the operator notices. Enhancement 0015 D15 names the case that motivates fixing it now: a module shipping two `transformer-registration` components renders two cluster-scoped `TransformerRegistration` objects under one instance-derived name (D12), so the second overwrites the first at server-side apply and the claim that survives is whichever was applied last. D15 says a second registration is refused loudly, naming both carrying components, before apply. The refusal is not registration-specific at all: it is "a render must not produce two objects with one identity", which the kernel's render output makes checkable in one place, with the provenance the rows already carry (`Compiled.Component`, `Compiled.Transformer`).

## What Changes

- **`opm/helper/objectset` (new).** An opt-in helper both runtimes call between render and apply: `Duplicates(compiled []*kernel.Compiled) []Duplicate` reads each object's apply identity (`apiVersion`, `kind`, `metadata.namespace`, `metadata.name`) off the rendered value and returns every identity two or more objects share, each row naming the identity and every producer (component and transformer) in render order. `*DuplicateIdentitiesError` aggregates the rows into the refusal a runtime raises, worded once so the CLI and the operator say the same thing. An object without a kind or a name is not a Kubernetes object and is skipped, not refused.
- **The kernel is untouched.** `Compiled` deliberately carries no platform vocabulary (kernel neutrality, Principle I; the doc comment says so), and apiVersion, kind, namespace and name are Kubernetes vocabulary, so the check lives in the helper tier a frontend MAY skip (Principle III, `helper-packages`). `Render` does not call it and returns no error declared under `opm/helper/`; the depguard rule stays green.
- **Docs.** `opm/helper/doc.go`, `CLAUDE.md` § Repository Layout and the `helper-packages` spec list the second subpackage.
- **Not in this change.** The refusals themselves: the CLI raises the error after every render (build and apply), the operator after its render before apply, each in its own change; the operator's completes D15, because the cluster is where "before apply" is decided. A CUE-side duplicate verdict in the render glue: it would put Kubernetes vocabulary into the kernel's build, which is the neutrality line this change keeps. Deduplicating instead of refusing: two producers for one object is a module authoring error, not an arbitration.

## Classification

**MINOR, additive** (Principle VI): one new package under `opm/helper/` with three exported types, one function and one error; nothing outside `opm/helper/` changes. Complexity added: one scan over the rendered set, with two named readers before it ships (the cli and operator changes that follow) and one motivating decision (0015 D15). A per-runtime copy was the alternative; two copies of one refusal with two wordings is what the helper tier exists to avoid.

## Downstream consumers

- **`cli`** (library `alpha.32`): its own change calls `objectset.Duplicates` in `internal/workflow/render` after `Render`, for `module build`, `instance build` and every apply, and formats the error through `cmdutil` like the other gate causes. The call site is `internal/workflow/render/render.go:142`, in `renderInstance` between the replacement warnings and `converted := make(...)` — verified to compile against this tree. Until then nothing changes.
- **`opm-operator`** (library `alpha.32`): its own change calls it in `resultFromRender` before inventory entries are built, refusing with `RenderFailed` naming the identity and both producers; that change declares D15. The call site is `internal/render/kernel_module_renderer.go:202`, immediately before `entries, err := buildInventoryEntries(resources)` — verified to compile against this tree. Until then `buildInventoryEntries` keeps emitting one inventory entry per object and Flux applies both.
- **`catalog_opm`**, **`modules`**: nothing. A module that trips the guard was already broken at apply.

## Capabilities

### New Capabilities

- `duplicate-object-identities`: what the helper detects, what a duplicate row carries, what is skipped, and the refusal's wording contract.

### Modified Capabilities

- `helper-packages`: the "Helper Layout for Future Subpackages" requirement lists `objectset` beside `platformmodule`.

## Impact

- `opm/helper/objectset/objectset.go`, `objectset_test.go`, `doc.go` (new).
- `opm/helper/doc.go`, `CLAUDE.md` § Repository Layout.
- `opm/kernel/flow_integration_test.go`: the shipped fixture render asserts no duplicate, so the healthy path is pinned against the published catalog.
- Release: `feat` (library `v1.0.0-alpha.33`).
