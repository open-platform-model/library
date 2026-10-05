## 1. Spike: pin the failure forms of the embedded CUE

Library tests run with an absolute private `TMPDIR`
(`export TMPDIR=$(mktemp -d -p <session scratchpad>)`). Every registry test in this change sets
`CUE_REGISTRY` to a local registry only (no catch-all, so nothing dials a public registry) and
`CUE_CACHE_DIR` to `t.TempDir()`, so a warm cache never hides a failure. Every commit task stages
the files it names with `git add <file>`.

- [ ] 1.1 `opm/internal/registrytest`: add `NewStatusRegistry(t, status int) string`, an `httptest.Server` that answers every request with `status` and an OCI error body for that code (`UNAUTHORIZED` for 401, `DENIED` for 403, `TOOMANYREQUESTS` for 429, none for a 5xx), and `UnreachableRegistry(t) string`, the address of a listener that was opened and closed. Both return a `CUE_REGISTRY` value with `+insecure`. Add a test for each helper in `registrytest_test.go`.
- [ ] 1.2 Add `opm/errors/classify_cue_test.go` (package `errors_test`). For each failure (version absent from a `registrytest` registry, unreachable, 401, 403, 429, 503, and a deadline that expires during the fetch), drive it through `loader.FetchArtifact` and through `loader.LoadDir` on a directory module whose `cue.mod/module.cue` depends on a module served by that registry. For each pair, record the error text and whether the chain holds an `ociregistry.HTTPError` (and its status), `modregistry.ErrNotFound`, a `net.Error` or `context.DeadlineExceeded`. Also drive the schema `OCILoader` against the unreachable registry. Assert what is observed, so the test pins today's forms.
- [ ] 1.3 Write the table of observed forms into design.md "Spike findings". Answer the three open questions: does the `%w` at `modpkgload/import.go:212` survive into `instances[0].Err`; what do flattened 401, 403, 429 and 5xx look like; does a load against an unreachable registry say `cannot do HTTP request`. Adjust the text-fallback list in design D3 to exactly the forms observed. If a 401 or 403 cannot be told apart from a 404 through `cue/load`, say so in D3 and in the spec scenario, and classify that form as `FetchOther`, never as `FetchNotFound`.
- [ ] 1.4 `task check` green, then commit `test(errors): pin the registry failure forms of the embedded cue`.

## 2. errors: typed fetch errors, ErrTransient and Classify

- [ ] 2.1 `opm/errors/fetch.go`: `ErrTransient`, `FetchKind` with `String()`, and `*FetchError` with `Error`, `Unwrap`, `Is` and `Transient` (design D1, D2). Each has godoc, and the `FetchKind` constants say which failures each kind covers.
- [ ] 2.2 `opm/errors/classify.go`: `Classify` with the typed branch and the text fallback from design D3. The fallback's comment names `classify_cue_test.go` as its pin. Add `cuelabs.dev/go/oci/ociregistry` as a direct require at the version already in `go.mod`, run `go mod tidy`, and check that the only `go.mod` diff is the require moving out of the indirect block.
- [ ] 2.3 `opm/errors/fetch_test.go` and `classify_test.go` (constructed errors): `Error()` equals the cause's text; a nil `Err` does not panic; `errors.Is(ErrTransient)` for each kind and for `Status` 429, 500 and 503; `Classify` on nil, on an error already holding a `*FetchError` (same value back), on `context.Canceled` (unchanged), on `context.DeadlineExceeded` (unreachable), on a constructed `ociregistry.NewHTTPError` for 401, 403, 404 and 500, on `modregistry.ErrNotFound` wrapped with `%w`, on a `*net.OpError`, on each text form, and on an unrecognised error (unchanged). Add a test that `cueerrors.Errors` on a `*FetchError` wrapping a CUE error list returns the list's errors with their positions.
- [ ] 2.4 Extend `classify_cue_test.go` so that each pinned form also asserts its `Kind`, its `Status` and its `ErrTransient` answer through `Classify`.
- [ ] 2.5 `opm/errors/errors.go` package doc: a paragraph on fetch failures (`*FetchError`, `ErrTransient`, `Classify`, network-level only). The `opm/errors` line in AGENTS.md "Repository Layout" names them.
- [ ] 2.6 `task check` green, then commit `feat(errors): add typed fetch errors and classify`.

## 3. Classify every fetch and resolution failure the library returns

