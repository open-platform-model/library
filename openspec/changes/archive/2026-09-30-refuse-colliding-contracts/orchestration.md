# Orchestration: refuse-colliding-contracts

This file is the brief for the worker agent that implements this change and for the supervisor that stitches the change set together. The same text sits in every change of the set; only the title and the "This change" section differ.

## Change set

Five OpenSpec changes, planned together on 2026-09-30, in four repos. Four of them do one job:

- core stops failing to evaluate a platform that enables two majors of one catalog sharing contract keys. It folds only the keys with exactly one enabled definer and reports the rest as `collisions` and `collidingEntries`, with `routable` false.
- The library turns a collision into a typed render refusal and decodes it in `Contracts()`.
- The operator and cli name the collision.

The fifth (E) is independent: it makes the operator's build-compatibility verdict deterministic on a platform carrying two majors of one catalog. The fix is an interim safety net and does not add side-by-side majors; enhancement 0026 D9 later makes two majors legitimate through per-resolution builds (0026 OQ17 and 05-risks recommend both fixes independent of 0026). No change claims a 0026 decision, so none carries an `enhancement.yaml`.

| ID | Repo | Change | Branch | Wave | Starts when | Merges when |
| --- | --- | --- | --- | --- | --- | --- |
| A | core | `fold-colliding-contract-keys` | `feat/fold-colliding-contract-keys` | 1 | now | first; release cut after |
| E | opm-operator | `index-build-compat-by-major` | `fix/index-build-compat-by-major` | 1 | now | any time, independent of A to D |
| B | library | `refuse-colliding-contracts` | `feat/refuse-colliding-contracts` | 2 | A released | after A is released |
| C | opm-operator | `name-contract-collisions` | `fix/name-contract-collisions` | 3 | B's branch pushed | after B is released |
| D | cli | `name-contract-collisions` | `fix/name-contract-collisions` | 3 | B's branch pushed | after B is released |

What each hands on:

- **A** publishes, in core `2.0.0-alpha.13` (actual tag reported under `surface`):
  - `#ContractInventory.collisions: [...#ContractFQNType]`: sorted keys with more than one enabled definer.
  - `#ContractInventory.collidingEntries: [#ContractFQNType]: [...#ModulePathType]`: the sorted registry keys defining each.
  - `routable: len(overSubscribed) == 0 && len(collisions) == 0`.
- **B** reads them in the render build and in `Contracts()`, raises a typed refusal, moves `DefaultSchemaModule` to A's tag, and publishes the interface below in a library release.
- **C** and **D** consume B's release.
- **E** consumes nothing and hands on nothing.

## Interface B -> C, D

The contract C and D consume. B may refine names only by reporting the change under `deviations`; C and D code against what B reports under `surface`.

```go
package platform // github.com/open-platform-model/library/opm/platform

type ContractInventory struct {
	// ...existing fields unchanged...

	// Routable is true exactly when OverSubscribed and Collisions are both empty.
	Routable bool `json:"routable"`

	// Collisions lists, ascending, every contract key more than one enabled
	// registry entry's catalog lists. Such a key is in none of DefinedBy,
	// RequiredBy, Unfulfilled or Comparable, so Fulfilled and Discriminated
	// can read true while Collisions is non-empty; Routable is false.
	Collisions []string `json:"collisions"`

	// CollidingEntries maps each Collisions key to the sorted registry keys
	// (path@major) of the enabled entries listing it.
	CollidingEntries map[string][]string `json:"collidingEntries"`
}

// Contracts() decodes an absent collisions/collidingEntries as empty: every
// core before A's tag fails to evaluate a colliding platform at acquire.
```

```go
package errors // github.com/open-platform-model/library/opm/errors

type ContractCollision struct {
	Key      string   `json:"key"`
	Catalogs []string `json:"catalogs"` // sorted registry keys, path@major
}

// Raised (joined, first) by the render gate, platform-wide, under SkipUnprovided too.
type ContractCollisionsError struct{ Contracts []ContractCollision }

// Raised only when #contracts.routable is false and no over-subscription or collision row explains it.
type NotRoutableError struct{}
```

