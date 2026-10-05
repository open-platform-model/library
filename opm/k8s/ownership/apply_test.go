package ownership_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/library/opm/k8s/labels"
	"github.com/open-platform-model/library/opm/k8s/ownership"
)

func TestCanApply(t *testing.T) {
	operatorNS := ownership.Object{Kind: "Namespace", Name: "opm-operator-system"}
	clusterRole := ownership.Object{Group: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "opm-operator-manager"}
	adoptKey := labels.AnnotationAdopt

	tests := []struct {
		name        string
		in          ownership.ApplyInput
		want        ownership.ApplyRefusal
		contains    []string
		notContains []string
	}{
		{
			name: "a new object is applied",
			in:   ownership.ApplyInput{Object: deployment, InstanceUUID: thisUUID},
		},
		{
			name: "a terminating object in the inventory is refused",
			in:   ownership.ApplyInput{Object: deployment, Live: liveDeployment(append(opm, terminating())...), InInventory: true, InstanceUUID: thisUUID},
			want: ownership.RefuseTerminating, contains: []string{"Deployment/web/api", "being deleted"},
		},
		{
			name: "a terminating object outside the inventory is refused",
			in:   ownership.ApplyInput{Object: deployment, Live: liveDeployment(append(opm, terminating())...), InstanceUUID: thisUUID},
			want: ownership.RefuseTerminating,
		},
		{
			name: "a terminating object adopted by this instance is refused",
			in:   ownership.ApplyInput{Object: deployment, Live: liveDeployment(managedBy("helm"), adopt(thisUUID), terminating()), InstanceUUID: thisUUID},
			want: ownership.RefuseTerminating,
		},
		{
			name: "admission never lifts the terminating refusal",
			in:   ownership.ApplyInput{Object: operatorNS, Live: live("v1", "Namespace", "", "opm-operator-system", managedBy("kustomize"), terminating()), InstanceUUID: thisUUID, Admit: true},
			want: ownership.RefuseTerminating, contains: []string{"Namespace/opm-operator-system"},
		},
		{
			name: "a terminating object annotated for another instance is refused as terminating",
			in:   ownership.ApplyInput{Object: deployment, Live: liveDeployment(append(opm, adopt(otherUUID), terminating())...), InInventory: true, InstanceUUID: thisUUID},
			want: ownership.RefuseTerminating,
		},
		{
			name: "an inventoried foreign object is applied",
			in:   ownership.ApplyInput{Object: deployment, Live: liveDeployment(managedBy("helm")), InInventory: true, InstanceUUID: thisUUID},
		},
		{
			name: "an inventoried object of this instance is applied",
			in:   ownership.ApplyInput{Object: deployment, Live: liveDeployment(opm...), InInventory: true, InstanceUUID: thisUUID},
		},
		{
			name: "an inventoried object annotated for another instance is refused",
			in:   ownership.ApplyInput{Object: deployment, Live: liveDeployment(append(opm, adopt(otherUUID))...), InInventory: true, InstanceUUID: thisUUID},
			want: ownership.RefuseAdoptedElsewhere,
			contains: []string{
				"module instance " + otherUUID, "drops it from its inventory",
				"to take it back, annotate it " + adoptKey + "=" + thisUUID,
			},
		},
		{
			name:     "an inventoried object another instance has taken is refused",
			in:       ownership.ApplyInput{Object: deployment, Live: liveDeployment(managedBy("opm-controller"), uuid(otherUUID)), InInventory: true, InstanceUUID: thisUUID},
			want:     ownership.RefuseAdoptedElsewhere,
			contains: []string{"module instance " + otherUUID, "drops it from its inventory"},
		},
		{
			name: "an annotation naming this instance takes an inventoried object back",
			in:   ownership.ApplyInput{Object: deployment, Live: liveDeployment(managedBy("opm-controller"), uuid(otherUUID), adopt(" "+thisUUID+"\n")), InInventory: true, InstanceUUID: thisUUID},
		},
		{
			name: "an empty instance UUID applies an inventoried object",
			in:   ownership.ApplyInput{Object: deployment, Live: liveDeployment(managedBy("opm-cli"), uuid(otherUUID), adopt(otherUUID)), InInventory: true},
		},
		{
			name:        "a dropped object is not taken back on the next apply",
			in:          ownership.ApplyInput{Object: deployment, Live: liveDeployment(append(opm, adopt(otherUUID))...), InstanceUUID: thisUUID},
			want:        ownership.RefuseAdoptedElsewhere,
			contains:    []string{"module instance " + otherUUID, "does not apply it", adoptKey + "=" + thisUUID},
			notContains: []string{"drops it from its inventory"},
		},
		{
			name: "an empty instance UUID refuses an annotated object outside the inventory",
			in:   ownership.ApplyInput{Object: deployment, Live: liveDeployment(managedBy("opm-cli"), adopt(otherUUID))},
			want: ownership.RefuseAdoptedElsewhere, notContains: []string{"annotate it"},
		},
		{
			name:        "a foreign object outside the inventory is refused",
			in:          ownership.ApplyInput{Object: deployment, Live: liveDeployment(managedBy("kustomize")), InstanceUUID: thisUUID},
			want:        ownership.RefuseForeignObject,
			contains:    []string{"Deployment/web/api", "not managed by OPM", adoptKey + "=" + thisUUID},
			notContains: []string{"names another instance"},
		},
		{
			name: "an object with no managed-by label is foreign",
			in:   ownership.ApplyInput{Object: deployment, Live: liveDeployment(), InstanceUUID: thisUUID},
			want: ownership.RefuseForeignObject, contains: []string{"Deployment/web/api", adoptKey, thisUUID},
		},
		{
			name: "another instance's object outside the inventory is refused",
			in:   ownership.ApplyInput{Object: deployment, Live: liveDeployment(managedBy("opm-controller"), uuid(otherUUID)), InstanceUUID: thisUUID},
			want: ownership.RefuseOtherInstance,
			contains: []string{
				"Deployment/web/api", "module instance " + otherUUID,
				"remove it from module instance " + otherUUID + ", then annotate it " + adoptKey + "=" + thisUUID,
			},
		},
		{
			name: "an OPM object without a UUID label is applied",
			in:   ownership.ApplyInput{Object: deployment, Live: liveDeployment(managedBy("opm-cli")), InstanceUUID: thisUUID},
		},
		{
			name: "an OPM object of this instance outside the inventory is applied",
			in:   ownership.ApplyInput{Object: deployment, Live: liveDeployment(opm...), InstanceUUID: thisUUID},
		},
		{
			name: "an empty instance UUID fails closed",
			in:   ownership.ApplyInput{Object: deployment, Live: liveDeployment(managedBy("opm-cli"), uuid(otherUUID))},
			want: ownership.RefuseOtherInstance, contains: []string{"Deployment/web/api"}, notContains: []string{"annotate it"},
		},
		{
			name: "an empty instance UUID never matches an adopt annotation",
			in:   ownership.ApplyInput{Object: deployment, Live: liveDeployment(managedBy("helm"), adopt(" "))},
			want: ownership.RefuseForeignObject, notContains: []string{"annotate it", "names another instance"},
		},
		{
			name: "adoption lifts a foreign refusal",
			in:   ownership.ApplyInput{Object: deployment, Live: liveDeployment(managedBy("helm"), adopt(thisUUID)), InstanceUUID: thisUUID},
		},
		{
			name: "an adopt value with surrounding whitespace is trimmed",
			in:   ownership.ApplyInput{Object: deployment, Live: liveDeployment(managedBy("helm"), adopt(" "+thisUUID+"\n")), InstanceUUID: thisUUID},
		},
		{
			name: "adoption lifts an other-instance refusal",
			in:   ownership.ApplyInput{Object: deployment, Live: liveDeployment(managedBy("opm-cli"), uuid(otherUUID), adopt(thisUUID)), InstanceUUID: thisUUID},
		},
		{
			name:     "an annotation naming another instance lifts nothing",
			in:       ownership.ApplyInput{Object: deployment, Live: liveDeployment(managedBy("helm"), adopt("u-7")), InstanceUUID: thisUUID},
			want:     ownership.RefuseForeignObject,
			contains: []string{"Deployment/web/api", adoptKey + " annotation names another instance (u-7)", adoptKey + "=" + thisUUID},
		},
		{
			name: "an annotation naming another instance does not lift other-instance",
			in:   ownership.ApplyInput{Object: deployment, Live: liveDeployment(managedBy("opm-cli"), uuid(otherUUID), adopt(otherUUID)), InstanceUUID: thisUUID},
			want: ownership.RefuseOtherInstance, contains: []string{"names another instance"},
		},
		{
			name: "an empty annotation counts as none",
			in:   ownership.ApplyInput{Object: deployment, Live: liveDeployment(managedBy("helm"), adopt("")), InstanceUUID: thisUUID},
			want: ownership.RefuseForeignObject, notContains: []string{"names another instance"},
		},
		{
			name: "a whitespace annotation counts as none",
			in:   ownership.ApplyInput{Object: deployment, Live: liveDeployment(managedBy("helm"), adopt("  \t")), InstanceUUID: thisUUID},
			want: ownership.RefuseForeignObject, notContains: []string{"names another instance"},
		},
		{
			name: "a proven earlier-manifest object is admitted on apply",
			in:   ownership.ApplyInput{Object: operatorNS, Live: live("v1", "Namespace", "", "opm-operator-system", managedBy("kustomize")), InstanceUUID: thisUUID, Admit: true},
		},
		{
			name: "a proven ClusterRole with no labels is admitted on apply",
			in:   ownership.ApplyInput{Object: clusterRole, Live: live("rbac.authorization.k8s.io/v1", "ClusterRole", "", "opm-operator-manager"), InstanceUUID: thisUUID, Admit: true},
		},
		{
			name: "an admitted object carrying this instance's UUID is applied",
			in:   ownership.ApplyInput{Object: deployment, Live: liveDeployment(managedBy("kustomize"), uuid(thisUUID)), InstanceUUID: thisUUID, Admit: true},
		},
		{
			name: "an admitted object carrying another identity is refused",
			in:   ownership.ApplyInput{Object: deployment, Live: liveDeployment(managedBy("kustomize"), uuid(otherUUID)), InstanceUUID: thisUUID, Admit: true},
			want: ownership.RefuseForeignObject,
		},
		{
			name: "admission never lifts the adopted-elsewhere refusal",
			in:   ownership.ApplyInput{Object: operatorNS, Live: live("v1", "Namespace", "", "opm-operator-system", managedBy("kustomize"), adopt(otherUUID)), InstanceUUID: thisUUID, Admit: true},
			want: ownership.RefuseAdoptedElsewhere,
		},
		{
			name: "admission never lifts other-instance",
			in:   ownership.ApplyInput{Object: deployment, Live: liveDeployment(managedBy("opm-cli"), uuid(otherUUID)), InstanceUUID: thisUUID, Admit: true},
			want: ownership.RefuseOtherInstance,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var before *unstructured.Unstructured
			if tt.in.Live != nil {
				before = tt.in.Live.DeepCopy()
			}
			got := ownership.CanApply(tt.in)
			assert.Equal(t, tt.want, got.Refuse)
			assert.Equal(t, tt.want == "", got.Allowed())
			if tt.want == "" {
				assert.Empty(t, got.Message)
			} else {
				assert.Contains(t, got.Message, tt.in.Object.String())
				assertCleanMessage(t, got.Message)
			}
			for _, s := range tt.contains {
				assert.Contains(t, got.Message, s)
			}
			for _, s := range tt.notContains {
				assert.NotContains(t, got.Message, s)
			}
			if before != nil {
				assert.Equal(t, before, tt.in.Live, "the live object is not modified")
			}
		})
	}
}

