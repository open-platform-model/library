# Render scenario instances

One CUE module, one instance package per matching or execution outcome the
single-build render tests exercise (`opm/kernel/render_test.go`). Each package
is a self-contained `#ModuleInstance` whose `#module` is authored inline
against the render fixture catalog, so the outcome under test is the only
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

Consumed on-disk (subpackage acquisition through `Kernel.AcquireInstanceFromDir`);
never published; not discovered by the repo's CUE tasks.

## Platforms (`testdata/render/platform*`)

Each is its own CUE module in the D5 shape, importing the served catalogs:

| Directory | Carries | Exercises |
| --- | --- | --- |
| `platform` | cat 0.1.0 | the happy path and every scenario above |
| `platform_next` | cat 0.2.0 | older-than-platform skew (data, not a warning) |
| `platform_two` | cat 0.1.0 + cat2 0.1.0 | catalog-fulfilled plurality: two catalogs supply the container contract and every candidate matches |
| `platform_oversubscribed` | cat 0.1.0 + cat2 0.2.0 | the single-provider guard: two catalogs supply the provider-fulfilled gateway contract, refused in-build |
