# ADR-006: Artifacts are constructed in one CUE build

## Status

Accepted (2026-09-03). Supersedes ADR-003. Records enhancement 0019 D9 (workspace root, `enhancements/0019/03-decisions.md`: the render step is one CUE build per render) and its application to instance synthesis. Implemented by `library-render-build` (`Kernel.Render`) and `library-render-cutover` (`Render` as the sole render path; `opm/materialize` and `opm/compile` deleted). ADR-005 records the lifetime and concurrency rules of that build.

## Context

OPM builds composite CUE artifacts from independently authored inputs: an instance imports a module; a render combines an instance, a platform and the catalogs the platform carries. The natural Go-API tactic is `FillPath` / `Unify` across builds.

Filling a value into a closed, independently built value corrupts lazy resolution of output-local hidden fields: a transformer's `#transform` reading a hidden field declared inside `output` evaluates to `non-concrete value _`. This is a CUE Go-API closedness bug, not a language or schema defect (`docs/design/transformer-output-hidden-field-scope-bug.md` §11 and §12), it is unfixed upstream, `cue vet` cannot catch it, and the corruption surfaces only at marshal time.

ADR-003 stated the principle, never `FillPath` into a closed independently built value, and admitted two tactics: single-build for instance synthesis and federation for the materialized platform. It explicitly rejected a single-build-only framing on one premise: a platform had to compose several versions of one catalog `path@major`, which minimum version selection cannot hold in one build. That premise was then removed in two steps. Enhancement 0010 D14 made each subscription name exactly one build, and 0019 D5 made the platform a CUE module that imports its catalogs, with core deriving each entry's version and the platform's `#composedTransformers`. What remained of federation was a self-described defensive path with nothing left to defend.

The render path meanwhile still crossed a build boundary: it filled `#moduleInstance`, `#component` and `#context` into a `#transform` reached through the materialized platform. That fill stripped the component's definitions and hidden fields on the way in (the parity gap 0019 D1 names) and was the write to shared evaluation state that ADR-002 mistook for a read (ADR-005).

Enhancement 0019 measured the alternative before adopting it (`enhancements/0019/experiments/`): a pure-CUE render flow over the real catalog renders correctly (01); `cue.mod/local-module.cue` directory replacements bring unpublished inputs into a build (02); matching expressed inside the build reaches the same pair set, and every embedded copy of a catalog resolves to the same bytes (05); a single build costs 90 ms per render against 42 ms for the held-platform baseline on a five-pair fixture (04), and sequentially the crossover against the old path sits between 4.6 and 14.2 components (07), with platform reuse worth at most about 85 ms.

## Decision

**Every artifact the kernel constructs is one CUE build that imports its inputs.** No value is filled across a build boundary into a closed value, and nothing is unified in Go across builds.

- **Instance synthesis** (`synth.Instance`, `Kernel.SynthesizeInstance`) stages a virtual package: a `cue.mod` naming the resolved core version and the module, an `instance.cue` that imports the module, a `values.cue` rendered from the caller's values. It evaluates that package once; the values merge happens in CUE through the schema's own `#module & {#config: values}`.
- **Render** (`Kernel.Render`) stages a generated render module: a `cue.mod` promoted from both inputs' module files (0019 D13), directory replacements importing the instance package and the platform package, and the embedded glue. It evaluates that module once. `#moduleInstance`, `#component` and `#context` reach each transformer by unification inside the build; the demand walk, the label predicate, the always-unify rung, the fail-closed gate and the single-provider guard are CUE comprehensions whose results are data; the kernel decodes `diagnostics` and `rendered` off the built value.
- **Inputs are packages, not values.** A platform is a CUE module on disk that imports its catalogs (0019 D5), an instance carries a `Source` naming its package, and a bare `cue.Value` is not a render input. The shape gate refuses a platform entry that embeds no catalog at acquisition.

Rejected alternatives:

- **Federation** (ADR-003's second tactic): its premise is gone, it kept two mechanisms for one principle, and the Go-built index beside a closed twin was exactly the surface the closedness bug needed.
- **`FillPath` in the caller Kernel's context** (the half of ADR-002 that landed): still a cross-build fill into a closed value, still stripping the component, and still the write that races under sharing.
- **Build the platform once and reuse it across renders**: measured retention grows with the module and render count (ADR-005); the single build makes dropping the context per render affordable, and the reuse it forgoes is worth tens of milliseconds.

## Consequences

**Positive:** Correctness by construction. There is no surface from which a corrupt `#transform` read is possible, so the regression guards that asserted the closed surface still corrupts went with the surface. Parity with a pure-CUE evaluation of the same inputs (0019 D1) stops being a property the kernel maintains and becomes one it cannot violate; the oracle harness (`testdata/parity`, `render-parity`) checks it structurally. Both artifacts are built by one mechanism, so a render bug surfaces in both paths or neither. Matching verdicts are data the build reports, not Go control flow.

**Negative:** Every render re-resolves and re-evaluates the platform and its catalogs: about 2.1x the old per-render cost on a small fixture, with the sequential crossover between five and fourteen components. Staging touches disk (a temporary directory per render, removed on return). The kernel depends on `cue/load` directory replacements, which sets the declared `language.version` floor at `v0.17.0`.

**Trade-off:** The upstream closedness bug remains unfixed, and `opm/internal/cueregression/closedness_test.go` is the canary that flips when it is. With no cross-build fill left, a fix upstream changes nothing here; the tactic is belt-and-braces from that day, not before. The lifetime and concurrency of the build (own context per render, dropped on return, pool sized by memory) are ADR-005's rules.
