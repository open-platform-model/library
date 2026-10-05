// Package errors provides the structured verdict rows and error types of OPM.
//
// The render path splits the two. A ROW is plain data with no Error method:
// [UnresolvedDemand], [UnifyRefusal], [UnmatchedComponent],
// [CandidateVerdict], [OverSubscribedContract] and [ContractCollision] are
// the verdicts the render build decided, decoded as they were emitted and
// carried on the kernel's render diagnostics. A CAUSE is a pointer-receiver
// aggregate over those rows that the fail-closed gate raises:
// [*ContractCollisionsError], [*UnresolvedDemandsError],
// [*UnmatchedComponentsError] and [*OverSubscribedContractsError]. Each
// carries its rows unchanged and in the same order, and wraps nothing, so
// errors.As on one never yields a cause of another kind. [*NotRoutableError]
// is the gate's rowless catch-all, raised from the decoded routable verdict
// alone. [*TransformError]
// and [*SkewError] are ordinary wrappers with a real cause underneath.
//
// A failed registry interaction, during a fetch or during dependency
// resolution, is a [*FetchError]: its [FetchKind] says whether the module was
// absent, the registry refused the credentials or could not be reached, and
// errors.Is(err, [ErrTransient]) says whether the same request may succeed
// later (network-level failures only: no answer, an expired deadline or a
// 5xx). An author-defect resolution failure, such as an import no module of
// the build provides, is a [*ResolutionError] instead: its [ResolutionKind]
// says which, and it is never transient. [Classify] builds both from the raw
// errors the CUE module machinery returns, reading the typed chain first and
// the text cue/load flattens only after; the library applies it at every
// site where such a failure leaves it, and a frontend applies it to the CUE
// errors it meets itself (a `cue mod tidy`, its own cue/load call).
//
// Configuration validation errors are CUE-native — see
// [cuelang.org/go/cue/errors] for the canonical interface and helpers
// (Errors, Positions, Print). The library does not wrap CUE diagnostics
// in custom Go-typed projections, nor does it ship a presentation-layer
// formatter; frontends walk the CUE error tree and render however their
// consumer requires.
package errors
