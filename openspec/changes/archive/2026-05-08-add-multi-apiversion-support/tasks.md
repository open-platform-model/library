> **Scope note (Constitution VIII).** This task list is large because it spans multiple `pkg/` packages and the embedded schemas. Each numbered group corresponds to one independently mergeable PR and stays within the small-batch envelope individually. Consider splitting into two follow-up OpenSpec changes after group 4 lands if downstream coordination needs more lead time.

## 1. Schema literal pin (PR 1)

- [x] 1.1 Replace `#ApiVersion: #ApiVersion` in `apis/core/v1alpha2/types.cue` with `#ApiVersion: "opmodel.dev/v1alpha2"`.
- [x] 1.2 Add a CUE-side smoke test under `apis/core/v1alpha2/` (or document existing coverage) asserting an evaluated `#Module` carries the literal in `apiVersion`.
- [x] 1.3 Run `task fmt`, `task vet`, `task test` to confirm no regression in existing v1alpha2 schema callers.

## 2. `pkg/apiversion` package (PR 2)

- [x] 2.1 Create `pkg/apiversion/apiversion.go` declaring `type Version string`, the `V1alpha2` constant, `ErrUnknownAPIVersion` sentinel, and the `Detect(cue.Value) (Version, error)` helper.
- [x] 2.2 Add `pkg/apiversion/apiversion_test.go` covering: recognised version, unknown literal, missing field, non-string field — each asserting `errors.Is(err, ErrUnknownAPIVersion)` for failure cases.
- [x] 2.3 Run `task fmt`, `task vet`, `task lint`, `task test`.

## 3. `pkg/api` interface and registry (PR 3)

- [x] 3.1 Create `pkg/api/api.go` defining `Paths` struct, the lowest-common-denominator `ModuleMetadata`/`ReleaseMetadata`/`ProviderMetadata` structs, the `ReleaseView` interface, and the `Binding` interface.
- [x] 3.2 Create `pkg/api/registry.go` (or in same file) with `Register`, `Lookup`, `For` plus the unexported `sync.RWMutex`-guarded map. `Register` panics on duplicate.
- [x] 3.3 Add `pkg/api/registry_test.go` covering: register + lookup happy path, lookup miss returns non-nil error, duplicate-register panics, `For` end-to-end.
- [x] 3.4 Add `pkg/api/doc.go` documenting the `init()`-registration contract and the panic-on-duplicate behaviour.
- [x] 3.5 Run `task fmt`, `task vet`, `task lint`, `task test`.

## 4. `pkg/api/v1alpha2` binding (PR 4)

- [x] 4.1 Create `pkg/api/v1alpha2/binding.go` implementing `api.Binding`. `Paths()` returns the path inventory mirroring today's hardcoded values in `pkg/render/{match,execute}.go`.
- [x] 4.2 Move the unexported `moduleReleaseContextData` and `componentContextData` structs out of `pkg/render/execute.go` into `pkg/api/v1alpha2/context.go`. Implement `BuildTransformerContext` using these structs. Keep a temporary alias in `execute.go` so the renderer still compiles.
- [x] 4.3 Implement `DecodeModuleMetadata`, `DecodeReleaseMetadata`, `DecodeProviderMetadata` mirroring the current decode logic in `pkg/loader/provider.go` and `pkg/module/parse.go`.
- [x] 4.4 Add `pkg/api/v1alpha2/init.go` with `init() { api.Register(&binding{}) }`.
- [x] 4.5 Add `pkg/api/v1alpha2/binding_test.go` covering: `Paths()` returns expected values; `DecodeReleaseMetadata` round-trips a fixture; `BuildTransformerContext` produces a `cue.Value` with the expected `#moduleReleaseMetadata`, `#componentMetadata`, `#runtimeName` fields.
- [x] 4.6 Run `task fmt`, `task vet`, `task lint`, `task test`.

## 5. Loader version detection (PR 5)

- [x] 5.1 Add `ApiVersion apiversion.Version` field to `module.Module`, `module.Release`, `provider.Provider`. Update zero-value docs.
- [x] 5.2 In `pkg/loader/release.go`, after `BuildInstance`, call `apiversion.Detect` on the result. Surface the detected version to callers (return signature change is acceptable here — internal-only consumer is `module.ParseModuleRelease`).
- [x] 5.3 In `pkg/loader/module.go`, do the same for `LoadModulePackage`.
- [x] 5.4 In `pkg/loader/provider.go`, populate `Provider.ApiVersion` after metadata extraction.
- [x] 5.5 In `pkg/module/parse.go`, propagate `Module.ApiVersion` into the resulting `*Release`.
- [x] 5.6 Add tests asserting: a v1alpha2 fixture loads with `ApiVersion=V1alpha2`; an artifact missing `apiVersion` returns `ErrUnknownAPIVersion`.
- [x] 5.7 Run `task fmt`, `task vet`, `task lint`, `task test`.

