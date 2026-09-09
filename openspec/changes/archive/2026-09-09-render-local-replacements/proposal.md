## Why

CUE v0.17's `cue.mod/local-module.cue` is the sanctioned way to develop against unpublished or locally edited modules (enhancement 0006 D37): a developer redirects a dependency to a directory, and every verb that loads their module as the main module honours it. `Kernel.Render` is the one verb that does not. It generates its own main module for the build (0019 D9) and promotes only each input's `cue.mod/module.cue` into it (`opm/internal/renderstage/stage.go`, `ReadModFile`), so a developer's replacements are dropped at render time and the published pin resolves instead. Measured on `main` 2026-09-09 with a throwaway probe:

| Developer writes | vet / build / validate | render today |
| --- | --- | --- |
| module replaces a shared library module | honoured | dropped, published pin resolves |
| platform replaces its catalog with a checkout | honoured | dropped, platform's own pin resolves |
| instance replaces a never-published module (version-less entry) | honoured | fails in `Promotion.ModuleFile` (modfile refuses an empty version), an error that never names local-module.cue |

The kernel already writes a `local-module.cue` for the render module with two `replaceWith` entries (the instance and the platform directories). Promoting the inputs' own replacements into that same file, under the precedence 0019 D13 already fixes for dependencies, closes the gap with no new mechanism.

**Enhancement declaration.** None: 0006 (D37) and 0019 (D9, D13) are archived as delivered, and this change closes a gap between decisions already delivered whole rather than delivering one, so it carries no `enhancement.yaml` (the same footing as `overlay-served-in-memory` and `kernel-owns-no-build-context`).

**Scope statement (Principle VIII).** One internal package (`opm/internal/renderstage`: `Stage`, `Promote`, `modfile.go`, `skew.go` untouched) plus two additive fields on `opm/kernel` types. A spike test opens the change because the claim that a promoted replacement of a *dependency* of a replaced input resolves inside one build rests on 0019 experiment 02's "replaced" mode, which the library has never exercised in its own tests.

## What Changes

**`opm/internal/renderstage`:**

- `Stage` reads each input's `cue.mod/local-module.cue` when present (through `sourcetree`, so overlay-mode and on-disk inputs behave alike), parses it against that input's `module.cue`, and resolves a relative directory target against the input's root. Absent is the normal case and changes nothing.
- `Promote` gains the two local views. Replacements promote exactly like dependencies: the platform's whole, the instance's only for paths the platform's dependency list does not name. A replaced path absent from the promoted list is listed in the render `module.cue` with the same placeholder version the two inputs already use, so `VerifyCoverage` holds by construction. `LocalModuleFile` already writes every entry in `Replacements`; it needs no change.
- A version-less dependency in an input's `module.cue` that no promoted replacement covers is refused by `Promote` with an error naming the path and the input, instead of surfacing as a modfile formatting error later (`ParseModFile` keeps accepting the shape, since cue does).
- `Staged` gains the list of honoured replacements.

**`opm/kernel`:**

- `RenderInput` gains a boolean opt-in for local replacements, off by default. When it is off and an input carries a replacement, `Render` refuses before staging with an error naming the input and the file. Silent dropping (today's behaviour) is what made the gap invisible.
- `RenderDiagnostics` gains a `Replacements` slice: one row per honoured replacement carrying the path, the target and which input supplied it. Rows, not strings, per the existing no-presentation-strings requirement.

**Docs:** `CLAUDE.md` § Render contract, `opm/internal/renderstage/doc.go`, `opm/kernel/doc.go`, and the security-audit skill's Dimension 4 invariant, which today reads "the directory replacements the staging writes point only at the two staged input trees, never at a path an artifact names" and becomes "… and, only when the frontend opted in, at the directories an on-disk input's own local-module.cue names".

**Not in this change:** any frontend behaviour (the cli's D19 warning wording and its opt-in are a cli change that follows the library release); the operator, which never passes an input carrying the file and never sets the opt-in; carrying a replacement into the render's skew rows (a replaced path keeps its pinned versions in `ResolvedVersions`; the replacement row says where it was served from).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `single-build-render`: the promotion requirement gains replacement promotion with its precedence, the placeholder listing and the version-less refusal; a new requirement states the opt-in, the refusal when it is off (its own scenario), the diagnostics rows, and that an input without the file is unaffected.

## Impact

**SemVer:** MINOR. Two additive fields (`RenderInput`, `RenderDiagnostics`) and one new row type in `opm/kernel`. One behaviour change on an existing path, called out explicitly: a render whose input carries `cue.mod/local-module.cue` with a replacement now refuses unless the caller opted in, where today it silently ignores the file. Downstream migration cost: the cli sets one field at its single `kernel.Render` call site when it bumps the library; the operator changes nothing.

**Downstream:** `cli` (opts in, then rewords its D19 warning from the new diagnostics rows; separate change). `opm-operator`: none.

**Library:** `opm/internal/renderstage/{stage,promote,modfile}.go` and their tests; `opm/kernel/render.go` (two fields, one refusal, one decode); four doc paragraphs. Depends on `overlay-served-in-memory` landing first: both edit `Stage`.

**Complexity justification (Principle VII):** about 120 lines in `renderstage`, no new package, one new internal row type mirrored by one exported row type. The opt-in is a security boundary, not an extension point: it is the one place the render may be told to read a directory an artifact names.
