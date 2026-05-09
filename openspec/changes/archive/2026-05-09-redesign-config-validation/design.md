## Context

The library's configuration validation has grown three layers thick over successive slices. `pkg/kernel/validate.go` holds the Tier-2 primitive (`runValidate` + `walkDisallowed` + `appendSchemaErrors`); `pkg/helper/values/` adds Tier-1 per-layer validation with source attribution (`Layer`/`Stack`/`MultiSourceError`); `pkg/errors/` defines a custom Go-typed error language (`ConfigError`, `ValidationError`, `FieldError`, `ErrorLocation`, `GroupedError`) that translates CUE diagnostics into projection structs with `groupCUEErrors` doing message+path collation. Each layer was added to solve a real problem — but together they form a Go-typed dialect of information CUE's `cuelang.org/go/cue/errors` package already exposes natively (`Error.Path()`, `Position()`, `InputPositions()`, `Msg()`, `errors.Errors`, `errors.Print`, `errors.Sanitize`).

The CLI has shipped without leveraging any of the typed projection. opm-operator and future XR composition functions face the same situation: they import the library, get back `*ConfigError` / `*MultiSourceError`, and then re-translate to whatever shape their consumer needs (CLI prose, `metav1.Condition`, XR composition status). The Go-typed error surface increases the library's public API without buying clarity.

A second symptom: per-source attribution today routes through library-defined types (`LayerError{LayerName, Source, *ConfigError}`). CUE already carries source identity via `token.Pos.Filename()`, populated from `cue.Filename(...)` at compile time and from `cue/load.Instances` automatically. The library could lean on that mechanism and reach the same outcome with no custom types — but only if loading discipline is enforced: callers MUST set `cue.Filename(Origin)` before passing a value to validation.

The redesign replaces all of this with a CUE-native surface. Three primitive functions on `*Kernel`. Eight typed convenience methods on `*Module`/`*Release`. Three loader helpers that bake `cue.Filename(Origin)` into compilation. Errors flow as `cuelang.org/go/cue/errors.Error`. Frontends use `cueerrors.Errors`/`Positions`/`Print` for traversal and rendering. The library owns one print helper that strips schema-internal prefixes for display ergonomics.

## Goals / Non-Goals

**Goals:**

- Reduce the library's validation public surface to: 3 kernel primitives, 6 typed convenience methods (3 each on the kernel for Module/Release), 3 source loaders, 1 `Source` struct, 1 `ValidateOption`/`Partial()` knob. The library does NOT ship a print helper — presentation belongs to the frontend.
- Eliminate every Go-typed validation error (`ConfigError`, `ValidationError`, `FieldError`, `ErrorLocation`, `GroupedError`, `MultiSourceError`, `LayerError`).
- Return CUE-native errors (`cuelang.org/go/cue/errors.Error`) from every validation function so frontends use the same error vocabulary they would use for any other CUE evaluation.
- Preserve per-source attribution by enforcing `cue.Filename(Source.Origin)` at load via library-provided loaders (no custom wrapper struct needed).
- Keep `Kernel.Validate(ctx, ValidateInput)` phase method's signature stable (its position in the four-phase pipeline is contractual).
- Provide the typed CLI/operator path: `Module.ValidateValues*` / `Release.ValidateValues*` so frontends do not need to look up `#config` themselves.
- Keep the codebase compiling at every commit during the transition (additive new surface first, callsite migration, deletion last).

**Non-Goals:**

- Designing a CLI rendering format. The library exposes data; the CLI owns presentation. Frontends call `cueerrors.Print` directly or walk `cueerrors.Errors`/`Positions` themselves; the library ships no formatter.
- Designing K8s status condition projection. The operator owns its status shape; if it wants a typed Go struct for Conditions, it builds one locally from the CUE error tree.
- Migrating downstream consumers (`cli/`, `opm-operator/`). Migration recipes are documented for them; their code changes ship in their own repos.
- A backward-compatibility shim. The change is MAJOR; no `Deprecated:` aliases survive. Downstream consumers update at their own pace by pinning the previous library tag.
- Validating `debugValues` automatically inside `NewModuleFromValue`. Validation of `debugValues` is the CLI's concern (`task vet`), achievable with `Module.ValidateValues(k, m.Package.LookupPath(b.Paths().DebugValues))`.
- Adding new validation modes (warnings-only, schema-lint). Future work.

