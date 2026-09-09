package kernel

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"

	oerrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/internal/cueenv"
	"github.com/open-platform-model/library/opm/internal/loader"
	"github.com/open-platform-model/library/opm/internal/sourcetree"
	"github.com/open-platform-model/library/opm/internal/valuesfile"
	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/platform"
	"github.com/open-platform-model/library/opm/schema"
)

// valuesFileName is the name of the rendered values file
// [Kernel.AcquireInstanceFromDir] overlays beside an instance package's
// on-disk files when values sources are supplied. The name is reserved so it
// can never shadow a file the package authored (an overlay entry replaces the
// on-disk file of the same path).
const valuesFileName = "opm-values.cue"

// loadEnv is the environment slice every kernel load consults: the kernel's
// [WithRegistry] mapping applied through load.Config.Env, never os.Setenv.
// Nil when no mapping was configured, so the load reads the process
// environment unchanged.
func (k *Kernel) loadEnv() []string {
	return cueenv.Override(k.registry, "")
}

// AcquireModuleFromRegistry loads a #Module published in an OCI registry by
// its major-qualified path (e.g. "example.com/modules/hello@v0") and version
// (e.g. "v0.0.2"), in a [cue.Context] created for the call, through the
// kernel's configured registry (set via [WithRegistry], inheriting
// CUE_REGISTRY from the process environment when unset). It returns a decoded
// [*module.Module] whose staged source ([module.Source]) is populated, so the
// module can be reused as the main module of a follow-on build — notably by
// [Kernel.SynthesizeInstance], which stages the instance inside the module's
// own root so the module's already-tidied cue.mod/module.cue drives
// transitive dependency resolution. A caller that wants only the raw module
// value reads Module.Package, which keeps the call's runtime alive for as
// long as the caller holds the module.
func (k *Kernel) AcquireModuleFromRegistry(ctx context.Context, modPath, version string) (*module.Module, error) {
	val, src, err := loader.FetchModule(ctx, cuecontext.New(), modPath, version, k.loadEnv())
	if err != nil {
		return nil, err
	}
	mod, err := module.NewModuleFromValue(val)
	if err != nil {
		return nil, err
	}
	mod.Source = src
	return mod, nil
}

// AcquireModuleFromDir loads a #Module CUE package from a directory and
// returns it as a typed, source-carrying [*module.Module]. It is the
// directory peer of [Kernel.AcquireModuleFromRegistry]: the package is
// evaluated and shape-gated exactly as the registry path gates a fetched
// module, [module.NewModuleFromValue] constructs the typed artifact, and
// [module.Source] is stamped in OVERLAY mode — Root the enclosing module
// root (the nearest ancestor holding cue.mod/module.cue, the directory
// itself when it is the root or when no ancestor holds one), Pkg the package
// directory relative to it, and Overlay every .cue file under Root (the
// module's own cue.mod/module.cue included) keyed by its absolute path.
//
// Stamping the overlay is what makes the acquired module a valid
// [Kernel.SynthesizeInstance] input ([module.Module.HasSource] reports true):
// synthesis stages the instance package inside the module's own tree, so a
// frontend rendering from a module directory no longer walks that tree
// itself. Because the synthesized package imports the module by its module
// path — which resolves to the module's ROOT package — synthesis refuses a
// module whose Source.Pkg is non-empty; acquiring a subdirectory package is
// still valid for reading its value and metadata.
//
// The package is built in a [cue.Context] created for the call; Module.Package
// keeps it alive for as long as the caller holds the module. The registry
// mapping is the kernel's ([WithRegistry]), applied via the load
// configuration's environment and never os.Setenv. The caller's directory is
// never written to.
//
// Shape-gate failures propagate unchanged (missing directory, no package, or
// a sentinel such as [oerrors.ErrWrongKind]); no partial module is returned.
func (k *Kernel) AcquireModuleFromDir(_ context.Context, dirPath string) (*module.Module, error) {
	absDir, err := filepath.Abs(dirPath)
	if err != nil {
		return nil, fmt.Errorf("Kernel.AcquireModuleFromDir: resolving module directory: %w", err)
	}
	val, err := loader.LoadDir(cuecontext.New(), absDir, ".", nil, k.loadEnv(), loader.ModuleSpec)
	if err != nil {
		return nil, err
	}
	mod, err := module.NewModuleFromValue(val)
	if err != nil {
		return nil, fmt.Errorf("Kernel.AcquireModuleFromDir: %w", err)
	}
	src := sourceForDir(absDir)
	overlay, err := sourcetree.OverlayFromDir(src.Root)
	if err != nil {
		return nil, fmt.Errorf("Kernel.AcquireModuleFromDir: %w", err)
	}
	src.Overlay = overlay
	mod.Source = src
	return mod, nil
}

