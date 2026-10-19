## 1. Batch library-B1: 28 requirements

Specs: `kubernetes-tier` (25), `api-diff-check` (3). Gate `N=28`. Section 1 also fixes the working method for the later batches: record the form choices in the ledger as the pattern.

- [x] 1.1 List `openspec/changes/` and confirm no open change other than `publish-go-api-bundle` edits a spec of this batch; list each spec's long requirements with `openspec validate <spec> --type spec --strict --json 2>&1`; the count must equal 28 (a difference is explained in the commit body or stops the section)
- [x] 1.2 For each long requirement choose the form (a, b, c2 or d) by the design.md table after reading its text, and search the repo for its old header before any rename, split or c2
- [x] 1.3 Write the delta in `specs/<spec>/spec.md` of this change for each spec of the batch: every body 500 or less with SHALL or MUST, each new requirement with a scenario that has a body, no `**Key**:` line, no heading of level 5 or deeper and no fenced block in a body
- [x] 1.4 Add the ledger tables for the batch to design.md (`### Ledger: <spec>`: scenario table and clause table)
- [x] 1.5 Run the byte-equality command of design.md for every rewritten requirement: exit 0, every `new:` line explained in the ledger
- [x] 1.6 Run the gate block of design.md with `N=28`: `GATE PASS`, WARNING lines fell by 28 from 153, no new ERROR item, no "Archive would refuse" line
- [x] 1.7 `task check` green, then commit `docs(openspec): split the long requirements of library-B1`

## 2. Batch library-B2: 28 requirements

Specs: `single-build-render` (19), `schema-dispatch` (9). Gate `N=56`.

- [ ] 2.1 List `openspec/changes/` and confirm no open change other than `publish-go-api-bundle` edits a spec of this batch; list each spec's long requirements with `openspec validate <spec> --type spec --strict --json 2>&1`; the count must equal 28 (a difference is explained in the commit body or stops the section)
- [ ] 2.2 For each long requirement choose the form (a, b, c2 or d) by the design.md table after reading its text, and search the repo for its old header before any rename, split or c2
- [ ] 2.3 Write the delta in `specs/<spec>/spec.md` of this change for each spec of the batch: every body 500 or less with SHALL or MUST, each new requirement with a scenario that has a body, no `**Key**:` line, no heading of level 5 or deeper and no fenced block in a body
- [ ] 2.4 Add the ledger tables for the batch to design.md (`### Ledger: <spec>`: scenario table and clause table)
- [ ] 2.5 Run the byte-equality command of design.md for every rewritten requirement: exit 0, every `new:` line explained in the ledger
- [ ] 2.6 Run the gate block of design.md with `N=56`: `GATE PASS`, WARNING lines fell by 56 from 153, no new ERROR item, no "Archive would refuse" line
- [ ] 2.7 `task check` green, then commit `docs(openspec): split the long requirements of library-B2`

## 3. Batch library-B3: 28 requirements

Specs: `kernel-runtime` (17), `artifact-types` (11). Gate `N=84`.

- [ ] 3.1 List `openspec/changes/` and confirm no open change other than `publish-go-api-bundle` edits a spec of this batch; list each spec's long requirements with `openspec validate <spec> --type spec --strict --json 2>&1`; the count must equal 28 (a difference is explained in the commit body or stops the section)
- [ ] 3.2 For each long requirement choose the form (a, b, c2 or d) by the design.md table after reading its text, and search the repo for its old header before any rename, split or c2
- [ ] 3.3 Write the delta in `specs/<spec>/spec.md` of this change for each spec of the batch: every body 500 or less with SHALL or MUST, each new requirement with a scenario that has a body, no `**Key**:` line, no heading of level 5 or deeper and no fenced block in a body
- [ ] 3.4 Add the ledger tables for the batch to design.md (`### Ledger: <spec>`: scenario table and clause table)
- [ ] 3.5 Run the byte-equality command of design.md for every rewritten requirement: exit 0, every `new:` line explained in the ledger
- [ ] 3.6 Run the gate block of design.md with `N=84`: `GATE PASS`, WARNING lines fell by 84 from 153, no new ERROR item, no "Archive would refuse" line
- [ ] 3.7 `task check` green, then commit `docs(openspec): split the long requirements of library-B3`

## 4. Batch library-B4: 28 requirements

