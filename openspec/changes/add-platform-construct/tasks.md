## 1. Platform Type

- [ ] 1.1 Create `pkg/platform/` directory with `platform.go`
- [ ] 1.2 Define `Platform` struct: `{ APIVersion apiversion.Version; Metadata *PlatformMetadata; Package cue.Value }`
- [ ] 1.3 Define `PlatformMetadata` struct: `Name`, `Type`, `Description`, `Labels`, `Annotations`
- [ ] 1.4 Add godoc explaining the artifact's role and that `Package` is source of truth

## 2. Constructor

- [ ] 2.1 Implement `func NewPlatformFromValue(k *kernel.Kernel, v cue.Value) (*Platform, error)` in `pkg/platform/platform.go`
- [ ] 2.2 Steps: detect `apiVersion`, look up binding, decode metadata, stamp `APIVersion`, set `Package`
- [ ] 2.3 Error path: unknown apiVersion wraps `apiversion.ErrUnknownAPIVersion`
- [ ] 2.4 Error path: malformed metadata returns a wrapped error

## 3. Binding Extensions — v1alpha2

- [ ] 3.1 Extend `pkg/api/v1alpha2/paths.go` (or equivalent) with `Registry`, `KnownResources`, `KnownTraits`, `ComposedTransformers`, `Matchers` paths
- [ ] 3.2 Implement `DecodePlatformMetadata(v cue.Value) (*platform.PlatformMetadata, error)` in v1alpha2 binding
- [ ] 3.3 Update the binding's `Binding` interface (in `pkg/api/`) to include the new paths and decoder
- [ ] 3.4 Add unit tests for each new path: load a Platform fixture, navigate via the path, assert non-empty result

## 4. Loader Helper

- [ ] 4.1 Create `pkg/helper/loader/file/platform.go` with `LoadPlatformFile(ctx *cue.Context, path string, opts LoadOptions) (cue.Value, string, error)`
- [ ] 4.2 Mirror `LoadReleaseFile` shape; reuse the same `LoadOptions`; resolve directory-or-file path
- [ ] 4.3 Add unit tests using a fixture `platform.cue`

## 5. Kernel Wrapper

- [ ] 5.1 Add `(k *Kernel) LoadPlatformFile(ctx context.Context, path string, opts loader.LoadOptions) (cue.Value, string, error)` to `pkg/kernel/`
- [ ] 5.2 Add `(k *Kernel) NewPlatformFromValue(v cue.Value) (*Platform, error)` thin wrapper

## 6. Phase Input Struct Extension

- [ ] 6.1 Add `Platform *platform.Platform` field to `MatchInput`, `PlanInput`, `CompileInput` (defined in slice 06)
- [ ] 6.2 Add godoc on each field stating: "Optional today; becomes required when slice 09 (`rewrite-match-around-platform`) lands."
- [ ] 6.3 Confirm slice-06 phase methods continue to work with `Provider` while ignoring `Platform` for now

## 7. Tests

- [ ] 7.1 Create `library/testdata/platform/v1alpha2/` with at least one fixture `platform.cue` file
- [ ] 7.2 `pkg/platform/platform_test.go` covering: successful construction, unknown apiVersion path, metadata decode path
- [ ] 7.3 `pkg/helper/loader/file/platform_test.go` covering file path and directory path inputs
- [ ] 7.4 `pkg/api/v1alpha2/` tests for new paths and decoder

## 8. Documentation

- [ ] 8.1 Update `library/README.md` to mention `Platform` as an artifact type
- [ ] 8.2 Cross-reference catalog enhancement 014 from `pkg/platform/doc.go`
- [ ] 8.3 CHANGELOG entry: new `Platform` type; loader helper; binding extensions; `Platform` becomes required after slice 09

## 9. Validation

- [ ] 9.1 Run `task fmt`
- [ ] 9.2 Run `task vet`
- [ ] 9.3 Run `task lint`
- [ ] 9.4 Run `task test`
- [ ] 9.5 Run `task check`
