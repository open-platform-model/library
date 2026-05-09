package schemas

/////////////////////////////////////////////////////////////////
//// Security Schemas
/////////////////////////////////////////////////////////////////

#WorkloadIdentitySchema: {
	name!:           string
	automountToken?: bool
}

// #ImagePullSecretsSchema lists pre-existing K8s Secrets used to pull
// container images. Each entry is a LocalObjectReference to a Secret
// of type kubernetes.io/dockerconfigjson in the same namespace as the pod.
//
// OPM does not create these Secrets — they must already exist in the
// cluster (typically managed by an external secret operator or platform team).
#ImagePullSecretsSchema: [...{name!: string}]

// Security context constraints for container and pod-level hardening.
//
// Fields apply at different levels in Kubernetes:
//   Pod-level (spec.securityContext):    runAsNonRoot, runAsUser, runAsGroup, fsGroup,
//                                        supplementalGroups
//   Container-level (containers[].securityContext): privileged, allowPrivilegeEscalation,
//                                        capabilities, readOnlyRootFilesystem,
//                                        runAsNonRoot, runAsUser, runAsGroup
//
// When privileged: true is set, it grants the container full host access and
// supersedes most other security constraints. Only set on workloads that
// explicitly require host-level access (e.g. Docker-in-Docker, GPU streaming).
#SecurityContextSchema: {
	// Grant full host access (equivalent to running as root on the host).
	// Required for Docker-in-Docker and workloads that need device-level access.
	// Supersedes allowPrivilegeEscalation, capabilities, and most other constraints.
	privileged?: bool

	// Run container as non-root user
	runAsNonRoot?: bool
	// Specific user ID to run as
	runAsUser?: int
	// Specific group ID to run as
	runAsGroup?: int
	// Group ID applied to mounted volumes; kubelet chowns volume contents to this GID on mount
	fsGroup?: int
	// Additional GIDs applied to the pod's processes (pod-level only).
	// Used to grant access to shared devices or files owned by supplemental groups.
	// Example: [44, 109] grants video and render group access for /dev/dri on Linux.
	supplementalGroups?: [...int]
	// Mount the root filesystem as read-only
	readOnlyRootFilesystem?: bool
	// Prevent privilege escalation via setuid/setgid binaries
	allowPrivilegeEscalation?: bool
	// Linux capabilities
	capabilities?: {
		add?: [...string]
		drop?: [...string] | ["ALL"]
	}
}

// Standalone service account identity
#ServiceAccountSchema: {
	name!:           string
	automountToken?: bool
}

// Single RBAC permission rule
#PolicyRuleSchema: {
	apiGroups!: [...string]
	resources!: [...string]
	verbs!: [...string]
}

// Role subject — embeds an identity directly via CUE reference
#RoleSubjectSchema: {#WorkloadIdentitySchema | #ServiceAccountSchema}

// RBAC role with rules and CUE-referenced subjects
#RoleSchema: {
	name!: string
	scope: *"namespace" | "cluster"
	rules!: [...#PolicyRuleSchema] & [_, ...]
	subjects!: [...#RoleSubjectSchema] & [_, ...]
}

#EncryptionConfigSchema: {
	atRest:    bool
	inTransit: bool
}
