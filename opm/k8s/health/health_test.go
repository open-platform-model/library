package health

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func makeResource(kind string, conditions []map[string]interface{}) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       kind,
			"metadata": map[string]interface{}{
				"name":      "test-resource",
				"namespace": "default",
			},
		},
	}

	if conditions != nil {
		rawConditions := make([]interface{}, len(conditions))
		for i, c := range conditions {
			rawConditions[i] = c
		}
		obj.Object["status"] = map[string]interface{}{
			"conditions": rawConditions,
		}
	}

	return obj
}

// makeWorkload builds an apps/v1 workload with the given metadata.generation,
// spec and status maps. A nil spec or status is omitted.
func makeWorkload(kind string, generation int64, spec, status map[string]interface{}) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       kind,
			"metadata": map[string]interface{}{
				"name":       "test-workload",
				"namespace":  "default",
				"generation": generation,
			},
		},
	}
	if spec != nil {
		obj.Object["spec"] = spec
	}
	if status != nil {
		obj.Object["status"] = status
	}
	return obj
}

func TestEvaluate_Deployment(t *testing.T) {
	progressDeadline := []interface{}{
		map[string]interface{}{"type": "Progressing", "status": "False", "reason": "ProgressDeadlineExceeded"},
		map[string]interface{}{"type": "Available", "status": "True"},
	}
	tests := []struct {
		name       string
		generation int64
		spec       map[string]interface{}
		status     map[string]interface{}
		expected   Status
	}{
		{
			name:       "fully rolled out",
			generation: 2,
			spec:       map[string]interface{}{"replicas": int64(3)},
			status: map[string]interface{}{
				"observedGeneration": int64(2), "replicas": int64(3),
				"updatedReplicas": int64(3), "availableReplicas": int64(3),
			},
			expected: Ready,
		},
		{
			name:       "stuck upgrade: old ReplicaSet serves, Available=True (cli#228)",
			generation: 2,
			spec:       map[string]interface{}{"replicas": int64(2)},
			status: map[string]interface{}{
				"observedGeneration": int64(2), "replicas": int64(2),
				"updatedReplicas": int64(1), "availableReplicas": int64(1), "readyReplicas": int64(1),
				"conditions": []interface{}{
					map[string]interface{}{"type": "Available", "status": "True", "reason": "MinimumReplicasAvailable"},
					map[string]interface{}{"type": "Progressing", "status": "True", "reason": "ReplicaSetUpdated"},
				},
			},
			expected: NotReady,
		},
		{
			name:       "controller has not observed the latest generation",
			generation: 3,
			spec:       map[string]interface{}{"replicas": int64(1)},
			status: map[string]interface{}{
				"observedGeneration": int64(2), "replicas": int64(1),
				"updatedReplicas": int64(1), "availableReplicas": int64(1),
			},
			expected: NotReady,
		},
		{
			name:       "all replicas updated but not all available",
			generation: 1,
			spec:       map[string]interface{}{"replicas": int64(2)},
			status: map[string]interface{}{
				"observedGeneration": int64(1), "replicas": int64(2),
				"updatedReplicas": int64(2), "availableReplicas": int64(1),
			},
			expected: NotReady,
		},
		{
			name:       "old replicas still terminating",
			generation: 1,
			spec:       map[string]interface{}{"replicas": int64(1)},
			status: map[string]interface{}{
				"observedGeneration": int64(1), "replicas": int64(2),
				"updatedReplicas": int64(1), "availableReplicas": int64(1),
			},
			expected: NotReady,
		},
		{
			name:       "ProgressDeadlineExceeded is never healthy",
			generation: 1,
			spec:       map[string]interface{}{"replicas": int64(1)},
			status: map[string]interface{}{
				"observedGeneration": int64(1), "replicas": int64(1),
				"updatedReplicas": int64(1), "availableReplicas": int64(1),
				"conditions": progressDeadline,
			},
			expected: NotReady,
		},
		{
			name:       "spec.replicas omitted defaults to 1, available",
			generation: 1,
			spec:       map[string]interface{}{},
			status: map[string]interface{}{
				"observedGeneration": int64(1), "replicas": int64(1),
				"updatedReplicas": int64(1), "availableReplicas": int64(1),
			},
			expected: Ready,
		},
		{
			name:       "spec.replicas omitted defaults to 1, none available",
			generation: 1,
			spec:       map[string]interface{}{},
			status:     map[string]interface{}{"observedGeneration": int64(1)},
			expected:   NotReady,
		},
		{
			name:       "scaled to zero",
			generation: 1,
			spec:       map[string]interface{}{"replicas": int64(0)},
			status:     map[string]interface{}{"observedGeneration": int64(1)},
			expected:   Ready,
		},
		{
			name:       "no status yet",
			generation: 1,
			spec:       map[string]interface{}{"replicas": int64(1)},
			status:     nil,
			expected:   NotReady,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resource := makeWorkload("Deployment", tc.generation, tc.spec, tc.status)
			assert.Equal(t, tc.expected, Evaluate(resource))
		})
	}
}

