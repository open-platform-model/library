// Package kernel exposes the OPM runtime as a single struct, [Kernel].
//
// Kernel owns its [*cue.Context] and its [*schema.Cache] for its entire
// lifetime. Construction is [New] plus two options, [WithSchemaLoader]
// and [WithRegistry]; the kernel exposes no injection slot that no
// kernel operation reads. Downstream binaries (CLI, controller,
// Crossplane function) construct one Kernel per goroutine and call
// methods on it instead of importing the individual loader / module /
// validate packages.
//
// # Goroutine safety
//
// A single Kernel is NOT safe for concurrent use across its own method calls.
// The owned [*cue.Context] (acquisition, synthesis and validation build in
// it) is driven single-threaded; sharing one Kernel between goroutines can
// cause data races inside CUE evaluation. Callers that need concurrency MUST
// construct one Kernel per goroutine.
//
// [Kernel.Render] shares nothing between renders (ADR-005, enhancement 0019
// D8). Each render is its own CUE build in a fresh cue.Context created for
// that call and dropped when Render returns; the Kernel's own context is not
// used, no built value is retained between calls, and a caller cannot obtain
// one to hold. Concurrency is across renders, never within one: a consumer
// rendering from several goroutines gives each goroutine its own Kernel and
// calls Render, with no shared platform value and no mutex. There is no
// materialized platform to share and no serialised render path; the earlier
// shared-platform contract (ADR-002) is superseded, not supported.
//
// A render is single-threaded and its working set grows with the module, so
// a render pool is sized by memory rather than by core count: about 61 MB
// plus 7.75 MB per component per concurrent render (0019 experiment 08), and
// throughput saturates at roughly physical cores divided by 1.6 renders in
// flight. Size against the largest module the pool will see.
//
// # One-Kernel-per-goroutine example
//
//	func renderAll(ctx context.Context, platformDir string, instanceDirs []string) error {
//	    var wg sync.WaitGroup
//	    errs := make(chan error, len(instanceDirs))
//	    for _, dir := range instanceDirs {
//	        wg.Add(1)
//	        go func(dir string) {
//	            defer wg.Done()
//	            k := kernel.New() // one Kernel per goroutine
//	            plat, err := k.AcquirePlatformFromDir(ctx, platformDir, loaderfile.LoadOptions{})
//	            if err != nil {
//	                errs <- err
//	                return
//	            }
//	            inst, err := k.AcquireInstanceFromDir(ctx, dir, loaderfile.LoadOptions{})
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
// both, builds it once, and decodes the matching verdicts
// ([RenderDiagnostics]) and the rendered output ([RenderResult.Compiled],
// one entry per rendered object with instance, component and transformer
// provenance). Matching and transformer execution are CUE inside the build,
// not Go; the build reports its verdicts as data and the kernel's fail-closed
// gate turns an unresolved demand, an unmatched component or an
// over-subscribed provider-fulfilled contract into a [*RenderError] that
// carries the full diagnostics, with the typed causes reachable through
// errors.As. Catalog version skew (the instance module requiring a newer
// OPM-namespace build than the platform carries) is warned by default
// ([SkewWarn]) or refused before evaluation ([SkewRefuse]).
//
// A dry run is Render with the output discarded: the build evaluates every
// matched pair regardless, and RenderDiagnostics carries the pairing
// diagnosis (Pairs, Unmatched, Unresolved, Unify, UnhandledTraits,
// OverSubscribed, ResolvedVersions). There is no separate match verb.
//
// Render consumes the instance as processed: values are validated where
// they are applied. [Kernel.AcquireInstanceFromDir] unifies [WithValues]
// sources inside the package build and checks them against the module's
// `#config` at their own positions; [Kernel.SynthesizeInstance] renders
// in.Values into the synthesized package; both then assert concreteness on
// the whole built spec. Render performs no validation pass of its own.
//
// # Configuration validation
//
// One primitive forms the validation surface: [Kernel.ValidateConfigDetailed]
// accepts an ordered slice of [Source], unifies in stack order, then
// validates the merged value against a schema with concreteness enforced. A
// single value is a one-element slice. Per-source attribution flows through
// [token.Pos.Filename] populated from [cue.Filename](Origin) at compile time;
// use [Kernel.LoadSourceFromFile] or [Kernel.LoadSourceFromBytes] to construct
// sources whose Value satisfies the filename contract automatically. There is
// no partial-mode entry: partial validation is an internal attribution pass
// under AcquireInstanceFromDir with extra values, not a public contract.
//
// The primitive returns CUE-native errors. Walk them via
// [cuelang.org/go/cue/errors.Errors] / [cuelang.org/go/cue/errors.Positions],
// or print via [cuelang.org/go/cue/errors.Print]. Presentation belongs to
// the frontend — the kernel does not ship a formatter.
//
// A caller holding a *module.Module or *module.Instance composes its
// ConfigSchema() accessor with the primitive, e.g.
// k.ValidateConfigDetailed(m.ConfigSchema(), []kernel.Source{src}).
//
// # Advanced: CueContext accessor
//
// [Kernel.CueContext] returns the underlying [*cue.Context] for callers that
// need to build [cue.Value]s outside the kernel (typically tests). Values
// built with this context are safe to pass back into Kernel methods. Most
// callers should not need this. Render never uses it: the render build has
// its own context.
package kernel
