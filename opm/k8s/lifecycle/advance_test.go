package lifecycle_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	"github.com/open-platform-model/library/opm/k8s/inventory"
	"github.com/open-platform-model/library/opm/k8s/labels"
	"github.com/open-platform-model/library/opm/k8s/lifecycle"
	"github.com/open-platform-model/library/opm/k8s/ownership"
)

func preconditionOn(u string) *metav1.Preconditions {
	id := types.UID(u)
	return &metav1.Preconditions{UID: &id}
}

func TestAdvance(t *testing.T) {
	notFound := apierrors.NewNotFound(webGR, "web")
	forbidden := apierrors.NewForbidden(webGR, "web", errBoom{})
	wrappedForbidden := fmt.Errorf("deleting: %w", forbidden)
	conflict := apierrors.NewConflict(webGR, "web", errBoom{})
	readWeb := lifecycle.State{Awaiting: lifecycle.AwaitRead}
	deleteWeb := lifecycle.State{Awaiting: lifecycle.AwaitDelete}
	webThenAPI := lifecycle.NewDeletionPlan([]inventory.Entry{web, api}, prune, thisUUID)
	roleWithNS := role
	roleWithNS.Namespace = "app"

	tests := []struct {
		name      string
		plan      lifecycle.DeletionPlan
		state     lifecycle.State
		ev        lifecycle.Event
		wantState lifecycle.State
		wantAct   lifecycle.Action
	}{
		{
			name:  "a protected kind is skipped without a read",
			plan:  lifecycle.NewDeletionPlan([]inventory.Entry{ns}, prune, thisUUID),
			state: lifecycle.State{},
			wantState: lifecycle.State{Next: 1, Outcomes: []lifecycle.Outcome{{
				Step: 0, Result: lifecycle.ResultSkipped, Skip: ownership.SkipSafetyExcluded,
				Message: "Namespace/app is a Namespace, which OPM never deletes; left in place",
			}}},
			wantAct: lifecycle.Action{
				Kind: lifecycle.ActionSkip, Step: 0, Entry: ns, Skip: ownership.SkipSafetyExcluded,
				Message: "Namespace/app is a Namespace, which OPM never deletes; left in place",
			},
		},
		{
			name:      "any other step is read first",
			plan:      webThenAPI,
			state:     lifecycle.State{},
			wantState: readWeb,
			wantAct:   lifecycle.Action{Kind: lifecycle.ActionRead, Step: 0, Entry: web},
		},
		{
			name:      "an owned object is deleted with Foreground propagation and its UID precondition",
			plan:      webThenAPI,
			state:     readWeb,
			ev:        lifecycle.Event{Live: liveOf(web, append(ours, uid("u1"))...)},
			wantState: deleteWeb,
			wantAct: lifecycle.Action{
				Kind: lifecycle.ActionDelete, Step: 0, Entry: web,
				Propagation:   metav1.DeletePropagationForeground,
				Preconditions: preconditionOn("u1"),
			},
		},
		{
			name:  "a foreign object is skipped with the verdict's reason",
			plan:  webThenAPI,
			state: readWeb,
			ev:    lifecycle.Event{Live: liveOf(web, managedBy("helm"))},
			wantState: lifecycle.State{Next: 1, Outcomes: []lifecycle.Outcome{{
				Step: 0, Result: lifecycle.ResultSkipped, Skip: ownership.SkipNotOPMManaged,
				Message: "Deployment/app/web is not managed by OPM; left in place",
			}}},
			wantAct: lifecycle.Action{
				Kind: lifecycle.ActionSkip, Step: 0, Entry: web, Skip: ownership.SkipNotOPMManaged,
				Message: "Deployment/app/web is not managed by OPM; left in place",
			},
		},
		{
			name:  "another instance's object is skipped",
			plan:  webThenAPI,
			state: readWeb,
			ev:    lifecycle.Event{Live: liveOf(web, managedBy(labels.ManagedByController), owner(otherUUID))},
			wantState: lifecycle.State{Next: 1, Outcomes: []lifecycle.Outcome{{
				Step: 0, Result: lifecycle.ResultSkipped, Skip: ownership.SkipOwnerMismatch,
				Message: "Deployment/app/web belongs to module instance u-1, not this one; left in place",
			}}},
			wantAct: lifecycle.Action{
				Kind: lifecycle.ActionSkip, Step: 0, Entry: web, Skip: ownership.SkipOwnerMismatch,
				Message: "Deployment/app/web belongs to module instance u-1, not this one; left in place",
			},
		},
		{
			name:    "a plan that does not prune names done at once",
			plan:    lifecycle.NewDeletionPlan([]inventory.Entry{web, ns}, lifecycle.Policy{}, thisUUID),
			wantAct: lifecycle.Action{Kind: lifecycle.ActionDone, Step: -1},
		},
		{
			name: "a prune built with the zero policy deletes nothing",
			plan: lifecycle.NewDeletionPlan(
				inventory.StaleSet([]inventory.Entry{web, cm}, nil), lifecycle.Policy{ForceOrphan: true}, thisUUID),
			wantAct: lifecycle.Action{Kind: lifecycle.ActionDone, Step: -1},
		},
		{
			name:    "an empty plan names done",
			plan:    lifecycle.NewDeletionPlan(nil, prune, thisUUID),
			wantAct: lifecycle.Action{Kind: lifecycle.ActionDone, Step: -1},
		},
		{
			name:  "a gone object is already absent",
			plan:  webThenAPI,
			state: readWeb,
			ev:    lifecycle.Event{Err: notFound},
			wantState: lifecycle.State{Next: 1, Outcomes: []lifecycle.Outcome{{
				Step: 0, Result: lifecycle.ResultSkipped, Skip: ownership.SkipAlreadyAbsent,
				Message: "Deployment/app/web no longer exists",
			}}},
			wantAct: lifecycle.Action{
				Kind: lifecycle.ActionSkip, Step: 0, Entry: web, Skip: ownership.SkipAlreadyAbsent,
				Message: "Deployment/app/web no longer exists",
			},
		},
		{
			name:  "a read that returns nothing is already absent",
			plan:  webThenAPI,
			state: readWeb,
			wantState: lifecycle.State{Next: 1, Outcomes: []lifecycle.Outcome{{
				Step: 0, Result: lifecycle.ResultSkipped, Skip: ownership.SkipAlreadyAbsent,
				Message: "Deployment/app/web no longer exists",
			}}},
			wantAct: lifecycle.Action{
				Kind: lifecycle.ActionSkip, Step: 0, Entry: web, Skip: ownership.SkipAlreadyAbsent,
				Message: "Deployment/app/web no longer exists",
			},
		},
		{
			name:  "an object gone before its delete is already absent",
			plan:  webThenAPI,
			state: deleteWeb,
			ev:    lifecycle.Event{Err: notFound},
			wantState: lifecycle.State{Next: 1, Outcomes: []lifecycle.Outcome{{
				Step: 0, Result: lifecycle.ResultSkipped, Skip: ownership.SkipAlreadyAbsent,
				Message: "Deployment/app/web no longer exists",
			}}},
			wantAct: lifecycle.Action{
				Kind: lifecycle.ActionSkip, Step: 0, Entry: web, Skip: ownership.SkipAlreadyAbsent,
				Message: "Deployment/app/web no longer exists",
			},
		},
		{
			name:  "a successful delete is recorded and the next step is read",
			plan:  webThenAPI,
			state: deleteWeb,
			wantState: lifecycle.State{Next: 1, Awaiting: lifecycle.AwaitRead, Outcomes: []lifecycle.Outcome{{
				Step: 0, Result: lifecycle.ResultDeleted,
			}}},
			wantAct: lifecycle.Action{Kind: lifecycle.ActionRead, Step: 1, Entry: api},
		},
		{
			name:  "a wrapped Forbidden error is classified",
			plan:  webThenAPI,
			state: deleteWeb,
			ev:    lifecycle.Event{Err: wrappedForbidden},
			wantState: lifecycle.State{Next: 1, Awaiting: lifecycle.AwaitRead, Outcomes: []lifecycle.Outcome{{
				Step: 0, Result: lifecycle.ResultFailed, Failure: lifecycle.FailureForbidden,
				Message: wrappedForbidden.Error(),
			}}},
			wantAct: lifecycle.Action{Kind: lifecycle.ActionRead, Step: 1, Entry: api},
		},
		{
			name:  "a failed precondition is a conflict",
			plan:  webThenAPI,
			state: deleteWeb,
			ev:    lifecycle.Event{Err: conflict},
			wantState: lifecycle.State{Next: 1, Awaiting: lifecycle.AwaitRead, Outcomes: []lifecycle.Outcome{{
				Step: 0, Result: lifecycle.ResultFailed, Failure: lifecycle.FailureConflict,
				Message: conflict.Error(),
			}}},
			wantAct: lifecycle.Action{Kind: lifecycle.ActionRead, Step: 1, Entry: api},
		},
		{
			name:  "a failure does not stop the plan",
			plan:  webThenAPI,
			state: readWeb,
			ev:    lifecycle.Event{Err: errInt},
			wantState: lifecycle.State{Next: 1, Awaiting: lifecycle.AwaitRead, Outcomes: []lifecycle.Outcome{{
				Step: 0, Result: lifecycle.ResultFailed, Failure: lifecycle.FailureError,
				Message: errInt.Error(),
			}}},
			wantAct: lifecycle.Action{Kind: lifecycle.ActionRead, Step: 1, Entry: api},
		},
		{
			name:  "a live object for another step is not deleted",
			plan:  webThenAPI,
			state: readWeb,
			ev:    lifecycle.Event{Live: liveOf(api, append(ours, uid("u2"))...)},
			wantState: lifecycle.State{Next: 1, Awaiting: lifecycle.AwaitRead, Outcomes: []lifecycle.Outcome{{
				Step: 0, Result: lifecycle.ResultFailed, Failure: lifecycle.FailureError,
				Message: "reading Deployment.apps/app/web returned Deployment.apps/app/api; not deleted",
			}}},
			wantAct: lifecycle.Action{Kind: lifecycle.ActionRead, Step: 1, Entry: api},
		},
		{
			name:  "a live object of another group is not deleted",
			plan:  lifecycle.NewDeletionPlan([]inventory.Entry{web}, prune, thisUUID),
			state: readWeb,
			ev: lifecycle.Event{Live: liveOf(inventory.Entry{
				Group: "example.com", Kind: "Deployment", Namespace: "app", Name: "web", Version: "v1",
			}, ours...)},
			wantState: lifecycle.State{Next: 1, Outcomes: []lifecycle.Outcome{{
				Step: 0, Result: lifecycle.ResultFailed, Failure: lifecycle.FailureError,
				Message: "reading Deployment.apps/app/web returned Deployment.example.com/app/web; not deleted",
			}}},
			wantAct: lifecycle.Action{Kind: lifecycle.ActionDone, Step: -1},
		},
		{
			name:  "an empty live namespace or kind is not a mismatch",
			plan:  lifecycle.NewDeletionPlan([]inventory.Entry{roleWithNS}, prune, thisUUID),
			state: readWeb,
			ev: lifecycle.Event{Live: func() *unstructured.Unstructured {
				u := &unstructured.Unstructured{Object: map[string]any{}}
				u.SetName("reader")
				for _, o := range append(ours, uid("u3")) {
					o(u)
				}
				return u
			}()},
			wantState: deleteWeb,
			wantAct: lifecycle.Action{
				Kind: lifecycle.ActionDelete, Step: 0, Entry: roleWithNS,
				Propagation:   metav1.DeletePropagationForeground,
				Preconditions: preconditionOn("u3"),
			},
		},
		{
			name:  "one call can finish two steps",
			plan:  lifecycle.NewDeletionPlan([]inventory.Entry{ns, web}, prune, thisUUID),
			state: readWeb,
			ev:    lifecycle.Event{Err: errInt},
			wantState: lifecycle.State{Next: 2, Outcomes: []lifecycle.Outcome{
				{Step: 0, Result: lifecycle.ResultFailed, Failure: lifecycle.FailureError, Message: errInt.Error()},
				{
					Step: 1, Result: lifecycle.ResultSkipped, Skip: ownership.SkipSafetyExcluded,
					Message: "Namespace/app is a Namespace, which OPM never deletes; left in place",
				},
			}},
			wantAct: lifecycle.Action{
				Kind: lifecycle.ActionSkip, Step: 1, Entry: ns, Skip: ownership.SkipSafetyExcluded,
				Message: "Namespace/app is a Namespace, which OPM never deletes; left in place",
			},
		},
		{
			name:      "a finished plan names done",
			plan:      webThenAPI,
			state:     lifecycle.State{Next: 2, Outcomes: []lifecycle.Outcome{{Step: 0, Result: lifecycle.ResultDeleted}, {Step: 1, Result: lifecycle.ResultDeleted}}},
			wantState: lifecycle.State{Next: 2, Outcomes: []lifecycle.Outcome{{Step: 0, Result: lifecycle.ResultDeleted}, {Step: 1, Result: lifecycle.ResultDeleted}}},
			wantAct:   lifecycle.Action{Kind: lifecycle.ActionDone, Step: -1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var live *unstructured.Unstructured
			if tt.ev.Live != nil {
				live = tt.ev.Live.DeepCopy()
			}
			got, act := mustAdvance(t, tt.plan, tt.state, tt.ev)
			assert.Equal(t, tt.wantState, got)
			assert.Equal(t, tt.wantAct, act)
			if live != nil {
				assert.Equal(t, live, tt.ev.Live, "the live object is not modified")
			}
		})
	}
}

