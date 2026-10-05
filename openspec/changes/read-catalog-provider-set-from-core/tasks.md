# Tasks: read-catalog-provider-set-from-core

Worktree: `library/.claude/worktrees/read-catalog-provider-set-from-core`. Branch: `feat/read-catalog-provider-set-from-core`, from `origin/main` `a8bfc76`.

Setup:

- Seed `.cue-cache` by copying the main checkout's (`cp -a /var/home/emil/dev/open-platform-model/library/.cue-cache <worktree>/`). Never symlink it.
- Run every command inside the worktree, with a private absolute `TMPDIR` (`mktemp -d` under the session scratchpad).
- Export the registry env on two lines:

```bash
export CUE_REGISTRY='opmodel.dev=ghcr.io/open-platform-model,testing.opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'
export OPM_REGISTRY="$CUE_REGISTRY"
```

Known flake: `TestGenerate_BuildsThroughTheKernel` can fail in a full-suite run from a known cross-process cache race. Rerun `go test ./opm/helper/platformmodule -count=1` alone before treating it as a finding.

Commit rules:
- No body line starts with `word(`, and no bare at-sign appears.
- The only trailer is `Co-Authored-By: Claude <noreply@anthropic.com>`.
- Decisions are cited as `NNNN:Dn`, never as a bare `Dn`.

## 1. Pin core v2.0.0-beta.3 (schema, testdata, docs examples; design PS1)

- [ ] 1.1 Run `task -x deps:cascade` in the worktree.
      - It resolves `../.github` through the common git dir. Pass `CASCADE_RESOLVER` with an absolute path if it does not.
      - Expect exit 0 and these edits: `DefaultSchemaModule` reads `"opmodel.dev/core@v2.0.0-beta.3"`; the core pin of `testdata/cue.mod`, every `testdata/render` tree and every `CUE_MODULE_GLOBS` module moves to `v2.0.0-beta.3`; the core examples in `docs/getting-started.md` and `AGENTS.md` follow.
      - Record whether `opmodel.dev/catalogs/opm@v4` moved, and to which build. Read `.git/cascade/warnings` and resolve or record each warning.
      - If the resolver cannot run, fall back to `DEFAULT_CORE=v2.0.0-beta.3 task cue:deps:update`, after editing `opm/schema/loader.go:43` by hand, and edit the two docs examples by hand.
      - Verify: `grep -rn 'v2.0.0-beta.2' --include=module.cue testdata modules` prints nothing. `.cascade-frozen` files keep their deliberate old cores.
- [ ] 1.2 Check the derived pins.
      - `go test ./opm/internal/registrytest ./opm/schema ./opm/internal/loader -count=1` is green. `TestDefaultCoreVersion_IsTheDefaultSchemaRelease` reads the new default.
      - `grep -rn 'beta\.2' opm --include=*.go` shows only `.cascade-frozen` files and the version-comparison tables (`modversion_test.go`, `renderstage/modfile_test.go`), not a core pin.
- [ ] 1.3 Catalog pin. No literal edit is needed: the parity harness reads the shipped build from `testdata/parity/cue.mod` (`shippedCatalogVersion`).
      - Expect 1.1 to move `opmodel.dev/catalogs/opm@v4` v4.5.1 -> v4.6.0, or 4.7.0 if catalog_opm#145 merged first. Record the build.
      - Run `go test ./opm/kernel -run 'TestParity_Shipped' -count=1`. Edit the bare-name `shippedCases` rows only if `assertRowsCoverPairs` or `TestParity_ShippedCatalogDiscriminated` fails at the new build.
- [ ] 1.4 Run `go test ./opm/kernel -run 'TestKernel_AcquireCatalog' -count=1`. It is green with unchanged expectations. These tests now run against a core whose catalogs carry `provides`, and the fold still answers.
- [ ] 1.5 `task check` green (fmt, vet, lint, test, the offline api-diff fixture test and the cascade wiring check). Then run `task api:diff` explicitly (it diffs this tree against the base tag; `task check` does not) and record that its Allowed section shows the `DefaultSchemaModule` beta.2 -> beta.3 line. Then commit `fix(deps): pin core v2.0.0-beta.3`. If the catalog moved, the body adds one line naming the catalog build and the parity literals.

## 2. Read core's provider set, with the fold as a deprecated fallback (schema, catalog; design PS2-PS7)

- [ ] 2.1 `opm/schema/paths.go`:
      - Add `CatalogProvides = cue.ParsePath("provides")` beside the catalog paths, and `const ProvidesSince = "2.0.0-beta.3"` after `CollisionsSince`, each with the doc comment design PS6 gives.
      - Reword the catalog-paths comment so the paths the fold reads name "the deprecated fallback inside (*catalog.Catalog).Provides" as their reader.
      - Verify: `go build ./opm/...` clean.
- [ ] 2.2 `opm/catalog/provides.go`, per design PS2, PS3 and PS5:
      - `Provides()` checks the receiver, then keeps today's `#transformers` `Err()` check (shared by both paths).
      - When the catalog carries a Source, it reads the committed core pin with `c.Requires()` (`modversion.CorePath`). A pin older than `schema.ProvidesSince` returns `c.providesFold()`, even when the catalog authors `provides`. A module file `Requires()` refuses, or an invalid version, is an error.
      - Otherwise, when `schema.CatalogProvides` is absent, it returns `c.providesFold()`.
      - When the field is chosen, it decodes into `[]string`, wraps a decode error as `reading catalog provides: %w` with a nil result, and returns the slice sorted, deduplicated and non-nil.
      - Move today's walk and `collectProviders` into the unexported `providesFold`, unchanged in behaviour. Its doc comment opens with a `Deprecated:` paragraph: it is the fallback for catalogs built against a core older than `schema.ProvidesSince`, and it is removed before GA, after catalog_opm is republished against a core that derives `provides`.
      - Rewrite the `Provides` doc comment to say where the answer comes from. Keep its ordering, empty-set and error contract. Keep ADR-012 out of the doc comment (AGENTS.md: no maintainer pointers in doc comments); cite it in a non-doc comment inside the function body.
      - Verify: `go build ./opm/...` clean, and `go test ./opm/catalog -count=1` green with the existing tables unchanged.
