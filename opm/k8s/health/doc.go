// Package health judges whether Kubernetes objects have become ready (0012:D3).
//
// [Evaluate] returns a [Status] for one object: whether a Deployment,
// StatefulSet or DaemonSet has rolled out, whether a Job has completed, the
// phase of a PersistentVolumeClaim, and the Ready condition of a custom
// resource; an object with no readiness concept, such as a ConfigMap or a
// custom resource without a Ready condition, reports [Applied].
// [Aggregate] folds the statuses of one instance's objects into one status
// with a ready count and a total, and [ProgressDeadlineExceeded] tells a
// stalled Deployment rollout from one still in progress.
//
// The package is pure. It reads no cluster, waits for nothing and polls
// nothing: the caller fetches each live object with its own client and passes
// it in, and decides how often to ask again. Read each object after the apply
// it should reflect, and read it uncached. A read taken before the apply, or
// served from a cache that has not caught up, shows the previous rollout,
// which the controller has already finished, so Evaluate reports it Ready.
//
// The status strings are stable output: frontends print them and serialise
// them, so a value never changes spelling.
package health

// Maintainer note, kept out of the package doc because it publishes into the
// Library reference: the evaluator was ported from the cli's
// internal/kubernetes/health.go at cli commit bd4d1a7c under ADR-011, with its
// rules and status strings unchanged. Once the cli has deleted its copy, this
// package is the only home of those rules and strings, and a fix lands here
// alone; until then, a fix to either copy is ported to the other by hand. The
// status strings are the cli's `opm instance status -o json|yaml` output, so
// changing one is a breaking change (feat!).
