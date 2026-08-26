## 1. Fill `#moduleInstance` (opm/schema, opm/compile)

- [x] 1.1 `opm/schema/paths.go`: add `ModuleInstance = cue.ParsePath("#moduleInstance")` to the "Inside #transform" group with a doc line naming it as the third `#transform` input.
- [x] 1.2 `executePair`: guard `inst.Package.Exists()` (design D2), fill `schema.ModuleInstance` with `inst.Package` before the `#component` fill, wrap the error as `filling #moduleInstance`; extend the function comment's flow list.

## 2. Regression test (opm/kernel, hermetic)

- [x] 2.1 Add `opm/kernel/instance_fill_test.go` (design D3): registrytest catalog whose transformer emits `#moduleInstance.metadata.name`, `.metadata.namespace` and `#moduleInstance.components[#component.metadata.name].metadata.name`; import-authored instance in a `t.TempDir()` module; assert the concrete values. Run it against the pre-fill tree first and record that it fails.

## 3. Parity harness bookkeeping (opm/kernel, testdata/parity)

- [x] 3.1 Run `TestParity_Probes`; delete `ExpectedDivergence` on the `instance-probe :: web` row when the harness reports it no longer reproduces, and update the row comment.
- [x] 3.2 `testdata/parity/oracle/render.cue`: correct the Phase 3 comment ("which the kernel has never filled") and the `parity_probe_test.go` file comment so neither says the kernel leaves `#moduleInstance` unfilled.

## 4. Docs

- [x] 4.1 `MIGRATIONS.md`: `### Changed — library-instance-fill` under `## Unreleased — Additive` (third input now filled; additive).
- [x] 4.2 `CLAUDE.md` / `README.md`: if either lists what `executePair` fills or names `#moduleInstance` as unfilled, update the line.

## 5. Validation

- [x] 5.1 `task fmt`, `task vet`, `task lint`, `task test` (includes `composed_open_test.go`, the CUE canary pair and the full parity harness against GHCR).
- [x] 5.2 Confirm the only exported change under `opm/` is the new `schema.ModuleInstance` constant (`go doc` diff against `main`).
