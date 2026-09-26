---
title: "Unresolved demands"
description: "A module needs a contract that no catalog on the platform provides."
type: how-to
sidebar:
  order: 20
---

<!-- Diagnostics entry for the kernel's *UnresolvedDemandsError, the fail-closed render gate's refusal for a demanded resource or trait contract that nothing on the platform resolves. The title is the aggregate's printed name ("unresolved demand(s)"). Often appears together with the No matching transformer entry: when a component's only demands are unresolved, the same render also reports the component unmatched, and the two causes are joined in one message. Check against: library/opm/errors/match.go, library/opm/kernel/render_decode.go -->

## The message

<!-- The kernel's text as UnresolvedDemandsError.Error() builds it: one header line, then one line per demand whose tail is one of four variants, with "; <n> candidate(s) disqualified" appended when candidates existed and the always-unify rung refused them. `<kind>` is "resource" or "trait"; alternatives print as a Go list, space-separated in brackets. The CLI prints it after "render failed: " and then a details block with one entry per demand: `component "<component>": unresolved <kind> demand "<contract-fqn>"` followed by one of "  nothing on this platform implements this contract", "  defined by \"<catalog-path>\", implemented by nothing on this platform", or "  defined by \"<catalog-path>\"" plus "  implemented at: <fqn>, <fqn>". The operator's condition message is the kernel text after "rendering module instance: render refused: ". Verify: the log-level prefix the CLI's logger puts in front of the first line. Check against: library/opm/errors/match.go, library/opm/kernel/render.go, cli/internal/cmdutil/output.go, cli/internal/workflow/render/validation.go, opm-operator/internal/render/kernel_module_renderer.go -->

```text
<count> unresolved demand(s):
  component "<component>": unresolved <kind> demand "<contract-fqn>": no enabled catalog defines this contract
  component "<component>": unresolved <kind> demand "<contract-fqn>": defined by "<catalog-path>" and nothing on this platform implements it
  component "<component>": unresolved <kind> demand "<contract-fqn>": implemented at a different apiVersion (alternatives: [<contract-fqn> ...])
  component "<component>": unresolved <kind> demand "<contract-fqn>": defined by "<catalog-path>", implemented at a different apiVersion (alternatives: [<contract-fqn> ...]); <n> candidate(s) disqualified
```

## What it means

<!-- Two sentences at most. A component carries a resource, or a trait whose `optional` posture is false, and no transformer on the platform both lists that contract and matches the component, so the render stops before any object is produced. Every declared resource is a required demand; a trait demand is only required when its effective `optional` is false. Link the concept page How matching works. Check against: library/opm/errors/match.go, library/opm/internal/renderstage/render.cue.tmpl, core/src/trait.cue -->

## Causes and fixes

### No enabled catalog defines the contract

<!-- Recognise it by the tail "no enabled catalog defines this contract" (CLI detail: "nothing on this platform implements this contract"). The module imports a resource or trait from a catalog the platform does not carry, or carries with `enable` set to false. Fix: add or enable the catalog on the platform. Cluster: a `spec.registry` entry on the Platform, keyed by the catalog's major-suffixed module path, with `version` and `enable`. Local: the platform module's `#registry` entry, read from `--platform`, the cluster Platform, or `~/.opm/platform/` in that order. Then re-run the render. Alternative fix: replace the member in the module with one the platform's catalogs define. `opm platform check` lists every contract the enabled catalogs define. Check against: library/opm/errors/match.go, core/src/platform.cue, opm-operator/api/v1alpha1/platform_types.go, cli/internal/cmdutil/flags.go, cli/internal/cmd/platform/check.go -->

### The contract waits for a provider nobody installed

<!-- Recognise it by "defined by \"<catalog-path>\" and nothing on this platform implements it". An enabled catalog lists the contract, and the contract declares `fulfilment: "provider"`: that catalog ships no transformer for it, and a separately installed provider must. Fix: install the provider module whose TransformerRegistration names this catalog and lists the contract in `provides`; the claim must reach Ready=True reason Accepted and Active=True before the Platform regenerates with its catalog. The Platform's ContractsFulfilled condition (reason UnfulfilledContracts) names every contract in this state before any module demands it. Verify: the same tail also appears when a transformer for the contract exists but was refused by the label predicate, because predicate refusals add no disqualified entry and no alternative; say so or leave that case to the No matching transformer entry. Check against: core/src/resource.cue, core/src/trait.cue, library/opm/internal/renderstage/render.cue.tmpl, opm-operator/internal/controller/platform_inventory.go, opm-operator/api/v1alpha1/transformerregistration_types.go -->

### The platform implements a different apiVersion of the contract

<!-- Recognise it by "implemented at a different apiVersion (alternatives: [...])" (CLI detail: "implemented at: <fqn>, <fqn>"). The contract FQN's trailing "@<apiVersion>" is not one the platform's transformers list, but the same contract at other apiVersions is; alternatives are ordered alpha, beta, GA. Fix: either import the member at an apiVersion from the alternatives list in the module, or move the platform to a catalog build whose transformers serve the apiVersion the module uses. Link the concept page Versions in OPM for the apiVersion axis. Check against: library/opm/errors/match.go, library/opm/internal/renderstage/render.cue.tmpl -->

### Every candidate transformer conflicts with the component

<!-- Recognise it by the suffix "; <n> candidate(s) disqualified". Transformers exist for the contract, but the always-unify rung refused each one: a body the transformer requires does not unify with the component's own body for that resource or trait. `opm module build --verbose` or `opm instance build --verbose` lists each candidate with "bodies conflict at <fqn>, ..." under the unmatched component. Fix: change the component's field values so they unify with what the transformer requires, or pick the member variant the transformer serves. Check against: library/opm/errors/match.go, library/opm/internal/renderstage/render.cue.tmpl, cli/internal/workflow/render/validation.go -->

### A required trait has no transformer that handles it

<!-- Recognise it by `<kind>` "trait". The trait's effective `optional` is false, either as the catalog's default (`bool | *false`) or narrowed at the attachment site, and no matched transformer lists it in its required or optional traits. Fix: set `optional: true` at the attachment site when losing the trait's effect is acceptable (the render then continues with a warning "trait ... is not handled by any matched transformer (values will be ignored)"), or add a catalog whose transformer handles it. Note in the comment only: a trait whose catalog states no posture fails the render with a different, untyped message ("states no optional posture"), which is not this entry. Link the page Attach a trait to a component. Check against: core/src/trait.cue, library/opm/internal/renderstage/render.cue.tmpl, library/opm/kernel/render_decode.go, cli/internal/workflow/render/render.go -->

## Where it is raised

<!-- Badge: kernel. Raised by the kernel's render gate after the build (Kernel.Render returns *kernel.RenderError; the cause is reachable with errors.As on *errors.UnresolvedDemandsError). CLI: every command that renders, which is `opm module build`, `opm module apply`, `opm instance build`, `opm instance apply`, `opm instance diff` and `opm instance vet`, exiting with code 2; `opm module vet` does not render and never raises it. Operator: ModuleInstance and ModulePackage report Ready=False and Stalled=True with reason ResolutionFailed, plus a Warning event of the same reason; see the Operator conditions entry. Check against: library/opm/kernel/render.go, cli/internal/workflow/render/render.go, cli/internal/cmd/module/vet.go, cli/internal/exit/exit.go, opm-operator/internal/reconcile/resolution.go, opm-operator/internal/reconcile/moduleinstance.go, opm-operator/internal/reconcile/modulepackage.go -->
