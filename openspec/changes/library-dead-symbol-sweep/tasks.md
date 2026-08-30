## 1. Verify the premise (all packages)

- [ ] 1.1 Re-run the reference sweep against current `main` of `library`, `cli` and `opm-operator`: for every identifier in proposal.md, confirm zero references outside its definition and tests (note the `oerrors` alias in both consumers resolves to their own `pkg/errors`).

## 2. opm/errors, opm/compile

- [ ] 2.1 Delete `opm/errors/sentinel.go`; remove `Wrap` from `opm/errors/errors.go` and its package doc mention; delete `opm/errors/errors_test.go` cases that only exercised them.
- [ ] 2.2 Remove `MaterializeKindCoreSchema` from `opm/errors/materialize.go` and reword the `Kind` doc to name the one kind; drop its test reference.
- [ ] 2.3 Remove the `ModuleResult` alias from `opm/compile/module.go`; delete `TestRender_ModuleResult_Aliased` in `opm/kernel/phase_test.go`.

## 3. opm/schema

- [ ] 3.1 Remove `DecodeProviderMetadata` from `decode.go` and `ProviderMetadata` from `metadata.go`; update the package doc in `doc.go`.
- [ ] 3.2 Remove the nine unused path variables from `paths.go` (`KnownResources`, `KnownTraits`, `ComposedTransformers`, `Matchers`, `MatchersResources`, `MatchersTraits`, `ContextModuleInstanceMetadata`, `ContextComponentMetadata`, `ContextRuntimeName`) and the comment block describing the platform views; update tests that referenced `ComposedTransformers` to build their lookups inline.
- [ ] 3.3 Delete `opm/schema/consts.go` (`AnnotationDefaultNamespace`).

## 4. opm/materialize, opm/helper

- [ ] 4.1 Delete `opm/materialize/cache/` (package and tests); remove the cache paragraph from `opm/materialize/doc.go` and the `Materialize` method doc in `opm/kernel/materialize.go` keeps the "kernel holds no cache" sentence.
- [ ] 4.2 Delete `opm/helper/loader/bytes/`; remove its bullet from `opm/helper/doc.go`.
- [ ] 4.3 In `opm/helper/loader/registry/module.go` remove `LoadModulePackage`; make `LoadModulePackageWithSource` the documented entry; retarget `module_test.go` / `module_internal_test.go` cases onto it.
- [ ] 4.4 Remove `Kernel.LoadModuleFromRegistry` from `opm/kernel/wrappers.go`; retarget `TestKernel_LoadModuleFromRegistry` in `registry_loader_test.go` onto `AcquireModuleFromRegistry` or delete it if `TestKernel_AcquireModuleFromRegistry` already covers the assertion; add a reflect-based negative test in the style of `TestKernel_NoFinalizeMethod`.

## 5. opm/compat

- [ ] 5.1 Delete `opm/compat/strip.go` and `strip_test.go`; move `provenanceDenylist` into `compat.go` (the walk still reads it); update the package doc in `level.go` and the `match.go` comment that cites the strip.

## 6. Docs

- [ ] 6.1 `MIGRATIONS.md`: `### Removed — library-dead-symbol-sweep` under `## Unreleased — Breaking`, one bullet per group with the replacement (or "none, no caller") named.
- [ ] 6.2 `CLAUDE.md`: drop `loader/bytes` and `materialize/cache` from the layout table; rewrite the "Materialize lifetime & registry contract" bullet that names the LRU; `README.md` layout likewise.
- [ ] 6.3 `docs/getting-started.md`: replace the `LoadModuleFromRegistry` mention with `AcquireModuleFromRegistry`.

## 7. Validation

- [ ] 7.1 `task fmt`, `task vet`, `task lint`, `task test` (including the `-race` leg).
- [ ] 7.2 `go doc -all` diff against `main` for each touched package: only the listed removals.
- [ ] 7.3 Build `cli` and `opm-operator` against this branch via a temporary `replace` (not committed): both compile unchanged.
