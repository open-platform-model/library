## Context

Core `v2.0.0-beta.4` adds ADR-013, decision j3: the key of each entry in a component's `#resources`, `#traits` and `#blueprints`, and in a `#Catalog`'s member maps, must equal the member's `metadata.fqn`. A mismatch is a vet error. The library pins core at `v2.0.0-beta.3` in one Go constant (`schema.DefaultSchemaModule`) and in the `cue.mod` of every served fixture. The `schema-dispatch` spec requires all of these to name the same release, and requires a default move to edit only the constant and those pins.

This change is the release cascade's core move, made by hand because the receiver runs in dry-run.

## Goals / Non-Goals

**Goals:**

- Every library core pin names `v2.0.0-beta.4`, moved by the same task the cascade runs.
- Every library CUE fixture and served registry module is shown to pass the stricter core.

**Non-Goals:**

- Any change to the render glue, decoders or public Go API. j3 adds a constraint and changes no path the kernel reads.
- Moving the opm catalog pin, which stays at `v4.6.0`.
- Publishing `modules/opm_platform`.
- A standing library check of j3 (a served short-key scenario and test). It would make the PR differ from the cascade's diff and add a library requirement that no owner decision asks for.

## Research & Decisions

### Which tool makes the move

**Context**: Three hand edits have moved the core pin so far. `#173` moved it to beta.2 with no OpenSpec change. `#195` moved it to beta.3 with `task -x deps:cascade`, as section 1 of `read-catalog-provider-set-from-core`.
**Explored**: The cascade was dry-run against a scratch clone of `origin/main` `ca7c56b`, with the resolver at `../.github/.github/scripts/cascade/cascade-resolve.sh`. It exited 0 with one warning, the expected `need-human-review` note for the loader. It edited `opm/schema/loader.go`, 35 `cue.mod/module.cue` files, `docs/getting-started.md` and `AGENTS.md`, and nothing else. `deps:cascade:title` printed `fix(deps): bump core to v2.0.0-beta.4`. The opm catalog did not move, because `v4.6.0` is the newest published build.
**Decision**: Section 1 runs `task -x deps:cascade` in the worktree. The fallback is `DEFAULT_CORE=v2.0.0-beta.4 task cue:deps:update` plus hand edits of the constant, the text pins and the two docs examples, used only if the resolver cannot run.
**Rationale**: The cascade will make every later move. Using it here keeps this PR's diff identical to the PR the bot would open, and its title is the one release-please will show.

### Whether any library fixture is mis-keyed

**Context**: The fixtures could have short keys that core's own re-vet (catalog_opm, the fleets) did not reach. Most of `testdata/render` is served in-process by `registrytest` and is not discovered by the CUE tasks.
**Explored**: In a scratch copy of the worktree, every `v: "v2.0.0-beta.3"` pin under `testdata/` and `modules/` and the constant were set to beta.4. Then:
- `go test ./...` passed in every package, including the CUE that Go tests embed. The kernel tests resolved `core@v2.0.0-beta.4` into the workspace cache, and none skipped.
- `task cue:vet` passed over `modules/opm_platform`, `testdata/modules/web_app`, `testdata/parity` and `testdata/parity/opm_platform`.
- A direct `cue vet` of the `testdata/render` modules fails at import resolution. Their catalogs exist only on the in-process registry, as on `main`. These modules are covered by the Go suite.
- A grep of the attachment and member maps in `testdata/` shows only `(X.metadata.fqn): X` keys.
**Decision**: No fixture edit. Section 1 repeats the full suite and `task cue:vet` on the real worktree and records the result.
**Rationale**: The served fixtures were already written FQN-keyed, matching what the catalog fixtures adopted under j3.

### Spec wording for the current default

**Context**: The "DefaultSchemaModule constant" requirement still says "At this change that is `opmodel.dev/core@v2.0.0-beta.1` ... It carries the schema of `2.0.0-alpha.13` unchanged". The beta.2 and beta.3 moves did not update it. The same requirement says a default move edits only the constant and the fixture pins, and the cascade never edits a spec.
**Decision**: MODIFIED, rewording only the first paragraph:
- The constant alone records the current release. The spec names none.
- No floor is added. The schema cache's default load evaluates core only and refuses nothing, and acquisition resolves through the instance's own `cue.mod`, so no library test could pin a beta.4 floor without a new fixture.
- The `2.0.0-alpha.13` attribution rule is kept ("SHALL NOT attribute those reports to a beta release").

The other two paragraphs and all seven scenarios are copied verbatim, because OpenSpec 1.12 refuses a MODIFIED that drops a scenario.
**Rationale**: "The constant is the record" stays true across every later cascade move. A literal would go stale on the next core release, as `beta.1` did.

## Risks / Trade-offs

- [Another library change merges a fixture with a short key before this PR] → The full suite at beta.4 catches it. Section 1's re-run happens after `git merge origin/main` in the PR stage.
- [A consumer's own module is mis-keyed] → Under beta.3 a short `#resources` or `#traits` key was already refused at render, as an unresolved demand. Under beta.4 it is refused earlier, at build or acquire, as a `metadata.fqn` conflict, so the failure moves and its message changes; a short `#blueprints` key is refused for the first time. The re-vet of every published catalog and both module fleets found none. The frontends' own suites were run against this tree through the consumer-build `go.work` on fresh clones: `go test ./internal/... ./pkg/...` passed in cli `main` `bd4d1a7`, and `go test ./internal/reconcile/... ./pkg/...` passed in opm-operator `main` `53ccaab`.
- [The cascade goes live before this merges and opens the same move] → Stop before opening this PR. The cascade PR carries the move, and this change is closed in its favour.
