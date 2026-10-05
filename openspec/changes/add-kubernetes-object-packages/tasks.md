# Tasks: add-kubernetes-object-packages

Worktree `library/.claude/worktrees/add-kubernetes-object-packages`, branch
`feat/add-kubernetes-object-packages` (from `origin/main` `a8bfc76`). Seed `.cue-cache` by
copying the main checkout's (`cp -a`), never by symlinking. Every command runs inside the
worktree with an absolute private `TMPDIR` from `mktemp -d` under the session scratchpad. The
registry env goes on two lines:

```bash
export CUE_REGISTRY='opmodel.dev=ghcr.io/open-platform-model,testing.opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'
export OPM_REGISTRY="$CUE_REGISTRY"
```

`TestGenerate_BuildsThroughTheKernel` can fail in a full-suite run because of a known
cross-process cache race. Rerun `go test ./opm/helper/platformmodule -count=1` on its own before
treating it as a finding. Commit bodies never start a line with `word(` and carry no bare
at-sign. The only trailer is `Co-Authored-By: Claude <noreply@anthropic.com>`. Source citations
for ported code: cli `origin/main` `0b37e3f2` (re-checked unchanged at `1338e700`), opm-operator
`origin/main` `9b83611`.

## 1. Spike: Go 1.26.0 floor, lint pin and the tier allow list (go.mod, lint; design KO1, KO9)

- [ ] 1.1 `go.mod`: `go 1.26.0` (SD10). `go mod tidy` leaves the requirements otherwise
      unchanged, because apimachinery arrives in section 3 with its first importer. Verify:
      `go build ./... && go vet ./...` clean.
- [ ] 1.2 Lint pin probe, not committed. Install golangci-lint `2.8.0` from its release
      archive into the scratch dir, with the sha256 check `lint.yml` uses, and run it against
      the tree. If it refuses the Go 1.26 target, set `lint.yml` `GOLANGCI_LINT_VERSION` to
      `2.11.3` (the cli's pin) and `GOLANGCI_LINT_SHA256` to the
      `golangci-lint-2.11.3-linux-amd64.tar.gz` line of that release's `checksums.txt`, both in
      one edit. If 2.8.0 accepts it, leave `lint.yml` alone. Record the outcome as one sentence
      under KO1 in design.md. Verify: the pinned version runs `task lint` green.
- [ ] 1.3 `.golangci.yml`: add `k8s-labels-imports-only-stdlib` and `k8s-tier-allow-list` as
      design KO9 describes (strict mode, `!**/*_test.go`). Each comment says what the rule
      keeps out and why, and the allow-list comment cites ADR-011 item 2's question as
      answered. Verify: `task lint` green on the tree as it is (no `opm/k8s` files yet).
- [ ] 1.4 Mutation checks, not committed. Create scratch files under `opm/k8s/labels/` and
      `opm/k8s/object/` with `package` lines only, then confirm that `task lint`:
      - fails for `opm/k8s/labels` importing `k8s.io/apimachinery/pkg/runtime/schema`;
      - fails for `opm/k8s/object` importing `github.com/google/uuid` (allow list);
      - fails for `opm/k8s/object` importing `k8s.io/client-go/...` (lax deny still bites
        beside the strict rule);
      - fails for `opm/k8s/object` importing `.../opm/internal/modversion`;
      - passes for `opm/k8s/object` importing `k8s.io/apimachinery/...`, `cuelang.org/go/cue`
        and `.../opm/kernel`.

      Delete the scratch files. Record the outcome in design KO9: either "every matching rule
      applies and `$gostd` works in <version>" or the adjusted rule shape. Verify: `git status`
      shows no scratch file left.
- [ ] 1.5 `task check` green, then commit
      `build: raise the Go floor to 1.26.0 and fence the tier's imports`. The body names the
      lint pin outcome and the two new depguard rules.

## 2. opm/k8s/labels (design KO2)

- [ ] 2.1 `opm/k8s/labels/labels.go` and `doc.go`. The constants and `IsOPMManagedBy` per
      design KO2, values byte-equal to cli and operator `pkg/core/labels.go`. Each doc comment
      says who writes the label. The package doc says it names and recognises labels and never
      stamps them on rendered objects, because core's CUE stamps them at render (0012:D6, cited
      once at the package). Verify: `go build ./opm/k8s/...` clean.
