## Context

See proposal.md, Why. The relevant current state:

- **Parity.** `TestParity_ShippedCatalog` (`opm/kernel/parity_harness_test.go:45-99`) builds the
  oracle from `testdata/parity` (`loadOracle(t, parityDir, "./shipped", ...)`, line 71). It builds
  the kernel side from `testdata/parity/opm_platform`, which is a separate module (line 65). It
  already ties the two together: it reads `catalogVersion` off the oracle
  (`testdata/parity/shipped/shipped.cue:27`, `catalog.metadata.version`) and requires the
  kernel's `ResolvedVersions` row for `opmodel.dev/catalogs/opm@v4` to carry it on both sides
  (lines 78-86). The only literal left is in `shippedCases` (lines 150-196): seven rows of
  `shippedCatalogPrefix + "<name>@4.4.2"`. The probe group (`parity_probe_test.go:102-107`) is
  hermetic (`version = "0.1.0"` on a `registrytest.UniquePath`) and out of scope.
- **Catalog pins.** Four modules in `CUE_MODULE_GLOBS` (`Taskfile.yml:17-21`) pin
  `opmodel.dev/catalogs/opm@v4` at `v4.4.2`: `modules/opm_platform`, `testdata/modules/web_app`,
  `testdata/parity` and `testdata/parity/opm_platform`. All four pin core `v2.0.0-beta.1` and
  `cue.dev/x/k8s.io@v0` `v0.12.0`. The flow tests match transformer ids by substring without a
  version (`flow_integration_test.go:121-160`).
- **Core.** `schema.DefaultSchemaModule = "opmodel.dev/core@v2.0.0-beta.1"`
  (`opm/schema/loader.go:44`) and `registrytest.DefaultCoreVersion = "v2.0.0-beta.1"`
  (`opm/internal/registrytest/registrytest.go:113`) are two literals.
  `TestDefaultCoreVersion_IsTheDefaultSchemaRelease` (`registrytest_test.go:42-44`) asserts they
  are equal. `registrytest` already imports `opm/schema` (`registrytest.go:42`), and `opm/schema`
  imports no `internal/registrytest` (checked with `go list -deps`), so deriving one from the
  other adds no import cycle. Several tests already derive the current pin:
  `render_collision_test.go:164,202`, `render_core_floor_test.go:65`,
  `platform/contracts_test.go:229,567-568`, `acquire_catalog_test.go:104,197,246`,
  `integration_fixtures_test.go:85,112`, and `platformmodule/build_test.go:60`.
- **Update task.** `cue:deps:update` (`Taskfile.yml:268-339`) loops over the dependencies one at
  a time with `cue mod get "$dep" >/dev/null 2>&1 || true`, then runs
  `cue mod tidy >/dev/null 2>&1 || true` (lines 315-321). The root workspace task explains in its
  own comment (`../Taskfile.yml:89-96`) why per-dependency gets silently no-op when siblings are
  stale. The root task also swallows errors, but fixing that is the workspace's job (W1), not
  this change's.

## Goals / Non-Goals

**Goals:**

- A catalog bump touches only `cue.mod/module.cue` files.
- A core bump touches only `opm/schema/loader.go` (the constant and its doc-comment examples)
  and `cue.mod/module.cue` files (the `CUE_MODULE_GLOBS` set, plus the text-pinned
  `testdata/cue.mod` and `testdata/render/**`).
- Every version literal that stays in test code is there on purpose and is listed in
  `.cascade-frozen` with a reason.

**Non-Goals:**

- No change to what any test asserts about kernel behavior. Only the source of a version
  string changes.
- No guard against new literals creeping back (follow-up in `add-deps-cascade-task`).
- No change to the root workspace `deps:update` (W1) or to `cue:catalog:drift`.
- No change to the text re-pin of `testdata/render/**`. Those modules are served in-process and
  are not in `CUE_MODULE_GLOBS`, so `cue mod get` cannot resolve them. The cascade rewrites their
  core line as text (workspace RELEASING.md, section "Cascade files"), and
  `TestDefaultCoreVersion_IsTheDefaultSchemaRelease` already checks the result.

## Decisions

### D1. Parity reads the pin from `testdata/parity/cue.mod/module.cue`

The table keeps a `Transformer` field, but it holds the bare name (`"configmap-transformer"`).
A helper builds the full id at run time:

