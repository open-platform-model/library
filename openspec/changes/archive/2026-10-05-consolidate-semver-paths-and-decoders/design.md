## Context

See proposal.md, Why. Four small consolidations share one change because ADR-013, decisions b3,
b4 and b5 bundle them, and x1 was folded in. Design-local decisions are numbered CS1 to CS8 so
they collide with no other numbering. Line numbers are at library `origin/main` `58f8151`
(v1.0.0-beta.4 plus #180 and #181; neither touched these files since the wave-2 research at
`93a892f`).

## Goals / Non-Goals

**Goals:** one home for version helpers and the core-path and language-floor constants, and
one SemVer implementation; each artifact constructor names a decode failure once; a dotted key
is one `Path()` segment; one local-path check; one module-file reader; a truthful Cache message.

**Non-Goals:** merging the four metadata decoders (the schema-dispatch rule stands); touching
the "since" floors (`schema/paths.go` `ProvidedBySince`, `CollisionsSince`, the
`platform/contracts.go:167` literal); the `OverlayFromDir` move to `os.DirFS` (b2); changing
`schema.DefaultSchemaModule`; making the Cache retry.

## Research & Decisions

Evidence for every site below is the wave-2 research entry "b3+b4+b5 consolidate" and the
follow-up "x1-lib-cache-zero-value-message" (`claude-stuff/kernel-plan-beta1/wave2-plan-result.json`),
re-checked against `58f8151`.

### CS1: modversion is the only importer of x/mod/semver

`opm/internal/modversion` grows to:

```go
const (
	// CoreModule is the module path of the OPM core schema, without a major qualifier.
	CoreModule = "opmodel.dev/core"
	// CorePath is the major-qualified core path the v2 kernel builds against.
	CorePath = CoreModule + "@v2"
	// LanguageFloor is the lowest CUE language version a generated or render module
	// declares: v0.17.0 introduced cue.mod/local-module.cue.
	LanguageFloor = "v0.17.0"
)

// Major returns the "@vN" qualifier of a version in either spelling: "0.1.0" -> "v0",
// "v2.0.0-beta.1" -> "v2", "v2" -> "v2". Like Canonical and Bare it validates nothing.
func Major(v string) string {
	major, _, _ := strings.Cut(Bare(v), ".")
	return "v" + major
}

// Valid reports whether v is a SemVer version in either spelling.
func Valid(v string) bool { return semver.IsValid(Canonical(v)) }

// Compare orders a and b by SemVer precedence (-1, 0, +1), accepting either spelling.
// An invalid input is refused as `invalid version "<as written>"`.
func Compare(a, b string) (int, error)
```

`renderstage` and every other package compare and validate through these functions and never
import `x/mod/semver` (or any other SemVer library) themselves, so "one SemVer implementation"
holds in the import graph. The depguard rule `one-semver-implementation` in `.golangci.yml`
enforces it: under `opm/` only `opm/internal/modversion` may import `x/mod/semver`, and
`github.com/Masterminds/semver` is denied everywhere.

`Major` keeps the string logic both copies use today rather than `semver.Major`, which returns
`""` for an invalid input: today's callers get `"v"` plus whatever precedes the first dot, and
the synthesized import or fixture then fails in CUE with its own error. No caller validates
through `Major`.

### CS2: one literal for the core path, derived forms for the rest

The core module path is written once, as `CoreModule`. `CorePath` is derived from it. `synth`
keeps deriving its import's major from the resolved core version
(`modversion.CoreModule + "@" + modversion.Major(coreVersion)`), as it does today, rather than
hard-wiring `CorePath`, so the import matches the release the kernel's loader resolved.
`registrytest` builds `coreDep` and the stand-in core's fixture directory from `CoreModule`.
`platformmodule` declares

```go
CorePath        = modversion.CorePath
LanguageVersion = modversion.LanguageFloor
```

