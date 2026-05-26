package schema

import (
	"fmt"

	"cuelang.org/go/cue"
)

// ModuleReleaseContextData is the Go-side mirror of
// #TransformerContext.#moduleReleaseMetadata. Field names use json tags that
// match the CUE definition fields.
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
// #TransformerContext.#componentMetadata.
type ComponentContextData struct {
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// BuildTransformerContext constructs the #context value for a single
// (release, component, transformer) pair. The caller fills the returned value
// at schema.Context on the unified transformer.
//
// schemaComp must be the schema-preserving component value (the one that
// still has metadata.labels and metadata.annotations as concrete fields).
// Decode errors on those metadata fields surface as warnings, not errors.
func BuildTransformerContext(
	cueCtx *cue.Context,
	rel ReleaseView,
	compName string,
	schemaComp cue.Value,
	runtimeName string,
) (cue.Value, []string, error) {
	if cueCtx == nil {
		return cue.Value{}, nil, fmt.Errorf("cue context is required")
	}
	if rel == nil {
		return cue.Value{}, nil, fmt.Errorf("release view is required")
	}
	if runtimeName == "" {
		return cue.Value{}, nil, fmt.Errorf("runtimeName must be non-empty")
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
	if labelsVal := schemaComp.LookupPath(MetadataLabels); labelsVal.Exists() {
		if err := labelsVal.Decode(&comp.Labels); err != nil {
			warnings = append(warnings, fmt.Sprintf(
				"component %q: metadata.labels could not be decoded; labels will be empty in transformer context: %v",
				compName, err,
			))
		}
	}
	if annotationsVal := schemaComp.LookupPath(MetadataAnnotations); annotationsVal.Exists() {
		if err := annotationsVal.Decode(&comp.Annotations); err != nil {
			warnings = append(warnings, fmt.Sprintf(
				"component %q: metadata.annotations could not be decoded; annotations will be empty in transformer context: %v",
				compName, err,
			))
		}
	}

	// Build the context value by encoding each sub-struct and assembling under
	// the #context definition. The returned cue.Value carries the three
	// #context fields with no surrounding shape constraint, so the caller's
	// FillPath into schema.Context unifies cleanly.
	root := cueCtx.CompileString("{}")
	if err := root.Err(); err != nil {
		return cue.Value{}, warnings, fmt.Errorf("building context root: %w", err)
	}
	ctxVal := root.
		FillPath(cue.ParsePath("#moduleReleaseMetadata"), cueCtx.Encode(mrm)).
		FillPath(cue.ParsePath("#componentMetadata"), cueCtx.Encode(comp)).
		FillPath(cue.ParsePath("#runtimeName"), cueCtx.Encode(runtimeName))
	if err := ctxVal.Err(); err != nil {
		return cue.Value{}, warnings, fmt.Errorf("assembling context: %w", err)
	}
	return ctxVal, warnings, nil
}