- [ ] 2.3 `opm/catalog/export_test.go` (package `catalog`): `var ProvidesFold = (*Catalog).providesFold`.
- [ ] 2.4 `opm/catalog/provides_test.go`: add `TestCatalog_Provides_DecodesCoreField` with the cases design PS7 lists:
      - present field wins over disagreeing transformers;
      - unsorted and duplicated input is normalised;
      - `provides: []` gives a non-nil empty slice;
      - `provides: [string]` errors with a nil result and a message containing `provides`;
      - a wrongly typed `provides` errors the same way;
      - age rule, offline, over a `Source` with a temp-dir module file: an old core pin with an authored `provides` gets the fold; a current pin decodes the field; an unparseable module file errors; an invalid core version errors.

      Update the comment on `newCatalog`: the literal table pins the fallback because its values carry no `provides`. Spec scenarios: "Core's provider set is read when the catalog carries it" and "An unreadable provider set is reported".

      Negative check, not committed: make `Provides()` always call the fold, and confirm the "present field wins" case fails.

      Verify: `go test ./opm/catalog -count=1` green.
- [ ] 2.5 `opm/catalog/provides_parity_test.go` (package `catalog_test`): `TestCatalog_Provides_ParityWithFold`, over the two fixture groups in design PS7.
      - Group 1: the unit provider shapes rebuilt as `c.#Catalog` directories pinned at `registrytest.DefaultCoreVersion` and acquired with `AcquireCatalogFromDir`. Key every member and demand by its `metadata.fqn` (SD13, core-j3).
      - Group 2: every `testdata/render/registry` catalog, served with `registrytest.NewRegistryFromDir` and acquired from the registry.
      - Each subtest requires `provides` to exist and the decoded field to be sorted and unique already, then asserts `Provides()` equals `catalog.ProvidesFold`.
      - At least one group-2 answer is non-empty.
      - Gated like the kernel's registry tests: skip under `-short`, and skip when GHCR is unreachable unless `OPM_FLOW_TEST_FORCE=1`.
      - Spec scenario: "Core's provider set and the fallback agree".
      - Verify: `go test ./opm/catalog -run ParityWithFold -count=1` green.
- [ ] 2.6 Same file: `TestCatalog_Provides_OldCatalogFallsBackToFold`.
      - Use a provider catalog directory whose `cue.mod` pins core `v2.0.0-beta.2`. The test asserts `modversion.Compare("v2.0.0-beta.2", "v"+schema.ProvidesSince) < 0`.
      - Case 1: assert the acquired `Package` has no `provides`, and that `Provides()` returns the expected FQNs and equals `catalog.ProvidesFold`.
      - Case 2: the same catalog authors a bogus `provides` beside its embedded `#Catalog`. Assert the `Package` carries it, and that `Provides()` still returns the fold's answer.
      - Same gating as 2.5.
      - Spec scenario: "A catalog built against an older core falls back".
      - Add a `.cascade-frozen` entry at the end of the file for this test file, pin `opmodel.dev/core@v2`, with the reason "Builds a catalog against core v2.0.0-beta.2, the release before #Catalog.provides, so the old core is the point."
      - Verify: the test is green from the copied `.cue-cache`. `task cascade:wiring:check` and `CASCADE_TEST_SET=offline task -x deps:cascade:test` still pass.
- [ ] 2.7 Docs. In `README.md:45` and `AGENTS.md` (the `catalog/` layout line, located by text), describe `Provides()` as reading core's `provides`, falling back to a deprecated fold for catalogs built against an older core. `adr/012-matching-stays-in-the-library-glue.md` gets one sentence saying the per-catalog provider set moved in core `v2.0.0-beta.3`, and that the library's fold is the deprecated fallback. Verify: `task docs:bundle:check` green.
- [ ] 2.8 Whole-tree checks on the final tree:
      - `task check`;
      - `go test -race ./opm/catalog ./opm/kernel -run 'Provides|AcquireCatalog' -count=1`;
      - `openspec validate read-catalog-provider-set-from-core --strict`.

      Consumer compile check, not committed: build the cli and opm-operator against this tree with a scratch `-modfile` that carries a `replace` to the worktree (`go build -C <repo> -modfile <scratch> ./...`), and run `go test -C /var/home/emil/dev/open-platform-model/opm-operator -modfile <scratch> ./internal/controller -run TransformerRegistration -count=1`. Both build, and the operator test is green.
- [ ] 2.9 `task check` green, then commit `feat(catalog): read the provider set core derives for a catalog`, with a body line saying the Go fold stays as a deprecated fallback for catalogs built against core older than v2.0.0-beta.3.

## 3. Archive (at PR time, on the supervisor's word)

- [ ] 3.1 Run `openspec archive read-catalog-provider-set-from-core --yes` on this branch, so the archive rides the implementing PR. Verify: the `catalog-acquisition` main spec carries the modified requirement, and `openspec validate --all --strict` passes. There is no `enhancement.yaml`, so no delivery log runs.
- [ ] 3.2 Commit `chore(openspec): archive read-catalog-provider-set-from-core`.
