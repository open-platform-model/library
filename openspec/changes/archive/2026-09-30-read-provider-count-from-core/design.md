## Context

See proposal.md for the two bugs and why the render's count becomes the only count. This change is B of a four-change set (orchestration.md): core (A) recounts `#Platform.#contracts` and publishes the count once as `#contracts.providedBy` in core `2.0.0-alpha.12`; this change makes the library read it; the operator (C) and the cli (D) then word their refusals from it.

Current state in this repo:

- `opm/internal/renderstage/render.cue.tmpl` computes `guard._providerSuppliers` (every enabled entry's `#transformers`, every required demand with `fulfilment: "provider"`, keyed by registry key), derives `guard.overSubscribed` rows from it, feeds it to `match.#providers` (the skip switch's unprovided test) and gates on `len(guard.overSubscribed) == 0`.
- `opm/platform/contracts.go` decodes eight fields of `#contracts` by path, each with a `since` release, and refuses a missing field with an untyped `fmt.Errorf`.
- `schema.DefaultSchemaModule` and every served fixture pin core `v2.0.0-alpha.10`. Core `alpha.11` was never adopted here; this change jumps to `alpha.12`.
- `Kernel.Render` never reads `#contracts`; `opm/schema/paths.go` says so ("never a kernel verb").

The counting rule itself is core's (change A, `design.md` there) and is not restated beyond what the glue must agree with: every transformer of every enabled registry entry, required demands only, fulfilment read off the transformer's own requirement, keyed by the registry key (path plus major), whether or not an enabled entry defines the contract. `overSubscribed` is every `providedBy` key with two or more entries; `routable` is `len(overSubscribed) == 0`.

## Goals / Non-Goals

**Goals:**

- One provider count, computed in core, read by the render glue, by `Contracts()` and (through `Contracts()`) by every frontend.
- A parity test that fails when the inventory and the render disagree, written before the fix and kept as the tripwire.
- A platform whose core predates the count is refused, with a typed error a frontend can word as a re-pin hint.

**Non-Goals:**

