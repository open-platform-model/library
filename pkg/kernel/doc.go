// Package kernel exposes the OPM runtime as a single struct, [Kernel].
//
// Kernel owns its [*cue.Context] for its entire lifetime and threads
// cross-cutting dependencies (logger, tracer, clock) through every operation.
// Downstream binaries (CLI, controller, Crossplane function) construct one
// Kernel per goroutine and call methods on it instead of importing the
// individual loader / module / compile / validate packages.
//
// # Goroutine safety
//
// Kernel is NOT safe for concurrent use across method calls. The owned
// [*cue.Context] is single-threaded — sharing one Kernel between goroutines
// can cause data races inside CUE evaluation. Callers that need concurrency
// MUST construct one Kernel per goroutine.
//
// # One-Kernel-per-goroutine example
//
//	func renderAll(ctx context.Context, paths []string) error {
//	    var wg sync.WaitGroup
//	    errs := make(chan error, len(paths))
//	    for _, p := range paths {
//	        wg.Add(1)
//	        go func(path string) {
//	            defer wg.Done()
//	            k := kernel.New() // one Kernel per goroutine
//	            if _, _, err := k.LoadModulePackage(ctx, path); err != nil {
//	                errs <- err
//	            }
//	        }(p)
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
// # Phase methods
//
// The kernel exposes four phase-explicit methods that mirror the OPM
// pipeline. Each accepts a phase-specific input struct and returns a
// phase-appropriate result:
//
//   - [Kernel.Validate] — Tier-2 schema validation of values against
//     the module's `#config`. Returns nil or an error wrapped with
//     `module %q:` framing whose underlying tree is walkable as
//     [cuelang.org/go/cue/errors.Error].
//   - [Kernel.Match] — component / transformer pairing. Returns
//     [*MatchPlan] without executing any transformer.
//   - [Kernel.Plan] — Validate + Match + summaries. Returns
//     [*PlanResult]; does NOT produce rendered values. This is the
//     verb every frontend's "plan" / "preview" subcommand wants.
//   - [Kernel.Compile] — full pipeline (Validate + Match + Execute +
//     Finalize). Returns [*CompileResult] containing rendered values
//     plus provenance. This is the terminal output and the verb every
//     frontend's "apply" / "render" subcommand wants.
//
// CLI subcommands map naturally onto these methods (vet → Validate,
// match → Match, plan → Plan, apply → Compile).
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
// Typed convenience methods on the kernel resolve `#config` for the
// caller: [Kernel.ValidateModuleValues] / [Kernel.ValidateReleaseValues]
// (plus their `Partial` and `Detailed` counterparts) take a *module.Module
// or *module.Release and delegate to the corresponding primitive.
//
// # Advanced: CueContext accessor
//
// [Kernel.CueContext] returns the underlying [*cue.Context] for callers that
// need to build [cue.Value]s outside the kernel (typically tests). Values
// built with this context are safe to pass back into Kernel methods. Most
// callers should not need this.
package kernel
