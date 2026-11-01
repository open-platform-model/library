## 1. Add the glossary

- [x] 1.1 Create `GLOSSARY.md` at the repo root with the domain terms of the library, each with a definition and an `_Avoid_` line.
- [x] 1.2 Add one rule line to `AGENTS.md` and one router row to `CONTEXT.md` that point to `GLOSSARY.md`.
- [x] 1.3 Commit `docs: add glossary of the domain terms`

## 2. Verify and archive

- [ ] 2.1 `openspec validate add-glossary --strict --no-interactive` passes.
- [ ] 2.2 Archive the change with `skip_specs: true` (no spec sync) and commit `chore(openspec): archive add-glossary`.
