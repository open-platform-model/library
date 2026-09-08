# Tasks: core-version-without-schema-build

## 1. The pinned release is known without a load

- [ ] 1.1 `opm/schema/loader.go`: `func (l OCILoader) PinnedVersion() (string, bool)` using `ast.SplitPackageVersion` on `l.Module` (or `DefaultSchemaModule` when empty), true only for a full release; `loader_test.go`: cases for the default, an explicit `-alpha.N` pin, a bare major and a malformed identifier; verify the tests pass and `(OCILoader{}).PinnedVersion()` equals `DefaultSchemaVersion()`.
- [ ] 1.2 `opm/kernel/synth.go`: `resolveCoreVersion` returns the pinned release when the configured loader (default when nil) is an `OCILoader` with one, else `Get()` plus `ResolvedVersion()`; delete the `#ModuleInstance` existence check; `synth_test.go`: a default kernel synthesizes and `SchemaCache().ResolvedVersion()` is still `""` afterwards; a bare-major loader against the in-process registry still resolves through the cache; a served core lacking `#ModuleInstance` fails inside the synth build with an error naming the definition; verify `go test ./opm/kernel/...` is green.

## 2. Docs

- [ ] 2.1 `CLAUDE.md` § Schema cache lifetime contract (the first schema-touching call is no longer synthesis on a pinned kernel; name the callers that still load it) and `opm/kernel/doc.go` (`SynthesizeInstance` paragraph); verify `grep -n 'resolved core version\|first schema-touching' CLAUDE.md opm/kernel/doc.go` reads consistently with the code.

## 3. Validation gates

- [ ] 3.1 `task fmt`, `task vet`, `task lint`, `task test` green; `task cue:test:flow` green against GHCR (a registry-module render synthesizes with no core fetch outside the module's own dependency resolution; verify by running with an empty `CUE_CACHE_DIR` and confirming the only `opmodel.dev/core` extraction is the version the module's `cue.mod` pins); verify `openspec validate --changes` passes.
