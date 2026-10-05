package ownership

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/library/opm/k8s/labels"
)

// Object is the identity of the object being judged, as a frontend reads it
// from an inventory entry or a rendered object: its API group (empty for the
// core group), kind, namespace (empty when cluster-scoped) and name.
type Object struct {
	Group     string
	Kind      string
	Namespace string
	Name      string
}

// String writes the object as Kind/namespace/name, or Kind/name when it is
// cluster-scoped. Every verdict message names the object this way.
func (o Object) String() string {
	if o.Namespace == "" {
		return o.Kind + "/" + o.Name
	}
	return o.Kind + "/" + o.Namespace + "/" + o.Name
}

// The group and kind of the two kinds OPM never deletes.
const (
	groupCore          = ""
	kindNamespace      = "Namespace"
	groupAPIExtensions = "apiextensions.k8s.io"
	kindCRD            = "CustomResourceDefinition"
)

// SafetyExcluded reports whether OPM never deletes objects of a group and
// kind: a Namespace of the core group, or a CustomResourceDefinition of
// apiextensions.k8s.io. Deleting either takes everything inside or every
// custom resource of it with it. The match is on group and kind, whatever the
// version, so a same-named kind in another group is not excluded. It needs no
// live object, so a frontend can skip the live read for these kinds.
func SafetyExcluded(group, kind string) bool {
	switch {
	case group == groupCore && kind == kindNamespace:
		return true
	case group == groupAPIExtensions && kind == kindCRD:
		return true
	default:
		return false
	}
}

// liveLabel returns one label of the live object, "" when it has none.
func liveLabel(live *unstructured.Unstructured, key string) string {
	return live.GetLabels()[key]
}

// liveUUID returns the live object's module instance UUID label.
func liveUUID(live *unstructured.Unstructured) string {
	return liveLabel(live, labels.ModuleInstanceUUID)
}

// opmManaged reports whether the live object's managed-by label is an OPM
// runtime's value.
func opmManaged(live *unstructured.Unstructured) bool {
	return labels.IsOPMManagedBy(liveLabel(live, labels.ManagedBy))
}
