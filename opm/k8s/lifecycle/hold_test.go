package lifecycle_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/library/opm/k8s/inventory"
	"github.com/open-platform-model/library/opm/k8s/labels"
	"github.com/open-platform-model/library/opm/k8s/lifecycle"
	"github.com/open-platform-model/library/opm/k8s/ownership"
)

func TestMayReleaseHold(t *testing.T) {
	two := []inventory.Entry{web, api}
	forceOrphan := lifecycle.Policy{Prune: true, ForceOrphan: true}
	planOf := func(pol lifecycle.Policy) lifecycle.DeletionPlan {
		return lifecycle.NewDeletionPlan(two, pol, thisUUID)
	}
	done := func(outcomes ...lifecycle.Outcome) lifecycle.State {
		return lifecycle.State{Next: 2, Outcomes: outcomes}
	}
	deleted := func(i int) lifecycle.Outcome { return lifecycle.Outcome{Step: i, Result: lifecycle.ResultDeleted} }
	failedAs := func(i int, c lifecycle.FailureClass) lifecycle.Outcome {
		return lifecycle.Outcome{Step: i, Result: lifecycle.ResultFailed, Failure: c, Message: "x"}
	}
	skipped := func(i int) lifecycle.Outcome {
		return lifecycle.Outcome{Step: i, Result: lifecycle.ResultSkipped, Skip: ownership.SkipNotOPMManaged}
	}

	tests := []struct {
		name     string
		plan     lifecycle.DeletionPlan
		state    lifecycle.State
		identity lifecycle.Identity
		release  bool
		because  lifecycle.HoldReason
		message  string
	}{
		{
			name: "branch 1: pruning disabled releases", plan: planOf(lifecycle.Policy{}),
			state: lifecycle.State{}, identity: lifecycle.IdentityFailed,
			release: true, because: lifecycle.HoldPruneDisabled,
			message: "pruning is disabled; the objects are left in place",
		},
		{
			name: "pruning disabled releases whatever the identity and state", plan: planOf(lifecycle.Policy{ForceOrphan: true}),
			state: done(failedAs(0, lifecycle.FailureForbidden)), identity: lifecycle.IdentityMissing,
			release: true, because: lifecycle.HoldPruneDisabled,
		},
		{
			name: "branch 2: an empty inventory releases", plan: lifecycle.NewDeletionPlan(nil, prune, thisUUID),
			identity: lifecycle.IdentityMissing, release: true, because: lifecycle.HoldInventoryEmpty,
			message: "the inventory is empty; there is nothing to delete",
		},
		{
			name: "branch 3: force-orphan releases for a missing identity", plan: planOf(forceOrphan),
			identity: lifecycle.IdentityMissing, release: true, because: lifecycle.HoldForceOrphan,
			message: "the deleting identity is missing and orphaning was forced; 2 objects left in place",
		},
		{
			name: "force-orphan does not lift a failed identity", plan: planOf(forceOrphan),
			identity: lifecycle.IdentityFailed, because: lifecycle.HoldIdentityUnavailable,
		},
		{
			name: "branch 4: a missing identity without force-orphan holds", plan: planOf(prune),
			identity: lifecycle.IdentityMissing, because: lifecycle.HoldIdentityUnavailable,
			message: "the deleting identity is missing; 2 objects not deleted",
		},
		{
			name: "branch 5: a failed identity holds", plan: planOf(prune),
			identity: lifecycle.IdentityFailed, because: lifecycle.HoldIdentityUnavailable,
			message: "the deleting identity could not be obtained; 2 objects not deleted",
		},
		{
			name: "an unfinished plan holds", plan: planOf(prune),
			state:   lifecycle.State{Next: 1, Outcomes: []lifecycle.Outcome{deleted(0)}},
			because: lifecycle.HoldCleanupIncomplete,
			message: "the deletion is not finished; 1 object still to process",
		},
		{
			name: "a plan awaiting a delete holds", plan: planOf(prune),
			state:   lifecycle.State{Next: 1, Awaiting: lifecycle.AwaitDelete, Outcomes: []lifecycle.Outcome{deleted(0)}},
			because: lifecycle.HoldCleanupIncomplete,
		},
		{
			name: "the zero state holds", plan: planOf(prune),
			because: lifecycle.HoldCleanupIncomplete,
			message: "the deletion is not finished; 2 objects still to process",
		},
		{
			name: "a finished step count with no outcomes holds", plan: planOf(prune),
			state:   lifecycle.State{Next: 2},
			because: lifecycle.HoldCleanupIncomplete,
			message: "the deletion state does not belong to this plan; 2 objects not confirmed deleted",
		},
		{
			name: "a state past the plan holds", plan: planOf(prune),
			state:   lifecycle.State{Next: 5, Outcomes: []lifecycle.Outcome{deleted(0), deleted(1), deleted(2), deleted(3), deleted(4)}},
			because: lifecycle.HoldCleanupIncomplete,
			message: "the deletion state does not belong to this plan; 2 objects not confirmed deleted",
		},
		{
			name: "a negative step holds", plan: planOf(prune),
			state:   lifecycle.State{Next: -1},
			because: lifecycle.HoldCleanupIncomplete,
			message: "the deletion state does not belong to this plan; 2 objects not confirmed deleted",
		},
		{
			name: "a state awaiting a read after the last step holds", plan: planOf(prune),
			state:   lifecycle.State{Next: 2, Awaiting: lifecycle.AwaitRead, Outcomes: []lifecycle.Outcome{deleted(0), deleted(1)}},
			because: lifecycle.HoldCleanupIncomplete,
			message: "the deletion state does not belong to this plan; 2 objects not confirmed deleted",
		},
		{
			name: "an outcome out of the plan's order holds", plan: planOf(prune),
			state:   lifecycle.State{Next: 2, Outcomes: []lifecycle.Outcome{deleted(0), deleted(9)}},
			because: lifecycle.HoldCleanupIncomplete,
			message: "the deletion state does not belong to this plan; 2 objects not confirmed deleted",
		},
		{
			name: "an outcome with an unknown result holds", plan: planOf(prune),
			state:   lifecycle.State{Next: 2, Outcomes: []lifecycle.Outcome{deleted(0), {Step: 1, Result: "weird"}}},
			because: lifecycle.HoldCleanupIncomplete,
			message: "the deletion state does not belong to this plan; 2 objects not confirmed deleted",
		},
		{
			name: "branch 6: a Forbidden failure holds", plan: planOf(prune),
			state:   done(deleted(0), failedAs(1, lifecycle.FailureForbidden)),
			because: lifecycle.HoldCleanupForbidden,
			message: "the deleting identity was forbidden to read or delete 1 object; 1 object left to retry",
		},
		{
			name: "a Forbidden failure wins over other failures", plan: planOf(prune),
			state:   done(failedAs(0, lifecycle.FailureError), failedAs(1, lifecycle.FailureForbidden)),
			because: lifecycle.HoldCleanupForbidden,
			message: "the deleting identity was forbidden to read or delete 1 object; 2 objects left to retry",
		},
		{
			name: "branch 7: any other failure holds", plan: planOf(prune),
			state:   done(deleted(0), failedAs(1, lifecycle.FailureError)),
			because: lifecycle.HoldCleanupIncomplete,
			message: "1 object could not be deleted; left to retry",
		},
		{
			name: "a failed precondition holds", plan: planOf(prune),
			state:   done(failedAs(0, lifecycle.FailureConflict), failedAs(1, lifecycle.FailureConflict)),
			because: lifecycle.HoldCleanupIncomplete,
			message: "2 objects could not be deleted; left to retry",
		},
		{
			name: "force-orphan does not lift a failed cleanup", plan: planOf(forceOrphan),
			state:   done(deleted(0), failedAs(1, lifecycle.FailureError)),
			because: lifecycle.HoldCleanupIncomplete,
		},
		{
			name: "force-orphan does not lift a Forbidden cleanup", plan: planOf(forceOrphan),
			state:   done(deleted(0), failedAs(1, lifecycle.FailureForbidden)),
			because: lifecycle.HoldCleanupForbidden,
		},
		{
			name: "branch 8: a finished plan with skips releases", plan: planOf(prune),
			state:   done(deleted(0), skipped(1)),
			release: true, because: lifecycle.HoldCleanupComplete,
			message: "every object was deleted or left in place",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lifecycle.MayReleaseHold(tt.plan, tt.state, lifecycle.HoldInput{Identity: tt.identity})
			assert.Equal(t, tt.release, got.Release)
			assert.Equal(t, tt.because, got.Because)
			if tt.message != "" {
				assert.Equal(t, tt.message, got.Message)
			}
			assert.NotEmpty(t, got.Message)
			for _, banned := range []string{"0012:", "opm.dev/", "operator", "cli", "finalizer", "ServiceAccount"} {
				assert.NotContains(t, got.Message, banned)
			}
		})
	}
}