inside its existing `const` block, so both stay exported untyped string constants with the
same values (a `var` would break callers that use them in constant expressions). Each doc
comment states the value (`"opmodel.dev/core@v2"`, `"v0.17.0"`), because the public Go API
reference excludes `opm/internal` and would otherwise show only a reference to an internal
constant.
`renderstage.MinLanguageVersion` is internal and has no user outside `renderstage`, so it is
removed and `promote.go` reads `modversion.LanguageFloor`.

**Left alone:** `schema.DefaultSchemaModule` (`schema/loader.go:43`) names an exact release, not
the path, and the release cascade reads it with
`grep -oP 'DefaultSchemaModule = "opmodel\.dev/core@\K[^"]+'` (`.tasks/cascade/lib.sh:49`), so
it must stay one literal. Doc comments that quote `opmodel.dev/core` or `v0.17.0` as prose
(`renderstage/modfile.go:21,43`, `errors/coretooold.go`) stay prose.

### CS3: the three Masterminds call sites

| Site | Today | After |
| --- | --- | --- |
| `skew.go` `isNewer` (:65-75) | `semver.NewVersion` both, `GreaterThan` | `c, err := modversion.Compare(a, b)`; `c > 0` |
| `promote.go` `ReplacedVersion` (:296) | `semver.NewVersion(major + ".0.0")` | `modversion.Valid(major + ".0.0")`, else the existing `module path %q carries no major qualifier` |
| `promote.go` `maxLanguage` (:321-339) | Masterminds parse, `"v" + best.String()` | start from `LanguageFloor`; `modversion.Compare(v, best)` (best is always valid, so an error names v) and refuse an invalid input as `invalid language version %q` alone, without wrapping Compare's `invalid version` text, so the message is not doubled; return the winning string as written |

Inputs are `modfile.Parse` output (`v`-prefixed and valid). x/mod accepts the same shorthand
Masterminds did (`vMAJOR`, `vMAJOR.MINOR`; the `v` prefix is supplied by `Canonical`), so which
inputs are accepted does not change at all; only `maxLanguage`'s printed form differs (below). Both libraries implement SemVer 2 precedence and ignore build metadata in
comparison; new skew tests pin `v2.0.0-beta.2 < v2.0.0-beta.10 < v2.0.0` so a regression in
prerelease ordering is caught. `maxLanguage` returning the string as written differs from today
only for a shorthand like `v0.18` (Masterminds printed `v0.18.0`); `modfile.Parse` output and
every fixture carry the full form, and CUE accepts both.

The error text keeps its leading words (`invalid version %q`, `invalid language version %q`);
x/mod gives no parse cause, so nothing follows the quoted version. No test in the library,
cli or opm-operator matches these strings beyond the prefix.

### CS4: constructors return the decoder's error as is

`NewModuleFromValue` (`module.go:80-83`), `NewPlatformFromValue` (`platform.go:64-67`) and
`NewCatalogFromValue` (`catalog.go:81-84`) return `decodeXMetadata`'s error unwrapped. The
decoders keep their own `decoding X metadata: %w` and `X metadata field is required` texts, so
every failure names its artifact exactly once. The platform's `decoding platform type: %w`
loses its outer prefix too, which is correct: it is a type failure, not a metadata one. The
instance path (`kernel/process.go`) is not doubled and is not touched. The decoders stay four
free functions in their own packages (schema-dispatch, "Metadata decoders are free functions").

### CS5: selectors through the disallowed-field walk

```go
func walkDisallowed(schema, val cue.Value, prefix []cue.Selector, acc cueerrors.Error) cueerrors.Error
	// fieldPath := append(slices.Clone(prefix), iter.Selector())

type fieldNotAllowedError struct {
	pos  token.Pos
	path []cue.Selector
}

func (e *fieldNotAllowedError) Path() []string {
	out := []string{"values"}
	for _, sel := range trimSchemaPrefix(e.path) {
		out = append(out, sel.String())
	}
	return out
}
```

