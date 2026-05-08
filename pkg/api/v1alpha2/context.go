package v1alpha2

import (
	"fmt"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/pkg/api"
)

// ModuleReleaseContextData is the Go-side mirror of
// #TransformerContext.#moduleReleaseMetadata for the v1alpha2 schema. Field
// names use json tags that match the CUE definition fields.
type ModuleReleaseContextData struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	FQN         string            `json:"fqn"`
	Version     string            `json:"version"`
	UUID        string            `json:"uuid"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ComponentContextData is the Go-side mirror of
// #TransformerContext.#componentMetadata for the v1alpha2 schema.
type ComponentContextData struct {
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// BuildTransformerContext constructs the v1alpha2 #context value for a single
// (release, component, transformer) pair. The renderer is responsible for
// filling the returned value at Paths().Context on the unified transformer.
//
// schemaComp must be the schema-preserving component value (the one that still
// has metadata.labels and metadata.annotations as concrete fields). Decode
// errors on those metadata fields are returned as warnings.
func (binding) BuildTransformerContext(
	cueCtx *cue.Context,
	rel api.ReleaseView,
	compName string,
	schemaComp cue.Value,
	runtimeName string,
) (cue.Value, []string, error) {
	if cueCtx == nil {
		return cue.Value{}, nil, fmt.Errorf("v1alpha2: cue context is required")
	}
	if rel == nil {
		return cue.Value{}, nil, fmt.Errorf("v1alpha2: release view is required")
	}
	if runtimeName == "" {
		return cue.Value{}, nil, fmt.Errorf("v1alpha2: runtimeName must be non-empty")
	}

	mrm := ModuleReleaseContextData{
		Name:        rel.ReleaseName(),
		Namespace:   rel.Namespace(),
		FQN:         rel.ModuleFQN(),
		Version:     rel.ModuleVersion(),
		UUID:        rel.ReleaseUUID(),
		Labels:      rel.Labels(),
		Annotations: rel.Annotations(),
	}

	var warnings []string
	comp := ComponentContextData{Name: compName}
	if labelsVal := schemaComp.LookupPath(paths.MetadataLabels); labelsVal.Exists() {
		if err := labelsVal.Decode(&comp.Labels); err != nil {
			warnings = append(warnings, fmt.Sprintf(
				"component %q: metadata.labels could not be decoded; labels will be empty in transformer context: %v",
				compName, err,
			))
		}
	}
	if annotationsVal := schemaComp.LookupPath(paths.MetadataAnnotations); annotationsVal.Exists() {
		if err := annotationsVal.Decode(&comp.Annotations); err != nil {
			warnings = append(warnings, fmt.Sprintf(
				"component %q: metadata.annotations could not be decoded; annotations will be empty in transformer context: %v",
				compName, err,
			))
		}
	}

	// Build the context value by encoding each sub-struct and assembling under
	// the #context definition. We construct a fresh value at the context root
	// so the returned cue.Value can be filled at Paths().Context by the caller
	// without any prior shape constraint.
	root := cueCtx.CompileString("{}")
	if err := root.Err(); err != nil {
		return cue.Value{}, warnings, fmt.Errorf("v1alpha2: building context root: %w", err)
	}
	ctxVal := root.
		FillPath(cue.ParsePath("#moduleReleaseMetadata"), cueCtx.Encode(mrm)).
		FillPath(cue.ParsePath("#componentMetadata"), cueCtx.Encode(comp)).
		FillPath(cue.ParsePath("#runtimeName"), cueCtx.Encode(runtimeName))
	if err := ctxVal.Err(); err != nil {
		return cue.Value{}, warnings, fmt.Errorf("v1alpha2: assembling context: %w", err)
	}
	return ctxVal, warnings, nil
}
