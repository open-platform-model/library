package kernel

import (
	"context"
	"fmt"

	"cuelang.org/go/cue"

	oerrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/internal/synth"
	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/schema"
)

// InstanceInput is the typed input [Kernel.SynthesizeInstance] takes. Module,
// Name and Namespace are required; the rest are optional and are filled into
// the instance only when present, so an empty field never displaces a
// schema-derived one.
//
// It carries no schema cache and no [*cue.Context]: the Kernel owns both, and
// the registry mapping is the Kernel's [WithRegistry].
type InstanceInput struct {
	// Module is the source #Module the instance deploys. Required, and it
	// MUST carry its staged source — acquire it with
	// [Kernel.AcquireModuleFromRegistry] or [Kernel.AcquireModuleFromDir].
	// Its metadata.modulePath / metadata.version identify the module the
	// synthesized package imports; because that import resolves to the
	// module's ROOT package, a module acquired from a subdirectory is
	// refused.
	Module *module.Module

	// Name is the instance name (metadata.name). Required. It must satisfy
	// the schema's #NameType regex; a violation surfaces as a CUE
	// unification error from the synthesized build.
	Name string

	// Namespace is the target namespace (metadata.namespace). Required.
	Namespace string

	// Values are the configuration sources, in stack order — the same
	// [Source] type [Kernel.ValidateConfigDetailed] and
	// [Kernel.AcquireInstanceFromDir] take. They are unified, rendered into
	// the synthesized package's values file so the merge is the schema's own
	// values unification in CUE, and checked against the module's #config at
	// their own positions after the build.
	//
	// Empty means "no values supplied": the values path is left unfilled and
	// the concreteness check then fails unless every #config field has a
	// default. Synthesis NEVER falls back to Module.debugValues; layering a
	// debug-values overlay is frontend policy.
	Values []Source

	// Labels and Annotations layer over the schema's stamped
	// module-instance.opmodel.dev/{name,uuid} labels. CUE unification merges
	// caller-supplied entries with schema-stamped ones; caller-supplied keys
	// MUST NOT collide with the schema's reserved keys.
	Labels      map[string]string
	Annotations map[string]string
}

