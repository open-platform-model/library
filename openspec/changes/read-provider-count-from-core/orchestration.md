# Orchestration: read-provider-count-from-core

This file is the brief for the worker agent that implements this change and for the supervisor that stitches the change set together.

## Change set

Four OpenSpec changes, planned together on 2026-09-29, each in its own repo. Together they make the render build's provider count the only count: core computes it once as `#contracts.providedBy`, the library render reads it instead of its own guard, and the operator and cli name the providing catalogs from it. No enhancement entry backs them (they correct the delivery of 0015:D2/D18, and 0015 is being closed as delivered), so none carries an `enhancement.yaml`.

| ID | Repo | Change | Branch | Wave | Starts when | Merges when |
| --- | --- | --- | --- | --- | --- | --- |
| A | core | `count-providers-per-registry-entry` | `feat/count-providers-per-registry-entry` | 1 | now | first; release cut after |
| B | library | `read-provider-count-from-core` | `feat/read-provider-count-from-core` | 2 | A released | after A is released |
| C | opm-operator | `single-source-provider-count` | `fix/single-source-provider-count` | 3 | B's branch pushed | after B is released |
| D | cli | `single-source-provider-count` | `fix/single-source-provider-count` | 3 | B's branch pushed | after B is released and the cli `deps:update` commit is on main |

What each hands on:

- **A** publishes `#ContractInventory.providedBy: [#ContractFQNType]: [...#ModulePathType]` (sorted registry keys), recounted `overSubscribed`/`unfulfilled`, in core `2.0.0-alpha.12` (actual tag reported under `surface`).
- **B** reads it in the render build (single source), decodes it, floors core, and publishes the interface below in a library release.
- **C** and **D** consume B's release.

## Interface B → C, D

The contract C and D consume. B may refine names only by reporting the change under `deviations`; C and D code against what B reports under `surface`.

```go
package platform // github.com/open-platform-model/library/opm/platform

type ContractInventory struct {
	// ...existing fields unchanged...

	// ProvidedBy maps every provider-fulfilled contract FQN some enabled
	// transformer requires (defined by an enabled catalog or not) to the
	// sorted registry keys (path@major) of the enabled entries whose
	// transformers require it. OverSubscribed is exactly its keys with two
	// or more entries; a key a defined provider contract lacks is Unfulfilled.
	ProvidedBy map[string][]string `json:"providedBy"`
}

// Contracts() refuses a platform whose #contracts lacks providedBy, naming
// the field and core 2.0.0-alpha.12.
```

```go
package errors // github.com/open-platform-model/library/opm/errors

// Returned (wrapped) by Kernel.Render before staging, and by Contracts(),
// when the platform module pins a core release predating a field the kernel reads.
type PlatformCoreTooOldError struct{ Platform, Field, Since string }
```

`schema.DefaultSchemaModule` = `opmodel.dev/core@v2.0.0-alpha.12`. `OverSubscribedContract{Key, Catalogs}` and the render refusal text are unchanged.

## Worker protocol

One worker agent per change, in its own git worktree of the change's repo.

1. Create the worktree from fresh `origin/main`, on the branch named in the change set: from the repo root, `git fetch origin`, then `git worktree add .claude/worktrees/<change> -b <branch> origin/main`, then work inside that directory. Read the repo's `AGENTS.md` and `openspec/config.yaml` first; they bind. The repo-specific setup in this file comes next.
2. Run the repo's apply workflow on the change: `openspec instructions apply --change <change> --json`, then `tasks.md` section by section. The commit task that closes each section is the only commit you make.
3. After the last section is green, push the branch: `git push -u origin <branch>`. Do not archive the change, open a PR, merge, tag or release; the supervisor does.
4. On a blocker, stop and report it. Do not widen scope, and do not edit another repo; a design question goes in the report.
5. End with exactly this block as your final message:

```text
change:      <ID> <repo>/<change>
branch:      <branch> @ <head sha> (pushed: yes|no)
sections:    <n>/<total> committed
commits:     <sha> <subject>   (one line per commit)
gates:       <command> -> pass|fail   (one line each; name every failing or skipped test)
surface:     <what the next change consumes: exported symbols, flags, SPEC sections, output strings>
deviations:  <every departure from design.md or this file, with the reason; "none">
supervisor:  <what only the supervisor can do: a release, a pin bump, merge order, a decision; "none">
follow-ups:  <work found outside this change's repo; "none">
```

## Supervisor protocol

