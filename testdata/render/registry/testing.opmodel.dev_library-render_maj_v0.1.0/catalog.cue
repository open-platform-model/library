// Render fixture catalog maj, major v0 (0.1.0): the first of two majors of
// one catalog that list the same contract keys. It lists the
// catalog-fulfilled container resource, the expose trait and the
// provider-fulfilled backup trait, and ships deployment and service
// transformers stamping a "built-by" annotation so a test can tell which
// transformer rendered an object. Beside maj 1.4.0, which lists the same
// three keys, the platform's contract inventory reports each key as a
// collision and the render refuses the platform.
package maj

import c "opmodel.dev/core@v2"

c.#Catalog

metadata: {
	modulePath:  "testing.opmodel.dev/library-render/maj@v0"
	version:     "0.1.0"
	description: "colliding-majors render fixture catalog"
}

_version: "0.1.0"
_res:     "testing.opmodel.dev/library-render/maj/resources"
_resmp:   "testing.opmodel.dev/library-render/maj/resources"
_traits:  "testing.opmodel.dev/library-render/maj/traits"
_trmp:    "testing.opmodel.dev/library-render/maj/traits"
_tx:      "testing.opmodel.dev/library-render/maj/transformers"

#ContainerResource: c.#Resource & {
	metadata: {
		name: "container", modulePath:             "\(_resmp)/v1", apiVersion: "v1", catalogVersion: _version
		fqn:  "\(_res)/container@v1", description: "container"
	}
	matchLabels: "render.test/workload": "stateless"
	spec: container: {image!: string, port: int | *8080}
}
#ExposeTrait: c.#Trait & {
	metadata: {
		name: "expose", modulePath:                "\(_trmp)/v1", apiVersion: "v1", catalogVersion: _version
		fqn:  "\(_traits)/expose@v1", description: "expose"
	}
	optional: bool | *true
	spec: expose: port: int | *80
	appliesTo: [#ContainerResource]
}
#BackupTrait: c.#Trait & {
	metadata: {
		name: "backup", modulePath:                "\(_trmp)/v1", apiVersion: "v1", catalogVersion: _version
		fqn:  "\(_traits)/backup@v1", description: "provider-fulfilled backup"
	}
	optional:   bool | *false
	fulfilment: "provider"
	spec: backup: schedule?: string
	appliesTo: [#ContainerResource]
}

#resources: {
	(#ContainerResource.metadata.fqn): #ContainerResource
}
#traits: {
	(#ExposeTrait.metadata.fqn): #ExposeTrait
	(#BackupTrait.metadata.fqn): #BackupTrait
}

#transformers: {
	"\(_tx)/deployment-transformer@\(_version)": {
		metadata: {name: "deployment-transformer", fqn: "\(_tx)/deployment-transformer@\(_version)", description: "deployment"}
		requiredLabels: "render.test/workload":               "stateless"
		requiredResources: (#ContainerResource.metadata.fqn): #ContainerResource
		optionalTraits: (#ExposeTrait.metadata.fqn):          #ExposeTrait
		#transform: {#component: _, output: {apiVersion: "apps/v1", kind: "Deployment", metadata: name: #component.#names.resourceName, metadata: annotations: "built-by": "\(_version)"}}
	}
	"\(_tx)/service-transformer@\(_version)": {
		metadata: {name: "service-transformer", fqn: "\(_tx)/service-transformer@\(_version)", description: "service"}
		requiredResources: (#ContainerResource.metadata.fqn): #ContainerResource
		requiredTraits: (#ExposeTrait.metadata.fqn):          #ExposeTrait
		#transform: {#component: _, output: {apiVersion: "v1", kind: "Service", metadata: name: #component.#names.resourceName, metadata: annotations: "built-by": "\(_version)"}}
	}
}
