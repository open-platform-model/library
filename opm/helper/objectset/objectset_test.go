package objectset_test

import (
	"strings"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/helper/objectset"
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
	return compiled(t, component, transformer, `
		apiVersion: "apps/v1"
		kind:       "Deployment"
		metadata: {
			name:      "`+name+`"
			namespace: "web-system"
		}
	`)
}

func TestDuplicates_TwoComponentsRenderOneClusterScopedObject(t *testing.T) {
	rows := objectset.Duplicates([]*kernel.Compiled{
		registration(t, "registration"),
		registration(t, "registration-copy"),
	})

	require.Len(t, rows, 1)
	assert.Equal(t, objectset.Identity{
		APIVersion: "opmodel.dev/v1alpha1",
		Kind:       "TransformerRegistration",
		Name:       "backup-system.k8up",
	}, rows[0].Identity)
	assert.Empty(t, rows[0].Identity.Namespace)
	assert.Equal(t, []objectset.Producer{
		{Component: "registration", Transformer: registrationTransformer},
		{Component: "registration-copy", Transformer: registrationTransformer},
	}, rows[0].Producers)
}

func TestDuplicates_ThreeObjectsTwoIdentities(t *testing.T) {
	rows := objectset.Duplicates([]*kernel.Compiled{
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
	rows := objectset.Duplicates([]*kernel.Compiled{
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

	rows := objectset.Duplicates([]*kernel.Compiled{
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
	rows := objectset.Duplicates([]*kernel.Compiled{
		registration(t, "registration"),
		registration(t, "registration-copy"),
	})
	err := &objectset.DuplicateIdentitiesError{Duplicates: rows}

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

	first := objectset.Duplicates(append(dupA(), dupB()...))
	second := objectset.Duplicates(append(dupB(), dupA()...))

	require.Len(t, first, 2)
	require.Len(t, second, 2)
	assert.Equal(t, "Deployment", first[0].Identity.Kind)
	assert.Equal(t, "TransformerRegistration", first[1].Identity.Kind)
	assert.Equal(t, "TransformerRegistration", second[0].Identity.Kind)
	assert.Equal(t, "Deployment", second[1].Identity.Kind)
}
