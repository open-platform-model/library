---
title: "Over-subscribed provider contracts"
description: "More than one catalog on the platform claims to be the single provider of a contract."
type: how-to
sidebar:
  order: 22
---

<!-- Diagnostics entry for the kernel's *OverSubscribedContractsError, the render gate's single-provider guard. The same condition is refused earlier, when the platform itself is generated or checked, under the Platform reason OverSubscribedContracts; this entry covers both surfaces because the fix is the same. Check against: library/opm/errors/oversubscribed.go, opm-operator/internal/controller/platform_inventory.go -->

## The message

<!-- The kernel's text as OverSubscribedContractsError.Error() builds it, one line per contract, catalogs sorted and quoted. The CLI prints it after "render failed: " and adds a details line `contract "<contract-fqn>": provided by more than one enabled catalog: <catalog-path>, <catalog-path>`. The operator's Platform condition words it differently: "platform is not routable: <n> over-subscribed contract(s); a platform package cannot be generated until one competing catalog is disabled or its claim removed:" followed by `  <contract-fqn> (defined by <catalog-path>) required by <transformer>, <transformer>`. `opm platform check` prints an "over-subscribed contracts: <n>" section. Check against: library/opm/errors/oversubscribed.go, cli/internal/workflow/render/validation.go, opm-operator/internal/controller/platform_inventory.go, cli/internal/platform/check.go -->

```text
<count> over-subscribed provider contract(s):
  contract "<contract-fqn>" declares fulfilment "provider" but is supplied by transformers from <n> catalogs ("<catalog-path>", "<catalog-path>"): a platform must carry exactly one provider for it
```

## What it means

<!-- Two sentences at most. A contract whose definition declares `fulfilment: "provider"` must be served by transformers from exactly one enabled catalog, and this platform enables two or more that require it. The guard reads the platform, not the module, so every render against this platform fails until the platform is fixed, whatever module is rendered. Link the concept page Platforms and catalogs. Check against: core/src/resource.cue, core/src/trait.cue, library/opm/internal/renderstage/render.cue.tmpl -->

## Causes and fixes

### Two subscribed catalogs ship a transformer for the same provider contract

<!-- Recognise it by the catalog list naming two registry keys that are both platform subscriptions. Fix: disable one of them, then re-check. Cluster: set `enable: false` on one `spec.registry` entry of the Platform; a disabled entry stays pinned and imported but contributes no transformer. Local: set `enable: false` on one `#registry` entry of the platform module. Check with `opm platform check` (exits 2 while any contract is over-subscribed) or by the Platform returning to Ready=True reason Generated. Check against: opm-operator/api/v1alpha1/platform_types.go, core/src/platform.cue, cli/internal/cmd/platform/check.go, opm-operator/internal/controller/platform_controller.go -->

### A provider's registration competes with a subscribed catalog

<!-- Recognise it on the operator: the registry key of a TransformerRegistration's catalog appears in the list beside a subscription. The operator refuses such a claim before it reaches the platform (claim Ready=False reason ContractSubscribed, or ContractClaimed when another active claim already provides the contract), so on a cluster this cause surfaces on the TransformerRegistration, not as this error. Fix: disable the subscription or delete the competing claim's provider, as the claim's message says. Verify: whether any path still lets a claim-contributed catalog and a subscription both reach a generated platform. Check against: opm-operator/internal/controller/transformerregistration_controller.go, opm-operator/internal/status/conditions.go -->

## Where it is raised

<!-- Badge: kernel. Raised three ways. Kernel render gate: Kernel.Render returns *kernel.RenderError carrying *errors.OverSubscribedContractsError; the CLI surfaces it on `opm module build`, `opm module apply`, `opm instance build`, `opm instance apply`, `opm instance diff` and `opm instance vet` against a local platform module, exiting with code 2. Platform generation: the operator refuses to record the platform package and sets Platform Ready=False and Stalled=True with reason OverSubscribedContracts; `opm platform check` exits 2. Instance render on the operator: RenderFailed. Verify: whether the operator's instance path can reach this error at all, since a refused platform package is never recorded and renders keep consuming the last good package (or report PlatformNotReady when there is none). Check against: library/opm/kernel/render.go, library/opm/kernel/render_decode.go, opm-operator/internal/controller/platform_controller.go, opm-operator/internal/reconcile/resolution.go, cli/internal/cmd/platform/check.go -->