```go
// shippedCatalogVersion is the catalogs/opm build the parity module pins: the
// module the oracle builds in. Bumping the pin is the whole catalog bump.
func shippedCatalogVersion(t *testing.T, parityDir string) string {
	t.Helper()
	src, err := os.ReadFile(filepath.Join(parityDir, "cue.mod", "module.cue"))
	require.NoError(t, err)
	mf, err := modfile.Parse(src, "cue.mod/module.cue")
	require.NoError(t, err)
	dep, ok := mf.Deps["opmodel.dev/catalogs/opm@v4"]
	require.True(t, ok, "the parity module pins no opmodel.dev/catalogs/opm@v4")
	return strings.TrimPrefix(dep.Version, "v")
}

func shippedTransformer(version, name string) string {
	return shippedCatalogPrefix + name + "@" + version
}
```

`TestParity_ShippedCatalog` reads the version once. Before it compares any case, it asserts
`oracleCatalog == pinned`, with a message that names both versions and says resolution lifted
the pin. It then resolves each row's id before `assertRowsCoverPairs` and the per-case loop.
`shippedCases` stays a package-level table. Rows carry the bare name, and a small loop builds a
resolved copy, so the table still reads as data.

**Alternatives.** (a) Use the oracle's `catalogVersion` directly. This needs no file read, but it
trusts whatever resolution produced. If MVS lifted the pin, the harness would silently test a
build nobody pinned. Reading the pin and asserting equality turns that into a named failure.
(b) Read `testdata/parity/opm_platform/cue.mod`. The existing `ResolvedVersions` assertion
already ties the platform side to the oracle, so either module would do. The oracle's module is
the reference by spec ("Pure-CUE unification is the render oracle"), so its pin is the one to
read. (c) Use `modules/opm_platform`. The parity harness does not build there (only the flow test
does), so no.

`cuelang.org/go/mod/modfile` is already a dependency (`opm/catalog/requires.go:7`). No new import
enters `go.mod`.

### D2. `registrytest.DefaultCoreVersion` derives from the schema default

```go
// DefaultCoreVersion is the opmodel.dev/core version every registrytest
// fixture declares ... It is the release [schema.DefaultSchemaModule] pins.
var DefaultCoreVersion = schema.DefaultSchemaVersion()
```

A package-level `var` in an `internal/` test-support package is not mutable state in the
Principle I sense. It is set once from a constant and never written. No caller uses it in a
constant expression (checked: every use is string concatenation, `fmt` or `assert`). The first
assertion of `TestDefaultCoreVersion_IsTheDefaultSchemaRelease` becomes true by construction and
is removed. The fixture walk over `testdata/render` stays, because it is the guard the text
re-pin needs.

**Alternative.** Keep two literals and the equality test. That is the status quo, and it is one
more file for every core bump. Rejected.

### D3. Current-core literals read `registrytest.DefaultCoreVersion`; the loader pin becomes structural

| File:line | Literal | Means | Action |
| --- | --- | --- | --- |
| `opm/kernel/render_test.go:157,206,491,526,992,1059` | `ResolvedVersions` core rows `v2.0.0-beta.1` | current core resolved from the served fixtures | derive |
| `opm/kernel/render_test.go:906` | authored instance `cue.mod` core pin | current core, resolved by the build | derive |
| `opm/kernel/render_test.go:32` | comment "pins core 2.0.0-beta.1" | prose | reword to name `DefaultCoreVersion` |
| `opm/internal/renderstage/stage_test.go:345,569` | authored module files, built | current core | derive |
| `opm/internal/renderstage/stage_test.go:475` | local-file core pin on the synthetic `platformModFile` | synthetic (spike decides) | derive or freeze per section 1 |
| `opm/schema/loader_test.go:167,176` | `DefaultSchemaModule` / `DefaultSchemaVersion()` pin assertion | tripwire | structural: `HasPrefix(DefaultSchemaModule, "opmodel.dev/core@v2.")`, `DefaultSchemaModule == "opmodel.dev/core@" + DefaultSchemaVersion()`, `semver.IsValid` and not the bare major |
| `opm/schema/loader_test.go:194,195` | parse-table rows | synthetic parse inputs | freeze |
| `opm/catalog/requires_test.go:18,41` | `test.example` module file read from an overlay | synthetic, never resolved | freeze |
| `opm/internal/renderstage/modfile_test.go` (11 core, 15 catalog) | in-memory module files and skew rows (`alpha.6`, `4.1.0`-`4.3.0`) | synthetic parse and compare | freeze |
| `opm/helper/platformmodule/generate_test.go:22,227,234` and `4.0.1` rows | golden `coreVersion = alpha.7`; "explicit core pin" | synthetic text generation | freeze |
| `opm/helper/platformmodule/closure_test.go` | `alpha.6/7/8`, `4.0.1`, `4.9.9` graph | synthetic dependency graph | freeze |
| `opm/helper/objectset/objectset_test.go:29` | `transformer-registration-transformer@4.4.0` | synthetic id on an in-memory value | freeze |
| `opm/internal/registrytest/registrytest_test.go:24-27` | `Major` parse table | synthetic parse inputs | freeze |
| `opm/kernel/render_collision_test.go:166,204` | `alpha.12` | core without the collision report | freeze |
| `opm/kernel/render_core_floor_test.go:43-44,68,103` | `alpha.10`, `alpha.12` | below and at the render floor | freeze |
| `opm/platform/contracts_test.go:372-442,568` | `alpha.9` to `alpha.12` | floor and pre-collision semantics | freeze |
| `opm/errors/coretooold_test.go` | `alpha.10`, `alpha.12` | error text | freeze |

