// Package inventory is the record of the objects an instance owns, as every
// Kubernetes frontend compares and digests it. It is part of the Kubernetes
// tier beside the kernel (ADR-011): the kernel never imports it, a depguard
// rule in .golangci.yml keeps it that way, and a frontend that applies to
// Kubernetes uses it instead of a copy of its own.
//
// [Entry] is one owned object: its group, kind, namespace and name, which
// identify it, plus the API version and the component that produced it, which
// are recorded and never part of the identity. [NewEntry] reads one from a
// live or rendered object. The package owns no wire shape: an Entry carries
// no struct tags, and each frontend maps it to its own CRD or record fields.
//
// [StaleSet] is the set of previously applied objects the current render no
// longer produces, the objects a prune removes. It is component-blind
// (0012:D7): an object that moved to another component, or to another API
// version of its group, is the same object and never stale. [SameObject] is
// that comparison on its own.
//
// [Digest] is the inventory digest: a hash of the entries' field values in a
// canonical encoding, the same in every frontend whatever wire shape it
// stores the entries in.
//
// The inventory digest and the render digest are stored values: each frontend
// records them and compares a later value against them. Their encodings are
// versioned by a tag line at the start of the hashed bytes, and change only
// under a new tag line, with a migration note in each frontend.
package inventory
