## Why

The owner's review recorded in ADR-013 found the same small things written
several times across the library, and two defects hiding in the copies. Items b3, b4 and b5 were
decided as one consolidation change; the supervisor folded in one follow-up from the wave-1
review (x1).

- **b3, version helpers.** Wave 1 (library#170, i1) created `opm/internal/modversion` with
  `Canonical` and `Bare`. The other version helpers still live where they were first needed.
  Version comparison uses `github.com/Masterminds/semver/v3` in `renderstage/skew.go` (`isNewer`)
  and `renderstage/promote.go` (`ReplacedVersion`, `maxLanguage`), while CUE's own module code
  uses `golang.org/x/mod/semver`. Two copies of "major of a version" exist (`synth/render.go`
  `major`, `registrytest.Major`) plus three inline `strings.Cut` copies in `registrytest`. The
  core module path is written in `platformmodule` (`CorePath`), `synth` (`corePath`) and
  `registrytest` (twice). The `v0.17.0` language floor is written in `renderstage`
  (`MinLanguageVersion`), `platformmodule` (`LanguageVersion`) and three `registrytest` literals.
  Owner decision: reuse modversion, drop Masterminds for `x/mod/semver`, one major function, one
  core-path constant, one language-floor constant; leave the "since" floors alone.
- **b4, decoder defects.** `NewModuleFromValue`, `NewPlatformFromValue` and
  `NewCatalogFromValue` wrap their decoder's error with the same prefix the decoder already
  added, so a failure reads `decoding module metadata: decoding module metadata: ...`. And the
  kernel's disallowed-field error flattens its path to a dotted string and splits it again, so a
  key that contains dots (`"app.kubernetes.io/name"`) comes back from `Path()` as three
  segments. The cli re-joins the segments with `.` today, so its output hides the defect, but
  any consumer that reads the segments gets the wrong path. Owner decision: fix the defects
  only. Drop the doubled wrap, carry `cue.Selector` so a dotted key stays one `Path()` segment,
  add a dotted-key test, and keep the per-package decoders (the schema-dispatch rule "Metadata
  decoders are free functions" stands).
- **b5, path checks and the module-file reader.** Three places hand-roll "is this relative path
  inside the root" (`module/source.go` `Source.WriteTo`, `renderstage/stage.go` `serveDir`,
  `helper/platformmodule/write.go` `Files.WriteTo`); the standard library's `filepath.IsLocal`
  says it in one call. `catalog.Requires` parses `cue.mod/module.cue` itself and redeclares the
  file name, beside `renderstage.ReadModFile`, which reads the same file in both source modes.
  Owner decision: `IsLocal` in all three, keeping the explicit `.` refusal in `platformmodule`,
  and `catalog.Requires` on `ReadModFile`. The `OverlayFromDir` swap to `os.DirFS` is NOT here;
  it rides b2.
- **x1, schema Cache message.** Wave 1 made `schema.Cache.Get` treat an errored loaded value as
  a load failure. A Loader that returns the zero `cue.Value` with a nil error also trips that
  check (`cue.Value{}.Err()` is "undefined value"), and the message calls it "carries a build
  error", which it does not.

## What Changes

- **One home for version helpers (b3).** `opm/internal/modversion` gains `Major` (the
  major-qualifier of a version, bare or `v`-prefixed), `Valid` and `Compare` (SemVer validity and
  precedence through `golang.org/x/mod/semver`, accepting either spelling), and the constants
  `CoreModule` (`"opmodel.dev/core"`), `CorePath` (`CoreModule + "@v2"`) and `LanguageFloor`
  (`"v0.17.0"`). `renderstage` compares through it (skew, `ReplacedVersion`, `maxLanguage`) and
  drops `MinLanguageVersion`; `synth` builds its core import from `CoreModule` and `Major`;
  `registrytest` uses `CoreModule`, `Major` and `LanguageFloor` and drops its own `Major`;
  `platformmodule.CorePath` and `platformmodule.LanguageVersion` stay exported constants, now
  defined as `modversion.CorePath` and `modversion.LanguageFloor`. `github.com/Masterminds/semver/v3`
  leaves `go.mod`; `golang.org/x/mod` becomes a direct requirement. The "since" floors
  (`schema.ProvidedBySince`, `schema.CollisionsSince`, the literal in `platform/contracts.go`)
  and `schema.DefaultSchemaModule` (a literal the release cascade reads with `grep`) are left as
  they are.
- **A metadata decode failure is named once (b4).** The three constructors return the
  decoder's error as is. The message keeps one `decoding <kind> metadata:` prefix; a missing
  `metadata` field still reads `<kind> metadata field is required`.
- **A dotted key is one path segment (b4).** The disallowed-field walk records the
  `cue.Selector` of each step, and `Path()` returns one segment per selector (its
  `Selector.String()` form, so a quoted label stays quoted and the cli's joined text is
  unchanged). The leading `#module`/`#config` steps are trimmed per segment instead of by string
  prefix. A new test asserts a dotted key comes back as one segment.
- **One local-path check and one module-file reader (b5).** `filepath.IsLocal` replaces the
  hand-rolled `..` checks in `Source.WriteTo` and `serveDir`, and in `Files.WriteTo` together
  with its explicit `.` refusal. `catalog.Requires` reads through `renderstage.ReadModFile` and
  its own `modFileName` constant goes.
- **An unusable loaded value is called unusable (x1).** `Cache.Get` words its `Err()` failure
  `schema Cache: loaded schema is unusable: <cause>`, which is true of both a value with a build
  error and the zero value a Loader may return with a nil error (cause `undefined value`). The
  cause is still wrapped, and the doc comment names both cases.

Not **BREAKING**. No exported signature changes, and the two exported `platformmodule`
constants keep their names, kinds and values. Observable differences: error text (the doubled
prefix becomes single; the Cache's errored-value message says "is unusable" instead of
"carries a build error"; `ReplacedVersion`'s refusal of a path whose major qualifier is not a
valid version now reads `module path %q carries no major qualifier` in place of
`module path %q: <parse cause>`, internal, matched by no consumer), `Path()` segments for
a key containing dots (one segment instead of several; the joined string is the same), and
`catalog.Requires` now refuses a module file whose dependency carries `replaceWith`, which the
render stage refuses too (`modfile.Parse` already refused an empty module path, so that refusal
is not new). A published catalog never carries `replaceWith`. `go.mod` drops one requirement and promotes another from `go.sum` to a direct
requirement, which the deps cascade picks up.

SemVer class: PATCH. Release class of the PR: `fix`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `config-validation`: the disallowed-field error's `Path()` returns one segment per field
  label, so a key containing dots is one segment.
- `schema-dispatch`: a constructor's metadata decode failure names its artifact once; the
  schema Cache covers a Loader that returns the zero value with a nil error, and words both
  that and an errored build as an unusable schema.
- `catalog-acquisition`: `Requires` reads the module file through the render stage's parser and
  refuses what that parser refuses.
- `single-build-render`: the skew comparison is stated to follow SemVer precedence for
  prerelease builds (behaviour unchanged across the dependency swap, now pinned by a scenario).

## Impact

- Packages: `opm/internal/modversion`, `opm/internal/renderstage` (`skew.go`, `promote.go`,
  `stage.go`), `opm/internal/synth` (`render.go`), `opm/internal/registrytest`,
  `opm/helper/platformmodule` (`generate.go`, `write.go`), `opm/module` (`module.go`,
  `source.go`), `opm/platform` (`platform.go`), `opm/catalog` (`catalog.go`, `requires.go`),
  `opm/kernel` (`validate.go`; tests that used `registrytest.Major`), `opm/schema` (`cache.go`),
  `go.mod`/`go.sum`, and the `AGENTS.md` repository layout (a line for `internal/modversion`).
- Public surface under `opm/`: no signature change.
- Downstream: the cli joins `Path()` with `.` (`pkg/errors/grouped_errors.go`), so its printed
  path is unchanged. No cli or opm-operator test matches the doubled prefix or "carries a build
  error" (grep at origin/main, 2026-10-04). Both frontends pick up the go.mod change through the
  deps cascade.
- Ordering (wave-2 serialization): this change merges before lib-b1g2 and lib-d1d3 (they gate on
  it), lib-e2e5 (go.mod), lib-h4 (`platform.go`), lib-i3d2 (`module.go`) and lib-g5a (`serveDir`
  in `stage.go`).
- No `enhancement.yaml`: the decisions are ADR-013, decisions b3, b4 and b5, not an
  enhancement.
