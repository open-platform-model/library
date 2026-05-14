## 1. Sentinels & shared helper (`loader`)

- [x] 1.1 Add `ErrInvalidPackage`, `ErrWrongKind`, `ErrMissingRequiredField` sentinel errors to `opm/helper/loader/file/`
- [x] 1.2 Add internal `artifactSpec` struct (`expectedKind`, `requiredConcreteFields`, `moduleRefPaths`) and an unexported `shapeGate(val cue.Value, spec artifactSpec) error` that checks root-is-struct, `kind` match, concrete identity fields, and recursive `#module.kind == "Module"` for each ref path
- [x] 1.3 Add unexported `requireConcrete(val cue.Value, path string) error` helper (Exists + IsConcrete + non-empty for strings), wrapping `ErrMissingRequiredField`

## 2. Wire loaders to the gate (`loader`)

- [x] 2.1 Tighten the instance-count guard in all three loaders from `len(instances) == 0` to `len(instances) != 1`, wrapping `ErrInvalidPackage`
- [x] 2.2 Call `shapeGate` in `LoadModulePackage` with the Module spec (`kind: "Module"`; fields `metadata.name`, `metadata.modulePath`, `metadata.version`) after build, before return
- [x] 2.3 Call `shapeGate` in `LoadReleasePackage` with the Release spec (`kind: "ModuleRelease"`; fields `metadata.name`, `metadata.namespace`; module ref `#module`)
- [x] 2.4 Call `shapeGate` in `LoadPlatformPackage` with the Platform spec (`kind: "Platform"`; fields `metadata.name`, `type`; module refs over `#registry[_].#module`)

## 3. Tests (`loader`)

- [x] 3.1 Add fixtures: wrong-kind directory, module missing `metadata.name`, release with non-module `#module`, platform registry entry with non-module `#module`, non-struct root, two files with conflicting `package` clauses
- [x] 3.2 Table-driven tests asserting each fixture returns an error wrapping the expected sentinel (via `errors.Is`)
- [x] 3.3 Regression test: a well-formed module/release/platform still returns the same `cue.Value` and `apiversion.Version` as before

## 4. Validation gates

- [x] 4.1 `task fmt`
- [x] 4.2 `task vet`
- [x] 4.3 `task lint`
- [x] 4.4 `task test`
