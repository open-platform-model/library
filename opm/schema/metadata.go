package schema

// ModuleMetadata contains module-level identity and version information.
// This is the module's canonical metadata, distinct from the release it is
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

// ReleaseMetadata contains release-level identity information.
// Used for inventory tracking, resource labeling, and CLI output.
type ReleaseMetadata struct {
	// Name is the release name (from --name or module.metadata.name).
	Name string `json:"name"`

	// Namespace is the target namespace.
	Namespace string `json:"namespace"`

	// UUID is the release identity UUID.
	// Computed by CUE as SHA1(OPMNamespace, moduleUUID:name:namespace).
	UUID string `json:"uuid"`

	// Labels are the merged release labels (module labels + standard opm labels).
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations are the merged release annotations.
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ProviderMetadata is the canonical decoded provider-level metadata.
type ProviderMetadata struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Version     string            `json:"version,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
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

// ReleaseView is the read-only view of a module release that
// BuildTransformerContext needs. The interface exists so the context builder
// stays decoupled from opm/module; any caller-supplied type that exposes
// these accessors can drive context construction (e.g. tests).
type ReleaseView interface {
	ReleaseName() string
	Namespace() string
	ReleaseUUID() string
	ModuleFQN() string
	ModuleVersion() string
	Labels() map[string]string
	Annotations() map[string]string
}
