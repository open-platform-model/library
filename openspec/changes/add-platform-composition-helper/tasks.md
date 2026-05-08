## 1. Package Skeleton

- [ ] 1.1 Create `pkg/helper/platform/` with `compose.go` and `errors.go`
- [ ] 1.2 Add package doc comment explaining the helper's role and pointing at slice 09's matcher contract

## 2. MultiFulfillerError Type

- [ ] 2.1 Define `MultiFulfillerError` struct in `errors.go`: fields `FQN`, `ConflictingModules []string`, `ConflictingTransformers []string`
- [ ] 2.2 Implement `Error() string` with a clear human-readable message
- [ ] 2.3 Optionally implement `Unwrap()` if needed for stdlib error walking

## 3. Compose Implementation

- [ ] 3.1 Implement `func Compose(k *kernel.Kernel, shell *platform.Platform, modules []*module.Module) (*platform.Platform, error)`
- [ ] 3.2 Look up the v1alpha2 binding via `apiversion.Detect(shell.Package)` + `api.Lookup`
- [ ] 3.3 Build the `#registry` path expression using `binding.Paths().Registry`
- [ ] 3.4 Iterate `modules`; for each, compute the registry key (`mod.Metadata.Name`)
- [ ] 3.5 For each Module, FillPath into `shell.Package` with a constructed `#ModuleRegistration` value: `{ #module: mod.Package, enabled: true }`
- [ ] 3.6 After all FillPaths, evaluate the resulting value; check for unification errors
- [ ] 3.7 If unification fails with a multi-fulfiller signature (catalog 014's `_invalid` constraint), parse out the offending FQN(s) and return a `*MultiFulfillerError`
- [ ] 3.8 If unification succeeds, return a fresh `*Platform` constructed via `platform.NewPlatformFromValue(k, composedValue)`

## 4. Kernel Wrapper

- [ ] 4.1 Add `(k *Kernel) ComposePlatform(shell *platform.Platform, modules []*module.Module) (*platform.Platform, error)` delegating to the helper

## 5. Multi-Fulfiller Error Detection

- [ ] 5.1 Investigate how catalog 014's `_noMultiFulfiller` constraint surfaces at the CUE error level (e.g. error message pattern, error position)
- [ ] 5.2 Write a parser that extracts the offending FQN and module names from the CUE error tree
- [ ] 5.3 If parsing fails, fall back to wrapping the raw CUE error inside `*MultiFulfillerError` with empty fields and a generic message — the bare CUE error is still informative

## 6. Tests

- [ ] 6.1 Unit test: empty modules list → returns a Platform identical to the shell (modulo `#registry: {}`)
- [ ] 6.2 Unit test: single Module with no conflicts → `#registry` has one entry, computed views populate correctly
- [ ] 6.3 Unit test: two Modules with disjoint transformers → both appear in `#composedTransformers`
- [ ] 6.4 Unit test: two Modules with conflicting transformer FQN → returns `*MultiFulfillerError` with the right FQN and module names
- [ ] 6.5 Idempotency test: calling Compose twice with the same inputs produces byte-equal `Package` values
- [ ] 6.6 Mutation test: inputs `shell` and `modules` are unchanged after Compose returns

## 7. Documentation

- [ ] 7.1 CHANGELOG entry: `helper/platform.Compose` available; one-line Platform composition; multi-fulfiller errors surface as `*MultiFulfillerError`
- [ ] 7.2 Update `library/README.md` Quick Start with a Compose example
- [ ] 7.3 `pkg/helper/platform/doc.go` package doc covering: purpose, ID scheme, multi-fulfiller behavior, idempotency
- [ ] 7.4 Update umbrella enhancement `02-design.md` to confirm slice 10 is shipped

## 8. Validation

- [ ] 8.1 Run `task fmt`
- [ ] 8.2 Run `task vet`
- [ ] 8.3 Run `task lint`
- [ ] 8.4 Run `task test`
- [ ] 8.5 Run `task check`