func TestAdvanceRefusesAStateOutsideThePlan(t *testing.T) {
	p := lifecycle.NewDeletionPlan([]inventory.Entry{web}, prune, thisUUID)
	one := []lifecycle.Outcome{{Step: 0, Result: lifecycle.ResultDeleted}}
	two := []lifecycle.Outcome{{Step: 0, Result: lifecycle.ResultDeleted}, {Step: 1, Result: lifecycle.ResultDeleted}}
	tests := []struct {
		name  string
		state lifecycle.State
	}{
		{"a state past the plan", lifecycle.State{Next: 2, Outcomes: two}},
		{"a negative step", lifecycle.State{Next: -1}},
		{"a state awaiting a read after the last step", lifecycle.State{Next: 1, Awaiting: lifecycle.AwaitRead, Outcomes: one}},
		{"a state awaiting a delete after the last step", lifecycle.State{Next: 1, Awaiting: lifecycle.AwaitDelete, Outcomes: one}},
		{"an unknown awaited value", lifecycle.State{Awaiting: "apply"}},
		{"fewer outcomes than finished steps", lifecycle.State{Next: 1}},
		{"more outcomes than finished steps", lifecycle.State{Outcomes: one}},
		{"an outcome out of the plan's order", lifecycle.State{Next: 1, Outcomes: []lifecycle.Outcome{{Step: 9, Result: lifecycle.ResultDeleted}}}},
		{"an outcome with an unknown result", lifecycle.State{Next: 1, Outcomes: []lifecycle.Outcome{{Step: 0, Result: "weird"}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.state
			got, act, err := lifecycle.Advance(p, in, lifecycle.Event{})
			require.Error(t, err)
			assert.Equal(t, in, got, "the input state comes back unchanged")
			assert.Equal(t, lifecycle.Action{}, act)
		})
	}
}

// cluster answers actions as a cluster holding the given live objects would;
// deleting removes an object, and errs overrides the answer for one step's
// action kind.
type cluster struct {
	live map[inventory.Entry]*unstructured.Unstructured
	errs map[string]error
}

func (c *cluster) answer(dryRun bool) responder {
	return func(act lifecycle.Action) lifecycle.Event {
		if err, ok := c.errs[fmt.Sprintf("%s/%s", act.Kind, act.Entry.Name)]; ok {
			return lifecycle.Event{Err: err}
		}
		switch act.Kind {
		case lifecycle.ActionRead:
			if u, ok := c.live[act.Entry]; ok {
				return lifecycle.Event{Live: u}
			}
			return lifecycle.Event{Err: apierrors.NewNotFound(webGR, act.Entry.Name)}
		case lifecycle.ActionDelete:
			if !dryRun {
				delete(c.live, act.Entry)
			}
		}
		return lifecycle.Event{}
	}
}

// mixed is an inventory that reaches every outcome kind.
func mixed() (lifecycle.DeletionPlan, *cluster) {
	gone := inventory.Entry{Kind: "Secret", Namespace: "app", Name: "gone", Version: "v1"}
	foreign := inventory.Entry{Kind: "Service", Namespace: "app", Name: "foreign", Version: "v1"}
	denied := inventory.Entry{Kind: "ServiceAccount", Namespace: "app", Name: "denied", Version: "v1"}
	p := lifecycle.NewDeletionPlan([]inventory.Entry{ns, cm, web, gone, foreign, denied, crd}, prune, thisUUID)
	c := &cluster{
		live: map[inventory.Entry]*unstructured.Unstructured{
			cm:      liveOf(cm, append(ours, uid("c1"))...),
			web:     liveOf(web, append(ours, uid("w1"))...),
			foreign: liveOf(foreign, managedBy("helm")),
			denied:  liveOf(denied, append(ours, uid("d1"))...),
		},
		errs: map[string]error{"delete/denied": apierrors.NewForbidden(webGR, "denied", errBoom{})},
	}
	return p, c
}

func TestAdvanceDrivesAMixedPlanToDone(t *testing.T) {
	p, c := mixed()
	actions, final := driveToDone(t, p, c.answer(false), false)

	results := map[string]lifecycle.Result{}
	for _, o := range final.Outcomes {
		results[p.Steps()[o.Step].Entry.Name] = o.Result
	}
	assert.Equal(t, map[string]lifecycle.Result{
		"app": lifecycle.ResultSkipped, "cfg": lifecycle.ResultDeleted, "web": lifecycle.ResultDeleted,
		"gone": lifecycle.ResultSkipped, "foreign": lifecycle.ResultSkipped, "denied": lifecycle.ResultFailed,
		"widgets.example.com": lifecycle.ResultSkipped,
	}, results)
	assert.Equal(t, p.Len(), final.Next)
	assert.Len(t, final.Outcomes, p.Len())
	for i, o := range final.Outcomes {
		assert.Equal(t, i, o.Step, "outcomes are in step order")
	}
	for _, a := range actions {
		if a.Kind == lifecycle.ActionDelete {
			assert.Equal(t, metav1.DeletePropagationForeground, a.Propagation)
			require.NotNil(t, a.Preconditions)
		}
	}
	assert.Contains(t, kinds(actions), lifecycle.ActionRead)
	assert.Contains(t, kinds(actions), lifecycle.ActionSkip)
	assert.Equal(t, lifecycle.ActionDone, actions[len(actions)-1].Kind)
}

func TestAdvanceDryRun(t *testing.T) {
	p, c := mixed()
	_, final := driveToDone(t, p, c.answer(true), false)
	deleted := 0
	for _, o := range final.Outcomes {
		if o.Result == lifecycle.ResultDeleted {
			deleted++
		}
	}
	assert.Equal(t, 2, deleted, "a dry run answering every delete with Event{} records it as deleted")
	assert.Len(t, c.live, 4, "nothing was removed")
}

func TestAdvanceRoundTripsThroughJSON(t *testing.T) {
	p, c := mixed()
	memActions, memFinal := driveToDone(t, p, c.answer(true), false)
	p2, c2 := mixed()
	jsonActions, jsonFinal := driveToDone(t, p2, c2.answer(true), true)
	assert.Equal(t, memActions, jsonActions)
	assert.Equal(t, memFinal, jsonFinal)
}

func TestAdvanceIsIdempotent(t *testing.T) {
	p, c := mixed()
	answer := c.answer(true)
	var (
		state lifecycle.State
		ev    lifecycle.Event
	)
	for range 64 {
		s1, a1 := mustAdvance(t, p, state, ev)
		s2, a2 := mustAdvance(t, p, state, ev)
		require.Equal(t, s1, s2)
		require.Equal(t, a1, a2)
		if a1.Kind == lifecycle.ActionDone {
			return
		}
		state, ev = s1, answer(a1)
	}
	t.Fatal("the plan did not reach done")
}

func TestAdvanceDoesNotWriteIntoTheCallersOutcomes(t *testing.T) {
	p := lifecycle.NewDeletionPlan([]inventory.Entry{web, api}, prune, thisUUID)
	backing := make([]lifecycle.Outcome, 1, 8)
	backing[0] = lifecycle.Outcome{Step: 0, Result: lifecycle.ResultDeleted}
	in := lifecycle.State{Next: 1, Awaiting: lifecycle.AwaitRead, Outcomes: backing}
	_, _ = mustAdvance(t, p, in, lifecycle.Event{Err: errInt})
	assert.Equal(t, lifecycle.Outcome{}, backing[:2][1], "the caller's spare capacity is untouched")
}

func TestStateJSONGolden(t *testing.T) {
	s := lifecycle.State{
		Next:     2,
		Awaiting: lifecycle.AwaitRead,
		Outcomes: []lifecycle.Outcome{
			{Step: 0, Result: lifecycle.ResultFailed, Failure: lifecycle.FailureForbidden, Message: "denied"},
			{Step: 1, Result: lifecycle.ResultSkipped, Skip: ownership.SkipAlreadyAbsent, Message: "gone"},
		},
	}
	got, err := json.Marshal(s)
	require.NoError(t, err)
	assert.Equal(t, `{"next":2,"awaiting":"read","outcomes":[{"step":0,"result":"failed","failure":"forbidden","message":"denied"},{"step":1,"result":"skipped","skip":"already-absent","message":"gone"}]}`, string(got))

	zero, err := json.Marshal(lifecycle.State{})
	require.NoError(t, err)
	assert.Equal(t, `{"next":0}`, string(zero))
}

func TestLiteralValues(t *testing.T) {
	assert.Equal(t, []lifecycle.ActionKind{"read", "delete", "skip", "done"},
		[]lifecycle.ActionKind{lifecycle.ActionRead, lifecycle.ActionDelete, lifecycle.ActionSkip, lifecycle.ActionDone})
	assert.Equal(t, []lifecycle.Awaiting{"", "read", "delete"},
		[]lifecycle.Awaiting{lifecycle.AwaitNothing, lifecycle.AwaitRead, lifecycle.AwaitDelete})
	assert.Equal(t, []lifecycle.Result{"deleted", "skipped", "failed"},
		[]lifecycle.Result{lifecycle.ResultDeleted, lifecycle.ResultSkipped, lifecycle.ResultFailed})
	assert.Equal(t, []lifecycle.FailureClass{"forbidden", "conflict", "error"},
		[]lifecycle.FailureClass{lifecycle.FailureForbidden, lifecycle.FailureConflict, lifecycle.FailureError})
}

// jsonCopy writes s to JSON and reads it back.
func jsonCopy(t *testing.T, s lifecycle.State) lifecycle.State {
	t.Helper()
	b, err := json.Marshal(s)
	require.NoError(t, err)
	var out lifecycle.State
	require.NoError(t, json.Unmarshal(b, &out))
	return out
}
