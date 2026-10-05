## 1. Spike: pin the author-defect forms of the embedded CUE

Library tests run with an absolute private `TMPDIR`
(`export TMPDIR=$(mktemp -d -p <session scratchpad>)`), against the worktree's own copy of the
main checkout's `.cue-cache`. Every registry case sets `CUE_REGISTRY` to a local registry only and
`CUE_CACHE_DIR` to a fresh cache, as `classify_cue_test.go` already does. Every commit task stages
the files it names with `git add <file>`.

- [ ] 1.1 `opm/errors/classify_cue_test.go`: add a `standalone/unprovided-import` case. Serve a module whose package imports a package that no module of its build provides, then load it standalone as `path@vX.Y.Z` (`loadStandalone`), the way the cli's `loadPublishedPackage` does. Record its text. It must not carry the exact-version form.
- [ ] 1.2 Add a `load/ambiguous-import` case: a main module and a declared dependency that both provide the imported package path (for example the main module `spike.example/main@v0` holding `dep/`, and a dependency `spike.example/main/dep@v0`). Record its text. If CUE v0.17.1 does not produce `ambiguous import: ` through `cue/load`, record what it produces instead.
- [ ] 1.3 Add a `load/malformed-main-module-file` case (the main module's own `cue.mod/module.cue` does not parse), and a case that reaches a malformed dependency module file on the direct import path instead of in graph expansion, if one exists (a declared dependency whose own module file does not parse, imported directly). Record whether each text carries `cannot parse module file`, and whether it carries `cannot fetch `.
- [ ] 1.4 Every new case is pinned at today's answer (`classified{}`: returned unchanged, or the `FetchKind` it gets today). Write the observed table under design.md "Spike findings". If a form differs from design D3, adjust D3 and the delta spec's text-fallback list to exactly what was observed. If the ambiguous form cannot be produced, drop `ResolutionImportAmbiguous` from D1, from the spec delta and from section 3, and say so in the findings.
- [ ] 1.5 `task check` green, then commit `test(errors): pin the author-defect resolution forms of the embedded cue`.

## 2. errors: type unprovided imports as resolution errors

- [ ] 2.1 `opm/errors/resolution.go`: add `ResolutionKind` with `String()`, the constants `ResolutionOther` and `ResolutionImportUnprovided`, and `*ResolutionError` with `Error` and `Unwrap` (design D1). Give each a godoc. The `ResolutionImportUnprovided` doc names the three causes and the `cue mod tidy` producer (design D2), and says the kind is never transient. Cite 0021:D8:R12 once, at the type.
- [ ] 2.2 `opm/errors/classify.go`: split `classifyText` into the fetch forms and a trailing `cannot fetch ` step. Add `classifyResolutionText` between them for `cannot find module providing package ` (design D3). Make the idempotence check also return a chain that holds a `*ResolutionError`. Rewrite the `Classify` godoc, the `textVersionNotProvided` comment and the `FetchNotFound` comment in `fetch.go`, which today say these forms are "not a FetchError" or are "never matched", so they say the forms are typed as `*ResolutionError`.
- [ ] 2.3 Tests in `opm/errors`. `resolution_test.go`: `Error()` equals the cause's text; a nil `Err` does not panic; `errors.Is(ErrTransient)` is false; `errors.As` finds no `*FetchError`; `cueerrors.Errors` through a `*ResolutionError` returns the list's errors and positions. In `classify_test.go`, move the two `cannot find module providing package a.b/c` forms from `TestClassify_UnrecognisedIsUnchanged` to a new `TestClassify_Resolution` table, and add rows there: the cli's constructed forms with `cannot do HTTP request` and with a 503 (`*FetchError`, no `*ResolutionError`), and classifying a `*ResolutionError` twice (the same value). In `classify_cue_test.go`, extend `classified` with the resolution kind, and set the `load/undeclared-import`, `load/own-path-missing-package`, `load/dependency-missing-package` and `standalone/unprovided-import` cases to `ResolutionImportUnprovided`. Keep every text assertion as it is: no message changes.
- [ ] 2.4 `opm/kernel/fetch_classify_test.go`: rename `TestFetchClassify_UnresolvableImportStaysPlain` to `TestFetchClassify_UnresolvableImportIsResolutionError`, keep both rows and the text assertion, and assert a `*ResolutionError` of kind `ResolutionImportUnprovided`, no `*FetchError` and no `ErrTransient`. Add a row for a package missing from a declared, served dependency (`registrytest`), and a row through `AcquireInstanceFromDir` with a file-backed values source that imports an undeclared module.
- [ ] 2.5 `opm/errors/errors.go` package doc: add one sentence for `*ResolutionError` beside the fetch paragraph. Update the `opm/errors` line in AGENTS.md "Repository Layout" (the fetch classification entry) to name `*ResolutionError` and `ResolutionKind`. Update `TestClassify_UnrecognisedIsUnchanged`'s comment so it no longer says an import no module provides comes back unchanged.
- [ ] 2.6 `task check` green, then commit `feat(errors): type unprovided imports as resolution errors`.

## 3. errors: type ambiguous imports and invalid module files

- [ ] 3.1 `opm/errors/resolution.go`: add `ResolutionImportAmbiguous` and `ResolutionModuleFileInvalid` with their docs and `String()` names (design D1). `classify.go`: add the `ambiguous import: ` and `cannot parse module file` forms to `classifyResolutionText`, ahead of `cannot fetch ` (design D3). Update the comment on the `cannot expand module graph` rule, which today says the malformed-dependency form stays unclassified.
- [ ] 3.2 Tests: in `classify_cue_test.go`, set `load/malformed-dependency-module-file`, the section 1 ambiguous case and the section 1 module-file cases to their kinds. In `classify_test.go`, move the `cannot expand module graph ... cannot parse module file` form from `TestClassify_UnrecognisedIsUnchanged` to `TestClassify_Resolution`, and add a constructed `cannot fetch P@V: cannot parse module file from P@V: ...` row (`ResolutionModuleFileInvalid`, not `FetchOther`) and a `cannot expand module graph: ...: cannot do HTTP request` row (`FetchUnreachable`).
- [ ] 3.3 `task check` green, then commit `feat(errors): type ambiguous imports and invalid module files`.

## 4. Full suite, consumer builds, api diff and the cli replacement proof

- [ ] 4.1 Run the full non-short suite with the network tests forced: `OPM_FLOW_TEST_FORCE=1 go test -race ./opm/...`. `TestParity_*`, `TestRender_InventoryParity` and the flow test must run, not skip. Record the result in design.md "Verification".
- [ ] 4.2 Clone cli `main` and opm-operator `main` into the scratch directory, and run `GOTOOLCHAIN=local bash .tasks/consumer-build.sh <clone> . <scratch work dir>` for each. Both must build and vet. Record the consumer commits. Then run `task api:diff`. It must charge no incompatible change to this branch, and the new `opm/errors` symbols must appear as compatible additions.
- [ ] 4.3 Prove the cli replacement (design D5). In the scratch cli clone, replace the body of `unprovidedImport` with the `*ResolutionError` check, and use `go work init <clone> <worktree>` in a scratch directory. Run the named `internal/publish` and `internal/cuemod` tests from D5 under that `GOWORK`, with an absolute `TMPDIR`. Every row of `TestLoadPublishedPackage_Pinned` and `TestUnprovidedImport_RegistryFailureIsNotAbsent` must pass unchanged. Then try the `platformBuildHint` replacement against `internal/config/platform_hint_pin_test.go`, and record that result as information for the cli change. Commit nothing to the cli, and remove the clone afterwards. Record the commands, the cli commit and each result in design.md "Verification".
- [ ] 4.4 In design.md "Verification", list the frontend follow-ups: in the cli, `compat.go` `unprovidedImport` moves onto `ResolutionImportUnprovided`, and the `platform.go` hint moves onto the types if 4.3 showed parity. In the operator, `isTerminalCause` gains `*ResolutionError`.
- [ ] 4.5 `task check` green, then commit `chore(openspec): record type-author-resolution-errors verification`.

## 5. Verify and archive

- [ ] 5.1 Run `openspec verify` for `type-author-resolution-errors` (the repo's openspec-verify-change skill). There must be no CRITICAL finding.
- [ ] 5.2 At PR time, not in the implement stage: run `openspec archive type-author-resolution-errors --yes`. Then check:
  - `fetch-error-classification` carries the restated `Classify` requirement and the two ADDED requirements;
  - its Purpose is edited by hand so that it no longer says `Classify` leaves every author defect unchanged;
  - `openspec validate --specs --strict` passes.
- [ ] 5.3 Run the gates green, then commit `chore(openspec): archive type-author-resolution-errors`. The archive rides the implementing PR.
