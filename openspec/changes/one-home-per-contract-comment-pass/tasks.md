Depends on: nothing unmerged. Merges after the other round-3 library changes, by owner decision c4 (proposal.md, Impact). Line numbers are at `origin/main` ca7c56b. Re-read every site by its symbol before editing. Comment, prose and spec text only: no identifier, signature, error string or test changes, and every section's diff touches only comment lines in `.go` and `.sh` files (`git diff -U0 -- '*.go' '*.sh' | grep '^[-+][^-+]' | grep -v '^[-+]\s*//\|^[-+]\s*#'` prints nothing). Every section's gate: `TMPDIR=<absolute mktemp -d> task check` (which includes `docs:bundle:check`, `api:diff:test`, `consumer-build:test` and `cascade:wiring:check`) and `openspec validate one-home-per-contract-comment-pass --strict` pass.

## 1. False claims in the Go comments

- [ ] 1.1 `opm/module/source.go:11-37`: cut the mode list to the two modes and their meaning. Replace the carrier paragraph with the per-artifact list in design.md D2: Module (registry and directory, overlay), Instance (synthesis overlay; directory on disk, overlay with values sources), Platform (directory, on disk), Catalog (registry and directory, overlay). Name the `platform.Source` and `catalog.Source` aliases, and keep the bare-value sentence.
- [ ] 1.2 `opm/module/source.go:55-56` and `opm/internal/loader/load.go:93-95`: the D3 texts. No count of cue/load overlay sites.
- [ ] 1.3 `opm/schema/paths.go:5-14` and the group comments: name each path's readers as the ADDED schema-dispatch requirement "Path inventory names every reader" lists them, including `Instance.Values`, `Instance.ModuleMetadata` and `Module.DebugValues`. Before writing, grep each path and confirm its reader set.
- [ ] 1.4 Verify: `grep -rn 'registry path only\|one place it calls\|The one place the library hands' opm` prints nothing. `go doc ./opm/module Source` reads as D2.
- [ ] 1.5 Gates green, then commit `docs(module): say which artifacts carry a source and in which mode`.

## 2. One home per runtime contract in godoc

- [ ] 2.1 `opm/kernel/acquire.go`: in the `loadEnv` doc (`:31-34`), say the mapping is applied through `cueenv.Override` and drop "never os.Setenv". In the exported verb docs (`:89-92` module, `:219-224` catalog, `:255-258` platform), the registry sentence becomes "The registry mapping is the kernel's ([WithRegistry])". The package doc already states that it is never written to the environment, so the verb docs do not repeat it. Check the instance verb and the registry verbs for the same phrase.
- [ ] 2.2 `opm/internal/loader/load.go:19-24` (`Options.Env`): link `[cueenv.Override]` for the rule; drop "never os.Setenv".
- [ ] 2.3 Confirm that the `opm/internal/cueenv` package doc and the `opm/kernel` package doc § Surface each state the env rule in full (D1). Add the missing clause to its home if either does not.
- [ ] 2.4 `opm/kernel/render.go:306-316`: reflow the `Render` doc paragraph to about 75 columns with the same words (`gofmt` keeps it). Check that "whose references the kernel drops when Render returns" still reads as holder-bounded.
- [ ] 2.5 D5: move the ADR pointers out of the package docs of `opm/k8s/object` (`doc.go:3`, `:13`), `opm/k8s/labels` (`doc.go:11`) and `opm/helper/objectset` (`doc.go:31`). Each gets one non-doc comment after the package clause, in the shape of `opm/kernel/doc.go:290-293`.
- [ ] 2.6 Verify: `grep -rn 'os.Setenv' --include='*.go' opm | grep -v _test.go` lists only the `cueenv` package doc and the `OCILoader` doc. No package doc of an exported package under `opm/` contains `ADR-`; check with `go doc` on each package or with a grep over the comment block above each `package` clause. Run `task docs:bundle` and read `out/library/` for the touched packages.
- [ ] 2.7 Gates green, then commit `docs(kernel): link restated contracts to their godoc homes`.

## 3. Prose copies become links

- [ ] 3.1 `AGENTS.md` § Kernel API surface and § Render pipeline (`:327-357`): apply D4. They become pointers to the godoc homes (the `opm/kernel` package doc, `Kernel.Render`, `RenderError`, `Compiled`, and the `opm/internal/renderstage` package doc), plus the maintainer-only bullets (the absence test in `opm/kernel/kernel_test.go`, and no reintroduced verbs). Before deleting a claim, find it in its home. If it is missing, add it to that godoc in this commit.
- [ ] 3.2 `AGENTS.md`: add the "Where a statement lives" subsection (D8) beside "Enhancement references in comments".
- [ ] 3.3 `README.md` § Render (`:66-86`): replace it with the D4 orientation paragraph and a link to `go doc ./opm/kernel`. Remove the pipeline block. Keep one sentence on concurrency and its link.
- [ ] 3.4 `docs/getting-started.md`: in the `RenderError` code comment (`:220-228`), replace the cause-order list with a pointer to the `RenderError` doc. The `WithRegistry` paragraph (`:39`) keeps its first sentence and links to the `opm/kernel` package doc.
- [ ] 3.5 Verify: `grep -n 'ContractCollisionsError' README.md AGENTS.md docs/getting-started.md` finds no ordered cause list. Each removed AGENTS/README claim is in the godoc (spot-check with `go doc ./opm/kernel`).
- [ ] 3.6 Gates green, then commit `docs: link the render contract instead of restating it`.

## 4. Sources a reader can open

- [ ] 4.1 `AGENTS.md:311` and `:317`: apply the D7 texts.
- [ ] 4.2 `.tasks/api-diff.sh:28-29`: apply the D7 comment text. The `ALLOW` array is not touched.
- [ ] 4.3 Verify: `grep -rnE '\bSD[0-9]+\b|supervisor' AGENTS.md README.md CONSTITUTION.md docs .tasks opm` prints nothing. The remaining hits under `openspec/specs/` are `api-diff-check` and `cascade-wiring`, whose deltas here replace them at archive. `task api:diff:test` is green.
- [ ] 4.4 Gates green, then commit `docs: cite resolvable sources for the api-diff and cascade rules`.

## 5. Absorb main and sweep

- [ ] 5.1 `git fetch origin` and rebase onto `origin/main` after the other round-3 library changes merge. Resolve conflicts keeping their code and this change's comments.
- [ ] 5.2 Re-run every verify step of sections 1 to 4 over the rebased tree. Apply the same rules to what round 3 landed: new `opm/k8s/*` package docs (ADR pointers, holder-bounded wording), new restatements of the env rule, and new cue/load overlay sites.
- [ ] 5.3 Cross-check: `task api:diff` reports no change of this branch. Run `.tasks/consumer-build.sh` against fresh clones of cli and opm-operator `main`, which build and vet green. `go vet ./...` is clean, and `task docs:bundle` builds.
- [ ] 5.4 Gates green. If 5.2 changed anything, commit `docs: apply the one-home rule to the round-3 packages`; otherwise record "no sweep changes" in this box.

## 6. Verify and archive

- [ ] 6.1 The repo's verify skill (`openspec verify`) reports no CRITICAL finding.
- [ ] 6.2 `openspec archive one-home-per-contract-comment-pass -y`, then `openspec validate --all --strict` passes. The archive rides the PR.
- [ ] 6.3 Commit `chore(openspec): archive one-home-per-contract-comment-pass`.
