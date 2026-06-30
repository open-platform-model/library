package inventory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func makeResource(group, version, kind, namespace, name, component string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{Group: group, Version: version, Kind: kind})
	obj.SetNamespace(namespace)
	obj.SetName(name)
	if component != "" {
		obj.SetLabels(map[string]string{LabelComponentName: component})
	}
	return obj
}

func TestNewEntryFromResource_Namespaced(t *testing.T) {
	r := makeResource("apps", "v1", "Deployment", "production", "my-app", "app")
	entry := NewEntryFromResource(r)
	assert.Equal(t, "apps", entry.Group)
	assert.Equal(t, "Deployment", entry.Kind)
	assert.Equal(t, "production", entry.Namespace)
	assert.Equal(t, "my-app", entry.Name)
	assert.Equal(t, "v1", entry.Version)
	assert.Equal(t, "app", entry.Component)
}

func TestNewEntryFromResource_ClusterScoped(t *testing.T) {
	r := makeResource("", "v1", "Namespace", "", "default", "")
	entry := NewEntryFromResource(r)
	assert.Equal(t, "", entry.Group)
	assert.Equal(t, "Namespace", entry.Kind)
	assert.Equal(t, "", entry.Namespace)
	assert.Equal(t, "default", entry.Name)
	assert.Equal(t, "v1", entry.Version)
	assert.Equal(t, "", entry.Component)
}

// Spec: Entry Identity Relations — "Same object, different component".
func TestIdentity_SameObjectDifferentComponent(t *testing.T) {
	a := InventoryEntry{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v1", Component: "web"}
	b := InventoryEntry{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v1", Component: "frontend"}
	assert.True(t, K8sIdentityEqual(a, b), "K8s identity should ignore component")
	assert.False(t, IdentityEqual(a, b), "full identity should include component")
}

// Spec: Entry Identity Relations — "Different object".
func TestIdentity_DifferentObject(t *testing.T) {
	a := InventoryEntry{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v1", Component: "web"}
	cases := map[string]InventoryEntry{
		"group":     {Group: "extensions", Kind: "Deployment", Namespace: "ns", Name: "app", Component: "web"},
		"kind":      {Group: "apps", Kind: "StatefulSet", Namespace: "ns", Name: "app", Component: "web"},
		"namespace": {Group: "apps", Kind: "Deployment", Namespace: "other", Name: "app", Component: "web"},
		"name":      {Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "other", Component: "web"},
	}
	for diff, b := range cases {
		t.Run(diff, func(t *testing.T) {
			assert.False(t, K8sIdentityEqual(a, b))
			assert.False(t, IdentityEqual(a, b))
		})
	}
}

// Version is recorded but excluded from both identity relations.
func TestIdentity_VersionExcluded(t *testing.T) {
	a := InventoryEntry{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v1", Component: "web"}
	b := InventoryEntry{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "app", Version: "v2", Component: "web"}
	assert.True(t, K8sIdentityEqual(a, b))
	assert.True(t, IdentityEqual(a, b), "version must not affect identity")
}
