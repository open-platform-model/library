// Package object turns the kernel's compiled output into Kubernetes objects
// and holds the Kubernetes facts about them that every frontend shares. It is
// part of the Kubernetes tier beside the kernel (ADR-011): the kernel never
// imports it, a depguard rule in .golangci.yml keeps it that way, and a
// frontend that applies to Kubernetes uses it instead of a copy of its own.
//
// [Resource] wraps one [kernel.Compiled]: the rendered CUE value with its
// instance, component and transformer provenance, and best-effort accessors
// for the fields Kubernetes addresses an object by. [NewResource] and
// [Resources] build it from the kernel's output.
//
// A Resource holds its CUE value, and a CUE value pins the whole build it
// came from (holder-bounded, ADR-007). A long-lived caller exports its
// Resources once and drops them; the library never drops them for it.
// [Export] is that one export: it exports each Resource from CUE exactly once
// and hands back, index-aligned, the JSON bytes, the object decoded from those
// same bytes and the provenance, so a digest, the apply objects and the
// inventory entries all read one result and nothing exports twice.
//
// [Weight] is the kind-class order every Kubernetes frontend applies by:
// definitions before their users on apply, the reverse on delete (0012:D5).
// [Sort] orders by it, and [Stages] cuts an apply set into the stages an
// apply engine that re-sorts within a call can take one at a time.
//
// [Duplicates] finds rendered objects that share one Kubernetes apply
// identity, so a runtime can refuse the render instead of letting the last
// write silently overwrite the first. Two objects with the same API group,
// kind, namespace and name reach apply as two writes to one object, whatever
// version of the group each names. Nothing in the kernel notices:
// kernel.Compiled deliberately carries no platform vocabulary, and Render
// never reads kind or metadata. Duplicates takes the render's compiled
// objects and returns every identity two or more of them share, each row
// naming the identity and every producing component and transformer in
// render order. It reads exactly four fields off each value (apiVersion,
// kind, metadata.namespace, metadata.name) and validates nothing else. A
// value with no kind or no metadata.name is not a Kubernetes object: it is
// skipped, never refused, because a frontend that requires manifests already
// fails at conversion with a better message. [DuplicateIdentitiesError] turns
// the rows into the refusal itself, worded once so every runtime says the
// same thing. A runtime calls it between render and apply, on the
// []*kernel.Compiled the kernel returned: the cli in its render workflow, so
// build refuses what apply would, and the operator before it builds
// inventory entries. The deprecated opm/helper/objectset holds an identical
// copy until both frontends have moved here.
package object
