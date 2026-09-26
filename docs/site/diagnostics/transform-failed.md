---
title: "Transform failed"
description: "A transformer ran on a component and its output failed to evaluate."
type: how-to
sidebar:
  order: 25
---

<!-- Diagnostics entry for the kernel's *TransformError, raised after matching succeeded, when a matched (component, transformer) pair's output is an error, is not concrete or has the wrong shape. Verify: the type's message carries no name of its own; the nearest printed words are "executing transforms" (kernel) and "transformer <fqn> failed" (CLI details), so the author may prefer one of those as the title. Check against: library/opm/errors/domain.go, library/opm/kernel/render_decode.go -->

## The message

<!-- The kernel's text: TransformError.Error() under the "executing transforms: " wrap, one line per failed pair. `<cause>` is one of: the CUE error the output carried; "transformer output is an error"; "output is not concrete: <cue-error>"; "unexpected output kind <kind> (must be struct for a single resource or list for multiple)"; "iterating output list: <error>"; "rendered output missing at \"<component> :: <transformer-fqn>\"". The CLI prints it after "render failed: " and adds `component "<component>": transformer <transformer-fqn> failed` for each pair whose output was an error (not for a non-concrete output). The operator's message is prefixed "rendering module instance: render refused: ". Check against: library/opm/errors/domain.go, library/opm/kernel/render_decode.go, library/opm/kernel/render.go, cli/internal/workflow/render/validation.go -->

```text
executing transforms: component "<component>", transformer "<transformer-fqn>": <cause>
```

## What it means

<!-- Two sentences at most. Matching found a transformer for the component, but evaluating that transformer against the component and the instance did not produce a concrete Kubernetes object or list of objects. Nothing from the render is applied, including the pairs that did evaluate. Link the concept page How matching works for where transformation sits after matching. Check against: library/opm/kernel/render_decode.go, library/opm/internal/renderstage/render.cue.tmpl -->

## Causes and fixes

### The component's values conflict with what the transformer produces

<!-- Recognise it by a `<cause>` that is a CUE conflict with file positions, or "transformer output is an error". The transformer unifies the component's spec, the instance and the runtime context into its output, and a value the module's configuration accepted conflicts with a constraint the transformer's output carries. Fix: read the CUE error's positions, find the component field it traces back to, and change the value in the module or the instance's values; rerun `opm module build` or `opm instance build` to confirm. Verify: pick a real catalog_opm example that passes #config and still conflicts in a transformer, for the worked example. Check against: library/opm/kernel/render_decode.go, library/opm/internal/renderstage/render.cue.tmpl, catalog_opm/opm/transformers -->

### A field in the output has no concrete value

<!-- Recognise it by "output is not concrete: ...". The output unified without conflict but left a field open: a value the transformer reads was never set by the component, its blueprint or the instance's values. Fix: supply the field the CUE error names, in the component or in the values. Check against: library/opm/kernel/render_decode.go -->

### The transformer returns something other than an object or a list

<!-- Recognise it by "unexpected output kind <kind> ..." or "iterating output list: ...". A transformer's `output` must be a struct for one object or a list for several; anything else is a defect in the catalog that ships the transformer, not in the module. Fix: report it to the catalog's owner with the transformer FQN from the message; authors of their own catalogs fix the transformer's `#transform` output. Link the page Write a transformer. Check against: library/opm/kernel/render_decode.go, core/src/transformer.cue -->

### The render build lost a matched pair's output

<!-- Recognise it by "rendered output missing at ...". The build matched the pair but produced no output entry for it, which no module or catalog can cause on purpose. Fix: none on the reader's side; report it against the library with the full message. Verify: whether this is reachable at all or only a defensive check, and drop the section if it is not. Check against: library/opm/kernel/render_decode.go, library/opm/internal/renderstage/render.cue.tmpl -->

## Where it is raised

<!-- Badge: kernel. Raised by Kernel.Render after the gate passes, while decoding each matched pair's output; it arrives inside *kernel.RenderError and is reachable with errors.As on *errors.TransformError, whose Unwrap returns the cause. CLI: `opm module build`, `opm module apply`, `opm instance build`, `opm instance apply`, `opm instance diff` and `opm instance vet`, exiting with code 2. Operator: ModuleInstance and ModulePackage report Ready=False and Stalled=True with reason RenderFailed, plus a Warning event; see the Operator conditions entry. Check against: library/opm/kernel/render.go, library/opm/errors/domain.go, cli/internal/workflow/render/render.go, opm-operator/internal/reconcile/resolution.go -->
