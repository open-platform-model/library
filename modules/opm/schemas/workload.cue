package schemas

import (
	"strings"
)

/////////////////////////////////////////////////////////////////
//// Container Schemas
/////////////////////////////////////////////////////////////////

// Container specification
#ContainerSchema: {
	// Name of the container
	name!: string

	// Container image (e.g., "nginx:latest")
	image!: #Image

	// Image pull policy
	// Deprecated: use #Image.pullPolicy instead.
	// imagePullPolicy: "Always" | "IfNotPresent" | "Never" | *"IfNotPresent"

	// Ports exposed by the container
	ports?: [portName=string]: #PortSchema & {name: portName} // Name is automatically set to the key in the ports map

	// Environment variables for the container
	env?: [envName=string]: #EnvVarSchema & {name: envName} // Name is automatically set to the key in the env map

	// Bulk injection of all keys from ConfigMaps/Secrets as env vars
	envFrom?: [...#EnvFromSource]

	// Command to run in the container
	command?: [...string]

	// Arguments to pass to the command
	args?: [...string]

	// Resource requirements for the container
	resources?: #ResourceRequirementsSchema

	// Volume mounts for the container
	volumeMounts?: [string]: #VolumeMountSchema // Name is automatically set to the key in the volumeMounts map

	// Probes for health checking (primarily for sidecar containers).
	// See #ProbeSchema for K8s init container constraints.
	livenessProbe?:  #ProbeSchema
	readinessProbe?: #ProbeSchema
	startupProbe?:   #ProbeSchema

	// Container-level security context.
	// Applies per-container constraints: privileged, capabilities, runAsUser, etc.
	// Pod-level constraints (fsGroup, supplementalGroups) belong on the component
	// spec.securityContext field instead.
	securityContext?: #SecurityContextSchema

	// Pre-stop lifecycle hook command (runs before SIGTERM)
	preStopCommand?: [...string]
}

// Image specification for container images, used in #ContainerSchema
// Borrowed from timoni's #Image schema
#Image: {

	// Repository is the address of a container registry repository.
	// An image repository is made up of slash-separated name components, optionally
	// prefixed by a registry hostname and port in the format [HOST[:PORT_NUMBER]/]PATH.
	repository!: string

	// Tag identifies an image in the repository.
	// A tag name may contain lowercase and uppercase characters, digits, underscores, periods and dashes.
	// A tag name may not start with a period or a dash and may contain a maximum of 128 characters.
	tag!: string & strings.MaxRunes(128)

	// Digest uniquely and immutably identifies an image in the repository.
	// Spec: https://github.com/opencontainers/image-spec/blob/v1.1.1/descriptor.md#digests.
	digest!: string

	// PullPolicy defines the pull policy for the image.
	// By default, it is set to IfNotPresent.
	pullPolicy: *"IfNotPresent" | "Always" | "Never"

	// Reference is the image address computed from repository, tag and digest
	// in the format [REPOSITORY]:[TAG]@[DIGEST].
	reference: string

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

// Environment variable specification.
// Exactly one of value, from, fieldRef, or resourceFieldRef must be set.
//   value:            inline literal (non-sensitive config)
//   from:             reference to a #Secret in #config (sensitive)
//   fieldRef:         downward API (pod metadata)
//   resourceFieldRef: container resource limits/requests
#EnvVarSchema: {
	name!: string

	// Source — exactly one must be set
	value?:            string                  // inline literal (non-sensitive)
	from?:             #Secret                 // reference to a #Secret in #config
	fieldRef?:         #FieldRefSchema         // downward API
	resourceFieldRef?: #ResourceFieldRefSchema // container resource fields
}

// Downward API field reference — expose pod/container metadata as env vars.
// Supported fieldPath values: metadata.name, metadata.namespace, metadata.uid,
// metadata.labels['<KEY>'], metadata.annotations['<KEY>'], spec.nodeName,
// spec.serviceAccountName, status.podIP, status.hostIP
#FieldRefSchema: {
	fieldPath!:  string
	apiVersion?: string | *"v1"
}

// Container resource field reference — expose resource limits/requests as env vars.
// Supported resource values: limits.cpu, limits.memory, limits.ephemeral-storage,
// requests.cpu, requests.memory, requests.ephemeral-storage
#ResourceFieldRefSchema: {
	resource!:      string
	containerName?: string // defaults to current container
	divisor?:       string // e.g., "1m" for millicores, "1Mi" for mebibytes
}

// Bulk injection source — inject all keys from a ConfigMap or Secret as env vars.
// Exactly one of secretRef or configMapRef must be set.
#EnvFromSource: {
	secretRef?: {name!: string} // K8s Secret name (= $secretName)
	configMapRef?: {name!: string} // K8s ConfigMap name
	prefix?: string // optional prefix for injected keys
}

// #GpuResourceSchema specifies a GPU extended resource claim for a container.
// The resource key must match the name reported by the device plugin on the node.
// Kubernetes requires extended resource requests == limits, so a single
// top-level field is used rather than splitting across requests/limits.
#GpuResourceSchema: {
	// Resource key as exposed by the device plugin on the node. Examples:
	//   "nvidia.com/gpu"      — NVIDIA GPU Operator
	//   "amd.com/gpu"         — AMD GPU device plugin
	//   "gpu.intel.com/i915"  — Intel i915 device plugin
	//   "gpu.intel.com/xe"    — Intel Xe device plugin
	resource: string // required — no default

	// Number of GPUs to allocate. Positive integer only (GPUs are non-divisible).
	count: int & >=1
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
	// GPU extended resource claim. Emitted to both requests and limits,
	// satisfying the Kubernetes constraint that extended resources cannot
	// be overcommitted (requests must equal limits).
	gpu?: #GpuResourceSchema
}

