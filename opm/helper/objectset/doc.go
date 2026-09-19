// Package objectset finds rendered objects that share one Kubernetes apply
// identity, so a runtime can refuse the render instead of letting the last
// write silently overwrite the first.
//
// Two objects with the same apiVersion, kind, namespace and name reach apply
// as two writes to one object. Nothing in the kernel notices: kernel.Compiled
// deliberately carries no platform vocabulary, and Render never reads kind or
// metadata. This package supplies the missing check in the one place that has
// both the objects and their provenance, without moving Kubernetes vocabulary
// into the kernel.
//
// [Duplicates] takes the render's compiled objects and returns every identity
// two or more of them share, each row naming the identity and every producing
// component and transformer in render order. It reads exactly four fields off
// each value — apiVersion, kind, metadata.namespace, metadata.name — and
// validates nothing else. A value with no kind or no metadata.name is not a
// Kubernetes object: it is skipped, never refused, because a frontend that
// requires manifests already fails at conversion with a better message.
// [DuplicateIdentitiesError] turns the rows into the refusal itself, worded
// once so every runtime says the same thing.
//
// The kernel never calls this package, and a depguard rule in .golangci.yml
// keeps it that way; a frontend that applies to something other than
// Kubernetes may skip it entirely. A runtime that does apply to Kubernetes
// calls it between render and apply, on the []*kernel.Compiled the kernel
// returned and before any wrapping that would lose the provenance fields: the
// CLI in its render workflow, so build refuses what apply would, and the
// operator before it builds inventory entries.
package objectset
