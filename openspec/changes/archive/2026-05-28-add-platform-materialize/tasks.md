## 1. De-risking spikes (settle before building)

- [x] 1.1 Spike `FillPath` onto the optional closed `#composedTransformers` / `#matchers` slots of a real `#Platform` value (built from `core@v0.3.0`); confirm the filled value answers `LookupPath(schema.ComposedTransformers)` and `MatchersResources/Traits`. Record outcome in design.md D2/Q1. If it fails, switch the design to accessor methods on `MaterializedPlatform` before proceeding.
- [x] 1.2 Spike `modregistrytest` fixture layout: stand up `New(fsys, prefix)`, push one inline `#Catalog` module at two versions, confirm `Host()` is reachable and `ModuleVersions` lists both. Capture the exact fixture layout (txtar vs `fstest.MapFS`, version-dir naming) as a reusable test helper.

## 2. errors

- [x] 2.1 Add `MaterializeError{Kind, Subscription, Version, Cause}` to `opm/errors` with `Error()` + `Unwrap()`; `Kind` constants `"catalog"` / `"core-schema"`.
- [x] 2.2 Unit-test `MaterializeError` formatting and `errors.Unwrap` reachability.

## 3. materialize package — resolution flow

- [x] 3.1 Define `MaterializedPlatform{Source, Package, Resolved}` in `opm/materialize` with godoc documenting the sealed post-realization contract.
- [x] 3.2 Implement version filtering: parse `filter.range` via `Masterminds/semver/v3`, apply `range` → `allow` → `deny` order (D10), normalize the `v`-prefix vs bare-SemVer boundary (D4). Pure function over a version list — table-test in isolation.
- [x] 3.3 Implement enumeration: `modconfig.NewResolver{Env}` → `modregistry.NewClientWithResolver` → `ModuleVersions(ctx, path)`; thread the registry mapping through `Config.Env` (no `os.Setenv`).
- [x] 3.4 Implement per-survivor pull via `load.Instances(["<path>@<exact-ver>"], &load.Config{Env})` + `BuildInstance`; wrap failures as `MaterializeError{Kind:"catalog"}`.
- [x] 3.5 Implement transformer indexing: read each build's `#Catalog.#transformers`, build the composed map + `#matchers.{resources,traits}` reverse index; collapse identical FQN bodies, surface divergent bodies as `MaterializeError`.
- [x] 3.6 Implement `Materialize(ctx, owner, registry, *platform.Platform) (*MaterializedPlatform, error)`: walk `#registry`, skip `enable:false`, run 3.2–3.5 per subscription, `FillPath` (or accessor per 1.1) the index onto a copy of `Source.Package`, record resolved versions. Inputs not mutated.

## 4. materialize/cache subpackage

- [x] 4.1 Define `MaterializeCache` interface (`Get`/`Put`) + reference LRU implementation in `opm/materialize/cache`.
- [x] 4.2 Implement key derivation hashing the canonicalized `#registry` subtree (resolve Q2: registry-only vs whole-spec key).
- [x] 4.3 Unit-test cache round-trip and key stability across semantically-identical registries.

## 5. kernel wiring

- [x] 5.1 Add `registry` field to `Kernel` + `WithRegistry(string)` option (no auto-default; inherits process `CUE_REGISTRY`).
- [x] 5.2 Add `(*Kernel).Materialize(ctx, *platform.Platform) (*MaterializedPlatform, error)` delegating to `opm/materialize` with the kernel's registry + context.
- [x] 5.3 Confirm `Match` / `Plan` / `Compile` signatures are unchanged in this slice.

## 6. Tests (modregistrytest-backed)

- [x] 6.1 Materialize happy path: subscribe to a fixture catalog, assert `#composedTransformers` + `#matchers` indexing and resolved version.
- [x] 6.2 Range/allow/deny: push `0.1.0`/`0.1.1`/`0.2.0`, assert survivor selection per filter combinations.
- [x] 6.3 Divergent same-FQN builds → `MaterializeError`; unresolvable path → `MaterializeError{Kind:"catalog"}`.
- [x] 6.4 `enable:false` skipped; idempotency + input-not-mutated assertions.

## 7. Docs + validation gates

- [x] 7.1 Add a `Materialize` lifetime/registry note to `library/CLAUDE.md` and a `MIGRATIONS.md` entry introducing the new surface (additive).
- [x] 7.2 `task fmt`.
- [x] 7.3 `task vet`.
- [x] 7.4 `task lint`.
- [x] 7.5 `task test`.