- [ ] 2.2 `labels_test.go`:
      - a table pinning each constant to its literal (kubernetes-tier scenario "The vocabulary
        matches the frontends' copies");
      - the `IsOPMManagedBy` table, ported from `pkg/core/labels_test.go` and extended with
        `helm`, `""` and `OPM-CLI` (scenario "Every OPM managed-by value is recognised").

      Verify: `go test ./opm/k8s/labels -count=1` green;
      `go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./opm/k8s/labels` prints
      only the package itself.
- [ ] 2.3 `task check` green, then commit `feat(k8s): add the opm/k8s/labels vocabulary`.

## 3. opm/k8s/object: Resource and the one-pass Export (go.mod; design KO3, KO4)

- [ ] 3.1 `opm/k8s/object/resource.go`: `Resource` with fields and accessors ported verbatim
      from `pkg/core/resource.go` (including `parseAPIVersion` and the best-effort lookups),
      plus `NewResource` and `Resources` per KO3. `convert.go`: `MarshalJSON` and
      `ToUnstructured` ported verbatim. `doc.go` covers four things:
      - the package's place in the tier;
      - that a `Resource` pins its build until the caller drops it (holder-bounded);
      - that `Export` is the one export a caller needs;
      - that the kernel never imports this package.

      `go get k8s.io/apimachinery@v0.36.4`, then `go mod tidy`. Verify: `go.mod` requires
      `k8s.io/apimachinery v0.36.4` directly; `go build ./...` clean.
- [ ] 3.2 `opm/k8s/object/export.go`: `Exported`, `Export`, `ExportStep`
      (`ExportMarshal`, `ExportDecode`) and `ExportError` (`Error`, `Unwrap`) per KO4. Each
      Resource is exported once through `MarshalJSON` and decoded from those bytes. The input
      is never written. Export stops at the first failure. Verify: `go vet ./opm/k8s/...`
      clean.
- [ ] 3.3 Tests:
      - `resource_test.go`: port the frontends' `pkg/core/resource_test.go` cases;
        `NewResource(nil)` is nil; `Resources` skips nil entries and keeps order; the
        best-effort accessors (scenario "Accessors are best-effort").
      - `export_test.go`: `Exported.JSON` equals each Resource's `MarshalJSON`; `Object`
        equals `json.Unmarshal` of that JSON; provenance and order are kept (scenario "One
        export feeds every consumer of the object"); input untouched (scenario "The input
        survives the export"); a value with a non-concrete field fails with `*ExportError`,
        `Step == ExportMarshal`, the right `Index` and the Resource named in the message
        (scenario "A failing export names the resource and the step"); a concrete list
        (`[1]`), a string and `null` each fail with `*ExportError`, `Step == ExportDecode` and
        the right `Index` (scenario "A value that is not an object fails at the decode");
        `errors.As` and `Unwrap` work.

      Verify: `go test ./opm/k8s/object -count=1` green.
- [ ] 3.4 `task check` green, then commit
      `feat(k8s): add opm/k8s/object with Resource and a one-pass Export`. The body says that
      `k8s.io/apimachinery v0.36.4` enters `go.mod`.

## 4. Kind-class order and apply stages (object; design KO5, KO6, KO7)

- [ ] 4.1 `opm/k8s/object/weights.go`: the constants, the GVK table and the kind table from
      cli `pkg/resourceorder/weights.go` at `0b37e3f2`, values unchanged. `GetWeight` becomes
      `Weight`. The file comment cites the source once: "ported from cli pkg/resourceorder
      (cli#289), values unchanged". `sort.go`: `Direction`, `Ascending`, `Descending` and
      `Sort[T]` unchanged. Verify: a scripted diff of the two tables against
      `git -C <cli> show 0b37e3f2:pkg/resourceorder/weights.go`, after normalising the package
      name and the `GetWeight` rename, shows no value change (not committed).
- [ ] 4.2 `opm/k8s/object/stages.go`: `Stage[T]` and `Stages` per KO6, with an unexported
      cluster-definition predicate on group and kind. The input is not reordered and no stage
      is empty. Verify: `go vet ./opm/k8s/...` clean.
- [ ] 4.3 Tests:
      - `weights_test.go` and `sort_test.go`: port the cli's tests; a guard table pinning
        every constant and every table entry; the kubernetes-tier scenarios "Apply order puts
        definitions before their users", "Delete order is the reverse and stable" and "An
        unknown version of a known kind falls back to the kind".
      - `stages_test.go`: the scenarios "Stages for a typical module", "A Namespace kind in
        another group is not a cluster definition" and "No cluster definitions, no definition
        stage"; the input slice is unchanged after the call.
      - `flux_order_test.go` (KO7): Flux ssa v0.77.0 `ReconcileOrder.First` and `.Last` as
        literals; the universe is those 25 kinds, the 26 kind-table kinds and `Widget`, with
        unlisted kinds at Flux rank 0; a pair counts only when the Flux ranks differ and the
        library weights are strictly opposite; the set is compared with a committed list (each
        entry commented), and CustomResourceDefinition before Namespace is asserted to be no
        contradiction (scenario "Kinds Flux does not list are compared at rank 0"). Negative check, not committed: lowering `WeightDeployment` below
        `WeightService` fails it naming that pair (scenario "A weight edit that creates a
        contradiction is caught").

      Verify: `go test ./opm/k8s/object -count=1` green.
- [ ] 4.4 `task check` green, then commit
      `feat(k8s): add the kind-class weight table and apply stages`. The body cites the cli
      source commit and says that SD11's per-weight-group apply reads `Stages`.

## 5. Duplicate identities in the tier; the helper copy deprecated (object, helper/objectset, kernel tests; design KO8, KO10)

- [ ] 5.1 `opm/k8s/object/duplicates.go`: copy of `opm/helper/objectset/objectset.go` with the
      same exported names, signature and wording. Move the objectset package doc's content
      into the `object` package doc's duplicates paragraph. `duplicates_test.go`: the helper
      test, ported whole. Append an entry for `opm/k8s/object/duplicates_test.go` with the pin
      `opmodel.dev/catalogs/opm@v4` at the end of `.cascade-frozen`. Verify: `go test ./opm/k8s/object -count=1` green; a scripted diff
      of the two implementation files and of the two test files, after normalising package
      and import names, is empty (duplicate-object-identities scenario "Both homes agree on a
      render"; not committed).
- [ ] 5.2 `opm/helper/objectset`: add a `Deprecated: use opm/k8s/object.<Name>` paragraph to
      the package doc and to each exported symbol (`Identity`, `Producer`, `Duplicate`,
      `Duplicates`, `DuplicateIdentitiesError`), with no code change (scenario "The deprecated
      home says where to go"). Add `//nolint:staticcheck // SA1019: the deprecated copy is
      tested until its removal` on the import and on the `objectset.Duplicates` call in
      `opm/kernel/flow_integration_test.go` and in `opm/kernel/render_test.go`.
      `objectset_test.go` needs none (staticcheck exempts a package's own external test
      package). Verify: `task lint` green, and with the nolints in
      `flow_integration_test.go` removed it reports SA1019 (not committed).
- [ ] 5.3 Whole-tree checks on the final code:
      - `task check`;
      - `go test -race ./opm/k8s/... -count=1`;
      - `openspec validate add-kubernetes-object-packages --strict`.

      Consumer check, not committed: build and vet the cli and the opm-operator at their
      `origin/main` against this tree through a scratch `go.work` or `-modfile` with a
      `replace` (`.tasks/consumer-build.sh` where it fits). Both must build and vet. Also run
      each frontend's `golangci-lint run --enable-only staticcheck` against this tree and
      record the SA1019 findings count per frontend for the report (design KO10). Verify:
      both builds green and the counts recorded.
- [ ] 5.4 `task check` green, then commit
      `feat(k8s): move duplicate identity detection into opm/k8s/object`. The body says that
      `opm/helper/objectset` stays, deprecated, until both frontends migrate.

## 6. Docs say what exists (README, AGENTS, CONSTITUTION, ADR-011, helper doc, kernel doc; design KO11)

- [ ] 6.1 Locate every edit by text, not by line number:
      - `README.md` § Helper boundary: "the planned Kubernetes tier" becomes the tier with
        `labels` and `object`; the lint paragraph counts the new rules. The Layout tree gains
        `k8s/labels/` and `k8s/object/` rows, and its `objectset/` row is marked Deprecated
        with its replacement. The bullet that presents `opm/helper/objectset` as the place for
        Kubernetes vocabulary names `opm/k8s/object` instead.
      - `AGENTS.md`: the rules line ("`opm/k8s/` (planned, no package yet)"); the layout
        block (the `k8s/` row becomes `k8s/labels/` and `k8s/object/` rows, and
        `helper/objectset/` is marked Deprecated with its replacement); `k8s` in the commit
        scopes; one bullet under "CUE toolchain pin" on the apimachinery MVS floor (no bump
        policy, SD24).
      - `CONSTITUTION.md` Principle III: "`opm/k8s/` (planned)" becomes the tier as it now
        exists; the helper list that names objectset marks it Deprecated.

      Verify: `grep -n "planned" README.md AGENTS.md CONSTITUTION.md` shows no `opm/k8s`
      hit; `grep -n objectset README.md AGENTS.md CONSTITUTION.md | grep -v -i deprecated`
      shows no line presenting it as current; the layout block is still one code fence.
- [ ] 6.2 `adr/011-kubernetes-tier-beside-the-kernel.md`: one Status sentence, "Amended
      2026-10-05 by add-kubernetes-object-packages", covering the first packages `labels` and
      `object`, item 9 carried out as copy, deprecate, then remove (SD1), and item 2's
      allow-list question answered yes. Item 2's "held by review until..." sentence is
      rewritten to the rule as it now stands. `opm/helper/doc.go`: the objectset bullet says
      it is Deprecated in favour of `opm/k8s/object`, and "opm/k8s/ (planned)" loses
      "(planned)". `opm/kernel/render.go` `Compiled` doc: a Kubernetes consumer wraps it in
      `opm/k8s/object.Resource` (prose only). Verify: `go doc ./opm/helper` and
      `go doc ./opm/kernel Compiled` read correctly; `task docs:bundle:check` green.
- [ ] 6.3 `task check` green and `openspec validate add-kubernetes-object-packages --strict`
      passes, then commit `docs(k8s): record the first Kubernetes tier packages`.

## 7. Archive (at PR time, on the supervisor's word)

- [ ] 7.1 `openspec archive add-kubernetes-object-packages --yes` on this branch, so the
      archive rides the implementing PR. Verify: the three main specs carry the changes and
      `openspec validate --all --strict` passes. `enhancement.yaml` claims no decision, so the
      delivery log runs with an empty claim, or the supervisor skips it.
- [ ] 7.2 Commit `chore(openspec): archive add-kubernetes-object-packages`.
