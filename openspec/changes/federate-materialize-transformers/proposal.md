## Why

`Materialize` builds the composed-transformer map and `#matchers` reverse index natively in the owner `*cue.Context` (where they render correctly), then **also** `FillPath`s them onto the closed `c.#Platform` value — producing a second `#composedTransformers` that *looks* readable but corrupts output-local hidden fields in transformer `#transform`s (the ADR-003 closedness bug; `docs/design/transformer-output-hidden-field-scope-bug.md` §12). The landed mitigation exposes a separate open `MaterializedPlatform.Composed` field with a standing WARNING that the executor MUST read transforms from it and never from `Package`. That is a live tripwire: a future read of `#transform` off `Package` silently ships broken output and `cue vet` cannot catch it.

The sibling spike (`rewrite-materialize-single-build`) tried to remove the seam by composing the platform and catalogs in **one CUE build**. That is structurally impossible for OPM's required model: a platform may subscribe to **multiple same-major catalog versions at once** (Module-A authored against `catalog@0.5.0`, Module-B against `catalog@0.5.1`, both live on one platform), and CUE Minimal Version Selection admits exactly one version per `path@major` per build. Single-build is therefore the wrong mechanism — it cannot hold the versions OPM must compose. This change keeps the goal (delete the footgun) and reaches it the other way: **federate** the native surfaces instead of collapsing them into the closed platform.

## What Changes

- Build `#composedTransformers` and `#matchers` natively in the owner `*cue.Context` (unchanged) and expose them as **first-class fields** on `MaterializedPlatform` — `Transformers` (FQN → `#ComponentTransformer`, transforms render concrete) and `Matchers` (`{resources, traits}` reverse index). Multi-version is preserved: distinct version-bearing FQNs are distinct keys.
- **Stop `FillPath`-ing the composed map / matchers onto the closed platform.** The corrupt twin is never created. **BREAKING**: remove `MaterializedPlatform.Composed` (replaced by `Transformers`) and remove `MaterializedPlatform.Package` (the closed spec remains reachable as `Source.Package`).
- Update the compile consumers to read the native fields: `match.go` reads `Matchers`/`Transformers` (was `mp.Package.LookupPath(...)`); `execute.go`/`module.go` read `Transformers` (was `Composed`); delete the WARNING and the closed-fill plumbing. Update `cmd/flow-inspect`.
- Preserve all materialize behavior unrelated to the seam: subscription resolution, SemVer range/allow/deny filtering, multi-version selection, transformer indexing/divergence conflicts, `Resolved` version-per-path, `MaterializeError` shape, non-mutation, idempotency, the opt-in cache, and concurrent read-only safety.
- Confirm matching stays **exact version-bearing FQN** (demand-side: the module's `cue.mod/module.cue` pins the catalog version, embedded into `#Component` resource/trait FQNs) — a component demanding `…/container@0.5.0` matches only `…/deployment-transformer@0.5.0`. Verify this also clears the §10.5 zero-pairs symptom, which stems from the matcher reading the corrupt closed `Package`, not from real ambiguity.
- Record **C2 (per-version build isolation)** as a documented future design (each selected version kept as its own build instance, executor routes per version) — out of scope here; C1 (one merged native map) is sufficient and proven.

## Capabilities

### Modified Capabilities

- `platform-materialization`: `MaterializedPlatform` exposes the composed transformers and reverse index as native first-class fields (`Transformers`, `Matchers`) produced in the owner context; it no longer `FillPath`s them onto the closed platform and no longer exposes `Composed` or `Package`. Transforms render concrete when read directly off `Transformers`; multi-version-per-major composition is preserved. All other materialize behavior (filtering, selection, diagnostics, idempotency, caching, concurrency) is unchanged.

## Impact

- **Affected packages**: `opm/materialize` (drop the closed-fill, reshape `MaterializedPlatform`), `opm/compile` (`match.go`, `execute.go`, `module.go` read native fields), `cmd/flow-inspect` (read native fields). Tests across materialize + compile + flow.
- **SemVer**: removing `MaterializedPlatform.Composed` and `MaterializedPlatform.Package` is a **breaking change to a public `opm/` struct → MAJOR**. Verified blast radius: only in-repo readers of `mp.Package` are `compile/match.go` and `cmd/flow-inspect` (updated in lockstep); `cli/` and `opm-operator/` treat `*MaterializedPlatform` as an opaque handle (constructed via `Kernel.Materialize`, passed to `Kernel.Compile`, held in the operator store) and do not read its fields — no external break. Recorded in `MIGRATIONS.md`.
- **Behavioral contract preserved**: identical materialize output, matching semantics, and concurrency guarantees; only the *surface* the executor/matcher read from changes (native fields vs. the closed `Package`).
- **Supersedes**: `rewrite-materialize-single-build` — its single-build premise is rejected (incompatible with required multi-version composition) and that change has been removed. The single-build approach, its Phase-1 spike result (PARTIAL), and the CUE-MVS reasoning that rules it out are documented in this change's `design.md` (Context + Decision D1).
- **Out of scope**: any change to subscription/filter/selection semantics; the matcher algorithm itself (only its read source changes); the v0.17 concurrency contract (preserved, not modified); C2 per-version isolation (future).
