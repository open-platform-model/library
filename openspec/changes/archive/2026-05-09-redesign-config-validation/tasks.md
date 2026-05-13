## 1. Source Type and Options

- [x] 1.1 Create `opm/kernel/source.go` with `Source struct {Value cue.Value; Name, Origin string}`, an unexported `config` struct, the `Option func(*config)` type, and the exported `Partial() Option` constructor. Include godoc on `Source.Value` stating the `cue.Filename(Origin)` contract. **Note**: the option type is named `ValidateOption` (not `Option`) to avoid colliding with the existing kernel-construction `Option` in `opm/kernel/kernel.go`. The unexported config struct is named `validateConfig` for the same reason.
- [x] 1.2 Add a unit test in `opm/kernel/source_test.go` verifying `Partial()` flips the unexported `config.partial` flag and zero-value `Source` is allowed.

## 2. New Validation Primitives (Additive)

- [x] 2.1 Add `(*Kernel).ValidateConfigDetailed(schema cue.Value, sources []Source, opts ...Option) (cue.Value, error)` in `opm/kernel/validate.go`. Implementation: short-circuit on empty sources to `(cue.Value{}, nil)`; unify `sources[i].Value` left-to-right; route to `unifiedValidate(schema, merged, requireConcrete)` with `requireConcrete = !cfg.partial` (where `cfg` is built from `opts`). Reuse the existing `walkDisallowed` + `appendSchemaErrors` internals.
- [x] 2.2 Add unit tests in `opm/kernel/validate_test.go` covering: empty sources, single source success, two-source unify success, conflict between sources surfaces both `cueerrors.Positions`, `Partial()` skips concrete check while `walkDisallowed` still fires.

## 3. Source Loaders

