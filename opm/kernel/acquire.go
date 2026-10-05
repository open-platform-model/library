package kernel

import (
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"

	"github.com/open-platform-model/library/opm/catalog"
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

// loadEnv is the environment slice a kernel load consults when the Kernel
// has no registry client (a Kernel not built by [New]): the kernel's
// [WithRegistry] mapping applied through load.Config.Env by
// [cueenv.Override], which owns the rule that the process environment is
// never written. Nil when no mapping was configured, so the load reads the
// process environment unchanged.
func (k *Kernel) loadEnv() []string {
	return cueenv.Override(k.registry, "")
}

// loadOptions starts one kernel operation and returns the load settings
// every load and fetch of that operation passes on. On a Kernel built by
// [New] they carry one operation of the Kernel's shared registry client: the
// client itself (resolver and transport, built on first use) under a module
// cache of this operation's own, and the environment slice that operation
// read. On any other Kernel (the zero value) they carry only the
// environment and leave Registry unset, so cue/load builds its own registry
// for each load as before; a nil pointer never reaches the interface field.
// Each verb calls it once.
func (k *Kernel) loadOptions() loader.Options {
	if k.registryClient == nil {
		return loader.Options{Env: k.loadEnv()}
	}
	op := k.registryClient.Operation()
	return loader.Options{Env: op.Env(), Registry: op}
}

// AcquireModuleFromRegistry loads a #Module published in an OCI registry by
// its major-qualified path (e.g. "example.com/modules/hello@v0") and version,
// written bare or v-prefixed (e.g. "0.0.2" or "v0.0.2"; both fetch the same
// tag), in a [cue.Context] created for the call, through the kernel's
// configured registry (set via [WithRegistry], inheriting CUE_REGISTRY from
// the process environment when unset). It returns a decoded [*module.Module]
// whose staged source ([module.Source]) is populated, so the module can be
// reused as the main module of a follow-on build — notably by
// [Kernel.SynthesizeInstance], which stages the instance inside the module's
// own root so the module's already-tidied cue.mod/module.cue drives
// transitive dependency resolution. A caller that wants only the raw module
// value reads Module.Package, which keeps the call's runtime alive for as
// long as the caller holds the module.
func (k *Kernel) AcquireModuleFromRegistry(ctx context.Context, modPath, version string) (*module.Module, error) {
	val, src, err := loader.FetchModule(ctx, cuecontext.New(), modPath, version, k.loadOptions())
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
// module's own cue.mod/module.cue included) keyed by its absolute path. The
// package is built from that overlay, so Package and Source.Overlay come
// from one read of the tree; a non-CUE file the package embeds is read from
// the directory beneath the overlay.
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
// mapping is the kernel's ([WithRegistry]). The caller's directory is never
// written to.
//
// Shape-gate failures propagate unchanged (missing directory, no package, or
// a sentinel such as [oerrors.ErrWrongKind]); no partial module is returned.
func (k *Kernel) AcquireModuleFromDir(ctx context.Context, dirPath string) (*module.Module, error) {
	const verb = "Kernel.AcquireModuleFromDir"
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	val, src, err := k.acquireDir(ctx, cuecontext.New(), verb, dirPath, loader.ModuleSpec, true)
	if err != nil {
		return nil, err
	}
	mod, err := module.NewModuleFromValue(val)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", verb, err)
	}
	mod.Source = src
	return mod, nil
}

// dirSource resolves dirPath, checks that it is a directory, and describes
// the package in it as a Source: Root the enclosing module root and Pkg the
// package directory relative to it ([sourceForDir]). With withOverlay it is
// overlay mode, every .cue file under Root read once into Overlay; without,
// it is on-disk mode (Overlay nil). The path check runs before anything else
// touches the tree, so a missing directory or a file path fails with the
// loader's path error ([loader.CheckDir]) rather than as an unreadable tree.
func dirSource(verb, dirPath string, spec loader.ArtifactSpec, withOverlay bool) (*module.Source, error) {
	absDir, err := filepath.Abs(dirPath)
	if err != nil {
		return nil, fmt.Errorf("%s: resolving %s directory: %w", verb, spec.Label, err)
	}
	if err := loader.CheckDir(absDir, spec); err != nil {
		return nil, err
	}
	src := sourceForDir(absDir)
	if withOverlay {
		overlay, err := sourcetree.OverlayFromDir(src.Root)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", verb, err)
		}
		src.Overlay = overlay
	}
	return src, nil
}

