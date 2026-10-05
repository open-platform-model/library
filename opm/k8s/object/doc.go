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
package object
