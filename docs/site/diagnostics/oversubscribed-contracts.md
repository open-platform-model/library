---
title: "Over-subscribed provider contracts"
description: "More than one catalog on the platform claims to be the single provider of a contract."
type: how-to
sidebar:
  order: 22
---

<!-- Diagnostics entry for the kernel's *OverSubscribedContractsError, the render gate's single-provider guard. The same condition is refused earlier, when the platform itself is generated or checked, under the Platform reason OverSubscribedContracts; this entry covers both surfaces because the fix is the same. Both surfaces now read one count, core's #Platform.#contracts.providedBy: the render glue reads it and the platform check reads it through Contracts(), so the render refusal and the platform refusal fire on exactly the same platforms. A platform module pinning core older than 2.0.0-alpha.12 carries no such count and is refused before any render with PlatformCoreTooOldError instead (fix: re-pin opmodel.dev/core in the platform module). Check against: library/opm/errors/oversubscribed.go, library/opm/errors/coretooold.go, opm-operator/internal/controller/platform_inventory.go -->

## The message

<!-- The kernel's text as OverSubscribedContractsError.Error() builds it, one line per contract, catalogs sorted and quoted. The CLI prints it after "render failed: " and adds a details line `contract "<contract-fqn>": provided by more than one enabled catalog: <catalog-path>, <catalog-path>`. The operator's Platform condition words it differently: "platform is not routable: <n> over-subscribed contract(s); a platform package cannot be generated until one competing catalog is disabled or its claim removed:" followed by `  <contract-fqn> (defined by <catalog-path>) provided by <registry-key>, <registry-key>`, the defined-by clause present only when an enabled catalog defines the contract. `opm platform check` prints an "over-subscribed contracts: <n>" section, each contract followed by `    provided by  <registry-key>, <registry-key>`. Both read the registry keys from ContractInventory.ProvidedBy, which carries them even when no enabled catalog defines the contract. When the platform also has colliding contracts, the kernel joins the collision refusal ahead of this one and the CLI prints the collision rows first; the operator reports reason ContractCollisions instead, with this finding after the collision in the same message. Check against: library/opm/errors/oversubscribed.go, cli/internal/workflow/render/validation.go, opm-operator/internal/controller/platform_inventory.go, cli/internal/platform/check.go -->

```text
<count> over-subscribed provider contract(s):
  contract "<contract-fqn>" declares fulfilment "provider" but is supplied by transformers from <n> catalogs ("<catalog-path>", "<catalog-path>"): a platform must carry exactly one provider for it
```

## What it means

<!-- Two sentences at most. A contract whose definition declares `fulfilment: "provider"` must be served by transformers from exactly one enabled registry entry (path plus major, so two majors of one catalog are two providers), and this platform enables two or more that require it; a disabled or absent defining catalog does not hide it. The guard reads the platform, not the module, so every render against this platform fails until the platform is fixed, whatever module is rendered. Link the concept page Platforms and catalogs. Check against: core/src/resource.cue, core/src/trait.cue, core/src/platform.cue, library/opm/internal/renderstage/render.cue.tmpl -->

## Causes and fixes

### Two subscribed catalogs ship a transformer for the same provider contract

<!-- Recognise it by the catalog list naming two registry keys that are both platform subscriptions. Fix: disable one of them, then re-check. Cluster: set `enable: false` on one `spec.registry` entry of the Platform; a disabled entry stays pinned and imported but contributes no transformer. Local: set `enable: false` on one `#registry` entry of the platform module. Check with `opm platform check` (exits 2 while any contract is over-subscribed) or by the Platform returning to Ready=True reason Generated. Check against: opm-operator/api/v1alpha1/platform_types.go, core/src/platform.cue, cli/internal/cmd/platform/check.go, opm-operator/internal/controller/platform_controller.go -->

### A provider's registration competes with a subscribed catalog

<!-- Recognise it on the operator: the registry key of a TransformerRegistration's catalog appears in the list beside a subscription. The operator refuses such a claim before it reaches the platform (claim Ready=False reason ContractSubscribed, or ContractClaimed when another active claim already provides the contract), so on a cluster this cause surfaces on the TransformerRegistration, not as this error. Fix: disable the subscription or delete the competing claim's provider, as the claim's message says. Verify: whether any path still lets a claim-contributed catalog and a subscription both reach a generated platform. Check against: opm-operator/internal/controller/transformerregistration_controller.go, opm-operator/internal/status/conditions.go -->

## Where it is raised

<!-- Badge: kernel. Raised three ways. Kernel render gate: Kernel.Render returns *kernel.RenderError carrying *errors.OverSubscribedContractsError; the CLI surfaces it on `opm module build`, `opm module vet`, `opm module apply`, `opm instance build`, `opm instance apply`, `opm instance diff` and `opm instance vet`, whichever platform the command resolves, exiting with code 2. Platform generation: the operator refuses to record the platform package and sets Platform Ready=False and Stalled=True with reason OverSubscribedContracts (ContractCollisions takes precedence when a contract key also collides; see Colliding contracts); `opm platform check` exits 2. Instance render on the operator: not reached for a platform the inventory refuses. A refused platform package is never recorded, renders keep consuming the last good package (or report PlatformNotReady when there is none), and since the render and the inventory read one count, a recorded package whose inventory reads routable never raises this error. Check against: library/opm/kernel/render.go, library/opm/kernel/render_decode.go, opm-operator/internal/controller/platform_controller.go, opm-operator/internal/controller/platform_inventory.go, opm-operator/internal/reconcile/resolution.go, cli/internal/cmd/platform/check.go -->
