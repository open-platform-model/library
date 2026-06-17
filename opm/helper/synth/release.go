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

// ReleaseInput is the typed input carried into Release. Required fields:
// Module, Name, Namespace, SchemaCache. Optional fields are filled into
// the release only when present (non-nil / non-empty / non-zero); empty
// values do not displace schema-derived fields.
type ReleaseInput struct {
	// Module is the source #Module the release deploys. Required. Its
	// metadata.modulePath / metadata.version identify the published module
	// the synthesized package imports (Release references the module by
	// import, not by inlining its value).
	Module *module.Module

	// Name is the release name (metadata.name). Required. Must satisfy the
	// schema's #NameType regex; violations surface as a CUE unification error
	// from Release.
	Name string

	// Namespace is the target namespace (metadata.namespace). Required.
	Namespace string

	// SchemaCache supplies the OPM core schema used to unify against
	// #ModuleRelease. REQUIRED. Typically the value of
	// kernel.SchemaCache() from the caller's Kernel — passing the
	// kernel's cache preserves the one-Cache-per-process invariant and
	// avoids a duplicate schema fetch. Release returns an error when
	// this field is nil.
	SchemaCache *schema.Cache

	// Values is the caller-supplied configuration value unified against the
	// module's #config. The zero cue.Value signals "no values supplied" — the
	// schema's values path is left unfilled and concreteness is enforced
	// downstream by Kernel.ProcessModuleRelease. Release NEVER falls back to
	// Module.debugValues; that is a frontend policy concern.
	Values cue.Value

	// Labels and Annotations layer over the schema's stamped
	// module-release.opmodel.dev/{name,uuid} labels. CUE unification merges
	// caller-supplied entries with schema-stamped ones; caller-supplied keys
	// MUST NOT collide with the schema's reserved keys.
	Labels      map[string]string
	Annotations map[string]string
}

// Sentinel errors. Frontends inspecting the failure surface match on these via
// errors.Is. Synthesis-time errors all wrap one of these so callers can
// distinguish "you forgot a required field" from "the schema is broken."
var (
	// ErrMissingModule is returned when ReleaseInput.Module is nil, or carries
	// no decoded metadata identity (modulePath / version) to import by.
	ErrMissingModule = errors.New("synth.Release: Module is required")

	// ErrMissingName is returned when ReleaseInput.Name is empty.
	ErrMissingName = errors.New("synth.Release: Name is required")

	// ErrMissingNamespace is returned when ReleaseInput.Namespace is empty.
	ErrMissingNamespace = errors.New("synth.Release: Namespace is required")

	// ErrMissingSchemaCache is returned when ReleaseInput.SchemaCache is
	// nil. Callers typically pass kernel.SchemaCache() from their Kernel.
	ErrMissingSchemaCache = errors.New("synth.Release: SchemaCache is required")

	// ErrSchemaUnavailable is returned when the caller-supplied
	// SchemaCache resolves but does not expose #ModuleRelease, or does not
	// surface a resolved version to pin the synthesized package's core dep.
	ErrSchemaUnavailable = errors.New("synth.Release: schema unavailable")
)

// synthRoot is the deterministic in-memory module root the synthesized release
// package is overlaid under. It need not exist on disk; the loader treats the
// overlaid files as present there and uses it as the module root so the
// fabricated cue.mod/module.cue drives dependency resolution. It is constant
// because the synthesized package is the MAIN module (rebuilt from the overlay
// on every call), never cached by module path the way fetched deps are.
func synthRoot() string {
	return string(filepath.Separator) + "opm-synth-release"
}

// Release builds a #ModuleRelease CUE value by synthesizing an in-memory CUE
// package and evaluating it in a single build (ADR-003), through the same
// loader build-and-shape-gate path LoadReleasePackage uses for on-disk release
// packages. The synthesized package consists of a fabricated cue.mod/module.cue
// (deps: the resolved core version + the module's path@version), a release.cue
// that imports core and the module and writes `#module: <import>` plus
// caller-supplied metadata, and — when Values is supplied — a values.cue
// rendered from ReleaseInput.Values.
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
// Kernel.ProcessModuleRelease, which the Kernel.SynthesizeRelease wrapper
// chains onto this call. See package doc for the recommended entry point.
//
// The returned cue.Value carries every schema-derived field automatically:
// metadata.uuid is computed by uuid.SHA1, components is fanned from the
// unified module, opm-secrets is added when #Secret instances are present,
// and the standard module-release.opmodel.dev/{name,uuid} labels are stamped.
// Release stamps only the caller-supplied fields and lets CUE derive the rest.
func Release(ctx *cue.Context, in ReleaseInput) (cue.Value, error) {
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

	// Resolve the schema to (a) confirm #ModuleRelease is present and (b) learn
	// the resolved core version, which the fabricated cue.mod/module.cue pins
	// so the synth build is reproducible. This is the same Cache the caller's
	// Kernel owns, so it reuses the already-memoized schema fetch.
	schemaPkg, err := in.SchemaCache.Get(ctx)
	if err != nil {
		return cue.Value{}, fmt.Errorf("synth.Release: loading schema: %w", err)
	}
	if !schemaPkg.LookupPath(cue.ParsePath("#ModuleRelease")).Exists() {
		return cue.Value{}, fmt.Errorf("%w: #ModuleRelease not found in resolved schema", ErrSchemaUnavailable)
	}
	coreVersion := in.SchemaCache.ResolvedVersion()
	if coreVersion == "" {
		return cue.Value{}, fmt.Errorf("%w: resolved core schema version unavailable, cannot pin synth module deps", ErrSchemaUnavailable)
	}

	overlay, err := buildOverlay(in, coreVersion)
	if err != nil {
		return cue.Value{}, fmt.Errorf("synth.Release: %w", err)
	}

	// Evaluate the synthesized package through the SAME build-and-shape-gate
	// step LoadReleasePackage uses; the only difference is overlay vs. on-disk
	// source. Registry resolution uses the process CUE_REGISTRY (the module and
	// core resolve from the same registry/cache the caller already used to
	// acquire the module).
	val, err := loaderfile.BuildReleaseOverlay(ctx, synthRoot(), overlay, loaderfile.LoadOptions{})
	if err != nil {
		return cue.Value{}, fmt.Errorf("synth.Release: %w", err)
	}
	return val, nil
}

// buildOverlay assembles the in-memory load.Source overlay for the synthesized
// release package: the fabricated module file, the release file, and (only when
// Values is supplied) the rendered values file. Keys are absolute paths under
// synthRoot so the loader treats them as the package's files.
func buildOverlay(in ReleaseInput, coreVersion string) (map[string]load.Source, error) {
	root := synthRoot()
	overlay := map[string]load.Source{
		filepath.Join(root, "cue.mod", "module.cue"): load.FromString(renderModuleFile(in, coreVersion)),
		filepath.Join(root, "release.cue"):           load.FromString(renderReleaseFile(in)),
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
