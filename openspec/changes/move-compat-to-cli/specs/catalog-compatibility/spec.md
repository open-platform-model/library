## REMOVED Requirements

### Requirement: Compatibility Comparison

**Reason**: The comparison walk has no consumer inside the library once the render glue sorts alternatives itself; its only callers are `opm catalog publish` and `opm catalog registry check --compat`, both the cli binary. Shipping it in the library made every comparator change a library release plus a cli re-pin. The code moves verbatim to the cli.

**Migration**: `library/opm/compat.Check` and `CheckAtLevel` become `cli/internal/compat.Check` and `CheckAtLevel` with identical signatures, semantics and tests; the cli's `catalog-compatibility` capability restates this requirement against that package.

### Requirement: Level Classification

**Reason**: Same as above: the ladder (`ParseLevel`, `Level.Enforced`, `CompareAPIVersions`) is read only by the cli's publish gate once the render glue carries its own apiVersion ordering.

**Migration**: `cli/internal/compat.ParseLevel`, `Level` and `CompareAPIVersions`, unchanged; the core-grammar parity pin (`TestAPIVersionPatternCoreParity`) moves with them.

### Requirement: Predecessor Selection

**Reason**: `HighestStable` is the template-resolution float selector (0011 D23), called only by the cli's scaffold. It never belonged beside the comparator.

**Migration**: an unexported selector in `cli/internal/scaffold` with the same rule (highest stable, unparseable entries skipped, highest overall when no stable exists) and the same four test cases.
