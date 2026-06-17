# Design — rewrite-materialize-single-build

Governing principle: **ADR-003** (construct artifacts by single-build CUE evaluation, not cross-build `FillPath` of closed values). This change applies ADR-003 to the materialize path — the third and largest instance of the seam.

> **This change is spike-first.** Materialize is inherently a gather-from-N-registry-pulls operation; whether it can be expressed as one CUE build is the central unknown. Phase 1 is a spike that must succeed (or surface a concrete fallback) before the rewrite tasks are committed. The design below states the target and the decisions already made; it deliberately leaves the composition mechanism's final shape to the spike.

## Research & Decisions

### The `Composed` field is a worked-around seam, not a fix
**Context**: Need to know whether materialize's current shape is essential or a workaround.
**Explored**: `opm/materialize/materialize.go` fills `#composedTransformers` onto the closed platform via `FillPath`. `opm/materialize/types.go` documents (lines ~40–55) that this corrupts lazy resolution of transformers' output-local hidden fields, and exposes `Composed` (the open pre-fill map) so `opm/compile/execute.go` can read `#transform` from it instead of from `Package`. `docs/design/transformer-output-hidden-field-scope-bug.md` §12 traces it to a CUE Go-API closedness bug.
**Decision**: Treat `Composed` + the "read transforms from `Composed`, not `Package`" rule as the same ADR-003 failure mode seen in the render path — a cross-build `FillPath` of (effectively) closed transformer values into a closed platform. The durable fix is to compose in a single build so `Package` is corruption-free, then delete the hatch.
**Rationale**: The hatch is a standing tripwire (`cue vet` cannot catch a future `Package` read; correctness depends on a comment). Removing the seam removes the tripwire.

### Target mechanism: single-build composition
**Context**: We need the composed transformer map and the platform to share one set of definition identities.
**Explored**: The render change establishes the overlay-virtual-package pattern (`load.Config.Overlay`, proven in `opm/helper/loader/registry`). Materialize already uses one `*cue.Context` throughout (per the kernel materialize contract) — but same-context is **not** sufficient; the seam is same-context-different-*build* identity. The fix is same-*build*.
**Decision (target, pending spike confirmation)**: Synthesize a virtual package that imports the platform source and each selected catalog `path@version`, and let CUE evaluate `#composedTransformers` / `#matchers` as part of that single build — no `FillPath` of a separately-built map onto a closed `#Platform`. Reuse subscription resolution + filtering to pick the versions; only the *composition* step changes.
**Rationale**: One build → one transformer-definition identity → `Package` carries concrete transformer output → no `Composed`, no executor special-case. Matches `cue eval` semantics, consistent with the render change.

### Subscription / filter / cache logic is reused unchanged
**Context**: Limit blast radius; materialize has substantial non-seam logic.
**Decision**: Keep version enumeration, SemVer range/allow/deny filtering, stable-version default selection, `MaterializeError` (`Kind` discriminator, path, version, cause), the opt-in `MaterializeCache` interface + LRU, and the non-mutation/idempotency guarantees exactly as they are. Only the index-and-attach step is rewritten.
**Rationale**: Those are orthogonal to the seam and already specified/tested; touching them widens risk for no benefit.

### Concurrency contract is preserved, not redesigned
**Context**: `MaterializedPlatform` is documented safe for concurrent read-only consumption across per-goroutine Kernels under v0.17.
**Decision**: The single-build `Package` must retain that property — built once, read-only thereafter, no mutation by concurrent compiles. The rewrite must not reintroduce per-render mutation of the shared value.
**Rationale**: The Platform-CR "materialize-once, reuse-many" model depends on it; it is a hard invariant for this change, asserted by the existing concurrency scenario.

### SemVer: removing a public field is a break
**Context**: `MaterializedPlatform.Composed` is exported `opm/` surface.
**Decision**: Prefer outright removal (MAJOR) once the executor reads `Package` safely. If a softer landing is wanted, retain `Composed` for one release as a documented alias of `Package` (deprecated), then remove. Record in `MIGRATIONS.md` either way.
**Rationale**: The whole point is to delete the tripwire; a permanent alias keeps the footgun. A one-release deprecation is the most compatibility this should grant.

## The spike (Phase 1 — must resolve before Phase 2+)

**Question**: Can `#composedTransformers` / `#matchers` be produced inside a single build that imports the platform + selected catalogs, such that the resulting closed `Package` renders transformers with output-local hidden fields concretely (the case `Composed` exists for) — without `FillPath`-ing a separately-built map?

**Method**: Build a minimal virtual package importing one catalog known to exercise the hidden-field pattern (the `opmodel.dev/catalogs/opm` transformer from the §12 repro) + a small platform, evaluate once, and read `#transform` output off the closed result. Compare against the current `Composed`-routed output.

**Outcomes**:
- **Green** → proceed with the single-build composition rewrite (Phase 2+), delete `Composed`.
- **Partial** (single-build works but version-keyed bucket collisions appear — cf. the materializer side-finding in §10.5 of the bug doc, where an unfiltered multi-version subscription collapses matching) → fold the relevant bucket-keying fix into this change or split it out explicitly.
- **Red** (single-build composition is not expressible with current CUE) → do not force it; document the finding, keep `Composed` but convert the WARNING into an enforced guard (e.g. a test that fails if any `Package` `#transform` read appears in `opm/compile`), and re-scope this change to "contain the seam" rather than "remove it."

## Phase impact (contingent on a green/partial spike)

- **materialize**: replace the index-then-`FillPath` step with single-build composition; drop `Composed` from `MaterializedPlatform` (or deprecate). Keep `Source`, `Package`, resolved-version map, error type.
- **compile**: `execute.go` reads `#transform` from `Package`; `module.go` stops threading `r.platform.Composed`; delete the WARNING and the `Composed` plumbing.
- **schema**: confirm `schema.ComposedTransformers` / `schema.Matchers*` paths resolve on the single-build `Package` (expected unchanged).
- **Public surface (`opm/`)**: `MaterializedPlatform.Composed` removed/deprecated (MAJOR or one-release deprecation).

## Open questions

1. Does single-build composition change the cost profile (memory/wall-time) of materializing wide SemVer ranges, given all selected catalogs now load into one build? (Measure in the spike.)
2. Does the v0.17 concurrent-read-only guarantee hold for the single-build `Package` exactly as for the current one? (Assert with the existing race scenario.)
3. Is the unfiltered-multi-version bucket-collision (bug doc §10.5) in scope here or a separate change? (Spike "partial" outcome decides.)