func TestEvaluate_StatefulSet(t *testing.T) {
	// ssStatus builds a StatefulSet status; revisions default to a settled "rev-1".
	ssStatus := func(observed, ready, updated int64, current, update string) map[string]interface{} {
		return map[string]interface{}{
			"observedGeneration": observed, "readyReplicas": ready, "updatedReplicas": updated,
			"currentRevision": current, "updateRevision": update,
		}
	}
	tests := []struct {
		name       string
		generation int64
		spec       map[string]interface{}
		status     map[string]interface{}
		expected   Status
	}{
		{
			name:       "3/3 ready and settled",
			generation: 1,
			spec:       map[string]interface{}{"replicas": int64(3)},
			status:     ssStatus(1, 3, 3, "rev-1", "rev-1"),
			expected:   Ready,
		},
		{
			name:       "0/1 ready",
			generation: 1,
			spec:       map[string]interface{}{"replicas": int64(1)},
			status:     ssStatus(1, 0, 1, "rev-1", "rev-1"),
			expected:   NotReady,
		},
		{
			name:       "1/3 ready",
			generation: 1,
			spec:       map[string]interface{}{"replicas": int64(3)},
			status:     ssStatus(1, 1, 3, "rev-1", "rev-1"),
			expected:   NotReady,
		},
		{
			name:       "rolling update in progress: updateRevision differs from currentRevision",
			generation: 2,
			spec:       map[string]interface{}{"replicas": int64(3), "updateStrategy": map[string]interface{}{"type": "RollingUpdate"}},
			status:     ssStatus(2, 3, 3, "rev-1", "rev-2"),
			expected:   NotReady,
		},
		{
			name:       "rolling update in progress: not all pods updated",
			generation: 2,
			spec:       map[string]interface{}{"replicas": int64(3)},
			status:     ssStatus(2, 3, 1, "rev-2", "rev-2"),
			expected:   NotReady,
		},
		{
			name:       "controller has not observed the latest generation",
			generation: 3,
			spec:       map[string]interface{}{"replicas": int64(1)},
			status:     ssStatus(2, 1, 1, "rev-1", "rev-1"),
			expected:   NotReady,
		},
		{
			name:       "partitioned rollout: revisions differ by design, updated >= replicas-partition",
			generation: 2,
			spec: map[string]interface{}{
				"replicas": int64(3),
				"updateStrategy": map[string]interface{}{
					"type":          "RollingUpdate",
					"rollingUpdate": map[string]interface{}{"partition": int64(2)},
				},
			},
			status:   ssStatus(2, 3, 1, "rev-1", "rev-2"),
			expected: Ready,
		},
		{
			name:       "partitioned rollout: partitioned pods not yet updated",
			generation: 2,
			spec: map[string]interface{}{
				"replicas": int64(3),
				"updateStrategy": map[string]interface{}{
					"type":          "RollingUpdate",
					"rollingUpdate": map[string]interface{}{"partition": int64(2)},
				},
			},
			status:   ssStatus(2, 3, 0, "rev-1", "rev-2"),
			expected: NotReady,
		},
		{
			name:       "OnDelete: revisions differ until pods are deleted, still healthy",
			generation: 2,
			spec: map[string]interface{}{
				"replicas":       int64(2),
				"updateStrategy": map[string]interface{}{"type": "OnDelete"},
			},
			status:   ssStatus(2, 2, 0, "rev-1", "rev-2"),
			expected: Ready,
		},
		{
			name:       "spec.replicas omitted defaults to 1, pod ready",
			generation: 1,
			spec:       map[string]interface{}{},
			status:     ssStatus(1, 1, 1, "rev-1", "rev-1"),
			expected:   Ready,
		},
		{
			name:       "spec.replicas omitted defaults to 1, pod not ready",
			generation: 1,
			spec:       map[string]interface{}{},
			status:     ssStatus(1, 0, 1, "rev-1", "rev-1"),
			expected:   NotReady,
		},
		{
			name:       "scaled to zero",
			generation: 1,
			spec:       map[string]interface{}{"replicas": int64(0)},
			status:     ssStatus(1, 0, 0, "rev-1", "rev-1"),
			expected:   Ready,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resource := makeWorkload("StatefulSet", tc.generation, tc.spec, tc.status)
			assert.Equal(t, tc.expected, Evaluate(resource))
		})
	}
}

