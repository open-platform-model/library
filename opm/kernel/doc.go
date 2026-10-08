// Package kernel exposes the OPM runtime as a single struct, [Kernel].
//
// Kernel owns its [*schema.Cache] for its entire lifetime and no build
// context: every operation that evaluates CUE creates its own [cue.Context],
// builds in it, and lets it go when it returns. Construction is [New] plus two
// options, [WithSchemaLoader] and [WithRegistry]; the kernel exposes no
// injection slot that no kernel operation reads. Downstream binaries (CLI,
// controller, Crossplane function) construct one Kernel per process and call
// methods on it instead of importing the individual loader / module / validate
// packages.
//
// # Surface
//
// One tier: every artifact a frontend can hold comes from an acquire verb, or
// from the package constructor it already has a value for.
//
//   - [Kernel.AcquireModuleFromRegistry] and [Kernel.AcquireModuleFromDir]
//     return a source-carrying [*module.Module].
//   - [Kernel.AcquireCatalogFromRegistry] and [Kernel.AcquireCatalogFromDir]
//     return a source-carrying [*catalog.Catalog], an acquired kind of its
//     own: the kernel reads it and derives from it
//     ([catalog.Catalog.Provides], [catalog.Catalog.Requires]) and judges
//     nothing beyond its shape.
//   - [Kernel.AcquirePlatformFromDir] returns a [*platform.Platform].
//   - [Kernel.AcquireInstanceFromDir] returns a validated [*module.Instance],
//     with optional values as trailing [Source] values.
//   - [Kernel.SynthesizeInstance] builds one from typed inputs
//     ([InstanceInput]); the module it takes comes from the two module
//     acquire verbs, and the synthesized package imports core at the major
//     of the kernel's schema release, read from the configured
//     [schema.OCILoader] with no schema load when it pins an exact release
//     (the default) and resolved through the schema cache otherwise; the
//     release that import resolves to is the one the module's own
//     cue.mod/module.cue pins.
//   - [Kernel.ValidateConfigDetailed] validates layered values.
//   - [Kernel.Render] renders an instance against a platform.
//
// There is no second, value-only tier: a caller that wants the raw value of an
// acquired artifact reads its Package field, and a caller holding a value it
// built itself calls [module.NewModuleFromValue] or
// [platform.NewPlatformFromValue] directly. The registry mapping is
// [WithRegistry] for every one of these operations, the schema cache and the
// compilation of file-backed values sources (a values file that imports a
// registry module) included; no verb takes a per-call override. Absent the
// option, every operation inherits the process CUE_REGISTRY and applies no
// default; the mapping is plumbed into the operation's load configuration and
// never written back to the environment. Every operation resolves through the
// Kernel's one registry client, under a module cache of that operation's own,
// so no fetch failure outlives the operation that saw it (see [Kernel]).
//
// # Every operation shares nothing
//
// The Kernel holds no [cue.Context]. Each acquire verb, synthesis,
// validation and render creates a context for the call, builds in it, and
// returns; the values an operation returns (an artifact's Package, a validated
// value) keep that operation's runtime alive for exactly as long as the caller
// holds them, and the Kernel retains nothing. Memory held by a long-lived
// Kernel is therefore bounded by the artifacts its caller holds, not by the
// number of operations it has run. The cross-artifact verbs read only Metadata
// and Source from their inputs, and no Package: Render's core floor (whether
// the platform's #contracts carries providedBy) reads the fact the platform
// recorded at construction, through platform.Platform.CoreFloor. That holds
// for a platform the constructor built; a Platform built as a struct literal
// decodes its Package once, on its first CoreFloor or Contracts call (see
// platform.Platform). Nothing is
// built into an input's context, so a module acquired by one Kernel
// synthesizes on another and an instance from either renders on a third, and
// one acquired platform may be shared by concurrent renders. No method returns
// or accepts a [*cue.Context]; a caller that must compile a value against the
// schema takes the context of the value [schema.Cache.Get] returns.
//
// # Goroutine safety
//
// A single Kernel is safe for concurrent use across its own method calls: no
// operation shares evaluation state with another, and the schema cache is
// memoized under synchronization into a private context of its own. A consumer
// that needs concurrent operations shares one Kernel across its goroutines;
// there is nothing to gain from constructing more than one. Concurrency is
// across operations, never within one.
//
// [Kernel.Render] shares nothing between renders (0019:D8). Each
// render is its own CUE build in a fresh cue.Context created for that call;
// the kernel drops its own references to it when Render returns and retains
// no built value between calls. A [Compiled] the caller holds keeps its
// render's build alive until released. A consumer rendering from several
// goroutines calls Render on one Kernel with no mutex, and may share one
// acquired platform across them: each render builds the platform from its
// Source in its own context and reads no Package (the core floor reads the
// fact recorded at construction; a struct-literal Platform decodes its
// Package once, on its first CoreFloor or Contracts call, under a sync.Once),
// so concurrent renders never write to the shared platform. No render reuses a platform
// value another render built, and there is no serialised render path; the
// earlier shared-platform contract (renders filling one shared platform
// value) is superseded, not supported.
//
// A render is single-threaded and its working set grows with the module, so a
// render pool is sized by memory rather than by core count: about 61 MB plus
// 7.75 MB per component per concurrent render (0019 experiment 08), and
// throughput saturates at roughly physical cores divided by 1.6 renders in
// flight. Size against the largest module the pool will see.
//
// # Cancellation
//
// Every verb that takes a context checks it between its stages: after the
// directory is read, after the values sources are merged, after a registry
// fetch returns, after the package or render module is built, and before the
// next stage starts. The directory verbs and [Kernel.SynthesizeInstance] also
// check it at entry, and [Kernel.Render] once its input is checked; argument
// errors come first, because they do not depend on time. When
// one of these checks finds the context done, the verb returns the context's
// own error unwrapped (so errors.Is(err, context.Canceled) or
// context.DeadlineExceeded holds) and no artifact. A running load or build is
// not interrupted: cue/load takes no context, so cancellation lands at the
// next stage boundary. Only the registry fetch itself observes the context
// while it runs; a cancellation it sees comes back wrapped in the fetch
// error, and errors.Is(err, context.Canceled) still holds (0009:D9).
//
// # One-Kernel-per-process example
//
//	func renderAll(ctx context.Context, k *kernel.Kernel, platformDir string, instanceDirs []string) error {
//		plat, err := k.AcquirePlatformFromDir(ctx, platformDir) // once; the platform is shared as data
//		if err != nil {
//			return err
//		}
//		var wg sync.WaitGroup
//		errs := make(chan error, len(instanceDirs))
//		for _, dir := range instanceDirs {
//			wg.Add(1)
//			go func(dir string) {
//				defer wg.Done()
//				inst, err := k.AcquireInstanceFromDir(ctx, dir) // the one Kernel, concurrently
//				if err != nil {
//					errs <- err
//					return
//				}
//				if _, err := k.Render(ctx, kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "opm-cli"}); err != nil {
//					errs <- err
//				}
//			}(dir)
//		}
//		wg.Wait()
//		close(errs)
//		for err := range errs {
//			if err != nil {
//				return err
//			}
//		}
//		return nil
//	}
//
// # Rendering
//
// [Kernel.Render] is the kernel's single render verb. It takes a
// source-carrying instance ([Kernel.AcquireInstanceFromDir] or
// [Kernel.SynthesizeInstance]) and a source-carrying platform
// ([Kernel.AcquirePlatformFromDir]: a platform is a CUE module on disk that
// imports its catalogs), stages one generated render module that imports both
// (an on-disk input in place, an overlay-mode input and the generated module
// itself served from memory, so a render writes no staging file), builds it
// once, and decodes the matching verdicts ([RenderDiagnostics]) and the
// rendered output ([RenderResult.Compiled], one entry per rendered object as a
// [*Compiled] carrying instance, component and transformer provenance).
// Matching and transformer execution are CUE inside the build, not Go; the
// build reports its verdicts as data and the kernel's fail-closed gate turns
// them into a [*RenderError] that carries the full diagnostics, with the typed
// causes reachable through errors.As and joined in this order: a contract
// collision (a key more than one enabled registry entry defines, read from
// core's #contracts.collisions and collidingEntries; the opm/errors
// ContractCollisionsError, first because the other rows are read against the
// inventory a collision distorts, platform-wide and standing under the skip
// switch), an unresolved demand, an over-subscribed provider-fulfilled
// contract (read from core's #contracts.overSubscribed and providedBy), an
// unmatched component, and last the NotRoutableError catch-all, raised only
// when core's decoded #contracts.routable reads false and no collision or
// over-subscription row explains it. The rows are the ones the platform's
// Contracts() reads, so the render refuses on a collision or an
// over-subscription exactly when the inventory reads not routable. Catalog
// version skew (the instance module requiring a newer OPM-namespace build than
// the platform carries) marks a resolved-versions row Newer by default
// ([SkewWarn]) or refuses before evaluation ([SkewRefuse]).
//
// The render module's own gate field agrees with the kernel: it errors exactly
// when the kernel refuses on a decoded verdict, so a staged module is
// self-refusing under a plain cue eval. The kernel decides from the decoded
// rows and the decoded routable verdict only, never by reading the gate. Each
// typed cause carries its diagnostics rows unchanged and wraps nothing. Inputs
// are never mutated, and nothing is staged on disk, success or failure;
// refusals before evaluation (a missing Source, a platform whose
// core predates #contracts.providedBy, an uncovered OPM-namespace path, skew
// under [SkewRefuse], a local replacement without the opt-in) are plain
// errors. The core floor runs before anything is staged: a platform module
// pinning core older than [schema.ProvidedBySince] is refused with an error
// wrapping the opm/errors PlatformCoreTooOldError that
// platform.Platform.CoreFloor returns, read from the fact the platform
// recorded at construction, and the render never falls back to a provider
// count of its own.
//
// A render result carries no presentation strings. The three advisory facts a
// render can report are rows on the diagnostics: an unhandled optional trait
// on RenderDiagnostics.UnhandledTraits, a module requiring a newer build than
// the platform carries on a RenderDiagnostics.ResolvedVersions row with Newer
// set, and a demand skipped under [RenderInput.SkipUnprovided] on a
// RenderDiagnostics.Skipped row. A frontend words all three:
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
//	for _, s := range result.Diagnostics.Skipped {
//		log.Printf("component %q: skipped %s %q, no provider on this platform (component rendered: %t)",
//			s.Component, s.Kind, s.FQN, !s.ComponentOmitted)
//	}
//
// A dry run is Render with the output discarded: the build evaluates every
// matched pair regardless, and RenderDiagnostics carries the pairing diagnosis
// (Pairs, Unmatched, Unresolved, Skipped, Unify, UnhandledTraits,
// OverSubscribed, Collisions, Routable, ResolvedVersions). There is no
// separate match verb. Beside the pairing diagnosis, RequiredContracts is
// the instance's contract demand (0013:D24): every #resources and #traits key
// of every component, omitted ones included, set on every RenderResult and
// every [*RenderError].
//
// A demand is unprovided when its contract declares fulfilment "provider" and
// no enabled registry entry carries a transformer requiring the key: the key
// is absent from core's #contracts.providedBy, the count the single-provider
// guard reads. The build marks every unresolved row with this fact on every
// render (UnresolvedDemand.Unprovided in opm/errors, and a
// "provider-fulfilled, no provider on this platform" suffix on its message),
// so a frontend can offer its skip switch without re-deriving fulfilment.
// Under [RenderInput.SkipUnprovided] the build moves exactly those demands out
// of the refusal: a skipped trait demand leaves its component rendering every
// pair it matched, and a skipped resource demand omits the whole component (no
// pair of it renders, it is not reported unmatched, and every skipped row of
// it carries ComponentOmitted). Every other refusal stands under the switch: a
// catalog-fulfilled unresolved demand, a provider that exists but did not
// match, a contract collision, an over-subscribed contract and an unmatched
// component. The switch is the caller's, per render; the kernel never sets it,
// and a frontend names its own flag.
//
// An input's own cue.mod/local-module.cue (a developer redirecting a
// dependency to a directory or another module) reaches the render only under
// [RenderInput.LocalReplacements]. Off, the default, Render refuses before
// staging an input whose file carries a replacement rather than silently
// rendering against the published pin. On, the replacements are promoted into
// the render module under the precedence dependencies get (the platform's
// whole, the instance's only for paths the platform's dependency list does not
// name) and each honoured one is a [RenderDiagnostics.Replacements] row naming
// the path, the target and the input that supplied it; a replaced path keeps
// its pinned versions on ResolvedVersions. A relative directory target is
// resolved against the input's own module root; a replaced path the promoted
// list lacks is listed with a placeholder version of its major so coverage
// holds; a version-less dependency no promoted replacement covers is refused
// naming the path and the input; the rows are path-sorted, and an input
// without the file renders identically under either setting. The switch is a
// security boundary, not an extension point: it is the one place a render may
// read a directory an artifact names, so a frontend sets it for a developer's
// checkout and never for an artifact it did not author (the operator never
// sets it). The frontend words the rows (an instance replacement the platform
// made inert is not a row, so the frontend computes the inert set from the
// file it read):
//
//	for _, r := range result.Diagnostics.Replacements {
//		log.Printf("%s: served from %s (%s local-module.cue)", r.Path, r.Target, r.By)
//	}
//
// Render consumes the instance as processed: values are validated where they
// are applied. [Kernel.AcquireInstanceFromDir] unifies its trailing [Source]
// values inside the package build and checks them, together with the package's
// own `values`, against the module's `#config` at their own positions, on
// every acquire; [Kernel.SynthesizeInstance] does the same for
// [InstanceInput.Values], rendering them into the synthesized package; both
// then assert concreteness on the whole built spec and on the instance's
// values unified with `#config`, so both refuse a required `#config` value the
// values leave unset, whether or not a component reads it. Render performs no
// validation pass of its own.
//
// # Configuration validation
//
// One primitive forms the validation surface: [Kernel.ValidateConfigDetailed]
// accepts an ordered slice of [Source], compiles each in the schema's own
// context, unifies in stack order, then validates the merged value against the
// schema with concreteness enforced. A single value is a one-element slice. A
// [Source] is CUE source bytes plus their origin and is bound to no context;
// per-source attribution flows through [token.Pos.Filename], populated from
// [cue.Filename](Origin) when the kernel compiles the source where it is used.
// Use [Kernel.LoadSourceFromFile] or [Kernel.LoadSourceFromBytes] to construct
// sources that are checked for syntax up front; a frontend needs no
// [cue.Context] of its own. A file-backed source is loaded at its file's
// directory, so its imports resolve through the kernel's [WithRegistry]
// mapping on every path that compiles it. There is no partial-mode entry:
// partial validation is an internal attribution pass under
// AcquireInstanceFromDir with extra values, not a public contract.
//
// Because the sources are compiled into the schema value's own context,
// validating against one acquired artifact from several goroutines at once
// shares that artifact's context; a consumer that needs that gives each
// goroutine its own acquired artifact. The kernel's own verbs never share a
// context this way.
//
// The primitive returns CUE-native errors, marked as a validation failure by
// the *ConfigValidationError of opm/errors (see ValidateConfigDetailed for
// what is marked). Walk them via
// [cuelang.org/go/cue/errors.Errors] / [cuelang.org/go/cue/errors.Positions],
// or print via [cuelang.org/go/cue/errors.Print]. Presentation belongs to the
// frontend — the kernel does not ship a formatter.
//
// A caller holding a *module.Module or *module.Instance composes its
// ConfigSchema() accessor with the primitive, e.g.
// k.ValidateConfigDetailed(m.ConfigSchema(), []kernel.Source{src}).
package kernel

// Design records behind the package doc above, for maintainers: the catalog
// as an acquired kind is ADR-009; the Kernel holding no cue.Context is
// ADR-007; renders sharing nothing is ADR-005, which supersedes ADR-002's
// shared-platform contract.
