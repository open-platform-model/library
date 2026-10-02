## Context

See proposal.md, Why. Design-local decisions are numbered DF1 to DF6, so they do not collide
with any other numbering. The relevant current state:

- **Parity.** `TestParity_ShippedCatalog` (`opm/kernel/parity_harness_test.go:45-99`) builds the
  oracle from `testdata/parity` (`loadOracle(t, parityDir, "./shipped", ...)`, line 71). It builds
  the kernel side from `testdata/parity/opm_platform`, which is a separate module (line 65). The
  instance package `testdata/parity/instance` has no cue.mod of its own; it lives in the
  `testdata/parity` module. The harness already ties the two sides together: it reads
  `catalogVersion` off the oracle (`testdata/parity/shipped/shipped.cue:27`,
  `catalog.metadata.version`) and requires the kernel's `ResolvedVersions` row for
  `opmodel.dev/catalogs/opm@v4` to carry it on both sides (lines 78-86). The only literal left is
  in `shippedCases` (lines 150-196): seven rows of `shippedCatalogPrefix + "<name>@4.4.2"`. The
  probe group (`parity_probe_test.go:102-107`) is hermetic (`version = "0.1.0"` on a
  `registrytest.UniquePath`) and out of scope.
- **How CUE v0.17.1 picks a version.** A build uses the version the main module's
  `cue.mod/module.cue` lists. It does not apply MVS over a dependency's requirements at load
  time. A probe module listing `catalogs/opm v4.4.4` (which requires core beta.1) and core
  `v2.0.0-alpha.12` evaluated `(c.#ContractInventory & {}).collisions != _|_` as `false`, so it
  built against alpha.12; the beta.1 control evaluated `true`. The workspace comment at
  `.tasks/deps/fixtures.sh:19-22` records the same behaviour. MVS runs only inside
  `cue mod get` and `cue mod tidy`.
- **Catalog pins.** Four modules in `CUE_MODULE_GLOBS` (`Taskfile.yml:17-21`) pin
  `opmodel.dev/catalogs/opm@v4` at `v4.4.2`: `modules/opm_platform`, `testdata/modules/web_app`,
  `testdata/parity` and `testdata/parity/opm_platform`. All four pin core `v2.0.0-beta.1`. Three
  of them also pin `cue.dev/x/k8s.io@v0` `v0.12.0`; `testdata/modules/web_app` pins only the
  catalog and core. The flow tests match transformer ids by substring without a version
  (`flow_integration_test.go:121-160`).
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
- **Doc comments that tie a release to the default.** `opm/schema/loader.go:21-23` ("2.0.0-beta.1
  is that release; it is core's first beta ..."), `loader.go:48` (the `DefaultSchemaVersion`
  example `("v2.0.0-beta.1")`) and `opm/schema/cache.go:63` (`"v2.0.0-beta.1" when the default
  identifier resolved to that instance`). Each goes false the moment a bot text-bumps the
  constant. Other release literals in doc comments (`loader.go:109`, `registrytest.go:127,135`,
  `requires.go:18`, `modfile.go:29`) are format examples and do not claim to be the default.
- **Update task.** `cue:deps:update` (`Taskfile.yml:268-339`) loops over the dependencies one at
  a time with `cue mod get "$dep" >/dev/null 2>&1 || true`, then runs
  `cue mod tidy >/dev/null 2>&1 || true` (lines 315-321). The root workspace task explains in its
  own comment (`../Taskfile.yml:89-96`) why per-dependency gets silently no-op when siblings are
  stale. The root task also swallows errors, but fixing that is the workspace's job (workspace
  `deps:update` rewire, W1), not this change's.

## Goals / Non-Goals

**Goals:**

- A catalog bump touches only `cue.mod/module.cue` files.
- A core bump touches only the `DefaultSchemaModule` constant in `opm/schema/loader.go` and
  `cue.mod/module.cue` files (the `CUE_MODULE_GLOBS` set, plus the text-pinned `testdata/cue.mod`
  and `testdata/render/**`).
- Every OPM-owned version literal that stays in test code is there on purpose and is listed in
  `.cascade-frozen` with a reason.

**Non-Goals:**

- No change to what any test asserts about kernel behavior. Only the source of a version
  string changes.
