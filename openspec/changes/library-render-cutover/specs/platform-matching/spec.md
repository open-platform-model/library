## REMOVED Requirements

### Requirement: Match Phase Consumes Platform

**Reason**: the Go matcher and `Kernel.Match` are deleted; matching runs inside the render build over the platform's derived `#composedTransformers` (0019 D10, `single-build-render`).

**Migration**: `Render`; read `RenderDiagnostics.Pairs`.

### Requirement: Demand Walking from Components

**Reason**: carried by the render glue's comprehensions over the imported instance's components; the behavior is specified under `single-build-render`.

**Migration**: none.

### Requirement: Lookup via Platform.#matchers

**Reason**: core removed `#Platform.#matchers` (0019 D17); the glue builds its own buckets from `#composedTransformers`.

**Migration**: none.

### Requirement: Execute Resolves Transformers by FQN

**Reason**: execution is unification inside the build; there is no Go executor.

**Migration**: none.

### Requirement: Provider Package Retired

**Reason**: historical migration record of a package that no longer exists; superseded by the deletion of `opm/compile` itself.

**Migration**: none.

### Requirement: render.Module Runtime Helper Updated

**Reason**: `compile.Module` (formerly `render.Module`) is deleted with `opm/compile`.

**Migration**: none.

### Requirement: Test Fixture Migration

**Reason**: historical record of a fixture migration to the materialize era; fixtures now carry D5-shaped platform modules.

**Migration**: none.

### Requirement: Always-Unify Before Pairing

**Reason**: carried by the glue as plain unification with no provenance exclusion (0019 D10); specified under `single-build-render`.

**Migration**: none.

### Requirement: Structured Unify-Error Diagnostic

**Reason**: carried by `RenderDiagnostics.Unify` (typed, naming the transformer and conflicting FQN); the verbatim CUE cause is not recoverable from inside the build (D10, accepted).

**Migration**: read `RenderDiagnostics.Unify`.

### Requirement: Label Predicate

**Reason**: restated against the build under `single-build-render` ("The label predicate covers every admitted label type"); unification replaces the string-only Go comparison.

**Migration**: none.

### Requirement: Alternatives Ordering

**Reason**: folded into `single-build-render`'s "Unresolved demands are diagnosed with alternatives" (alternatives sorted, deterministic).

**Migration**: none.

### Requirement: Unresolved Demand Failure

**Reason**: restated against the build under `single-build-render`.

**Migration**: match on `*oerrors.UnresolvedDemandsError` from `Render` (reachable through `*kernel.RenderError`).

### Requirement: Trait Posture

**Reason**: restated against the build under `single-build-render` ("Trait posture governs unhandled traits"), with the measured unstated-posture boundary (build error naming the trait's `optional` field).

**Migration**: none.
