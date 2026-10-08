package platform

// PlatformMetadata is the canonical decoded platform-level metadata. Type is
// the top-level #Platform.type field hoisted into the metadata projection so
// callers see one Go-level identity record per Platform artifact.
//
//nolint:revive // stutter intentional: platform.PlatformMetadata reads clearly at call sites
type PlatformMetadata struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Description string            `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}