- No guard against new literals creeping back (follow-up in `add-deps-cascade-task`).
- No change to the root workspace `deps:update` (W1) or to `cue:catalog:drift`.
- No change to the text re-pin of `testdata/cue.mod` and `testdata/render/**`. Those modules are
  served in-process and are not in `CUE_MODULE_GLOBS`, so `cue mod get` cannot resolve them. They
  are library test pins (workspace RELEASING.md, section "Pin classes", library test row); the
  cascade rewrites their core line as text, and `TestDefaultCoreVersion_IsTheDefaultSchemaRelease`
  already checks the result.

## Decisions

### DF1. Parity reads the pin from `testdata/parity/cue.mod/module.cue`

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

`TestParity_ShippedCatalog` reads the version once. Before it compares any case, it checks
`oracleCatalog == pinned` with a neutral message: "the oracle's catalog reports X, the parity
module pins Y". Because CUE builds against the version the main module lists, this cannot fail
by a resolution lift. It fails only when the harness reads the wrong file or the catalog's
`metadata.version` disagrees with its tag, and both are worth a named failure instead of seven
"no pair" rows. It then resolves each row's id before `assertRowsCoverPairs` and the per-case
loop. `shippedCases` stays a package-level table. Rows carry the bare name, and a small loop
builds a resolved copy, so the table still reads as data.

**Alternatives.** (a) Use the oracle's `catalogVersion` directly. This needs no file read and,
given how CUE picks a version, would name the same build. Rejected because the pin is the thing a
bump edits, and reading it keeps the table's source obvious to a reader of the test; the
equality check costs one line. (b) Read `testdata/parity/opm_platform/cue.mod`. The existing
`ResolvedVersions` assertion already ties the platform side to the oracle, so either module would
do. The oracle's module is the reference by spec ("Pure-CUE unification is the render oracle"),
so its pin is the one to read. (c) Use `modules/opm_platform`. The parity harness does not build
there (only the flow test does), so no.

`cuelang.org/go/mod/modfile` is already a dependency (`opm/catalog/requires.go:7`). No new import
enters `go.mod`.

### DF2. `registrytest.DefaultCoreVersion` derives from the schema default

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

### DF3. Derive what a default move breaks or what means "current core"; freeze the rest

The classification comes from the default-move dry run (DF6). A literal is *derive* when the dry
run turns its test red, or when it is a module file a test authors beside the served fixtures
and so means "the current core" by the spec's own wording ("Served fixtures pin the default
release"). Everything else is *freeze*.

| File:line | Literal | Means | Action | Evidence |
| --- | --- | --- | --- | --- |
| `opm/internal/registrytest/registrytest.go:113` | `DefaultCoreVersion` | the fixture harness's declared core | derive (DF2) | dry run: every floor, collision and contracts test that reads it red |
| `opm/kernel/render_test.go:157,206,491,526,992,1059` | `ResolvedVersions` core rows | current core resolved from the served fixtures | derive | dry run red at 156, 205, 490, 525, 991, 1058 |
| `opm/schema/loader_test.go:167,176` | pin assertion | tripwire | structural | dry run red at 167, 176 |
| `opm/kernel/render_test.go:906` | authored instance `cue.mod` core pin | current core, authored beside served fixtures | derive | dry run green (a downward move; the build uses the listed core); derived by meaning |
| `opm/internal/renderstage/stage_test.go:345,569` | authored module files, built | current core, authored beside served fixtures | derive | dry run green; derived by meaning |
| `opm/kernel/render_test.go:32` | comment "pins core 2.0.0-beta.1" | prose | reword to name `DefaultCoreVersion` | n/a |
| `opm/internal/renderstage/stage_test.go:475,533` | local-file core pin and `"v4.2.0"` catalog on the synthetic `platformModFile` (`modfile_test.go:24`, via `diskPlatform`, `stage_test.go:103`) | synthetic, coupled to a frozen module file | freeze | dry run green; deriving 475 would decouple it from `platformModFile` |
| `opm/internal/renderstage/modfile_test.go` (11 core, 15 catalog) | in-memory module files and skew rows (`alpha.6`, `4.1.0` to `4.3.0`) | synthetic parse and compare | freeze | dry run green |
| `opm/schema/loader_test.go:194,195` | `PinnedVersion` parse-table rows | synthetic parse inputs | freeze | dry run green |
| `opm/catalog/requires_test.go:18,41` | `test.example` module file read from an overlay | synthetic, never resolved | freeze | dry run green |
| `opm/helper/platformmodule/generate_test.go` (`coreVersion = alpha.7`, `4.0.1` rows, `TestGenerate_ExplicitCorePin` `beta.1`, `catalogs/k8s` `1.0.0-alpha.2` at 36, 41, 61, 81, 209, 212) | golden text generation, explicit pin | synthetic | freeze | dry run green |
| `opm/helper/platformmodule/closure_test.go` | `alpha.6/7/8`, `4.0.1`, `4.9.9`, `catalogs/k8s` `1.0.0-alpha.2` graph | synthetic dependency graph | freeze | dry run green |
| `opm/helper/objectset/objectset_test.go:29` | `transformer-registration-transformer@4.4.0` | synthetic id on an in-memory value | freeze | dry run green |
| `opm/internal/registrytest/registrytest_test.go:24-27` | `Major` parse table | synthetic parse inputs | freeze | dry run green |
| `opm/kernel/render_collision_test.go:166,204` | `alpha.12` | core without the collision report | freeze | value is the point |
| `opm/kernel/render_core_floor_test.go:43-44,68,103` | `alpha.10`, `alpha.12` | below and at the render floor | freeze | value is the point |
| `opm/platform/contracts_test.go:372-442,568` | `alpha.9` to `alpha.12` | floor and pre-collision semantics | freeze | value is the point |
| `opm/errors/coretooold_test.go` | `alpha.10`, `alpha.12` | error text | freeze | value is the point |

