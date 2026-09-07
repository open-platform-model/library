// Package errors provides the structured verdict rows and error types of OPM.
//
// The render path splits the two. A ROW is plain data with no Error method:
// [UnresolvedDemand], [UnifyRefusal], [UnmatchedComponent],
// [CandidateVerdict] and [OverSubscribedContract] are the verdicts the render
// build decided, decoded as they were emitted and carried on the kernel's
// render diagnostics. A CAUSE is a pointer-receiver aggregate over those rows
// that the fail-closed gate raises: [*UnresolvedDemandsError],
// [*UnmatchedComponentsError] and [*OverSubscribedContractsError]. Each
// carries its rows unchanged and in the same order, and wraps nothing, so
// errors.As on one never yields a cause of another kind. [*TransformError]
// and [*SkewError] are ordinary wrappers with a real cause underneath.
//
// Configuration validation errors are CUE-native — see
// [cuelang.org/go/cue/errors] for the canonical interface and helpers
// (Errors, Positions, Print). The library does not wrap CUE diagnostics
// in custom Go-typed projections, nor does it ship a presentation-layer
// formatter; frontends walk the CUE error tree and render however their
// consumer requires.
package errors
