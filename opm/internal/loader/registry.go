package loader

import (
	"context"
	"fmt"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/mod/modconfig"
	"cuelang.org/go/mod/module"

	oerrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/internal/sourcetree"
	opmmodule "github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/schema"
)

// FetchModule loads a #Module published in an OCI registry, identified by its
// major-qualified module path (e.g. "example.com/modules/hello@v0") and
// version (e.g. "v0.0.2"), and returns the value built in cueCtx together
// with the staged source tree the build used, as the artifact
// [opmmodule.Source] in overlay mode: the deterministic synthetic Root every
// overlay key sits under, plus the Overlay carrying the module's .cue files
// (its own cue.mod/module.cue included, nothing else: the set cue/load
// reads). A consumer reuses it to build a follow-on package INSIDE the
// module's own main module — letting the module's already-tidied
// cue.mod/module.cue drive transitive resolution — without a second registry
// fetch (Principle V, CUE-native resolution). The returned Overlay is the
// build's own map; callers that mutate it (e.g. to overlay additional files)
// MUST clone it first.
//
// It fetches the module's source via CUE's native module machinery
// (mod/modconfig) and loads it IN MEMORY AS THE MAIN MODULE: the fetched files
// are injected through load.Config.Overlay under a deterministic synthetic
// root, so the module's own cue.mod/module.cue drives transitive dependency
// resolution and its kind/metadata are evaluated at the package root. No
// wrapper package is synthesized and no temporary directory is written.
//
// The built value is validated with the same module shape gate [LoadDir] runs
// for a directory (concrete kind == "Module"; concrete metadata.name,
// metadata.modulePath, metadata.version), wrapping the shared
// ErrInvalidPackage / ErrWrongKind / ErrMissingRequiredField sentinels, so a
// directory-acquired and a registry-acquired module fail identically. It does
// NOT perform full schema validation, which remains the kernel's contract.
//
// env is the environment slice the fetch resolver and the load both consult —
// the kernel's CUE_REGISTRY mapping via [cueenv.Override], nil to read the
// process environment unchanged. The process environment is never mutated.
// Parse failures on caller input are wrapped rather than panicked.
func FetchModule(ctx context.Context, cueCtx *cue.Context, modPath, version string, env []string) (cue.Value, *opmmodule.Source, error) {
	mv, err := module.NewVersion(modPath, version)
	if err != nil {
		return cue.Value{}, nil, fmt.Errorf("parsing module version %s@%s: %w", modPath, version, err)
	}

	reg, err := modconfig.NewRegistry(&modconfig.Config{Env: env})
	if err != nil {
		return cue.Value{}, nil, fmt.Errorf("building module registry resolver: %w", err)
	}

	// Fetch downloads if necessary and returns the extracted module's source
	// location (the modcache returns {FS: OSDirFS(extractDir), Dir: "."}).
	loc, err := reg.Fetch(ctx, mv)
	if err != nil {
		return cue.Value{}, nil, fmt.Errorf("fetching module %s: %w", mv, err)
	}

	// Stage the fetched module's .cue files in memory under a deterministic
	// synthetic root. A fetch carrying no .cue file is not a module.
	synthRoot := sourcetree.SyntheticRoot(modPath, version)
	overlay, err := sourcetree.OverlayFromFS(loc.FS, loc.Dir, synthRoot)
	if err != nil {
		return cue.Value{}, nil, fmt.Errorf("staging module %s in overlay: %w", mv, err)
	}
	if len(overlay) == 0 {
		return cue.Value{}, nil, fmt.Errorf("staging module %s in overlay: fetched module source has no CUE files: %w", mv, oerrors.ErrInvalidPackage)
	}

	// Overlay (with FS left nil), NOT load.Config.FS. The spike confirmed that
	// pinning load.Config.FS to the fetched module's SourceLoc.FS FAILS on
	// transitive deps: the loader then reads ALL source — including deps — only
	// through that one FS, and the module's catalog/core deps live in separate
	// cache directories ("cannot find package opmodel.dev/catalogs/opm/resources").
	// Overlay injects only the target module's files while leaving normal
	// registry/cache dependency resolution intact. Do not "simplify" this to
	// FS-pinning. See design.md § Research & Decisions (add-registry-module-loader).
	cueOverlay := make(map[string]load.Source, len(overlay))
	for path, data := range overlay {
		cueOverlay[path] = load.FromBytes(data)
	}
	cfg := &load.Config{
		Dir:        synthRoot,
		ModuleRoot: synthRoot,
		Overlay:    cueOverlay,
		Env:        env,
	}
	instances := load.Instances([]string{"."}, cfg)
	if len(instances) != 1 {
		return cue.Value{}, nil, fmt.Errorf("expected exactly one CUE package in module %s, found %d: %w", mv, len(instances), oerrors.ErrInvalidPackage)
	}
	if instances[0].Err != nil {
		return cue.Value{}, nil, fmt.Errorf("loading module package %s: %w", mv, instances[0].Err)
	}

	val := cueCtx.BuildInstance(instances[0])
	if err := val.Err(); err != nil {
		return cue.Value{}, nil, fmt.Errorf("building module package %s: %w", mv, err)
	}

	if err := gate(val, ModuleSpec); err != nil {
		return cue.Value{}, nil, fmt.Errorf("validating module package %s: %w", mv, err)
	}

	if err := verifyModuleIdentity(val, modPath, version); err != nil {
		return cue.Value{}, nil, err
	}

	return val, &opmmodule.Source{Root: synthRoot, Overlay: overlay}, nil
}

// verifyModuleIdentity compares the acquired module's declared identity
// against the coordinate it was fetched by (0010 D11; version clause D9): the
// declared metadata.modulePath must equal the requested major-qualified path
// as a string, and the declared metadata.version must equal the fetched tag
// with the `v` prefix stripped. The shape gate has already guaranteed both
// fields present and concrete (ModuleSpec.RequiredConcreteFields), so
// the check cannot misfire on absence. A mismatch returns a bare
// oerrors.IdentityError naming both values. Sitting after the gate in
// FetchModule, the registry path's single entry, the check runs for every
// caller (Kernel.AcquireModuleFromRegistry and the frontends behind it):
// D11's one implementation.
//
// There is no alternative check for a major-free declaration: the core
// schema the library consumes requires the major-suffixed form
// (#ModulePathType), so a modulePath without it cannot equal the fetched
// path and is refused with the same typed error. The kernel verifies; it
// never composes a module address from a parent path and a name.
func verifyModuleIdentity(val cue.Value, modPath, version string) error {
	coordinate := modPath + " " + version

	meta := val.LookupPath(schema.Metadata)
	declaredPath, err := meta.LookupPath(cue.ParsePath("modulePath")).String()
	if err != nil {
		return fmt.Errorf("reading metadata.modulePath of %s: %w", coordinate, err)
	}
	if declaredPath != modPath {
		return oerrors.IdentityError{
			Field:      "path",
			Declared:   declaredPath,
			Fetched:    modPath,
			Coordinate: coordinate,
		}
	}

	declaredVersion, err := meta.LookupPath(cue.ParsePath("version")).String()
	if err != nil {
		return fmt.Errorf("reading metadata.version of %s: %w", coordinate, err)
	}
	if fetched := strings.TrimPrefix(version, "v"); declaredVersion != fetched {
		return oerrors.IdentityError{
			Field:      "version",
			Declared:   declaredVersion,
			Fetched:    fetched,
			Coordinate: coordinate,
		}
	}

	return nil
}
