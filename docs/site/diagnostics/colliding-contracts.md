---
title: "Colliding contracts"
description: "Two or more enabled catalogs on the platform define the same contract key."
type: how-to
weight: 19
---

<!-- Diagnostics entry for the kernel's *ContractCollisionsError, the first cause the render gate joins. It is raised when the catalogs of two or more enabled registry entries list the same contract key in their contract maps, most often two majors of one catalog enabled side by side. Core (from 2.0.0-alpha.13) reports such keys on #Platform.#contracts as collisions and collidingEntries, folds only keys with exactly one enabled definer into definedBy, and reads routable false; the render glue reads that report and never computes it, and Platform.Contracts() decodes the same fields (ContractInventory.Collisions, CollidingEntries), so the render refusal and the inventory agree on every platform. A platform module pinning core older than 2.0.0-alpha.13 cannot evaluate such a platform at all: acquisition fails on a definedBy (or defined) conflict, "conflicting values", and this error is never reached. Check against: library/opm/errors/collision.go, library/opm/kernel/render_decode.go, library/opm/platform/contracts.go, core/src/platform.cue -->

## The message

<!-- The kernel's text as ContractCollisionsError.Error() builds it, one line per colliding key in ascending key order, the registry keys (path@major) sorted and quoted. The operator and the CLI word the same rows on their own surfaces (a Platform readiness reason and an `opm platform check` section); name them here once they land. Check against: library/opm/errors/collision.go, library/opm/errors/collision_test.go -->

```text
<count> colliding contract(s):
  contract "<contract-fqn>" is defined by <n> enabled registry entries ("<catalog-path>@v0", "<catalog-path>@v1"); a contract key must have exactly one enabled definer until side-by-side catalog majors are supported, so disable all but one of these entries
```

## What it means

<!-- Two sentences at most. Every contract key must have exactly one enabled definer, and this platform enables two or more registry entries whose catalogs list the key, so no single definition of the key exists for a module or a transformer to be matched against. The refusal reads the platform, not the module, so every render against this platform fails until the platform is fixed, whatever module is rendered and whether or not unprovided demands are skipped. Link the concept page Platforms and catalogs. Check against: core/src/platform.cue, library/opm/internal/renderstage/render.cue.tmpl -->

## Causes and fixes

### Two majors of one catalog are enabled together

<!-- Recognise it by the entries naming one catalog path at two majors (path@v0 and path@v1). Side-by-side catalog majors are not supported yet. Fix: disable all but one major, then re-check. Cluster: set `enable: false` on one `spec.registry` entry of the Platform. Local: set `enable: false` on one `#registry` entry of the platform module. A disabled entry never counts as a definer. Check against: core/src/platform.cue, opm-operator/api/v1alpha1/platform_types.go -->

### Two different catalogs list the same key

<!-- Recognise it by the entries naming two different catalog paths. Fix: disable one entry; the key belongs to one catalog. Check against: core/src/platform.cue -->

## A known blind spot while collisions exist

<!-- A colliding key is left out of definedBy, requiredBy, unfulfilled and comparable, so ContractInventory.Fulfilled and Discriminated can read true on a colliding platform: a transformer requiring the colliding key is neither counted as unfulfilled nor compared with another under an equal predicate. Never read either as safe while Collisions is non-empty; Routable reads false. providedBy and overSubscribed are unaffected, so an over-subscription of a colliding provider-fulfilled key is reported beside the collision (and refused after it). An unresolved demand on a colliding key carries no defining catalog; its row names the colliding entries (UnresolvedDemand.Colliding) and says "defined by more than one enabled registry entry". Check against: library/opm/platform/contracts.go, library/opm/errors/match.go, core/src/platform_contracts_pins.cue -->

## Where it is raised

<!-- Badge: kernel. Kernel render gate: Kernel.Render returns *kernel.RenderError whose first joined cause is *errors.ContractCollisionsError, carrying the rows RenderDiagnostics.Collisions carries, with RenderDiagnostics.Routable false; no object is rendered. Platform inventory: Platform.Contracts() returns the collisions as a report, never an error. Check against: library/opm/kernel/render.go, library/opm/kernel/render_decode.go, library/opm/platform/contracts.go -->

## Not routable with nothing to explain it

<!-- One line: *errors.NotRoutableError ("platform is not routable: its contract inventory reads routable false and reports no over-subscribed or colliding contract to explain it") is the render gate's catch-all, joined last, raised only when core's routable reads false and no collision or over-subscription row explains it. No released core produces it; seeing it means the platform's core added a routable term this kernel does not decode, so upgrade the library. Check against: library/opm/errors/collision.go, library/opm/kernel/render_decode.go -->
