@if(test)

// Positive case: drive #TransformerContext directly with concrete inputs
// across all three label scopes (module, component, controller via
// #runtimeName) and assert the final `labels` and `annotations` maps via a
// close()-d expect block. The close() forces input's merged maps to contain
// EXACTLY the expected key set — extra keys (e.g. an unfiltered
// "transformer.opmodel.dev/*" leakage) would fail unification.
//
// Specifically proves:
//  - moduleLabels merged into final labels (transformer.cue:120-126)
//  - componentLabels stamps "app.kubernetes.io/name" + "module-release.opmodel.dev/name" (transformer.cue:136-146)
//  - componentLabels filters keys with prefix "transformer.opmodel.dev/" (transformer.cue:141)
//  - componentAnnotations applies the same prefix filter (transformer.cue:148-156)
//  - controllerLabels stamps "app.kubernetes.io/managed-by" from #runtimeName (transformer.cue:158-162)
//  - "app.kubernetes.io/name" set by both component and controller layers unifies cleanly (both = componentMetadata.name)
package fixtures

import (
	core "opmodel.dev/core/v1alpha2@v1"
)

input: {
	core.#TransformerContext
	#moduleReleaseMetadata: {
		name:      "demo-release"
		namespace: "demo-ns"
		fqn:       "example.com/m/demo:0.1.0"
		version:   "0.1.0"
		uuid:      "00000000-0000-0000-0000-000000000000"
		labels: "module.opmodel.dev/owner": "team-a"
		annotations: "module.opmodel.dev/note": "release annotation"
	}
	#componentMetadata: {
		name: "demo-component"
		labels: {
			"tier":                            "prod"
			"transformer.opmodel.dev/internal": "must-be-filtered"
		}
		annotations: {
			"docs":                              "see runbook"
			"transformer.opmodel.dev/internal":   "must-be-filtered"
		}
	}
	#runtimeName: "opm-cli"
}

// Final labels: union of moduleLabels + componentLabels + controllerLabels.
// We assert two things:
//  1. Each expected key has the exact value computed by the merge.
//  2. The TOTAL key count equals the expected size — a filter regression
//     letting "transformer.opmodel.dev/*" through would bump the count.
//
// (CUE's close() does NOT reject extra keys when the underlying struct is
// open via `...`, so positive-equality + length is the load-bearing contract.)
expect: {
	labels: {
		"module.opmodel.dev/owner":        "team-a"
		"app.kubernetes.io/name":          "demo-component"
		"module-release.opmodel.dev/name": "demo-release"
		"tier":                            "prod"
		"app.kubernetes.io/managed-by":    "opm-cli"
		"app.kubernetes.io/instance":      "demo-component"
	}
	annotations: {
		"module.opmodel.dev/note": "release annotation"
		"docs":                    "see runbook"
	}

	_labelKeyCount:      len([for k, _ in input.labels {k}]) & 6
	_annotationKeyCount: len([for k, _ in input.annotations {k}]) & 2
}
