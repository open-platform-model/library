# Render scenario instances

One CUE module, one instance package per matching or execution outcome the
single-build render tests exercise (`opm/kernel/render_test.go`,
`opm/kernel/render_skip_test.go`). Each package
is a self-contained `#ModuleInstance` whose `#module` is authored inline
against the render fixture catalogs, so the outcome under test is the only
thing that differs between them:

| Package | Outcome |
| --- | --- |
| `missing` | a resource demand with an empty bucket, plus a load-bearing unhandled trait: refused, both as diagnostics rows |
| `disqualified` | the demand's only candidate falls out of the always-unify rung: refused, the candidate named as disqualified |
| `mislabeled` | the demand's only candidate requires a different (non-string) label value: refused, the candidate's missing label named on the unmatched-components error |
| `warning` | an effectively-optional unhandled trait: renders with a warning |
| `unstated` | an unhandled trait whose posture the catalog never stated: refused as a build error naming `optional` |
| `incomplete` | a pair whose output never becomes concrete: refused at a path naming the pair |
| `failing` | a healthy pair beside a pair whose output conflicts: the failing pair is reported as data |
| `unlisted` | a resource demand for a contract no catalog lists or implements (authored inline): refused, the row carries no defining catalog |
| `unprovided` | on `platform_providers`: component `app` attaches the provider-fulfilled snapshot trait nothing provides, component `ledger` also declares the provider-fulfilled ledger resource nothing provides, beside a healthy sibling: refused with both rows marked unprovided, or, under `RenderInput.SkipUnprovided`, `app` renders, `ledger` is omitted and every skipped demand is a row |
| `provided_unmatched` | on `platform_providers`: the provider-fulfilled archive trait whose one provider requires a label the component lacks: refused, the row not marked unprovided, with or without the switch |
| `optional_unprovided` | on `platform_providers`: the snapshot trait made effectively optional at the attachment site: renders with an unhandled-trait entry and no skipped row, with or without the switch |
| `skipped_beside_refused` | on `platform_providers`: `app` attaches the unprovided snapshot trait, sibling `vault` the label-less archive trait: under the switch, refused on the archive row alone with the snapshot row readable as skipped |
| `omitted_refused` | on `platform_providers`: `ledger` is omitted for the unprovided ledger resource and also attaches the advisory sidecar, the catalog-fulfilled backup and the label-less archive trait, beside `web` with the sidecar: under the switch, only `web` stays on the unhandled-trait table, and backup and archive still refuse |

Consumed on-disk (subpackage acquisition through `Kernel.AcquireInstanceFromDir`);
never published; not discovered by the repo's CUE tasks.

## Platforms (`testdata/render/platform*`)

Each is its own CUE module in the D5 shape, importing the served catalogs:

| Directory | Carries | Exercises |
| --- | --- | --- |
| `platform` | cat 0.1.0 | the happy path and every scenario above not named on another platform |
| `platform_next` | cat 0.2.0 | older-than-platform skew (data, not a warning) |
| `platform_two` | cat 0.1.0 + cat2 0.1.0 | catalog-fulfilled plurality: two catalogs supply the container contract and every candidate matches |
| `platform_oversubscribed` | cat 0.1.0 + cat2 0.2.0 | the single-provider guard: two catalogs supply the provider-fulfilled gateway contract, refused in-build; the contract inventory reports it as over-subscribed |
| `platform_providers` | cat 0.1.0 + providers 0.1.0 | provider-fulfilled contracts: `providers` lists snapshot and ledger with no transformer (unprovided) and archive with one label-gated provider; kept out of cat so `platform` stays fulfilled |
| `platform_disabled` | cat 0.1.0 (disabled) + cat2 0.1.0 | a disabled catalog defines nothing: a demand for a contract only cat lists carries no defining catalog on its unresolved row |
| `platform_two_majors` | cat 0.1.0 + cat 1.0.0 | two majors of one catalog are two providers: both require the provider-fulfilled gateway contract, so the render refuses it and the contract inventory reports it over-subscribed |
| `platform_definer_disabled` | cat 0.1.0 (disabled) + cat2 0.2.0 + cat 1.0.0 | a disabled definer does not hide over-subscription: cat2 and cat 1.0.0 both require the gateway contract, so the render refuses it and the inventory reports it over-subscribed with no defining catalog |