func TestEvaluate_DaemonSet(t *testing.T) {
	dsStatus := func(observed, desired, updated, available int64) map[string]interface{} {
		return map[string]interface{}{
			"observedGeneration": observed, "desiredNumberScheduled": desired,
			"updatedNumberScheduled": updated, "numberAvailable": available,
		}
	}
	tests := []struct {
		name       string
		generation int64
		status     map[string]interface{}
		expected   Status
	}{
		{name: "all nodes updated and available", generation: 1, status: dsStatus(1, 3, 3, 3), expected: Ready},
		{name: "no node matches the selector", generation: 1, status: dsStatus(1, 0, 0, 0), expected: Ready},
		{name: "rollout in progress: not all nodes updated", generation: 2, status: dsStatus(2, 3, 1, 3), expected: NotReady},
		{name: "updated pods not yet available", generation: 1, status: dsStatus(1, 3, 3, 2), expected: NotReady},
		{name: "controller has not observed the latest generation", generation: 2, status: dsStatus(1, 3, 3, 3), expected: NotReady},
		{name: "no status yet", generation: 1, status: nil, expected: NotReady},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resource := makeWorkload("DaemonSet", tc.generation, nil, tc.status)
			assert.Equal(t, tc.expected, Evaluate(resource))
		})
	}
}

func TestEvaluate_Job(t *testing.T) {
	tests := []struct {
		name       string
		conditions []map[string]interface{}
		expected   Status
	}{
		{
			name: "Job completed",
			conditions: []map[string]interface{}{
				{"type": "Complete", "status": "True"},
			},
			expected: Complete,
		},
		{
			name: "Job failed",
			conditions: []map[string]interface{}{
				{"type": "Failed", "status": "True"},
			},
			expected: NotReady,
		},
		{
			name: "Failed before Complete in condition order",
			conditions: []map[string]interface{}{
				{"type": "Failed", "status": "True"},
				{"type": "Complete", "status": "True"},
			},
			expected: NotReady,
		},
		{
			name: "Complete before Failed in condition order",
			conditions: []map[string]interface{}{
				{"type": "Complete", "status": "True"},
				{"type": "Failed", "status": "True"},
			},
			expected: Complete,
		},
		{
			name:       "Job in progress",
			conditions: nil,
			expected:   NotReady,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resource := makeResource("Job", tc.conditions)
			assert.Equal(t, tc.expected, Evaluate(resource))
		})
	}
}

func TestEvaluate_CronJob(t *testing.T) {
	resource := makeResource("CronJob", nil)
	assert.Equal(t, Applied, Evaluate(resource))
}

