## 1. Remove opm/helper/objectset

- [x] 1.1 Delete `opm/helper/objectset/` and `opm/k8s/object/objectset_parity_test.go`; verify `go build ./... && go vet ./...` pass and `git grep helper/objectset -- '*.go'` is empty
- [x] 1.2 Remove the package from `AGENTS.md`, `README.md`, `CONSTITUTION.md`, `docs-kit.cue`, `.cascade-frozen`, `.golangci.yml` and the `opm/helper` and `opm/k8s/object` docs where they name it as present; verify with `git grep -n objectset` that only history (ADR-011, archives, api-diff fixtures) is left
- [x] 1.3 `task check` green, then commit `feat(helper)!: remove the deprecated objectset package`

## 2. One shape for typed errors (opm/errors, opm/kernel, opm/internal/loader)

- [x] 2.1 Add a test in `opm/errors` that every exported type with an `Error` method implements `error` on the pointer only, and loader tests that match `*IdentityError`; see them fail
- [x] 2.2 Give `IdentityError` a pointer receiver and return `&IdentityError{}` from the loader; the tests of 2.1 pass
- [x] 2.3 Add tests: `ValidateConfigDetailed` on bad values matches `*ConfigValidationError` through a `%w` wrap, `cueerrors.Errors` gives the same errors as on `Err`, the text is unchanged, and a source that does not compile is not marked; see them fail
- [x] 2.4 Add `errors.ConfigValidationError` and wrap in `ValidateConfigDetailed`; update the `opm/errors` and `ValidateConfigDetailed` docs; the tests of 2.3 pass
- [x] 2.5 `task check` green, then commit `feat(errors)!: give typed errors one shape and type the validation failure`

## 3. One name per type (opm/schema, opm/module, opm/platform, opm/catalog)

- [ ] 3.1 Move the four metadata type declarations from `opm/schema/metadata.go` to their artifact packages and delete the aliases; move their tests; verify `go build ./... && go vet ./...`
- [ ] 3.2 Delete `platform.Source` and `catalog.Source`; fields and signatures name `module.Source`; verify with `go doc` that neither package exports `Source`
- [ ] 3.3 `task check` green, then commit `feat(schema)!: keep one name for each metadata type and for Source`

## 4. Shrink the opm/schema exports

- [ ] 4.1 Create `opm/internal/corepath` with the twelve paths and their reader inventory; `opm/schema/paths.go` keeps `Metadata`, `Module`, `CatalogProvides`, `ProvidedBySince`, `ProvidesSince`; delete `CollisionsSince`; move every reader; verify `go build ./... && go vet ./...`
- [ ] 4.2 Add a test that pins the exported `cue.Path` variables of `opm/schema` to the three names (parse the package with `go/parser`); verify it fails when a fourth is added
- [ ] 4.3 `task check` green, then commit `feat(schema)!: export only the paths a consumer reads`

## 5. Record and verify

- [ ] 5.1 Write `adr/015-one-shape-for-the-public-surface.md` (receiver rule, validation marker, one name per type, path split) from `adr/TEMPLATE.md`; update `AGENTS.md` and `README.md` layout lines for `opm/errors`, `opm/schema` and `opm/internal/corepath`
- [ ] 5.2 Run `task api:diff` and check that it lists exactly the intended removals; run `task check`; run `openspec validate tidy-public-surface --strict`
- [ ] 5.3 Commit `docs(adr): record the public surface rules for v1`