// acquireDir is the one read-then-build step every directory verb runs:
// [dirSource] describes the directory, and the package is built in cueCtx
// from that same Source, so an overlay-mode artifact's Package and
// Source.Overlay come from one read. Which verb stamps which mode:
//
//   - module and catalog: overlay mode, built from the overlay they stamp;
//   - platform, and an instance with no values sources: on-disk mode.
//
// A load or shape-gate error is returned unwrapped, as each verb reports it,
// and so is ctx's error when ctx is done once the tree is read or once the
// package is built.
func (k *Kernel) acquireDir(ctx context.Context, cueCtx *cue.Context, verb, dirPath string, spec loader.ArtifactSpec, withOverlay bool) (cue.Value, *module.Source, error) {
	src, err := dirSource(verb, dirPath, spec, withOverlay)
	if err != nil {
		return cue.Value{}, nil, err
	}
	if err := ctx.Err(); err != nil {
		return cue.Value{}, nil, err
	}
	val, err := loader.LoadDir(cueCtx, src, k.loadOptions(), spec)
	if err != nil {
		return cue.Value{}, nil, err
	}
	if err := ctx.Err(); err != nil {
		return cue.Value{}, nil, err
	}
	return val, src, nil
}

// AcquireCatalogFromRegistry loads a #Catalog published in an OCI registry by
// its major-qualified path (e.g. "opmodel.dev/catalogs/opm@v4") and version,
// written bare or v-prefixed (e.g. "4.3.0" or "v4.3.0"; both fetch the same
// tag), in a [cue.Context] created for the call, through the kernel's
// configured registry (set via [WithRegistry], inheriting CUE_REGISTRY from
// the process environment when unset). It is the registry peer of
// [Kernel.AcquireCatalogFromDir] and the exact counterpart of
// [Kernel.AcquireModuleFromRegistry]: one fetch routine serves both, and the
// only thing that differs is the shape it gates to.
//
// It returns a decoded [*catalog.Catalog] whose staged source
// ([catalog.Source]) is populated in overlay mode, so
// [catalog.Catalog.Requires] reads the catalog's committed
// cue.mod/module.cue without a second fetch. A caller that wants the raw
// value reads Catalog.Package, which keeps the call's runtime alive for as
// long as the caller holds the catalog.
//
// The catalog is evaluated and shape-gated (concrete kind == "Catalog";
// concrete metadata.modulePath and metadata.version), never fully
// schema-validated, which remains the kernel's contract. A gate failure
// propagates unchanged, wrapping the shared sentinels
// ([oerrors.ErrWrongKind] for an artifact of another kind or of none); no
// partial catalog is returned. Unlike the module path, the fetched
// coordinate is not compared against the catalog's declared one — see
// loader.FetchModule for why that check is the module's alone.
//
// The kernel judges nothing beyond the shape: what a catalog provides and
// what it requires are reported by [catalog.Catalog.Provides] and
// [catalog.Catalog.Requires] on demand, and what either means is the
// caller's.
func (k *Kernel) AcquireCatalogFromRegistry(ctx context.Context, modPath, version string) (*catalog.Catalog, error) {
	// The catalog kind and the fetch routine it shares with modules are ADR-009.
	val, src, err := loader.FetchArtifact(ctx, cuecontext.New(), modPath, version, k.loadOptions(), loader.CatalogSpec)
	if err != nil {
		return nil, err
	}
	cat, err := catalog.NewCatalogFromValue(val)
	if err != nil {
		return nil, fmt.Errorf("Kernel.AcquireCatalogFromRegistry: %w", err)
	}
	cat.Source = src
	return cat, nil
}