## 6. Render path lookups (PR 6)

- [x] 6.1 Change `render.Match` signature to `Match(components cue.Value, p *provider.Provider, b api.Binding) (*MatchPlan, error)`. Replace every `cue.ParsePath` / `cue.MakePath(cue.Def(...))` call inside with `b.Paths().X`.
- [x] 6.2 In `render/process_module.go`, resolve the binding via `api.Lookup(rel.ApiVersion)`. Pass it to `Match` and through to `executeTransforms`. Return a typed error if release/provider versions disagree.
- [x] 6.3 In `render/execute.go`, plumb `binding api.Binding` through `executeTransforms` and `executePair`. Replace hardcoded paths (`#transformers`, `#transform`, `#component`, `output`) with the binding's paths.
- [x] 6.4 Update existing render tests to construct a binding-aware fixture (load a real `apis/core/v1alpha2/` artifact and let the loader populate `ApiVersion`).
- [x] 6.5 Add a regression test for "release/provider version mismatch" returning an error before any transformer runs.
- [x] 6.6 Add a regression test for "no binding registered for release version" returning an error before any transformer runs.
- [x] 6.7 Run `task fmt`, `task vet`, `task lint`, `task test`.

## 7. Render context injection via binding (PR 7)

- [x] 7.1 Replace `injectContext` in `pkg/render/execute.go` with a one-line delegation: `ctxVal, warnings, err := binding.BuildTransformerContext(cueCtx, rel, compName, schemaComp, runtimeName); unified = unified.FillPath(binding.Paths().Context, ctxVal)`.
- [x] 7.2 Delete the temporary `moduleReleaseContextData` / `componentContextData` aliases left in `execute.go` from group 4.
- [x] 7.3 Confirm `pkg/module/release.Release` exposes the methods required by `api.ReleaseView` (`ReleaseName`, `Namespace`, `ReleaseUUID`, `ModuleFQN`, `ModuleVersion`, `Labels`, `Annotations`); add thin accessors if missing.
- [x] 7.4 Add a snapshot test rendering a fixture before and after the refactor (use a captured byte-stable rendered output from `main` as the baseline).
- [x] 7.5 Run `task fmt`, `task vet`, `task lint`, `task test`.

## 8. Embedded schemas (PR 8)

- [x] 8.1 Add `apis/core/v1alpha2/embed.go` declaring `//go:embed *.cue cue.mod/module.cue` exposing an `embed.FS` at package scope.
- [x] 8.2 Add `pkg/apiversion/embed.go` (or a small `pkg/api/embed.go`) exposing `EmbeddedSchema(version) (fs.FS, error)` keyed off the registered binding.
- [x] 8.3 Wire `pkg/api/v1alpha2/binding.go` to expose its embedded FS through the binding (e.g. `Binding.EmbeddedSchema() fs.FS`).
- [x] 8.4 Add a test enumerating the embedded FS for `v1alpha2`, asserting the file list and byte contents match the on-disk source.
- [x] 8.5 Run `task fmt`, `task vet`, `task lint`, `task test`.

## 9. Final validation

- [x] 9.1 Run `task check` from the library root and confirm fmt, vet, lint, test all pass.
- [x] 9.2 Update `library/README.md` "Multi-version OPM schema support — current state" section to remove the now-stale "currently a self-reference" caveat.
- [x] 9.3 Update `library/CONSTITUTION.md` if any of the new packages need to be listed in the package layout table.
- [x] 9.4 Add a CHANGELOG entry calling out the `render.Match` signature break under the next MINOR release header.

## 10. Post-verification cleanups (follow-up after `/opsx:verify`)

### 10. ComponentTransformer rename

- [x] 10.1 Rename `#Transformer` → `#ComponentTransformer` in `apis/core/v1alpha2/transformer.cue` (type, kind literal, comment, `#TransformerMap` value type). Keep `#TransformerMap` and `#TransformerContext` names; keep `package transformer`.
- [x] 10.2 Regenerate `apis/core/v1alpha2/INDEX.md` via `cd apis && task generate:index`.

### 11. Drop typed `DefaultNamespace`

