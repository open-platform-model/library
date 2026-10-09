package ownership_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/library/opm/k8s/ownership"
)

var (
	deployment = ownership.Object{Group: "apps", Kind: "Deployment", Namespace: "web", Name: "api"}
	namespace  = ownership.Object{Kind: "Namespace", Name: "team-a"}
	crd        = ownership.Object{Group: "apiextensions.k8s.io", Kind: "CustomResourceDefinition", Name: "widgets.example.com"}
	roleBind   = ownership.Object{Group: "rbac.authorization.k8s.io", Kind: "RoleBinding", Namespace: "opm-operator-system", Name: "leader"}
	clusterRB  = ownership.Object{Group: "rbac.authorization.k8s.io", Kind: "ClusterRoleBinding", Name: "opm-operator-manager"}
)

func TestCanDelete(t *testing.T) {
	tests := []struct {
		name     string
		in       ownership.DeleteInput
		want     ownership.SkipReason
		contains []string
	}{
		{
			name: "a protected kind is skipped without a live read",
			in:   ownership.DeleteInput{Object: namespace, InstanceUUID: thisUUID},
			want: ownership.SkipSafetyExcluded, contains: []string{"Namespace/team-a"},
		},
		{
			name: "a protected kind is skipped whatever the live object",
			in:   ownership.DeleteInput{Object: crd, Live: live("apiextensions.k8s.io/v1", "CustomResourceDefinition", "", "widgets.example.com", opm...), InstanceUUID: thisUUID},
			want: ownership.SkipSafetyExcluded,
		},
		{
			name: "a gone object is skipped as already absent",
			in:   ownership.DeleteInput{Object: deployment, InstanceUUID: thisUUID},
			want: ownership.SkipAlreadyAbsent, contains: []string{"Deployment/web/api"},
		},
		{
			name: "a foreign object is skipped",
			in:   ownership.DeleteInput{Object: deployment, Live: liveDeployment(managedBy("helm")), InstanceUUID: thisUUID},
			want: ownership.SkipNotOPMManaged, contains: []string{"Deployment/web/api"},
		},
		{
			name: "the adopt annotation is no delete override",
			in:   ownership.DeleteInput{Object: deployment, Live: liveDeployment(managedBy("helm"), adopt(thisUUID)), InstanceUUID: thisUUID},
			want: ownership.SkipNotOPMManaged,
		},
		{
			name: "an object with no labels is skipped",
			in:   ownership.DeleteInput{Object: deployment, Live: liveDeployment(), InstanceUUID: thisUUID},
			want: ownership.SkipNotOPMManaged,
		},
		{
			name: "another instance's object is skipped",
			in:   ownership.DeleteInput{Object: deployment, Live: liveDeployment(managedBy("opm-controller"), uuid(otherUUID)), InstanceUUID: thisUUID},
			want: ownership.SkipOwnerMismatch, contains: []string{"Deployment/web/api", otherUUID},
		},
		{
			name: "an empty live UUID passes",
			in:   ownership.DeleteInput{Object: deployment, Live: liveDeployment(managedBy("opm-cli")), InstanceUUID: thisUUID},
		},
		{
			name: "an empty instance UUID passes",
			in:   ownership.DeleteInput{Object: deployment, Live: liveDeployment(managedBy("opm-cli"), uuid(otherUUID))},
		},
		{
			name: "the legacy managed-by value is OPM's",
			in:   ownership.DeleteInput{Object: deployment, Live: liveDeployment(managedBy("open-platform-model"), uuid(thisUUID)), InstanceUUID: thisUUID},
		},
		{
			name: "an object being deleted proceeds",
			in:   ownership.DeleteInput{Object: deployment, Live: liveDeployment(append(opm, terminating())...), InstanceUUID: thisUUID},
		},
		{
			name: "a Namespace of another group is no protected kind",
			in:   ownership.DeleteInput{Object: ownership.Object{Group: "example.com", Kind: "Namespace", Name: "team-a"}, Live: live("example.com/v1", "Namespace", "", "team-a", append(opm, uid("u-ns"))...), InstanceUUID: thisUUID},
		},
		{
			name: "an object annotated for another instance is left in place",
			in:   ownership.DeleteInput{Object: deployment, Live: liveDeployment(append(opm, adopt(otherUUID))...), InstanceUUID: thisUUID},
			want: ownership.SkipAdoptedElsewhere, contains: []string{"Deployment/web/api", "module instance " + otherUUID},
		},
		{
			name: "an annotation naming this instance changes nothing on delete",
			in:   ownership.DeleteInput{Object: deployment, Live: liveDeployment(append(opm, adopt(" "+thisUUID))...), InstanceUUID: thisUUID},
		},
		{
			name: "a blank annotation passes the adoption comparison",
			in:   ownership.DeleteInput{Object: deployment, Live: liveDeployment(append(opm, adopt(" "))...), InstanceUUID: thisUUID},
		},
		{
			name: "an empty instance UUID skips an annotated object",
			in:   ownership.DeleteInput{Object: deployment, Live: liveDeployment(managedBy("opm-cli"), adopt(otherUUID))},
			want: ownership.SkipAdoptedElsewhere,
		},
		{
			name: "an earlier install manifest's Deployment is not deleted",
			in:   ownership.DeleteInput{Object: deployment, Live: liveDeployment(managedBy("kustomize"), uid("u-dep")), InstanceUUID: thisUUID},
			want: ownership.SkipNotOPMManaged,
		},
		{
			name: "an earlier install manifest's RoleBinding with no labels is not deleted",
			in:   ownership.DeleteInput{Object: roleBind, Live: live("rbac.authorization.k8s.io/v1", "RoleBinding", "opm-operator-system", "leader"), InstanceUUID: thisUUID},
			want: ownership.SkipNotOPMManaged,
		},
		{
			name: "an earlier install manifest's ClusterRoleBinding is not deleted",
			in:   ownership.DeleteInput{Object: clusterRB, Live: live("rbac.authorization.k8s.io/v1", "ClusterRoleBinding", "", "opm-operator-manager", managedBy("kustomize")), InstanceUUID: thisUUID},
			want: ownership.SkipNotOPMManaged,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var before *unstructured.Unstructured
			if tt.in.Live != nil {
				before = tt.in.Live.DeepCopy()
			}
			got := ownership.CanDelete(tt.in)
			assert.Equal(t, tt.want, got.Skip)
			assert.Equal(t, tt.want == "", got.Proceed())
			if tt.want == "" {
				assert.Empty(t, got.Message)
				assert.Equal(t, tt.in.Live.GetUID(), got.UID)
				assert.Equal(t, tt.in.Live.GetResourceVersion(), got.ResourceVersion)
			} else {
				assert.NotEmpty(t, got.Message)
				assert.Empty(t, got.UID)
				assert.Empty(t, got.ResourceVersion)
				assert.Nil(t, got.Preconditions())
				assertCleanMessage(t, got.Message)
			}
			for _, s := range tt.contains {
				assert.Contains(t, got.Message, s)
			}
			if before != nil {
				assert.Equal(t, before, tt.in.Live, "the live object is not modified")
			}
		})
	}
}

