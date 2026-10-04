# Tasks: consolidate-semver-paths-and-decoders

Worktree `library/.claude/worktrees/consolidate-semver-paths-and-decoders`, branch
`fix/consolidate-semver-paths-and-decoders` (from `origin/main`). `.cue-cache` is seeded by
copying the main checkout's (`cp -a`), never symlinking. Every command runs inside the worktree
with a private `TMPDIR` and the registry env exported on two lines:

```bash
export CUE_REGISTRY='opmodel.dev=ghcr.io/open-platform-model,testing.opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'
export OPM_REGISTRY="$CUE_REGISTRY"
```

`TestGenerate_BuildsThroughTheKernel` can fail in a full-suite run from a known cross-process
cache race; rerun `go test ./opm/helper/platformmodule -count=1` alone before treating it as a
finding. Commit bodies never start a line with `word(` and carry no bare at-sign. The only
trailer is `Co-Authored-By: Claude <noreply@anthropic.com>`.

## 1. One home for version helpers (modversion, renderstage, synth, registrytest, platformmodule; design CS1-CS3)

- [x] 1.1 `opm/internal/modversion/modversion.go`: add `CoreModule`, `CorePath`,
      `LanguageFloor`, `Major`, `Valid` and `Compare` (design CS1), each with a doc comment;
      `Compare` refuses an invalid input as `invalid version %q` naming the caller's spelling.
      Update the package doc to say it is the library's one home for version helpers and the
      only importer of `golang.org/x/mod/semver`. Verify: `go build ./opm/...` clean.
- [x] 1.2 `modversion_test.go`: table tests for `Major` (move `registrytest_test.go`'s
      `TestMajor` table here, plus `""` → `"v"`), `Valid` (bare, prefixed, prerelease,
      `not-a-version`), and `Compare` (equal, bare vs prefixed equal, `v2.0.0-beta.2` <
      `v2.0.0-beta.10` < `v2.0.0`, invalid on either side names that input). Verify:
      `go test ./opm/internal/modversion -count=1` green.
- [x] 1.3 `opm/internal/renderstage`: `skew.go` `isNewer` and `promote.go` `ReplacedVersion` and
      `maxLanguage` go through modversion per the design CS3 table; delete
      `MinLanguageVersion` and point its doc mentions at `modversion.LanguageFloor`; drop the
      Masterminds import. Add `TestCompareSkew_PrereleasePrecedence` to `modfile_test.go`
      (instance `v2.0.0-beta.10` vs platform `v2.0.0-beta.2` is newer; vs `v2.0.0` is not),
      the single-build-render scenario "Prerelease builds compare by SemVer precedence".
      Verify: `go test ./opm/internal/renderstage -count=1` green.
- [x] 1.4 `opm/internal/synth/render.go`: delete `corePath` and `major`; the import line is
      `modversion.CoreModule + "@" + modversion.Major(coreVersion)` (design CS2); move the
      `corePath` doc's reasoning to the call site in one sentence. Verify:
      `go test ./opm/internal/synth -count=1` green.
- [x] 1.5 `opm/internal/registrytest/registrytest.go`: `coreDep` and the stand-in core's
      directory use `modversion.CoreModule`; the three `v0.17.0` literals use
      `modversion.LanguageFloor`; the inline `strings.Cut(version, ".")` majors use
      `modversion.Major`; delete `registrytest.Major` and its test. Callers in
      `opm/kernel/synth_schema_test.go` and `opm/kernel/integration_fixtures_test.go` call
      `modversion.Major`, and the hand-written major in `opm/kernel/acquire_catalog_test.go`
      (`providerCatalogBody`, `strings.Cut(version, ".")`) becomes `modversion.Major` too.
      Verify: `grep -n 'v0\.17\.0\|registrytest.Major' opm/internal/registrytest/registrytest.go
      opm/internal/renderstage/promote.go opm/helper/platformmodule/generate.go
      opm/internal/synth/render.go` prints only doc-comment prose; `grep -rn 'registrytest.Major' opm`
      prints nothing; `go vet ./opm/...` clean. (Test fixtures that declare
      `language: version: "v0.17.0"` in CUE text stay as they are.)
- [x] 1.6 `opm/helper/platformmodule/generate.go`: `CorePath = modversion.CorePath` and
      `LanguageVersion = modversion.LanguageFloor` inside the existing `const` block; each doc
      comment states its value (`"opmodel.dev/core@v2"`, `"v0.17.0"`) so the public Go API
      reference entry stays self-contained. Verify: `go doc ./opm/helper/platformmodule CorePath` shows a const;
      `go test ./opm/helper/platformmodule -count=1` green.
- [x] 1.7 `go mod tidy`: `github.com/Masterminds/semver/v3` leaves `go.mod`,
      `golang.org/x/mod` is a direct requirement. Verify:
      `grep -rn Masterminds --include=*.go --include=go.mod .` prints nothing and
      `grep -rln 'golang.org/x/mod/semver' opm` lists only `opm/internal/modversion/modversion.go`.
- [x] 1.8 `AGENTS.md` repository layout: one line for `internal/modversion/` (version spelling,
      major, SemVer validity and order, the core path and language-floor constants; the only
      SemVer importer). Also reword the CUE-version paragraph's "the literals in
      `opm/internal/registrytest`" to name `modversion.LanguageFloor` (the one Go constant behind
      the render floor, the generated platform module and the registrytest fixtures). Locate both
      edits by text, not line number. Verify: the layout block still renders as one code fence.
- [x] 1.9 `task check` green, then commit
      `refactor(renderstage): share version helpers and compare with x/mod semver`.

## 2. A dotted key is one Path() segment (kernel; design CS5)