Specs: `deps-cascade` (8), `instance-synthesis` (7), `cascade-wiring` (6), `config-validation` (6), `workflow-hardening` (1). Gate `N=112`.

- [ ] 4.1 List `openspec/changes/` and confirm no open change other than `publish-go-api-bundle` edits a spec of this batch; list each spec's long requirements with `openspec validate <spec> --type spec --strict --json 2>&1`; the count must equal 28 (a difference is explained in the commit body or stops the section)
- [ ] 4.2 For each long requirement choose the form (a, b, c2 or d) by the design.md table after reading its text, and search the repo for its old header before any rename, split or c2
- [ ] 4.3 Write the delta in `specs/<spec>/spec.md` of this change for each spec of the batch: every body 500 or less with SHALL or MUST, each new requirement with a scenario that has a body, no `**Key**:` line, no heading of level 5 or deeper and no fenced block in a body
- [ ] 4.4 Add the ledger tables for the batch to design.md (`### Ledger: <spec>`: scenario table and clause table)
- [ ] 4.5 Run the byte-equality command of design.md for every rewritten requirement: exit 0, every `new:` line explained in the ledger
- [ ] 4.6 Run the gate block of design.md with `N=112`: `GATE PASS`, WARNING lines fell by 112 from 153, no new ERROR item, no "Archive would refuse" line
- [ ] 4.7 `task check` green, then commit `docs(openspec): split the long requirements of library-B4`

## 5. Batch library-B5: 28 requirements

Specs: `fetch-error-classification` (5), `platform-artifact` (5), `platform-module-generation` (4), `registry-module-loading` (4), `render-parity` (4), `catalog-acquisition` (3), `fixture-pin-maintenance` (3). Gate `N=140`.

- [ ] 5.1 List `openspec/changes/` and confirm no open change other than `publish-go-api-bundle` edits a spec of this batch; list each spec's long requirements with `openspec validate <spec> --type spec --strict --json 2>&1`; the count must equal 28 (a difference is explained in the commit body or stops the section)
- [ ] 5.2 For each long requirement choose the form (a, b, c2 or d) by the design.md table after reading its text, and search the repo for its old header before any rename, split or c2
- [ ] 5.3 Write the delta in `specs/<spec>/spec.md` of this change for each spec of the batch: every body 500 or less with SHALL or MUST, each new requirement with a scenario that has a body, no `**Key**:` line, no heading of level 5 or deeper and no fenced block in a body
- [ ] 5.4 Add the ledger tables for the batch to design.md (`### Ledger: <spec>`: scenario table and clause table)
- [ ] 5.5 Run the byte-equality command of design.md for every rewritten requirement: exit 0, every `new:` line explained in the ledger
- [ ] 5.6 Run the gate block of design.md with `N=140`: `GATE PASS`, WARNING lines fell by 140 from 153, no new ERROR item, no "Archive would refuse" line
- [ ] 5.7 `task check` green, then commit `docs(openspec): split the long requirements of library-B5`

## 6. Batch library-B6: 12 requirements

Specs: `migration-docs` (3), `release-pipeline` (3), `duplicate-object-identities` (2), `helper-packages` (2), `test-fixture-registry` (2). Gate `N=152`.

- [ ] 6.1 List `openspec/changes/` and confirm no open change other than `publish-go-api-bundle` edits a spec of this batch; list each spec's long requirements with `openspec validate <spec> --type spec --strict --json 2>&1`; the count must equal 12 (a difference is explained in the commit body or stops the section)
- [ ] 6.2 For each long requirement choose the form (a, b, c2 or d) by the design.md table after reading its text, and search the repo for its old header before any rename, split or c2
- [ ] 6.3 Write the delta in `specs/<spec>/spec.md` of this change for each spec of the batch: every body 500 or less with SHALL or MUST, each new requirement with a scenario that has a body, no `**Key**:` line, no heading of level 5 or deeper and no fenced block in a body
- [ ] 6.4 Add the ledger tables for the batch to design.md (`### Ledger: <spec>`: scenario table and clause table)
- [ ] 6.5 Run the byte-equality command of design.md for every rewritten requirement: exit 0, every `new:` line explained in the ledger
- [ ] 6.6 Run the gate block of design.md with `N=152`: `GATE PASS`, WARNING lines fell by 152 from 153, no new ERROR item, no "Archive would refuse" line
- [ ] 6.7 `task check` green, then commit `docs(openspec): split the long requirements of library-B6`