func TestCanDeleteCarriesTheJudgedObject(t *testing.T) {
	got := ownership.CanDelete(ownership.DeleteInput{
		Object:       deployment,
		Live:         liveDeployment(append(opm, uid("u-1"), rv("42"))...),
		InstanceUUID: thisUUID,
	})
	require.True(t, got.Proceed())
	assert.Equal(t, "u-1", string(got.UID))
	assert.Equal(t, "42", got.ResourceVersion)

	pre := got.Preconditions()
	require.NotNil(t, pre)
	require.NotNil(t, pre.UID)
	assert.Equal(t, "u-1", string(*pre.UID))
	assert.Nil(t, pre.ResourceVersion)
}

func TestPreconditionsNilWithoutAUID(t *testing.T) {
	got := ownership.CanDelete(ownership.DeleteInput{Object: deployment, Live: liveDeployment(opm...), InstanceUUID: thisUUID})
	require.True(t, got.Proceed())
	assert.Nil(t, got.Preconditions(), "a live object without a UID yields no precondition")

	skipped := ownership.CanDelete(ownership.DeleteInput{Object: deployment, InstanceUUID: thisUUID})
	assert.Equal(t, ownership.SkipAlreadyAbsent, skipped.Skip)
	assert.Nil(t, skipped.Preconditions(), "a skip verdict yields no precondition")
}

func TestSkipReasonLiterals(t *testing.T) {
	assert.Equal(t, ownership.SkipReason("safety-excluded"), ownership.SkipSafetyExcluded)
	assert.Equal(t, ownership.SkipReason("already-absent"), ownership.SkipAlreadyAbsent)
	assert.Equal(t, ownership.SkipReason("not-opm-managed"), ownership.SkipNotOPMManaged)
	assert.Equal(t, ownership.SkipReason("owner-mismatch"), ownership.SkipOwnerMismatch)
	assert.Equal(t, ownership.SkipReason("adopted-elsewhere"), ownership.SkipAdoptedElsewhere)
}

// TestSkipMessageWording pins the adopted-elsewhere skip wording both
// frontends print.
func TestSkipMessageWording(t *testing.T) {
	got := ownership.CanDelete(ownership.DeleteInput{Object: deployment, Live: liveDeployment(append(opm, adopt(otherUUID))...), InstanceUUID: thisUUID})
	assert.Equal(t, "Deployment/web/api is being adopted by module instance u-1, not this one; left in place", got.Message)
}

// assertCleanMessage checks that a message names no override and carries no
// enhancement reference: it reaches users as it stands.
func assertCleanMessage(t *testing.T, msg string) {
	t.Helper()
	assert.NotContains(t, msg, "--")
	assert.NotContains(t, msg, "force")
	assert.NotContains(t, msg, "0012:")
}
