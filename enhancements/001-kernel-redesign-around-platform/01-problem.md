# Problem

The OPM kernel is intended to be the single reference runtime that every OPM implementation embeds — `cli`, `opm-operator`, the planned Crossplane composition function, and any future runtime. Today's shape is structured for a single embedding pattern (CLI / file-driven) and forces non-CLI consumers to fight the API rather than inherit it.

## What "kernel as reference runtime" requires

Three downstream embeddings, each with a different process model and a different idea of where artifacts come from:

| Frontend                  | Process model                  | Where artifacts come from                                  | What it actually wants from the kernel                                  |
| ------------------------- | ------------------------------ | ---------------------------------------------------------- | ----------------------------------------------------------------------- |
| `cli`                     | One-shot, terminal-bound       | Filesystem (`release.cue`, `module/`, `platform.cue`)      | Entire pipeline; phase-explicit subcommands (`vet`, `plan`, `apply`)    |
| `opm-operator`            | Long-running controller        | Kubernetes objects + CRD status; multi-tenant; concurrent  | Phase-explicit methods for admission, status, reconcile; goroutine-safe |
| Crossplane fn (planned)   | Short-lived gRPC handler       | Pre-resolved bundle / structured composition input         | Single-call `Compile`; no FS, no network, fast cold-start               |

A reference runtime must satisfy all three without each frontend reinventing matching, validation, or rendering. Today's kernel is structured around the CLI's needs.

## Why today's shape blocks two of the three

1. **Loader is filesystem-coupled.** `loader.LoadReleaseFile` resolves paths via `os.Stat`, walks parent directories for `cue.mod`, and reads `CUE_REGISTRY` from process environment. The Crossplane function has no filesystem and may receive artifacts as raw bytes or pre-built CUE values; the operator may receive them as Kubernetes object payloads. Both have to bypass the loader entirely or fake a filesystem to use it.

2. **`cue.Context` leaks into every public signature.** Every loader function, every render entry point takes `*cue.Context` as a parameter. Three frontends each have to construct, lifecycle, and reason about CUE plumbing — plumbing the kernel exists to encapsulate.

3. **Public API is a bag of free functions across packages.** `loader.LoadReleaseFile`, `module.ParseModuleRelease`, `render.ProcessModuleRelease`, `render.NewModule`, `render.Match`, `validate.Config` — each downstream binary independently composes the call sequence. Adding a cross-cutting dependency (logger, tracer) means changing every entry-point signature, breaking every consumer.

4. **Values input is a slice with implicit merge semantics.** `validate.Config(schema, values []cue.Value, ...)` and `module.ParseModuleRelease(_, spec, mod, values []cue.Value)` accept a slice and unify internally. The kernel bakes in one merge order; different frontends layer values differently (CLI: `-f` stack; operator: ConfigMap → Secret → CR overlay; XR: composition input). No frontend gets to express its policy cleanly.

5. **Provider concept is being retired.** Library enhancements 003 (`#Platform`) and 005 (`#Claim`) replace the flat `#Provider` artifact with a `#Platform` register that holds `#Module` registrations and computes `#composedTransformers` / `#matchers` from them. Today's `opm/render/match.go` walks a Provider's transformer list directly; that walk is structurally incompatible with the new design.

6. **Multi-version dispatch is hardcoded.** Every CUE path constant in `opm/render/match.go`, `opm/render/execute.go`, `opm/module/parse.go`, `opm/loader/*.go` — `metadata`, `components`, `#transformers`, `#component`, `#context`, `#config`, etc. — hardcodes v1alpha2 layout. Adding `v1alpha3` or supporting `v1alpha1` for compatibility requires touching every file. The `add-multi-apiversion-support` change addresses this; this enhancement assumes the binding interface lands first, then builds on it.

7. **Loader / render / module / provider / validate are not unified by a single entry-point type.** A consumer wants "give me a thing that is the kernel and lets me drive it." Today the consumer must learn the layout of the library, not the contract of a kernel.

## What this enhancement aims to fix

- A single `Kernel` type that owns its `cue.Context`, its DI (logger, tracer, options), and exposes phase-explicit methods (`Validate`, `Match`, `Plan`, `Compile`).
- A uniform artifact shape `(APIVersion, Metadata, Package cue.Value)` for `Module`, `ModuleRelease`, `Platform` — and any future artifact type — so the kernel input contract is one rule, not five.
- Loading, layering, and composition pushed into opt-in `opm/helper/...` packages. Frontends that have a filesystem use them; frontends that do not (XR fn) bypass them and feed the kernel directly.
- Two-tier validation: the helper package (Tier 1) produces source-positioned diagnostics for the human; the kernel (Tier 2) is a correctness safety net that never proceeds past invalid input.
- Single pre-unified values `cue.Value` at the kernel boundary; layering is helper / frontend policy.
- Match phase rewritten to consume `Platform.#composedTransformers` + `Platform.#matchers` from enhancement 003. `#Provider` retired in lockstep with the schema.
- `#ModuleDebug` retired entirely; `Module.debugValues` is the only debug surface, and the frontend decides whether to layer it into the values stack.

The `Render` method is renamed to `Compile` end-to-end. "Render" carries graphics-flavored connotations and undersells the lowering operation the kernel performs (declarative OPM model → platform-neutral resource values). "Compile" mirrors the source-to-target compiler analogy and pairs cleanly with the other phase verbs.

## What this enhancement explicitly does not address

- `#Claim` support in the kernel — deferred until enhancement 005 stabilizes. The matcher rewrite (slice 09) covers Resources/Traits only; Claims fold in later.
- `#Workflow` and `#Lifecycle` execution — kernel will host them eventually (per `CONSTITUTION.md`), but not in this enhancement. This enhancement sets the foundation; workflow/lifecycle slices come later.
- Cross-version migration tooling — out of scope; lives in CLI when it lands.
- Bundle format for offline / pre-resolved artifact distribution — useful for XR fn cold-start, but a separate concern from kernel shape and out of scope here.
