## Context

`#ModuleDebug` was previously contemplated as a top-level artifact analogous to `#Module` and `#ModuleRelease`. The catalog redesign (enhancement 015) folds debug into a `debugValues` field on `Module` itself. The kernel inherits this simplification: there is no debug artifact passed to the kernel; debug is just a value subtree the frontend may layer into the stack of values.

This slice ensures the kernel codebase reflects the new model. In practice the work is primarily a sweep: remove any vestigial debug-specific code paths, sharpen documentation, and ensure the version binding from `add-multi-apiversion-support` does not expose a debug-specific path or decoder.

## Goals / Non-Goals

**Goals:**
- Confirm and enforce that the kernel has zero awareness of `#ModuleDebug` as a top-level construct.
- Document `Module.debugValues` (a CUE field within `Module.Package`) as the only debug surface.
- Make explicit that frontend code (CLI, operator, XR) decides whether to layer `debugValues` into the values stack — this is policy, not kernel concern.

**Non-Goals:**
- Implementing the layering helper. That is part of slice 05 (`introduce-tiered-validation`), which adds `pkg/helper/values/` and may include a layer-set convention covering `debugValues`.
- Forbidding all use of debug values at runtime. The kernel renders whatever values it receives; whether `debugValues` was layered in is invisible to the kernel.
- Modifying CUE schemas in `apis/v1alpha2/`. Schema-side changes live in catalog 015.

## Decisions

**No deprecation cycle.** If a `ModuleDebug` Go type exists in intermediate code, remove it cleanly. There is no documented downstream consumer of such a type today; deprecation noise is wasted work.

**Documentation-first.** The largest tangible artifact of this slice is doc changes. Code changes are minor. The slice's value is in making the contract crisp: "the kernel accepts Module, ModuleRelease, Platform — full stop."

**Frontend layering convention.** The umbrella's two-tier validation pattern (slice 05) defines the layering convention. This slice does not prescribe it; it only asserts that debug is a layer, not an artifact.

## Risks / Trade-offs

**Risk — frontends that already have a `LoadModuleDebug` flow break.** Mitigation: search downstream repos (`cli`, `opm-operator`) before merging. Provide a one-line migration: read `debugValues` from `module.Package`.

**Risk — vestigial code is missed.** If a slice-04 or slice-05 author rediscovers a debug path, the cleanup was incomplete. Mitigation: explicit grep tasks in the task list; CHANGELOG entry naming the construct as retired.

**Trade-off — small slice.** This is the smallest slice in the umbrella. The value is decoupling: shipping it independently means later slices can assume the contract without dragging this cleanup along.