// AcquireCatalogFromDir loads a #Catalog CUE package from a directory and
// returns it as a typed, source-carrying [*catalog.Catalog]. It is the
// directory peer of [Kernel.AcquireCatalogFromRegistry], and it mirrors
// [Kernel.AcquireModuleFromDir] exactly: the package is evaluated and
// shape-gated as the registry path gates a fetched catalog,
// [catalog.NewCatalogFromValue] constructs the typed artifact, and
// [catalog.Source] is stamped in OVERLAY mode — Root the enclosing module
// root (the nearest ancestor holding cue.mod/module.cue, the directory
// itself when it is the root or when no ancestor holds one), Pkg the package
// directory relative to it, and Overlay every .cue file under Root (the
// module's own cue.mod/module.cue included) keyed by its absolute path. As
// for the module, the package is built from that overlay, so Package and
// Source.Overlay come from one read of the tree.
//
// Overlay mode is what makes [catalog.Catalog.Requires] answer identically
// whichever route acquired the catalog: the committed cue.mod/module.cue is
// carried on the artifact either way.
//
// The package is built in a [cue.Context] created for the call;
// Catalog.Package keeps it alive for as long as the caller holds the
// catalog. The registry mapping for the catalog's own imports is the
// kernel's ([WithRegistry]). The caller's directory is never written to.
//
// Shape-gate failures propagate unchanged (missing directory, no package, or
// a sentinel such as [oerrors.ErrWrongKind]); no partial catalog is returned.
func (k *Kernel) AcquireCatalogFromDir(ctx context.Context, dirPath string) (*catalog.Catalog, error) {
	const verb = "Kernel.AcquireCatalogFromDir"
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	val, src, err := k.acquireDir(ctx, cuecontext.New(), verb, dirPath, loader.CatalogSpec, true)
	if err != nil {
		return nil, err
	}
	cat, err := catalog.NewCatalogFromValue(val)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", verb, err)
	}
	cat.Source = src
	return cat, nil
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
// ([WithRegistry]); the verb takes no per-call override.
//
// Loader failures propagate unchanged (missing directory, no package, or a
// shape-gate sentinel such as [oerrors.ErrWrongKind]); no partial platform is
// returned.
func (k *Kernel) AcquirePlatformFromDir(ctx context.Context, dirPath string) (*platform.Platform, error) {
	const verb = "Kernel.AcquirePlatformFromDir"
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	val, src, err := k.acquireDir(ctx, cuecontext.New(), verb, dirPath, loader.PlatformSpec, false)
	if err != nil {
		return nil, err
	}
	plat, err := platform.NewPlatformFromValue(val)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", verb, err)
	}
	plat.Source = src
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
// The package's own `values` (values.cue, or a `values` field in
// instance.cue) are checked against the module's #config on every acquire,
// with or without sources, the same way a source is: a key the schema lacks,
// a type mismatch or a violated constraint is refused at the position of the
// file that holds it. The check does not require concreteness; the instance
// processing step that follows does.
//
// The build runs in a [cue.Context] created for the call; Instance.Package
// keeps it alive for as long as the caller holds the instance. This is the
// same bar [Kernel.SynthesizeInstance] output meets. Loader failures
// propagate unchanged (missing directory, no package, or a shape-gate
// sentinel); a non-concrete package surfaces the concreteness error, framed
// `instance "<name>": …`. No partial instance is returned.
func (k *Kernel) AcquireInstanceFromDir(ctx context.Context, dirPath string, values ...Source) (*module.Instance, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	cueCtx := cuecontext.New()
	var (
		spec     cue.Value
		src      *module.Source
		compiled []cue.Value
		err      error
	)
	if len(values) == 0 {
		spec, src, err = k.acquireDir(ctx, cueCtx, "Kernel.AcquireInstanceFromDir", dirPath, loader.InstanceSpec, false)
	} else {
		spec, src, compiled, err = k.loadInstanceWithValues(ctx, cueCtx, dirPath, values)
	}
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if err := checkInstanceValues(spec, compiled); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	inst, err := processInstance(spec)
	if err != nil {
		return nil, fmt.Errorf("Kernel.AcquireInstanceFromDir: %w", err)
	}
	inst.Source = src
	return inst, nil
}