// AcquirePlatformFromDir loads a #Platform CUE package from a directory and
// returns it as a typed, source-carrying [*platform.Platform]. The package is
// evaluated and run through the platform shape gate, then
// [platform.NewPlatformFromValue] constructs the typed artifact and
// [platform.Platform.Source] is stamped in on-disk mode: Root is the enclosing
// module root (the nearest ancestor holding cue.mod/module.cue, the directory
// itself when it is the root), Pkg the package directory relative to it, and
// Overlay nil.
//
// It is the directory peer of [Kernel.AcquireModuleFromRegistry] ("Acquire"
// returns a typed artifact that knows where its source lives) and the only
// way to obtain a platform: a caller that wants the raw value reads
// Platform.Package, which keeps the call's [cue.Context] alive for as long
// as the caller holds the platform; the kernel retains nothing. The registry
// mapping used for the platform's catalog imports is the kernel's
// ([WithRegistry]), applied via the load configuration's environment and
// never os.Setenv; the verb takes no per-call override.
//
// Loader failures propagate unchanged (missing directory, no package, or a
// shape-gate sentinel such as [oerrors.ErrWrongKind]); no partial platform is
// returned.
func (k *Kernel) AcquirePlatformFromDir(_ context.Context, dirPath string) (*platform.Platform, error) {
	absDir, err := filepath.Abs(dirPath)
	if err != nil {
		return nil, fmt.Errorf("Kernel.AcquirePlatformFromDir: resolving platform directory: %w", err)
	}
	val, err := loader.LoadDir(cuecontext.New(), absDir, ".", nil, k.loadEnv(), loader.PlatformSpec)
	if err != nil {
		return nil, err
	}
	plat, err := platform.NewPlatformFromValue(val)
	if err != nil {
		return nil, fmt.Errorf("Kernel.AcquirePlatformFromDir: %w", err)
	}
	plat.Source = sourceForDir(absDir)
	return plat, nil
}

