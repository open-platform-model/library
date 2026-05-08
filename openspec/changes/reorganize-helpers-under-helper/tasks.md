## 1. Helper Boundary Documentation

- [ ] 1.1 Create `pkg/helper/doc.go` with a package comment that defines the helper boundary, lists current subpackages, and explicitly states what is in/out of scope
- [ ] 1.2 Cross-reference the umbrella enhancement and slice 07 in the doc

## 2. Move Loader to helper/loader/file

- [ ] 2.1 `git mv pkg/loader/module.go pkg/helper/loader/file/module.go`
- [ ] 2.2 `git mv pkg/loader/provider.go pkg/helper/loader/file/provider.go`
- [ ] 2.3 `git mv pkg/loader/release.go pkg/helper/loader/file/release.go`
- [ ] 2.4 Change package declarations from `package loader` to `package file`
- [ ] 2.5 Update internal imports (within the moved files) — none expected, but verify
- [ ] 2.6 Move any loader tests `pkg/loader/*_test.go` similarly
- [ ] 2.7 Verify symbols (`LoadModulePackage`, `LoadReleaseFile`, `LoadValuesFile`, `LoadProvider`, `LoadOptions`) compile under the new package

## 3. Skeleton helper/loader/bytes

- [ ] 3.1 Create `pkg/helper/loader/bytes/doc.go` with a package comment describing intent (in-memory loading; Crossplane fn / tests / fuzzing target)
- [ ] 3.2 Reference the slice that will implement it (TBD, follow-up)

## 4. Deprecation Shim at pkg/loader

- [ ] 4.1 Create `pkg/loader/loader.go` (or per-function shim files) that imports `pkg/helper/loader/file` and re-exports each function as a `// Deprecated:` thin wrapper
- [ ] 4.2 Re-export `LoadOptions` as a type alias: `type LoadOptions = file.LoadOptions`
- [ ] 4.3 Confirm every previously exported symbol has a shim
- [ ] 4.4 Add a `// Deprecated:` package-level comment in `pkg/loader/doc.go`

## 5. Update Kernel Wrappers

- [ ] 5.1 Update `pkg/kernel/` wrapper methods (slice 01) to call `pkg/helper/loader/file` directly, not the deprecated `pkg/loader/`
- [ ] 5.2 Confirm wrapper godoc points to the new path

## 6. Update Internal Callers

- [ ] 6.1 `grep -rn "pkg/loader" pkg/` and update internal imports to `pkg/helper/loader/file`
- [ ] 6.2 Confirm only the shim itself imports the old path indirectly (via re-export)

## 7. Test Migration

- [ ] 7.1 Update test imports across `pkg/` to the new path
- [ ] 7.2 Add a parity test that confirms calling through the deprecated shim produces results identical to calling the new path
- [ ] 7.3 Confirm all existing fixtures pass

## 8. Documentation

- [ ] 8.1 CHANGELOG entry: import-path move; deprecation shim and removal schedule; migration recipe (one-line import path change)
- [ ] 8.2 Update `library/README.md` import examples
- [ ] 8.3 Update umbrella enhancement (`02-design.md`) to confirm the helper layout matches what shipped

## 9. Validation

- [ ] 9.1 Run `task fmt`
- [ ] 9.2 Run `task vet`
- [ ] 9.3 Run `task lint`
- [ ] 9.4 Run `task test`
- [ ] 9.5 Run `task check`
- [ ] 9.6 `go build ./...` from repo root
