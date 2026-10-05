package ownership_test

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	"github.com/open-platform-model/library/opm/k8s/labels"
)

const (
	thisUUID  = "u-9"
	otherUUID = "u-1"
)

// liveOpt shapes a live object for a test case.
type liveOpt func(*unstructured.Unstructured)

func managedBy(v string) liveOpt {
	return func(u *unstructured.Unstructured) { setLabel(u, labels.ManagedBy, v) }
}

func uuid(v string) liveOpt {
	return func(u *unstructured.Unstructured) { setLabel(u, labels.ModuleInstanceUUID, v) }
}

func adopt(v string) liveOpt {
	return func(u *unstructured.Unstructured) {
		a := u.GetAnnotations()
		if a == nil {
			a = map[string]string{}
		}
		a[labels.AnnotationAdopt] = v
		u.SetAnnotations(a)
	}
}

func terminating() liveOpt {
	return func(u *unstructured.Unstructured) {
		ts := metav1.Unix(1700000000, 0)
		u.SetDeletionTimestamp(&ts)
	}
}

func uid(v string) liveOpt {
	return func(u *unstructured.Unstructured) { u.SetUID(types.UID(v)) }
}

func rv(v string) liveOpt {
	return func(u *unstructured.Unstructured) { u.SetResourceVersion(v) }
}

func setLabel(u *unstructured.Unstructured, k, v string) {
	l := u.GetLabels()
	if l == nil {
		l = map[string]string{}
	}
	l[k] = v
	u.SetLabels(l)
}

// opm marks a live object as this instance's OPM-managed object.
var opm = []liveOpt{managedBy(labels.ManagedByCLI), uuid(thisUUID)}

func live(apiVersion, kind, ns, name string, opts ...liveOpt) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetAPIVersion(apiVersion)
	u.SetKind(kind)
	u.SetNamespace(ns)
	u.SetName(name)
	for _, o := range opts {
		o(u)
	}
	return u
}

func liveDeployment(opts ...liveOpt) *unstructured.Unstructured {
	return live("apps/v1", "Deployment", "web", "api", opts...)
}