func TestEvaluate_Passive(t *testing.T) {
	// Kinds with no readiness concept report Applied, never Ready (cli#46).
	// PersistentVolumeClaim is intentionally excluded — it has its own evaluatePVCHealth branch.
	passiveResources := []string{
		"ConfigMap", "Secret", "Service",
		"ServiceAccount", "Namespace", "ClusterRole", "ClusterRoleBinding",
		"Role", "RoleBinding", "Ingress", "NetworkPolicy", "PodDisruptionBudget",
		"ResourceQuota", "LimitRange", "StorageClass", "PriorityClass",
	}

	for _, kind := range passiveResources {
		t.Run(kind, func(t *testing.T) {
			resource := makeResource(kind, nil)
			assert.Equal(t, Applied, Evaluate(resource))
			assert.True(t, IsHealthy(Evaluate(resource)))
		})
	}
}

func TestEvaluate_PVC(t *testing.T) {
	tests := []struct {
		name     string
		phase    string
		expected Status
	}{
		{name: "Bound", phase: "Bound", expected: Bound},
		{name: "Pending", phase: "Pending", expected: Status("Pending")},
		{name: "Lost", phase: "Lost", expected: Status("Lost")},
		{name: "no phase (fallback)", phase: "", expected: Ready},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pvc := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "PersistentVolumeClaim",
					"metadata":   map[string]interface{}{"name": "data", "namespace": "ns"},
				},
			}
			if tc.phase != "" {
				_ = unstructured.SetNestedField(pvc.Object, tc.phase, "status", "phase")
			}
			assert.Equal(t, tc.expected, Evaluate(pvc))
		})
	}
}

func TestEvaluate_Custom(t *testing.T) {
	tests := []struct {
		name       string
		conditions []map[string]interface{}
		expected   Status
	}{
		{
			name: "Custom with Ready=True",
			conditions: []map[string]interface{}{
				{"type": "Ready", "status": "True"},
			},
			expected: Ready,
		},
		{
			name: "Custom with Ready=False",
			conditions: []map[string]interface{}{
				{"type": "Ready", "status": "False"},
			},
			expected: NotReady,
		},
		{
			name:       "Custom without Ready condition (passive fallback)",
			conditions: nil,
			expected:   Applied,
		},
		{
			name: "Custom with other conditions but no Ready",
			conditions: []map[string]interface{}{
				{"type": "Synced", "status": "True"},
			},
			expected: Applied,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resource := makeResource("MyCustomResource", tc.conditions)
			assert.Equal(t, tc.expected, Evaluate(resource))
		})
	}
}

func TestIsHealthy(t *testing.T) {
	tests := []struct {
		status   Status
		expected bool
	}{
		{Ready, true},
		{Applied, true},
		{Complete, true},
		{Bound, true},
		{NotReady, false},
		{Unknown, false},
		{Missing, false},
		{Status("Pending"), false},
		{Status("Lost"), false},
		{Status(""), false},
	}
	for _, tc := range tests {
		t.Run(string(tc.status), func(t *testing.T) {
			assert.Equal(t, tc.expected, IsHealthy(tc.status))
		})
	}
}

// evaluateAll evaluates each resource, the way a caller builds the statuses it
// passes to Aggregate.
func evaluateAll(resources []*unstructured.Unstructured) []Status {
	statuses := make([]Status, 0, len(resources))
	for _, r := range resources {
		statuses = append(statuses, Evaluate(r))
	}
	return statuses
}

func TestAggregate_AppliedCountsAsHealthy(t *testing.T) {
	resources := []*unstructured.Unstructured{
		makeResource("ClusterRole", nil),
		makeResource("ServiceAccount", nil),
	}
	status, ready, total := Aggregate(evaluateAll(resources), 0)
	assert.Equal(t, Ready, status)
	assert.Equal(t, 2, ready)
	assert.Equal(t, 2, total)
}