The plan counted about 28 literals in seven files that need hand edits. This table says only
`render_test.go`, `stage_test.go` and `loader_test.go` hold literals that must move for a bump to
stay green. `modfile_test.go` and `requires_test.go` were edited in `6081afd` for consistency,
not because they failed. That is an **unverified assumption**, so section 1 is a spike that
checks it by mutation (D6). `generate_test.go:227,234` ("explicit core pin") currently equals the
default. That makes the test weaker than its name, because an explicit pin should differ from the
default. The literal moves to `v2.0.0-alpha.13`, a frozen non-default release, so the test proves
the explicit pin wins.

**Alternative.** Derive every `v2.0.0-beta.1` in test code, synthetic ones included, as the plan
listed. Rejected. It threads `fmt.Sprintf` or concatenation through about 15 raw-string module
files whose version does not matter, and it hides the one distinction worth keeping: which
literals mean "now".

### D4. Comments stop naming the release

The 13 fixture platform headers (`testdata/render/platform*/platform.cue:1`,
`modules/opm_platform/platform.cue:2`) say "core 2.0.0-beta.1". They become "the core its
cue.mod pins". The `AGENTS.md` layout line for `testdata/` ("all pinned to core 2.0.0-beta.1")
becomes "all pinned to the default core release (`schema.DefaultSchemaModule`)". The CUE edit is
comments only, and `task cue:fmt` and `task cue:vet` must leave no diff. `testdata/modules/web_app`
is checksum-tracked in `cue-versions.yml`, but its `platform.cue` is not touched.
`docs/getting-started.md:51-56` and `AGENTS.md:329` keep their explicit pin, because they
illustrate a caller choosing a release.

### D5. `cue:deps:update`: one pass, core held, loud

Per module:

```bash
default_core=$(grep -oP 'DefaultSchemaModule = "opmodel\.dev/core@\K[^"]+' opm/schema/loader.go)
# every direct dep: core at the default, everything else at its major's latest
args=()
for dep in $deps; do
  case "$dep" in
    opmodel.dev/core@v2) args+=("opmodel.dev/core@${default_core}") ;;
    *)                   args+=("$dep") ;;           # "<path>@vN" = newest on that major
  esac
done
if ! out=$(cd "$dir" && cue mod get "${args[@]}" 2>&1 && cue mod tidy 2>&1); then
  printf '%s: cue mod get/tidy failed:\n%s\n' "$dir" "$out" >&2; exit 1
fi
got=$(cue export "$dir/cue.mod/module.cue" --out json | jq -r '.deps["opmodel.dev/core@v2"].v // empty')
if [ -n "$got" ] && [ "$got" != "$default_core" ]; then
  printf '%s: core resolved to %s, default is %s; a dependency needs a newer core: advance schema.DefaultSchemaModule first\n' \
    "$dir" "$got" "$default_core" >&2; exit 1
fi
```

The old-to-new report stays. CUE v0.17.1 has no `cue mod graph` (checked: `cue mod --help` lists
edit, fix, get, init, mirror, publish, registry, rename, resolve and tidy). So, to name the
dependency that forced a core lift, the task reads each moved dependency's own module file from
the cache: `$(cue env CUE_CACHE_DIR)/mod/extract/<path>@<new>/cue.mod/module.cue`. It names every
moved dependency whose own `opmodel.dev/core@v2` requirement is newer than the default. `set -euo
pipefail` already applies. The task keeps processing modules in glob order and stops at the first
failure.

