## Context

See proposal.md, Why. Line numbers are at library `origin/main` `a8bfc76` (#183). Design-local decisions are numbered PS1 to PS7 so they collide with no other numbering.

Core's half is core#120 (`517ae7ba`), released as core `v2.0.0-beta.3`. In `src/catalog.cue`, `#Catalog` gains:

```cue
let providerSet = {
	for _, tf in #transformers {
		if tf.requiredResources != _|_ {
			for fqn, req in tf.requiredResources if req.fulfilment == "provider" {(fqn): true}
		}
		if tf.requiredTraits != _|_ {
			for fqn, req in tf.requiredTraits if req.fulfilment == "provider" {(fqn): true}
		}
	}
}
provides: [...#ContractFQNType] & list.Sort([for fqn, _ in providerSet {fqn}], list.Ascending)
```

`provides` is a regular field at the catalog root, so a catalog package that embeds `c.#Catalog` carries it at `Package.LookupPath("provides")`.

Absence alone does not identify an older core. Real catalogs embed `c.#Catalog` at file level (`catalog_opm/src/catalog.cue:25`, every `testdata/render/registry` catalog), and an embedding admits fields declared beside it. Checked with cue v0.17.1: `x: {#Catalog, provides: ["fake"]}` evaluates, while `#Catalog & {provides: ["fake"]}` is refused. So a catalog built against core `v2.0.0-beta.2` or older can author any `provides`. Against core `v2.0.0-beta.3` or later an authored value conflicts with the computed list, unless the two are equal. The gap therefore exists only on old cores, and only core age closes it.

Today's fold, `opm/catalog/provides.go:44-107` (with `collectProviders`), walks the same two demand maps and returns a sorted, deduplicated, non-nil slice. It errors only when a `fulfilment` is present but not a concrete string.

## Goals / Non-Goals

**Goals:**

- The library builds against core `v2.0.0-beta.3`, with every core pin in the tree moved together.
- `Provides()` answers from core's field when it is there, and from the deprecated fold when it is not.
- A parity test shows the two agree on every provider fixture.
- A test shows an old catalog still gets the fold's answer.

**Non-Goals:**

- Removing the fold. That is a later change, before GA, after catalog_opm is republished.
- Any change to `Provides()`'s signature or result, or to the operator's 0015:D11 claim check.
- Republishing catalog_opm and the operator's old-catalog test; each is a separate change in its own repo.
- A floor. An old catalog is answered, not refused.

## Decisions

### PS1: The bump runs through `task -x deps:cascade`

`task -x deps:cascade` (`Taskfile.yml`, `deps:cascade`; `.tasks/cascade/cascade.sh`) is the task the release cascade's bot PR runs. It moves `DefaultSchemaModule`, then the core pin of `testdata/cue.mod` and every `testdata/render` tree as text, then the `CUE_MODULE_GLOBS` modules through `cue mod get` and `tidy`, and then the two docs examples (step C4). It honours `.cascade-frozen`, whose entries keep the deliberately old cores in the floor and collision tests. Running it in the worktree produces the same diff a live cascade PR would. The cascade is in dry-run, so no bot PR will.

The same run moves `opmodel.dev/catalogs/opm@v4` to the newest published build the new core allows. That is the cascade's normal behaviour. The library pins v4.5.1. Builds v4.5.2 and v4.6.0 are published and both pin core `v2.0.0-beta.1`, so the cascade allows them, and catalog_opm#145 (4.7.0) is open. Expect the move v4.5.1 -> v4.6.0, which includes the seccomp change (catalog_opm `9bdafe8`), or 4.7.0 if #145 merges first.

The parity harness has no catalog-version literals. derive-fixture-versions (archived 2026-10-02) retired them, and `shippedCatalogVersion` (`opm/kernel/parity_harness_test.go:148-159`) reads the pin from `testdata/parity/cue.mod`. Bumping the pin is the whole catalog bump. The bare-name `shippedCases` rows are edited only if `assertRowsCoverPairs` or `TestParity_ShippedCatalogDiscriminated` fails at the new build.

`task cue:deps:update` with `DEFAULT_CORE=v2.0.0-beta.3`, after hand-editing `loader.go`, is the fallback if the resolver cannot run. It does not move the docs examples, so those are then edited by hand.

`registrytest.DefaultCoreVersion` derives from `DefaultSchemaModule` (`opm/internal/registrytest/registrytest.go:115`), and `writeCatalogDir` in `opm/kernel/acquire_catalog_test.go` reads it, so neither needs an edit. The plan entry lists `acquire_catalog_test.go:203,278` because those assertions now run through the decode path once the pin moves; their expected values do not change.

### PS2: Core age decides first, presence second

```go
func (c *Catalog) Provides() ([]string, error) {
	if c == nil {
		return nil, fmt.Errorf("catalog is nil")
	}
	// Shared by both paths: a #transformers that does not evaluate is an error.
	if tf := c.Package.LookupPath(schema.Transformers); tf.Exists() && tf.Err() != nil {
		return nil, fmt.Errorf("catalog %s did not evaluate: %w", schema.Transformers, tf.Err())
	}
	old, err := c.pinsCoreBeforeProvides()
	if err != nil {
		return nil, err
	}
	field := c.Package.LookupPath(schema.CatalogProvides)
	if old || !field.Exists() {
		return c.providesFold()
	}
	var fqns []string
	if err := field.Decode(&fqns); err != nil {
		return nil, fmt.Errorf("reading catalog %s: %w", schema.CatalogProvides, err)
	}
	return sortedUnique(fqns), nil // sorted, deduplicated, non-nil
}
```

- **Core age decides when the catalog carries a Source.** `pinsCoreBeforeProvides` reads the committed core pin with `c.Requires()` (`modversion.CorePath`) and reports whether it is older than `schema.ProvidesSince` (`modversion.Compare`). This is the owner's wording: h2 says the fold answers "for older catalogs", which is about core age. It also closes the gap the Context describes: a catalog built against core `v2.0.0-beta.2` or older that authors a `provides` beside its embedded `#Catalog` still gets the fold's derived answer. Without that rule, the operator's 0015:D11 claim check would compare a claim against an authored list.
- **Presence decides only when core age is unknown.** That covers a catalog with no Source (`NewCatalogFromValue`), a Source with no core requirement, and a core requirement carrying no version (a local replacement). Such a value is answered from `provides` when present, and by the fold otherwise.
- **An unreadable committed module file is an error.** A catalog carrying a Source whose `cue.mod/module.cue` `Requires()` refuses is reported, not silently treated as "age unknown". Acquisition already parsed that file, so this does not happen in practice, but a silent fallback would hide a defect. An invalid version string is reported the same way.
- **The `#transformers` evaluation check runs first and is shared.** Core's comprehension guards each demand map with `!= _|_`, so a bottom demand map could be left out of `provides` when the fold would report it. Running the check before either path keeps one error rule.
- **`Decode` into `[]string`.** It refuses a non-list, a list with a non-string element, a non-concrete element (`[string]` gives `incomplete value`) and an errored field. The `Exists` check comes first because `LookupPath` on a missing field returns a non-existent value, not an error.
- **The decoded field always wins when it is chosen.** The fold does not run alongside it in production, so the work is not done twice. The parity test is what shows the two agree.

### PS3: The result is normalised after decoding

Core sorts and deduplicates `provides`. The library still sorts and deduplicates the decoded slice, and returns `[]string{}` rather than `nil` for an empty list. The doc comment promises deterministic, deduplicated, non-nil output for any `*Catalog`. `NewCatalogFromValue` also accepts values that core did not build, so the promise must hold there too. On core's output the step is the identity, and the parity test asserts that the decoded field is already in order before normalising.

### PS4: Error handling matches the fold's rule

Both paths follow one rule: a provider set that cannot be read is an error with no partial set, never an empty set.

- The fold errors on a present but non-concrete `fulfilment`.
- In core, the same input makes the `provides` comprehension incomplete. The decode then fails, and the error is wrapped as `reading catalog provides: <cause>`.

The two messages differ. The fold names the contract and the transformer; the decode names the field, and CUE's cause carries the path. `TestCatalog_Provides_NonConcreteFulfilmentIsReported` keeps its contract and transformer asserts, because its literal has no `provides` and so pins the fold. A new decode case asserts the error, the nil result and the field name.

Through real core, a demand entry's `fulfilment` defaults to `"catalog"`, so a non-concrete `fulfilment` cannot reach a built catalog. The decode's error path is therefore exercised with CUE literals that author `provides` directly.

### PS5: The fold is kept unexported and marked Deprecated

`collectProviders` and the walk move into `providesFold` in the same file. Its doc comment opens with a `Deprecated:` paragraph. That paragraph says it is the fallback for catalogs built against a core older than `schema.ProvidesSince`. It also says the fold is removed before GA, after catalog_opm is republished against a core that derives `provides`, and that `Provides()` is the entry point. The function is unexported, so the paragraph does not mark any public API; it records the removal condition at the code. No exported symbol is removed or renamed.

The schema paths the fold reads (`schema.Transformers`, `RequiredResources`, `RequiredTraits`, `Fulfilment`) are exported. Their doc comment in `opm/schema/paths.go:61-71` changes from "Their one reader is (*catalog.Catalog).Provides" to "the deprecated fallback inside it". They stay; deleting them is the fold-removal change's decision.

A test-only `export_test.go` in `opm/catalog` exposes `ProvidesFold = (*Catalog).providesFold` to the package's external tests, so the parity test can call both paths.

### PS6: Schema constants

In `opm/schema/paths.go`, beside the catalog paths:

```go
// CatalogProvides is #Catalog.provides: the provider-fulfilled contracts the
// catalog implements, sorted and deduplicated, derived by core since
// [ProvidesSince]. (*catalog.Catalog).Provides decodes it.
CatalogProvides = cue.ParsePath("provides")
```

After `CollisionsSince`:

```go
// ProvidesSince is the first core release deriving #Catalog.provides
// ([CatalogProvides]), without the "v" prefix. It documents the field and
// decides which path (*catalog.Catalog).Provides takes for a catalog whose
// committed core pin is known; it is never a floor: a catalog built against
// an older core is answered through the deprecated fold, not refused.
const ProvidesSince = "2.0.0-beta.3"
```

### PS7: Where the tests live

- **Unit cases** stay in `opm/catalog/provides_test.go`. The existing table compiles CUE literals that are not built against core, so it keeps pinning the fold. A new table, `TestCatalog_Provides_DecodesCoreField`, authors `provides` directly. Its cases:
  - a present field wins over a disagreeing `#transformers` on a value with no Source (proof the decode path ran);
  - an unsorted, duplicated field comes back sorted and deduplicated;
  - `provides: []` gives a non-nil empty slice;
  - `provides: [string]` errors with a nil result and names `provides`;
  - `provides: "x"` errors the same way;
  - a `#transformers` that does not evaluate, next to a well-formed `provides: []`, errors with a nil result (the shared check).
- **Parity** goes in a new `opm/catalog/provides_parity_test.go` (package `catalog_test`, test name `TestCatalog_Provides_ParityWithFold`). It acquires real catalogs through `kernel.Kernel`; an external test package may import `opm/kernel`, and depguard fences only `opm/helper` and `opm/k8s` from the kernel tier. Each subtest:
  - requires that the catalog's `provides` exists;
  - requires that the decoded field is already sorted and free of duplicates;
  - asserts the decoded field equals `ProvidesFold` (core's rule and the fold agree);
  - asserts `Provides()` returns the decoded field;
  - rebuilds the catalog from its `metadata` and `provides` alone, keeping its real `Source` (so the committed core pin routes it), and asserts `Provides()` still returns the field. On core's output the field and the fold are equal, so only this check fails for a `Provides()` that always folds.

  The fixtures are:
  1. the unit table's provider shapes, rebuilt as `c.#Catalog` packages pinned at `registrytest.DefaultCoreVersion` and acquired from a directory. Every member and every demand key is keyed by its `metadata.fqn`, so the core bump that binds those keys to `metadata.fqn` (owner decision j3) stays green. They cover both demand arms, a catalog-fulfilled demand, an optional demand, one contract required by two transformers, no transformers, and a transformer with no demand maps. Where a literal shape does not unify with core's `#ComponentTransformer`, it is adjusted to the nearest valid shape that keeps what the case is about. The adjustment is noted in the case name.
  2. every committed `testdata/render/registry` catalog (each directory whose package embeds `c.#Catalog`), served by `registrytest.NewRegistryFromDir` and acquired by path and version. Several of them define provider-fulfilled contracts (`cat`, `maj`, `providers`) and some transformers require them, so the set includes both empty and non-empty answers. The test requires at least one non-empty answer among them, so the fixture tree cannot drift into covering only the empty case.
- **Old catalog** goes in the same file (`TestCatalog_Provides_OldCatalogFallsBackToFold`). It uses a provider catalog directory whose `cue.mod` pins core `v2.0.0-beta.2`, a literal older than `schema.ProvidesSince` (asserted with `modversion.Compare`). Two cases:
  - the catalog authors nothing extra. The acquired package has no `provides`, and `Provides()` returns the expected FQNs and equals `ProvidesFold`;
  - the catalog authors a bogus `provides` beside its embedded `#Catalog`. The acquired package carries it, and `Provides()` still returns the fold's answer, not the authored list.

  Core `v2.0.0-beta.2` resolves from GHCR, or from the user's CUE module cache when it already holds it; Go tests do not read the worktree's `.cue-cache`.
- **Gating.** The parity and old-catalog tests resolve core from GHCR (beta.3 and beta.2). They skip under `-short` and skip when GHCR is unreachable, the way the kernel's registry-backed tests do (`testing.Short`, and the `skipUnlessRegistry` probe with its `OPM_FLOW_TEST_FORCE=1` override). The probe is copied into the catalog test file, because a test helper in `opm/kernel` is not importable.

  The literal is deliberate, so the file gets a `.cascade-frozen` entry for `opmodel.dev/core@v2` with that reason. Otherwise a later cascade run would warn about the literal or rewrite it.
- **The kernel acquisition tests** (`opm/kernel/acquire_catalog_test.go`) are unchanged. After the bump they exercise the decode path end to end with the same expected sets.
- **Unit coverage of the age rule** stays offline: cases build a `*Catalog` with a `Source` over a temp-dir module file, as `requires_test.go` does. An old core pin with an authored `provides` gets the fold. A current pin decodes. An unparseable module file and an invalid version both error.

## Risks / Trade-offs

- **The committed pin can lag the build core.** A tidy `cue.mod` records the MVS-selected core, so the committed pin is the build core. If a catalog's file were untidy and named an older core than its build resolved, the fold would answer instead of the decode. The fold computes the same derived set, so the answer is still right; only the source differs.

- **The catalog pin moves with core.** This brings parity-literal and golden churn into section 1. It is mechanical, and it is the same churn a live cascade PR would carry. Holding the catalog back would need a `.cascade-hold` edit, and the next cascade run would undo it.
- **Two answers for one question until the fold goes.** The decode and the fold could drift if core changes the rule. The parity test fails in that case, which is what ADR-012 asks for. After the fold is removed, the parity test goes with it; the `testdata/render` and kernel tests keep pinning expected sets.
- **`TestGenerate_BuildsThroughTheKernel`** can fail in a full-suite run from a known cross-process cache race. Rerun `go test ./opm/helper/platformmodule -count=1` alone before treating it as a finding.

## Research & Decisions

### Is there a bump PR to build on?

**Context**: The supervisor note says to build on an open deps/cascade PR rather than duplicate one.

**Explored**: `gh pr list -R open-platform-model/library --state open` on 2026-10-05 lists only release PR #192 (`chore(main): release 1.0.0-beta.5`). `.github/workflows/deps-cascade.yml` gates every write on `vars.CASCADE_DRY_RUN == 'false'`, and the cascade is in dry-run.

**Decision**: This change carries the core bump (section 1).

**Rationale**: No PR carries it, and none will while the cascade is in dry-run.

### Does any consumer depend on the fold's error text?

**Context**: PS4 changes the error text when an undecodable `provides` is present.

**Explored**: Callers of `.Provides()` at `origin/main`:
- opm-operator: `internal/controller/transformerregistration_controller.go:323` and `test/integration/reconcile/backup_fixture_test.go:107`;
- library tests.

None matches the error text. The cli has no caller.

**Decision**: The error text is not a compatibility surface. A plain `feat` is the right class.

**Rationale**: The result set is unchanged for every input that evaluates, and an input that does not still errors.
