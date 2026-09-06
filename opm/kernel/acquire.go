package kernel

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"

	loaderfile "github.com/open-platform-model/library/opm/helper/loader/file"
	"github.com/open-platform-model/library/opm/internal/sourcetree"
	"github.com/open-platform-model/library/opm/internal/valuesfile"
	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/platform"
	"github.com/open-platform-model/library/opm/schema"
)

// valuesFileName is the name of the rendered values file
// [Kernel.AcquireInstanceFromDir] overlays beside an instance package's
// on-disk files when [WithValues] is given. The name is reserved so it can
// never shadow a file the package authored (an overlay entry replaces the
// on-disk file of the same path).
const valuesFileName = "opm-values.cue"

// AcquireOption configures [Kernel.AcquireInstanceFromDir]. Options compose
// via the functional-options pattern; new options can be added in MINOR
// releases without breaking existing call sites.
type AcquireOption func(*acquireConfig)

type acquireConfig struct {
	values []Source
}

// WithValues layers extra values sources onto the on-disk instance package
// [Kernel.AcquireInstanceFromDir] acquires: the same [Source] type
// [Kernel.ValidateConfigDetailed] takes, in stack order. The sources are
// unified, rendered as a package file declaring the top-level `values`
// field (opm-values.cue) and placed beside the package's own files in an
// in-memory overlay, so the schema's own values unification performs the
// merge in one CUE build; nothing is filled from Go and nothing is written
// into the caller's directory. The acquired instance's Source is then
// overlay mode. A source conflicting with the package's own values or the
// module's #config fails acquisition with the conflict attributed to the
// source (its Origin), exactly as layered validation reports it.
//
// Passing no sources is the same as omitting the option.
func WithValues(sources ...Source) AcquireOption {
	return func(c *acquireConfig) {
		c.values = append(c.values, sources...)
	}
}