// AcquireInstanceFromDir loads a #ModuleInstance CUE package from a directory
// and returns it as a validated, source-carrying [*module.Instance]. The
// package is evaluated, run through the instance shape gate and then the
// kernel's instance processing — concreteness on the whole built spec, so the
// package must already be fully concrete, as an authored instance package is,
// and metadata decoding — and [module.Instance.Source] is stamped in on-disk
// mode: Overlay is nil, Root is the enclosing module root (the nearest
// ancestor holding cue.mod/module.cue, the directory itself when it is the
// root) and Pkg the package directory relative to it, so a package in a
// subdirectory of its module imports correctly from a follow-on build.
//
// The trailing values sources — the same [Source] type
// [Kernel.ValidateConfigDetailed] takes, in stack order — layer extra values
// onto the package: the sources are compiled in the call's own context, the
// on-disk files under the module root are read into an in-memory overlay, the
// unified sources are rendered as a package file declaring `values`
// (opm-values.cue) beside the package's own files, and the package is built
// in one pass through the same instance shape gate, so the merge is the
// schema's own values unification in CUE. Nothing is filled from Go and
// nothing is written into the caller's directory. The returned Source is
// then overlay mode: the same Root and Pkg, with Overlay carrying every
// on-disk .cue file plus the rendered values file, exactly as
// load.Config.Overlay expects, so [Kernel.Render] imports the layered package
// by source. A source conflicting with the package's own values or the
// module's #config fails acquisition with the conflict attributed to the
// source (its Origin), exactly as layered validation reports it.
//
// Passing no sources is the "no values supplied" path: the package is built
// from disk as authored and the Source stays on-disk mode.
//
// The build runs in a [cue.Context] created for the call; Instance.Package
// keeps it alive for as long as the caller holds the instance. This is the
// same bar [Kernel.SynthesizeInstance] output meets. Loader failures
// propagate unchanged (missing directory, no package, or a shape-gate
// sentinel); a non-concrete package surfaces the concreteness error, framed
// `instance "<name>": …`. No partial instance is returned.
func (k *Kernel) AcquireInstanceFromDir(_ context.Context, dirPath string, values ...Source) (*module.Instance, error) {
	absDir, err := filepath.Abs(dirPath)
	if err != nil {
		return nil, fmt.Errorf("Kernel.AcquireInstanceFromDir: resolving instance directory: %w", err)
	}

	cueCtx := cuecontext.New()
	var (
		spec cue.Value
		src  *module.Source
	)
	if len(values) == 0 {
		spec, err = loader.LoadDir(cueCtx, absDir, ".", nil, k.loadEnv(), loader.InstanceSpec)
		if err != nil {
			return nil, err
		}
		src = sourceForDir(absDir)
	} else {
		spec, src, err = k.loadInstanceWithValues(cueCtx, absDir, values)
		if err != nil {
			return nil, err
		}
	}

	inst, err := processInstance(spec)
	if err != nil {
		return nil, fmt.Errorf("Kernel.AcquireInstanceFromDir: %w", err)
	}
	inst.Source = src
	return inst, nil
}

// mergeSources compiles a values stack in cueCtx, unifies it in order and
// returns the merged value. It is the one merge both values paths run: the
// extra sources of [Kernel.AcquireInstanceFromDir] and InstanceInput.Values
// on [Kernel.SynthesizeInstance], each in the context of the build the merged
// value is rendered into. An empty stack, or one whose sources carry no
// values, is the zero value with no error — the "no values supplied" path.
func mergeSources(cueCtx *cue.Context, sources []Source) (cue.Value, error) {
	values, err := compileSources(cueCtx, sources)
	if err != nil {
		return cue.Value{}, fmt.Errorf("compiling values sources: %w", err)
	}
	merged := unifyValues(values)
	if !merged.Exists() {
		return cue.Value{}, nil
	}
	if err := merged.Err(); err != nil {
		return cue.Value{}, fmt.Errorf("unifying values sources: %w", err)
	}
	return merged, nil
}

