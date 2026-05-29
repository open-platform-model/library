package resources

import (
	id "opmodel.dev/catalogs/opm/identity"
	"strings"

	c "opmodel.dev/core@v0"
)

/////////////////////////////////////////////////////////////////
//// Container Resource
/////////////////////////////////////////////////////////////////

#ContainerResource: c.#Resource & {
	metadata: {
		modulePath:  "\(id.ModulePath)/resources"
		version:     id.Version
		name:        "container"
		description: "A container definition for workloads"
		labels: {
			"resource.opmodel.dev/category": "workload"
		}
	}

	spec: container: #ContainerSchema
}

#Container: c.#Component & {
	metadata: labels: {
		"core.opmodel.dev/workload-type"!: "stateless" | "stateful" | "daemon" | "task" | "scheduled-task"
	}

	#resources: (#ContainerResource.metadata.fqn): #ContainerResource
}

/////////////////////////////////////////////////////////////////
//// Container Schemas
/////////////////////////////////////////////////////////////////

// Container specification
#ContainerSchema: {
	// Name of the container
	name!: string

	// Container image (e.g., "nginx:latest")
	image!: #Image

	// Ports exposed by the container
	ports?: [portName=string]: #PortSchema & {name: portName}

	// Environment variables for the container
	env?: [envName=string]: #EnvVarSchema & {name: envName}

	// Bulk injection of all keys from ConfigMaps/Secrets as env vars
	envFrom?: [...#EnvFromSource]

	// Command to run in the container
	command?: [...string]

	// Arguments to pass to the command
	args?: [...string]

	// Resource requirements for the container
	resources?: #ResourceRequirementsSchema

	// Volume mounts for the container
	volumeMounts?: [string]: #VolumeMountSchema

	// Probes for health checking. See #ProbeSchema for K8s init container constraints.
	livenessProbe?:  #ProbeSchema
	readinessProbe?: #ProbeSchema
	startupProbe?:   #ProbeSchema

	// Container-level security context. Pod-level constraints (fsGroup,
	// supplementalGroups) belong on the component spec.securityContext field.
	securityContext?: #SecurityContextSchema

	// Pre-stop lifecycle hook command (runs before SIGTERM)
	preStopCommand?: [...string]
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

// Environment variable. Exactly one of value/from/fieldRef/resourceFieldRef must be set.
#EnvVarSchema: {
	name!: string

	value?:            string
	from?:             #Secret
	fieldRef?:         #FieldRefSchema
	resourceFieldRef?: #ResourceFieldRefSchema
}

// Downward API field reference.
#FieldRefSchema: {
	fieldPath!:  string
	apiVersion?: string
}

#FieldRefDefaults: #FieldRefSchema & {
	apiVersion: "v1"
}

// Container resource field reference.
#ResourceFieldRefSchema: {
	resource!:      string
	containerName?: string
	divisor?:       string
}

// Bulk injection source — inject all keys from a ConfigMap or Secret as env vars.
#EnvFromSource: {
	secretRef?: {name!: string}
	configMapRef?: {name!: string}
	prefix?: string
}

// GPU extended resource claim.
#GpuResourceSchema: {
	resource: string
	count:    int & >=1
}

#ResourceRequirementsSchema: {
	requests?: {
		cpu?:    number | string & =~#"^([0-9]+(\.[0-9]+)?|[0-9]+m)$"#
		memory?: number | string & =~"^[0-9]+[MG]i$"
	}
	limits?: {
		cpu?:    number | string & =~#"^([0-9]+(\.[0-9]+)?|[0-9]+m)$"#
		memory?: number | string & =~"^[0-9]+[MG]i$"
	}
	gpu?: #GpuResourceSchema
}

// Probe specification used by liveness/readiness/startup probes.
#ProbeSchema: {
	httpGet?: {
		path!: string
		port!: uint & >0 & <65536
	}
	exec?: {
		command!: [...string]
	}
	tcpSocket?: {
		port!: uint & >0 & <65536
	}
	initialDelaySeconds?: uint | *0
	periodSeconds?:       uint | *10
	timeoutSeconds?:      uint | *1
	successThreshold?:    uint | *1
	failureThreshold?:    uint | *3
}

/////////////////////////////////////////////////////////////////
//// Port Schemas (used by #ContainerSchema and #ExposeSchema)
/////////////////////////////////////////////////////////////////

// RFC 1123 IANA service name validator.
#IANA_SVC_NAME: string & strings.MinRunes(1) & strings.MaxRunes(15) & =~"^[a-z]([-a-z0-9]{0,13}[a-z0-9])?$"

#PortSchema: {
	name!:        #IANA_SVC_NAME
	targetPort!:  uint & >=1 & <=65535
	protocol:     *"TCP" | "UDP" | "SCTP"
	hostIP?:      string
	hostPort?:    uint & >=1 & <=65535
	exposedPort?: uint & >=1 & <=65535
}

/////////////////////////////////////////////////////////////////
//// Security Context Schema
//// Lives here because #ContainerSchema embeds it (per-container
//// securityContext). Pod-level securityContext via #SecurityContextTrait
//// in traits/security_context.cue references this schema.
/////////////////////////////////////////////////////////////////

#SecurityContextSchema: {
	privileged?:   bool
	runAsNonRoot?: bool
	runAsUser?:    int
	runAsGroup?:   int
	fsGroup?:      int
	supplementalGroups?: [...int]
	readOnlyRootFilesystem?:   bool
	allowPrivilegeEscalation?: bool
	capabilities?: {
		add?: [...string]
		drop?: [...string] | ["ALL"]
	}
}