- The counting rule (core's, change A). If the parity test disagrees after the re-pin, the fix is in core, not in the glue.
- A fallback count for older cores (Decision 3).
- The pre-existing conflict when two majors of a contract's DEFINING catalog are enabled together. Measured during review on core's prototype and on today's core: the conflict in `defined` makes the whole platform value bottom, and `routable`, `providedBy` and `#composedTransformers` all read it. The earlier reasoning that the gate is unaffected because `routable` does not depend on `defined` was wrong. The conclusion stands for a different reason: nothing on that shape renders before or after this change, and the glue already read the conflicting `definedBy`. A separate core issue.
- Operator and cli wording (changes C and D).

## Decisions

### 1. The glue reads `providedBy`, `overSubscribed` and `routable`; its own count is deleted

The measured diff (scratch library under `design/library`, core replaced by the prototype through `LocalReplacements`):

```cue
match: #Match & {
	#transformers:   _transformers
	#components:     _components
	#definedBy:      platform.#contracts.definedBy
	#providers:      platform.#contracts.providedBy
	#skipUnprovided: _skipUnprovided
}

guard: {
	overSubscribed: list.Sort([
		for _k, _ps in platform.#contracts.providedBy
		if list.Contains(platform.#contracts.overSubscribed, _k) {{
			key:      _k
			catalogs: _ps
		}},
	], {x: _, y: _, less: x.key < y.key})
}

gate: match.resolved & platform.#contracts.routable & true
```

`guard._providerSuppliers` and its comprehension are deleted. `diagnostics.overSubscribed` keeps the row shape `{key, catalogs}`, so `oerrors.OverSubscribedContract`, `OverSubscribedContractsError` and the refusal text are byte-identical (measured on `platform_oversubscribed`: rows `[{gateway, [cat2@v0, cat@v0]}]`, `Contracts()` reading `routable=false`, identical text; the healthy `platform` renders). `#Match.#providers: [string]: _` already accepts a map of lists: the skip switch tests presence only (`#providers[_f] == _|_`), so no matcher change is needed.

**Alternatives.** Keeping the glue's count and asserting it equals core's inside the build: two formulas survive, which is the defect. Deriving rows from `overSubscribed` alone (`for _k in overSubscribed {key: _k, catalogs: providedBy[_k]}`): fail-open on older cores (Decision 2).

### 2. The rows iterate `providedBy` unconditionally (fail-closed belt)

Measured on the `alpha.10` fixtures:

- **Naive switch** (only `#providers` plus rows derived from `overSubscribed`) is fail-open: `TestRenderSkip_ProvidedButUnmatchedStillRefuses`, `_SkippedRowsBesideRefusal` and `_OmittedComponentDropsWarningsKeepsRefusals` returned nil errors, and most renders passed on the old, buggy `routable`.
- **Unconditional iteration** fails every build-reaching `TestRender*` (35 of 41). The six that passed all refuse before the build or expect an error (`RefusesSourcelessInputs`, `UnstatedPostureIsBuildError`, `Skew_RefusePolicyStopsBeforeEvaluation`, `CancelledContextRefused`, `LocalReplacementRefusedUnlessEnabled`, `VersionlessDependencyWithoutReplacementRefused`).

So the row comprehension MUST range over `platform.#contracts.providedBy` directly, and review MUST reject any `== _|_` presence fallback in the glue. With Decision 3 in front, this belt is unreachable through `Render`; it guards a staged module evaluated outside the kernel and any future caller of `renderstage.Build`.

### 3. Older cores are refused with a typed error, never degraded

**Decision.** `Kernel.Render` checks `in.Platform.Package.LookupPath(schema.Contracts).LookupPath(providedBy).Exists()` after the input checks and before `os.MkdirTemp`. When absent it returns `fmt.Errorf("render refused before staging: %w", &oerrors.PlatformCoreTooOldError{Platform: name, Field: "providedBy", Since: "2.0.0-alpha.12"})`. Not a `*RenderError`, no staging directory created. `Contracts()` returns the same typed error for the same condition.

**Why sound.** The render build evaluates exactly the platform module's own core pin: promotion adopts the platform's deps whole (explorer 2, `TestMeasure_CoreSelection`), and the acquired `Package` was built from the same `cue.mod` (confirmed: under a core replacement, `Package` carried `providedBy` and the render read it). So "the Package lacks it" and "the build would lack it" are one fact.

**Why no fallback.** With a fallback guard, render and inventory still disagree on an old core, because the old inventory is the buggy formula, and the second formula lives on in the glue. With the floor, no render runs on a platform whose inventory disagrees, so parity holds by construction.

**Contract change: `Render` now reads the platform's `Package`.** Until this change the cross-artifact verbs read only `Metadata` and `Source` from their inputs, never `Package` (`opm/kernel/doc.go` "The cross-artifact verbs read only Metadata and Source", AGENTS.md § Kernel API surface, ADR-007 Context). The floor breaks that sentence: `Render` performs one read-only `LookupPath(...).Exists()` on `in.Platform.Package`, with no unification and no fill, so an artifact acquired by one Kernel still renders on another. The concern is concurrency: the package doc's own example shares one acquired platform across goroutines rendering on one Kernel, and a `cue.Value` is not documented as safe for concurrent reads. Measured during review (a scratch copy at this change's base, `go test -race`, 10 rounds of 32 goroutines each running `LookupPath(schema.Contracts)` plus `Exists`/`Bool` on `routable`, `overSubscribed` and a missing `providedBy`, on `platform_oversubscribed`, `platform_providers` and `platform_two`): zero races, since acquisition leaves the Package evaluated. That is evidence, not a guarantee across CUE bumps, so section 3 rewrites the three sentences to say what `Render` reads and pins it with a race test that shares one acquired platform across concurrent `Render` calls on one Kernel. The alternative, decoding the presence bit at acquisition into the `Platform` value so `Render` reads only `Metadata`, would keep the sentence true but miss platforms built with `platform.NewPlatformFromValue`. It is not taken.

**Surface.**

```go
package errors // opm/errors

// PlatformCoreTooOldError reports a platform module pinning a core release
// older than the first one deriving a #Platform.#contracts field the kernel
// reads. Returned (wrapped) by Kernel.Render before staging and by
// Platform.Contracts(). The fix is re-pinning opmodel.dev/core in the
// platform module.
type PlatformCoreTooOldError struct {
	Platform string // the platform's metadata.name
	Field    string // the missing field, e.g. "providedBy"
	Since    string // the first core release deriving it, e.g. "2.0.0-alpha.12"
	Require  string // the oldest core the kernel accepts (callers fill schema.ProvidedBySince)
}

func (e *PlatformCoreTooOldError) Error() string
```

The message names the platform (`<unnamed>` when it has no name), the field and its `Since`, and says to re-pin `opmodel.dev/core` in the platform module to `Require`, so one re-pin is enough whichever field was missing. `opm/errors` stays a leaf package: the call sites fill `Require`, it does not import `opm/schema`. (`Require` was added in the review round.) Pointer receiver, like the other gate causes in `opm/errors`.

`Contracts()` migrates every since-guard (including the missing-`#contracts` case, `Field: "#contracts"`, `Since: "2.0.0-alpha.9"`) to the typed error, so a frontend matches one type for every "re-pin core" refusal. The existing message substrings (field name, release) are kept, so the existing `contracts_test` assertions hold.

The field name and its since-release are needed by two packages (`opm/platform` for `Contracts()`, `opm/kernel` for `Render`). They are declared once in `opm/schema/paths.go` beside `schema.Contracts`: `schema.ContractsProvidedBy` (the path) and `schema.ProvidedBySince = "2.0.0-alpha.12"`. The `paths.go` comment "never a kernel verb" is rewritten: `Render` reads the presence of `providedBy`, nothing more. If A reports a different release tag, every `2.0.0-alpha.12` in this change uses A's tag.

**Alternatives.** `Render` calling `Contracts()`: decodes the whole inventory on every render and fails on unrelated inventory errors (the defining-catalog conflict above). A public `(*Platform).CheckCoreFloor()`: a method whose only callers are the kernel and `Contracts()` itself.

### 4. `ContractInventory.ProvidedBy`

```go
// ProvidedBy maps every provider-fulfilled contract FQN some enabled
// transformer requires (defined by an enabled catalog or not) to the
// sorted registry keys (path@major) of the enabled entries whose
// transformers require it. OverSubscribed is exactly its keys with two
// or more entries; a key a defined provider contract lacks is Unfulfilled.
ProvidedBy map[string][]string `json:"providedBy"`
```

Decoded by path like the other fields, row `{"providedBy", &inv.ProvidedBy, "2.0.0-alpha.12"}`. The `ContractInventory`, `OverSubscribed` and `Unfulfilled` doc comments say "registry entries (path plus major)" instead of "catalogs", and name `ProvidedBy` as the field a frontend prints when `DefinedBy` and `RequiredBy` lack the key (Bug 2).

### 5. The parity test lives in `opm/kernel`, over the served fixture tree

The kernel is the only place both counts meet: `Contracts()` reads the inventory, `Render` reads the glue's rows. The test (`opm/kernel/render_inventory_parity_test.go`, name the worker's choice) serves `testdata/render/registry` through `newRenderKernel` and resolves core from the shared cache, like every render test.

Table-driven over every served render platform, enumerated by globbing `testdata/render/platform*` (eight after this change), so a platform fixture added later joins the tripwire without anyone remembering to list it; the test fails if the glob finds a directory the explicit-pin table does not know as either a control or a pinned bad fixture. For each, acquire, `p.Contracts()`, then `Render` the `instance` fixture. The rows come from `res.Diagnostics.OverSubscribed` on success or `*RenderError.Diagnostics.OverSubscribed` on refusal; the render may also refuse for unrelated unresolved demands, and the rows stay decodable. Assertions:

- `sorted(inv.OverSubscribed) == [row.Key for row in rows]`;
- `inv.Routable == (len(rows) == 0)`;
- from section 2 on, `row.Catalogs == inv.ProvidedBy[row.Key]` for every row;
- the bad fixtures' verdicts pinned explicitly: `platform_two_majors` gives rows `[{gateway, [cat@v0, cat@v1]}]` and `Routable` false; `platform_definer_disabled` gives rows `[{gateway, [cat2@v0, cat@v1]}]` and `Routable` false.

`Discriminated` is never asserted, and neither is the overall render outcome: disabling `cat@v0` changes comparability and resolution, which is not what the test is about.

| Platform | Carries | Role | Rows | Routable |
| --- | --- | --- | --- | --- |
| `platform` | cat 0.1.0 | control, healthy | none | true |
| `platform_next` | cat 0.2.0 | control, the skew fixture (not previously listed; it is served, so it belongs in the tripwire) | none | true |
| `platform_two` | cat 0.1.0 + cat2 0.1.0 | control, catalog-fulfilled plurality | none | true |
| `platform_oversubscribed` | cat 0.1.0 + cat2 0.2.0 | control, both counts already agree | gateway: cat2@v0, cat@v0 | false |
| `platform_providers` | cat 0.1.0 + providers 0.1.0 | control, unfulfilled but routable | none | true |
| `platform_disabled` | cat 0.1.0 (disabled) + cat2 0.1.0 | control | none | true |
| `platform_two_majors` (new) | cat 0.1.0 + cat 1.0.0 | Bug 1 | gateway: cat@v0, cat@v1 | false |
| `platform_definer_disabled` (new) | cat 0.1.0 (disabled) + cat2 0.2.0 + cat 1.0.0 | Bug 2 | gateway: cat2@v0, cat@v1 | false |

**New fixtures.**

1. Registry dir `testdata/render/registry/testing.opmodel.dev_library-render_cat_v1.0.0/`: module `testing.opmodel.dev/library-render/cat@v1` at `v1.0.0`, depending on `cat@v0` `v0.1.0` and core. One transformer whose `requiredResources` is exactly `cat@v0`'s provider-fulfilled gateway contract (`cat.#GatewayResource`), with nothing catalog-fulfilled, so comparability cannot confound. Empty contract maps, like `cat2`. Its transformer FQN carries version `1.0.0`, distinct from `cat@v0`'s `0.1.0` transformer. Core stamps both catalogs' transformers with the same major-free `metadata.modulePath` (`.../library-render/cat/transformers`), which is exactly what made the old inventory count one provider.
2. `testdata/render/platform_two_majors/`: `cat@v0` and `cat@v1` enabled, the two imports aliased apart.
3. `testdata/render/platform_definer_disabled/`: the `cat@v0` entry present with `enable: false`, `cat2@v0` (`v0.2.0`) and `cat@v1` enabled.

Both platforms follow the D5 shape and the header convention of the existing platforms, and join the table in `testdata/render/scenarios/README.md`.

**Red first.** Section 1 writes the fixtures and the test pinned to `alpha.10` and runs it. Expected: the two new platforms fail (inventory `Routable` true, render rows present), the six controls pass. The worker records the output in its report. It then re-pins to `alpha.12` in the same section, where the test goes green with the glue still unchanged, because core now agrees with the guard. No red commit lands.

### 6. Section plan

1. **Parity test and core pin** (`fix(deps)`): fixtures, red run, `DefaultSchemaModule` and every fixture and literal to `alpha.12`, green. One commit, `fix(deps)`, because `DefaultSchemaModule` is shipped and moves generated platforms' pins downstream. The precedent (`read-comparable-predicates`) split the pin and the fixture re-pin into `fix(deps)` plus `test(fixtures)`; here the red-to-green proof needs both in one section, and a `test` commit would not release.
2. **`ProvidedBy` and the typed error** (`feat(platform)!`): `Contracts()` decodes the field, all since-guards return `PlatformCoreTooOldError`, the parity test gains the `ProvidedBy` assertion.
3. **Single source** (`feat(kernel)!`): the glue switch, the `Render` core floor, the older-core render test, the shared-platform race test, doc comments (including the rewritten "never Package" sentence).
4. **Docs** (`docs(kernel)`): AGENTS.md, the diagnostics page, the scenarios README if not already done. It MAY fold into section 3.

Every section ends green under `task check`.

## Research & Decisions

### Where the count is computed

**Context**: the inventory and the glue counted differently; the user asked for one count, preferably computed once in core.
**Explored**: core prototype `design/core/src/platform.cue` with a verbatim replica of the glue's guard (`zz_parity_pins.cue`) on seven platforms; key sets equal on all seven, and on today's core the same check diverges on exactly the three bad platforms (two-majors, definer-disabled, definer-absent).
**Decision**: core computes `providedBy`; the library reads it (Decision 1).
**Rationale**: a count in two places drifts; the platform inventory is what the frontends gate on, so it must be the one the render agrees with.

### Fail-open versus fail-closed glue

**Context**: a glue that reads a field older cores lack must not read "absent" as "zero providers".
**Explored**: naive switch versus unconditional iteration on the `alpha.10` fixtures (Decision 2 numbers).
**Decision**: unconditional iteration plus a pre-staging core floor.
**Rationale**: measured fail-open on the naive form; the floor turns the build failure into a typed, actionable error.

### Typed error versus string error

**Context**: frontends (C, D) must word "re-pin core" for both `opm platform check` and render-bearing commands.
**Explored**: the existing untyped since-guards in `contracts.go`.
**Decision**: `oerrors.PlatformCoreTooOldError`, used by every since-guard.
**Rationale**: `errors.As` on one type instead of string matching across two consumers.

## Risks / Trade-offs

- [Every render and platform check on a platform module pinned to core `alpha.10` or `alpha.11` is refused] → generated platforms move with `DefaultSchemaVersion`; hand-written ones re-pin, guided by the typed error. Named in the release notes (the `feat(kernel)!` commit body).
- [The core tag is not `alpha.12`] → section 1.1 confirms A's reported tag; every since, pin and literal in this change uses it.
- [PLAUSIBLE, not measured: `platform_definer_disabled` disables `cat@v0` while `cat2` and `cat@v1` import it; the `instance` render may refuse with a non-`RenderError` build error instead of decodable rows] → the parity test fails loudly rather than skipping; if it happens, the worker records it and renders a narrower instance (a scenario package demanding only the gateway resource) for the two new platforms, reported under deviations.
- [`task cue:deps:update` moves `opmodel.dev/catalogs/opm` as well as core in the discovered modules] → as in the precedent, the parity harness literals move with it and the harness (`OPM_FLOW_TEST_FORCE=1`) proves the rendered bytes; a divergence is recorded here before continuing.
- [The glue's belt (Decision 2) is unreachable through `Render` once the floor is in place, so a regression that re-adds a presence fallback would pass the suite] → `TestRenderSkip_*` and `TestRender_OverSubscribedProviderRefused` stay green on `alpha.12` fixtures, the parity test pins agreement, and review rejects any `== _|_` fallback on `providedBy`.
- [cli offline tests using `--platform hack/platform` pinned to `alpha.10` break on this release] → change D merges only after the workspace `task deps:update` re-pins them (orchestration.md).

## Migration Plan

Pre-GA: no migration fragment. The release notes (commit bodies) say: platform modules must pin `opmodel.dev/core` at `v2.0.0-alpha.12` or later; `Contracts()` and `Render` refuse older ones with `PlatformCoreTooOldError`; Bug 1 and Bug 2 platforms now read `Routable: false`. Rollback is reverting the release; the core pin is the only coupling.
