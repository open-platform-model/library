package object_test

import (
	"strings"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/k8s/object"
	"github.com/open-platform-model/library/opm/kernel"
)

// compiled builds one rendered object the way Render hands it over: a
// concrete CUE value plus the component and transformer that produced it.
func compiled(t *testing.T, component, transformer, src string) *kernel.Compiled {
	t.Helper()
	v := cuecontext.New().CompileString(src)
	require.NoError(t, v.Err())
	return &kernel.Compiled{
		Value:       v,
		Instance:    "app",
		Component:   component,
		Transformer: transformer,
	}
}

const registrationTransformer = "opmodel.dev/catalogs/opm/transformer-registration-transformer@4.4.0"

func registration(t *testing.T, component string) *kernel.Compiled {
	t.Helper()
	return compiled(t, component, registrationTransformer, `
		apiVersion: "opmodel.dev/v1alpha1"
		kind:       "TransformerRegistration"
		metadata: name: "backup-system.k8up"
	`)
}

func deployment(t *testing.T, component, transformer, name string) *kernel.Compiled {
	t.Helper()
	return deploymentAt(t, component, transformer, "apps/v1", name)
}

func deploymentAt(t *testing.T, component, transformer, apiVersion, name string) *kernel.Compiled {
	t.Helper()
	return compiled(t, component, transformer, `
		apiVersion: "`+apiVersion+`"
		kind:       "Deployment"
		metadata: {
			name:      "`+name+`"
			namespace: "web-system"
		}
	`)
}

func TestDuplicates_TwoComponentsRenderOneClusterScopedObject(t *testing.T) {
	rows := object.Duplicates([]*kernel.Compiled{
		registration(t, "registration"),
		registration(t, "registration-copy"),
	})

	require.Len(t, rows, 1)
	assert.Equal(t, object.Identity{
		APIVersion: "opmodel.dev/v1alpha1",
		Kind:       "TransformerRegistration",
		Name:       "backup-system.k8up",
	}, rows[0].Identity)
	assert.Empty(t, rows[0].Identity.Namespace)
	assert.Equal(t, []object.Producer{
		{Component: "registration", Transformer: registrationTransformer, APIVersion: "opmodel.dev/v1alpha1"},
		{Component: "registration-copy", Transformer: registrationTransformer, APIVersion: "opmodel.dev/v1alpha1"},
	}, rows[0].Producers)
}

func TestDuplicates_ThreeObjectsTwoIdentities(t *testing.T) {
	rows := object.Duplicates([]*kernel.Compiled{
		deployment(t, "web", "…/deployment@1.2.0", "web"),
		compiled(t, "web", "…/service@1.2.0", `
			apiVersion: "v1"
			kind:       "Service"
			metadata: {
				name:      "web"
				namespace: "web-system"
			}
		`),
		deployment(t, "web", "…/legacy-deployment@1.0.0", "web"),
	})

	require.Len(t, rows, 1)
	assert.Equal(t, "Deployment", rows[0].Identity.Kind)
	assert.Equal(t, "apps/v1 Deployment web-system/web", rows[0].Identity.String())
	assert.Len(t, rows[0].Producers, 2)
}

func TestDuplicates_CleanRenderReturnsNothing(t *testing.T) {
	rows := object.Duplicates([]*kernel.Compiled{
		deployment(t, "web", "…/deployment@1.2.0", "web"),
		deployment(t, "api", "…/deployment@1.2.0", "api"),
		registration(t, "registration"),
	})

	assert.Empty(t, rows)
}

func TestDuplicates_ValueWithoutNameIsSkipped(t *testing.T) {
	nameless := compiled(t, "platform", "…/platform@1.0.0", `
		apiVersion: "opmodel.dev/v1alpha2"
		kind:       "Platform"
		metadata: labels: tier: "core"
	`)

	rows := object.Duplicates([]*kernel.Compiled{
		nameless,
		registration(t, "registration"),
		registration(t, "registration-copy"),
	})

	require.Len(t, rows, 1)
	assert.Equal(t, "TransformerRegistration", rows[0].Identity.Kind)
	for _, p := range rows[0].Producers {
		assert.NotEqual(t, "platform", p.Component)
	}
}

func TestDuplicateIdentitiesError_NamesTheIdentityOnceAndBothComponents(t *testing.T) {
	rows := object.Duplicates([]*kernel.Compiled{
		registration(t, "registration"),
		registration(t, "registration-copy"),
	})
	err := &object.DuplicateIdentitiesError{Duplicates: rows}

	msg := err.Error()
	assert.Equal(t, 1, strings.Count(msg, "opmodel.dev/v1alpha1 TransformerRegistration backup-system.k8up"))
	assert.Contains(t, msg, "2 rendered objects share one identity")
	assert.Contains(t, msg, `component "registration" (`+registrationTransformer+`)`)
	assert.Contains(t, msg, `component "registration-copy" (`+registrationTransformer+`)`)
	assert.Contains(t, msg, " and ")
	assert.Len(t, strings.Split(msg, "\n"), 2)
}

