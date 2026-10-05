// Package objectset finds rendered objects that share one Kubernetes apply
// identity, so a runtime can refuse the render instead of letting the last
// write silently overwrite the first.
//
// Two objects with the same API group, kind, namespace and name reach apply
// as two writes to one object, whatever version of the group each names.
// Nothing in the kernel notices: kernel.Compiled deliberately carries no
// platform vocabulary, and Render never reads kind or metadata. This package
// supplies the missing check in the one place that has both the objects and
// their provenance, without moving Kubernetes vocabulary into the kernel.
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
//
// Deprecated: the check's home is opm/k8s/object in the Kubernetes tier,
// which carries the same exported names, signatures,
// behaviour and error wording. This copy is frozen and stays only until both
// the cli and the operator have moved their imports, so their library bump
// keeps compiling; a later change removes it.
package objectset

// Design record behind the package doc above, for maintainers: the move of
// this check to opm/k8s/object is ADR-011 item 9.
