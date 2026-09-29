## Why

A component that attaches a provider-fulfilled contract (the catalog's `backup` trait today) cannot render anywhere the provider is not installed: on a module author's laptop, where the render uses the module's own deps, and on a cluster that runs no operator, where no provider can ever be registered. The kernel refuses the whole render, so the author sees nothing of the objects that would render. The cli is adding `--skip-unprovided` to render what the platform can, and needs the kernel to do the skipping, because matching and the fail-closed gate live in the build and a frontend must not re-derive them.

This change implements the rule core states in the `skip-unprovided-provider-demands` change (`core/SPEC.md` §2.1 and §3.1, capability `contract-fulfilment`): a render's caller may ask to skip a provider-fulfilled demand that nothing on the platform provides, and every skipped demand is reported.

## What Changes

- `kernel.RenderInput` gains `SkipUnprovided bool`. Off, the default, nothing changes: every current caller renders exactly as before.
- A demand is **unprovided** when its contract declares `fulfilment: "provider"` and no enabled registry entry carries a transformer that requires that key (the count the in-build single-provider guard already computes). The build marks every unresolved row with this fact on every render, switch on or off: `oerrors.UnresolvedDemand` gains `Unprovided bool`, and its message gains a suffix saying so.
- With the switch on, the build moves unprovided demands out of the refusal and into a new `skipped` verdict:
  - a skipped trait demand leaves its component rendering every pair it matched;
  - a skipped resource demand omits its whole component (no pair of it renders, and it is not reported unmatched), because a partly satisfied component never renders.
- `kernel.RenderDiagnostics` gains `Skipped []SkippedDemand`, decoded from the build as a row per skipped demand: component, contract key, kind, defining catalog, same-base alternatives, and whether the component was omitted.
- Everything else still refuses under the switch: a catalog-fulfilled unresolved demand, a provider-fulfilled demand whose provider exists but did not match, an over-subscribed contract, an unmatched component. Optional traits are unaffected.
- The render module's own `gate` agrees with the kernel under both switch values.
- The `unresolved-demands` diagnostics page and the `opm/kernel` package doc describe the switch and the new message suffix.

SemVer: **MINOR**. Every public change is additive (one input field, one diagnostics field, one new row type, one row field); the default path's verdicts and rendered output are unchanged.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `single-build-render`: adds the caller's switch to skip unprovided provider-fulfilled demands and its reporting; the unresolved-demand row gains the unprovided marker and message suffix; the skipped rows join the advisory facts a result exposes as data.

## Impact

- **Packages:** `opm/kernel` (`RenderInput`, `RenderDiagnostics`, new `SkippedDemand`, decode, package doc), `opm/errors` (`UnresolvedDemand.Unprovided`, `describe()`), `opm/internal/renderstage` (glue slot and template: the unprovided predicate, the skipped verdict, omitted components leaving `pairs` and `unmatched`), test fixtures under `testdata/render/scenarios`.
- **Downstream:**
  - `cli` change `add-skip-unprovided-flag` consumes the switch, the `Skipped` rows and `Unprovided` after this change is released (library 1.0.0-alpha.34 expected).
  - `opm-operator` needs nothing: it never sets the switch, and the new fields are zero-valued or informational for it. Its refusal messages gain the suffix on provider-fulfilled rows when it next bumps the library.
- **Depends on:** the rule text of core change `skip-unprovided-provider-demands` (SPEC-only, no core release, no pin moves). This change implements against that change's spec delta and merges after it.
- **No enhancement:** planned without an enhancement entry (user decision 2026-09-29), so no `enhancement.yaml`.
