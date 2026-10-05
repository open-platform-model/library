package inventory

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/library/opm/k8s/labels"
)

// Entry is one object an instance owns, as the tier compares and digests it.
// Group, Kind, Namespace and Name identify the object ([SameObject]). Version
// and Component are recorded and never part of the identity. Entry carries no
// struct tags: the library does not own the wire shape of an inventory.
type Entry struct {
	// Group is the API group; empty for the core group.
	Group string
	// Kind is the object's kind.
	Kind string
	// Namespace is the object's namespace; empty for a cluster-scoped object.
	Namespace string
	// Name is the object's name.
	Name string
	// Version is the API version the object was applied at.
	Version string
	// Component is the value of the object's labels.ComponentName label: the
	// component that produced it, recorded as provenance. Empty when the
	// label is absent.
	Component string
}

// NewEntry reads obj's group, version, kind, namespace and name, and its
// labels.ComponentName label. obj must not be nil.
func NewEntry(obj *unstructured.Unstructured) Entry {
	gvk := obj.GroupVersionKind()
	return Entry{
		Group:     gvk.Group,
		Kind:      gvk.Kind,
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
		Version:   gvk.Version,
		Component: obj.GetLabels()[labels.ComponentName],
	}
}

// SameObject reports whether a and b name the same Kubernetes object: equal
// group, kind, namespace and name. It ignores Version and Component.
func SameObject(a, b Entry) bool {
	return identityOf(a) == identityOf(b)
}

// identity is the part of an Entry that names the object.
type identity struct{ group, kind, namespace, name string }

func identityOf(e Entry) identity {
	return identity{group: e.Group, kind: e.Kind, namespace: e.Namespace, name: e.Name}
}
