# Tasks: fix-duplicate-identity-and-schema-error-memo

Worktree `library/.claude/worktrees/fix-duplicate-identity-and-schema-error-memo`, branch
`fix/duplicate-identity-and-schema-error-memo` (from `origin/main`). `.cue-cache` was seeded by
copying the main checkout's, never symlinking. Three independent sections, each green on its own;
design.md carries no unverified assumption (its probe is recorded under Research & Decisions), so
section 1 is not a spike. Code comments describe each rule in their own words and never cite
design.md's DM numbers.
If `TestGenerate_BuildsThroughTheKernel` fails in a full run, re-run it alone: it is the known
cross-process cache race, not this change.

## 1. Duplicate identities keyed on group (opm/helper/objectset)

- [x] 1.1 In `opm/helper/objectset/objectset.go`, key `Duplicates` on an unexported
  `{group, kind, namespace, name}` with `groupOf` per design.md DM1; a row's `Identity` stays the
  first-rendered object's (exported `Identity` and its `String()` unchanged). Verify:
  `go build ./opm/helper/...`.
- [x] 1.2 Add `APIVersion string` to `Producer`, set from each object's `apiVersion`; in
  `DuplicateIdentitiesError.Error()` append ` as <apiVersion>` to each producer only when a row's
  producers do not all share one `APIVersion` (DM2). `Producer.String()` unchanged.
- [x] 1.3 `objectset_test.go`: update existing expectations for the new `Producer.APIVersion`;
  add `apps/v1` + `apps/v1beta2` Deployment `web` (one row, identity `apps/v1`, producers carry
  their own versions), the same kind and name in two groups (no rows), a core-group `v1` pair
  (one row), a `v1` ConfigMap `x` beside a ConfigMap `x` with no `apiVersion` (one row, the
  message wording the second producer `as <no apiVersion>`), and the message naming both versions for the mixed row while a same-version row's
  message is byte-identical to today's. Verify: `go test ./opm/helper/objectset/...`.
- [x] 1.4 Say "group, kind, namespace and name" where the identity is spelled as four fields:
  the `Identity` doc comment in `opm/helper/objectset/objectset.go`, `opm/helper/objectset/doc.go`
  (the opening "same apiVersion, kind, namespace and name" sentence) and `README.md` (the
  `objectset` line).
  Verify: `go doc github.com/open-platform-model/library/opm/helper/objectset` reads correctly.
- [x] 1.5 `task check` green, then commit `fix(helper): key duplicate identities on api group`.

## 2. Errored schema builds are load failures (opm/schema)

- [x] 2.1 In `opm/schema/loader.go` `loadVersioned`, return the zero value, `""` and an error
  naming the module when `val.Err()` is non-nil (DM3). Say in the `Cache` doc comment in
  `cache.go` that an errored build is memoised as an error whichever Loader produced it (DM4).
- [x] 2.2 `loader_test.go`: serve a stand-in core with an unresolved reference through
  `registrytest.NewRegistryWithCore` and assert `OCILoader{}.Load` returns the zero value and an
  error naming the module. `cache_test.go`: a `Cache` over that loader returns the same error on
  two `Get` calls and `ResolvedVersion()` is `""`. Verify: `go test ./opm/schema/...`, and
  `TestOCILoader_ZeroValueResolvesDefault` still passes against the real core.
- [x] 2.3 `task check` green, then commit `fix(schema): refuse a core module whose build fails`.
- [x] 2.4 In `Cache.Get`, after the load, turn a nil error with an errored value from any Loader
  into the zero value, `""` and a wrapped build error (DM4). `cache_test.go`: a stub Loader
  returning `x: 1 & 2` yields the same error on two `Get` calls, one Load, and an empty
  `ResolvedVersion()`. Verify: `go test ./opm/schema/...`, then commit
  `fix(schema): refuse an errored schema from any loader`.

## 3. Cold concurrent renders on a shared platform (opm/kernel, tests only)

- [x] 3.1 In `opm/kernel/render_core_floor_test.go`, add
  `TestRender_SharedPlatformConcurrentRendersCold` per design.md DM5: eight goroutines render a
  never-rendered platform and instance as the test's first renders; after `wg.Wait()` a separately
  acquired pair on the same kernel is rendered as the baseline, and every render matches it. Its comment cites the
  single-build-render scenario "A platform shared by concurrent renders stays race-free" like the
  warm test's does. Verify: `go test -race -run 'TestRender_SharedPlatformConcurrentRenders' ./opm/kernel/`.
- [x] 3.2 `task check` green, then commit `test(kernel): race the first render on a shared platform`.