The stale comment on `.github/workflows/cue.yml:39-42` ("older than the latest published") is
corrected to match the task (existence, not currency, `Taskfile.yml:345-361`).

**Alternatives.** (a) Leave core out of the get. Then an unconstrained MVS lift still happens
silently when a catalog needs a newer core. (b) Let core float to the newest release. Then the parity and
flow modules test a core the kernel's default never renders against, which is the drift
"Served fixtures pin the default release" forbids for the render fixtures. Both rejected.

### D6. Section 1 spike: classify by mutation

The spike runs in the worktree and the source edits are reverted before its commit. Each
literal the D3 table marks *freeze (synthetic)* is replaced with `v2.0.0-beta.99` (core) or
`4.99.0` (catalog), and the owning package's tests run with `-count=1`. A package that stays
green confirms that the literal is synthetic. A package that fails moves the literal to *derive*.
The reverse check runs too: each *derive* row is mutated the same way and must fail, which shows
that derivation is needed and not cosmetic. The result is written into this design as a table
with one row per literal and its outcome. That table becomes the input for `.cascade-frozen`.

## Research & Decisions

### Where the catalog version should come from

**Context**: Seven literals have broken parity on every catalog bump (T1).
**Explored**: `parity_harness_test.go:45-196`, `testdata/parity/shipped/shipped.cue:27`, the
four catalog-pinning cue.mods, and the library bump commits `588a638`, `ae9e247`, `fad87fd` and
`7f0dd5b` (each edits the same 14 lines of the harness).
**Decision**: D1. Read the pin from the oracle's module and assert that resolution equals it.
**Rationale**: The pin is what a bump edits. The resolved build is what was tested. Asserting
that the two are equal keeps the harness from testing a build no one chose.

### Which core literals are load-bearing

**Context**: The plan estimated about 28 hand-edited literals per core bump.
**Explored**: Every `2.0.0-(alpha|beta).N` in `opm/**/*_test.go` (recounted: 78 lines in 12
files), and the `6081afd` file list.
**Decision**: D3. Derive the literals that resolve against or compare to the fixtures (seven in
`render_test.go`, two or three in `stage_test.go`), make the two `loader_test.go` pin assertions
structural, freeze the rest, and verify by mutation (D6).
**Rationale**: Only a literal that a bump breaks belongs in the cascade's path. Rewriting
synthetic text adds churn without adding safety.

### Single-pass update and core handling

**Context**: `|| true` per dependency hides both staleness and real errors.
**Explored**: The root `Taskfile.yml:62-116` (one-pass `cue mod get $deps`) and
`.tasks/deps/fixtures.sh:38`.
**Decision**: D5.
**Rationale**: It matches the root task's proven resolution behavior and adds the loud failure
the root task still lacks. Holding core at the default implements the consistent-set rule
locally: the parity catalog moves only to builds on the library's own core (workspace
RELEASING.md, section "Bump rule").

## Risks / Trade-offs

- [Catalog `4.4.4` renders differently from `4.4.2`, so parity or flow fails in section 5] →
  `4.4.3` and `4.4.4` are the catalog's core-beta cutover releases (catalog_opm `b35d644`, which
  moved the catalog onto core `v2.0.0-beta.1`). A rendered-output change is unlikely but
  possible. If it happens, section 5 stops, the divergence is written into Risks here, and the
  bump becomes its own change. Sections 1 to 4 stand on their own.
- [`cue mod get` in section 5 moves `cue.dev/x/k8s.io` too] → Allowed. It is a test pin and lands
  in the same `test(fixtures)` commit. The verification checks the suite, not a one-line diff. If
  it breaks, the task's own fix applies: hold that dependency with an explicit version in the
  get and record why.
- [`DefaultCoreVersion` becomes a `var`] → Any future constant-expression use fails at compile
  time, which is loud. No exported surface outside `internal/`.
- [The structural loader assertion lets a default bump through with no test edit] → Intended
  (D11 makes the bump PR carry `need-human-review`). Deliberateness moves from a test literal to
  review.
- [Full-suite flake in `TestGenerate_BuildsThroughTheKernel`] → The eviction race was fixed
  (test-fixture-registry spec, "The shared workspace cache is never deleted from"). If it
  reappears, rerun the package alone before treating it as caused by this change.

## Migration Plan

None. Test-only. Rollback is a revert of the PR.

## Open Questions

None. The owner settled the approach in decision D8, and the one factual unknown (which literals
are synthetic) is the section 1 spike.
