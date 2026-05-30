# Tasks

## 1. Delete catalog source

- [x] 1.1 `git rm -r library/modules/opm/` (whole tree: resources, traits,
      blueprints, transformers, schemas, `catalog.cue`, `identity/`, `cue.mod/`).
- [x] 1.2 `git rm library/modules/out.cue` (stale `#Platform` artifact importing
      dead `opmodel.dev/modules/opm/*` paths).

## 2. Remove the library-side publish path

- [x] 2.1 `git rm library/.github/workflows/publish-catalog.yml`.
- [x] 2.2 Remove the `cue:publish:catalog` task from `library/Taskfile.yml`
      (the whole task block, incl. `CATALOG_DIR` / `.build/catalog` staging).
- [x] 2.3 Remove the `catalogs/opm:` entry from `library/cue-versions.yml` and
      rewrite the header comment so it no longer describes a catalog publish flow.

## 3. Fix stale doc/comment references

- [x] 3.1 `library/CLAUDE.md`: drop `modules/opm` from the "vendors CUE modules
      under …" line and remove the `task cue:publish PATH=modules/opm` example.
- [x] 3.2 `library/Taskfile.yml`: update the CUE-module-lifecycle header comment
      (line ~160) and the `cue:publish` help example to use `modules/opm_platform`.

## 4. Retire the specs

- [x] 4.1 `git rm -r openspec/specs/catalog-packaging/`.
- [x] 4.2 `git rm -r openspec/specs/catalog-publishing/`.

## 5. Validation gates

- [x] 5.1 `task cue:discover` lists only `modules/opm_platform` +
      `testdata/modules/web_app` (no `modules/opm`).
- [x] 5.2 `task cue:vet` passes (fixtures resolve the catalog from GHCR).
- [x] 5.3 `task cue:catalog:drift` passes (web_app on latest `v0.4.0`;
      opm_platform reported as "does not depend on the catalog").
- [x] 5.4 `task cue:test:flow` passes (or skips cleanly if GHCR unreachable).
- [x] 5.5 `task check:fast` (fmt + vet + test) passes — no Go change, sanity only.
- [x] 5.6 `grep -rn "modules/opm\b" library/ --include=*.go --include=*.cue
      --include=*.yml` returns no live references outside `openspec/changes/archive`.
