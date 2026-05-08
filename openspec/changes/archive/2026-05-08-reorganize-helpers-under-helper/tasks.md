## 1. Helper Boundary Documentation

- [x] 1.1 Create `pkg/helper/doc.go` with a package comment that defines the helper boundary, lists current subpackages, and explicitly states what is in/out of scope
- [x] 1.2 Cross-reference the umbrella enhancement and slice 07 in the doc

## 2. Move Loader to helper/loader/file

- [x] 2.1 `git mv pkg/loader/module.go pkg/helper/loader/file/module.go`
- [x] 2.2 `git mv pkg/loader/provider.go pkg/helper/loader/file/provider.go`
- [x] 2.3 `git mv pkg/loader/release.go pkg/helper/loader/file/release.go`
- [x] 2.4 Change package declarations from `package loader` to `package file`
- [x] 2.5 Update internal imports (within the moved files) — none expected, but verify
- [x] 2.6 Move any loader tests `pkg/loader/*_test.go` similarly
- [x] 2.7 Verify symbols (`LoadModulePackage`, `LoadReleaseFile`, `LoadValuesFile`, `LoadProvider`, `LoadOptions`) compile under the new package

## 3. Skeleton helper/loader/bytes

- [x] 3.1 Create `pkg/helper/loader/bytes/doc.go` with a package comment describing intent (in-memory loading; Crossplane fn / tests / fuzzing target)
- [x] 3.2 Reference the slice that will implement it (TBD, follow-up)

## 4. Deprecation Shim at pkg/loader

- [x] 4.1 Create `pkg/loader/loader.go` (or per-function shim files) that imports `pkg/helper/loader/file` and re-exports each function as a `// Deprecated:` thin wrapper
- [x] 4.2 Re-export `LoadOptions` as a type alias: `type LoadOptions = file.LoadOptions`
- [x] 4.3 Confirm every previously exported symbol has a shim
- [x] 4.4 Add a `// Deprecated:` package-level comment in `pkg/loader/doc.go`

## 5. Update Kernel Wrappers

- [x] 5.1 Update `pkg/kernel/` wrapper methods (slice 01) to call `pkg/helper/loader/file` directly, not the deprecated `pkg/loader/`
- [x] 5.2 Confirm wrapper godoc points to the new path

## 6. Update Internal Callers

- [x] 6.1 `grep -rn "pkg/loader" pkg/` and update internal imports to `pkg/helper/loader/file`
- [x] 6.2 Confirm only the shim itself imports the old path indirectly (via re-export)

## 7. Test Migration

- [x] 7.1 Update test imports across `pkg/` to the new path
- [x] 7.2 Add a parity test that confirms calling through the deprecated shim produces results identical to calling the new path
- [x] 7.3 Confirm all existing fixtures pass

## 8. Documentation

- [x] 8.1 CHANGELOG entry: import-path move; deprecation shim and removal schedule; migration recipe (one-line import path change)
- [x] 8.2 Update `library/README.md` import examples
- [x] 8.3 Update umbrella enhancement (`02-design.md`) to confirm the helper layout matches what shipped

## 9. Validation

- [x] 9.1 Run `task fmt`
- [x] 9.2 Run `task vet`
- [x] 9.3 Run `task lint`
- [x] 9.4 Run `task test`
- [x] 9.5 Run `task check`
- [x] 9.6 `go build ./...` from repo root
