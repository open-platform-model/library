## 1. Package Skeleton

- [x] 1.1 Create `opm/helper/platform/` with `compose.go` and `errors.go`
- [x] 1.2 Add package doc comment explaining the helper's role and pointing at slice 09's matcher contract

## 2. MultiFulfillerError Type

- [x] 2.1 Define `MultiFulfillerError` struct in `errors.go`: fields `FQN`, `ConflictingModules []string`, `ConflictingTransformers []string`
- [x] 2.2 Implement `Error() string` with a clear human-readable message
- [x] 2.3 Optionally implement `Unwrap()` if needed for stdlib error walking

## 3. Compose Implementation

- [x] 3.1 Implement `func Compose(k *kernel.Kernel, shell *platform.Platform, modules []*module.Module) (*platform.Platform, error)`
- [x] 3.2 Look up the v1alpha2 binding via `apiversion.Detect(shell.Package)` + `api.Lookup`
- [x] 3.3 Build the `#registry` path expression using `binding.Paths().Registry`
- [x] 3.4 Iterate `modules`; for each, compute the registry key (`mod.Metadata.Name`)
- [x] 3.5 For each Module, FillPath into `shell.Package` with a constructed `#ModuleRegistration` value: `{ #module: mod.Package, enabled: true }`
- [x] 3.6 After all FillPaths, evaluate the resulting value; check for unification errors
- [x] 3.7 If unification fails with a multi-fulfiller signature (catalog 014's `_invalid` constraint), parse out the offending FQN(s) and return a `*MultiFulfillerError`
- [x] 3.8 If unification succeeds, return a fresh `*Platform` constructed via `platform.NewPlatformFromValue(k, composedValue)`

## 4. Kernel Wrapper

- [x] 4.1 Add `(k *Kernel) ComposePlatform(shell *platform.Platform, modules []*module.Module) (*platform.Platform, error)` delegating to the helper

## 5. Multi-Fulfiller Error Detection

- [x] 5.1 Investigate how catalog 014's `_noMultiFulfiller` constraint surfaces at the CUE error level (e.g. error message pattern, error position)
- [x] 5.2 Write a parser that extracts the offending FQN and module names from the CUE error tree
- [x] 5.3 If parsing fails, fall back to wrapping the raw CUE error inside `*MultiFulfillerError` with empty fields and a generic message — the bare CUE error is still informative

## 6. Tests

- [x] 6.1 Unit test: empty modules list → returns a Platform identical to the shell (modulo `#registry: {}`)
- [x] 6.2 Unit test: single Module with no conflicts → `#registry` has one entry, computed views populate correctly
- [x] 6.3 Unit test: two Modules with disjoint transformers → both appear in `#composedTransformers`
- [x] 6.4 Unit test: two Modules with conflicting transformer FQN → returns `*MultiFulfillerError` with the right FQN and module names
- [x] 6.5 Idempotency test: calling Compose twice with the same inputs produces byte-equal `Package` values
- [x] 6.6 Mutation test: inputs `shell` and `modules` are unchanged after Compose returns

## 7. Documentation

- [x] 7.1 CHANGELOG entry: `helper/platform.Compose` available; one-line Platform composition; multi-fulfiller errors surface as `*MultiFulfillerError`
- [x] 7.2 Update `library/README.md` Quick Start with a Compose example
- [x] 7.3 `opm/helper/platform/doc.go` package doc covering: purpose, ID scheme, multi-fulfiller behavior, idempotency
- [x] 7.4 Update umbrella enhancement `02-design.md` to confirm slice 10 is shipped

## 8. Validation

- [x] 8.1 Run `task fmt`
- [x] 8.2 Run `task vet`
- [x] 8.3 Run `task lint`
- [x] 8.4 Run `task test`
- [x] 8.5 Run `task check`
