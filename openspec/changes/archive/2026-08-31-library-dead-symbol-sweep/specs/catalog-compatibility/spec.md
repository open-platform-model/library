## REMOVED Requirements

### Requirement: Provenance Stripping
**Reason**: `opm/compat.StripProvenance` had no caller. Both places 0010 D30's denylist applies already implement it without a syntax round-trip: the comparison walk (`Check` / `CheckAtLevel`) skips `catalogVersion` and `description` directly under any `metadata` field at every depth, and the matcher's always-unify rung excludes diagnostics located at those paths from the verdict. The strip's round-trip also discarded document positions, a cost the walk-side skip does not pay. D30 remains implemented; only the unused mechanism is removed.
**Migration**: None for the publish gate or the matcher. A consumer that wants a provenance-free copy of a value performs its own syntax round-trip; the library offers no helper.