1. **Wave 1.** Launch the worker for A, with this change's `orchestration.md` as its brief.
2. **Check each report.** Compare `deviations` and `surface` against `design.md` and the interface above. Send a worker back with a precise ask rather than fixing its branch yourself.
3. **Finalize each change.** For each, in the worktree: `openspec archive <change> --yes`, commit the archive (`chore(openspec): archive <change>`), push, open the PR per the repo's `AGENTS.md`. Merge order: A, then B, then C and D. After A merges, merge the core release PR that release-please opens and note the released version; after B merges, do the same for the library release PR.
4. **Wave 2.** Once A is released, launch B's worker with the core version. Then run the workspace root `task deps:update` and land its per-repo output (`fix(deps)` / `test(fixtures)` per the workspace commit skill); cli's must be on main before D merges.
5. **Wave 3.** Once B's branch is pushed, launch C's and D's workers. Until B is released they develop against B's pushed head as a Go pseudo-version. Once B is released, tell each worker the version; it rebases on `origin/main`, pins the release, reruns its gates and pushes. Then finalize C and D as in step 3.
6. **Follow-ups.** Run the out-of-repo follow-ups each change's `orchestration.md` lists, after the merge they wait on.

Every PR follows its repo's `AGENTS.md`: a body of at most 250 words, no bare `@name`, only the plain co-author trailer. Merging, releasing and pushing to `main` need the user's go-ahead.

## This change (B)

**Release class.** `feat(kernel)!` overall. The library now refuses (`Render` and `Contracts()`) any platform pinning core older than `2.0.0-alpha.12`, and `ContractInventory` gains `ProvidedBy`. Pre-GA, so no migration fragment. The section 1 commit is `fix(deps)`, because `DefaultSchemaModule` is shipped; section 2 is `feat(platform)!`, section 3 `feat(kernel)!`, section 4 `docs(kernel)`. Every commit releases (none is `chore` or `test`). The PR title carries `feat(kernel)!`.

**Waits on:** A's release tag. Section 1.1 refuses to start until `opmodel.dev/core@v2.0.0-alpha.12` (or A's reported tag) resolves from GHCR. If the tag differs, every since, pin and literal in this change uses A's tag, and the difference is reported under `deviations`.

**Worktree setup (library).**

- Branch `feat/read-provider-count-from-core` from `origin/main`, worktree at `library/.claude/worktrees/read-provider-count-from-core`. The planning commit (`chore(openspec): plan read-provider-count-from-core`) is already on this branch; rebase it onto fresh `origin/main` before section 1 if main has moved.
- Copy the main checkout's CUE cache into the worktree, never symlink: `mkdir -p .cue-cache && cp -a ../../../.cue-cache/mod .cue-cache/` when it exists. A cold cache fetches core once from GHCR.
- Export the workspace registry mapping in two lines (a one-line `export A=x B="$A"` leaves `OPM_REGISTRY` empty):

  ```bash
  export CUE_REGISTRY='opmodel.dev=ghcr.io/open-platform-model,testing.opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'
  export OPM_REGISTRY="$CUE_REGISTRY"
  ```

- The render tests serve `testdata/render/registry` in-process (`registrytest.NewRegistryFromDir`), so no fixture is published and Registry Policy rule 3 is not triggered. Never start the local registry for this change.

**Hazards.**

- **Red first, no red commit.** Section 1 runs the parity test on `alpha.10` and must see exactly the two new platforms fail. Record that output in the report under `gates` (as `<ParityTest> on alpha.10 -> fail (expected): platform_two_majors, platform_definer_disabled`). The section's only commit is after the re-pin, green.
- **Fail-closed glue.** The over-subscription rows iterate `platform.#contracts.providedBy` unconditionally. Deriving rows from `overSubscribed` alone, or guarding `providedBy` with `== _|_`, is fail-open on older cores (measured; design.md § 2). Do not "simplify" it.
- **The floor runs before staging.** `Kernel.Render` refuses with the typed error before `os.MkdirTemp`: no staging directory, not a `*RenderError`.
- **`task cue:deps:update` may move `opmodel.dev/catalogs/opm`** in the discovered modules as well as core. Move the parity harness literals with it and run the harness under `OPM_FLOW_TEST_FORCE=1`.
- **Definer-disabled fixture (plausible, not measured).** The `instance` render on `platform_definer_disabled` may refuse for unrelated unresolved demands; the rows stay decodable. If it instead fails with a non-`RenderError` build error, use the fallback in design.md § Risks and report it.
- **Consumers.** Section 3.6 builds `cli` and `opm-operator` against this tree through a scratch `replace` only; never edit either repo. cli's offline tests with `--platform hack/platform` pinned to `alpha.10` will be refused by this release; that is D's merge gate, not this change's.

**Hands off to C and D.** Under `surface`: the library commit (pushed head, then the release version from the supervisor), `ContractInventory.ProvidedBy`, the final name of `PlatformCoreTooOldError` and its fields, `schema.ContractsProvidedBy` and `schema.ProvidedBySince`, and `DefaultSchemaModule` at `alpha.12`. The render rows and message are unchanged. The render refuses older-core platforms before staging.

**Follow-ups outside this repo:** none owned by this change. The supervisor's workspace `task deps:update` (step 4) re-pins `cli/hack/platform`, `cli/examples` and the other workspace platforms to the new core; the personal `opm-kind-demo` platform re-pin is D's listed follow-up.