- [x] 3.1 Create `opm/kernel/source_loader.go`. Implement `(*Kernel).LoadSourceFromBytes(origin, name string, b []byte) (Source, error)` calling `k.cueCtx.CompileBytes(b, cue.Filename(origin))` and returning `Source{Value: v, Name: name, Origin: origin}` with `v.Err()` surfaced as the error.
- [x] 3.2 Implement `(*Kernel).LoadSourceFromString(origin, name, s string) (Source, error)` mirroring 3.1 with `CompileString`.
- [x] 3.3 Implement `(*Kernel).LoadSourceFromFile(path string) (Source, error)` delegating to `opm/helper/loader/file.LoadValuesFile` (which already populates `cue.Filename` via `cue/load.Instances`); set `Origin = path` and `Name = filepath.Base(path)`. **Note**: Origin is set to the absolute path (matching `cue/load.Instances`'s baked filename) rather than the caller's possibly-relative path, so the contract `Origin == pos.Filename()` holds.
- [x] 3.4 Add unit tests in `opm/kernel/source_loader_test.go` asserting that errors from the returned `Source.Value` carry `pos.Filename() == origin` for the bytes and string variants and `pos.Filename() == path` for the file variant.

## 4. Print Helper — REVERSED

**Decision reversal**: PrintErrors was implemented and then removed. Reason: presentation-layer code in the kernel violates Constitution principles I (Kernel Neutrality) and IV (Composability: "Output formatting and presentation MUST stay outside the library"). Frontends use `cueerrors.Print` directly or walk the error tree themselves.

- [x] 4.1 ~~Create `opm/kernel/print.go`~~ **REVERSED** — file deleted; library ships no display helper.
- [x] 4.2 ~~Add unit tests in `opm/kernel/print_test.go`~~ **REVERSED** — file deleted alongside `print.go`.

## 5. Module and Release Convenience Methods

**Spec deviation**: Methods originally placed on `*Module`/`*Release` (e.g. `m.ValidateValues(k, ...)`) cause an import cycle (`opm/kernel` already imports `opm/module`; methods on `*Module` taking `*kernel.Kernel` would close the cycle). Methods are placed on `*Kernel` instead: `k.ValidateModuleValues(m, ...)`, `k.ValidateReleaseValues(r, ...)`. Same typed dispatch, no cycle. Specs/design will be updated at the end of the run to reflect this.

- [x] 5.1 Add `(*Module).ConfigSchema() cue.Value` in `opm/module/module.go` using `api.Lookup(m.APIVersion).Paths().Config` against `m.Package`. (Release already has it.)
- [x] 5.2 Add `(*Kernel).ValidateModuleValues`, `ValidateModuleValuesPartial`, `ValidateModuleValuesDetailed` as 1-line delegations to the kernel primitives.
- [x] 5.3 Confirm `(*Release).ConfigSchema() cue.Value` already exists in `opm/module/release.go`. (No new code; verify.)
- [x] 5.4 Add `(*Kernel).ValidateReleaseValues`, `ValidateReleaseValuesPartial`, `ValidateReleaseValuesDetailed` as 1-line delegations.
- [x] 5.5 Add unit tests in `opm/kernel/validate_typed_test.go` (or extend `validate_test.go`) covering each new typed shortcut against fixtures with valid and invalid values.

## 6. Migrate Internal Callsites and Replace Old Primitives

- [x] 6.1 Replace the body of `(*Kernel).ValidateConfig` and `(*Kernel).ValidateConfigPartial` in `opm/kernel/validate.go`. New signatures: `ValidateConfig(schema, values cue.Value) (cue.Value, error)` and `ValidateConfigPartial(schema, values cue.Value) (cue.Value, error)`. Drop `contextLabel`, `name`. Return CUE-native error from the existing `appendSchemaErrors` plumbing instead of `*oerrors.ConfigError`. Keep `walkDisallowed`, `fieldNotAllowedError`, `normalizeFieldPath` as private internals.
- [x] 6.2 Update `opm/kernel/phases.go` `(*Kernel).Validate` phase method: call `k.ValidateConfig(schema, in.Values)`, wrap any non-nil error with `fmt.Errorf("module %q: %w", releaseDisplayName(in.ModuleRelease), err)`. Public signature unchanged.
- [x] 6.3 Update `opm/kernel/process.go` `(*Kernel).ProcessModuleRelease`: replace `runValidate(...)` call with `k.ValidateConfig(schema, values)`. Wrap any non-nil error with `fmt.Errorf("release %q: %w", name, err)`. Leave the subsequent `spec.Validate(cue.Concrete(true))` (CUE stdlib) untouched.
- [x] 6.4 Delete `(*Kernel).ValidateAndUnify` wrapper from `opm/kernel/wrappers.go`. Remove the corresponding import of `opm/helper/values`.
- [x] 6.5 Rewrite tests in `opm/kernel/validate_test.go`, `opm/kernel/process_test.go`, and `opm/kernel/phase_test.go` to assert on CUE-native errors via `cueerrors.Errors`/`Position()` and on the `module "<name>": ` framing for phase-method results.

## 7. Delete Helper Values Package

- [x] 7.1 Delete `opm/helper/values/` entirely (`values.go`, `errors.go`, `doc.go`, `values_test.go`, and any other files in the directory).
- [x] 7.2 Verify no remaining imports of `github.com/open-platform-model/library/opm/helper/values` in any package via grep; remove any stragglers.

## 8. Delete Custom Validation Error Types

- [x] 8.1 Delete `opm/errors/config_error.go` entirely (`ConfigError`, `GroupedErrors`, `GroupedErrorsFromError`, `groupCUEErrors`, `normalizeCUEPath`).
- [x] 8.2 Edit `opm/errors/domain.go`: remove `ValidationError`, `FieldError`, `ErrorLocation`, `GroupedError`, plus their methods. Keep `TransformError` and its methods.
- [x] 8.3 Remove `NewValidationError` from `opm/errors/errors.go` if present; audit `opm/errors/sentinel.go` for validation-only sentinels and remove. **Note**: `DetailError` was also removed (orphaned by `NewValidationError` deletion). `ErrValidation` sentinel kept — useful for non-CUE-tree generic validation paths.
- [x] 8.4 Audit and update `opm/errors/errors_test.go` to drop tests for removed types.
- [x] 8.5 Verify no remaining references to deleted types via `grep -r "ConfigError\|ValidationError\|FieldError\|ErrorLocation\|GroupedError\|MultiSourceError\|LayerError" opm/`. (Two cosmetic refs remain: `opm/kernel/doc.go` outdated comment and `opm/kernel/phase_test.go` test function name. Both addressed in task 9.)

## 9. Documentation and Changelog

- [x] 9.1 Update `opm/kernel/doc.go` to describe the three primitives, `Source`, `Option`, the loader helpers, and `PrintErrors`. Drop references to deleted types.
- [x] 9.2 Add godoc on `Source`, `Option`, `Partial`, the three `ValidateConfig*` methods, the three `LoadSourceFrom*` methods, `PrintErrors`, and the eight `*Module`/`*Release` convenience methods. Each entry MUST cover purpose, pre-conditions, and the CUE-native error format.
- [x] 9.3 Update `CHANGELOG.md` with a MAJOR version entry covering: deleted types, new types, migration recipes (mirror the table in design.md), the `cue.Filename(Origin)` contract for `Source.Value`.
- [x] 9.4 No `library/CLAUDE.md` exists; updated `library/README.md` instead (it had the layered validation example using the deleted `helper/values` package).
- [x] 9.5 Rewrite `docs/design/kernel-validate-flow.md` so it represents the post-redesign `Kernel.Validate` end-to-end. Required updates: drop `runValidate`/`ConfigError` from the class and sequence diagrams; replace return type with the wrapped CUE-native `error` (showing the `fmt.Errorf("module %q: %w", name, err)` framing); add `Module.ConfigSchema()` to the class diagram alongside the existing `Release.ConfigSchema()`; add a section covering the three primitives (`ValidateConfig`, `ValidateConfigPartial`, `ValidateConfigDetailed`) with their relationship; replace the "Tier 1 vs Tier 2" section with a "Single-source vs Layered" framing that points at `ValidateConfigDetailed` and the `Source` struct; update source-file references to the new `opm/kernel/{validate,source,source_loader,print}.go` layout. Verify the doc no longer mentions `opm/helper/values`, `opm/errors/config_error.go`, or `*oerrors.ConfigError`. (`runValidate` is mentioned only as the *internal* shared helper that backs the three primitives — kept for accuracy.)

## 10. Validation Gates

- [x] 10.1 Run `task fmt` and commit any formatting fixes.
- [x] 10.2 Run `task vet` and resolve any reported issues.
- [x] 10.3 Run `task lint` and resolve any reported issues.
- [x] 10.4 Run `task test` and ensure all unit tests pass.
- [x] 10.5 Run `task check` as the final composite gate before considering the change apply-ready.