// The second argument counts every tracked resource that is missing or could
// not be read; each counts toward the total and not toward ready.
func TestAggregate_UnhealthyCountIncludesUnreadable(t *testing.T) {
	resources := []*unstructured.Unstructured{
		makeResource("ConfigMap", nil),
		makeResource("ServiceAccount", nil),
	}
	missing, unreadable := 1, 2
	status, ready, total := Aggregate(evaluateAll(resources), missing+unreadable)
	assert.Equal(t, NotReady, status)
	assert.Equal(t, 2, ready)
	assert.Equal(t, 5, total)
}

// The status strings are the cli's `opm instance status` output; each
// constant is pinned to its literal so a rename cannot change them.
func TestStatusStrings(t *testing.T) {
	tests := []struct {
		status Status
		want   string
	}{
		{Ready, "Ready"},
		{NotReady, "NotReady"},
		{Complete, "Complete"},
		{Unknown, "Unknown"},
		{Missing, "Missing"},
		{Applied, "Applied"},
		{Bound, "Bound"},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			assert.Equal(t, tc.want, string(tc.status))
		})
	}
}

func TestAggregate(t *testing.T) {
	tests := []struct {
		name       string
		statuses   []Status
		unhealthy  int
		wantStatus Status
		wantReady  int
		wantTotal  int
	}{
		{name: "all healthy", statuses: []Status{Ready, Complete, Bound}, wantStatus: Ready, wantReady: 3, wantTotal: 3},
		{name: "Applied counts as healthy", statuses: []Status{Applied, Ready}, wantStatus: Ready, wantReady: 2, wantTotal: 2},
		{name: "missing and unreadable objects count against the aggregate", statuses: []Status{Ready, Ready, Ready}, unhealthy: 2, wantStatus: NotReady, wantReady: 3, wantTotal: 5},
		{name: "one unhealthy status", statuses: []Status{Ready, NotReady, Status("Pending")}, wantStatus: NotReady, wantReady: 1, wantTotal: 3},
		{name: "unhealthy objects only", unhealthy: 2, wantStatus: NotReady, wantReady: 0, wantTotal: 2},
		{name: "nothing to aggregate is unknown", wantStatus: Unknown, wantReady: 0, wantTotal: 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, ready, total := Aggregate(tc.statuses, tc.unhealthy)
			assert.Equal(t, tc.wantStatus, status)
			assert.Equal(t, tc.wantReady, ready)
			assert.Equal(t, tc.wantTotal, total)
		})
	}
}

// quickInstanceHealth is the cli's QuickInstanceHealth loop at cli commit
// bd4d1a7c, kept here only to show that Aggregate over Evaluate gives the
// same result for the same inputs.
func quickInstanceHealth(resources []*unstructured.Unstructured, unhealthyCount int) (Status, int, int) {
	total := len(resources) + unhealthyCount
	if total == 0 {
		return Unknown, 0, 0
	}
	ready := 0
	for _, res := range resources {
		if IsHealthy(Evaluate(res)) {
			ready++
		}
	}
	if ready == total {
		return Ready, ready, total
	}
	return NotReady, ready, total
}

func TestAggregate_MatchesTheCLILoop(t *testing.T) {
	pvc := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1", "kind": "PersistentVolumeClaim",
		"metadata": map[string]interface{}{"name": "data", "namespace": "ns"},
		"status":   map[string]interface{}{"phase": "Pending"},
	}}
	ready := makeWorkload("Deployment", 1, map[string]interface{}{"replicas": int64(1)}, map[string]interface{}{
		"observedGeneration": int64(1), "replicas": int64(1),
		"updatedReplicas": int64(1), "availableReplicas": int64(1),
	})
	sets := map[string][]*unstructured.Unstructured{
		"empty":   nil,
		"passive": {makeResource("ConfigMap", nil), makeResource("ClusterRole", nil)},
		"mixed": {
			ready,
			makeWorkload("StatefulSet", 2, map[string]interface{}{"replicas": int64(3)}, map[string]interface{}{"observedGeneration": int64(1)}),
			makeResource("Job", []map[string]interface{}{{"type": "Complete", "status": "True"}}),
			makeResource("MyCustomResource", []map[string]interface{}{{"type": "Ready", "status": "False"}}),
			pvc,
		},
		"healthy": {ready, makeResource("Secret", nil), makeResource("CronJob", nil)},
	}
	for name, resources := range sets {
		for _, n := range []int{0, 1, 3} {
			wantStatus, wantReady, wantTotal := quickInstanceHealth(resources, n)
			status, ready, total := Aggregate(evaluateAll(resources), n)
			assert.Equal(t, wantStatus, status, "%s, unhealthy=%d", name, n)
			assert.Equal(t, wantReady, ready, "%s, unhealthy=%d", name, n)
			assert.Equal(t, wantTotal, total, "%s, unhealthy=%d", name, n)
		}
	}
}

