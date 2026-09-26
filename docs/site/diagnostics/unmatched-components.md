---
title: "No matching transformer"
description: "A component matches no transformer, so nothing can render it."
type: how-to
sidebar:
  order: 21
---

<!-- Diagnostics entry for the kernel's *UnmatchedComponentsError, the render gate's refusal for a component that no transformer matched. When the render also carries unresolved demands, both messages print together; the Unresolved demands entry covers that half. Check against: library/opm/errors/unmatched.go, library/opm/kernel/render_decode.go -->

## The message

<!-- The kernel's text as UnmatchedComponentsError.Error() builds it. The "missing labels" line appears only when the label predicate refused the candidate; a candidate refused for a body conflict, or for a missing required resource or trait, prints the "did not match" line alone. The CLI prints it after "render failed: ", then one details line per component, `component "<component>": no transformer matched`; with `--verbose` each refused candidate follows as `  candidate "<transformer-fqn>" did not match: <reason>`, where the reason is "missing labels <key>, <key>", "bodies conflict at <fqn>, <fqn>" or "bodies do not unify". The operator's condition message is the kernel text after "rendering module instance: render refused: ". Check against: library/opm/errors/unmatched.go, cli/internal/workflow/render/validation.go, opm-operator/internal/render/kernel_module_renderer.go -->

```text
<count> component(s) have no matching transformer: [<component> ...]
  component "<component>":
    transformer "<transformer-fqn>" did not match:
      missing labels:    [<label-key> ...]
```

## What it means

<!-- Two sentences at most. For each component the kernel collects candidate transformers through the contracts the component carries, and a candidate matches only if its required labels, required resources and required traits are all present and its required bodies unify with the component's; here none did, so nothing would produce this component's objects. Link the concept pages How matching works and Components and blueprints. Check against: library/opm/internal/renderstage/render.cue.tmpl, core/src/transformer.cue -->

## Causes and fixes

### The component's labels select no transformer

<!-- Recognise it by "missing labels: [<key>]" (CLI verbose: "missing labels <key>"). A candidate's `requiredLabels` names a key the component's `matchLabels` lacks or carries with a different value; matching reads `matchLabels`, never `metadata.labels`. The common case in catalog_opm is `core.opmodel.dev/workload-type`, set by the blueprint (for example "stateless" on the stateless workload blueprint) and read by the Deployment, StatefulSet, DaemonSet, Job and CronJob transformers. Fix: build the component from the blueprint whose workload type an enabled transformer serves, or enable the catalog that carries the transformer for the workload type the component declares. Link the pages Choose a blueprint and How matching works. Check against: core/src/transformer.cue, core/src/resource.cue, catalog_opm/opm/blueprints/v1beta1/stateless_workload.cue, catalog_opm/opm/resources/v1beta1/container.cue, catalog_opm/opm/transformers/deployment_transformer.cue -->

### The component lacks a resource or trait the transformer requires

<!-- Recognise it by a candidate that "did not match" with no missing labels and no conflict. The predicate rung also refuses a candidate whose `requiredResources` or `requiredTraits` names a member the component does not carry, but the kernel's candidate row records only missing labels, so the message does not name the missing member. Fix: compare the candidate transformer's `requiredResources` and `requiredTraits` with what the component carries, and attach what is missing. Verify: the CLI's verbose output words this case "bodies do not unify", which is wrong for it; the author should either say so or ask for the row to carry the missing members first. Check against: library/opm/internal/renderstage/render.cue.tmpl, library/opm/errors/unmatched.go, cli/internal/workflow/render/validation.go -->

### A required body conflicts with the component

<!-- Recognise it with `--verbose`: "bodies conflict at <fqn>, <fqn>". The always-unify rung unifies each body the transformer requires with the component's body for the same resource or trait, and a conflict disqualifies the candidate. The same conflict shows up as "; <n> candidate(s) disqualified" on the Unresolved demands message. Fix: change the component's values at the listed FQNs so they unify with the transformer's requirement, or use the member variant the transformer serves. Check against: library/opm/internal/renderstage/render.cue.tmpl, library/opm/errors/match.go, cli/internal/workflow/render/validation.go -->

### No transformer lists anything the component carries

<!-- Recognise it by a component line with no transformer lines under it: no candidate existed at all. Every contract the component carries is then also reported as an unresolved demand, and the fix is on that entry: enable the catalog that defines and serves the contracts. Check against: library/opm/errors/unmatched.go, library/opm/internal/renderstage/render.cue.tmpl -->

## Where it is raised

<!-- Badge: kernel. Raised by the kernel's render gate after the build (Kernel.Render returns *kernel.RenderError; the cause is reachable with errors.As on *errors.UnmatchedComponentsError). CLI: `opm module build`, `opm module apply`, `opm instance build`, `opm instance apply`, `opm instance diff` and `opm instance vet`, exiting with code 2; `--verbose` adds the per-candidate reasons. Operator: ModuleInstance and ModulePackage report Ready=False and Stalled=True with reason ResolutionFailed, plus a Warning event; see the Operator conditions entry. Check against: library/opm/kernel/render.go, cli/internal/workflow/render/validation.go, cli/internal/cmd/root.go, opm-operator/internal/reconcile/resolution.go -->
