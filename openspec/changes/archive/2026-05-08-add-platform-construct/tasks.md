## 1. Platform Type

- [x] 1.1 Create `opm/platform/` directory with `platform.go`
- [x] 1.2 Define `Platform` struct: `{ APIVersion apiversion.Version; Metadata *PlatformMetadata; Package cue.Value }`
- [x] 1.3 Define `PlatformMetadata` struct: `Name`, `Type`, `Description`, `Labels`, `Annotations`
- [x] 1.4 Add godoc explaining the artifact's role and that `Package` is source of truth

## 2. Constructor

- [x] 2.1 Implement `func NewPlatformFromValue(k *kernel.Kernel, v cue.Value) (*Platform, error)` in `opm/platform/platform.go`
- [x] 2.2 Steps: detect `apiVersion`, look up binding, decode metadata, stamp `APIVersion`, set `Package`
- [x] 2.3 Error path: unknown apiVersion wraps `apiversion.ErrUnknownAPIVersion`
- [x] 2.4 Error path: malformed metadata returns a wrapped error

## 3. Binding Extensions — v1alpha2

- [x] 3.1 Extend `opm/api/v1alpha2/paths.go` (or equivalent) with `Registry`, `KnownResources`, `KnownTraits`, `ComposedTransformers`, `Matchers` paths
- [x] 3.2 Implement `DecodePlatformMetadata(v cue.Value) (*platform.PlatformMetadata, error)` in v1alpha2 binding
- [x] 3.3 Update the binding's `Binding` interface (in `opm/api/`) to include the new paths and decoder
- [x] 3.4 Add unit tests for each new path: load a Platform fixture, navigate via the path, assert non-empty result

## 4. Loader Helper

- [x] 4.1 Create `opm/helper/loader/file/platform.go` with `LoadPlatformFile(ctx *cue.Context, path string, opts LoadOptions) (cue.Value, string, error)`
- [x] 4.2 Mirror `LoadReleaseFile` shape; reuse the same `LoadOptions`; resolve directory-or-file path
- [x] 4.3 Add unit tests using a fixture `platform.cue`

## 5. Kernel Wrapper

- [x] 5.1 Add `(k *Kernel) LoadPlatformFile(ctx context.Context, path string, opts loader.LoadOptions) (cue.Value, string, error)` to `opm/kernel/`
- [x] 5.2 Add `(k *Kernel) NewPlatformFromValue(v cue.Value) (*Platform, error)` thin wrapper

## 6. Phase Input Struct Extension

- [x] 6.1 Add `Platform *platform.Platform` field to `MatchInput`, `PlanInput`, `CompileInput` (defined in slice 06)
- [x] 6.2 Add godoc on each field stating: "Optional today; becomes required when slice 09 (`rewrite-match-around-platform`) lands."
- [x] 6.3 Confirm slice-06 phase methods continue to work with `Provider` while ignoring `Platform` for now

## 7. Tests

- [x] 7.1 Create `library/testdata/platform/v1alpha2/` with at least one fixture `platform.cue` file
- [x] 7.2 `opm/platform/platform_test.go` covering: successful construction, unknown apiVersion path, metadata decode path
- [x] 7.3 `opm/helper/loader/file/platform_test.go` covering file path and directory path inputs
- [x] 7.4 `opm/api/v1alpha2/` tests for new paths and decoder

## 8. Documentation

- [x] 8.1 Update `library/README.md` to mention `Platform` as an artifact type
- [x] 8.2 Cross-reference catalog enhancement 014 from `opm/platform/doc.go`
- [x] 8.3 CHANGELOG entry: new `Platform` type; loader helper; binding extensions; `Platform` becomes required after slice 09

## 9. Validation

- [x] 9.1 Run `task fmt`
- [x] 9.2 Run `task vet`
- [x] 9.3 Run `task lint`
- [x] 9.4 Run `task test`
- [x] 9.5 Run `task check`