func TestProgressDeadlineExceeded(t *testing.T) {
	progressing := func(reason string) []interface{} {
		return []interface{}{
			map[string]interface{}{"type": "Progressing", "status": "False", "reason": reason},
			map[string]interface{}{"type": "Available", "status": "True"},
		}
	}
	tests := []struct {
		name       string
		kind       string
		generation int64
		status     map[string]interface{}
		want       bool
	}{
		{
			name: "stalled and observed", kind: "Deployment", generation: 2,
			status: map[string]interface{}{"observedGeneration": int64(2), "conditions": progressing("ProgressDeadlineExceeded")},
			want:   true,
		},
		{
			name: "condition left from the previous generation", kind: "Deployment", generation: 3,
			status: map[string]interface{}{"observedGeneration": int64(2), "conditions": progressing("ProgressDeadlineExceeded")},
			want:   false,
		},
		{
			name: "still rolling out", kind: "Deployment", generation: 2,
			status: map[string]interface{}{
				"observedGeneration": int64(2), "replicas": int64(2),
				"updatedReplicas": int64(1), "availableReplicas": int64(1),
				"conditions": []interface{}{
					map[string]interface{}{"type": "Progressing", "status": "True", "reason": "ReplicaSetUpdated"},
				},
			},
			want: false,
		},
		{
			name: "rolled out", kind: "Deployment", generation: 1,
			status: map[string]interface{}{"observedGeneration": int64(1), "conditions": progressing("NewReplicaSetAvailable")},
			want:   false,
		},
		{name: "no status yet", kind: "Deployment", generation: 1, want: false},
		{
			name: "not a Deployment", kind: "StatefulSet", generation: 1,
			status: map[string]interface{}{"observedGeneration": int64(1), "conditions": progressing("ProgressDeadlineExceeded")},
			want:   false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			obj := makeWorkload(tc.kind, tc.generation, map[string]interface{}{"replicas": int64(2)}, tc.status)
			assert.Equal(t, tc.want, ProgressDeadlineExceeded(obj))
		})
	}
}

// Evaluate does not tell a stall from a rollout in progress: both are
// NotReady. ProgressDeadlineExceeded is what separates them.
func TestProgressDeadlineExceeded_EvaluateFoldsBothIntoNotReady(t *testing.T) {
	stalled := makeWorkload("Deployment", 1, map[string]interface{}{"replicas": int64(1)}, map[string]interface{}{
		"observedGeneration": int64(1), "replicas": int64(1),
		"updatedReplicas": int64(1), "availableReplicas": int64(1),
		"conditions": []interface{}{
			map[string]interface{}{"type": "Progressing", "status": "False", "reason": "ProgressDeadlineExceeded"},
		},
	})
	rolling := makeWorkload("Deployment", 1, map[string]interface{}{"replicas": int64(2)}, map[string]interface{}{
		"observedGeneration": int64(1), "replicas": int64(2),
		"updatedReplicas": int64(1), "availableReplicas": int64(1),
		"conditions": []interface{}{
			map[string]interface{}{"type": "Progressing", "status": "True", "reason": "ReplicaSetUpdated"},
		},
	})
	assert.Equal(t, NotReady, Evaluate(stalled))
	assert.Equal(t, NotReady, Evaluate(rolling))
	assert.True(t, ProgressDeadlineExceeded(stalled))
	assert.False(t, ProgressDeadlineExceeded(rolling))
}
