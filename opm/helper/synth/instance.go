package synth

import (
	"errors"
	"fmt"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"

	loaderfile "github.com/open-platform-model/library/opm/helper/loader/file"
	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/schema"
)

// InstanceInput is the typed input carried into Instance. Required fields:
// Module, Name, Namespace, SchemaCache. Optional fields are filled into
// the instance only when present (non-nil / non-empty / non-zero); empty
// values do not displace schema-derived fields.
//
// Was: ReleaseInput
type InstanceInput struct {
	// Module is the source #Module the instance deploys. Required. Its
	// metadata.modulePath / metadata.version identify the published module
	// the synthesized package imports (Instance references the module by
	// import, not by inlining its value).
	Module *module.Module

	// Name is the instance name (metadata.name). Required. Must satisfy the
	// schema's #NameType regex; violations surface as a CUE unification error
	// from Instance.
	Name string

	// Namespace is the target namespace (metadata.namespace). Required.
	Namespace string

	// SchemaCache supplies the OPM core schema used to unify against
	// #ModuleInstance. REQUIRED. Typically the value of
	// kernel.SchemaCache() from the caller's Kernel — passing the
	// kernel's cache preserves the one-Cache-per-process invariant and
	// avoids a duplicate schema fetch. Instance returns an error when
	// this field is nil.
	SchemaCache *schema.Cache

	// Values is the caller-supplied configuration value unified against the
	// module's #config. The zero cue.Value signals "no values supplied" — the
	// schema's values path is left unfilled and concreteness is enforced
	// downstream by Kernel.ProcessModuleInstance. Instance NEVER falls back to
	// Module.debugValues; that is a frontend policy concern.
	Values cue.Value

	// Labels and Annotations layer over the schema's stamped
	// module-instance.opmodel.dev/{name,uuid} labels. CUE unification merges
	// caller-supplied entries with schema-stamped ones; caller-supplied keys
	// MUST NOT collide with the schema's reserved keys.
	Labels      map[string]string
	Annotations map[string]string
}

// Sentinel errors. Frontends inspecting the failure surface match on these via
// errors.Is. Synthesis-time errors all wrap one of these so callers can
// distinguish "you forgot a required field" from "the schema is broken."
var (
	// ErrMissingModule is returned when InstanceInput.Module is nil, or carries
	// no decoded metadata identity (modulePath / version) to import by.
	ErrMissingModule = errors.New("synth.Instance: Module is required")

	// ErrMissingName is returned when InstanceInput.Name is empty.
	ErrMissingName = errors.New("synth.Instance: Name is required")

	// ErrMissingNamespace is returned when InstanceInput.Namespace is empty.
	ErrMissingNamespace = errors.New("synth.Instance: Namespace is required")

	// ErrMissingSchemaCache is returned when InstanceInput.SchemaCache is
	// nil. Callers typically pass kernel.SchemaCache() from their Kernel.
	ErrMissingSchemaCache = errors.New("synth.Instance: SchemaCache is required")

	// ErrSchemaUnavailable is returned when the caller-supplied
	// SchemaCache resolves but does not expose #ModuleInstance, or does not
	// surface a resolved version to pin the synthesized package's core dep.
	ErrSchemaUnavailable = errors.New("synth.Instance: schema unavailable")
)

// synthRoot is the deterministic in-memory module root the synthesized instance
// package is overlaid under. It need not exist on disk; the loader treats the
// overlaid files as present there and uses it as the module root so the
// fabricated cue.mod/module.cue drives dependency resolution. It is constant
// because the synthesized package is the MAIN module (rebuilt from the overlay
// on every call), never cached by module path the way fetched deps are.
func synthRoot() string {
	return string(filepath.Separator) + "opm-synth-instance"
}

