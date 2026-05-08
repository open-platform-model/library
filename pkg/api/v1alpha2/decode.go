package v1alpha2

import (
	"fmt"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/pkg/api"
)

// DecodeModuleMetadata extracts api.ModuleMetadata from a #Module artifact
// root. Mirrors the existing decode pattern (val.Decode into a struct) used
// elsewhere in the kernel.
func (binding) DecodeModuleMetadata(v cue.Value) (*api.ModuleMetadata, error) {
	metaVal := v.LookupPath(paths.Metadata)
	if !metaVal.Exists() {
		return nil, fmt.Errorf("v1alpha2: module metadata field is required")
	}
	meta := &api.ModuleMetadata{}
	if err := metaVal.Decode(meta); err != nil {
		return nil, fmt.Errorf("v1alpha2: decoding module metadata: %w", err)
	}
	return meta, nil
}

// DecodeReleaseMetadata extracts api.ReleaseMetadata from a #ModuleRelease
// artifact root.
func (binding) DecodeReleaseMetadata(v cue.Value) (*api.ReleaseMetadata, error) {
	metaVal := v.LookupPath(paths.Metadata)
	if !metaVal.Exists() {
		return nil, fmt.Errorf("v1alpha2: release metadata field is required")
	}
	meta := &api.ReleaseMetadata{}
	if err := metaVal.Decode(meta); err != nil {
		return nil, fmt.Errorf("v1alpha2: decoding release metadata: %w", err)
	}
	return meta, nil
}

// DecodePlatformMetadata extracts api.PlatformMetadata from a #Platform value.
// metadata.{name,description,labels,annotations} is decoded directly into the
// struct; the top-level #Platform.type field is read separately and merged
// into Metadata.Type so callers see one identity record per Platform.
func (binding) DecodePlatformMetadata(v cue.Value) (*api.PlatformMetadata, error) {
	metaVal := v.LookupPath(paths.Metadata)
	if !metaVal.Exists() {
		return nil, fmt.Errorf("v1alpha2: platform metadata field is required")
	}
	meta := &api.PlatformMetadata{}
	if err := metaVal.Decode(meta); err != nil {
		return nil, fmt.Errorf("v1alpha2: decoding platform metadata: %w", err)
	}
	if typeVal := v.LookupPath(cue.ParsePath("type")); typeVal.Exists() {
		s, err := typeVal.String()
		if err != nil {
			return nil, fmt.Errorf("v1alpha2: decoding platform type: %w", err)
		}
		meta.Type = s
	}
	return meta, nil
}

// DecodeProviderMetadata extracts api.ProviderMetadata from a #Provider value.
// fallbackName is used when the artifact's metadata is absent or its name
// field decoded as empty — typically the config map key under which the
// provider was loaded.
func (binding) DecodeProviderMetadata(v cue.Value, fallbackName string) (*api.ProviderMetadata, error) {
	meta := &api.ProviderMetadata{Name: fallbackName}
	metaVal := v.LookupPath(paths.Metadata)
	if !metaVal.Exists() {
		return meta, nil
	}
	if err := metaVal.Decode(meta); err != nil {
		return nil, fmt.Errorf("v1alpha2: decoding provider metadata: %w", err)
	}
	if meta.Name == "" {
		meta.Name = fallbackName
	}
	return meta, nil
}
