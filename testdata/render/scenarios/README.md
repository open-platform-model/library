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
| `warning` | an effectively-optional unhandled trait: renders with a warning |
| `unstated` | an unhandled trait whose posture the catalog never stated: refused as a build error naming `optional` |
| `incomplete` | a pair whose output never becomes concrete: refused at a path naming the pair |
| `failing` | a healthy pair beside a pair whose output conflicts: the failing pair is reported as data |

Consumed on-disk (subpackage acquisition through `Kernel.AcquireInstanceFromDir`);
never published; not discovered by the repo's CUE tasks.
