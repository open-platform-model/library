// Package labels is the OPM label vocabulary for Kubernetes objects: the keys
// and values the cli and the operator read and write, and [IsOPMManagedBy],
// which recognises every managed-by value an OPM runtime has stamped. Besides
// the labels it names one annotation, [AnnotationAdopt], which a user writes
// on a live object and no OPM runtime ever writes.
//
// The package names and recognises labels. It never stamps them on rendered
// objects: core's CUE stamps those at render, where #runtimeName fills the
// managed-by value for the runtime that renders (0012:D6). The only labels a
// frontend writes itself are the ones on its own bookkeeping objects, such as
// the cli's inventory Secret; each constant's doc says who writes it.
//
// It is part of the Kubernetes tier and imports nothing outside the Go
// standard library, which a depguard rule in .golangci.yml enforces, so a
// frontend can read the vocabulary without the rest of the tier.
package labels

// Maintainer pointer, kept out of the package doc because it publishes into
// the Library reference: the tier this package belongs to is ADR-011.
