package lifecycle

import (
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/library/opm/k8s/inventory"
	"github.com/open-platform-model/library/opm/k8s/ownership"
)

// ActionKind is what an [Action] asks the frontend to do.
type ActionKind string

// The four kinds of action [Advance] names. There are no others.
const (
	// ActionRead: read the step's object with the frontend's own client and
	// hand back Event{Live} (nil when not found) or Event{Err}.
	ActionRead ActionKind = "read"
	// ActionDelete: delete the step's object with the action's Propagation
	// and Preconditions, and hand back Event{Err}.
	ActionDelete ActionKind = "delete"
	// ActionSkip: report the step as left in place, with the action's Skip
	// and Message. The returned state has already moved past it; hand back
	// Event{}.
	ActionSkip ActionKind = "skip"
	// ActionDone: the plan is finished.
	ActionDone ActionKind = "done"
)

// Action is the one thing [Advance] asks the frontend to do next.
type Action struct {
	// Kind is the action.
	Kind ActionKind
	// Step is the index into the plan's steps the action is about; -1 for
	// done.
	Step int
	// Entry is that step's object; zero for done.
	Entry inventory.Entry
	// Propagation is the delete's propagation policy, Foreground; set on
	// delete only.
	Propagation metav1.DeletionPropagation
	// Preconditions holds the UID of the object that was judged, so the
	// delete removes that object and not one recreated under its name. Set
	// on delete only; nil when the judged object had no UID.
	Preconditions *metav1.Preconditions
	// Skip is why the object is left in place; set on skip only.
	Skip ownership.SkipReason
	// Message words the skip for a user; set on skip only.
	Message string
}

// Event is what the frontend saw when it performed the last action.
type Event struct {
	// Live is the object a read returned; nil when the read found nothing.
	// Advance does not change it.
	Live *unstructured.Unstructured
	// Err is the action's raw error, nil on success. Advance classifies it;
	// a wrapped API error is classified through the wrapping.
	Err error
}

// Awaiting is what a [State] waits for the frontend to hand back.
type Awaiting string

// The values of [Awaiting].
const (
	// AwaitNothing: the next call names the next step's action.
	AwaitNothing Awaiting = ""
	// AwaitRead: the next call carries the read of the step at Next.
	AwaitRead Awaiting = "read"
	// AwaitDelete: the next call carries the delete of the step at Next.
	AwaitDelete Awaiting = "delete"
)

// Result is how a finished step ended.
type Result string

// The values of [Result].
const (
	// ResultDeleted: the delete succeeded.
	ResultDeleted Result = "deleted"
	// ResultSkipped: the object was left in place; the outcome's Skip says
	// why.
	ResultSkipped Result = "skipped"
	// ResultFailed: a read or delete failed, or a read returned another
	// object; the outcome's Failure says how.
	ResultFailed Result = "failed"
)

// FailureClass classifies a failed step's error.
type FailureClass string

// The values of [FailureClass].
const (
	// FailureForbidden: the API server refused the action as Forbidden.
	FailureForbidden FailureClass = "forbidden"
	// FailureConflict: a Conflict, which is how a failed UID precondition
	// comes back.
	FailureConflict FailureClass = "conflict"
	// FailureError: any other error, including a read that returned another
	// object than the step's.
	FailureError FailureClass = "error"
)

// Outcome records one finished step.
type Outcome struct {
	// Step is the index into the plan's steps.
	Step int `json:"step"`
	// Result is how the step ended.
	Result Result `json:"result"`
	// Skip is why a skipped step was left in place.
	Skip ownership.SkipReason `json:"skip,omitempty"`
	// Failure classifies a failed step's error.
	Failure FailureClass `json:"failure,omitempty"`
	// Message is the skip's wording or the failure's error text.
	Message string `json:"message,omitempty"`
}

