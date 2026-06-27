package schema

import (
	"fmt"

	"cuelang.org/go/cue"
)

// DecodeModuleMetadata extracts ModuleMetadata from a #Module artifact root.
// A missing metadata field is fatal.
func DecodeModuleMetadata(v cue.Value) (*ModuleMetadata, error) {
	metaVal := v.LookupPath(Metadata)
	if !metaVal.Exists() {
		return nil, fmt.Errorf("module metadata field is required")
	}
	meta := &ModuleMetadata{}
	if err := metaVal.Decode(meta); err != nil {
		return nil, fmt.Errorf("decoding module metadata: %w", err)
	}
	return meta, nil
}

// DecodeInstanceMetadata extracts InstanceMetadata from a #ModuleInstance
// artifact root. A missing metadata field is fatal.
//
// Was: DecodeReleaseMetadata
func DecodeInstanceMetadata(v cue.Value) (*InstanceMetadata, error) {
	metaVal := v.LookupPath(Metadata)
	if !metaVal.Exists() {
		return nil, fmt.Errorf("instance metadata field is required")
	}
	meta := &InstanceMetadata{}
	if err := metaVal.Decode(meta); err != nil {
		return nil, fmt.Errorf("decoding instance metadata: %w", err)
	}
	return meta, nil
}

// DecodePlatformMetadata extracts PlatformMetadata from a #Platform value.
// metadata.{name,description,labels,annotations} is decoded directly into the
// struct; the top-level #Platform.type field is read separately and merged
// into Metadata.Type so callers see one identity record per Platform.
// A missing metadata field is fatal.
func DecodePlatformMetadata(v cue.Value) (*PlatformMetadata, error) {
	metaVal := v.LookupPath(Metadata)
	if !metaVal.Exists() {
		return nil, fmt.Errorf("platform metadata field is required")
	}
	meta := &PlatformMetadata{}
	if err := metaVal.Decode(meta); err != nil {
		return nil, fmt.Errorf("decoding platform metadata: %w", err)
	}
	if typeVal := v.LookupPath(cue.ParsePath("type")); typeVal.Exists() {
		s, err := typeVal.String()
		if err != nil {
			return nil, fmt.Errorf("decoding platform type: %w", err)
		}
		meta.Type = s
	}
	return meta, nil
}

// DecodeProviderMetadata extracts ProviderMetadata from a #Provider value.
// fallbackName is used when the artifact's metadata is absent or its name
// field decoded as empty — typically the config map key under which the
// provider was loaded.
func DecodeProviderMetadata(v cue.Value, fallbackName string) (*ProviderMetadata, error) {
	meta := &ProviderMetadata{Name: fallbackName}
	metaVal := v.LookupPath(Metadata)
	if !metaVal.Exists() {
		return meta, nil
	}
	if err := metaVal.Decode(meta); err != nil {
		return nil, fmt.Errorf("decoding provider metadata: %w", err)
	}
	if meta.Name == "" {
		meta.Name = fallbackName
	}
	return meta, nil
}
