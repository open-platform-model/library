## 1. Glue: the unprovided and skipped verdicts (spike)

- [x] 1.1 Fixtures: in `testdata/render/registry/testing.opmodel.dev_library-render_cat_v0.1.0/catalog.cue`, add `#SnapshotTrait` (provider-fulfilled, `optional: bool | *false`, applies to the container resource, no transformer), `#LedgerResource` (provider-fulfilled, no transformer) and `#ArchiveTrait` (provider-fulfilled, `optional: bool | *false`, required by a new `archive-transformer` that also requires the label `render.test/archive: "on"`), each listed in the catalog's contract maps. Add scenario packages `unprovided` (component `app`: container plus snapshot; component `ledger`: container plus ledger; one healthy sibling) and `provided_unmatched` (archive attached without the label), and their rows in `testdata/render/scenarios/README.md`. Verify with `go test ./opm/kernel/... ./opm/platform/... ./opm/catalog/...`: every existing test passes, and any inventory assertion over this fixture that the new members change is updated. (Spike finding, see design.md: the members live in a new `providers` fixture catalog on a new `platform_providers` platform, not in `cat`, so the `platform` fixture's inventory stays fulfilled.)
- [x] 1.2 `opm/internal/renderstage/glue.go` and `stage.go`: add `GlueInputs.SkipUnprovided` and the template slot `_skipUnprovided: <<.SkipUnprovided>>` (rendered with `strconv.FormatBool`), and give `Stage` the switch (fold it and `localReplacements` into one options struct if that reads better). `Kernel.render` passes `false` for now. Verify with `TestRenderGlue_*`, extended to assert that the literal renders `true` and `false`.
- [x] 1.3 `opm/internal/renderstage/render.cue.tmpl`: add `#providers` (bound to `guard._providerSuppliers`) and `#skipUnprovided` to `#Match`; the `unprovided` field on both unresolved row builders; per component `_skipResources`, `_skipTraits`, `refusedResources`, `refusedTraits` and `omitted`; `pairs`, `unmatched`, `unresolved` and `warnings` reading the split; the new `skipped` list; `diagnostics.skipped`. Shape per design.md. Update the template's header and section comments to describe the verdict.
- [x] 1.4 Spike test `opm/internal/renderstage/skip_test.go`: stage and build the `unprovided` and `provided_unmatched` scenarios against `testdata/render/platform` with the switch on and off (registrytest serving the fixture registry, as `TestStageBuild_OverlayInstanceServedFromMemory` does). Assert on the built value:
  - with the switch on, `diagnostics.skipped` carries the snapshot row (not omitted) and the ledger row (omitted); `diagnostics.pairs` carries `app`'s pairs and none of `ledger`'s; `diagnostics.unmatched` does not name `ledger`; `gate` is `true`;
  - with it off, `gate` is an error, `skipped` is empty, and the snapshot and ledger rows are in `unresolved` with `unprovided: true`;
  - `provided_unmatched` keeps its archive row in `unresolved` with `unprovided: false` under both values;
  - the build is concrete, with no cycle, in every case.

  If an assumption in design.md fails (a cycle, or a `fulfilment` default that does not resolve in the marker's guard), write the finding and the chosen alternative into design.md before continuing.
- [x] 1.5 `go test ./opm/kernel/...` passes unchanged with the switch hard-wired off: the default path's rows decode with the new `unprovided` field present in the build. Then `task check` green, and commit `feat(renderstage): compute unprovided and skipped demands in the render glue`.

## 2. Kernel API: the switch, the skipped rows and the unprovided marker

- [ ] 2.1 `opm/errors/match.go`: add `UnresolvedDemand.Unprovided` with its doc comment. In `describe()`, append `; provider-fulfilled, no provider on this platform` to an unprovided row after the case text and before the disqualified count; every existing tail is unchanged. Verify with new cases in `opm/errors/match_test.go`: the suffix for each of the three case texts, its placement before `; <n> candidate(s) disqualified`, and no suffix when `Unprovided` is false.
- [ ] 2.2 `opm/kernel/render.go`: add `RenderInput.SkipUnprovided`, passed to `Stage`; add `SkippedDemand` and `RenderDiagnostics.Skipped`, and extend the `RenderDiagnostics` doc to name skipped demands as the third advisory fact. In `opm/kernel/render_decode.go`, decode `diagnostics.skipped` into `glueDiagnostics` and copy it onto the diagnostics as emitted. `gateErrors` is unchanged. Verify with `go build ./...` and `go vet ./...`.
- [ ] 2.3 `opm/kernel/render_skip_test.go`: one test per scenario in `specs/single-build-render/spec.md`, "A caller may skip unprovided provider-fulfilled demands":
  - switch off: refuses, and the snapshot row is marked;
  - snapshot skipped: `app` renders, and `Skipped` holds one trait row with `DefinedBy` set to the fixture catalog's registry key;
  - ledger component omitted: no `Compiled` entry for `ledger`, the healthy sibling renders, and the omission flag is set on every `ledger` row;
  - catalog-fulfilled `backup` in the existing `missing` scenario: still refuses under the switch;
  - `provided_unmatched`: still refuses;
  - `platform_oversubscribed`: still refuses under the switch;
  - `assertGateAgrees` under both switch values.

  Add a scenario to `render_definedby_test.go` for the unprovided message suffix. Verify with `go test ./opm/kernel/... -race`.
- [ ] 2.4 `task check` green, then commit `feat(kernel): let a render skip unprovided provider-fulfilled demands`.

## 3. Docs and the consumer check

- [ ] 3.1 `opm/kernel/doc.go`: describe the switch in the render contract, and list skipped demands with the other advisory rows in the frontend example. Update the `RenderInput` shape named in `AGENTS.md` (Kernel API surface, Render pipeline) and `README.md`. Verify `go doc ./opm/kernel` shows the text.
- [ ] 3.2 `docs/site/diagnostics/unresolved-demands.md`, keeping the page's placeholder-comment convention:
  - add the unprovided suffix to "The message";
  - in "The contract waits for a provider nobody installed", recognise the case by the suffix, and add the alternative fix: a caller may render without the contract through `RenderInput.SkipUnprovided`, which reports each skipped demand, and a frontend names its own switch;
  - add `library/opm/kernel/render.go` to the check-against list.
- [ ] 3.3 Consumer check, with nothing committed to either consumer: point a scratch copy of `cli` and of `opm-operator` at this branch (a temporary `go.work` or `go mod edit -replace` outside the repos' tracked files). Run `go build ./...` and `go vet ./...` in both, `go test ./internal/workflow/render/... ./internal/cmdutil/...` in cli, and `go test ./internal/render/...` in opm-operator. Record any consumer test that compares a whole `UnresolvedDemand` and now fails on the new field, as a report item for the consumer's bump. Then run `task cue:test:flow`.
- [ ] 3.4 `task check` green, then commit `docs(kernel): describe skipping unprovided provider-fulfilled demands`.