// State is how far a plan has advanced. The caller owns it and hands it to
// each call of [Advance]; the zero State is the start of a plan. Its JSON
// encoding is fixed, so a state written out and read back advances exactly
// as the one held in memory.
type State struct {
	// Next is the index of the step being worked on, or the number of steps
	// once every step is finished.
	Next int `json:"next"`
	// Awaiting is what the next call is to carry back for step Next.
	Awaiting Awaiting `json:"awaiting,omitempty"`
	// Outcomes holds one outcome per finished step, in step order. A call
	// only appends to it.
	Outcomes []Outcome `json:"outcomes,omitempty"`
}

// Advance takes the plan, the caller's state and what the caller saw when it
// performed the last action, and returns the next state and the one action to
// perform next. It performs nothing itself.
//
// With a policy that does not prune, it names done at once. Otherwise each
// step is either skipped up front (a safety-excluded kind), or read and then
// judged by [ownership.CanDelete] against the plan's owner UUID: a skip
// verdict is reported as skipped, and a proceed verdict names a delete with
// Foreground propagation and the verdict's UID precondition. A NotFound read
// or delete, or a read that found nothing, is skipped as already absent. A
// live object that names another object than the step is never judged: the
// step fails. Any other error fails the step. A failure does not stop the
// plan, and the next step's action follows in the same call.
//
// Each call appends the outcomes of the steps it finished, in step order, and
// never changes an earlier one or the caller's backing array. It returns an
// error, the input state and the zero Action only when the state cannot
// belong to the plan.
func Advance(plan DeletionPlan, state State, ev Event) (State, Action, error) {
	if err := checkState(plan, state); err != nil {
		return state, Action{}, err
	}
	if !plan.policy.Prune {
		return state, doneAction(), nil
	}
	// Copy the outcomes, so appending never writes into the caller's array.
	next := state
	next.Outcomes = append([]Outcome(nil), state.Outcomes...)
	switch state.Awaiting {
	case AwaitRead:
		next, act := answerRead(plan, next, ev)
		return next, act, nil
	case AwaitDelete:
		next, act := answerDelete(plan, next, ev)
		return next, act, nil
	default:
		next, act := proceed(plan, next)
		return next, act, nil
	}
}

// proceed names the action for the step at s.Next, which awaits nothing.
func proceed(plan DeletionPlan, s State) (State, Action) {
	if s.Next == plan.Len() {
		return s, doneAction()
	}
	i := s.Next
	step := plan.steps[i]
	if step.Skip != "" {
		// CanDelete words the safety exclusion before it needs a live object.
		v := ownership.CanDelete(ownership.DeleteInput{Object: objectOf(step.Entry)})
		return skipStep(s, i, step.Entry, v.Skip, v.Message)
	}
	s.Awaiting = AwaitRead
	return s, Action{Kind: ActionRead, Step: i, Entry: step.Entry}
}

// answerRead judges the read of the step at s.Next.
func answerRead(plan DeletionPlan, s State, ev Event) (State, Action) {
	i := s.Next
	entry := plan.steps[i].Entry
	live := ev.Live
	switch {
	case ev.Err != nil && apierrors.IsNotFound(ev.Err):
		live = nil
	case ev.Err != nil:
		return proceed(plan, finish(s, failed(i, ev.Err)))
	}
	if live != nil {
		if got, ok := sameObject(entry, live); !ok {
			msg := "reading " + describe(objectOf(entry)) + " returned " + got + "; not deleted"
			return proceed(plan, finish(s, Outcome{Step: i, Result: ResultFailed, Failure: FailureError, Message: msg}))
		}
	}
	v := ownership.CanDelete(ownership.DeleteInput{
		Object:       objectOf(entry),
		Live:         live,
		InstanceUUID: plan.ownerUUID,
	})
	if !v.Proceed() {
		return skipStep(s, i, entry, v.Skip, v.Message)
	}
	s.Awaiting = AwaitDelete
	return s, Action{
		Kind:          ActionDelete,
		Step:          i,
		Entry:         entry,
		Propagation:   metav1.DeletePropagationForeground,
		Preconditions: v.Preconditions(),
	}
}