`kernel.RenderDiagnostics.Collisions []oerrors.ContractCollision` (sorted by key) and `RenderDiagnostics.Routable bool`. `schema.DefaultSchemaModule` = A's tag. `schema.ProvidedBySince` (the floor) is unchanged. `UnresolvedDemand.Colliding []string` is diagnostic only.

The rule behind it (A computes it, B reads it, C and D print it):

1. A definer is an ENABLED registry entry whose catalog lists the key in `#resources`, `#traits` or `#blueprints`. A disabled entry never counts.
2. A key with exactly one definer folds into `defined` and `definedBy` as before. A key with more is a collision: it is in `collisions` and `collidingEntries` and in none of `defined`, `definedBy`, `requiredBy`, `unfulfilled` or `comparable`. `providedBy` and `overSubscribed` are unaffected and can co-occur with a collision.
3. `routable` is false while any collision exists. `fulfilled` and `discriminated` can still read true (the stated limitation), so no consumer reads either as safe while `collisions` is non-empty.
4. The render refuses a colliding platform with `ContractCollisionsError`, whatever the instance and whatever `SkipUnprovided` says. Its rows are `{key, catalogs: collidingEntries[key]}` for every key in `collisions`, guarded on presence. Absence is provably empty: an older core fails to evaluate such a platform. A `routable` false that no row explains raises `NotRoutableError`.
5. The operator words it as reason `ContractCollisions`, ahead of `OverSubscribedContracts` and `ComparablePredicates`. The cli prints a colliding-contracts section and counts collisions in the routable verdict and the exit message.

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

1. **Wave 1.** Launch the workers for A and E, each with its change's `orchestration.md` as its brief.
2. **Check each report.** Compare `deviations` and `surface` against `design.md` and the interface above. Send a worker back with a precise ask rather than fixing its branch yourself.
3. **Finalize each change.** For each, in the worktree: `openspec archive <change> --yes`, commit the archive (`chore(openspec): archive <change>`), push, open the PR per the repo's `AGENTS.md`. Merge order: A, then B, then C and D; E whenever it is green. After A merges, merge the core release PR that release-please opens and note the released version; after B merges, do the same for the library release PR.
4. **Wave 2.** Once A is released, launch B's worker with the core version. Do NOT run the workspace root `task deps:update` yet. Run it once B is released, and land its per-repo output (`fix(deps)` / `test(fixtures)` per the workspace commit skill). This departs from the previous set on purpose: a kernel older than B renders a colliding platform pinned to A's core (duplicate objects, measured), so no workspace platform moves to A's core before a refusing kernel exists.
5. **Wave 3.** Once B's branch is pushed, launch C's and D's workers. Until B is released they develop against B's pushed head as a Go pseudo-version. Once B is released, tell each worker the version; it rebases on `origin/main`, pins the release, reruns its gates and pushes. Then finalize C and D as in step 3.
6. **Follow-ups.** Run the out-of-repo follow-ups each change's `orchestration.md` lists, after the merge they wait on.

Every PR follows its repo's `AGENTS.md`: a body of at most 250 words, no bare `@name`, only the plain co-author trailer. Merging, releasing and pushing to `main` need the user's go-ahead.

## This change (B)

**Release class.** `feat(kernel)`, not breaking: no floor is raised, and the only behaviour change is on platforms that could not be acquired before. Four sections, four commits, every one releasing: `fix(deps)` (section 1, the shipped `DefaultSchemaModule`), `feat(platform)` (section 2, `Contracts()` decode and docs), `feat(kernel)` (section 3, glue rows, causes, `gateErrors`), `docs(kernel)` (section 4). The PR title carries `feat(kernel)`. No migration fragment (pre-GA, no floor). No `enhancement.yaml`. The repo has an unrelated active change (`restructure-migration-docs`); leave it alone.

**Waits on:** A's release tag resolving from GHCR. Task 1.1 refuses to start until `opmodel.dev/core@v2.0.0-alpha.13` (or A's reported tag) resolves and carries the two fields. If the tag differs, every constant, fixture pin, spec scenario literal and `CollisionsSince` in this change uses A's tag, and the difference is reported under `deviations`.

