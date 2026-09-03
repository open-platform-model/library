package schema

// ModuleMetadata contains module-level identity and version information.
// This is the module's canonical metadata, distinct from the instance it is
// deployed as. Populated by DecodeModuleMetadata.
type ModuleMetadata struct {
	// Name is the canonical module name from module.metadata.name (kebab-case).
	Name string `json:"name"`

	// Description is a brief description of the module.
	Description string `json:"description,omitempty"`

	// ModulePath is the CUE registry module path from metadata.modulePath.
	// This is the registry path (e.g., "opmodel.dev/modules"), NOT a filesystem path.
	ModulePath string `json:"modulePath"`

	// Version is the module version (semver).
	Version string `json:"version"`

	// FQN is the fully qualified module name (modulePath/name:version).
	// Example: "opmodel.dev/modules/my-app:1.0.0"
	FQN string `json:"fqn"`

	// UUID is the module identity UUID (from #Module.metadata.identity).
	UUID string `json:"uuid"`

	// Labels from the module definition (pre-build, author-declared).
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations from the module definition.
	Annotations map[string]string `json:"annotations,omitempty"`
}

// InstanceMetadata contains instance-level identity information.
// Used for inventory tracking, resource labeling, and CLI output.
//
// Was: ReleaseMetadata
type InstanceMetadata struct {
	// Name is the instance name (from --name or module.metadata.name).
	Name string `json:"name"`

	// Namespace is the target namespace.
	Namespace string `json:"namespace"`

	// FQN is the instance's OWN fully qualified name
	// (registryPath:name:namespace), defined by core v2 on
	// #ModuleInstance.metadata and decoded with the rest of the metadata.
	// Distinct from the source module's FQN, which lives on the instance's
	// Package at the module-metadata path.
	FQN string `json:"fqn,omitempty"`

	// UUID is the instance identity UUID.
	// Computed by CUE as SHA1(OPMNamespace, moduleUUID:name:namespace).
	UUID string `json:"uuid"`

	// Labels are the merged instance labels (module labels + standard opm labels).
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations are the merged instance annotations.
	Annotations map[string]string `json:"annotations,omitempty"`
}

// PlatformMetadata is the canonical decoded platform-level metadata. Type is
// the top-level #Platform.type field hoisted into the metadata projection so
// callers see one Go-level identity record per Platform artifact.
type PlatformMetadata struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Description string            `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}