func TestHoldReasonLiterals(t *testing.T) {
	assert.Equal(t, []lifecycle.HoldReason{
		"prune-disabled", "inventory-empty", "cleanup-complete", "force-orphan",
		"cleanup-incomplete", "cleanup-forbidden", "identity-unavailable",
	}, []lifecycle.HoldReason{
		lifecycle.HoldPruneDisabled, lifecycle.HoldInventoryEmpty, lifecycle.HoldCleanupComplete, lifecycle.HoldForceOrphan,
		lifecycle.HoldCleanupIncomplete, lifecycle.HoldCleanupForbidden, lifecycle.HoldIdentityUnavailable,
	})
	assert.Equal(t, []lifecycle.Identity{"", "missing", "failed"},
		[]lifecycle.Identity{lifecycle.IdentityAvailable, lifecycle.IdentityMissing, lifecycle.IdentityFailed})
}

func TestMayReleaseHoldAfterAdvance(t *testing.T) {
	foreign := inventory.Entry{Kind: "Service", Namespace: "app", Name: "foreign", Version: "v1"}
	p := lifecycle.NewDeletionPlan([]inventory.Entry{web, foreign, ns}, prune, thisUUID)
	answer := (&cluster{
		live: map[inventory.Entry]*unstructured.Unstructured{
			web:     liveOf(web, append(ours, uid("w1"))...),
			foreign: liveOf(foreign, managedBy("helm")),
		},
	}).answer(false)
	_, final := driveToDone(t, p, answer, false)
	require.Len(t, final.Outcomes, 3)
	got := lifecycle.MayReleaseHold(p, final, lifecycle.HoldInput{})
	assert.True(t, got.Release)
	assert.Equal(t, lifecycle.HoldCleanupComplete, got.Because)
}

