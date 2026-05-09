// Package v1alpha2 implements the api.Binding for the
// "opmodel.dev/v1alpha2" OPM schema. It self-registers from init() — see
// init.go.
package v1alpha2

import (
	"io/fs"

	"cuelang.org/go/cue"

	schema "github.com/open-platform-model/library/apis/core"
	"github.com/open-platform-model/library/pkg/api"
	"github.com/open-platform-model/library/pkg/apiversion"
)

// binding is the v1alpha2 implementation of api.Binding. It is unexported on
// purpose — callers reach it through api.Lookup / api.For.
type binding struct{}

var paths = api.Paths{
	APIVersion: cue.ParsePath("apiVersion"),
	Metadata:   cue.ParsePath("metadata"),

	Components:     cue.ParsePath("components"),
	Values:         cue.ParsePath("values"),
	Config:         cue.MakePath(cue.Def("config")),
	Module:         cue.MakePath(cue.Def("module")),
	ModuleMetadata: cue.MakePath(cue.Def("moduleMetadata")),

	DebugValues: cue.ParsePath("debugValues"),

	Transformers: cue.ParsePath("#transformers"),

	Registry:             cue.ParsePath("#registry"),
	KnownResources:       cue.ParsePath("#knownResources"),
	KnownTraits:          cue.ParsePath("#knownTraits"),
	ComposedTransformers: cue.ParsePath("#composedTransformers"),
	Matchers:             cue.ParsePath("#matchers"),
	MatchersResources:    cue.ParsePath("#matchers.resources"),
	MatchersTraits:       cue.ParsePath("#matchers.traits"),

	Transform:                    cue.ParsePath("#transform"),
	TransformerRequiredLabels:    cue.ParsePath("requiredLabels"),
	TransformerRequiredResources: cue.ParsePath("requiredResources"),
	TransformerRequiredTraits:    cue.ParsePath("requiredTraits"),
	TransformerOptionalTraits:    cue.ParsePath("optionalTraits"),

	Component: cue.ParsePath("#component"),
	Context:   cue.MakePath(cue.Def("context")),
	Output:    cue.ParsePath("output"),

	ContextModuleReleaseMetadata: cue.MakePath(cue.Def("context"), cue.Def("moduleReleaseMetadata")),
	ContextComponentMetadata:     cue.MakePath(cue.Def("context"), cue.Def("componentMetadata")),
	ContextRuntimeName:           cue.MakePath(cue.Def("context"), cue.Def("runtimeName")),

	MetadataLabels:      cue.ParsePath("metadata.labels"),
	MetadataAnnotations: cue.ParsePath("metadata.annotations"),
	MetadataFQN:         cue.ParsePath("metadata.fqn"),
	ComponentResources:  cue.MakePath(cue.Def("resources")),
	ComponentTraits:     cue.MakePath(cue.Def("traits")),
}

func (binding) Version() apiversion.Version { return apiversion.V1alpha2 }

func (binding) Paths() api.Paths { return paths }

// EmbeddedSchema returns the v1alpha2 CUE schema filesystem. The Schema is
// embedded into the apis/core/v1alpha2 Go package via go:embed so it ships
// with every kernel build and is available for offline validation.
func (binding) EmbeddedSchema() fs.FS { return schema.Schema }
