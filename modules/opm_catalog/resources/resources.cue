// Package resources holds the fixture catalog's resource contracts — the
// minimal core-v2 members the web_app fixture demands (container,
// config-maps). Shapes are trimmed copies of the real catalog_opm members;
// identity fields follow the v2 authoring rules: apiVersion is the contract
// key's only version component, fqn is authored at the definition site from
// the identity package (enhancement 0010 D4/D21).
package resources

import (
	"strings"

	c "opmodel.dev/core@v2"
	id "testing.opmodel.dev/catalogs/opm/identity"
)

/////////////////////////////////////////////////////////////////
//// Container Resource
/////////////////////////////////////////////////////////////////

#ContainerResource: c.#Resource & {
	metadata: {
		name:           "container"
		modulePath:     id.kindPrefix.resources
		apiVersion:     "v1beta1"
		catalogVersion: id.Version
		fqn:            "\(id.kindPrefix.resources)/container@v1beta1"
		description:    "A container definition for workloads"
		labels: "resource.opmodel.dev/category": "workload"
	}

	// The matching key this catalog introduces. Required, so a component must
	// answer it — typically by attaching a workload blueprint.
	matchLabels: "core.opmodel.dev/workload-type"!: "stateless" | "stateful" | "daemon" | "task" | "scheduled-task"

	spec: container: #ContainerSchema
}

#Container: c.#Component & {
	// metadata.labels duplicates the matching key explicitly — transitional
	// invariant 2 of the library-core-retarget change: the kernel's matcher
	// still reads metadata.labels until library-match-labels flips the read
	// to matchLabels.
	metadata: labels: {
		"core.opmodel.dev/workload-type"!: "stateless" | "stateful" | "daemon" | "task" | "scheduled-task"
	}

	#resources: (#ContainerResource.metadata.fqn): #ContainerResource
}

/////////////////////////////////////////////////////////////////
//// Container Schemas
/////////////////////////////////////////////////////////////////

#ContainerSchema: {
	// Name of the container
	name!: string

	// Container image
	image!: #Image

	// Ports exposed by the container
	ports?: [portName=string]: #PortSchema & {name: portName}

	// Environment variables for the container
	env?: [envName=string]: {
		name:   string | *envName
		value?: string
	}
}

// Image specification for container images. Borrowed from timoni's #Image.
#Image: {
	repository!: string
	tag!:        string & strings.MaxRunes(128)
	digest!:     string
	pullPolicy:  *"IfNotPresent" | "Always" | "Never"
	reference:   string

	if digest != "" && tag != "" {
		reference: "\(repository):\(tag)@\(digest)"
	}
	if digest != "" && tag == "" {
		reference: "\(repository)@\(digest)"
	}
	if digest == "" && tag != "" {
		reference: "\(repository):\(tag)"
	}
	if digest == "" && tag == "" {
		reference: "\(repository):latest"
	}
}

// RFC 1123 IANA service name validator.
#IANA_SVC_NAME: string & strings.MinRunes(1) & strings.MaxRunes(15) & =~"^[a-z]([-a-z0-9]{0,13}[a-z0-9])?$"

#PortSchema: {
	name!:        #IANA_SVC_NAME
	targetPort!:  uint & >=1 & <=65535
	protocol:     *"TCP" | "UDP" | "SCTP"
	exposedPort?: uint & >=1 & <=65535
}

/////////////////////////////////////////////////////////////////
//// ConfigMaps Resource
/////////////////////////////////////////////////////////////////

#ConfigMapsResource: c.#Resource & {
	metadata: {
		name:           "config-maps"
		modulePath:     id.kindPrefix.resources
		apiVersion:     "v1beta1"
		catalogVersion: id.Version
		fqn:            "\(id.kindPrefix.resources)/config-maps@v1beta1"
		description:    "A ConfigMap definition for external configuration"
		labels: "resource.opmodel.dev/category": "config"
	}

	spec: configMaps: [cmName=string]: #ConfigMapSchema & {name: string | *cmName}
}

#ConfigMaps: c.#Component & {
	#resources: (#ConfigMapsResource.metadata.fqn): #ConfigMapsResource
}

// ConfigMap specification. `name` is auto-populated from the map key.
#ConfigMapSchema: {
	name!: string
	// Default false so a module that omits `immutable` still renders concrete.
	immutable: bool | *false
	data: [string]: string
}
