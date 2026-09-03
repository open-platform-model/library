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
// it) is driven single-threaded — sharing one Kernel between goroutines can
// cause data races inside CUE evaluation. Callers that need concurrency MUST
// construct one Kernel per goroutine.
//
// [Kernel.Render] shares nothing between renders (enhancement 0019 D8): each
// render is its own CUE build in a fresh cue.Context that is released when
// Render returns, the Kernel's own context is not used, and no built value is
// retained between calls. Concurrency is across renders, never within one: a
// consumer rendering from several goroutines gives each goroutine its own
// Kernel and calls Render, with no shared platform value and no mutex. There
// is no materialized platform to share and no serialised render path; the
// earlier shared-platform contract is retracted (ADR-002).
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
// imports its catalogs), stages one render module that imports both, builds
// it once, and decodes the matching verdicts ([RenderDiagnostics]) and the
// rendered output ([RenderResult.Compiled], one entry per rendered object
// with instance, component and transformer provenance). A dry run is Render
// with the output discarded: the build evaluates every matched pair
// regardless, and RenderDiagnostics carries the pairing diagnosis (Pairs,
// Unmatched, Unresolved, Unify, UnhandledTraits).
//
// Render consumes the instance as processed: [Kernel.ProcessModuleInstance]
// is the validated entry point — it validates user values against the
// module's `#config` schema (via [Kernel.ValidateConfig]) and fills them
// before the instance is rendered — and Render performs no validation pass
// of its own.
//
// # Configuration validation
//
// Three primitives form the validation surface:
//
//   - [Kernel.ValidateConfig] — concrete check on a single, pre-merged
//     [cue.Value]. Returns the unified value and a CUE-native error.
//   - [Kernel.ValidateConfigPartial] — same, without the concreteness
//     requirement. Used by lint subcommands, IDE/LSP, admission webhooks,
//     and other callsites that intentionally validate a draft.
//   - [Kernel.ValidateConfigDetailed] — accepts an ordered slice of
//     [Source], unifies in stack order, then validates the merged value.
//     Per-source attribution flows through [token.Pos.Filename] populated
//     from [cue.Filename](Origin) at compile time. Use
//     [Kernel.LoadSourceFromFile], [Kernel.LoadSourceFromBytes], or
//     [Kernel.LoadSourceFromString] to construct sources whose Value
//     satisfies the filename contract automatically.
//
// All three return CUE-native errors. Walk them via
// [cuelang.org/go/cue/errors.Errors] / [cuelang.org/go/cue/errors.Positions],
// or print via [cuelang.org/go/cue/errors.Print]. Presentation belongs to
// the frontend — the kernel does not ship a formatter.
//
// A caller holding a *module.Module or *module.Instance composes its
// ConfigSchema() accessor with the primitive it wants, e.g.
// k.ValidateConfig(m.ConfigSchema(), values).
//
// # Advanced: CueContext accessor
//
// [Kernel.CueContext] returns the underlying [*cue.Context] for callers that
// need to build [cue.Value]s outside the kernel (typically tests). Values
// built with this context are safe to pass back into Kernel methods. Most
// callers should not need this.
package kernel