- [x] 11.1 Remove `DefaultNamespace` field from `pkg/api/api.go` `ModuleMetadata`.
- [x] 11.2 Remove `DefaultNamespace` field from `pkg/module/module.go` `ModuleMetadata`.
- [x] 11.3 Add `pkg/api/v1alpha2/consts.go` exporting `AnnotationDefaultNamespace = "module.opmodel.dev/defaultNamespace"` per ADR-001.
- [x] 11.4 Flip `adr/001-module-default-namespace-as-annotation.md` status `Proposed` → `Accepted`.

### 12. Drop `ComponentBlueprints` path

- [x] 12.1 Remove `ComponentBlueprints` from `pkg/api/api.go` `Paths`; add doc note explaining why blueprints are not walked at render time.
- [x] 12.2 Remove `ComponentBlueprints` entry from `pkg/api/v1alpha2/binding.go` `paths` var.

### 13. `ApiVersion` → `APIVersion` (idiomatic Go casing)

- [x] 13.1 Rename field on `module.Module`, `module.Release`, `provider.Provider` and their docstrings.
- [x] 13.2 Update all caller sites in `pkg/module/parse.go`, `pkg/render/{module,process_module}.go`, `pkg/loader/provider.go`.
- [x] 13.3 Update tests in `pkg/render/render_test.go` and `pkg/loader/{module,release,provider}_test.go` (struct literals + `Test*Missing*ApiVersion` → `*APIVersion` test names).
- [x] 13.4 Update `CHANGELOG.md` and `README.md` Go-field references.
- [x] 13.5 Propagate the rename to draft slices that already reference the field: `unify-artifact-shape/{proposal,design,tasks,specs/artifact-types/spec}.md`, `add-platform-construct/{proposal,design,tasks,specs/platform-artifact/spec}.md`, `add-platform-composition-helper/design.md`, `add-phase-methods-and-rename-compile/{proposal,design,tasks,specs/kernel-runtime/spec}.md` (`DetectApiVersion` → `DetectAPIVersion`).
- [x] 13.6 Propagate to umbrella enhancement (`enhancements/001-kernel-redesign-around-platform/{README,01-problem,02-design,03-decisions}.md`).

### 14. Validation

- [x] 14.1 Run `task fmt`, `task vet`, `task lint`, `task test`.
- [x] 14.2 Run `cd apis && task generate:index:check` to confirm INDEX matches regenerated output.
- [x] 14.3 Confirm `grep -rn '\.ApiVersion\b' pkg/` returns zero hits and `grep -rn '\.APIVersion\b' pkg/` returns the renamed accesses.

## 15. Spec residuals (post `/opsx:verify` second pass)

The second verification pass surfaced four spec sites that G13/G14 missed and one factual error in the v1alpha2 artifact enumeration. All are markdown-only.

- [x] 15.1 In `specs/api-version-dispatch/spec.md` line 4, replace `(`#Module`, `#ModuleRelease`, `#Provider`, `#Component`)` with `(e.g. `#Module`, `#ModuleRelease`, `#Component`, `#ComponentTransformer`)`. v1alpha2 has no `#Provider` schema definition; `#ComponentTransformer` is the v1alpha2 artifact that references `#ApiVersion`.
- [x] 15.2 In `specs/api-version-dispatch/spec.md`, fix the missed `ApiVersion` casing on lines 83 (`rel.ApiVersion` → `rel.APIVersion`), 90 (two occurrences), and 94 (`release whose ApiVersion`).
- [x] 15.3 Confirm `grep -n 'ApiVersion' openspec/changes/add-multi-apiversion-support/specs/` returns only `#ApiVersion` (CUE schema literal — keep) and `apiVersion` (CUE field — keep). No Go-field `ApiVersion` (camelCase) remains.

The remaining SUGGESTIONs from the second pass are deliberately left as no-action items (rationale captured in the plan file `lazy-gathering-island.md`):

- v1alpha1 `matcher/matcher.cue:57` references its own `#Transformer` — correct for v1alpha1.
- `pkg/api/v1alpha2/consts.go` location is correctly version-scoped per ADR-001.
- `*provider.Provider` and `LoadProvider` references in the spec are correct (they describe the Go type/function, retired in slice 09).
- v1alpha1 `module.cue:25` `defaultNamespace?: string` field — out of scope; future v1alpha1 binding work owns it.
- CHANGELOG mirror hint to ADR-001 — skipped; ADR-001 Consequences already document the cross-repo concern.