// mergeSources compiles a values stack in cueCtx once, unifies it in order
// and returns both the compiled values and their merge. It is the one merge
// both values paths run: the extra sources of [Kernel.AcquireInstanceFromDir]
// and InstanceInput.Values on [Kernel.SynthesizeInstance], each in the
// context of the build the merged value is rendered into, and each
// file-backed source through opts, the operation's load settings
// ([Kernel.loadOptions]). The compiled values live in that same context, so the
// checks after the build validate them as they are, with no second compile.
// An empty stack, or one whose sources carry no values, merges to the zero
// value with no error — the "no values supplied" path.
func mergeSources(cueCtx *cue.Context, sources []Source, opts loader.Options) (compiled []cue.Value, merged cue.Value, err error) {
	compiled, err = compileSources(cueCtx, sources, opts)
	if err != nil {
		return nil, cue.Value{}, fmt.Errorf("compiling values sources: %w", err)
	}
	merged = unifyValues(compiled)
	if !merged.Exists() {
		return compiled, cue.Value{}, nil
	}
	if err := merged.Err(); err != nil {
		return nil, cue.Value{}, fmt.Errorf("unifying values sources: %w", err)
	}
	return compiled, merged, nil
}

// loadInstanceWithValues builds the instance package at dirPath in cueCtx
// with the unified values sources layered on as a rendered package file. The
// directory is read once into the authored overlay ([dirSource]); the build
// runs from a copy of it plus the rendered values file, and the returned
// overlay-mode Source is the one that build used. The authored overlay
// itself is kept unchanged for attributing a failed build to the sources.
// The sources' compiled values are returned with the build, for the check
// that follows it.
func (k *Kernel) loadInstanceWithValues(ctx context.Context, cueCtx *cue.Context, dirPath string, sources []Source) (cue.Value, *module.Source, []cue.Value, error) {
	const verb = "Kernel.AcquireInstanceFromDir"
	authored, err := dirSource(verb, dirPath, loader.InstanceSpec, true)
	if err != nil {
		return cue.Value{}, nil, nil, err
	}
	if err := ctx.Err(); err != nil {
		return cue.Value{}, nil, nil, err
	}

	opts := k.loadOptions()
	compiled, merged, err := mergeSources(cueCtx, sources, opts)
	if err != nil {
		return cue.Value{}, nil, nil, fmt.Errorf("%s: %w", verb, err)
	}
	if err := ctx.Err(); err != nil {
		return cue.Value{}, nil, nil, err
	}

	pkgName, err := sourcetree.PackageName(authored)
	if err != nil {
		return cue.Value{}, nil, nil, fmt.Errorf("%s: %w: %w", verb, err, oerrors.ErrInvalidPackage)
	}
	rendered, err := valuesfile.Render(pkgName, merged)
	if err != nil {
		return cue.Value{}, nil, nil, fmt.Errorf("%s: %w", verb, err)
	}

	// The rendered values file joins the package directory (Root joined with
	// Pkg), replacing an authored file of the same name in the layered build
	// only.
	overlay := maps.Clone(authored.Overlay)
	if rendered != nil {
		overlay[filepath.Join(authored.Root, filepath.FromSlash(authored.Pkg), valuesFileName)] = rendered
	}
	src := &module.Source{Root: authored.Root, Pkg: authored.Pkg, Overlay: overlay}

	spec, err := loader.LoadDir(cueCtx, src, opts, loader.InstanceSpec)
	if err != nil {
		if vErr := attributeValuesError(cueCtx, opts, authored, compiled); vErr != nil {
			return cue.Value{}, nil, nil, vErr
		}
		return cue.Value{}, nil, nil, err
	}

	return spec, src, compiled, nil
}

