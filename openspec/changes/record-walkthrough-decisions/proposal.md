## Why

On 2026-10-02 and 2026-10-03 the owner walked through the beta.1 kernel plan task by task. Five of the answers change what the library's ADRs and repo docs say, and no Go behaviour. Until they are written into the library, the ADRs keep saying things the owner has now decided against:

- ADR-008 rule 4 and ADR-011 item 7 say that module-declared order (hooks, `dependsOn`, phases) is data off the CUE build, as if `dependsOn` edges and phases were coming. The owner never intended ordering within a module. Lifecycle hooks stay what they are, the parked 0009 work. The kernel applies in the order it wants and Kubernetes eventual consistency handles the rest. Cross-module ordering belongs to a future Bundle definition, which would use the order its modules are defined in. Kind-class staging (a CRD and a Namespace first) stays, because it is a fact about the Kubernetes API. 0012:D5 is amended to match in the enhancements change of the same name.
- ADR-008 does not say where a deletion plan comes from when nothing is rendered. The owner decided that deletion plans are built from the persisted inventory plus the live objects, with no render and no stored plan. The questions of planning from a render and of lifecycle transition detection (install, upgrade, reconfigure, no-op, uninstall) are deferred to the parked 0009 hook work. That is distinct from the deletion step transition the tier owns.
- No ADR says where matching lives. core#62 proposes moving `#Match` into core. The owner decided that the matching algorithm stays in the library glue (0019:D10 and 0019:D17 stand), and that single derived rules move into core one at a time, each with a parity test, as `#contracts.providedBy` did. That needs its own ADR, ADR-012.
- ADR-005 rule 2 and the `opm/kernel` package doc say that a caller cannot obtain a built value to hold. A `*kernel.Compiled` carries a live `cue.Value`, which keeps its render's build alive for as long as the caller holds it. ADR-007 already states the holder-bounded rule for acquired artifacts. The owner decided to keep `Compiled.Value` for now, to state the rule as it is, and to switch Render output to bytes before GA at the latest.
- ADR-004 says the GA arming condition is picked up by a "GA release checklist" that does not exist. The GA exit criteria are 0021:D8. The enhancements change of the same name adds the library's own criteria to it as `0021:D8:R11` to `0021:D8:R14`.

## What Changes

- **ADR-008** is amended in place with a dated Status sentence and a rule 4 clarification: no module-internal ordering (`dependsOn` edges, phases) is planned (if one is ever needed, it comes off the build), lifecycle hooks remain the parked 0009 work carried as build data under rule 4, in the OPM model ordering across modules belongs to a future Bundle definition (definition order) and the library adds none before then, a frontend's readiness gating between its own custom resources is outside the rule, and kind-class staging stays in `opm/k8s/object`. A second dated sentence, with one new paragraph under Consequences, records the deletion-only answer: deletion plans are built from the persisted inventory plus the live objects, with no render and no stored plan, and plan-from-render and lifecycle transition detection are deferred to the parked 0009 hook work.
- **ADR-011** item 7 gains the same ordering clarification, so the library's record of 0012:D5 does not contradict the amended ADR-008.
- **ADR-012** (`adr/012-matching-stays-in-the-library-glue.md`, new) records that the matching algorithm stays in the library's render glue under 0019:D10 and 0019:D17. Single derived rules move into core one at a time, each with a parity test against the glue or the inventory it replaces. `#contracts.providedBy` is the precedent and the per-catalog provider set is next. Moving `#Match` as a whole (core#62) is rejected for now.
- **ADR-005** rule 2 (heading, "drops the context", the stale `[]*core.Compiled` and its last sentence) and the retention sentence in `opm/kernel/doc.go` (comment only) are reworded to ADR-007's holder-bounded rule: the kernel retains no built value between calls, and a `*kernel.Compiled` the caller holds keeps its render's build alive until the caller releases it. A dated Status sentence records that Render output moves to bytes before GA at the latest.
- **ADR-004**, `README.md` and `migrations/README.md` name 0021:D8 as the GA exit criteria (the arming condition is 0021:D8:R4), in place of the nonexistent GA release checklist.
- **Specs:** `kubernetes-tier` gets the ordering amendment and a deletion-plan requirement. `single-build-render` records where matching lives. `kernel-runtime` records the holder-bounded rule for Render output, and it and `single-build-render` stop saying the render context is released when `Render` returns. `migration-docs` points the arming condition at 0021:D8.

## Not in this change

- Closing core#62. That is a GitHub action on core, done separately by the owner with a comment pointing at ADR-012.
- Any Go code. `Compiled.Value` stays, and the switch to bytes is its own breaking change before GA.
- The operator's plaintext `status.lastAppliedVersion`, the per-catalog provider set in core, and the enhancements text of 0012:D5 and 0021:D8 (each is its own change).
- The rest of the `opm/kernel` package doc, which `restore-kernel-godoc` edits. This change touches only the retention sentence.

## Classification

**No release.** ADRs, Markdown, one Go doc comment and specs. No exported symbol, signature or behaviour changes. Section commits are `docs(adr)` and `docs(kernel)`, types release-please hides. Principle VII: one new ADR, which adds a written rule and no code.

## Downstream consumers

- **cli** and **opm-operator**: none. The ADR-005 amendment describes what a held `Compiled` already does. Releasing it promptly is how a consumer bounds memory today, and the operator's planned nil-out after conversion relies on that.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `kubernetes-tier`: the kind-class order requirement states that no module-internal ordering (`dependsOn` edges, phases) is planned, that hooks stay the parked 0009 work, and that ordering across modules belongs to a future Bundle definition. A new requirement says deletion plans come from the persisted inventory plus the live objects.
- `single-build-render` (also): a new requirement keeps the matching algorithm in the glue and moves single derived rules into core one at a time, each with a parity test.
- `kernel-runtime`: a new requirement states the holder-bounded retention rule for Render output and the move to bytes before GA; the Goroutine Safety Contract says the kernel releases its references when `Render` returns.
- `single-build-render`: "Each render is its own build in its own context" says the same.
- `migration-docs`: the GA arming requirement names 0021:D8 as the GA exit criteria.

## Impact

- `adr/004-migration-docs-structure.md`, `adr/005-shares-nothing-renders.md`, `adr/008-kernel-plans-caller-runs.md`, `adr/011-kubernetes-tier-beside-the-kernel.md`, `adr/012-matching-stays-in-the-library-glue.md` (new).
- `opm/kernel/doc.go` (one comment sentence).
- `README.md`, `migrations/README.md`.