func TestDuplicates_RowsFollowEachRendersOwnFirstSeenOrder(t *testing.T) {
	dupA := func() []*kernel.Compiled {
		return []*kernel.Compiled{
			deployment(t, "web", "…/deployment@1.2.0", "web"),
			deployment(t, "web", "…/legacy-deployment@1.0.0", "web"),
		}
	}
	dupB := func() []*kernel.Compiled {
		return []*kernel.Compiled{
			registration(t, "registration"),
			registration(t, "registration-copy"),
		}
	}

	first := object.Duplicates(append(dupA(), dupB()...))
	second := object.Duplicates(append(dupB(), dupA()...))

	require.Len(t, first, 2)
	require.Len(t, second, 2)
	assert.Equal(t, "Deployment", first[0].Identity.Kind)
	assert.Equal(t, "TransformerRegistration", first[1].Identity.Kind)
	assert.Equal(t, "TransformerRegistration", second[0].Identity.Kind)
	assert.Equal(t, "Deployment", second[1].Identity.Kind)
}

func configMap(t *testing.T, component, apiVersionField string) *kernel.Compiled {
	t.Helper()
	return compiled(t, component, "…/configmap@1.0.0", apiVersionField+`
		kind: "ConfigMap"
		metadata: {
			name:      "x"
			namespace: "web-system"
		}
	`)
}

func TestDuplicates_OneObjectUnderTwoVersionsOfItsGroup(t *testing.T) {
	rows := object.Duplicates([]*kernel.Compiled{
		deploymentAt(t, "a", "…/deployment@1.0.0", "apps/v1", "web"),
		deploymentAt(t, "b", "…/legacy@1.0.0", "apps/v1beta2", "web"),
	})

	require.Len(t, rows, 1)
	assert.Equal(t, object.Identity{
		APIVersion: "apps/v1",
		Kind:       "Deployment",
		Namespace:  "web-system",
		Name:       "web",
	}, rows[0].Identity)
	assert.Equal(t, []object.Producer{
		{Component: "a", Transformer: "…/deployment@1.0.0", APIVersion: "apps/v1"},
		{Component: "b", Transformer: "…/legacy@1.0.0", APIVersion: "apps/v1beta2"},
	}, rows[0].Producers)
}

func TestDuplicates_OneKindAndNameInTwoGroupsStaysDistinct(t *testing.T) {
	widget := func(component, apiVersion string) *kernel.Compiled {
		return compiled(t, component, "…/widget@1.0.0", `
			apiVersion: "`+apiVersion+`"
			kind:       "Widget"
			metadata: name: "web"
		`)
	}

	rows := object.Duplicates([]*kernel.Compiled{
		widget("a", "a.example.com/v1"),
		widget("b", "b.example.com/v1"),
	})

	assert.Empty(t, rows)
}

func TestDuplicates_CoreGroupPairIsOneRow(t *testing.T) {
	rows := object.Duplicates([]*kernel.Compiled{
		configMap(t, "a", `apiVersion: "v1"`),
		configMap(t, "b", `apiVersion: "v1"`),
	})

	require.Len(t, rows, 1)
	assert.Equal(t, "v1 ConfigMap web-system/x", rows[0].Identity.String())
	assert.Len(t, rows[0].Producers, 2)
}

func TestDuplicates_MissingAPIVersionFallsInTheCoreGroup(t *testing.T) {
	rows := object.Duplicates([]*kernel.Compiled{
		configMap(t, "a", `apiVersion: "v1"`),
		configMap(t, "b", ""),
	})

	require.Len(t, rows, 1)
	assert.Equal(t, "v1", rows[0].Identity.APIVersion)
	assert.Equal(t, "v1", rows[0].Producers[0].APIVersion)
	assert.Empty(t, rows[0].Producers[1].APIVersion)

	msg := (&object.DuplicateIdentitiesError{Duplicates: rows}).Error()
	assert.Contains(t, msg, `component "a" (…/configmap@1.0.0) as v1 and component "b" (…/configmap@1.0.0) as <no apiVersion>`)
}

func TestDuplicateIdentitiesError_NamesAVersionMismatch(t *testing.T) {
	rows := object.Duplicates([]*kernel.Compiled{
		deploymentAt(t, "a", "…/deployment@1.0.0", "apps/v1", "web"),
		deploymentAt(t, "b", "…/legacy@1.0.0", "apps/v1beta2", "web"),
	})

	msg := (&object.DuplicateIdentitiesError{Duplicates: rows}).Error()
	assert.Equal(t, "2 rendered objects share one identity, so the last apply would silently overwrite the first:\n"+
		`  apps/v1 Deployment web-system/web rendered by component "a" (…/deployment@1.0.0) as apps/v1 and component "b" (…/legacy@1.0.0) as apps/v1beta2`,
		msg)
}

func TestDuplicateIdentitiesError_SameVersionRowCarriesNoVersions(t *testing.T) {
	rows := object.Duplicates([]*kernel.Compiled{
		registration(t, "registration"),
		registration(t, "registration-copy"),
	})

	msg := (&object.DuplicateIdentitiesError{Duplicates: rows}).Error()
	assert.Equal(t, "2 rendered objects share one identity, so the last apply would silently overwrite the first:\n"+
		`  opmodel.dev/v1alpha1 TransformerRegistration backup-system.k8up rendered by `+
		`component "registration" (`+registrationTransformer+`) and `+
		`component "registration-copy" (`+registrationTransformer+`)`,
		msg)
	assert.NotContains(t, msg, " as ")
}
