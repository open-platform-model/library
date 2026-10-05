package object_test

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/k8s/object"
	"github.com/open-platform-model/library/opm/kernel"
)

// deploymentCUE is a minimal concrete Kubernetes Deployment manifest in CUE.
const deploymentCUE = `
{
	apiVersion: "apps/v1"
	kind:       "Deployment"
	metadata: {
		name:      "my-app"
		namespace: "default"
		labels: {
			"app.kubernetes.io/name": "my-app"
		}
		annotations: {
			"example.com/note": "test"
		}
	}
	spec: {
		replicas: 1
	}
}
`

func newTestResource(t *testing.T, cueSrc string) *object.Resource {
	t.Helper()
	ctx := cuecontext.New()
	v := ctx.CompileString(cueSrc)
	require.NoError(t, v.Err())
	return &object.Resource{
		Value:       v,
		Instance:    "test-instance",
		Component:   "test-component",
		Transformer: "kubernetes#deployment",
	}
}

func TestResource_Accessors(t *testing.T) {
	r := newTestResource(t, deploymentCUE)

	assert.Equal(t, "Deployment", r.Kind())
	assert.Equal(t, "my-app", r.Name())
	assert.Equal(t, "default", r.Namespace())
	assert.Equal(t, "apps/v1", r.APIVersion())
	assert.Equal(t, map[string]string{"app.kubernetes.io/name": "my-app"}, r.Labels())
	assert.Equal(t, map[string]string{"example.com/note": "test"}, r.Annotations())
	assert.Equal(t, "Deployment/default/my-app", r.String())
}

func TestResource_GVK(t *testing.T) {
	r := newTestResource(t, deploymentCUE)
	gvk := r.GVK()
	assert.Equal(t, "apps", gvk.Group)
	assert.Equal(t, "v1", gvk.Version)
	assert.Equal(t, "Deployment", gvk.Kind)
}

func TestResource_GVK_CoreGroup(t *testing.T) {
	r := newTestResource(t, `{
		apiVersion: "v1"
		kind:       "Service"
		metadata: name: "my-svc"
	}`)
	gvk := r.GVK()
	assert.Equal(t, "", gvk.Group)
	assert.Equal(t, "v1", gvk.Version)
	assert.Equal(t, "Service", gvk.Kind)
}

func TestResource_MarshalJSON(t *testing.T) {
	r := newTestResource(t, deploymentCUE)
	b, err := r.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(b), `"kind":"Deployment"`)
	assert.Contains(t, string(b), `"name":"my-app"`)
}

func TestResource_ToUnstructured(t *testing.T) {
	r := newTestResource(t, deploymentCUE)
	u, err := r.ToUnstructured()
	require.NoError(t, err)
	assert.Equal(t, "Deployment", u.GetKind())
	assert.Equal(t, "my-app", u.GetName())
	assert.Equal(t, "default", u.GetNamespace())
}

func TestResource_Namespace_Empty(t *testing.T) {
	r := newTestResource(t, `{
		apiVersion: "apiextensions.k8s.io/v1"
		kind:       "CustomResourceDefinition"
		metadata: name: "foos.example.com"
	}`)
	assert.Equal(t, "", r.Namespace())
	assert.Equal(t, "CustomResourceDefinition/foos.example.com", r.String())
}

// TestResource_BestEffort covers the kubernetes-tier scenario "Accessors are
// best-effort": an absent field reads as empty or nil, never as an error.
func TestResource_BestEffort(t *testing.T) {
	r := newTestResource(t, `{
		kind: "ConfigMap"
		metadata: name: "cm"
	}`)
	assert.Equal(t, "", r.Namespace())
	assert.Nil(t, r.Annotations())
	assert.Nil(t, r.Labels())
	assert.Equal(t, "", r.APIVersion())
	assert.Equal(t, "", r.GVK().Group)

	notObject := newTestResource(t, `[1, 2]`)
	assert.Equal(t, "", notObject.Kind())
	assert.Equal(t, "", notObject.Name())
	assert.Nil(t, notObject.Labels())
}

func TestNewResource(t *testing.T) {
	assert.Nil(t, object.NewResource(nil))

	v := cuecontext.New().CompileString(deploymentCUE)
	c := &kernel.Compiled{Value: v, Instance: "i", Component: "c", Transformer: "t"}
	r := object.NewResource(c)
	require.NotNil(t, r)
	assert.Equal(t, "i", r.Instance)
	assert.Equal(t, "c", r.Component)
	assert.Equal(t, "t", r.Transformer)
	assert.Equal(t, "Deployment", r.Kind())
}

func TestResources(t *testing.T) {
	ctx := cuecontext.New()
	a := &kernel.Compiled{Value: ctx.CompileString(`{kind: "A", metadata: name: "a"}`), Component: "a"}
	b := &kernel.Compiled{Value: ctx.CompileString(`{kind: "B", metadata: name: "b"}`), Component: "b"}

	got := object.Resources([]*kernel.Compiled{a, nil, b, nil})
	require.Len(t, got, 2)
	assert.Equal(t, "a", got[0].Component)
	assert.Equal(t, "b", got[1].Component)

	assert.Empty(t, object.Resources(nil))
}