The synthetic rows are listed in `.cascade-frozen` rather than left out of it, including
`closure_test.go`, whose `4.0.1`, `4.9.9` and `alpha.*` literals are nodes of an in-memory fake
module graph and never a pin anything resolves. They are listed because the reader that
`add-deps-cascade-task` adds is a literal scan: it fails on any OPM-owned version literal in a
`*_test.go` that `.cascade-frozen` does not name. Leaving the synthetic files out would need a
second, file-specific exclusion mechanism in that scan. One list with a reason per file is
simpler, and its `reason` says the literal is synthetic, not an old pin. Workspace RELEASING.md,
section "Pin classes" (library frozen row), already names the `closure_test.go` literals.

`TestGenerate_ExplicitCorePin` spells `v2.0.0-beta.1`, which equals the default today, so it does
not yet prove an explicit pin wins over the default. It is frozen as it is: the next core bump
makes it a non-default pin with no edit, which is the property it wants. Editing it here would
touch a file this change freezes.

`render_test.go:906` and `stage_test.go:345,569` are kept on the derive side even though the
dry run cannot break them. The dry run moves the default downward (to alpha.13, the only other
published release on the schema); on an upward move, a frozen `beta.1` there would leave an
authored module building against an older core than the served fixtures it sits beside, with no
test failing. That is the drift "Served fixtures pin the default release" forbids.

**Alternative.** Derive every `v2.0.0-beta.1` in test code, synthetic ones included, as the plan
listed. Rejected. It threads concatenation through about 15 raw-string module files whose
version does not matter, and it hides the one distinction worth keeping: which literals mean
"now".

### DF4. Comments stop naming the release as the default

The 12 fixture platform headers (`testdata/render/platform*/platform.cue:1`, 11 files, and
`modules/opm_platform/platform.cue:2`) say "core 2.0.0-beta.1". They become "the core its
cue.mod pins". The `AGENTS.md` layout line for `testdata/` ("all pinned to core 2.0.0-beta.1")
becomes "all pinned to the default core release (`schema.DefaultSchemaModule`)". The CUE edit is
comments only, and `task cue:fmt` and `task cue:vet` must leave no diff.
`testdata/modules/web_app` is checksum-tracked in `cue-versions.yml`, but its `platform.cue` is
not touched.

Go doc comments that present a release as the default name the constant instead:

- `opm/schema/loader.go:21-23`: "2.0.0-beta.1 is that release; it is core's first beta and
  carries the 2.0.0-alpha.13 schema unchanged" becomes a sentence that says the constant below
  names that release. The `2.0.0-alpha.13` sentence that follows stays, because it is a fixed
  fact about [CollisionsSince], not about the default.