- [x] 2.1 `opm/kernel/validate.go`: `walkDisallowed` carries `[]cue.Selector`;
      `fieldNotAllowedError.path` is `[]cue.Selector`; `Path()` maps each selector to
      `sel.String()` after a selector-wise trim of a leading `#module` `#config` pair or
      `#config`; `normalizeFieldPath` and the `strings` import go if unused. Verify:
      `go build ./opm/...` clean.
- [x] 2.2 `opm/kernel/validate_internal_test.go`: a test that validates
      `labels: "app.kubernetes.io/name": "web"` against a schema whose closed `labels` declares
      another field, and asserts the disallowed-field error's `Path()` equals
      `["values", "labels", "\"app.kubernetes.io/name\""]` (config-validation scenario "A label
      containing dots is one path segment"); a table test for the trim over selector paths
      (`#module #config x` → `x`, `#config x` → `x`, a label `#configMap` kept). Negative
      check, not committed: with the old string split the dotted test fails. Verify:
      `go test ./opm/kernel -run 'TestValidate|TestWalk|TestFieldNotAllowed' -count=1` green
      and `go test ./opm/kernel -run TestKernel_AcquireInstanceFromDir -count=1` green
      (`hasErrorPath` joins with `.`).
- [x] 2.3 `task check` green, then commit
      `fix(kernel): keep a dotted values key one Path segment`.

## 3. A metadata decode failure is named once (module, platform, catalog; design CS4)

- [x] 3.1 `opm/module/module.go`, `opm/platform/platform.go`, `opm/catalog/catalog.go`: the
      constructor returns the decoder's error unwrapped; decoders unchanged. Verify:
      `go build ./opm/...` clean.
- [x] 3.2 Tests in `module_test.go`, `platform_test.go` and the catalog's constructor test: a
      `metadata` that does not decode (module and platform `metadata: name: 1`; catalog
      `metadata: version: 1`, since `CatalogMetadata` has no `Name` and Decode ignores unknown
      fields) yields a nil artifact and a non-nil error (asserted first) containing
      `decoding <kind> metadata:` exactly once (`strings.Count == 1`), the
      schema-dispatch scenario "A decode failure names its artifact once". Existing
      "metadata field is required" asserts stay green. Verify:
      `go test ./opm/module ./opm/platform ./opm/catalog -count=1` green.
- [x] 3.3 `task check` green, then commit
      `fix(opm): name a metadata decode failure once` (the change spans module, platform and
      catalog).

## 4. One local-path check and one module-file reader (module, renderstage, platformmodule, catalog; design CS6, CS7)

- [ ] 4.1 `filepath.IsLocal` in `opm/module/source.go` (`Source.WriteTo`),
      `opm/internal/renderstage/stage.go` (`serveDir`) and
      `opm/helper/platformmodule/write.go` (`Files.WriteTo`, keeping `rel == "."`); refusal
      messages unchanged; drop `strings` imports left unused. Verify: the existing outside-root
      and `.` refusal tests in `opm/module`, `opm/internal/renderstage` and
      `opm/helper/platformmodule` stay green.
- [ ] 4.2 `opm/catalog/requires.go`: `Requires` reads through `renderstage.ReadModFile`
      (design CS7); `modFileName` and the direct `modfile.Parse` go; the doc comment says the
      file is parsed by the render stage's reader and what that refuses. Verify:
      `go build ./opm/...` and `task lint` clean (depguard).
- [ ] 4.3 `opm/catalog/requires_test.go`: a subtest in `TestCatalog_Requires_Errors` for a
      module file with a `replaceWith` dependency, asserting the error names
      `cue.mod/module.cue` (catalog-acquisition scenario "A module file the render stage would
      refuse is refused"); the existing cases stay green. Verify:
      `go test ./opm/catalog -count=1` green.
- [ ] 4.4 `task check` green, then commit
      `fix(catalog): read Requires through the shared module-file reader`, with a body line naming
      the IsLocal sites (`opm/module`, `renderstage`, `helper/platformmodule`) and the newly refused
      `replaceWith` dependency.

## 5. An unusable loaded schema says so (schema; design CS8)

- [ ] 5.1 `opm/schema/cache.go`: the `Err()` failure reads
      `schema Cache: loaded schema is unusable: %w`; the `Get` doc comment names both cases
      (a build error, and the zero value a Loader returns with a nil error). Verify:
      `go build ./opm/...` clean.
- [ ] 5.2 `opm/schema/cache_test.go`: `TestCache_ZeroValueFromLoaderIsUnusable` with a Loader
      returning `cue.Value{}, nil`: two `Get` calls return the zero value and the same error,
      which contains `is unusable` and not `build error`, the Loader runs once, and
      `ResolvedVersion()` is `""` (schema-dispatch scenario "A zero value from the Loader is an
      unusable schema"). Verify: `go test ./opm/schema -count=1` green.
- [ ] 5.3 Whole-tree checks on the final tree: `task check`;
      `go test -race ./opm/kernel ./opm/internal/renderstage -count=1`;
      `openspec validate consolidate-semver-paths-and-decoders --strict`. Consumer compile
      check, not committed: build the cli and opm-operator against this tree with a scratch
      `-modfile` carrying a `replace` to the worktree (`go build -C <repo> -modfile <scratch>
      ./...`); both build. Verify: all green.
- [ ] 5.4 `task check` green, then commit
      `fix(schema): call an unusable loaded schema unusable`.

## 6. Archive (at PR time, on the supervisor's word)

- [ ] 6.1 `openspec archive consolidate-semver-paths-and-decoders --yes` on this branch, so the
      archive rides the implementing PR. Verify: the four main specs carry the modified
      requirements and `openspec validate --all --strict` passes. No `enhancement.yaml`, so no
      delivery log runs.
- [ ] 6.2 Commit `chore(openspec): archive consolidate-semver-paths-and-decoders`.
