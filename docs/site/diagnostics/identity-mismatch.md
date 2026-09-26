---
title: "Identity mismatch"
description: "A fetched module declares a different path or version than the one it was fetched as."
type: how-to
sidebar:
  order: 23
---

<!-- Diagnostics entry for the kernel's errors.IdentityError, returned when a module fetched from a registry declares a different identity than the coordinate it was fetched by. Modules only: catalogs acquired from a registry get no such check, and a platform's catalog builds are verified by CUE when the platform builds, with a different message. Check against: library/opm/errors/identity.go, library/opm/internal/loader/registry.go -->

## The message

<!-- The kernel's text as IdentityError.Error() builds it. `<field>` is "path" or "version". For a version mismatch both values are bare SemVer (the tag's "v" is stripped from the fetched side). The operator's message is prefixed "acquiring module: acquiring module \"<module-path>\"@\"<version>\": ". The CLI's only surface, `opm module init` from a template, prefixes "registry unreachable: acquiring <module-path> <version> (registry <registry>): ". Check against: library/opm/errors/identity.go, opm-operator/internal/moduleacquire/acquire.go, opm-operator/internal/render/kernel_module_renderer.go, cli/internal/scaffold/scaffold.go, cli/internal/publish/publish.go -->

```text
identity mismatch at <field>: metadata declares "<declared>" but the artifact was fetched as "<fetched>" (<module-path> <version>)
```

## What it means

<!-- Two sentences at most. The module's own `metadata.modulePath` or `metadata.version` does not equal the major-qualified path or the tag it was fetched by, so the bytes at that coordinate are not the module they claim to be. The kernel verifies the declared identity and never rewrites it. Link the concept page Identity and names. Check against: library/opm/internal/loader/registry.go, core/src/identity_package.cue -->

## Causes and fixes

### The module declares a different path than the one it was fetched by

<!-- Recognise it by "identity mismatch at path". Either the reference names the wrong path (on the operator, the ModuleInstance's `spec.module.path`), or the artifact at that path declares another one, for example a module copied from another and published without re-identifying it, or a `modulePath` without its "@v<major>" suffix, which can never equal the fetched path. Fix: if the reference is wrong, correct `spec.module.path` to the path the module declares. If the artifact is wrong, fix `ModulePath` in the module's `identity/identity.cue` and publish a new version with `opm module publish`, whose gates refuse this state before it reaches a registry. Check against: library/opm/internal/loader/registry.go, opm-operator/api/v1alpha1/common_types.go, cli/internal/publish/gates.go, cli/internal/publish/identity.go -->

### The module declares a different version than its tag

<!-- Recognise it by "identity mismatch at version". The tag the registry served does not equal `metadata.version`, which happens when an artifact was pushed by a tool other than `opm module publish` (the publish pipeline derives the tag from the declared version). Fix: publish the module again with `opm module publish`, which takes the tag from `identity/identity.cue`, and point `spec.module.version` at the new tag. Verify: whether `cue mod publish` with a hand-picked version is the realistic way to produce this state. Check against: library/opm/internal/loader/registry.go, cli/internal/publish/identity.go, cli/internal/publish/registry.go -->

## Where it is raised

<!-- Badge: kernel. Raised at module acquisition from a registry, before any render: Kernel.AcquireModuleFromRegistry returns it bare, as a value type, so an embedder matches it with errors.As on errors.IdentityError, not a pointer. Operator: ModuleInstance reports Ready=False and Stalled=True with reason ResolutionFailed and a Warning event; ModulePackage never raises it, because packages load from a Flux artifact. CLI: `opm module init <new-module-path> <template>`, which acquires the template module from the registry; no render command acquires a module from a registry. Verify: the CLI wraps this refusal as a connectivity error, so it prints "registry unreachable" and exits with code 3, which misdescribes it; flag for the cli owners before documenting the exit code. Check against: library/opm/kernel/acquire.go, opm-operator/internal/reconcile/resolution.go, opm-operator/internal/moduleacquire/acquire.go, cli/internal/scaffold/scaffold.go, cli/internal/cmd/module/init.go -->