// loadInstanceWithValues builds the on-disk instance package at absDir in
// cueCtx with the unified values sources overlaid as a rendered package
// file, and returns the built value with the overlay-mode Source the build
// used.
func (k *Kernel) loadInstanceWithValues(cueCtx *cue.Context, absDir string, sources []Source) (cue.Value, *module.Source, error) {
	info, err := os.Stat(absDir)
	if err != nil {
		return cue.Value{}, nil, fmt.Errorf("accessing instance directory %q: %w", absDir, err)
	}
	if !info.IsDir() {
		return cue.Value{}, nil, fmt.Errorf("instance path %q is not a directory", absDir)
	}

	merged, err := mergeSources(cueCtx, sources)
	if err != nil {
		return cue.Value{}, nil, fmt.Errorf("Kernel.AcquireInstanceFromDir: %w", err)
	}

	// The source's package directory (Root joined with Pkg) is absDir itself,
	// the directory the rendered values file joins.
	src := sourceForDir(absDir)
	pkgName, err := sourcetree.PackageName(src)
	if err != nil {
		return cue.Value{}, nil, fmt.Errorf("Kernel.AcquireInstanceFromDir: %w: %w", err, oerrors.ErrInvalidPackage)
	}
	rendered, err := valuesfile.Render(pkgName, merged)
	if err != nil {
		return cue.Value{}, nil, fmt.Errorf("Kernel.AcquireInstanceFromDir: %w", err)
	}

	overlay, err := sourcetree.OverlayFromDir(src.Root)
	if err != nil {
		return cue.Value{}, nil, fmt.Errorf("Kernel.AcquireInstanceFromDir: %w", err)
	}
	if rendered != nil {
		overlay[filepath.Join(absDir, valuesFileName)] = rendered
	}
	src.Overlay = overlay

	pkg := "."
	if src.Pkg != "" {
		pkg = "./" + src.Pkg
	}
	spec, err := loader.LoadDir(cueCtx, src.Root, pkg, overlay, k.loadEnv(), loader.InstanceSpec)
	if err != nil {
		if vErr := k.attributeValuesError(cueCtx, absDir, sources); vErr != nil {
			return cue.Value{}, nil, vErr
		}
		return cue.Value{}, nil, err
	}

	// The build merged the sources into `values`; check the sources against
	// the module's #config the way layered validation does, so a type or
	// constraint violation is reported at the source's own positions rather
	// than left for a later evaluation to trip over. Concreteness of the
	// whole instance is enforced by processInstance afterwards.
	configSchema := spec.LookupPath(schema.Module).LookupPath(schema.Config)
	if _, vErr := validateSources(configSchema, sources, false); vErr != nil {
		name := bestEffortInstanceName(spec)
		return cue.Value{}, nil, fmt.Errorf("Kernel.AcquireInstanceFromDir: instance %q: %w", name, vErr)
	}
	return spec, src, nil
}

// attributeValuesError explains a failed layered build in terms of the
// values sources: it loads the package as authored in cueCtx, compiles the
// sources in that same context, unifies the package's own values with them
// and validates the result against the module's #config exactly as
// [Kernel.ValidateConfigDetailed] does, so a conflict is reported at
// positions attributable to the source (its Origin) rather than at the
// rendered overlay file. It returns nil when the failure is not a values
// problem (the caller then reports the build error itself).
func (k *Kernel) attributeValuesError(cueCtx *cue.Context, absDir string, sources []Source) error {
	authored, err := loader.LoadDir(cueCtx, absDir, ".", nil, k.loadEnv(), loader.InstanceSpec)
	if err != nil {
		return nil
	}
	configSchema := authored.LookupPath(schema.Module).LookupPath(schema.Config)
	all := make([]cue.Value, 0, len(sources)+1)
	if own := authored.LookupPath(schema.Values); own.Exists() {
		all = append(all, own)
	}
	compiled, err := compileSources(cueCtx, sources)
	if err != nil {
		return fmt.Errorf("Kernel.AcquireInstanceFromDir: instance %q: compiling values sources: %w", bestEffortInstanceName(authored), err)
	}
	all = append(all, compiled...)
	if _, vErr := validateValues(configSchema, all, true); vErr != nil {
		name := bestEffortInstanceName(authored)
		return fmt.Errorf("Kernel.AcquireInstanceFromDir: instance %q: %w", name, vErr)
	}
	return nil
}

// sourceForDir describes an on-disk package directory as a Source: Root is
// the nearest ancestor (the directory itself included) holding
// cue.mod/module.cue, Pkg the slash-separated path of dir relative to it.
// A directory with no enclosing module is its own root with an empty Pkg,
// which is what a module-less package loads as today.
func sourceForDir(absDir string) *module.Source {
	dir := absDir
	for {
		if _, err := os.Stat(filepath.Join(dir, "cue.mod", "module.cue")); err == nil {
			rel, err := filepath.Rel(dir, absDir)
			if err != nil {
				break
			}
			if rel == "." {
				rel = ""
			}
			return &module.Source{Root: dir, Pkg: filepath.ToSlash(rel)}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return &module.Source{Root: absDir}
}
