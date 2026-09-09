# Design: render-local-replacements

## Context

See `proposal.md` § Why. The render module today, after `overlay-served-in-memory` (which this change follows, both edit `Stage`):

```
  <tmp>/opm-render-XXXX/
    cue.mod/module.cue        promoted deps; the two inputs listed with placeholder versions, marked default
    cue.mod/local-module.cue  the same list, plus replaceWith on the two inputs -> their serve directories
    render.cue                the glue
```

`Stage` reads only each input's `cue.mod/module.cue` (`ReadModFile`); `Promote` merges the two lists and records exactly two replacements; `LocalModuleFile` writes every entry of `Promotion.Replacements`, including a replace-only entry for a path the base list does not carry (the "hand-built Promotion" branch). cue/load reads `local-module.cue` from the main module only (`cue/load/config.go:581`), resolves a relative `replaceWith` against the main module's root, and, for a directory target, opens `<dir>/cue.mod/module.cue` through its own filesystem to check the module path.

Spikes run 2026-09-09 on `main` (throwaway `go test -overlay`, nothing committed):

- cue's `modfile.Parse` and `ParseNonStrict` both accept a version-less dependency (`deps: "x@v0": {}`); `ParseModFile` copies it with `Version == ""`; `Promotion.ModuleFile` then fails inside `modfile.Format` ("conflicting values "" and null"). This is today's failure for an instance module that redirects a never-published module.
- `modfile.ParseLocal` accepts a replace-only entry absent from `module.cue`, and also one listed with a `v0.0.0` placeholder plus `replaceWith`.
- 0019 experiment 02, mode `replaced`, measured that a directory replacement of the catalog in the render main module serves the directory's bytes with platform authority intact.

Task 1.1's spike (`TestStageBuild_LocalReplacementsResolveInOneBuild`, kept as a regression test) passed on the first build: a hand-written render module replacing an instance-only, never-published path with a directory and the platform's catalog path with a patched copy resolved both inside the one build; the instance's import came from the directory and the deployment's label from the copy. Two facts measured while implementing, both of which a frontend needs:

- `cue.mod/local-module.cue` is the *whole* main-module dependency view, not a patch: `modfile.ParseLocal` takes `Deps` from the local file alone (versions and default markers inherited from `module.cue` where omitted). A local file listing only the replaced path makes every other import fail with "cannot find module providing package" when the input is built on its own (the platform copy in the tests only loaded because the replaced catalog's own `module.cue` pulled core in transitively). `cue mod tidy` writes the full list; a hand-written file must too. Promotion keys precedence on `module.cue` (D13) and carries local-listed entries only for paths the promoted list lacks (a module-path replacement's target), so a local file that only re-pins a version without replacing anything is inert at render time: out of scope here, worth a follow-up if a developer expects `vet` and `render` to agree on it.
- `cue/load` classifies a `replaceWith` value as a directory iff it starts with `.` or `/` (or is a host-absolute path); everything else is a module path. `ReadLocalModFile` mirrors that rule.

## Goals / Non-Goals

**Goals:**

- A developer's `cue.mod/local-module.cue` in a module, instance or platform directory reaches the render build under the precedence 0019 D13 fixes for dependencies.
- The kernel never silently drops the file again: it is either honoured (opt-in) or refused.
- Render output and diagnostics for inputs without the file are byte-identical to today.

**Non-Goals:**

- Deciding what a frontend says about an inert replacement; the cli reads the rows and words it.
- Module-path replacements (`replaceWith: "fork@v0"`) beyond passing them through; the target must already be listed with a version in the same local file, as cue requires, and promotion carries that entry with the rest.
- Any change to `VerifyCoverage`. `CompareSkew` gains one guard only (see Risks): a version-less entry has no version to compare, so the row records it and flags nothing.

## Research & Decisions

### Replacements promote like dependencies, platform whole, instance only on instance-only paths

**Context**: two inputs can each redirect paths; the render module has one main-module view.
**Explored**: (A) union with instance winning (a module author could swap the catalog bytes a platform executes: the privilege boundary experiment 02 exists to protect); (B) union with platform winning on shared *replacement* paths only (an instance could still replace a path the platform pins but does not replace, again swapping executed bytes); (C) platform replacements whole, instance replacements only for paths absent from the platform's *dependency list*.
**Decision**: C. The comparison key is the platform's dependency list, the same authority rule D13 applies to pins.
**Rationale**: "the platform decides which bytes execute for every path it names" stays one sentence, and a catalog developer has an unambiguous place to redirect: the platform's file. An inert instance replacement is not an error; the cli reports it.

### Relative targets resolve against the input's root, at promotion

**Context**: cue/load resolves a relative `replaceWith` against `ModuleRoot`, which is the staging directory.
**Explored**: rewrite at promotion; or serve the input from a directory laid out so relative paths still work (impossible for two inputs with independent roots).
**Decision**: `Promote` receives each local view with directory targets already absolute (`filepath.Join(src.Root, target)` for `./`, `../`; absolute kept). For an overlay-mode input with a synthetic root (a registry fetch) the file cannot exist: cue rejects a downloaded module that carries it. The cli's local-directory overlay has a real root.
**Rationale**: the written file is what a reader sees in the staging directory; absolute targets make it self-explanatory and `checkReplaceDirModulePath` opens the right directory.

