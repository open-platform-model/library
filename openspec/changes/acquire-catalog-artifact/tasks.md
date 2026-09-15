# Tasks: acquire-catalog-artifact

Three sections. design.md's two research entries are readings of existing code, not assumptions to
prove, so section 1 is not a spike — but it is deliberately the ADR, because if that argument cannot
be written convincingly the rest should not be built.

## 1. The boundary argument

- [x] 1.1 Write `adr/009-catalog-is-an-acquired-kind.md` following `adr/TEMPLATE.md` (Status, Context, Decision, Consequences) and design.md § The kernel grows a fourth acquired kind: a catalog is already a transitive kernel input, the need is proven by two consumers per `CONSTITUTION.md` line 138, the alternative reverses the kernel migration, and the preserved boundary is read-versus-judge. State the test a fifth kind would have to pass. Verify: the ADR stands alone — a reader who has not seen enhancement 0015 can judge the argument.
- [x] 1.2 Record in ADR-009's Consequences that `loader.FetchModule`'s name is now kind-agnostic in fact but not in spelling, and — explicitly — that `cli/internal/publish` is NOT a consumer and its loading is not a collapse candidate, with the reason (it probes published subpackages, gates to no kind, treats absence as a scan signal, and carries source positions). Verify: the non-consumer note is present, so a later reader does not rediscover the resemblance and write a migration that cannot work.
- [x] 1.3 `task check` green, then commit `docs(adr): record why a catalog is an acquired kind`.

## 2. The catalog artifact and its shape gate

- [x] 2.1 Add `opm/internal/loader/shape.go`'s `CatalogSpec` beside `ModuleSpec`, `InstanceSpec` and `PlatformSpec`: kind `"Catalog"` plus the metadata fields core requires of a catalog. Verify: no new gating routine — the existing one handles it, and the sentinels are unchanged.
- [x] 2.2 Add `opm/catalog` with `Catalog` (Metadata, Package, Source) and `NewCatalogFromValue`, mirroring `opm/module`. Verify: the package has no dependency on `opm/kernel`, matching `opm/module`'s direction.
- [x] 2.3 Add `Catalog.Provides()`: fold over the catalog's own `#transformers`, collecting every required contract whose value carries `fulfilment: "provider"`, sorted. Decode on demand, never at construction, following `Platform.Contracts()`. A catalog implementing none returns an empty set and no error. Verify: a table test covers the provider case, the non-provider-fulfilment case, the empty case, and that two runs are equal element for element.
- [x] 2.4 Add `Catalog.Requires()` returning the `cue.mod/module.cue` requirements as path to version. Verify: the return type exposes no CUE module types to the caller.
- [x] 2.5 `task check` green, then commit `feat(catalog): add the Catalog artifact, its shape gate and its derivations`.

## 3. The kernel verbs

- [x] 3.1 Add `Kernel.AcquireCatalogFromRegistry(ctx, modPath, version)` over the loader's one registry routine, then the shape gate and `NewCatalogFromValue`. That routine hard-wires `ModuleSpec` and the module coordinate check, so it is parameterized rather than reused as-is: `FetchArtifact(..., spec)` carries the body and `FetchModule` becomes the wrapper passing `ModuleSpec` (design.md § The registry plumbing is kind-agnostic only up to the build). Verify: no second fetch path exists; `FetchModule`'s signature, behaviour and call site are unchanged; `cuecontext.New()` is called at entry per ADR-007.
- [x] 3.2 Add `Kernel.AcquireCatalogFromDir(ctx, dirPath)` as the directory peer, stamping `Source` in overlay mode as `AcquireModuleFromDir` does. Verify: the caller's directory is not written to.
- [x] 3.3 Cover the refusals: a `kind: "Module"` artifact acquired as a catalog fails wrapping `ErrWrongKind` naming both kinds; an artifact with no `kind`, or a non-concrete one, fails the same way; no partial catalog is returned on any gate failure. Verify: each case asserts the sentinel with `errors.Is`, not a message match.
- [x] 3.4 Exercise the registry path against the test fixture registry (`opm/internal/registrytest`), so acquisition is proven end to end and not only against a directory. Verify: the negative registry case runs before any positive load warms the cache.
- [x] 3.5 `task check` green, then commit `feat(kernel): acquire a catalog from a registry or a directory`.