**Worktree setup (library).**

- Branch `feat/refuse-colliding-contracts` from `origin/main`, worktree at `library/.claude/worktrees/refuse-colliding-contracts`. The planning commit (`chore(openspec): plan refuse-colliding-contracts`) is already on this branch; rebase it onto fresh `origin/main` before section 1 if main has moved.
- Copy the main checkout's CUE cache into the worktree, never symlink: `mkdir -p .cue-cache && cp -a ../../../.cue-cache/mod .cue-cache/` when it exists. A cold cache fetches core once from GHCR.
- Export the workspace registry mapping in two lines (a one-line `export A=x B="$A"` leaves `OPM_REGISTRY` empty):

  ```bash
  export CUE_REGISTRY='opmodel.dev=ghcr.io/open-platform-model,testing.opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'
  export OPM_REGISTRY="$CUE_REGISTRY"
  ```

- The render tests serve `testdata/render/registry` in-process (`registrytest.NewRegistryFromDir`), so no fixture is published and Registry Policy rule 3 is not triggered. Never start the local registry for this change.

**Hazards.**

- **Red first, no red commit.** Task 3.1 runs the bridge render on the unchanged kernel with A's core and must see success with two `probe-maj0-web` Deployments (built by `0.1.0` and `bridge-1.4.0`). Record it under `gates` as `<test> on unchanged kernel -> fail (expected): renders two probe-maj0-web Deployments`. The section's only commit is green.
- **Absent means empty, present-and-erroring refuses.** Guard only on presence, in the glue and in `Contracts()`. Write the documented exception into the `Contracts()` doc, citing the `definedBy` conflict proof (design.md § 2). Never add a since-guard for the collision fields: it would refuse every `alpha.12` platform.
- **Do not touch the over-subscription belt.** `guard.overSubscribed` keeps iterating `providedBy` unconditionally (fail-closed on a core without it). Only the new collision rows are guarded.
- **Measure the hand-built case (task 3.6).** Determine whether a Platform built without `AcquirePlatformFromDir` over an old-core colliding value reaches `Render`; write the finding into design.md § 2 and report it under `surface` or `supervisor`.
- **Catch-all.** `NotRoutableError` must fire on no served fixture; the parity test asserts that. A fixture that raises it means a routable term the kernel does not decode: stop and report.
- **Limitation pins are deliberate.** The `Fulfilled` true and `Discriminated` true assertions on the collide platforms pin a known blind spot; label them so a future core fix updates them on purpose.
- **Fixtures.** The collide fixtures are ported from enhancement 0026 experiment 07's overlay (enhancements `origin/main` 6b94e1c, `0026/experiments/07-render-shipped-core/overlay/testdata/render`), not copied verbatim: `maj` 1.4.0 is new (experiment 07's 1.3.0 plus the three `maj@v0` keys in its own contract maps), and every pin is A's tag. Never place a fixture under `opmodel.dev/*`.
- **Consumers.** Task 3.7 builds `cli` and `opm-operator` against this tree through a scratch `replace` only; never edit either repo.

**Hands off to C and D.** Under `surface`: the interface above as implemented (`ContractInventory.Collisions`, `CollidingEntries` and the new `Routable` meaning; `errors.ContractCollision`, `ContractCollisionsError` with its exact message; `errors.NotRoutableError` with its message; `UnresolvedDemand.Colliding` and its message case; `RenderDiagnostics.Collisions` and `Routable`), `schema.ContractsCollisions`, `ContractsCollidingEntries` and `CollisionsSince`, `DefaultSchemaModule` at A's tag, B's pushed head (for a Go pseudo-version), then the release version from the supervisor. Also report the collide fixture shapes (C's and D's tests may mirror them).

**Follow-ups outside this repo (supervisor):**

1. After B is released: run the workspace root `task deps:update` (held since A's release, supervisor step 4) and launch or resume C and D on the released version.
2. `opm-suite-installer` and the personal `opm-kind-demo` platforms are outside `task deps:update`; nothing breaks while they stay on `2.0.0-alpha.12` (no floor moves), but they must not move to A's core before their pinned `opm` carries B.
