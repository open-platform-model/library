# Tasks: accept-bare-semver-in-registry-verbs

Worktree `library/.claude/worktrees/accept-bare-semver-in-registry-verbs`, branch
`fix/accept-bare-semver-in-registry-verbs` (from `origin/main`). Seed `.cue-cache` by copying the
main checkout's (`cp -a`), never symlinking. Every command runs inside the worktree with the
registry env exported on two lines:

```bash
export CUE_REGISTRY='opmodel.dev=ghcr.io/open-platform-model,testing.opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'
export OPM_REGISTRY="$CUE_REGISTRY"
```

`TestGenerate_BuildsThroughTheKernel` can fail in a full-suite run from a known cross-process
cache race; rerun `go test ./opm/helper/platformmodule -count=1` alone before treating it as a
finding. Commit bodies never start a line with `word(` and carry no bare at-sign. The only
trailer is `Co-Authored-By: Claude <noreply@anthropic.com>`.

## 1. Lift canonicalVersion into a shared internal helper (modversion, platformmodule; design BS1)

- [ ] 1.1 New package `opm/internal/modversion` with `Canonical(v string) string` (design BS1)
      and a package doc comment saying it holds the library's CUE module version helpers.
      Verify: `go build ./opm/...` clean.
- [ ] 1.2 `opm/internal/modversion/modversion_test.go`: a table test over `"1.0.0"` →
      `"v1.0.0"`, `"v1.0.0"` unchanged, `"1.0.0-alpha.2"` → `"v1.0.0-alpha.2"`, `""` unchanged.
      Verify: `go test ./opm/internal/modversion -count=1` green.
- [ ] 1.3 `opm/helper/platformmodule/generate.go`: delete `canonicalVersion`; `Roots` calls
      `modversion.Canonical`. `closure_test.go` (`fixtureRoots`) calls it too. Drop the
      `strings` import if nothing else uses it. Verify:
      `grep -rn canonicalVersion opm` prints nothing; `go test ./opm/helper/platformmodule
      -count=1` green.
- [ ] 1.4 `task check` green, then commit
      `refactor(helper): share version canonicalisation from an internal package`.

## 2. Registry verbs accept both spellings (loader, sourcetree, kernel; design BS2, BS3, BS4)

- [ ] 2.1 `opm/internal/sourcetree/sourcetree.go`: `SyntheticRoot` canonicalises its version
      argument (design BS3); its doc comment says both spellings give the same root.
      `sourcetree_test.go` `TestSyntheticRoot` asserts
      `SyntheticRoot(p, "0.0.2") == SyntheticRoot(p, "v0.0.2")`. Verify:
      `go test ./opm/internal/sourcetree -count=1` green.
- [ ] 2.2 `opm/internal/loader/registry.go`: `FetchArtifact` canonicalises the version before
      `module.NewVersion` and uses the canonical form for `SyntheticRoot` and later error text;
      the parse-failure error keeps the caller's spelling (design BS2). `FetchModule`
      canonicalises before calling `FetchArtifact` and `verifyModuleIdentity`. The
      `FetchArtifact` doc comment says the version may be bare or `v`-prefixed. Verify:
      `go build ./opm/...` clean.
- [ ] 2.3 `opm/kernel/acquire.go`: the godoc of `AcquireModuleFromRegistry` and
      `AcquireCatalogFromRegistry` say the version may be written either way, e.g. `"4.3.0"` or
      `"v4.3.0"`. Verify: `go doc ./opm/kernel Kernel.AcquireCatalogFromRegistry` shows both forms.
- [ ] 2.4 Loader tests through the `registrytest` fixture (`opm/internal/loader/registry_test.go`):
      `TestFetchModule_BareVersion` fetches a served module with `"0.0.2"` first (so the bare
      spelling drives the registry fetch, not a warmed cache) and then with `"v0.0.2"`,
      asserting both succeed, `metadata.version` reads `0.0.2`, and both sources carry the same
      `Root`; `TestFetchArtifact_CatalogBareVersion` does the same, bare first, for a served
      catalog with `loader.CatalogSpec`. `TestFetchModule_BadVersionWrapped` also asserts the
      exact outer prefix `parsing artifact version test.example/x@v0@not-a-version:`, which
      proves the message keeps the caller's spelling (the wrapped cue error alone already
      contains `not-a-version`). The module scenario is covered at the loader because
      `Kernel.AcquireModuleFromRegistry` delegates to `loader.FetchModule` in one line.
      Verify: `go test ./opm/internal/loader -count=1` green.
- [ ] 2.5 Kernel test (`opm/kernel/acquire_catalog_test.go`): in
      `TestKernel_AcquireCatalogFromRegistry`, acquire the catalog with the bare `version`
      before the mapped `v`-prefixed acquisition and assert the same `Metadata` and the same
      `Source.Root` (spec scenario "A catalog is acquired by a bare version"). Keep the
      unmapped negative subtest first, before any positive load warms the cache. Verify:
      `go test ./opm/kernel -run 'TestKernel_AcquireCatalogFromRegistry' -count=1` green.
- [ ] 2.6 Negative check, not committed: revert only the canonicalisation line in
      `FetchArtifact` and rerun 2.4 and 2.5; the two catalog bare-version tests
      (`TestFetchArtifact_CatalogBareVersion` and the 2.5 kernel test) fail with "not well
      formed", while `TestFetchModule_BareVersion` stays green because `FetchModule`
      canonicalises first. Then revert the `FetchModule` line as well; now all three bare-version
      tests fail. Restore both. Verify: `git diff --stat` shows the restored file unchanged from 2.2.
- [ ] 2.7 `task check` green, then commit
      `fix(loader): accept bare semver in the registry verbs`.

## 3. Verify and archive

- [ ] 3.1 Whole-tree gates on the final tree: `task check` and
      `go test -race ./opm/kernel ./opm/internal/renderstage -count=1`. Verify: all green.
- [ ] 3.2 `openspec validate accept-bare-semver-in-registry-verbs --strict` passes.
- [ ] 3.3 Archive on this branch so the archive rides the implementing PR:
      `openspec archive accept-bare-semver-in-registry-verbs --yes`. Verify:
      `openspec/specs/registry-module-loading/spec.md` carries the new requirement with its four
      scenarios, and `openspec validate --all --strict` passes. There is no `enhancement.yaml`,
      so no delivery log runs.
- [ ] 3.4 Commit `chore(openspec): archive accept-bare-semver-in-registry-verbs`.