func TestApplyRefusalLiterals(t *testing.T) {
	assert.Equal(t, ownership.ApplyRefusal("terminating"), ownership.RefuseTerminating)
	assert.Equal(t, ownership.ApplyRefusal("foreign-object"), ownership.RefuseForeignObject)
	assert.Equal(t, ownership.ApplyRefusal("other-instance"), ownership.RefuseOtherInstance)
	assert.Equal(t, ownership.ApplyRefusal("adopted-elsewhere"), ownership.RefuseAdoptedElsewhere)
}

// TestRefusalMessageWording pins the wording both frontends print, so a change
// to it is a deliberate one.
func TestRefusalMessageWording(t *testing.T) {
	foreign := ownership.CanApply(ownership.ApplyInput{Object: deployment, Live: liveDeployment(managedBy("helm")), InstanceUUID: thisUUID})
	assert.Equal(t, "Deployment/web/api exists and is not managed by OPM; to let this instance take it over, annotate it opmodel.dev/adopt=u-9", foreign.Message)

	other := ownership.CanApply(ownership.ApplyInput{Object: deployment, Live: liveDeployment(managedBy("opm-cli"), uuid(otherUUID)), InstanceUUID: thisUUID})
	assert.Equal(t, "Deployment/web/api belongs to module instance u-1; to move it to this instance, remove it from module instance u-1, then annotate it opmodel.dev/adopt=u-9", other.Message)

	term := ownership.CanApply(ownership.ApplyInput{Object: deployment, Live: liveDeployment(terminating())})
	assert.Equal(t, "Deployment/web/api is being deleted; wait for the deletion to finish, then apply again", term.Message)

	inventoried := ownership.CanApply(ownership.ApplyInput{Object: deployment, Live: liveDeployment(append(opm, adopt(otherUUID))...), InInventory: true, InstanceUUID: thisUUID})
	assert.Equal(t, "Deployment/web/api was adopted by module instance u-1; this instance no longer applies it and drops it from its inventory; to take it back, annotate it opmodel.dev/adopt=u-9", inventoried.Message)

	outside := ownership.CanApply(ownership.ApplyInput{Object: deployment, Live: liveDeployment(append(opm, adopt(otherUUID))...), InstanceUUID: thisUUID})
	assert.Equal(t, "Deployment/web/api is being adopted by module instance u-1; this instance does not apply it; to let this instance take it over, annotate it opmodel.dev/adopt=u-9", outside.Message)

	noUUID := ownership.CanApply(ownership.ApplyInput{Object: deployment, Live: liveDeployment(managedBy("opm-cli"), adopt(otherUUID))})
	assert.Equal(t, "Deployment/web/api is being adopted by module instance u-1; this instance does not apply it", noUUID.Message)
}
