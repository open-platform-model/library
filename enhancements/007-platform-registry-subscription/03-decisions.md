# Design Decisions — Platform Registry Subscription

## Summary

Decision log for all architectural and design choices made during this enhancement. Each decision is numbered sequentially and recorded as it is made. Decisions are append-only — do not remove or renumber existing entries. If a decision is reversed, add a new decision that supersedes it.

---

## Decisions

### D1: Path-keyed `#registry` replaces Module-valued `#registry`

**Decision:** `#Platform.#registry` is a map keyed by short Id (kebab-case `#NameType`). Each value is a `#Subscription` carrying `path` (the catalog's CUE module path), `enable`, and an optional `filter`. The `#Module` value is no longer embedded in the registry entry.

**Alternatives considered:**

- *Keep `#module: #Module`, add a version-range field alongside it.* Rejected: CUE imports pin one version, so the embedded value would still represent a single build, defeating the multi-version goal.
- *Make `#registry` keyed by FQN, one entry per primitive.* Rejected: explodes the platform definition into hundreds of entries and removes the natural unit of subscription (the catalog).

**Rationale:** The catalog is the natural unit of versioning and policy. A path is stable across builds; one subscription stands for "every version of this catalog the filter selects." Per design goal "one platform can host multiple SemVer builds of the same catalog at once."

**Source:** User direction during explore session 2026-05-23.

---

### D2: Primitive FQNs are SemVer-suffixed, not MAJOR-only

**Decision:** `#FQNType` regex changes from `…@v[0-9]+$` to `…@<SemVer 2.0>$`. `metadata.version` on `#Resource`, `#Trait`, `#Blueprint`, `#ComponentTransformer` changes type from `#MajorVersionType` to `#VersionType`. Two builds of the same primitive at different SemVers are distinct keys in `#composedTransformers`.

**Alternatives considered:**

- *Keep MAJOR FQNs, add a `version` predicate field on transformers.* Rejected: pushes version drift into predicate-match logic, where errors surface as generic match failures rather than structured diagnostics. Authors would have to opt every catalog into the convention.
- *Per-primitive SemVer independent of the catalog's SemVer.* Rejected as part of D5 (1:1 coupling).

**Rationale:** SemVer in the FQN makes the multi-version map natural — same-SemVer rebuilds with identical content collapse via CUE map unification; divergent content fails CUE evaluation. MAJOR-only loses every patch-grade signal that the matcher needs.

**Source:** User direction during explore session 2026-05-23.

---

### D3: `#Module.#defines` is removed; catalogs are plain CUE packages

**Decision:** Delete `#defines.{resources,traits,transformers}` from `core/module.cue`. `#Module` becomes the consumer artifact only (components, `#config`, `debugValues`). Catalog modules are plain CUE packages that export `#Resource` / `#Trait` / `#Blueprint` / `#ComponentTransformer` as top-level definitions. The kernel discovers transformers by scanning top-level package values at materialize time.

**Alternatives considered:**

- *Keep `#defines` and walk it at materialize time.* Rejected: dual-role `#Module` (consumer + publisher) is the source of the "one Module = one version" coupling. Decoupling them removes the constraint at the root.
- *Introduce a new `#Catalog` type holding the primitives.* Rejected: redundant with a plain CUE package; adds a wrapper without buying anything the matcher needs.

**Rationale:** Catalog authoring and application authoring have different shapes; sharing `#Module` forced unused slots on both sides. A plain package is the simplest publication mechanism and matches how every other CUE library is published.

**Source:** User direction 2026-05-23 ("Also remove defines from #Module in @../core/module.cue").

---

### D4: `#knownResources` and `#knownTraits` are removed

**Decision:** Drop the two computed views. Resources and traits surface only as `requiredResources` / `optionalResources` / `requiredTraits` / `optionalTraits` of materialized transformers. Primitives that no transformer references are unreachable on the platform.

**Alternatives considered:**

- *Keep them and derive from materialized transformers' required-or-optional unions.* Rejected: callers don't read these views on the hot path; they were authoring-time discovery aids in 003. Discovery moves to a future `opm catalog` tool.
- *Keep them as opt-in fields populated by kernel only when requested.* Rejected: yet-another-conditional-field with no current consumer.

**Rationale:** Unreachable primitives shouldn't be claimed by the platform. Dropping the views removes a projection that has no run-time consumer and removes one motivation to keep `#defines`.

**Source:** User selection during explore session 2026-05-23 ("Drop — derive from transformers").

---

### D5: Primitive SemVer = catalog SemVer (1:1)

**Decision:** Every primitive in a catalog at version `X.Y.Z` carries `metadata.version: "X.Y.Z"`. Authors do not declare per-primitive versions; the version comes from the catalog's `Catalog.Version` constant (D6). Catalog versions ship as monolithic releases — bump the catalog to bump every primitive together.

**Alternatives considered:**

- *Independent per-primitive SemVer.* Rejected: higher fidelity but materially more authoring burden, and divergent primitive versions inside one catalog make the publish flow ambiguous (which version of `container` does `opm@1.4.0` carry?).
- *Hybrid: catalog SemVer + per-primitive MAJOR.* Rejected: keeps MAJOR FQNs alive, defeats D2's signal.

**Rationale:** Monolithic catalog releases keep the publish flow simple and the FQN space predictable. "What version of `container` does this catalog have?" has one answer: the catalog's own version.

**Source:** User selection during explore session 2026-05-23 ("1:1 — primitive = module version").

---

### D6: Catalog identity lives in a single root-package constant, stamped at publish

**Decision:** Each catalog's root package declares `Catalog: { Version: #VersionType | *"0.0.0-dev", ModulePath: <constant> }`. Every primitive sources `metadata.version: opm.Catalog.Version` and `metadata.modulePath: "\(opm.Catalog.ModulePath)/<sub>"`. The publish task overwrites `Catalog.Version` with the concrete SemVer in a temp build dir before `cue mod publish`. The OCI artifact ships fully concrete; source-tree `Catalog.Version` retains its dev default for local `cue vet`.

**Alternatives considered:**

- *Kernel injects `Catalog.Version` at materialize time via `FillPath`.* Rejected: catalog is not self-consistent standalone (FQN regex fails without a stamped version); makes "what version is this catalog?" a runtime question rather than a property of the artifact.
- *Generate a separate `version.cue` per subpackage.* Rejected: N files to stamp instead of one; per-subpackage no-cross-import wins over by a small margin but the cross-package import pattern is trivial and the single-stamp ergonomics are worth the import line.
- *Author hand-writes `version` on every primitive.* Rejected: certain drift between primitive declarations and the OCI tag.

**Rationale:** A burned-in published artifact is the strongest possible guarantee against drift between OCI tag and CUE content. The single shared constant is the smallest possible authoring footprint that keeps the artifact self-consistent. Cross-package import of `opm.Catalog` is one extra import per primitive file — trivially mechanical.

**Source:** User direction 2026-05-23 ("I want the solution to be burned in… stamp the version at publish time").

---

### D7: Filter is `range` + `allow` + `deny`, resolution order range → allow → deny

**Decision:** `#SubscriptionFilter` carries an optional SemVer `range` (Masterminds/semver constraint syntax), an optional `allow` list of explicit SemVers (force-include), and an optional `deny` list (force-exclude). The resolver applies `range` to produce a base set, adds `allow` entries (even if outside the range), then subtracts `deny` entries (even if inside the range). Empty filter selects all published builds of the path.

**Alternatives considered:**

- *Range only.* Rejected: no escape hatch for emergency exclusions (a known-bad patch) without bumping the range.
- *Allowlist only.* Rejected: forces full enumeration; bad ergonomics for "track minor line."
- *Range + allow + deny but ambiguous order.* Rejected: silent surprises.

**Rationale:** Range is the everyday case; allow/deny are the operational escape hatches. Specifying the resolution order in the spec eliminates surprise.

**Source:** User selection during explore session 2026-05-23 ("Both — range with allow/deny overrides").

---

### D8: Match always unifies consumer primitive with transformer's required entry

**Decision:** After FQN lookup, before predicate evaluation, the matcher unifies `consumer_component.#resources[FQN]` with `transformer.requiredResources[FQN]` (and the analogous traits step). Unification failure marks the pair non-matched with a `UnifyError` diagnostic. No `--strict` mode, no skip in production.

**Alternatives considered:**

- *Only in dev / `--strict` mode.* Rejected: defense-in-depth flag that exists only for debugging is the worst kind of safety belt.
- *Never; FQN identity is sufficient.* Rejected: same-SemVer rebuilds with divergent content shouldn't silently match, and local overrides via CUE imports can produce divergent schemas at the same FQN.

**Rationale:** FQN match is necessary but not sufficient. Unification cost is bounded (typically a few CUE evaluations per matched pair) and catches the failure mode that would otherwise propagate to render time as a confusing error.

**Source:** User selection during explore session 2026-05-23 ("Always — every matched pair").

---

### D9: Missing FQN produces one structured error per occurrence

**Decision:** When the consumer Module declares a primitive FQN that's absent from materialized `#composedTransformers`, the matcher emits one error per (component, FQN) pair carrying the component name, the missing FQN, and any adjacent-version FQNs that are present (as a hint). Match does not fail-fast; it accumulates every miss in one pass.

**Alternatives considered:**

- *Aggregate error / fail-fast on first miss.* Rejected: punishing iteration loop when many components need adjusting.
- *Soft-fail in dev mode, hard-fail in prod.* Rejected: developer ergonomics are better served by clear errors than by silent permissiveness.

**Rationale:** A platform team flipping a filter benefits from seeing the full diff in one shot. Existing MatchPlan diagnostic style accumulates; this fits that pattern.

**Source:** User selection during explore session 2026-05-23 ("Single error per missing FQN at Match").

---

### D10: Module discovery and pull route through `cuelang.org/go/mod`

**Decision:** Catalog pulls use `cuelang.org/go/mod` — the same OCI machinery that CUE's import resolver uses. The kernel exposes a `Registry` field on `*Kernel` (default GHCR), which threads to `CUE_REGISTRY` (or equivalent) for these calls. No custom OCI client.

**Alternatives considered:**

- *Custom Go OCI client.* Rejected: duplicates work CUE already does, decoupling that buys nothing the kernel needs.
- *Filesystem-only workspace catalog (defer remote pull).* Rejected: too narrow; production use needs remote resolution.

**Rationale:** Coherence with CUE's own module system. One way for catalogs to be published and pulled. Caching is the CUE module proxy's job.

**Source:** User direction during explore session 2026-05-23 ("OCI registry via cuelang.org/go/mod… `Registry` field added with a default to ghcr").

---

### D11: Materialize is an explicit `Kernel.Materialize` step

**Decision:** Add `Kernel.Materialize(*Platform) (*MaterializedPlatform, error)`. `Match` takes `*MaterializedPlatform`, not `*Platform`. Callers materialize once and reuse the result across many Module Release matches.

**Alternatives considered:**

- *Implicit materialize inside `Match`.* Rejected: would re-pull/re-load on every call unless the kernel internally memoized — adds hidden state to `Kernel`.
- *Both APIs side-by-side.* Rejected: more surface, ambiguous best practice.

**Rationale:** Two-stage API is the simplest honest expression: pulling and indexing have non-trivial cost and obvious caching boundaries (one platform spec → one materialized platform). Explicit lifetimes leave caching to the caller.

**Source:** User selection during explore session 2026-05-23 ("Explicit kernel.Materialize(Platform) → *MaterializedPlatform").

---

### D12: api binding / apiVersion machinery is out of scope

**Decision:** This enhancement does not touch `opm/api`, `opm/apiversion`, or any binding logic. The user has a separate change removing apiVersion handling from the library; core/ has no apiVersion field currently. Schema changes in this enhancement happen directly on core/* without a coexisting alpha alias.

**Alternatives considered:**

- *Carry a v1alpha2 → v1alpha3 alias for one release.* Rejected: nothing currently consumes the v1alpha2 alias beyond the OPM core itself, and the apiVersion field is being removed anyway.

**Rationale:** Two adjacent reshapes are easier to ship sequentially than bundled. This enhancement assumes the api binding cleanup either ships first or in parallel; neither interferes.

**Source:** User direction 2026-05-23 ("i have recently removed apiVersion from core, so there is nothing to bump… I have a separate change for removing the api binding logic from the library").

---

### D13: Catalog source-tree `Catalog.Version` carries a `0.0.0-dev` default

**Decision:** The checked-in `version.cue` declares `Catalog.Version: #VersionType | *"0.0.0-dev"`. Source-tree `cue vet` evaluates with the default; primitives compute FQNs at `…@0.0.0-dev`. Publish overwrites with the concrete SemVer in a temp build dir.

**Alternatives considered:**

- *Gitignored, generated each time.* Rejected: dev tooling has to stamp before `cue vet` works.
- *Always checked-in with a real version.* Rejected: drift between source and OCI tag becomes the default state.

**Rationale:** Dev experience stays cheap; publish stays unambiguous. The synthetic `0.0.0-dev` version is acceptable for development because nothing in the catalog references this value externally.

**Source:** User selection during explore session 2026-05-23 ("Checked-in with dev fallback default").

---

### D14: Publish stamps in a temp build dir; source tree is never mutated

**Decision:** The publish task `rsync`s the catalog into `.build/catalog/`, overwrites `version.cue` there, runs `cue vet` and `cue mod publish` from the build dir, and exits. The source tree is identical before and after. Failure mid-flow leaves the build dir for inspection; no git revert needed.

**Alternatives considered:**

- *Stamp in place, revert via `git checkout` in a trap.* Rejected: trap-on-exit reliability is OS-dependent; failure modes that leave a stamped file checked in are real.
- *Stamp in place, commit-then-publish-then-revert.* Rejected: each publish creates a release commit just to track stamped state; noisy history.

**Rationale:** Pure-function publish step. No git mutations during publish. Trivial cleanup.

**Source:** User selection during explore session 2026-05-23 ("Stamp in temp build dir; source tree untouched").

---

### D15: Cross-package access uses an exported `Catalog` struct, not `_`-prefixed identifiers

**Decision:** The catalog's root package declares `Catalog: { Version, ModulePath }` with a capitalized (exported) name. Subpackages import the root package by path and read `opm.Catalog.Version` / `opm.Catalog.ModulePath`. Identifiers do not begin with `_` because CUE's `_`-prefix convention makes them package-private and inaccessible from subpackages.

**Alternatives considered:**

- *`_Version` / `_ModulePath` at the root with all primitives in the same package.* Rejected: flat package layout fights CUE's subdirectory-as-package convention and grows unwieldy for large catalogs.
- *Per-subpackage constants.* Rejected: N stamps instead of one (D6).

**Rationale:** Single source of truth + one extra import per primitive file is the right trade. The empirical CUE spike confirmed the pattern works end-to-end.

**Source:** CUE spike 2026-05-23 (see Experiments section in this enhancement once the spike artifacts land); cross-package access fails with `_`-prefixed names due to CUE visibility rules.

---

### D16: Blueprints follow the same SemVer / stamping trail; no extra logic

**Decision:** `#Blueprint` adopts the same FQN regex change, the same per-primitive `version` source-from-Catalog, and the same publish-time stamping. The kernel does not materialize blueprints into platform-side views — blueprints are consumed by Modules at the component level (`#Component.#blueprints`), unchanged by this enhancement.

**Alternatives considered:**

- *Surface blueprints in `#composedTransformers`-style platform views.* Rejected: blueprints are consumer-side composition aids; the platform doesn't render them. Adding a view multiplies materializer surface for zero gain.

**Rationale:** Blueprint FQNs share the same shape as Resource / Trait / Transformer FQNs and benefit from the same SemVer signal. Skipping platform-side projection keeps the materializer focused on what the matcher actually consults.

**Source:** User direction 2026-05-23 ("Blueprints should also be changed to semver and follow the same trail, but they are just still blueprints. No extra logic to handle them is required.").

---

## Open Questions

Track unresolved questions surfaced during design. `task enhancements:check` requires this block (with or without entries) starting at `status: accepted`, in either this file or `README.md`. Each entry SHOULD carry a `Status:` line once the enhancement reaches `implemented`.

- **OQ1: Top-level discovery scan rules.** Status: open. The kernel discovers transformers by walking *top-level* values in the catalog package and unifying with `#ComponentTransformer`. Should it also recurse into nested fields (e.g., a `Transformers: { … }` grouping struct)? Default to "top-level only" for simplicity; revisit if catalog authors complain.
- **OQ2: Materialize cache keying and invalidation.** Status: open. `Materialize` produces a value derived from (path × filter × OCI tag set at fetch time). Caching it across many `Match` calls is sound; invalidating it when the registry advances is up to the caller for now. A future `opm` CLI may add a "refresh" hook.
- **OQ3: Same-SemVer rebuilds with content divergence.** Status: resolved-by-D8. Two builds of `opm@1.0.4` with different content would collide in `#composedTransformers` and fail at materialize time via CUE map unification. The always-unify match step (D8) covers the case where divergence arrives via a local override path.
- **OQ4: Multi-fulfiller behavior in the new model.** Status: open. 003's D13 (revised) allowed multiple transformers to require the same primitive FQN, disambiguated by predicate evaluation. This enhancement preserves that semantics — `#matchers.{resources,traits}[FQN]` is a list — but the SemVer-FQN expansion reduces collision likelihood. Confirm at implementation time whether 003's predicate-evaluation logic still applies unchanged or needs simplification.
- **OQ5: Filter parser library choice.** Status: open. Masterminds/semver is the natural Go dependency for the range syntax; confirm there's no friction with `cuelang.org/go/mod`'s own version parsing before locking it in.
- **OQ6: Cross-catalog primitive references.** Status: open. A transformer in catalog A may reference a resource published by catalog B — its `requiredResources` map carries the B-side value. With multiple catalogs subscribed, this works as long as both are pulled. Document this as an explicit supported pattern or defer to a follow-up.
