## Why

Three small gaps found by the alpha.35 kernel review, each owner-decided on 2026-10-02/03.

- **A duplicate the helper cannot see.** `objectset.Duplicates` keys rendered objects on
  `Identity{APIVersion, Kind, Namespace, Name}` (`opm/helper/objectset/objectset.go:15-20,56-85`).
  Kubernetes addresses an object by group, kind, namespace and name; the version is only the
  serialization. So `apps/v1` and `apps/v1beta2` Deployment `web` are one object to the API server
  but two keys to the helper, nothing is reported, and the later apply silently overwrites the
  earlier one: the exact failure the helper exists to refuse.
- **An errored schema build is memoised as success.** `OCILoader.loadVersioned`
  (`opm/schema/loader.go:166-168`) returns `ctx.BuildInstance(...)` without checking `val.Err()`.
  A core module that loads but does not build (an unresolved reference, a conflict) comes back as
  a value with a nil error, and `schema.Cache` memoises it as a success. The operator's start-up
  `verifyCoreSchema` then passes an errored schema, and the CLI's `mod vet` and publish report a
  misleading "does not define #IdentityPackage" instead of the build error.
- **The shared-platform race test only runs warm.** `TestRender_SharedPlatformConcurrentRenders`
  (`opm/kernel/render_core_floor_test.go:109-141`) renders once before starting its goroutines, so
  the first core-floor lookup on the shared platform `Package` is never raced. Cold concurrent
  renders passed under `-race` by hand during the review; nothing pins that.

## What Changes

- **`opm/helper/objectset`: duplicates keyed on group.** `Duplicates` groups objects internally on
  group (the `apiVersion` before `/`, empty for the core group), kind, namespace and name. The
  exported `Identity` keeps its shape and fields; a row's `Identity` is the first-rendered
  object's, its `apiVersion` verbatim. `Producer` gains an additive `APIVersion` field carrying
  each producing object's own `apiVersion`, and `DuplicateIdentitiesError` names each producer's
  version when a row's producers disagree on it, so the refusal shows the version mismatch that
  caused it. A row whose producers share one version reads exactly as today.
- **`opm/schema`: an errored build is a load failure.** `OCILoader` returns the zero value, no
  resolved version and a wrapped error naming the module when the built value carries an error.
  `Cache.Get` also refuses an errored value any other `Loader` returns with a nil error, and
  stays never-retry: it memoises that error, never an errored value.
- **`opm/kernel` (tests only): a cold concurrent-render test.** A sibling of
  `TestRender_SharedPlatformConcurrentRenders` that skips the warm-up render, so the goroutines
  race the platform's first floor lookup, compared against a baseline rendered from a separately
  acquired platform and instance. Run under `-race` by `task test`.

Not in this change: the operator half of the same finding (a fresh `ModFileSource` per Platform
reconcile), which is opm-operator's own change; a retryable schema `Cache`, which the owner
declined; changing the exported `Identity` struct.

## Classification

**PATCH** (Principle VI), released as the next `-beta.N`: `fix` and `test` commits.
Nothing exported is removed or retyped. `Producer.APIVersion` is an additive field, MINOR in kind
under Principle VI, carried by the `fix(helper)` commit because it exists only to word the fixed
refusal; both downstream test literals of `Producer` are keyed, so they keep compiling. Behaviour tightens in
two places a caller can observe: `Duplicates` now returns a row for objects that differ only in
version within one group, and `OCILoader.Load` / `Cache.Get` return an error for a schema whose
build fails. Both are the defects being fixed. No complexity is added beyond one string split and
one error check (Principle VII).

PR title: `fix: key duplicate identities on group and refuse errored schema builds`.

## Downstream consumers

- **`cli`**: a library pin bump only. Its duplicate refusal (`internal/workflow/render`) gains the
  group-keyed rows; `mod vet` and publish surface the real build error of a broken core.
- **`opm-operator`**: a library pin bump only. Its render refusal gains the group-keyed rows and
  `verifyCoreSchema` fails fast on an errored core build. Its `ModFiles` memo fix is separate.
- **`catalog_opm`**, **`modules`**: nothing.

## Capabilities

### Modified Capabilities

- `duplicate-object-identities`: identities match on group, not apiVersion; producers carry their
  own apiVersion and the refusal names mixed versions.
- `schema-dispatch`: `OCILoader` fails an errored build; the `Cache` memoises that failure as an
  error.

`single-build-render` is not modified: its scenario "A platform shared by concurrent renders stays
race-free" already states the behaviour; the cold test only makes the suite check it from the
first render.

## Impact

- `opm/helper/objectset/objectset.go`, `objectset_test.go`, `doc.go`; `README.md` where it
  spells the identity as four fields.
- `opm/schema/loader.go`, `loader_test.go`, `cache.go` (doc comment), `cache_test.go`.
- `opm/kernel/render_core_floor_test.go`.
