# Tasks: overlay-served-in-memory

## 1. Spike

- [x] 1.1 `opm/internal/renderstage/stage_test.go`: stage an overlay-mode instance (a `Source` with `Overlay` re-keyed under `<dir>/instance`) against an on-disk platform, assert `<dir>/instance` does not exist on disk after `Stage`, call `Build` with the overlay passed as `load.Config.Overlay`, and assert `_components` on the built value resolves without error; verify the test passes against a local prototype of the change, or record the cue/load error it produces in `design.md` and stop.

## 2. Serve overlays from memory

- [x] 2.1 `stage.go`: `Staged` gains `Overlay map[string][]byte`; `serveDir` returns `<dir>/<name>` for an overlay-mode source and appends its entries re-keyed from `Source.Root` to that path (entries outside `Root` refused as `WriteTo` refuses them), writing nothing; `Build` sets `cfg.Overlay` from `Staged.Overlay` with `load.FromBytes`; verify `go test ./opm/internal/renderstage/... ./opm/kernel/...` is green, the parity harness passes, and `TestStage_*` asserts no `instance/` or `platform/` directory exists under the staging directory for overlay inputs.

## 3. Docs

- [x] 3.1 `CLAUDE.md` § Render contract ("stages one generated render module": say overlay inputs are served from memory and the directory holds `cue.mod` and the glue), `opm/internal/renderstage/doc.go`, `opm/kernel/doc.go`; verify `grep -rn 'WriteTo' opm/internal/renderstage` is empty and the artifact-types spec's writer sentence matches the code after archive.

## 4. Validation gates

- [x] 4.1 `task fmt`, `task vet`, `task lint`, `task test` green; `go test -race ./opm/kernel/... ./opm/internal/renderstage/...` green; `task cue:test:flow` green against GHCR; verify `openspec validate --changes` passes.
