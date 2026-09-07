# Tasks: move-compat-to-cli

## 1. Precondition and the enhancements record

- [ ] 1.1 Verify `cue-owned-verdicts` has landed: `opm/internal/renderstage/alternatives.go` no longer exists on `main`, `go list -deps ./opm/kernel/... ./opm/internal/... | grep opm/compat` is empty, and `grep -rn 'opm/compat' opm --include='*.go' | grep -v '^opm/compat/'` is empty.
- [ ] 1.2 In `../enhancements`, as its own commit: `0020/06-operational.md` Cross-Repo Coordination item 3 states that the cross-build rules live in `cli/internal/compat` and that the library-before-cli constraint no longer applies to them; the four comments in `0020/schemas/target.cue` naming `library/opm/compat` (near lines 34, 161, 174 and 232) name `cli/internal/compat`; verify `task vet` and `task check` pass in `enhancements/` and `grep -rn 'library/opm/compat' 0020` is empty.

## 2. Remove the package

- [ ] 2.1 `git rm -r opm/compat`; `.golangci.yml`: drop `**/opm/compat/**` from the `kernel-never-imports-helper` file list; verify `go build ./...`, `task lint` and `task test` are green and `grep -rn 'opm/compat' opm .golangci.yml` is empty.

## 3. Docs

- [ ] 3.1 `CONSTITUTION.md` III: drop the `opm/compat/` line; `CLAUDE.md`: drop the `compat/` layout line and `compat` from the commit-scope list (the "two independent compat tracks" paragraph is about SemVer and stays); `README.md`: drop the `compat/` layout line and `opm/compat` from the depguard sentence; verify `grep -rn 'opm/compat\|  compat/' CLAUDE.md CONSTITUTION.md README.md docs` is empty.

## 4. Validation gates

- [ ] 4.1 `task fmt`, `task vet`, `task lint`, `task test` green; verify `openspec validate --changes` passes.
- [ ] 4.2 In `../cli` against a temporary `replace github.com/open-platform-model/library => ../library`: copy `compat.go`, `level.go`, `compat_test.go` and `level_test.go` into `internal/compat/` (package doc without the library-matching consumer), declare `highestStable` in `internal/scaffold` with `predecessor.go`'s doc comment and the four `predecessor_test.go` cases, repoint the import in `internal/publish/compat.go`, run `go mod tidy`; verify `go build ./...`, `go vet ./...` and `go test ./internal/compat/... ./internal/scaffold/... ./internal/publish/...` pass and `grep -rn 'library/opm/compat' .` is empty; then revert the replace (the edits ship in the cli PR).
