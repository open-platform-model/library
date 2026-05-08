// Package values is the opt-in Tier-1 helper for layered values validation
// with source-positioned diagnostics.
//
// # Two-tier validation
//
// The kernel implements Tier-2 correctness validation: given a single
// pre-unified [cue.Value], it confirms the result satisfies the module's
// `#config` schema. That tier is non-negotiable — every frontend pays it.
// What it cannot give the user is per-source attribution: by the time the
// kernel sees one merged value, the per-file CUE positions are gone.
//
// Tier-1 lives here. [ValidateAndUnify] takes a [Stack] of labeled [Layer]s
// (each carrying its own [cue.Value]), validates each layer independently
// against the schema, and only on success unifies in order to produce the
// single value the kernel expects. Per-layer failures aggregate into a
// [*MultiSourceError] that frontends format as they see fit (CLI prose,
// K8s status conditions, XR composition status). The two tiers compose:
// frontends call Tier-1, then pass the unified result to the kernel for
// Tier-2.
//
// This pattern is decisions D1 and D5 of the kernel-redesign-around-platform
// umbrella (enhancements/001-kernel-redesign-around-platform/).
//
// # Layer ordering
//
// [Stack] is a slice ordered later-overrides-earlier. `Stack{a, b, c}`
// unifies as `a → a∪b → a∪b∪c`; field conflicts resolve to the layer that
// wrote them last. The CLI mental model is `-f a -f b -f c`. Operators and
// composition functions construct stacks in their own preferred order.
//
// # When to use
//
// A frontend with two or more values sources benefits from Tier-1: the
// merged kernel error "values do not satisfy #config" becomes per-source
// "in user-values.cue at line 14: ...". A frontend with a single source
// can still call this helper (the diagnostics are unchanged) or skip it
// and pass its single value directly to the kernel — Tier-2 alone is
// correct, just less helpful.
//
// # Example: three-layer stack
//
//	defaults := k.CueContext().CompileString(`{ replicas: 1 }`)
//	user := k.CueContext().CompileString(`{ replicas: 3, image: "nginx" }`)
//	overlay := k.CueContext().CompileString(`{ env: "prod" }`)
//
//	stack := values.Stack{
//	    {Name: "defaults", Source: "embedded", Value: defaults},
//	    {Name: "values.cue", Source: "/etc/opm/values.cue", Value: user},
//	    {Name: "-f overlay.cue", Source: "./overlay.cue", Value: overlay},
//	}
//
//	merged, err := values.ValidateAndUnify(k, schema, stack)
//	if err != nil {
//	    for _, le := range err.Errors() {
//	        // present le.LayerName + le.Source + le.Err to the user
//	    }
//	    return err
//	}
//	// pass merged to the kernel — Tier-2 runs there
//	rel, _ := k.ParseModuleRelease(ctx, releaseVal, *mod, merged)
//
// # Kernel anchor
//
// [ValidateAndUnify] takes a [KernelOwner] — any type exposing
// `CueContext() *cue.Context`. [*kernel.Kernel] satisfies this interface,
// so `values.ValidateAndUnify(k, schema, stack)` is the typical call.
// The interface lives here so this package does not import pkg/kernel
// (which would cycle through the kernel's convenience wrapper).
//
// This is slice 05 of the kernel-redesign-around-platform enhancement.
// See enhancements/001-kernel-redesign-around-platform/03-decisions.md
// for D1 and D5.
package values
