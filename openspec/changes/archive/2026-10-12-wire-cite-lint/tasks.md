## 1. Move the pin

- [x] 1.1 Replace `6df4000f6cabbf460a5cad6431b6bc9044fa3027` with `2b66da51fede4332969c324b069012632d8edea9` on every `.github` reference in `.github/workflows`, keeping ` # .github main`.
- [x] 1.2 Replace `.tasks/cascade/wiring-check.sh` with the canonical file at `2b66da51`; `cmp` reports no difference.
- [x] 1.3 Declare `{file: lint.yml, kind: cite-lint}` in `.tasks/cascade/wiring-check.yaml`. Commit the pin move and the copy first; commit the declaration with the job in 2.2 so that the check passes at each commit.

## 2. Wire cite-lint

- [x] 2.1 Add the task `cite-lint` to `Taskfile.yml`.
- [x] 2.2 Add the job `cite-lint` to `.github/workflows/lint.yml`.
- [x] 2.3 `task cite-lint`, `task cascade:wiring:check` and `task check` pass; `actionlint` and `zizmor` are clean on the changed files.

## 3. Verify and archive

- [x] 3.1 `openspec validate wire-cite-lint --strict` passes.
- [x] 3.2 Archive the change and commit.
