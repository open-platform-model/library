// Package v1alpha2 implements the api.Binding for the
// "opmodel.dev/v1alpha2" OPM schema. It self-registers from init() — see
// init.go.
package v1alpha2

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sync"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"

	schema "github.com/open-platform-model/library/apis/core"
	"github.com/open-platform-model/library/pkg/api"
	"github.com/open-platform-model/library/pkg/apiversion"
)

// binding is the v1alpha2 implementation of api.Binding. It is unexported on
// purpose — callers reach it through api.Lookup / api.For. A pointer receiver
// is required so the schema-load cache (sync.Once + cached value/error) is
// shared across calls; the registry stores *binding, not binding.
type binding struct {
	schemaOnce sync.Once
	schemaVal  cue.Value
	schemaErr  error
}

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

func (*binding) Version() apiversion.Version { return apiversion.V1alpha2 }

func (*binding) Paths() api.Paths { return paths }

// EmbeddedSchema returns the v1alpha2 CUE schema filesystem. The Schema is
// embedded into the apis/core/v1alpha2 Go package via go:embed so it ships
// with every kernel build and is available for offline validation.
func (*binding) EmbeddedSchema() fs.FS { return schema.Schema }

// SchemaValue loads the v1alpha2 schema package as a cue.Value, lazily and
// exactly once per binding instance. See api.Binding.SchemaValue for the
// contract.
func (b *binding) SchemaValue(ctx *cue.Context) (cue.Value, error) {
	b.schemaOnce.Do(func() {
		b.schemaVal, b.schemaErr = loadSchemaValue(ctx, schema.Schema, "apis/core", "v1alpha2")
	})
	return b.schemaVal, b.schemaErr
}

// loadSchemaValue builds a CUE instance from an embed.FS-shaped read-only
// filesystem. fsys MUST contain a cue.mod/module.cue at its root and a CUE
// package at <pkgDir>. virtualRoot is the absolute path used to key the
// load.Config.Overlay map; CUE walks parents from <virtualRoot>/<pkgDir> to
// locate cue.mod, so the overlay paths MUST agree.
//
// The helper is package-private so binding_broken_test.go can drive it with a
// deliberately broken filesystem (asserts that load failures are wrapped).
func loadSchemaValue(ctx *cue.Context, fsys fs.FS, virtualRoot, pkgDir string) (cue.Value, error) {
	if ctx == nil {
		return cue.Value{}, fmt.Errorf("v1alpha2 SchemaValue: nil *cue.Context")
	}
	if fsys == nil {
		return cue.Value{}, fmt.Errorf("v1alpha2 SchemaValue: nil embed filesystem")
	}

	// Build an overlay mapping virtualRoot-joined paths to file contents. Using
	// a synthetic absolute root keeps the load entirely off-disk — no CUE_REGISTRY,
	// no filesystem read.
	root := filepath.Join(string(filepath.Separator), "embed", virtualRoot)
	overlay := map[string]load.Source{}
	walkErr := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, readErr := fs.ReadFile(fsys, p)
		if readErr != nil {
			return readErr
		}
		overlay[filepath.Join(root, filepath.FromSlash(p))] = load.FromBytes(data)
		return nil
	})
	if walkErr != nil {
		return cue.Value{}, fmt.Errorf("v1alpha2 SchemaValue: walking embed: %w", walkErr)
	}

	pkgDirAbs := filepath.Join(root, filepath.FromSlash(pkgDir))
	cfg := &load.Config{
		Dir:     pkgDirAbs,
		Overlay: overlay,
	}
	instances := load.Instances([]string{"."}, cfg)
	if len(instances) == 0 {
		return cue.Value{}, fmt.Errorf("v1alpha2 SchemaValue: load.Instances returned no instances")
	}
	if instances[0].Err != nil {
		return cue.Value{}, fmt.Errorf("v1alpha2 SchemaValue: loading schema: %w", instances[0].Err)
	}

	val := ctx.BuildInstance(instances[0])
	if err := val.Err(); err != nil {
		return cue.Value{}, fmt.Errorf("v1alpha2 SchemaValue: building schema: %w", err)
	}
	return val, nil
}
