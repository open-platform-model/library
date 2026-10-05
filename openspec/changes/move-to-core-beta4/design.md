## Context

Core `v2.0.0-beta.4` adds owner decision j3: the key of each entry in a component's `#resources`, `#traits` and `#blueprints`, and in a `#Catalog`'s member maps, must equal the member's `metadata.fqn`. A mismatch is a vet error. The library pins core at `v2.0.0-beta.3` in one Go constant (`schema.DefaultSchemaModule`) and in the `cue.mod` of every served fixture. The `schema-dispatch` spec requires all of these to name the same release, and requires a default move to edit only the constant and those pins.

This change is the release cascade's core move, made by hand because the receiver runs in dry-run. It adds one fixture and one test that make the j3 rule a standing check of the library's served fixtures.

## Goals / Non-Goals

**Goals:**

- Every library core pin names `v2.0.0-beta.4`, moved by the same task the cascade runs.
- Every library CUE fixture and served registry module is shown to pass the stricter core.
- A test fails if the served fixtures resolve a core that accepts a short attachment key.

**Non-Goals:**

- Any change to the render glue, decoders or public Go API. j3 adds a constraint and changes no path the kernel reads.
- Moving the opm catalog pin, which stays at `v4.6.0`.
- Publishing `modules/opm_platform`.

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

### How to keep j3 checked in the library

**Context**: Core's must-fail pins prove the rule in core. In the library, nothing fails if a served fixture's pin drops back below beta.4, or if the kernel stops surfacing the error at acquisition.
**Explored**: A probe scenario `short_key` (a `#ModuleInstance` like `bad_traits`, attaching `cat.#ContainerResource` under `container:`) was acquired with `AcquireInstanceFromDir`.
- At beta.4 acquisition refused it with `#module.#components.web.#resources.container.metadata.fqn: conflicting values "testing.opmodel.dev/library-render/cat/resources/container@v1" and "container"`.
- With `testdata/render/scenarios/cue.mod` set back to beta.3, the same package was accepted.
**Decision**: Add the `short_key` scenario and `TestRender_ShortAttachmentKeyRefuses` in `opm/kernel/render_test.go`, beside `TestRender_ComponentWithConflictingTraitsRefuses`, in the same shape:
- `AcquireInstanceFromDir` errors, and the message contains `#resources.container` and `metadata.fqn`.
- `Render` of `scenarioLiteralInstance(t, "short_key")` on the `platform` fixture errors, returns a nil result, and is not a `*kernel.RenderError`.

The test spells no core release, so a later default move edits no test file, as the "A default move edits no test file" scenario requires.

If implementation shows that the struct-literal render takes another path (for example, it reports a diagnostics row), the test asserts what the kernel does and the PR body names the difference. The acquisition half is the j3 check and is not optional.
**Rationale**: It is one small fixture and one test. It turns "re-vetted against beta.4" into a check that runs on every PR. It uses the scenario mechanism that `bad_traits` already uses, and needs no new helper.

### Spec wording for the current default

**Context**: The "DefaultSchemaModule constant" requirement still says "At this change that is `opmodel.dev/core@v2.0.0-beta.1` ... It carries the schema of `2.0.0-alpha.13` unchanged". The beta.2 and beta.3 moves did not update it. The same requirement says a default move edits only the constant and the fixture pins, and the cascade never edits a spec.
**Decision**: MODIFIED, rewording only the first paragraph:
- The constant alone records the current release. The spec names none.
- The default is no older than `2.0.0-beta.4`, the first core that binds attachment map keys to `metadata.fqn` (owner decision j3).
- The `2.0.0-alpha.13` attribution rule is kept ("SHALL NOT attribute those reports to a beta release").

The other two paragraphs and all seven scenarios are copied verbatim, because OpenSpec 1.12 refuses a MODIFIED that drops a scenario. The j3 check is an ADDED requirement with one scenario.
**Rationale**: A floor plus "the constant is the record" stays true across every later cascade move. A literal would go stale on the next core release, as `beta.1` did.

## Risks / Trade-offs

- [Another library change merges a fixture with a short key before this PR] → `short_key` does not catch it; the full suite at beta.4 does. Section 1's re-run happens after `git merge origin/main` in the PR stage.
- [A consumer's own module is mis-keyed] → It now fails at render against a generated platform. The re-vet of every published catalog and both module fleets found none. The consumer-build run in section 1 checks only compilation, so the cli and opm-operator CUE fixtures meet beta.4 when each frontend adopts the next library release. Their test suites run then.
- [The cascade goes live before this merges and opens the same move] → Stop before opening this PR. The cascade PR carries the move, and this change is closed in its favour.
