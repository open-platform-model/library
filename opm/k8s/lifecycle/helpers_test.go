package lifecycle_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"github.com/open-platform-model/library/opm/k8s/inventory"
	"github.com/open-platform-model/library/opm/k8s/labels"
	"github.com/open-platform-model/library/opm/k8s/lifecycle"
)

const (
	thisUUID  = "u-9"
	otherUUID = "u-1"
)

var (
	prune = lifecycle.Policy{Prune: true}

	cm     = inventory.Entry{Kind: "ConfigMap", Namespace: "app", Name: "cfg", Version: "v1"}
	web    = inventory.Entry{Group: "apps", Kind: "Deployment", Namespace: "app", Name: "web", Version: "v1"}
	api    = inventory.Entry{Group: "apps", Kind: "Deployment", Namespace: "app", Name: "api", Version: "v1"}
	ns     = inventory.Entry{Kind: "Namespace", Name: "app", Version: "v1"}
	crd    = inventory.Entry{Group: "apiextensions.k8s.io", Kind: "CustomResourceDefinition", Name: "widgets.example.com", Version: "v1"}
	hook   = inventory.Entry{Group: "admissionregistration.k8s.io", Kind: "ValidatingWebhookConfiguration", Name: "guard", Version: "v1"}
	role   = inventory.Entry{Group: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "reader", Version: "v1"}
	webGR  = schema.GroupResource{Group: "apps", Resource: "deployments"}
	errInt = apierrors.NewInternalError(errBoom{})
)

type errBoom struct{}

func (errBoom) Error() string { return "boom" }

// liveOpt shapes a live object for a test case.
type liveOpt func(*unstructured.Unstructured)

func managedBy(v string) liveOpt {
	return func(u *unstructured.Unstructured) { setLabel(u, labels.ManagedBy, v) }
}

func owner(v string) liveOpt {
	return func(u *unstructured.Unstructured) { setLabel(u, labels.ModuleInstanceUUID, v) }
}

func uid(v string) liveOpt {
	return func(u *unstructured.Unstructured) { u.SetUID(types.UID(v)) }
}

// ours is a live object this instance manages.
var ours = []liveOpt{managedBy(labels.ManagedByCLI), owner(thisUUID)}

func setLabel(u *unstructured.Unstructured, k, v string) {
	l := u.GetLabels()
	if l == nil {
		l = map[string]string{}
	}
	l[k] = v
	u.SetLabels(l)
}

// liveOf is the live object an entry names, shaped by opts.
func liveOf(e inventory.Entry, opts ...liveOpt) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{Group: e.Group, Version: e.Version, Kind: e.Kind})
	u.SetNamespace(e.Namespace)
	u.SetName(e.Name)
	for _, o := range opts {
		o(u)
	}
	return u
}

// mustAdvance calls Advance and fails the test on an error.
func mustAdvance(t *testing.T, p lifecycle.DeletionPlan, s lifecycle.State, ev lifecycle.Event) (lifecycle.State, lifecycle.Action) {
	t.Helper()
	next, act, err := lifecycle.Advance(p, s, ev)
	require.NoError(t, err)
	return next, act
}

// responder answers an action the way a cluster would.
type responder func(act lifecycle.Action) lifecycle.Event

// driveToDone advances p from the zero state, answering each action with
// answer, until done. It asserts the protocol's invariants on every call and
// returns the actions in order and the final state. When roundTrip is set,
// the state is re-decoded from JSON before every call.
func driveToDone(t *testing.T, p lifecycle.DeletionPlan, answer responder, roundTrip bool) ([]lifecycle.Action, lifecycle.State) {
	t.Helper()
	steps := p.Steps()
	var (
		state   lifecycle.State
		ev      lifecycle.Event
		actions []lifecycle.Action
	)
	for range 4*len(steps) + 2 {
		if roundTrip {
			state = jsonCopy(t, state)
		}
		var before *unstructured.Unstructured
		if ev.Live != nil {
			before = ev.Live.DeepCopy()
		}
		next, act := mustAdvance(t, p, state, ev)
		if before != nil {
			require.Equal(t, before, ev.Live, "Advance does not change the live object")
		}
		actions = append(actions, act)
		switch act.Kind {
		case lifecycle.ActionDone:
			require.Equal(t, -1, act.Step)
			require.Equal(t, inventory.Entry{}, act.Entry)
			return actions, next
		case lifecycle.ActionRead, lifecycle.ActionDelete, lifecycle.ActionSkip:
			require.GreaterOrEqual(t, act.Step, 0)
			require.Less(t, act.Step, len(steps))
			require.Equal(t, steps[act.Step].Entry, act.Entry, "every action names a step of the plan")
		default:
			t.Fatalf("unknown action kind %q", act.Kind)
		}
		state = next
		ev = answer(act)
	}
	t.Fatal("the plan did not reach done")
	return nil, lifecycle.State{}
}

// kinds lists the action kinds in order, for compact assertions.
func kinds(actions []lifecycle.Action) []lifecycle.ActionKind {
	out := make([]lifecycle.ActionKind, len(actions))
	for i, a := range actions {
		out[i] = a.Kind
	}
	return out
}