// SynthesizeInstance builds a [*module.Instance] from typed in-memory inputs.
// It is the entry point for a caller that holds a Module and needs a fully
// validated instance, mirroring [Kernel.AcquireInstanceFromDir] for a
// directory-based CUE package; the module it takes comes from
// [Kernel.AcquireModuleFromRegistry] or [Kernel.AcquireModuleFromDir].
//
// The method stages a package importing the module inside the module's own
// staged source tree, renders the unified in.Values into it, and builds it
// once so CUE derives uuid, components, auto-secrets and standard labels and
// performs the values merge against the module's #config. It then checks the
// values sources against #config at their own positions — so a violation is
// reported with the source's Origin rather than the rendered values file —
// asserts concreteness on the whole built spec and decodes instance metadata.
// The schema is the Kernel's own cache; no additional values source is
// consulted.
//
// The returned instance carries [module.Instance.Source]: the staged tree the
// build evaluated, in overlay mode, with Pkg naming the reserved instance
// subdirectory inside the module's staged root.
//
// A missing required input fails before any build runs, wrapping the matching
// sentinel from opm/errors ([oerrors.ErrMissingModule],
// [oerrors.ErrMissingName], [oerrors.ErrMissingNamespace]); a module with no
// staged source wraps [oerrors.ErrMissingSource]; a module acquired from a
// subdirectory of its CUE module fails stating the root-package requirement.
//
// Was: SynthesizeRelease
func (k *Kernel) SynthesizeInstance(_ context.Context, in InstanceInput) (*module.Instance, error) {
	if in.Module == nil {
		return nil, fmt.Errorf("Kernel.SynthesizeInstance: %w", oerrors.ErrMissingModule)
	}
	if in.Name == "" {
		return nil, fmt.Errorf("Kernel.SynthesizeInstance: %w", oerrors.ErrMissingName)
	}
	if in.Namespace == "" {
		return nil, fmt.Errorf("Kernel.SynthesizeInstance: %w", oerrors.ErrMissingNamespace)
	}
	if !in.Module.HasSource() {
		return nil, fmt.Errorf("Kernel.SynthesizeInstance: %w", oerrors.ErrMissingSource)
	}
	// The synthesized package imports the module by its metadata.modulePath,
	// which resolves to the module's ROOT package; a module acquired from a
	// subdirectory (Source.Pkg non-empty, only Kernel.AcquireModuleFromDir
	// produces one) would import a package the build cannot reach. Refuse
	// before any build runs.
	if in.Module.Source.Pkg != "" {
		return nil, fmt.Errorf("Kernel.SynthesizeInstance: module was acquired from the subdirectory %q of its CUE module; a synthesizable module is its module's root package, because the synthesized instance imports it by module path", in.Module.Source.Pkg)
	}

	coreVersion, err := k.resolveCoreVersion()
	if err != nil {
		return nil, fmt.Errorf("Kernel.SynthesizeInstance: %w", err)
	}

	merged, err := mergeSources(in.Values)
	if err != nil {
		return nil, fmt.Errorf("Kernel.SynthesizeInstance: %w", err)
	}

	spec, src, err := synth.Instance(k.cueCtx, coreVersion, synth.Input{
		Module:      in.Module,
		Name:        in.Name,
		Namespace:   in.Namespace,
		Values:      merged,
		Labels:      in.Labels,
		Annotations: in.Annotations,
		Env:         k.loadEnv(),
	})
	if err != nil {
		return nil, fmt.Errorf("Kernel.SynthesizeInstance: %w", err)
	}

	// The build merged the sources into `values`; check them against the
	// module's #config the way layered validation does, so a type or
	// constraint violation is reported at the source's own positions. This is
	// the same pass Kernel.AcquireInstanceFromDir runs over its extra values;
	// concreteness of the whole instance is enforced by processInstance
	// afterwards.
	configSchema := spec.LookupPath(schema.Module).LookupPath(schema.Config)
	if _, vErr := validateSources(configSchema, in.Values, false); vErr != nil {
		return nil, fmt.Errorf("Kernel.SynthesizeInstance: instance %q: %w", bestEffortInstanceName(spec), vErr)
	}

	// synth.Instance bakes the merged values into the single build (as
	// values.cue), so the spec already carries them — exactly like an authored
	// instance.cue package. processInstance checks concreteness and decodes
	// metadata, the same way it processes a directory-acquired instance whose
	// values live in the package.
	inst, err := processInstance(spec)
	if err != nil {
		return nil, fmt.Errorf("Kernel.SynthesizeInstance: %w", err)
	}
	// Stamp the staged tree synth built (the module's cloned overlay plus the
	// synthesized instance package under its reserved subdirectory) so a
	// follow-on build can import the instance as a package.
	inst.Source = src
	return inst, nil
}

// resolveCoreVersion reads the core release the kernel's schema cache
// resolved, confirming the schema exposes #ModuleInstance on the way. Only the
// major reaches the synthesized package's core import; the concrete version
// the import resolves to comes from the module's own cue.mod/module.cue
// (0019 D4).
func (k *Kernel) resolveCoreVersion() (string, error) {
	schemaPkg, err := k.schemaCache.Get(k.cueCtx)
	if err != nil {
		return "", fmt.Errorf("loading schema: %w", err)
	}
	if !schemaPkg.LookupPath(cue.ParsePath("#ModuleInstance")).Exists() {
		return "", fmt.Errorf("%w: #ModuleInstance not found in resolved schema", oerrors.ErrSchemaUnavailable)
	}
	version := k.schemaCache.ResolvedVersion()
	if version == "" {
		return "", fmt.Errorf("%w: resolved core schema version unavailable, cannot derive synth core import major", oerrors.ErrSchemaUnavailable)
	}
	return version, nil
}
