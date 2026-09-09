// Package kernel exposes the OPM runtime as a single struct, [Kernel].
//
// Kernel owns its [*schema.Cache] for its entire lifetime and no build
// context: every operation that evaluates CUE creates its own [cue.Context],
// builds in it, and lets it go when it returns. Construction is [New] plus
// two options, [WithSchemaLoader] and [WithRegistry]; the kernel exposes no
// injection slot that no kernel operation reads. Downstream binaries (CLI,
// controller, Crossplane function) construct one Kernel per process and
// call methods on it instead of importing the individual loader / module /
// validate packages.
//
// # Surface
//
// One tier: every artifact a frontend can hold comes from an acquire verb, or
// from the package constructor it already has a value for.
//
//   - [Kernel.AcquireModuleFromRegistry] and [Kernel.AcquireModuleFromDir]
//     return a source-carrying [*module.Module];
//   - [Kernel.AcquirePlatformFromDir] returns a [*platform.Platform];
//   - [Kernel.AcquireInstanceFromDir] returns a validated
//     [*module.Instance], with optional values as trailing [Source] values;
//   - [Kernel.SynthesizeInstance] builds one from typed inputs
//     ([InstanceInput]); the module it takes comes from the two module
//     acquire verbs, and the core release the synthesized package imports is
//     the kernel's pinned schema release, read from the configured
//     [schema.OCILoader] with no schema load when it pins an exact release
//     (the default) and resolved through the schema cache otherwise;
//   - [Kernel.ValidateConfigDetailed] validates layered values;
//   - [Kernel.Render] renders an instance against a platform.
//
// There is no second, value-only tier: a caller that wants the raw value of
// an acquired artifact reads its Package field, and a caller holding a value
// it built itself calls [module.NewModuleFromValue] or
// [platform.NewPlatformFromValue] directly. The registry mapping is
// [WithRegistry] for every one of these operations, the schema cache
// included; no verb takes a per-call override.
//
// # Every operation shares nothing
//
// The Kernel holds no [cue.Context] (ADR-007). Each acquire verb, synthesis,
// validation and render creates a context for the call, builds in it, and
// returns; the values an operation returns (an artifact's Package, a
// validated value) keep that operation's runtime alive for exactly as long
// as the caller holds them, and the Kernel retains nothing. Memory held by a
// long-lived Kernel is therefore bounded by the artifacts its caller holds,
// not by the number of operations it has run. The cross-artifact verbs read
// only Metadata and Source from their inputs, never Package, so a module
// acquired by one Kernel synthesizes on another and an instance from either
// renders on a third. No method returns or accepts a [*cue.Context]; a
// caller that must compile a value against the schema takes the context of
// the value [schema.Cache.Get] returns.
//
// # Goroutine safety
//
// A single Kernel is safe for concurrent use across its own method calls:
// no operation shares evaluation state with another, and the schema cache is
// memoized under synchronization into a private context of its own. A
// consumer that needs concurrent operations shares one Kernel across its
// goroutines; there is nothing to gain from constructing more than one.
// Concurrency is across operations, never within one.
//
// [Kernel.Render] shares nothing between renders (ADR-005, enhancement 0019
// D8). Each render is its own CUE build in a fresh cue.Context created for
// that call and dropped when Render returns; no built value is retained
// between calls, and a caller cannot obtain one to hold. A consumer
// rendering from several goroutines calls Render on one Kernel, with no
// shared platform value and no mutex. There is no materialized platform to
// share and no serialised render path; the earlier shared-platform contract
// (ADR-002) is superseded, not supported.
//
// A render is single-threaded and its working set grows with the module, so
// a render pool is sized by memory rather than by core count: about 61 MB
// plus 7.75 MB per component per concurrent render (0019 experiment 08), and
// throughput saturates at roughly physical cores divided by 1.6 renders in
// flight. Size against the largest module the pool will see.
//
// # One-Kernel-per-process example
//
//	func renderAll(ctx context.Context, k *kernel.Kernel, platformDir string, instanceDirs []string) error {
//	    plat, err := k.AcquirePlatformFromDir(ctx, platformDir) // once; the platform is shared as data
//	    if err != nil {
//	        return err
//	    }
//	    var wg sync.WaitGroup
//	    errs := make(chan error, len(instanceDirs))
//	    for _, dir := range instanceDirs {
//	        wg.Add(1)
//	        go func(dir string) {
//	            defer wg.Done()
//	            inst, err := k.AcquireInstanceFromDir(ctx, dir) // the one Kernel, concurrently
//	            if err != nil {
//	                errs <- err
//	                return
//	            }
//	            if _, err := k.Render(ctx, kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "opm-cli"}); err != nil {
//	                errs <- err
//	            }
//	        }(dir)
//	    }
//	    wg.Wait()
//	    close(errs)
//	    for err := range errs {
//	        if err != nil {
//	            return err
//	        }
//	    }
//	    return nil
//	}
//
// # Rendering
//
// [Kernel.Render] is the kernel's single render verb. It takes a
// source-carrying instance ([Kernel.AcquireInstanceFromDir] or
// [Kernel.SynthesizeInstance]) and a source-carrying platform
// ([Kernel.AcquirePlatformFromDir]: a platform is a CUE module on disk that
// imports its catalogs), stages one generated render module that imports
// both (an on-disk input in place, an overlay-mode input served from memory;
// the per-render staging directory holds only the generated module), builds
// it once, and decodes the matching verdicts
// ([RenderDiagnostics]) and the rendered output ([RenderResult.Compiled],
// one entry per rendered object as a [*Compiled] carrying instance,
// component and transformer provenance). Matching and transformer execution are CUE inside the build,
// not Go; the build reports its verdicts as data and the kernel's fail-closed
// gate turns an unresolved demand, an unmatched component or an
// over-subscribed provider-fulfilled contract into a [*RenderError] that
// carries the full diagnostics, with the typed causes reachable through
// errors.As. Catalog version skew (the instance module requiring a newer
// OPM-namespace build than the platform carries) marks a resolved-versions
// row Newer by default ([SkewWarn]) or refuses before evaluation
// ([SkewRefuse]).
//
// A render result carries no presentation strings. The two advisory facts a
// render can report are rows on the diagnostics: an unhandled optional trait
// on RenderDiagnostics.UnhandledTraits, and a module requiring a newer build
// than the platform carries on a RenderDiagnostics.ResolvedVersions row with
// Newer set. A frontend words both:
//
//	for comp, traits := range result.Diagnostics.UnhandledTraits {
//		for _, fqn := range traits {
//			log.Printf("component %q: trait %q is unhandled", comp, fqn)
//		}
//	}
//	for _, r := range result.Diagnostics.ResolvedVersions {
//		if r.Newer {
//			log.Printf("%s: module requires %s, platform carries %s", r.Path, r.ModuleVersion, r.PlatformVersion)
//		}
//	}
//
// A dry run is Render with the output discarded: the build evaluates every
// matched pair regardless, and RenderDiagnostics carries the pairing
// diagnosis (Pairs, Unmatched, Unresolved, Unify, UnhandledTraits,
// OverSubscribed, ResolvedVersions). There is no separate match verb.
//
// An input's own cue.mod/local-module.cue (a developer redirecting a
// dependency to a directory or another module) reaches the render only under
// [RenderInput.LocalReplacements]. Off, the default, Render refuses before
// staging an input whose file carries a replacement rather than silently
// rendering against the published pin. On, the replacements are promoted
// into the render module under the precedence dependencies get (the
// platform's whole, the instance's only for paths the platform's dependency
// list does not name) and each honoured one is a
// [RenderDiagnostics.Replacements] row naming the path, the target and the
// input that supplied it; a replaced path keeps its pinned versions on
// ResolvedVersions. A frontend sets the flag for a developer's checkout and
// words the rows (an instance replacement the platform made inert is not a
// row, so the frontend computes the inert set from the file it read):
//
//	for _, r := range result.Diagnostics.Replacements {
//		log.Printf("%s: served from %s (%s local-module.cue)", r.Path, r.Target, r.By)
//	}
//
// Render consumes the instance as processed: values are validated where
// they are applied. [Kernel.AcquireInstanceFromDir] unifies its trailing
// [Source] values inside the package build and checks them against the
// module's `#config` at their own positions; [Kernel.SynthesizeInstance]
// does the same for [InstanceInput.Values], rendering them into the
// synthesized package; both then assert concreteness on the whole built
// spec. Render performs no validation pass of its own.
//
// # Configuration validation
//
// One primitive forms the validation surface: [Kernel.ValidateConfigDetailed]
// accepts an ordered slice of [Source], compiles each in the schema's own
// context, unifies in stack order, then validates the merged value against
// the schema with concreteness enforced. A single value is a one-element
// slice. A [Source] is CUE source bytes plus their origin and is bound to no
// context; per-source attribution flows through [token.Pos.Filename],
// populated from [cue.Filename](Origin) when the kernel compiles the source
// where it is used. Use [Kernel.LoadSourceFromFile] or
// [Kernel.LoadSourceFromBytes] to construct sources that are checked for
// syntax up front; a frontend needs no [cue.Context] of its own. There is
// no partial-mode entry: partial validation is an internal attribution pass
// under AcquireInstanceFromDir with extra values, not a public contract.
//
// Because the sources are compiled into the schema value's own context,
// validating against one acquired artifact from several goroutines at once
// shares that artifact's context; a consumer that needs that gives each
// goroutine its own acquired artifact. The kernel's own verbs never share a
// context this way.
//
// The primitive returns CUE-native errors. Walk them via
// [cuelang.org/go/cue/errors.Errors] / [cuelang.org/go/cue/errors.Positions],
// or print via [cuelang.org/go/cue/errors.Print]. Presentation belongs to
// the frontend — the kernel does not ship a formatter.
//
// A caller holding a *module.Module or *module.Instance composes its
// ConfigSchema() accessor with the primitive, e.g.
// k.ValidateConfigDetailed(m.ConfigSchema(), []kernel.Source{src}).
package kernel