// answerDelete records the delete of the step at s.Next.
func answerDelete(plan DeletionPlan, s State, ev Event) (State, Action) {
	i := s.Next
	entry := plan.steps[i].Entry
	switch {
	case ev.Err == nil:
		return proceed(plan, finish(s, Outcome{Step: i, Result: ResultDeleted}))
	case apierrors.IsNotFound(ev.Err):
		// The object went between the read and the delete.
		v := ownership.CanDelete(ownership.DeleteInput{Object: objectOf(entry)})
		return skipStep(s, i, entry, v.Skip, v.Message)
	default:
		return proceed(plan, finish(s, failed(i, ev.Err)))
	}
}

// skipStep records step i as skipped and names the skip.
func skipStep(s State, i int, entry inventory.Entry, reason ownership.SkipReason, msg string) (State, Action) {
	s = finish(s, Outcome{Step: i, Result: ResultSkipped, Skip: reason, Message: msg})
	return s, Action{Kind: ActionSkip, Step: i, Entry: entry, Skip: reason, Message: msg}
}

// finish appends the outcome of the step at s.Next and moves on.
func finish(s State, o Outcome) State {
	s.Outcomes = append(s.Outcomes, o)
	s.Next++
	s.Awaiting = AwaitNothing
	return s
}

func failed(i int, err error) Outcome {
	return Outcome{Step: i, Result: ResultFailed, Failure: classify(err), Message: err.Error()}
}

// classify sorts a failed action's error. NotFound never reaches it.
func classify(err error) FailureClass {
	switch {
	case apierrors.IsForbidden(err):
		return FailureForbidden
	case apierrors.IsConflict(err):
		return FailureConflict
	default:
		return FailureError
	}
}

// sameObject reports whether live is the object entry names. Only the
// identity fields live sets are compared: an empty field contradicts nothing,
// since a client may return a cluster-scoped object without the namespace an
// inventory recorded, or an object without its apiVersion and kind. It also
// returns live's identity, for the failure message.
func sameObject(entry inventory.Entry, live *unstructured.Unstructured) (string, bool) {
	gvk := live.GroupVersionKind()
	got := ownership.Object{Group: gvk.Group, Kind: gvk.Kind, Namespace: live.GetNamespace(), Name: live.GetName()}
	ok := agrees(live.GetAPIVersion() != "", got.Group, entry.Group) &&
		agrees(got.Kind != "", got.Kind, entry.Kind) &&
		agrees(got.Namespace != "", got.Namespace, entry.Namespace) &&
		agrees(got.Name != "", got.Name, entry.Name)
	return describe(got), ok
}

// agrees reports whether a live identity field matches the entry's, counting
// a field the live object does not set as a match.
func agrees(set bool, live, entry string) bool {
	return !set || live == entry
}

// describe writes an object as Kind/namespace/name, with the group after the
// kind when it has one, so two objects differing only in group read apart.
// An empty kind reads as "object".
func describe(o ownership.Object) string {
	if o.Kind == "" {
		o.Kind = "object"
	}
	if o.Group != "" {
		o.Kind += "." + o.Group
	}
	return o.String()
}

func doneAction() Action { return Action{Kind: ActionDone, Step: -1} }

func checkState(plan DeletionPlan, s State) error {
	if s.Next < 0 || s.Next > plan.Len() {
		return fmt.Errorf("deletion state names step %d, outside a plan of %d steps", s.Next, plan.Len())
	}
	switch s.Awaiting {
	case AwaitNothing:
		return nil
	case AwaitRead, AwaitDelete:
		if s.Next == plan.Len() {
			return fmt.Errorf("deletion state awaits a %s after the last of %d steps", s.Awaiting, plan.Len())
		}
		return nil
	default:
		return fmt.Errorf("deletion state awaits %q, which is not a deletion action", string(s.Awaiting))
	}
}