// checkInstanceValues validates, on every acquire, the values the built spec
// carries against the module's #config, so a key the schema does not have, a
// type mismatch or a constraint violation is refused wherever the value was
// written: the package's own `values` (values.cue, or a `values` field in
// instance.cue) or a trailing source. #ModuleInstance only unifies `values`
// into a let binding the output never reads, so CUE alone reports nothing
// for a key no component consumes; this is the check that does.
//
// The sources' values (compiled once by [mergeSources], in the context the
// spec was built in; nil when the call passed none) are checked first, on
// their own, so their errors name each source's Origin rather than the
// rendered overlay file the build merged them through. The built `values` is
// checked second: with the sources already clean, any disallowed key left in
// it comes from the package's own files, whose positions the built value
// keeps. Concreteness is not asserted (requireConcrete false): valid but
// incomplete values pass here and are held to concreteness by the instance
// processing step afterwards.
func checkInstanceValues(spec cue.Value, compiled []cue.Value) error {
	configSchema := spec.LookupPath(schema.Module).LookupPath(schema.Config)
	if _, err := validateCompiled(configSchema, compiled, false); err != nil {
		return fmt.Errorf("Kernel.AcquireInstanceFromDir: instance %q: %w", bestEffortInstanceName(spec), err)
	}
	built := spec.LookupPath(schema.Values)
	if !configSchema.Exists() || !built.Exists() {
		return nil
	}
	if _, err := validateValues(configSchema, []cue.Value{built}, false); err != nil {
		return fmt.Errorf("Kernel.AcquireInstanceFromDir: instance %q: %w", bestEffortInstanceName(spec), err)
	}
	return nil
}

// attributeValuesError explains a failed layered build in terms of the
// values sources: it builds the package as authored in cueCtx from the
// authored Source (the overlay already read for the layered build, without
// the rendered values file), unifies the package's own values with the
// sources' values (compiled once by [mergeSources], in that same context)
// and validates the result against the module's #config without requiring
// concreteness (see [valuesConflict]), so a conflict is reported at
// positions attributable to the source (its Origin) rather than at the
// rendered overlay file. It returns nil when the failure is not a values
// problem (the caller then reports the build error itself). opts are the
// load settings of the operation whose build failed.
func attributeValuesError(cueCtx *cue.Context, opts loader.Options, authoredSrc *module.Source, compiled []cue.Value) error {
	authored, err := loader.LoadDir(cueCtx, authoredSrc, opts, loader.InstanceSpec)
	if err != nil {
		return nil
	}
	if vErr := valuesConflict(authored, compiled); vErr != nil {
		return fmt.Errorf("Kernel.AcquireInstanceFromDir: instance %q: %w", bestEffortInstanceName(authored), vErr)
	}
	return nil
}

// valuesConflict is the failure-path attribution both instance verbs share.
// authored is the instance package built without the rendered values file
// (the authored package on the acquire path, the values-free synthesized
// package on the synthesis path), and compiled the call's values, compiled
// once in the context authored was built in. It unifies authored's own
// values with compiled and validates the result against authored's #config
// without requiring concreteness: a field the values leave unset is not a
// conflict, and a missing-field message with no source position must never
// replace the real build error. Concreteness is enforced by the instance
// processing step on a build that succeeds. It returns the raw CUE error, or
// nil when the values are clean or there is nothing to check.
func valuesConflict(authored cue.Value, compiled []cue.Value) error {
	configSchema := authored.LookupPath(schema.Module).LookupPath(schema.Config)
	all := make([]cue.Value, 0, len(compiled)+1)
	if own := authored.LookupPath(schema.Values); own.Exists() {
		all = append(all, own)
	}
	all = append(all, compiled...)
	_, err := validateCompiled(configSchema, all, false)
	return err
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