## Decisions

### D1 — CUE-native errors over Go-typed projection

The library returns `error` from every validation function. The returned error implements `cuelang.org/go/cue/errors.Error` (or wraps a tree of them), so frontends call `cueerrors.Errors`, `cueerrors.Positions`, `cueerrors.Print` directly. No custom `DetailedError` / `FieldError` types are introduced.

**Alternatives considered:**

- **Keep `ConfigError` as the Tier-2 wrapper**: rejected. The `Context`/`Name`/`RawError` shape is a wrapper around what `fmt.Errorf("module %q: %w", name, err)` produces, with the cost of a type that frontends have to special-case.
- **Introduce a new `DetailedError` with `BySource map[string][]FieldError`**: rejected. Per-source bucketing is a frontend concern (different consumers want different bucket keys, sort orders, redaction rules); CUE already exposes the position metadata to do it in 5–10 lines of frontend code. Centralizing the bucket in the library imposes one shape on all consumers.
- **Wrap each `cueerrors.Error` in a thin `pathStrippedError` to normalize `#module.#config.` prefixes globally**: rejected. Mutating the tree hides the truth from frontends that want raw paths (e.g., for round-trip testing). Path stripping is presentation, which belongs to the frontend (see D6).

**Rationale:**

The library's job is to evaluate CUE and surface CUE's diagnostics. Inventing a Go vocabulary for diagnostics CUE already produces creates drift between what the library reports and what `cue vet` would report on the same input. CUE-native errors keep them aligned.

### D2 — Three primitives + eight typed wrappers (Variation C, strong)

Kernel exposes `ValidateConfig(schema, values)`, `ValidateConfigPartial(schema, values)`, `ValidateConfigDetailed(schema, sources, opts...)`. Module and Release each gain `ConfigSchema()`, `ValidateValues`, `ValidateValuesPartial`, `ValidateValuesDetailed`.

**Alternatives considered:**

- **Two primitives + `Partial()` option on Simple** (Variation A): rejected. Partial is a sufficiently common mode (CLI vet, IDE/LSP, admission webhooks, internal Detailed steps) that the option pattern hides it from grep and from autocomplete. A named function reads as clearly at the call site as it does at the receiver.
- **Four named primitives** (Variation B, full matrix): rejected. `ValidateConfigDetailedPartial` reads worse than `ValidateConfigDetailed(..., Partial())`. Detailed has multiple knobs (Partial, future Strict, future StopOnFirst); options scale better there.
- **Convenience wrappers as schema-accessors only** (weak reading): rejected. Frontends that already have a `*Module` benefit from `m.ValidateValues(k, vals)` reading like a one-step intent. The duplication cost is six 2-line methods that delegate to the kernel primitive — no logic duplication, low maintenance.

**Rationale:**

Detailed is the only function with axes that grow over time (Partial today; potentially StopOnFirst, ContextLines, Quiet later). Options compose. Simple has a fixed-shape mode (concrete vs partial) that maps onto its frequency-of-use: explicit functions read better than a knob nobody changes after the first call.

### D3 — `Source.Origin` doubles as `cue.Filename`; library provides loaders

`Source{Value, Name, Origin}` requires `Value` to have been compiled with `cue.Filename(Origin)`. Library provides `Kernel.LoadSourceFromFile`, `LoadSourceFromBytes`, `LoadSourceFromString` that bake the filename automatically. Callers using the loaders never trip the foot-gun; callers compiling directly via `ctx.CompileBytes` are responsible for setting the option themselves.

**Alternatives considered:**

- **Document and trust** (Option A from brainstorm): rejected for the byte/string case. Operators and XR fns synthesize values from non-file sources (ConfigMap data, CR overlays, composition input); without a loader, they have to know about `cue.Filename` themselves. A loader closes that footgun.
- **Validate at `ValidateConfigDetailed` entry that every Source has a non-empty `pos.Filename()`**: rejected. False negatives (a partially valid value where one inner expression has no filename) would surface as confusing error-on-input-shape rather than the actual schema error. Defensive validation here trades one diagnostic for a worse one.
- **Make `Source.Value` a private field constructed only via loaders**: rejected. Tests and ad-hoc programs need to hand-build a Source with a `cue.Value` they compiled themselves. Public field + clear documentation is the better contract.

**Rationale:**

