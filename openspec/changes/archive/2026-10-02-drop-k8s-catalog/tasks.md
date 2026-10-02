## 1. Test samples and generator comment (opm/helper/platformmodule)

- [x] 1.1 `generate_test.go`: rename `k8sPath` to `extraPath`, value `example.com/catalogs/extra@v1`; update the four literal uses. `closure_test.go`: the fake graph node and every `k8sPath` use follow. Verify: `grep -rn 'k8s' opm/helper/platformmodule` shows only `cue.dev/x/k8s.io`.
- [x] 1.2 `.cascade-frozen`: drop the retired path from the two `pins` lists. Verify: the file names only paths those tests still hold as OPM-owned literals.
- [x] 1.3 `generate.go`: the comment says "the convention the first-party catalog follows".
- [x] 1.4 `go test ./opm/helper/platformmodule/...` green, then commit `test(helper): use a neutral second catalog in samples`.

## 2. Beta-line lists (README, CONSTITUTION, openspec/config.yaml)

- [x] 2.1 Drop the retired path from the three beta-line lists; `adr/010` stays as written.
- [x] 2.2 `task check` green, then commit `docs: drop the retired k8s catalog from the beta-line lists`.

## 3. Spec text and archive (openspec)

- [x] 3.1 Delta specs for `fixture-pin-maintenance` and `platform-module-generation`; `openspec validate drop-k8s-catalog --strict` green. Commit `docs(openspec): propose dropping the retired k8s catalog`.
- [x] 3.2 `openspec archive` the change; main specs carry the new text; `openspec validate --all --strict` green; commit `docs(openspec): archive drop-k8s-catalog`.
