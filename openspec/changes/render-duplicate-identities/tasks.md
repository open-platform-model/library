# Tasks: render-duplicate-identities

Two sections per design.md. design.md carries no unverified assumption: the identity fields are read off values the kernel already returns, and the helper boundary is the established convention, so section 1 is not a spike.

## 1. The objectset helper (opm/helper/objectset)

- [ ] 1.1 Add `opm/helper/objectset/objectset.go` with `Identity` (and `String()`), `Producer`, `Duplicate`, `Duplicates` and `*DuplicateIdentitiesError` per design.md § The package: fields read by `LookupPath`, a missing `kind` or `metadata.name` skips the object, first-seen order, producers in render order. Verify: `go build ./opm/helper/...` and `go vet ./opm/helper/...` pass and `task lint` shows the depguard rule green (nothing outside `opm/helper/` imports it).
- [ ] 1.2 `objectset_test.go` over hand-built `*kernel.Compiled` values (`cuecontext.New().CompileString`): two components rendering one cluster-scoped `TransformerRegistration` (one row, empty namespace, two producers); a Deployment twice beside a Service once (one row); a clean render (no rows); a nameless value beside a duplicate (skipped); the error message naming the identity once and both components with their transformers; and two renders in different pair order producing rows in their own first-seen order. Verify: `go test ./opm/helper/objectset/...` passes.
- [ ] 1.3 Add `opm/helper/objectset/doc.go` stating the contract (reads four fields, skips non-objects, never called by the kernel, where a runtime calls it), and list the subpackage in `opm/helper/doc.go` and `CLAUDE.md` § Repository Layout. Verify: `go doc github.com/open-platform-model/library/opm/helper/objectset` renders the contract.
- [ ] 1.4 `task check` green, then commit `feat(helper): detect duplicate rendered object identities`.

## 2. The healthy path and the consumers

- [ ] 2.1 In `opm/kernel/flow_integration_test.go`, after the shipped fixture renders, assert `objectset.Duplicates(res.Compiled)` is empty, so the published catalog's outputs are pinned distinct. Verify: `task cue:test:flow` passes (skips offline, `OPM_FLOW_TEST_FORCE=1` requires it).
- [ ] 2.2 Cross-cutting: build `cli` and `opm-operator` against this tree with a `replace` in a scratch copy of each `go.mod` (not committed) and confirm each can import `opm/helper/objectset` and call `Duplicates` on its render result at the site design.md § Where it runs names (a throwaway edit, not committed). Verify: both build; the two call sites are recorded in the proposal's downstream section by file and line.
- [ ] 2.3 `task check` green, then commit `test(kernel): pin the shipped fixture free of duplicate object identities`.
