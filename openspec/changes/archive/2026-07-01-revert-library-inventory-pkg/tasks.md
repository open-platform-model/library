## 1. Delete the package

- [x] 1.1 `git rm -r opm/inventory/` (all 10 files: `entry.go`, `entry_test.go`,
      `digest.go`, `digest_test.go`, `stale.go`, `stale_test.go`, `existence.go`,
      `existence_test.go`, `imports_test.go`, `doc.go`).

## 2. Clean up dependencies

- [x] 2.1 `task tidy` (go mod tidy) — expect `k8s.io/apimachinery` and its
      transitive closure (`fxamacker/cbor`, `go-logr/logr`, `json-iterator/go`,
      `modern-go/*`, `x448/float16`, `go.yaml.in/yaml/v2`, `gopkg.in/inf.v0`,
      `k8s.io/klog/v2`, `k8s.io/kube-openapi`, `k8s.io/utils`, `sigs.k8s.io/json`,
      `sigs.k8s.io/randfill`, `sigs.k8s.io/structured-merge-diff/v6`) to drop out
      of `go.mod`/`go.sum`. Diff against PR #34's `go.mod`/`go.sum` change to
      confirm a clean reversal.

## 3. Retire the spec

- [x] 3.1 `git rm -r openspec/specs/inventory/` (main spec for the removed
      capability — this change's `specs/inventory/spec.md` REMOVED-delta
      supersedes it on archive/sync).

## 4. Record the breaking change

- [x] 4.1 Add an entry to `MIGRATIONS.md` under `## Unreleased — Breaking`:
      `### Removed — \`revert-library-inventory-pkg\`` — the `opm/inventory`
      package (shipped in `1.0.0-alpha.4`) is deleted in full; no known
      importer exists (`opm-operator` remained pinned to `v1.0.0-alpha.3`,
      `cli` never depended on it). Reference D31 (`enhancements/0006`).

## 5. Validation gates

- [x] 5.1 `grep -rn "opm/inventory" --include=*.go --include=*.md .` returns no
      live references outside `openspec/changes/archive/2026-06-30-library-inventory-pkg/`
      and this change's own directory.
- [x] 5.2 `task check` (fmt + vet + lint + test) passes with the package gone.
- [x] 5.3 `task build` passes (no dangling import anywhere in the module).

## 6. Land the change

- [x] 6.1 Commit as `revert(inventory): remove runtime-neutral shared inventory
      package`, referencing D31 and PR #34 / commit `27acbfa`, with a
      `BREAKING CHANGE:` footer and a `Migration: revert-library-inventory-pkg`
      trailer. Branch first (not on `main`).
- [x] 6.2 Open the PR; do not merge without explicit go-ahead. Opened as
      https://github.com/open-platform-model/library/pull/36, merged by the
      user as `738a694`.
- [x] 6.3 After merge, archive this OpenSpec change
      (`openspec/changes/archive/YYYY-MM-DD-revert-library-inventory-pkg/`),
      syncing the `REMOVED` delta so `openspec/specs/inventory/` is gone from
      the main spec tree. (The delta was already applied directly in the PR
      itself — `openspec/specs/inventory/spec.md` was deleted in the same
      commit — so archiving here is a move-only operation, no further sync.)
- [x] 6.4 Update `enhancements/0006/planned-changes.md` (A3 row: `reverted-pending`
      → `reverted`, with the real PR/commit + this change's slug) and append a
      `history` event to `enhancements/0006/config.yaml`.
