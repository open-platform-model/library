# Tasks: render-local-replacements

## 1. Spike

- [x] 1.1 `opm/internal/renderstage/stage_test.go`: hand-write a render module whose `local-module.cue` replaces (a) an instance-only path with a temp directory holding a never-published module and (b) the platform's catalog path with a copy of the fixture catalog carrying one changed label, build it with `Build`, and assert the instance's import resolved from (a) and the rendered label came from (b); verify the test passes against a hand-written staging directory before any promotion code exists, or record the cue/load error in `design.md` and stop.

## 2. Local view

- [x] 2.1 `opm/internal/sourcetree/sourcetree.go`: a missing overlay entry in `ReadFile` returns an error wrapping `fs.ErrNotExist`; verify `errors.Is(err, fs.ErrNotExist)` holds for both modes in `sourcetree_test.go`.
- [x] 2.2 `opm/internal/renderstage/modfile.go`: `ReadModFile` keeps the parsed `*modfile.File`; add `ReadLocalModFile(src, base)` returning a local view (replaced path → target, directory targets made absolute against `src.Root`, module-path targets verbatim, plus the local file's own listed entries), nil when absent; verify `modfile_test.go` covers absent, relative directory, absolute directory, module-path target, and a malformed file naming the input.

## 3. Promotion

- [x] 3.1 `opm/internal/renderstage/promote.go`: `Promote` takes the two local views; platform replacements whole, instance replacements only for paths absent from `platform.Deps`; every replaced path listed in `Deps` with `ReplacedVersion` when its version is empty or the path is absent; `Promotion` gains `Rows []ReplacementRow` (path-sorted, `By` platform|instance); verify `promote_test.go`/`modfile_test.go` cover: platform catalog replacement, inert instance replacement on a platform path, honoured instance-only replacement, replace-only version-less dependency gets the placeholder, and `modfile.ParseLocal` accepts the written pair.
- [x] 3.2 `promote.go`: a dependency with an empty version that no promoted replacement covers is refused with an error naming the path and the input; verify a test pins the message and that `ModuleFile` is never reached with an empty version.
- [x] 3.3 `stage.go`: `Stage` gains a `localReplacements bool` argument; when false and either local view carries a replacement it returns an error naming the input and `cue.mod/local-module.cue` before writing anything; when true it reads both local views, passes them to `Promote`, and sets `Staged.Replacements`; verify `TestStage_*` cover refusal (no files written), absence under both settings (byte-identical `cue.mod` pair), and the two honoured cases writing the expected `replaceWith` lines.

## 4. Kernel surface

- [x] 4.1 `opm/kernel/render.go`: `RenderInput.LocalReplacements bool`, `Replacement{Path, Target, By}` and `RenderDiagnostics.Replacements` (copied from `Staged.Replacements` in path order); `render` passes the flag to `Stage` and wraps its refusal; verify `render_test.go` adds: refusal with the flag off against a temp platform copy carrying a replacement (no staging dir left behind), rows present with the flag on, and byte-identical output for fixture inputs under both settings.
- [x] 4.2 `render_test.go`: end-to-end through `Kernel.Render`: a temp copy of `testdata/render/platform` whose `local-module.cue` replaces the fixture catalog with a temp directory copy carrying one changed label renders that label; a temp copy of `testdata/render/instance` importing a never-published `test.example/lib@v0` (version-less in `module.cue`, replaced in `local-module.cue`) renders; verify both pass with `-race`.

## 5. Docs

- [x] 5.1 `CLAUDE.md` § Render contract (local replacements: opt-in, precedence, refusal), `opm/internal/renderstage/doc.go`, `opm/kernel/doc.go`, and `.claude/skills/security-audit/SKILL.md` Dimension 4 (the invariant now reads: replacements point at the two staged input trees and, only under `RenderInput.LocalReplacements`, at directories an input's own `local-module.cue` names); verify `grep -rn 'never at a path an artifact names' .claude CLAUDE.md` is empty.

## 6. Validation gates

- [x] 6.1 `task fmt`, `task vet`, `task lint`, `task test` green; `go test -race ./opm/kernel/... ./opm/internal/renderstage/...` green; parity harness green against GHCR; verify `openspec validate --changes` passes.
