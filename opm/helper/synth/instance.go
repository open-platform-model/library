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
	// surface a resolved version to derive the synthesized package's core import
	// major.
	ErrSchemaUnavailable = errors.New("synth.Instance: schema unavailable")

	// ErrMissingSource is returned when InstanceInput.Module carries no staged
	// registry source (Module.HasSource() is false). Instance constructs the
	// #ModuleInstance INSIDE the module's own staged source tree so the module's
	// already-tidied cue.mod/module.cue drives transitive resolution; it cannot
	// do so without that source. Callers acquire a source-carrying module via
	// Kernel.AcquireModuleFromRegistry. Instance never performs a registry fetch
	// of its own.
	ErrMissingSource = errors.New("synth.Instance: Module has no staged source; acquire it via Kernel.AcquireModuleFromRegistry")
)

// synthPkgDir is the reserved subdirectory, under the acquired module's staged
// root, that the synthesized instance package is overlaid into. It is a single
// non-"_"-prefixed segment so CUE does not treat it as an ignored directory and
// it cannot collide with a real module package. The instance is loaded from
// this subdirectory while the module's own root (and its cue.mod/module.cue)
// remains the build's module root.
const synthPkgDir = "opm-synth-instance"

// Instance builds a #ModuleInstance CUE value by synthesizing an in-memory CUE
// package INSIDE the acquired module's own staged source tree and evaluating it
// in a single build (ADR-003), through the same loader build-and-shape-gate path
// LoadInstancePackage uses for on-disk instance packages. The module's own
// (already-tidied at publish time) cue.mod/module.cue is the build's module
// file, so it — not a fabricated dep list — drives transitive dependency
// resolution. The synthesized package consists of an instance.cue overlaid under
// a reserved subdirectory of the module root (importing core and the module's
// own package, writing `#module: <import>` plus caller-supplied metadata) and —
// when Values is supplied — a values.cue rendered from InstanceInput.Values.
//
// Because the module is the build's main module, its own package import resolves
// LOCALLY (no fabricated dependency, no registry round-trip for the module
// itself), and the module's transitive imports (core, catalog subpackages, …)
// resolve through the module's own cue.mod/module.cue. This is why a module that
// imports a catalog subpackage synthesizes correctly where the previous
// fabricated-{core, module}-deps approach failed (library#31). The module enters
// the build by import, so there is no closed-into-closed FillPath and no Go-side
// value pre-merge: the schema's own `let unifiedModule = #module & {#config:
// values}` performs the values merge in CUE.
//
// Instance REQUIRES the module to carry staged source (Module.HasSource());
// acquire it via Kernel.AcquireModuleFromRegistry. It never fetches from a
// registry itself.
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
	if !in.Module.HasSource() {
		return cue.Value{}, ErrMissingSource
	}

	// Resolve the schema to (a) confirm #ModuleInstance is present and (b) learn
	// the core import major. Per design D4, the synth build's core (and thus
	// #ModuleInstance) resolves from the MODULE's own tidied cue.mod/module.cue,
	// not from this Cache — the Cache is consulted to confirm availability and to
	// supply the import's major selector, but it no longer pins the build's core
	// version. This is the same Cache the caller's Kernel owns, so it reuses the
	// already-memoized schema fetch.
	schemaPkg, err := in.SchemaCache.Get(ctx)
	if err != nil {
		return cue.Value{}, fmt.Errorf("synth.Instance: loading schema: %w", err)
	}
	if !schemaPkg.LookupPath(cue.ParsePath("#ModuleInstance")).Exists() {
		return cue.Value{}, fmt.Errorf("%w: #ModuleInstance not found in resolved schema", ErrSchemaUnavailable)
	}
	coreVersion := in.SchemaCache.ResolvedVersion()
	if coreVersion == "" {
		return cue.Value{}, fmt.Errorf("%w: resolved core schema version unavailable, cannot derive synth core import major", ErrSchemaUnavailable)
	}

	moduleRoot, overlay, err := buildOverlay(in, coreVersion)
	if err != nil {
		return cue.Value{}, fmt.Errorf("synth.Instance: %w", err)
	}

	// Evaluate the synthesized package through the SAME build-and-shape-gate
	// step LoadInstancePackage uses; the instance package lives in a subdirectory
	// of the module's staged root, with the module's own cue.mod/module.cue as the
	// module file. Registry resolution uses the process CUE_REGISTRY (core and any
	// catalog deps resolve from the same registry/cache the caller already used to
	// acquire the module).
	val, err := loaderfile.BuildInstanceOverlayAt(ctx, moduleRoot, "./"+synthPkgDir, overlay, loaderfile.LoadOptions{})
	if err != nil {
		return cue.Value{}, fmt.Errorf("synth.Instance: %w", err)
	}
	return val, nil
}

// buildOverlay clones the acquired module's staged overlay and adds the
// synthesized instance package's files (instance.cue, and values.cue when Values
// is supplied) under the reserved synthPkgDir subdirectory of the module's
// staged root. It returns the module root (the load.Config.ModuleRoot, so the
// module's own cue.mod/module.cue governs the build) and the augmented overlay.
//
// The module's overlay is cloned, never mutated, so a module value can be
// synthesized more than once.
func buildOverlay(in InstanceInput, coreVersion string) (string, map[string]load.Source, error) {
	src := in.Module.Source
	overlay := make(map[string]load.Source, len(src.Overlay)+2)
	for k, v := range src.Overlay {
		overlay[k] = v
	}

	pkgRoot := filepath.Join(src.Root, synthPkgDir)
	overlay[filepath.Join(pkgRoot, "instance.cue")] = load.FromString(renderInstanceFile(in, coreVersion))

	valuesSrc, err := renderValuesFile(in)
	if err != nil {
		return "", nil, err
	}
	if valuesSrc != nil {
		overlay[filepath.Join(pkgRoot, "values.cue")] = load.FromBytes(valuesSrc)
	}

	return src.Root, overlay, nil
}