The `cue.Filename` mechanism IS CUE's per-source attribution surface. Ducking it in favor of a Go-typed wrapper means re-implementing what the SDK already does. Loader helpers make the right thing the easy thing for the common cases (file, bytes, string).

### D4 — `walkDisallowed` and `fieldNotAllowedError` stay internal, implement `cueerrors.Error`

When a closed schema rejects an unknown field, CUE's native `Validate` reports "field not allowed" but loses the source position of the offending field (the position points to the schema's closure declaration, not the user's stray field). The library has compensated for this with `walkDisallowed` since the Tier-2 slice; `fieldNotAllowedError` is the position-aware diagnostic it emits. Both remain in the new design as private internals of `pkg/kernel/validate.go`. `fieldNotAllowedError` already implements `cueerrors.Error`, so it walks alongside CUE-native errors transparently.

**Alternatives considered:**

- **Drop the workaround; accept losing positions for closed-schema violations**: rejected. Closed schemas are common in OPM (`#config` is implicitly closed); position-less "field not allowed" is a poor frontend experience.
- **File a CUE upstream issue and wait**: future work, not blocking.

### D5 — Module name framing via `fmt.Errorf` in phase methods only

`Kernel.ValidateConfig` returns the raw CUE error tree. `Kernel.Validate(ctx, in)` and `Kernel.ProcessModuleRelease` wrap with `fmt.Errorf("module %q: %w", name, err)` because they have the name in hand and the framing is part of their contract. Convenience methods on `*Module`/`*Release` do not wrap (callers calling those have already asserted the module/release identity by their type). Frontends bypassing the phase methods are responsible for their own framing.

**Alternatives considered:**

- **Keep a minimal `ModuleConfigError{Name, Cause}`** type for typed access to the module name: rejected. Every caller that has the error also has the name (it came from their own input — `cr.Metadata.Name`, `*module.Release.Metadata.Name`, the file they parsed). Surfacing it through the error is solving a problem nobody has.
- **Wrap in every public function, not just phase methods**: rejected. The kernel primitive is a primitive; layering text on top of every call hides the CUE error tree's natural shape and double-wraps when callers compose primitives.

### D6 — Library ships no display helper

The library returns CUE-native errors and stops. No `PrintErrors`, no path-stripping helper, no opinionated formatter. Per Constitution principle I (Kernel Neutrality) and IV (Composability via Stable Contracts: "Output formatting and presentation MUST stay outside the library"), display is the frontend's job.