// Instance builds a #ModuleInstance CUE value by synthesizing an in-memory CUE
// package and evaluating it in a single build (ADR-003), through the same
// loader build-and-shape-gate path LoadInstancePackage uses for on-disk instance
// packages. The synthesized package consists of a fabricated cue.mod/module.cue
// (deps: the resolved core version + the module's path@version), an instance.cue
// that imports core and the module and writes `#module: <import>` plus
// caller-supplied metadata, and — when Values is supplied — a values.cue
// rendered from InstanceInput.Values.
//
// Because the module enters the build by import (one CUE evaluation, one
// #Image / #Secret closure), there is no closed-into-closed FillPath and no
// Go-side value pre-merge: the schema's own
// `let unifiedModule = #module & {#config: values}` performs the values merge
// in CUE. The previous cue.Scope / userModule workaround and the
// FillPath(#config, Values) pre-merge are gone.
//
// The function does NOT validate values against #config and does NOT enforce
// concreteness. Both responsibilities live downstream in
// Kernel.ProcessModuleInstance, which the Kernel.SynthesizeInstance wrapper
// chains onto this call. See package doc for the recommended entry point.
//
// The returned cue.Value carries every schema-derived field automatically:
// metadata.uuid is computed by uuid.SHA1, components is fanned from the
// unified module, opm-secrets is added when #Secret instances are present,
// and the standard module-instance.opmodel.dev/{name,uuid} labels are stamped.
// Instance stamps only the caller-supplied fields and lets CUE derive the rest.
//
// Was: Release
func Instance(ctx *cue.Context, in InstanceInput) (cue.Value, error) {
	if in.Module == nil {
		return cue.Value{}, ErrMissingModule
	}
	if in.Name == "" {
		return cue.Value{}, ErrMissingName
	}
	if in.Namespace == "" {
		return cue.Value{}, ErrMissingNamespace
	}
	if in.SchemaCache == nil {
		return cue.Value{}, ErrMissingSchemaCache
	}
	if in.Module.Metadata == nil || in.Module.Metadata.ModulePath == "" || in.Module.Metadata.Version == "" {
		return cue.Value{}, fmt.Errorf("%w: module has no modulePath/version identity to import by", ErrMissingModule)
	}

	// Resolve the schema to (a) confirm #ModuleInstance is present and (b) learn
	// the resolved core version, which the fabricated cue.mod/module.cue pins
	// so the synth build is reproducible. This is the same Cache the caller's
	// Kernel owns, so it reuses the already-memoized schema fetch.
	schemaPkg, err := in.SchemaCache.Get(ctx)
	if err != nil {
		return cue.Value{}, fmt.Errorf("synth.Instance: loading schema: %w", err)
	}
	if !schemaPkg.LookupPath(cue.ParsePath("#ModuleInstance")).Exists() {
		return cue.Value{}, fmt.Errorf("%w: #ModuleInstance not found in resolved schema", ErrSchemaUnavailable)
	}
	coreVersion := in.SchemaCache.ResolvedVersion()
	if coreVersion == "" {
		return cue.Value{}, fmt.Errorf("%w: resolved core schema version unavailable, cannot pin synth module deps", ErrSchemaUnavailable)
	}

	overlay, err := buildOverlay(in, coreVersion)
	if err != nil {
		return cue.Value{}, fmt.Errorf("synth.Instance: %w", err)
	}

	// Evaluate the synthesized package through the SAME build-and-shape-gate
	// step LoadInstancePackage uses; the only difference is overlay vs. on-disk
	// source. Registry resolution uses the process CUE_REGISTRY (the module and
	// core resolve from the same registry/cache the caller already used to
	// acquire the module).
	val, err := loaderfile.BuildInstanceOverlay(ctx, synthRoot(), overlay, loaderfile.LoadOptions{})
	if err != nil {
		return cue.Value{}, fmt.Errorf("synth.Instance: %w", err)
	}
	return val, nil
}

// buildOverlay assembles the in-memory load.Source overlay for the synthesized
// instance package: the fabricated module file, the instance file, and (only when
// Values is supplied) the rendered values file. Keys are absolute paths under
// synthRoot so the loader treats them as the package's files.
func buildOverlay(in InstanceInput, coreVersion string) (map[string]load.Source, error) {
	root := synthRoot()
	overlay := map[string]load.Source{
		filepath.Join(root, "cue.mod", "module.cue"): load.FromString(renderModuleFile(in, coreVersion)),
		filepath.Join(root, "instance.cue"):          load.FromString(renderInstanceFile(in, coreVersion)),
	}

	valuesSrc, err := renderValuesFile(in)
	if err != nil {
		return nil, err
	}
	if valuesSrc != nil {
		overlay[filepath.Join(root, "values.cue")] = load.FromBytes(valuesSrc)
	}

	return overlay, nil
}