- `opm/schema/loader.go:48`: the `DefaultSchemaVersion` example becomes a format description
  ("the canonical `v`-prefixed form a cue.mod dependency carries") without a release.
- `opm/schema/cache.go:63`: "e.g. "v2.0.0-beta.1" when the default identifier resolved to that
  instance" becomes "e.g. the version suffix of [DefaultSchemaModule] when the loader used the
  default".

`docs/getting-started.md:51-56` and `AGENTS.md:329` keep their explicit pin, because they
illustrate a caller choosing a release.

### DF5. `cue:deps:update`: one pass, core held, loud, culprit named

Per module:

```bash
cache_dir=${CUE_CACHE_DIR:-${XDG_CACHE_HOME:-$HOME/.cache}/cue}
default_core=${DEFAULT_CORE:-}
if [ -z "$default_core" ]; then
  # grep exits 1 on no match; without this branch set -e would exit silently
  if ! default_core=$(grep -oP 'DefaultSchemaModule = "opmodel\.dev/core@\K[^"]+' opm/schema/loader.go); then
    echo "cannot read DefaultSchemaModule from opm/schema/loader.go; set DEFAULT_CORE" >&2; exit 1
  fi
fi
# newer A B: true when A sorts after B as semver (prerelease '-' becomes '~' so
# GNU sort -V orders v2.0.0-beta.1 before v2.0.0)
newer() { [ "$1" != "$2" ] && [ "$(printf '%s\n%s\n' "${1/-/\~}" "${2/-/\~}" | sort -V | tail -1)" = "${1/-/\~}" ]; }

args=(); others=()
for dep in $deps; do
  case "$dep" in
    opmodel.dev/core@v2) args+=("opmodel.dev/core@${default_core}") ;;
    *)                   args+=("$dep"); others+=("$dep") ;;   # "<path>@vN" = newest on that major
  esac
done
if ! out=$(cd "$dir" && cue mod get "${args[@]}" 2>&1 && cue mod tidy 2>&1); then
  printf '%s: cue mod get/tidy failed:\n%s\n' "$dir" "$out" >&2
  # Name the culprit: resolve again without core in a throwaway copy, then read
  # each moved dependency's own module file from the cache.
  tmp=$(mktemp -d); cp -a "$dir/." "$tmp/"
  if (cd "$tmp" && cue mod get "${others[@]}" >/dev/null 2>&1); then
    for dep in "${others[@]}"; do
      new=$(cue export "$tmp/cue.mod/module.cue" --out json | jq -r --arg d "$dep" '.deps[$d].v // empty')
      [ -n "$new" ] || continue   # unmoved deps too: a DEFAULT_CORE below the current pin
      mf="$cache_dir/mod/extract/${dep%@*}@${new}/cue.mod/module.cue"
      [ -f "$mf" ] || { printf '  %s@%s: module file not in cache (%s)\n' "${dep%@*}" "$new" "$mf" >&2; continue; }
      req=$(cue export "$mf" --out json | jq -r '.deps["opmodel.dev/core@v2"].v // empty')
      if [ -n "$req" ] && newer "$req" "$default_core"; then
        printf '  %s@%s requires opmodel.dev/core %s; the default is %s: advance schema.DefaultSchemaModule first\n' \
          "${dep%@*}" "$new" "$req" "$default_core" >&2
      fi
    done
  else
    echo "  (the get without core also failed; no culprit named)" >&2
  fi
  rm -rf "$tmp"; exit 1
fi
```

Facts behind it, checked on CUE v0.17.1:

- A held get that conflicts with a dependency's core requirement fails inside `cue mod get`
  itself, with `other requirements prevent changing module opmodel.dev/core@v2 to version ...
  (actual selected version: ...)`. That error does not name the dependency, and module.cue is
  left unchanged. So there is no "resolved to another core" state after tidy to inspect; the
  failure path is where the culprit has to be named.
- `cue env` does not exist in v0.17.1. The cache root follows `cue help environment`:
  `$CUE_CACHE_DIR`, else `$XDG_CACHE_HOME/cue`, else `~/.cache/cue`. Extracted modules live under
  `mod/extract/<path>@<version>/`, keyed by the path without its major suffix
  (`mod/extract/opmodel.dev/catalogs/opm@v4.4.4/`).
