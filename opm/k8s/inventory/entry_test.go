package inventory_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/library/opm/k8s/inventory"
)

// kubernetes-tier: "An entry reads the object's identity and component".
func TestNewEntry_ReadsIdentityAndComponent(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]any{
			"name":      "web",
			"namespace": "team",
			"labels":    map[string]any{"component.opmodel.dev/name": "web"},
		},
	}}
	assert.Equal(t, inventory.Entry{
		Group: "apps", Kind: "Deployment", Namespace: "team", Name: "web", Version: "v1", Component: "web",
	}, inventory.NewEntry(obj))
}

// kubernetes-tier: "A core-group, cluster-scoped object without the label".
func TestNewEntry_CoreGroupClusterScopedNoLabel(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1",
		"kind":       "Namespace",
		"metadata":   map[string]any{"name": "team"},
	}}
	assert.Equal(t, inventory.Entry{Kind: "Namespace", Name: "team", Version: "v1"}, inventory.NewEntry(obj))
}

// kubernetes-tier: "The entry type carries no tags". A tag would suggest the
// library owns the wire shape of an inventory, which it does not (0012:OQ9).
func TestEntry_CarriesNoTags(t *testing.T) {
	typ := reflect.TypeFor[inventory.Entry]()
	assert.Equal(t, 6, typ.NumField())
	for i := range typ.NumField() {
		f := typ.Field(i)
		assert.Equal(t, reflect.String, f.Type.Kind(), "field %s is a string", f.Name)
		assert.Empty(t, string(f.Tag), "field %s carries no struct tag", f.Name)
	}
}

func TestSameObject(t *testing.T) {
	base := inventory.Entry{Group: "apps", Kind: "Deployment", Namespace: "team", Name: "web", Version: "v1", Component: "web"}
	with := func(edit func(*inventory.Entry)) inventory.Entry {
		e := base
		edit(&e)
		return e
	}
	cases := []struct {
		name string
		b    inventory.Entry
		want bool
	}{
		{"equal", base, true},
		{"component ignored", with(func(e *inventory.Entry) { e.Component = "api" }), true},
		{"version ignored", with(func(e *inventory.Entry) { e.Version = "v2" }), true},
		{"group counts", with(func(e *inventory.Entry) { e.Group = "extensions" }), false},
		{"kind counts", with(func(e *inventory.Entry) { e.Kind = "StatefulSet" }), false},
		{"namespace counts", with(func(e *inventory.Entry) { e.Namespace = "other" }), false},
		{"name counts", with(func(e *inventory.Entry) { e.Name = "api" }), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, inventory.SameObject(base, c.b))
			assert.Equal(t, c.want, inventory.SameObject(c.b, base), "symmetric")
		})
	}
}