`trimSchemaPrefix` drops a leading `#module` `#config` pair, or a leading `#config`, comparing
whole selectors. It replaces `normalizeFieldPath`'s string `TrimPrefix` and keeps today's
behaviour: a regular label such as `#configMap` comes out of `Selector.String()` quoted, and
`walkDisallowed` iterates `Fields(cue.Optional(true))`, which yields no definitions, so the
string trim could not misfire either; the selector-wise trim is for the selector-typed path,
not a defect fix. `Selector.String()` is the form
`walkDisallowed` records today, so a definition stays `#config`, a plain label stays bare and a
label that needs quoting stays quoted (`"app.kubernetes.io/name"`). `Unquoted()` was rejected:
it would change the cli's printed key and panics on a non-string selector. The cli joins the
segments with `.` (`pkg/errors/grouped_errors.go:44`), so its text is byte-identical; only the
segment count of a dotted key changes, from three to one.

### CS6: IsLocal, with the explicit `.` refusal where it existed

`Source.WriteTo` and `serveDir` compute `rel, err := filepath.Rel(root, key)` and refuse
`err != nil || !filepath.IsLocal(rel)`. `IsLocal(".")` is true, and a `.` key was accepted by
both before, so their behaviour is unchanged. `Files.WriteTo` refuses
`!filepath.IsLocal(rel) || rel == "."`: it refused `.` and absolute names before, and
`IsLocal` covers the absolute and `..` cases. The refusal messages are unchanged.

### CS7: catalog.Requires on ReadModFile

```go
if c.Source == nil || c.Source.Root == "" {
	return nil, fmt.Errorf("catalog carries no source tree, so it has no committed %s to read", renderstage.ModFileName)
}
mf, err := renderstage.ReadModFile(c.Source)
if err != nil {
	return nil, fmt.Errorf("catalog %s: %w", renderstage.ModFileName, err)
}
// reqs[path] = dep.Version for every entry of mf.Deps (version-less entries map to "")
```

`ReadModFile`'s own messages name the file path (`parsing <path>: ...`), so one wrap is enough;
the absent-file test (asserts `cue.mod`) and the unparseable-file test (asserts an error) hold.
`opm/catalog` may import `opm/internal/renderstage`: depguard only bars the kernel tier from
`opm/helper` and `opm/k8s`, and `renderstage` imports `opm/module` and `opm/internal/sourcetree`,
never `opm/catalog`, so there is no cycle. `ParseModFile` additionally refuses a dependency with
`replaceWith`; that is the only newly refused shape (`modfile.Parse` already refuses an empty
module path). The catalog-acquisition delta states both refusals.

### CS8: one wording for an unusable loaded value

`Cache.Get` keeps a single `Err()` check and words it `schema Cache: loaded schema is unusable:
%w`. A separate `!Exists()` branch was rejected: `Value.Exists` is also false for some errored
values, so a second branch would trade one mislabel for another. "Unusable" is true of both the
zero value (cause `undefined value`) and a value with a build error, and the cause stays
wrapped. A new unit test drives a Loader that returns `cue.Value{}, nil` and asserts the zero
value, the message, one Loader run and an empty `ResolvedVersion()`. No library, cli or
opm-operator test matches "carries a build error" (grep at origin/main, 2026-10-04).

## Risks / Trade-offs

- `go mod tidy` decides the exact `golang.org/x/mod` version (v0.39.0, already selected by MVS
  through `golang.org/x/tools` v0.49.0 and in go.sum). If tidy moves it or the `go` directive, the section commit says so
  and the deps cascade carries it.
- `Path()` for a dotted key changes shape. Only the cli reads it, and it joins; an unknown
  consumer reading segments was getting a wrong answer before.
- `catalog.Requires` refuses one shape it tolerated (a `replaceWith` dependency). It appears in
  no published catalog or fixture; the full test suite is the check.

## Migration Plan

None. No exported signature changes. The deps cascade carries the go.mod change to the cli and
the operator.