// Probe specification used by liveness, readiness, and startup probes.
// Valid for sidecar containers: all three probe types are supported.
// Valid for init containers: only startupProbe is honoured by Kubernetes
// (traditional init containers run to completion; native sidecar init containers
// with restartPolicy: Always support all three probes, requires K8s >= 1.28).
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

//////////////////////////////////////////////////////////////////
//// Scaling Schema
//////////////////////////////////////////////////////////////////

#ScalingSchema: {
	count: int & >=0 & <=1000 | *1
	auto?: #AutoscalingSpec
}

#AutoscalingSpec: {
	min!: int & >=1
	max!: int & >=1
	metrics!: [_, ...#MetricSpec]
	behavior?: {
		scaleUp?: {stabilizationWindowSeconds?: int}
		scaleDown?: {stabilizationWindowSeconds?: int}
	}
}

#MetricSpec: {
	type!:   "cpu" | "memory" | "custom"
	target!: #MetricTargetSpec
	if type == "custom" {
		metricName!: string
	}
}

#MetricTargetSpec: {
	averageUtilization?: int & >=1 & <=100
	averageValue?:       string
}

//////////////////////////////////////////////////////////////////
//// Sizing Schema
//////////////////////////////////////////////////////////////////

#SizingSchema: {
	resources?:   #ResourceRequirementsSchema
	autoScaling?: #VerticalScalingSchema
}

#VerticalScalingSchema: {}

//////////////////////////////////////////////////////////////////
//// RestartPolicy Schema
//////////////////////////////////////////////////////////////////

#RestartPolicySchema: "Always" | "OnFailure" | "Never" | *"Always"

//////////////////////////////////////////////////////////////////
//// UpdateStrategy Schema
//////////////////////////////////////////////////////////////////

#UpdateStrategySchema: {
	type: "RollingUpdate" | "Recreate" | "OnDelete" | *"RollingUpdate"
	if type == "RollingUpdate" {
		rollingUpdate?: {
			maxUnavailable?: uint | string | *1
			maxSurge?:       uint | string | *1
			partition?:      uint
		}
	}
}

//////////////////////////////////////////////////////////////////
//// InitContainers Schema
//////////////////////////////////////////////////////////////////

#InitContainersSchema: #ContainerSchema

//////////////////////////////////////////////////////////////////
//// SidecarContainers Schema
//////////////////////////////////////////////////////////////////

#SidecarContainersSchema: #ContainerSchema

//////////////////////////////////////////////////////////////////
//// JobConfig Schema
//////////////////////////////////////////////////////////////////

#JobConfigSchema: {
	completions?:             uint | *1
	parallelism?:             uint | *1
	backoffLimit?:            uint | *6
	activeDeadlineSeconds?:   uint | *300
	ttlSecondsAfterFinished?: uint | *100
}

//////////////////////////////////////////////////////////////////
//// CronJobConfig Schema
//////////////////////////////////////////////////////////////////

#CronJobConfigSchema: {
	scheduleCron!:               string
	concurrencyPolicy?:          "Allow" | "Forbid" | "Replace" | *"Allow"
	startingDeadlineSeconds?:    uint
	successfulJobsHistoryLimit?: uint | *3
	failedJobsHistoryLimit?:     uint | *1
}

//////////////////////////////////////////////////////////////////
//// Stateless Workload Schema
//////////////////////////////////////////////////////////////////

#StatelessWorkloadSchema: {
	container:       #ContainerSchema
	scaling?:        #ScalingSchema
	restartPolicy?:  #RestartPolicySchema
	updateStrategy?: #UpdateStrategySchema
	sidecarContainers?: [...#SidecarContainersSchema]
	initContainers?: [...#InitContainersSchema]
	securityContext?: #SecurityContextSchema
	hostPid?:         bool
	hostIpc?:         bool
}

//////////////////////////////////////////////////////////////////
//// DisruptionBudget Schema
//////////////////////////////////////////////////////////////////

// Availability constraints during voluntary disruptions.
// Exactly one of minAvailable or maxUnavailable must be set.
#DisruptionBudgetSchema: {
	minAvailable!: int | string & =~"^[0-9]+%$"
} | {maxUnavailable!: int | string & =~"^[0-9]+%$"
}

//////////////////////////////////////////////////////////////////
//// GracefulShutdown Schema
//////////////////////////////////////////////////////////////////

// Termination behavior for graceful workload shutdown
#GracefulShutdownSchema: {
	// Grace period before forceful termination (must be non-negative)
	terminationGracePeriodSeconds: uint | *30
}

//////////////////////////////////////////////////////////////////
//// Placement Schema
//////////////////////////////////////////////////////////////////

// Provider-agnostic workload placement intent
#PlacementSchema: {
	// Failure domain distribution target
	spreadAcross?: *"zones" | "regions" | "hosts"
	// Node/host selection criteria (string-to-string map)
	requirements?: [string]: string
	// Escape hatch for provider-specific placement details
	platformOverrides?: {...}
}
