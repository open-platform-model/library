---
title: "Version skew"
description: "A module needs a newer catalog build than the platform carries."
type: how-to
sidebar:
  order: 24
---

<!-- Diagnostics entry for the kernel's *SkewError. It is an error only under the refuse skew policy; under the default warn policy the same fact is a warning and the render proceeds. The page covers both, because a reader may arrive holding either line. Check against: library/opm/errors/skew.go, library/opm/kernel/render.go -->

## The message

<!-- The kernel's text as SkewError.Error() builds it, one line per skewed path, joined under "render refused before evaluation: ". The CLI prints it after "render failed: ". The operator's message is prefixed "rendering module instance: ". Under the warn policy the CLI and the operator both print the warning form instead: `version skew on "<module-path>": module requires <module-version>, platform carries <platform-version>; rendering against the platform's build` (operator: a Warning event with reason RenderWarning). Check against: library/opm/errors/skew.go, library/opm/kernel/render.go, cli/internal/workflow/render/validation.go, cli/internal/workflow/render/render.go, opm-operator/internal/render/warnings.go -->

```text
render refused before evaluation: version skew on "<module-path>": module requires <module-version>, platform carries <platform-version> (refused by policy)
```

## What it means

<!-- Two sentences at most. The module's `cue.mod/module.cue` pins a newer build of an OPM path (core or a catalog under opmodel.dev or a subdomain of it) than the platform module pins, and the render always evaluates against the platform's build. Under the refuse policy the render stops before evaluation rather than run the module against older definitions than it was written for. Link the concept page Versions in OPM. Check against: library/opm/internal/renderstage/skew.go, library/opm/internal/renderstage/modfile.go, library/opm/kernel/render.go -->

## Causes and fixes

### The module pins a newer catalog build than the platform subscribes to

<!-- Recognise it by a catalog path in the message, for example "opmodel.dev/catalogs/opm@v4". The module was tidied or bumped against a catalog release the platform has not adopted. Fix, either direction: raise the platform's pin (cluster: the Platform's `spec.registry["<catalog-path>"].version`; local: the platform module's pin, refreshed from the cluster with `opm platform pull <dir>` when the cluster is ahead), or lower the module's dependency to the platform's version and publish the module again. Check against: opm-operator/api/v1alpha1/platform_types.go, library/opm/internal/renderstage/skew.go, cli/internal/cmd/platform/pull.go -->

### The module pins a newer core build than the platform resolves

<!-- Recognise it by the core path in the message, "opmodel.dev/core@v<major>". The platform does not pin core directly; it resolves core through the catalogs it subscribes to. Fix: move the platform to a catalog build that requires the newer core, or lower the module's core pin. Verify: that the Platform CR offers no direct core pin and the dependency closure is the only way core's version moves. Check against: opm-operator/internal/controller/platform_controller.go, library/opm/internal/renderstage/skew.go -->

### The skew policy is set to refuse

<!-- Recognise it by "(refused by policy)": the same skew under warn only warns. Where the policy comes from: on the operator, the Platform's `spec.skewPolicy` ("Warn", the default, or "Refuse"); on the CLI, `skewPolicy` in `~/.opm/config.cue` ("warn" or "refuse"), except when the cluster Platform is the platform source, where the Platform's `spec.skewPolicy` wins; there is no flag. The CLI's provenance line names the active policy and its source ("skew policy: refuse (config)" or "(cluster Platform)"). Fix: resolve the skew with one of the causes above. Switching to warn renders the module against the platform's older build, which is the risk the policy exists to refuse. Check against: cli/internal/workflow/render/env.go, cli/internal/config/config.go, cli/internal/config/templates.go, opm-operator/api/v1alpha1/platform_types.go -->

## Where it is raised

<!-- Badge: kernel. Raised by Kernel.Render before evaluation when RenderInput.Skew is SkewRefuse; it is a plain joined error, not a *kernel.RenderError, and the cause is reachable with errors.As on *errors.SkewError. CLI: `opm module build`, `opm module apply`, `opm instance build`, `opm instance apply`, `opm instance diff` and `opm instance vet`, exiting with code 2. Operator: ModuleInstance and ModulePackage report Ready=False and Stalled=True with reason SkewRefused, plus a Warning event; see the Operator conditions entry. Check against: library/opm/kernel/render.go, cli/internal/workflow/render/validation.go, opm-operator/internal/reconcile/resolution.go, opm-operator/internal/status/conditions.go -->