// AcquirePlatformFromDir loads a #Platform CUE package from a directory and
// returns it as a typed, source-carrying [*platform.Platform]. It composes
// [Kernel.LoadPlatformPackage] (evaluation and the platform shape gate,
// identical to a direct call) with [platform.NewPlatformFromValue], then
// stamps [platform.Platform.Source] in on-disk mode: Root is the enclosing
// module root (the nearest ancestor holding cue.mod/module.cue, the directory
// itself when it is the root), Pkg the package directory relative to it, and
// Overlay nil.
//
// It is the directory peer of [Kernel.AcquireModuleFromRegistry] ("Acquire"
// returns a typed artifact that knows where its source lives) and the
// recommended entry point when the platform will be imported as a package by
// a follow-on build. A caller that wants only the raw value keeps using
// [Kernel.LoadPlatformPackage]. opts.Registry, when non-empty, is applied via
// the load configuration's environment, never os.Setenv, exactly as the
// loader applies it.
//
// Loader failures propagate unchanged (missing directory, no package, or a
// shape-gate sentinel such as [loaderfile.ErrWrongKind]); no partial platform
// is returned.
func (k *Kernel) AcquirePlatformFromDir(ctx context.Context, dirPath string, opts loaderfile.LoadOptions) (*platform.Platform, error) {
	absDir, err := filepath.Abs(dirPath)
	if err != nil {
		return nil, fmt.Errorf("Kernel.AcquirePlatformFromDir: resolving platform directory: %w", err)
	}
	val, err := k.LoadPlatformPackage(ctx, absDir, opts)
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
// and returns it as a validated, source-carrying [*module.Instance]. It
// composes [Kernel.LoadInstancePackage] (evaluation and the instance shape
// gate) with the kernel's instance processing — concreteness on the whole
// built spec, so the package must already be fully concrete, as an authored
// instance package is, and metadata decoding — then stamps [module.Instance.Source] in
// on-disk mode: Overlay is nil, Root is the enclosing module root (the
// nearest ancestor holding cue.mod/module.cue, the directory itself when it
// is the root) and Pkg the package directory relative to it, so a
// package in a subdirectory of its module imports correctly from a follow-on
// build.
//
// With [WithValues], caller-supplied values sources are layered onto the
// package: the on-disk files under the module root are read into an
// in-memory overlay, the unified sources are rendered as a package file
// declaring `values` (opm-values.cue) beside the package's own files, and
// the package is built in one pass through the same instance shape gate, so
// the merge is the schema's own values unification in CUE. The returned
// Source is then overlay mode: the same Root and Pkg, with Overlay carrying
// every on-disk .cue file plus the rendered values file, exactly as
// load.Config.Overlay expects, so [Kernel.Render] imports the layered
// package by source. The caller's directory is never written to.
//
// This is the same bar [Kernel.SynthesizeInstance] output meets, and the
// recommended entry point when the instance will be imported as a package by a
// follow-on build. Draft flows that need an unvalidated value keep using
// [Kernel.LoadInstancePackage]. opts.Registry is applied as the loader applies
// it (load configuration environment, never os.Setenv).
//
// Loader failures propagate unchanged (missing directory, no package, or a
// shape-gate sentinel); a non-concrete package surfaces the concreteness
// error, framed `instance "<name>": …`. No partial instance is returned.
func (k *Kernel) AcquireInstanceFromDir(ctx context.Context, dirPath string, opts loaderfile.LoadOptions, acquireOpts ...AcquireOption) (*module.Instance, error) {
	absDir, err := filepath.Abs(dirPath)
	if err != nil {
		return nil, fmt.Errorf("Kernel.AcquireInstanceFromDir: resolving instance directory: %w", err)
	}
	cfg := acquireConfig{}
	for _, opt := range acquireOpts {
		opt(&cfg)
	}

	var (
		spec cue.Value
		src  *module.Source
	)
	if len(cfg.values) == 0 {
		spec, err = k.LoadInstancePackage(ctx, absDir, opts)
		if err != nil {
			return nil, err
		}
		src = sourceForDir(absDir)
	} else {
		spec, src, err = k.loadInstanceWithValues(ctx, absDir, opts, cfg.values)
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

// loadInstanceWithValues builds the on-disk instance package at absDir with
// the unified values sources overlaid as a rendered package file, and
// returns the built value with the overlay-mode Source the build used.
func (k *Kernel) loadInstanceWithValues(ctx context.Context, absDir string, opts loaderfile.LoadOptions, sources []Source) (cue.Value, *module.Source, error) {
	info, err := os.Stat(absDir)
	if err != nil {
		return cue.Value{}, nil, fmt.Errorf("accessing instance directory %q: %w", absDir, err)
	}
	if !info.IsDir() {
		return cue.Value{}, nil, fmt.Errorf("instance path %q is not a directory", absDir)
	}

	merged := sources[0].Value
	for _, s := range sources[1:] {
		merged = merged.Unify(s.Value)
	}
	if err := merged.Err(); err != nil {
		return cue.Value{}, nil, fmt.Errorf("Kernel.AcquireInstanceFromDir: unifying values sources: %w", err)
	}

	// The source's package directory (Root joined with Pkg) is absDir itself,
	// the directory the rendered values file joins.
	src := sourceForDir(absDir)
	pkgName, err := sourcetree.PackageName(src)
	if err != nil {
		return cue.Value{}, nil, fmt.Errorf("Kernel.AcquireInstanceFromDir: %w: %w", err, loaderfile.ErrInvalidPackage)
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
		overlay[filepath.Join(absDir, valuesFileName)] = load.FromBytes(rendered)
	}
	src.Overlay = overlay

	pkg := "."
	if src.Pkg != "" {
		pkg = "./" + src.Pkg
	}
	spec, err := loaderfile.BuildInstanceOverlayAt(k.cueCtx, src.Root, pkg, overlay, opts)
	if err != nil {
		if vErr := k.attributeValuesError(ctx, absDir, opts, sources); vErr != nil {
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
// values sources: it loads the package as authored, unifies the package's
// own values with the sources and validates the result against the module's
// #config exactly as [Kernel.ValidateConfigDetailed] does, so a conflict is
// reported at positions attributable to the source (its Origin) rather than
// at the rendered overlay file. It returns nil when the failure is not a
// values problem (the caller then reports the build error itself).
func (k *Kernel) attributeValuesError(ctx context.Context, absDir string, opts loaderfile.LoadOptions, sources []Source) error {
	authored, err := k.LoadInstancePackage(ctx, absDir, opts)
	if err != nil {
		return nil
	}
	configSchema := authored.LookupPath(schema.Module).LookupPath(schema.Config)
	all := make([]Source, 0, len(sources)+1)
	if own := authored.LookupPath(schema.Values); own.Exists() {
		all = append(all, Source{Value: own, Origin: absDir})
	}
	all = append(all, sources...)
	if _, vErr := validateSources(configSchema, all, true); vErr != nil {
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
