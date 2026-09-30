---
title: "Colliding contracts"
description: "Two or more enabled catalogs on the platform define the same contract key."
type: how-to
weight: 19
---

<!-- Diagnostics entry for the kernel's *ContractCollisionsError, the first cause the render gate joins. It is raised when the catalogs of two or more enabled registry entries list the same contract key in their contract maps, most often two majors of one catalog enabled side by side. Core (from 2.0.0-alpha.13) reports such keys on #Platform.#contracts as collisions and collidingEntries, folds only keys with exactly one enabled definer into definedBy, and reads routable false; the render glue reads that report and never computes it, and Platform.Contracts() decodes the same fields (ContractInventory.Collisions, CollidingEntries), so the render refusal and the inventory agree on every platform. A platform module pinning core older than 2.0.0-alpha.13 cannot evaluate such a platform at all: acquisition fails on a definedBy (or defined) conflict, "conflicting values", and this error is never reached. Check against: library/opm/errors/collision.go, library/opm/kernel/render_decode.go, library/opm/platform/contracts.go, core/src/platform.cue -->

## The message

<!-- The kernel's text as ContractCollisionsError.Error() builds it, one line per colliding key in ascending key order, the registry keys (path@major) sorted and quoted. The CLI prints it after "render failed: " and adds one details line per key, `contract "<contract-fqn>": defined by more than one enabled registry entry: <catalog-path>@v0, <catalog-path>@v1`, ahead of every other row, and prints no remediation hint under this refusal: the other rows are read against an inventory the collision distorts, so a provider hint would misdirect. `opm platform check` prints a "colliding contracts: <n>" section, each key followed by `    defined by  <registry-key>, <registry-key>` and the note "(a colliding contract is left out of the defined, required, unfulfilled and comparable sections; keep one of its defining entries enabled)"; its routable verdict reads no, naming the count of colliding contracts before the over-subscribed count, its exit message counts the colliding contracts first, and its --help lists the colliding row among the refusals that exit with the validation error code. The operator's Platform condition words it "platform is not routable: <n> colliding contracts; a platform package cannot be generated until all but one of the registry entries defining each is disabled:" (the noun agrees with the count: "1 colliding contract") followed by `  <contract-fqn> defined by <registry-key>, <registry-key>`, each further finding after a blank line. Check against: library/opm/errors/collision.go, library/opm/errors/collision_test.go, cli/internal/workflow/render/validation.go, cli/internal/platform/check.go, cli/internal/cmd/platform/check.go, opm-operator/internal/controller/platform_inventory.go -->

```text
<count> colliding contract(s):
  contract "<contract-fqn>" is defined by <n> enabled registry entries ("<catalog-path>@v0", "<catalog-path>@v1"); a contract key must have exactly one enabled definer until side-by-side catalog majors are supported, so disable all but one of these entries
```

## What it means

<!-- Two sentences at most. Every contract key must have exactly one enabled definer, and this platform enables two or more registry entries whose catalogs list the key, so no single definition of the key exists for a module or a transformer to be matched against. The refusal reads the platform, not the module, so every render against this platform fails until the platform is fixed, whatever module is rendered and whether or not unprovided demands are skipped. Link the concept page Platforms and catalogs. Check against: core/src/platform.cue, library/opm/internal/renderstage/render.cue.tmpl -->

## Causes and fixes

### Two majors of one catalog are enabled together

<!-- Recognise it by the entries naming one catalog path at two majors (path@v0 and path@v1). Side-by-side catalog majors are not supported yet. Fix: disable all but one major, then re-check. Cluster: set `enable: false` on one `spec.registry` entry of the Platform. Local: set `enable: false` on one `#registry` entry of the platform module. A disabled entry never counts as a definer. Re-check with `opm platform check` (the colliding section disappears) or by the Platform leaving reason ContractCollisions. Check against: core/src/platform.cue, opm-operator/api/v1alpha1/platform_types.go -->

### Two different catalogs list the same key

<!-- Recognise it by the entries naming two different catalog paths. Fix: disable one entry; the key belongs to one catalog. Check against: core/src/platform.cue -->

## A known blind spot while collisions exist

<!-- A colliding key is left out of definedBy, requiredBy, unfulfilled and comparable, so ContractInventory.Fulfilled and Discriminated can read true on a colliding platform: a transformer requiring the colliding key is neither counted as unfulfilled nor compared with another under an equal predicate. Never read either as safe while Collisions is non-empty; Routable reads false. providedBy and overSubscribed are unaffected, so an over-subscription of a colliding provider-fulfilled key is reported beside the collision (and refused after it). An unresolved demand on a colliding key carries no defining catalog; its row names the colliding entries (UnresolvedDemand.Colliding) and says "defined by more than one enabled registry entry". Check against: library/opm/platform/contracts.go, library/opm/errors/match.go, core/src/platform_contracts_pins.cue -->

## Where it is raised

<!-- Badge: kernel. Kernel render gate: Kernel.Render returns *kernel.RenderError whose first joined cause is *errors.ContractCollisionsError, carrying the rows RenderDiagnostics.Collisions carries, with RenderDiagnostics.Routable false; no object is rendered. Platform inventory: Platform.Contracts() returns the collisions as a report, never an error. CLI: the render refusal on every command that renders (`opm module build`, `opm module vet`, `opm module apply`, `opm instance build`, `opm instance apply`, `opm instance diff`, `opm instance vet`), exiting with code 2, and `opm platform check`, which reads the platform not routable and also exits 2. Operator: the Platform refuses to record the platform package and reports Ready=False and Stalled=True with reason ContractCollisions, plus a Warning event of the same reason; ContractCollisions takes precedence over OverSubscribedContracts and ComparablePredicates, and the message carries every finding, the collision first. Renders keep consuming the last good package (or report PlatformNotReady when there is none), so a ModuleInstance never meets this refusal on the operator. Check against: library/opm/kernel/render.go, library/opm/kernel/render_decode.go, library/opm/platform/contracts.go, cli/internal/cmd/platform/check.go, opm-operator/internal/controller/platform_inventory.go, opm-operator/internal/status/conditions.go, opm-operator/internal/controller/platform_controller.go -->

## Not routable with nothing to explain it

<!-- One line: *errors.NotRoutableError ("platform is not routable: its contract inventory reads routable false and reports no over-subscribed or colliding contract to explain it") is the render gate's catch-all, joined last, raised only when core's routable reads false and no collision or over-subscription row explains it. No released core produces it; seeing it means the platform's core added a routable term this kernel does not decode, so upgrade the library. The operator's counterpart is a Platform finding under reason OverSubscribedContracts, "platform is not routable; the contract inventory names no over-subscribed or colliding contract, so a platform package cannot be generated". Check against: library/opm/errors/collision.go, library/opm/kernel/render_decode.go, opm-operator/internal/controller/platform_inventory.go -->