### A replaced path missing from the promoted list is listed with the input placeholder

**Context**: `VerifyCoverage` requires every OPM-namespace path from either input's list to appear in the render `module.cue`; a replace-only dependency may be version-less in the input.
**Explored**: replace-only entry in `local-module.cue` only (accepted by cue, but an OPM path would then fail coverage, and a version-less entry would reach `modfile.Format`); placeholder version in `module.cue` plus `replaceWith` in the local view (the mechanism the two inputs already use, `ReplacedVersion`).
**Decision**: placeholder. `Promote` substitutes `ReplacedVersion(path)` for an empty version when a promoted replacement covers the path. `ParseModFile` keeps accepting an empty version; `Promote` refuses one that no replacement covers, naming the path and input.
**Rationale**: one representation for every replaced path; coverage holds by construction; the refusal message names the actual cause instead of a modfile schema error.

### Opt-in on `RenderInput`, refusal when off

**Context**: the audit skill records the invariant that staging replacements never point at a path an artifact names. Honouring the file relaxes that for directories the developer owns.
**Explored**: honour unconditionally (relies on CUE rejecting fetched modules that carry the file and on the operator's generated platform dir never carrying one; true today, but nothing in the kernel states it); honour only for on-disk inputs (does not exclude the operator's on-disk platform dir); an explicit bool on `RenderInput`.
**Decision**: `RenderInput.LocalReplacements bool`, default false. Off and a file with a replacement present → `Render` fails before staging: `input %q carries cue.mod/local-module.cue with replacements; the caller did not enable local replacements`. On → promoted and reported.
**Rationale**: the boundary becomes a sentence in the kernel's API rather than a property of upstream tooling. Refusal over silent ignore because silent ignore is the failure this change exists to remove; a tampered operator platform dir now surfaces loudly. Principle VII: this is a security switch, not an extension point.

### Rows, not strings

**Decision**: `renderstage.ReplacementRow{Path, Target, By string}` on `Staged`; `kernel.Replacement` with the same fields on `RenderDiagnostics.Replacements`, path-sorted, `By` one of `"platform"`, `"instance"`. `ResolvedVersions` is unchanged: a replaced OPM path keeps its pinned versions there, and the replacement row says where bytes came from.
**Rationale**: the no-presentation-strings requirement; the cli needs the honoured set to compute the inert set.

### Reading the local view

**Decision**: `ReadLocalModFile(src, base) (localView, error)` in `modfile.go`, absent → nil view. `sourcetree.ReadFile` gains an `fs.ErrNotExist`-wrapped error for a missing overlay entry so absence is one check for both modes. Parsing uses `modfile.ParseLocal(data, path, baseFile)`, which requires the base as a `*modfile.File`; `ReadModFile` keeps the parsed file on `ModFile` (unexported field) to avoid a second parse.

```go
// Promote, sketch of the new step after the dependency union:
for path, r := range platformLocal.Replacements { repl[path] = r; rows = append(rows, row(path, r, "platform")) }
for path, r := range instanceLocal.Replacements {
    if _, named := platform.Deps[path]; named { continue }   // inert: platform authority
    repl[path] = r; rows = append(rows, row(path, r, "instance"))
}
for path := range repl {
    if d, ok := deps[path]; !ok || d.Version == "" { deps[path] = Dep{Version: ReplacedVersion(path), Default: d.Default} }
}
```

## Risks / Trade-offs

- [cue/load refuses a directory replacement whose `cue.mod/module.cue` declares a different path than the replaced one] → that is `checkReplaceDirModulePath` working as designed; the error names both paths. Task 1.1 exercises the happy path; a mismatched directory is the developer's error and is reported verbatim.
- [A module-path replacement target is not listed with a version] → cue refuses with "replacement target … must be listed as a dependency with a version"; the local view carries the target entry, promotion carries it through, so a tidy local file passes. Not special-cased.
- [The cli renders with a replacement, applies, and the operator's re-render diverges] → by design (0006 D7.4 digest abort); provenance already flips to `source: local` on file presence.
- [Two changes editing `Stage`] → sequenced after `overlay-served-in-memory`; rebase cost is one function.
- [A platform developing a never-published catalog lists it version-less and replaces it, while the instance pins a published build] → `CompareSkew` compared `"v4.2.0"` against `""` and failed with a semver error naming neither input. It now skips the comparison when either side is empty (the row still carries both strings, `Newer` false), the one guard added to a function this change otherwise leaves alone. Pinned by `TestCompareSkew_VersionlessReplacedPathIsNotCompared`.
- [A local file replaces one of the two render inputs' own module paths] → refused by `Promote` ("replaces …, a render input"): the inputs are served from their staged directories by construction, and a row pointing elsewhere would misreport what executed.
- [Refusal-by-default breaks a consumer that passes such inputs today] → only the cli does, and it sets the field in its own change; called out in the proposal's Impact.

## Migration Plan

One library PR after `overlay-served-in-memory` merges; `task check` green including the race pass; parity harness green. Rollback is a revert. The cli bumps the library and sets `LocalReplacements: true` at its one `kernel.Render` call in the same PR that adopts the release.

## Open Questions

None.