- CUE v0.17.1 has no `cue mod graph` (`cue mod --help` lists edit, fix, get, init, mirror,
  publish, registry, rename, resolve and tidy), hence the cache read.
- The positive path works: a held get at the current default plus tidy exits 0 on
  `testdata/modules/web_app` and moves the catalog from 4.4.2 to 4.4.4 with core staying beta.1.

`DEFAULT_CORE` is a task var passed into the environment. Task 4.3(c) uses it, and
`add-deps-cascade-task` can use it to run the update against a default it is about to commit.
The old-to-new report stays. `set -euo pipefail` already applies. The task keeps processing
modules in glob order and stops at the first failure.

The stale comment on `.github/workflows/cue.yml:40-41` ("older than the latest published") and
the matching sentence in `cue-versions.yml:9-12` ("checks that fixture pins track the latest
published tag") are corrected to match the task (existence, not currency, `Taskfile.yml:345-361`).

**Alternatives.** (a) Leave core out of the get. Then tidy lifts core silently when a catalog
needs a newer one. (b) Let core float to the newest release. Then the parity and flow modules test
a core the kernel's default never renders against, which is the drift "Served fixtures pin the
default release" forbids for the render fixtures. (c) Inspect the module after tidy for a lifted
core. Dead code on v0.17.1, because the held get fails first. All rejected.

### DF6. Default-move dry run (done during planning; repeated in section 1)

The classification in DF3 comes from a dry run on the untouched tree (`origin/main` at this
change's base). An earlier plan to classify by mutating single literals to an unpublished
release was dropped: an unpublished version turns every resolved literal red whether or not it
tracks the default, and mutating one coupled synthetic literal alone breaks its partner.

Procedure: set `DefaultSchemaModule` and `registrytest.DefaultCoreVersion` (the two constants a
bump edits today) to `v2.0.0-alpha.13`; text-replace `v: "v2.0.0-beta.1"` with
`v: "v2.0.0-alpha.13"` in every `cue.mod/module.cue` under `testdata/` and `modules/` (35 files);
run `go test ./opm/... -count=1 -short` with the GHCR registry env; revert.

**Spike result (2026-10-01):**

| Package | Failing tests | Lines |
| --- | --- | --- |
| `opm/kernel` | `TestRender_HappyOnDiskInputs`, `TestRender_OverlayModeInstance`, `TestRender_Skew_NewerModuleWarnsByDefault`, `TestRender_Skew_OlderModuleIsData`, `TestRender_PlatformLocalReplacementRendersTheDirectory`, `TestRender_InstanceReplacementOnPlatformPathIsInert` | `render_test.go:156, 205, 490, 525, 991, 1058` (the `ResolvedVersions` core rows) |
| `opm/schema` | `TestDefaultSchemaModule_PinsVerifiedRelease`, `TestDefaultSchemaVersion_IsTheDefaultModulesVersion` | `loader_test.go:167, 176` |
| every other package | none | none |

Moving only `DefaultSchemaModule` (leaving `DefaultCoreVersion` old) additionally failed
`TestDefaultCoreVersion_IsTheDefaultSchemaRelease`, `TestGenerate_BuildsThroughTheKernel`
(`build_test.go:60`), the four older-core `TestRender_*` tests in `render_collision_test.go` and
`render_core_floor_test.go`, and `TestContracts_InventoryPredatingTheCollisionReportReadsEmpty`
(`contracts_test.go:567`). All of them read `DefaultCoreVersion` to find the fixture pin, so they
are fixed by DF2, not by editing those files.

`stage_test.go`, `modfile_test.go`, `requires_test.go`, `generate_test.go`, `closure_test.go`,
`objectset_test.go` and `registrytest_test.go`'s parse table all stayed green. `6081afd` edited
`modfile_test.go` and `requires_test.go` for consistency, not because they failed.

Section 3's final check (task 3.7) is the reverse: the same dry run on the finished tree, moving
only `DefaultSchemaModule`, must be green with no `*_test.go` edited.

## Research & Decisions

### Where the catalog version should come from

**Context**: Seven literals have broken parity on every catalog bump.
**Explored**: `parity_harness_test.go:45-196`, `testdata/parity/shipped/shipped.cue:27`, the
four catalog-pinning cue.mods, and the library bump commits `588a638`, `ae9e247`, `fad87fd` and
`7f0dd5b` (each edits the same 14 lines of the harness). A probe of CUE v0.17.1's version
selection at load time (Context, "How CUE v0.17.1 picks a version").
**Decision**: DF1. Read the pin from the oracle's module; check the oracle reports the same build.
**Rationale**: The pin is what a bump edits, and CUE builds against it. The equality check is a
cheap consistency check with a clear message, not a resolution guard.

### Which core literals are load-bearing

**Context**: The plan estimated about 28 hand-edited literals per core bump.
**Explored**: Every `2.0.0-(alpha|beta).N` in `opm/**/*_test.go` (78 lines in 12 files), the
`6081afd` file list, and the default-move dry run (DF6).
**Decision**: DF3. Derive the six `ResolvedVersions` rows and `DefaultCoreVersion`, make the two
`loader_test.go` assertions structural, derive the three authored-beside-fixtures module files
by meaning, and freeze the rest.
**Rationale**: Only a literal that a bump breaks, or one whose meaning is "now", belongs in the
cascade's path. Rewriting synthetic text adds churn without adding safety.

### Single-pass update and core handling

**Context**: `|| true` per dependency hides both staleness and real errors.
**Explored**: The root `Taskfile.yml:62-116` (one-pass `cue mod get $deps`),
`.tasks/deps/fixtures.sh:38`, and a scratch run of a held get on `testdata/modules/web_app`.
**Decision**: DF5.
**Rationale**: It matches the root task's proven resolution behavior and adds the loud failure
the root task still lacks. Workspace RELEASING.md, section "The cascade" (receiver rules,
"Consistent set"), says that where a repo pins a catalog and core together, core moves to the
version that catalog pins. The library applies a stricter variant: core in its test modules
equals `DefaultSchemaModule`, because the kernel renders only against that release.
`add-deps-cascade-task` advances `DefaultSchemaModule` first (labelled `need-human-review`) and
then runs `cue:deps:update`, so a catalog that needs a newer core arrives together with the
default move instead of lifting the test pins on its own.

## Risks / Trade-offs

- [Catalog `4.4.4` renders differently from `4.4.2`, so parity or flow fails in section 5] →
  catalog `4.4.3` moved the catalog onto core `v2.0.0-alpha.13` (catalog_opm `14869b5`), and
  `4.4.4` moved it onto core `v2.0.0-beta.1` (catalog_opm `b35d644`), which carries the alpha.13
  schema unchanged. A rendered-output change is unlikely but possible. If it happens, section 5
  stops, the divergence is written into Risks here, and the bump becomes its own change.
  Sections 1 to 4 stand on their own.
- [`cue mod get` in section 5 moves `cue.dev/x/k8s.io` too] → Allowed. It is a test pin and lands
  in the same `test(fixtures)` commit. The verification checks the suite, not a one-line diff. If
  it breaks, the task's own fix applies: hold that dependency with an explicit version in the
  get and record why.
- [`DefaultCoreVersion` becomes a `var`] → Any future constant-expression use fails at compile
  time, which is loud. No exported surface outside `internal/`.
- [The structural loader assertion lets a default bump through with no test edit] → Intended. A
  literal tripwire adds no deliberateness once a bot text-bumps it together with the constant.
  The `need-human-review` label on the bump PR (workspace RELEASING.md, section "The cascade",
  labels) carries it, and the "default" row of `TestOCILoader_PinnedVersion` already checks the
  shape.
- [The culprit search relies on the cache layout] → The layout is CUE's, not a public contract.
  When the extracted module file is missing, the task says so and prints the path it looked at,
  and the `cue mod get` error is always printed first, so the failure stays loud.
- [Full-suite flake in `TestGenerate_BuildsThroughTheKernel`] → The eviction race was fixed
  (test-fixture-registry spec, "The shared workspace cache is never deleted from"). If it
  reappears, rerun the package alone before treating it as caused by this change.

## Migration Plan

None. Test-only. Rollback is a revert of the PR.

## Open Questions

None. The one factual unknown (which literals a default move breaks) was settled by the DF6 dry
run during planning; section 1 repeats it on the implementation base.
