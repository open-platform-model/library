## 1. api: extend Binding interface with SchemaValue

- [x] 1.1 Add `SchemaValue(ctx *cue.Context) (cue.Value, error)` to the `Binding` interface in `opm/api/api.go` with godoc covering caching contract, concurrency, and "fatal on error" guidance
- [x] 1.2 Add unit test in `opm/api/registry_test.go` (or a new file) that asserts the interface has the method via compile-time assignment to a typed nil

## 2. api/v1alpha2: implement SchemaValue on the binding

- [x] 2.1 Convert `binding` in `opm/api/v1alpha2/binding.go` from a zero-sized struct to a pointer-receivable struct that can hold a `sync.Once`, cached `cue.Value`, and cached `error`
- [x] 2.2 Update the binding registration to use the pointer so the cache survives across calls
- [x] 2.3 Implement `SchemaValue` using `load.Instances` over `schema.Schema` (the embed.FS already returned by `EmbeddedSchema`); load the package at `apis/core/v1alpha2`
- [x] 2.4 Write `opm/api/v1alpha2/binding_test.go` cases: repeated calls cache; result exposes `#ModuleRelease`; concurrent first calls coalesce to one load (goroutine race detector clean); environments without `CUE_REGISTRY` succeed
- [x] 2.5 Add a deliberately-broken-embed test (in a sibling file using a fake binding) that asserts schema-load errors are wrapped and cached

## 3. helper/synth: package skeleton and doc

- [x] 3.1 Create `opm/helper/synth/doc.go` documenting purpose, helper-boundary peer relationship to `opm/helper/loader/`, and "Kernel.SynthesizeRelease is the recommended entry point" note
- [x] 3.2 Create `opm/helper/synth/release.go` with the `ReleaseInput` struct and the `Release(ctx, in) (cue.Value, error)` function stub returning a "not yet implemented" error

## 4. helper/synth: implement Release

- [x] 4.1 Implement required-field validation in `synth.Release` (Module, Name, Namespace) with explicit error messages
- [x] 4.2 Resolve binding via `api.Lookup(in.Module.APIVersion)`, fetch `#ModuleRelease` via `binding.SchemaValue(ctx).LookupPath("#ModuleRelease")`, error on missing definition
- [x] 4.3 Fill `metadata.name`, `metadata.namespace`, and `paths.Module` on the schema definition value
- [x] 4.4 Conditionally fill `metadata.labels`, `metadata.annotations`, and `paths.Values` only when caller supplied non-empty / non-zero values
- [x] 4.5 Return `spec, spec.Err()` to surface any unification error without invoking concreteness checks

## 5. helper/synth: tests

- [x] 5.1 Add `opm/helper/synth/release_test.go` covering: required-input rejection (nil Module, empty Name, empty Namespace) → typed errors
- [x] 5.2 Test: returned value carries `apiVersion`, `kind`, and a `metadata.uuid` matching the expected `uuid.SHA1(OPMNamespace, "<uuid>:<name>:<namespace>")`
- [x] 5.3 Test: changing only `Namespace` produces a different UUID; identical inputs produce identical UUIDs
- [x] 5.4 Test: caller-supplied `Labels` coexist with schema-stamped `module-release.opmodel.dev/{name,uuid}` labels
- [x] 5.5 Test: empty `Values` leaves the schema's values path unfilled (no debugValues fallback)
- [x] 5.6 Test: a module with at least one `#Secret` instance produces an `opm-secrets` component in the synthesized release
- [x] 5.7 Test: no `CUE_REGISTRY` env, no filesystem read — `synth.Release` succeeds against an embedded schema

## 6. kernel: add SynthesizeRelease wrapper

- [x] 6.1 Add `opm/kernel/synth.go` (or extend `wrappers.go`) with `func (k *Kernel) SynthesizeRelease(ctx context.Context, in synth.ReleaseInput) (*module.Release, error)` chaining `synth.Release` into `Kernel.ProcessModuleRelease`
- [x] 6.2 Godoc the method as the recommended in-memory entry point; cross-link to `synth.Release` for the no-Kernel path
- [x] 6.3 Add tests in `opm/kernel/kernel_test.go` (or a new `synth_test.go`): happy path against an embedded fixture module; unconcrete result rejected; nil Module errors before validation runs

## 7. integration: end-to-end coverage

- [x] 7.1 Extend `opm/kernel/flow_integration_test.go` (or add a sibling synth integration test) with a path that uses `Kernel.SynthesizeRelease` instead of the hand-rolled `CompileString` skeleton
- [x] 7.2 Assert the synthesized release passes Match → Compile and produces output equivalent to the file-loaded equivalent release for the same inputs

## 8. docs and changelog

- [x] 8.1 Add a CHANGELOG entry under the next unreleased MAJOR section: new `Binding.SchemaValue`, new `opm/helper/synth/`, new `Kernel.SynthesizeRelease`
- [x] 8.2 Update `opm/helper/doc.go` (if it enumerates subpackages) to mention `synth/`

## 9. validation gates

- [x] 9.1 `task fmt`
- [x] 9.2 `task vet`
- [x] 9.3 `task lint`
- [x] 9.4 `task test`
- [x] 9.5 `openspec validate add-release-synth-helper --strict`
