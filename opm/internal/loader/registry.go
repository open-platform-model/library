package loader

import (
	"context"
	"fmt"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/mod/modconfig"
	"cuelang.org/go/mod/module"

	oerrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/internal/sourcetree"
	opmmodule "github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/schema"
)

// FetchModule loads a #Module published in an OCI registry: [FetchArtifact]
// with [ModuleSpec], plus the coordinate identity check that is the module
// path's alone ([verifyModuleIdentity], 0010 D11). It is the registry path's
// single entry for modules, so the check runs for every caller behind it.
//
// Other kinds go through FetchArtifact with their own spec and do NOT get the
// coordinate check: the requirement admitting the catalog kind (ADR-009) names
// only the shape refusal, and a kernel that refuses on something no
// requirement describes is worse than an unchecked coordinate. Generalizing
// the check is its own change.
func FetchModule(ctx context.Context, cueCtx *cue.Context, modPath, version string, env []string) (cue.Value, *opmmodule.Source, error) {
	val, src, err := FetchArtifact(ctx, cueCtx, modPath, version, env, ModuleSpec)
	if err != nil {
		return cue.Value{}, nil, err
	}
	if err := verifyModuleIdentity(val, modPath, version); err != nil {
		return cue.Value{}, nil, err
	}
	return val, src, nil
}

// FetchArtifact loads an OPM artifact published in an OCI registry, identified
// by its major-qualified module path (e.g. "example.com/modules/hello@v0") and
// version (e.g. "v0.0.2"), gated to the shape spec names, and returns the
// value built in cueCtx together with the staged source tree the build used,
// as the artifact
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
// (mod/modconfig) and builds it IN MEMORY AS THE MAIN MODULE through
// [LoadDir]'s overlay mode: the fetched files are the overlay under a
// deterministic synthetic root, so the module's own cue.mod/module.cue
// drives transitive dependency resolution and its kind/metadata are
// evaluated at the package root. No wrapper package is synthesized and no
// temporary directory is written.
//
// Because the build IS [LoadDir] — the kernel's one evaluate-and-shape-gate
// routine, the same call directory acquisition makes — the fetched artifact
// is evaluated, shape-gated against spec (its concrete kind and its concrete
// identity fields) and error-wrapped exactly as a directory artifact of that
// kind is, wrapping the shared ErrInvalidPackage / ErrWrongKind /
// ErrMissingRequiredField sentinels. The two acquisition paths differ only in
// where the package files come from. Neither performs full schema validation,
// which remains the kernel's contract.
//
// Everything up to and including the staging reads no kind: fetching and
// unpacking a published CUE module artifact is the same work whatever OPM
// kind its root package declares, which is why one routine serves every kind
// and the kernel holds no second fetch path (ADR-009). spec is the only thing
// that differs between them.
//
// env is the environment slice the fetch resolver and the load both consult —
// the kernel's CUE_REGISTRY mapping via [cueenv.Override], nil to read the
// process environment unchanged. The process environment is never mutated.
// Parse failures on caller input are wrapped rather than panicked.
func FetchArtifact(ctx context.Context, cueCtx *cue.Context, modPath, version string, env []string, spec ArtifactSpec) (cue.Value, *opmmodule.Source, error) {
	mv, err := module.NewVersion(modPath, version)
	if err != nil {
		return cue.Value{}, nil, fmt.Errorf("parsing artifact version %s@%s: %w", modPath, version, err)
	}

	reg, err := modconfig.NewRegistry(&modconfig.Config{Env: env})
	if err != nil {
		return cue.Value{}, nil, fmt.Errorf("building module registry resolver: %w", err)
	}

	// Fetch downloads if necessary and returns the extracted artifact's source
	// location (the modcache returns {FS: OSDirFS(extractDir), Dir: "."}).
	loc, err := reg.Fetch(ctx, mv)
	if err != nil {
		return cue.Value{}, nil, fmt.Errorf("fetching %s %s: %w", spec.Label, mv, err)
	}

	// Stage the fetched artifact's .cue files in memory under a deterministic
	// synthetic root. A fetch carrying no .cue file is not an artifact.
	synthRoot := sourcetree.SyntheticRoot(modPath, version)
	overlay, err := sourcetree.OverlayFromFS(loc.FS, loc.Dir, synthRoot)
	if err != nil {
		return cue.Value{}, nil, fmt.Errorf("staging %s %s in overlay: %w", spec.Label, mv, err)
	}
	if len(overlay) == 0 {
		return cue.Value{}, nil, fmt.Errorf("staging %s %s in overlay: fetched source has no CUE files: %w", spec.Label, mv, oerrors.ErrInvalidPackage)
	}

	// Build and shape-gate through LoadDir's overlay mode: the staged files
	// enter cue/load as an Overlay under the synthetic root (with FS left
	// nil), NOT through load.Config.FS. The spike confirmed that pinning
	// load.Config.FS to the fetched module's SourceLoc.FS FAILS on transitive
	// deps: the loader then reads ALL source — including deps — only through
	// that one FS, and the module's catalog/core deps live in separate cache
	// directories ("cannot find package opmodel.dev/catalogs/opm/resources").
	// Overlay injects only the target module's files while leaving normal
	// registry/cache dependency resolution intact. Do not "simplify" this to
	// FS-pinning; registry_internal_test.go pins the negative result. See
	// design.md § Research & Decisions (add-registry-module-loader).
	val, err := LoadDir(cueCtx, synthRoot, ".", overlay, env, spec)
	if err != nil {
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
// FetchModule, the registry path's single entry FOR MODULES, the check runs
// for every caller (Kernel.AcquireModuleFromRegistry and the frontends behind
// it): D11's one implementation. It is module-only by decision, not by
// oversight — see FetchModule.
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
