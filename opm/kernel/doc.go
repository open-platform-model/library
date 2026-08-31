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
// A single Kernel is NOT safe for concurrent use across its own method calls.
// The owned [*cue.Context] is driven single-threaded — sharing one Kernel
// between goroutines can cause data races inside CUE evaluation. Callers that
// need concurrency MUST construct one Kernel per goroutine.
//
// A [*materialize.MaterializedPlatform] is owned by the Kernel that built it
// and is NOT safe to render against from several goroutines at once, whether
// through one Kernel or many. [Kernel.Compile] fills each transformer's
// #transform (FillPath of #moduleInstance, #component and #context), and
// filling a value is a write to its evaluation state, not a read. Measured
// against the real catalog (enhancement 0019, experiment 06): 2321
// race-detector reports rendering concurrently against one shared platform,
// 1540 with the platform fully pre-evaluated first. No wrong output was
// observed; the behaviour is undefined. The earlier "materialize once, render
// concurrently, no mutex" contract is retracted, and ADR-002 records the
// supersession.
//
// Until the shares-nothing render model of enhancement 0019 lands (D8: one
// CUE build per render, in a context that does not outlive the render), a
// consumer that renders from several goroutines MUST either serialize every
// use of a materialized platform behind one mutex, or give each goroutine
// its own Kernel and its own [Kernel.Materialize] call.
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
//	            if _, _, err := k.LoadModulePackage(ctx, path, loaderfile.LoadOptions{}); err != nil {
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
// # Rendering from several goroutines
//
// Serialize the render path, or materialize per goroutine. The mutex form is
// the cheaper stopgap while a platform is expensive to materialize; it holds
// the Kernel that built the platform and the platform itself behind one lock:
//
//	var renderMu sync.Mutex // guards k0 and shared together
//
//	func renderOne(ctx context.Context, k0 *kernel.Kernel, shared *materialize.MaterializedPlatform, inst *module.Instance) error {
//	    renderMu.Lock()
//	    defer renderMu.Unlock()
//	    _, err := k0.Compile(ctx, kernel.CompileInput{
//	        ModuleInstance: inst,
//	        Platform:       shared,
//	        RuntimeName:    "opm-operator",
//	    })
//	    return err
//	}
//
// The per-goroutine form costs one Materialize (registry I/O) per goroutine
// and shares nothing, which is the model enhancement 0019 D8 makes the only
// one.
//
// # Phase methods
//
// The kernel exposes two phase-explicit methods that mirror the OPM
// pipeline. Each accepts a phase-specific input struct and returns a
// phase-appropriate result:
//
//   - [Kernel.Match] — component / transformer pairing. Returns
//     [*MatchPlan] without executing any transformer.
//   - [Kernel.Compile] — full pipeline (Match + Execute). Returns
//     [*CompileResult] containing rendered values plus provenance.
//     This is the terminal output and the verb every frontend's
//     "apply" / "render" subcommand wants.
//
// Both phases consume the instance as processed:
// [Kernel.ProcessModuleInstance] is the validated entry point — it
// validates user values against the module's `#config` schema (via
// [Kernel.ValidateConfig]) and fills them before either phase runs.
// A caller wanting a dry run calls Match for the pairing diagnosis, or
// Compile and discards the rendered slice.
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