- [ ] 3.1 `opm/internal/loader/registry.go`: classify the `modconfig.NewRegistry` error and the `reg.Fetch` error inside their existing wraps, setting `Coordinate` to `mv.String()` on a `*FetchError` (design D4, D5). `opm/internal/loader/load.go`: classify `instances[0].Err` only, not the build or gate errors.
- [ ] 3.2 `opm/internal/renderstage/stage.go` (`Build`), `opm/schema/loader.go` (`OCILoader.Load`) and `opm/helper/platformmodule/closure.go` (`Closure`): classify the load or `ModFile` error inside the existing wrap. Nothing else changes.
- [ ] 3.3 Tests through the public verbs (`opm/kernel`, with `registrytest`):
  - `AcquireModuleFromRegistry` and `AcquireCatalogFromRegistry` for an absent version: `FetchNotFound`, `Coordinate` is `path@vX.Y.Z`, and the message equals the one asserted at the base (copy today's text into the test before editing 3.1);
  - the same verbs against an unreachable registry (`FetchUnreachable`, transient) and a 401 registry (`FetchUnauthorized`, not transient, or the kind section 1 recorded);
  - `AcquireModuleFromDir` on a module whose dependency registry is unreachable: transient;
  - `AcquireInstanceFromDir` on a package with a conflict between two concrete values: no `*FetchError`, not transient, and `errors.Is` on the existing sentinels unchanged where the existing tests assert them;
  - `SynthesizeInstance` whose module's dependency registry is unreachable: transient;
  - the render module load: `renderstage.Build` on a staged module whose dependency must come from an unreachable registry, with a fresh `CUE_CACHE_DIR`, is transient, and `Render`'s wrap keeps it (`errors.Is` through `building render module: %w`);
  - `schema.OCILoader.Load` against an unreachable registry (in `opm/schema`): transient;
  - `platformmodule.Closure` with a `ModFileSource` returning a constructed `ociregistry` 404: `FetchNotFound`.
- [ ] 3.4 `go test ./...` with the existing error-text assertions unchanged (none may be edited to pass), then `task check` green, then commit `feat(kernel): classify every fetch and resolution failure`.

## 4. Check cancellation at entry and between stages

- [ ] 4.1 `opm/kernel/acquire.go`: rename the `_ context.Context` parameters to `ctx` and add `ctx.Err()` checks at the points in design D6. `acquireDir` and `loadInstanceWithValues` take `ctx`. Each check returns the bare context error.
- [ ] 4.2 `opm/kernel/synth.go`: the same for `SynthesizeInstance`, at the points in design D6. `opm/internal/loader/registry.go`: check after `reg.Fetch` returns. `opm/kernel/render.go`: check after `renderstage.Build`.
- [ ] 4.3 Godoc: add a `# Cancellation` section to `opm/kernel/doc.go` (design D6), outside the `Surface` list and the code examples, and the one sentence on each of the five verbs, both registry verbs and `Render`. The doc guard test must still pass unchanged.
- [ ] 4.4 Add `opm/kernel/cancel_test.go` with one test per verb: each of the four directory verbs and `SynthesizeInstance` is called with a pre-cancelled context and valid inputs, and returns `context.Canceled` (asserted with `errors.Is` and as the bare error) and a nil artifact. Also add: `SynthesizeInstance` with a cancelled context and no `Module` wraps `ErrMissingModule`; `AcquireModuleFromRegistry` with a cancelled context after a warm-up acquire of the same coordinate in the same `CUE_CACHE_DIR` returns `context.Canceled`; `AcquireInstanceFromDir` with values sources and a pre-cancelled context returns `context.Canceled`; a `context.WithDeadline` in the past gives `context.DeadlineExceeded` on one verb.
- [ ] 4.5 `task check` green, then commit `feat(kernel): check cancellation at entry and between stages`.

## 5. Full suite, consumer builds and api diff

- [ ] 5.1 Run the full non-short suite with the network tests forced: `OPM_FLOW_TEST_FORCE=1 go test -race ./opm/...`. `TestParity_*`, `TestRender_InventoryParity` and the flow test must run, not skip. Record the result in design.md "Verification".
- [ ] 5.2 Run the consumer build against a fresh clone of cli `main` and of opm-operator `main` (`GOTOOLCHAIN=local bash .tasks/consumer-build.sh <clone> . <scratch work dir>`). Both must build and vet. Record the consumer commits in design.md "Verification". Then run `task api:diff`: it must list no incompatible change charged to this branch, and the new `opm/errors` symbols appear as compatible additions.
- [ ] 5.3 In design.md "Verification", list the cli sites and the kind each moves onto (design D2), so the cli change starts from it. Read every call of the five verbs in cli and opm-operator `main`, confirm each passes the caller's live context, and record the call sites.
- [ ] 5.4 `task check` green, then commit `chore(openspec): record type-fetch-errors-and-check-cancellation verification`.

## 6. Verify and archive

- [ ] 6.1 Run `openspec verify` for `type-fetch-errors-and-check-cancellation` (the repo's openspec-verify-change skill). Verify: no CRITICAL finding.
- [ ] 6.2 At PR time, not in the implement stage: run `openspec archive type-fetch-errors-and-check-cancellation --yes`. Verify the following:
  - the new main spec `fetch-error-classification` exists, and its Purpose is edited to a real sentence (archive writes a placeholder);
  - `kernel-runtime` carries "Kernel verbs check cancellation at entry and between stages";
  - `openspec validate --specs --strict` passes.
- [ ] 6.3 Run the gates green, then commit `chore(openspec): archive type-fetch-errors-and-check-cancellation` (the archive rides the implementing PR).
