# Tasks: library-render-build

Ordered so tasks 1 to 4 have no dependency on the D5 core prerelease; tasks 5 onward gate on `core-registry-import` being published.

## opm/internal/renderstage (core-independent)

- [ ] 1. Modfile intake: parse an artifact `Source`'s `cue.mod/module.cue` (`cuelang.org/go/mod/modfile`), exposing the dependency entries with default-major markers intact. Unit tests over synthetic module files.
- [ ] 2. Promotion (D13): platform list whole ∪ instance-only paths, platform wins shared; emit the render `module.cue` (fresh never-published module path, max language version) and `local-module.cue` replacements for the two input roots. Unit tests: shared-path win, instance-only survival, default-major preservation.
- [ ] 3. Refusal invariant: re-parse the written `module.cue` and refuse when any OPM-namespace path from either input list is absent. Unit test: a doctored promotion result refuses naming the path.
- [ ] 4. Skew (D7/D18): per-OPM-path compare of the two committed lists; `SkewPolicy` (warn default / refuse) verdict plus unconditional resolved-versions rows. Unit tests: newer-warns, newer-refuses, older-is-data-only.

## Glue and fixtures (gate: D5 core prerelease published)

- [ ] 5. `testdata/render/` fixtures: registrytest-served catalog module; platform module with `#CatalogEntry` imports; instance module demanding its contracts; all `cue.mod`s pinned to the exact D5 core prerelease.
- [ ] 6. Embedded `render.cue` glue template: generated imports (instance path+pkg, platform path), `#runtimeName` as a formatted literal, experiment 05's `#Match` body (buckets from `#composedTransformers`, plain-`&` unify rung, predicate rung, verdicts-as-data, `resolved & true` gate), interim `#context` derivation (sheds when core-context-projection lands), `rendered` assembly with provenance keys.

## opm/kernel

- [ ] 7. Staging + build: `Render` materializes the temp dir (including overlay-mode instance trees), creates a fresh `cue.Context`, builds the render module once via `cue/load` with the caller registry env, and cleans up on return. Input validation: Source-carrying inputs, non-empty runtime name.
- [ ] 8. Decode: `diagnostics` → `RenderDiagnostics` (existing `oerrors` types + skew rows), gate enforcement (unresolved/unmatched refuse with decoded diagnostics attached); `rendered` → `[]*core.Compiled` with provenance; kernel-side per-pair concreteness check with pair-naming errors.
- [ ] 9. Behavior tests against the fixtures: happy render (pair set matches `Kernel.Compile`'s on an equivalent old-path fixture), missing-FQN refusal with alternatives, disqualified-candidate data, effectively-optional trait warning, unstated-posture build error, incomplete-pair refusal, shares-nothing repeat render byte-identity.

## Verification

- [ ] 10. Full old-path suite passes untouched (no existing test modified); `task check` green.
