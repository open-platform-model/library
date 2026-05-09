## 1. pkg/module — Release Config Schema Accessor

- [x] 1.1 Add `(r *Release) ConfigSchema() cue.Value` method in `pkg/module/release.go`. Return zero `cue.Value` on nil receiver, on `api.Lookup(r.APIVersion)` error, and when the looked-up path does not exist.
- [x] 1.2 Add godoc explaining that the accessor reads `Paths().Module` then `Paths().Config` and returns the zero value (not an error) on every failure mode.
- [x] 1.3 Add unit tests in `pkg/module/release_test.go` covering: schema reachable on a well-formed release, zero value on unregistered binding, zero value on missing `#config` path, nil-receiver safety.

## 2. pkg/kernel — Slim Input Structs

- [x] 2.1 Remove `Module *module.Module` field from `MatchInput` in `pkg/kernel/inputs.go`. Update godoc on the struct to reflect the new shape.
- [x] 2.2 Remove `Module *module.Module` field from `PlanInput` in `pkg/kernel/inputs.go`. Update godoc on the struct.
- [x] 2.3 Remove `Module *module.Module` field from `CompileInput` in `pkg/kernel/inputs.go`. Update godoc on the struct.
- [x] 2.4 Confirm `ValidateInput` is unchanged — its `Module` field stays in this slice.

## 3. pkg/kernel — Migrate Internal Callers

- [x] 3.1 In `pkg/kernel/phases.go`, remove the `if in.Module == nil { ... }` guard from `Plan` and `Compile`.
- [x] 3.2 In `Compile`, replace the `Module: in.Module` field passed into the embedded `Validate(ValidateInput{...})` call. Source the module from `in.ModuleRelease.Package.LookupPath(b.Paths().Module)` and synthesize a transient `*module.Module` via `module.NewModuleFromValue(k, embeddedModuleValue)`. Add a small private helper `moduleFromRelease(k, rel) (*module.Module, error)` to keep the call site readable.
- [x] 3.3 Confirm `Match` no longer touches `in.Module` (it currently does not — verify with grep).
- [x] 3.4 Update any kernel-internal call sites in `pkg/kernel/wrappers.go` that constructed `MatchInput` / `PlanInput` / `CompileInput` to match the new shape.

## 4. pkg/kernel — Update Tests

- [x] 4.1 In `pkg/kernel/phase_test.go`, update `newPhaseFixture` to embed an `#module` block (with `apiVersion`, `metadata`, `#config: { replicas: int & >0, name: string }`) into the `relPkg` literal. Drop the standalone `mod` field from `phaseFixture` once no test reads it.
- [x] 4.2 Update test cases in `phase_test.go` that previously passed `Module: f.mod` in `ValidateInput` / `MatchInput` / `PlanInput` / `CompileInput` literals: leave `ValidateInput.Module` populated; remove the field from the other three literals.
- [x] 4.3 In `pkg/kernel/kernel_test.go`, update fixtures and call sites that constructed `CompileInput{Module: ...}` to drop the field. Leave any `ValidateInput` constructions untouched.
- [x] 4.4 Add a regression test confirming `Compile` succeeds when given only a release whose `Package` carries an embedded `#module` and no separate `*module.Module`.

## 5. Documentation

- [x] 5.1 CHANGELOG entry under "BREAKING" documenting the removal of `Module` from `MatchInput`, `PlanInput`, and `CompileInput`. Show before/after struct literal examples.
- [x] 5.2 CHANGELOG entry under "Added" for `(*Release).ConfigSchema()`.
- [x] 5.3 Confirm the umbrella enhancement `enhancements/001-kernel-redesign-around-platform/README.md` slice table either references this slice (e.g. as slice 11) or links to it from the open-questions footer. If neither feels right, leave the table untouched and rely on the proposal's cross-reference.

## 6. Validation Gates

- [x] 6.1 Run `task fmt`
- [x] 6.2 Run `task vet`
- [x] 6.3 Run `task lint`
- [x] 6.4 Run `task test`
- [x] 6.5 Run `task check`