// TestAPruneOfADroppedObjectLeavesItForTheAdoptingInstance covers the end of
// a hand-over: the instance dropped an object another instance adopted from
// its inventory, so the object is in the stale set, but its UUID label is
// still this instance's until the adopting instance applies it. The prune
// leaves it in place and the hold still releases.
func TestAPruneOfADroppedObjectLeavesItForTheAdoptingInstance(t *testing.T) {
	stale := inventory.StaleSet([]inventory.Entry{web, cm}, []inventory.Entry{cm})
	require.Equal(t, []inventory.Entry{web}, stale)
	p := lifecycle.NewDeletionPlan(stale, prune, thisUUID)

	dropped := liveOf(web, append(ours, uid("w1"))...)
	dropped.SetAnnotations(map[string]string{labels.AnnotationAdopt: otherUUID})
	answer := (&cluster{live: map[inventory.Entry]*unstructured.Unstructured{web: dropped}}).answer(false)

	actions, final := driveToDone(t, p, answer, false)
	assert.NotContains(t, kinds(actions), lifecycle.ActionDelete)
	require.Len(t, final.Outcomes, 1)
	assert.Equal(t, lifecycle.ResultSkipped, final.Outcomes[0].Result)
	assert.Equal(t, ownership.SkipAdoptedElsewhere, final.Outcomes[0].Skip)

	got := lifecycle.MayReleaseHold(p, final, lifecycle.HoldInput{})
	assert.True(t, got.Release)
}