Practical consequences: the error tree carries paths of the form `["#module", "#config", "db", "port"]` (CUE's internal view). Frontends that want user-facing paths like `["db", "port"]` strip the prefix themselves at presentation time — typically a 5-line helper colocated with whatever rendering pipeline the frontend already has (CLI text, `metav1.Condition`, XR composition status). This keeps each frontend's display contract aligned with its consumer; the library does not impose a single shape on all of them.

**Alternatives considered:**

- **`kernel.PrintErrors(w, err, *cueerrors.Config)` that strips `#module.#config.` prefixes**: rejected. The path-stripping logic itself is fine; placing it in the library violates the kernel-neutrality contract. Frontends that need it own it.
- **Wrap each error in a `pathStrippedError`**: see D1. Information loss without a recovery path.

## Risks / Trade-offs

- **Risk: callers compile values without `cue.Filename(Origin)`, causing empty filenames in error positions.** → Mitigation: ship loader helpers (`LoadSourceFromFile/Bytes/String`) that bake the filename automatically. Document the contract on `Source`. Add a runtime check in `ValidateConfigDetailed` that issues a clear error if any `Source.Value` evaluates with `pos.Filename() == "" && pos.IsValid()` — defer this to follow-up if the failure mode proves theoretical.

- **Risk: large surface change in one shot violates Principle VIII (Small Batch Sizes).** → Mitigation: the change is staged in `tasks.md` so each numbered task lands in its own commit. Additive surface lands first, callsite migration second, deletions last; the codebase compiles and tests pass at every step. The constitution warns against multi-package refactors but this redesign is monolithic in capability — splitting it across releases would leave the library shipping two parallel error vocabularies and contradictory documentation.

- **Risk: downstream consumers (CLI, opm-operator) broken until they migrate.** → Mitigation: this is a MAJOR version bump per Principle VI; consumers pin the previous tag until they migrate. Migration recipes are captured here so the work is mechanical.

- **Risk: `walkDisallowed` becomes stale relative to CUE's evolving closedness semantics.** → Mitigation: existing test coverage stays in `pkg/kernel/validate_test.go`. Future CUE upgrades exercise the workaround through those tests; if CUE fixes positions for closed-schema rejections natively, the workaround is removable cleanly.

- **Trade-off: frontends now write 5–10 lines to bucket errors by source instead of consuming `*MultiSourceError.Errors()`.** Worth it: each frontend bucketing differently (file basename vs full path vs Source.Name) is a real divergence the library shouldn't impose a single shape on. CUE's own Position metadata is the substrate; bucket how you want.

- **Trade-off: convenience methods on `*Module`/`*Release` (eight of them) expand the typed-API surface.** Justified by the call-site readability win and the fact that each is a 2-line schema-lookup wrapper — Principle VII (YAGNI) applies to logic, not to thin convenience shells over already-justified primitives.

## Migration Plan

Sequencing (each step its own commit, each step keeps the codebase green):

1. **OpenSpec proposal/design/specs/tasks land** as the contract.
2. **Add new kernel surface** (`source.go`, `source_loader.go`, `print.go`, new `validate.go` symbols) alongside the old ones. Old API still works.
3. **Add convenience methods** on `*Module` and `*Release`. Old API still works.
4. **Migrate internal callsites** (`Kernel.Validate` phase method, `Kernel.ProcessModuleRelease`, `Kernel.Compile`) to call the new primitives, wrapping with `fmt.Errorf` where they used to receive a typed `*ConfigError`.
5. **Migrate tests** to assert on CUE-native errors (`cueerrors.Errors`, `pos.Filename()`, etc.) instead of `*ConfigError`/`*MultiSourceError` shape.
6. **Delete `Kernel.ValidateAndUnify` wrapper** in `wrappers.go` (no callers remain after step 4–5).
7. **Delete old kernel validation primitives** (`runValidate`, old `ValidateConfig`, old `ValidateConfigPartial`, `appendSchemaErrors` if not reused).
8. **Delete `pkg/helper/values/` package** (now unreferenced).
9. **Delete custom error types** in `pkg/errors/` (`ConfigError`, `ValidationError`, `FieldError`, `ErrorLocation`, `GroupedError`, related helpers).
10. **Update godoc and CHANGELOG**; tag the MAJOR version.

Migration recipes for downstream consumers (captured for `cli/` and `opm-operator/` to consume):

```
OLD                                                    NEW
─────────────────────────────────────────────────────────────────────────────────────
k.ValidateConfig(schema, vals, "module", name)         val, err := k.ValidateConfig(schema, vals)
                                                       if err != nil { return fmt.Errorf("module %q: %w", name, err) }

cfgErr.GroupedErrors()                                 for _, ce := range cueerrors.Errors(err) {
                                                         for _, pos := range cueerrors.Positions(ce) { ... }
                                                       }

cfgErr.Error()                                         var buf bytes.Buffer
                                                       cueerrors.Print(&buf, err, nil)
                                                       // (frontend owns formatting; kernel ships no printer)

helpervalues.Stack{ Layer{Name, Source, Value}, ... }  []kernel.Source{ {Value, Name, Origin}, ... }
helpervalues.ValidateAndUnify(k, schema, stack)        k.ValidateConfigDetailed(schema, sources)

multiSrcErr.Errors()                                   for _, ce := range cueerrors.Errors(err) {
                                                         filename := ce.Position().Filename()
                                                         // bucket / present as desired
                                                       }

ctx.CompileBytes(b)                                    src, err := k.LoadSourceFromBytes(origin, name, b)
                                                       // src.Value carries cue.Filename(origin)
```

## Open Questions

- Should `ValidateConfigDetailed` reject a `Source` whose `Value` has `pos.Filename() == ""` at entry, or trust the loader contract? Lean trust+document; revisit if footguns surface in practice.
- Should `Kernel.LoadSourceFromFile` reuse `pkg/helper/loader/file.LoadValuesFile` (which already does the filename baking via `cue/load.Instances`) or be a thin wrapper that calls into it? Lean wrap+delegate to avoid duplicating the load.Config dance; resolve in tasks.md.
- Do we add a `SourceForPos(sources []Source, pos token.Pos) *Source` helper for frontends that want friendly Name lookup from a position's filename? Defer until a frontend actually needs it; the 3-line map-build is fine in the meantime.
