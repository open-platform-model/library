package inventory

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// LabelComponentName is the label injected by the CUE catalog onto every
// rendered application resource to record which component produced it. Its value
// is the component name, used by inventory to track provenance for the
// component-rename safety check (see ApplyComponentRenameSafetyCheck).
//
// This mirrors the constant of the same name in the operator and CLI core
// packages; it is duplicated here rather than imported to keep this package free
// of any runtime-specific dependency.
const LabelComponentName = "component.opmodel.dev/name"

// InventoryEntry is the runtime-neutral, in-memory representation of a single
// Kubernetes object owned by a rendered release. It carries no Kubernetes
// framework baggage, so importing it never drags controller-runtime or Flux into
// a consumer. It is distinct from any CRD serialization type (e.g. the
// operator's api/v1alpha1.InventoryEntry); consumers map their own shapes to and
// from this type at their boundary.
//
//nolint:revive // Inventory* prefix is intentional: this type is referenced by name across consumers.
type InventoryEntry struct {
	// Group is the API group of the object (empty for the core group).
	Group string `json:"group"`
	// Kind is the object Kind (e.g. "Deployment").
	Kind string `json:"kind"`
	// Namespace is the object namespace (empty for cluster-scoped objects).
	Namespace string `json:"namespace"`
	// Name is the object name.
	Name string `json:"name"`
	// Version is the API version (e.g. "v1"). It is recorded for reference but
	// is deliberately excluded from identity comparison so that Kubernetes API
	// version migrations (e.g. v1beta1 → v1) do not produce false orphans.
	//
	// The JSON key is the abbreviated "v" rather than "version" deliberately:
	// ComputeDigest hashes this struct's JSON encoding, and both the operator's
	// CRD InventoryEntry and the CLI's struct already use `json:"v,omitempty"`.
	// Keeping the key identical preserves digest continuity with inventories
	// those actors stored before adopting this package. Do NOT rename it.
	Version string `json:"v,omitempty"`
	// Component is the name of the OPM component that produced the object. It is
	// excluded from Kubernetes object identity (K8sIdentityEqual) but included in
	// full identity (IdentityEqual); a component move is handled explicitly by
	// ApplyComponentRenameSafetyCheck rather than by treating the object as stale.
	Component string `json:"component,omitempty"`
}

// NewEntryFromResource builds an InventoryEntry from a rendered Kubernetes
// object, reading its GroupVersionKind, namespace, name, and the OPM component
// label. It performs no I/O and reads only the supplied object.
func NewEntryFromResource(r *unstructured.Unstructured) InventoryEntry {
	gvk := r.GroupVersionKind()
	component := r.GetLabels()[LabelComponentName]
	return InventoryEntry{
		Group:     gvk.Group,
		Kind:      gvk.Kind,
		Namespace: r.GetNamespace(),
		Name:      r.GetName(),
		Version:   gvk.Version,
		Component: component,
	}
}

// IdentityEqual reports whether two entries identify the same owned resource
// with full, component-aware identity. It compares Group, Kind, Namespace, Name,
// and Component. Version is excluded to prevent false orphans during Kubernetes
// API version migrations.
//
// This is NOT the comparator used by ComputeStaleSet — that uses
// K8sIdentityEqual so that a component rename (same GVK + namespace + name,
// different component label) does not produce stale entries for live objects
// that an SSA apply patches in place. It is deterministic and side-effect free.
func IdentityEqual(a, b InventoryEntry) bool {
	return a.Group == b.Group &&
		a.Kind == b.Kind &&
		a.Namespace == b.Namespace &&
		a.Name == b.Name &&
		a.Component == b.Component
}

// K8sIdentityEqual reports whether two entries identify the same Kubernetes
// object as the apiserver sees it: one live object per Group + Kind + Namespace
// + Name. It ignores Version and Component, and is deterministic and
// side-effect free. This is the canonical base relation for stale-set
// computation.
func K8sIdentityEqual(a, b InventoryEntry) bool {
	return a.Group == b.Group &&
		a.Kind == b.Kind &&
		a.Namespace == b.Namespace &&
		a.Name == b.Name
}
