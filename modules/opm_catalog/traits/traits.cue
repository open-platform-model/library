// Package traits holds the fixture catalog's trait contracts — the minimal
// core-v2 members the web_app fixture demands (scaling, restart-policy,
// update-strategy, expose, http-route). Every trait states its optionality
// posture as a DEFAULT (core's #TraitOptionalGate refuses both an unstated
// and a pinned posture); all fixture traits are advisory.
package traits

import (
	c "opmodel.dev/core@v2"
	id "testing.opmodel.dev/catalogs/opm/identity"
	res "testing.opmodel.dev/catalogs/opm/resources"
)

/////////////////////////////////////////////////////////////////
//// Scaling
/////////////////////////////////////////////////////////////////

#ScalingTrait: c.#Trait & {
	metadata: {
		name:           "scaling"
		modulePath:     id.kindPrefix.traits
		apiVersion:     "v1beta1"
		catalogVersion: id.Version
		fqn:            "\(id.kindPrefix.traits)/scaling@v1beta1"
		description:    "A trait to specify scaling behavior for a workload"
		labels: "trait.opmodel.dev/category": "workload"
	}

	optional: bool | *true

	appliesTo: [res.#ContainerResource]

	spec: scaling: #ScalingSchema
}

#Scaling: c.#Component & {
	#traits: (#ScalingTrait.metadata.fqn): #ScalingTrait
}

#ScalingSchema: {
	count: int & >=0 & <=1000
}

/////////////////////////////////////////////////////////////////
//// Restart Policy
/////////////////////////////////////////////////////////////////

#RestartPolicyTrait: c.#Trait & {
	metadata: {
		name:           "restart-policy"
		modulePath:     id.kindPrefix.traits
		apiVersion:     "v1beta1"
		catalogVersion: id.Version
		fqn:            "\(id.kindPrefix.traits)/restart-policy@v1beta1"
		description:    "A trait to specify the restart policy for a workload"
		labels: "trait.opmodel.dev/category": "workload"
	}

	optional: bool | *true

	appliesTo: [res.#ContainerResource]

	spec: restartPolicy: #RestartPolicySchema
}

#RestartPolicy: c.#Component & {
	#traits: (#RestartPolicyTrait.metadata.fqn): #RestartPolicyTrait
}

#RestartPolicySchema: "Always" | "OnFailure" | "Never"

/////////////////////////////////////////////////////////////////
//// Update Strategy
/////////////////////////////////////////////////////////////////

#UpdateStrategyTrait: c.#Trait & {
	metadata: {
		name:           "update-strategy"
		modulePath:     id.kindPrefix.traits
		apiVersion:     "v1beta1"
		catalogVersion: id.Version
		fqn:            "\(id.kindPrefix.traits)/update-strategy@v1beta1"
		description:    "A trait to specify the update strategy for a workload"
		labels: "trait.opmodel.dev/category": "workload"
	}

	optional: bool | *true

	appliesTo: [res.#ContainerResource]

	spec: updateStrategy: #UpdateStrategySchema
}

#UpdateStrategy: c.#Component & {
	#traits: (#UpdateStrategyTrait.metadata.fqn): #UpdateStrategyTrait
}

#UpdateStrategySchema: {
	type: "RollingUpdate" | "Recreate" | "OnDelete"
	if type == "RollingUpdate" {
		rollingUpdate?: {
			maxUnavailable?: uint | string
			maxSurge?:       uint | string
		}
	}
}

/////////////////////////////////////////////////////////////////
//// Expose
/////////////////////////////////////////////////////////////////

#ExposeTrait: c.#Trait & {
	metadata: {
		name:           "expose"
		modulePath:     id.kindPrefix.traits
		apiVersion:     "v1beta1"
		catalogVersion: id.Version
		fqn:            "\(id.kindPrefix.traits)/expose@v1beta1"
		description:    "A trait to expose a workload via a service"
		labels: "trait.opmodel.dev/category": "network"
	}

	optional: bool | *true

	appliesTo: [res.#ContainerResource]

	spec: expose: #ExposeSchema
}

#Expose: c.#Component & {
	#traits: (#ExposeTrait.metadata.fqn): #ExposeTrait
}

#ExposeSchema: {
	ports: [portName=string]: res.#PortSchema & {name: portName}
	type: "ClusterIP" | "NodePort" | "LoadBalancer"
}

/////////////////////////////////////////////////////////////////
//// HTTP Route
/////////////////////////////////////////////////////////////////

#HttpRouteTrait: c.#Trait & {
	metadata: {
		name:           "http-route"
		modulePath:     id.kindPrefix.traits
		apiVersion:     "v1beta1"
		catalogVersion: id.Version
		fqn:            "\(id.kindPrefix.traits)/http-route@v1beta1"
		description:    "HTTP routing rules for a workload"
		labels: "trait.opmodel.dev/category": "network"
	}

	optional: bool | *true

	appliesTo: [res.#ContainerResource]

	spec: httpRoute: #HttpRouteSchema
}

#HttpRoute: c.#Component & {
	#traits: (#HttpRouteTrait.metadata.fqn): #HttpRouteTrait
}

#HttpRouteMatchSchema: {
	path?: {
		type:   "PathPrefix" | "Exact" | "RegularExpression"
		value!: string
	}
}

#HttpRouteRuleSchema: {
	backendPort!: uint & >=1 & <=65535
	matches?: [...#HttpRouteMatchSchema]
}

#HttpRouteSchema: {
	hostnames?: [...string]
	rules: [#HttpRouteRuleSchema, ...#HttpRouteRuleSchema]
}
