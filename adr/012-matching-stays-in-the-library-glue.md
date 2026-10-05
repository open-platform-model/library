# ADR-012: Matching stays in the library glue

## Status

Accepted (2026-10-03). Records the owner's answer to where the matching algorithm lives, given in the beta.1 kernel plan walkthrough. Leaves 0019:D10 and 0019:D17 (workspace root, `enhancements/archive/0019/03-decisions.md`) standing. Answers core#62, "Adopt #Match from the library render glue into core", with "not now". Recorded by `record-walkthrough-decisions`, which ships no code.

## Context

0019:D10 moved matching out of Go and into the render build. The generated render module's glue, `opm/internal/renderstage/render.cue.tmpl`, defines `#Match`: from the platform's composed transformers and the instance's components it derives the pairs, the demand buckets, the unresolved demands, the unify failures and the optional-trait warnings, and it emits all of them as data the kernel decodes. 0019:D17 removed `#Platform.#matchers` from core, on the grounds that the render build's glue owns the reverse index and nothing else consumes it. The template is about 570 lines today, of which `#Match` is the larger part.

core#62 proposes moving `#Match` into core as a definition beside `#Platform`, with three arguments. A platform author or a second runtime cannot read what matches from core without embedding the kernel, because the contract exists only in the glue and in prose in core's `SPEC.md`. Core already owns analogous derivations: 0019:D12 moved the `#context` projection into core so the glue would not hand-roll it, and core carries the `#composedTransformers` fold. And the matcher is the one render semantic chosen by the kernel binary rather than by the platform's pinned `opmodel.dev/core` version. The issue itself notes that the move reverses half of 0019:D17 and needs its own decision entry, and that the kernel's decoder would then read field names whose version the platform pins.

One derived rule has already moved. `read-provider-count-from-core` made the glue read the provider count from the platform's `#contracts.providedBy`, a field core derives, instead of computing it. `opm/kernel/render_inventory_parity_test.go` is the tripwire: it checks that the render's verdict rows name exactly the registry keys the platform inventory's `providedBy` holds, so a drift between core's derivation and the glue's use of it fails a test rather than a render.

Core v2 is a beta line. A change to a rule core owns is a core release, a break needs a `feat!` with a migration note and owner sign-off when it reaches `catalogs/opm`, and after GA a break needs a new major. A fix to such a rule reaches a render only when the platform re-pins.

## Decision

The matching algorithm stays in the library's render glue. 0019:D10 and 0019:D17 stand.

A single derived rule may move into core, one at a time, when core can compute it from core shapes alone. Each move is an additive core release and ships with a parity test proving that the core field and the verdict or derivation it replaces agree over the served fixtures, the way `render_inventory_parity_test.go` does for `#contracts.providedBy`. The per-catalog provider set, which `Catalog.Provides()` derived in Go, was the next candidate. It moved in core `v2.0.0-beta.3` as `#Catalog.provides`: `Provides()` decodes it, `opm/catalog/provides_parity_test.go` is its parity test, and the Go fold stays only as a deprecated fallback for catalogs built against an older core, removed before GA.

Rejected alternatives:

- **Move `#Match` whole into core (core#62).** Not now. Every matching fix would become a core release, and every matching break a `feat!` on core (a new major after GA), the decoder's field names would become core contract, and it reverses half of 0019:D17, which needs an enhancement decision rather than an ADR. The readability argument is real; moving rules one at a time answers part of it without taking on the rest of the cost at once.
- **A core reverse index, reversing 0019:D17.** It brings back the shape 0019:D17 removed, a core field the glue does not consume, and has the same release cost for every change to it.

## Consequences

**Positive:** Matching fixes stay library changes and ship on the library's cadence, with the parity oracle and the render scenario tests covering them. Each rule that does move into core arrives with a test that proves it agrees with what it replaces, so a move cannot silently change a verdict. The question core#62 raised gets an answer that the next rule (the per-catalog provider set) can follow without a new decision.

**Negative:** The matching contract is still not readable from core alone. A second runtime, or a platform author with only `cue export`, still has to embed the kernel to see what matches. The matcher version is still chosen by the kernel binary, not by the platform's core pin, for every rule that has not moved.

**Trade-off:** One rule at a time is slower than one move, and each move costs a core release plus a library re-pin. In exchange no move is larger than one parity test can check.

**Follow-up outside this ADR:** core#62 is to be closed as not now, with a comment pointing at this ADR. Moving `#Match` whole later needs an enhancement decision amending 0019:D17, and this ADR would then be superseded.
