// Package blueprints sits FLAT under the catalog root — one path segment per
// kind, no grouping directory beneath it (enhancement 0010 D42). This is what
// moves the web_app fixture's import from …/blueprints/workload to
// …/blueprints.
package blueprints

import (
	c "opmodel.dev/core@v2"
	id "testing.opmodel.dev/catalogs/opm/identity"
	res "testing.opmodel.dev/catalogs/opm/resources"
	tr "testing.opmodel.dev/catalogs/opm/traits"
)

#StatelessWorkloadSchema: {
	container:       res.#ContainerSchema
	scaling?:        tr.#ScalingSchema
	restartPolicy?:  tr.#RestartPolicySchema
	updateStrategy?: tr.#UpdateStrategySchema
}

#StatelessWorkloadBlueprint: c.#Blueprint & {
	metadata: {
		name:           "stateless-workload"
		modulePath:     id.kindPrefix.blueprints
		apiVersion:     "v1beta1"
		catalogVersion: id.Version
		fqn:            "\(id.kindPrefix.blueprints)/stateless-workload@v1beta1"
		description:    "A stateless workload with no requirement for stable identity or storage"
	}

	// Answers the container resource's required matching key, which is what
	// completes the matching identity of every component attaching this
	// blueprint.
	matchLabels: "core.opmodel.dev/workload-type": "stateless"

	composedResources: [res.#ContainerResource]

	composedTraits: [
		tr.#ScalingTrait,
		tr.#RestartPolicyTrait,
		tr.#UpdateStrategyTrait,
	]

	spec: statelessWorkload: #StatelessWorkloadSchema
}

#StatelessWorkload: c.#Component & {
	// Explicit duplication of the matching key into metadata.labels —
	// transitional invariant 2: the kernel's matcher reads metadata.labels
	// until library-match-labels flips the read to matchLabels.
	metadata: labels: "core.opmodel.dev/workload-type": "stateless"

	#blueprints: (#StatelessWorkloadBlueprint.metadata.fqn): #StatelessWorkloadBlueprint

	res.#Container
	tr.#Scaling
	tr.#RestartPolicy
	tr.#UpdateStrategy

	// Override spec to propagate values from statelessWorkload.
	//
	// The `if … != _|_` guards MUST stay hoisted at component level (outside
	// the spec block): a guard whose condition references a nested non-scalar
	// field from *inside* the spec struct trips the v0.17.x CUE evaluator
	// closedness regression that rejects the guarded field as "field not
	// allowed". See docs/design/cue-closedness-regression-alpha2.md.
	spec: {
		statelessWorkload: #StatelessWorkloadSchema
		container:         spec.statelessWorkload.container
	}
	if spec.statelessWorkload.scaling != _|_ {
		spec: scaling: spec.statelessWorkload.scaling
	}
	if spec.statelessWorkload.restartPolicy != _|_ {
		spec: restartPolicy: spec.statelessWorkload.restartPolicy
	}
	if spec.statelessWorkload.updateStrategy != _|_ {
		spec: updateStrategy: spec.statelessWorkload.updateStrategy
	}
}
